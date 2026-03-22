"""Constants for the Diário Oficial feature."""

# API base URL — Querido Diário (Open Knowledge Brasil)
QUERIDO_DIARIO_API = "https://queridodiario.ok.org.br/api"

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
