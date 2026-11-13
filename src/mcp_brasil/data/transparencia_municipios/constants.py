"""Constantes e URLs da feature Transparência Municípios e Estados."""

from pathlib import Path

# Caminho para o binário nativo em Go (priorizado para velocidade extrema)
PROJECT_ROOT = Path(__file__).resolve().parents[4]
GO_BINARY_PATH = PROJECT_ROOT / "bin" / "transparencia-go"

# Caminho para dados embutidos
DATA_DIR = Path(__file__).resolve().parent / "data"
ESTADOS_CSV_PATH = DATA_DIR / "estados.csv"
MUNICIPIOS_CSV_PATH = DATA_DIR / "municipios.csv"
MEGASOFT_CSV_PATH = DATA_DIR / "megasoft_municipios.csv"

# Portais e bases públicas nacionais de transparência
RADAR_ATRICON_BASE = "https://radardatransparencia.atricon.org.br"
SICONFI_BASE = "https://apidatalake.tesouro.gov.br/ords/siconfi/tt"
PNCP_BASE = "https://pncp.gov.br"
TRANSFEREGOV_BASE = "https://transferegov.sistema.gov.br"
TRANSPARENCIA_FEDERAL_BASE = "https://portaldatransparencia.gov.br"

TOTAL_MUNICIPIOS_OFICIAL = 5571
TOTAL_UFS_OFICIAL = 27
