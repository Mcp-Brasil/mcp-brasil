"""HTTP client for the Portal Brasileiro de Dados Abertos API.

All communication with https://dados.gov.br goes through this module.
Requires API key via DADOS_GOV_BR_API_KEY environment variable.
"""

from __future__ import annotations

import logging
import os
from typing import Any

from mcp_brasil._shared.http_client import http_get
from mcp_brasil.exceptions import AuthError, HttpClientError

from .constants import (
    AUTH_ENV_VAR,
    AUTH_HEADER_NAME,
    CONJUNTOS_URL,
    ORGANIZACAO_URL,
    REUSO_URL,
    REUSOS_URL,
    TAGS_URL,
    TEMAS_URL,
)
from .schemas import (
    ODS,
    ConjuntoDetalhe,
    ConjuntoIndice,
    ConjuntoResultado,
    ObservanciaLegal,
    OrganizacaoDetalhe,
    OrganizacaoIndice,
    OrganizacaoResultado,
    Recurso,
    ReusoDetalhe,
    ReusoIndice,
    Tag,
    Tema,
)

logger = logging.getLogger(__name__)


# --- Auth ---


def _get_api_key() -> str:
    """Return the API key or raise AuthError."""
    key = os.environ.get(AUTH_ENV_VAR, "")
    if not key:
        raise AuthError(
            f"Variável de ambiente {AUTH_ENV_VAR} não configurada. "
            "Acesse dados.gov.br com conta Gov.br e gere o token em 'Minha Conta'."
        )
    return key


def _auth_headers() -> dict[str, str]:
    """Build auth headers for API requests."""
    return {AUTH_HEADER_NAME: _get_api_key()}


async def _get(url: str, params: dict[str, Any] | None = None) -> Any:
    """Make an authenticated GET request."""
    try:
        return await http_get(url, params=params, headers=_auth_headers())
    except HttpClientError as exc:
        msg = str(exc)
        if "401" in msg:
            logger.warning(
                "Acesso negado (HTTP 401) para %s — verifique se a chave API (%s) é válida.",
                url,
                AUTH_ENV_VAR,
            )
        raise


# --- Conjuntos de dados ---


def _parse_conjunto_indice(item: dict[str, Any]) -> ConjuntoIndice:
    """Parse a dataset index item from the API response."""
    return ConjuntoIndice.model_validate(item)


async def buscar_conjuntos(
    pagina: int = 1,
    nome: str | None = None,
    id_organizacao: str | None = None,
    dados_abertos: bool | None = None,
) -> ConjuntoResultado:
    """Search datasets in the portal catalog.

    Args:
        pagina: Page number (1-based).
        nome: Filter by dataset name.
        id_organizacao: Filter by organization ID.
        dados_abertos: Filter only open data.

    Returns:
        Paginated result with matching datasets.
    """
    params: dict[str, str] = {
        "pagina": str(pagina),
        "isPrivado": "false",
    }
    if nome:
        params["nomeConjuntoDados"] = nome
    if id_organizacao:
        params["idOrganizacao"] = id_organizacao
    if dados_abertos is not None:
        params["dadosAbertos"] = str(dados_abertos).lower()

    try:
        data: Any = await _get(CONJUNTOS_URL, params=params)
    except (HttpClientError, AuthError) as exc:
        logger.warning("dados.gov.br API error searching datasets: %s", exc)
        return ConjuntoResultado()

    if not data:
        return ConjuntoResultado()

    # API may return list directly or paginated wrapper
    items: list[Any]
    total: int
    if isinstance(data, list):
        items = data
        total = len(items)
    elif isinstance(data, dict):
        raw_items = data.get("registros", data.get("result", []))
        if isinstance(raw_items, list):
            items = raw_items
            raw_total = data.get("totalRegistros", data.get("count", len(items)))
            total = int(raw_total) if raw_total is not None else len(items)
        else:
            items = []
            total = 0
    else:
        return ConjuntoResultado()

    conjuntos = [_parse_conjunto_indice(i) for i in items if isinstance(i, dict)]
    return ConjuntoResultado(total=total, conjuntos=conjuntos)


