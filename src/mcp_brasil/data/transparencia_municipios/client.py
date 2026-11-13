"""Async client for Brazilian municipal and state transparency.

Prioritizes the native high-speed Go binary (bin/transparencia-go) for
sub-millisecond execution, with a robust pure-Python in-memory fallback.
"""

from __future__ import annotations

import asyncio
import csv
import json
import logging
import re
import unicodedata
from typing import Any

from .constants import (
    ESTADOS_CSV_PATH,
    GO_BINARY_PATH,
    MEGASOFT_CSV_PATH,
    MUNICIPIOS_CSV_PATH,
)
from .schemas import (
    ComparativoItem,
    ComparativoRelatorio,
    ConcursosResult,
    ContratosResult,
    ConveniosResult,
    DespesasResult,
    DiariasResult,
    EntregaItem,
    Estado,
    EstadoDossier,
    FiscalItem,
    LegislacaoResult,
    LicitacoesResult,
    MegasoftMunicipio,
    Municipio,
    MunicipioDossier,
    ObrasResult,
    PortalDiscoveryResult,
    ProbeResult,
    ReceitasResult,
    SearchResult,
    ServidorDetalheResult,
    ServidoresResult,
    Stats,
    TransparenciaCompletaReport,
)

logger = logging.getLogger(__name__)


async def _run_go_binary(cmd: list[str]) -> dict[str, Any] | list[Any]:
    """Execute the native Go binary with JSON output."""
    if not GO_BINARY_PATH.exists():
        raise FileNotFoundError(f"Go binary not found at {GO_BINARY_PATH}")

    proc = await asyncio.create_subprocess_exec(
        str(GO_BINARY_PATH),
        *cmd,
        "--json",
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE,
    )
    stdout, stderr = await proc.communicate()
    if proc.returncode != 0:
        err_msg = stderr.decode().strip() or f"Exited with code {proc.returncode}"
        raise RuntimeError(f"transparencia-go error: {err_msg}")

    return json.loads(stdout.decode())  # type: ignore[no-any-return]


# ---------------------------------------------------------------------------
# Python Fallback Cache & Normalizer (used if Go binary is absent)
# ---------------------------------------------------------------------------
_PY_ESTADOS: list[Estado] = []
_PY_MUNICIPIOS: list[Municipio] = []
_PY_BY_IBGE: dict[str, Municipio] = {}
_PY_BY_SIAFI: dict[str, Municipio] = {}
_PY_BY_CNPJ: dict[str, Municipio] = {}
_PY_BY_UF: dict[str, list[Municipio]] = {}
_PY_ESTADOS_BY_UF: dict[str, Estado] = {}


def _remove_accents(s: str) -> str:
    nfkd = unicodedata.normalize("NFKD", s)
    return "".join(c for c in nfkd if not unicodedata.combining(c)).lower()


def _slugify(s: str) -> str:
    clean = _remove_accents(s)
    clean = re.sub(r"[^a-z0-9]+", "-", clean).strip("-")
    return clean


def _clean_digits(s: str) -> str:
    return "".join(c for c in s if c.isdigit())


