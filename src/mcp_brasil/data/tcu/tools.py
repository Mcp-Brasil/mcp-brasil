"""Tool functions for the TCU feature.

Rules (ADR-001):
    - tools.py NEVER makes HTTP directly — delegates to client.py
    - Returns formatted strings for LLM consumption
    - Uses Context for structured logging and progress reporting
"""

from __future__ import annotations

from fastmcp import Context

from mcp_brasil._shared.formatting import format_brl, markdown_table

from . import client
from .schemas import ParcelaDebito


async def consultar_acordaos(
    ctx: Context,
    quantidade: int = 10,
    inicio: int = 0,
) -> str:
    """Consulta acórdãos (decisões colegiadas) do TCU.

    Acórdãos são decisões dos colegiados do TCU (Plenário, 1ª e 2ª Câmaras).
    Retorna título, relator, colegiado, data da sessão e sumário.

    Args:
        quantidade: Quantidade de acórdãos a retornar (padrão: 10, máximo recomendado: 50).
        inicio: Índice inicial para paginação (padrão: 0).

    Returns:
        Tabela com os acórdãos encontrados.
    """
    await ctx.info(f"Buscando {quantidade} acórdãos do TCU (início: {inicio})...")
    acordaos = await client.consultar_acordaos(inicio=inicio, quantidade=quantidade)
    await ctx.info(f"{len(acordaos)} acórdãos encontrados")

    if not acordaos:
        return "Nenhum acórdão encontrado."

    rows = [
        (
            a.numero_acordao,
            a.ano_acordao,
            a.colegiado,
            a.relator,
            a.data_sessao,
            a.sumario[:100] + "..." if len(a.sumario) > 100 else a.sumario,
        )
        for a in acordaos
    ]
    return markdown_table(
        ["Número", "Ano", "Colegiado", "Relator", "Data Sessão", "Sumário"],
        rows,
    )


async def consultar_inabilitados(
    ctx: Context,
    cpf: str | None = None,
    limite: int = 25,
    inicio: int = 0,
) -> str:
    """Consulta pessoas inabilitadas para exercer cargo/função pública pelo TCU.

    Inabilitados são pessoas que foram proibidas pelo TCU de exercer cargo em
    comissão ou função de confiança na Administração Pública Federal.

    Pode buscar todos os inabilitados ou filtrar por CPF específico.

    Args:
        cpf: CPF (somente números) para buscar pessoa específica. Se omitido, lista todos.
        limite: Quantidade de registros por página (padrão: 25).
        inicio: Deslocamento para paginação (padrão: 0).

    Returns:
        Tabela com os inabilitados encontrados.
    """
    if cpf:
        await ctx.info(f"Buscando inabilitado com CPF {cpf}...")
    else:
        await ctx.info(f"Buscando inabilitados (limite: {limite}, início: {inicio})...")

    inabilitados = await client.consultar_inabilitados(cpf=cpf, offset=inicio, limit=limite)
    await ctx.info(f"{len(inabilitados)} inabilitado(s) encontrado(s)")

    if not inabilitados:
        return "Nenhum inabilitado encontrado."

    rows = [
        (
            i.nome,
            i.cpf,
            i.processo,
            i.deliberacao,
            i.data_final[:10] if i.data_final else "—",
            i.uf,
        )
        for i in inabilitados
    ]
    return markdown_table(
        ["Nome", "CPF", "Processo", "Deliberação", "Data Final", "UF"],
        rows,
    )


async def consultar_inidoneos(
    ctx: Context,
    cpf_cnpj: str | None = None,
    limite: int = 25,
    inicio: int = 0,
) -> str:
    """Consulta licitantes declarados inidôneos pelo TCU.

    Inidôneos são empresas ou pessoas que foram declaradas inidôneas pelo TCU
    e estão impedidas de participar de licitações na Administração Pública.

    Pode buscar todos os inidôneos ou filtrar por CPF/CNPJ específico.

    Args:
        cpf_cnpj: CPF ou CNPJ (somente números) para buscar. Se omitido, lista todos.
        limite: Quantidade de registros por página (padrão: 25).
        inicio: Deslocamento para paginação (padrão: 0).

    Returns:
        Tabela com os licitantes inidôneos encontrados.
    """
    if cpf_cnpj:
        await ctx.info(f"Buscando inidôneo com CPF/CNPJ {cpf_cnpj}...")
    else:
        await ctx.info(f"Buscando inidôneos (limite: {limite}, início: {inicio})...")

    inidoneos = await client.consultar_inidoneos(cpf_cnpj=cpf_cnpj, offset=inicio, limit=limite)
    await ctx.info(f"{len(inidoneos)} inidôneo(s) encontrado(s)")

    if not inidoneos:
        return "Nenhum licitante inidôneo encontrado."

    rows = [
        (
            i.nome,
            i.cpf_cnpj,
            i.processo,
            i.deliberacao,
            i.data_final[:10] if i.data_final else "—",
            i.uf,
        )
        for i in inidoneos
    ]
    return markdown_table(
        ["Nome", "CPF/CNPJ", "Processo", "Deliberação", "Data Final", "UF"],
        rows,
    )


