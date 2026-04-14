"""Feature dados_gov_br — Portal Brasileiro de Dados Abertos."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="dados_gov_br",
    description=(
        "Portal Brasileiro de Dados Abertos (dados.gov.br): catálogo central "
        "de datasets abertos do governo federal. Busca, listagem e detalhamento "
        "de conjuntos de dados, organizações, temas, tags e reusos."
    ),
    version="2.0.0",
    api_base="https://dados.gov.br/dados/api",
    requires_auth=True,
    auth_env_var="DADOS_GOV_BR_API_KEY",
    tags=["dados-abertos", "datasets", "portal", "governo-federal", "catalogo"],
)
