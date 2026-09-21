"""HTTP client for Megasoft municipal transparency portals.

Every portal (``{municipio}.megasofttransparencia.com.br``) exposes a read-only
JSON API under ``/api/``. Each request must carry two headers:

    - ``cliente-integrado: megasoft-portal-da-transparencia``
    - ``Authorization: Bearer <token>``

The bearer token is not a personal credential: the portal publishes it in its
own public configuration payload (``/api/configuracao/carregamento-inicial``),
so it can be obtained over plain HTTP, without a browser. When the optional
``MEGASOFT_TOKEN`` environment variable is set, it is used instead.

Response shape varies per endpoint, but the record list is always under one of
``dados``/``content``/``registros``/``items``/``itens`` and the declared total
under one of ``total``/``totalElements``/``totalRegistros``/``count``.
"""

from __future__ import annotations

import json
import os
import re
import time
from datetime import datetime
from typing import Any
from urllib.parse import urlsplit

from mcp_brasil._shared.http_client import http_get
from mcp_brasil.exceptions import HttpClientError

from .constants import (
    CLIENTE_INTEGRADO_HEADER,
    CLIENTE_INTEGRADO_VALOR,
    CONFIG_PATH,
    DEFAULT_PAGE,
    DEFAULT_PAGE_SIZE,
    EMPENHO_PATH,
    LIST_KEYS,
    MAX_PAGE_SIZE,
    ORDEM_PAGAMENTO_PATH,
    ORDEM_PAGAMENTO_TOTAIS_PATH,
    PORTAL_HOST_SUFFIX,
    RECEITA_PATH,
    TOKEN_CACHE_TTL_SECONDS,
    TOKEN_ENV_VAR,
    TOTAIS_KEYS,
    TOTAL_KEYS,
)
from .schemas import (
    Empenho,
    OrdemPagamento,
    Receita,
    ResultadoEmpenhos,
    ResultadoOrdensPagamento,
    ResultadoReceitas,
    TotaisOrdensPagamento,
)

_DATA_RE = re.compile(r"^\d{2}/\d{2}/\d{4}$")
_LABEL_RE = re.compile(r"^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$")


class RespostaInesperadaError(HttpClientError):
    """O portal respondeu 200, mas com corpo ou envelope fora do contrato.

    É subclasse de ``HttpClientError`` para que as tools já devolvam a
    mensagem clara de falha de consulta em vez de um resultado vazio.
    """


# Token cache: portal base -> (token, monotonic timestamp).
_TOKEN_CACHE: dict[str, tuple[str, float]] = {}


def limpar_cache_token() -> None:
    """Clear the in-memory token cache (used by tests)."""
    _TOKEN_CACHE.clear()


def _portal_base(municipio: str) -> str:
    """Return the HTTPS base URL of the portal.

    Accepts only a single subdomain label under ``*.megasofttransparencia.com.br``
    (e.g. ``serranopolis`` or ``serranopolis.megasofttransparencia.com.br``).
    Any other host, scheme, port, credentials, path or IP is rejected, so no
    request can ever be sent to a host outside that domain.
    """
    alvo = municipio.strip()
    if not alvo:
        raise ValueError("Informe o município (subdomínio do portal Megasoft).")

    if "://" not in alvo and "/" not in alvo and ":" not in alvo and "." not in alvo:
        label = alvo.lower()
        if not _LABEL_RE.match(label):
            raise ValueError(
                f"Subdomínio inválido: '{municipio}'. Use apenas letras, números e hífen."
            )
        return f"https://{label}{PORTAL_HOST_SUFFIX}"

    partes = urlsplit(alvo if "://" in alvo else f"https://{alvo}")
    if partes.scheme != "https":
        raise ValueError(f"Esquema não permitido: '{partes.scheme}'. Use apenas HTTPS.")
    if partes.port is not None:
        raise ValueError("Porta não é permitida no endereço do portal Megasoft.")
    if partes.username or partes.password:
        raise ValueError("Credenciais no endereço do portal Megasoft não são permitidas.")
    if partes.query or partes.fragment:
        raise ValueError("Query string ou fragmento não são permitidos no endereço do portal.")
    if partes.path not in ("", "/"):
        raise ValueError("Caminho não é permitido no endereço do portal Megasoft.")

    host = (partes.hostname or "").lower().rstrip(".")
    if not host.endswith(PORTAL_HOST_SUFFIX):
        raise ValueError(
            f"Host não permitido: '{host or municipio}'. "
            f"O portal deve estar sob '*{PORTAL_HOST_SUFFIX}'."
        )
    label = host[: -len(PORTAL_HOST_SUFFIX)]
    if "." in label or not _LABEL_RE.match(label):
        raise ValueError(
            f"Host não permitido: '{host}'. Informe um único subdomínio "
            f"sob '*{PORTAL_HOST_SUFFIX}'."
        )
    return f"https://{label}{PORTAL_HOST_SUFFIX}"


