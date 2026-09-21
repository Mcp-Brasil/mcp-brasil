"""Constants for the Megasoft municipal transparency portal feature.

Megasoft powers transparency portals of many Brazilian municipalities under
the domain ``{municipio}.megasofttransparencia.com.br``. Data is exposed as a
paginated JSON API under ``/api/`` that requires a short-lived bearer token
issued by the portal itself (see ``client._obter_token``).

Endpoints used by this feature (all under ``/api/``):
    - configuracao/carregamento-inicial                     -> portal config (token)
    - receitas-e-despesas/empenho/paginado                  -> empenhos
    - receitas-e-despesas/ordem-de-pagamento/paginado       -> ordens de pagamento
    - receitas-e-despesas/ordem-de-pagamento/dados-totalizados -> totais financeiros
    - receitas-e-despesas/receita/paginado                  -> receitas
"""

MEGASOFT_DOMAIN = "megasofttransparencia.com.br"

# Only hosts under this suffix may ever be requested (single subdomain label,
# HTTPS, no port). See client._portal_base.
PORTAL_HOST_SUFFIX = "." + MEGASOFT_DOMAIN

# Portal configuration endpoint (also the source of the bearer token).
CONFIG_PATH = "configuracao/carregamento-inicial"

# Header required by every data call. Missing it yields a misleading HTTP 500.
CLIENTE_INTEGRADO_HEADER = "cliente-integrado"
CLIENTE_INTEGRADO_VALOR = "megasoft-portal-da-transparencia"

# Data endpoints.
EMPENHO_PATH = "receitas-e-despesas/empenho/paginado"
ORDEM_PAGAMENTO_PATH = "receitas-e-despesas/ordem-de-pagamento/paginado"
ORDEM_PAGAMENTO_TOTAIS_PATH = "receitas-e-despesas/ordem-de-pagamento/dados-totalizados"
RECEITA_PATH = "receitas-e-despesas/receita/paginado"

# Optional environment override for the portal bearer token. When unset, the
# token is obtained automatically from the portal configuration endpoint.
TOKEN_ENV_VAR = "MEGASOFT_TOKEN"
TOKEN_CACHE_TTL_SECONDS = 1800.0

# Pagination.
DEFAULT_PAGE = 1
DEFAULT_PAGE_SIZE = 100
MAX_PAGE_SIZE = 100

# Response envelopes vary per endpoint, so the list and total keys are probed.
LIST_KEYS = ("dados", "content", "registros", "items", "itens")
TOTAL_KEYS = ("total", "totalElements", "totalRegistros", "count")

# Fields expected in the non-paginated totals endpoint envelope.
TOTAIS_KEYS = ("valorPago", "valorAnulacao", "valorTotal", "valorRetencao")
