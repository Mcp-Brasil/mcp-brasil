"""TCU feature server — registers tools, resources, and prompts.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .prompts import analise_acordaos, verificar_empresa
from .resources import catalogo_endpoints, colegiados, situacoes_acordao
from .tools import (
    calcular_debito,
    consultar_acordaos,
    consultar_cadirreg,
    consultar_certidoes,
    consultar_inabilitados,
    consultar_inidoneos,
    consultar_pedidos_congresso,
    consultar_termos_contratuais,
)

mcp = FastMCP("mcp-brasil-tcu")

# Tools
mcp.tool(consultar_acordaos, tags={"consulta", "acordaos", "decisoes", "jurisprudencia"})
mcp.tool(consultar_inabilitados, tags={"consulta", "inabilitados", "funcao-publica"})
mcp.tool(consultar_inidoneos, tags={"consulta", "inidoneos", "licitacoes"})
mcp.tool(consultar_certidoes, tags={"consulta", "certidoes", "cnpj", "compliance"})
mcp.tool(consultar_pedidos_congresso, tags={"consulta", "congresso", "pedidos"})
mcp.tool(calcular_debito, tags={"calculo", "debito", "selic", "correcao"})
mcp.tool(consultar_termos_contratuais, tags={"consulta", "contratos", "licitacoes"})
mcp.tool(consultar_cadirreg, tags={"consulta", "cadirreg", "contas-irregulares"})

# Resources
mcp.resource("data://catalogo-endpoints", mime_type="application/json")(catalogo_endpoints)
mcp.resource("data://colegiados", mime_type="application/json")(colegiados)
mcp.resource("data://situacoes-acordao", mime_type="application/json")(situacoes_acordao)

# Prompts
mcp.prompt(analise_acordaos)
mcp.prompt(verificar_empresa)
