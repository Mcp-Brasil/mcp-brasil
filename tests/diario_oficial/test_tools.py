"""Tests for the Diário Oficial tool functions.

Tools are tested by mocking client functions (never HTTP).
Context is mocked via MagicMock with async methods.
"""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from mcp_brasil.diario_oficial import tools
from mcp_brasil.diario_oficial.schemas import (
    CidadeQueridoDiario,
    DiarioOficial,
    DiarioResultado,
)

CLIENT_MODULE = "mcp_brasil.diario_oficial.client"


def _mock_ctx() -> MagicMock:
    """Create a mock Context with async log methods."""
    ctx = MagicMock()
    ctx.info = AsyncMock()
    ctx.warning = AsyncMock()
    return ctx


# ---------------------------------------------------------------------------
# buscar_diarios
# ---------------------------------------------------------------------------


class TestBuscarDiarios:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = DiarioResultado(
            total_gazettes=2,
            gazettes=[
                DiarioOficial(
                    territory_id="3550308",
                    territory_name="São Paulo",
                    state_code="SP",
                    date="2024-01-15",
                    edition_number="1234",
                    is_extra_edition=False,
                    txt_url="https://example.com/gazette.txt",
                    excerpts=["Trecho com a palavra licitação encontrada no diário"],
                ),
                DiarioOficial(
                    territory_id="3304557",
                    territory_name="Rio de Janeiro",
                    state_code="RJ",
                    date="2024-01-14",
                    edition_number="5678",
                    is_extra_edition=True,
                    txt_url=None,
                    excerpts=[],
                ),
            ],
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_diarios",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_diarios("licitação", ctx)
        assert "São Paulo" in result
        assert "SP" in result
        assert "2024-01-15" in result
        assert "1234" in result
        assert "licitação" in result
        assert "Rio de Janeiro" in result
        assert "Edição Extra" in result
        assert "2 diários encontrados" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        mock_data = DiarioResultado(total_gazettes=0, gazettes=[])
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_diarios",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_diarios("inexistente", ctx)
        assert "Nenhum diário oficial encontrado" in result
        assert "inexistente" in result


# ---------------------------------------------------------------------------
# buscar_cidades
# ---------------------------------------------------------------------------


class TestBuscarCidades:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            CidadeQueridoDiario(
                territory_id="3550308",
                territory_name="São Paulo",
                state_code="SP",
            ),
            CidadeQueridoDiario(
                territory_id="3549904",
                territory_name="São José dos Campos",
                state_code="SP",
            ),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_cidades",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_cidades("São", ctx)
        assert "3550308" in result
        assert "São Paulo" in result
        assert "SP" in result
        assert "São José dos Campos" in result
        assert "Código IBGE" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_cidades",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.buscar_cidades("Inexistente", ctx)
        assert "Nenhuma cidade encontrada" in result
        assert "Inexistente" in result


# ---------------------------------------------------------------------------
# listar_territorios
# ---------------------------------------------------------------------------


class TestListarTerritorios:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            CidadeQueridoDiario(
                territory_id="3550308",
                territory_name="São Paulo",
                state_code="SP",
            ),
            CidadeQueridoDiario(
                territory_id="2408102",
                territory_name="Natal",
                state_code="RN",
            ),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_cidades",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.listar_territorios(ctx)
        assert "2 municípios" in result
        assert "São Paulo" in result
        assert "3550308" in result
        assert "Natal" in result
        assert "RN" in result
        assert "Código IBGE" in result

    @pytest.mark.asyncio
    async def test_pagination_hint_with_many(self) -> None:
        """When >100 cities, only first 100 are shown in table."""
        mock_data = [
            CidadeQueridoDiario(
                territory_id=str(i),
                territory_name=f"Cidade {i}",
                state_code="XX",
            )
            for i in range(150)
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_cidades",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.listar_territorios(ctx)
        assert "150 municípios" in result
        # The table is truncated to 100 rows, so Cidade 149 should not appear
        assert "Cidade 0" in result
