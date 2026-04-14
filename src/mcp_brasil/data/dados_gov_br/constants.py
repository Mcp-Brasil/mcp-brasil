"""Constants for the dados_gov_br feature."""

# API base — authenticated endpoints
API_BASE = "https://dados.gov.br/dados/api"
API_PUBLICO = f"{API_BASE}/publico"

# Endpoints
CONJUNTOS_URL = f"{API_PUBLICO}/conjuntos-dados"
ORGANIZACAO_URL = f"{API_PUBLICO}/organizacao"
REUSOS_URL = f"{API_PUBLICO}/reusos"
REUSO_URL = f"{API_PUBLICO}/reuso"
TEMAS_URL = f"{API_BASE}/temas"
TAGS_URL = f"{API_BASE}/tags"

# Auth
AUTH_HEADER_NAME = "chave-api-dados-abertos"
AUTH_ENV_VAR = "DADOS_GOV_BR_API_KEY"

# Pagination
DEFAULT_PAGE_SIZE = 10
MAX_PAGE_SIZE = 50
