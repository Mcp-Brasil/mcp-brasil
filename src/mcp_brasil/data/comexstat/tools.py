"""Tool functions for the ComexStat feature.

Rules (ADR-001):
    - tools.py NEVER makes HTTP directly — delegates to client.py
    - Returns formatted strings for LLM consumption
    - Uses Context for structured logging and progress reporting
"""

from __future__ import annotations

from fastmcp import Context

from mcp_brasil._shared.formatting import format_number_br, markdown_table

from . import client


def _fmt_usd(v: float) -> str:
    return f"US$ {format_number_br(v)}"


def _fmt_kg(v: float) -> str:
    return f"{format_number_br(v)} kg"


async def consultar_exportacoes(
    periodo_inicio: str,
    periodo_fim: str,
    ctx: Context,
    ncm: str | None = None,
    pais: str | None = None,
    detalhar_por_mes: bool = False,
) -> str:
    """Query Brazilian export statistics from ComexStat (MDIC).

    Returns FOB value (USD) and net weight (kg) per period, optionally broken
    down by country and/or NCM code. Data covers monthly records since 1997.
    Source: ComexStat MDIC — no authentication required.

    Args:
        periodo_inicio: Start period in YYYY-MM format (e.g. "2023-01").
        periodo_fim: End period in YYYY-MM format (e.g. "2023-12").
        ncm: Optional 8-digit NCM code to filter by product.
        pais: Optional country ID to filter by trading partner (use listar_paises).
        detalhar_por_mes: If True, return monthly rows; if False, aggregate by period.

    Returns:
        Markdown table with export records.
    """
    await ctx.info(f"Consultando exportações ComexStat: {periodo_inicio} a {periodo_fim}...")
    itens = await client.consultar_exportacoes(
        periodo_inicio, periodo_fim, ncm=ncm, pais=pais, detalhar_por_mes=detalhar_por_mes
    )
    if not itens:
        return "Nenhum dado de exportação encontrado para os filtros informados."
    await ctx.info(f"{len(itens)} registro(s) encontrado(s).")
    rows = [
        (
            item.periodo,
            item.pais or "—",
            item.ncm or "—",
            _fmt_usd(item.fob_usd),
            _fmt_kg(item.kg_liquido),
        )
        for item in itens
    ]
    header = (
        f"**Exportações brasileiras** — {periodo_inicio} a {periodo_fim}: "
        f"{len(itens)} registro(s)\n"
    )
    return header + markdown_table(["Período", "País", "NCM", "FOB (USD)", "Peso líq. (kg)"], rows)


async def consultar_importacoes(
    periodo_inicio: str,
    periodo_fim: str,
    ctx: Context,
    ncm: str | None = None,
    pais: str | None = None,
    detalhar_por_mes: bool = False,
) -> str:
    """Query Brazilian import statistics from ComexStat (MDIC).

    Returns FOB value (USD) and net weight (kg) per period, optionally broken
    down by country and/or NCM code. Data covers monthly records since 1997.
    Source: ComexStat MDIC — no authentication required.

    Args:
        periodo_inicio: Start period in YYYY-MM format (e.g. "2023-01").
        periodo_fim: End period in YYYY-MM format (e.g. "2023-12").
        ncm: Optional 8-digit NCM code to filter by product.
        pais: Optional country ID to filter by trading partner (use listar_paises).
        detalhar_por_mes: If True, return monthly rows; if False, aggregate by period.

    Returns:
        Markdown table with import records.
    """
    await ctx.info(f"Consultando importações ComexStat: {periodo_inicio} a {periodo_fim}...")
    itens = await client.consultar_importacoes(
        periodo_inicio, periodo_fim, ncm=ncm, pais=pais, detalhar_por_mes=detalhar_por_mes
    )
    if not itens:
        return "Nenhum dado de importação encontrado para os filtros informados."
    await ctx.info(f"{len(itens)} registro(s) encontrado(s).")
    rows = [
        (
            item.periodo,
            item.pais or "—",
            item.ncm or "—",
            _fmt_usd(item.fob_usd),
            _fmt_kg(item.kg_liquido),
        )
        for item in itens
    ]
    header = (
        f"**Importações brasileiras** — {periodo_inicio} a {periodo_fim}: "
        f"{len(itens)} registro(s)\n"
    )
    return header + markdown_table(["Período", "País", "NCM", "FOB (USD)", "Peso líq. (kg)"], rows)


async def balanca_comercial(
    periodo_inicio: str,
    periodo_fim: str,
    ctx: Context,
    ncm: str | None = None,
) -> str:
    """Compute monthly trade balance (exports FOB - imports FOB) from ComexStat.

    Fetches export and import data in parallel and returns the net balance
    per month. A positive saldo indicates a trade surplus; negative is a deficit.
    Source: ComexStat MDIC — no authentication required.

    Args:
        periodo_inicio: Start period in YYYY-MM format (e.g. "2023-01").
        periodo_fim: End period in YYYY-MM format (e.g. "2023-12").
        ncm: Optional 8-digit NCM code to restrict the balance to one product.

    Returns:
        Markdown table with export FOB, import FOB and trade balance per period.
    """
    await ctx.info(f"Calculando balança comercial: {periodo_inicio} a {periodo_fim}...")
    itens = await client.balanca_comercial(periodo_inicio, periodo_fim, ncm=ncm)
    if not itens:
        return "Nenhum dado de balança comercial encontrado para os filtros informados."
    rows = [
        (
            item.periodo,
            _fmt_usd(item.exportacoes_fob),
            _fmt_usd(item.importacoes_fob),
            ("+" if item.saldo_fob >= 0 else "") + _fmt_usd(item.saldo_fob),
        )
        for item in itens
    ]
    header = f"**Balança Comercial** — {periodo_inicio} a {periodo_fim}: {len(itens)} período(s)\n"
    return header + markdown_table(
        ["Período", "Exportações FOB", "Importações FOB", "Saldo FOB"], rows
    )


async def listar_paises(ctx: Context) -> str:
    """List all countries available in ComexStat trade statistics.

    Returns country IDs and names (in Portuguese). Use the ID to filter
    consultar_exportacoes or consultar_importacoes by trading partner.
    Source: ComexStat MDIC — no authentication required.

    Returns:
        Markdown table with country ID and name.
    """
    await ctx.info("Listando países disponíveis no ComexStat...")
    paises = await client.listar_paises()
    if not paises:
        return "Nenhum país encontrado."
    rows = [(p.id, p.nome) for p in paises]
    return f"**Países disponíveis no ComexStat:** {len(paises)}\n" + markdown_table(
        ["ID", "Nome"], rows
    )
