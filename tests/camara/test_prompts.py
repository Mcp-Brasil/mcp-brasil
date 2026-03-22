"""Tests for Câmara prompts — analysis templates."""

import pytest

from mcp_brasil.camara.prompts import acompanhar_proposicao, analise_votacao, perfil_deputado


class TestAcompanharProposicao:
    def test_returns_string(self) -> None:
        result = acompanhar_proposicao("PL", 1234, 2024)
        assert isinstance(result, str)

    def test_contains_proposicao(self) -> None:
        result = acompanhar_proposicao("PL", 1234, 2024)
        assert "PL 1234/2024" in result

    def test_contains_tool_instructions(self) -> None:
        result = acompanhar_proposicao("PL", 1234, 2024)
        assert "buscar_proposicao" in result
        assert "consultar_tramitacao" in result
        assert "buscar_votacao" in result
        assert "votos_nominais" in result


class TestPerfilDeputado:
    def test_returns_string(self) -> None:
        result = perfil_deputado(204554)
        assert isinstance(result, str)

    def test_contains_deputado_id(self) -> None:
        result = perfil_deputado(204554)
        assert "204554" in result

    def test_contains_tool_instructions(self) -> None:
        result = perfil_deputado(204554)
        assert "buscar_deputado" in result
        assert "despesas_deputado" in result


class TestAnaliseVotacao:
    def test_returns_string(self) -> None:
        result = analise_votacao("vot-123")
        assert isinstance(result, str)

    def test_contains_votacao_id(self) -> None:
        result = analise_votacao("vot-123")
        assert "vot-123" in result

    def test_contains_tool_instructions(self) -> None:
        result = analise_votacao("vot-123")
        assert "votos_nominais" in result


class TestPromptsIntegration:
    @pytest.mark.asyncio
    async def test_prompts_registered(self) -> None:
        from fastmcp import Client

        from mcp_brasil.camara.server import mcp

        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            assert "acompanhar_proposicao" in names
            assert "perfil_deputado" in names
            assert "analise_votacao" in names

    @pytest.mark.asyncio
    async def test_get_acompanhar_prompt(self) -> None:
        from fastmcp import Client

        from mcp_brasil.camara.server import mcp

        async with Client(mcp) as c:
            result = await c.get_prompt(
                "acompanhar_proposicao",
                {"sigla_tipo": "PL", "numero": "1234", "ano": "2024"},
            )
            text = result.messages[0].content.text  # type: ignore[union-attr]
            assert "PL 1234/2024" in text
