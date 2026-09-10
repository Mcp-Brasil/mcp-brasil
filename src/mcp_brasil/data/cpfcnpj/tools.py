"""Tools da feature CPF.CNPJ.

Regras (ADR-001):
    - tools.py NUNCA faz HTTP direto, delega ao client.py
    - Retorna strings formatadas para consumo do LLM
    - Usa Context para log estruturado

LGPD: CPF de responsável e de sócios pessoa natural saem MASCARADOS por padrão,
respeitando ``MCP_BRASIL_LGPD_ALLOW_PII`` (escopo ``cpfcnpj``) do mesmo modo que
o projeto faz nos datasets. CNPJ de sócio pessoa jurídica não é PII e sai inteiro.
Toda tool valida o documento localmente ANTES de chamar o client (sem gastar crédito).
"""

from __future__ import annotations

from fastmcp import Context

from mcp_brasil import settings
from mcp_brasil._shared.datasets.lgpd import mask_value
from mcp_brasil._shared.formatting import markdown_table

from . import client
from .constants import PACOTE_CADASTRO, PACOTE_COMPLETO
from .schemas import (
    InscricaoEstadual,
    MatrizEndereco,
    Qualificacao,
    Situacao,
    Socio,
)
from .validacao import (
    classificar_documento,
    normalizar_cnpj,
    normalizar_cpf,
    validar_cnpj,
    validar_cpf,
)

_CNPJ_INVALIDO = (
    "CNPJ inválido ou incompleto. Verifique o documento informado "
    "(numérico ou alfanumérico, com ou sem pontuação)."
)


def _pii_liberada() -> bool:
    """Indica se a PII do escopo cpfcnpj foi liberada via MCP_BRASIL_LGPD_ALLOW_PII."""
    return "cpfcnpj" in settings.DATASETS_ALLOW_PII


def _mascarar_documento_socio(documento: str | None) -> str:
    """Mascara CPF de sócio pessoa natural; CNPJ de sócio PJ sai inteiro."""
    if not documento:
        return "N/A"
    digitos = "".join(c for c in documento if c.isdigit())
    if len(digitos) == 14:  # CNPJ de sócio PJ, não é PII
        return documento
    if _pii_liberada():
        return documento
    return mask_value(documento)


def _mascarar_cpf(documento: str | None) -> str:
    """Mascara um CPF de pessoa natural, salvo liberação explícita de PII."""
    if not documento:
        return "N/A"
    if _pii_liberada():
        return documento
    return mask_value(documento)


def _descricao_qualificacao(qualificacao: Qualificacao | str | None) -> str:
    if qualificacao is None:
        return "N/A"
    if isinstance(qualificacao, str):
        return qualificacao.strip() or "N/A"
    return (qualificacao.descricao or "").strip() or "N/A"


def _sim_nao(valor: bool | str | None) -> str:
    """Normaliza um indicador booleano ou textual ('Sim'/'Não') para 'sim'/'não'."""
    if valor is None:
        return "N/A"
    if isinstance(valor, bool):
        return "sim" if valor else "não"
    texto = str(valor).strip().lower()
    if texto in ("sim", "true", "1", "optante", "s"):
        return "sim"
    if texto in ("não", "nao", "false", "0", "n"):
        return "não"
    return str(valor)


def _formatar_endereco(endereco: MatrizEndereco | None) -> str:
    if endereco is None:
        return "N/A"
    partes = [endereco.logradouro, endereco.numero, endereco.complemento, endereco.bairro]
    linha = ", ".join(p for p in partes if p)
    cidade_uf = "/".join(p for p in (endereco.cidade, endereco.uf) if p)
    if cidade_uf:
        linha = f"{linha}, {cidade_uf}" if linha else cidade_uf
    if endereco.cep:
        linha = f"{linha}, CEP {endereco.cep}" if linha else f"CEP {endereco.cep}"
    return linha or "N/A"


def _formatar_situacao(situacao: Situacao | None) -> str:
    if situacao is None:
        return "N/A"
    partes = [situacao.nome]
    if situacao.data:
        partes.append(f"desde {situacao.data}")
    if situacao.motivo and situacao.motivo.descricao:
        partes.append(f"motivo: {situacao.motivo.descricao}")
    return ", ".join(p for p in partes if p) or "N/A"


def _tabela_socios(socios: list[Socio]) -> str:
    rows = [
        (
            (s.nome or "").strip() or "N/A",
            _descricao_qualificacao(s.qualificacao_socio),
            s.data_entrada or "N/A",
            _mascarar_documento_socio(s.cpf_cnpj_socio),
        )
        for s in socios
    ]
    return markdown_table(["Sócio", "Qualificação", "Entrada", "Documento"], rows)


def _tabela_inscricoes(inscricoes: list[InscricaoEstadual]) -> str:
    rows = []
    for ie in inscricoes:
        uf = ie.estado.sigla if ie.estado else None
        rows.append(
            (
                uf or "N/A",
                ie.inscricao_estadual or "N/A",
                "ativa" if ie.ativo else "inativa",
            )
        )
    return markdown_table(["UF", "Inscrição Estadual", "Situação"], rows)


