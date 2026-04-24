"""Canned SQL query tools for anac_vra dataset."""

from __future__ import annotations

from typing import Any

from fastmcp import Context

from mcp_brasil._shared.datasets import executar_query, get_status
from mcp_brasil._shared.formatting import format_number_br, markdown_table

from . import DATASET_SPEC, DATASET_TABLE


async def info_anac_vra(ctx: Context) -> str:
    """Estado do cache local do dataset ANAC VRA.

    Returns:
        Métricas do cache (linhas, tamanho, frescor).
    """
    await ctx.info("Consultando estado do cache ANAC VRA...")
    st = await get_status(DATASET_SPEC)
    return (
        "**ANAC VRA (Voo Regular Ativo) — cache**\n\n"
        f"- Cached: {'sim' if st['cached'] else 'não'}\n"
        f"- Linhas: {format_number_br(st['row_count'], 0) if st['cached'] else '—'}\n"
        f"- Tamanho: {st['size_bytes'] / 1024 / 1024:.1f} MB\n"
        f"- Fresh (TTL={st['ttl_days']}d): {'sim' if st['fresh'] else 'não'}\n"
    )


async def voos_empresa(
    ctx: Context,
    icao_empresa: str,
    limite: int = 50,
) -> str:
    """Lista voos operados por uma companhia (pelo ICAO da empresa).

    Args:
        icao_empresa: Código ICAO da companhia (ex: GLO=GOL, TAM=LATAM,
            AZU=Azul, VRG=Gol antigo, PTB=Passaredo).
        limite: Máx linhas (padrão 50, máximo 500).

    Returns:
        Tabela número voo, origem/destino, situação.
    """
    limite = max(1, min(limite, 500))
    needle = icao_empresa.upper().strip()
    await ctx.info(f"Buscando voos de {needle}...")
    sql = (
        "SELECT icao_empresa_aerea, numero_voo, "
        "icao_aerodromo_origem, icao_aerodromo_destino, "
        "partida_prevista, partida_real, chegada_prevista, chegada_real, "
        "situacao_voo "
        f'FROM "{DATASET_TABLE}" '
        "WHERE UPPER(icao_empresa_aerea) = ? "
        f"LIMIT {limite}"
    )
    rows = await executar_query(DATASET_SPEC, sql, [needle])
    if not rows:
        return f"Nenhum voo da empresa '{icao_empresa}'."
    table = [
        (
            r.get("numero_voo") or "—",
            r.get("icao_aerodromo_origem") or "—",
            r.get("icao_aerodromo_destino") or "—",
            (r.get("partida_prevista") or "—")[:16],
            (r.get("partida_real") or "—")[:16],
            (r.get("situacao_voo") or "—")[:18],
        )
        for r in rows
    ]
    return f"**Voos de {icao_empresa}** — {len(rows)} registro(s)\n\n" + markdown_table(
        ["Voo", "Origem", "Destino", "Prev.Saída", "Real.Saída", "Situação"],
        table,
    )


async def pontualidade_aeroporto(
    ctx: Context,
    icao_aerodromo: str,
    limite: int = 30,
) -> str:
    """Análise de pontualidade dos voos de um aeroporto.

    Conta voos com situação REALIZADO, CANCELADO ou outras.

    Args:
        icao_aerodromo: Código ICAO do aeroporto (ex: SBGR, SBKP, SBGL).
        limite: Máx situações (padrão 30).

    Returns:
        Tabela situação x ocorrências no mês/dataset.
    """
    limite = max(1, min(limite, 100))
    needle = icao_aerodromo.upper().strip()
    await ctx.info(f"Pontualidade em {needle}...")
    sql = (
        "SELECT situacao_voo, COUNT(*) AS n "
        f'FROM "{DATASET_TABLE}" '
        "WHERE UPPER(icao_aerodromo_origem) = ? OR UPPER(icao_aerodromo_destino) = ? "
        "GROUP BY situacao_voo "
        f"ORDER BY n DESC LIMIT {limite}"
    )
    rows = await executar_query(DATASET_SPEC, sql, [needle, needle])
    if not rows:
        return f"Sem voos registrados para {icao_aerodromo}."
    total = sum(int(r.get("n") or 0) for r in rows)
    body = []
    for r in rows:
        n = int(r.get("n") or 0)
        pct = 100 * n / total if total else 0
        body.append(
            ((r.get("situacao_voo") or "(null)")[:35], format_number_br(n, 0), f"{pct:.1f}%")
        )
    return (
        f"**Pontualidade {icao_aerodromo}** — {format_number_br(total, 0)} voo(s)\n\n"
        + markdown_table(["Situação", "Voos", "%"], body)
    )


