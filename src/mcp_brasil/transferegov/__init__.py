"""Feature TransfereGov — emendas parlamentares pix (transferências especiais)."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="transferegov",
    description=(
        "TransfereGov — emendas parlamentares pix (transferências especiais): "
        "busca por autor, município, ano e detalhamento."
    ),
    version="0.1.0",
    api_base="https://api.transferegov.gestao.gov.br",
    requires_auth=False,
    tags=["emendas", "pix", "transferencias", "parlamentar", "municipio"],
)
