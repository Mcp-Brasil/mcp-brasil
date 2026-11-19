"""Transparência Municípios e Estados feature server — registers tools, resources, and prompts."""

from fastmcp import FastMCP

from .prompts import auditoria_transparencia_municipio, comparativo_transparencia_regional
from .resources import (
    capitais_brasileiras,
    estados_brasileiros,
    estatisticas_nacionais_resource,
    megasoft_municipios_resource,
)
from .tools import (
    buscar_municipios,
    buscar_por_cnpj,
    comparar_transparencia,
    conectar_estado,
    conectar_municipio,
    consultar_concursos_municipio,
    consultar_contratos_municipio,
    consultar_convenios_municipio,
    consultar_despesas_municipio,
    consultar_detalhe_servidor_municipio,
    consultar_diarias_municipio,
    consultar_entregas,
    consultar_fiscal,
    consultar_legislacao_municipio,
    consultar_licitacoes_municipio,
    consultar_obras_municipio,
    consultar_receitas_municipio,
    consultar_servidores_municipio,
    consultar_transparencia_completa,
    descobrir_portal_municipio,
    estatisticas_nacionais,
    listar_estados,
    listar_municipios_megasoft,
    pesquisar_transparencia,
)

mcp = FastMCP("mcp-brasil-transparencia-municipios")

# Tools
mcp.tool(
    buscar_municipios,
    tags={"busca", "municipios", "transparencia", "geografia", "localidades"},
)
mcp.tool(
    buscar_por_cnpj,
    tags={"cnpj", "municipios", "prefeituras", "transparencia", "busca"},
)
mcp.tool(
    listar_estados,
    tags={"listagem", "estados", "ufs", "transparencia", "geografia"},
)
mcp.tool(
    conectar_municipio,
    tags={"conexao", "municipios", "transparencia", "dossie", "prefeituras", "tce"},
)
mcp.tool(
    conectar_estado,
    tags={"conexao", "estados", "ufs", "transparencia", "dossie", "tce"},
)
mcp.tool(
    descobrir_portal_municipio,
    tags={"descoberta", "portal", "municipios", "transparencia", "url"},
)
mcp.tool(
    consultar_servidores_municipio,
    tags={"servidores", "folha", "remuneracao", "salarios", "municipios", "transparencia", "tce"},
)
mcp.tool(
    consultar_despesas_municipio,
    tags={"despesas", "empenhos", "gastos", "municipios", "transparencia", "tce"},
)
mcp.tool(
    consultar_receitas_municipio,
    tags={"receitas", "arrecadacao", "municipios", "transparencia", "tce"},
)
mcp.tool(
    consultar_licitacoes_municipio,
    tags={"licitacoes", "editais", "compras", "pncp", "municipios", "transparencia"},
)
mcp.tool(
    consultar_contratos_municipio,
    tags={"contratos", "fornecedores", "pncp", "municipios", "transparencia"},
)
mcp.tool(
    consultar_transparencia_completa,
    tags={
        "transparencia",
        "auditoria",
        "servidores",
        "despesas",
        "receitas",
        "municipios",
        "completa",
    },
)
mcp.tool(
    consultar_fiscal,
    tags={"siconfi", "fiscal", "balanco", "orcamento", "dca", "rreo", "rgf", "transparencia"},
)
mcp.tool(
    consultar_entregas,
    tags={"siconfi", "lrf", "entregas", "conformidade", "fiscal", "transparencia"},
)
mcp.tool(
    pesquisar_transparencia,
    tags={"pesquisa", "transparencia", "fiscal", "compras", "siconfi", "pncp"},
)
mcp.tool(
    comparar_transparencia,
    tags={"comparacao", "transparencia", "municipios", "estados", "auditoria"},
)
mcp.tool(
    estatisticas_nacionais,
    tags={"estatisticas", "municipios", "estados", "panorama"},
)
mcp.tool(
    listar_municipios_megasoft,
    tags={"megasoft", "municipios", "transparencia", "catalogo", "portais"},
)
mcp.tool(
    consultar_diarias_municipio,
    tags={"diarias", "viagens", "servidores", "municipios", "transparencia", "megasoft"},
)
mcp.tool(
    consultar_obras_municipio,
    tags={"obras", "infraestrutura", "projetos", "municipios", "transparencia", "megasoft"},
)
mcp.tool(
    consultar_convenios_municipio,
    tags={"convenios", "parcerias", "transferencias", "municipios", "transparencia", "megasoft"},
)
mcp.tool(
    consultar_legislacao_municipio,
    tags={"legislacao", "leis", "decretos", "atos", "municipios", "transparencia", "megasoft"},
)
mcp.tool(
    consultar_concursos_municipio,
    tags={"concursos", "seletivos", "aprovados", "municipios", "transparencia", "megasoft"},
)
mcp.tool(
    consultar_detalhe_servidor_municipio,
    tags={
        "servidor",
        "contracheque",
        "detalhe",
        "rubricas",
        "salario",
        "municipios",
        "transparencia",
        "megasoft",
    },
)

# Resources
mcp.resource("data://estados", mime_type="application/json")(estados_brasileiros)
mcp.resource("data://estatisticas", mime_type="application/json")(estatisticas_nacionais_resource)
mcp.resource("data://capitais", mime_type="application/json")(capitais_brasileiras)
mcp.resource(
    "data://megasoft-municipios",
    mime_type="application/json",
)(megasoft_municipios_resource)

# Prompts
mcp.prompt(auditoria_transparencia_municipio)
mcp.prompt(comparativo_transparencia_regional)
