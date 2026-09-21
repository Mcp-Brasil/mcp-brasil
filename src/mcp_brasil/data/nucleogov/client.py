"""HTTP client for NucleoGov municipal transparency portals.

`NucleoGov <https://www.nucleogov.com.br/>`_ hosts the transparency portal of
most municipalities in Goiás and several other states, usually under the host
pattern ``acessoainformacao.<municipio>.<uf>.gov.br``. The portal's own
front-end talks to the public API this client uses::

    POST {portal}/api/{acao}
    Content-Type: application/x-www-form-urlencoded

    multi_request=true
    params={"<request-id>": {<filtros>, "limit": "<offset>, <n>", "acao": "<acao>"}}

Response::

    {"<request-id>": {"dados": [...], "total": N}}

The data is public under the Access to Information Law (Lei 12.527/2011): no
login, no key. The portal's WAF rejects requests that identify as a program,
so requests carry a browser User-Agent — not a credential.

Actions per module (the ``_frl`` suffix is the common case; some portals use
``_cnt``)::

    despesas   despesas_frl/listar    datamin, datamax, cpf_cnpj, txtbusca
    contratos  contratos_frl/listar   tipo(contrato|aditivo), anobase, txtbusca
    licitacoes licitacoes_frl/listar  dispensas(0|1), fracassadas_desertas, anobase, txtbusca
    legislacao atos_pf_{slug}/listar  tipo(DECRETO|PORTARIA|...), txtbusca

For safety, ``portal`` must be HTTPS on a ``.gov.br`` host; IP literals,
explicit ports, embedded credentials and redirects to other hosts are rejected.
"""

from __future__ import annotations

import calendar
import datetime
import ipaddress
import json
import re
import unicodedata
from typing import Any
from urllib.parse import urlsplit, urlunsplit

import httpx

from mcp_brasil._shared.formatting import parse_brl_number
from mcp_brasil._shared.http_client import create_client
from mcp_brasil.exceptions import HttpClientError

from .constants import (
    ACAO_ATOS_TEMPLATE,
    ACAO_CONTRATOS,
    ACAO_DESPESAS,
    ACAO_LICITACOES,
    BROWSER_USER_AGENT,
    CAMPO_BUSCA,
    DEFAULT_LIMITE,
    DEFAULT_TIMEOUT,
)
from .schemas import (
    AtoNormativo,
    Contrato,
    Despesa,
    Licitacao,
    ResultadoAtos,
    ResultadoContratos,
    ResultadoDespesas,
    ResultadoLicitacoes,
)

#: Echoed by the API as the response key; any stable value works.
_REQUEST_ID = "0-mcp"

_RE_DATA_BR = re.compile(r"\d{2}/\d{2}/\d{4}")
_RE_DATA_ISO = re.compile(r"(\d{4})-(\d{2})(?:-(\d{2}))?")

#: Only government hosts are accepted — the tool must not be pointable at an
#: internal service (SSRF).
_HOST_SUFIXO_GOV = ".gov.br"

#: Error keys the portal may return inside the envelope.
_ERRO_KEYS = ("erro", "error", "mensagem")

#: Keys of a pure pagination-metadata block (e.g. ``{"total": 0}``), which is a
#: legitimate empty result — never a record.
_META_KEYS = {"total", "pagina", "page", "limit", "offset", "quant_linhas", "quantidade", "count"}


def _normalizar_portal(portal: str) -> str:
    """Validate and normalize the portal URL.

    Only HTTPS on a ``.gov.br`` host is accepted. IP literals, explicit ports,
    embedded credentials, query strings and fragments are rejected so the tool
    cannot be aimed at a local or internal service (SSRF).
    """
    bruto = portal.strip()
    if not bruto:
        raise ValueError("informe o domínio do portal (ex.: acessoainformacao.cidade.go.gov.br)")
    if "://" not in bruto:
        bruto = "https://" + bruto
    partes = urlsplit(bruto)
    if partes.scheme != "https":
        raise ValueError("NucleoGov: o portal deve usar HTTPS.")
    if partes.username or partes.password:
        raise ValueError("NucleoGov: o endereço do portal não pode conter credenciais.")
    try:
        porta = partes.port
    except ValueError as exc:
        raise ValueError("NucleoGov: porta inválida no endereço do portal.") from exc
    if porta is not None:
        raise ValueError("NucleoGov: o endereço do portal não pode conter porta.")
    host = (partes.hostname or "").lower()
    if not host:
        raise ValueError("NucleoGov: informe o domínio do portal.")
    try:
        ipaddress.ip_address(host)
    except ValueError:
        pass
    else:
        raise ValueError("NucleoGov: informe o domínio do portal, não um endereço IP.")
    if not host.endswith(_HOST_SUFIXO_GOV):
        raise ValueError(
            "NucleoGov: o portal deve ser um domínio gov.br "
            "(ex.: acessoainformacao.cidade.go.gov.br)."
        )
    if partes.query or partes.fragment:
        raise ValueError("NucleoGov: o endereço do portal não pode conter query ou fragmento.")
    return urlunsplit(("https", host, partes.path.rstrip("/"), "", ""))


