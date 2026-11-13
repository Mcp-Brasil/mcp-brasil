"""Feature Transparência Municípios e Estados.

Conexão e busca nativa de transparência para todos os 5.570 municípios e 27 UFs.
"""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="transparencia_municipios",
    description=(
        "Conexão, descoberta de portais, servidores públicos e transparência completa "
        "para todos os 5.570 municípios e 27 estados do Brasil (Go engine)"
    ),
    version="0.2.0",
    api_base="https://radardatransparencia.atricon.org.br",
    requires_auth=False,
    tags=[
        "transparencia",
        "municipios",
        "estados",
        "servidores",
        "despesas",
        "receitas",
        "tce",
        "siconfi",
        "pncp",
        "orcamento",
    ],
)
