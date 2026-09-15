"""Tests for the ANA HTTP client."""

import httpx
import pytest
import respx

from mcp_brasil.data.ana import client
from mcp_brasil.data.ana.constants import (
    AUTH_ENV_VAR,
    ESTACOES_URL,
    RESERVATORIOS_URL,
    TELEMETRIA_URL,
)

# ---------------------------------------------------------------------------
# buscar_estacoes
# ---------------------------------------------------------------------------


class TestBuscarEstacoes:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_stations(self) -> None:
        respx.get(ESTACOES_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "codigoEstacao": "60435000",
                        "nomeEstacao": "Itaipu",
                        "codigoRio": "123",
                        "nomeRio": "Rio Paraná",
                        "bacia": "Paraná",
                        "subBacia": "Alto Paraná",
                        "municipio": "Foz do Iguaçu",
                        "estado": "PR",
                        "latitude": -25.4372,
                        "longitude": -54.5892,
                        "tipoEstacao": "Fluviométrica",
                        "responsavel": "ANA",
                    }
                ],
            )
        )
        result = await client.buscar_estacoes(codigo_estacao="60435000")
        assert len(result) == 1
        assert result[0].codigo_estacao == "60435000"
        assert result[0].nome_estacao == "Itaipu"
        assert result[0].nome_rio == "Rio Paraná"
        assert result[0].estado == "PR"
        assert result[0].latitude == -25.4372

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(ESTACOES_URL).mock(return_value=httpx.Response(200, json=[]))
        result = await client.buscar_estacoes()
        assert result == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_with_type_filter(self) -> None:
        route = respx.get(ESTACOES_URL).mock(return_value=httpx.Response(200, json=[]))
        await client.buscar_estacoes(tipo_estacao=1)
        req_url = str(route.calls[0].request.url)
        assert "tipoEstacao=1" in req_url

    @pytest.mark.asyncio
    @respx.mock
    async def test_with_multiple_filters(self) -> None:
        route = respx.get(ESTACOES_URL).mock(return_value=httpx.Response(200, json=[]))
        await client.buscar_estacoes(codigo_bacia="7", nome_estacao="Itaipu")
        req_url = str(route.calls[0].request.url)
        assert "codigoBacia=7" in req_url
        assert "nomeEstacao=Itaipu" in req_url


# ---------------------------------------------------------------------------
# consultar_telemetria
# ---------------------------------------------------------------------------


class TestConsultarTelemetria:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_telemetry(self) -> None:
        respx.get(TELEMETRIA_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "codigoEstacao": "60435000",
                        "dataHora": "2024-03-15 12:00:00",
                        "nivel": 220.5,
                        "vazao": 8500.0,
                        "chuva": 0.0,
                    }
                ],
            )
        )
        result = await client.consultar_telemetria("60435000")
        assert len(result) == 1
        assert result[0].codigo_estacao == "60435000"
        assert result[0].nivel == 220.5
        assert result[0].vazao == 8500.0
        assert result[0].chuva == 0.0

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(TELEMETRIA_URL).mock(return_value=httpx.Response(200, json=[]))
        result = await client.consultar_telemetria("60435000")
        assert result == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_with_date_range(self) -> None:
        route = respx.get(TELEMETRIA_URL).mock(return_value=httpx.Response(200, json=[]))
        await client.consultar_telemetria(
            "60435000", data_inicio="01/03/2024", data_fim="15/03/2024"
        )
        req_url = str(route.calls[0].request.url)
        assert "codigoEstacao=60435000" in req_url
        assert "dataInicio=01%2F03%2F2024" in req_url or "dataInicio=01/03/2024" in req_url
        assert "dataFim=15%2F03%2F2024" in req_url or "dataFim=15/03/2024" in req_url

    @pytest.mark.asyncio
    @respx.mock
    async def test_null_values(self) -> None:
        respx.get(TELEMETRIA_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "codigoEstacao": "60435000",
                        "dataHora": "2024-03-15 12:00:00",
                        "nivel": None,
                        "vazao": None,
                        "chuva": None,
                    }
                ],
            )
        )
        result = await client.consultar_telemetria("60435000")
        assert result[0].nivel is None
        assert result[0].vazao is None
        assert result[0].chuva is None


# ---------------------------------------------------------------------------
# monitorar_reservatorios
# ---------------------------------------------------------------------------


class TestMonitorarReservatorios:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_reservoirs(self) -> None:
        respx.get(RESERVATORIOS_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "nomeReservatorio": "Sobradinho",
                        "rio": "São Francisco",
                        "estado": "BA",
                        "data": "2024-03-15",
                        "volumeUtil": 65.3,
                        "cotaAtual": 380.5,
                        "vazaoAfluente": 1200.0,
                        "vazaoDefluente": 1100.0,
                    }
                ],
            )
        )
        result = await client.monitorar_reservatorios()
        assert len(result) == 1
        assert result[0].nome_reservatorio == "Sobradinho"
        assert result[0].rio == "São Francisco"
        assert result[0].volume_util == 65.3
        assert result[0].cota_atual == 380.5
        assert result[0].vazao_afluente == 1200.0

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(RESERVATORIOS_URL).mock(return_value=httpx.Response(200, json=[]))
        result = await client.monitorar_reservatorios()
        assert result == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_with_filters(self) -> None:
        route = respx.get(RESERVATORIOS_URL).mock(return_value=httpx.Response(200, json=[]))
        await client.monitorar_reservatorios(
            codigo_reservatorio="ABC",
            data_inicio="01/01/2024",
            data_fim="31/03/2024",
        )
        req_url = str(route.calls[0].request.url)
        assert "codigoReservatorio=ABC" in req_url


# ---------------------------------------------------------------------------
# Optional auth (issue #37 — Hidroweb started requiring a token)
# ---------------------------------------------------------------------------


class TestOptionalAuth:
    @pytest.mark.asyncio
    @respx.mock
    async def test_no_auth_header_by_default(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """Without ANA_HIDROWEB_TOKEN, requests stay anonymous."""
        monkeypatch.delenv(AUTH_ENV_VAR, raising=False)
        route = respx.get(ESTACOES_URL).mock(return_value=httpx.Response(200, json=[]))
        await client.buscar_estacoes(codigo_estacao="60435000")
        assert "authorization" not in route.calls[0].request.headers

    @pytest.mark.asyncio
    @respx.mock
    async def test_sends_bearer_token_when_env_set(self, monkeypatch: pytest.MonkeyPatch) -> None:
        """With ANA_HIDROWEB_TOKEN, requests carry a Bearer credential."""
        monkeypatch.setenv(AUTH_ENV_VAR, "token-abc-123")
        route = respx.get(ESTACOES_URL).mock(return_value=httpx.Response(200, json=[]))
        await client.buscar_estacoes(codigo_estacao="60435000")
        assert route.calls[0].request.headers["authorization"] == "Bearer token-abc-123"
