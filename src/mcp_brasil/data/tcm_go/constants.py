"""Constants for the TCM-GO feature."""

# eSiproc — JSON (Elasticsearch) API of electronic proceedings.
ESIPROC_BASE = "https://virtual.tcmgo.tc.br/eSiproc/api/elastic"
ESIPROC_PESQUISAR_URL = f"{ESIPROC_BASE}/pesquisar"
ESIPROC_LISTAR_CAMPO_URL = f"{ESIPROC_BASE}/listar-campo"

# TCMJuris — server-rendered HTML search of julgados.
TCMJURIS_URL = "https://www.tcmgo.tc.br/ecs/s/tcmjuris"
TCMJURIS_BASE = "https://www.tcmgo.tc.br"

# TCMJuris rejects requests from headless clients (it blocks a "HeadlessChrome"
# User-Agent), so a regular desktop-browser User-Agent is sent on every request
# to that surface. No authentication is required.
BROWSER_USER_AGENT = (
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
    "(KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36"
)
TCMJURIS_HEADERS = {
    "User-Agent": BROWSER_USER_AGENT,
    "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
    "Accept-Language": "pt-BR,pt;q=0.9",
}

# eSiproc returns about 10 records per page (0-indexed).
ESIPROC_PAGE_SIZE = 10

# TCMJuris caps a single response at 1000 rows.
TCMJURIS_MAX_ROWS = 1000

# Filter fields accepted by the eSiproc proceedings search.
CAMPOS_ESIPROC = (
    "municipio.keyword",
    "unidade_gestora.keyword",
    "relator.keyword",
    "regiao.keyword",
    "tipo_protocolo.keyword",
    "ano_competencia",
    "periodo_competencia.keyword",
    "lotacao.keyword",
    "finalizado.keyword",
)

# Aggregatable fields exposed by the eSiproc value-listing endpoint.
CAMPOS_AGREGAVEIS = (
    "municipio.keyword",
    "unidade_gestora.keyword",
    "relator.keyword",
    "regiao.keyword",
    "tipo_protocolo.keyword",
    "ano_competencia",
    "periodo_competencia.keyword",
    "lotacao.keyword",
    "area.keyword",
    "finalizado.keyword",
)

# Search operators accepted by TCMJuris free-text search.
OPLOG_VALIDOS = ("AND", "OR")

# Explicit "no results" markers rendered by TCMJuris. A 200 HTML response
# without the results table is only accepted as a legitimate empty result when
# one of these markers is present; otherwise the layout is unexpected and the
# client raises instead of reporting a silent empty list.
TCMJURIS_SEM_RESULTADOS = (
    "nenhum registro encontrado",
    "nenhum resultado encontrado",
    "nenhum acordao encontrado",
    "nenhum resultado foi encontrado",
    "nao foram encontrados",
    "sem resultados",
)
