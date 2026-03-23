"""Transparência feature server — registers tools, resources, and prompts.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .prompts import analise_despesas, auditoria_fornecedor, verificacao_compliance
from .resources import bases_sancoes, categorias_beneficios, endpoints_disponiveis, info_api
from .tools import (
    buscar_acordos_leniencia,
    buscar_cartoes_pagamento,
    buscar_contratos,
    buscar_convenios,
    buscar_emendas,
    buscar_licitacoes,
    buscar_notas_fiscais,
    buscar_pep,
    buscar_sancoes,
    buscar_servidores,
    consultar_beneficio_social,
    consultar_bolsa_familia,
    consultar_cnpj,
    consultar_cpf,
    consultar_despesas,
    consultar_viagens,
    detalhar_contrato,
    detalhar_servidor,
)

mcp = FastMCP("mcp-brasil-transparencia")

# Tools
mcp.tool(buscar_contratos)
mcp.tool(consultar_despesas)
mcp.tool(buscar_servidores)
mcp.tool(buscar_licitacoes)
mcp.tool(consultar_bolsa_familia)
mcp.tool(buscar_sancoes)
mcp.tool(buscar_emendas)
mcp.tool(consultar_viagens)
mcp.tool(buscar_convenios)
mcp.tool(buscar_cartoes_pagamento)
mcp.tool(buscar_pep)
mcp.tool(buscar_acordos_leniencia)
mcp.tool(buscar_notas_fiscais)
mcp.tool(consultar_beneficio_social)
mcp.tool(consultar_cpf)
mcp.tool(consultar_cnpj)
mcp.tool(detalhar_contrato)
mcp.tool(detalhar_servidor)

# Resources (URIs without namespace prefix — mount adds "transparencia/" automatically)
mcp.resource("data://endpoints", mime_type="application/json")(endpoints_disponiveis)
mcp.resource("data://bases-sancoes", mime_type="application/json")(bases_sancoes)
mcp.resource("data://info-api", mime_type="application/json")(info_api)
mcp.resource("data://categorias-beneficios", mime_type="application/json")(categorias_beneficios)

# Prompts
mcp.prompt(auditoria_fornecedor)
mcp.prompt(analise_despesas)
mcp.prompt(verificacao_compliance)
