"""Constants for the Anuncios Eleitorais feature."""

# API base URL
META_GRAPH_API_BASE = "https://graph.facebook.com/v21.0"

# Endpoints
ADS_ARCHIVE_URL = f"{META_GRAPH_API_BASE}/ads_archive"

# Default fields for political ads in Brazil
CAMPOS_ANUNCIO_BASICO = (
    "id,ad_creation_time,ad_creative_bodies,ad_creative_link_captions,"
    "ad_creative_link_descriptions,ad_creative_link_titles,"
    "ad_delivery_start_time,ad_delivery_stop_time,ad_snapshot_url,"
    "bylines,currency,languages,page_id,page_name,publisher_platforms"
)

CAMPOS_ANUNCIO_POLITICO = (
    f"{CAMPOS_ANUNCIO_BASICO},"
    "spend,impressions,demographic_distribution,delivery_by_region,"
    "estimated_audience_size,br_total_reach,"
    "age_country_gender_reach_breakdown,target_ages,target_gender,target_locations"
)

# Ad active status
STATUS_ATIVO = "ACTIVE"
STATUS_INATIVO = "INACTIVE"
STATUS_TODOS = "ALL"

# Ad types
TIPO_POLITICO = "POLITICAL_AND_ISSUE_ADS"
TIPO_TODOS = "ALL"
TIPO_EMPREGO = "EMPLOYMENT_ADS"
TIPO_FINANCEIRO = "FINANCIAL_PRODUCTS_AND_SERVICES_ADS"
TIPO_HABITACAO = "HOUSING_ADS"

# Search types
BUSCA_PALAVRAS = "KEYWORD_UNORDERED"
BUSCA_FRASE_EXATA = "KEYWORD_EXACT_PHRASE"

# Media types
MIDIA_TODAS = "ALL"
MIDIA_IMAGEM = "IMAGE"
MIDIA_MEME = "MEME"
MIDIA_VIDEO = "VIDEO"
MIDIA_NENHUMA = "NONE"

# Publisher platforms
PLATAFORMAS = [
    "FACEBOOK",
    "INSTAGRAM",
    "AUDIENCE_NETWORK",
    "MESSENGER",
    "WHATSAPP",
    "THREADS",
]

# Brazilian states for delivery_by_region
ESTADOS_BRASILEIROS: dict[str, str] = {
    "AC": "Acre",
    "AL": "Alagoas",
    "AM": "Amazonas",
    "AP": "Amapá",
    "BA": "Bahia",
    "CE": "Ceará",
    "DF": "Distrito Federal",
    "ES": "Espírito Santo",
    "GO": "Goiás",
    "MA": "Maranhão",
    "MG": "Minas Gerais",
    "MS": "Mato Grosso do Sul",
    "MT": "Mato Grosso",
    "PA": "Pará",
    "PB": "Paraíba",
    "PE": "Pernambuco",
    "PI": "Piauí",
    "PR": "Paraná",
    "RJ": "Rio de Janeiro",
    "RN": "Rio Grande do Norte",
    "RO": "Rondônia",
    "RR": "Roraima",
    "RS": "Rio Grande do Sul",
    "SC": "Santa Catarina",
    "SE": "Sergipe",
    "SP": "São Paulo",
    "TO": "Tocantins",
}

# Estimated audience size boundaries
FAIXAS_AUDIENCIA = [100, 1000, 5000, 10000, 50000, 100000, 500000, 1000000]

# Default limit per request
LIMITE_PADRAO = 25
LIMITE_MAXIMO = 500
