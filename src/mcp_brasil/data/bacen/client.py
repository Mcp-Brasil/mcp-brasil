"""HTTP client for the BCB (Banco Central) SGS API.

Ported from bcb-br-mcp/src/tools.ts fetchBcbApi + handler logic.

Endpoints:
    - /dados/serie/bcdata.sgs.{codigo}/dados              → buscar_valores
    - /dados/serie/bcdata.sgs.{codigo}/dados/ultimos/{n}   → buscar_ultimos
    - /dados/serie/bcdata.sgs.{codigo}/metadados           → buscar_metadados
"""

from __future__ import annotations

import asyncio
from typing import Any
from urllib.parse import quote

from mcp_brasil._shared.http_client import http_get

from .constants import BCB_API_BASE, INDICADORES_CHAVE
from .schemas import ExpectativaFocus, SerieMetadados, SerieValor


def _format_date(date_str: str) -> str:
    """Convert yyyy-MM-dd → dd/MM/yyyy for the BCB API."""
    if "-" in date_str and len(date_str) == 10:
        year, month, day = date_str.split("-")
        return f"{day}/{month}/{year}"
    return date_str


def _parse_valor(raw: dict[str, Any]) -> SerieValor:
    """Parse a raw API entry into SerieValor."""
    return SerieValor(data=raw["data"], valor=float(raw["valor"]))


async def buscar_valores(
    codigo: int,
    data_inicial: str | None = None,
    data_final: str | None = None,
) -> list[SerieValor]:
    """Fetch historical values for a BCB series.

    API: GET .{codigo}/dados?formato=json[&dataInicial=...&dataFinal=...]

    Args:
        codigo: SGS series code.
        data_inicial: Start date (yyyy-MM-dd or dd/MM/yyyy).
        data_final: End date (yyyy-MM-dd or dd/MM/yyyy).
    """
    url = f"{BCB_API_BASE}.{codigo}/dados"
    params: dict[str, str] = {"formato": "json"}
    if data_inicial:
        params["dataInicial"] = _format_date(data_inicial)
    if data_final:
        params["dataFinal"] = _format_date(data_final)

    data: list[dict[str, Any]] = await http_get(url, params=params)
    if not isinstance(data, list):
        return []
    return [_parse_valor(d) for d in data]


async def buscar_ultimos(codigo: int, quantidade: int = 10) -> list[SerieValor]:
    """Fetch the last N values for a BCB series.

    API: GET .{codigo}/dados/ultimos/{n}?formato=json

    Args:
        codigo: SGS series code.
        quantidade: Number of values to return (default 10).
    """
    url = f"{BCB_API_BASE}.{codigo}/dados/ultimos/{quantidade}"
    params: dict[str, str] = {"formato": "json"}

    data: list[dict[str, Any]] = await http_get(url, params=params)
    if not isinstance(data, list):
        return []
    return [_parse_valor(d) for d in data]


async def buscar_metadados(codigo: int) -> SerieMetadados:
    """Fetch metadata for a BCB series.

    API: GET .{codigo}/metadados?formato=json

    Args:
        codigo: SGS series code.
    """
    url = f"{BCB_API_BASE}.{codigo}/metadados"
    params: dict[str, str] = {"formato": "json"}

    data: dict[str, Any] = await http_get(url, params=params)

    # periodicidade may be a nested dict or plain string depending on series
    periodicidade = data.get("periodicidade", "Não informada")
    if isinstance(periodicidade, dict):
        periodicidade = periodicidade.get("descricaoPortugues", "Não informada")

    return SerieMetadados(
        codigo=data.get("codigo", codigo),
        nome=data.get("nome", f"Série {codigo}"),
        unidade=data.get("unidade", "Não informada"),
        periodicidade=str(periodicidade),
        fonte=data.get("fonte", "Banco Central do Brasil"),
        especial=data.get("especial", False),
    )


async def buscar_indicadores_atuais() -> list[dict[str, Any]]:
    """Fetch latest values for key economic indicators in parallel.

    Fetches: Selic, IPCA, IPCA 12m, Dólar PTAX, IBC-Br.
    """

    async def _fetch_one(ind: dict[str, Any]) -> dict[str, Any]:
        try:
            valores = await buscar_ultimos(ind["codigo"], 1)
            if valores:
                return {
                    "indicador": ind["nome"],
                    "codigo": ind["codigo"],
                    "data": valores[0].data,
                    "valor": valores[0].valor,
                }
            return {"indicador": ind["nome"], "codigo": ind["codigo"], "erro": "Sem dados"}
        except Exception as exc:
            return {"indicador": ind["nome"], "codigo": ind["codigo"], "erro": str(exc)}

    return list(await asyncio.gather(*[_fetch_one(ind) for ind in INDICADORES_CHAVE]))


def _build_odata_url(base: str, params: dict[str, str]) -> str:
    """Build an OData URL with literal single quotes in query parameters.

    httpx percent-encodes single quotes (``'`` -> ``%27``) in query parameter
    values, but the BCB OData API requires literal single quotes to delimit
    string values in ``$filter`` expressions.  This helper assembles the query
    string manually, encoding spaces as ``%20`` while preserving quotes.
    """
    safe = "'"
    parts = [f"{k}={quote(v, safe=safe)}" for k, v in params.items()]
    return f"{base}?{'&'.join(parts)}"


async def buscar_expectativas_focus(
    indicador: str = "IPCA",
    data_inicio: str | None = None,
    limite: int = 10,
) -> list[ExpectativaFocus]:
    """Fetch Focus survey expectations for a given indicator.

    API: GET ExpectativasMercadoAnuais?$filter=...&$top=...&$format=json

    Args:
        indicador: Economic indicator (IPCA, IGP-M, Selic, Câmbio, PIB).
        data_inicio: Minimum date for expectations (YYYY-MM-DD).
        limite: Maximum number of records (default 10).
    """
    from .constants import FOCUS_ENDPOINT, FOCUS_INDICADORES

    if indicador not in FOCUS_INDICADORES:
        return []

    odata_filter = f"Indicador eq '{indicador}'"
    if data_inicio:
        odata_filter += f" and Data ge '{data_inicio}'"

    odata_params: dict[str, str] = {
        "$filter": odata_filter,
        "$top": str(limite),
        "$format": "json",
        "$orderby": "Data desc",
    }

    # Build URL manually to preserve literal single quotes required by OData.
    # httpx would percent-encode them (%27) causing a 400 error from the BCB API.
    url = _build_odata_url(FOCUS_ENDPOINT, odata_params)
    data: dict[str, Any] = await http_get(url)
    items = data.get("value", [])
    if not isinstance(items, list):
        return []

    return [
        ExpectativaFocus(
            indicador=item.get("Indicador", indicador),
            data=item.get("Data", ""),
            data_referencia=str(item.get("DataReferencia", "")),
            media=item.get("Media"),
            mediana=item.get("Mediana"),
            desvio_padrao=item.get("DesvioPadrao"),
            minimo=item.get("Minimo"),
            maximo=item.get("Maximo"),
            base_calculo=item.get("baseCalculo"),
        )
        for item in items
    ]
