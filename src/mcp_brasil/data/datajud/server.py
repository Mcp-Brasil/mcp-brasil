"""DataJud feature server — registers tools, resources, and prompts.

This file only registers components. Zero business logic (ADR-001 rule #4).
"""

from fastmcp import FastMCP

from .prompts import analisar_mpus, analise_processo, monitorar_mpu, pesquisa_juridica
from .resources import (
    classes_processuais,
    info_api,
    mpu_classes,
    mpu_complementos,
    mpu_movimentos,
    tribunais_disponiveis,
)
from .tools import (
    buscar_medidas_protetivas,
    buscar_mpu_concedidas,
    buscar_mpu_por_tipo,
    buscar_processo_por_numero,
    buscar_processos,
    buscar_processos_avancado,
    buscar_processos_por_assunto,
    buscar_processos_por_classe,
    buscar_processos_por_orgao,
    consultar_movimentacoes,
    estatisticas_mpu,
    timeline_mpu,
)

mcp = FastMCP("mcp-brasil-datajud")

# Tools — Processos (7)
mcp.tool(buscar_processos, tags={"busca", "processos", "judicial"})
mcp.tool(buscar_processo_por_numero, tags={"busca", "processos", "npu"})
mcp.tool(buscar_processos_por_classe, tags={"busca", "processos", "classe-processual"})
mcp.tool(buscar_processos_por_assunto, tags={"busca", "processos", "assunto"})
mcp.tool(buscar_processos_por_orgao, tags={"busca", "processos", "orgao-julgador"})
mcp.tool(buscar_processos_avancado, tags={"busca", "processos", "judicial", "avancado"})
mcp.tool(consultar_movimentacoes, tags={"consulta", "movimentacoes", "processos"})

# Tools — MPU (5)
mcp.tool(buscar_medidas_protetivas, tags={"mpu", "medida-protetiva", "maria-da-penha"})
mcp.tool(buscar_mpu_concedidas, tags={"mpu", "medida-protetiva", "concessao"})
mcp.tool(buscar_mpu_por_tipo, tags={"mpu", "medida-protetiva", "tipo-medida"})
mcp.tool(estatisticas_mpu, tags={"mpu", "medida-protetiva", "estatisticas"})
mcp.tool(timeline_mpu, tags={"mpu", "medida-protetiva", "timeline"})

# Resources — Gerais (3)
mcp.resource("data://tribunais", mime_type="application/json")(tribunais_disponiveis)
mcp.resource("data://classes-processuais", mime_type="application/json")(classes_processuais)
mcp.resource("data://info-api", mime_type="application/json")(info_api)

# Resources — MPU (3)
mcp.resource("data://mpu/classes", mime_type="application/json")(mpu_classes)
mcp.resource("data://mpu/movimentos", mime_type="application/json")(mpu_movimentos)
mcp.resource("data://mpu/complementos", mime_type="application/json")(mpu_complementos)

# Prompts — Gerais (2)
mcp.prompt(analise_processo)
mcp.prompt(pesquisa_juridica)

# Prompts — MPU (2)
mcp.prompt(analisar_mpus)
mcp.prompt(monitorar_mpu)
