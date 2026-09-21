"""Feature NucleoGov — portais da transparência municipais."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="nucleogov",
    description=(
        "NucleoGov: portais da transparência municipais — empenhos/despesas "
        "(inclusive por CNPJ do fornecedor), contratos, licitações, dispensas e "
        "inexigibilidades e legislação municipal (leis, decretos, portarias)."
    ),
    version="0.1.0",
    api_base="",
    requires_auth=False,
    tags=[
        "nucleogov",
        "transparencia",
        "municipal",
        "despesas",
        "contratos",
        "licitacoes",
        "legislacao",
    ],
)
