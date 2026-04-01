"""HTTP client for the Câmara dos Deputados API.

All functions return typed Pydantic models. No LLM formatting here (ADR-001).

API docs: https://dadosabertos.camara.leg.br/swagger/api.html

The Câmara API wraps results in ``{"dados": [...], "links": [...]}``.
The ``_get()`` helper extracts the ``dados`` field automatically.
"""

from __future__ import annotations

import logging
from typing import Any

from mcp_brasil._shared.http_client import http_get
from mcp_brasil._shared.rate_limiter import RateLimiter

from .constants import (
    DEFAULT_ITENS,
    DEPUTADOS_URL,
    EVENTOS_URL,
    FRENTES_URL,
    ORGAOS_URL,
    PROPOSICOES_URL,
    VOTACOES_URL,
)
from .schemas import (
    Deputado,
    DespesaDeputado,
    Evento,
    FrenteParlamentar,
    Orgao,
    Proposicao,
    Tramitacao,
    Votacao,
    VotoNominal,
)

logger = logging.getLogger(__name__)

# Conservative limit — Câmara API does not document rate limits
_rate_limiter = RateLimiter(max_requests=60, period=60.0)


async def _get(url: str, params: dict[str, Any] | None = None) -> Any:
    """GET request extracting the ``dados`` field from the Câmara API envelope."""
    async with _rate_limiter:
        data = await http_get(url, params=params)
    if isinstance(data, dict):
        return data.get("dados", data)
    return data


def _safe_list(data: Any, endpoint: str) -> list[dict[str, Any]]:
    """Ensure data is a list, logging a warning otherwise."""
    if isinstance(data, list):
        return data
    logger.warning(
        "Resposta inesperada (esperava list) do endpoint %s: %s",
        endpoint,
        type(data).__name__,
    )
    return []


# --- Parsing helpers --------------------------------------------------------


def _parse_deputado(raw: dict[str, Any]) -> Deputado:
    return Deputado(
        id=raw.get("id"),
        nome=raw.get("nome") or raw.get("nomeCivil"),
        sigla_partido=raw.get("siglaPartido"),
        sigla_uf=raw.get("siglaUf"),
        email=raw.get("email"),
        foto=raw.get("urlFoto"),
        legislatura=raw.get("idLegislatura"),
    )


def _parse_proposicao(raw: dict[str, Any]) -> Proposicao:
    status = raw.get("statusProposicao") or {}
    orgao_status = status.get("orgao") or {} if isinstance(status, dict) else {}
    return Proposicao(
        id=raw.get("id"),
        sigla_tipo=raw.get("siglaTipo"),
        numero=raw.get("numero"),
        ano=raw.get("ano"),
        ementa=raw.get("ementa"),
        data_apresentacao=raw.get("dataApresentacao"),
        situacao=status.get("descricaoSituacao") if isinstance(status, dict) else None,
        orgao_situacao=orgao_status.get("sigla") if isinstance(orgao_status, dict) else None,
        regime=status.get("regime") if isinstance(status, dict) else None,
        url_inteiro_teor=raw.get("urlInteiroTeor"),
        uri=raw.get("uri"),
    )


def _parse_proposicao_detalhe(raw: dict[str, Any]) -> Proposicao:
    """Parse proposição from the detail endpoint (includes author info)."""
    status = raw.get("statusProposicao") or {}
    orgao_status = status.get("orgao") or {} if isinstance(status, dict) else {}

    # Author can be in uriAutores or directly in the response
    # The detail endpoint has nested status with author info
    return Proposicao(
        id=raw.get("id"),
        sigla_tipo=raw.get("siglaTipo"),
        numero=raw.get("numero"),
        ano=raw.get("ano"),
        ementa=raw.get("ementa") or raw.get("ementaDetalhada"),
        data_apresentacao=raw.get("dataApresentacao"),
        situacao=status.get("descricaoSituacao") if isinstance(status, dict) else None,
        orgao_situacao=orgao_status.get("sigla") if isinstance(orgao_status, dict) else None,
        regime=status.get("regime") if isinstance(status, dict) else None,
        url_inteiro_teor=raw.get("urlInteiroTeor"),
        uri=raw.get("uri"),
    )


def _parse_tramitacao(raw: dict[str, Any]) -> Tramitacao:
    orgao = raw.get("orgao") if isinstance(raw.get("orgao"), dict) else {}
    return Tramitacao(
        data=raw.get("dataHora") or raw.get("data"),
        descricao=raw.get("descricaoTramitacao"),
        orgao=orgao.get("sigla") or orgao.get("nome") if orgao else None,
        situacao=raw.get("descricaoSituacao"),
        despacho=raw.get("despacho"),
    )


def _parse_votacao(raw: dict[str, Any]) -> Votacao:
    prop = raw.get("proposicaoObjeto") or raw.get("proposicao") or {}
    return Votacao(
        id=raw.get("id"),
        data=raw.get("dataHoraRegistro") or raw.get("data"),
        descricao=raw.get("descricao"),
        aprovacao=raw.get("aprovacao"),
        proposicao_id=prop.get("id") if isinstance(prop, dict) else None,
        proposicao_descricao=prop.get("ementa") if isinstance(prop, dict) else None,
    )


