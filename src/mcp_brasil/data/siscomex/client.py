"""HTTP client for the Siscomex feature.

Endpoints:
    - GET /classif/api/publico/nomenclatura/download/json?perfil=PUBLICO → _fetch_all
"""

from __future__ import annotations

import time
from typing import Any

from mcp_brasil._shared.http_client import http_get
from mcp_brasil._shared.rate_limiter import RateLimiter

from .constants import CACHE_TTL_SECONDS, NCM_BASE_URL, NCM_MAX_RESULTS, NCM_PARAMS
from .schemas import NcmItem

# Cache (24h TTL) is the primary protection; rate limiter is a safety net
_rate_limiter = RateLimiter(max_requests=60, period=60.0)

_cache: list[NcmItem] | None = None
_cache_expiry: float = 0.0


def _parse_item(row: dict[str, Any]) -> NcmItem:
    """Map raw API fields (PascalCase/underscored) to NcmItem schema fields."""
    data_fim = row.get("Data_Fim")
    ano_raw = row.get("Ano_Ato_Ini")
    return NcmItem(
        codigo=row["Codigo"],
        descricao=row["Descricao"],
        dataInicio=row.get("Data_Inicio"),
        # "31/12/9999" is the API sentinel for "no expiry" — treat as None (active)
        dataFim=None if data_fim == "31/12/9999" else data_fim,
        tipoOrgaoAtoIni=row.get("Tipo_Ato_Ini"),
        numeroAtoIni=row.get("Numero_Ato_Ini"),
        anoAtoIni=int(ano_raw) if ano_raw is not None else None,
    )


async def _fetch_all() -> list[NcmItem]:
    """Fetch and cache the full NCM nomenclature table (TTL=24h).

    Concurrent initial fetches are benign in asyncio (idempotent result).
    """
    global _cache, _cache_expiry
    if _cache is not None and time.monotonic() < _cache_expiry:
        return _cache
    async with _rate_limiter:
        data: dict[str, Any] = await http_get(NCM_BASE_URL, params=NCM_PARAMS)
    nomenclaturas: list[dict[str, Any]] = data["Nomenclaturas"]
    _cache = [_parse_item(row) for row in nomenclaturas]
    _cache_expiry = time.monotonic() + CACHE_TTL_SECONDS
    return _cache


async def buscar_ncm(query: str, apenas_ativos: bool = True) -> list[NcmItem]:
    """Search NCM codes by partial code or description keyword."""
    todos = await _fetch_all()
    if apenas_ativos:
        todos = [item for item in todos if item.dataFim is None]
    q = query.lower().strip()
    resultados = [item for item in todos if q in item.codigo or q in item.descricao.lower()]
    return resultados[:NCM_MAX_RESULTS]


async def consultar_ncm(codigo: str) -> NcmItem | None:
    """Fetch a specific NCM code by exact 8-digit match."""
    todos = await _fetch_all()
    clean = codigo.strip()
    return next((item for item in todos if item.codigo == clean), None)
