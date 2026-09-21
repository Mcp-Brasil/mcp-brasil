"""HTTP client for the MapBiomas Alerta GraphQL API (v2).

Fluxo:
    1. ``signIn`` (e-mail + senha) devolve um token Bearer.
    2. As consultas enviam ``Authorization: Bearer <token>``.

Credenciais vêm de ``MAPBIOMAS_EMAIL`` e ``MAPBIOMAS_PASSWORD``. Sem elas, o
client levanta ``AuthError`` com instrução de configuração. O token é mantido
em cache em memória e renovado automaticamente numa recusa de credencial.

Documentação: https://plataforma.alerta.mapbiomas.org
"""

from __future__ import annotations

import asyncio
import logging
import os
from typing import Any

from mcp_brasil._shared.http_client import http_post
from mcp_brasil.exceptions import AuthError, HttpClientError

from .constants import (
    AUTH_ENV_EMAIL,
    AUTH_ENV_PASSWORD,
    AUTH_SIGNUP_URL,
    BUSCAR_ALERTAS_POR_CAR_QUERY,
    BUSCAR_ALERTAS_QUERY,
    CONSULTAR_ALERTA_QUERY,
    DEFAULT_LIMIT,
    DEFAULT_PAGE,
    GRAPHQL_URL,
    LISTAR_TERRITORIOS_QUERY,
    MAX_LIMIT,
    SIGN_IN_MUTATION,
)
from .schemas import Alerta, ImovelRural, ResultadoAlertas, Territorio

logger = logging.getLogger(__name__)

# Cache de token (por processo). Reiniciado por ``_reset_token`` nos testes.
_token: str | None = None
_token_lock = asyncio.Lock()


def _reset_token() -> None:
    """Descarta o token em cache (útil para testes e renovação)."""
    global _token
    _token = None


def _get_credentials() -> tuple[str, str]:
    """Lê as credenciais do ambiente ou levanta AuthError."""
    email = os.environ.get(AUTH_ENV_EMAIL, "")
    password = os.environ.get(AUTH_ENV_PASSWORD, "")
    if not email or not password:
        raise AuthError(
            f"Credenciais do MapBiomas Alerta não configuradas. Defina "
            f"{AUTH_ENV_EMAIL} e {AUTH_ENV_PASSWORD}. Cadastro gratuito em "
            f"{AUTH_SIGNUP_URL}."
        )
    return email, password


def _graphql_errors(data: Any) -> str:
    """Extrai a mensagem de erro de um payload GraphQL (HTTP 200)."""
    if not isinstance(data, dict):
        return ""
    errors = data.get("errors")
    if not isinstance(errors, list):
        return ""
    partes = [str(e.get("message")) for e in errors if isinstance(e, dict) and e.get("message")]
    return "; ".join(partes)


def _is_auth_error(message: str) -> bool:
    """Heurística para erro de credencial/token na mensagem do servidor."""
    texto = message.lower()
    return any(
        termo in texto
        for termo in ("senha", "usuário", "usuario", "token", "acesso", "credencial")
    )


async def _execute(query: str, variables: dict[str, Any], *, token: str | None) -> dict[str, Any]:
    """Executa uma operação GraphQL e devolve o bloco ``data``.

    Erros GraphQL chegam com HTTP 200: viram ``AuthError`` quando a mensagem
    indica problema de credencial e ``HttpClientError`` nos demais casos.
    """
    headers: dict[str, str] = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"

    payload: Any = await http_post(
        GRAPHQL_URL,
        json_body={"query": query, "variables": variables},
        headers=headers,
        timeout=60.0,
    )
    if not isinstance(payload, dict):
        raise HttpClientError("Resposta inesperada do MapBiomas Alerta.")

    message = _graphql_errors(payload)
    if message:
        if _is_auth_error(message):
            raise AuthError(f"MapBiomas Alerta recusou a credencial: {message}")
        raise HttpClientError(f"MapBiomas Alerta recusou a consulta: {message}")

    data = payload.get("data")
    return data if isinstance(data, dict) else {}


async def _get_token(*, force: bool = False) -> str:
    """Obtém (e memoriza) um token Bearer pela mutation ``signIn``."""
    global _token
    if _token and not force:
        return _token

    async with _token_lock:
        if _token and not force:
            return _token
        email, password = _get_credentials()
        data = await _execute(SIGN_IN_MUTATION, {"email": email, "password": password}, token=None)
        login = data.get("signIn")
        token = login.get("token") if isinstance(login, dict) else None
        if not isinstance(token, str) or not token:
            raise AuthError(
                "Login no MapBiomas Alerta não devolveu token. "
                f"Confira {AUTH_ENV_EMAIL} e {AUTH_ENV_PASSWORD}."
            )
        _token = token
        return token


async def _query(query: str, variables: dict[str, Any]) -> dict[str, Any]:
    """Executa uma consulta autenticada, renovando o token uma vez se recusado."""
    token = await _get_token()
    try:
        return await _execute(query, variables, token=token)
    except AuthError:
        _reset_token()
        token = await _get_token(force=True)
        return await _execute(query, variables, token=token)


