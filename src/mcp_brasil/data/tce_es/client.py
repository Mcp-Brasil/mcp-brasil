"""HTTP client for the TCE-ES feature.

Consome a API CKAN Datastore do portal dados.es.gov.br.

Endpoints:
    datastore_search?resource_id=RESOURCE_LICITACOES       → buscar_licitacoes
    datastore_search?resource_id=RESOURCE_CONTRATOS        → buscar_contratos
    datastore_search?resource_id=RESOURCE_CONTRATACOES_MUNICIPIOS → buscar_contratacoes_municipios
    datastore_search?resource_id=RESOURCE_OBRAS            → buscar_obras
"""

from __future__ import annotations

import json
from typing import Any

from mcp_brasil._shared.http_client import http_get

from .constants import (
    CKAN_BASE,
    DEFAULT_LIMIT,
    RESOURCE_CONTRATACOES_MUNICIPIOS,
    RESOURCE_CONTRATOS,
    RESOURCE_LICITACOES,
    RESOURCE_OBRAS,
)
from .schemas import ContratacaoMunicipio, Contrato, Licitacao, Obra


def _parse_ckan(raw: Any) -> tuple[list[dict[str, Any]], int]:
    """Extrai records e total de uma resposta CKAN Datastore."""
    result: dict[str, Any] = raw.get("result", {}) if isinstance(raw, dict) else {}
    records: list[dict[str, Any]] = result.get("records", [])
    total: int = result.get("total", len(records))
    return records, total


def _fields(record: dict[str, Any]) -> dict[str, str | None]:
    """Converte um record CKAN em dict de campos tipados (exclui _id)."""
    return {k: str(v) if v is not None else None for k, v in record.items() if k != "_id"}


async def buscar_licitacoes(
    *,
    q: str | None = None,
    ano: int | None = None,
    situacao: str | None = None,
    limit: int = DEFAULT_LIMIT,
    offset: int = 0,
) -> tuple[list[Licitacao], int]:
    """Busca licitações do próprio TCEES.

    Args:
        q: Texto livre para busca no objeto da licitação.
        ano: Filtro por ano do edital (ex: 2024).
        situacao: Filtro por situação (ex: "Homologado", "Em Andamento").
        limit: Número máximo de resultados.
        offset: Deslocamento para paginação.
    """
    params: dict[str, Any] = {
        "resource_id": RESOURCE_LICITACOES,
        "limit": limit,
        "offset": offset,
    }
    if q:
        params["q"] = q
    filters: dict[str, Any] = {}
    if ano:
        filters["AnoEdital"] = ano
    if situacao:
        filters["Situacao"] = situacao
    if filters:
        params["filters"] = json.dumps(filters)

    raw = await http_get(CKAN_BASE, params=params)
    records, total = _parse_ckan(raw)
    return [Licitacao(**_fields(r)) for r in records], total


async def buscar_contratos(
    *,
    q: str | None = None,
    ano: int | None = None,
    limit: int = DEFAULT_LIMIT,
    offset: int = 0,
) -> tuple[list[Contrato], int]:
    """Busca contratos celebrados pelo TCEES.

    Args:
        q: Texto livre para busca no objeto/fornecedor.
        ano: Filtro por ano do contrato (ex: 2024).
        limit: Número máximo de resultados.
        offset: Deslocamento para paginação.
    """
    params: dict[str, Any] = {
        "resource_id": RESOURCE_CONTRATOS,
        "limit": limit,
        "offset": offset,
    }
    if q:
        params["q"] = q
    if ano:
        params["filters"] = json.dumps({"ContratoAno": ano})

    raw = await http_get(CKAN_BASE, params=params)
    records, total = _parse_ckan(raw)
    return [Contrato(**_fields(r)) for r in records], total


async def buscar_contratacoes_municipios(
    *,
    q: str | None = None,
    ano_referencia: int | None = None,
    esfera: str | None = None,
    limit: int = DEFAULT_LIMIT,
    offset: int = 0,
) -> tuple[list[ContratacaoMunicipio], int]:
    """Busca contratações de municípios e órgãos do ES monitorados pelo TCE-ES.

    Args:
        q: Texto livre para busca (objeto, unidade gestora, etc.).
        ano_referencia: Ano de referência (ex: 2024).
        esfera: Filtro por esfera administrativa (ex: "Municipal", "Estadual").
        limit: Número máximo de resultados.
        offset: Deslocamento para paginação.
    """
    params: dict[str, Any] = {
        "resource_id": RESOURCE_CONTRATACOES_MUNICIPIOS,
        "limit": limit,
        "offset": offset,
    }
    if q:
        params["q"] = q
    filters: dict[str, Any] = {}
    if ano_referencia:
        filters["AnoReferencia"] = ano_referencia
    if esfera:
        filters["NomeEsferaAdministrativa"] = esfera
    if filters:
        params["filters"] = json.dumps(filters)

    raw = await http_get(CKAN_BASE, params=params)
    records, total = _parse_ckan(raw)
    return [ContratacaoMunicipio(**_fields(r)) for r in records], total


async def buscar_obras(
    *,
    q: str | None = None,
    situacao: str | None = None,
    limit: int = DEFAULT_LIMIT,
    offset: int = 0,
) -> tuple[list[Obra], int]:
    """Busca obras públicas cadastradas no TCE-ES.

    Args:
        q: Texto livre (empresa, município, objeto).
        situacao: Filtro por situação da obra.
        limit: Número máximo de resultados.
        offset: Deslocamento para paginação.
    """
    params: dict[str, Any] = {
        "resource_id": RESOURCE_OBRAS,
        "limit": limit,
        "offset": offset,
    }
    if q:
        params["q"] = q
    if situacao:
        params["filters"] = json.dumps({"Situacao": situacao})

    raw = await http_get(CKAN_BASE, params=params)
    records, total = _parse_ckan(raw)
    return [Obra(**_fields(r)) for r in records], total
