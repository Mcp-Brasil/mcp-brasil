"""Tool functions for the NucleoGov feature.

Rules (ADR-001):
    - tools.py NEVER makes HTTP directly — delegates to client.py
    - Returns formatted strings for LLM consumption
    - Uses Context for structured logging
"""

from __future__ import annotations

from fastmcp import Context

from mcp_brasil._shared.formatting import format_brl
from mcp_brasil.exceptions import HttpClientError

from . import client
from .schemas import Contrato, Despesa, Licitacao

_MAX_ITENS = 20


def _valor(valor: float | None) -> str:
    return format_brl(valor) if valor is not None else "—"


def _rodape(total: int, mostrados: int, inicio: int, limite: int) -> list[str]:
    if total > mostrados:
        return [
            f"*Mostrando {mostrados} de {total}. "
            f"Use inicio={inicio + limite} para a próxima página.*"
        ]
    return []


def _vazio(total: int, portal: str, inicio: int, limite: int) -> str:
    """Message for an empty page that never asserts non-existence."""
    if total > 0:
        return (
            f"O portal {portal} declara {total} registros para estes filtros, mas nenhum "
            f"foi retornado nesta página (inicio={inicio}, limite={limite}); "
            "ajuste a paginação ou o limite."
        )
    return f"Nenhum resultado para estes filtros no portal {portal}."


async def buscar_despesas(
    ctx: Context,
    portal: str,
    data_inicio: str | None = None,
    data_fim: str | None = None,
    cnpj: str | None = None,
    busca: str | None = None,
    inicio: int = 0,
    limite: int = 100,
) -> str:
    """Busca empenhos (despesas) de um portal municipal NucleoGov.

    Use quando precisar de despesas/empenhos de uma prefeitura ou câmara cujo
    Portal da Transparência está na plataforma NucleoGov (domínio no padrão
    acessoainformacao.<municipio>.go.gov.br). Traz órgão, fornecedor, histórico,
    valor empenhado e valor pago, inclusive para consultar pagamentos a um
    fornecedor por CNPJ.

    Args:
        ctx: Contexto MCP.
        portal: Domínio ou URL do portal (ex.: acessoainformacao.cidade.go.gov.br).
        data_inicio: Início do período (AAAA-MM-DD, AAAA-MM ou dd/mm/aaaa).
        data_fim: Fim do período (AAAA-MM-DD, AAAA-MM ou dd/mm/aaaa). Com AAAA-MM
            o período cobre o mês inteiro (último dia).
        cnpj: CNPJ ou CPF do fornecedor (filtro remoto da fonte, cobre todas as
            páginas, e revalidado sobre a página retornada).
        busca: Texto livre (nome do fornecedor, número do empenho etc.).
        inicio: Offset para paginação.
        limite: Quantidade máxima de registros.

    Returns:
        Lista de empenhos com fornecedor, histórico e valores.
    """
    await ctx.info(f"Buscando despesas no portal {portal}...")
    try:
        resultado = await client.buscar_despesas(
            portal,
            data_inicio=data_inicio,
            data_fim=data_fim,
            cnpj=cnpj,
            busca=busca,
            inicio=inicio,
            limite=limite,
        )
    except (HttpClientError, ValueError) as exc:
        return f"Não foi possível consultar as despesas em {portal}: {exc}"

    if not resultado.despesas:
        return _vazio(resultado.total, portal, inicio, limite)

    mostrados = resultado.despesas[:_MAX_ITENS]
    linhas: list[str] = [f"**{resultado.total} despesas no portal {portal}:**\n"]
    if cnpj:
        linhas.append(f"*(filtro por CNPJ/CPF {cnpj} aplicado sobre a página retornada)*\n")
    for d in mostrados:
        linhas.extend(_formatar_despesa(d))
    linhas.extend(_rodape(resultado.total, len(mostrados), inicio, limite))
    return "\n".join(linhas)


def _formatar_despesa(d: Despesa) -> list[str]:
    titulo = d.numero_empenho or d.numero_processo or d.id or "—"
    linhas = [f"### Empenho {titulo} — {d.data or '—'}"]
    linhas.append(f"- **Órgão:** {d.orgao or '—'}")
    if d.unidade:
        linhas.append(f"- **Unidade:** {d.unidade}")
    linhas.append(f"- **Fornecedor:** {d.fornecedor or '—'} {d.cpf_cnpj or ''}".rstrip())
    if d.historico:
        linhas.append(f"- **Histórico:** {d.historico}")
    linhas.append(f"- **Empenhado:** {_valor(d.valor_empenho)} | **Pago:** {_valor(d.valor_pago)}")
    linhas.append("")
    return linhas


