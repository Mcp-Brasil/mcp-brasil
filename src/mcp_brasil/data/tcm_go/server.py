"""TCM-GO feature server — registers tools, resources, and prompts.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .prompts import analisar_municipio_goias
from .resources import endpoints_disponiveis
from .tools import buscar_acordaos, buscar_processos, listar_valores_campo

mcp = FastMCP("mcp-brasil-tcm-go")

# Tools
mcp.tool(buscar_processos)
mcp.tool(listar_valores_campo)
mcp.tool(buscar_acordaos)

# Resources
mcp.resource("data://endpoints-disponiveis", mime_type="application/json")(endpoints_disponiveis)

# Prompts
mcp.prompt(analisar_municipio_goias)