async def consultar_certidoes(ctx: Context, cnpj: str) -> str:
    """Consulta certidões consolidadas de pessoa jurídica junto ao TCU, CNJ e CGU.

    Verifica a situação de uma empresa em 4 cadastros simultaneamente:
    - TCU: Licitantes Inidôneos
    - CNJ: CNIA (Condenações Cíveis por Improbidade)
    - CGU: CEIS (Empresas Inidôneas e Suspensas)
    - CGU: CNEP (Empresas Punidas)

    Args:
        cnpj: CNPJ da empresa (somente números, sem formatação).

    Returns:
        Resultado consolidado das certidões.
    """
    await ctx.info(f"Consultando certidões para CNPJ {cnpj}...")
    certidao = await client.consultar_certidoes(cnpj)
    await ctx.info(f"Certidões recebidas para {certidao.razao_social}")

    header = (
        f"**{certidao.razao_social}**"
        + (f" ({certidao.nome_fantasia})" if certidao.nome_fantasia else "")
        + f"\nCNPJ: {certidao.cnpj}\n\n"
    )

    if not certidao.certidoes:
        return header + "Nenhuma certidão retornada."

    rows = [
        (
            c.emissor,
            c.tipo,
            c.situacao,
            c.observacao or "—",
        )
        for c in certidao.certidoes
    ]
    return header + markdown_table(
        ["Emissor", "Tipo", "Situação", "Observação"],
        rows,
    )


async def consultar_pedidos_congresso(
    ctx: Context,
    numero_processo: str | None = None,
    pagina: int = 0,
) -> str:
    """Consulta solicitações e pedidos do Congresso Nacional ao TCU.

    Inclui requerimentos (REQ) e solicitações de informação (SIT) feitos
    por parlamentares ao Tribunal de Contas da União.

    Args:
        numero_processo: Número do processo TCU para buscar pedido específico.
        pagina: Página dos resultados (padrão: 0).

    Returns:
        Tabela com os pedidos encontrados.
    """
    if numero_processo:
        await ctx.info(f"Buscando pedido do processo {numero_processo}...")
    else:
        await ctx.info(f"Buscando pedidos do Congresso (página: {pagina})...")

    pedidos = await client.consultar_pedidos_congresso(
        numero_processo=numero_processo, page=pagina
    )
    await ctx.info(f"{len(pedidos)} pedido(s) encontrado(s)")

    if not pedidos:
        return "Nenhum pedido do Congresso encontrado."

    rows = [
        (
            p.tipo,
            str(p.numero),
            p.data_aprovacao[:10] if p.data_aprovacao else "—",
            p.autor or "—",
            p.processo_scn,
            p.assunto[:80] + "..." if len(p.assunto) > 80 else p.assunto,
        )
        for p in pedidos
    ]
    return markdown_table(
        ["Tipo", "Número", "Data Aprovação", "Autor", "Processo", "Assunto"],
        rows,
    )


