"""Tests for the Siscomex tool functions.

Tools are tested by mocking client functions (never HTTP).
Context is mocked via MagicMock with async methods.
"""

from __future__ import annotations

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from mcp_brasil.data.siscomex import tools
from mcp_brasil.data.siscomex.schemas import NcmItem

CLIENT_MODULE = "mcp_brasil.data.siscomex.client"


def _mock_ctx() -> MagicMock:
    ctx = MagicMock()
    ctx.info = AsyncMock()
    ctx.warning = AsyncMock()
    return ctx


# ---------------------------------------------------------------------------
# buscar_ncm
# ---------------------------------------------------------------------------


class TestBuscarNcm:
    @pytest.mark.asyncio
    async def test_formats_results_as_table(self) -> None:
        items = [
            NcmItem(
                codigo="01031000",
                descricao="Reprodutores de raça pura",
                dataInicio="2022-01-01",
            )
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_ncm",
            new_callable=AsyncMock,
            return_value=items,
        ):
            result = await tools.buscar_ncm("reprod", ctx)
        assert "01031000" in result
        assert "Reprodutores" in result

    @pytest.mark.asyncio
    async def test_empty_results_message(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_ncm",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.buscar_ncm("xyz123", ctx)
        assert "Nenhum" in result
        assert "xyz123" in result

    @pytest.mark.asyncio
    async def test_max_results_note_when_50(self) -> None:
        items = [NcmItem(codigo=str(i).zfill(8), descricao=f"Item {i}") for i in range(50)]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_ncm",
            new_callable=AsyncMock,
            return_value=items,
        ):
            result = await tools.buscar_ncm("item", ctx)
        assert "máximo 50" in result

    @pytest.mark.asyncio
    async def test_truncates_long_description(self) -> None:
        long_desc = "A" * 100
        items = [NcmItem(codigo="12345678", descricao=long_desc)]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_ncm",
            new_callable=AsyncMock,
            return_value=items,
        ):
            result = await tools.buscar_ncm("1234", ctx)
        assert "..." in result


# ---------------------------------------------------------------------------
# consultar_ncm
# ---------------------------------------------------------------------------


class TestConsultarNcm:
    @pytest.mark.asyncio
    async def test_formats_found_item(self) -> None:
        item = NcmItem(
            codigo="01031000",
            descricao="Reprodutores de raça pura",
            dataInicio="2022-01-01",
            dataFim=None,
            dataUltimaAlteracao="2024-01-01",
            tipoOrgaoAtoIni="Resolução",
            numeroAtoIni="123",
            anoAtoIni=2022,
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_ncm",
            new_callable=AsyncMock,
            return_value=item,
        ):
            result = await tools.consultar_ncm("01031000", ctx)
        assert "01031000" in result
        assert "Reprodutores" in result
        assert "Resolução nº 123/2022" in result
        assert "Em vigor" in result

    @pytest.mark.asyncio
    async def test_shows_end_date_when_expired(self) -> None:
        item = NcmItem(
            codigo="01039000",
            descricao="Outros suínos",
            dataInicio="2019-01-01",
            dataFim="2021-12-31",
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_ncm",
            new_callable=AsyncMock,
            return_value=item,
        ):
            result = await tools.consultar_ncm("01039000", ctx)
        assert "2021-12-31" in result
        assert "Em vigor" not in result

    @pytest.mark.asyncio
    async def test_not_found_message(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_ncm",
            new_callable=AsyncMock,
            return_value=None,
        ):
            result = await tools.consultar_ncm("99999999", ctx)
        assert "não encontrado" in result
        assert "99999999" in result

    @pytest.mark.asyncio
    async def test_missing_legal_act_shows_dash(self) -> None:
        item = NcmItem(
            codigo="01031000",
            descricao="Reprodutores de raça pura",
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_ncm",
            new_callable=AsyncMock,
            return_value=item,
        ):
            result = await tools.consultar_ncm("01031000", ctx)
        assert "**Ato legal:** —" in result
