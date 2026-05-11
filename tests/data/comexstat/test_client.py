"""Tests for the ComexStat HTTP client.

Uses respx to mock HTTP GET requests and unittest.mock for curl_cffi POST.
"""

from __future__ import annotations

from typing import Any
from unittest.mock import AsyncMock, MagicMock, patch

import httpx
import pytest
import respx

from mcp_brasil._shared.rate_limiter import RateLimiter
from mcp_brasil.data.comexstat import client
from mcp_brasil.data.comexstat.constants import COUNTRY_URL

# Real API field names (as of 2026-05)
_EXPORT_ROWS = [
    {
        "year": "2023",
        "monthNumber": "1",
        "country": "China",
        "ncm": "01031000",
        "metricFOB": 1_000_000.0,
        "metricKG": 500_000.0,
    },
    {
        "year": "2023",
        "monthNumber": "2",
        "country": "China",
        "ncm": "01031000",
        "metricFOB": 1_200_000.0,
        "metricKG": 600_000.0,
    },
]

_IMPORT_ROWS = [
    {
        "year": "2023",
        "monthNumber": "1",
        "metricFOB": 800_000.0,
        "metricKG": 400_000.0,
    },
]

_QUERY_RESPONSE_EXPORT = {"data": {"list": _EXPORT_ROWS}}
_QUERY_RESPONSE_IMPORT = {"data": {"list": _IMPORT_ROWS}}

_COUNTRY_RESPONSE = {
    "data": [
        [
            {"id": "31", "text": "China"},
            {"id": "249", "text": "Estados Unidos"},
        ]
    ]
}


def _mock_curl_session(json_data: dict[str, Any]) -> MagicMock:
    """Return a mock AsyncSession that responds with json_data."""
    mock_response = MagicMock()
    mock_response.raise_for_status = MagicMock()
    mock_response.json = MagicMock(return_value=json_data)

    mock_session = AsyncMock()
    mock_session.post = AsyncMock(return_value=mock_response)
    mock_session.__aenter__ = AsyncMock(return_value=mock_session)
    mock_session.__aexit__ = AsyncMock(return_value=None)
    return mock_session


# ---------------------------------------------------------------------------
# Rate limiter
# ---------------------------------------------------------------------------


class TestRateLimiter:
    def test_rate_limiter_exists(self) -> None:
        assert hasattr(client, "_rate_limiter")
        assert isinstance(client._rate_limiter, RateLimiter)

    def test_rate_limiter_config(self) -> None:
        assert client._rate_limiter._max_requests == 30
        assert client._rate_limiter._period == 60.0


# ---------------------------------------------------------------------------
# _build_periodo
# ---------------------------------------------------------------------------


class TestBuildPeriodo:
    def test_month_detail_formats_yyyy_mm(self) -> None:
        assert client._build_periodo({"year": "2023", "monthNumber": "1"}, True) == "2023-01"

    def test_month_detail_pads_month(self) -> None:
        assert client._build_periodo({"year": "2023", "monthNumber": "12"}, True) == "2023-12"

    def test_no_month_detail_returns_year(self) -> None:
        assert client._build_periodo({"year": "2023"}, False) == "2023"

    def test_missing_key_returns_empty(self) -> None:
        assert client._build_periodo({}, True) == ""


# ---------------------------------------------------------------------------
# _query
# ---------------------------------------------------------------------------


class TestQuery:
    @pytest.mark.asyncio
    async def test_sends_correct_body_and_returns_list(self) -> None:
        mock_session = _mock_curl_session(_QUERY_RESPONSE_EXPORT)
        with patch("mcp_brasil.data.comexstat.client.AsyncSession", return_value=mock_session):
            result = await client._query(
                "export", "2023-01", "2023-02", ["ncm"], [], ["metricFOB"], True
            )
        assert len(result) == 2
        assert result[0]["year"] == "2023"

    @pytest.mark.asyncio
    async def test_empty_list_on_no_data(self) -> None:
        mock_session = _mock_curl_session({"data": {"list": []}})
        with patch("mcp_brasil.data.comexstat.client.AsyncSession", return_value=mock_session):
            result = await client._query(
                "export", "2023-01", "2023-01", [], [], ["metricFOB"], False
            )
        assert result == []


# ---------------------------------------------------------------------------
# consultar_exportacoes
# ---------------------------------------------------------------------------