def _data_br(valor: str, *, campo: str, fim: bool = False) -> str:
    """Accept ``AAAA-MM-DD``, ``AAAA-MM`` or ``dd/mm/aaaa``; return ``dd/mm/aaaa``.

    With ``AAAA-MM`` and ``fim=True`` the last day of the month is used, so the
    final limit covers the whole month. Real calendar dates are validated.
    """
    texto = valor.strip()
    dia: int | None
    if _RE_DATA_BR.fullmatch(texto):
        dia_txt, mes_txt, ano_txt = texto.split("/")
        ano, mes, dia = int(ano_txt), int(mes_txt), int(dia_txt)
    else:
        match = _RE_DATA_ISO.fullmatch(texto)
        if not match:
            raise ValueError(
                f"{campo}: use AAAA-MM-DD, AAAA-MM ou dd/mm/aaaa (recebido: {valor!r})"
            )
        ano, mes = int(match.group(1)), int(match.group(2))
        dia = int(match.group(3)) if match.group(3) is not None else None
    try:
        if dia is None:
            dia = calendar.monthrange(ano, mes)[1] if fim else 1
        datetime.date(ano, mes, dia)
    except ValueError as exc:
        raise ValueError(f"{campo}: data inválida (recebido: {valor!r})") from exc
    return f"{dia:02d}/{mes:02d}/{ano:04d}"


def _somente_digitos(valor: object) -> str:
    return "".join(ch for ch in str(valor or "") if ch.isdigit())


def _mascarar_documento(documento: str) -> str:
    """Format a CNPJ/CPF the way the remote ``cpf_cnpj`` filter expects.

    The portal only matches the masked form (``00.000.000/0000-00`` or
    ``000.000.000-00``); digits-only input returns zero results.
    """
    d = _somente_digitos(documento)
    if len(d) == 14:
        return f"{d[:2]}.{d[2:5]}.{d[5:8]}/{d[8:12]}-{d[12:]}"
    if len(d) == 11:
        return f"{d[:3]}.{d[3:6]}.{d[6:9]}-{d[9:]}"
    return documento.strip()


def _texto(valor: object) -> str | None:
    if valor is None:
        return None
    texto = str(valor).strip()
    return texto or None


def _inteiro(valor: object) -> int | None:
    if valor is None or isinstance(valor, bool):
        return None
    if isinstance(valor, int):
        return valor
    if isinstance(valor, (str, float)):
        try:
            return int(valor)
        except (TypeError, ValueError, OverflowError):
            return None
    return None


def _slug(municipio: str) -> str:
    """Turn a municipality name into the portal's legislation slug."""
    base = unicodedata.normalize("NFKD", municipio.strip().lower())
    base = "".join(ch for ch in base if not unicodedata.combining(ch))
    base = re.sub(r"[^a-z0-9]+", "_", base).strip("_")
    if not base:
        raise ValueError("município inválido para a consulta de legislação")
    return base


def _corpo(acao: str, filtros: dict[str, object], inicio: int, limite: int) -> dict[str, str]:
    """Build the form body for a single request."""
    inner: dict[str, object] = dict(filtros)
    inner["limit"] = f"{inicio}, {limite}"
    inner["acao"] = acao
    return {
        "multi_request": "true",
        "params": json.dumps({_REQUEST_ID: inner}, ensure_ascii=False),
    }


