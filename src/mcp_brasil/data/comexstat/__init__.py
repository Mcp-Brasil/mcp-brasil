"""Feature ComexStat — Brazilian foreign trade statistics (MDIC)."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="comexstat",
    description=(
        "ComexStat MDIC: estatísticas de exportações e importações brasileiras — "
        "dados mensais desde 1997 por NCM, país e período, sem autenticação."
    ),
    version="0.1.0",
    api_base="https://api-comexstat.mdic.gov.br",
    requires_auth=False,
    tags=["comercio-exterior", "exportacao", "importacao", "mdic", "comexstat"],
)
