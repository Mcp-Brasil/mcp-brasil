"""BNDES feature server — registers tools.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .tools import (
    buscar_datasets_bndes,
    consultar_operacoes_bndes,
    detalhar_dataset_bndes,
    listar_instituicoes_bndes,
)

mcp = FastMCP("mcp-brasil-bndes")

# Tools
mcp.tool(buscar_datasets_bndes, tags={"bndes", "datasets", "busca"})
mcp.tool(detalhar_dataset_bndes, tags={"bndes", "datasets", "detalhes"})
mcp.tool(consultar_operacoes_bndes, tags={"bndes", "financiamento", "operacoes"})
mcp.tool(listar_instituicoes_bndes, tags={"bndes", "instituicoes", "credenciadas"})
