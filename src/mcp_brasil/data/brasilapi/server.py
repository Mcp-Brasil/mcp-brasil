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
mcp.tool(consultar_cep, tags={"consulta", "cep", "endereco"})
mcp.tool(consultar_cnpj, tags={"consulta", "cnpj", "empresa"})
mcp.tool(consultar_ddd, tags={"consulta", "ddd", "telefone"})
mcp.tool(listar_bancos, tags={"listagem", "bancos", "financeiro"})
mcp.tool(consultar_banco, tags={"detalhe", "bancos", "financeiro"})
mcp.tool(listar_moedas, tags={"listagem", "moedas", "cambio"})
mcp.tool(consultar_cotacao, tags={"consulta", "cotacao", "cambio"})
mcp.tool(consultar_feriados, tags={"consulta", "feriados", "calendario"})
mcp.tool(consultar_taxa, tags={"consulta", "taxas", "indicadores"})
mcp.tool(listar_tabelas_fipe, tags={"listagem", "fipe", "veiculos"})
mcp.tool(listar_marcas_fipe, tags={"listagem", "fipe", "veiculos"})
mcp.tool(buscar_veiculos_fipe, tags={"busca", "fipe", "veiculos"})
mcp.tool(consultar_isbn, tags={"consulta", "isbn", "livros"})
mcp.tool(buscar_ncm, tags={"busca", "ncm", "comercio-exterior"})
mcp.tool(consultar_pix_participantes, tags={"consulta", "pix", "financeiro"})
mcp.tool(consultar_registro_br, tags={"consulta", "dominio", "registro-br"})

# Resources
mcp.resource("data://taxas", mime_type="application/json")(taxas_disponiveis)
mcp.resource("data://tipos-veiculo-fipe", mime_type="application/json")(tipos_veiculo_fipe)
mcp.resource("data://endpoints", mime_type="application/json")(endpoints_brasilapi)

# Prompts
mcp.prompt(analise_empresa)
mcp.prompt(panorama_economico)
