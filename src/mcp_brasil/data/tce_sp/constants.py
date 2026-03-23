"""Constants for the TCE-SP feature."""

API_BASE = "https://transparencia.tce.sp.gov.br/api"

# JSON endpoints
MUNICIPIOS_URL = f"{API_BASE}/json/municipios"
DESPESAS_URL = f"{API_BASE}/json/despesas"  # /{municipio}/{exercicio}/{mes}
RECEITAS_URL = f"{API_BASE}/json/receitas"  # /{municipio}/{exercicio}/{mes}
