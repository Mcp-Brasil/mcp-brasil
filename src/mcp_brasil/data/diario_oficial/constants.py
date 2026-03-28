"""Constants for the Diário Oficial feature."""

# =============================================================================
# Querido Diário (municipal) — Open Knowledge Brasil
# =============================================================================

# API base URL (canônica — sem redirect)
QUERIDO_DIARIO_API = "https://api.queridodiario.ok.org.br"

# Endpoints
GAZETTES_URL = f"{QUERIDO_DIARIO_API}/gazettes"
CITIES_URL = f"{QUERIDO_DIARIO_API}/cities"

# Limites de paginação
DEFAULT_PAGE_SIZE = 10
MAX_PAGE_SIZE = 100

# Capitais com cobertura confirmada (IBGE codes)
CAPITAIS_COBERTAS = {
    "2408102": "Natal/RN",
    "5208707": "Goiânia/GO",
    "2927408": "Salvador/BA",
    "5002704": "Campo Grande/MS",
    "4205407": "Florianópolis/SC",
    "1721000": "Palmas/TO",
    "3304557": "Rio de Janeiro/RJ",
    "2507507": "João Pessoa/PB",
    "2211001": "Teresina/PI",
    "1400100": "Boa Vista/RR",
    "2704302": "Maceió/AL",
    "1302603": "Manaus/AM",
    "3550308": "São Paulo/SP",
    "4106902": "Curitiba/PR",
    "5300108": "Brasília/DF",
    "3106200": "Belo Horizonte/MG",
    "4314902": "Porto Alegre/RS",
    "2304400": "Fortaleza/CE",
    "2611606": "Recife/PE",
    "1501402": "Belém/PA",
}

# UFs cobertas (derivado das capitais — para filtros regionais)
UFS_COBERTAS = sorted({v.split("/")[1] for v in CAPITAIS_COBERTAS.values()})

# =============================================================================
# DOU — Diário Oficial da União (Imprensa Nacional / in.gov.br)
# =============================================================================

DOU_API_BASE = "https://www.in.gov.br"
DOU_SEARCH_URL = f"{DOU_API_BASE}/consulta/-/buscar/dou"
DOU_ARTICLE_URL = f"{DOU_API_BASE}/en/web/dou/-"

DOU_SECTIONS = {
    "SECAO_1": "do1",
    "SECAO_2": "do2",
    "SECAO_3": "do3",
    "EDICAO_EXTRA": "doe",
    "EDICAO_SUPLEMENTAR": "dos",
    "TODOS": "",
}

DOU_FIELDS = {
    "TUDO": "TUDO",
    "TITULO": "TITULO",
    "CONTEUDO": "CONTEUDO",
}

DOU_PERIODS = {
    "DIA": "dia",
    "SEMANA": "semana",
    "MES": "mes",
    "ANO": "ano",
    "PERSONALIZADO": "personalizado",
}

# Seções com descrição (para resources)
DOU_SECTIONS_INFO = {
    "do1": "Seção 1 — Leis, decretos, medidas provisórias, resoluções",
    "do2": "Seção 2 — Atos de pessoal (nomeações, exonerações, aposentadorias)",
    "do3": "Seção 3 — Contratos, licitações, avisos, editais",
    "doe": "Edição Extra — Atos urgentes publicados fora do calendário regular",
    "dos": "Edição Suplementar — Atos complementares",
}

# Tipos de publicação comuns no DOU
DOU_TIPOS_PUBLICACAO = [
    "Decreto",
    "Lei",
    "Medida Provisória",
    "Portaria",
    "Resolução",
    "Instrução Normativa",
    "Despacho",
    "Aviso de Licitação",
    "Extrato de Contrato",
    "Extrato de Convênio",
    "Nomeação",
    "Exoneração",
    "Edital",
    "Parecer",
]
