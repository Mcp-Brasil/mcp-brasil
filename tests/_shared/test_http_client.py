"""Testes do HTTP client compartilhado."""

from unittest.mock import patch

import httpx
import pytest
import respx

from mcp_brasil._shared.http_client import create_client, http_get, http_post
from mcp_brasil.exceptions import HttpClientError


class TestCreateClient:
    def test_returns_async_client(self) -> None:
        client = create_client()
        assert isinstance(client, httpx.AsyncClient)

    def test_sets_default_headers(self) -> None:
        client = create_client()
        assert "mcp-brasil" in client.headers["user-agent"]
        assert client.headers["accept"] == "application/json"

    def test_custom_base_url(self) -> None:
        client = create_client(base_url="https://api.ibge.gov.br")
        assert str(client.base_url) == "https://api.ibge.gov.br"

    def test_custom_timeout(self) -> None:
        client = create_client(timeout=5.0)
        assert client.timeout.connect == 5.0

    def test_custom_headers_merged(self) -> None:
        client = create_client(headers={"X-Api-Key": "secret"})
        assert client.headers["x-api-key"] == "secret"
        assert "mcp-brasil" in client.headers["user-agent"]

    def test_follow_redirects_enabled(self) -> None:
        client = create_client()
        assert client.follow_redirects is True


class TestHttpGet:
    @pytest.mark.asyncio
    @respx.mock
    async def test_success_returns_json(self) -> None:
        respx.get("https://api.example.com/data").mock(
            return_value=httpx.Response(200, json={"ok": True})
        )
        result = await http_get("https://api.example.com/data")
        assert result == {"ok": True}

    @pytest.mark.asyncio
    @respx.mock
    async def test_with_params(self) -> None:
        respx.get("https://api.example.com/search").mock(
            return_value=httpx.Response(200, json=[1, 2, 3])
        )
        result = await http_get("https://api.example.com/search", params={"q": "test"})
        assert result == [1, 2, 3]

    @pytest.mark.asyncio
    @respx.mock
    async def test_404_raises_immediately(self) -> None:
        """4xx errors (except 429) should not retry."""
        respx.get("https://api.example.com/missing").mock(
            return_value=httpx.Response(404, text="Not Found")
        )
        with pytest.raises(HttpClientError, match="HTTP 404"):
            await http_get("https://api.example.com/missing", max_retries=2)

    @pytest.mark.asyncio
    @respx.mock
    async def test_500_retries_then_succeeds(self) -> None:
        """Server error on first attempt, success on second."""
        route = respx.get("https://api.example.com/flaky")
        route.side_effect = [
            httpx.Response(500, text="Internal Server Error"),
            httpx.Response(200, json={"recovered": True}),
        ]
        with patch("mcp_brasil._shared.http_client.asyncio.sleep"):
            result = await http_get("https://api.example.com/flaky", max_retries=2)
        assert result == {"recovered": True}

    @pytest.mark.asyncio
    @respx.mock
    async def test_429_retries(self) -> None:
        """Rate limited requests should retry."""
        route = respx.get("https://api.example.com/limited")
        route.side_effect = [
            httpx.Response(429, text="Too Many Requests"),
            httpx.Response(200, json={"ok": True}),
        ]
        with patch("mcp_brasil._shared.http_client.asyncio.sleep"):
            result = await http_get("https://api.example.com/limited", max_retries=2)
        assert result == {"ok": True}

    @pytest.mark.asyncio
    @respx.mock
    async def test_all_retries_exhausted(self) -> None:
        """After all retries fail, raises HttpClientError."""
        respx.get("https://api.example.com/down").mock(
            return_value=httpx.Response(503, text="Service Unavailable")
        )
        with (
            patch("mcp_brasil._shared.http_client.asyncio.sleep"),
            pytest.raises(HttpClientError, match="failed after"),
        ):
            await http_get("https://api.example.com/down", max_retries=1)

    @pytest.mark.asyncio
    @respx.mock
    async def test_timeout_retries(self) -> None:
        """Timeout errors should retry."""
        route = respx.get("https://api.example.com/slow")
        route.side_effect = [
            httpx.ReadTimeout("timeout"),
            httpx.Response(200, json={"ok": True}),
        ]
        with patch("mcp_brasil._shared.http_client.asyncio.sleep"):
            result = await http_get("https://api.example.com/slow", max_retries=2)
        assert result == {"ok": True}

    @pytest.mark.asyncio
    @respx.mock
    async def test_zero_retries_no_retry(self) -> None:
        """With max_retries=0, no retries happen."""
        respx.get("https://api.example.com/once").mock(
            return_value=httpx.Response(500, text="Error")
        )
        with pytest.raises(HttpClientError, match="HTTP 500"):
            await http_get("https://api.example.com/once", max_retries=0)

    @pytest.mark.asyncio
    @respx.mock
    async def test_non_json_body_raises_http_client_error(self) -> None:
        """HTTP 200 with an HTML error page must not leak JSONDecodeError.

        Regression test for issue #37: SAR/ANA returned an HTML error page
        and tools crashed with "Expecting value: line 1 column 1 (char 0)".
        """
        respx.get("https://api.example.com/page").mock(
            return_value=httpx.Response(
                200,
                text="<html><body>Erro</body></html>",
                headers={"content-type": "text/html"},
            )
        )
        with pytest.raises(HttpClientError, match="non-JSON body"):
            await http_get("https://api.example.com/page")


class TestHttpPost:
    @pytest.mark.asyncio
    @respx.mock
    async def test_non_json_body_raises_http_client_error(self) -> None:
        """Same non-JSON wrapping guarantee for POST responses."""
        respx.post("https://api.example.com/submit").mock(
            return_value=httpx.Response(
                200,
                text="<html><body>Erro</body></html>",
                headers={"content-type": "text/html"},
            )
        )
        with pytest.raises(HttpClientError, match="non-JSON body"):
            await http_post("https://api.example.com/submit", json_body={"k": "v"})
