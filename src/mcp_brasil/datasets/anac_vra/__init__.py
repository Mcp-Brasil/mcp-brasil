"""Feature anac_vra — Voo Regular Ativo mensal (ANAC).

Agrega CSVs mensais de voos realizados vs previstos: partidas/chegadas
reais, atrasos, cancelamentos com código de justificativa.

Cobertura inicial: 12 meses de 2024 (multi-source DatasetSpec).
Pode expandir pra outros anos editando ``_ANOS`` e ``_MESES``.

Ativação: ``MCP_BRASIL_DATASETS=anac_vra`` no ``.env``.
"""

from urllib.parse import quote

from mcp_brasil import settings
from mcp_brasil._shared.datasets import DatasetSpec
from mcp_brasil._shared.feature import FeatureMeta

DATASET_ID = "anac_vra"
DATASET_TABLE = "vra"

_BASE_URL = (
    "https://sistemas.anac.gov.br/dadosabertos/Voos e operações aéreas/Voo Regular Ativo (VRA)"
)

# (mês_num, nome_pt)
_MESES = [
    (1, "Janeiro"),
    (2, "Fevereiro"),
    (3, "Março"),
    (4, "Abril"),
    (5, "Maio"),
    (6, "Junho"),
    (7, "Julho"),
    (8, "Agosto"),
    (9, "Setembro"),
    (10, "Outubro"),
    (11, "Novembro"),
    (12, "Dezembro"),
]

_ANOS = [2024]


def _build_sources() -> tuple[tuple[str, str | None, str], ...]:
    out: list[tuple[str, str | None, str]] = []
    for ano in _ANOS:
        for mes_num, mes_nome in _MESES:
            pasta = f"{mes_num:02d} - {mes_nome}"
            # 2024 e antes: filename com mês zero-padded; 2025+: sem padding
            arquivo = f"VRA_{ano}{mes_num:02d}.csv" if ano <= 2024 else f"VRA_{ano}{mes_num}.csv"
            url = f"{_BASE_URL}/{ano}/{pasta}/{arquivo}"
            # Monta URL percent-encoded
            encoded = (
                url.replace(" ", "%20")
                .replace("(", "%28")
                .replace(")", "%29")
                .replace("ç", quote("ç"))
                .replace("õ", quote("õ"))
                .replace("é", quote("é"))
                .replace("Ã", quote("Ã"))
                .replace("á", quote("á"))
            )
            suffix = f"{ano}{mes_num:02d}"
            out.append((encoded, None, suffix))
    return tuple(out)


_SOURCES = _build_sources()

DATASET_SPEC = DatasetSpec(
    id=DATASET_ID,
    url=_SOURCES[-1][0],
    table=DATASET_TABLE,
    ttl_days=30,
    approx_size_mb=150,  # ~12 MB/mês x 12 meses
    source="ANAC — Voo Regular Ativo (VRA), dados abertos mensais",
    description=(
        "Voos regulares domésticos e internacionais — previstos vs realizados, "
        "atrasos, cancelamentos, justificativas. 12 meses de 2024."
    ),
    source_encoding="cp1252",
    csv_options={
        "delim": ";",
        "header": True,
        "skip": 1,  # pula linha 'Atualizado em: YYYY-MM-DD'
        "quote": '"',
        "ignore_errors": True,
        "sample_size": -1,
        "all_varchar": True,
        "normalize_names": True,
    },
    pii_columns=frozenset(),
    sources=_SOURCES,
)

FEATURE_META = FeatureMeta(
    name=DATASET_ID,
    description=(
        "ANAC VRA — Voo Regular Ativo mensal (2024) com consulta SQL via "
        "DuckDB. ~12 CSVs agregados. Opt-in: MCP_BRASIL_DATASETS=anac_vra."
    ),
    version="0.1.0",
    api_base="https://sistemas.anac.gov.br",
    requires_auth=False,
    enabled=DATASET_ID in settings.DATASETS_ENABLED,
    tags=[
        "anac",
        "aviacao",
        "voos",
        "vra",
        "pontualidade",
        "dataset",
        "duckdb",
    ],
)
