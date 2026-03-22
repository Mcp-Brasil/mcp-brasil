"""Tests for IBGE prompts — analysis templates."""

import pytest

from mcp_brasil.ibge.prompts import comparativo_regional, resumo_estado


class TestResumoEstado:
    def test_returns_string(self) -> None:
        result = resumo_estado("SP")
        assert isinstance(result, str)

    def test_contains_uf(self) -> None:
        result = resumo_estado("pi")
        assert "PI" in result

    def test_contains_tool_instructions(self) -> None:
        result = resumo_estado("RJ")
        assert "listar_estados" in result
        assert "buscar_municipios" in result
        assert "consultar_agregado" in result


class TestComparativoRegional:
    def test_returns_string(self) -> None:
        result = comparativo_regional()
        assert isinstance(result, str)

    def test_contains_tool_instructions(self) -> None:
        result = comparativo_regional()
        assert "listar_regioes" in result
        assert "consultar_agregado" in result

    def test_mentions_regions(self) -> None:
        result = comparativo_regional()
        assert "regiao" in result.lower() or "região" in result.lower()


class TestPromptsIntegration:
    @pytest.mark.asyncio
    async def test_prompts_registered(self) -> None:
        from fastmcp import Client

        from mcp_brasil.ibge.server import mcp

        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            assert "resumo_estado" in names
            assert "comparativo_regional" in names

    @pytest.mark.asyncio
    async def test_get_resumo_estado_prompt(self) -> None:
        from fastmcp import Client

        from mcp_brasil.ibge.server import mcp

        async with Client(mcp) as c:
            result = await c.get_prompt("resumo_estado", {"uf": "BA"})
            text = result.messages[0].content.text  # type: ignore[union-attr]
            assert "BA" in text
