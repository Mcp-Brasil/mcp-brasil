"""Feature BrasilAPI — agregador de APIs públicas brasileiras."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="brasilapi",
    description=(
        "BrasilAPI: CEP, CNPJ, DDD, bancos, câmbio, feriados, taxas, "
        "FIPE, ISBN, NCM, PIX e Registro.br — dados públicos unificados."
    ),
    version="0.1.0",
    api_base="https://brasilapi.com.br/api",
    requires_auth=False,
    tags=["cep", "cnpj", "bancos", "cambio", "fipe", "feriados", "isbn", "ncm", "pix"],
)
