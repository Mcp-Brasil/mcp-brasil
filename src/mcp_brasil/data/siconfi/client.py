"""Async HTTP client for the SICONFI API (Tesouro Nacional).

Respects the documented 1 req/s rate limit via a shared RateLimiter.
Endpoints follow the ORDS REST pattern: base + /<resource>?params.

Docs: https://apidatalake.tesouro.gov.br/docs/siconfi/
"""

from __future__ import annotations

from typing import Any

from mcp_brasil._shared.http_client import http_get
from mcp_brasil._shared.rate_limiter import RateLimiter

from .constants import (
    RATE_LIMIT_MAX_REQUESTS,
    RATE_LIMIT_PERIOD_SECONDS,
    SICONFI_API_BASE,
)
from .schemas import AnexoRelatorio, Ente, ItemDeclaracao

# Module-level limiter: shared across tool calls, 1 req/s.
_limiter = RateLimiter(
    max_requests=RATE_LIMIT_MAX_REQUESTS,
    period=RATE_LIMIT_PERIOD_SECONDS,
)


async def _get_items(path: str, params: dict[str, Any]) -> list[dict[str, Any]]:
    """Low-level GET to a SICONFI endpoint, returning the ``items`` list."""
    url = f"{SICONFI_API_BASE}{path}"
    clean = {k: v for k, v in params.items() if v is not None}
    async with _limiter:
        data = await http_get(url, params=clean)
    if isinstance(data, dict):
        items = data.get("items")
        if isinstance(items, list):
            return [i for i in items if isinstance(i, dict)]
    return []


async def listar_entes() -> list[Ente]:
    """GET /entes — full list of declarant entities (UF, IBGE code, population)."""
    rows = await _get_items("/entes", {})
    return [Ente.model_validate(r) for r in rows]


async def consultar_rreo(
    *,
    an_exercicio: int,
    nr_periodo: int,
    co_tipo_demonstrativo: str,
    id_ente: int,
    no_anexo: str | None = None,
    co_esfera: str | None = None,
) -> list[ItemDeclaracao]:
    """GET /rreo — Relatório Resumido de Execução Orçamentária.

    Args:
        an_exercicio: Exercício (ano) do relatório.
        nr_periodo: Bimestre (1-6).
        co_tipo_demonstrativo: "RREO" ou "RREO Simplificado".
        id_ente: Código IBGE do ente.
        no_anexo: Filtra por anexo específico (ex: "RREO-Anexo 03").
        co_esfera: Filtra por esfera (U/E/M/D).
    """
    rows = await _get_items(
        "/rreo",
        {
            "an_exercicio": an_exercicio,
            "nr_periodo": nr_periodo,
            "co_tipo_demonstrativo": co_tipo_demonstrativo,
            "id_ente": id_ente,
            "no_anexo": no_anexo,
            "co_esfera": co_esfera,
        },
    )
    return [ItemDeclaracao.model_validate(r) for r in rows]


async def consultar_rgf(
    *,
    an_exercicio: int,
    in_periodicidade: str,
    nr_periodo: int,
    co_tipo_demonstrativo: str,
    co_poder: str,
    id_ente: int,
    no_anexo: str | None = None,
    co_esfera: str | None = None,
) -> list[ItemDeclaracao]:
    """GET /rgf — Relatório de Gestão Fiscal (LRF).

    Args:
        an_exercicio: Exercício (ano).
        in_periodicidade: "Q" (quadrimestral) ou "S" (semestral).
        nr_periodo: Número do período (1-3 quadrimestral, 1-2 semestral).
        co_tipo_demonstrativo: "RGF" ou "RGF Simplificado".
        co_poder: E/L/J/M/D (Executivo, Legislativo, Judiciário, MP, Defensoria).
        id_ente: Código IBGE do ente.
        no_anexo: Filtra por anexo.
        co_esfera: Filtra por esfera.
    """
    rows = await _get_items(
        "/rgf",
        {
            "an_exercicio": an_exercicio,
            "in_periodicidade": in_periodicidade,
            "nr_periodo": nr_periodo,
            "co_tipo_demonstrativo": co_tipo_demonstrativo,
            "co_poder": co_poder,
            "id_ente": id_ente,
            "no_anexo": no_anexo,
            "co_esfera": co_esfera,
        },
    )
    return [ItemDeclaracao.model_validate(r) for r in rows]


async def consultar_dca(
    *,
    an_exercicio: int,
    id_ente: int,
    no_anexo: str | None = None,
) -> list[ItemDeclaracao]:
    """GET /dca — Declaração de Contas Anuais."""
    rows = await _get_items(
        "/dca",
        {"an_exercicio": an_exercicio, "id_ente": id_ente, "no_anexo": no_anexo},
    )
    return [ItemDeclaracao.model_validate(r) for r in rows]


async def consultar_extrato_entregas(
    *,
    id_ente: int,
    an_referencia: int,
) -> list[ItemDeclaracao]:
    """GET /extrato_entregas — status de entregas declaratórias do ente no ano."""
    rows = await _get_items(
        "/extrato_entregas",
        {"id_ente": id_ente, "an_referencia": an_referencia},
    )
    return [ItemDeclaracao.model_validate(r) for r in rows]


async def listar_anexos() -> list[AnexoRelatorio]:
    """GET /anexos-relatorios — catálogo de anexos disponíveis por esfera."""
    rows = await _get_items("/anexos-relatorios", {})
    return [AnexoRelatorio.model_validate(r) for r in rows]
