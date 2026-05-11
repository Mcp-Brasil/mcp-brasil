"""HTTP client for the ComexStat feature.

Endpoints:
    - POST /general              → _query (export/import data)
    - GET  /general/filters/country   → listar_paises
"""

from __future__ import annotations

import asyncio
from typing import Any

from curl_cffi.requests import AsyncSession

from mcp_brasil._shared.http_client import http_get
from mcp_brasil._shared.rate_limiter import RateLimiter

from .constants import (
    COUNTRY_URL,
    FLOW_EXPORT,
    FLOW_IMPORT,
    METRIC_FOB,
    METRIC_KG,
    QUERY_URL,
    REQUEST_HEADERS,
)
from .schemas import BalancaItem, ComexItem, PaisItem

_rate_limiter = RateLimiter(max_requests=30, period=60.0)


def _build_periodo(row: dict[str, Any], month_detail: bool) -> str:
    """Extract a YYYY-MM or YYYY period string from a ComexStat row."""
    year = str(row.get("year", ""))
    if month_detail:
        month = str(row.get("monthNumber", "")).zfill(2)
        return f"{year}-{month}" if year else ""
    return year


async def _query(
    flow: str,
    period_from: str,
    period_to: str,
    details: list[str],
    filters: list[dict[str, Any]],
    metrics: list[str],
    month_detail: bool,
) -> list[dict[str, Any]]:
    """POST to ComexStat query endpoint and return the data list.

    Uses curl_cffi to impersonate Chrome TLS fingerprint, bypassing
    Cloudflare WAF which blocks non-browser TLS on POST /general.
    Retries once after 12s on HTTP 429 (API rate limit window is 10s).
    """
    body: dict[str, Any] = {
        "flow": flow,
        "period": {"from": period_from, "to": period_to},
        "details": details,
        "filters": filters,
        "metrics": metrics,
        "monthDetail": month_detail,
    }
    async with _rate_limiter, AsyncSession(impersonate="chrome124") as session:
        for attempt in range(2):
            response = await session.post(
                QUERY_URL,
                json=body,
                headers=REQUEST_HEADERS,
                timeout=30,
            )
            if response.status_code == 429 and attempt == 0:
                await asyncio.sleep(12)
                continue
            response.raise_for_status()
            break
        raw: dict[str, Any] = response.json()
    data: dict[str, Any] = raw.get("data") or {}
    return list(data.get("list") or [])


def _rows_to_comex(rows: list[dict[str, Any]], month_detail: bool) -> list[ComexItem]:
    return [
        ComexItem(
            periodo=_build_periodo(row, month_detail),
            pais=row.get("country") or None,
            ncm=row.get("ncm") or None,
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
    """Compute trade balance (exports FOB - imports FOB) per period."""
    filters: list[dict[str, Any]] = []
    if ncm:
        filters.append({"filter": "ncm", "values": [ncm]})
    # Sequential with delay — concurrent POSTs trigger API 429 rate limit
    exp_rows = await _query(
        FLOW_EXPORT, periodo_inicio, periodo_fim, [], filters, [METRIC_FOB], True
    )
    await asyncio.sleep(3)
    imp_rows = await _query(
        FLOW_IMPORT, periodo_inicio, periodo_fim, [], filters, [METRIC_FOB], True
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
        raw: dict[str, Any] = await http_get(COUNTRY_URL, headers=REQUEST_HEADERS)
    # API returns {"data": [[{id, text}, ...]]} — outer list wraps one inner list
    data_raw: list[Any] = raw.get("data") or []
    items: list[dict[str, Any]] = data_raw[0] if data_raw else []
    return [
        PaisItem(
            id=str(item.get("id", "")),
            nome=item.get("text", ""),
        )
        for item in items
    ]