def _load_python_cache() -> None:
    global _PY_ESTADOS, _PY_MUNICIPIOS, _PY_BY_IBGE, _PY_BY_SIAFI
    global _PY_BY_CNPJ, _PY_BY_UF, _PY_ESTADOS_BY_UF
    if _PY_ESTADOS:
        return

    # Load Estados
    if ESTADOS_CSV_PATH.exists():
        with open(ESTADOS_CSV_PATH, encoding="utf-8-sig") as f:
            for row in csv.DictReader(f):
                uf = row["uf"].upper().strip()
                code_uf = int(row["codigo_uf"])
                est = Estado(
                    codigo_uf=code_uf,
                    uf=uf,
                    nome=row["nome"].strip(),
                    latitude=float(row["latitude"]),
                    longitude=float(row["longitude"]),
                    regiao=row["regiao"].strip(),
                    capital="",
                    total_municipios=0,
                    portal_transparencia=f"https://transparencia.{uf.lower()}.gov.br",
                    portal_governo=f"https://www.{uf.lower()}.gov.br",
                    tribunal_contas_nome=f"Tribunal de Contas do Estado ({uf})",
                    tribunal_contas_url=f"https://www.tce.{uf.lower()}.gov.br",
                    assembleia_nome=f"Assembleia Legislativa de {row['nome']}",
                    assembleia_url=f"https://www.al.{uf.lower()}.leg.br",
                    siconfi_ente_id=str(code_uf),
                    radar_transparencia_url="https://radardatransparencia.atricon.org.br/radar-da-transparencia-publica.html",
                    pncp_url=f"https://pncp.gov.br/app/editais?uf={uf}",
                )
                _PY_ESTADOS.append(est)
                _PY_ESTADOS_BY_UF[uf] = est

    # Load Municipios
    if MUNICIPIOS_CSV_PATH.exists():
        with open(MUNICIPIOS_CSV_PATH, encoding="utf-8-sig") as f:
            for row in csv.DictReader(f):
                code_uf = int(row["codigo_uf"])
                est_match = next((e for e in _PY_ESTADOS if e.codigo_uf == code_uf), None)
                uf = est_match.uf if est_match else ""
                regiao = est_match.regiao if est_match else ""
                nome = row["nome"].strip()
                cnpj = _clean_digits(row.get("cnpj", ""))
                pop_val = row.get("populacao", "0")
                populacao = int(pop_val) if pop_val and pop_val.isdigit() else 0

                m = Municipio(
                    codigo_ibge=row["codigo_ibge"].strip(),
                    nome=nome,
                    latitude=float(row["latitude"]),
                    longitude=float(row["longitude"]),
                    capital=row["capital"].strip() == "1",
                    codigo_uf=code_uf,
                    uf=uf,
                    regiao=regiao,
                    siafi_id=row["siafi_id"].strip(),
                    ddd=row["ddd"].strip(),
                    fuso_horario=row["fuso_horario"].strip(),
                    slug=_slugify(nome),
                    cnpj=cnpj,
                    populacao=populacao,
                )
                _PY_MUNICIPIOS.append(m)
                _PY_BY_IBGE[m.codigo_ibge] = m
                if m.siafi_id:
                    _PY_BY_SIAFI[m.siafi_id] = m
                if cnpj:
                    _PY_BY_CNPJ[cnpj] = m
                _PY_BY_UF.setdefault(uf, []).append(m)

        for est in _PY_ESTADOS:
            munis = _PY_BY_UF.get(est.uf, [])
            est.total_municipios = len(munis)
            cap = next((m.nome for m in munis if m.capital), "")
            est.capital = cap


# ---------------------------------------------------------------------------
# API Functions
# ---------------------------------------------------------------------------


async def listar_estados() -> list[Estado]:
    """Lista todas as 27 Unidades Federativas do Brasil com seus portais de transparência."""
    if GO_BINARY_PATH.exists():
        try:
            data = await _run_go_binary(["list-ufs"])
            return [Estado(**item) for item in data]  # type: ignore[arg-type]
        except Exception as e:
            logger.warning("Falha no binário Go, usando fallback Python: %s", e)

    _load_python_cache()
    return _PY_ESTADOS


async def buscar_municipios(
    query: str = "",
    uf: str | None = None,
    limit: int = 20,
) -> SearchResult:
    """Busca rápida em todos os 5.570 municípios (Go engine priorizado)."""
    if GO_BINARY_PATH.exists():
        try:
            args = ["search", query, "--limit", str(limit)]
            if uf:
                args.extend(["--uf", uf.upper().strip()])
            data = await _run_go_binary(args)
            return SearchResult(**data)  # type: ignore[arg-type]
        except Exception as e:
            logger.warning("Falha no binário Go, usando fallback Python: %s", e)

    _load_python_cache()
    q_norm = _remove_accents(query.strip())
    uf_upper = uf.upper().strip() if uf else ""

    matched: list[Municipio] = []
    for m in _PY_MUNICIPIOS:
        if uf_upper and m.uf != uf_upper:
            continue
        match_found = (
            not q_norm
            or q_norm in _remove_accents(m.nome)
            or q_norm == m.codigo_ibge
            or q_norm == m.siafi_id
        )
        if match_found:
            matched.append(m)

        if len(matched) >= limit:
            break

    return SearchResult(
        query=query,
        uf=uf or "",
        total_encontrado=len(matched),
        municipios=matched,
    )


async def obter_municipio(codigo_ibge_ou_siafi: str) -> Municipio | None:
    """Obtém detalhes do município pelo código IBGE ou SIAFI."""
    code = codigo_ibge_ou_siafi.strip()
    if GO_BINARY_PATH.exists():
        try:
            data = await _run_go_binary(["get", code])
            return Municipio(**data)  # type: ignore[arg-type]
        except Exception:
            pass

    _load_python_cache()
    if len(code) == 7 and code in _PY_BY_IBGE:
        return _PY_BY_IBGE[code]
    if code in _PY_BY_SIAFI:
        return _PY_BY_SIAFI[code]
    return None


