"""Diário Oficial feature server — registers tools, resources, and prompts.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .prompts import investigar_empresa
from .resources import capitais_cobertas
from .tools import buscar_cidades, buscar_diarios, buscar_trechos, listar_territorios

mcp = FastMCP("mcp-brasil-diario-oficial")

# Tools
mcp.tool(buscar_diarios, tags={"busca", "diarios-oficiais", "publicacoes"})
mcp.tool(buscar_trechos, tags={"busca", "trechos", "diarios-oficiais"})
mcp.tool(buscar_cidades, tags={"busca", "municipios", "cobertura"})
mcp.tool(listar_territorios, tags={"listagem", "municipios", "cobertura"})

# Resources
mcp.resource("data://capitais-cobertas", mime_type="application/json")(capitais_cobertas)

# Prompts
mcp.prompt(investigar_empresa)