async def detalhar_conjunto(conjunto_id: str) -> ConjuntoDetalhe | None:
    """Get full details of a dataset including its resources.

    Args:
        conjunto_id: Dataset ID.

    Returns:
        Dataset details or None if not found.
    """
    url = f"{CONJUNTOS_URL}/{conjunto_id}"
    try:
        data: Any = await _get(url)
    except (HttpClientError, AuthError) as exc:
        logger.warning("dados.gov.br API error for dataset '%s': %s", conjunto_id, exc)
        return None

    if not data or not isinstance(data, dict):
        return None

    # Parse recursos from embedded array
    recursos_raw = data.get("recursos", []) or []
    recursos = [Recurso.model_validate(r) for r in recursos_raw if isinstance(r, dict)]

    # Parse tags
    tags_raw = data.get("tags", []) or []
    tags = [Tag.model_validate(t) for t in tags_raw if isinstance(t, dict)]

    # Parse temas
    temas_raw = data.get("temas", []) or []
    temas: list[dict[str, str]] = []
    for t in temas_raw:
        if isinstance(t, dict):
            temas.append(t)
        elif isinstance(t, str):
            temas.append({"name": t, "title": t})

    return ConjuntoDetalhe(
        id=data.get("id"),
        titulo=data.get("titulo"),
        nome=data.get("nome"),
        descricao=data.get("descricao"),
        organizacao=data.get("organizacao"),
        licenca=data.get("licenca"),
        periodicidade=data.get("periodicidade"),
        responsavel=data.get("responsavel"),
        emailResponsavel=data.get("emailResponsavel"),
        versao=data.get("versao"),
        visibilidade=data.get("visibilidade"),
        coberturaEspacial=data.get("coberturaEspacial"),
        coberturaTemporalInicio=data.get("coberturaTemporalInicio"),
        coberturaTemporalFim=data.get("coberturaTemporalFim"),
        temas=temas,
        tags=tags,
        recursos=recursos,
        dadosAbertos=data.get("dadosAbertos"),
        descontinuado=data.get("descontinuado"),
        dataUltimaAtualizacaoMetadados=data.get("dataUltimaAtualizacaoMetadados"),
        dataUltimaAtualizacaoArquivo=data.get("dataUltimaAtualizacaoArquivo"),
        dataCatalogacao=data.get("dataCatalogacao"),
        observanciaLegal=data.get("observanciaLegal"),
        ods=data.get("ods", []) or [],
    )


async def listar_tags_conjunto(conjunto_id: str) -> list[Tag]:
    """List tags of a specific dataset.

    Args:
        conjunto_id: Dataset ID.

    Returns:
        List of tags.
    """
    url = f"{CONJUNTOS_URL}/{conjunto_id}/tag"
    try:
        data: Any = await _get(url)
    except (HttpClientError, AuthError) as exc:
        logger.warning("dados.gov.br API error for tags of '%s': %s", conjunto_id, exc)
        return []

    if not data or not isinstance(data, list):
        return []

    return [Tag.model_validate(t) for t in data if isinstance(t, dict)]


async def listar_formatos() -> list[str]:
    """List available resource formats in the portal.

    Returns:
        List of format strings.
    """
    url = f"{CONJUNTOS_URL}/formatos"
    try:
        data: Any = await _get(url)
    except (HttpClientError, AuthError) as exc:
        logger.warning("dados.gov.br API error listing formats: %s", exc)
        return []

    if isinstance(data, list):
        return [str(f) for f in data]
    return []


async def listar_observancia_legal() -> list[ObservanciaLegal]:
    """List legal compliance options.

    Returns:
        List of legal compliance options.
    """
    url = f"{CONJUNTOS_URL}/observancia-legal"
    try:
        data: Any = await _get(url)
    except (HttpClientError, AuthError) as exc:
        logger.warning("dados.gov.br API error listing legal compliance: %s", exc)
        return []

    if not isinstance(data, list):
        return []
    return [ObservanciaLegal.model_validate(item) for item in data if isinstance(item, dict)]


async def listar_ods() -> list[ODS]:
    """List Sustainable Development Goals (ODS) options.

    Returns:
        List of ODS options.
    """
    url = f"{CONJUNTOS_URL}/objetivos-desenvolvimento-sustentavel"
    try:
        data: Any = await _get(url)
    except (HttpClientError, AuthError) as exc:
        logger.warning("dados.gov.br API error listing ODS: %s", exc)
        return []

    if not isinstance(data, list):
        return []
    return [ODS.model_validate(item) for item in data if isinstance(item, dict)]


# --- Organizações ---


