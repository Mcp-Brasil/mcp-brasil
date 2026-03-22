"""BrasilAPI feature server — registers tools, resources, and prompts.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .prompts import analise_empresa, panorama_economico
from .resources import endpoints_brasilapi, taxas_disponiveis, tipos_veiculo_fipe
from .tools import (
    buscar_ncm,
    buscar_veiculos_fipe,
    consultar_banco,
    consultar_cep,
    consultar_cnpj,
    consultar_cotacao,
    consultar_ddd,
    consultar_feriados,
    consultar_isbn,
    consultar_pix_participantes,
    consultar_registro_br,
    consultar_taxa,
    listar_bancos,
    listar_marcas_fipe,
    listar_moedas,
    listar_tabelas_fipe,
)

mcp = FastMCP("mcp-brasil-brasilapi")

# Tools (16)
mcp.tool(consultar_cep)
mcp.tool(consultar_cnpj)
mcp.tool(consultar_ddd)
mcp.tool(listar_bancos)
mcp.tool(consultar_banco)
mcp.tool(listar_moedas)
mcp.tool(consultar_cotacao)
mcp.tool(consultar_feriados)
mcp.tool(consultar_taxa)
mcp.tool(listar_tabelas_fipe)
mcp.tool(listar_marcas_fipe)
mcp.tool(buscar_veiculos_fipe)
mcp.tool(consultar_isbn)
mcp.tool(buscar_ncm)
mcp.tool(consultar_pix_participantes)
mcp.tool(consultar_registro_br)

# Resources
mcp.resource("data://taxas", mime_type="application/json")(taxas_disponiveis)
mcp.resource("data://tipos-veiculo-fipe", mime_type="application/json")(tipos_veiculo_fipe)
mcp.resource("data://endpoints", mime_type="application/json")(endpoints_brasilapi)

# Prompts
mcp.prompt(analise_empresa)
mcp.prompt(panorama_economico)
