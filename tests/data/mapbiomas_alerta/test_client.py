"""Tests for the MapBiomas Alerta HTTP client (GraphQL + login)."""

from __future__ import annotations

import json
from collections.abc import Callable
from typing import Any

import httpx
import pytest
import respx

from mcp_brasil.data.mapbiomas_alerta import client
from mcp_brasil.data.mapbiomas_alerta.constants import GRAPHQL_URL
from mcp_brasil.exceptions import AuthError, HttpClientError

Handler = Callable[[httpx.Request], httpx.Response]


@pytest.fixture(autouse=True)
def _ambiente(monkeypatch: pytest.MonkeyPatch) -> None:
    """Credenciais sintéticas e cache de token limpo para cada teste."""
    monkeypatch.setenv("MAPBIOMAS_EMAIL", "conta@example.com")
    monkeypatch.setenv("MAPBIOMAS_PASSWORD", "senha-sintetica")
    client._reset_token()


def _body(request: httpx.Request) -> dict[str, Any]:
    parsed = json.loads(request.content)
    assert isinstance(parsed, dict)
    return parsed


def _handler(respostas: dict[str, Any], *, signin: Any = None) -> Handler:
    """Handler que responde conforme o trecho da query GraphQL recebida."""

    def handler(request: httpx.Request) -> httpx.Response:
        query = str(_body(request).get("query", ""))
        if "signIn" in query:
            payload = signin if signin is not None else {"data": {"signIn": {"token": "tok-1"}}}
            return httpx.Response(200, json=payload)
        for trecho, payload in respostas.items():
            if trecho in query:
                return httpx.Response(200, json=payload)
        return httpx.Response(200, json={"data": {}})

    return handler


# ---------------------------------------------------------------------------
# listar_territorios (consulta pública)
# ---------------------------------------------------------------------------


class TestListarTerritorios:
    @pytest.mark.asyncio
    @respx.mock
    async def test_parses_and_filters(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.delenv("MAPBIOMAS_EMAIL", raising=False)
        monkeypatch.delenv("MAPBIOMAS_PASSWORD", raising=False)
        respx.post(GRAPHQL_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "data": {
                        "territoryOptions": [
                            {
                                "category": "city",
                                "categoryName": "Município",
                                "territories": [
                                    {"code": 5211909, "name": "Jataí"},
                                    {"code": 5208707, "name": "Goiânia"},
                                ],
                            },
                            {
                                "category": "indigenous_land",
                                "categoryName": "Terra Indígena",
                                "territories": [{"code": 999, "name": "TI Exemplo"}],
                            },
                        ]
                    }
                },
            )
        )
        resultado = await client.listar_territorios(categoria="city", busca="jatai")
        assert len(resultado) == 1
        assert resultado[0].codigo == 5211909
        assert resultado[0].categoria == "city"

    @pytest.mark.asyncio
    @respx.mock
    async def test_limits_results(self) -> None:
        respx.post(GRAPHQL_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "data": {
                        "territoryOptions": [
                            {
                                "category": "city",
                                "categoryName": "Município",
                                "territories": [
                                    {"code": i, "name": f"Cidade {i}"} for i in range(1, 6)
                                ],
                            }
                        ]
                    }
                },
            )
        )
        resultado = await client.listar_territorios(limite=2)
        assert len(resultado) == 2


# ---------------------------------------------------------------------------
# buscar_alertas
# ---------------------------------------------------------------------------


