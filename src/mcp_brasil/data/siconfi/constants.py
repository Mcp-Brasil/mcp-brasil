"""Constants for the SICONFI feature — URLs, codes, and catalogs.

Reference: https://apidatalake.tesouro.gov.br/docs/siconfi/
"""

from __future__ import annotations

SICONFI_API_BASE = "https://apidatalake.tesouro.gov.br/ords/siconfi/tt"

# SICONFI enforces one request per second per client.
RATE_LIMIT_MAX_REQUESTS = 1
RATE_LIMIT_PERIOD_SECONDS = 1.0

# Tipos de demonstrativo (co_tipo_demonstrativo)
TIPO_DEMONSTRATIVO_RREO = "RREO"
TIPO_DEMONSTRATIVO_RREO_SIMPLIFICADO = "RREO Simplificado"
TIPO_DEMONSTRATIVO_RGF = "RGF"
TIPO_DEMONSTRATIVO_RGF_SIMPLIFICADO = "RGF Simplificado"

# Periodicidade RGF (in_periodicidade)
PERIODICIDADE_RGF_QUADRIMESTRAL = "Q"
PERIODICIDADE_RGF_SEMESTRAL = "S"

# Poder (co_poder) — usado em RGF
PODER_EXECUTIVO = "E"
PODER_LEGISLATIVO = "L"
PODER_JUDICIARIO = "J"
PODER_MINISTERIO_PUBLICO = "M"
PODER_DEFENSORIA = "D"

# Esfera (co_esfera) — U=União, E=Estado, M=Município, D=DF
ESFERAS = {"U": "União", "E": "Estado", "M": "Município", "D": "Distrito Federal"}

# Tipos de ente
TIPO_ENTE = {"1": "União", "2": "Estado", "3": "Município", "4": "Distrito Federal"}

# Anexos RREO mais consultados (não exaustivo — usar /anexos-relatorios para lista completa)
ANEXOS_RREO_POPULARES = {
    "RREO-Anexo 01": "Balanço Orçamentário",
    "RREO-Anexo 02": "Demonstrativo da Execução das Despesas por Função/Subfunção",
    "RREO-Anexo 03": "Demonstrativo da Receita Corrente Líquida",
    "RREO-Anexo 06": "Demonstrativo dos Resultados Primário e Nominal",
    "RREO-Anexo 08": "MDE — Manutenção e Desenvolvimento do Ensino",
    "RREO-Anexo 12": "Receitas e Despesas com Ações e Serviços Públicos de Saúde",
    "RREO-Anexo 14": "Demonstrativo Simplificado do RREO",
}

# Anexos RGF mais consultados
ANEXOS_RGF_POPULARES = {
    "RGF-Anexo 01": "Demonstrativo da Despesa com Pessoal",
    "RGF-Anexo 02": "Demonstrativo da Dívida Consolidada Líquida",
    "RGF-Anexo 03": "Demonstrativo das Garantias e Contragarantias",
    "RGF-Anexo 04": "Demonstrativo das Operações de Crédito",
    "RGF-Anexo 06": "Demonstrativo Simplificado do RGF",
}

# Periodos RREO (nr_periodo) — 1 a 6 bimestres
PERIODOS_RREO_BIMESTRAIS = {
    1: "Jan-Fev",
    2: "Mar-Abr",
    3: "Mai-Jun",
    4: "Jul-Ago",
    5: "Set-Out",
    6: "Nov-Dez",
}