def _parse_voto(raw: dict[str, Any]) -> VotoNominal:
    deputado = raw.get("deputado_") or raw.get("deputado") or {}
    return VotoNominal(
        deputado_id=deputado.get("id") if isinstance(deputado, dict) else None,
        deputado_nome=deputado.get("nome") if isinstance(deputado, dict) else None,
        partido=deputado.get("siglaPartido") if isinstance(deputado, dict) else None,
        uf=deputado.get("siglaUf") if isinstance(deputado, dict) else None,
        voto=raw.get("tipoVoto"),
    )


def _parse_despesa(raw: dict[str, Any]) -> DespesaDeputado:
    return DespesaDeputado(
        tipo_despesa=raw.get("tipoDespesa"),
        fornecedor=raw.get("nomeFornecedor"),
        cnpj_cpf=raw.get("cnpjCpfFornecedor"),
        valor_documento=raw.get("valorDocumento"),
        valor_liquido=raw.get("valorLiquido"),
        data_documento=raw.get("dataDocumento"),
        mes=raw.get("mes"),
        ano=raw.get("ano"),
    )


def _parse_evento(raw: dict[str, Any]) -> Evento:
    orgaos = raw.get("orgaos") or []
    orgaos_str = (
        ", ".join(o.get("sigla") or o.get("nome") or "" for o in orgaos if isinstance(o, dict))
        if isinstance(orgaos, list)
        else None
    )
    return Evento(
        id=raw.get("id"),
        data_inicio=raw.get("dataHoraInicio"),
        data_fim=raw.get("dataHoraFim"),
        titulo=raw.get("descricaoTipo"),
        descricao=raw.get("descricao"),
        local=raw.get("localExterno") or raw.get("localCamara", {}).get("nome"),
        situacao=raw.get("situacao"),
        orgaos=orgaos_str or None,
    )


def _parse_orgao(raw: dict[str, Any]) -> Orgao:
    return Orgao(
        id=raw.get("id"),
        sigla=raw.get("sigla"),
        nome=raw.get("nome"),
        tipo=raw.get("tipoOrgao") or raw.get("codTipoOrgao"),
        situacao=raw.get("situacao"),
    )


def _parse_frente(raw: dict[str, Any]) -> FrenteParlamentar:
    coordenador = raw.get("coordenador") or {}
    return FrenteParlamentar(
        id=raw.get("id"),
        titulo=raw.get("titulo"),
        legislatura=raw.get("idLegislatura"),
        coordenador=coordenador.get("nome") if isinstance(coordenador, dict) else None,
        situacao=raw.get("situacao"),
    )


# --- Public API functions ---------------------------------------------------


async def listar_deputados(
    nome: str | None = None,
    sigla_partido: str | None = None,
    sigla_uf: str | None = None,
    legislatura: int | None = None,
    pagina: int = 1,
) -> list[Deputado]:
    """Lista deputados com filtros opcionais."""
    params: dict[str, Any] = {
        "pagina": pagina,
        "itens": DEFAULT_ITENS,
        "ordem": "ASC",
        "ordenarPor": "nome",
    }
    if nome:
        params["nome"] = nome
    if sigla_partido:
        params["siglaPartido"] = sigla_partido
    if sigla_uf:
        params["siglaUf"] = sigla_uf
    if legislatura:
        params["idLegislatura"] = legislatura
    data = await _get(DEPUTADOS_URL, params)
    return [_parse_deputado(d) for d in _safe_list(data, "deputados")]


async def obter_deputado(deputado_id: int) -> Deputado | None:
    """Obtém detalhes de um deputado pelo ID."""
    data = await _get(f"{DEPUTADOS_URL}/{deputado_id}")
    if isinstance(data, dict):
        return _parse_deputado(data)
    return None


async def buscar_proposicoes(
    sigla_tipo: str | None = None,
    numero: int | None = None,
    ano: int | None = None,
    keywords: str | None = None,
    id_deputado_autor: int | None = None,
    pagina: int = 1,
) -> list[Proposicao]:
    """Busca proposições com filtros."""
    params: dict[str, Any] = {
        "pagina": pagina,
        "itens": DEFAULT_ITENS,
        "ordem": "DESC",
        "ordenarPor": "id",
    }
    if sigla_tipo:
        params["siglaTipo"] = sigla_tipo
    if numero:
        params["numero"] = numero
    if ano:
        params["ano"] = ano
    if keywords:
        params["keywords"] = keywords
    if id_deputado_autor is not None:
        params["idDeputadoAutor"] = id_deputado_autor
    data = await _get(PROPOSICOES_URL, params)
    return [_parse_proposicao(p) for p in _safe_list(data, "proposicoes")]


