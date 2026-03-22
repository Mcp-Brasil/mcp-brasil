"""Câmara feature server — registers tools, resources, and prompts.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .prompts import acompanhar_proposicao, analise_votacao, perfil_deputado
from .resources import info_api, legislaturas_recentes, tipos_proposicao
from .tools import (
    agenda_legislativa,
    buscar_comissoes,
    buscar_deputado,
    buscar_proposicao,
    buscar_votacao,
    consultar_tramitacao,
    despesas_deputado,
    frentes_parlamentares,
    listar_deputados,
    votos_nominais,
)

mcp = FastMCP("mcp-brasil-camara")

# Tools (10)
mcp.tool(listar_deputados)
mcp.tool(buscar_deputado)
mcp.tool(buscar_proposicao)
mcp.tool(consultar_tramitacao)
mcp.tool(buscar_votacao)
mcp.tool(votos_nominais)
mcp.tool(despesas_deputado)
mcp.tool(agenda_legislativa)
mcp.tool(buscar_comissoes)
mcp.tool(frentes_parlamentares)

# Resources (URIs without namespace prefix — mount adds "camara/" automatically)
mcp.resource("data://tipos-proposicao", mime_type="application/json")(tipos_proposicao)
mcp.resource("data://legislaturas", mime_type="application/json")(legislaturas_recentes)
mcp.resource("data://info-api", mime_type="application/json")(info_api)

# Prompts
mcp.prompt(acompanhar_proposicao)
mcp.prompt(perfil_deputado)
mcp.prompt(analise_votacao)
