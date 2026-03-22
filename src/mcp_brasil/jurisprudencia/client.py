"""HTTP client for Jurisprudência (STF, STJ, TST).

Uses REST APIs where available. Each court has its own search mechanism.

Note: These APIs are reverse-engineered from the courts' web interfaces.
Some may return HTML or have undocumented rate limits.
"""

from __future__ import annotations

import logging
import re
from typing import Any

from mcp_brasil._shared.http_client import http_get
from mcp_brasil._shared.rate_limiter import RateLimiter

from .constants import (
    DEFAULT_PAGE_SIZE,
    STF_API_BASE,
    STF_SEARCH_PARAMS,
    STJ_API_BASE,
    TST_API_BASE,
)
from .schemas import Jurisprudencia, RepercussaoGeral, Sumula

logger = logging.getLogger(__name__)

_rate_limiter = RateLimiter(max_requests=20, period=60.0)

_HEADERS = {
    "Accept": "application/json",
    "User-Agent": (
        "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 "
        "(KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36"
    ),
}


async def _get(url: str, params: dict[str, Any] | None = None) -> Any:
    """GET request with rate limiting and browser-like headers."""
    async with _rate_limiter:
        return await http_get(url, params=params, headers=_HEADERS)


# === STF ===


def _parse_stf_resultado(raw: dict[str, Any]) -> Jurisprudencia:
    """Parse an STF search result."""
    return Jurisprudencia(
        tribunal="STF",
        ementa=raw.get("ementa") or raw.get("resumo"),
        relator=raw.get("ministro") or raw.get("relator"),
        numero_processo=raw.get("incidente") or raw.get("processo"),
        classe=raw.get("classe"),
        data_julgamento=raw.get("dataJulgamento") or raw.get("dtJulgamento"),
        data_publicacao=raw.get("dataPublicacao") or raw.get("dtPublicacao"),
        orgao_julgador=raw.get("orgaoJulgador") or raw.get("colegiado"),
        decisao=raw.get("decisao"),
        url=raw.get("link"),
    )


async def buscar_stf(
    query: str,
    pagina: int = 1,
    tamanho: int = DEFAULT_PAGE_SIZE,
) -> list[Jurisprudencia]:
    """Search STF jurisprudence (acórdãos)."""
    params = {
        **STF_SEARCH_PARAMS,
        "queryString": query,
        "page": pagina,
        "pageSize": tamanho,
    }
    try:
        data = await _get(STF_API_BASE, params=params)
    except Exception:
        logger.warning("STF API retornou erro para query: %s", query)
        return []

    if isinstance(data, dict):
        resultados = data.get("result", [])
        if isinstance(resultados, list):
            return [_parse_stf_resultado(r) for r in resultados if isinstance(r, dict)]
    return []


# === STJ ===


def _parse_stj_resultado(raw: dict[str, Any]) -> Jurisprudencia:
    """Parse an STJ search result."""
    return Jurisprudencia(
        tribunal="STJ",
        ementa=raw.get("ementa"),
        relator=raw.get("relator") or raw.get("ministro"),
        numero_processo=raw.get("processo") or raw.get("registroAcordao"),
        classe=raw.get("classe") or raw.get("siglaClasse"),
        data_julgamento=raw.get("dtJulgamento") or raw.get("dataJulgamento"),
        data_publicacao=raw.get("dtPublicacao") or raw.get("dataPublicacao"),
        orgao_julgador=raw.get("orgaoJulgador"),
        decisao=raw.get("decisao"),
        url=raw.get("link"),
    )


async def buscar_stj(
    query: str,
    pagina: int = 1,
    tamanho: int = DEFAULT_PAGE_SIZE,
) -> list[Jurisprudencia]:
    """Search STJ jurisprudence (acórdãos via SCON)."""
    params = {
        **{
            k: v
            for k, v in {
                "livre": query,
                "b": "ACOR",
                "p": "true",
                "tp": "T",
                "l": str(tamanho),
                "i": str((pagina - 1) * tamanho + 1),
                "tipo_visualizacao": "RESUMO",
            }.items()
        },
    }
    try:
        data = await _get(STJ_API_BASE, params=params)
    except Exception:
        logger.warning("STJ API retornou erro para query: %s", query)
        return []

    if isinstance(data, dict):
        documentos = data.get("documentos", [])
        if isinstance(documentos, list):
            return [_parse_stj_resultado(d) for d in documentos if isinstance(d, dict)]
    return []


# === TST ===


