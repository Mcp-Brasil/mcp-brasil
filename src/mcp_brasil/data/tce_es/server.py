"""TCE-ES feature server — registers tools, resources, and prompts.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .prompts import analisar_gestao_es
from .resources import datasets_tce_es
from .tools import (
    buscar_contratacoes_municipios_es,
    buscar_contratos_es,
    buscar_licitacoes_es,
    buscar_obras_es,
)

mcp = FastMCP("mcp-brasil-tce-es")

# Tools
mcp.tool(buscar_licitacoes_es)
mcp.tool(buscar_contratos_es)
mcp.tool(buscar_contratacoes_municipios_es)
mcp.tool(buscar_obras_es)

# Resources
mcp.resource("data://datasets", mime_type="application/json")(datasets_tce_es)

# Prompts
mcp.prompt(analisar_gestao_es)
