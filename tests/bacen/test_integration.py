"""Integration tests for the bacen feature using fastmcp.Client.

These tests verify the full pipeline: server → tools → client (mocked HTTP).
"""

from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil.bacen.schemas import SerieValor
from mcp_brasil.bacen.server import mcp

CLIENT_MODULE = "mcp_brasil.bacen.client"


class TestToolsRegistered:
    @pytest.mark.asyncio
    async def test_all_8_tools_registered(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            names = {t.name for t in tool_list}
            expected = {
                "consultar_serie",
                "ultimos_valores",
                "metadados_serie",
                "series_populares",
                "buscar_serie",
                "indicadores_atuais",
                "calcular_variacao",
                "comparar_series",
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
                "data://catalogo",
                "data://categorias",
                "data://indicadores-chave",
            }
            assert expected.issubset(uris), f"Missing: {expected - uris}"

    @pytest.mark.asyncio
    async def test_catalogo_resource_returns_json(self) -> None:
        async with Client(mcp) as c:
            contents = await c.read_resource("data://catalogo")
            text = contents[0].text if hasattr(contents[0], "text") else str(contents[0])
            assert "432" in text  # Selic code should be in catalog


class TestToolExecution:
    @pytest.mark.asyncio
    async def test_consultar_serie_e2e(self) -> None:
        valores = [
            SerieValor(data="01/01/2024", valor=11.75),
            SerieValor(data="02/01/2024", valor=11.50),
        ]
        with patch(
            f"{CLIENT_MODULE}.buscar_valores", new_callable=AsyncMock, return_value=valores
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("consultar_serie", {"codigo": 432})
                assert "11,7500" in result.data

    @pytest.mark.asyncio
    async def test_buscar_serie_e2e(self) -> None:
        async with Client(mcp) as c:
            result = await c.call_tool("buscar_serie", {"termo": "selic"})
            assert "Selic" in result.data

    @pytest.mark.asyncio
    async def test_series_populares_e2e(self) -> None:
        async with Client(mcp) as c:
            result = await c.call_tool("series_populares", {})
            assert "Catálogo BCB" in result.data

    @pytest.mark.asyncio
    async def test_comparar_series_invalid_count(self) -> None:
        async with Client(mcp) as c:
            result = await c.call_tool(
                "comparar_series",
                {"codigos": [432], "data_inicial": "01/01/2024", "data_final": "01/06/2024"},
            )
            assert "entre 2 e 5" in result.data
