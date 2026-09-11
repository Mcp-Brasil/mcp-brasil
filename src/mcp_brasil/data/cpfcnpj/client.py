"""Cliente HTTP da API CPF.CNPJ.

Formato: ``GET https://api.cpfcnpj.com.br/{token}/{pacote}/{documento}``.
Sucesso traz ``status: 1``; erros de negócio trazem ``status: 0`` + ``erroCodigo``;
erros de gateway trazem ``status: "error"`` + ``message``; rotas inválidas podem
devolver HTML. Os três casos são tratados e o token nunca aparece em log,
exceção ou mensagem (a URL é mascarada como ``/***/{pacote}/{documento}``).

Desvio consciente do molde do PR #142: ``create_client`` do ``_shared`` NÃO faz
retry (quem faz é ``http_get``), então este cliente replica a política de retry
do ``http_get`` (mesmos status 429/5xx, mesmo backoff exponencial e os mesmos
settings ``HTTP_MAX_RETRIES``/``HTTP_BACKOFF_BASE``). Não reusamos ``http_get``
porque ele coloca a URL completa (com o token) na mensagem de ``HttpClientError``
e trunca o corpo do 400, o que vazaria o token e perderia o detalhe do erro. O
RateLimiter compartilhado de módulo é adquirido uma vez por chamada lógica, não
por tentativa de retry.
"""

from __future__ import annotations

import asyncio
import json
import logging
import os
from typing import Any

import httpx

from mcp_brasil._shared.http_client import create_client
from mcp_brasil._shared.rate_limiter import RateLimiter
from mcp_brasil.settings import HTTP_BACKOFF_BASE, HTTP_MAX_RETRIES

from .constants import (
    CPFCNPJ_BASE,
    MAX_REQ_POR_SEGUNDO,
    MENSAGENS_ERRO,
    PACOTE_INSCRICOES,
    SEGMENTO_SALDO,
)
from .schemas import EmpresaCnpj, Saldo

logger = logging.getLogger(__name__)

# Espelha ``_shared.http_client._RETRYABLE_STATUS_CODES`` (privado, prefixo _):
# como não há símbolo público para importar, mantemos o frozenset local.
_RETRYABLE_STATUS = frozenset({429, 500, 502, 503, 504})

# RateLimiter compartilhado (singleton de módulo): 20 req/s, teto da API.
_rate_limiter = RateLimiter(max_requests=MAX_REQ_POR_SEGUNDO, period=1.0)


class CpfCnpjError(Exception):
    """Erro de consulta à API CPF.CNPJ, com código e mensagem amigável."""

    def __init__(self, codigo: int, mensagem: str) -> None:
        self.codigo = codigo
        self.mensagem = mensagem
        super().__init__(mensagem)


def _token() -> str:
    """Retorna o token da conta CPF.CNPJ a partir do ambiente."""
    return os.environ.get("CPFCNPJ_TOKEN", "").strip()


def _url_mascarada(pacote: int | str, documento: str) -> str:
    """URL com o token substituído por ``***`` (para logs e exceções)."""
    return f"{CPFCNPJ_BASE}/***/{pacote}/{documento}"


def _para_int(valor: Any) -> int | None:
    """Coage ``status``/``erroCodigo`` para int; a API pode devolvê-los como string.

    Aceita ``int`` e string numérica (com bordas), devolvendo ``None`` para
    qualquer outro valor (inclusive ``bool``, que é subtipo de ``int``).
    """
    if isinstance(valor, bool):
        return None
    if isinstance(valor, int):
        return valor
    if isinstance(valor, str):
        texto = valor.strip()
        if texto.lstrip("-").isdigit():
            return int(texto)
    return None


def _tratar_resposta(resp: httpx.Response, url_mascarada: str) -> dict[str, Any]:
    """Interpreta o corpo da resposta e mapeia os dois formatos de erro."""
    try:
        data = resp.json()
    except (json.JSONDecodeError, ValueError):
        raise CpfCnpjError(
            400,
            "Resposta inválida da API CPF.CNPJ (verifique o documento e o pacote). "
            f"Requisição: {url_mascarada}",
        ) from None
    if not isinstance(data, dict):
        raise CpfCnpjError(
            400, f"Resposta inesperada da API CPF.CNPJ. Requisição: {url_mascarada}"
        )

    status = _para_int(data.get("status"))
    if status == 1:
        return data
    if status == 0:
        cod_int = _para_int(data.get("erroCodigo")) or 0
        mensagem = (
            MENSAGENS_ERRO.get(cod_int) or data.get("erro") or "Consulta CPF.CNPJ sem sucesso."
        )
        raise CpfCnpjError(cod_int, mensagem)
    # status == "error" (gateway) ou ausente: erro de formatação da requisição.
    mensagem = data.get("message") or data.get("erro") or "Requisição inválida à API CPF.CNPJ."
    raise CpfCnpjError(400, str(mensagem))


async def _get(pacote: int | str, documento: str) -> dict[str, Any]:
    """GET rate-limitado com retry em erros transitórios; token mascarado."""
    token = _token()
    if not token:
        raise CpfCnpjError(
            1000,
            "CPFCNPJ_TOKEN vazio ou inválido. Configure um token válido para "
            "consultar a fonte CPF.CNPJ.",
        )

    caminho = f"{CPFCNPJ_BASE}/{token}/{pacote}/{documento}"
    url_mascarada = _url_mascarada(pacote, documento)
    retries = HTTP_MAX_RETRIES

    async with _rate_limiter, create_client() as client:
        for tentativa in range(retries + 1):
            ultima = tentativa >= retries
            try:
                resp = await client.get(caminho)
            except (httpx.TimeoutException, httpx.ConnectError) as exc:
                if ultima:
                    raise CpfCnpjError(
                        1006, "Falha de conexão com a API CPF.CNPJ. Tente novamente."
                    ) from exc
                await asyncio.sleep(HTTP_BACKOFF_BASE * (2**tentativa))
                continue

            if resp.status_code in _RETRYABLE_STATUS:
                if ultima:
                    raise CpfCnpjError(
                        1007,
                        f"API CPF.CNPJ indisponível (HTTP {resp.status_code}) após "
                        f"{retries + 1} tentativas. Requisição: {url_mascarada}",
                    )
                await asyncio.sleep(HTTP_BACKOFF_BASE * (2**tentativa))
                continue

            return _tratar_resposta(resp, url_mascarada)

    # Só alcançável com retries negativo (range vazio): guarda o mypy contra
    # "missing return"; em operação normal todo caminho retorna ou levanta acima.
    raise CpfCnpjError(1006, "Falha de conexão com a API CPF.CNPJ.")


async def consultar_cnpj(documento: str, pacote: int) -> EmpresaCnpj:
    """Consulta um CNPJ no pacote informado (5 ou 6). O documento já vem normalizado."""
    data = await _get(pacote, documento)
    return EmpresaCnpj(**data)


async def inscricoes_estaduais(documento: str) -> EmpresaCnpj:
    """Consulta as inscrições estaduais de um CNPJ (pacote 16)."""
    data = await _get(PACOTE_INSCRICOES, documento)
    return EmpresaCnpj(**data)


async def saldo(pacote: int) -> Saldo:
    """Consulta o saldo de créditos de um pacote (endpoint gratuito)."""
    data = await _get(SEGMENTO_SALDO, str(pacote))
    pacote_obj = data.get("pacote")
    if not isinstance(pacote_obj, dict):
        pacote_obj = {}
    return Saldo(**pacote_obj)
