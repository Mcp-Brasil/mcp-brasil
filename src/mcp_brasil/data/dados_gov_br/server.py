"""dados_gov_br feature server — registers tools, resources, and prompts.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .prompts import descobrir_fonte, explorar_dados, listar_dados_orgao, panorama_portal
from .resources import documentacao, formatos_disponiveis, legislacao
from .tools import (
    buscar_conjuntos,
    buscar_tags,
    detalhar_conjunto,
    detalhar_organizacao,
    detalhar_reuso,
    listar_formatos,
    listar_observancia_legal,
    listar_ods,
    listar_organizacoes,
    listar_reusos,
    listar_temas,
)

mcp = FastMCP("mcp-brasil-dados-gov-br")

# Tools — 11 tools
mcp.tool(buscar_conjuntos, tags={"busca", "datasets", "dados-abertos"})
mcp.tool(detalhar_conjunto, tags={"detalhe", "datasets", "dados-abertos"})
mcp.tool(listar_organizacoes, tags={"listagem", "organizacoes", "dados-abertos"})
mcp.tool(detalhar_organizacao, tags={"detalhe", "organizacoes", "dados-abertos"})
mcp.tool(listar_temas, tags={"listagem", "temas", "dados-abertos"})
mcp.tool(buscar_tags, tags={"busca", "tags", "dados-abertos"})
mcp.tool(listar_formatos, tags={"listagem", "formatos", "dados-abertos"})
mcp.tool(listar_ods, tags={"listagem", "ods", "dados-abertos"})
mcp.tool(listar_observancia_legal, tags={"listagem", "legal", "dados-abertos"})
mcp.tool(listar_reusos, tags={"listagem", "reusos", "dados-abertos"})
mcp.tool(detalhar_reuso, tags={"detalhe", "reusos", "dados-abertos"})

# Resources — 3 resources
mcp.resource("data://formatos", mime_type="application/json")(formatos_disponiveis)
mcp.resource("data://documentacao", mime_type="text/markdown")(documentacao)
mcp.resource("data://legislacao", mime_type="text/markdown")(legislacao)

# Prompts — 4 prompts
mcp.prompt(explorar_dados)
mcp.prompt(listar_dados_orgao)
mcp.prompt(panorama_portal)
mcp.prompt(descobrir_fonte)