async def validar_documento(documento: str, ctx: Context) -> str:
    """Valida um CPF ou CNPJ localmente, sem rede e sem consumir crédito.

    Verifica o dígito verificador de CPF (11 dígitos) e de CNPJ numérico ou
    alfanumérico (12 posições alfanuméricas + 2 dígitos verificadores numéricos,
    módulo 11 sobre ``ord(c) - 48``, conforme a IN RFB 2.229/2024). Aceita
    pontuação padrão (``.``, ``/``, ``-``); espaços internos e outros caracteres
    tornam o documento inválido. Não consome crédito da conta CPF.CNPJ.

    Args:
        documento: CPF ou CNPJ, com ou sem pontuação.

    Returns:
        Tipo do documento, se é válido e a forma normalizada.
    """
    await ctx.info("Validando documento localmente (sem rede)...")
    tipo = classificar_documento(documento)
    if tipo == "cpf":
        valido = validar_cpf(documento)
        normalizado = normalizar_cpf(documento)
        lines = [
            "**Tipo:** CPF",
            f"**Válido:** {'sim' if valido else 'não'}",
            f"**Normalizado:** {normalizado}",
        ]
        return "\n".join(lines)
    if tipo == "cnpj":
        valido = validar_cnpj(documento)
        normalizado = normalizar_cnpj(documento)
        lines = [
            "**Tipo:** CNPJ",
            f"**Válido:** {'sim' if valido else 'não'}",
            f"**Normalizado:** {normalizado}",
            f"**Alfanumérico:** {'sim' if not normalizado.isdigit() else 'não'}",
        ]
        return "\n".join(lines)
    return "**Documento:** não reconhecido como CPF (11 dígitos) nem CNPJ (14 posições)."


async def consultar_cnpj(cnpj: str, completo: bool = False, *, ctx: Context) -> str:
    """Consulta dados cadastrais de um CNPJ na fonte CPF.CNPJ.

    Usa o pacote 5 (padrão) ou o pacote 6 quando ``completo=True``.
    Consome crédito da conta CPF.CNPJ.

    O pacote 5 traz razão social, nome fantasia, endereço e códigos IBGE. O
    pacote 6 acrescenta situação cadastral (com data e motivo), porte, natureza
    jurídica, CNAE principal e secundárias, Simples/MEI, regimes tributários,
    responsável e quadro de sócios (QSA). Aceita CNPJ numérico e alfanumérico.

    Args:
        cnpj: CNPJ com ou sem pontuação (numérico ou alfanumérico).
        completo: Se True, usa o pacote 6 (dados completos); senão o pacote 5.

    Returns:
        Dados cadastrais da empresa em markdown.
    """
    if not validar_cnpj(cnpj):
        return _CNPJ_INVALIDO
    documento = normalizar_cnpj(cnpj)
    pacote = PACOTE_COMPLETO if completo else PACOTE_CADASTRO
    await ctx.info(f"Consultando CNPJ no pacote {pacote} da CPF.CNPJ...")
    emp = await client.consultar_cnpj(documento, pacote)

    lines = [
        f"**CNPJ:** {emp.cnpj or documento}",
        f"**Razão Social:** {emp.razao or 'N/A'}",
        f"**Nome Fantasia:** {emp.fantasia or 'N/A'}",
        f"**Tipo:** {emp.tipo or 'N/A'}",
        f"**Endereço:** {_formatar_endereco(emp.matrizEndereco)}",
    ]
    if emp.ibge:
        municipio = emp.ibge.cidade.ibge_id if emp.ibge.cidade else None
        uf = emp.ibge.estado.sigla if emp.ibge.estado else None
        lines.append(f"**IBGE:** município {municipio or 'N/A'}, UF {uf or 'N/A'}")

    if not completo:
        return "\n".join(lines)

    lines.append(f"**Situação:** {_formatar_situacao(emp.situacao)}")
    lines.append(f"**Porte:** {emp.porte.descricao if emp.porte else 'N/A'}")
    if emp.naturezaJuridica:
        codigo = emp.naturezaJuridica.codigo
        descricao = emp.naturezaJuridica.descricao or "N/A"
        lines.append(f"**Natureza Jurídica:** {f'{codigo}: ' if codigo else ''}{descricao}")
    if emp.cnae:
        fiscal = emp.cnae.fiscal
        descricao = emp.cnae.descricao or "N/A"
        lines.append(f"**CNAE principal:** {f'{fiscal}: ' if fiscal else ''}{descricao}")
        if emp.cnae.secundarias:
            secundarias = ", ".join(s.descricao for s in emp.cnae.secundarias if s.descricao)
            if secundarias:
                lines.append(f"**CNAE secundárias:** {secundarias}")
    if emp.simplesNacional:
        optante = _sim_nao(emp.simplesNacional.optante)
        mei = _sim_nao(emp.simplesNacional.mei)
        lines.append(f"**Simples Nacional:** {optante} | **MEI:** {mei}")
    if emp.regimesTributarios:
        regimes = ", ".join(
            f"{r.ano}: {r.regime_tributario}"
            for r in emp.regimesTributarios
            if r.ano and r.regime_tributario
        )
        if regimes:
            lines.append(f"**Regimes Tributários:** {regimes}")
    if emp.responsavel:
        lines.append(
            f"**Responsável:** {emp.responsavel} (CPF {_mascarar_cpf(emp.responsavelCpf)})"
        )
    if emp.socios:
        lines.append("\n**Quadro de Sócios (QSA):**\n")
        lines.append(_tabela_socios(emp.socios))
    return "\n".join(lines)


