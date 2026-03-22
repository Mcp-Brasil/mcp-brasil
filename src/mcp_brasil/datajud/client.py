"""HTTP client for the DataJud (CNJ) API.

The DataJud API uses Elasticsearch-based queries via POST requests.
All endpoints require an API key sent via the ``Authorization: APIKey`` header.

API docs: https://datajud-wiki.cnj.jus.br/api-publica/
"""

from __future__ import annotations

import logging
import os
from typing import Any

from mcp_brasil._shared.rate_limiter import RateLimiter

from .constants import DATAJUD_API_BASE, DEFAULT_PAGE_SIZE, TRIBUNAIS
from .schemas import (
    Assunto,
    Movimentacao,
    Parte,
    Processo,
    ProcessoDetalhe,
)

logger = logging.getLogger(__name__)

_rate_limiter = RateLimiter(max_requests=30, period=60.0)


def _get_api_key() -> str:
    """Return the DataJud API key from environment."""
    key = os.environ.get("DATAJUD_API_KEY", "")
    if not key:
        logger.warning("DATAJUD_API_KEY não configurada")
    return key


def _get_headers() -> dict[str, str]:
    """Return headers with API key for DataJud requests."""
    return {
        "Authorization": f"APIKey {_get_api_key()}",
        "Content-Type": "application/json",
    }


def _tribunal_url(tribunal: str) -> str:
    """Build the DataJud API URL for a tribunal."""
    sigla = tribunal.lower().strip()
    if sigla not in TRIBUNAIS:
        raise ValueError(
            f"Tribunal '{tribunal}' não suportado. "
            f"Use um dos: {', '.join(sorted(TRIBUNAIS.keys()))}"
        )
    return f"{DATAJUD_API_BASE}{TRIBUNAIS[sigla]}/_search"


async def _post(url: str, body: dict[str, Any]) -> Any:
    """POST request to DataJud Elasticsearch API."""
    import httpx

    async with _rate_limiter, httpx.AsyncClient(timeout=30.0) as client:
        response = await client.post(url, json=body, headers=_get_headers())
        response.raise_for_status()
        return response.json()


def _parse_hits(data: Any) -> list[dict[str, Any]]:
    """Extract hits from Elasticsearch response."""
    if not isinstance(data, dict):
        return []
    hits = data.get("hits", {})
    if isinstance(hits, dict):
        result = hits.get("hits", [])
        return result if isinstance(result, list) else []
    return []


def _parse_processo(hit: dict[str, Any]) -> Processo:
    """Parse an Elasticsearch hit into a Processo model."""
    source = hit.get("_source", {})
    classe = source.get("classe", {})
    assuntos = source.get("assuntos", [])
    orgao = source.get("orgaoJulgador", {})

    assunto_str = ""
    if isinstance(assuntos, list) and assuntos:
        first = assuntos[0]
        if isinstance(first, dict):
            assunto_str = first.get("nome", "")

    return Processo(
        numero=source.get("numeroProcesso"),
        classe=classe.get("nome") if isinstance(classe, dict) else str(classe),
        assunto=assunto_str,
        tribunal=source.get("tribunal"),
        orgao_julgador=(orgao.get("nome") if isinstance(orgao, dict) else str(orgao)),
        data_ajuizamento=source.get("dataAjuizamento"),
        data_ultima_atualizacao=source.get("dataHoraUltimaAtualizacao"),
        grau=source.get("grau"),
        nivel_sigilo=source.get("nivelSigilo"),
        formato_numero=source.get("formato", {}).get("numero"),
    )