async def conectar_municipio(
    municipio: str,
    uf: str | None = None,
) -> MunicipioDossier:
    """Conecta a um município e retorna seu dossiê completo de transparência."""
    if GO_BINARY_PATH.exists():
        try:
            args = ["connect-municipio", municipio]
            if uf:
                args.extend(["--uf", uf.upper().strip()])
            data = await _run_go_binary(args)
            return MunicipioDossier(**data)  # type: ignore[arg-type]
        except Exception as e:
            logger.warning("Falha no binário Go, usando fallback Python: %s", e)

    _load_python_cache()
    search = await buscar_municipios(query=municipio, uf=uf, limit=1)
    if not search.municipios:
        raise ValueError(f"Município não encontrado: {municipio}")

    m = search.municipios[0]
    est = _PY_ESTADOS_BY_UF.get(m.uf)
    slug = m.slug
    uf_lower = m.uf.lower()

    tcm_or_tce = f"Tribunal de Contas do Estado ({m.uf})"
    tcm_url = f"https://www.tce.{uf_lower}.gov.br"
    if m.codigo_ibge == "3550308":
        tcm_or_tce = "Tribunal de Contas do Município de São Paulo (TCM-SP)"
        tcm_url = "https://www.tcm.sp.gov.br"
    elif m.codigo_ibge == "3304557":
        tcm_or_tce = "Tribunal de Contas do Município do Rio de Janeiro (TCM-Rio)"
        tcm_url = "https://www.tcm.rj.gov.br"

    pncp_cnpj = f"https://pncp.gov.br/app/editais?q={m.cnpj}&uf={m.uf}" if m.cnpj else ""
    dca_ep = f"https://apidatalake.tesouro.gov.br/ords/siconfi/tt/dca?an_exercicio=2023&id_ente={m.codigo_ibge}"
    rreo_ep = f"https://apidatalake.tesouro.gov.br/ords/siconfi/tt/rreo?an_exercicio=2024&nr_periodo=1&co_tipo_demonstrativo=RREO&id_ente={m.codigo_ibge}"
    entregas_ep = f"https://apidatalake.tesouro.gov.br/ords/siconfi/tt/extrato_entregas?an_referencia=2024&id_ente={m.codigo_ibge}"

    return MunicipioDossier(
        municipio=m,
        estado=est,
        cnpj=m.cnpj,
        populacao=m.populacao,
        portal_transparencia_prefeitura=f"https://transparencia.{slug}.{uf_lower}.gov.br",
        portal_prefeitura=f"https://www.{slug}.{uf_lower}.gov.br",
        portal_camara=f"https://camara{slug}.{uf_lower}.leg.br",
        tribunal_contas_competente=tcm_or_tce,
        tribunal_contas_url=tcm_url,
        radar_transparencia_url="https://radardatransparencia.atricon.org.br/radar-da-transparencia-publica.html",
        siconfi_url=f"https://siconfi.tesouro.gov.br/siconfi/index.jsf?cod_ibge={m.codigo_ibge}",
        siconfi_api=f"https://apidatalake.tesouro.gov.br/ords/siconfi/tt/entes?cod_ibge={m.codigo_ibge}",
        pncp_url=f"https://pncp.gov.br/app/editais?uf={m.uf}",
        pncp_cnpj_url=pncp_cnpj,
        transferegov_url=f"https://transferegov.sistema.gov.br/portal/painel-repasses?uf={m.uf}",
        transparencia_federal_url=f"https://portaldatransparencia.gov.br/localidades/{m.codigo_ibge}-{slug}",
        siconfi_dca_endpoint=dca_ep,
        siconfi_rreo_endpoint=rreo_ep,
        siconfi_entregas_endpoint=entregas_ep,
    )


async def conectar_estado(uf: str) -> EstadoDossier:
    """Conecta a uma UF e retorna seu dossiê completo de transparência."""
    uf_clean = uf.upper().strip()
    if GO_BINARY_PATH.exists():
        try:
            data = await _run_go_binary(["connect-estado", uf_clean])
            return EstadoDossier(**data)  # type: ignore[arg-type]
        except Exception as e:
            logger.warning("Falha no binário Go, usando fallback Python: %s", e)

    _load_python_cache()
    est = _PY_ESTADOS_BY_UF.get(uf_clean)
    if not est:
        raise ValueError(f"Estado/UF não encontrado: {uf}")

    munis = _PY_BY_UF.get(uf_clean, [])
    principais = [m for m in munis if m.capital] or munis[:5]

    return EstadoDossier(
        estado=est,
        total_municipios=len(munis),
        municipios_principais=principais,
        portal_transparencia=est.portal_transparencia,
        portal_governo=est.portal_governo,
        tribunal_contas_nome=est.tribunal_contas_nome,
        tribunal_contas_url=est.tribunal_contas_url,
        assembleia_nome=est.assembleia_nome,
        assembleia_url=est.assembleia_url,
        radar_transparencia_url=est.radar_transparencia_url,
        siconfi_url=f"https://siconfi.tesouro.gov.br/siconfi/index.jsf?cod_ibge={est.codigo_uf}",
        siconfi_api=f"https://apidatalake.tesouro.gov.br/ords/siconfi/tt/entes?cod_ibge={est.codigo_uf}",
        pncp_url=est.pncp_url,
    )


