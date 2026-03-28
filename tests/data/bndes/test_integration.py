"""Integration tests for the BNDES feature via fastmcp.Client."""

import pytest
from fastmcp import Client

from mcp_brasil.data.bndes.server import mcp


@pytest.mark.asyncio
async def test_server_lists_tools() -> None:
    """All BNDES tools should be registered."""
    async with Client(mcp) as c:
        tools = await c.list_tools()
    names = {t.name for t in tools}
    expected = {
        "buscar_datasets_bndes",
        "detalhar_dataset_bndes",
        "consultar_operacoes_bndes",
        "listar_instituicoes_bndes",
    }
    assert expected.issubset(names), f"Missing tools: {expected - names}"
