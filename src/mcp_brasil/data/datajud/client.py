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

from .constants import (
    DATAJUD_API_BASE,
    DEFAULT_PAGE_SIZE,
    MAX_PAGE_SIZE,
    MPU_CLASSES_HENRY_BOREL,
    MPU_CLASSES_MARIA_PENHA,
    MPU_CLASSES_NOMES,
    MPU_CLASSES_TODAS,
    MPU_COMPLEMENTO_DESTINATARIO,
    MPU_COMPLEMENTO_TIPO_MEDIDA,
    MPU_DESTINATARIOS,
    MPU_MOV_CONCESSAO_TODOS,
    MPU_MOV_NOMES,
    MPU_MOV_TODOS,
    MPU_TIPOS_MEDIDA,
    TRIBUNAIS,
)
from .schemas import (
    Assunto,
    Movimentacao,
    MPUEstatisticas,
    MPUProcesso,
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


def _extract_sort_token(hits: list[dict[str, Any]]) -> list[Any] | None:
    """Extract the sort value from the last hit for search_after pagination."""
    if not hits:
        return None
    last = hits[-1]
    sort_val = last.get("sort")
    if isinstance(sort_val, list) and sort_val:
        return sort_val
    return None


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
        "query": {
            "multi_match": {
                "query": query,
                "fields": [
                    "numeroProcesso",
                    "classe.nome",
                    "assuntos.nome",
                    "orgaoJulgador.nome",
                ],
            }
        },
        "size": min(tamanho, MAX_PAGE_SIZE),
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
        "size": min(tamanho, MAX_PAGE_SIZE),
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
        "size": min(tamanho, MAX_PAGE_SIZE),
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
        "size": min(tamanho, MAX_PAGE_SIZE),
    }
    data = await _post(url, body)
    return [_parse_processo(h) for h in _parse_hits(data)]


async def buscar_processos_avancado(
    tribunal: str = "tjsp",
    classe_codigo: int | None = None,
    orgao_codigo: int | None = None,
    tamanho: int = DEFAULT_PAGE_SIZE,
    search_after: list[Any] | None = None,
) -> tuple[list[Processo], list[Any] | None]:
    """Advanced search using Elasticsearch bool query with codes and pagination.

    Returns a tuple of (processos, next_search_after_token).
    The token can be passed back to continue pagination.
    """
    url = _tribunal_url(tribunal)

    must_clauses: list[dict[str, Any]] = []
    if classe_codigo is not None:
        must_clauses.append({"match": {"classe.codigo": classe_codigo}})
    if orgao_codigo is not None:
        must_clauses.append({"match": {"orgaoJulgador.codigo": orgao_codigo}})

    if not must_clauses:
        must_clauses.append({"match_all": {}})

    body: dict[str, Any] = {
        "query": {"bool": {"must": must_clauses}},
        "size": min(tamanho, MAX_PAGE_SIZE),
        "sort": [{"@timestamp": {"order": "asc"}}],
    }

    if search_after is not None:
        body["search_after"] = search_after

    data = await _post(url, body)
    hits = _parse_hits(data)
    processos = [_parse_processo(h) for h in hits]
    next_token = _extract_sort_token(hits)
    return processos, next_token


async def consultar_movimentacoes(
    numero_processo: str,
    tribunal: str = "tjsp",
) -> list[Movimentacao]:
    """Get movements for a specific process."""
    detalhe = await buscar_processo_por_numero(numero_processo, tribunal)
    if detalhe is None or detalhe.movimentacoes is None:
        return []
    return detalhe.movimentacoes


# --- MPU (Medidas Protetivas de Urgência) ---


def _mpu_classes_for_lei(lei: str) -> list[int]:
    """Return MPU class codes for the given law filter."""
    if lei == "maria_penha":
        return MPU_CLASSES_MARIA_PENHA
    if lei == "henry_borel":
        return MPU_CLASSES_HENRY_BOREL
    return MPU_CLASSES_TODAS


