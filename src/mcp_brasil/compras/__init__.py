"""Feature Compras — contratações públicas via PNCP."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="compras",
    description=(
        "PNCP — Portal Nacional de Contratações Públicas: licitações, contratos, "
        "atas de registro de preço e fornecedores (Lei 14.133/2021)."
    ),
    version="0.1.0",
    api_base="https://pncp.gov.br/api/consulta",
    requires_auth=False,
    tags=["licitacoes", "contratos", "compras", "pncp", "fornecedores"],
)
