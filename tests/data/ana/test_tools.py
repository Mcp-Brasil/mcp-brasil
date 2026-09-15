"""Tests for the ANA tool functions.

Tools are tested by mocking client functions (never HTTP).
Context is mocked via MagicMock with async methods.
"""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from mcp_brasil.data.ana import tools
from mcp_brasil.data.ana.schemas import DadoTelemetria, Estacao, Reservatorio
from mcp_brasil.exceptions import HttpClientError

CLIENT_MODULE = "mcp_brasil.data.ana.client"


def _mock_ctx() -> MagicMock:
    """Create a mock Context with async log methods."""
    ctx = MagicMock()
    ctx.info = AsyncMock()
    ctx.warning = AsyncMock()
    return ctx


# ---------------------------------------------------------------------------
# buscar_estacoes
# ---------------------------------------------------------------------------


class TestBuscarEstacoes:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Estacao(
                codigo_estacao="60435000",
                nome_estacao="Itaipu",
                nome_rio="Rio Paraná",
                municipio="Foz do Iguaçu",
                estado="PR",
                tipo_estacao="Fluviométrica",
            ),
            Estacao(
                codigo_estacao="12345678",
                nome_estacao="Sobradinho",
                nome_rio="São Francisco",
                municipio="Sobradinho",
                estado="BA",
                tipo_estacao="Fluviométrica",
            ),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_estacoes",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_estacoes(ctx)
        assert "60435000" in result
        assert "Itaipu" in result
        assert "Rio Paraná" in result
        assert "PR" in result
        assert "2 encontradas" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_estacoes",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.buscar_estacoes(ctx)
        assert "Nenhuma estação" in result

    @pytest.mark.asyncio
    async def test_with_filters(self) -> None:
        mock_data = [
            Estacao(
                codigo_estacao="60435000",
                nome_estacao="Itaipu",
            ),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_estacoes",
            new_callable=AsyncMock,
            return_value=mock_data,
        ) as mock_fn:
            result = await tools.buscar_estacoes(ctx, codigo_estacao="60435000", tipo_estacao=1)
        mock_fn.assert_awaited_once_with(
            codigo_estacao="60435000",
            codigo_rio=None,
            codigo_bacia=None,
            codigo_sub_bacia=None,
            nome_estacao=None,
            tipo_estacao=1,
        )
        assert "Itaipu" in result


# ---------------------------------------------------------------------------
# consultar_telemetria
# ---------------------------------------------------------------------------


class TestConsultarTelemetria:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            DadoTelemetria(
                codigo_estacao="60435000",
                data_hora="2024-03-15 12:00:00",
                nivel=220.5,
                vazao=8500.0,
                chuva=0.0,
            ),
            DadoTelemetria(
                codigo_estacao="60435000",
                data_hora="2024-03-15 13:00:00",
                nivel=221.0,
                vazao=8600.0,
                chuva=1.5,
            ),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_telemetria",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_telemetria(ctx, codigo_estacao="60435000")
        assert "60435000" in result
        assert "220,50" in result
        assert "8.500,0" in result
        assert "2 leituras" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_telemetria",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.consultar_telemetria(ctx, codigo_estacao="99999999")
        assert "Nenhum dado telemétrico" in result

    @pytest.mark.asyncio
    async def test_null_values(self) -> None:
        mock_data = [
            DadoTelemetria(
                codigo_estacao="60435000",
                data_hora="2024-03-15 12:00:00",
                nivel=None,
                vazao=None,
                chuva=None,
            ),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_telemetria",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_telemetria(ctx, codigo_estacao="60435000")
        # Null values rendered as "—"
        assert result.count("—") >= 3


# ---------------------------------------------------------------------------
# monitorar_reservatorios
# ---------------------------------------------------------------------------


class TestMonitorarReservatorios:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Reservatorio(
                nome_reservatorio="Sobradinho",
                rio="São Francisco",
                estado="BA",
                data="2024-03-15",
                volume_util=65.3,
                vazao_afluente=1200.0,
                vazao_defluente=1100.0,
            ),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.monitorar_reservatorios",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.monitorar_reservatorios(ctx)
        assert "Sobradinho" in result
        assert "São Francisco" in result
        assert "BA" in result
        assert "65,3%" in result
        assert "1.200,0" in result
        assert "1 registros" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.monitorar_reservatorios",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.monitorar_reservatorios(ctx)
        assert "Nenhum dado de reservatório" in result

    @pytest.mark.asyncio
    async def test_null_values(self) -> None:
        mock_data = [
            Reservatorio(
                nome_reservatorio="Teste",
                volume_util=None,
                vazao_afluente=None,
                vazao_defluente=None,
            ),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.monitorar_reservatorios",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.monitorar_reservatorios(ctx)
        assert result.count("—") >= 3

    @pytest.mark.asyncio
    async def test_with_filters(self) -> None:
        mock_data = [
            Reservatorio(
                nome_reservatorio="Sobradinho",
                rio="São Francisco",
                estado="BA",
                data="2024-03-15",
                volume_util=65.3,
            ),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.monitorar_reservatorios",
            new_callable=AsyncMock,
            return_value=mock_data,
        ) as mock_fn:
            await tools.monitorar_reservatorios(
                ctx,
                codigo_reservatorio="ABC",
                data_inicio="01/01/2024",
                data_fim="31/03/2024",
            )
        mock_fn.assert_awaited_once_with(
            codigo_reservatorio="ABC",
            data_inicio="01/01/2024",
            data_fim="31/03/2024",
        )


# ---------------------------------------------------------------------------
# API failures (issue #37)
# ---------------------------------------------------------------------------


class TestApiFailures:
    @pytest.mark.asyncio
    async def test_401_points_to_token_env_var(self) -> None:
        """HTTP 401 surfaces an actionable auth hint, not a raw error."""
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_estacoes",
            new_callable=AsyncMock,
            side_effect=HttpClientError("HTTP 401 from https://...: Unauthorized"),
        ):
            result = await tools.buscar_estacoes(ctx, codigo_estacao="60435000")
        assert "401" in result
        assert "ANA_HIDROWEB_TOKEN" in result

    @pytest.mark.asyncio
    async def test_generic_failure_reports_outage(self) -> None:
        """Non-auth failures report an outage instead of crashing."""
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.monitorar_reservatorios",
            new_callable=AsyncMock,
            side_effect=HttpClientError("HTTP 503 from https://...: Unavailable"),
        ):
            result = await tools.monitorar_reservatorios(ctx)
        assert "indisponível" in result
        assert "Nenhum dado de reservatório" not in result
