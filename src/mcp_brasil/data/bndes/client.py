"""HTTP client for the BNDES CKAN API.

Base URL: https://dadosabertos.bndes.gov.br/api/3/action
Auth: None required
CKAN response format: {"success": true, "result": ...}
DataStore pagination: limit/offset (default limit=100)
"""

from __future__ import annotations

import json
from typing import Any

from mcp_brasil._shared.http_client import http_get

from .constants import (
    DATASTORE_SEARCH_URL,
    DEFAULT_LIMIT,
    PACKAGE_LIST_URL,
    PACKAGE_SEARCH_URL,
    PACKAGE_SHOW_URL,
    RESOURCE_INSTITUICOES_CREDENCIADAS,
    RESOURCE_OPERACOES_AUTOMATICAS,
    RESOURCE_OPERACOES_NAO_AUTOMATICAS,
)
from .schemas import (
    Dataset,
    DatasetResource,
    DatastoreResult,
    InstituicaoCredenciada,
)


def _extract_result(data: dict[str, Any]) -> Any:
    """Extract 'result' from CKAN API response. Returns None on failure."""
    if isinstance(data, dict):
        if data.get("success"):
            return data.get("result")
        if "success" in data:
            return None
    return data


async def listar_datasets() -> list[str]:
    """List all dataset names."""
    data: dict[str, Any] = await http_get(PACKAGE_LIST_URL)
    result = _extract_result(data)
    return result if isinstance(result, list) else []


async def buscar_datasets(query: str, limite: int = 10) -> list[Dataset]:
    """Search datasets by keyword."""
    params: dict[str, str] = {"q": query, "rows": str(limite)}
    data: dict[str, Any] = await http_get(PACKAGE_SEARCH_URL, params=params)
    result = _extract_result(data)
    if not isinstance(result, dict):
        return []
    results = result.get("results", [])
    return [_parse_dataset(d) for d in results]


async def detalhar_dataset(nome: str) -> Dataset | None:
    """Get full dataset metadata by name."""
    params: dict[str, str] = {"id": nome}
    data: dict[str, Any] = await http_get(PACKAGE_SHOW_URL, params=params)
    result = _extract_result(data)
    if not isinstance(result, dict):
        return None
    return _parse_dataset(result)


async def consultar_operacoes(
    uf: str | None = None,
    setor: str | None = None,
    porte: str | None = None,
    busca: str | None = None,
    limite: int = DEFAULT_LIMIT,
    offset: int = 0,
    nao_automaticas: bool = False,
) -> DatastoreResult:
    """Query BNDES financing operations via DataStore."""
    resource_id = (
        RESOURCE_OPERACOES_NAO_AUTOMATICAS if nao_automaticas else RESOURCE_OPERACOES_AUTOMATICAS
    )
    params: dict[str, str] = {
        "resource_id": resource_id,
        "limit": str(limite),
        "offset": str(offset),
    }

    filters: dict[str, str] = {}
    if uf:
        # API has leading space in UF field for automatic operations
        filters["uf"] = f" {uf}" if not nao_automaticas else uf
    if setor:
        filters["setor_cnae"] = setor
    if porte:
        filters["porte_do_cliente"] = porte
    if filters:
        params["filters"] = json.dumps(filters)
    if busca:
        params["q"] = busca

    data: dict[str, Any] = await http_get(DATASTORE_SEARCH_URL, params=params)
    result = _extract_result(data)
    return _parse_datastore_result(result)


async def listar_instituicoes_credenciadas() -> list[InstituicaoCredenciada]:
    """List BNDES accredited financial institutions."""
    params: dict[str, str] = {
        "resource_id": RESOURCE_INSTITUICOES_CREDENCIADAS,
        "limit": "200",
    }
    data: dict[str, Any] = await http_get(DATASTORE_SEARCH_URL, params=params)
    result = _extract_result(data)
    if not isinstance(result, dict):
        return []
    records = result.get("records", [])
    return [
        InstituicaoCredenciada(
            cnpj=r.get("cnpj"),
            nome=r.get("razao_social"),
            pagina_na_internet=r.get("site"),
        )
        for r in records
    ]


def _parse_dataset(d: dict[str, Any]) -> Dataset:
    resources = [
        DatasetResource(
            id=r.get("id"),
            name=r.get("name"),
            description=r.get("description"),
            format=r.get("format"),
            url=r.get("url"),
        )
        for r in d.get("resources", [])
    ]
    return Dataset(
        name=d.get("name"),
        title=d.get("title"),
        notes=d.get("notes"),
        num_resources=d.get("num_resources"),
        resources=resources,
    )


def _parse_datastore_result(result: Any) -> DatastoreResult:
    if not isinstance(result, dict):
        return DatastoreResult()
    fields_raw = result.get("fields", [])
    field_names = [f.get("id", "") for f in fields_raw if isinstance(f, dict)]
    return DatastoreResult(
        total=result.get("total", 0),
        fields=field_names,
        records=result.get("records", []),
    )
