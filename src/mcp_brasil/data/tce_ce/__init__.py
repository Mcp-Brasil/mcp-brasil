"""Feature TCE-CE — Tribunal de Contas do Estado do Ceará."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="tce_ce",
    description=(
        "TCE-CE: licitações, contratos (com o contratado) e empenhos dos municípios "
        "cearenses via API de Dados Abertos do SIM 2.0 (Sistema de Informações Municipais)."
    ),
    version="0.2.0",
    api_base="https://api-dados-abertos.tce.ce.gov.br/sim",
    requires_auth=False,
    tags=["tce", "ce", "licitacoes", "contratos", "empenhos"],
)