# --- Conversão de valores brutos ---


def _text(value: Any) -> str | None:
    if value is None:
        return None
    if isinstance(value, str):
        return value or None
    return str(value)


def _number(value: Any) -> float | None:
    if isinstance(value, bool):
        return None
    if isinstance(value, (int, float)):
        return float(value)
    return None


def _integer(value: Any) -> int | None:
    if isinstance(value, bool):
        return None
    if isinstance(value, int):
        return value
    if isinstance(value, float):
        return int(value)
    return None


def _boolean(value: Any) -> bool | None:
    return value if isinstance(value, bool) else None


def _list(value: Any) -> list[str]:
    if isinstance(value, list):
        return [str(v) for v in value if v is not None]
    return []


def _parse_alerta(item: dict[str, Any]) -> Alerta:
    """Converte um item GraphQL (camelCase) no modelo ``Alerta``."""
    return Alerta(
        codigo=_integer(item.get("alertCode")),
        area_ha=_number(item.get("areaHa")),
        detectado_em=_text(item.get("detectedAt")),
        publicado_em=_text(item.get("publishedAt")),
        status=_text(item.get("statusName")),
        status_em=_text(item.get("statusAt")),
        fontes=_list(item.get("sources")),
        classes_desmatamento=_list(item.get("deforestationClasses")),
        velocidade_desmatamento=_text(item.get("deforestationSpeed")),
        republicado=_boolean(item.get("republished")),
        biomas=_list(item.get("crossedBiomes")),
        biomas_area_ha=_number(item.get("crossedBiomesArea")),
        estados=_list(item.get("crossedStates")),
        estados_area_ha=_number(item.get("crossedStatesArea")),
        municipios=_list(item.get("crossedCities")),
        municipios_area_ha=_number(item.get("crossedCitiesArea")),
        app_area_ha=_number(item.get("crossedPermanentProtectedArea")),
        app_sobreposicoes=_integer(item.get("crossedPermanentProtectedAreaTotal")),
        reserva_legal_area_ha=_number(item.get("crossedLegalReservesArea")),
        reserva_legal_sobreposicoes=_integer(item.get("crossedLegalReservesTotal")),
        unidades_conservacao=_list(item.get("crossedConservationUnits")),
        unidades_conservacao_area_ha=_number(item.get("crossedConservationUnitsArea")),
        terras_indigenas=_list(item.get("crossedIndigenousLands")),
        terras_indigenas_area_ha=_number(item.get("crossedIndigenousLandsArea")),
        quilombos=_list(item.get("crossedQuilombos")),
        quilombos_area_ha=_number(item.get("crossedQuilombosArea")),
        assentamentos=_list(item.get("crossedSettlements")),
        assentamentos_area_ha=_number(item.get("crossedSettlementsArea")),
        bacias=_list(item.get("crossedWatersheds")),
        bacias_area_ha=_number(item.get("crossedWatershedsArea")),
        nascentes_area_ha=_number(item.get("crossedRiverSourcesArea")),
        embargos_total=_integer(item.get("crossedEmbargoesTotal")),
        embargos_area_ha=_number(item.get("crossedEmbargoesArea")),
        autorizacoes_area_ha=_number(item.get("crossedDeforestationAuthorizationsArea")),
        autorizacoes_categorias=_list(item.get("crossedDeforestationAuthorizationsCategories")),
        autorizacoes_atividades=_list(item.get("crossedDeforestationAuthorizationsActivities")),
        manejo_florestal_area_ha=_number(item.get("crossedForestManagementsArea")),
        manejo_florestal_atividades=_list(item.get("crossedForestManagementsActivities")),
        car_codigos=_list(item.get("ruralPropertiesCodes")),
        car_total=_integer(item.get("ruralPropertiesTotal")),
        imagem_antes_em=_text(item.get("imageAcquiredBeforeAt")),
        imagem_depois_em=_text(item.get("imageAcquiredAfterAt")),
        datum=_text(item.get("datum")),
        bbox=_text(item.get("bbox")),
        geometria_wkt=_text(item.get("geometryWkt")),
    )


# --- API pública ---


async def listar_territorios(
    *,
    categoria: str | None = None,
    busca: str | None = None,
    limite: int = 50,
) -> list[Territorio]:
    """Lista territórios e seus códigos (município, estado, UC, TI etc.).

    Consulta pública (não exige login). O filtro ``busca`` é aplicado no
    client, sem diferenciar acentos e maiúsculas.
    """
    data = await _execute(LISTAR_TERRITORIOS_QUERY, {}, token=None)
    options = data.get("territoryOptions")
    if not isinstance(options, list):
        return []

    termo = _normalizar(busca) if busca else ""
    encontrados: list[Territorio] = []
    for option in options:
        if not isinstance(option, dict):
            continue
        cat = _text(option.get("category")) or ""
        cat_nome = _text(option.get("categoryName")) or cat
        if categoria and cat != categoria:
            continue
        territorios = option.get("territories")
        if not isinstance(territorios, list):
            continue
        for t in territorios:
            if not isinstance(t, dict):
                continue
            nome = _text(t.get("name")) or ""
            if termo and termo not in _normalizar(nome):
                continue
            codigo = _integer(t.get("code"))
            if codigo is None:
                continue
            encontrados.append(
                Territorio(codigo=codigo, nome=nome, categoria=cat, categoria_nome=cat_nome)
            )
    return encontrados[:limite]


