"""Feature MapBiomas Alerta — alertas de supressão de vegetação nativa.

Fonte: MapBiomas Alerta (https://plataforma.alerta.mapbiomas.org), API GraphQL v2.
Exige conta própria (login por e-mail/senha) para consultar alertas.
"""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="mapbiomas_alerta",
    description=(
        "MapBiomas Alerta: alertas validados de supressão de vegetação nativa — "
        "área em hectares, data de detecção, sobreposição com APP, Reserva Legal, "
        "Unidade de Conservação e Terra Indígena, embargos, autorizações e CAR cruzado"
    ),
    version="0.1.0",
    api_base="https://plataforma.alerta.mapbiomas.org/api/v2/graphql",
    requires_auth=True,
    auth_env_var="MAPBIOMAS_EMAIL",
    tags=["meio-ambiente", "desmatamento", "vegetacao", "alertas", "car", "biomas"],
)
