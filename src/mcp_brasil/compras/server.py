"""Compras feature server — registers tools, resources, and prompts.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .prompts import investigar_fornecedor
from .resources import modalidades_licitacao
from .tools import buscar_atas, buscar_contratacoes, buscar_contratos

mcp = FastMCP("mcp-brasil-compras")

# Tools
mcp.tool(buscar_contratacoes)
mcp.tool(buscar_contratos)
mcp.tool(buscar_atas)

# Resources
mcp.resource("data://modalidades", mime_type="application/json")(modalidades_licitacao)

# Prompts
mcp.prompt(investigar_fornecedor)
