"""Tests for anuncios_eleitorais HTTP client — respx mock HTTP."""

from __future__ import annotations

import os
from unittest.mock import patch

import pytest
import respx
from httpx import Response

from mcp_brasil.data.anuncios_eleitorais.client import buscar_anuncios, buscar_proxima_pagina
from mcp_brasil.data.anuncios_eleitorais.constants import ADS_ARCHIVE_URL

MOCK_TOKEN = "test_access_token_123"

MOCK_RESPONSE = {
    "data": [
        {
            "id": "111222333",
            "page_id": "444555666",
            "page_name": "Candidato Teste",
            "ad_delivery_start_time": "2024-06-01T00:00:00+0000",
            "ad_delivery_stop_time": "2024-06-30T00:00:00+0000",
            "ad_snapshot_url": "https://www.facebook.com/ads/archive/render_ad/?id=111222333",
            "ad_creative_bodies": ["Vote no candidato teste para um futuro melhor"],
            "ad_creative_link_titles": ["Candidato Teste 2024"],
            "ad_creative_link_descriptions": ["Conheça as propostas"],
            "ad_creative_link_captions": ["candidatoteste.com.br"],
            "bylines": "Partido Teste",
            "currency": "BRL",
            "spend": {"lower_bound": "100", "upper_bound": "499"},
            "impressions": {"lower_bound": "1000", "upper_bound": "5000"},
            "demographic_distribution": [
                {"age": "25-34", "gender": "Male", "percentage": "0.25"},
                {"age": "25-34", "gender": "Female", "percentage": "0.20"},
                {"age": "35-44", "gender": "Male", "percentage": "0.15"},
            ],
            "delivery_by_region": [
                {"region": "São Paulo", "percentage": "0.35"},
                {"region": "Rio de Janeiro", "percentage": "0.20"},
            ],
            "estimated_audience_size": {"lower_bound": "10000", "upper_bound": "50000"},
            "br_total_reach": 15000,
            "languages": ["pt"],
            "publisher_platforms": ["facebook", "instagram"],
            "target_ages": ["25", "65"],
            "target_gender": "All",
            "target_locations": [{"name": "Brazil"}],
        }
    ],
    "paging": {
        "cursors": {"before": "ABC123", "after": "DEF456"},
        "next": "https://graph.facebook.com/v21.0/ads_archive?after=DEF456",
    },
}

MOCK_EMPTY_RESPONSE = {"data": []}


@pytest.fixture(autouse=True)
def _set_token() -> None:
    """Set META_ACCESS_TOKEN for all tests."""
    with patch.dict(os.environ, {"META_ACCESS_TOKEN": MOCK_TOKEN}):
        yield  # type: ignore[misc]


@respx.mock
@pytest.mark.asyncio
async def test_buscar_anuncios_basico() -> None:
    """Test basic ad search with default parameters."""
    respx.get(ADS_ARCHIVE_URL).mock(return_value=Response(200, json=MOCK_RESPONSE))

    result = await buscar_anuncios(search_terms="candidato")

    assert len(result.data) == 1
    ad = result.data[0]
    assert ad.id == "111222333"
    assert ad.page_name == "Candidato Teste"
    assert ad.bylines == "Partido Teste"
    assert ad.currency == "BRL"
    assert ad.br_total_reach == 15000


@respx.mock
@pytest.mark.asyncio
async def test_buscar_anuncios_com_filtros() -> None:
    """Test ad search with date and status filters."""
    respx.get(ADS_ARCHIVE_URL).mock(return_value=Response(200, json=MOCK_RESPONSE))

    result = await buscar_anuncios(
        search_terms="educação",
        ad_active_status="ALL",
        ad_delivery_date_min="2024-01-01",
        ad_delivery_date_max="2024-12-31",
        media_type="VIDEO",
        limit=50,
    )

    assert len(result.data) == 1


@respx.mock
@pytest.mark.asyncio
async def test_buscar_anuncios_por_pagina() -> None:
    """Test search by page IDs."""
    respx.get(ADS_ARCHIVE_URL).mock(return_value=Response(200, json=MOCK_RESPONSE))

    result = await buscar_anuncios(search_page_ids=["444555666"])

    assert len(result.data) == 1
    assert result.data[0].page_id == "444555666"


@respx.mock
@pytest.mark.asyncio
async def test_buscar_anuncios_por_bylines() -> None:
    """Test search by bylines (funder)."""
    respx.get(ADS_ARCHIVE_URL).mock(return_value=Response(200, json=MOCK_RESPONSE))

    result = await buscar_anuncios(bylines=["Partido Teste"])

    assert len(result.data) == 1
    assert result.data[0].bylines == "Partido Teste"


