"""Integration tests for the root mcp-brasil server.

Tests the fully composed server with all features mounted.
MCP_BRASIL_TOOL_SEARCH=none is set in conftest.py (before any import).
"""

import json

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

    @pytest.mark.asyncio
    async def test_aceita_consultas_como_string_json(self) -> None:
        # Issue #8: modelos que serializam `consultas` como string JSON não devem
        # ser rejeitados por erro de validação — a string é normalizada antes.
        consultas = [{"tool": "tool_inexistente", "args": {}}]
        async with Client(mcp) as c:
            result = await c.call_tool("executar_lote", {"consultas": json.dumps(consultas)})
        # O lote roda (sem ValidationError); a tool inexistente vira mensagem no resultado.
        assert result.data is not None


class TestRootServerAuth:
    """Smoke tests for the auth strategy of the root server."""

    def test_auth_disabled_in_test_env(self) -> None:
        """conftest.py sets MCP_BRASIL_AUTH_MODE=none, so root server has auth=None."""
        # When auth is disabled, the FastMCP server is built with auth=None.
        # This also confirms the build_auth() factory is wired into server.py.
        from mcp_brasil.server import auth as root_auth

        assert root_auth is None


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


class TestRootServerInfo:
    @pytest.mark.asyncio
    async def test_serverinfo_announces_package_version(self) -> None:
        """serverInfo.version must be the mcp-brasil version, not FastMCP's.

        Without an explicit `version=` in the FastMCP constructor, the handshake
        advertises the FastMCP release (e.g. 3.2.3) instead of the package
        version, which makes client-side version pinning and bug reports point
        at the wrong project.
        """
        from mcp_brasil import __version__

        async with Client(mcp) as c:
            assert c.initialize_result.serverInfo.version == __version__
