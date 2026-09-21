"""Megasoft feature server — registers tools, resources, and prompts.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .prompts import analisar_municipio_megasoft
from .resources import endpoints_disponiveis
from .tools import (
    buscar_empenhos,
    buscar_ordens_pagamento,
    buscar_receitas,
    totais_ordens_pagamento,
)

mcp = FastMCP("mcp-brasil-megasoft")

# Tools
mcp.tool(buscar_empenhos)
mcp.tool(buscar_ordens_pagamento)
mcp.tool(totais_ordens_pagamento)
mcp.tool(buscar_receitas)

# Resources
mcp.resource("data://endpoints-disponiveis", mime_type="application/json")(endpoints_disponiveis)

# Prompts
mcp.prompt(analisar_municipio_megasoft)
