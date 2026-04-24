"""BCB Olinda feature server."""

from fastmcp import FastMCP

from .prompts import panorama_focus
from .resources import catalogo_indicadores_focus
from .tools import (
    focus_anual,
    focus_mensal,
    focus_selic,
    listar_indicadores_focus,
    listar_moedas_ptax,
    ptax_dolar,
    ptax_dolar_periodo,
    ptax_moeda,
)

mcp = FastMCP("mcp-brasil-bcb_olinda")

mcp.tool(ptax_dolar, tags={"ptax", "cambio", "dolar"})
mcp.tool(ptax_dolar_periodo, tags={"ptax", "cambio", "serie"})
mcp.tool(ptax_moeda, tags={"ptax", "cambio", "moedas"})
mcp.tool(listar_moedas_ptax, tags={"listagem", "moedas"})
mcp.tool(focus_anual, tags={"focus", "expectativas", "anual"})
mcp.tool(focus_mensal, tags={"focus", "expectativas", "mensal"})
mcp.tool(focus_selic, tags={"focus", "selic", "copom"})
mcp.tool(listar_indicadores_focus, tags={"listagem", "focus", "catalogo"})

mcp.resource("data://indicadores-focus", mime_type="application/json")(catalogo_indicadores_focus)

mcp.prompt(panorama_focus)
