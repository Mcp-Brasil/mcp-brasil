"""HTTP client for the TCE-CE API (SIM 2.0).

Endpoints (base https://api-dados-abertos.tce.ce.gov.br/sim):
    - /municipios                              → listar_municipios
    - /processos_administrativos_contratacoes  → buscar_licitacoes
    - /contratos                               → buscar_contratos
    - /contratados                             → buscar_contratados
    - /notas_empenhos                          → buscar_empenhos

Contrato medido em 14/10/2026:
    - Resposta: {"name": ..., "elements": [...], "links": [{"rel": "next", ...}]}.
      Nao ha total; "tem mais" = existe link rel=next.
    - Paginacao: $count (max 1000) e $start_index (offset).
    - Datas de filtro de contratos/contratados/processos: yyyyMMdd
      ("2026-01-01" devolve erro de parse). Empenhos: exercicio_orcamento
      = AAAA00 e data_referencia_doc = AAAAMM, ambos obrigatorios.
    - O registro de contrato NAO traz o contratado: ele vem de /contratados,
      cruzado por (numero_contrato, data_contrato).
    - CPFs de gestores vem cifrados; acesso so de IP brasileiro.
"""

from __future__ import annotations

import re
from typing import Any

from mcp_brasil._shared.http_client import http_get

from .constants import (
    CONTRATADOS_URL,
    CONTRATOS_URL,
    DEFAULT_QUANTIDADE,
    EMPENHOS_URL,
    MAX_CONTRATADOS_JOIN,
    MAX_QUANTIDADE,
    MUNICIPIOS_URL,
    PROCESSOS_URL,
)
from .schemas import (
    Contratado,
    Contrato,
    ContratoResultado,
    Empenho,
    EmpenhoResultado,
    Licitacao,
    LicitacaoResultado,
    Municipio,
)

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def normalizar_codigo_municipio(codigo: str | int) -> str:
    """Codigo do municipio no SIM tem 3 digitos ("57" -> "057")."""
    digitos = re.sub(r"\D", "", str(codigo))
    if not digitos:
        raise ValueError(f"Codigo de municipio invalido: {codigo!r}")
    return digitos.zfill(3)


def _data_para_sim(valor: str) -> str:
    """Converte yyyy-mm-dd, dd/mm/yyyy ou yyyymmdd para yyyyMMdd."""
    v = valor.strip()
    if re.fullmatch(r"\d{8}", v):
        return v
    m = re.fullmatch(r"(\d{4})-(\d{2})-(\d{2})", v)
    if m:
        return "".join(m.groups())
    m = re.fullmatch(r"(\d{2})/(\d{2})/(\d{4})", v)
    if m:
        return m.group(3) + m.group(2) + m.group(1)
    raise ValueError(f"Data invalida: {valor!r} (use yyyy-mm-dd, dd/mm/yyyy ou yyyymmdd)")


def parse_periodo(periodo: str) -> tuple[str, str]:
    """Converte um periodo em (data_inicio, data_fim) no formato yyyyMMdd.

    Aceita: "2026" (ano inteiro), "2026-01-01" (um dia),
    "2026-01-01_2026-12-31" (intervalo; tambem com dd/mm/yyyy ou yyyymmdd).
    """
    p = periodo.strip()
    if re.fullmatch(r"\d{4}", p):
        return f"{p}0101", f"{p}1231"
    if "_" in p:
        ini, fim = p.split("_", 1)
        return _data_para_sim(ini), _data_para_sim(fim)
    d = _data_para_sim(p)
    return d, d


def _elements(raw: Any) -> list[dict[str, Any]]:
    if isinstance(raw, dict):
        items = raw.get("elements", [])
        return items if isinstance(items, list) else []
    return []


def _tem_mais(raw: Any) -> bool:
    if not isinstance(raw, dict):
        return False
    return any(
        isinstance(link, dict) and link.get("rel") == "next" for link in raw.get("links") or []
    )


def _strip(valor: Any) -> str | None:
    if valor is None:
        return None
    texto = str(valor).strip()
    return texto or None


def _dia(valor: Any) -> str | None:
    """ "2026-08-31T00:00:00" -> "2026-08-31"."""
    texto = _strip(valor)
    return texto[:10] if texto else None


def _documento(valor: Any) -> str | None:
    """CNPJ/CPF em digitos; o SIM cifra o CPF de pessoa fisica (base64) -> None."""
    texto = _strip(valor)
    return texto if texto and texto.isdigit() else None


def _paginacao(quantidade: int, deslocamento: int) -> dict[str, Any]:
    return {
        "$count": max(1, min(quantidade, MAX_QUANTIDADE)),
        "$start_index": max(0, deslocamento),
    }


# ---------------------------------------------------------------------------
# Municipios
# ---------------------------------------------------------------------------


async def listar_municipios(nome_municipio: str | None = None) -> list[Municipio]:
    """Lista os municipios cearenses (opcionalmente filtrados pelo nome exato)."""
    params: dict[str, Any] = {"$count": MAX_QUANTIDADE}
    if nome_municipio:
        params["nome_municipio"] = nome_municipio.strip().upper()
    raw = await http_get(MUNICIPIOS_URL, params=params)
    return [
        Municipio(
            codigo_municipio=_strip(item.get("codigo_municipio")),
            nome_municipio=_strip(item.get("nome_municipio")),
            codigo_municipio_ibge=_strip(item.get("codigo_municipio_ibge")),
        )
        for item in _elements(raw)
    ]