async def buscar_contratos(
    ctx: Context,
    portal: str,
    ano: int | None = None,
    tipo: str = "contrato",
    busca: str | None = None,
    inicio: int = 0,
    limite: int = 100,
) -> str:
    """Busca contratos ou aditivos de um portal municipal NucleoGov.

    Use quando precisar de contratos administrativos (parte, objeto, vigência,
    valor) de um município cujo Portal da Transparência está na plataforma
    NucleoGov. Use tipo="aditivo" para ver termos aditivos.

    Args:
        ctx: Contexto MCP.
        portal: Domínio ou URL do portal.
        ano: Ano-base do contrato.
        tipo: "contrato" ou "aditivo".
        busca: Texto livre (objeto, contratado etc.).
        inicio: Offset para paginação.
        limite: Quantidade máxima de registros.

    Returns:
        Lista de contratos com contratado, objeto, vigência e valor.
    """
    await ctx.info(f"Buscando contratos no portal {portal}...")
    try:
        resultado = await client.buscar_contratos(
            portal, ano=ano, tipo=tipo, busca=busca, inicio=inicio, limite=limite
        )
    except (HttpClientError, ValueError) as exc:
        return f"Não foi possível consultar os contratos em {portal}: {exc}"

    if not resultado.contratos:
        return _vazio(resultado.total, portal, inicio, limite)

    mostrados = resultado.contratos[:_MAX_ITENS]
    linhas: list[str] = [f"**{resultado.total} contratos no portal {portal}:**\n"]
    for c in mostrados:
        linhas.extend(_formatar_contrato(c))
    linhas.extend(_rodape(resultado.total, len(mostrados), inicio, limite))
    return "\n".join(linhas)


def _formatar_contrato(c: Contrato) -> list[str]:
    linhas = [f"### {c.numero or c.id or '—'} ({c.ano or '—'})"]
    linhas.append(f"- **Órgão:** {c.orgao or '—'}")
    contratado = f"{c.contratado_nome or '—'} {c.contratado_documento or ''}".rstrip()
    linhas.append(f"- **Contratado:** {contratado}")
    linhas.append(f"- **Objeto:** {c.objeto or '—'}")
    linhas.append(f"- **Valor:** {_valor(c.valor)}")
    if c.inicio_vigencia or c.fim_vigencia:
        linhas.append(f"- **Vigência:** {c.inicio_vigencia or '—'} a {c.fim_vigencia or '—'}")
    linhas.append(f"- **Situação:** {c.situacao or '—'}")
    linhas.append("")
    return linhas


async def buscar_licitacoes(
    ctx: Context,
    portal: str,
    ano: int | None = None,
    busca: str | None = None,
    inicio: int = 0,
    limite: int = 100,
) -> str:
    """Busca licitações de um portal municipal NucleoGov.

    Use quando precisar de processos licitatórios (modalidade, objeto, situação,
    valor estimado) de um município cujo Portal da Transparência está na
    plataforma NucleoGov. Para dispensas e inexigibilidades, use
    ``buscar_dispensas``.

    Args:
        ctx: Contexto MCP.
        portal: Domínio ou URL do portal.
        ano: Ano-base do processo.
        busca: Texto livre (objeto, número do processo etc.).
        inicio: Offset para paginação.
        limite: Quantidade máxima de registros.

    Returns:
        Lista de licitações com modalidade, objeto, situação e valor estimado.
    """
    await ctx.info(f"Buscando licitações no portal {portal}...")
    try:
        resultado = await client.buscar_licitacoes(
            portal, ano=ano, dispensas=False, busca=busca, inicio=inicio, limite=limite
        )
    except (HttpClientError, ValueError) as exc:
        return f"Não foi possível consultar as licitações em {portal}: {exc}"

    if not resultado.licitacoes:
        return _vazio(resultado.total, portal, inicio, limite)

    mostrados = resultado.licitacoes[:_MAX_ITENS]
    linhas: list[str] = [f"**{resultado.total} licitações no portal {portal}:**\n"]
    for lic in mostrados:
        linhas.extend(_formatar_licitacao(lic))
    linhas.extend(_rodape(resultado.total, len(mostrados), inicio, limite))
    return "\n".join(linhas)


