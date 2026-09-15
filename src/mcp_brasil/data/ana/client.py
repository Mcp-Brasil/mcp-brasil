"""HTTP client for the ANA API (Hidroweb + SAR).

Endpoints:
    - /estacao/codigoestacao         → buscar_estacoes
    - /estacao/telemetrica           → consultar_telemetria
    - SAR /Medicao                   → monitorar_reservatorios
"""

from __future__ import annotations

import os
from typing import Any

from mcp_brasil._shared.http_client import http_get

from .constants import (
    AUTH_ENV_VAR,
    AUTH_HEADER_NAME,
    ESTACOES_URL,
    RESERVATORIOS_URL,
    TELEMETRIA_URL,
)
from .schemas import DadoTelemetria, Estacao, Reservatorio


def _auth_headers() -> dict[str, str] | None:
    """Return auth headers when ANA_HIDROWEB_TOKEN is set, else None.

    The Hidroweb/SNIRH API started requiring a token on previously public
    endpoints (issue #37). The token is optional: anonymous requests keep
    working wherever the API still allows them.
    """
    token = os.environ.get(AUTH_ENV_VAR, "").strip()
    if not token:
        return None
    return {AUTH_HEADER_NAME: f"Bearer {token}"}


async def buscar_estacoes(
    codigo_estacao: str | None = None,
    codigo_rio: str | None = None,
    codigo_bacia: str | None = None,
    codigo_sub_bacia: str | None = None,
    nome_estacao: str | None = None,
    tipo_estacao: int | None = None,
) -> list[Estacao]:
    """Search hydrological stations on Hidroweb.

    Args:
        codigo_estacao: Station code (e.g. "60435000").
        codigo_rio: River code.
        codigo_bacia: Basin code.
        codigo_sub_bacia: Sub-basin code.
        nome_estacao: Station name (partial match).
        tipo_estacao: Station type (1=fluvio, 2=pluvio).

    Returns:
        List of matching stations.
    """
    params: dict[str, str] = {}
    if codigo_estacao:
        params["codigoEstacao"] = codigo_estacao
    if codigo_rio:
        params["codigoRio"] = codigo_rio
    if codigo_bacia:
        params["codigoBacia"] = codigo_bacia
    if codigo_sub_bacia:
        params["codigoSubBacia"] = codigo_sub_bacia
    if nome_estacao:
        params["nomeEstacao"] = nome_estacao
    if tipo_estacao is not None:
        params["tipoEstacao"] = str(tipo_estacao)

    data: list[dict[str, Any]] = await http_get(
        ESTACOES_URL, params=params or None, headers=_auth_headers()
    )

    return [
        Estacao(
            codigo_estacao=str(e.get("codigoEstacao", "")),
            nome_estacao=e.get("nomeEstacao", ""),
            codigo_rio=str(e.get("codigoRio", "")),
            nome_rio=e.get("nomeRio", ""),
            bacia=e.get("bacia", ""),
            sub_bacia=e.get("subBacia", ""),
            municipio=e.get("municipio", ""),
            estado=e.get("estado", ""),
            latitude=e.get("latitude"),
            longitude=e.get("longitude"),
            tipo_estacao=e.get("tipoEstacao", ""),
            responsavel=e.get("responsavel", ""),
        )
        for e in data
    ]


async def consultar_telemetria(
    codigo_estacao: str,
    data_inicio: str | None = None,
    data_fim: str | None = None,
) -> list[DadoTelemetria]:
    """Fetch telemetry data for a station.

    Args:
        codigo_estacao: Station code (e.g. "60435000").
        data_inicio: Start date in dd/MM/yyyy format.
        data_fim: End date in dd/MM/yyyy format.

    Returns:
        List of telemetry readings.
    """
    params: dict[str, str] = {"codigoEstacao": codigo_estacao}
    if data_inicio:
        params["dataInicio"] = data_inicio
    if data_fim:
        params["dataFim"] = data_fim

    data: list[dict[str, Any]] = await http_get(
        TELEMETRIA_URL, params=params, headers=_auth_headers()
    )

    return [
        DadoTelemetria(
            codigo_estacao=str(d.get("codigoEstacao", codigo_estacao)),
            data_hora=d.get("dataHora", ""),
            nivel=d.get("nivel"),
            vazao=d.get("vazao"),
            chuva=d.get("chuva"),
        )
        for d in data
    ]


async def monitorar_reservatorios(
    codigo_reservatorio: str | None = None,
    data_inicio: str | None = None,
    data_fim: str | None = None,
) -> list[Reservatorio]:
    """Fetch reservoir monitoring data from SAR/ANA.

    Args:
        codigo_reservatorio: Reservoir code (optional).
        data_inicio: Start date in dd/MM/yyyy format.
        data_fim: End date in dd/MM/yyyy format.

    Returns:
        List of reservoir measurements.
    """
    params: dict[str, str] = {}
    if codigo_reservatorio:
        params["codigoReservatorio"] = codigo_reservatorio
    if data_inicio:
        params["dataInicio"] = data_inicio
    if data_fim:
        params["dataFim"] = data_fim

    data: list[dict[str, Any]] = await http_get(
        RESERVATORIOS_URL, params=params or None, headers=_auth_headers()
    )

    return [
        Reservatorio(
            nome_reservatorio=r.get("nomeReservatorio", ""),
            rio=r.get("rio", ""),
            estado=r.get("estado", ""),
            data=r.get("data", ""),
            volume_util=r.get("volumeUtil"),
            cota_atual=r.get("cotaAtual"),
            vazao_afluente=r.get("vazaoAfluente"),
            vazao_defluente=r.get("vazaoDefluente"),
        )
        for r in data
    ]
