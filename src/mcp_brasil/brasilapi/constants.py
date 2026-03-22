"""Constants for the BrasilAPI feature."""

# API base URL
BRASILAPI_BASE = "https://brasilapi.com.br/api"

# Endpoint URLs
CEP_URL = f"{BRASILAPI_BASE}/cep/v1"
CNPJ_URL = f"{BRASILAPI_BASE}/cnpj/v1"
DDD_URL = f"{BRASILAPI_BASE}/ddd/v1"
BANKS_URL = f"{BRASILAPI_BASE}/banks/v1"
CAMBIO_MOEDAS_URL = f"{BRASILAPI_BASE}/cambio/v1/moedas"
CAMBIO_COTACAO_URL = f"{BRASILAPI_BASE}/cambio/v1/cotacao"
FERIADOS_URL = f"{BRASILAPI_BASE}/feriados/v1"
TAXAS_URL = f"{BRASILAPI_BASE}/taxas/v1"
FIPE_TABELAS_URL = f"{BRASILAPI_BASE}/fipe/tabelas/v1"
FIPE_MARCAS_URL = f"{BRASILAPI_BASE}/fipe/marcas/v1"
FIPE_VEICULOS_URL = f"{BRASILAPI_BASE}/fipe/veiculos/v1"
ISBN_URL = f"{BRASILAPI_BASE}/isbn/v1"
NCM_URL = f"{BRASILAPI_BASE}/ncm/v1"
PIX_URL = f"{BRASILAPI_BASE}/pix/v1/participants"
REGISTRO_BR_URL = f"{BRASILAPI_BASE}/registrobr/v1"

# Tipos de veículo válidos para FIPE
TIPOS_VEICULO = {"carros", "caminhoes", "motos"}

# Taxas oficiais conhecidas
TAXAS_CONHECIDAS = {
    "SELIC": "Taxa Selic — taxa básica de juros da economia",
    "CDI": "CDI — Certificado de Depósito Interbancário",
    "IPCA": "IPCA — Índice de Preços ao Consumidor Amplo",
    "TR": "TR — Taxa Referencial",
    "INPC": "INPC — Índice Nacional de Preços ao Consumidor",
    "IGPM": "IGP-M — Índice Geral de Preços do Mercado",
}
