"""Client for the OpenDataSUS feature.

API: CKAN standard at https://opendatasus.saude.gov.br/api/3/action
Auth: None required.
"""

from __future__ import annotations

import json
from typing import Any

from mcp_brasil._shared.http_client import http_get

from .constants import (
    DATASTORE_SEARCH_URL,
    DEFAULT_LIMIT,
    MAX_LIMIT,
    PACKAGE_SEARCH_URL,
    PACKAGE_SHOW_URL,
)
from .schemas import DatasetOpenDataSUS, RecursoDataset, RegistroDataStore


def _extract_result(data: Any) -> dict[str, Any]:
    """Extract the 'result' key from a CKAN response."""
    if isinstance(data, dict):
        result = data.get("result", data)
        if isinstance(result, dict):
            return result
        return data
    return {}


def _parse_recurso(raw: dict[str, Any]) -> RecursoDataset:
    """Parse a CKAN resource dict into a RecursoDataset."""
    return RecursoDataset(
        id=str(raw.get("id", "")),
        nome=raw.get("name") or raw.get("title"),
        formato=raw.get("format"),
        url=raw.get("url"),
        descricao=raw.get("description"),
    )


def _parse_dataset(raw: dict[str, Any]) -> DatasetOpenDataSUS:
    """Parse a CKAN package dict into a DatasetOpenDataSUS."""
    recursos_raw = raw.get("resources", [])
    recursos = [_parse_recurso(r) for r in recursos_raw]
    tags = [t.get("display_name", t.get("name", "")) for t in raw.get("tags", [])]
    org = raw.get("organization")
    org_name = org.get("title", org.get("name", "")) if isinstance(org, dict) else None

    return DatasetOpenDataSUS(
        id=str(raw.get("id", "")),
        nome=raw.get("name", ""),
        titulo=raw.get("title"),
        descricao=raw.get("notes"),
        organizacao=org_name,
        tags=tags,
        recursos=recursos,
        total_recursos=len(recursos),
        data_criacao=raw.get("metadata_created"),
        data_atualizacao=raw.get("metadata_modified"),
    )


async def buscar_datasets(
    query: str,
    limite: int = DEFAULT_LIMIT,
) -> tuple[list[DatasetOpenDataSUS], int]:
    """Search datasets on OpenDataSUS by keyword.

    Args:
        query: Search keyword.
        limite: Max results (default 20).

    Returns:
        Tuple of (datasets, total_count).
    """
    rows = min(limite, MAX_LIMIT)
    params: dict[str, str] = {"q": query, "rows": str(rows)}
    data = await http_get(PACKAGE_SEARCH_URL, params=params)
    result = _extract_result(data)
    datasets = [_parse_dataset(d) for d in result.get("results", [])]
    total = result.get("count", len(datasets))
    return datasets, total


async def detalhar_dataset(dataset_id: str) -> DatasetOpenDataSUS | None:
    """Get details of a specific dataset by name or ID.

    Args:
        dataset_id: Dataset name (slug) or UUID.

    Returns:
        Dataset details or None if not found.
    """
    params: dict[str, str] = {"id": dataset_id}
    data = await http_get(PACKAGE_SHOW_URL, params=params)
    result = _extract_result(data)
    if not result or not result.get("id"):
        return None
    return _parse_dataset(result)


async def consultar_datastore(
    resource_id: str,
    query: str | None = None,
    filtros: dict[str, str] | None = None,
    limite: int = DEFAULT_LIMIT,
    offset: int = 0,
) -> tuple[list[RegistroDataStore], int]:
    """Query records from a CKAN DataStore resource.

    Args:
        resource_id: UUID of the CKAN resource.
        query: Full-text search across all fields.
        filtros: Field-value filters (exact match).
        limite: Max records (default 20).
        offset: Pagination offset.

    Returns:
        Tuple of (records, total_count).
    """
    rows = min(limite, MAX_LIMIT)
    params: dict[str, str] = {
        "resource_id": resource_id,
        "limit": str(rows),
        "offset": str(offset),
    }
    if query:
        params["q"] = query
    if filtros:
        params["filters"] = json.dumps(filtros)

    data = await http_get(DATASTORE_SEARCH_URL, params=params)
    result = _extract_result(data)
    records_raw = result.get("records", [])
    records = [
        RegistroDataStore(campos={k: v for k, v in r.items() if k != "_id"}) for r in records_raw
    ]
    total = result.get("total", len(records))
    return records, total