async def situacao_cnpj(cnpj: str, ctx: Context) -> str:
    """Consulta a situação cadastral de um CNPJ na fonte CPF.CNPJ.

    Usa o pacote 6 e resume a situação cadastral (descrição, data e motivo),
    o porte e o início de atividades. Consome crédito da conta CPF.CNPJ.

    Args:
        cnpj: CNPJ com ou sem pontuação (numérico ou alfanumérico).

    Returns:
        Situação cadastral resumida em markdown.
    """
    if not validar_cnpj(cnpj):
        return _CNPJ_INVALIDO
    documento = normalizar_cnpj(cnpj)
    await ctx.info("Consultando situação cadastral no pacote 6 da CPF.CNPJ...")
    emp = await client.consultar_cnpj(documento, PACOTE_COMPLETO)
    lines = [
        f"**CNPJ:** {emp.cnpj or documento}",
        f"**Razão Social:** {emp.razao or 'N/A'}",
        f"**Situação:** {_formatar_situacao(emp.situacao)}",
        f"**Porte:** {emp.porte.descricao if emp.porte else 'N/A'}",
        f"**Início de Atividades:** {emp.inicioAtividade or 'N/A'}",
    ]
    return "\n".join(lines)


async def qsa_cnpj(cnpj: str, ctx: Context) -> str:
    """Consulta o quadro de sócios (QSA) de um CNPJ na fonte CPF.CNPJ.

    Usa o pacote 6 e devolve uma tabela de sócios com nome, qualificação, data
    de entrada e documento (CPF de sócio pessoa natural sai MASCARADO por padrão;
    CNPJ de sócio pessoa jurídica sai inteiro). Consome crédito da conta CPF.CNPJ.

    Args:
        cnpj: CNPJ com ou sem pontuação (numérico ou alfanumérico).

    Returns:
        Tabela do quadro societário em markdown.
    """
    if not validar_cnpj(cnpj):
        return _CNPJ_INVALIDO
    documento = normalizar_cnpj(cnpj)
    await ctx.info("Consultando quadro de sócios no pacote 6 da CPF.CNPJ...")
    emp = await client.consultar_cnpj(documento, PACOTE_COMPLETO)
    header = f"**Quadro de Sócios de {emp.razao or documento}**\n\n"
    if not emp.socios:
        return header + "Nenhum sócio informado."
    return header + _tabela_socios(emp.socios)


async def inscricoes_estaduais(cnpj: str, ctx: Context) -> str:
    """Consulta as inscrições estaduais de um CNPJ na fonte CPF.CNPJ.

    Usa o pacote 16 e devolve uma tabela de inscrições estaduais por UF, com a
    situação de cada uma. Consome crédito da conta CPF.CNPJ.

    Args:
        cnpj: CNPJ com ou sem pontuação (numérico ou alfanumérico).

    Returns:
        Tabela de inscrições estaduais em markdown.
    """
    if not validar_cnpj(cnpj):
        return _CNPJ_INVALIDO
    documento = normalizar_cnpj(cnpj)
    await ctx.info("Consultando inscrições estaduais no pacote 16 da CPF.CNPJ...")
    emp = await client.inscricoes_estaduais(documento)
    header = f"**Inscrições Estaduais de {emp.razao or documento}**\n\n"
    if not emp.inscricoesEstaduais:
        return header + "Nenhuma inscrição estadual encontrada."
    return header + _tabela_inscricoes(emp.inscricoesEstaduais)


async def saldo(pacote: int, ctx: Context) -> str:
    """Consulta o saldo de créditos de um pacote na conta CPF.CNPJ.

    Consulta gratuita: não consome crédito da conta CPF.CNPJ.

    Args:
        pacote: Número do pacote (ex: 5, 6 ou 16).

    Returns:
        Créditos restantes do pacote.
    """
    await ctx.info(f"Consultando saldo do pacote {pacote} na CPF.CNPJ...")
    dados = await client.saldo(pacote)
    nome = f" ({dados.nome})" if dados.nome else ""
    restante = dados.saldo if dados.saldo is not None else "N/A"
    lines = [
        f"**Pacote:** {dados.id or pacote}{nome}",
        f"**Créditos restantes:** {restante}",
    ]
    return "\n".join(lines)
