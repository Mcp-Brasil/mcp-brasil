"""Feature Camara — Câmara dos Deputados (Dados Abertos)."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="camara",
    description="Câmara dos Deputados: deputados, proposições, votações, despesas, comissões",
    version="0.1.0",
    api_base="https://dadosabertos.camara.leg.br/api/v2",
    requires_auth=False,
    tags=["legislativo", "deputados", "proposições", "votações", "despesas", "comissões"],
)