# ---------------------------------------------------------------------------
# Licitacoes (processos administrativos de contratacao)
# ---------------------------------------------------------------------------


async def buscar_licitacoes(
    *,
    codigo_municipio: str | int,
    data_realizacao: str,
    quantidade: int = DEFAULT_QUANTIDADE,
    deslocamento: int = 0,
) -> LicitacaoResultado:
    """Busca processos de contratacao (licitacoes, dispensas, adesoes) de um municipio.

    Args:
        codigo_municipio: Codigo do municipio (ex: "057" para Fortaleza).
        data_realizacao: Periodo ("2026", "2026-08-01" ou "2026-01-01_2026-12-31").
        quantidade: Registros por pagina (max 1000).
        deslocamento: Offset.
    """
    inicio, fim = parse_periodo(data_realizacao)
    params: dict[str, Any] = {
        "codigo_municipio": normalizar_codigo_municipio(codigo_municipio),
        "data_inicio": inicio,
        "data_fim": fim,
        **_paginacao(quantidade, deslocamento),
    }
    raw = await http_get(PROCESSOS_URL, params=params)
    licitacoes = [
        Licitacao(
            codigo_municipio=_strip(item.get("codigo_municipio")),
            numero_licitacao=_strip(item.get("numero_licitacao")),
            data_realizacao=_dia(item.get("data_realizacao_autuacao_licitacao")),
            modalidade_licitacao=_strip(item.get("modalidade_licitacao")),
            objeto=_strip(item.get("descricao_objeto_licitacao")),
            valor_orcado_estimado=item.get("valor_orcado_estimado"),
            data_homologacao=_dia(item.get("data_homologacao")),
            nome_responsavel_homologacao=_strip(item.get("nome_responsavel_homologacao")),
            modalidade_processo_administrativo=_strip(
                item.get("modalidade_processo_administrativo")
            ),
        )
        for item in _elements(raw)
    ]
    return LicitacaoResultado(
        licitacoes=licitacoes, total=len(licitacoes), tem_mais=_tem_mais(raw)
    )


# ---------------------------------------------------------------------------
# Contratos + contratados
# ---------------------------------------------------------------------------


def _chave_contrato(numero: Any, data: Any) -> tuple[str, str]:
    return (_strip(numero) or "", _dia(data) or "")


async def buscar_contratados(
    *,
    codigo_municipio: str | int,
    data_contrato: str,
    limite: int = MAX_CONTRATADOS_JOIN,
) -> tuple[list[Contratado], bool]:
    """Lista os contratados (nome e CNPJ/CPF) dos contratos celebrados no periodo.

    Returns:
        (contratados, truncado). truncado=True quando o teto `limite` parou a
        paginacao antes do fim: a lista NAO e o universo do periodo.
    """
    inicio, fim = parse_periodo(data_contrato)
    codigo = normalizar_codigo_municipio(codigo_municipio)
    contratados: list[Contratado] = []
    deslocamento = 0
    truncado = False
    while True:
        if len(contratados) >= limite:
            truncado = True
            break
        params: dict[str, Any] = {
            "codigo_municipio": codigo,
            "data_inicio": inicio,
            "data_fim": fim,
            **_paginacao(MAX_QUANTIDADE, deslocamento),
        }
        raw = await http_get(CONTRATADOS_URL, params=params)
        pagina = _elements(raw)
        contratados.extend(
            Contratado(
                numero_contrato=_strip(item.get("numero_contrato")),
                data_contrato=_dia(item.get("data_contrato")),
                numero_documento_negociante=_documento(item.get("numero_documento_negociante")),
                nome_negociante=_strip(item.get("nome_negociante")),
                nome_municipio_negociante=_strip(item.get("nome_municipio_negociante")),
                codigo_uf=_strip(item.get("codigo_uf")),
            )
            for item in pagina
        )
        if not pagina or not _tem_mais(raw):
            break
        deslocamento += len(pagina)
    return contratados, truncado


