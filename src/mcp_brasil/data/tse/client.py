"""HTTP client for the TSE (DivulgaCandContas) API.

REST API without authentication.
API docs: Swagger at divulgacandcontas.tse.jus.br (unofficial).
"""

from __future__ import annotations

import logging
from typing import Any

from mcp_brasil._shared.http_client import http_get
from mcp_brasil._shared.rate_limiter import RateLimiter

from .constants import (
    CANDIDATURA_URL,
    CARGO_CODES_CDN,
    ELEICAO_URL,
    ELEICOES_CDN,
    PRESTADOR_URL,
    RESULTADOS_CDN_BASE,
    UFS_BRASIL,
)
from .schemas import (
    Candidato,
    CandidatoResumo,
    Cargo,
    Eleicao,
    MunicipioEleitoral,
    PrestaContas,
    ResultadoCandidato,
    ResultadoCDN,
    ResultadoRegiao,
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
        descricao_totalizacao=raw.get("descricaoTotalizacao"),
        total_votos=_safe_int(raw.get("totalVotos")),
        gasto_campanha=_safe_float(raw.get("gastoCampanha")),
        total_bens=_safe_float(raw.get("totalDeBens")),
        emails=emails if isinstance(emails, list) else [],
        sites=sites if isinstance(sites, list) else [],
        foto_url=raw.get("fotoUrl"),
        candidato_inapto=raw.get("isCandidatoInapto"),
        motivo_ficha_limpa=raw.get("st_MOTIVO_FICHA_LIMPA"),
    )