async def obter_proposicao(proposicao_id: int) -> Proposicao | None:
    """Obtém detalhes completos de uma proposição, incluindo autores."""
    data = await _get(f"{PROPOSICOES_URL}/{proposicao_id}")
    if not isinstance(data, dict):
        return None

    prop = _parse_proposicao_detalhe(data)

    # Fetch authors separately (the detail endpoint doesn't inline them)
    try:
        autores_data = await _get(f"{PROPOSICOES_URL}/{proposicao_id}/autores")
        autores_list = _safe_list(autores_data, "autores")
        if autores_list:
            primeiro = autores_list[0]
            prop.autor = primeiro.get("nome")
            if isinstance(primeiro, dict):
                prop.autor_partido = primeiro.get("siglaPartido")
                prop.autor_uf = primeiro.get("siglaUf")
            if len(autores_list) > 1:
                prop.autor = f"{prop.autor} e outros ({len(autores_list)} autores)"
    except Exception:
        logger.warning("Falha ao buscar autores da proposição %d", proposicao_id)

    return prop


async def obter_tramitacoes(proposicao_id: int) -> list[Tramitacao]:
    """Obtém tramitações de uma proposição."""
    url = f"{PROPOSICOES_URL}/{proposicao_id}/tramitacoes"
    data = await _get(url)
    return [_parse_tramitacao(t) for t in _safe_list(data, "tramitacoes")]


async def listar_votacoes(
    proposicao_id: int | None = None,
    data_inicio: str | None = None,
    data_fim: str | None = None,
    pagina: int = 1,
) -> list[Votacao]:
    """Lista votações com filtros opcionais."""
    if proposicao_id:
        url = f"{PROPOSICOES_URL}/{proposicao_id}/votacoes"
        data = await _get(url)
    else:
        params: dict[str, Any] = {
            "pagina": pagina,
            "itens": DEFAULT_ITENS,
            "ordem": "DESC",
            "ordenarPor": "dataHoraRegistro",
        }
        if data_inicio:
            params["dataInicio"] = data_inicio
        if data_fim:
            params["dataFim"] = data_fim
        data = await _get(VOTACOES_URL, params)
    return [_parse_votacao(v) for v in _safe_list(data, "votacoes")]


async def obter_votos(votacao_id: str) -> list[VotoNominal]:
    """Obtém votos nominais de uma votação."""
    url = f"{VOTACOES_URL}/{votacao_id}/votos"
    data = await _get(url)
    return [_parse_voto(v) for v in _safe_list(data, "votos")]


async def listar_despesas(
    deputado_id: int,
    ano: int | None = None,
    mes: int | None = None,
    pagina: int = 1,
) -> list[DespesaDeputado]:
    """Lista despesas de cota parlamentar (CEAP) de um deputado."""
    url = f"{DEPUTADOS_URL}/{deputado_id}/despesas"
    params: dict[str, Any] = {
        "pagina": pagina,
        "itens": DEFAULT_ITENS,
        "ordem": "DESC",
        "ordenarPor": "ano",
    }
    if ano:
        params["ano"] = ano
    if mes:
        params["mes"] = mes
    data = await _get(url, params)
    return [_parse_despesa(d) for d in _safe_list(data, "despesas")]


async def listar_eventos(
    data_inicio: str | None = None,
    data_fim: str | None = None,
    pagina: int = 1,
) -> list[Evento]:
    """Lista eventos legislativos (sessões, audiências, reuniões)."""
    params: dict[str, Any] = {
        "pagina": pagina,
        "itens": DEFAULT_ITENS,
        "ordem": "DESC",
        "ordenarPor": "dataHoraInicio",
    }
    if data_inicio:
        params["dataInicio"] = data_inicio
    if data_fim:
        params["dataFim"] = data_fim
    data = await _get(EVENTOS_URL, params)
    return [_parse_evento(e) for e in _safe_list(data, "eventos")]


async def listar_orgaos(
    sigla: str | None = None,
    tipo: str | None = None,
    pagina: int = 1,
) -> list[Orgao]:
    """Lista órgãos legislativos (comissões, CPIs, etc.)."""
    params: dict[str, Any] = {
        "pagina": pagina,
        "itens": DEFAULT_ITENS,
        "ordem": "ASC",
        "ordenarPor": "nome",
    }
    if sigla:
        params["sigla"] = sigla
    if tipo:
        params["codTipoOrgao"] = tipo
    data = await _get(ORGAOS_URL, params)
    return [_parse_orgao(o) for o in _safe_list(data, "orgaos")]


async def listar_frentes(
    legislatura: int | None = None,
    pagina: int = 1,
) -> list[FrenteParlamentar]:
    """Lista frentes parlamentares."""
    params: dict[str, Any] = {"pagina": pagina, "itens": DEFAULT_ITENS}
    if legislatura:
        params["idLegislatura"] = legislatura
    data = await _get(FRENTES_URL, params)
    return [_parse_frente(f) for f in _safe_list(data, "frentes")]
