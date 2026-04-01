"""Integration tests for the Câmara feature using fastmcp.Client.

These tests verify the full pipeline: server → tools → client (mocked HTTP).
"""

from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil.data.camara.schemas import Deputado, Proposicao, VotoNominal
from mcp_brasil.data.camara.server import mcp

CLIENT_MODULE = "mcp_brasil.data.camara.client"


class TestToolsRegistered:
    @pytest.mark.asyncio
    async def test_all_11_tools_registered(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            names = {t.name for t in tool_list}
            expected = {
                "listar_deputados",
                "buscar_deputado",
                "buscar_proposicao",
                "detalhar_proposicao",
                "consultar_tramitacao",
                "buscar_votacao",
                "votos_nominais",
                "despesas_deputado",
                "agenda_legislativa",
                "buscar_comissoes",
                "frentes_parlamentares",
            }
            assert expected.issubset(names), f"Missing: {expected - names}"

    @pytest.mark.asyncio
    async def test_tools_have_docstrings(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            for tool in tool_list:
                assert tool.description, f"Tool {tool.name} has no description"


class TestResourcesRegistered:
    @pytest.mark.asyncio
    async def test_all_3_resources_registered(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            expected = {"data://tipos-proposicao", "data://legislaturas", "data://info-api"}
            assert expected.issubset(uris), f"Missing: {expected - uris}"


class TestPromptsRegistered:
    @pytest.mark.asyncio
    async def test_all_3_prompts_registered(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            expected = {"acompanhar_proposicao", "perfil_deputado", "analise_votacao"}
            assert expected.issubset(names), f"Missing: {expected - names}"


class TestToolExecution:
    @pytest.mark.asyncio
    async def test_listar_deputados_e2e(self) -> None:
        mock_data = [Deputado(id=204554, nome="Dep. E2E", sigla_partido="PT", sigla_uf="SP")]
        with patch(
            f"{CLIENT_MODULE}.listar_deputados",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("listar_deputados", {})
                assert "Dep. E2E" in result.data

    @pytest.mark.asyncio
    async def test_buscar_proposicao_e2e(self) -> None:
        mock_data = [
            Proposicao(
                sigla_tipo="PL",
                numero=1234,
                ano=2024,
                ementa="Proposição E2E",
            )
        ]
        with patch(
            f"{CLIENT_MODULE}.buscar_proposicoes",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "buscar_proposicao",
                    {"sigla_tipo": "PL", "ano": 2024, "id_deputado_autor": 204554},
                )
                assert "Proposição E2E" in result.data

    @pytest.mark.asyncio
    async def test_detalhar_proposicao_e2e(self) -> None:
        mock_data = Proposicao(
            id=2300001,
            sigla_tipo="PL",
            numero=1234,
            ano=2024,
            ementa="Proposição detalhe E2E",
            autor="Dep. Autor",
            situacao="Tramitando",
        )
        with patch(
            f"{CLIENT_MODULE}.obter_proposicao",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("detalhar_proposicao", {"proposicao_id": 2300001})
                assert "Proposição detalhe E2E" in result.data
                assert "Dep. Autor" in result.data

    @pytest.mark.asyncio
    async def test_votos_nominais_e2e(self) -> None:
        mock_data = [VotoNominal(deputado_nome="Dep. Voto", partido="PL", uf="RJ", voto="Sim")]
        with patch(
            f"{CLIENT_MODULE}.obter_votos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("votos_nominais", {"votacao_id": "vot-123"})
                assert "Dep. Voto" in result.data