async def obter_estatisticas() -> Stats:
    """Retorna estatísticas consolidadas nacionais sobre municípios e UFs."""
    if GO_BINARY_PATH.exists():
        try:
            data = await _run_go_binary(["stats"])
            return Stats(**data)  # type: ignore[arg-type]
        except Exception as e:
            logger.warning("Falha no binário Go, usando fallback Python: %s", e)

    _load_python_cache()
    por_regiao: dict[str, int] = {}
    por_uf: dict[str, int] = {}
    capitais: list[str] = []
    pop_total = 0

    for m in _PY_MUNICIPIOS:
        por_regiao[m.regiao] = por_regiao.get(m.regiao, 0) + 1
        por_uf[m.uf] = por_uf.get(m.uf, 0) + 1
        pop_total += m.populacao
        if m.capital:
            capitais.append(f"{m.nome} ({m.uf})")

    capitais.sort()
    return Stats(
        total_municipios=len(_PY_MUNICIPIOS),
        total_ufs=len(_PY_ESTADOS),
        populacao_total=pop_total,
        municipios_por_regiao=por_regiao,
        municipios_por_uf=por_uf,
        capitais=capitais,
    )


async def comparar_transparencia(alvos: list[str]) -> ComparativoRelatorio:
    """Compara estrutura de transparência entre múltiplos municípios e/ou estados."""
    if GO_BINARY_PATH.exists():
        try:
            data = await _run_go_binary(["compare", *alvos])
            return ComparativoRelatorio(**data)  # type: ignore[arg-type]
        except Exception as e:
            logger.warning("Falha no binário Go, usando fallback Python: %s", e)

    itens: list[ComparativoItem] = []
    for alvo in alvos:
        alvo_trim = alvo.strip()
        if len(alvo_trim) == 2:
            try:
                est_dos = await conectar_estado(alvo_trim)
                est = est_dos.estado
                itens.append(
                    ComparativoItem(
                        tipo="estado",
                        nome=est.nome,
                        uf=est.uf,
                        codigo_ibge=str(est.codigo_uf),
                        regiao=est.regiao,
                        capital=False,
                        portal_transparencia=est.portal_transparencia,
                        tribunal_contas_nome=est.tribunal_contas_nome,
                        tribunal_contas_url=est.tribunal_contas_url,
                        radar_transparencia_url=est.radar_transparencia_url,
                        siconfi_url=est_dos.siconfi_url,
                        pncp_url=est.pncp_url,
                    )
                )
                continue
            except Exception:
                pass

        try:
            muni_dos = await conectar_municipio(alvo_trim)
            m = muni_dos.municipio
            itens.append(
                ComparativoItem(
                    tipo="municipio",
                    nome=m.nome,
                    uf=m.uf,
                    codigo_ibge=m.codigo_ibge,
                    siafi_id=m.siafi_id,
                    cnpj=m.cnpj,
                    populacao=m.populacao,
                    regiao=m.regiao,
                    capital=m.capital,
                    portal_transparencia=muni_dos.portal_transparencia_prefeitura,
                    tribunal_contas_nome=muni_dos.tribunal_contas_competente,
                    tribunal_contas_url=muni_dos.tribunal_contas_url,
                    radar_transparencia_url=muni_dos.radar_transparencia_url,
                    siconfi_url=muni_dos.siconfi_url,
                    pncp_url=muni_dos.pncp_url,
                )
            )
        except Exception:
            pass

    return ComparativoRelatorio(total_itens=len(itens), itens=itens)


async def buscar_por_cnpj(cnpj: str) -> MunicipioDossier:
    """Busca município e transparência diretamente pelo CNPJ oficial da Prefeitura."""
    clean = _clean_digits(cnpj)
    if GO_BINARY_PATH.exists():
        try:
            data = await _run_go_binary(["cnpj", clean])
            return MunicipioDossier(**data)  # type: ignore[arg-type]
        except Exception as e:
            logger.warning("Falha no binário Go para CNPJ, usando fallback Python: %s", e)

    _load_python_cache()
    m = _PY_BY_CNPJ.get(clean)
    if not m:
        raise ValueError(f"Nenhum município encontrado para o CNPJ: {cnpj}")

    return await conectar_municipio(municipio=m.codigo_ibge, uf=m.uf)


