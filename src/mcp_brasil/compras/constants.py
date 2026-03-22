"""Constants for the Compras feature."""

# API base URL — Portal Nacional de Contratações Públicas
PNCP_API_BASE = "https://pncp.gov.br/api/consulta/v1"

# Endpoints
CONTRATACOES_URL = f"{PNCP_API_BASE}/contratacoes/publicacao"
CONTRATOS_URL = f"{PNCP_API_BASE}/contratos"
ATAS_URL = f"{PNCP_API_BASE}/atas"

# Paginação
DEFAULT_PAGE_SIZE = 10
MAX_PAGE_SIZE = 50

# Modalidades de licitação (Lei 14.133/2021)
MODALIDADES = {
    1: "Pregão eletrônico",
    2: "Concorrência",
    3: "Concurso",
    4: "Leilão",
    5: "Diálogo competitivo",
    6: "Dispensa de licitação",
    7: "Inexigibilidade",
    8: "Pregão presencial",
    9: "Pré-qualificação",
    10: "Credenciamento",
    11: "Manifestação de interesse",
    12: "Regime diferenciado de contratações",
    13: "Tomada de preço",
    14: "Convite",
}
