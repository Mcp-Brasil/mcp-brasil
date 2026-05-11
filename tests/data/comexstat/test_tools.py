"""Tests for the ComexStat tool functions.

Tools are tested by mocking client functions (never HTTP).
Context is mocked via MagicMock with async methods.
"""

from __future__ import annotations

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from mcp_brasil.data.comexstat import tools
from mcp_brasil.data.comexstat.schemas import BalancaItem, ComexItem, PaisItem

CLIENT_MODULE = "mcp_brasil.data.comexstat.client"


def _mock_ctx() -> MagicMock:
    ctx = MagicMock()
    ctx.info = AsyncMock()
    ctx.warning = AsyncMock()
    return ctx


# ---------------------------------------------------------------------------
# consultar_exportacoes
# ---------------------------------------------------------------------------


class TestConsultarExportacoes:
    @pytest.mark.asyncio
    async def test_formats_as_table(self) -> None:
        itens = [
            ComexItem(
                periodo="2023-01",
                pais="CHINA",
                ncm="01031000",
                fob_usd=1_000_000.0,
                kg_liquido=500_000.0,
            )
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_exportacoes",
            new_callable=AsyncMock,
            return_value=itens,
        ):
            result = await tools.consultar_exportacoes("2023-01", "2023-01", ctx)
        assert "2023-01" in result
        assert "CHINA" in result
        assert "01031000" in result
        assert "US$" in result
        assert "kg" in result

    @pytest.mark.asyncio
    async def test_empty_returns_message(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_exportacoes",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.consultar_exportacoes("2023-01", "2023-01", ctx)
        assert "Nenhum" in result

    @pytest.mark.asyncio
    async def test_none_pais_shows_dash(self) -> None:
        itens = [
            ComexItem(periodo="2023-01", pais=None, ncm=None, fob_usd=500.0, kg_liquido=100.0)
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_exportacoes",
            new_callable=AsyncMock,
            return_value=itens,
        ):
            result = await tools.consultar_exportacoes("2023-01", "2023-01", ctx)
        assert "—" in result

    @pytest.mark.asyncio
    async def test_header_contains_periodo_range(self) -> None:
        itens = [ComexItem(periodo="2023-01", fob_usd=0.0, kg_liquido=0.0)]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_exportacoes",
            new_callable=AsyncMock,
            return_value=itens,
        ):
            result = await tools.consultar_exportacoes("2023-01", "2023-06", ctx)
        assert "2023-01" in result
        assert "2023-06" in result


# ---------------------------------------------------------------------------
# consultar_importacoes
# ---------------------------------------------------------------------------


class TestConsultarImportacoes:
    @pytest.mark.asyncio
    async def test_formats_as_table(self) -> None:
        itens = [
            ComexItem(
                periodo="2023-03",
                pais="EUA",
                ncm="84713012",
                fob_usd=2_500_000.0,
                kg_liquido=10_000.0,
            )
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_importacoes",
            new_callable=AsyncMock,
            return_value=itens,
        ):
            result = await tools.consultar_importacoes("2023-01", "2023-03", ctx)
        assert "2023-03" in result
        assert "EUA" in result
        assert "Importações" in result

    @pytest.mark.asyncio
    async def test_empty_returns_message(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_importacoes",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.consultar_importacoes("2023-01", "2023-01", ctx)
        assert "Nenhum" in result


# ---------------------------------------------------------------------------
# balanca_comercial
# ---------------------------------------------------------------------------


class TestBalancaComercial:
    @pytest.mark.asyncio
    async def test_surplus_shows_plus_sign(self) -> None:
        itens = [
            BalancaItem(
                periodo="2023-01",
                exportacoes_fob=1_000_000.0,
                importacoes_fob=600_000.0,
                saldo_fob=400_000.0,
            )
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.balanca_comercial",
            new_callable=AsyncMock,
            return_value=itens,
        ):
            result = await tools.balanca_comercial("2023-01", "2023-01", ctx)
        assert "+US$" in result

    @pytest.mark.asyncio
    async def test_deficit_shows_negative(self) -> None:
        itens = [
            BalancaItem(
                periodo="2023-01",
                exportacoes_fob=300_000.0,
                importacoes_fob=800_000.0,
                saldo_fob=-500_000.0,
            )
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.balanca_comercial",
            new_callable=AsyncMock,
            return_value=itens,
        ):
            result = await tools.balanca_comercial("2023-01", "2023-01", ctx)
        assert "US$ -" in result or "-US$" in result or "500" in result

    @pytest.mark.asyncio
    async def test_empty_returns_message(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.balanca_comercial",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.balanca_comercial("2023-01", "2023-01", ctx)
        assert "Nenhum" in result

    @pytest.mark.asyncio
    async def test_header_contains_balanca(self) -> None:
        itens = [
            BalancaItem(periodo="2023-01", exportacoes_fob=0.0, importacoes_fob=0.0, saldo_fob=0.0)
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.balanca_comercial",
            new_callable=AsyncMock,
            return_value=itens,
        ):
            result = await tools.balanca_comercial("2023-01", "2023-03", ctx)
        assert "Balança" in result
        assert "2023-01" in result
        assert "2023-03" in result


# ---------------------------------------------------------------------------
# listar_paises
# ---------------------------------------------------------------------------


class TestListarPaises:
    @pytest.mark.asyncio
    async def test_formats_table_with_id_and_nome(self) -> None:
        paises = [
            PaisItem(id="31", nome="CHINA"),
            PaisItem(id="249", nome="ESTADOS UNIDOS"),
        ]
        ctx = _mock_ctx()
        with patch(f"{CLIENT_MODULE}.listar_paises", new_callable=AsyncMock, return_value=paises):
            result = await tools.listar_paises(ctx)
        assert "31" in result
        assert "CHINA" in result
        assert "249" in result
        assert "ESTADOS UNIDOS" in result

    @pytest.mark.asyncio
    async def test_empty_returns_message(self) -> None:
        ctx = _mock_ctx()
        with patch(f"{CLIENT_MODULE}.listar_paises", new_callable=AsyncMock, return_value=[]):
            result = await tools.listar_paises(ctx)
        assert "Nenhum" in result

    @pytest.mark.asyncio
    async def test_header_shows_count(self) -> None:
        paises = [PaisItem(id=str(i), nome=f"País {i}") for i in range(5)]
        ctx = _mock_ctx()
        with patch(f"{CLIENT_MODULE}.listar_paises", new_callable=AsyncMock, return_value=paises):
            result = await tools.listar_paises(ctx)
        assert "5" in result
