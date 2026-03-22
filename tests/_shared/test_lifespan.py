"""Tests for the shared lifespan — HTTP client lifecycle."""

import contextlib

import pytest

from mcp_brasil._shared.lifespan import http_lifespan


class TestHttpLifespan:
    """Test the http_lifespan creates and closes an httpx.AsyncClient."""

    @pytest.mark.asyncio
    async def test_lifespan_creates_and_closes_client(self) -> None:
        """Lifespan should yield an http_client and close it on exit."""
        from fastmcp import FastMCP

        server = FastMCP("test")

        # Simulate the lifespan generator
        gen = http_lifespan._fn(server)
        context = await gen.__anext__()

        assert context is not None
        assert "http_client" in context

        client = context["http_client"]
        assert not client.is_closed

        # Cleanup
        with contextlib.suppress(StopAsyncIteration):
            await gen.__anext__()

        assert client.is_closed

    @pytest.mark.asyncio
    async def test_lifespan_client_has_correct_headers(self) -> None:
        """The HTTP client should have User-Agent and Accept headers."""
        from fastmcp import FastMCP

        server = FastMCP("test")

        gen = http_lifespan._fn(server)
        context = await gen.__anext__()

        assert context is not None
        client = context["http_client"]
        assert "User-Agent" in client.headers
        assert client.headers["Accept"] == "application/json"

        # Cleanup
        with contextlib.suppress(StopAsyncIteration):
            await gen.__anext__()
