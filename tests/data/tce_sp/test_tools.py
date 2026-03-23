"""Tests for the TCE-SP tool functions."""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from mcp_brasil.data.tce_sp import tools
from mcp_brasil.data.tce_sp.schemas import Despesa, Municipio, Receita

CLIENT_MODULE = "mcp_brasil.data.tce_sp.client"


def _mock_ctx() -> MagicMock:
    ctx = MagicMock()
    ctx.info = AsyncMock()
    ctx.warning = AsyncMock()
    return ctx


# ---------------------------------------------------------------------------
# listar_municipios_sp
# ---------------------------------------------------------------------------


class TestListarMunicipiosSp:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = [
            Municipio(municipio="campinas", municipio_extenso="Campinas"),
            Municipio(municipio="sao-paulo", municipio_extenso="São Paulo"),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_municipios",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.listar_municipios_sp(ctx)
        assert "Campinas" in result
        assert "`campinas`" in result
        assert "São Paulo" in result
        assert "2 municípios" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_municipios",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.listar_municipios_sp(ctx)
        assert "Nenhum município encontrado" in result


# ---------------------------------------------------------------------------
# consultar_despesas_sp
# ---------------------------------------------------------------------------


class TestConsultarDespesasSp:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = [
            Despesa(
                orgao="PREFEITURA MUNICIPAL",
                mes="Janeiro",
                evento="Empenhado",
                nr_empenho="110-2025",
                nm_fornecedor="EMPRESA X LTDA",
                vl_despesa=5034.11,
            ),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_despesas",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_despesas_sp(ctx, "balsamo", 2025, 1)
        assert "EMPRESA X LTDA" in result
        assert "R$ 5.034,11" in result
        assert "110-2025" in result
        assert "Empenhado" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_despesas",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.consultar_despesas_sp(ctx, "balsamo", 2025, 1)
        assert "Nenhuma despesa encontrada" in result


# ---------------------------------------------------------------------------
# consultar_receitas_sp
# ---------------------------------------------------------------------------


class TestConsultarReceitasSp:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = [
            Receita(
                orgao="PREFEITURA MUNICIPAL",
                mes="Janeiro",
                ds_fonte_recurso="01 - TESOURO",
                ds_alinea="13210000 - Juros e Correções Monetárias",
                vl_arrecadacao=2314.94,
            ),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_receitas",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_receitas_sp(ctx, "balsamo", 2025, 1)
        assert "R$ 2.314,94" in result
        assert "TESOURO" in result
        assert "Juros" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_receitas",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.consultar_receitas_sp(ctx, "balsamo", 2025, 1)
        assert "Nenhuma receita encontrada" in result
