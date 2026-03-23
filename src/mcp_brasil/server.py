"""mcp-brasil root server — auto-discovers and mounts all features.

This file uses FeatureRegistry for zero-touch feature onboarding.
You should NEVER need to edit this file to add a new feature.
Just create a new directory following the convention in ADR-001/002.

Usage:
    fastmcp run mcp_brasil.server:mcp
    fastmcp run mcp_brasil.server:mcp --transport http --port 8000
"""

import logging
import time

import mcp.types as mt
from fastmcp import Context, FastMCP
from fastmcp.prompts import PromptResult
from fastmcp.resources import ResourceResult
from fastmcp.server.middleware import CallNext, Middleware, MiddlewareContext
from fastmcp.tools import ToolResult

from ._shared.batch import build_dispatch, execute_batch
from ._shared.feature import FeatureRegistry
from ._shared.lifespan import http_lifespan
from .settings import TOOL_SEARCH

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(name)s] %(levelname)s: %(message)s",
)
logger = logging.getLogger("mcp-brasil")


# ---------------------------------------------------------------------------
# Middleware — lightweight request logging
# ---------------------------------------------------------------------------
class RequestLoggingMiddleware(Middleware):
    """Log all tool calls, resource reads, and prompt requests."""

    async def on_call_tool(
        self,
        context: MiddlewareContext[mt.CallToolRequestParams],
        call_next: CallNext[mt.CallToolRequestParams, ToolResult],
    ) -> ToolResult:
        name = context.message.name
        logger.info("Tool call: %s", name)
        start = time.monotonic()
        result = await call_next(context)
        elapsed = time.monotonic() - start
        logger.info("Tool %s completed in %.2fs", name, elapsed)
        return result

    async def on_read_resource(
        self,
        context: MiddlewareContext[mt.ReadResourceRequestParams],
        call_next: CallNext[mt.ReadResourceRequestParams, ResourceResult],
    ) -> ResourceResult:
        uri = context.message.uri
        logger.info("Resource read: %s", uri)
        return await call_next(context)

    async def on_get_prompt(
        self,
        context: MiddlewareContext[mt.GetPromptRequestParams],
        call_next: CallNext[mt.GetPromptRequestParams, PromptResult],
    ) -> PromptResult:
        name = context.message.name
        logger.info("Prompt get: %s", name)
        return await call_next(context)


# ---------------------------------------------------------------------------
# Server setup
# ---------------------------------------------------------------------------

# Create the root server
mcp = FastMCP("mcp-brasil 🇧🇷", lifespan=http_lifespan)

# Add middleware
mcp.add_middleware(RequestLoggingMiddleware())

# Auto-discover and mount all features
registry = FeatureRegistry()
registry.discover("mcp_brasil.data")
registry.discover("mcp_brasil.agentes")
registry.mount_all(mcp)

logger.info("\n%s", registry.summary())

# Build batch dispatch table for executar_lote
build_dispatch(registry)


# Expose a meta-tool for introspection
@mcp.tool(tags={"meta", "discovery"})
def listar_features() -> str:
    """Lista todas as features (APIs) disponíveis no mcp-brasil.

    Use esta tool para saber quais APIs governamentais estão conectadas
    e quais tools cada uma oferece.

    Returns:
        Resumo das features ativas com descrição e status de autenticação.
    """
    return registry.summary()


# Expose an LLM-powered recommendation tool
@mcp.tool(tags={"meta", "discovery"})
async def recomendar_tools(query: str, ctx: Context) -> str:
    """Recomenda tools relevantes a partir de uma pergunta em linguagem natural.

    Usa IA para entender sua intenção e sugerir as tools mais adequadas
    do mcp-brasil, explicando quando e como usar cada uma.

    Args:
        query: Pergunta ou descrição do que você precisa
               (ex: "quero dados sobre gastos do governo federal").
    """
    from ._shared.discovery import build_catalog, recomendar_tools_impl

    await ctx.info(f"Buscando recomendações para: {query}")
    catalog = build_catalog(registry)
    return await recomendar_tools_impl(query, catalog)


@mcp.tool(tags={"meta", "discovery", "planejamento"})
async def planejar_consulta(query: str, ctx: Context) -> str:
    """Cria um plano de execução para consultas complexas.

    Analisa a pergunta, identifica quais tools usar, em que ordem,
    e quais etapas dependem de outras. Útil para consultas que
    precisam de múltiplas chamadas combinadas.

    Args:
        query: Pergunta em linguagem natural
               (ex: "compare os gastos do deputado X com a média").
    """
    from ._shared.discovery import build_catalog
    from ._shared.planner import planejar_consulta_impl

    await ctx.info(f"Planejando consulta: {query}")
    catalog = build_catalog(registry)
    return await planejar_consulta_impl(query, catalog)


@mcp.tool(tags={"meta", "batch"})
async def executar_lote(consultas: list[dict[str, object]], ctx: Context) -> str:
    """Executa múltiplas tools em uma única chamada, em paralelo.

    Use esta tool para evitar chamadas sequenciais quando precisar de dados
    de várias fontes ou de vários anos/parâmetros ao mesmo tempo.

    Cada consulta deve ter o nome completo da tool (com namespace, ex:
    "camara_buscar_proposicao") e seus argumentos.

    Args:
        consultas: Lista de consultas. Cada item é um objeto com:
                   - "tool": nome completo da tool (ex: "camara_despesas_deputado")
                   - "args": objeto com os argumentos da tool
                   Exemplo: [
                     {"tool": "camara_despesas_deputado",
                      "args": {"deputado_id": 204554, "ano": 2024}},
                     {"tool": "camara_despesas_deputado",
                      "args": {"deputado_id": 204554, "ano": 2023}}
                   ]
    """
    await ctx.info(f"Executando lote de {len(consultas)} consulta(s)...")
    return await execute_batch(consultas, ctx)


# ---------------------------------------------------------------------------
# Tool Search Transform — configurable via MCP_BRASIL_TOOL_SEARCH
# ---------------------------------------------------------------------------
_always_visible = [
    "listar_features",
    "recomendar_tools",
    "planejar_consulta",
    "executar_lote",
]

if TOOL_SEARCH == "bm25":
    from fastmcp.server.transforms.search import BM25SearchTransform

    mcp.add_transform(
        BM25SearchTransform(
            max_results=10,
            always_visible=_always_visible,
        )
    )
    logger.info("Tool search: BM25 (search_tools + call_tool)")

elif TOOL_SEARCH == "code_mode":
    try:
        from fastmcp.experimental.transforms.code_mode import (
            CodeMode,
            GetSchemas,
            GetTags,
            Search,
        )

        mcp.add_transform(
            CodeMode(
                discovery_tools=[GetTags(name="get_tags"), Search(name="search"), GetSchemas()],
            )
        )
        logger.info("Tool search: CodeMode (experimental)")
    except ImportError:
        logger.warning(
            "CodeMode requires pydantic-monty. "
            "Install with: pip install 'fastmcp[code-mode]'. "
            "Falling back to BM25."
        )
        from fastmcp.server.transforms.search import BM25SearchTransform

        mcp.add_transform(
            BM25SearchTransform(
                max_results=10,
                always_visible=_always_visible,
            )
        )

else:
    logger.info("Tool search: none (all %d+ tools visible)", len(registry.features))


if __name__ == "__main__":
    mcp.run()
