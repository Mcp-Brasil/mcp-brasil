"""HTTP client for the TCM-GO public search surfaces.

Endpoints:
    POST /eSiproc/api/elastic/pesquisar    → buscar_processos
    POST /eSiproc/api/elastic/listar-campo → listar_valores_campo
    GET  /ecs/s/tcmjuris                   → buscar_acordaos

Both surfaces are public and require no authentication. eSiproc is a JSON
(Elasticsearch) API of electronic proceedings (metadata only, no full text of
the merits). TCMJuris is a server-rendered HTML search over published julgados
with a direct PDF link.
"""

from __future__ import annotations

import re
import unicodedata
from typing import Any

from bs4 import BeautifulSoup

from mcp_brasil._shared.http_client import create_client, http_post
from mcp_brasil.exceptions import HttpClientError

from .constants import (
    ESIPROC_LISTAR_CAMPO_URL,
    ESIPROC_PESQUISAR_URL,
    TCMJURIS_BASE,
    TCMJURIS_HEADERS,
    TCMJURIS_MAX_ROWS,
    TCMJURIS_SEM_RESULTADOS,
    TCMJURIS_URL,
)
from .schemas import (
    Acordao,
    AcordaoResultado,
    Decisao,
    Envolvido,
    Processo,
    ProcessoResultado,
    ValorCampo,
)


def _payload_processos() -> dict[str, Any]:
    """Base payload for the eSiproc proceedings search.

    The API requires every filter key to be present (null when unused).
    """
    return {
        "numero": None,
        "termo": None,
        "campo": None,
        "filtros": {
            "municipio.keyword": None,
            "unidade_gestora.keyword": None,
            "relator.keyword": None,
            "regiao.keyword": None,
            "tipo_protocolo.keyword": None,
            "tags.nome.keyword": None,
            "area.keyword": None,
            "ano_competencia": None,
            "periodo_competencia.keyword": None,
            "lotacao.keyword": None,
            "envolvidos.situacao.keyword": None,
            "envolvidos.tipo_envolvimento.keyword": None,
            "range.data_protocolo.gte": None,
            "range.data_protocolo.lte": None,
            "finalizado.keyword": None,
        },
        "filtrosLike": {"numero.keyword": None},
        "filtrosLikeNested": {},
        "pagina": 0,
        "filtroPeriodoJulgamento": {"inicio": None, "fim": None},
    }


def _parse_processo(source: dict[str, Any]) -> Processo:
    """Map a raw eSiproc ``_source`` document into a Processo."""
    envolvidos = [
        Envolvido(nome=e.get("nome"), tipo_envolvimento=e.get("tipo_envolvimento"))
        for e in (source.get("envolvidos") or [])
        if isinstance(e, dict)
    ]
    decisoes = [
        Decisao(nome=d.get("nome"), arquivo=d.get("nome_arquivo"), data=d.get("data_criacao"))
        for d in (source.get("decisoes") or [])
        if isinstance(d, dict)
    ]
    return Processo(
        numero=source.get("numero"),
        municipio=source.get("municipio"),
        unidade_gestora=source.get("unidade_gestora"),
        poder=source.get("poder"),
        tipo_protocolo=source.get("tipo_protocolo"),
        ano_competencia=source.get("ano_competencia"),
        periodo_competencia=source.get("periodo_competencia"),
        relator=source.get("relator"),
        procurador_contas=source.get("procurador_contas"),
        st_fase=source.get("st_fase"),
        finalizado=source.get("finalizado"),
        passo=source.get("passo"),
        data_protocolo=source.get("data_protocolo"),
        data_ultima_tramitacao=source.get("data_ultima_tramitacao"),
        data_finalizacao=source.get("data_finalizacao"),
        envolvidos=envolvidos,
        decisoes=decisoes,
    )


