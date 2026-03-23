"""TransfereGov feature server — registers tools, resources, and prompts.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .prompts import analise_emendas_pix
from .resources import info_api
from .tools import (
    buscar_emenda_por_autor,
    buscar_emendas_pix,
    detalhe_emenda,
    emendas_por_municipio,
    resumo_emendas_ano,
)

mcp = FastMCP("mcp-brasil-transferegov")

# Tools
mcp.tool(buscar_emendas_pix)
mcp.tool(buscar_emenda_por_autor)
mcp.tool(detalhe_emenda)
mcp.tool(emendas_por_municipio)
mcp.tool(resumo_emendas_ano)

# Resources
mcp.resource("data://info-api", mime_type="application/json")(info_api)

# Prompts
mcp.prompt(analise_emendas_pix)