def _extrair(data: Any, acao: str) -> tuple[list[dict[str, Any]], int]:
    """Pull the records and the declared total out of the API envelope.

    A structured error envelope (``{"erro": ...}``) is always an error — never
    an empty list. A pure metadata block (``{"total": 0}``) is a legitimate
    empty result. Any other shape without records is treated as unexpected.
    """
    if not isinstance(data, dict) or not data:
        raise HttpClientError(
            f"NucleoGov: ação '{acao}' não reconhecida neste portal (resposta vazia)."
        )
    bloco = next(iter(data.values()))
    if isinstance(bloco, str):
        raise HttpClientError(f"NucleoGov: o portal recusou a ação '{acao}': {bloco}")
    if not isinstance(bloco, dict):
        raise HttpClientError(f"NucleoGov: resposta inesperada do portal na ação '{acao}'.")

    registros = bloco.get("dados")
    if registros is None:
        registros = bloco.get("registros")

    if registros is None:
        for chave in _ERRO_KEYS:
            valor = bloco.get(chave)
            if valor is True or (isinstance(valor, str) and valor.strip()):
                raise HttpClientError(f"NucleoGov: o portal recusou a ação '{acao}': {valor}")
        if bloco and set(bloco) <= _META_KEYS:
            registros = []
        else:
            raise HttpClientError(f"NucleoGov: resposta inesperada do portal na ação '{acao}'.")

    if not isinstance(registros, list):
        raise HttpClientError(f"NucleoGov: campo de registros inválido na ação '{acao}'.")
    itens = [item for item in registros if isinstance(item, dict)]
    try:
        total = int(bloco.get("total") or 0)
    except (TypeError, ValueError):
        total = len(itens)
    return itens, total


async def _post(
    portal: str,
    acao: str,
    filtros: dict[str, object],
    *,
    inicio: int,
    limite: int,
    timeout: float,
) -> tuple[list[dict[str, Any]], int]:
    """POST one page to ``{portal}/api/{acao}`` and return ``(registros, total)``."""
    base = _normalizar_portal(portal)
    url = f"{base}/api/{acao}"
    headers = {
        "User-Agent": BROWSER_USER_AGENT,
        "X-Requested-With": "XMLHttpRequest",
        "Accept": "application/json, text/plain, */*",
    }
    async with create_client(headers=headers, timeout=timeout) as http:
        # Never follow redirects: a redirect to a host outside the allowlist
        # would defeat the URL validation (SSRF).
        http.follow_redirects = False
        try:
            response = await http.post(url, data=_corpo(acao, filtros, inicio, limite))
        except httpx.HTTPError as exc:
            raise HttpClientError(f"NucleoGov: falha de rede ao consultar {url}: {exc}") from exc

    if response.is_redirect:
        destino = response.headers.get("location", "")
        raise HttpClientError(
            f"NucleoGov: o portal {base} redirecionou a requisição para {destino!r}; "
            "por segurança o redirecionamento não é seguido."
        )
    if response.status_code == 403:
        raise HttpClientError(
            f"NucleoGov: o portal {base} recusou a requisição (HTTP 403). "
            "O dado é público (Lei 12.527/2011), mas o portal pode filtrar clientes "
            "automatizados; confirme o endereço ou consulte o portal no navegador."
        )
    if response.status_code != 200:
        raise HttpClientError(
            f"NucleoGov: o portal {base} respondeu HTTP {response.status_code} na ação '{acao}'."
        )
    try:
        data = response.json()
    except ValueError as exc:
        raise HttpClientError(
            f"NucleoGov: o portal {base} devolveu resposta não-JSON na ação '{acao}'."
        ) from exc
    return _extrair(data, acao)


def _despesa(item: dict[str, Any]) -> Despesa:
    return Despesa(
        id=_texto(item.get("id")),
        orgao=_texto(item.get("orgao")),
        unidade=_texto(item.get("unidade_nome") or item.get("unidade")),
        numero_processo=_texto(item.get("numero_processo")),
        numero_empenho=_texto(item.get("numero_empenho")),
        tipo=_texto(item.get("tipo")),
        data=_texto(item.get("data")),
        fornecedor=_texto(item.get("fornecedor")),
        cpf_cnpj=_texto(item.get("cpf_cnpj")),
        historico=_texto(item.get("historico")),
        funcao=_texto(item.get("funcao_nome")),
        subfuncao=_texto(item.get("subfuncao_nome")),
        programa=_texto(item.get("programa_nome")),
        natureza=_texto(item.get("natureza_nome")),
        fonte=_texto(item.get("fonte_nome")),
        elemento=_texto(item.get("elemento")),
        modalidade=_texto(item.get("modalidade")),
        numero_licitacao=_texto(item.get("numero_licitacao")),
        valor_empenho=parse_brl_number(item.get("valor_empenho")),
        valor_pago=parse_brl_number(item.get("valor_pago")),
        valor_saldo=parse_brl_number(item.get("valor_saldo")),
    )


