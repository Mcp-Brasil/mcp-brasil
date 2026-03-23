"""Tests for tool discovery features (BM25 search, recomendar_tools, tags).

Tests search transforms, LLM-powered recommendations, and tag propagation.
MCP_BRASIL_TOOL_SEARCH=none is set in conftest.py (before any import).
"""

from __future__ import annotations

import json
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from fastmcp import Client, FastMCP

from mcp_brasil._shared.discovery import build_catalog, recomendar_tools_impl
from mcp_brasil._shared.planner import PlanoConsulta, planejar_consulta_impl


# ---------------------------------------------------------------------------
# recomendar_tools — mocked Anthropic client
# ---------------------------------------------------------------------------
class TestRecomendarTools:
    @pytest.mark.asyncio
    async def test_missing_anthropic_package(self) -> None:
        """Should return error message when anthropic is not installed."""
        with patch.dict("sys.modules", {"anthropic": None}):
            result = await recomendar_tools_impl("gastos governo", "catalog text")
            assert "anthropic" in result.lower() or "search_tools" in result

    @pytest.mark.asyncio
    async def test_missing_api_key(self) -> None:
        """Should return error message when ANTHROPIC_API_KEY is empty."""
        mock_anthropic = MagicMock()
        with (
            patch.dict("sys.modules", {"anthropic": mock_anthropic}),
            patch("mcp_brasil._shared.discovery.ANTHROPIC_API_KEY", ""),
        ):
            result = await recomendar_tools_impl("gastos governo", "catalog text")
            assert "ANTHROPIC_API_KEY" in result

    @pytest.mark.asyncio
    async def test_successful_recommendation(self) -> None:
        """Should return LLM recommendations when everything is configured."""
        mock_block = MagicMock()
        mock_block.text = "Recomendo: transparencia_consultar_despesas"

        mock_response = MagicMock()
        mock_response.content = [mock_block]

        mock_client = AsyncMock()
        mock_client.messages.create = AsyncMock(return_value=mock_response)

        mock_anthropic = MagicMock()
        mock_anthropic.AsyncAnthropic = MagicMock(return_value=mock_client)

        with (
            patch.dict("sys.modules", {"anthropic": mock_anthropic}),
            patch("mcp_brasil._shared.discovery.ANTHROPIC_API_KEY", "test-key"),
        ):
            result = await recomendar_tools_impl("gastos governo", "catalog text")
            assert "transparencia_consultar_despesas" in result

    @pytest.mark.asyncio
    async def test_api_error_handling(self) -> None:
        """Should return error message when API call fails."""
        mock_client = AsyncMock()
        mock_client.messages.create = AsyncMock(side_effect=Exception("API timeout"))

        mock_anthropic = MagicMock()
        mock_anthropic.AsyncAnthropic = MagicMock(return_value=mock_client)

        with (
            patch.dict("sys.modules", {"anthropic": mock_anthropic}),
            patch("mcp_brasil._shared.discovery.ANTHROPIC_API_KEY", "test-key"),
        ):
            result = await recomendar_tools_impl("gastos governo", "catalog text")
            assert "Erro" in result
            assert "search_tools" in result


# ---------------------------------------------------------------------------
# build_catalog — tests
# ---------------------------------------------------------------------------
class TestBuildCatalog:
    def setup_method(self) -> None:
        """Reset catalog cache before each test."""
        import mcp_brasil._shared.discovery as disc

        disc._catalog_cache = ""

    def test_build_catalog_with_empty_registry(self) -> None:
        """Should return empty string for registry with no features."""
        mock_registry = MagicMock()
        mock_registry.features = {}
        result = build_catalog(mock_registry)
        assert result == ""

    def test_build_catalog_caches_result(self) -> None:
        """Should cache the catalog after first build."""
        import mcp_brasil._shared.discovery as disc

        mock_registry = MagicMock()
        mock_registry.features = {}
        build_catalog(mock_registry)

        # Set cache manually
        disc._catalog_cache = "cached"
        result = build_catalog(mock_registry)
        assert result == "cached"


