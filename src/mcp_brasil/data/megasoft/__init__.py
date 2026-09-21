"""Feature Megasoft — portais da transparência municipais.

Megasoft é a plataforma que hospeda o Portal da Transparência de diversos
municípios brasileiros em ``{municipio}.megasofttransparencia.com.br``. A API
pública expõe empenhos, ordens de pagamento, totais financeiros e receitas.
O token de acesso é publicado pelo próprio portal, sem exigir cadastro.
"""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="megasoft",
    description=(
        "Portais da Transparência municipais (Megasoft): empenhos, ordens de "
        "pagamento, totais financeiros e receitas por exercício."
    ),
    version="0.1.0",
    api_base="https://{municipio}.megasofttransparencia.com.br",
    requires_auth=False,
    auth_env_var="MEGASOFT_TOKEN",
    tags=["transparencia", "municipal", "empenhos", "pagamentos", "receitas", "fiscal"],
)
