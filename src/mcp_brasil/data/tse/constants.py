"""Constants for the TSE (Tribunal Superior Eleitoral) feature."""

# API base URL — DivulgaCandContas (candidatos, prestação de contas)
TSE_API_BASE = "https://divulgacandcontas.tse.jus.br/divulga/rest/v1"

# Endpoints — DivulgaCandContas
ELEICAO_URL = f"{TSE_API_BASE}/eleicao"
CANDIDATURA_URL = f"{TSE_API_BASE}/candidatura"
PRESTADOR_URL = f"{TSE_API_BASE}/prestador"

# Default pagination
DEFAULT_PAGE_SIZE = 50

# ---------------------------------------------------------------------------
# CDN de Resultados — arquivos JSON estáticos pré-gerados pelo TSE
# https://resultados.tse.jus.br/oficial/
# ---------------------------------------------------------------------------
RESULTADOS_CDN_BASE = "https://resultados.tse.jus.br/oficial"
RESULTADOS_CONFIG_URL = f"{RESULTADOS_CDN_BASE}/comum/config/ele-c.json"

# Mapeamento cargo → código CDN (formato 4 dígitos zero-padded)
CARGO_CODES_CDN: dict[str, str] = {
    "presidente": "0001",
    "vice_presidente": "0002",
    "governador": "0003",
    "vice_governador": "0004",
    "senador": "0005",
    "deputado_federal": "0006",
    "deputado_estadual": "0007",
    "deputado_distrital": "0008",
    "prefeito": "0011",
    "vice_prefeito": "0012",
    "vereador": "0013",
}

# UFs brasileiras para iteração
UFS_BRASIL = [
    "ac",
    "al",
    "am",
    "ap",
    "ba",
    "ce",
    "df",
    "es",
    "go",
    "ma",
    "mg",
    "ms",
    "mt",
    "pa",
    "pb",
    "pe",
    "pi",
    "pr",
    "rj",
    "rn",
    "ro",
    "rr",
    "rs",
    "sc",
    "se",
    "sp",
    "to",
]

# Mapeamento ano+turno → ciclo+eleição (cache manual para eleições conhecidas)
# Formato: (ano, turno) → (ciclo, eleicao_padded, eleicao_unpadded)
# - padded: usado em filenames (e000544)
# - unpadded: usado em paths do CDN (/544/)
ELEICOES_CDN: dict[tuple[int, int], tuple[str, str, str]] = {
    (2022, 1): ("ele2022", "000544", "544"),
    (2022, 2): ("ele2022", "000545", "545"),
    (2024, 1): ("ele2024", "000619", "619"),
    (2024, 2): ("ele2024", "000620", "620"),
}

# Códigos de cargos eleitorais
CARGOS_ELEITORAIS: list[dict[str, str | int]] = [
    {"codigo": 1, "nome": "Presidente", "tipo": "Federal"},
    {"codigo": 2, "nome": "Vice-Presidente", "tipo": "Federal"},
    {"codigo": 3, "nome": "Governador", "tipo": "Estadual"},
    {"codigo": 4, "nome": "Vice-Governador", "tipo": "Estadual"},
    {"codigo": 5, "nome": "Senador", "tipo": "Federal"},
    {"codigo": 6, "nome": "Deputado Federal", "tipo": "Federal"},
    {"codigo": 7, "nome": "Deputado Estadual", "tipo": "Estadual"},
    {"codigo": 8, "nome": "Deputado Distrital", "tipo": "Distrital"},
    {"codigo": 9, "nome": "1º Suplente Senador", "tipo": "Federal"},
    {"codigo": 10, "nome": "2º Suplente Senador", "tipo": "Federal"},
    {"codigo": 11, "nome": "Prefeito", "tipo": "Municipal"},
    {"codigo": 12, "nome": "Vice-Prefeito", "tipo": "Municipal"},
    {"codigo": 13, "nome": "Vereador", "tipo": "Municipal"},
]
