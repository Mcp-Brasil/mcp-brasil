"""Tests for the Siscomex HTTP client.

Uses respx to mock HTTP requests and verify response parsing and caching.
"""

from __future__ import annotations

import time

import httpx
import pytest
import respx

from mcp_brasil._shared.rate_limiter import RateLimiter
from mcp_brasil.data.siscomex import client
from mcp_brasil.data.siscomex.constants import NCM_BASE_URL
from mcp_brasil.data.siscomex.schemas import NcmItem

_NCM_SAMPLE: dict = {
    "Data_Ultima_Atualizacao_NCM": "Vigente em 11/05/2026",
    "Ato": "Resolução Gecex nº 812/2025",
    "Nomenclaturas": [
        {
            "Codigo": "01031000",
            "Descricao": "Reprodutores de raça pura",
            "Data_Inicio": "01/01/2022",
            "Data_Fim": "31/12/9999",
            "Tipo_Ato_Ini": "Resolução",
            "Numero_Ato_Ini": "123",
            "Ano_Ato_Ini": "2022",
        },
        {
            "Codigo": "01039000",
            "Descricao": "Outros suínos vivos",
            "Data_Inicio": "01/01/2019",
            "Data_Fim": "31/12/2021",
            "Tipo_Ato_Ini": None,
            "Numero_Ato_Ini": None,
            "Ano_Ato_Ini": None,
        },
    ],
}


def _reset_cache() -> None:
    client._cache = None
    client._cache_expiry = 0.0


# ---------------------------------------------------------------------------
# Rate limiter
# ---------------------------------------------------------------------------


class TestRateLimiter:
    def test_rate_limiter_exists(self) -> None:
        assert hasattr(client, "_rate_limiter")
        assert isinstance(client._rate_limiter, RateLimiter)

    def test_rate_limiter_config(self) -> None:
        assert client._rate_limiter._max_requests == 60
        assert client._rate_limiter._period == 60.0


# ---------------------------------------------------------------------------
# _fetch_all
# ---------------------------------------------------------------------------


class TestFetchAll:
    @pytest.mark.asyncio
    @respx.mock
    async def test_parses_ncm_items(self) -> None:
        _reset_cache()
        respx.get(NCM_BASE_URL).mock(return_value=httpx.Response(200, json=_NCM_SAMPLE))
        result = await client._fetch_all()
        assert len(result) == 2
        assert result[0].codigo == "01031000"
        assert result[0].descricao == "Reprodutores de raça pura"
        assert result[1].dataFim == "31/12/2021"

    @pytest.mark.asyncio
    @respx.mock
    async def test_populates_cache(self) -> None:
        _reset_cache()
        respx.get(NCM_BASE_URL).mock(return_value=httpx.Response(200, json=_NCM_SAMPLE))
        await client._fetch_all()
        assert client._cache is not None
        assert client._cache_expiry > time.monotonic()

    @pytest.mark.asyncio
    @respx.mock
    async def test_cache_hit_skips_http(self) -> None:
        client._cache = [NcmItem(codigo="99999999", descricao="Cached item")]
        client._cache_expiry = time.monotonic() + 9999
        route = respx.get(NCM_BASE_URL).mock(return_value=httpx.Response(200, json=[]))
        result = await client._fetch_all()
        assert not route.called
        assert result[0].codigo == "99999999"
        _reset_cache()


# ---------------------------------------------------------------------------
# buscar_ncm
# ---------------------------------------------------------------------------


class TestBuscarNcm:
    @pytest.mark.asyncio
    @respx.mock
    async def test_filters_by_code_prefix(self) -> None:
        _reset_cache()
        respx.get(NCM_BASE_URL).mock(return_value=httpx.Response(200, json=_NCM_SAMPLE))
        result = await client.buscar_ncm("01031")
        assert len(result) == 1
        assert result[0].codigo == "01031000"

    @pytest.mark.asyncio
    @respx.mock
    async def test_filters_by_description(self) -> None:
        _reset_cache()
        respx.get(NCM_BASE_URL).mock(return_value=httpx.Response(200, json=_NCM_SAMPLE))
        result = await client.buscar_ncm("reprodutores")
        assert len(result) == 1
        assert result[0].codigo == "01031000"

    @pytest.mark.asyncio
    @respx.mock
    async def test_apenas_ativos_excludes_expired(self) -> None:
        _reset_cache()
        respx.get(NCM_BASE_URL).mock(return_value=httpx.Response(200, json=_NCM_SAMPLE))
        result = await client.buscar_ncm("01", apenas_ativos=True)
        assert all(item.dataFim is None for item in result)
        assert len(result) == 1

    @pytest.mark.asyncio
    @respx.mock
    async def test_apenas_ativos_false_includes_all(self) -> None:
        _reset_cache()
        respx.get(NCM_BASE_URL).mock(return_value=httpx.Response(200, json=_NCM_SAMPLE))
        result = await client.buscar_ncm("01", apenas_ativos=False)
        assert len(result) == 2

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_when_no_match(self) -> None:
        _reset_cache()
        respx.get(NCM_BASE_URL).mock(return_value=httpx.Response(200, json=_NCM_SAMPLE))
        result = await client.buscar_ncm("xyzxyz")
        assert result == []


# ---------------------------------------------------------------------------
# consultar_ncm
# ---------------------------------------------------------------------------


class TestConsultarNcm:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_exact_match(self) -> None:
        _reset_cache()
        respx.get(NCM_BASE_URL).mock(return_value=httpx.Response(200, json=_NCM_SAMPLE))
        result = await client.consultar_ncm("01031000")
        assert result is not None
        assert result.codigo == "01031000"
        assert result.descricao == "Reprodutores de raça pura"

    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_none_when_not_found(self) -> None:
        _reset_cache()
        respx.get(NCM_BASE_URL).mock(return_value=httpx.Response(200, json=_NCM_SAMPLE))
        result = await client.consultar_ncm("99999999")
        assert result is None

    @pytest.mark.asyncio
    @respx.mock
    async def test_strips_whitespace(self) -> None:
        _reset_cache()
        respx.get(NCM_BASE_URL).mock(return_value=httpx.Response(200, json=_NCM_SAMPLE))
        result = await client.consultar_ncm("  01031000  ")
        assert result is not None
        assert result.codigo == "01031000"