def _contrato(item: dict[str, Any]) -> Contrato:
    return Contrato(
        id=_texto(item.get("id")),
        numero=_texto(item.get("numero")),
        numero_detalhado=_texto(item.get("numero_detalhado")),
        licitacao=_texto(item.get("licitacao")),
        ano=_inteiro(item.get("ano")),
        orgao=_texto(item.get("orgao")),
        situacao=_texto(item.get("situacao")),
        tipo=_texto(item.get("tipo")),
        data_publicacao=_texto(item.get("data_publicacao")),
        data_firmatura=_texto(item.get("data_firmatura")),
        inicio_vigencia=_texto(item.get("inicio_vigencia")),
        fim_vigencia=_texto(item.get("fim_vigencia")),
        contratado_nome=_texto(item.get("contratado_nome")),
        contratado_documento=_texto(item.get("contratado_documento")),
        fiscal_contrato=_texto(item.get("fiscal_contrato")),
        objeto=_texto(item.get("objeto")),
        valor=parse_brl_number(item.get("valor")),
    )


def _licitacao(item: dict[str, Any]) -> Licitacao:
    return Licitacao(
        id=_texto(item.get("id")),
        numero=_texto(item.get("numero")),
        numero_processo=_texto(item.get("numero_processo")),
        modalidade=_texto(item.get("modalidade")),
        modalidade_tce=_texto(item.get("modalidade_tce")),
        situacao=_texto(item.get("situacao")),
        orgao=_texto(item.get("orgao")),
        ano=_inteiro(item.get("ano")),
        anobase=_inteiro(item.get("anobase")),
        data_abertura=_texto(item.get("data_abertura")),
        data_edital=_texto(item.get("data_edital")),
        descricao=_texto(item.get("descricao")),
        valor_estimado=parse_brl_number(item.get("valor_estimado")),
    )


def _ato(item: dict[str, Any]) -> AtoNormativo:
    return AtoNormativo(
        id=_texto(item.get("id")),
        tipo=_texto(item.get("tipo")),
        numero=_texto(item.get("numero")),
        ementa=_texto(item.get("ementa")),
        orgao=_texto(item.get("orgao")),
        data_publicacao=_texto(item.get("data_publicacao")),
        url=_texto(item.get("url")),
    )


async def buscar_despesas(
    portal: str,
    *,
    data_inicio: str | None = None,
    data_fim: str | None = None,
    cnpj: str | None = None,
    busca: str | None = None,
    inicio: int = 0,
    limite: int = DEFAULT_LIMITE,
    timeout: float = DEFAULT_TIMEOUT,
) -> ResultadoDespesas:
    """Busca empenhos (despesas) de um portal municipal NucleoGov.

    Args:
        portal: Domínio ou URL do portal (ex.: ``acessoainformacao.cidade.go.gov.br``).
        data_inicio: Início do período (``AAAA-MM-DD``, ``AAAA-MM`` ou ``dd/mm/aaaa``).
            Com ``AAAA-MM`` o período começa no primeiro dia do mês.
        data_fim: Fim do período (mesmos formatos). Com ``AAAA-MM`` o período
            termina no último dia do mês.
        cnpj: CNPJ/CPF do fornecedor. Enviado no filtro remoto ``cpf_cnpj`` da
            API (cobre todas as páginas) e revalidado localmente sobre a página.
        busca: Texto livre (nome do fornecedor, número do empenho etc.).
        inicio: Offset para paginação.
        limite: Quantidade máxima de registros na página.
        timeout: Timeout da requisição em segundos.

    Returns:
        ResultadoDespesas com os empenhos e o total declarado pelo portal para
        os filtros (não o tamanho da página).
    """
    filtros: dict[str, object] = {}
    if data_inicio:
        filtros["datamin"] = _data_br(data_inicio, campo="data_inicio")
    if data_fim:
        filtros["datamax"] = _data_br(data_fim, campo="data_fim", fim=True)
    if cnpj:
        filtros["cpf_cnpj"] = _mascarar_documento(cnpj)
    if busca:
        filtros[CAMPO_BUSCA] = busca

    itens, total = await _post(
        portal, ACAO_DESPESAS, filtros, inicio=inicio, limite=limite, timeout=timeout
    )
    despesas = [_despesa(item) for item in itens]
    if cnpj:
        alvo = _somente_digitos(cnpj)
        despesas = [d for d in despesas if alvo and alvo in _somente_digitos(d.cpf_cnpj)]
    return ResultadoDespesas(despesas=despesas, total=total)


