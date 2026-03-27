"""Tests for anuncios_eleitorais tool functions — mock client."""

from __future__ import annotations

from unittest.mock import AsyncMock, patch

import pytest

from mcp_brasil.data.anuncios_eleitorais.schemas import (
    AnuncioEleitoral,
    DistribuicaoDemografica,
    DistribuicaoRegional,
    FaixaValor,
    RespostaAnuncios,
)
from mcp_brasil.data.anuncios_eleitorais.tools import (
    analisar_demografia_anuncios,
    buscar_anuncios_eleitorais,
    buscar_anuncios_frase_exata,
    buscar_anuncios_por_financiador,
    buscar_anuncios_por_pagina,
    buscar_anuncios_por_regiao,
)


def _make_ad(**overrides: object) -> AnuncioEleitoral:
    """Create a test ad with sensible defaults."""
    defaults = {
        "id": "111222333",
        "page_id": "444555666",
        "page_name": "Candidato Teste",
        "ad_delivery_start_time": "2024-06-01T00:00:00+0000",
        "ad_delivery_stop_time": "2024-06-30T00:00:00+0000",
        "ad_snapshot_url": "https://www.facebook.com/ads/archive/render_ad/?id=111222333",
        "ad_creative_bodies": ["Vote no candidato teste"],
        "bylines": "Partido Teste",
        "currency": "BRL",
        "spend": FaixaValor(lower_bound="100", upper_bound="499"),
        "impressions": FaixaValor(lower_bound="1000", upper_bound="5000"),
        "br_total_reach": 15000,
        "publisher_platforms": ["facebook", "instagram"],
    }
    defaults.update(overrides)
    return AnuncioEleitoral(**defaults)  # type: ignore[arg-type]


def _make_response(ads: list[AnuncioEleitoral] | None = None) -> RespostaAnuncios:
    """Create a test response."""
    return RespostaAnuncios(data=ads or [_make_ad()])


@pytest.fixture
def mock_ctx() -> AsyncMock:
    """Create a mock Context."""
    ctx = AsyncMock()
    ctx.info = AsyncMock()
    ctx.report_progress = AsyncMock()
    return ctx


@patch("mcp_brasil.data.anuncios_eleitorais.tools.client")
@pytest.mark.asyncio
async def test_buscar_anuncios_eleitorais(mock_client: AsyncMock, mock_ctx: AsyncMock) -> None:
    """Test buscar_anuncios_eleitorais formats results correctly."""
    mock_client.buscar_anuncios = AsyncMock(return_value=_make_response())

    result = await buscar_anuncios_eleitorais("candidato", mock_ctx)

    assert "Candidato Teste" in result
    assert "111222333" in result
    assert "Partido Teste" in result
    assert "100 - 499 BRL" in result
    assert "facebook" in result
    mock_client.buscar_anuncios.assert_called_once()


@patch("mcp_brasil.data.anuncios_eleitorais.tools.client")
@pytest.mark.asyncio
async def test_buscar_anuncios_eleitorais_vazio(
    mock_client: AsyncMock, mock_ctx: AsyncMock
) -> None:
    """Test empty results message."""
    mock_client.buscar_anuncios = AsyncMock(return_value=RespostaAnuncios(data=[]))

    result = await buscar_anuncios_eleitorais("inexistente", mock_ctx)

    assert "Nenhum anúncio encontrado" in result


@patch("mcp_brasil.data.anuncios_eleitorais.tools.client")
@pytest.mark.asyncio
async def test_buscar_anuncios_por_pagina(mock_client: AsyncMock, mock_ctx: AsyncMock) -> None:
    """Test search by page IDs."""
    mock_client.buscar_anuncios = AsyncMock(return_value=_make_response())

    result = await buscar_anuncios_por_pagina(["444555666"], mock_ctx)

    assert "Candidato Teste" in result
    mock_client.buscar_anuncios.assert_called_once()
    call_kwargs = mock_client.buscar_anuncios.call_args.kwargs
    assert call_kwargs["search_page_ids"] == ["444555666"]


@patch("mcp_brasil.data.anuncios_eleitorais.tools.client")
@pytest.mark.asyncio
async def test_buscar_anuncios_por_financiador(
    mock_client: AsyncMock, mock_ctx: AsyncMock
) -> None:
    """Test search by bylines."""
    mock_client.buscar_anuncios = AsyncMock(return_value=_make_response())

    result = await buscar_anuncios_por_financiador(["Partido Teste"], mock_ctx)

    assert "Partido Teste" in result
    call_kwargs = mock_client.buscar_anuncios.call_args.kwargs
    assert call_kwargs["bylines"] == ["Partido Teste"]


@patch("mcp_brasil.data.anuncios_eleitorais.tools.client")
@pytest.mark.asyncio
async def test_buscar_anuncios_por_regiao(mock_client: AsyncMock, mock_ctx: AsyncMock) -> None:
    """Test search by region."""
    mock_client.buscar_anuncios = AsyncMock(return_value=_make_response())

    result = await buscar_anuncios_por_regiao(["São Paulo"], mock_ctx)

    assert "Candidato Teste" in result
    call_kwargs = mock_client.buscar_anuncios.call_args.kwargs
    assert call_kwargs["delivery_by_region"] == ["São Paulo"]


