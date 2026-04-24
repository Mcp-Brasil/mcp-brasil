"""anac_vra feature server."""

from fastmcp import FastMCP

from .tools import (
    info_anac_vra,
    pontualidade_aeroporto,
    ranking_empresas,
    top_justificativas,
    top_rotas,
    voos_empresa,
)

mcp: FastMCP = FastMCP("mcp-brasil-anac_vra")

mcp.tool(info_anac_vra)
mcp.tool(voos_empresa)
mcp.tool(pontualidade_aeroporto)
mcp.tool(top_rotas)
mcp.tool(ranking_empresas)
mcp.tool(top_justificativas)
