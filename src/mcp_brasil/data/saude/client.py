"""HTTP client for the CNES/DataSUS API.

Endpoints:
    - /estabelecimentos          → buscar_estabelecimentos
    - /estabelecimentos/{cnes}   → buscar_estabelecimento_por_cnes
    - /profissionais             → buscar_profissionais
    - /tipodeestabelecimento     → listar_tipos_estabelecimento
    - /leitos                    → consultar_leitos
"""

from __future__ import annotations

from typing import Any

from mcp_brasil._shared.http_client import http_get
from mcp_brasil.exceptions import HttpClientError

from .constants import (
    DEFAULT_LIMIT,
    ESTABELECIMENTOS_URL,
    LEITOS_URL,
    MAX_LIMIT,
    MAX_LIMIT_LEITOS,
    TIPOS_URL,
)
from .schemas import (
    Estabelecimento,
    EstabelecimentoDetalhe,
    Leito,
    TipoEstabelecimento,
)


def _ensure_list(data: Any, url: str) -> list[dict[str, Any]]:
    """Extract a list of dicts from the API response.

    The DataSUS API returns either a list or a dict with a single key
    containing the list (e.g. {"estabelecimentos": [...]}).
    """
    if isinstance(data, list):
        return data
    if isinstance(data, dict):
        # Try to extract the first list-valued key (e.g. "estabelecimentos")
        for v in data.values():
            if isinstance(v, list):
                return v
        return []
    raise HttpClientError(
        f"Unexpected response from {url}: expected list/dict, got {type(data).__name__}"
    )


def _ensure_dict(data: Any, url: str) -> dict[str, Any]:
    """Validate that API response is a dict."""
    if isinstance(data, dict):
        return data
    raise HttpClientError(
        f"Unexpected response from {url}: expected dict, got {type(data).__name__}"
    )


def _parse_estabelecimento(raw: dict[str, Any]) -> Estabelecimento:
    """Parse a raw establishment dict into an Estabelecimento model."""
    return Estabelecimento(
        codigo_cnes=str(raw.get("codigo_cnes", "") or ""),
        nome_fantasia=raw.get("nome_fantasia"),
        nome_razao_social=raw.get("nome_razao_social"),
        natureza_organizacao=raw.get("natureza_organizacao_entidade"),
        tipo_gestao=raw.get("tipo_gestao"),
        codigo_tipo=str(raw.get("codigo_tipo_unidade", "") or ""),
        descricao_tipo=raw.get("descricao_turno_atendimento"),
        codigo_municipio=str(raw.get("codigo_municipio", "") or ""),
        codigo_uf=str(raw.get("codigo_uf", "") or ""),
        endereco=raw.get("endereco_estabelecimento"),
    )


def _parse_tipo(raw: dict[str, Any]) -> TipoEstabelecimento:
    """Parse a raw type dict into a TipoEstabelecimento model."""
    return TipoEstabelecimento(
        codigo=str(raw.get("codigo_tipo_unidade", "") or ""),
        descricao=raw.get("descricao_tipo_unidade"),
    )


def _parse_leito(raw: dict[str, Any]) -> Leito:
    """Parse a raw hospital/bed dict from hospitais-e-leitos endpoint."""
    return Leito(
        codigo_cnes=str(raw.get("nome_do_hospital", "") or ""),
        tipo_leito=raw.get("descricao_do_tipo_da_unidade"),
        especialidade=raw.get("descricao_da_natureza_juridica_do_hosptial"),
        existente=raw.get("quantidade_total_de_leitos_do_hosptial"),
        sus=raw.get("quantidade_total_de_leitos_sus_do_hosptial"),
    )


async def buscar_estabelecimentos(
    *,
    codigo_municipio: str | None = None,
    codigo_uf: str | None = None,
    status: int | None = None,
    limit: int = DEFAULT_LIMIT,
    offset: int = 0,
) -> list[Estabelecimento]:
    """Search health establishments from CNES.

    API: GET /estabelecimentos

    Args:
        codigo_municipio: IBGE municipality code (e.g. "355030").
        codigo_uf: IBGE state code (e.g. "35").
        status: 1 for active, 0 for inactive.
        limit: Max results per page.
        offset: Pagination offset.
    """
    params: dict[str, Any] = {
        "limit": min(limit, MAX_LIMIT),
        "offset": offset,
    }
    if codigo_municipio:
        params["codigo_municipio"] = codigo_municipio
    if codigo_uf:
        params["codigo_uf"] = codigo_uf
    if status is not None:
        params["status"] = status

    raw = await http_get(ESTABELECIMENTOS_URL, params=params)
    data = _ensure_list(raw, ESTABELECIMENTOS_URL)
    return [_parse_estabelecimento(item) for item in data]


