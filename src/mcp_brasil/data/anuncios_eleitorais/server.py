"""Anuncios Eleitorais feature server — registers tools, resources, and prompts.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .prompts import analise_candidato, comparar_candidatos, panorama_eleitoral
from .resources import campos_disponiveis, estados_brasileiros, parametros_busca
from .tools import (
    analisar_demografia_anuncios,
    buscar_anuncios_eleitorais,
    buscar_anuncios_frase_exata,
    buscar_anuncios_por_financiador,
    buscar_anuncios_por_pagina,
    buscar_anuncios_por_regiao,
)

mcp = FastMCP("mcp-brasil-anuncios_eleitorais")

# Tools
mcp.tool(buscar_anuncios_eleitorais)
mcp.tool(buscar_anuncios_por_pagina)
mcp.tool(buscar_anuncios_por_financiador)
mcp.tool(buscar_anuncios_por_regiao)
mcp.tool(analisar_demografia_anuncios)
mcp.tool(buscar_anuncios_frase_exata)

# Resources (URIs without namespace prefix — mount adds "anuncios_eleitorais/" automatically)
mcp.resource("data://estados-brasileiros", mime_type="application/json")(estados_brasileiros)
mcp.resource("data://parametros-busca", mime_type="application/json")(parametros_busca)
mcp.resource("data://campos-disponiveis", mime_type="application/json")(campos_disponiveis)

# Prompts
mcp.prompt(analise_candidato)
mcp.prompt(panorama_eleitoral)
mcp.prompt(comparar_candidatos)