def _parse_processo_detalhe(hit: dict[str, Any]) -> ProcessoDetalhe:
    """Parse an Elasticsearch hit into a ProcessoDetalhe model."""
    source = hit.get("_source", {})
    classe = source.get("classe", {})
    orgao = source.get("orgaoJulgador", {})

    # Assuntos
    assuntos_raw = source.get("assuntos", [])
    assuntos: list[Assunto] = []
    if isinstance(assuntos_raw, list):
        for a in assuntos_raw:
            if isinstance(a, dict):
                assuntos.append(Assunto(codigo=a.get("codigo"), nome=a.get("nome")))

    # Movimentações
    movs_raw = source.get("movimentos", [])
    movimentacoes: list[Movimentacao] = []
    if isinstance(movs_raw, list):
        for m in movs_raw:
            if isinstance(m, dict):
                complementos = m.get("complementosTabelados", [])
                comp_str = ""
                if isinstance(complementos, list) and complementos:
                    comp_parts = [
                        c.get("descricao", "") for c in complementos if isinstance(c, dict)
                    ]
                    comp_str = "; ".join(p for p in comp_parts if p)
                movimentacoes.append(
                    Movimentacao(
                        data=m.get("dataHora"),
                        nome=m.get("nome"),
                        codigo=m.get("codigo"),
                        complemento=comp_str or m.get("complemento"),
                    )
                )

    # Partes (polo ativo e passivo)
    partes: list[Parte] = []
    for polo, label in [("poloAtivo", "Ativo"), ("poloPassivo", "Passivo")]:
        polo_data = source.get(polo, [])
        if isinstance(polo_data, list):
            for p in polo_data:
                if isinstance(p, dict):
                    partes.append(
                        Parte(
                            nome=p.get("nome"),
                            tipo=p.get("tipoPessoa"),
                            polo=label,
                            documento=p.get("documento"),
                        )
                    )

    return ProcessoDetalhe(
        numero=source.get("numeroProcesso"),
        classe=classe.get("nome") if isinstance(classe, dict) else str(classe),
        assuntos=assuntos,
        tribunal=source.get("tribunal"),
        orgao_julgador=(orgao.get("nome") if isinstance(orgao, dict) else str(orgao)),
        data_ajuizamento=source.get("dataAjuizamento"),
        data_ultima_atualizacao=source.get("dataHoraUltimaAtualizacao"),
        grau=source.get("grau"),
        partes=partes,
        movimentacoes=movimentacoes[:20],  # Limit movimentacoes to 20 most recent
        nivel_sigilo=source.get("nivelSigilo"),
    )


# --- Public API functions ---


async def buscar_processos(
    query: str,
    tribunal: str = "tjsp",
    tamanho: int = DEFAULT_PAGE_SIZE,
) -> list[Processo]:
    """Search processes in a tribunal by free text query."""
    url = _tribunal_url(tribunal)
    body: dict[str, Any] = {
        "query": {"match": {"_all": query}},
        "size": min(tamanho, 100),
    }
    data = await _post(url, body)
    return [_parse_processo(h) for h in _parse_hits(data)]


async def buscar_processo_por_numero(
    numero_processo: str,
    tribunal: str = "tjsp",
) -> ProcessoDetalhe | None:
    """Search a specific process by its NPU (número único do processo)."""
    numero_limpo = numero_processo.replace(".", "").replace("-", "").replace("/", "")
    url = _tribunal_url(tribunal)
    body: dict[str, Any] = {
        "query": {"match": {"numeroProcesso": numero_limpo}},
        "size": 1,
    }
    data = await _post(url, body)
    hits = _parse_hits(data)
    if not hits:
        return None
    return _parse_processo_detalhe(hits[0])


async def buscar_processos_por_classe(
    classe: str,
    tribunal: str = "tjsp",
    tamanho: int = DEFAULT_PAGE_SIZE,
) -> list[Processo]:
    """Search processes by procedural class."""
    url = _tribunal_url(tribunal)
    body: dict[str, Any] = {
        "query": {"match": {"classe.nome": classe}},
        "size": min(tamanho, 100),
    }
    data = await _post(url, body)
    return [_parse_processo(h) for h in _parse_hits(data)]


async def buscar_processos_por_assunto(
    assunto: str,
    tribunal: str = "tjsp",
    tamanho: int = DEFAULT_PAGE_SIZE,
) -> list[Processo]:
    """Search processes by subject."""
    url = _tribunal_url(tribunal)
    body: dict[str, Any] = {
        "query": {"match": {"assuntos.nome": assunto}},
        "size": min(tamanho, 100),
    }
    data = await _post(url, body)
    return [_parse_processo(h) for h in _parse_hits(data)]


async def buscar_processos_por_orgao(
    orgao_julgador: str,
    tribunal: str = "tjsp",
    tamanho: int = DEFAULT_PAGE_SIZE,
) -> list[Processo]:
    """Search processes by judging body (órgão julgador)."""
    url = _tribunal_url(tribunal)
    body: dict[str, Any] = {
        "query": {"match": {"orgaoJulgador.nome": orgao_julgador}},
        "size": min(tamanho, 100),
    }
    data = await _post(url, body)
    return [_parse_processo(h) for h in _parse_hits(data)]


async def consultar_movimentacoes(
    numero_processo: str,
    tribunal: str = "tjsp",
) -> list[Movimentacao]:
    """Get movements for a specific process."""
    detalhe = await buscar_processo_por_numero(numero_processo, tribunal)
    if detalhe is None or detalhe.movimentacoes is None:
        return []
    return detalhe.movimentacoes
