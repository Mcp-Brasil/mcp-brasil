"""TCE-SP feature server — registers tools, resources, and prompts.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .prompts import analisar_financas_municipio_sp
from .resources import endpoints_tce_sp
from .tools import consultar_despesas_sp, consultar_receitas_sp, listar_municipios_sp

mcp = FastMCP("mcp-brasil-tce-sp")

# Tools
mcp.tool(listar_municipios_sp)
mcp.tool(consultar_despesas_sp)
mcp.tool(consultar_receitas_sp)

# Resources
mcp.resource("data://endpoints", mime_type="application/json")(endpoints_tce_sp)

# Prompts
mcp.prompt(analisar_financas_municipio_sp)
