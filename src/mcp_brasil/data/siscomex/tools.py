"""Tool functions for the Siscomex feature.

Rules (ADR-001):
    - tools.py NEVER makes HTTP directly — delegates to client.py
    - Returns formatted strings for LLM consumption
    - Uses Context for structured logging and progress reporting
"""

from __future__ import annotations

from fastmcp import Context

from mcp_brasil._shared.formatting import markdown_table

from . import client


async def buscar_ncm(query: str, ctx: Context, apenas_ativos: bool = True) -> str:
    """Search official NCM codes by partial code or description (Siscomex source).

    Returns up to 50 results from the Nomenclatura Comum do Mercosul (NCM)
    table downloaded directly from Portal Único Siscomex — official government source.

    Unlike BrasilAPI, provides official validity tracking (dataInicio/dataFim)
    and legal act references required for regulatory compliance.

    Args:
        query: Partial NCM code (e.g. "0103") or description keyword (e.g. "bovino").
        apenas_ativos: If True (default), return only codes with no end date (currently active).

    Returns:
        Markdown table with matching NCM codes (up to 50 results).
    """
    await ctx.info(f"Buscando NCM Siscomex: '{query}' (apenas_ativos={apenas_ativos})...")
    resultados = await client.buscar_ncm(query, apenas_ativos=apenas_ativos)
    if not resultados:
        return f"Nenhum código NCM encontrado para '{query}'."
    await ctx.info(f"{len(resultados)} resultado(s) encontrado(s).")
    rows = [
        (
            item.codigo,
            item.descricao[:80] + ("..." if len(item.descricao) > 80 else ""),
            item.dataInicio or "—",
            item.dataFim or "—",
        )
        for item in resultados
    ]
    header = "**NCM encontrados"
    if len(resultados) == 50:
        header += " (máximo 50 — refine a busca)"
    header += f":** {len(resultados)} resultado(s)\n"
    return header + markdown_table(
        ["Código", "Descrição", "Vigência início", "Vigência fim"], rows
    )


async def consultar_ncm(codigo: str, ctx: Context) -> str:
    """Fetch details of a specific NCM code from the official Siscomex table.

    Returns full description, validity period, and legal act references.
    Source: Portal Único Siscomex — authoritative government data updated daily.

    Args:
        codigo: 8-digit NCM code (e.g. "01031000").

    Returns:
        NCM code details or a 'not found' message.
    """
    clean = codigo.strip()
    await ctx.info(f"Consultando NCM Siscomex: {clean}...")
    item = await client.consultar_ncm(clean)
    if item is None:
        return f"Código NCM **{clean}** não encontrado na tabela Siscomex."
    ato = "—"
    if item.tipoOrgaoAtoIni and item.numeroAtoIni and item.anoAtoIni:
        ato = f"{item.tipoOrgaoAtoIni} nº {item.numeroAtoIni}/{item.anoAtoIni}"
    lines = [
        f"**Código NCM:** {item.codigo}",
        f"**Descrição:** {item.descricao}",
        f"**Vigência início:** {item.dataInicio or 'N/A'}",
        f"**Vigência fim:** {item.dataFim or 'Em vigor'}",
        f"**Última alteração:** {item.dataUltimaAlteracao or 'N/A'}",
        f"**Ato legal:** {ato}",
    ]
    return "\n".join(lines)
