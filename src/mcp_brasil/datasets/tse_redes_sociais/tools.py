"""Canned SQL query tools for tse_redes_sociais dataset."""

from __future__ import annotations

from typing import Any

from fastmcp import Context

from mcp_brasil._shared.datasets import executar_query, get_status
from mcp_brasil._shared.formatting import format_number_br, markdown_table

from . import DATASET_SPEC, DATASET_TABLE

_REDES_PATTERNS: dict[str, tuple[str, ...]] = {
    "instagram": ("instagram.com", "instagr.am"),
    "facebook": ("facebook.com", "fb.com", "fb.me"),
    "twitter": ("twitter.com", "x.com"),
    "tiktok": ("tiktok.com", "tiktok"),
    "youtube": ("youtube.com", "youtu.be"),
    "kwai": ("kwai.com", "kwai.app"),
    "linkedin": ("linkedin.com", "lnkd.in"),
    "site": ("http://", "https://"),  # catch-all se nenhuma rede bate
}


def _classify_url(url: str | None) -> str:
    if not url:
        return "?"
    u = url.lower()
    for rede, patterns in _REDES_PATTERNS.items():
        if any(p in u for p in patterns):
            return rede
    return "outro"


async def info_tse_redes_sociais(ctx: Context) -> str:
    """Estado do cache local do dataset TSE redes sociais.

    Returns:
        Métricas do cache.
    """
    await ctx.info("Consultando estado do cache TSE redes sociais...")
    st = await get_status(DATASET_SPEC)
    return (
        "**TSE redes sociais 2018-2024 — cache**\n\n"
        f"- Cached: {'sim' if st['cached'] else 'não'}\n"
        f"- Linhas: {format_number_br(st['row_count'], 0) if st['cached'] else '—'}\n"
        f"- Tamanho: {st['size_bytes'] / 1024 / 1024:.1f} MB\n"
    )


async def redes_do_candidato(
    ctx: Context,
    sq_candidato: str,
) -> str:
    """URLs de redes sociais declaradas por um candidato.

    Args:
        sq_candidato: ID do candidato.

    Returns:
        Tabela com ano x rede x URL.
    """
    await ctx.info(f"Buscando redes sociais do candidato {sq_candidato}...")
    # Nota: coluna de ano é AA_ELEICAO (rede_social_candidato), não ANO_ELEICAO
    sql = (
        "SELECT aa_eleicao AS ano, "
        "ds_url "
        f'FROM "{DATASET_TABLE}" '
        "WHERE CAST(sq_candidato AS VARCHAR) = ? "
        "ORDER BY ano DESC"
    )
    rows = await executar_query(DATASET_SPEC, sql, [str(sq_candidato).strip()])
    if not rows:
        return f"Nenhuma rede social declarada por sq_candidato={sq_candidato!r}."
    table = [
        (
            str(r.get("ano") or "—"),
            _classify_url(r.get("ds_url")),
            (r.get("ds_url") or "—")[:100],
        )
        for r in rows
    ]
    return (
        f"**Redes sociais do candidato {sq_candidato}** — {len(rows)} URL(s)\n\n"
        + markdown_table(["Ano", "Rede", "URL"], table)
    )


