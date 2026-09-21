"""NucleoGov feature server — registers the tools.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .tools import (
    buscar_contratos,
    buscar_despesas,
    buscar_dispensas,
    buscar_legislacao,
    buscar_licitacoes,
)

mcp = FastMCP("mcp-brasil-nucleogov")

# Tools
mcp.tool(buscar_despesas)
mcp.tool(buscar_contratos)
mcp.tool(buscar_licitacoes)
mcp.tool(buscar_dispensas)
mcp.tool(buscar_legislacao)