async def calcular_debito(
    ctx: Context,
    data_atualizacao: str,
    data_fato: str,
    valor_original: float,
    aplica_juros: bool = True,
) -> str:
    """Calcula débito atualizado com correção monetária pela variação SELIC.

    Usa a calculadora pública do TCU para atualizar valores de débitos
    com correção monetária e juros de mora.

    Args:
        data_atualizacao: Data para atualização do valor (DD/MM/AAAA).
        data_fato: Data do fato gerador do débito (DD/MM/AAAA).
        valor_original: Valor original do débito em reais.
        aplica_juros: Se deve aplicar juros de mora (padrão: True).

    Returns:
        Resultado do cálculo com valor original, correção e total.
    """
    await ctx.info(
        f"Calculando débito: {format_brl(valor_original)} de {data_fato} "
        f"atualizado para {data_atualizacao}..."
    )

    parcela = ParcelaDebito(data_fato=data_fato, indicativo="D", valor_original=valor_original)
    resultado = await client.calcular_debito(
        data_atualizacao=data_atualizacao,
        parcelas=[parcela],
        aplica_juros=aplica_juros,
    )

    return (
        f"**Cálculo de Débito — TCU**\n\n"
        f"Data do fato: {data_fato}\n"
        f"Data de atualização: {data_atualizacao}\n"
        f"Juros de mora: {'Sim' if aplica_juros else 'Não'}\n\n"
        + markdown_table(
            ["Item", "Valor"],
            [
                ("Valor original", format_brl(resultado.saldo_debito)),
                ("Correção SELIC", format_brl(resultado.saldo_variacao_selic)),
                ("Juros de mora", format_brl(resultado.saldo_juros)),
                ("**Total atualizado**", f"**{format_brl(resultado.saldo_total)}**"),
            ],
        )
    )


async def consultar_termos_contratuais(
    ctx: Context,
    ano: int | None = None,
    fornecedor: str | None = None,
    limite: int = 20,
) -> str:
    """Consulta termos contratuais (contratos) firmados pelo próprio TCU.

    Retorna contratos, aditamentos e termos contratuais do tribunal.
    A API retorna todos os registros (~3800+), por isso é possível filtrar
    por ano e/ou nome do fornecedor.

    Args:
        ano: Ano do contrato para filtrar (ex: 2025).
        fornecedor: Parte do nome do fornecedor para filtrar (case-insensitive).
        limite: Quantidade máxima de registros a exibir (padrão: 20).

    Returns:
        Tabela com os termos contratuais encontrados.
    """
    await ctx.info("Buscando termos contratuais do TCU...")
    termos = await client.consultar_termos_contratuais()
    await ctx.info(f"{len(termos)} termos encontrados no total")

    # Filtrar
    if ano:
        termos = [t for t in termos if t.ano == ano]
    if fornecedor:
        termo_upper = fornecedor.upper()
        termos = [t for t in termos if termo_upper in t.nome_fornecedor.upper()]

    if not termos:
        return "Nenhum termo contratual encontrado com os filtros informados."

    await ctx.info(f"{len(termos)} termos após filtros")
    termos_exibir = termos[:limite]

    rows = [
        (
            f"{t.numero}/{t.ano}",
            t.nome_fornecedor[:40],
            t.objeto[:60] + "..." if len(t.objeto) > 60 else t.objeto,
            format_brl(t.valor_atualizado),
            t.modalidade_licitacao,
        )
        for t in termos_exibir
    ]
    header = f"Termos contratuais do TCU ({len(termos)} encontrados"
    if ano:
        header += f", ano: {ano}"
    if fornecedor:
        header += f", fornecedor: '{fornecedor}'"
    header += f", exibindo {len(termos_exibir)}):\n\n"

    return header + markdown_table(
        ["Contrato", "Fornecedor", "Objeto", "Valor Atualizado", "Modalidade"],
        rows,
    )


async def consultar_cadirreg(ctx: Context, cpf: str) -> str:
    """Consulta pessoa no CADIRREG — Cadastro de Responsáveis com Contas Irregulares.

    Verifica se uma pessoa (por CPF) possui contas julgadas irregulares pelo TCU.
    Retorna vazio se o CPF não constar no cadastro.

    Args:
        cpf: CPF da pessoa (somente números, sem formatação).

    Returns:
        Dados do cadastro ou mensagem de não encontrado.
    """
    await ctx.info(f"Consultando CADIRREG para CPF {cpf}...")
    registros = await client.consultar_cadirreg(cpf)
    await ctx.info(f"{len(registros)} registro(s) encontrado(s)")

    if not registros:
        return f"CPF {cpf} não consta no CADIRREG (sem contas irregulares)."

    rows = [
        (
            r.nome_responsavel,
            r.num_cpf,
            f"{r.num_processo}/{r.ano_processo}",
            r.julgamento[:60] + "..." if len(r.julgamento) > 60 else r.julgamento,
            r.se_detentor_cargo,
        )
        for r in registros
    ]
    return markdown_table(
        ["Nome", "CPF", "Processo", "Julgamento", "Cargo Público"],
        rows,
    )
