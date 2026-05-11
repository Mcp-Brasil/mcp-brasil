"""Feature Siscomex — Official NCM (Mercosur Common Nomenclature) table."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="siscomex",
    description=(
        "Siscomex Portal Único: Nomenclatura Comum do Mercosul (NCM) — "
        "tabela oficial diretamente do governo federal, com rastreamento de "
        "vigência e referências legais para conformidade regulatória."
    ),
    version="0.1.0",
    api_base="https://portalunico.siscomex.gov.br",
    requires_auth=False,
    tags=["ncm", "comercio-exterior", "siscomex", "receita-federal"],
)
