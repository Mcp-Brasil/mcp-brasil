"""Feature Diário Oficial — busca em diários oficiais municipais via Querido Diário."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="diario_oficial",
    description=(
        "Querido Diário: busca textual em diários oficiais municipais de 5.000+ cidades. "
        "Contratos, nomeações, sanções, licitações e atos administrativos."
    ),
    version="0.1.0",
    api_base="https://queridodiario.ok.org.br",
    requires_auth=False,
    tags=["diario-oficial", "transparencia", "municipios", "licitacoes", "contratos"],
)
