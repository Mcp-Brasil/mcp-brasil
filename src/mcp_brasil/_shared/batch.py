"""Batch execution for running multiple tools in a single call.

Builds a dispatch table mapping namespaced tool names to their underlying
Python functions, then executes them concurrently with asyncio.gather().
"""

from __future__ import annotations

import asyncio
import importlib
import inspect
import json
import logging
import pkgutil
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from collections.abc import Callable, Coroutine

    ToolFn = Callable[..., Coroutine[Any, Any, str]]

    from mcp_brasil._shared.feature import FeatureRegistry

logger = logging.getLogger(__name__)

_dispatch: dict[str, Any] = {}


def build_dispatch(registry: FeatureRegistry) -> dict[str, Any]:
    """Build a mapping of namespaced tool names → async functions.

    Scans tools.py modules from all registered features, including
    nested sub-packages (e.g., compras/pncp, compras/dadosabertos).
    """
    global _dispatch
    if _dispatch:
        return _dispatch

    for name, feat in registry.features.items():
        base = feat.module_path
        _scan_tools_module(base, name)

        # Sub-packages (e.g., compras → compras/pncp, compras/dadosabertos)
        try:
            pkg = importlib.import_module(base)
            if hasattr(pkg, "__path__"):
                for _, sub_path, is_pkg in pkgutil.iter_modules(pkg.__path__, base + "."):
                    sub_name = sub_path.rsplit(".", 1)[-1]
                    if is_pkg and not sub_name.startswith("_"):
                        _scan_tools_module(sub_path, f"{name}_{sub_name}")
        except Exception:
            pass

    logger.info("Batch dispatch: %d tools registered", len(_dispatch))
    return _dispatch


def _scan_tools_module(module_path: str, namespace: str) -> None:
    """Import a tools module and register its async functions."""
    try:
        mod = importlib.import_module(f"{module_path}.tools")
    except ImportError:
        return

    for fn_name, fn in inspect.getmembers(mod, inspect.iscoroutinefunction):
        if not fn_name.startswith("_"):
            key = f"{namespace}_{fn_name}"
            _dispatch[key] = fn


async def execute_batch(
    queries: list[dict[str, Any]],
    ctx: Any,
) -> str:
    """Execute multiple tool calls concurrently.

    Args:
        queries: List of {"tool": "name", "args": {}} dicts.
        ctx: FastMCP Context to pass to tools that accept it.

    Returns:
        Formatted markdown with all results.
    """
    if not queries:
        return "Nenhuma consulta fornecida."

    if len(queries) > 10:
        return "Máximo de 10 consultas por lote. Reduza a lista."

    async def _run_one(q: dict[str, Any]) -> tuple[str, str]:
        tool_name = q.get("tool", "")
        args = q.get("args", {})

        if isinstance(args, str):
            try:
                args = json.loads(args)
            except json.JSONDecodeError as exc:
                return tool_name, f"'args' não é JSON válido: {exc}"
        if not isinstance(args, dict):
            return tool_name, f"'args' deve ser um objeto JSON, recebido: {type(args).__name__}"

        fn = _dispatch.get(tool_name)

        if fn is None:
            return tool_name, f"Tool '{tool_name}' não encontrada."

        try:
            sig = inspect.signature(fn)
            if "ctx" in sig.parameters:
                result = await fn(ctx=ctx, **args)
            else:
                result = await fn(**args)
            return tool_name, result
        except Exception as exc:
            return tool_name, f"Erro ao executar '{tool_name}': {exc}"

    results = await asyncio.gather(*[_run_one(q) for q in queries])

    parts: list[str] = []
    for tool_name, output in results:
        parts.append(f"## {tool_name}\n\n{output}")

    return "\n\n---\n\n".join(parts)
