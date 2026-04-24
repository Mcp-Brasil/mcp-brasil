"""SICONFI feature server — registers tools, resources, prompts."""

from fastmcp import FastMCP

from .prompts import analise_fiscal_municipio, comparar_entes
from .resources import catalogo_anexos, referencia_codigos
from .tools import (
    anexos_populares,
    consultar_dca,
    consultar_rgf,
    consultar_rreo,
    extrato_entregas,
    listar_anexos_relatorios,
    listar_entes,
)

mcp = FastMCP("mcp-brasil-siconfi")

# Tools
mcp.tool(listar_entes, tags={"listagem", "entes", "municipios", "estados"})
mcp.tool(consultar_rreo, tags={"consulta", "rreo", "orcamento", "fiscal"})
mcp.tool(consultar_rgf, tags={"consulta", "rgf", "lrf", "fiscal"})
mcp.tool(consultar_dca, tags={"consulta", "dca", "contas-anuais", "fiscal"})
mcp.tool(extrato_entregas, tags={"consulta", "entregas", "status"})
mcp.tool(listar_anexos_relatorios, tags={"catalogo", "anexos"})
mcp.tool(anexos_populares, tags={"catalogo", "anexos", "referencia"})

# Resources
mcp.resource("data://catalogo-anexos", mime_type="application/json")(catalogo_anexos)
mcp.resource("data://referencia-codigos", mime_type="application/json")(referencia_codigos)

# Prompts
mcp.prompt(analise_fiscal_municipio)
mcp.prompt(comparar_entes)