def _validar_data(valor: str, nome: str) -> str:
    """Validate a date string in DD/MM/AAAA format."""
    if not _DATA_RE.match(valor.strip()):
        raise ValueError(f"Data inválida para '{nome}': '{valor}'. Use o formato DD/MM/AAAA.")
    try:
        datetime.strptime(valor.strip(), "%d/%m/%Y")
    except ValueError as exc:
        raise ValueError(
            f"Data inválida para '{nome}': '{valor}'. Use o formato DD/MM/AAAA."
        ) from exc
    return valor.strip()


def _validar_paginacao(pagina: int, tamanho: int) -> int:
    """Validate page/size and clamp the page size to the API limit."""
    if pagina < 1:
        raise ValueError("'pagina' deve ser maior ou igual a 1.")
    if tamanho < 1:
        raise ValueError("'tamanho' deve ser maior ou igual a 1.")
    return min(tamanho, MAX_PAGE_SIZE)


def _to_float(valor: Any) -> float | None:
    """Coerce an API numeric value to float (accepts Brazilian-formatted strings)."""
    if isinstance(valor, bool):
        return None
    if isinstance(valor, (int, float)):
        return float(valor)
    if isinstance(valor, str):
        limpo = valor.strip().replace(".", "").replace(",", ".")
        try:
            return float(limpo)
        except ValueError:
            return None
    return None


def _to_int(valor: Any) -> int | None:
    """Coerce an API numeric value to int."""
    if isinstance(valor, bool):
        return None
    if isinstance(valor, int):
        return valor
    if isinstance(valor, float):
        return int(valor)
    if isinstance(valor, str):
        try:
            return int(valor.strip())
        except ValueError:
            return None
    return None


def _total(data: dict[str, Any], fallback: int) -> int:
    """Read the declared total from an envelope, falling back to the list length."""
    for chave in TOTAL_KEYS:
        valor = data.get(chave)
        if isinstance(valor, bool):
            continue
        if isinstance(valor, int):
            return valor
        if isinstance(valor, str) and valor.strip().isdigit():
            return int(valor.strip())
    return fallback


def _extrair_registros(data: Any, contexto: str) -> tuple[list[dict[str, Any]], int]:
    """Extract the record list and declared total from a variable envelope.

    Raises ``RespostaInesperadaError`` when the 200 response does not carry a
    recognizable record list — never silently treats a contract change as an
    empty result.
    """
    if isinstance(data, list):
        registros = [item for item in data if isinstance(item, dict)]
        return registros, len(registros)
    if isinstance(data, dict):
        for chave in LIST_KEYS:
            valor = data.get(chave)
            if isinstance(valor, list):
                registros = [item for item in valor if isinstance(item, dict)]
                return registros, _total(data, len(registros))
        raise RespostaInesperadaError(
            f"Resposta inesperada de {contexto}: o envelope não contém uma lista de "
            f"registros (chaves procuradas: {', '.join(LIST_KEYS)})."
        )
    raise RespostaInesperadaError(
        f"Resposta inesperada de {contexto}: esperava um objeto JSON com a lista de "
        f"registros, recebeu {type(data).__name__}."
    )


async def _get_json(
    url: str, *, params: dict[str, Any], headers: dict[str, str], contexto: str
) -> Any:
    """GET returning parsed JSON, converting a malformed body into a clear error."""
    try:
        return await http_get(url, params=params, headers=headers)
    except json.JSONDecodeError as exc:
        raise RespostaInesperadaError(f"Resposta de {contexto} não é JSON válido: {exc}.") from exc


