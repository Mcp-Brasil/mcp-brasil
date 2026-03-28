"""HTTP client for the DOU (Diário Oficial da União) via Imprensa Nacional.

Endpoints:
    - /consulta/-/buscar/dou  → buscar_dou (full-text search via HTML scraping)
    - /en/web/dou/-/{url}     → ler_publicacao_dou (article content)

Note: The search endpoint returns HTML with embedded JSON in a <script> tag.
Approach based on Ro-dou project (https://github.com/gestaogovbr/Ro-dou).
"""

from __future__ import annotations

import json
import logging
import re
from typing import Any

from mcp_brasil._shared.http_client import create_client, http_get
from mcp_brasil.exceptions import HttpClientError

from .constants import DOU_ARTICLE_URL, DOU_PERIODS, DOU_SEARCH_URL, DOU_SECTIONS
from .schemas import PublicacaoDOU, ResultadoDOU

logger = logging.getLogger(__name__)

# Script tag ID containing the search results JSON
_SCRIPT_TAG_ID = "_br_com_seatecnologia_in_buscadou_BuscaDouPortlet_params"

# Regex to extract JSON content from the script tag
_SCRIPT_RE = re.compile(
    rf'<script[^>]+id="{_SCRIPT_TAG_ID}"[^>]*>(.*?)</script>',
    re.DOTALL,
)

# Headers to mimic browser requests (in.gov.br blocks bare API calls)
_DOU_HEADERS = {
    "Accept": "text/html,application/xhtml+xml;q=0.9,*/*;q=0.8",
    "Referer": "https://www.in.gov.br/consulta",
    "Cache-Control": "no-cache",
}


def _to_dmy(date_str: str) -> str:
    """Convert YYYY-MM-DD → dd-mm-yyyy for the DOU API."""
    if "-" in date_str and len(date_str) == 10 and date_str[4] == "-":
        y, m, d = date_str.split("-")
        return f"{d}-{m}-{y}"
    return date_str


def _parse_publicacao(item: dict[str, Any]) -> PublicacaoDOU:
    """Parse a single DOU search result item into a PublicacaoDOU."""
    return PublicacaoDOU(
        titulo=item.get("title") or item.get("titulo"),
        resumo=item.get("abstract") or item.get("resumo"),
        url_titulo=item.get("urlTitle") or item.get("url_titulo"),
        orgao=item.get("pubName") or item.get("orgao"),
        tipo_publicacao=item.get("artType") or item.get("tipo_publicacao"),
        secao=item.get("pubType") or item.get("secao"),
        data_publicacao=item.get("pubDate") or item.get("data_publicacao"),
        edicao=item.get("numberPage") or item.get("edicao"),
        pagina=item.get("pageNumber") or item.get("pagina"),
        conteudo=item.get("content") or item.get("conteudo"),
        assinante=item.get("assina") or item.get("assinante"),
        cargo_assinante=item.get("cargo") or item.get("cargo_assinante"),
    )


def _extract_json_from_html(html: str) -> dict[str, Any]:
    """Extract search results JSON from the DOU HTML response.

    The in.gov.br search page embeds results in a <script> tag with a
    known ID. This function extracts and parses that JSON.
    """
    match = _SCRIPT_RE.search(html)
    if not match:
        logger.warning("DOU script tag '%s' not found in response", _SCRIPT_TAG_ID)
        return {}

    raw = match.group(1).strip()
    if not raw:
        return {}

    try:
        result: dict[str, Any] = json.loads(raw)
        return result
    except json.JSONDecodeError as exc:
        logger.warning("Failed to parse DOU embedded JSON: %s", exc)
        return {}


async def buscar_dou(
    termo: str,
    secao: str = "TODOS",
    periodo: str = "MES",
    data_inicio: str | None = None,
    data_fim: str | None = None,
    orgao: str | None = None,
    tipo_publicacao: str | None = None,
    campo: str = "TUDO",
    pagina: int = 0,
    tamanho: int = 20,
) -> ResultadoDOU:
    """Search DOU publications via Imprensa Nacional.

    The search endpoint returns HTML with embedded JSON.

    Args:
        termo: Search keyword.
        secao: DOU section (SECAO_1, SECAO_2, SECAO_3, EDICAO_EXTRA, TODOS).
        periodo: Time period (DIA, SEMANA, MES, ANO, PERSONALIZADO).
        data_inicio: Start date YYYY-MM-DD (for PERSONALIZADO period).
        data_fim: End date YYYY-MM-DD (for PERSONALIZADO period).
        orgao: Publishing body name filter.
        tipo_publicacao: Publication type (Decreto, Portaria, etc.).
        campo: Search field (TUDO, TITULO, CONTEUDO).
        pagina: Page number (0-indexed).
        tamanho: Results per page (default 20).
    """
    # Map section/period names to API codes
    secao_code = DOU_SECTIONS.get(secao, secao)
    periodo_code = DOU_PERIODS.get(periodo, periodo.lower())

    params: dict[str, str] = {
        "q": termo,
        "s": secao_code,
        "exactDate": periodo_code,
        "sortType": "0",
        "delta": str(tamanho),
        "currentPage": str(pagina),
    }

    if data_inicio and data_fim:
        params["exactDate"] = "personalizado"
        # API expects dd-mm-yyyy format
        params["publishFrom"] = _to_dmy(data_inicio)
        params["publishTo"] = _to_dmy(data_fim)

    if orgao:
        params["orgPrin"] = orgao

    if tipo_publicacao:
        params["artType"] = tipo_publicacao

    if campo != "TUDO":
        params["searchType"] = campo

    try:
        async with create_client(headers=_DOU_HEADERS) as client:
            response = await client.get(DOU_SEARCH_URL, params=params)
            response.raise_for_status()
            html = response.text
    except Exception as exc:
        logger.warning("DOU API error for '%s': %s", termo, exc)
        return ResultadoDOU(total=0, publicacoes=[])

    data = _extract_json_from_html(html)
    if not data:
        return ResultadoDOU(total=0, publicacoes=[])

    items = data.get("jsonArray", [])
    total = data.get("total", len(items))

    publicacoes = [_parse_publicacao(item) for item in items]
    return ResultadoDOU(total=total, publicacoes=publicacoes)


async def ler_publicacao_dou(url_titulo: str) -> PublicacaoDOU | None:
    """Read full content of a DOU publication by its urlTitle.

    Args:
        url_titulo: The urlTitle identifier from search results.
    """
    url = f"{DOU_ARTICLE_URL}/{url_titulo}"

    try:
        data: dict[str, Any] = await http_get(url, headers=_DOU_HEADERS)
    except HttpClientError as exc:
        logger.warning("DOU article error for '%s': %s", url_titulo, exc)
        return None

    if not data or not isinstance(data, dict):
        return None

    return _parse_publicacao(data)