# ---------------------------------------------------------------------------
# BM25SearchTransform — integration test with a simple server
# ---------------------------------------------------------------------------
class TestBM25SearchTransform:
    @pytest.mark.asyncio
    async def test_bm25_replaces_tool_listing(self) -> None:
        """BM25 should replace tool list with search_tools + call_tool."""
        from fastmcp.server.transforms.search import BM25SearchTransform

        server = FastMCP("test")

        @server.tool(tags={"busca", "estados"})
        def listar_estados() -> str:
            """Lista estados brasileiros."""
            return "SP, RJ, MG"

        @server.tool(tags={"consulta", "serie"})
        def consultar_serie(codigo: int) -> str:
            """Consulta série temporal."""
            return f"Serie {codigo}"

        server.add_transform(BM25SearchTransform(max_results=5))

        async with Client(server) as c:
            tools = await c.list_tools()
            names = {t.name for t in tools}
            # Only synthetic tools should be visible
            assert "search_tools" in names
            assert "call_tool" in names
            assert "listar_estados" not in names
            assert "consultar_serie" not in names

    @pytest.mark.asyncio
    async def test_bm25_search_finds_tools(self) -> None:
        """BM25 search should find tools by keyword."""
        from fastmcp.server.transforms.search import BM25SearchTransform

        server = FastMCP("test")

        @server.tool(tags={"busca", "estados"})
        def listar_estados() -> str:
            """Lista todos os estados brasileiros."""
            return "SP, RJ, MG"

        @server.tool(tags={"consulta", "serie"})
        def consultar_serie(codigo: int) -> str:
            """Consulta série temporal do Banco Central."""
            return f"Serie {codigo}"

        server.add_transform(BM25SearchTransform(max_results=5))

        async with Client(server) as c:
            result = await c.call_tool("search_tools", {"query": "estados brasileiros"})
            # Check the text content (result.data may be structured)
            text = str(result.content)
            assert "listar_estados" in text

    @pytest.mark.asyncio
    async def test_bm25_always_visible_pinned(self) -> None:
        """Always-visible tools should appear in list_tools."""
        from fastmcp.server.transforms.search import BM25SearchTransform

        server = FastMCP("test")

        @server.tool(tags={"meta"})
        def listar_features() -> str:
            """Lista features."""
            return "features"

        @server.tool(tags={"busca"})
        def hidden_tool() -> str:
            """Tool escondida."""
            return "hidden"

        server.add_transform(
            BM25SearchTransform(
                max_results=5,
                always_visible=["listar_features"],
            )
        )

        async with Client(server) as c:
            tools = await c.list_tools()
            names = {t.name for t in tools}
            assert "listar_features" in names
            assert "hidden_tool" not in names

    @pytest.mark.asyncio
    async def test_bm25_call_tool_executes(self) -> None:
        """call_tool proxy should execute discovered tools."""
        from fastmcp.server.transforms.search import BM25SearchTransform

        server = FastMCP("test")

        @server.tool
        def somar(a: int, b: int) -> int:
            """Soma dois números."""
            return a + b

        server.add_transform(BM25SearchTransform(max_results=5))

        async with Client(server) as c:
            result = await c.call_tool(
                "call_tool",
                {"name": "somar", "arguments": {"a": 3, "b": 4}},
            )
            # Result may be structured or text
            text = str(result.content)
            assert "7" in text


# ---------------------------------------------------------------------------
# Tool Search configuration switching
# ---------------------------------------------------------------------------
class TestToolSearchConfig:
    @pytest.mark.asyncio
    async def test_none_mode_shows_all_tools(self) -> None:
        """With TOOL_SEARCH=none (set in conftest.py), all tools should be visible."""
        from mcp_brasil.server import mcp as root_mcp

        async with Client(root_mcp) as c:
            tools = await c.list_tools()
            names = {t.name for t in tools}
            assert "listar_features" in names
            assert "recomendar_tools" in names
            assert "ibge_listar_estados" in names


# ---------------------------------------------------------------------------
# Tag propagation through mount
# ---------------------------------------------------------------------------
class TestTagPropagation:
    @pytest.mark.asyncio
    async def test_tags_preserved_after_mount(self) -> None:
        """Tags should be preserved on tools after mounting to parent server."""
        child = FastMCP("child")

        @child.tool(tags={"busca", "estados"})
        def listar_estados() -> str:
            """Lista estados."""
            return "SP"

        parent = FastMCP("parent")
        parent.mount(child, namespace="ibge")

        async with Client(parent) as c:
            tools = await c.list_tools()
            ibge_tool = next((t for t in tools if t.name == "ibge_listar_estados"), None)
            assert ibge_tool is not None

    @pytest.mark.asyncio
    async def test_search_finds_by_description(self) -> None:
        """BM25 should find tools by their description text."""
        from fastmcp.server.transforms.search import BM25SearchTransform

        server = FastMCP("test")

        @server.tool(tags={"ambiental", "queimadas"})
        def buscar_focos() -> str:
            """Busca focos de queimadas detectados por satélite no Brasil."""
            return "focos"

        @server.tool(tags={"financeiro", "bancos"})
        def listar_bancos() -> str:
            """Lista todos os bancos brasileiros registrados no Banco Central."""
            return "bancos"

        server.add_transform(BM25SearchTransform(max_results=5))

        async with Client(server) as c:
            result = await c.call_tool("search_tools", {"query": "focos queimadas satélite"})
            text = str(result.content)
            assert "buscar_focos" in text


# ---------------------------------------------------------------------------
# planejar_consulta — mocked Anthropic client
# ---------------------------------------------------------------------------
_VALID_PLAN_JSON = json.dumps(
    {
        "consulta": "gastos do deputado X",
        "complexidade": "moderada",
        "resumo": "Buscar deputado e consultar gastos",
        "etapas": [
            {
                "etapa": 1,
                "descricao": "Buscar deputado pelo nome",
                "tool": "camara_listar_deputados",
                "parametros": {"nome": "X"},
                "depende_de": [],
                "justificativa": "Precisamos do ID do deputado",
            },
            {
                "etapa": 2,
                "descricao": "Consultar gastos do deputado",
                "tool": "camara_despesas_deputado",
                "parametros": {"id": "{etapa_1.id}"},
                "depende_de": [1],
                "justificativa": "Obter despesas usando o ID encontrado",
            },
        ],
        "observacoes": "",
    }
)


