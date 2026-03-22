"""HTTP client for the TSE (DivulgaCandContas) API.

REST API without authentication.
API docs: Swagger at divulgacandcontas.tse.jus.br (unofficial).
"""

from __future__ import annotations

import logging
from typing import Any

from mcp_brasil._shared.http_client import http_get
from mcp_brasil._shared.rate_limiter import RateLimiter

from .constants import CANDIDATURA_URL, ELEICAO_URL, PRESTADOR_URL
from .schemas import (
    Candidato,
    CandidatoResumo,
    Cargo,
    Eleicao,
    PrestaContas,
)

logger = logging.getLogger(__name__)

_rate_limiter = RateLimiter(max_requests=30, period=60.0)


async def _get(url: str, params: dict[str, Any] | None = None) -> Any:
    """GET request with rate limiting for TSE API."""
    async with _rate_limiter:
        return await http_get(url, params=params)


def _safe_list(data: Any, endpoint: str) -> list[dict[str, Any]]:
    """Ensure data is a list."""
    if isinstance(data, list):
        return data
    logger.warning("Resposta inesperada (esperava list) do endpoint %s", endpoint)
    return []


def _safe_int(val: Any) -> int | None:
    """Safely convert to int."""
    if val is None:
        return None
    try:
        return int(val)
    except (ValueError, TypeError):
        return None


def _safe_float(val: Any) -> float | None:
    """Safely convert to float."""
    if val is None:
        return None
    try:
        return float(val)
    except (ValueError, TypeError):
        return None


# --- Parsing helpers ---


def _parse_eleicao(raw: dict[str, Any]) -> Eleicao:
    return Eleicao(
        id=_safe_int(raw.get("id")),
        sigla_uf=raw.get("siglaUF"),
        ano=_safe_int(raw.get("ano")),
        codigo=raw.get("codigo"),
        nome=raw.get("nomeEleicao"),
        tipo=raw.get("tipoEleicao"),
        turno=raw.get("turno"),
        tipo_abrangencia=raw.get("tipoAbrangencia"),
        data_eleicao=raw.get("dataEleicao"),
        descricao=raw.get("descricaoEleicao"),
    )


def _parse_cargo(raw: dict[str, Any]) -> Cargo:
    return Cargo(
        codigo=_safe_int(raw.get("codigo")),
        sigla=raw.get("sigla"),
        nome=raw.get("nome"),
        titular=raw.get("titular"),
        contagem=_safe_int(raw.get("contagem")),
    )


def _parse_candidato_resumo(raw: dict[str, Any]) -> CandidatoResumo:
    return CandidatoResumo(
        id=_safe_int(raw.get("id")),
        nome_urna=raw.get("nomeUrna"),
        numero=_safe_int(raw.get("numero")),
        partido=(
            raw.get("partido", {}).get("sigla")
            if isinstance(raw.get("partido"), dict)
            else raw.get("partido")
        ),
        situacao=raw.get("descricaoSituacao"),
        foto_url=raw.get("fotoUrl"),
    )


def _parse_candidato(raw: dict[str, Any]) -> Candidato:
    partido_raw = raw.get("partido", {})
    partido = partido_raw.get("sigla") if isinstance(partido_raw, dict) else None
    emails = raw.get("emails", [])
    sites = raw.get("sites", [])

    return Candidato(
        id=_safe_int(raw.get("id")),
        nome_urna=raw.get("nomeUrna"),
        nome_completo=raw.get("nomeCompleto"),
        numero=_safe_int(raw.get("numero")),
        cpf=raw.get("cpf"),
        data_nascimento=raw.get("dataDeNascimento"),
        sexo=raw.get("descricaoSexo"),
        estado_civil=raw.get("descricaoEstadoCivil"),
        cor_raca=raw.get("descricaoCorRaca"),
        nacionalidade=raw.get("nacionalidade"),
        grau_instrucao=raw.get("grauInstrucao"),
        ocupacao=raw.get("ocupacao"),
        uf_nascimento=raw.get("sgUfNascimento"),
        municipio_nascimento=raw.get("nomeMunicipioNascimento"),
        partido=partido,
        situacao=raw.get("descricaoSituacao"),
        situacao_candidato=raw.get("descricaoSituacaoCandidato"),
        coligacao=raw.get("nomeColigacao"),
        composicao_coligacao=raw.get("composicaoColigacao"),
        gasto_campanha=_safe_float(raw.get("gastoCampanha")),
        total_bens=_safe_float(raw.get("totalDeBens")),
        emails=emails if isinstance(emails, list) else [],
        sites=sites if isinstance(sites, list) else [],
        foto_url=raw.get("fotoUrl"),
        candidato_inapto=raw.get("isCandidatoInapto"),
        motivo_ficha_limpa=raw.get("st_MOTIVO_FICHA_LIMPA"),
    )


