"""Feature BNDES — dados abertos do Banco Nacional de Desenvolvimento."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="bndes",
    description=(
        "BNDES dados abertos: operações de financiamento, exportação, "
        "desembolsos, instituições credenciadas e datasets via CKAN API."
    ),
    version="0.1.0",
    api_base="https://dadosabertos.bndes.gov.br/api/3/action",
    requires_auth=False,
    tags=["bndes", "financiamento", "desenvolvimento", "ckan", "dados-abertos"],
)
