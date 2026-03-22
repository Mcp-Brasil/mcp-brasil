"""HTTP client for the Querido Diário API.

Endpoints:
    - /gazettes?querystring=...        → buscar_diarios
    - /gazettes/{territory_id}?...     → buscar_diarios (por município)
    - /cities?city_name=...            → buscar_cidades
    - /cities                          → listar_cidades
"""

from __future__ import annotations

from typing import Any

from mcp_brasil._shared.http_client import http_get

from .constants import CITIES_URL, DEFAULT_PAGE_SIZE, GAZETTES_URL
from .schemas import CidadeQueridoDiario, DiarioOficial, DiarioResultado


async def buscar_diarios(
    querystring: str,
    territory_id: str | None = None,
    since: str | None = None,
    until: str | None = None,
    offset: int = 0,
    size: int = DEFAULT_PAGE_SIZE,
) -> DiarioResultado:
    """Search gazettes by keyword, optionally filtered by territory and date range."""
    url = f"{GAZETTES_URL}/{territory_id}" if territory_id else GAZETTES_URL

    params: dict[str, str] = {
        "querystring": querystring,
        "offset": str(offset),
        "size": str(size),
    }
    if since:
        params["since"] = since
    if until:
        params["until"] = until

    data: dict[str, Any] = await http_get(url, params=params)
    gazettes = [DiarioOficial(**g) for g in data.get("gazettes", [])]
    return DiarioResultado(
        total_gazettes=data.get("total_gazettes", len(gazettes)),
        gazettes=gazettes,
    )


async def buscar_cidades(nome_cidade: str) -> list[CidadeQueridoDiario]:
    """Search cities available in Querido Diário by name."""
    data: list[dict[str, Any]] = await http_get(CITIES_URL, params={"city_name": nome_cidade})
    return [CidadeQueridoDiario(**c) for c in data]


async def listar_cidades() -> list[CidadeQueridoDiario]:
    """List all cities available in Querido Diário."""
    data: list[dict[str, Any]] = await http_get(CITIES_URL)
    return [CidadeQueridoDiario(**c) for c in data]
