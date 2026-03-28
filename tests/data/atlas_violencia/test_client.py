"""Tests for the Atlas da Violência HTTP client (respx-mocked)."""

from __future__ import annotations

import httpx
import pytest
import respx

from mcp_brasil.data.atlas_violencia import client
from mcp_brasil.data.atlas_violencia.constants import (
    FONTES_URL,
    PERIODICIDADES_URL,
    SERIES_URL,
    TEMAS_URL,
    UNIDADES_URL,
    VALORES_REGIOES_URL,
    VALORES_URL,
)


class TestListarTemas:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_temas(self) -> None:
        respx.get(TEMAS_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {"id": 1, "titulo": "Homicidios", "tema_id": 0, "tipo": 0},
                    {"id": 5, "titulo": "Obitos por Armas de Fogo", "tema_id": 0, "tipo": 0},
                ],
            )
        )
        temas = await client.listar_temas()
        assert len(temas) == 2
        assert temas[0].titulo == "Homicidios"
        assert temas[1].id == 5


class TestListarSeries:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_series(self) -> None:
        respx.get(f"{SERIES_URL}/1").mock(
            return_value=httpx.Response(
                200,
                json=[
                    {"id": 328, "titulo": "Homicidios"},
                    {"id": 329, "titulo": "Homicidios - Masculino"},
                ],
            )
        )
        series = await client.listar_series(1)
        assert len(series) == 2
        assert series[0].id == 328

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_series(self) -> None:
        respx.get(f"{SERIES_URL}/99").mock(return_value=httpx.Response(200, json=[]))
        series = await client.listar_series(99)
        assert series == []


class TestConsultarValores:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_values(self) -> None:
        respx.get(f"{VALORES_URL}/328/3").mock(
            return_value=httpx.Response(
                200,
                json=[
                    {"cod": "33", "sigla": "RJ", "valor": "5000", "periodo": "2023-01-15"},
                    {"cod": "35", "sigla": "SP", "valor": "8000", "periodo": "2023-01-15"},
                ],
            )
        )
        valores = await client.consultar_valores(328, 3)
        assert len(valores) == 2
        assert valores[0].sigla == "RJ"
        assert valores[0].valor == "5000"


class TestConsultarValoresPorRegioes:
    @pytest.mark.asyncio
    @respx.mock
    async def test_filters_by_region(self) -> None:
        respx.get(f"{VALORES_REGIOES_URL}/328/3/33,35").mock(
            return_value=httpx.Response(
                200,
                json=[
                    {"cod": "33", "sigla": "RJ", "valor": "5000", "periodo": "2023-01-15"},
                ],
            )
        )
        valores = await client.consultar_valores_por_regioes(328, 3, "33,35")
        assert len(valores) == 1
        assert valores[0].cod == "33"


class TestListarFontes:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_fontes(self) -> None:
        respx.get(FONTES_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {"id": 1, "titulo": "IPEA"},
                    {"id": 2, "titulo": "SIM/DataSUS"},
                ],
            )
        )
        fontes = await client.listar_fontes()
        assert len(fontes) == 2
        assert fontes[1].titulo == "SIM/DataSUS"


class TestListarUnidades:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_unidades(self) -> None:
        respx.get(UNIDADES_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {"id": 7, "titulo": "Quantidade"},
                    {"id": 8, "titulo": "Taxa por 100 mil habitantes"},
                ],
            )
        )
        unidades = await client.listar_unidades()
        assert len(unidades) == 2
        assert unidades[0].id == 7


class TestListarPeriodicidades:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_periodicidades(self) -> None:
        respx.get(PERIODICIDADES_URL).mock(
            return_value=httpx.Response(
                200,
                json=[{"id": 1, "titulo": "Anual"}],
            )
        )
        periodicidades = await client.listar_periodicidades()
        assert len(periodicidades) == 1
        assert periodicidades[0].titulo == "Anual"