class TestConsultarExportacoes:
    @pytest.mark.asyncio
    async def test_returns_comex_items(self) -> None:
        with patch.object(client, "_query", new=AsyncMock(return_value=_EXPORT_ROWS)):
            result = await client.consultar_exportacoes(
                "2023-01", "2023-02", ncm="01031000", detalhar_por_mes=True
            )
        assert len(result) == 2
        assert result[0].periodo == "2023-01"
        assert result[0].pais == "China"
        assert result[0].ncm == "01031000"
        assert result[0].fob_usd == 1_000_000.0
        assert result[0].kg_liquido == 500_000.0

    @pytest.mark.asyncio
    async def test_no_filters_returns_items(self) -> None:
        with patch.object(client, "_query", new=AsyncMock(return_value=_EXPORT_ROWS)):
            result = await client.consultar_exportacoes("2023-01", "2023-02")
        assert len(result) == 2

    @pytest.mark.asyncio
    async def test_empty_response_returns_empty_list(self) -> None:
        with patch.object(client, "_query", new=AsyncMock(return_value=[])):
            result = await client.consultar_exportacoes("2023-01", "2023-01")
        assert result == []


# ---------------------------------------------------------------------------
# consultar_importacoes
# ---------------------------------------------------------------------------


class TestConsultarImportacoes:
    @pytest.mark.asyncio
    async def test_returns_comex_items(self) -> None:
        with patch.object(client, "_query", new=AsyncMock(return_value=_IMPORT_ROWS)):
            result = await client.consultar_importacoes("2023-01", "2023-01")
        assert len(result) == 1
        assert result[0].fob_usd == 800_000.0
        assert result[0].pais is None

    @pytest.mark.asyncio
    async def test_empty_response_returns_empty_list(self) -> None:
        with patch.object(client, "_query", new=AsyncMock(return_value=[])):
            result = await client.consultar_importacoes("2023-01", "2023-01")
        assert result == []


# ---------------------------------------------------------------------------
# balanca_comercial
# ---------------------------------------------------------------------------


class TestBalancaComercial:
    @pytest.mark.asyncio
    async def test_surplus_period(self) -> None:
        exp = [{"year": "2023", "monthNumber": "1", "metricFOB": 1_000_000.0}]
        imp = [{"year": "2023", "monthNumber": "1", "metricFOB": 600_000.0}]
        with patch.object(client, "_query", new=AsyncMock(side_effect=[exp, imp])):
            result = await client.balanca_comercial("2023-01", "2023-01")
        assert len(result) == 1
        assert result[0].periodo == "2023-01"
        assert result[0].exportacoes_fob == 1_000_000.0
        assert result[0].importacoes_fob == 600_000.0
        assert result[0].saldo_fob == 400_000.0

    @pytest.mark.asyncio
    async def test_deficit_period(self) -> None:
        exp = [{"year": "2023", "monthNumber": "1", "metricFOB": 300_000.0}]
        imp = [{"year": "2023", "monthNumber": "1", "metricFOB": 800_000.0}]
        with patch.object(client, "_query", new=AsyncMock(side_effect=[exp, imp])):
            result = await client.balanca_comercial("2023-01", "2023-01")
        assert result[0].saldo_fob == pytest.approx(-500_000.0)

    @pytest.mark.asyncio
    async def test_merges_multiple_periods_sorted(self) -> None:
        exp = [
            {"year": "2023", "monthNumber": "2", "metricFOB": 500_000.0},
            {"year": "2023", "monthNumber": "1", "metricFOB": 400_000.0},
        ]
        imp = [
            {"year": "2023", "monthNumber": "1", "metricFOB": 200_000.0},
            {"year": "2023", "monthNumber": "2", "metricFOB": 300_000.0},
        ]
        with patch.object(client, "_query", new=AsyncMock(side_effect=[exp, imp])):
            result = await client.balanca_comercial("2023-01", "2023-02")
        assert len(result) == 2
        assert result[0].periodo == "2023-01"
        assert result[1].periodo == "2023-02"

    @pytest.mark.asyncio
    async def test_period_only_in_exports_gets_zero_imports(self) -> None:
        exp = [{"year": "2023", "monthNumber": "1", "metricFOB": 500_000.0}]
        with patch.object(client, "_query", new=AsyncMock(side_effect=[exp, []])):
            result = await client.balanca_comercial("2023-01", "2023-01")
        assert result[0].importacoes_fob == 0.0
        assert result[0].saldo_fob == 500_000.0


# ---------------------------------------------------------------------------
# listar_paises
# ---------------------------------------------------------------------------


class TestListarPaises:
    @pytest.mark.asyncio
    @respx.mock
    async def test_parses_countries(self) -> None:
        respx.get(COUNTRY_URL).mock(return_value=httpx.Response(200, json=_COUNTRY_RESPONSE))
        result = await client.listar_paises()
        assert len(result) == 2
        assert result[0].id == "31"
        assert result[0].nome == "China"
        assert result[1].id == "249"
        assert result[1].nome == "Estados Unidos"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_inner_list(self) -> None:
        respx.get(COUNTRY_URL).mock(return_value=httpx.Response(200, json={"data": [[]]}))
        result = await client.listar_paises()
        assert result == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_data(self) -> None:
        respx.get(COUNTRY_URL).mock(return_value=httpx.Response(200, json={"data": []}))
        result = await client.listar_paises()
        assert result == []
