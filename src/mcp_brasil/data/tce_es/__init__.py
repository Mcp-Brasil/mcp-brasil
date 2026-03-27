"""Feature TCE-ES — Tribunal de Contas do Estado do Espírito Santo."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="tce_es",
    description=(
        "TCE-ES: licitações, contratos, contratações municipais e obras públicas "
        "do Espírito Santo via API de Dados Abertos (dados.es.gov.br)."
    ),
    version="0.1.0",
    api_base="https://dados.es.gov.br/api/3/action",
    requires_auth=False,
    tags=["tce", "es", "licitacoes", "contratos", "obras", "municipios"],
)
