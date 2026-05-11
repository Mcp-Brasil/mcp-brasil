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


async def _fetch_all() -> list[NcmItem]:
    """Fetch and cache the full NCM nomenclature table (TTL=24h).

    Concurrent initial fetches are benign in asyncio (idempotent result).
    """
    global _cache, _cache_expiry
    if _cache is not None and time.monotonic() < _cache_expiry:
        return _cache
    async with _rate_limiter:
        data: list[dict[str, Any]] = await http_get(NCM_BASE_URL, params=NCM_PARAMS)
    _cache = [NcmItem(**item) for item in data]
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
