"""Integration tests for the DataJud feature using fastmcp.Client."""

from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil.datajud.schemas import Processo
from mcp_brasil.datajud.server import mcp

CLIENT_MODULE = "mcp_brasil.datajud.client"


class TestToolsRegistered:
    @pytest.mark.asyncio
    async def test_all_6_tools_registered(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            names = {t.name for t in tool_list}
            expected = {
                "buscar_processos",
                "buscar_processo_por_numero",
                "buscar_processos_por_classe",
                "buscar_processos_por_assunto",
                "buscar_processos_por_orgao",
                "consultar_movimentacoes",
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
            expected = {
                "data://tribunais",
                "data://classes-processuais",
                "data://info-api",
            }
            assert expected.issubset(uris), f"Missing: {expected - uris}"


class TestPromptsRegistered:
    @pytest.mark.asyncio
    async def test_all_2_prompts_registered(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            expected = {"analise_processo", "pesquisa_juridica"}
            assert expected.issubset(names), f"Missing: {expected - names}"


class TestToolExecution:
    @pytest.mark.asyncio
    async def test_buscar_processos_e2e(self) -> None:
        mock_data = [Processo(numero="0001234", classe="Cível", tribunal="TJSP")]
        with patch(
            f"{CLIENT_MODULE}.buscar_processos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "buscar_processos",
                    {"query": "teste", "tribunal": "tjsp"},
                )
                assert "0001234" in result.data