async def listar_tipos_estabelecimento() -> list[TipoEstabelecimento]:
    """Fetch all establishment types from CNES.

    API: GET /tipodeestabelecimento
    """
    raw = await http_get(TIPOS_URL)
    data = _ensure_list(raw, TIPOS_URL)
    return [_parse_tipo(item) for item in data]


async def consultar_leitos(
    *,
    limit: int = DEFAULT_LIMIT,
    offset: int = 0,
) -> list[Leito]:
    """Search hospitals and beds from DataSUS.

    API: GET /assistencia-a-saude/hospitais-e-leitos

    Args:
        limit: Max results per page (max 1000).
        offset: Pagination offset.
    """
    params: dict[str, Any] = {
        "limit": min(limit, MAX_LIMIT_LEITOS),
        "offset": offset,
    }

    raw = await http_get(LEITOS_URL, params=params)
    data = _ensure_list(raw, LEITOS_URL)
    return [_parse_leito(item) for item in data]


def _parse_estabelecimento_detalhe(raw: dict[str, Any]) -> EstabelecimentoDetalhe:
    """Parse a raw establishment dict into an EstabelecimentoDetalhe model."""
    return EstabelecimentoDetalhe(
        codigo_cnes=str(raw.get("codigo_cnes", "") or ""),
        nome_fantasia=raw.get("nome_fantasia"),
        nome_razao_social=raw.get("nome_razao_social"),
        natureza_organizacao=raw.get("natureza_organizacao_entidade"),
        tipo_gestao=raw.get("tipo_gestao"),
        codigo_tipo=str(raw.get("codigo_tipo_unidade", "") or ""),
        descricao_tipo=raw.get("descricao_turno_atendimento"),
        codigo_municipio=str(raw.get("codigo_municipio", "") or ""),
        codigo_uf=str(raw.get("codigo_uf", "") or ""),
        endereco=raw.get("endereco_estabelecimento"),
        bairro=raw.get("bairro_estabelecimento"),
        cep=raw.get("codigo_cep_estabelecimento"),
        telefone=raw.get("numero_telefone_estabelecimento"),
        latitude=raw.get("latitude_estabelecimento_decimo_grau"),
        longitude=raw.get("longitude_estabelecimento_decimo_grau"),
        cnpj=raw.get("numero_cnpj"),
        data_atualizacao=raw.get("data_atualizacao"),
    )


async def buscar_estabelecimento_por_cnes(cnes: str) -> EstabelecimentoDetalhe | None:
    """Fetch a single establishment by its CNES code.

    API: GET /estabelecimentos/{cnes}

    Args:
        cnes: CNES code (7 digits).
    """
    url = f"{ESTABELECIMENTOS_URL}/{cnes}"
    raw = await http_get(url)
    if not raw:
        return None
    data = _ensure_dict(raw, url)
    return _parse_estabelecimento_detalhe(data)


async def buscar_estabelecimentos_por_tipo(
    *,
    codigo_tipo: str,
    codigo_municipio: str | None = None,
    codigo_uf: str | None = None,
    status: int = 1,
    limit: int = DEFAULT_LIMIT,
    offset: int = 0,
) -> list[Estabelecimento]:
    """Search establishments filtered by type code.

    API: GET /estabelecimentos

    Args:
        codigo_tipo: Establishment type code (e.g. "73" for Pronto Atendimento).
        codigo_municipio: IBGE municipality code.
        codigo_uf: IBGE state code.
        status: 1 for active, 0 for inactive.
        limit: Max results per page.
        offset: Pagination offset.
    """
    params: dict[str, Any] = {
        "codigo_tipo_unidade": codigo_tipo,
        "status": status,
        "limit": min(limit, MAX_LIMIT),
        "offset": offset,
    }
    if codigo_municipio:
        params["codigo_municipio"] = codigo_municipio
    if codigo_uf:
        params["codigo_uf"] = codigo_uf

    raw = await http_get(ESTABELECIMENTOS_URL, params=params)
    data = _ensure_list(raw, ESTABELECIMENTOS_URL)
    return [_parse_estabelecimento(item) for item in data]