class TestBuscarAlertas:
    @pytest.mark.asyncio
    @respx.mock
    async def test_parses_page(self) -> None:
        respx.post(GRAPHQL_URL).mock(
            side_effect=_handler(
                {
                    "alerts": {
                        "data": {
                            "alerts": {
                                "metadata": {
                                    "currentPage": 1,
                                    "limitValue": 20,
                                    "totalCount": 1,
                                    "totalPages": 1,
                                },
                                "collection": [
                                    {
                                        "alertCode": 123456,
                                        "areaHa": 11.6,
                                        "detectedAt": "2024-05-10",
                                        "publishedAt": "2024-07-01",
                                        "statusName": "Publicado",
                                        "sources": ["DETER Cerrado"],
                                        "crossedBiomes": ["Cerrado"],
                                        "crossedStates": ["GO"],
                                        "crossedCities": ["Jataí"],
                                        "crossedPermanentProtectedArea": 2.5,
                                        "crossedLegalReservesArea": 4.0,
                                        "crossedConservationUnits": [],
                                        "crossedIndigenousLands": [],
                                        "crossedEmbargoesTotal": 1,
                                        "crossedDeforestationAuthorizationsArea": 0.0,
                                        "ruralPropertiesCodes": ["GO-5211909-EXEMPLO"],
                                        "ruralPropertiesTotal": 1,
                                    }
                                ],
                            }
                        }
                    }
                }
            )
        )
        resultado = await client.buscar_alertas(territorio_id=5211909)
        assert resultado.total == 1
        assert resultado.pagina == 1
        assert len(resultado.alertas) == 1
        alerta = resultado.alertas[0]
        assert alerta.codigo == 123456
        assert alerta.area_ha == 11.6
        assert alerta.app_area_ha == 2.5
        assert alerta.car_codigos == ["GO-5211909-EXEMPLO"]

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_collection(self) -> None:
        respx.post(GRAPHQL_URL).mock(
            side_effect=_handler(
                {
                    "alerts": {
                        "data": {
                            "alerts": {
                                "metadata": {
                                    "currentPage": 1,
                                    "limitValue": 20,
                                    "totalCount": 0,
                                    "totalPages": 0,
                                },
                                "collection": [],
                            }
                        }
                    }
                }
            )
        )
        resultado = await client.buscar_alertas(territorio_id=1)
        assert resultado.alertas == []
        assert resultado.total == 0

    @pytest.mark.asyncio
    @respx.mock
    async def test_graphql_error_raises_http_error(self) -> None:
        respx.post(GRAPHQL_URL).mock(
            side_effect=_handler(
                {
                    "alerts": {
                        "data": None,
                        "errors": [{"message": "Campo inválido no filtro"}],
                    }
                }
            )
        )
        with pytest.raises(HttpClientError, match="recusou a consulta"):
            await client.buscar_alertas(territorio_id=1)

    @pytest.mark.asyncio
    @respx.mock
    async def test_missing_credentials_raises_auth_error(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        monkeypatch.delenv("MAPBIOMAS_EMAIL", raising=False)
        monkeypatch.delenv("MAPBIOMAS_PASSWORD", raising=False)
        respx.post(GRAPHQL_URL).mock(return_value=httpx.Response(200, json={"data": {}}))
        with pytest.raises(AuthError, match="não configuradas"):
            await client.buscar_alertas(territorio_id=1)

    @pytest.mark.asyncio
    @respx.mock
    async def test_wrong_credentials_raises_auth_error(self) -> None:
        respx.post(GRAPHQL_URL).mock(
            side_effect=_handler(
                {},
                signin={
                    "data": None,
                    "errors": [{"message": "Usuário ou senha incorretos"}],
                },
            )
        )
        with pytest.raises(AuthError, match="recusou a credencial"):
            await client.buscar_alertas(territorio_id=1)

    @pytest.mark.asyncio
    @respx.mock
    async def test_refreshes_token_once(self) -> None:
        chamadas = {"signin": 0, "consultas": 0}

        def handler(request: httpx.Request) -> httpx.Response:
            query = str(_body(request).get("query", ""))
            if "signIn" in query:
                chamadas["signin"] += 1
                return httpx.Response(
                    200, json={"data": {"signIn": {"token": f"tok-{chamadas['signin']}"}}}
                )
            chamadas["consultas"] += 1
            auth = request.headers.get("authorization", "")
            if auth == "Bearer tok-1":
                return httpx.Response(
                    200,
                    json={"data": None, "errors": [{"message": "Token de acesso inválido"}]},
                )
            return httpx.Response(
                200,
                json={
                    "data": {
                        "alerts": {
                            "metadata": {
                                "currentPage": 1,
                                "limitValue": 20,
                                "totalCount": 0,
                                "totalPages": 0,
                            },
                            "collection": [],
                        }
                    }
                },
            )

        respx.post(GRAPHQL_URL).mock(side_effect=handler)
        resultado = await client.buscar_alertas(territorio_id=1)
        assert resultado.alertas == []
        assert chamadas == {"signin": 2, "consultas": 2}


# ---------------------------------------------------------------------------
# consultar_alerta
# ---------------------------------------------------------------------------


class TestConsultarAlerta:
    @pytest.mark.asyncio
    @respx.mock
    async def test_parses_detail(self) -> None:
        respx.post(GRAPHQL_URL).mock(
            side_effect=_handler(
                {
                    "alert(alertCode": {
                        "data": {
                            "alert": {
                                "alertCode": 123456,
                                "areaHa": 11.6,
                                "detectedAt": "2024-05-10",
                                "crossedIndigenousLands": ["TI Exemplo"],
                                "crossedIndigenousLandsArea": 3.2,
                                "ruralPropertiesCodes": ["GO-5211909-EXEMPLO"],
                                "ruralPropertiesTotal": 1,
                                "geometryWkt": "POLYGON((-51 -18, -51 -17, -50 -17, -51 -18))",
                                "datum": "SIRGAS2000",
                            }
                        }
                    }
                }
            )
        )
        alerta = await client.consultar_alerta(123456)
        assert alerta is not None
        assert alerta.terras_indigenas == ["TI Exemplo"]
        assert alerta.terras_indigenas_area_ha == 3.2
        assert alerta.geometria_wkt is not None
        assert alerta.datum == "SIRGAS2000"

    @pytest.mark.asyncio
    @respx.mock
    async def test_not_found(self) -> None:
        respx.post(GRAPHQL_URL).mock(
            side_effect=_handler({"alert(alertCode": {"data": {"alert": None}}})
        )
        assert await client.consultar_alerta(999999) is None


# ---------------------------------------------------------------------------
# buscar_alertas_por_car
# ---------------------------------------------------------------------------


class TestBuscarAlertasPorCar:
    @pytest.mark.asyncio
    @respx.mock
    async def test_parses_property_and_sums_area(self) -> None:
        respx.post(GRAPHQL_URL).mock(
            side_effect=_handler(
                {
                    "ruralProperty": {
                        "data": {
                            "ruralProperty": {
                                "propertyCode": "GO-5211909-EXEMPLO",
                                "propertyType": "IRU",
                                "stateAcronym": "GO",
                                "areaHa": 120.0,
                                "carUpdatedAt": "2024-01-01",
                                "version": 3,
                                "alerts": [
                                    {"alertCode": 1, "areaHa": 10.0, "detectedAt": "2024-01-01"},
                                    {"alertCode": 2, "areaHa": 5.5, "detectedAt": "2024-02-01"},
                                ],
                            }
                        }
                    }
                }
            )
        )
        imovel = await client.buscar_alertas_por_car("GO-5211909-EXEMPLO")
        assert imovel is not None
        assert imovel.uf == "GO"
        assert len(imovel.alertas) == 2
        assert imovel.area_alertada_ha == 15.5

    @pytest.mark.asyncio
    @respx.mock
    async def test_not_found(self) -> None:
        respx.post(GRAPHQL_URL).mock(
            side_effect=_handler({"ruralProperty": {"data": {"ruralProperty": None}}})
        )
        assert await client.buscar_alertas_por_car("CAR-INEXISTENTE") is None


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


class TestParsers:
    def test_number_ignores_bool(self) -> None:
        assert client._number(True) is None
        assert client._number(2) == 2.0

    def test_integer_from_float(self) -> None:
        assert client._integer(3.0) == 3
        assert client._integer(True) is None

    def test_list_from_non_list(self) -> None:
        assert client._list(None) == []
        assert client._list(["a", None, "b"]) == ["a", "b"]

    def test_normalizar_remove_acentos(self) -> None:
        assert client._normalizar("Goiânia") == "goiania"

    def test_reset_token(self) -> None:
        client._token = "x"
        client._reset_token()
        assert client._token is None
