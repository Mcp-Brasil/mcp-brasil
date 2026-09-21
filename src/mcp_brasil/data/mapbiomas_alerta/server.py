"""MapBiomas Alerta feature server — registers tools, resources, and prompts.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .prompts import analisar_alertas
from .resources import categorias_territorio, info_api
from .tools import (
    buscar_alertas,
    buscar_alertas_por_car,
    consultar_alerta,
    listar_territorios,
)

mcp = FastMCP("mcp-brasil-mapbiomas-alerta")

# Tools
mcp.tool(listar_territorios)
mcp.tool(buscar_alertas)
mcp.tool(consultar_alerta)
mcp.tool(buscar_alertas_por_car)

# Resources
mcp.resource("data://mapbiomas-alerta/info-api", mime_type="application/json")(info_api)
mcp.resource("data://mapbiomas-alerta/categorias-territorio", mime_type="application/json")(
    categorias_territorio
)

# Prompts
mcp.prompt(analisar_alertas)
