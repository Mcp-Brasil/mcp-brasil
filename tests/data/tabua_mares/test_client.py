"""HTTP client tests for tabua_mares (respx mock)."""

from __future__ import annotations

import pytest
import respx
from httpx import Response

from mcp_brasil.data.tabua_mares.client import (
    buscar_portos,
    consultar_tabua_mare,
    listar_estados,
    listar_portos_estado,
    porto_mais_proximo,
    porto_mais_proximo_geral,
    tabua_mare_por_geolocalizacao,
)
from mcp_brasil.data.tabua_mares.constants import (
    GEO_TABUA_MARE_URL,
    HARBOR_NAMES_URL,
    HARBORS_URL,
    NEAREST_HARBOR_INDEPENDENT_URL,
    NEAREST_HARBOR_URL,
    STATES_URL,
    TABUA_MARE_URL,
)


@respx.mock
@pytest.mark.asyncio
async def test_listar_estados() -> None:
    respx.get(STATES_URL).mock(
        return_value=Response(200, json={"data": ["pb", "rj", "sp"], "total": 3})
    )
    result = await listar_estados()
    assert result == ["pb", "rj", "sp"]


@respx.mock
@pytest.mark.asyncio
async def test_listar_portos_estado() -> None:
    respx.get(f"{HARBOR_NAMES_URL}/pb").mock(
        return_value=Response(
            200,
            json={
                "data": [
                    {
                        "id": "pb01",
                        "year": 2025,
                        "harbor_name": "PORTO DE CABEDELO",
                        "data_collection_institution": "DNPVN",
                    }
                ],
                "total": 1,
            },
        )
    )
    result = await listar_portos_estado("pb")
    assert len(result) == 1
    assert result[0].harbor_name == "PORTO DE CABEDELO"


@respx.mock
@pytest.mark.asyncio
async def test_buscar_portos() -> None:
    respx.get(f"{HARBORS_URL}/[pb01]").mock(
        return_value=Response(
            200,
            json={
                "data": [
                    {
                        "id": "pb01",
                        "harbor_name": "PORTO DE CABEDELO",
                        "state": "pb",
                        "timezone": "UTC -03.0",
                        "card": "921",
                        "geo_location": [],
                        "mean_level": 1.16,
                    }
                ],
                "total": 1,
            },
        )
    )
    result = await buscar_portos(["pb01"])
    assert len(result) == 1
    assert result[0].state == "pb"


@respx.mock
@pytest.mark.asyncio
async def test_consultar_tabua_mare() -> None:
    respx.get(f"{TABUA_MARE_URL}/pb01/1/[1,2,3]").mock(
        return_value=Response(
            200,
            json={
                "data": [
                    {
                        "year": 2025,
                        "harbor_name": "PORTO DE CABEDELO",
                        "state": "pb",
                        "timezone": "UTC -03.0",
                        "card": "921",
                        "data_collection_institution": "DHN",
                        "mean_level": 1.16,
                        "months": [
                            {
                                "month_name": "Janeiro",
                                "month": 1,
                                "days": [
                                    {
                                        "weekday_name": "sexta",
                                        "day": 3,
                                        "hours": [
                                            {"hour": "06:01:00", "level": 1.87},
                                        ],
                                    }
                                ],
                            }
                        ],
                    }
                ],
                "total": 1,
            },
        )
    )
    result = await consultar_tabua_mare("pb01", 1, "1,2,3")
    assert len(result) == 1
    assert result[0].months[0].days[0].hours[0].level == 1.87


@respx.mock
@pytest.mark.asyncio
async def test_porto_mais_proximo() -> None:
    respx.get(f"{NEAREST_HARBOR_URL}/pb/[-7.11,{-34.86}]").mock(
        return_value=Response(
            200,
            json={
                "data": [
                    {
                        "id": "pb01",
                        "harbor_name": "PORTO DE CABEDELO",
                        "state": "pb",
                        "timezone": "UTC -03.0",
                        "card": "921",
                        "geo_location": [],
                        "mean_level": 1.16,
                    }
                ],
                "total": 1,
            },
        )
    )
    result = await porto_mais_proximo("pb", -7.11, -34.86)
    assert len(result) == 1


@respx.mock
@pytest.mark.asyncio
async def test_porto_mais_proximo_geral() -> None:
    respx.get(f"{NEAREST_HARBOR_INDEPENDENT_URL}/[-7.11,{-34.86}]").mock(
        return_value=Response(
            200,
            json={
                "data": [
                    {
                        "id": "pb01",
                        "harbor_name": "PORTO DE CABEDELO",
                        "state": "pb",
                        "timezone": "UTC -03.0",
                        "card": "921",
                        "geo_location": [],
                        "mean_level": 1.16,
                    }
                ],
                "total": 1,
            },
        )
    )
    result = await porto_mais_proximo_geral(-7.11, -34.86)
    assert len(result) == 1


@respx.mock
@pytest.mark.asyncio
async def test_tabua_mare_por_geolocalizacao() -> None:
    respx.get(f"{GEO_TABUA_MARE_URL}/[-7.11,{-34.86}]/pb/1/[1,2,3]").mock(
        return_value=Response(
            200,
            json={
                "data": [
                    {
                        "year": 2025,
                        "harbor_name": "PORTO DE CABEDELO",
                        "state": "pb",
                        "timezone": "UTC -03.0",
                        "card": "921",
                        "data_collection_institution": "DHN",
                        "mean_level": 1.16,
                        "months": [],
                    }
                ],
                "total": 1,
            },
        )
    )
    result = await tabua_mare_por_geolocalizacao(-7.11, -34.86, "pb", 1, "1,2,3")
    assert len(result) == 1
    assert result[0].harbor_name == "PORTO DE CABEDELO"
