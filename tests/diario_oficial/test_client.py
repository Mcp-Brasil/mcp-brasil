"""Tests for the Diário Oficial HTTP client."""

import httpx
import pytest
import respx

from mcp_brasil.diario_oficial import client
from mcp_brasil.diario_oficial.constants import CITIES_URL, GAZETTES_URL

# ---------------------------------------------------------------------------
# buscar_diarios
# ---------------------------------------------------------------------------


class TestBuscarDiarios:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_gazettes(self) -> None:
        respx.get(GAZETTES_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "total_gazettes": 1,
                    "gazettes": [
                        {
                            "territory_id": "3550308",
                            "territory_name": "São Paulo",
                            "state_code": "SP",
                            "date": "2024-01-15",
                            "edition_number": "1234",
                            "is_extra_edition": False,
                            "url": "https://example.com/gazette.pdf",
                            "txt_url": "https://example.com/gazette.txt",
                            "excerpts": ["Trecho relevante"],
                            "highlight_texts": [],
                        }
                    ],
                },
            )
        )
        result = await client.buscar_diarios(querystring="licitação")
        assert result.total_gazettes == 1
        assert len(result.gazettes) == 1
        assert result.gazettes[0].territory_name == "São Paulo"
        assert result.gazettes[0].state_code == "SP"
        assert result.gazettes[0].date == "2024-01-15"
        assert result.gazettes[0].excerpts == ["Trecho relevante"]

    @pytest.mark.asyncio
    @respx.mock
    async def test_querystring_in_params(self) -> None:
        route = respx.get(GAZETTES_URL).mock(
            return_value=httpx.Response(
                200,
                json={"total_gazettes": 0, "gazettes": []},
            )
        )
        await client.buscar_diarios(querystring="contrato")
        req_url = str(route.calls[0].request.url)
        assert "querystring=contrato" in req_url

    @pytest.mark.asyncio
    @respx.mock
    async def test_territory_id_in_url(self) -> None:
        route = respx.get(f"{GAZETTES_URL}/3550308").mock(
            return_value=httpx.Response(
                200,
                json={"total_gazettes": 0, "gazettes": []},
            )
        )
        await client.buscar_diarios(querystring="teste", territory_id="3550308")
        assert route.called

    @pytest.mark.asyncio
    @respx.mock
    async def test_date_filters_in_params(self) -> None:
        route = respx.get(GAZETTES_URL).mock(
            return_value=httpx.Response(
                200,
                json={"total_gazettes": 0, "gazettes": []},
            )
        )
        await client.buscar_diarios(
            querystring="teste",
            since="2024-01-01",
            until="2024-12-31",
        )
        req_url = str(route.calls[0].request.url)
        assert "since=2024-01-01" in req_url
        assert "until=2024-12-31" in req_url

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(GAZETTES_URL).mock(
            return_value=httpx.Response(
                200,
                json={"total_gazettes": 0, "gazettes": []},
            )
        )
        result = await client.buscar_diarios(querystring="nada")
        assert result.total_gazettes == 0
        assert result.gazettes == []


# ---------------------------------------------------------------------------
# buscar_cidades
# ---------------------------------------------------------------------------


class TestBuscarCidades:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_cities(self) -> None:
        respx.get(CITIES_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "territory_id": "3550308",
                        "territory_name": "São Paulo",
                        "state_code": "SP",
                        "publication_urls": [],
                        "level": "city",
                    },
                    {
                        "territory_id": "3549904",
                        "territory_name": "São José dos Campos",
                        "state_code": "SP",
                    },
                ],
            )
        )
        result = await client.buscar_cidades("São")
        assert len(result) == 2
        assert result[0].territory_id == "3550308"
        assert result[0].territory_name == "São Paulo"
        assert result[1].territory_name == "São José dos Campos"

    @pytest.mark.asyncio
    @respx.mock
    async def test_city_name_in_params(self) -> None:
        route = respx.get(CITIES_URL).mock(return_value=httpx.Response(200, json=[]))
        await client.buscar_cidades("Natal")
        req_url = str(route.calls[0].request.url)
        assert "city_name=Natal" in req_url

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(CITIES_URL).mock(return_value=httpx.Response(200, json=[]))
        result = await client.buscar_cidades("Inexistente")
        assert result == []


# ---------------------------------------------------------------------------
# listar_cidades
# ---------------------------------------------------------------------------


class TestListarCidades:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_all_cities(self) -> None:
        respx.get(CITIES_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "territory_id": "3550308",
                        "territory_name": "São Paulo",
                        "state_code": "SP",
                    },
                ],
            )
        )
        result = await client.listar_cidades()
        assert len(result) == 1
        assert result[0].territory_id == "3550308"
        assert result[0].state_code == "SP"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(CITIES_URL).mock(return_value=httpx.Response(200, json=[]))
        result = await client.listar_cidades()
        assert result == []
