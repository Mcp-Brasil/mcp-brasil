"""Tolerância a argumentos enviados como string JSON nas meta-tools.

Alguns modelos (ex.: Qwen) serializam campos de objeto/array como string JSON em
vez de objeto/array. Sem tratamento, a validação Pydantic rejeita e agentes com
amostragem determinística (temperature=0) entram em loop de retry com o mesmo erro.

A normalização roda no middleware `on_call_tool`, que executa ANTES da validação
dos argumentos do tool (o que ele alterar em `context.message.arguments` segue para
a execução). Só uma allowlist exata de (tool, campo, tipo esperado) é tratada —
nunca coerção genérica, que mudaria a semântica de campos textuais legítimos cujo
conteúdo por acaso é JSON válido (ex.: "2024", "true", "[]").
"""

from __future__ import annotations

import json
from typing import Any

import mcp.types as mt
from fastmcp.server.middleware import CallNext, Middleware, MiddlewareContext
from fastmcp.tools import ToolResult

# Tool name -> {campo: tipo Python esperado após json.loads}.
# Apenas estes campos, nestas tools, são tolerantes a string JSON.
_JSON_STRING_FIELDS: dict[str, dict[str, type]] = {
    "executar_lote": {"consultas": list},
    "call_tool": {"arguments": dict},
}


def normalize_json_string_args(name: str, arguments: dict[str, Any] | None) -> None:
    """Converte, no lugar, campos string-JSON da allowlist para o objeto/array.

    Só altera quando o valor é `str`, faz `json.loads` sem erro, e o resultado bate
    com o tipo esperado. Caso contrário, deixa intacto — a validação normal do tool
    decide (e recusa entradas realmente inválidas).
    """
    if not arguments:
        return
    fields = _JSON_STRING_FIELDS.get(name)
    if fields is None:
        return
    for field, expected_type in fields.items():
        value = arguments.get(field)
        if not isinstance(value, str):
            continue
        try:
            parsed = json.loads(value)
        except (ValueError, TypeError):
            continue  # não é JSON válido -> deixa a validação normal recusar
        if isinstance(parsed, expected_type):
            arguments[field] = parsed


class TolerantArgumentsMiddleware(Middleware):
    """Aceita `consultas`/`arguments` como objeto/array OU string JSON nas meta-tools."""

    async def on_call_tool(
        self,
        context: MiddlewareContext[mt.CallToolRequestParams],
        call_next: CallNext[mt.CallToolRequestParams, ToolResult],
    ) -> ToolResult:
        normalize_json_string_args(context.message.name, context.message.arguments)
        return await call_next(context)
