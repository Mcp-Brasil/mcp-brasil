"""Constants for the BNDES CKAN API."""

# CKAN API base
API_BASE = "https://dadosabertos.bndes.gov.br/api/3/action"

# CKAN metadata endpoints
PACKAGE_LIST_URL = f"{API_BASE}/package_list"
PACKAGE_SHOW_URL = f"{API_BASE}/package_show"
PACKAGE_SEARCH_URL = f"{API_BASE}/package_search"
DATASTORE_SEARCH_URL = f"{API_BASE}/datastore_search"

# Key DataStore resource IDs (queryable via datastore_search)
RESOURCE_OPERACOES_AUTOMATICAS = "612faa0b-b6be-4b2c-9317-da5dc2c0b901"
RESOURCE_OPERACOES_NAO_AUTOMATICAS = "6f56b78c-510f-44b6-8274-78a5b7e931f4"
RESOURCE_INSTITUICOES_CREDENCIADAS = "532c7024-471f-424a-b1a0-475302507c33"

# Pagination defaults
DEFAULT_LIMIT = 20
MAX_LIMIT = 100
