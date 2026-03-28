"""Constants for the Saúde feature."""

SAUDE_API_BASE = "https://apidadosabertos.saude.gov.br"
CNES_API_BASE = f"{SAUDE_API_BASE}/cnes"

ESTABELECIMENTOS_URL = f"{CNES_API_BASE}/estabelecimentos"
TIPOS_URL = f"{CNES_API_BASE}/tipounidades"
LEITOS_URL = f"{SAUDE_API_BASE}/assistencia-a-saude/hospitais-e-leitos"

DEFAULT_LIMIT = 20
MAX_LIMIT = 20  # API enforces max 20 for estabelecimentos
MAX_LIMIT_LEITOS = 1000  # API allows up to 1000 for hospitais-e-leitos

# Códigos de tipo de unidade para urgência/emergência (CNES)
TIPOS_URGENCIA: dict[str, str] = {
    "36": "Clínica/Centro de Especialidade",
    "39": "Unidade de Serviço de Apoio de Diagnose e Terapia",
    "40": "Unidade Móvel Terrestre",
    "42": "Unidade Móvel de Nível Pré-Hospitalar na Área de Urgência",
    "73": "Pronto Atendimento",
    "74": "Polo Academia da Saúde",
    "76": "Central de Regulação Médica das Urgências",
}
