"""Feature Anuncios Eleitorais — Biblioteca de Anúncios da Meta."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="anuncios_eleitorais",
    description=(
        "Biblioteca de Anúncios da Meta — busca e análise de anúncios eleitorais "
        "e sobre temas sociais, eleições ou política veiculados no Brasil"
    ),
    version="0.1.0",
    api_base="https://graph.facebook.com/v21.0",
    requires_auth=True,
    auth_env_var="META_ACCESS_TOKEN",
    tags=["eleicoes", "politica", "anuncios", "meta", "facebook", "instagram", "transparencia"],
)
