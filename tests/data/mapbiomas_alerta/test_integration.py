"""Integration tests for the MapBiomas Alerta feature using fastmcp.Client."""

from __future__ import annotations

from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil.data.mapbiomas_alerta.schemas import Alerta, ResultadoAlertas
from mcp_brasil.data.mapbiomas_alerta.server import mcp

CLIENT_MODULE = "mcp_brasil.data.mapbiomas_alerta.client"


class TestToolsRegistered:
    @pytest.mark.asyncio
    async def test_all_tools_registered(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            names = {t.name for t in tool_list}
            expected = {
                "listar_territorios",
                "buscar_alertas",
                "consultar_alerta",
                "buscar_alertas_por_car",
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
    async def test_resources_registered(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            expected = {
                "data://mapbiomas-alerta/info-api",
                "data://mapbiomas-alerta/categorias-territorio",
            }
            assert expected.issubset(uris), f"Missing: {expected - uris}"

    @pytest.mark.asyncio
    async def test_info_api_content(self) -> None:
        async with Client(mcp) as c:
            content = await c.read_resource("data://mapbiomas-alerta/info-api")
            text = content[0].text if isinstance(content, list) else str(content)
            assert "MapBiomas Alerta" in text
            assert "MAPBIOMAS_EMAIL" in text


class TestPromptsRegistered:
    @pytest.mark.asyncio
    async def test_prompt_registered(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            assert "analisar_alertas" in names, f"Prompts: {names}"


class TestToolExecution:
    @pytest.mark.asyncio
    async def test_buscar_alertas_e2e(self) -> None:
        mock_data = ResultadoAlertas(
            alertas=[Alerta(codigo=123456, area_ha=11.6, municipios=["Jataí"], estados=["GO"])],
            total=1,
            total_paginas=1,
            pagina=1,
        )
        with patch(
            f"{CLIENT_MODULE}.buscar_alertas",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_alertas", {"territorio_id": 5211909})
                assert "123456" in result.data

    @pytest.mark.asyncio
    async def test_buscar_alertas_empty_e2e(self) -> None:
        with patch(
            f"{CLIENT_MODULE}.buscar_alertas",
            new_callable=AsyncMock,
            return_value=ResultadoAlertas(alertas=[], total=0),
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_alertas", {})
                assert "Nenhum alerta encontrado para estes filtros" in result.data