async def buscar_contratos(
    portal: str,
    *,
    ano: int | None = None,
    tipo: str = "contrato",
    busca: str | None = None,
    inicio: int = 0,
    limite: int = DEFAULT_LIMITE,
    timeout: float = DEFAULT_TIMEOUT,
) -> ResultadoContratos:
    """Busca contratos (ou aditivos) de um portal municipal NucleoGov.

    Args:
        portal: Domínio ou URL do portal.
        ano: Ano-base do contrato.
        tipo: ``"contrato"`` ou ``"aditivo"``.
        busca: Texto livre (objeto, contratado etc.).
        inicio: Offset para paginação.
        limite: Quantidade máxima de registros na página.
        timeout: Timeout da requisição em segundos.

    Returns:
        ResultadoContratos com os contratos e o total declarado pelo portal.
    """
    filtros: dict[str, object] = {"tipo": tipo}
    if ano:
        filtros["anobase"] = str(ano)
    if busca:
        filtros[CAMPO_BUSCA] = busca

    itens, total = await _post(
        portal, ACAO_CONTRATOS, filtros, inicio=inicio, limite=limite, timeout=timeout
    )
    return ResultadoContratos(contratos=[_contrato(item) for item in itens], total=total)


async def buscar_licitacoes(
    portal: str,
    *,
    ano: int | None = None,
    dispensas: bool = False,
    busca: str | None = None,
    inicio: int = 0,
    limite: int = DEFAULT_LIMITE,
    timeout: float = DEFAULT_TIMEOUT,
) -> ResultadoLicitacoes:
    """Busca licitações de um portal municipal NucleoGov.

    Args:
        portal: Domínio ou URL do portal.
        ano: Ano-base do processo.
        dispensas: Quando ``True``, traz dispensas e inexigibilidades em vez de
            licitações regulares.
        busca: Texto livre (objeto, número do processo etc.).
        inicio: Offset para paginação.
        limite: Quantidade máxima de registros na página.
        timeout: Timeout da requisição em segundos.

    Returns:
        ResultadoLicitacoes com os processos e o total declarado pelo portal.
    """
    filtros: dict[str, object] = {
        "dispensas": 1 if dispensas else 0,
        "fracassadas_desertas": 0,
    }
    if ano:
        filtros["anobase"] = str(ano)
    if busca:
        filtros[CAMPO_BUSCA] = busca

    itens, total = await _post(
        portal, ACAO_LICITACOES, filtros, inicio=inicio, limite=limite, timeout=timeout
    )
    return ResultadoLicitacoes(licitacoes=[_licitacao(item) for item in itens], total=total)


async def buscar_legislacao(
    portal: str,
    *,
    municipio: str,
    tipo: str | None = None,
    busca: str | None = None,
    inicio: int = 0,
    limite: int = DEFAULT_LIMITE,
    timeout: float = DEFAULT_TIMEOUT,
) -> ResultadoAtos:
    """Busca leis, decretos e portarias de um portal municipal NucleoGov.

    Args:
        portal: Domínio ou URL do portal.
        municipio: Nome do município ou slug usado pelo portal na ação
            ``atos_pf_{slug}`` (ex.: ``senador_canedo``).
        tipo: Tipo do ato (``"DECRETO"``, ``"PORTARIA"``, ``"LEI"`` etc.).
        busca: Texto livre (ementa, número etc.).
        inicio: Offset para paginação.
        limite: Quantidade máxima de registros na página.
        timeout: Timeout da requisição em segundos.

    Returns:
        ResultadoAtos com os atos e o total declarado pelo portal.
    """
    acao = ACAO_ATOS_TEMPLATE.format(slug=_slug(municipio))
    filtros: dict[str, object] = {}
    if tipo:
        filtros["tipo"] = tipo.upper()
    if busca:
        filtros[CAMPO_BUSCA] = busca

    itens, total = await _post(
        portal, acao, filtros, inicio=inicio, limite=limite, timeout=timeout
    )
    return ResultadoAtos(atos=[_ato(item) for item in itens], total=total)
