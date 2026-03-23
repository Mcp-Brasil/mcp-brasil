"""Tool functions for the TCE-SP feature.

Rules (ADR-001):
    - tools.py NEVER makes HTTP directly — delegates to client.py
    - Returns formatted strings for LLM consumption
    - Uses Context for structured logging and progress reporting
"""

from __future__ import annotations

from fastmcp import Context

from mcp_brasil._shared.formatting import format_brl

from . import client


async def listar_municipios_sp(ctx: Context) -> str:
    """Lista os 645 municípios paulistas sob jurisdição do TCE-SP.

    Retorna nome completo e slug (usado como parâmetro nas demais tools).
    Dados estáticos — não muda entre consultas.

    Args:
        ctx: Contexto MCP.

    Returns:
        Lista de municípios com slug e nome completo.
    """
    await ctx.info("Listando municípios do TCE-SP...")
    municipios = await client.listar_municipios()

    if not municipios:
        return "Nenhum município encontrado no TCE-SP."

    lines: list[str] = [f"**{len(municipios)} municípios no TCE-SP:**\n"]
    for m in municipios[:50]:
        lines.append(f"- **{m.municipio_extenso}** (`{m.municipio}`)")

    if len(municipios) > 50:
        lines.append(f"\n*Mostrando 50 de {len(municipios)} municípios.*")
    return "\n".join(lines)


async def consultar_despesas_sp(
    ctx: Context,
    municipio: str,
    exercicio: int,
    mes: int,
) -> str:
    """Consulta despesas de um município paulista em um mês específico.

    Retorna empenhos, pagamentos e anulações com fornecedor e valor.
    Dados do sistema Audesp do TCE-SP desde 2014.

    Args:
        ctx: Contexto MCP.
        municipio: Slug do município (ex: "campinas", "sao-paulo").
            Use listar_municipios_sp para obter slugs válidos.
        exercicio: Ano fiscal (2014 a atual, ex: 2025).
        mes: Mês (1 a 12).

    Returns:
        Lista de despesas com fornecedor, evento e valor.
    """
    await ctx.info(f"Buscando despesas de {municipio} ({exercicio}/{mes})...")
    despesas = await client.buscar_despesas(municipio, exercicio, mes)

    if not despesas:
        return f"Nenhuma despesa encontrada para {municipio} em {mes}/{exercicio}."

    total_valor = sum(d.vl_despesa for d in despesas if d.vl_despesa and d.vl_despesa > 0)
    lines: list[str] = [
        f"**{len(despesas)} registros de despesas em {municipio} ({mes}/{exercicio}):**\n"
    ]
    if total_valor:
        lines.append(f"**Total empenhado (positivo):** {format_brl(total_valor)}\n")

    for d in despesas[:30]:
        valor = format_brl(d.vl_despesa) if d.vl_despesa else "—"
        lines.append(f"- [{d.evento or '—'}] {d.nm_fornecedor or '—'}: {valor}")
        if d.nr_empenho:
            lines[-1] += f" (empenho {d.nr_empenho})"

    if len(despesas) > 30:
        lines.append(f"\n*Mostrando 30 de {len(despesas)} registros.*")
    return "\n".join(lines)


async def consultar_receitas_sp(
    ctx: Context,
    municipio: str,
    exercicio: int,
    mes: int,
) -> str:
    """Consulta receitas de um município paulista em um mês específico.

    Retorna arrecadação por fonte de recurso, alinea e subalinea.
    Dados do sistema Audesp do TCE-SP desde 2014.

    Args:
        ctx: Contexto MCP.
        municipio: Slug do município (ex: "campinas", "sao-paulo").
            Use listar_municipios_sp para obter slugs válidos.
        exercicio: Ano fiscal (2014 a atual, ex: 2025).
        mes: Mês (1 a 12).

    Returns:
        Lista de receitas com fonte, classificação e valor.
    """
    await ctx.info(f"Buscando receitas de {municipio} ({exercicio}/{mes})...")
    receitas = await client.buscar_receitas(municipio, exercicio, mes)

    if not receitas:
        return f"Nenhuma receita encontrada para {municipio} em {mes}/{exercicio}."

    total_arrecadado = sum(r.vl_arrecadacao for r in receitas if r.vl_arrecadacao)
    lines: list[str] = [
        f"**{len(receitas)} registros de receitas em {municipio} ({mes}/{exercicio}):**\n"
    ]
    if total_arrecadado:
        lines.append(f"**Total arrecadado:** {format_brl(total_arrecadado)}\n")

    for r in receitas[:30]:
        valor = format_brl(r.vl_arrecadacao) if r.vl_arrecadacao else "—"
        alinea = r.ds_alinea or "—"
        fonte = r.ds_fonte_recurso or "—"
        lines.append(f"- **{alinea}**: {valor}")
        lines.append(f"  Fonte: {fonte}")

    if len(receitas) > 30:
        lines.append(f"\n*Mostrando 30 de {len(receitas)} registros.*")
    return "\n".join(lines)
