"""Constants for the TCE-CE feature.

API de Dados Abertos do SIM 2.0 (Denodo). Documentacao Swagger em
https://api-dados-abertos.tce.ce.gov.br/sim/ (spec: openapi_prod.yaml).

A versao anterior (sem o prefixo /sim, parametros quantidade/deslocamento)
foi desligada: os caminhos antigos respondem 404 desde 2026.
"""

API_BASE = "https://api-dados-abertos.tce.ce.gov.br/sim"

# Endpoints
MUNICIPIOS_URL = f"{API_BASE}/municipios"
PROCESSOS_URL = f"{API_BASE}/processos_administrativos_contratacoes"
CONTRATOS_URL = f"{API_BASE}/contratos"
CONTRATADOS_URL = f"{API_BASE}/contratados"
EMPENHOS_URL = f"{API_BASE}/notas_empenhos"

# Paginacao: $count (max 1000 por requisicao) + $start_index (offset)
DEFAULT_QUANTIDADE = 100
MAX_QUANTIDADE = 1000

# Freio de registros de contratados lidos para cruzar com uma pagina de contratos.
# A busca usa so o intervalo de datas da pagina; se o freio parar a leitura,
# o resultado sai com contratado_incompleto=True (nunca corte silencioso).
MAX_CONTRATADOS_JOIN = 20000
