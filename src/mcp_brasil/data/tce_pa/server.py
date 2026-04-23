"""tce_pa feature server — registers tools, resources, and prompts.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .prompts import (
    analisar_diario_oficial_pa,
    analisar_jurisprudencia_pa,
    analisar_sessoes_plenarias_pa,
    explorar_conteudo_pa,
)
from .resources import endpoints_tce_pa
from .tools import (
    buscar_conteudo_pa,
    buscar_diario_oficial_pa,
    buscar_jurisprudencia_pa,
    buscar_sessoes_plenarias_pa,
)

mcp = FastMCP("mcp-brasil-tce-pa")

# Tools
mcp.tool(
    buscar_diario_oficial_pa,
    tags={"busca", "diário-oficial", "tce", "pará"},
)
mcp.tool(
    buscar_sessoes_plenarias_pa,
    tags={"sessões", "plenária", "pauta", "ata", "extrato", "vídeo", "tce", "pará"},
)
mcp.tool(
    buscar_jurisprudencia_pa,
    tags={"acórdão", "resolução", "portaria", "jurisprudência", "normas", "tce", "pará"},
)
mcp.tool(
    buscar_conteudo_pa,
    tags={"notícias", "acervo", "educação", "youtube", "comunicação", "tce", "pará"},
)

# Resources
mcp.resource("data://endpoints", mime_type="application/json")(endpoints_tce_pa)

# Prompts
mcp.prompt(analisar_jurisprudencia_pa)
mcp.prompt(analisar_diario_oficial_pa)
mcp.prompt(analisar_sessoes_plenarias_pa)
mcp.prompt(explorar_conteudo_pa)
