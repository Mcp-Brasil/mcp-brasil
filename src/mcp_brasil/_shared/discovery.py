"""LLM-powered tool recommendation for mcp-brasil.

Uses the Anthropic API (claude-haiku-4-5) to understand user intent
and recommend the most relevant tools from the mcp-brasil catalog.
"""

from __future__ import annotations

import logging

from ..settings import ANTHROPIC_API_KEY

logger = logging.getLogger("mcp-brasil.discovery")

# Catalog is built once and cached at module level
_catalog_cache: str = ""


def _format_tool_signature(feature_name: str, tool_name: str, tool: object) -> str:
    """Format a tool as a readable signature with params and description.

    Produces output like:
        - camara_listar_deputados(nome?: str, siglaUf?: str) — Lista deputados federais.
    """
    params = getattr(tool, "parameters", {})
    properties: dict[str, dict[str, object]] = params.get("properties", {})
    required: list[str] = params.get("required", [])

    # Build param list: "nome: str" or "nome?: str" for optional
    param_parts: list[str] = []
    for pname, pschema in properties.items():
        if pname == "ctx":
            continue
        ptype = pschema.get("type", "any")
        opt = "" if pname in required else "?"
        param_parts.append(f"{pname}{opt}: {ptype}")

    signature = ", ".join(param_parts)
    full_name = f"{feature_name}_{tool_name}"

    # Use first line of description as summary
    desc = (getattr(tool, "description", "") or "").split("\n")[0]

    return f"- `{full_name}({signature})` — {desc}"


def build_catalog(registry: object) -> str:
    """Build a rich text catalog of all tools from the registry.

    Uses FeatureMeta (name, description, auth) and tool schemas (params,
    types, descriptions) to produce a detailed catalog for LLM consumption.

    Args:
        registry: FeatureRegistry instance with discovered features.

    Returns:
        Markdown-formatted catalog with feature context and tool signatures.
    """
    global _catalog_cache
    if _catalog_cache:
        return _catalog_cache

    lines: list[str] = []
    features = getattr(registry, "features", {})
    for feat in features.values():
        meta = feat.meta
        auth_info = (
            f"Requer autenticação ({meta.auth_env_var})"
            if meta.requires_auth
            else "Sem autenticação"
        )
        lines.append(f"\n## {meta.name}: {meta.description}")
        lines.append(f"Auth: {auth_info}")

        # Get tools from the feature's server
        server = feat.server
        if hasattr(server, "_tool_manager") and hasattr(server._tool_manager, "_tools"):
            for tool_name, tool in server._tool_manager._tools.items():
                lines.append(_format_tool_signature(meta.name, tool_name, tool))

    _catalog_cache = "\n".join(lines)
    return _catalog_cache


async def recomendar_tools_impl(query: str, catalog: str) -> str:
    """Call Anthropic API to recommend tools based on user query.

    Args:
        query: Natural language question from the user.
        catalog: Pre-built catalog string of all tools.

    Returns:
        LLM-generated recommendations with explanations.
    """
    try:
        import anthropic
    except ImportError:
        return (
            "Erro: O pacote 'anthropic' não está instalado. "
            "Instale com: pip install 'mcp-brasil[llm]'\n\n"
            "Alternativa: use a tool 'search_tools' para buscar por palavras-chave."
        )

    api_key = ANTHROPIC_API_KEY
    if not api_key:
        return (
            "Erro: ANTHROPIC_API_KEY não configurada. "
            "Defina a variável de ambiente ANTHROPIC_API_KEY para usar esta tool.\n\n"
            "Alternativa: use a tool 'search_tools' para buscar por palavras-chave."
        )

    client = anthropic.AsyncAnthropic(api_key=api_key)

    system_prompt = (
        "Você é um assistente que recomenda tools do mcp-brasil. "
        "Dado o catálogo de tools disponíveis e a pergunta do usuário, "
        "recomende as 3-5 tools mais relevantes. Para cada tool:\n"
        "1. Nome completo da tool (com prefixo da feature)\n"
        "2. Por que é relevante para a pergunta\n"
        "3. Exemplo de como usar (parâmetros principais)\n\n"
        "Responda em português. Seja conciso e prático.\n\n"
        f"## Catálogo de Tools\n{catalog}"
    )

    try:
        response = await client.messages.create(
            model="claude-haiku-4-5-20251001",
            max_tokens=1024,
            system=system_prompt,
            messages=[{"role": "user", "content": query}],
        )
        block = response.content[0]
        return str(getattr(block, "text", ""))
    except Exception as e:
        logger.error("Erro ao chamar Anthropic API: %s", e)
        return f"Erro ao consultar IA: {e}\n\nUse 'search_tools' como alternativa."