async def buscar_contratos(
    *,
    codigo_municipio: str | int,
    data_contrato: str,
    quantidade: int = DEFAULT_QUANTIDADE,
    deslocamento: int = 0,
    incluir_contratado: bool = True,
) -> ContratoResultado:
    """Busca contratos de um municipio cearense, com o contratado de cada um.

    Args:
        codigo_municipio: Codigo do municipio.
        data_contrato: Periodo ("2026", "2026-08-31" ou "2026-01-01_2026-12-31").
        quantidade: Registros por pagina (max 1000).
        deslocamento: Offset.
        incluir_contratado: Cruza com /contratados para trazer nome e CNPJ/CPF.
    """
    inicio, fim = parse_periodo(data_contrato)
    params: dict[str, Any] = {
        "codigo_municipio": normalizar_codigo_municipio(codigo_municipio),
        "data_inicio": inicio,
        "data_fim": fim,
        **_paginacao(quantidade, deslocamento),
    }
    raw = await http_get(CONTRATOS_URL, params=params)
    itens = _elements(raw)

    por_contrato: dict[tuple[str, str], Contratado] = {}
    contratado_incompleto = False
    if incluir_contratado and itens:
        # Busca os contratados so no intervalo de datas DESTA pagina (menor universo que o
        # periodo pedido), paginando ate o fim; o teto so existe como freio e e declarado.
        dias = sorted(d for d in (_dia(i.get("data_contrato")) for i in itens) if d)
        periodo_pagina = f"{dias[0]}_{dias[-1]}" if dias else data_contrato
        contratados, contratado_incompleto = await buscar_contratados(
            codigo_municipio=codigo_municipio, data_contrato=periodo_pagina
        )
        for c in contratados:
            por_contrato.setdefault(_chave_contrato(c.numero_contrato, c.data_contrato), c)

    contratos: list[Contrato] = []
    for item in itens:
        contratado = por_contrato.get(
            _chave_contrato(item.get("numero_contrato"), item.get("data_contrato"))
        )
        contratos.append(
            Contrato(
                codigo_municipio=_strip(item.get("codigo_municipio")),
                numero_contrato=_strip(item.get("numero_contrato")),
                data_contrato=_dia(item.get("data_contrato")),
                tipo_contrato=_strip(item.get("tipo_contrato")),
                # o SIM grafa o campo como "modalide_contrato"
                modalidade_contrato=_strip(
                    item.get("modalide_contrato") or item.get("modalidade_contrato")
                ),
                objeto=_strip(item.get("descricao_objeto_contrato")),
                valor_total_contrato=item.get("valor_total_contrato"),
                data_inicio_vigencia=_dia(item.get("data_inicio_vigencia_contrato")),
                data_fim_vigencia=_dia(item.get("data_fim_vigencia_contrato")),
                numero_contrato_original=_strip(item.get("numero_contrato_original")),
                numero_processo_adm=_strip(item.get("numero_processo_adm")),
                nome_fiscal_contrato=_strip(item.get("nome_fiscal_contrato")),
                nome_contratado=contratado.nome_negociante if contratado else None,
                documento_contratado=(
                    contratado.numero_documento_negociante if contratado else None
                ),
                municipio_contratado=(
                    contratado.nome_municipio_negociante if contratado else None
                ),
            )
        )
    return ContratoResultado(
        contratos=contratos,
        total=len(contratos),
        tem_mais=_tem_mais(raw),
        contratado_incompleto=contratado_incompleto,
    )


# ---------------------------------------------------------------------------
# Empenhos
# ---------------------------------------------------------------------------


def _ano_mes(data_referencia: int | str) -> tuple[str, str]:
    """202608 / "2026-08" / "08/2026" -> ("202600", "202608")."""
    v = re.sub(r"\D", "", str(data_referencia))
    if len(v) != 6:
        raise ValueError(f"data_referencia invalida: {data_referencia!r} (use AAAAMM)")
    if int(v[:2]) <= 12 and not v.startswith(("19", "20")):
        v = v[2:] + v[:2]  # MMAAAA -> AAAAMM
    return f"{v[:4]}00", v


async def buscar_empenhos(
    *,
    codigo_municipio: str | int,
    data_referencia: int | str,
    codigo_orgao: str | None = None,
    quantidade: int = DEFAULT_QUANTIDADE,
    deslocamento: int = 0,
) -> EmpenhoResultado:
    """Busca notas de empenho de um municipio cearense em um mes.

    Args:
        codigo_municipio: Codigo do municipio.
        data_referencia: Ano-mes AAAAMM (ex: 202608).
        codigo_orgao: Codigo do orgao no SIM (opcional; varia por municipio).
        quantidade: Registros por pagina (max 1000).
        deslocamento: Offset.
    """
    exercicio, referencia = _ano_mes(data_referencia)
    params: dict[str, Any] = {
        "codigo_municipio": normalizar_codigo_municipio(codigo_municipio),
        "exercicio_orcamento": exercicio,
        "data_referencia_doc": referencia,
        **_paginacao(quantidade, deslocamento),
    }
    if codigo_orgao:
        params["codigo_orgao"] = codigo_orgao
    raw = await http_get(EMPENHOS_URL, params=params)
    empenhos = [
        Empenho(
            codigo_municipio=_strip(item.get("codigo_municipio")),
            numero_empenho=_strip(item.get("numero_empenho")),
            data_emissao=_dia(item.get("data_emissao_empenho")),
            valor_empenho=item.get("valor_empenhado"),
            nome_negociante=_strip(item.get("nome_negociante")),
            numero_documento_negociante=_documento(item.get("numero_documento_negociante")),
            historico=_strip(item.get("descricao_historico_empenho")),
            codigo_orgao=_strip(item.get("codigo_orgao")),
            codigo_funcao=_strip(item.get("codigo_funcao")),
        )
        for item in _elements(raw)
    ]
    return EmpenhoResultado(empenhos=empenhos, total=len(empenhos), tem_mais=_tem_mais(raw))