async def consultar_fiscal(
    cod_ibge: str,
    ano: int = 0,
    tipo: str = "dca",
    periodo: int = 1,
) -> list[FiscalItem]:
    """Consulta dados fiscais oficiais no SICONFI/Tesouro Nacional (DCA, RREO ou RGF)."""
    if GO_BINARY_PATH.exists():
        try:
            args = [
                "fiscal",
                "--ibge",
                cod_ibge.strip(),
                "--tipo",
                tipo.lower().strip(),
            ]
            if ano > 0:
                args.extend(["--ano", str(ano)])
            if periodo > 0:
                args.extend(["--periodo", str(periodo)])
            data = await _run_go_binary(args)
            return [FiscalItem(**item) for item in data]  # type: ignore[arg-type]
        except Exception as e:
            logger.warning("Falha no binário Go para consulta fiscal, usando fallback: %s", e)

    # Fallback via HTTP direto
    from mcp_brasil._shared.http_client import http_get

    tipo_low = tipo.lower()
    if tipo_low == "rreo":
        p_ano = ano if ano > 0 else 2024
        p_periodo = periodo if periodo > 0 else 1
        url = (
            f"https://apidatalake.tesouro.gov.br/ords/siconfi/tt/rreo?"
            f"an_exercicio={p_ano}&nr_periodo={p_periodo}&co_tipo_demonstrativo=RREO&id_ente={cod_ibge}"
        )
    elif tipo_low == "rgf":
        p_ano = ano if ano > 0 else 2024
        p_periodo = periodo if periodo > 0 else 2
        url = (
            f"https://apidatalake.tesouro.gov.br/ords/siconfi/tt/rgf?"
            f"an_exercicio={p_ano}&in_periodicidade=S&nr_periodo={p_periodo}"
            f"&co_tipo_demonstrativo=RGF&co_poder=E&id_ente={cod_ibge}"
        )
    else:
        p_ano = ano if ano > 0 else 2023
        url = f"https://apidatalake.tesouro.gov.br/ords/siconfi/tt/dca?an_exercicio={p_ano}&id_ente={cod_ibge}"

    payload = await http_get(url)
    items = payload.get("items", []) if isinstance(payload, dict) else []
    return [FiscalItem(**it) for it in items]


async def consultar_entregas(
    cod_ibge: str,
    ano: int = 0,
) -> list[EntregaItem]:
    """Consulta o extrato de entregas de demonstrativos fiscais da LRF no SICONFI."""
    if GO_BINARY_PATH.exists():
        try:
            args = ["entregas", "--ibge", cod_ibge.strip()]
            if ano > 0:
                args.extend(["--ano", str(ano)])
            data = await _run_go_binary(args)
            return [EntregaItem(**item) for item in data]  # type: ignore[arg-type]
        except Exception as e:
            logger.warning("Falha no binário Go para entregas, usando fallback: %s", e)

    from mcp_brasil._shared.http_client import http_get

    p_ano = ano if ano > 0 else 2024
    url = f"https://apidatalake.tesouro.gov.br/ords/siconfi/tt/extrato_entregas?an_referencia={p_ano}&id_ente={cod_ibge}"
    payload = await http_get(url)
    items = payload.get("items", []) if isinstance(payload, dict) else []
    return [EntregaItem(**it) for it in items]


async def testar_conectividade(alvo: str, uf: str | None = None) -> list[ProbeResult]:
    """Testa conectividade dos portais de transparência."""
    if GO_BINARY_PATH.exists():
        args = ["probe", alvo]
        if uf:
            args.extend(["--uf", uf])
        data = await _run_go_binary(args)
        return [ProbeResult(**item) for item in data]  # type: ignore[arg-type]
    return []


def _build_hub_args(
    cmd: str,
    municipio: str,
    uf: str | None = None,
    nome: str = "",
    inicio: str = "",
    fim: str = "",
    ano: int = 0,
    mes: int = 0,
    limit: int = 20,
    url: str | None = None,
    ordem_salario_desc: bool = False,
    ordem_valor_desc: bool = False,
) -> list[str]:
    args = [cmd, municipio]
    if uf:
        args.extend(["--uf", uf.upper().strip()])
    if nome:
        args.extend(["--nome", nome])
    if inicio:
        args.extend(["--inicio", inicio])
    if fim:
        args.extend(["--fim", fim])
    if ano > 0:
        args.extend(["--ano", str(ano)])
    if mes > 0:
        args.extend(["--mes", str(mes)])
    if limit > 0:
        args.extend(["--limit", str(limit)])
    if url:
        args.extend(["--url", url.strip()])
    if ordem_salario_desc:
        args.append("--ordem-salario-desc")
    if ordem_valor_desc:
        args.append("--ordem-valor-desc")
    return args


async def descobrir_portal_municipio(
    municipio: str,
    uf: str | None = None,
    url: str | None = None,
) -> PortalDiscoveryResult:
    """Descobre dinamicamente o portal da transparência real e a plataforma de um município."""
    if GO_BINARY_PATH.exists():
        args = _build_hub_args("discover", municipio, uf=uf, url=url, limit=0)
        data = await _run_go_binary(args)
        return PortalDiscoveryResult(**data)  # type: ignore[arg-type]

    dos = await conectar_municipio(municipio=municipio, uf=uf)
    return PortalDiscoveryResult(
        municipio=dos.municipio,
        url_resolvida=url or dos.portal_transparencia_prefeitura,
        site_prefeitura=dos.portal_prefeitura,
        portal_camara=dos.portal_camara,
        plataforma="nao_identificado",
        observacoes=(
            "Binário Go não encontrado; utilize make build-go "
            "para descoberta e conectores completos."
        ),
    )