async def top_rotas(ctx: Context, top: int = 20) -> str:
    """Rotas mais operadas (origem→destino).

    Args:
        top: Quantidade (padrão 20, máximo 100).

    Returns:
        Tabela origem x destino x voos.
    """
    top = max(1, min(top, 100))
    await ctx.info(f"Top {top} rotas...")
    sql = (
        "SELECT icao_aerodromo_origem AS orig, "
        "icao_aerodromo_destino AS dest, "
        "COUNT(*) AS n "
        f'FROM "{DATASET_TABLE}" '
        "WHERE icao_aerodromo_origem IS NOT NULL "
        "AND icao_aerodromo_destino IS NOT NULL "
        "GROUP BY orig, dest "
        f"ORDER BY n DESC LIMIT {top}"
    )
    rows = await executar_query(DATASET_SPEC, sql)
    if not rows:
        return "Sem dados."
    body = [
        (
            r.get("orig") or "—",
            r.get("dest") or "—",
            format_number_br(int(r.get("n") or 0), 0),
        )
        for r in rows
    ]
    return f"**ANAC VRA — Top {len(rows)} rotas**\n\n" + markdown_table(
        ["Origem", "Destino", "Voos"], body
    )


async def ranking_empresas(ctx: Context, top: int = 20) -> str:
    """Ranking de companhias por volume de voos.

    Args:
        top: Quantidade (padrão 20, máximo 50).

    Returns:
        Tabela empresa x voos.
    """
    top = max(1, min(top, 50))
    await ctx.info(f"Top {top} empresas...")
    sql = (
        "SELECT icao_empresa_aerea AS emp, COUNT(*) AS n "
        f'FROM "{DATASET_TABLE}" '
        "WHERE icao_empresa_aerea IS NOT NULL "
        "GROUP BY emp "
        f"ORDER BY n DESC LIMIT {top}"
    )
    rows = await executar_query(DATASET_SPEC, sql)
    if not rows:
        return "Sem dados."
    body = [(r.get("emp") or "—", format_number_br(int(r.get("n") or 0), 0)) for r in rows]
    return f"**ANAC VRA — Top {len(rows)} companhias**\n\n" + markdown_table(
        ["ICAO Empresa", "Voos"], body
    )


async def top_justificativas(ctx: Context, top: int = 15) -> str:
    """Principais códigos de justificativa para atrasos/cancelamentos.

    Args:
        top: Quantidade (padrão 15, máximo 50).

    Returns:
        Tabela código x ocorrências.
    """
    top = max(1, min(top, 50))
    await ctx.info(f"Top {top} justificativas...")
    sql = (
        "SELECT codigo_justificativa AS cod, COUNT(*) AS n "
        f'FROM "{DATASET_TABLE}" '
        "WHERE codigo_justificativa IS NOT NULL "
        "AND LENGTH(TRIM(codigo_justificativa)) > 0 "
        "GROUP BY cod "
        f"ORDER BY n DESC LIMIT {top}"
    )
    rows: list[dict[str, Any]] = await executar_query(DATASET_SPEC, sql)
    if not rows:
        return "Sem justificativas no dataset."
    body = [(r.get("cod") or "—", format_number_br(int(r.get("n") or 0), 0)) for r in rows]
    return f"**ANAC VRA — Top {len(rows)} códigos de justificativa**\n\n" + markdown_table(
        ["Código", "Ocorrências"], body
    )
