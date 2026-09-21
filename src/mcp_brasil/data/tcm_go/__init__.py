"""Feature TCM-GO — Tribunal de Contas dos Municípios do Estado de Goiás."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="tcm_go",
    description=(
        "TCM-GO: processos eletrônicos (eSiproc) e acórdãos (TCMJuris) do "
        "Tribunal de Contas dos Municípios do Estado de Goiás, com busca "
        "full-text e link do PDF do julgado."
    ),
    version="0.1.0",
    api_base="https://virtual.tcmgo.tc.br/eSiproc/api/elastic",
    requires_auth=False,
    tags=["tcm", "go", "tribunal-de-contas", "processos", "acordaos", "jurisprudencia"],
)
