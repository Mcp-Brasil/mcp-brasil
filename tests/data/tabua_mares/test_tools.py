"""Unit tests for tabua_mares tools (mock client)."""

from __future__ import annotations

from unittest.mock import AsyncMock, patch

import pytest

from mcp_brasil.data.tabua_mares.schemas import (
    DiaMare,
    GeoLocalizacao,
    HoraMare,
    MesMare,
    Porto,
    PortoResumo,
    TabuaMare,
)
from mcp_brasil.data.tabua_mares.tools import (
    buscar_portos,
    consultar_tabua_mare,
    listar_estados_costeiros,
    listar_portos,
    porto_mais_proximo,
    porto_mais_proximo_geral,
    tabua_mare_por_geolocalizacao,
)

MODULE = "mcp_brasil.data.tabua_mares.client"


@pytest.fixture()
def ctx() -> AsyncMock:
    mock = AsyncMock()
    mock.info = AsyncMock()
    mock.report_progress = AsyncMock()
    return mock


@pytest.fixture()
def sample_porto() -> Porto:
    return Porto(
        id="pb01",
        harbor_name="PORTO DE CABEDELO (ESTADO DA PARAÍBA)",
        state="pb",
        timezone="UTC -03.0",
        card="921",
        geo_location=[
            GeoLocalizacao(
                lat="-6.97",
                lng="-34.84",
                decimal_lat="06° 58' S",
                decimal_lng="34° 50' W",
                lat_direction="s",
                lng_direction="w",
            )
        ],
        mean_level=1.16,
    )


@pytest.fixture()
def sample_tabua() -> TabuaMare:
    return TabuaMare(
        year=2025,
        harbor_name="PORTO DE CABEDELO (ESTADO DA PARAÍBA)",
        state="pb",
        timezone="UTC -03.0",
        card="921",
        data_collection_institution="DHN",
        mean_level=1.16,
        months=[
            MesMare(
                month_name="Janeiro",
                month=1,
                days=[
                    DiaMare(
                        weekday_name="sexta",
                        day=3,
                        hours=[
                            HoraMare(hour="06:01:00", level=1.87),
                            HoraMare(hour="12:04:00", level=0.35),
                        ],
                    )
                ],
            )
        ],
    )


@pytest.mark.asyncio
async def test_listar_estados_costeiros(ctx: AsyncMock) -> None:
    with patch(f"{MODULE}.listar_estados", new_callable=AsyncMock) as mock:
        mock.return_value = ["pb", "rj", "sp"]
        result = await listar_estados_costeiros(ctx)
    assert "PB" in result
    assert "Paraíba" in result
    assert "RJ" in result
    assert "SP" in result


@pytest.mark.asyncio
async def test_listar_portos(ctx: AsyncMock) -> None:
    with patch(f"{MODULE}.listar_portos_estado", new_callable=AsyncMock) as mock:
        mock.return_value = [
            PortoResumo(
                id="pb01",
                year=2025,
                harbor_name="PORTO DE CABEDELO",
                data_collection_institution="DNPVN",
            )
        ]
        result = await listar_portos("pb", ctx)
    assert "PORTO DE CABEDELO" in result
    assert "DNPVN" in result


@pytest.mark.asyncio
async def test_buscar_portos(ctx: AsyncMock, sample_porto: Porto) -> None:
    with patch(f"{MODULE}.buscar_portos", new_callable=AsyncMock) as mock:
        mock.return_value = [sample_porto]
        result = await buscar_portos(["pb01"], ctx)
    assert "PORTO DE CABEDELO" in result
    assert "PB" in result
    assert "1.16" in result


@pytest.mark.asyncio
async def test_consultar_tabua_mare(ctx: AsyncMock, sample_tabua: TabuaMare) -> None:
    with patch(f"{MODULE}.consultar_tabua_mare", new_callable=AsyncMock) as mock:
        mock.return_value = [sample_tabua]
        result = await consultar_tabua_mare("pb01", 1, "1,2,3", ctx)
    assert "PORTO DE CABEDELO" in result
    assert "Janeiro" in result
    assert "06:01:00" in result
    assert "1.87" in result


@pytest.mark.asyncio
async def test_consultar_tabua_mare_empty(ctx: AsyncMock) -> None:
    with patch(f"{MODULE}.consultar_tabua_mare", new_callable=AsyncMock) as mock:
        mock.return_value = []
        result = await consultar_tabua_mare("pb01", 1, "1", ctx)
    assert "Nenhum dado" in result


@pytest.mark.asyncio
async def test_porto_mais_proximo(ctx: AsyncMock, sample_porto: Porto) -> None:
    with patch(f"{MODULE}.porto_mais_proximo", new_callable=AsyncMock) as mock:
        mock.return_value = [sample_porto]
        result = await porto_mais_proximo("pb", -7.11, -34.86, ctx)
    assert "PORTO DE CABEDELO" in result


@pytest.mark.asyncio
async def test_porto_mais_proximo_geral(ctx: AsyncMock, sample_porto: Porto) -> None:
    with patch(f"{MODULE}.porto_mais_proximo_geral", new_callable=AsyncMock) as mock:
        mock.return_value = [sample_porto]
        result = await porto_mais_proximo_geral(-7.11, -34.86, ctx)
    assert "PORTO DE CABEDELO" in result


@pytest.mark.asyncio
async def test_tabua_mare_por_geolocalizacao(ctx: AsyncMock, sample_tabua: TabuaMare) -> None:
    with patch(f"{MODULE}.tabua_mare_por_geolocalizacao", new_callable=AsyncMock) as mock:
        mock.return_value = [sample_tabua]
        result = await tabua_mare_por_geolocalizacao(-7.11, -34.86, "pb", 1, "1,2,3", ctx)
    assert "PORTO DE CABEDELO" in result
    assert "Janeiro" in result
