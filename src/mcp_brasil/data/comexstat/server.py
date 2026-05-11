"""ComexStat feature server — registers tools.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .tools import balanca_comercial, consultar_exportacoes, consultar_importacoes, listar_paises

mcp = FastMCP("mcp-brasil-comexstat")

# Tools (4)
mcp.tool(consultar_exportacoes, tags={"exportacao", "comercio-exterior", "comexstat"})
mcp.tool(consultar_importacoes, tags={"importacao", "comercio-exterior", "comexstat"})
mcp.tool(balanca_comercial, tags={"balanca", "comercio-exterior", "comexstat"})
mcp.tool(listar_paises, tags={"paises", "comercio-exterior", "comexstat"})
