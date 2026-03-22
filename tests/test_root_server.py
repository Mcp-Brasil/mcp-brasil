"""Integration tests for the root mcp-brasil server.

Tests the fully composed server with all features mounted.
"""

import pytest
from fastmcp import Client

from mcp_brasil.server import mcp


class TestRootServerTools:
    @pytest.mark.asyncio
    async def test_listar_features_registered(self) -> None:
        async with Client(mcp) as c:
            tools = await c.list_tools()
            names = {t.name for t in tools}
            assert "listar_features" in names

    @pytest.mark.asyncio
    async def test_ibge_tools_namespaced(self) -> None:
        async with Client(mcp) as c:
            tools = await c.list_tools()
            names = {t.name for t in tools}
            assert "ibge_listar_estados" in names
            assert "ibge_buscar_municipios" in names

    @pytest.mark.asyncio
    async def test_bacen_tools_namespaced(self) -> None:
        async with Client(mcp) as c:
            tools = await c.list_tools()
            names = {t.name for t in tools}
            assert "bacen_consultar_serie" in names
            assert "bacen_buscar_serie" in names

    @pytest.mark.asyncio
    async def test_listar_features_returns_summary(self) -> None:
        async with Client(mcp) as c:
            result = await c.call_tool("listar_features", {})
            assert "ibge" in result.data
            assert "bacen" in result.data


class TestRootServerResources:
    @pytest.mark.asyncio
    async def test_ibge_resources_namespaced(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            assert "data://ibge/estados" in uris
            assert "data://ibge/regioes" in uris
            assert "data://ibge/niveis-territoriais" in uris

    @pytest.mark.asyncio
    async def test_bacen_resources_namespaced(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            assert "data://bacen/catalogo" in uris
            assert "data://bacen/categorias" in uris
            assert "data://bacen/indicadores-chave" in uris


class TestRootServerPrompts:
    @pytest.mark.asyncio
    async def test_ibge_prompts_namespaced(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            assert "ibge_resumo_estado" in names
            assert "ibge_comparativo_regional" in names

    @pytest.mark.asyncio
    async def test_bacen_prompts_namespaced(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            assert "bacen_analise_economica" in names
            assert "bacen_comparar_indicadores" in names
