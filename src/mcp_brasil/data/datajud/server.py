"""DataJud feature server — registers tools, resources, and prompts.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .prompts import analise_processo, pesquisa_juridica
from .resources import classes_processuais, info_api, tribunais_disponiveis
from .tools import (
    buscar_processo_por_numero,
    buscar_processos,
    buscar_processos_avancado,
    buscar_processos_por_assunto,
    buscar_processos_por_classe,
    buscar_processos_por_orgao,
    consultar_movimentacoes,
)

mcp = FastMCP("mcp-brasil-datajud")

# Tools (7)
mcp.tool(buscar_processos, tags={"busca", "processos", "judicial"})
mcp.tool(buscar_processo_por_numero, tags={"busca", "processos", "npu"})
mcp.tool(buscar_processos_por_classe, tags={"busca", "processos", "classe-processual"})
mcp.tool(buscar_processos_por_assunto, tags={"busca", "processos", "assunto"})
mcp.tool(buscar_processos_por_orgao, tags={"busca", "processos", "orgao-julgador"})
mcp.tool(buscar_processos_avancado, tags={"busca", "processos", "judicial", "avancado"})
mcp.tool(consultar_movimentacoes, tags={"consulta", "movimentacoes", "processos"})

# Resources
mcp.resource("data://tribunais", mime_type="application/json")(tribunais_disponiveis)
mcp.resource("data://classes-processuais", mime_type="application/json")(classes_processuais)
mcp.resource("data://info-api", mime_type="application/json")(info_api)

# Prompts
mcp.prompt(analise_processo)
mcp.prompt(pesquisa_juridica)