def _parse_tst_resultado(raw: dict[str, Any]) -> Jurisprudencia:
    """Parse a TST search result."""
    return Jurisprudencia(
        tribunal="TST",
        ementa=raw.get("ementa") or raw.get("ementaHtml"),
        relator=raw.get("relator") or raw.get("ministroRelator"),
        numero_processo=raw.get("processo") or raw.get("numeroProcesso"),
        classe=raw.get("classe") or raw.get("siglaClasse"),
        data_julgamento=raw.get("dtJulgamento") or raw.get("dataJulgamento"),
        data_publicacao=raw.get("dtPublicacao") or raw.get("dataPublicacao"),
        orgao_julgador=raw.get("orgaoJulgador"),
        decisao=raw.get("decisao"),
        url=raw.get("link"),
    )


async def buscar_tst(
    query: str,
    pagina: int = 1,
    tamanho: int = DEFAULT_PAGE_SIZE,
) -> list[Jurisprudencia]:
    """Search TST jurisprudence."""
    params = {
        "ementa": query,
        "pagina": pagina,
        "tamanho": tamanho,
    }
    try:
        data = await _get(TST_API_BASE, params=params)
    except Exception:
        logger.warning("TST API retornou erro para query: %s", query)
        return []

    if isinstance(data, dict):
        documentos = data.get("documentos") or data.get("items", [])
        if isinstance(documentos, list):
            return [_parse_tst_resultado(d) for d in documentos if isinstance(d, dict)]
    return []


# === Súmulas ===


def _parse_sumula(raw: dict[str, Any], tribunal: str) -> Sumula:
    """Parse a súmula result."""
    numero = raw.get("numero") or raw.get("numeroSumula")
    if isinstance(numero, str):
        try:
            numero = int(re.sub(r"\D", "", numero))
        except ValueError:
            numero = None

    return Sumula(
        tribunal=tribunal.upper(),
        numero=numero,
        enunciado=raw.get("enunciado") or raw.get("texto") or raw.get("descricao"),
        referencia_legislativa=raw.get("referenciaLegislativa"),
        situacao=raw.get("situacao"),
        data_aprovacao=raw.get("dataAprovacao"),
        vinculante=raw.get("vinculante") or ("vinculante" in str(raw).lower()),
    )


async def buscar_sumulas_stf(query: str | None = None) -> list[Sumula]:
    """Search STF súmulas (including vinculantes)."""
    params: dict[str, Any] = {
        "base": "sumulas",
        "pesquisa_inteiro_teor": "false",
        "sinonimo": "true",
        "plural": "true",
    }
    if query:
        params["queryString"] = query
    try:
        data = await _get(STF_API_BASE, params=params)
    except Exception:
        logger.warning("STF Súmulas API retornou erro")
        return []

    if isinstance(data, dict):
        resultados = data.get("result", [])
        if isinstance(resultados, list):
            return [_parse_sumula(r, "STF") for r in resultados if isinstance(r, dict)]
    return []


# === Repercussão Geral ===


def _parse_repercussao(raw: dict[str, Any]) -> RepercussaoGeral:
    """Parse a repercussão geral theme."""
    numero = raw.get("numeroTema") or raw.get("numero")
    if isinstance(numero, str):
        try:
            numero = int(re.sub(r"\D", "", numero))
        except ValueError:
            numero = None

    return RepercussaoGeral(
        numero_tema=numero,
        titulo=raw.get("titulo") or raw.get("tituloTema"),
        descricao=raw.get("descricao") or raw.get("questaoSubmetida"),
        relator=raw.get("relator") or raw.get("ministro"),
        leading_case=raw.get("leadingCase") or raw.get("processoReferencia"),
        situacao=raw.get("situacao") or raw.get("statusTema"),
        data_reconhecimento=raw.get("dataReconhecimento"),
        tese=raw.get("tese") or raw.get("teseFixada"),
    )


async def buscar_repercussao_geral(
    query: str | None = None,
    tema: int | None = None,
) -> list[RepercussaoGeral]:
    """Search STF repercussão geral themes."""
    params: dict[str, Any] = {
        "base": "repercussao_geral",
        "pesquisa_inteiro_teor": "false",
    }
    if query:
        params["queryString"] = query
    if tema:
        params["tema"] = tema
    try:
        data = await _get(STF_API_BASE, params=params)
    except Exception:
        logger.warning("STF Repercussão Geral API retornou erro")
        return []

    if isinstance(data, dict):
        resultados = data.get("result", [])
        if isinstance(resultados, list):
            return [_parse_repercussao(r) for r in resultados if isinstance(r, dict)]
    return []
