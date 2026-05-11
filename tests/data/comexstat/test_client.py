"""Tests for the ComexStat HTTP client.

Uses respx to mock HTTP requests and verify response parsing.
"""

from __future__ import annotations

import httpx
import pytest
import respx

from mcp_brasil._shared.rate_limiter import RateLimiter
from mcp_brasil.data.comexstat import client
from mcp_brasil.data.comexstat.constants import COUNTRY_URL, QUERY_URL

_QUERY_RESPONSE_EXPORT = {
    "data": {
        "list": [
            {
                "coAnoMes": "202301",
                "noPaisPt": "CHINA",
                "coNcm": "01031000",
                "metricFOB": 1_000_000.0,
                "metricKG": 500_000.0,
            },
            {
                "coAnoMes": "202302",
                "noPaisPt": "CHINA",
                "coNcm": "01031000",
                "metricFOB": 1_200_000.0,
                "metricKG": 600_000.0,
            },
        ]
    }
}

_QUERY_RESPONSE_IMPORT = {
    "data": {
        "list": [
            {
                "coAnoMes": "202301",
                "metricFOB": 800_000.0,
                "metricKG": 400_000.0,
            },
        ]
    }
}

_COUNTRY_RESPONSE = {
    "data": {
        "list": [
            {"id_country": "31", "no_country_pt": "CHINA", "no_country_en": "CHINA"},
            {"id_country": "249", "no_country_pt": "ESTADOS UNIDOS", "no_country_en": "USA"},
        ]
    }
}


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
# _query
# ---------------------------------------------------------------------------


class TestQuery:
    @pytest.mark.asyncio
    @respx.mock
    async def test_sends_correct_body_and_returns_list(self) -> None:
        route = respx.post(QUERY_URL).mock(
            return_value=httpx.Response(200, json=_QUERY_RESPONSE_EXPORT)
        )
        result = await client._query(
            "export", "2023-01", "2023-02", ["ncm"], [], ["metricFOB"], True
        )
        assert route.called
        assert len(result) == 2
        assert result[0]["coAnoMes"] == "202301"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_list_on_no_data(self) -> None:
        respx.post(QUERY_URL).mock(return_value=httpx.Response(200, json={"data": {"list": []}}))
        result = await client._query("export", "2023-01", "2023-01", [], [], ["metricFOB"], False)
        assert result == []


# ---------------------------------------------------------------------------
# _build_periodo
# ---------------------------------------------------------------------------


class TestBuildPeriodo:
    def test_month_detail_formats_yyyy_mm(self) -> None:
        assert client._build_periodo({"coAnoMes": "202301"}, True) == "2023-01"

    def test_month_detail_short_string_passthrough(self) -> None:
        assert client._build_periodo({"coAnoMes": "2023"}, True) == "2023"

    def test_no_month_detail_returns_year(self) -> None:
        assert client._build_periodo({"coAno": "2023"}, False) == "2023"

    def test_missing_key_returns_empty(self) -> None:
        assert client._build_periodo({}, True) == ""


# ---------------------------------------------------------------------------
# consultar_exportacoes
# ---------------------------------------------------------------------------


class TestConsultarExportacoes:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_comex_items(self) -> None:
        respx.post(QUERY_URL).mock(return_value=httpx.Response(200, json=_QUERY_RESPONSE_EXPORT))
        # _QUERY_RESPONSE_EXPORT uses coAnoMes keys → detalhar_por_mes=True
        result = await client.consultar_exportacoes(
            "2023-01", "2023-02", ncm="01031000", detalhar_por_mes=True
        )
        assert len(result) == 2
        assert result[0].periodo == "2023-01"
        assert result[0].pais == "CHINA"
        assert result[0].ncm == "01031000"
        assert result[0].fob_usd == 1_000_000.0
        assert result[0].kg_liquido == 500_000.0

    @pytest.mark.asyncio
    @respx.mock
    async def test_no_filters_returns_items(self) -> None:
        respx.post(QUERY_URL).mock(return_value=httpx.Response(200, json=_QUERY_RESPONSE_EXPORT))
        result = await client.consultar_exportacoes("2023-01", "2023-02")
        assert len(result) == 2

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response_returns_empty_list(self) -> None:
        respx.post(QUERY_URL).mock(return_value=httpx.Response(200, json={"data": {"list": []}}))
        result = await client.consultar_exportacoes("2023-01", "2023-01")
        assert result == []


