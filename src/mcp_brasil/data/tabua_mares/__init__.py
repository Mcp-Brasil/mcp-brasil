"""Feature Tabua Mares — Dados de marés do litoral brasileiro."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="tabua_mares",
    description="Tábua de marés — previsão de marés para portos do litoral brasileiro",
    version="0.1.0",
    api_base="https://tabuamare.api.br/api/v2",
    requires_auth=False,
    tags=["mares", "portos", "litoral", "navegacao", "oceanografia"],
)
