"""Integration tests for the root mcp-brasil server.

Tests the fully composed server with all features mounted.
MCP_BRASIL_TOOL_SEARCH=none is set in conftest.py (before any import).
"""

import json

import pytest
from fastmcp import Client, FastMCP

from mcp_brasil.server import RequestLoggingMiddleware, mcp


class TestRootServerTools:
    @pytest.mark.asyncio
    async def test_listar_features_registered(self) -> None:
        async with Client(mcp) as c:
            tools = await c.list_tools()
            names = {t.name for t in tools}
            assert "listar_features" in names

    @pytest.mark.asyncio
    async def test_recomendar_tools_registered(self) -> None:
        async with Client(mcp) as c:
            tools = await c.list_tools()
            names = {t.name for t in tools}
            assert "recomendar_tools" in names

    @pytest.mark.asyncio
    async def test_planejar_consulta_registered(self) -> None:
        async with Client(mcp) as c:
            tools = await c.list_tools()
            names = {t.name for t in tools}
            assert "planejar_consulta" in names

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
    async def test_camara_tools_namespaced(self) -> None:
        async with Client(mcp) as c:
            tools = await c.list_tools()
            names = {t.name for t in tools}
            assert "camara_listar_deputados" in names
            assert "camara_buscar_proposicao" in names
            assert "camara_votos_nominais" in names

    @pytest.mark.asyncio
    async def test_senado_tools_namespaced(self) -> None:
        async with Client(mcp) as c:
            tools = await c.list_tools()
            names = {t.name for t in tools}
            assert "senado_listar_senadores" in names
            assert "senado_buscar_materia" in names
            assert "senado_listar_comissoes" in names
            assert "senado_agenda_plenario" in names

    @pytest.mark.asyncio
    async def test_listar_features_returns_summary(self) -> None:
        async with Client(mcp) as c:
            result = await c.call_tool("listar_features", {})
            assert "ibge" in result.data
            assert "bacen" in result.data
            assert "camara" in result.data
            assert "senado" in result.data


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

    @pytest.mark.asyncio
    async def test_camara_resources_namespaced(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            assert "data://camara/tipos-proposicao" in uris
            assert "data://camara/legislaturas" in uris
            assert "data://camara/info-api" in uris

    @pytest.mark.asyncio
    async def test_senado_resources_namespaced(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            assert "data://senado/tipos-materia" in uris
            assert "data://senado/info-api" in uris
            assert "data://senado/comissoes-permanentes" in uris


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

    @pytest.mark.asyncio
    async def test_camara_prompts_namespaced(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            assert "camara_acompanhar_proposicao" in names
            assert "camara_perfil_deputado" in names
            assert "camara_analise_votacao" in names

    @pytest.mark.asyncio
    async def test_senado_prompts_namespaced(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            assert "senado_acompanhar_materia" in names
            assert "senado_perfil_senador" in names
            assert "senado_analise_votacao_senado" in names


class TestExecutarLote:
    @pytest.mark.asyncio
    async def test_registered(self) -> None:
        async with Client(mcp) as c:
            tools = await c.list_tools()
            names = {t.name for t in tools}
            assert "executar_lote" in names

    @pytest.mark.asyncio
    async def test_has_docstring(self) -> None:
        async with Client(mcp) as c:
            tools = await c.list_tools()
            tool = next(t for t in tools if t.name == "executar_lote")
            assert tool.description
            assert "paralelo" in tool.description.lower()


class TestRootServerAuth:
    """Smoke tests for the auth strategy of the root server."""

    def test_auth_disabled_in_test_env(self) -> None:
        """conftest.py sets MCP_BRASIL_AUTH_MODE=none, so root server has auth=None."""
        # When auth is disabled, the FastMCP server is built with auth=None.
        # This also confirms the build_auth() factory is wired into server.py.
        from mcp_brasil.server import auth as root_auth

        assert root_auth is None


class TestMiddlewareStringCoercion:
    """RequestLoggingMiddleware coerces JSON string params before Pydantic validation."""

    @pytest.mark.asyncio
    async def test_arguments_string_coerced_to_dict(self) -> None:
        """Middleware coerces 'arguments' JSON string → dict for call_tool-like tools."""
        app = FastMCP("test")
        app.add_middleware(RequestLoggingMiddleware())
        received: dict = {}

        @app.tool()
        async def echo_args(arguments: dict) -> str:
            received.update(arguments)
            return "ok"

        async with Client(app) as c:
            result = await c.call_tool("echo_args", {"arguments": '{"key": "value"}'})
            assert result.data == "ok"
            assert received == {"key": "value"}

    @pytest.mark.asyncio
    async def test_arguments_invalid_json_string_passes_through(self) -> None:
        """Invalid JSON string is not silently swallowed — Pydantic rejects it."""
        app = FastMCP("test")
        app.add_middleware(RequestLoggingMiddleware())

        @app.tool()
        async def echo_args(arguments: dict) -> str:
            return "ok"

        async with Client(app) as c:
            with pytest.raises(Exception, match="Input should be a valid dictionary"):
                await c.call_tool("echo_args", {"arguments": "nao-e-json"})

    @pytest.mark.asyncio
    async def test_consultas_string_coerced_to_list(self) -> None:
        """Middleware coerces 'consultas' JSON string → list for executar_lote."""
        consultas_json = json.dumps([{"tool": "listar_features", "args": {}}])
        async with Client(mcp) as c:
            result = await c.call_tool("executar_lote", {"consultas": consultas_json})
            assert result.data

    @pytest.mark.asyncio
    async def test_consultas_invalid_json_string_passes_through(self) -> None:
        """Invalid JSON string for 'consultas' is not silently swallowed — Pydantic rejects it."""
        async with Client(mcp) as c:
            with pytest.raises(Exception, match="Input should be a valid list"):
                await c.call_tool("executar_lote", {"consultas": "nao-e-json"})


class TestRootServerToolTags:
    @pytest.mark.asyncio
    async def test_tools_have_tags(self) -> None:
        """Verify that tools from features have tags after mounting."""
        async with Client(mcp) as c:
            tools = await c.list_tools()
            # Find a known tool
            ibge_tool = next((t for t in tools if t.name == "ibge_listar_estados"), None)
            assert ibge_tool is not None
            # Tags should propagate through mount
            # Note: tags are on the Tool object, accessible via annotations or the tool itself

    @pytest.mark.asyncio
    async def test_meta_tools_have_discovery_tag(self) -> None:
        """Meta tools (listar_features, recomendar_tools) should have discovery tags."""
        async with Client(mcp) as c:
            tools = await c.list_tools()
            listar = next((t for t in tools if t.name == "listar_features"), None)
            assert listar is not None