async def buscar_dispensas(
    ctx: Context,
    portal: str,
    ano: int | None = None,
    busca: str | None = None,
    inicio: int = 0,
    limite: int = 100,
) -> str:
    """Busca dispensas e inexigibilidades de licitação em portal NucleoGov.

    Use quando precisar de contratações diretas (dispensa ou inexigibilidade)
    de um município cujo Portal da Transparência está na plataforma NucleoGov.

    Args:
        ctx: Contexto MCP.
        portal: Domínio ou URL do portal.
        ano: Ano-base do processo.
        busca: Texto livre (objeto, número do processo etc.).
        inicio: Offset para paginação.
        limite: Quantidade máxima de registros.

    Returns:
        Lista de contratações diretas com modalidade, objeto e valor estimado.
    """
    await ctx.info(f"Buscando dispensas e inexigibilidades no portal {portal}...")
    try:
        resultado = await client.buscar_licitacoes(
            portal, ano=ano, dispensas=True, busca=busca, inicio=inicio, limite=limite
        )
    except (HttpClientError, ValueError) as exc:
        return f"Não foi possível consultar as contratações diretas em {portal}: {exc}"

    if not resultado.licitacoes:
        return _vazio(resultado.total, portal, inicio, limite)

    mostrados = resultado.licitacoes[:_MAX_ITENS]
    linhas: list[str] = [f"**{resultado.total} dispensas/inexigibilidades no portal {portal}:**\n"]
    for lic in mostrados:
        linhas.extend(_formatar_licitacao(lic))
    linhas.extend(_rodape(resultado.total, len(mostrados), inicio, limite))
    return "\n".join(linhas)


def _formatar_licitacao(lic: Licitacao) -> list[str]:
    linhas = [f"### {lic.numero or lic.id or '—'} — {lic.modalidade or lic.modalidade_tce or '—'}"]
    linhas.append(f"- **Órgão:** {lic.orgao or '—'}")
    linhas.append(f"- **Objeto:** {lic.descricao or '—'}")
    linhas.append(f"- **Valor estimado:** {_valor(lic.valor_estimado)}")
    linhas.append(f"- **Situação:** {lic.situacao or '—'}")
    if lic.data_abertura or lic.data_edital:
        linhas.append(f"- **Abertura:** {lic.data_abertura or lic.data_edital}")
    linhas.append("")
    return linhas


async def buscar_legislacao(
    ctx: Context,
    portal: str,
    municipio: str,
    tipo: str | None = None,
    busca: str | None = None,
    inicio: int = 0,
    limite: int = 100,
) -> str:
    """Busca leis, decretos e portarias de um portal municipal NucleoGov.

    Use quando precisar da legislação municipal (atos normativos) publicada no
    Portal da Transparência de um município na plataforma NucleoGov. O portal
    serve a legislação por município, na ação atos_pf_{slug}; informe o nome do
    município ou o slug usado pelo portal (ex.: senador_canedo).

    Args:
        ctx: Contexto MCP.
        portal: Domínio ou URL do portal.
        municipio: Nome do município ou slug usado pelo portal.
        tipo: Tipo do ato (DECRETO, PORTARIA, LEI etc.).
        busca: Texto livre (ementa, número etc.).
        inicio: Offset para paginação.
        limite: Quantidade máxima de registros.

    Returns:
        Lista de atos normativos com tipo, número, ementa e data.
    """
    await ctx.info(f"Buscando legislação no portal {portal}...")
    try:
        resultado = await client.buscar_legislacao(
            portal, municipio=municipio, tipo=tipo, busca=busca, inicio=inicio, limite=limite
        )
    except (HttpClientError, ValueError) as exc:
        return f"Não foi possível consultar a legislação em {portal}: {exc}"

    if not resultado.atos:
        return _vazio(resultado.total, portal, inicio, limite)

    mostrados = resultado.atos[:_MAX_ITENS]
    linhas: list[str] = [f"**{resultado.total} atos normativos no portal {portal}:**\n"]
    for ato in mostrados:
        linhas.append(f"### {ato.tipo or '—'} {ato.numero or '—'} — {ato.data_publicacao or '—'}")
        linhas.append(f"- **Ementa:** {ato.ementa or '—'}")
        if ato.orgao:
            linhas.append(f"- **Órgão:** {ato.orgao}")
        if ato.url:
            linhas.append(f"- **Link:** {ato.url}")
        linhas.append("")
    linhas.extend(_rodape(resultado.total, len(mostrados), inicio, limite))
    return "\n".join(linhas)
