"""Feature SICONFI — Tesouro Nacional.

Sistema de Informações Contábeis e Fiscais do Setor Público Brasileiro.
Dados de RREO, RGF, DCA e MSC de todos os entes subnacionais (municípios,
estados, DF e União).
"""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="siconfi",
    description=(
        "Finanças públicas subnacionais: RREO, RGF, DCA de municípios/estados via Tesouro Nacional"
    ),
    version="0.1.0",
    api_base="https://apidatalake.tesouro.gov.br/ords/siconfi/tt",
    requires_auth=False,
    tags=["transparencia", "fiscal", "orcamento", "municipios", "estados", "lrf"],
)