async def listar_organizacoes(
    pagina: int = 1,
    nome: str | None = None,
) -> OrganizacaoResultado:
    """List organizations that publish datasets.

    Args:
        pagina: Page number (1-based).
        nome: Filter by organization name.

    Returns:
        Paginated result with organizations.
    """
    params: dict[str, str] = {"pagina": str(pagina)}
    if nome:
        params["nome"] = nome

    try:
        data: Any = await _get(ORGANIZACAO_URL, params=params)
    except (HttpClientError, AuthError) as exc:
        logger.warning("dados.gov.br API error listing organizations: %s", exc)
        return OrganizacaoResultado()

    if not data:
        return OrganizacaoResultado()

    items: list[Any]
    total: int
    if isinstance(data, list):
        items = data
        total = len(items)
    elif isinstance(data, dict):
        raw_items = data.get("registros", data.get("result", []))
        if isinstance(raw_items, list):
            items = raw_items
            raw_total = data.get("totalRegistros", data.get("count", len(items)))
            total = int(raw_total) if raw_total is not None else len(items)
        else:
            items = []
            total = 0
    else:
        return OrganizacaoResultado()

    orgs = [OrganizacaoIndice.model_validate(i) for i in items if isinstance(i, dict)]
    return OrganizacaoResultado(total=total, organizacoes=orgs)


async def detalhar_organizacao(org_id: str) -> OrganizacaoDetalhe | None:
    """Get full details of an organization.

    Args:
        org_id: Organization ID.

    Returns:
        Organization details or None if not found.
    """
    url = f"{ORGANIZACAO_URL}/{org_id}"
    try:
        data: Any = await _get(url)
    except (HttpClientError, AuthError) as exc:
        logger.warning("dados.gov.br API error for org '%s': %s", org_id, exc)
        return None

    if not data or not isinstance(data, dict):
        return None
    return OrganizacaoDetalhe.model_validate(data)


# --- Temas ---


async def listar_temas() -> list[Tema]:
    """List all themes (thematic groups).

    Returns:
        List of themes.
    """
    try:
        data: Any = await _get(TEMAS_URL)
    except (HttpClientError, AuthError) as exc:
        logger.warning("dados.gov.br API error listing themes: %s", exc)
        return []

    if not isinstance(data, list):
        return []
    return [Tema.model_validate(t) for t in data if isinstance(t, dict)]


# --- Tags ---


async def buscar_tags(nome: str) -> list[Tag]:
    """Search tags by name.

    Args:
        nome: Tag name to search for.

    Returns:
        List of matching tags.
    """
    try:
        data: Any = await _get(TAGS_URL, params={"nome": nome})
    except (HttpClientError, AuthError) as exc:
        logger.warning("dados.gov.br API error searching tags '%s': %s", nome, exc)
        return []

    if not isinstance(data, list):
        return []
    return [Tag.model_validate(t) for t in data if isinstance(t, dict)]


# --- Reusos ---


async def listar_reusos(
    nome_reuso: str | None = None,
    nome_autor: str | None = None,
    id_organizacao: str | None = None,
) -> list[ReusoIndice]:
    """List data reuses.

    Args:
        nome_reuso: Filter by reuse name.
        nome_autor: Filter by author name.
        id_organizacao: Filter by organization ID.

    Returns:
        List of reuses.
    """
    params: dict[str, str] = {}
    if nome_reuso:
        params["nomeReuso"] = nome_reuso
    if nome_autor:
        params["nomeAutor"] = nome_autor
    if id_organizacao:
        params["idOrganizacao"] = id_organizacao

    try:
        data: Any = await _get(REUSOS_URL, params=params)
    except (HttpClientError, AuthError) as exc:
        logger.warning("dados.gov.br API error listing reuses: %s", exc)
        return []

    if isinstance(data, list):
        return [ReusoIndice.model_validate(r) for r in data if isinstance(r, dict)]
    if isinstance(data, dict):
        items = data.get("registros", data.get("result", []))
        if isinstance(items, list):
            return [ReusoIndice.model_validate(r) for r in items if isinstance(r, dict)]
    return []


async def detalhar_reuso(reuso_id: str) -> ReusoDetalhe | None:
    """Get full details of a data reuse.

    Args:
        reuso_id: Reuse ID.

    Returns:
        Reuse details or None if not found.
    """
    url = f"{REUSO_URL}/{reuso_id}"
    try:
        data: Any = await _get(url)
    except (HttpClientError, AuthError) as exc:
        logger.warning("dados.gov.br API error for reuse '%s': %s", reuso_id, exc)
        return None

    if not data or not isinstance(data, dict):
        return None
    return ReusoDetalhe.model_validate(data)