async def _obter_token(portal: str) -> str:
    """Return the portal bearer token, from env or from the public config endpoint."""
    override = os.environ.get(TOKEN_ENV_VAR, "").strip()
    if override:
        return override

    cached = _TOKEN_CACHE.get(portal)
    if cached is not None and time.monotonic() - cached[1] < TOKEN_CACHE_TTL_SECONDS:
        return cached[0]

    data = await _get_json(
        f"{portal}/api/{CONFIG_PATH}", params={}, headers={}, contexto=CONFIG_PATH
    )
    token = data.get("tokenPortal") if isinstance(data, dict) else None
    if not isinstance(token, str) or not token.strip():
        raise RuntimeError(
            "O portal não retornou 'tokenPortal' em /api/configuracao/carregamento-inicial. "
            f"Defina a variável de ambiente {TOKEN_ENV_VAR} com um token válido."
        )
    _TOKEN_CACHE[portal] = (token.strip(), time.monotonic())
    return token.strip()


def _headers(token: str) -> dict[str, str]:
    """Build the required headers for a Megasoft data call."""
    return {
        CLIENTE_INTEGRADO_HEADER: CLIENTE_INTEGRADO_VALOR,
        "Authorization": f"Bearer {token}",
    }


async def _consultar_pagina(
    portal: str,
    path: str,
    *,
    data_inicial: str,
    data_final: str,
    pagina: int,
    tamanho: int,
) -> tuple[list[dict[str, Any]], int]:
    """Fetch a single page from a paginated endpoint."""
    token = await _obter_token(portal)
    params = {
        "dataInicial": data_inicial,
        "dataFinal": data_final,
        "pagina": pagina,
        "tamanhoDaPagina": tamanho,
    }
    data = await _get_json(
        f"{portal}/api/{path}", params=params, headers=_headers(token), contexto=path
    )
    return _extrair_registros(data, path)


def _parse_empenho(item: dict[str, Any]) -> Empenho:
    return Empenho(
        codigo=_to_int(item.get("codigo")),
        numero_tcm=_to_int(item.get("numeroDoTcm")),
        data=item.get("data"),
        fornecedor=item.get("nomeDoFornecedor"),
        cnpj_fornecedor=item.get("cnpjENomeDoFornecedor"),
        classificacao_orcamentaria=item.get("classificacaoOrcamentaria"),
        valor_empenho=_to_float(item.get("valorDoEmpenho")),
        valor_liquidacao=_to_float(item.get("valorDaLiquidacao")),
        valor_pagamento=_to_float(item.get("valorDoPagamento")),
        orgao=item.get("codigoENomeDoOrgao"),
        unidade=item.get("codigoENomeDaUnidade"),
        funcao=item.get("codigoENomeDaFuncao"),
        natureza_despesa=item.get("codigoENomeDaNaturezaDespesa"),
        fonte=item.get("tituloDaFonte"),
        historico=item.get("historico"),
    )


def _parse_ordem(item: dict[str, Any]) -> OrdemPagamento:
    return OrdemPagamento(
        codigo=_to_int(item.get("codigo")),
        numero=_to_int(item.get("numero")),
        orgao=item.get("nomeDoOrgao"),
        data_pagamento=item.get("dataPagamento"),
        fornecedor=item.get("nomeDoFornecedor"),
        cpf_cnpj_fornecedor=item.get("cpfOuCnpjDoFornecedor"),
        numero_empenho=_to_int(item.get("numeroDoEmpenho")),
        numero_liquidacao=_to_int(item.get("numeroDaLiquidacao")),
        valor_pago=_to_float(item.get("valorPago")),
        valor_anulacao=_to_float(item.get("valorAnulacao")),
        valor_total=_to_float(item.get("valorTotal")),
        valor_retencao=_to_float(item.get("valorRetencao")),
        fase=item.get("fase"),
    )


def _parse_receita(item: dict[str, Any]) -> Receita:
    chave = item.get("chave")
    chave = chave if isinstance(chave, dict) else {}
    return Receita(
        codigo_receita=_to_int(chave.get("codigoDaReceita")),
        ano=_to_int(chave.get("ano")),
        mes=str(chave["mes"]) if chave.get("mes") is not None else None,
        codigo_orgao=_to_int(chave.get("codigoDoOrgao")),
        fonte=item.get("fonteDaReceita"),
        receita=item.get("receita"),
        natureza=item.get("naturezaDaReceita"),
        origem_recurso=item.get("origemDoRecurso"),
        valor_orcado=_to_float(item.get("valorOrcado")),
        arrecadacao=_to_float(item.get("arrecadacao")),
        valor_acumulado=_to_float(item.get("valorAcumulado")),
    )


