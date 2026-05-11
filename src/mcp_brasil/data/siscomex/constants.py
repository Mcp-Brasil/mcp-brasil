"""Constants for the Siscomex feature."""

NCM_BASE_URL = "https://portalunico.siscomex.gov.br/classif/api/publico/nomenclatura/download/json"
NCM_PARAMS = {"perfil": "PUBLICO"}

CACHE_TTL_SECONDS = 86400  # 24h — endpoint updates at midnight daily
NCM_MAX_RESULTS = 50