async def listar_valores_campo(
    *,
    campo: str,
    municipio: str | None = None,
) -> list[ValorCampo]:
    """List the distinct values of an eSiproc field (aggregation).

    Args:
        campo: Field name, e.g. ``"municipio.keyword"`` or ``"relator.keyword"``.
        municipio: Optional municipality (official name, upper case with
            accents) to scope the aggregation.

    Returns:
        Distinct values with the number of matching proceedings.
    """
    payload: dict[str, Any] = {
        "numero": None,
        "termo": None,
        "campo": campo,
        "filtros": {},
        "filtrosLike": None,
        "filtrosLikeNested": None,
        "pagina": None,
        "filtroPeriodoJulgamento": None,
    }
    if municipio:
        payload["filtros"]["municipio.keyword"] = municipio

    data = await http_post(ESIPROC_LISTAR_CAMPO_URL, json_body=payload)
    if not isinstance(data, dict):
        raise HttpClientError("eSiproc (listar-campo) retornou uma resposta inesperada")

    aggregations = data.get("aggregations")
    if not isinstance(aggregations, dict):
        raise HttpClientError("eSiproc (listar-campo) sem 'aggregations' na resposta")
    distinct = aggregations.get("distinct")
    if not isinstance(distinct, dict) or not isinstance(distinct.get("buckets"), list):
        raise HttpClientError(
            "eSiproc (listar-campo) sem 'aggregations.distinct.buckets' na resposta"
        )

    buckets = distinct["buckets"]
    return [
        ValorCampo(valor=b.get("key"), quantidade=b.get("doc_count"))
        for b in buckets
        if isinstance(b, dict)
    ]


async def buscar_processos(
    *,
    municipio: str | None = None,
    unidade_gestora: str | None = None,
    tipo_protocolo: str | None = None,
    ano: int | None = None,
    relator: str | None = None,
    finalizado: str | None = None,
    numero: str | None = None,
    pagina: int = 0,
) -> ProcessoResultado:
    """Search electronic proceedings in the eSiproc API.

    Args:
        municipio: Official municipality name, upper case with accents.
        unidade_gestora: Managing unit name.
        tipo_protocolo: Protocol type (e.g. ``"Contas de Gestão"``).
        ano: Competence year (e.g. ``2024``).
        relator: Reporting councillor name.
        finalizado: ``"SIM"`` or ``"NÃO"``.
        numero: Partial proceeding number.
        pagina: Zero-indexed page (about 10 records per page).

    Returns:
        Paginated proceedings with their metadata.
    """
    payload = _payload_processos()
    filtros = payload["filtros"]
    if municipio:
        filtros["municipio.keyword"] = municipio
    if unidade_gestora:
        filtros["unidade_gestora.keyword"] = unidade_gestora
    if tipo_protocolo:
        filtros["tipo_protocolo.keyword"] = tipo_protocolo
    if ano:
        filtros["ano_competencia"] = ano
    if relator:
        filtros["relator.keyword"] = relator
    if finalizado:
        filtros["finalizado.keyword"] = finalizado
    if numero:
        payload["filtrosLike"]["numero.keyword"] = numero
    payload["pagina"] = pagina

    data = await http_post(ESIPROC_PESQUISAR_URL, json_body=payload)
    if not isinstance(data, dict):
        raise HttpClientError("eSiproc (pesquisar) retornou uma resposta inesperada")

    hits = data.get("hits")
    if not isinstance(hits, dict):
        raise HttpClientError("eSiproc (pesquisar) sem 'hits' na resposta")
    total_obj = hits.get("total")
    if not isinstance(total_obj, dict) or not isinstance(total_obj.get("value"), int):
        raise HttpClientError("eSiproc (pesquisar) sem 'hits.total.value' na resposta")
    hits_list = hits.get("hits")
    if not isinstance(hits_list, list):
        raise HttpClientError("eSiproc (pesquisar) sem 'hits.hits' na resposta")

    processos = [
        _parse_processo(hit.get("_source", {})) for hit in hits_list if isinstance(hit, dict)
    ]
    return ProcessoResultado(total=total_obj["value"], pagina=pagina, processos=processos)


def _normalizar(texto: str | None) -> str:
    """Upper-case, accent- and apostrophe-free form used by TCMJuris results."""
    if not texto:
        return ""
    nfkd = unicodedata.normalize("NFKD", texto)
    sem_acento = "".join(c for c in nfkd if not unicodedata.combining(c))
    return re.sub(r"[''`]", "", sem_acento).upper().strip()


def _url_absoluta(href: str) -> str:
    """Turn a possibly relative TCMJuris PDF link into an absolute URL."""
    href = href.strip()
    if href.startswith("http"):
        return href
    if href.startswith("/"):
        return f"{TCMJURIS_BASE}{href}"
    return f"{TCMJURIS_BASE}/{href}"