def _parse_presta_contas(raw: dict[str, Any]) -> PrestaContas:
    consolidados = raw.get("dadosConsolidados", {}) or {}
    despesas = raw.get("despesas", {}) or {}

    return PrestaContas(
        candidato_id=raw.get("idCandidato"),
        nome=raw.get("nomeCandidato"),
        partido=raw.get("siglaPartido"),
        cnpj=raw.get("cnpj"),
        total_recebido=_safe_float(consolidados.get("totalRecebido")),
        total_despesas=_safe_float(despesas.get("totalDespesasPagas")),
        total_bens=_safe_float(raw.get("totalDeBens")),
        limite_gastos=_safe_float(despesas.get("valorLimiteDeGastos")),
        divida_campanha=raw.get("dividaCampanha"),
        sobra_financeira=raw.get("sobraFinanceira"),
        total_receita_pf=_safe_float(consolidados.get("totalReceitaPF")),
        total_receita_pj=_safe_float(consolidados.get("totalReceitaPJ")),
        total_fundo_partidario=_safe_float(consolidados.get("totalPartidos")),
        total_fundo_especial=_safe_float(consolidados.get("totalDoacaoFcc")),
    )


# --- Public API functions ---


async def anos_eleitorais() -> list[int]:
    """List available electoral years."""
    data = await _get(f"{ELEICAO_URL}/anos-eleitorais")
    if isinstance(data, list):
        return [int(a) for a in data if a is not None]
    return []


async def listar_eleicoes() -> list[Eleicao]:
    """List ordinary elections."""
    data = await _get(f"{ELEICAO_URL}/ordinarias")
    return [_parse_eleicao(e) for e in _safe_list(data, "eleicoes")]


async def listar_eleicoes_suplementares(ano: int, uf: str) -> list[Eleicao]:
    """List supplementary elections for a year and state."""
    data = await _get(f"{ELEICAO_URL}/suplementares/{ano}/{uf.upper()}")
    return [_parse_eleicao(e) for e in _safe_list(data, "eleicoes_suplementares")]


async def listar_cargos(eleicao_id: int, municipio: int) -> list[Cargo]:
    """List positions available in a municipality for an election."""
    url = f"{ELEICAO_URL}/listar/municipios/{eleicao_id}/{municipio}/cargos"
    data = await _get(url)
    if isinstance(data, dict):
        cargos_raw = data.get("cargos", [])
        return [_parse_cargo(c) for c in _safe_list(cargos_raw, "cargos")]
    return []


async def listar_candidatos(
    ano: int,
    municipio: int,
    eleicao_id: int,
    cargo: int,
) -> list[CandidatoResumo]:
    """List candidates for a specific position in a municipality."""
    url = f"{CANDIDATURA_URL}/listar/{ano}/{municipio}/{eleicao_id}/{cargo}/candidatos"
    data = await _get(url)
    if isinstance(data, dict):
        cands_raw = data.get("candidatos", [])
        return [_parse_candidato_resumo(c) for c in _safe_list(cands_raw, "candidatos")]
    return []


async def buscar_candidato(
    ano: int,
    municipio: int,
    eleicao_id: int,
    candidato_id: int,
) -> Candidato | None:
    """Get full details for a candidate."""
    url = f"{CANDIDATURA_URL}/buscar/{ano}/{municipio}/{eleicao_id}/candidato/{candidato_id}"
    data = await _get(url)
    if isinstance(data, dict) and data.get("id"):
        return _parse_candidato(data)
    return None


async def consultar_prestacao_contas(
    eleicao_id: int,
    ano: int,
    municipio: int,
    cargo: int,
    candidato_id: int,
) -> PrestaContas | None:
    """Get campaign account information for a candidate."""
    url = f"{PRESTADOR_URL}/consulta/{eleicao_id}/{ano}/{municipio}/{cargo}/90/90/{candidato_id}"
    data = await _get(url)
    if isinstance(data, dict):
        return _parse_presta_contas(data)
    return None
