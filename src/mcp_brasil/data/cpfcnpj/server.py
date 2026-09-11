"""CPF.CNPJ feature server. Registers tools only.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .tools import (
    consultar_cnpj,
    inscricoes_estaduais,
    qsa_cnpj,
    saldo,
    situacao_cnpj,
    validar_documento,
)

mcp = FastMCP("mcp-brasil-cpfcnpj")

# Tools (6)
mcp.tool(validar_documento, tags={"validacao", "cpf", "cnpj"})
mcp.tool(consultar_cnpj, tags={"consulta", "cnpj", "empresa"})
mcp.tool(situacao_cnpj, tags={"consulta", "cnpj", "situacao-cadastral"})
mcp.tool(qsa_cnpj, tags={"consulta", "cnpj", "socios"})
mcp.tool(inscricoes_estaduais, tags={"consulta", "cnpj", "inscricao-estadual"})
mcp.tool(saldo, tags={"conta", "saldo", "creditos"})
