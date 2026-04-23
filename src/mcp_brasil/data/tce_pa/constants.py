"""Constants for the TCE-PA feature."""

API_BASE = "https://sistemas.tcepa.tc.br/dadosabertos/api"

# Endpoint — Diário Oficial publications (year 2018+)
DIARIO_OFICIAL_URL = f"{API_BASE}/v1/diario_oficial"

# Valid TipoAto values accepted by the API
TIPO_ATO = {
    "PESSOAL_REGISTRO": "Atos de Pessoal para Fins de Registro",
    "ATOS_NORMAS": "Atos e Normas",
    "CONTRATOS": "Contratos",
    "CONVENIOS": "Convênios e Instrumentos Congêneres",
    "LICITACOES": "Licitações",
    "OUTROS_PESSOAL": "Outros Atos de Pessoal",
}

# Pesquisa Integrada — Sessões Plenárias
PESQUISA_BASE_URL = "https://www.tcepa.tc.br/pesquisaintegrada"
SESSOES_SEARCH_URL = f"{PESQUISA_BASE_URL}/pesquisa/resultados"

# Database slugs for each type
TIPO_SESSAO: dict[str, str] = {
    "sessoes": "sessoes-plenarias",
    "pautas": "pautas-sessoes-plenarias",
    "atas": "atas-extratos-sessoes-plenarias",
    "videos": "videos-sessoes-plenarias",
}

SESSOES_DEFAULT_RPP = 20

# Pesquisa Integrada — Jurisprudência
BASES_JURISPRUDENCIA: dict[str, str] = {
    "acordaos": "acordaos",
    "acordaos-virtual": "acordaos-plenario-virtual",
    "resolucoes": "resolucoes",
    "portarias": "portarias-tcepa",
    "atos": "atos",
    "informativos": "informativos-jurisprudencia",
    "prejulgados": "prejulgados",
}

# Pesquisa Integrada — Comunicação e Educação
BASES_CONTEUDO: dict[str, str] = {
    "noticias": "noticias-portal-internet",
    "acervo": "acervo-bibliografico",
    "educacao": "acoes-educacionais",
    "youtube": "canal-youtube-tce-pa",
    "imagens": "banco-imagens",
}

# Bases cujo documento é PDF (usa /download); restantes usam /conteudo-original
BASES_PDF_DOWNLOAD = {
    "acordaos",
    "acordaos-plenario-virtual",
    "resolucoes",
    "portarias-tcepa",
    "atos",
    "informativos-jurisprudencia",
    "prejulgados",
    "acervo-bibliografico",
    "atas-extratos-sessoes-plenarias",
    "pautas-sessoes-plenarias",
}
