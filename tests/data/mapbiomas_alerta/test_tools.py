"""Tests for the MapBiomas Alerta tool functions."""

from __future__ import annotations

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from mcp_brasil.data.mapbiomas_alerta import tools
from mcp_brasil.data.mapbiomas_alerta.schemas import (
    Alerta,
    ImovelRural,
    ResultadoAlertas,
    Territorio,
)
from mcp_brasil.exceptions import AuthError, HttpClientError

CLIENT_MODULE = "mcp_brasil.data.mapbiomas_alerta.client"


def _mock_ctx() -> MagicMock:
    ctx = MagicMock()
    ctx.info = AsyncMock()
    ctx.warning = AsyncMock()
    return ctx


# ---------------------------------------------------------------------------
# listar_territorios
# ---------------------------------------------------------------------------


class TestListarTerritorios:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        dados = [
            Territorio(
                codigo=5211909,
                nome="Jataí",
                categoria="city",
                categoria_nome="Município",
            )
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_territorios",
            new_callable=AsyncMock,
            return_value=dados,
        ):
            result = await tools.listar_territorios(ctx, busca="jatai")
        assert "5211909" in result
        assert "Jataí" in result
        assert "Município" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_territorios",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.listar_territorios(ctx)
        assert "Nenhum território encontrado" in result


# ---------------------------------------------------------------------------
# buscar_alertas
# ---------------------------------------------------------------------------


class TestBuscarAlertas:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        dados = ResultadoAlertas(
            alertas=[
                Alerta(
                    codigo=123456,
                    area_ha=11.6,
                    detectado_em="2024-05-10",
                    publicado_em="2024-07-01",
                    municipios=["Jataí"],
                    estados=["GO"],
                    app_area_ha=2.5,
                    reserva_legal_area_ha=4.0,
                    car_total=1,
                )
            ],
            total=1,
            total_paginas=1,
            pagina=1,
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_alertas",
            new_callable=AsyncMock,
            return_value=dados,
        ):
            result = await tools.buscar_alertas(ctx, territorio_id=5211909)
        assert "123456" in result
        assert "11,60 ha" in result
        assert "Jataí, GO" in result

    @pytest.mark.asyncio
    async def test_empty_does_not_claim_inexistence(self) -> None:
        dados = ResultadoAlertas(alertas=[], total=0, total_paginas=0, pagina=1)
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_alertas",
            new_callable=AsyncMock,
            return_value=dados,
        ):
            result = await tools.buscar_alertas(ctx, territorio_id=1)
        assert "Nenhum alerta encontrado para estes filtros" in result
        assert "não prova ausência de supressão" in result

    @pytest.mark.asyncio
    async def test_shows_next_page(self) -> None:
        dados = ResultadoAlertas(
            alertas=[Alerta(codigo=1, area_ha=1.0)],
            total=40,
            total_paginas=2,
            pagina=1,
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_alertas",
            new_callable=AsyncMock,
            return_value=dados,
        ):
            result = await tools.buscar_alertas(ctx)
        assert "página=2" in result

    @pytest.mark.asyncio
    async def test_auth_error_message(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_alertas",
            new_callable=AsyncMock,
            side_effect=AuthError("Credenciais do MapBiomas Alerta não configuradas."),
        ):
            result = await tools.buscar_alertas(ctx)
        assert "Não foi possível consultar o MapBiomas Alerta" in result
        assert "não configuradas" in result

    @pytest.mark.asyncio
    async def test_http_error_message(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_alertas",
            new_callable=AsyncMock,
            side_effect=HttpClientError("timeout"),
        ):
            result = await tools.buscar_alertas(ctx)
        assert "Não foi possível consultar o MapBiomas Alerta" in result


# ---------------------------------------------------------------------------
# consultar_alerta
# ---------------------------------------------------------------------------


class TestConsultarAlerta:
    @pytest.mark.asyncio
    async def test_formats_detail(self) -> None:
        alerta = Alerta(
            codigo=123456,
            area_ha=11.6,
            detectado_em="2024-05-10",
            publicado_em="2024-07-01",
            status="Publicado",
            fontes=["DETER Cerrado"],
            biomas=["Cerrado"],
            municipios=["Jataí"],
            estados=["GO"],
            app_area_ha=2.5,
            reserva_legal_area_ha=4.0,
            terras_indigenas=["TI Exemplo"],
            terras_indigenas_area_ha=3.2,
            embargos_total=1,
            car_codigos=["GO-5211909-EXEMPLO"],
            car_total=1,
            imagem_antes_em="2024-01-01",
            imagem_depois_em="2024-05-10",
            geometria_wkt="POLYGON((-51 -18, -51 -17, -50 -17, -51 -18))",
            datum="SIRGAS2000",
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_alerta",
            new_callable=AsyncMock,
            return_value=alerta,
        ):
            result = await tools.consultar_alerta(ctx, 123456)
        assert "Alerta MapBiomas 123456" in result
        assert "Terra Indígena" in result
        assert "GO-5211909-EXEMPLO" in result
        assert "SIRGAS2000" in result
        assert "não somar as áreas" in result

    @pytest.mark.asyncio
    async def test_without_geometry(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_alerta",
            new_callable=AsyncMock,
            return_value=Alerta(codigo=1, area_ha=1.0),
        ):
            result = await tools.consultar_alerta(ctx, 1)
        assert "Geometria:** não disponível" in result

    @pytest.mark.asyncio
    async def test_not_found(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_alerta",
            new_callable=AsyncMock,
            return_value=None,
        ):
            result = await tools.consultar_alerta(ctx, 999999)
        assert "Nenhum alerta encontrado para o código 999999" in result


# ---------------------------------------------------------------------------
# buscar_alertas_por_car
# ---------------------------------------------------------------------------


class TestBuscarAlertasPorCar:
    @pytest.mark.asyncio
    async def test_formats_property(self) -> None:
        imovel = ImovelRural(
            codigo="GO-5211909-EXEMPLO",
            tipo="IRU",
            uf="GO",
            area_ha=120.0,
            alertas=[Alerta(codigo=1, area_ha=10.0, detectado_em="2024-01-01")],
            area_alertada_ha=10.0,
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_alertas_por_car",
            new_callable=AsyncMock,
            return_value=imovel,
        ):
            result = await tools.buscar_alertas_por_car(ctx, "GO-5211909-EXEMPLO")
        assert "GO-5211909-EXEMPLO" in result
        assert "10,00 ha" in result

    @pytest.mark.asyncio
    async def test_no_alerts(self) -> None:
        imovel = ImovelRural(codigo="GO-5211909-EXEMPLO", uf="GO", alertas=[])
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_alertas_por_car",
            new_callable=AsyncMock,
            return_value=imovel,
        ):
            result = await tools.buscar_alertas_por_car(ctx, "GO-5211909-EXEMPLO")
        assert "Nenhum alerta encontrado para este imóvel" in result

    @pytest.mark.asyncio
    async def test_property_not_found(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_alertas_por_car",
            new_callable=AsyncMock,
            return_value=None,
        ):
            result = await tools.buscar_alertas_por_car(ctx, "CAR-INEXISTENTE")
        assert "Nenhum imóvel encontrado" in result