def _normalizar(texto: str) -> str:
    """Compara nomes sem acento e sem caixa (aproximação, não exaustiva)."""
    mapa = str.maketrans(
        "áàâãäéèêëíìîïóòôõöúùûüçÁÀÂÃÄÉÈÊËÍÌÎÏÓÒÔÕÖÚÙÛÜÇ",
        "aaaaaeeeeiiiiooooouuuucAAAAAEEEEIIIIOOOOOUUUUC",
    )
    return texto.translate(mapa).strip().lower()


async def buscar_alertas(
    *,
    territorio_id: int | None = None,
    de: str | None = None,
    ate: str | None = None,
    min_ha: float | None = None,
    max_ha: float | None = None,
    em_area_embargada: bool | None = None,
    com_autorizacao: bool | None = None,
    intersecta_car: bool | None = None,
    pagina: int = DEFAULT_PAGE,
    limite: int = DEFAULT_LIMIT,
    ordenar_por: str = "AREA_HA",
    direcao: str = "DESC",
) -> ResultadoAlertas:
    """Busca alertas por território, período e área.

    ``territorio_id`` é o código devolvido por ``listar_territorios``. O
    filtro de territórios é uma **união (OR)**: um único id por consulta.
    ``de``/``ate`` filtram pela data de detecção (AAAA-MM-DD).
    """
    variaveis: dict[str, Any] = {
        "page": max(1, pagina),
        "limit": max(1, min(limite, MAX_LIMIT)),
        "sortField": ordenar_por,
        "sortDirection": direcao,
    }
    if territorio_id is not None:
        variaveis["territoryIds"] = [territorio_id]
    if de:
        variaveis["startDate"] = de
    if ate:
        variaveis["endDate"] = ate
    if min_ha is not None:
        variaveis["startSize"] = min_ha
    if max_ha is not None:
        variaveis["endSize"] = max_ha
    if em_area_embargada is not None:
        variaveis["isInEmbargoedArea"] = em_area_embargada
    if com_autorizacao is not None:
        variaveis["isInAuthorizedArea"] = com_autorizacao
    if intersecta_car is not None:
        variaveis["intersectWithCar"] = intersecta_car

    data = await _query(BUSCAR_ALERTAS_QUERY, variaveis)
    bloco = data.get("alerts")
    if not isinstance(bloco, dict):
        return ResultadoAlertas(pagina=pagina)

    metadata = bloco.get("metadata")
    total = 0
    total_paginas = 0
    pagina_atual = max(1, pagina)
    if isinstance(metadata, dict):
        total = _integer(metadata.get("totalCount")) or 0
        total_paginas = _integer(metadata.get("totalPages")) or 0
        pagina_atual = _integer(metadata.get("currentPage")) or pagina_atual

    colecao = bloco.get("collection")
    alertas = [
        _parse_alerta(item)
        for item in (colecao if isinstance(colecao, list) else [])
        if isinstance(item, dict)
    ]
    return ResultadoAlertas(
        alertas=alertas, total=total, total_paginas=total_paginas, pagina=pagina_atual
    )


async def consultar_alerta(codigo: int) -> Alerta | None:
    """Dossiê completo de um alerta pelo código (inclui a geometria WKT, se houver)."""
    data = await _query(CONSULTAR_ALERTA_QUERY, {"codigo": codigo})
    item = data.get("alert")
    if not isinstance(item, dict):
        return None
    return _parse_alerta(item)


async def buscar_alertas_por_car(codigo_car: str) -> ImovelRural | None:
    """Lista os alertas associados a um imóvel rural, pelo código do CAR."""
    data = await _query(BUSCAR_ALERTAS_POR_CAR_QUERY, {"codigo": codigo_car})
    item = data.get("ruralProperty")
    if not isinstance(item, dict):
        return None

    colecao = item.get("alerts")
    alertas = [
        _parse_alerta(a)
        for a in (colecao if isinstance(colecao, list) else [])
        if isinstance(a, dict)
    ]
    return ImovelRural(
        codigo=_text(item.get("propertyCode")),
        tipo=_text(item.get("propertyType")),
        uf=_text(item.get("stateAcronym")),
        area_ha=_number(item.get("areaHa")),
        car_atualizado_em=_text(item.get("carUpdatedAt")),
        versao=_integer(item.get("version")),
        alertas=alertas,
        area_alertada_ha=round(sum(a.area_ha or 0.0 for a in alertas), 4),
    )
