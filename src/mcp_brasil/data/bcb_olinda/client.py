"""HTTP client for BCB Olinda (OData v4)."""

from __future__ import annotations

from typing import Any

from mcp_brasil._shared.http_client import http_get

from .constants import EXPECT_BASE, PTAX_BASE, TAXA_JUROS_BASE


async def _get_value(url: str, params: dict[str, Any]) -> list[dict[str, Any]]:
    # BCB Olinda does not accept '+' as space in $filter — httpx default
    # quote_plus encoding breaks OData. Manually build the query string.
    from urllib.parse import quote

    merged = {**params, "$format": "json"}
    parts = []
    for k, v in merged.items():
        parts.append(f"{quote(k, safe='$')}={quote(str(v), safe=chr(39))}")
    qs = "&".join(parts)
    full_url = f"{url}?{qs}" if qs else url
    data = await http_get(full_url, params={})
    if isinstance(data, dict):
        v = data.get("value")
        if isinstance(v, list):
            return [x for x in v if isinstance(x, dict)]
    return []


async def cotacao_dolar_dia(data_cotacao: str) -> list[dict[str, Any]]:
    """PTAX — cotação dólar em data específica (formato MM-DD-YYYY)."""
    url = f"{PTAX_BASE}/CotacaoDolarDia(dataCotacao=@dataCotacao)"
    return await _get_value(url, {"@dataCotacao": f"'{data_cotacao}'"})


async def cotacao_dolar_periodo(data_inicial: str, data_final: str) -> list[dict[str, Any]]:
    """PTAX — cotação dólar em período (MM-DD-YYYY).

    Retorna todas as cotações de abertura/intermediárias/fechamento do período.
    """
    url = (
        f"{PTAX_BASE}/CotacaoDolarPeriodo"
        "(dataInicial=@dataInicial,dataFinalCotacao=@dataFinalCotacao)"
    )
    return await _get_value(
        url,
        {
            "@dataInicial": f"'{data_inicial}'",
            "@dataFinalCotacao": f"'{data_final}'",
        },
    )


async def listar_moedas() -> list[dict[str, Any]]:
    """PTAX — lista moedas disponíveis (sigla, nome, tipo)."""
    return await _get_value(f"{PTAX_BASE}/Moedas", {"$top": "200"})


async def cotacao_moeda_dia(moeda: str, data_cotacao: str) -> list[dict[str, Any]]:
    """PTAX — cotação de uma moeda (sigla ISO, ex: 'EUR', 'GBP') em data específica."""
    url = f"{PTAX_BASE}/CotacaoMoedaDia(moeda=@moeda,dataCotacao=@dataCotacao)"
    return await _get_value(
        url,
        {"@moeda": f"'{moeda.upper()}'", "@dataCotacao": f"'{data_cotacao}'"},
    )


async def expectativas_focus(
    indicador: str,
    top: int = 30,
    filtro_data: str | None = None,
) -> list[dict[str, Any]]:
    """Expectativas Focus para um indicador.

    Args:
        indicador: Nome do indicador (ver INDICADORES_FOCUS).
        top: Máximo de pontos (padrão 30).
        filtro_data: Filtra expectativas publicadas >= esta data (YYYY-MM-DD).
    """
    url = f"{EXPECT_BASE}/ExpectativasMercadoAnuais"
    esc = indicador.replace("'", "''")
    filter_parts = [f"Indicador eq '{esc}'"]
    if filtro_data:
        filter_parts.append(f"Data ge '{filtro_data}'")
    params = {"$top": str(top), "$filter": " and ".join(filter_parts)}
    return await _get_value(url, params)


async def expectativas_focus_mensais(
    indicador: str,
    top: int = 30,
    filtro_data: str | None = None,
) -> list[dict[str, Any]]:
    """Expectativas Focus mensais para um indicador (IPCA/IGP-M/etc.)."""
    url = f"{EXPECT_BASE}/ExpectativaMercadoMensais"
    esc = indicador.replace("'", "''")
    filter_parts = [f"Indicador eq '{esc}'"]
    if filtro_data:
        filter_parts.append(f"Data ge '{filtro_data}'")
    params = {"$top": str(top), "$filter": " and ".join(filter_parts)}
    return await _get_value(url, params)


async def expectativas_focus_selic(top: int = 20) -> list[dict[str, Any]]:
    """Expectativas Focus para reuniões do Copom (Selic).

    Nota: endpoint BCB não aceita $orderby em ExpectativasMercadoSelic.
    """
    url = f"{EXPECT_BASE}/ExpectativasMercadoSelic"
    return await _get_value(url, {"$top": str(top)})


async def taxa_juros_modalidades() -> list[dict[str, Any]]:
    """Lista todas as modalidades de crédito com taxas divulgadas."""
    url = f"{TAXA_JUROS_BASE}/Modalidades"
    return await _get_value(url, {"$top": "200"})
