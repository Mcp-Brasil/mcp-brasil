"""Constants for the Câmara dos Deputados feature."""

# Pagination
DEFAULT_PAGE_SIZE = 15
DEFAULT_ITENS = 15

# API base URL
CAMARA_API_BASE = "https://dadosabertos.camara.leg.br/api/v2"

# Endpoints
DEPUTADOS_URL = f"{CAMARA_API_BASE}/deputados"
PROPOSICOES_URL = f"{CAMARA_API_BASE}/proposicoes"
VOTACOES_URL = f"{CAMARA_API_BASE}/votacoes"
EVENTOS_URL = f"{CAMARA_API_BASE}/eventos"
ORGAOS_URL = f"{CAMARA_API_BASE}/orgaos"
FRENTES_URL = f"{CAMARA_API_BASE}/frentes"