async def consultar_servidores_municipio(
    municipio: str,
    uf: str | None = None,
    nome: str = "",
    inicio: str = "",
    fim: str = "",
    ano: int = 0,
    mes: int = 0,
    limit: int = 20,
    url: str | None = None,
    ordem_salario_desc: bool = False,
) -> ServidoresResult:
    """Consulta servidores públicos e remuneração de um município brasileiro."""
    if GO_BINARY_PATH.exists():
        args = _build_hub_args(
            "servidores",
            municipio,
            uf=uf,
            nome=nome,
            inicio=inicio,
            fim=fim,
            ano=ano,
            mes=mes,
            limit=limit,
            url=url,
            ordem_salario_desc=ordem_salario_desc,
        )
        data = await _run_go_binary(args)
        return ServidoresResult(**data)  # type: ignore[arg-type]

    dos = await conectar_municipio(municipio=municipio, uf=uf)
    return ServidoresResult(
        municipio=dos.municipio,
        fonte="Indisponível (binário Go ausente)",
        plataforma="fallback",
        portal_url=dos.portal_transparencia_prefeitura,
        disponivel=False,
        mensagem="Binário Go bin/transparencia-go não encontrado.",
    )


async def consultar_despesas_municipio(
    municipio: str,
    uf: str | None = None,
    nome: str = "",
    inicio: str = "",
    fim: str = "",
    ano: int = 0,
    mes: int = 0,
    limit: int = 20,
    url: str | None = None,
    ordem_valor_desc: bool = False,
) -> DespesasResult:
    """Consulta despesas e empenhos de um município brasileiro."""
    if GO_BINARY_PATH.exists():
        args = _build_hub_args(
            "despesas",
            municipio,
            uf=uf,
            nome=nome,
            inicio=inicio,
            fim=fim,
            ano=ano,
            mes=mes,
            limit=limit,
            url=url,
            ordem_valor_desc=ordem_valor_desc,
        )
        data = await _run_go_binary(args)
        return DespesasResult(**data)  # type: ignore[arg-type]

    dos = await conectar_municipio(municipio=municipio, uf=uf)
    return DespesasResult(
        municipio=dos.municipio,
        fonte="Indisponível (binário Go ausente)",
        plataforma="fallback",
        portal_url=dos.portal_transparencia_prefeitura,
        disponivel=False,
        mensagem="Binário Go bin/transparencia-go não encontrado.",
    )


async def consultar_receitas_municipio(
    municipio: str,
    uf: str | None = None,
    inicio: str = "",
    fim: str = "",
    ano: int = 0,
    mes: int = 0,
    limit: int = 20,
    url: str | None = None,
) -> ReceitasResult:
    """Consulta receitas arrecadadas de um município brasileiro."""
    if GO_BINARY_PATH.exists():
        args = _build_hub_args(
            "receitas",
            municipio,
            uf=uf,
            inicio=inicio,
            fim=fim,
            ano=ano,
            mes=mes,
            limit=limit,
            url=url,
        )
        data = await _run_go_binary(args)
        return ReceitasResult(**data)  # type: ignore[arg-type]

    dos = await conectar_municipio(municipio=municipio, uf=uf)
    return ReceitasResult(
        municipio=dos.municipio,
        fonte="Indisponível (binário Go ausente)",
        plataforma="fallback",
        portal_url=dos.portal_transparencia_prefeitura,
        disponivel=False,
        mensagem="Binário Go bin/transparencia-go não encontrado.",
    )


async def consultar_licitacoes_municipio(
    municipio: str,
    uf: str | None = None,
    nome: str = "",
    inicio: str = "",
    fim: str = "",
    ano: int = 0,
    limit: int = 20,
    url: str | None = None,
) -> LicitacoesResult:
    """Consulta licitações e contratações de um município brasileiro (portal local ou PNCP)."""
    if GO_BINARY_PATH.exists():
        args = _build_hub_args(
            "licitacoes",
            municipio,
            uf=uf,
            nome=nome,
            inicio=inicio,
            fim=fim,
            ano=ano,
            limit=limit,
            url=url,
        )
        data = await _run_go_binary(args)
        return LicitacoesResult(**data)  # type: ignore[arg-type]

    dos = await conectar_municipio(municipio=municipio, uf=uf)
    return LicitacoesResult(
        municipio=dos.municipio,
        fonte="Indisponível (binário Go ausente)",
        plataforma="fallback",
        portal_url=dos.portal_transparencia_prefeitura,
        disponivel=False,
        mensagem="Binário Go bin/transparencia-go não encontrado.",
    )


