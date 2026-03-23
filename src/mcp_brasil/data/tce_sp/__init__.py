"""Feature TCE-SP — Tribunal de Contas do Estado de São Paulo."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="tce_sp",
    description=(
        "TCE-SP: despesas e receitas dos 645 municípios paulistas, "
        "com dados mensais desde 2014 até o exercício atual."
    ),
    version="0.1.0",
    api_base="https://transparencia.tce.sp.gov.br/api",
    requires_auth=False,
    tags=["tce", "sp", "despesas", "receitas", "municipios"],
)
