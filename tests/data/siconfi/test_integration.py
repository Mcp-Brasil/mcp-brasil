"""Integration tests — feature registration and FastMCP end-to-end."""

from __future__ import annotations

import pytest
from fastmcp import Client

from mcp_brasil.data.siconfi import FEATURE_META
from mcp_brasil.data.siconfi.server import mcp


def test_feature_meta_valid() -> None:
    assert FEATURE_META.name == "siconfi"
    assert FEATURE_META.requires_auth is False
    assert "municipios" in FEATURE_META.tags


@pytest.mark.asyncio
async def test_server_lists_tools() -> None:
    async with Client(mcp) as c:
        tools = await c.list_tools()
    names = {t.name for t in tools}
    assert "listar_entes" in names
    assert "consultar_rreo" in names
    assert "consultar_rgf" in names
    assert "consultar_dca" in names


@pytest.mark.asyncio
async def test_server_lists_resources() -> None:
    async with Client(mcp) as c:
        resources = await c.list_resources()
    uris = {str(r.uri) for r in resources}
    assert any("catalogo-anexos" in u for u in uris)


@pytest.mark.asyncio
async def test_server_lists_prompts() -> None:
    async with Client(mcp) as c:
        prompts = await c.list_prompts()
    names = {p.name for p in prompts}
    assert "analise_fiscal_municipio" in names
