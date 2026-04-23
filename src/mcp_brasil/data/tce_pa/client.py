"""HTTP client for the TCE-PA Dados Abertos API and Pesquisa Integrada.

Endpoints:
    GET /v1/diario_oficial → buscar_diario_oficial
    GET /pesquisaintegrada/pesquisa/resultados → buscar_sessoes_plenarias
    GET /pesquisaintegrada/pesquisa/resultados → buscar_pesquisa_integrada
"""

from __future__ import annotations

import contextlib
import re
from typing import Any

from bs4 import BeautifulSoup

from mcp_brasil._shared.http_client import create_client, http_get
from mcp_brasil.exceptions import HttpClientError

from .constants import (
    BASES_PDF_DOWNLOAD,
    DIARIO_OFICIAL_URL,
    SESSOES_DEFAULT_RPP,
    SESSOES_SEARCH_URL,
    TIPO_SESSAO,
)
from .schemas import DiarioOficial, ResultadoPesquisa, SessaoPlenaria


def _build_query(query: str, ano: int | None, mes: int | None) -> str:
    parts = [query] if query else []
    if ano:
        parts.append(f"ano:{ano}")
    if mes:
        parts.append(f"mes:{mes:02d}")
    return " ".join(parts)


def _strip_html(text: str) -> str:
    """Remove HTML tags from a string, preserving whitespace."""
    return re.sub(r"<[^>]+>", " ", text).strip()


async def buscar_diario_oficial(
    *,
    ano: int = 2018,
    mes: int | None = None,
    numero_publicacao: int | None = None,
    tipo_ato: str | None = None,
) -> list[DiarioOficial]:
    """Search TCE-PA Diário Oficial publications.

    Args:
        ano: Year to search (default: 2018, minimum: 2018).
        mes: Month (1-12, optional).
        numero_publicacao: Specific publication number (optional).
        tipo_ato: Filter by act type (optional). Valid values:
            "Atos de Pessoal para Fins de Registro",
            "Atos e Normas", "Contratos",
            "Convênios e Instrumentos Congêneres",
            "Licitações", "Outros Atos de Pessoal".

    Returns:
        List of Diário Oficial publications.
    """
    params: dict[str, str] = {"ano": str(ano)}
    if mes is not None:
        params["mes"] = str(mes)
    if numero_publicacao is not None:
        params["numero_publicacao"] = str(numero_publicacao)
    if tipo_ato:
        params["tipo_ato"] = tipo_ato

    data: dict[str, Any] = await http_get(DIARIO_OFICIAL_URL, params=params)

    items = data.get("data", []) if isinstance(data, dict) else data
    return [
        DiarioOficial(
            numero_publicacao=item.get("NumeroPublicacao"),
            data_publicacao=item.get("DataPublicacao", ""),
            tipo_ato=item.get("TipoAto", ""),
            publicacao=_strip_html(item.get("Publicacao", "")),
        )
        for item in items
    ]


async def buscar_sessoes_plenarias(
    *,
    tipo: str = "sessoes",
    query: str = "",
    ano: int | None = None,
    mes: int | None = None,
    pagina: int = 1,
    rpp: int = SESSOES_DEFAULT_RPP,
) -> list[SessaoPlenaria]:
    """Scrape TCE-PA Pesquisa Integrada for plenary sessions.

    Args:
        tipo: Database type — "sessoes", "pautas", "atas", or "videos".
        query: Free-text search query (optional).
        ano: Year filter (optional).
        mes: Month filter 1-12 (optional, approximate text search).
        pagina: Page number (default: 1).
        rpp: Results per page (default: 20).

    Returns:
        List of SessaoPlenaria records.
    """
    slug = TIPO_SESSAO[tipo]
    q = _build_query(query, ano, mes)

    params: dict[str, str] = {
        "b": slug,
        "q": q,
        "p": str(pagina),
        "rpp": str(rpp),
        "o": "data",
        "or": "True",
    }

    async with create_client(headers={"Accept": "text/html,application/xhtml+xml"}) as http:
        try:
            resp = await http.get(SESSOES_SEARCH_URL, params=params)
            resp.raise_for_status()
        except Exception as exc:
            raise HttpClientError(f"TCE-PA sessões plenárias failed: {exc}") from exc

    return _parse_sessoes_html(resp.text, tipo=tipo)