async def redes_por_partido(
    ctx: Context,
    partido: str,
    ano: int | None = None,
    limite: int = 50,
) -> str:
    """Distribuição de redes sociais usadas por candidatos de um partido.

    Args:
        partido: Sigla.
        ano: Ano eleitoral opcional (2018, 2020, 2022, 2024).
        limite: Máx linhas no detalhe.

    Returns:
        Duas tabelas: distribuição por rede + amostra de candidatos.
        **Requer tse_candidatos ativo** (para join e filtro por partido).
    """
    limite = max(1, min(limite, 200))
    await ctx.info(f"Redes por partido — {partido} ({ano or 'todos'})...")
    extra = ""
    extra_params: list[Any] = []
    if ano is not None:
        extra = " AND CAST(r.aa_eleicao AS INTEGER) = ?"
        extra_params = [int(ano)]
    sql = (
        "SELECT ds_url "
        f'FROM "{DATASET_TABLE}" r '
        'JOIN "candidatos" c USING (sq_candidato) '
        "WHERE UPPER(c.sg_partido) = ? "
        f"{extra} LIMIT {limite}"
    )
    try:
        rows = await _execute_with_candidatos(sql, [partido.strip().upper(), *extra_params])
    except Exception as exc:
        return (
            f"ERRO ao juntar com tse_candidatos ({type(exc).__name__}): {exc}\n"
            "Certifique-se que `tse_candidatos` está em MCP_BRASIL_DATASETS."
        )
    if not rows:
        return f"Sem redes sociais para {partido} (ano={ano})."

    # Agrega contagem por rede
    counts: dict[str, int] = {}
    for r in rows:
        rede = _classify_url(r.get("ds_url"))
        counts[rede] = counts.get(rede, 0) + 1
    distrib = sorted(counts.items(), key=lambda x: x[1], reverse=True)
    distrib_table = [(rede, format_number_br(n, 0)) for rede, n in distrib]

    amostra = [((r.get("ds_url") or "")[:100],) for r in rows[:15]]

    return (
        f"**Redes sociais — {partido} ({ano or 'todos os anos'})**\n\n"
        f"### Distribuição por plataforma ({len(rows)} URLs amostradas)\n\n"
        + markdown_table(["Rede", "URLs"], distrib_table)
        + "\n\n### Amostra de URLs\n\n"
        + markdown_table(["URL"], amostra)
    )


async def top_redes_por_ano(ctx: Context, ano: int) -> str:
    """Contagem total de URLs por plataforma num ano eleitoral.

    Args:
        ano: Ano eleitoral.

    Returns:
        Tabela com rede x URLs x candidatos distintos.
    """
    await ctx.info(f"Top redes {ano}...")
    sql = (
        f'SELECT ds_url, sq_candidato FROM "{DATASET_TABLE}" WHERE CAST(aa_eleicao AS INTEGER) = ?'
    )
    rows = await executar_query(DATASET_SPEC, sql, [int(ano)])
    if not rows:
        return f"Sem dados para {ano}."
    counts: dict[str, int] = {}
    cands: dict[str, set[str]] = {}
    for r in rows:
        rede = _classify_url(r.get("ds_url"))
        counts[rede] = counts.get(rede, 0) + 1
        cands.setdefault(rede, set()).add(str(r.get("sq_candidato") or ""))
    distrib = sorted(counts.items(), key=lambda x: x[1], reverse=True)
    body = [
        (
            rede,
            format_number_br(n, 0),
            format_number_br(len(cands[rede]), 0),
        )
        for rede, n in distrib
    ]
    return f"**TSE redes sociais — {ano}**\n\n" + markdown_table(
        ["Rede", "URLs", "Candidatos distintos"], body
    )


async def _execute_with_candidatos(sql: str, params: list[Any]) -> list[dict[str, Any]]:
    """Execute a query that joins redes_sociais (this DB) with candidatos (attached).

    Requires tse_candidatos to be enabled and cached.
    """
    import asyncio

    import duckdb

    from mcp_brasil._shared.datasets.cache import db_path
    from mcp_brasil._shared.datasets.loader import ensure_loaded

    await ensure_loaded(DATASET_SPEC)
    from mcp_brasil.datasets.tse_candidatos import DATASET_SPEC as CAND_SPEC

    await ensure_loaded(CAND_SPEC)

    this_path = str(db_path(DATASET_SPEC.id))
    cand_path = str(db_path(CAND_SPEC.id))

    def _run() -> list[dict[str, Any]]:
        con = duckdb.connect(this_path, read_only=True)
        try:
            con.execute(f"ATTACH '{cand_path}' AS cand (READ_ONLY)")
            sql_rewritten = sql.replace('"candidatos"', 'cand."candidatos"')
            cursor = con.execute(sql_rewritten, params)
            cols = [c[0] for c in cursor.description] if cursor.description else []
            return [dict(zip(cols, row, strict=False)) for row in cursor.fetchall()]
        finally:
            con.close()

    return await asyncio.to_thread(_run)