async def buscar_empenhos(
    municipio: str,
    *,
    data_inicial: str,
    data_final: str,
    pagina: int = DEFAULT_PAGE,
    tamanho: int = DEFAULT_PAGE_SIZE,
) -> ResultadoEmpenhos:
    """Buscar empenhos de um município em um período (DD/MM/AAAA)."""
    portal = _portal_base(municipio)
    ini = _validar_data(data_inicial, "data_inicial")
    fim = _validar_data(data_final, "data_final")
    tam = _validar_paginacao(pagina, tamanho)
    registros, total = await _consultar_pagina(
        portal,
        EMPENHO_PATH,
        data_inicial=ini,
        data_final=fim,
        pagina=pagina,
        tamanho=tam,
    )
    return ResultadoEmpenhos(
        empenhos=[_parse_empenho(item) for item in registros],
        total=total,
        pagina=pagina,
    )


async def buscar_ordens_pagamento(
    municipio: str,
    *,
    data_inicial: str,
    data_final: str,
    pagina: int = DEFAULT_PAGE,
    tamanho: int = DEFAULT_PAGE_SIZE,
) -> ResultadoOrdensPagamento:
    """Buscar ordens de pagamento de um município em um período (DD/MM/AAAA)."""
    portal = _portal_base(municipio)
    ini = _validar_data(data_inicial, "data_inicial")
    fim = _validar_data(data_final, "data_final")
    tam = _validar_paginacao(pagina, tamanho)
    registros, total = await _consultar_pagina(
        portal,
        ORDEM_PAGAMENTO_PATH,
        data_inicial=ini,
        data_final=fim,
        pagina=pagina,
        tamanho=tam,
    )
    return ResultadoOrdensPagamento(
        ordens=[_parse_ordem(item) for item in registros],
        total=total,
        pagina=pagina,
    )


async def buscar_receitas(
    municipio: str,
    *,
    data_inicial: str,
    data_final: str,
    pagina: int = DEFAULT_PAGE,
    tamanho: int = DEFAULT_PAGE_SIZE,
) -> ResultadoReceitas:
    """Buscar receitas de um município em um período (DD/MM/AAAA)."""
    portal = _portal_base(municipio)
    ini = _validar_data(data_inicial, "data_inicial")
    fim = _validar_data(data_final, "data_final")
    tam = _validar_paginacao(pagina, tamanho)
    registros, total = await _consultar_pagina(
        portal,
        RECEITA_PATH,
        data_inicial=ini,
        data_final=fim,
        pagina=pagina,
        tamanho=tam,
    )
    return ResultadoReceitas(
        receitas=[_parse_receita(item) for item in registros],
        total=total,
        pagina=pagina,
    )


async def totais_ordens_pagamento(
    municipio: str,
    *,
    data_inicial: str,
    data_final: str,
) -> TotaisOrdensPagamento:
    """Buscar os totais financeiros das ordens de pagamento de um período."""
    portal = _portal_base(municipio)
    ini = _validar_data(data_inicial, "data_inicial")
    fim = _validar_data(data_final, "data_final")
    token = await _obter_token(portal)
    params = {
        "dataInicial": ini,
        "dataFinal": fim,
        "pagina": DEFAULT_PAGE,
        "tamanhoDaPagina": DEFAULT_PAGE_SIZE,
    }
    data = await _get_json(
        f"{portal}/api/{ORDEM_PAGAMENTO_TOTAIS_PATH}",
        params=params,
        headers=_headers(token),
        contexto=ORDEM_PAGAMENTO_TOTAIS_PATH,
    )
    if not isinstance(data, dict) or not any(chave in data for chave in TOTAIS_KEYS):
        raise RespostaInesperadaError(
            "Resposta inesperada de "
            f"{ORDEM_PAGAMENTO_TOTAIS_PATH}: o envelope não contém os totais "
            f"esperados (chaves procuradas: {', '.join(TOTAIS_KEYS)})."
        )
    return TotaisOrdensPagamento(
        valor_pago=_to_float(data.get("valorPago")),
        valor_anulacao=_to_float(data.get("valorAnulacao")),
        valor_total=_to_float(data.get("valorTotal")),
        valor_retencao=_to_float(data.get("valorRetencao")),
    )