@patch("mcp_brasil.data.anuncios_eleitorais.tools.client")
@pytest.mark.asyncio
async def test_buscar_anuncios_frase_exata(mock_client: AsyncMock, mock_ctx: AsyncMock) -> None:
    """Test exact phrase search."""
    mock_client.buscar_anuncios = AsyncMock(return_value=_make_response())

    result = await buscar_anuncios_frase_exata("governo federal", mock_ctx)

    assert "Candidato Teste" in result
    call_kwargs = mock_client.buscar_anuncios.call_args.kwargs
    assert call_kwargs["search_type"] == "KEYWORD_EXACT_PHRASE"


@patch("mcp_brasil.data.anuncios_eleitorais.tools.client")
@pytest.mark.asyncio
async def test_analisar_demografia_anuncios(mock_client: AsyncMock, mock_ctx: AsyncMock) -> None:
    """Test demographic analysis."""
    ad = _make_ad(
        demographic_distribution=[
            DistribuicaoDemografica(age="25-34", gender="Male", percentage="0.25"),
            DistribuicaoDemografica(age="25-34", gender="Female", percentage="0.20"),
            DistribuicaoDemografica(age="35-44", gender="Male", percentage="0.15"),
        ],
        delivery_by_region=[
            DistribuicaoRegional(region="São Paulo", percentage="0.35"),
            DistribuicaoRegional(region="Rio de Janeiro", percentage="0.20"),
        ],
    )
    mock_client.buscar_anuncios = AsyncMock(return_value=_make_response([ad]))

    result = await analisar_demografia_anuncios("teste", mock_ctx)

    assert "Distribuição por idade e gênero" in result
    assert "Distribuição por região" in result
    assert "São Paulo" in result
    assert "25-34" in result


@patch("mcp_brasil.data.anuncios_eleitorais.tools.client")
@pytest.mark.asyncio
async def test_analisar_demografia_sem_dados(mock_client: AsyncMock, mock_ctx: AsyncMock) -> None:
    """Test demographic analysis with no demographic data."""
    ad = _make_ad(demographic_distribution=None, delivery_by_region=None)
    mock_client.buscar_anuncios = AsyncMock(return_value=_make_response([ad]))

    result = await analisar_demografia_anuncios("teste", mock_ctx)

    assert "não disponíveis" in result


@patch("mcp_brasil.data.anuncios_eleitorais.tools.client")
@pytest.mark.asyncio
async def test_formatar_anuncio_sem_stop_time(mock_client: AsyncMock, mock_ctx: AsyncMock) -> None:
    """Test formatting an ad without stop time (still running)."""
    ad = _make_ad(ad_delivery_stop_time=None)
    mock_client.buscar_anuncios = AsyncMock(return_value=_make_response([ad]))

    result = await buscar_anuncios_eleitorais("teste", mock_ctx)

    assert "em veiculação" in result


@patch("mcp_brasil.data.anuncios_eleitorais.tools.client")
@pytest.mark.asyncio
async def test_formatar_anuncio_texto_longo(mock_client: AsyncMock, mock_ctx: AsyncMock) -> None:
    """Test formatting an ad with long text (should truncate)."""
    long_text = "A" * 500
    ad = _make_ad(ad_creative_bodies=[long_text])
    mock_client.buscar_anuncios = AsyncMock(return_value=_make_response([ad]))

    result = await buscar_anuncios_eleitorais("teste", mock_ctx)

    assert "..." in result
    # Original text is 500 chars, truncated to 300 + "..."
    assert "A" * 300 in result


@patch("mcp_brasil.data.anuncios_eleitorais.tools.client")
@pytest.mark.asyncio
async def test_buscar_anuncios_com_filtros(mock_client: AsyncMock, mock_ctx: AsyncMock) -> None:
    """Test that tool passes filters to client correctly."""
    mock_client.buscar_anuncios = AsyncMock(return_value=_make_response())

    await buscar_anuncios_eleitorais(
        "educação",
        mock_ctx,
        ad_active_status="ALL",
        ad_delivery_date_min="2024-01-01",
        ad_delivery_date_max="2024-12-31",
        media_type="VIDEO",
        publisher_platforms=["FACEBOOK"],
        search_type="KEYWORD_EXACT_PHRASE",
        limit=50,
    )

    call_kwargs = mock_client.buscar_anuncios.call_args.kwargs
    assert call_kwargs["search_terms"] == "educação"
    assert call_kwargs["ad_active_status"] == "ALL"
    assert call_kwargs["ad_delivery_date_min"] == "2024-01-01"
    assert call_kwargs["ad_delivery_date_max"] == "2024-12-31"
    assert call_kwargs["media_type"] == "VIDEO"
    assert call_kwargs["publisher_platforms"] == ["FACEBOOK"]
    assert call_kwargs["search_type"] == "KEYWORD_EXACT_PHRASE"
    assert call_kwargs["limit"] == 50
