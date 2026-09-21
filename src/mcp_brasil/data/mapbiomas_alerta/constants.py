"""Constants for the MapBiomas Alerta feature.

Endpoint público: https://plataforma.alerta.mapbiomas.org/api/v2/graphql
O login (mutation ``signIn``) devolve um token Bearer usado nas consultas.
"""

# --- Endpoint e autenticação ---

GRAPHQL_URL = "https://plataforma.alerta.mapbiomas.org/api/v2/graphql"

AUTH_ENV_EMAIL = "MAPBIOMAS_EMAIL"
AUTH_ENV_PASSWORD = "MAPBIOMAS_PASSWORD"
AUTH_SIGNUP_URL = "https://plataforma.alerta.mapbiomas.org/sign-up"

# --- Paginação ---

DEFAULT_LIMIT = 20
MAX_LIMIT = 200
DEFAULT_PAGE = 1

# --- Enums da API (valores aceitos no servidor) ---

# Campo de data usado para filtrar o período (detectedAt é o padrão do servidor).
DATE_TYPES = ("DetectedAt", "PublishedAt")

# Campos de ordenação expostos pela API.
SORT_FIELDS = ("ALERT_CODE", "AREA_HA", "DETECTED_AT", "STATE", "CITY")
SORT_DIRECTIONS = ("ASC", "DESC")

# Categorias de território devolvidas por ``territoryOptions``.
TERRITORY_CATEGORIES: dict[str, str] = {
    "city": "Município",
    "state": "Estado",
    "conservation_unit": "Unidade de Conservação",
    "indigenous_land": "Terra Indígena",
    "quilombo": "Quilombo",
    "settlement": "Assentamento",
    "biosphere_reserve": "Reserva da Biosfera",
    "geopark": "Geoparque",
    "special_territory": "Território Especial",
    "priority_territories": "Municípios Prioritários",
    "refined_biome": "Bioma",
}

# --- Documentos GraphQL ---

SIGN_IN_MUTATION = """
mutation Entrar($email: String!, $password: String!) {
  signIn(email: $email, password: $password) {
    token
  }
}
"""

# Campos enxutos para listagem (evita payload grande em páginas cheias).
ALERT_LIST_FIELDS = """
    alertCode areaHa detectedAt publishedAt statusName
    sources deforestationClasses
    crossedBiomes crossedStates crossedCities
    crossedPermanentProtectedArea crossedLegalReservesArea
    crossedConservationUnits crossedIndigenousLands
    crossedEmbargoesTotal crossedDeforestationAuthorizationsArea
    ruralPropertiesCodes ruralPropertiesTotal
"""

# Campos completos para o dossiê de um alerta (todos os cruzamentos espaciais).
ALERT_DETAIL_FIELDS = (
    ALERT_LIST_FIELDS
    + """
    statusAt deforestationSpeed republished
    crossedBiomesArea crossedStatesArea crossedCitiesArea
    crossedPermanentProtectedAreaTotal
    crossedLegalReservesTotal
    crossedConservationUnitsArea crossedIndigenousLandsArea
    crossedQuilombos crossedQuilombosArea
    crossedSettlements crossedSettlementsArea
    crossedWatersheds crossedWatershedsArea crossedRiverSourcesArea
    crossedEmbargoesArea
    crossedDeforestationAuthorizationsCategories
    crossedDeforestationAuthorizationsActivities
    crossedForestManagementsArea crossedForestManagementsActivities
    imageAcquiredBeforeAt imageAcquiredAfterAt
    datum bbox geometryWkt
"""
)

BUSCAR_ALERTAS_QUERY = (
    """
query Alertas(
  $page: Int, $limit: Int, $territoryIds: [Int!],
  $startDate: BaseDate, $endDate: BaseDate,
  $startSize: Float, $endSize: Float,
  $isInEmbargoedArea: Boolean, $isInAuthorizedArea: Boolean,
  $intersectWithCar: Boolean,
  $sortField: AlertSortField, $sortDirection: SortDirection
) {
  alerts(
    page: $page, limit: $limit, territoryIds: $territoryIds,
    startDate: $startDate, endDate: $endDate,
    startSize: $startSize, endSize: $endSize,
    isInEmbargoedArea: $isInEmbargoedArea,
    isInAuthorizedArea: $isInAuthorizedArea,
    intersectWithCar: $intersectWithCar,
    sortField: $sortField, sortDirection: $sortDirection
  ) {
    metadata { currentPage limitValue totalCount totalPages }
    collection {"""
    + ALERT_LIST_FIELDS
    + """}
  }
}
"""
)

CONSULTAR_ALERTA_QUERY = (
    """
query Alerta($codigo: Int!) {
  alert(alertCode: $codigo) {"""
    + ALERT_DETAIL_FIELDS
    + """}
}
"""
)

BUSCAR_ALERTAS_POR_CAR_QUERY = (
    """
query ImovelRural($codigo: String) {
  ruralProperty(propertyCode: $codigo) {
    propertyCode propertyType stateAcronym areaHa carUpdatedAt version
    alerts {"""
    + ALERT_LIST_FIELDS
    + """}
  }
}
"""
)

LISTAR_TERRITORIOS_QUERY = """
{
  territoryOptions {
    category
    categoryName
    territories { code name }
  }
}
"""