async def consultar_contratos_municipio(
    municipio: str,
    uf: str | None = None,
    nome: str = "",
    inicio: str = "",
    fim: str = "",
    ano: int = 0,
    limit: int = 20,
    url: str | None = None,
) -> ContratosResult:
    """Consulta contratos administrativos de um município brasileiro (portal local ou PNCP)."""
    if GO_BINARY_PATH.exists():
        args = _build_hub_args(
            "contratos",
            municipio,
            uf=uf,
            nome=nome,
            inicio=inicio,
            fim=fim,
            ano=ano,
            limit=limit,
            url=url,
        )
        data = await _run_go_binary(args)
        return ContratosResult(**data)  # type: ignore[arg-type]

    dos = await conectar_municipio(municipio=municipio, uf=uf)
    return ContratosResult(
        municipio=dos.municipio,
        fonte="Indisponível (binário Go ausente)",
        plataforma="fallback",
        portal_url=dos.portal_transparencia_prefeitura,
        disponivel=False,
        mensagem="Binário Go bin/transparencia-go não encontrado.",
    )


async def consultar_transparencia_completa(
    municipio: str,
    uf: str | None = None,
    ano: int = 0,
    limit: int = 10,
    url: str | None = None,
) -> TransparenciaCompletaReport:
    """Executa auditoria completa e paralela de transparência municipal (portal, TCE, PNCP)."""
    if GO_BINARY_PATH.exists():
        args = _build_hub_args(
            "transparencia",
            municipio,
            uf=uf,
            ano=ano,
            limit=limit,
            url=url,
        )
        data = await _run_go_binary(args)
        return TransparenciaCompletaReport(**data)  # type: ignore[arg-type]

    dos = await conectar_municipio(municipio=municipio, uf=uf)
    return TransparenciaCompletaReport(
        dossier=dos,
        fontes_consultadas=["Cadastro IBGE/SICONFI (fallback Python)"],
        limitacoes=["Binário Go bin/transparencia-go ausente para coleta ativa."],
    )


async def listar_municipios_megasoft(
    uf: str | None = None,
) -> list[MegasoftMunicipio]:
    """Lista todos os municípios brasileiros com portais Megasoft verificados."""
    if GO_BINARY_PATH.exists():
        args = ["megasoft-list"]
        if uf:
            args.extend(["--uf", uf.upper().strip()])
        data = await _run_go_binary(args)
        if isinstance(data, list):
            return [MegasoftMunicipio(**item) for item in data]
        return []

    if not MEGASOFT_CSV_PATH.exists():
        return []
    uf_filter = uf.upper().strip() if uf else None
    results: list[MegasoftMunicipio] = []
    with open(MEGASOFT_CSV_PATH, encoding="utf-8") as f:
        reader = csv.reader(f)
        header = next(reader, None)
        _ = header
        for row in reader:
            if len(row) < 4:
                continue
            row_uf = row[2].strip().upper()
            if uf_filter and row_uf != uf_filter:
                continue
            cod_orgao = int(row[5].strip()) if len(row) >= 6 and row[5].strip().isdigit() else 0
            results.append(
                MegasoftMunicipio(
                    codigo_ibge=row[0].strip(),
                    nome=row[1].strip(),
                    uf=row_uf,
                    portal_url=row[3].strip(),
                    portal_camara=row[4].strip() if len(row) >= 5 else "",
                    codigo_orgao=cod_orgao,
                    site_prefeitura=row[6].strip() if len(row) >= 7 else "",
                )
            )
    return results


async def consultar_diarias_municipio(
    municipio: str,
    uf: str | None = None,
    nome: str = "",
    inicio: str = "",
    fim: str = "",
    ano: int = 0,
    limit: int = 20,
    url: str | None = None,
) -> DiariasResult:
    """Consulta pagamentos de diárias e viagens a servidores e agentes públicos municipais."""
    if GO_BINARY_PATH.exists():
        args = _build_hub_args(
            "diarias",
            municipio,
            uf=uf,
            nome=nome,
            inicio=inicio,
            fim=fim,
            ano=ano,
            limit=limit,
            url=url,
        )
        data = await _run_go_binary(args)
        return DiariasResult(**data)  # type: ignore[arg-type]

    dos = await conectar_municipio(municipio=municipio, uf=uf)
    return DiariasResult(
        municipio=dos.municipio,
        fonte="Indisponível (binário Go ausente)",
        plataforma="fallback",
        portal_url=dos.portal_transparencia_prefeitura,
        disponivel=False,
        mensagem="Binário Go bin/transparencia-go não encontrado.",
    )


