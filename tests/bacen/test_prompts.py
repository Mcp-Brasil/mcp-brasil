"""Tests for Bacen prompts — economic analysis templates."""

import pytest

from mcp_brasil.bacen.prompts import analise_economica, comparar_indicadores


class TestAnaliseEconomica:
    def test_returns_string(self) -> None:
        result = analise_economica()
        assert isinstance(result, str)

    def test_default_periodo(self) -> None:
        result = analise_economica()
        assert "últimos 12 meses" in result

    def test_custom_periodo(self) -> None:
        result = analise_economica(periodo="2024")
        assert "2024" in result

    def test_contains_tool_instructions(self) -> None:
        result = analise_economica()
        assert "indicadores_atuais" in result
        assert "ultimos_valores" in result


class TestCompararIndicadores:
    def test_returns_string(self) -> None:
        result = comparar_indicadores("432,433")
        assert isinstance(result, str)

    def test_contains_codigos(self) -> None:
        result = comparar_indicadores("432,433,3698")
        assert "432" in result
        assert "433" in result
        assert "3698" in result

    def test_contains_tool_instructions(self) -> None:
        result = comparar_indicadores("432,433")
        assert "metadados_serie" in result
        assert "ultimos_valores" in result


class TestPromptsIntegration:
    @pytest.mark.asyncio
    async def test_prompts_registered(self) -> None:
        from fastmcp import Client

        from mcp_brasil.bacen.server import mcp

        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            assert "analise_economica" in names
            assert "comparar_indicadores" in names

    @pytest.mark.asyncio
    async def test_get_analise_economica_prompt(self) -> None:
        from fastmcp import Client

        from mcp_brasil.bacen.server import mcp

        async with Client(mcp) as c:
            result = await c.get_prompt("analise_economica", {})
            text = result.messages[0].content.text  # type: ignore[union-attr]
            assert "indicadores_atuais" in text
