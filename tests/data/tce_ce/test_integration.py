"""Integration tests for the TCE-CE feature using fastmcp.Client."""

from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil.data.tce_ce.schemas import LicitacaoResultado, Municipio
from mcp_brasil.data.tce_ce.server import mcp

CLIENT_MODULE = "mcp_brasil.data.tce_ce.client"


class TestToolsRegistered:
    @pytest.mark.asyncio
    async def test_all_4_tools_registered(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            names = {t.name for t in tool_list}
            expected = {
                "listar_municipios_ce",
                "buscar_licitacoes_ce",
                "buscar_contratos_ce",
                "buscar_empenhos_ce",
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
    async def test_endpoints_resource(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            assert "data://endpoints" in uris, f"URIs: {uris}"

    @pytest.mark.asyncio
    async def test_endpoints_content(self) -> None:
        async with Client(mcp) as c:
            content = await c.read_resource("data://endpoints")
            text = content[0].text if isinstance(content, list) else str(content)
            assert "municipios" in text
            assert "licitacoes" in text
            assert "contrato" in text
            assert "notas_empenhos" in text


class TestPromptsRegistered:
    @pytest.mark.asyncio
    async def test_analisar_municipio_prompt(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            assert "analisar_municipio_ce" in names, f"Prompts: {names}"


class TestToolExecution:
    @pytest.mark.asyncio
    async def test_listar_municipios_e2e(self) -> None:
        mock_data = [
            Municipio(codigo_municipio="057", nome_municipio="FORTALEZA"),
        ]
        with patch(
            f"{CLIENT_MODULE}.listar_municipios",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("listar_municipios_ce", {})
                assert "FORTALEZA" in result.data

    @pytest.mark.asyncio
    async def test_buscar_licitacoes_empty_e2e(self) -> None:
        with patch(
            f"{CLIENT_MODULE}.buscar_licitacoes",
            new_callable=AsyncMock,
            return_value=LicitacaoResultado(),
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "buscar_licitacoes_ce",
                    {"codigo_municipio": "057", "data_realizacao": "2024-01-01"},
                )
                assert "Nenhuma licitação" in result.data