def _parse_sessoes_html(html: str, *, tipo: str) -> list[SessaoPlenaria]:
    soup = BeautifulSoup(html, "lxml")
    results: list[SessaoPlenaria] = []

    for card in soup.select("div.resultado-organico"):
        title_a = card.select_one("h2.resultado-organico-titulo a.titulo")
        if not title_a:
            continue

        titulo = title_a.get_text(strip=True)
        raw_href = str(title_a.get("href") or "")
        # Strip query params and /conteudo-original suffix
        clean_href = raw_href.split("?")[0].removesuffix("/conteudo-original")

        m = re.search(r"/codigo/(\d+)/", clean_href)
        codigo = int(m.group(1)) if m else None

        # Extract named campos
        tipo_sessao: str | None = None
        data_sessao: str | None = None
        ano_val: int | None = None

        for campo in card.select("div.resultado-organico-campo"):
            nome_el = campo.select_one("dfn.resultado-organico-campo-nome")
            valor_el = campo.select_one("span.resultado-organico-campo-valor")
            if not nome_el or not valor_el:
                continue
            nome = nome_el.get_text(strip=True).lower()
            valor = valor_el.get_text(strip=True)
            if "tipo" in nome:
                tipo_sessao = valor
            elif "data" in nome:
                data_sessao = valor
            elif "ano" in nome:
                with contextlib.suppress(ValueError):
                    ano_val = int(valor)

        url_documento = _build_doc_url(clean_href, tipo)

        results.append(
            SessaoPlenaria(
                codigo=codigo,
                titulo=titulo,
                data_sessao=data_sessao,
                tipo_sessao=tipo_sessao,
                ano=ano_val,
                url_pagina=clean_href,
                url_documento=url_documento,
            )
        )

    return results


def _build_doc_url(base_url: str, tipo: str) -> str | None:
    if tipo in ("pautas", "atas"):
        return f"{base_url}/download"
    if tipo == "videos":
        return f"{base_url}/conteudo-original"
    return None


async def buscar_pesquisa_integrada(
    *,
    slug: str,
    query: str = "",
    ano: int | None = None,
    mes: int | None = None,
    pagina: int = 1,
    rpp: int = SESSOES_DEFAULT_RPP,
) -> list[ResultadoPesquisa]:
    """Scrape TCE-PA Pesquisa Integrada for any database.

    Args:
        slug: Exact database slug (e.g. "acordaos", "resolucoes").
        query: Free-text search query (optional).
        ano: Year filter appended to query (optional).
        mes: Month filter 1-12 (optional, approximate text search).
        pagina: Page number (default: 1).
        rpp: Results per page (default: 20).

    Returns:
        List of ResultadoPesquisa records.
    """
    q = _build_query(query, ano, mes)

    params: dict[str, str] = {
        "b": slug,
        "q": q,
        "p": str(pagina),
        "rpp": str(rpp),
        "o": "data",
        "or": "True",
    }

    async with create_client(headers={"Accept": "text/html,application/xhtml+xml"}) as http:
        try:
            resp = await http.get(SESSOES_SEARCH_URL, params=params)
            resp.raise_for_status()
        except Exception as exc:
            raise HttpClientError(f"TCE-PA pesquisa integrada ({slug}) failed: {exc}") from exc

    return _parse_resultados_html(resp.text, slug=slug)


def _parse_resultados_html(html: str, *, slug: str) -> list[ResultadoPesquisa]:
    soup = BeautifulSoup(html, "lxml")
    results: list[ResultadoPesquisa] = []

    for card in soup.select("div.resultado-organico"):
        title_a = card.select_one("h2.resultado-organico-titulo a.titulo")
        if not title_a:
            continue

        titulo = title_a.get_text(strip=True)
        raw_href = str(title_a.get("href") or "")
        clean_href = raw_href.split("?")[0].removesuffix("/conteudo-original")

        campos: dict[str, str] = {}
        for campo in card.select("div.resultado-organico-campo"):
            nome_el = campo.select_one("dfn.resultado-organico-campo-nome")
            valor_el = campo.select_one("span.resultado-organico-campo-valor")
            if nome_el and valor_el:
                nome = nome_el.get_text(strip=True).rstrip(":")
                valor = valor_el.get_text(strip=True)
                if nome and valor and nome.lower() != "fonte":
                    campos[nome] = valor[:200]

        url_documento = (
            f"{clean_href}/download"
            if slug in BASES_PDF_DOWNLOAD
            else f"{clean_href}/conteudo-original"
        )

        results.append(
            ResultadoPesquisa(
                titulo=titulo,
                url_pagina=clean_href,
                url_documento=url_documento,
                campos=campos,
            )
        )

    return results