def _parse_acordaos_html(html: str) -> list[Acordao]:
    """Parse the TCMJuris results table into Acordao records.

    Each result row has seven sequential cells: sequence number, PDF link with
    the acórdão number, proceeding, judgment date, reporting councillor,
    municipality and subject.
    """
    soup = BeautifulSoup(html, "lxml")
    acordaos: list[Acordao] = []
    for tr in soup.find_all("tr"):
        cells = tr.find_all("td")
        if len(cells) < 7:
            continue
        if not cells[0].get_text(strip=True).isdigit():
            continue
        link = cells[1].find("a", href=True)
        if link is None:
            continue
        acordaos.append(
            Acordao(
                numero_acordao=link.get_text(strip=True) or None,
                processo=cells[2].get_text(strip=True) or None,
                data_julgamento=cells[3].get_text(strip=True) or None,
                relator=cells[4].get_text(strip=True) or None,
                municipio=cells[5].get_text(strip=True) or None,
                assunto=cells[6].get_text(strip=True) or None,
                url_pdf=_url_absoluta(str(link.get("href") or "")),
            )
        )
    return acordaos


def _tem_tabela_resultados(html: str) -> bool:
    """Whether the HTML contains a results table with the expected columns.

    Used to tell a legitimate empty result (table rendered, zero rows) from an
    unexpected layout, which must never be reported as an empty list.
    """
    soup = BeautifulSoup(html, "lxml")
    for table in soup.find_all("table"):
        texto = _normalizar(table.get_text(" ", strip=True))
        if "ACORDAO" in texto and "PROCESSO" in texto:
            return True
    return False


def _tem_marca_sem_resultados(html: str) -> bool:
    """Whether the HTML carries an explicit 'no results' marker."""
    texto = _normalizar(BeautifulSoup(html, "lxml").get_text(" ", strip=True))
    return any(_normalizar(marca) in texto for marca in TCMJURIS_SEM_RESULTADOS)


async def buscar_acordaos(
    *,
    pesquisa: str | None = None,
    municipio: str | None = None,
    assunto: str | None = None,
    relator: str | None = None,
    numero_processo: str | None = None,
    data_sessao_ini: str | None = None,
    data_sessao_fin: str | None = None,
    oplog: str = "AND",
) -> AcordaoResultado:
    """Search published acórdãos in the TCMJuris full-text search.

    The server-side ``municipio``/``assunto`` filters match loosely, so results
    are re-filtered client-side by exact normalized equality. A response is
    capped at 1000 rows by the source.

    Args:
        pesquisa: Free-text query (accents optional).
        municipio: Municipality name (re-filtered exactly, accent-insensitive).
        assunto: Subject (re-filtered exactly, accent-insensitive).
        relator: Reporting councillor.
        numero_processo: Proceeding number (format ``NNNNN/AA``).
        data_sessao_ini: Session start date (``DD/MM/AAAA``).
        data_sessao_fin: Session end date (``DD/MM/AAAA``).
        oplog: Free-text operator, ``"AND"`` or ``"OR"``.

    Returns:
        Published acórdãos with the PDF link, plus server/re-filter counts.
    """
    params: dict[str, str] = {"guest": "true", "expand": "1", "oplog": oplog}
    if pesquisa:
        params["pesquisa"] = pesquisa
    if municipio:
        params["municipio"] = municipio
    if assunto:
        params["assunto"] = assunto
    if relator:
        params["relator"] = relator
    if numero_processo:
        params["numero_processo"] = numero_processo
    if data_sessao_ini:
        params["data_sessao_ini"] = data_sessao_ini
    if data_sessao_fin:
        params["data_sessao_fin"] = data_sessao_fin

    async with create_client(headers=TCMJURIS_HEADERS) as http:
        try:
            response = await http.get(TCMJURIS_URL, params=params)
            response.raise_for_status()
        except Exception as exc:
            raise HttpClientError(f"Falha ao consultar o TCMJuris: {exc}") from exc

    brutos = _parse_acordaos_html(response.text)

    if not brutos and not (
        _tem_tabela_resultados(response.text) or _tem_marca_sem_resultados(response.text)
    ):
        raise HttpClientError(
            "TCMJuris retornou HTML sem a tabela de resultados esperada "
            "(layout inesperado ou indisponível)"
        )

    acordaos = brutos
    if municipio:
        alvo = _normalizar(municipio)
        acordaos = [a for a in acordaos if _normalizar(a.municipio) == alvo]
    if assunto:
        alvo = _normalizar(assunto)
        acordaos = [a for a in acordaos if _normalizar(a.assunto) == alvo]

    return AcordaoResultado(
        total_servidor=len(brutos),
        total_apos_refiltro=len(acordaos),
        truncado=len(brutos) >= TCMJURIS_MAX_ROWS,
        acordaos=acordaos,
    )
