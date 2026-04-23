"""Feature TCE-PA — Tribunal de Contas do Estado do Pará."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="tce_pa",
    description=(
        "TCE-PA: Diário Oficial, sessões plenárias, jurisprudência (acórdãos, "
        "resoluções, portarias, prejulgados) e conteúdo informativo do Tribunal "
        "de Contas do Estado do Pará via API de Dados Abertos e Pesquisa Integrada."
    ),
    version="0.1.0",
    api_base="https://sistemas.tcepa.tc.br/dadosabertos/api",
    requires_auth=False,
    tags=["tce", "pa", "diario-oficial", "sessoes", "acordaos", "jurisprudencia", "pará"],
)