# ---------------------------------------------------------------------------
# consultar_importacoes
# ---------------------------------------------------------------------------


class TestConsultarImportacoes:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_comex_items(self) -> None:
        respx.post(QUERY_URL).mock(return_value=httpx.Response(200, json=_QUERY_RESPONSE_IMPORT))
        result = await client.consultar_importacoes("2023-01", "2023-01")
        assert len(result) == 1
        assert result[0].fob_usd == 800_000.0
        assert result[0].pais is None

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response_returns_empty_list(self) -> None:
        respx.post(QUERY_URL).mock(return_value=httpx.Response(200, json={"data": {"list": []}}))
        result = await client.consultar_importacoes("2023-01", "2023-01")
        assert result == []


# ---------------------------------------------------------------------------
# balanca_comercial
# ---------------------------------------------------------------------------


class TestBalancaComercial:
    @pytest.mark.asyncio
    @respx.mock
    async def test_surplus_period(self) -> None:
        respx.post(QUERY_URL).mock(
            side_effect=[
                httpx.Response(
                    200,
                    json={"data": {"list": [{"coAnoMes": "202301", "metricFOB": 1_000_000.0}]}},
                ),
                httpx.Response(
                    200,
                    json={"data": {"list": [{"coAnoMes": "202301", "metricFOB": 600_000.0}]}},
                ),
            ]
        )
        result = await client.balanca_comercial("2023-01", "2023-01")
        assert len(result) == 1
        assert result[0].periodo == "2023-01"
        assert result[0].exportacoes_fob == 1_000_000.0
        assert result[0].importacoes_fob == 600_000.0
        assert result[0].saldo_fob == 400_000.0

    @pytest.mark.asyncio
    @respx.mock
    async def test_deficit_period(self) -> None:
        respx.post(QUERY_URL).mock(
            side_effect=[
                httpx.Response(
                    200,
                    json={"data": {"list": [{"coAnoMes": "202301", "metricFOB": 300_000.0}]}},
                ),
                httpx.Response(
                    200,
                    json={"data": {"list": [{"coAnoMes": "202301", "metricFOB": 800_000.0}]}},
                ),
            ]
        )
        result = await client.balanca_comercial("2023-01", "2023-01")
        assert result[0].saldo_fob == pytest.approx(-500_000.0)

    @pytest.mark.asyncio
    @respx.mock
    async def test_merges_multiple_periods_sorted(self) -> None:
        respx.post(QUERY_URL).mock(
            side_effect=[
                httpx.Response(
                    200,
                    json={
                        "data": {
                            "list": [
                                {"coAnoMes": "202302", "metricFOB": 500_000.0},
                                {"coAnoMes": "202301", "metricFOB": 400_000.0},
                            ]
                        }
                    },
                ),
                httpx.Response(
                    200,
                    json={
                        "data": {
                            "list": [
                                {"coAnoMes": "202301", "metricFOB": 200_000.0},
                                {"coAnoMes": "202302", "metricFOB": 300_000.0},
                            ]
                        }
                    },
                ),
            ]
        )
        result = await client.balanca_comercial("2023-01", "2023-02")
        assert len(result) == 2
        assert result[0].periodo == "2023-01"
        assert result[1].periodo == "2023-02"

    @pytest.mark.asyncio
    @respx.mock
    async def test_period_only_in_exports_gets_zero_imports(self) -> None:
        respx.post(QUERY_URL).mock(
            side_effect=[
                httpx.Response(
                    200,
                    json={"data": {"list": [{"coAnoMes": "202301", "metricFOB": 500_000.0}]}},
                ),
                httpx.Response(200, json={"data": {"list": []}}),
            ]
        )
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
        assert result[0].nome == "CHINA"
        assert result[1].id == "249"
        assert result[1].nome == "ESTADOS UNIDOS"

    @pytest.mark.asyncio
    @respx.mock
    async def test_fallback_to_english_name(self) -> None:
        respx.get(COUNTRY_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "data": {
                        "list": [
                            {"id_country": "1", "no_country_pt": None, "no_country_en": "UNKNOWN"}
                        ]
                    }
                },
            )
        )
        result = await client.listar_paises()
        assert result[0].nome == "UNKNOWN"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_list(self) -> None:
        respx.get(COUNTRY_URL).mock(return_value=httpx.Response(200, json={"data": {"list": []}}))
        result = await client.listar_paises()
        assert result == []