def _parse_resultado_candidato(raw: dict[str, Any]) -> ResultadoCandidato:
    partido_raw = raw.get("partido", {})
    partido = partido_raw.get("sigla") if isinstance(partido_raw, dict) else partido_raw

    return ResultadoCandidato(
        nome_urna=raw.get("nomeUrna"),
        numero=_safe_int(raw.get("numero")),
        partido=partido if isinstance(partido, str) else None,
        total_votos=_safe_int(raw.get("totalVotos")),
        percentual=raw.get("percentual"),
        descricao_totalizacao=raw.get("descricaoTotalizacao"),
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


async def listar_estados_suplementares(ano: int) -> list[str]:
    """List states with supplementary elections in a given year."""
    data = await _get(f"{ELEICAO_URL}/estados/{ano}/ano")
    if isinstance(data, list):
        return [str(e.get("uf", "")) for e in data if isinstance(e, dict) and e.get("uf")]
    return []


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


async def resultado_eleicao(
    ano: int,
    municipio: int,
    eleicao_id: int,
    cargo: int,
) -> list[ResultadoCandidato]:
    """Get election results ranked by votes for a position in a municipality."""
    url = f"{CANDIDATURA_URL}/listar/{ano}/{municipio}/{eleicao_id}/{cargo}/candidatos"
    data = await _get(url)
    if isinstance(data, dict):
        cands_raw = data.get("candidatos", [])
        resultados = [_parse_resultado_candidato(c) for c in _safe_list(cands_raw, "resultado")]
        return sorted(resultados, key=lambda r: r.total_votos or 0, reverse=True)
    return []


# --- CDN de Resultados (resultados.tse.jus.br) ---


def _resolve_eleicao(ano: int, turno: int = 1) -> tuple[str, str, str]:
    """Resolve ano+turno em ciclo+padded+unpadded.

    Returns:
        (ciclo, padded, unpadded) ex: ("ele2022", "000544", "544")
        - padded: usado em filenames (e000544)
        - unpadded: usado em paths do CDN (/544/)

    Raises:
        ValueError: se a eleição não está mapeada.
    """
    key = (ano, turno)
    if key not in ELEICOES_CDN:
        disponiveis = ", ".join(f"{a} T{t}" for a, t in sorted(ELEICOES_CDN.keys()))
        raise ValueError(f"Eleição {ano} turno {turno} não mapeada. Disponíveis: {disponiveis}")
    return ELEICOES_CDN[key]


def _resolve_cargo(cargo: str) -> str:
    """Resolve nome do cargo em código CDN.

    Args:
        cargo: Nome do cargo (ex: "presidente", "governador").

    Returns:
        Código CDN do cargo (ex: "0001").

    Raises:
        ValueError: se o cargo não está mapeado.
    """
    cargo_lower = cargo.lower().strip().replace(" ", "_")
    if cargo_lower not in CARGO_CODES_CDN:
        disponiveis = ", ".join(CARGO_CODES_CDN.keys())
        raise ValueError(f"Cargo '{cargo}' não mapeado. Disponíveis: {disponiveis}")
    return CARGO_CODES_CDN[cargo_lower]


def _parse_resultado_cdn(raw: dict[str, Any]) -> ResultadoCDN:
    """Parse a candidate result from CDN JSON."""
    return ResultadoCDN(
        sequencia=raw.get("seq"),
        nome=raw.get("nm"),
        numero=raw.get("n"),
        nome_vice=raw.get("nv"),
        coligacao=raw.get("cc"),
        votos=_safe_int(raw.get("vap")),
        percentual=raw.get("pvap"),
        eleito=raw.get("e") == "s",
        situacao=raw.get("st"),
        validade_voto=raw.get("dvt"),
    )


def _parse_resultado_regiao(data: dict[str, Any]) -> ResultadoRegiao:
    """Parse region-level result from CDN JSON."""
    cands = [_parse_resultado_cdn(c) for c in data.get("cand", [])]
    cands.sort(key=lambda c: c.votos or 0, reverse=True)

    return ResultadoRegiao(
        codigo=data.get("cdabr"),
        tipo=data.get("tpabr"),
        uf=data.get("cdabr"),
        data_eleicao=data.get("dt"),
        total_secoes=_safe_int(data.get("s")),
        pct_apurado=data.get("pst"),
        total_eleitores=_safe_int(data.get("e")),
        total_comparecimento=_safe_int(data.get("c")),
        total_abstencoes=_safe_int(data.get("a")),
        candidatos=cands,
    )


async def resultado_simplificado(
    ano: int, cargo: str, uf: str = "br", turno: int = 1
) -> ResultadoRegiao | None:
    """Busca resultado simplificado de um cargo em uma região.

    Args:
        ano: Ano da eleição (ex: 2022, 2024).
        cargo: Nome do cargo (ex: "presidente", "governador", "prefeito").
        uf: Sigla da UF ou "br" para nacional.
        turno: Turno da eleição (1 ou 2).

    Returns:
        ResultadoRegiao com candidatos rankeados por votos, ou None se 404.
    """
    ciclo, padded, unpadded = _resolve_eleicao(ano, turno)
    cargo_code = _resolve_cargo(cargo)
    uf_lower = uf.lower().strip()

    url = (
        f"{RESULTADOS_CDN_BASE}/{ciclo}/{unpadded}/"
        f"dados-simplificados/{uf_lower}/{uf_lower}-c{cargo_code}-e{padded}-r.json"
    )

    try:
        data = await _get(url)
    except Exception:
        logger.warning("CDN resultado indisponível: %s", url)
        return None

    if not isinstance(data, dict) or "cand" not in data:
        return None

    return _parse_resultado_regiao(data)


async def resultado_todos_estados(ano: int, cargo: str, turno: int = 1) -> list[ResultadoRegiao]:
    """Busca resultado de um cargo em todos os estados (27 UFs).

    Faz requests paralelos para cada UF.

    Args:
        ano: Ano da eleição.
        cargo: Nome do cargo.
        turno: Turno.

    Returns:
        Lista de ResultadoRegiao, um por UF (ignora falhas silenciosamente).
    """
    import asyncio

    tasks = [resultado_simplificado(ano, cargo, uf, turno) for uf in UFS_BRASIL]
    results = await asyncio.gather(*tasks)
    return [r for r in results if r is not None]


async def listar_municipios_eleitorais(
    ano: int, uf: str, turno: int = 1
) -> list[MunicipioEleitoral]:
    """Lista municípios eleitorais de uma UF com códigos TSE e IBGE.

    Args:
        ano: Ano da eleição (ex: 2024).
        uf: Sigla da UF (ex: "SP").
        turno: Turno da eleição (1 ou 2).

    Returns:
        Lista de MunicipioEleitoral com códigos TSE e IBGE.
    """
    ciclo, padded, unpadded = _resolve_eleicao(ano, turno)
    url = f"{RESULTADOS_CDN_BASE}/{ciclo}/{unpadded}/config/mun-e{padded}-cm.json"

    try:
        data = await _get(url)
    except Exception:
        logger.warning("CDN config municípios indisponível: %s", url)
        return []

    if not isinstance(data, dict):
        return []

    uf_upper = uf.upper().strip()
    municipios: list[MunicipioEleitoral] = []

    # Format: {"abr": [{"cd": "SP", "mu": [{"cd": "71072", ...}]}]}
    for estado in data.get("abr", []):
        if estado.get("cd", "").upper() != uf_upper:
            continue
        for mun in estado.get("mu", []):
            municipios.append(
                MunicipioEleitoral(
                    codigo_tse=mun.get("cd"),
                    codigo_ibge=mun.get("cdi"),
                    nome=mun.get("nm"),
                    capital=mun.get("c", "N") == "S",
                    uf=uf_upper,
                )
            )
        break

    return municipios


def _parse_resultado_unificado(data: dict[str, Any]) -> ResultadoRegiao:
    """Parse do formato unificado (-u.json) do CDN de resultados.

    Formato: {"carg": [{"agr": [{"par": [{"cand": [...]}]}]}]}
    Stats de seções em s:{ts, pst}, eleitores em e:{te, c, a}.
    """
    candidatos: list[ResultadoCDN] = []

    for cargo in data.get("carg", []):
        for agr in cargo.get("agr", []):
            for partido in agr.get("par", []):
                for cand in partido.get("cand", []):
                    candidatos.append(_parse_resultado_cdn(cand))

    candidatos.sort(key=lambda c: c.votos or 0, reverse=True)

    # Stats
    secoes = data.get("s", {})
    eleitores = data.get("e", {})

    return ResultadoRegiao(
        codigo=data.get("cdabr"),
        tipo=data.get("tpabr"),
        uf=data.get("cdabr"),
        data_eleicao=data.get("dt"),
        total_secoes=_safe_int(secoes.get("ts") if isinstance(secoes, dict) else secoes),
        pct_apurado=(secoes.get("pst") if isinstance(secoes, dict) else None),
        total_eleitores=_safe_int(
            eleitores.get("te") if isinstance(eleitores, dict) else eleitores
        ),
        total_comparecimento=_safe_int(
            eleitores.get("c") if isinstance(eleitores, dict) else None
        ),
        total_abstencoes=_safe_int(eleitores.get("a") if isinstance(eleitores, dict) else None),
        candidatos=candidatos,
    )


async def resultado_municipio(
    ano: int, cargo: str, uf: str, cod_tse: str, turno: int = 1
) -> ResultadoRegiao | None:
    """Busca resultado de um cargo em um município específico.

    Disponível apenas para eleições municipais (2024).

    Args:
        ano: Ano da eleição (ex: 2024).
        cargo: Nome do cargo (ex: "prefeito", "vereador").
        uf: Sigla da UF (ex: "SP").
        cod_tse: Código TSE do município (5 dígitos, ex: "71072").
        turno: Turno da eleição (1 ou 2).

    Returns:
        ResultadoRegiao com candidatos rankeados por votos, ou None se 404.
    """
    ciclo, padded, unpadded = _resolve_eleicao(ano, turno)
    cargo_code = _resolve_cargo(cargo)
    uf_lower = uf.lower().strip()

    url = (
        f"{RESULTADOS_CDN_BASE}/{ciclo}/{unpadded}/"
        f"dados/{uf_lower}/{uf_lower}{cod_tse}-c{cargo_code}-e{padded}-u.json"
    )

    try:
        data = await _get(url)
    except Exception:
        logger.warning("CDN resultado município indisponível: %s", url)
        return None

    if not isinstance(data, dict) or "carg" not in data:
        return None

    return _parse_resultado_unificado(data)


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
