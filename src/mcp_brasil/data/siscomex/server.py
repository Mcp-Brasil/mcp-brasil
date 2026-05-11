"""Siscomex feature server — registers tools.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .tools import buscar_ncm, consultar_ncm

mcp = FastMCP("mcp-brasil-siscomex")

# Tools (2)
mcp.tool(buscar_ncm, tags={"busca", "ncm", "comercio-exterior"})
mcp.tool(consultar_ncm, tags={"detalhe", "ncm", "comercio-exterior"})