async def consultar_obras_municipio(
    municipio: str,
    uf: str | None = None,
    ano: int = 0,
    limit: int = 20,
    url: str | None = None,
) -> ObrasResult:
    """Consulta obras públicas municipais com situação, valores e prazos."""
    if GO_BINARY_PATH.exists():
        args = _build_hub_args(
            "obras",
            municipio,
            uf=uf,
            ano=ano,
            limit=limit,
            url=url,
        )
        data = await _run_go_binary(args)
        return ObrasResult(**data)  # type: ignore[arg-type]

    dos = await conectar_municipio(municipio=municipio, uf=uf)
    return ObrasResult(
        municipio=dos.municipio,
        fonte="Indisponível (binário Go ausente)",
        plataforma="fallback",
        portal_url=dos.portal_transparencia_prefeitura,
        disponivel=False,
        mensagem="Binário Go bin/transparencia-go não encontrado.",
    )


async def consultar_convenios_municipio(
    municipio: str,
    uf: str | None = None,
    ano: int = 0,
    limit: int = 20,
    url: str | None = None,
) -> ConveniosResult:
    """Consulta convênios e transferências voluntárias do município."""
    if GO_BINARY_PATH.exists():
        args = _build_hub_args(
            "convenios",
            municipio,
            uf=uf,
            ano=ano,
            limit=limit,
            url=url,
        )
        data = await _run_go_binary(args)
        return ConveniosResult(**data)  # type: ignore[arg-type]

    dos = await conectar_municipio(municipio=municipio, uf=uf)
    return ConveniosResult(
        municipio=dos.municipio,
        fonte="Indisponível (binário Go ausente)",
        plataforma="fallback",
        portal_url=dos.portal_transparencia_prefeitura,
        disponivel=False,
        mensagem="Binário Go bin/transparencia-go não encontrado.",
    )


async def consultar_legislacao_municipio(
    municipio: str,
    uf: str | None = None,
    ano: int = 0,
    limit: int = 20,
    url: str | None = None,
) -> LegislacaoResult:
    """Consulta leis, decretos, portarias e atos normativos municipais."""
    if GO_BINARY_PATH.exists():
        args = _build_hub_args(
            "legislacao",
            municipio,
            uf=uf,
            ano=ano,
            limit=limit,
            url=url,
        )
        data = await _run_go_binary(args)
        return LegislacaoResult(**data)  # type: ignore[arg-type]

    dos = await conectar_municipio(municipio=municipio, uf=uf)
    return LegislacaoResult(
        municipio=dos.municipio,
        fonte="Indisponível (binário Go ausente)",
        plataforma="fallback",
        portal_url=dos.portal_transparencia_prefeitura,
        disponivel=False,
        mensagem="Binário Go bin/transparencia-go não encontrado.",
    )


async def consultar_concursos_municipio(
    municipio: str,
    uf: str | None = None,
    ano: int = 0,
    limit: int = 20,
    url: str | None = None,
) -> ConcursosResult:
    """Consulta aprovados e convocados em concursos e processos seletivos públicos municipais."""
    if GO_BINARY_PATH.exists():
        args = _build_hub_args(
            "concursos",
            municipio,
            uf=uf,
            ano=ano,
            limit=limit,
            url=url,
        )
        data = await _run_go_binary(args)
        return ConcursosResult(**data)  # type: ignore[arg-type]

    dos = await conectar_municipio(municipio=municipio, uf=uf)
    return ConcursosResult(
        municipio=dos.municipio,
        fonte="Indisponível (binário Go ausente)",
        plataforma="fallback",
        portal_url=dos.portal_transparencia_prefeitura,
        disponivel=False,
        mensagem="Binário Go bin/transparencia-go não encontrado.",
    )


async def consultar_detalhe_servidor_municipio(
    municipio: str,
    matricula: str,
    uf: str | None = None,
    ano: int = 0,
    mes: int = 0,
    url: str | None = None,
) -> ServidorDetalheResult:
    """Consulta o contracheque detalhado de um servidor público municipal (LGPD: CPF mascarado)."""
    if GO_BINARY_PATH.exists():
        args = [
            "servidor-detalhe",
            municipio,
            "--matricula",
            str(matricula).strip(),
        ]
        if uf:
            args.extend(["--uf", uf.upper().strip()])
        if ano > 0:
            args.extend(["--ano", str(ano)])
        if mes > 0:
            args.extend(["--mes", str(mes)])
        if url:
            args.extend(["--url", url.strip()])
        data = await _run_go_binary(args)
        return ServidorDetalheResult(**data)  # type: ignore[arg-type]

    dos = await conectar_municipio(municipio=municipio, uf=uf)
    return ServidorDetalheResult(
        municipio=dos.municipio,
        nome="Indisponível sem binário Go",
        matricula=str(matricula),
        salario_bruto=0.0,
        descontos=0.0,
        salario_liquido=0.0,
        portal_url=dos.portal_transparencia_prefeitura,
        fonte="Indisponível (binário Go ausente)",
    )
