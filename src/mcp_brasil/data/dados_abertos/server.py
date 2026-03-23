"""Dados Abertos feature server — registers tools, resources, and prompts.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .prompts import explorar_dados
from .resources import formatos_disponiveis
from .tools import buscar_conjuntos, buscar_recursos, detalhar_conjunto, listar_organizacoes

mcp = FastMCP("mcp-brasil-dados-abertos")

# Tools
mcp.tool(buscar_conjuntos, tags={"busca", "datasets", "dados-abertos"})
mcp.tool(detalhar_conjunto, tags={"detalhe", "datasets", "dados-abertos"})
mcp.tool(listar_organizacoes, tags={"listagem", "organizacoes", "dados-abertos"})
mcp.tool(buscar_recursos, tags={"busca", "recursos", "downloads"})

# Resources
mcp.resource("data://formatos", mime_type="application/json")(formatos_disponiveis)

# Prompts
mcp.prompt(explorar_dados)