@respx.mock
@pytest.mark.asyncio
async def test_buscar_anuncios_por_regiao() -> None:
    """Test search with delivery_by_region filter."""
    respx.get(ADS_ARCHIVE_URL).mock(return_value=Response(200, json=MOCK_RESPONSE))

    result = await buscar_anuncios(delivery_by_region=["São Paulo"])

    assert len(result.data) == 1


@respx.mock
@pytest.mark.asyncio
async def test_buscar_anuncios_resposta_vazia() -> None:
    """Test empty response."""
    respx.get(ADS_ARCHIVE_URL).mock(return_value=Response(200, json=MOCK_EMPTY_RESPONSE))

    result = await buscar_anuncios(search_terms="inexistente")

    assert len(result.data) == 0
    assert result.paging is None


@respx.mock
@pytest.mark.asyncio
async def test_buscar_anuncios_paginacao() -> None:
    """Test pagination data is parsed correctly."""
    respx.get(ADS_ARCHIVE_URL).mock(return_value=Response(200, json=MOCK_RESPONSE))

    result = await buscar_anuncios(search_terms="teste")

    assert result.paging is not None
    assert result.paging.cursors is not None
    assert result.paging.cursors.after == "DEF456"
    assert result.paging.next is not None


@respx.mock
@pytest.mark.asyncio
async def test_buscar_proxima_pagina() -> None:
    """Test following a pagination cursor."""
    next_url = "https://graph.facebook.com/v21.0/ads_archive?after=DEF456"
    respx.get(next_url).mock(return_value=Response(200, json=MOCK_RESPONSE))

    result = await buscar_proxima_pagina(next_url)

    assert len(result.data) == 1


@respx.mock
@pytest.mark.asyncio
async def test_buscar_anuncios_schemas_completos() -> None:
    """Test that all schema fields are properly parsed."""
    respx.get(ADS_ARCHIVE_URL).mock(return_value=Response(200, json=MOCK_RESPONSE))

    result = await buscar_anuncios(search_terms="teste")
    ad = result.data[0]

    # Check spend range
    assert ad.spend is not None
    assert ad.spend.lower_bound == "100"
    assert ad.spend.upper_bound == "499"

    # Check impressions range
    assert ad.impressions is not None
    assert ad.impressions.lower_bound == "1000"
    assert ad.impressions.upper_bound == "5000"

    # Check demographics
    assert ad.demographic_distribution is not None
    assert len(ad.demographic_distribution) == 3
    assert ad.demographic_distribution[0].age == "25-34"

    # Check regions
    assert ad.delivery_by_region is not None
    assert len(ad.delivery_by_region) == 2
    assert ad.delivery_by_region[0].region == "São Paulo"

    # Check estimated audience
    assert ad.estimated_audience_size is not None
    assert ad.estimated_audience_size.lower_bound == "10000"

    # Check target data
    assert ad.target_ages == ["25", "65"]
    assert ad.target_gender == "All"
    assert ad.target_locations is not None
    assert ad.target_locations[0].name == "Brazil"

    # Check platforms
    assert ad.publisher_platforms == ["facebook", "instagram"]
    assert ad.languages == ["pt"]


@respx.mock
@pytest.mark.asyncio
async def test_buscar_anuncios_com_plataformas() -> None:
    """Test search with publisher_platforms filter."""
    respx.get(ADS_ARCHIVE_URL).mock(return_value=Response(200, json=MOCK_RESPONSE))

    result = await buscar_anuncios(
        search_terms="teste",
        publisher_platforms=["FACEBOOK", "INSTAGRAM"],
    )

    assert len(result.data) == 1


@respx.mock
@pytest.mark.asyncio
async def test_buscar_anuncios_com_audiencia() -> None:
    """Test search with audience size filters."""
    respx.get(ADS_ARCHIVE_URL).mock(return_value=Response(200, json=MOCK_RESPONSE))

    result = await buscar_anuncios(
        search_terms="teste",
        estimated_audience_size_min=1000,
        estimated_audience_size_max=50000,
    )

    assert len(result.data) == 1


def test_get_access_token_missing() -> None:
    """Test error when META_ACCESS_TOKEN is not set."""
    with patch.dict(os.environ, {}, clear=True):
        from mcp_brasil.data.anuncios_eleitorais.client import _get_access_token

        with pytest.raises(RuntimeError, match="META_ACCESS_TOKEN"):
            _get_access_token()