def _build_mpu_query(
    classes: list[int],
    data_inicio: str | None = None,
    data_fim: str | None = None,
    movimentos: list[int] | None = None,
    destinatario: int | None = None,
    tipo_medida: int | None = None,
    size: int = DEFAULT_PAGE_SIZE,
) -> dict[str, Any]:
    """Build Elasticsearch query for MPU searches."""
    must: list[dict[str, Any]] = [
        {"terms": {"classe.codigo": classes}},
    ]

    if data_inicio or data_fim:
        date_range: dict[str, str] = {}
        if data_inicio:
            date_range["gte"] = data_inicio
        if data_fim:
            date_range["lte"] = data_fim
        must.append({"range": {"dataAjuizamento": date_range}})

    comp_codigo_field = "movimentos.complementosTabelados.codigo"
    comp_valor_field = "movimentos.complementosTabelados.valor"

    if movimentos:
        nested_must: list[dict[str, Any]] = [
            {"terms": {"movimentos.codigo": movimentos}},
        ]
        if destinatario is not None:
            nested_must.append(
                {
                    "nested": {
                        "path": "movimentos.complementosTabelados",
                        "query": {
                            "bool": {
                                "must": [
                                    {"term": {comp_codigo_field: MPU_COMPLEMENTO_DESTINATARIO}},
                                    {"term": {comp_valor_field: destinatario}},
                                ]
                            }
                        },
                    }
                }
            )
        if tipo_medida is not None:
            nested_must.append(
                {
                    "nested": {
                        "path": "movimentos.complementosTabelados",
                        "query": {
                            "bool": {
                                "must": [
                                    {"term": {comp_codigo_field: MPU_COMPLEMENTO_TIPO_MEDIDA}},
                                    {"term": {comp_valor_field: tipo_medida}},
                                ]
                            }
                        },
                    }
                }
            )
        must.append(
            {
                "nested": {
                    "path": "movimentos",
                    "query": {"bool": {"must": nested_must}},
                }
            }
        )

    return {
        "query": {"bool": {"must": must}},
        "size": min(size, MAX_PAGE_SIZE),
        "sort": [{"dataAjuizamento": {"order": "desc"}}],
        "_source": [
            "numeroProcesso",
            "classe",
            "assuntos",
            "dataAjuizamento",
            "orgaoJulgador",
            "movimentos",
        ],
    }


def _parse_mpu_processo(hit: dict[str, Any]) -> MPUProcesso:
    """Parse an Elasticsearch hit into an MPUProcesso model."""
    source = hit.get("_source", {})
    classe = source.get("classe", {})
    orgao = source.get("orgaoJulgador", {})
    classe_codigo = classe.get("codigo") if isinstance(classe, dict) else None

    movs_raw = source.get("movimentos", [])
    movimentos: list[Movimentacao] = []
    mpu_mov_set = set(MPU_MOV_TODOS)
    if isinstance(movs_raw, list):
        for m in movs_raw:
            if isinstance(m, dict) and m.get("codigo") in mpu_mov_set:
                complementos = m.get("complementosTabelados", [])
                comp_str = ""
                if isinstance(complementos, list) and complementos:
                    comp_parts = [
                        c.get("descricao", "") for c in complementos if isinstance(c, dict)
                    ]
                    comp_str = "; ".join(p for p in comp_parts if p)
                movimentos.append(
                    Movimentacao(
                        data=m.get("dataHora"),
                        nome=m.get("nome") or MPU_MOV_NOMES.get(m.get("codigo", 0), ""),
                        codigo=m.get("codigo"),
                        complemento=comp_str or None,
                    )
                )

    return MPUProcesso(
        numero=source.get("numeroProcesso"),
        classe_codigo=classe_codigo,
        classe_nome=(
            MPU_CLASSES_NOMES.get(classe_codigo or 0, classe.get("nome"))
            if isinstance(classe, dict)
            else str(classe)
        ),
        tribunal=source.get("tribunal"),
        orgao_julgador=orgao.get("nome") if isinstance(orgao, dict) else str(orgao),
        data_ajuizamento=source.get("dataAjuizamento"),
        movimentos=movimentos if movimentos else None,
    )


async def buscar_medidas_protetivas(
    tribunal: str = "tjsp",
    data_inicio: str | None = None,
    data_fim: str | None = None,
    lei: str = "ambas",
    size: int = DEFAULT_PAGE_SIZE,
) -> list[MPUProcesso]:
    """Search MPU processes by tribunal, period, and law type."""
    url = _tribunal_url(tribunal)
    classes = _mpu_classes_for_lei(lei)
    body = _build_mpu_query(classes=classes, data_inicio=data_inicio, data_fim=data_fim, size=size)
    data = await _post(url, body)
    return [_parse_mpu_processo(h) for h in _parse_hits(data)]


async def buscar_mpu_concedidas(
    tribunal: str = "tjsp",
    data_inicio: str | None = None,
    data_fim: str | None = None,
    destinatario: str = "todos",
    size: int = DEFAULT_PAGE_SIZE,
) -> list[MPUProcesso]:
    """Search granted MPUs (total or partial) with optional recipient filter."""
    url = _tribunal_url(tribunal)
    dest_code = MPU_DESTINATARIOS.get(destinatario)
    body = _build_mpu_query(
        classes=MPU_CLASSES_TODAS,
        data_inicio=data_inicio,
        data_fim=data_fim,
        movimentos=MPU_MOV_CONCESSAO_TODOS,
        destinatario=dest_code,
        size=size,
    )
    data = await _post(url, body)
    return [_parse_mpu_processo(h) for h in _parse_hits(data)]


