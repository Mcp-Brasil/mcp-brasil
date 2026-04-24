"""Feature bcb_olinda — APIs OData do Banco Central (Olinda).

Complementa ``data/bacen`` (SGS) com:
- **PTAX** — cotações oficiais diárias USD e outras moedas
- **Expectativas Focus** — expectativas de mercado para Selic, IPCA, PIB, câmbio
- **Taxa de Juros** — taxas médias bancárias por modalidade
"""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="bcb_olinda",
    description=(
        "BCB Olinda — PTAX (câmbio oficial), Expectativas Focus (Selic/IPCA/PIB), "
        "taxas de juros bancárias"
    ),
    version="0.1.0",
    api_base="https://olinda.bcb.gov.br/olinda/servico",
    requires_auth=False,
    tags=["bcb", "ptax", "cambio", "focus", "expectativas", "selic", "ipca", "juros"],
)
