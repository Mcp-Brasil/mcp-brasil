"""HTTP client for the ComexStat feature.

Endpoints:
    - POST /general/query              → _query (export/import data)
    - GET  /general/filters/country   → listar_paises
"""

from __future__ import annotations

import asyncio
from typing import Any

from mcp_brasil._shared.http_client import http_get, http_post
from mcp_brasil._shared.rate_limiter import RateLimiter

from .constants import COUNTRY_URL, FLOW_EXPORT, FLOW_IMPORT, METRIC_FOB, METRIC_KG, QUERY_URL
from .schemas import BalancaItem, ComexItem, PaisItem

_rate_limiter = RateLimiter(max_requests=30, period=60.0)


def _build_periodo(row: dict[str, Any], month_detail: bool) -> str:
    """Extract a YYYY-MM or YYYY period string from a ComexStat row."""
    if month_detail:
        ano_mes = str(row.get("coAnoMes", ""))
        if len(ano_mes) == 6:
            return f"{ano_mes[:4]}-{ano_mes[4:]}"
        return ano_mes
    return str(row.get("coAno", ""))


async def _query(
    flow: str,
    period_from: str,
    period_to: str,
    details: list[str],
    filters: list[dict[str, Any]],
    metrics: list[str],
    month_detail: bool,
) -> list[dict[str, Any]]:
    """POST to ComexStat query endpoint and return the data list."""
    body: dict[str, Any] = {
        "flow": flow,
        "period": {"from": period_from, "to": period_to},
        "details": details,
        "filters": filters,
        "metrics": metrics,
        "monthDetail": month_detail,
    }
    async with _rate_limiter:
        raw: dict[str, Any] = await http_post(QUERY_URL, json_body=body)
    data: dict[str, Any] = raw.get("data") or {}
    return list(data.get("list") or [])


def _rows_to_comex(rows: list[dict[str, Any]], month_detail: bool) -> list[ComexItem]:
    return [
        ComexItem(
            periodo=_build_periodo(row, month_detail),
            pais=row.get("noPaisPt") or row.get("noPais") or None,
            ncm=row.get("coNcm") or None,
            fob_usd=float(row.get("metricFOB") or 0),
            kg_liquido=float(row.get("metricKG") or 0),
        )
        for row in rows
    ]


async def consultar_exportacoes(
    periodo_inicio: str,
    periodo_fim: str,
    ncm: str | None = None,
    pais: str | None = None,
    detalhar_por_mes: bool = False,
) -> list[ComexItem]:
    """Fetch Brazilian export statistics from ComexStat."""
    details: list[str] = []
    filters: list[dict[str, Any]] = []
    if ncm:
        details.append("ncm")
        filters.append({"filter": "ncm", "values": [ncm]})
    if pais:
        details.append("country")
        filters.append({"filter": "country", "values": [pais]})
    rows = await _query(
        FLOW_EXPORT,
        periodo_inicio,
        periodo_fim,
        details,
        filters,
        [METRIC_FOB, METRIC_KG],
        detalhar_por_mes,
    )
    return _rows_to_comex(rows, detalhar_por_mes)


async def consultar_importacoes(
    periodo_inicio: str,
    periodo_fim: str,
    ncm: str | None = None,
    pais: str | None = None,
    detalhar_por_mes: bool = False,
) -> list[ComexItem]:
    """Fetch Brazilian import statistics from ComexStat."""
    details: list[str] = []
    filters: list[dict[str, Any]] = []
    if ncm:
        details.append("ncm")
        filters.append({"filter": "ncm", "values": [ncm]})
    if pais:
        details.append("country")
        filters.append({"filter": "country", "values": [pais]})
    rows = await _query(
        FLOW_IMPORT,
        periodo_inicio,
        periodo_fim,
        details,
        filters,
        [METRIC_FOB, METRIC_KG],
        detalhar_por_mes,
    )
    return _rows_to_comex(rows, detalhar_por_mes)


async def balanca_comercial(
    periodo_inicio: str,
    periodo_fim: str,
    ncm: str | None = None,
) -> list[BalancaItem]:
    """Compute trade balance (exports FOB - imports FOB) per month."""
    filters: list[dict[str, Any]] = []
    if ncm:
        filters.append({"filter": "ncm", "values": [ncm]})
    exp_rows, imp_rows = await asyncio.gather(
        _query(FLOW_EXPORT, periodo_inicio, periodo_fim, [], filters, [METRIC_FOB], True),
        _query(FLOW_IMPORT, periodo_inicio, periodo_fim, [], filters, [METRIC_FOB], True),
    )

    exp_by: dict[str, float] = {}
    for row in exp_rows:
        p = _build_periodo(row, True)
        exp_by[p] = exp_by.get(p, 0.0) + float(row.get("metricFOB") or 0)

    imp_by: dict[str, float] = {}
    for row in imp_rows:
        p = _build_periodo(row, True)
        imp_by[p] = imp_by.get(p, 0.0) + float(row.get("metricFOB") or 0)

    all_periods = sorted(set(exp_by) | set(imp_by))
    return [
        BalancaItem(
            periodo=p,
            exportacoes_fob=exp_by.get(p, 0.0),
            importacoes_fob=imp_by.get(p, 0.0),
            saldo_fob=exp_by.get(p, 0.0) - imp_by.get(p, 0.0),
        )
        for p in all_periods
    ]


async def listar_paises() -> list[PaisItem]:
    """Fetch available countries from ComexStat filters."""
    async with _rate_limiter:
        raw: dict[str, Any] = await http_get(COUNTRY_URL)
    items: list[dict[str, Any]] = raw.get("data", {}).get("list", [])
    return [
        PaisItem(
            id=str(item.get("id_country", "")),
            nome=item.get("no_country_pt") or item.get("no_country_en", ""),
        )
        for item in items
    ]