async def buscar_mpu_por_tipo(
    tribunal: str = "tjsp",
    tipo_medida: str = "afastamento_lar",
    data_inicio: str | None = None,
    data_fim: str | None = None,
    size: int = DEFAULT_PAGE_SIZE,
) -> list[MPUProcesso]:
    """Search MPUs by type of protective measure."""
    url = _tribunal_url(tribunal)
    tipo_code = MPU_TIPOS_MEDIDA.get(tipo_medida)
    if tipo_code is None:
        raise ValueError(
            f"Tipo de medida '{tipo_medida}' não reconhecido. "
            f"Opções: {', '.join(sorted(MPU_TIPOS_MEDIDA.keys()))}"
        )
    body = _build_mpu_query(
        classes=MPU_CLASSES_TODAS,
        data_inicio=data_inicio,
        data_fim=data_fim,
        movimentos=MPU_MOV_CONCESSAO_TODOS,
        tipo_medida=tipo_code,
        size=size,
    )
    data = await _post(url, body)
    return [_parse_mpu_processo(h) for h in _parse_hits(data)]


async def estatisticas_mpu(
    tribunal: str | None = None,
    ano: int | None = None,
) -> MPUEstatisticas:
    """Get aggregated MPU statistics for a tribunal/year."""
    trib = tribunal or "tjsp"
    url = _tribunal_url(trib)

    data_inicio = f"{ano}-01-01" if ano else None
    data_fim = f"{ano}-12-31" if ano else None

    must: list[dict[str, Any]] = [
        {"terms": {"classe.codigo": MPU_CLASSES_TODAS}},
    ]
    if data_inicio and data_fim:
        must.append({"range": {"dataAjuizamento": {"gte": data_inicio, "lte": data_fim}}})

    body: dict[str, Any] = {
        "query": {"bool": {"must": must}},
        "size": 0,
        "aggs": {
            "por_classe": {
                "terms": {"field": "classe.codigo", "size": 10},
            },
            "por_movimento": {
                "nested": {"path": "movimentos"},
                "aggs": {
                    "mpu_movimentos": {
                        "filter": {"terms": {"movimentos.codigo": MPU_MOV_TODOS}},
                        "aggs": {
                            "por_tipo": {
                                "terms": {"field": "movimentos.codigo", "size": 20},
                            }
                        },
                    }
                },
            },
        },
    }

    data = await _post(url, body)
    total_hits = 0
    if isinstance(data, dict):
        hits_meta = data.get("hits", {})
        if isinstance(hits_meta, dict):
            total_val = hits_meta.get("total", {})
            if isinstance(total_val, dict):
                total_hits = total_val.get("value", 0)
            elif isinstance(total_val, int):
                total_hits = total_val

    por_classe: dict[str, int] = {}
    por_decisao: dict[str, int] = {}
    aggs = data.get("aggregations", {}) if isinstance(data, dict) else {}

    classe_buckets = aggs.get("por_classe", {}).get("buckets", [])
    if isinstance(classe_buckets, list):
        for b in classe_buckets:
            if isinstance(b, dict):
                key = b.get("key")
                key_int = int(key) if key is not None else 0
                por_classe[MPU_CLASSES_NOMES.get(key_int, str(key))] = b.get("doc_count", 0)

    mov_agg = aggs.get("por_movimento", {}).get("mpu_movimentos", {})
    mov_buckets = mov_agg.get("por_tipo", {}).get("buckets", [])
    if isinstance(mov_buckets, list):
        for b in mov_buckets:
            if isinstance(b, dict):
                key = b.get("key")
                key_int = int(key) if key is not None else 0
                por_decisao[MPU_MOV_NOMES.get(key_int, str(key))] = b.get("doc_count", 0)

    periodo = f"{ano}" if ano else "todos"

    return MPUEstatisticas(
        total=total_hits,
        tribunal=trib,
        periodo=periodo,
        por_decisao=por_decisao if por_decisao else None,
        por_classe=por_classe if por_classe else None,
    )


async def timeline_mpu(
    numero_processo: str,
    tribunal: str = "tjsp",
) -> list[Movimentacao]:
    """Get the MPU-specific movement timeline for a process."""
    detalhe = await buscar_processo_por_numero(numero_processo, tribunal)
    if detalhe is None or detalhe.movimentacoes is None:
        return []

    mpu_mov_set = set(MPU_MOV_TODOS)
    return [m for m in detalhe.movimentacoes if m.codigo in mpu_mov_set]
