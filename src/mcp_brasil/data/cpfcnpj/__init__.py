"""Feature CPF.CNPJ: cadastro de CNPJ (cpfcnpj.com.br, terceiro privado)."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="cpfcnpj",
    description=(
        "CPF.CNPJ: cadastro de CNPJ com QSA, situação cadastral e inscrições "
        "estaduais (terceiro privado, opcional)"
    ),
    version="0.1.0",
    api_base="https://api.cpfcnpj.com.br",
    requires_auth=True,
    auth_env_var="CPFCNPJ_TOKEN",
    tags=["cnpj", "empresa", "cadastro", "qsa", "receita", "inscricao-estadual"],
)
