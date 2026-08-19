"""Constants for the Tabua Mares feature."""

# API base URL
# O dominio antigo (tabuademares.com) passou a responder 403 e a v1 foi
# removida pelo mantenedor da API; tudo migrou para tabuamare.api.br (ver #28).
TABUA_MARES_BASE = "https://tabuamare.api.br"
TABUA_MARES_API = f"{TABUA_MARES_BASE}/api/v2"

# Endpoints
STATES_URL = f"{TABUA_MARES_API}/states"
HARBOR_NAMES_URL = f"{TABUA_MARES_API}/harbor_names"
HARBORS_URL = f"{TABUA_MARES_API}/harbors"
TABUA_MARE_URL = f"{TABUA_MARES_API}/tabua-mare"
NEAREST_HARBOR_URL = f"{TABUA_MARES_API}/nearested-harbor"
NEAREST_HARBOR_INDEPENDENT_URL = f"{TABUA_MARES_API}/nearest-harbor-independent-state"
# Antes ficava na raiz do dominio; no dominio novo esta sob /api/v2 (verificado).
GEO_TABUA_MARE_URL = f"{TABUA_MARES_API}/geo-tabua-mare"

# 17 estados costeiros do Brasil
ESTADOS_COSTEIROS: dict[str, str] = {
    "al": "Alagoas",
    "ap": "Amapá",
    "ba": "Bahia",
    "ce": "Ceará",
    "es": "Espírito Santo",
    "ma": "Maranhão",
    "pa": "Pará",
    "pb": "Paraíba",
    "pe": "Pernambuco",
    "pi": "Piauí",
    "pr": "Paraná",
    "rj": "Rio de Janeiro",
    "rn": "Rio Grande do Norte",
    "rs": "Rio Grande do Sul",
    "sc": "Santa Catarina",
    "se": "Sergipe",
    "sp": "São Paulo",
}