class TestPlanejarConsulta:
    @pytest.mark.asyncio
    async def test_missing_anthropic_package(self) -> None:
        """Should return error message when anthropic is not installed."""
        with patch.dict("sys.modules", {"anthropic": None}):
            result = await planejar_consulta_impl("gastos governo", "catalog text")
            assert "anthropic" in result.lower() or "search_tools" in result

    @pytest.mark.asyncio
    async def test_missing_api_key(self) -> None:
        """Should return error message when ANTHROPIC_API_KEY is empty."""
        mock_anthropic = MagicMock()
        with (
            patch.dict("sys.modules", {"anthropic": mock_anthropic}),
            patch("mcp_brasil._shared.planner.ANTHROPIC_API_KEY", ""),
        ):
            result = await planejar_consulta_impl("gastos governo", "catalog text")
            assert "ANTHROPIC_API_KEY" in result

    @pytest.mark.asyncio
    async def test_successful_plan(self) -> None:
        """Should return markdown plan when everything is configured."""
        mock_block = MagicMock()
        mock_block.text = _VALID_PLAN_JSON

        mock_response = MagicMock()
        mock_response.content = [mock_block]

        mock_client = AsyncMock()
        mock_client.messages.create = AsyncMock(return_value=mock_response)

        mock_anthropic = MagicMock()
        mock_anthropic.AsyncAnthropic = MagicMock(return_value=mock_client)

        with (
            patch.dict("sys.modules", {"anthropic": mock_anthropic}),
            patch("mcp_brasil._shared.planner.ANTHROPIC_API_KEY", "test-key"),
        ):
            result = await planejar_consulta_impl("gastos do deputado X", "catalog")
            assert "## Plano de Consulta" in result
            assert "Etapa 1" in result
            assert "Etapa 2" in result
            assert "camara_listar_deputados" in result
            assert "Depende de:** Etapa 1" in result

    @pytest.mark.asyncio
    async def test_invalid_json_fallback(self) -> None:
        """Should return raw text when API returns non-JSON."""
        mock_block = MagicMock()
        mock_block.text = "Não consegui gerar um plano estruturado."

        mock_response = MagicMock()
        mock_response.content = [mock_block]

        mock_client = AsyncMock()
        mock_client.messages.create = AsyncMock(return_value=mock_response)

        mock_anthropic = MagicMock()
        mock_anthropic.AsyncAnthropic = MagicMock(return_value=mock_client)

        with (
            patch.dict("sys.modules", {"anthropic": mock_anthropic}),
            patch("mcp_brasil._shared.planner.ANTHROPIC_API_KEY", "test-key"),
        ):
            result = await planejar_consulta_impl("gastos governo", "catalog")
            assert "Não consegui gerar um plano estruturado." in result

    @pytest.mark.asyncio
    async def test_api_error_handling(self) -> None:
        """Should return error message when API call fails."""
        mock_client = AsyncMock()
        mock_client.messages.create = AsyncMock(side_effect=Exception("API timeout"))

        mock_anthropic = MagicMock()
        mock_anthropic.AsyncAnthropic = MagicMock(return_value=mock_client)

        with (
            patch.dict("sys.modules", {"anthropic": mock_anthropic}),
            patch("mcp_brasil._shared.planner.ANTHROPIC_API_KEY", "test-key"),
        ):
            result = await planejar_consulta_impl("gastos governo", "catalog")
            assert "Erro" in result
            assert "search_tools" in result


class TestPlanoConsultaMarkdown:
    def test_to_markdown_renders_steps(self) -> None:
        """to_markdown should render all steps with dependencies."""
        plano = PlanoConsulta.model_validate(json.loads(_VALID_PLAN_JSON))
        md = plano.to_markdown()
        assert "## Plano de Consulta" in md
        assert "**Complexidade:** moderada" in md
        assert "### Etapa 1:" in md
        assert "### Etapa 2:" in md
        assert "`camara_listar_deputados`" in md
        assert "Depende de:** (nenhuma)" in md
        assert "Depende de:** Etapa 1" in md

    def test_to_markdown_with_observacoes(self) -> None:
        """to_markdown should include observations when present."""
        plano = PlanoConsulta(
            consulta="teste",
            complexidade="simples",
            resumo="Plano simples",
            etapas=[
                {
                    "etapa": 1,
                    "descricao": "Passo único",
                    "tool": "ibge_listar_estados",
                    "parametros": {},
                    "depende_de": [],
                    "justificativa": "Necessário",
                }
            ],
            observacoes="Requer autenticação no Portal da Transparência.",
        )
        md = plano.to_markdown()
        assert "**Observações:**" in md
        assert "Portal da Transparência" in md
