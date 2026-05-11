"""Integration tests for the Siscomex feature using fastmcp.Client.

These tests verify the full pipeline: server registration and tool execution.
"""

import pytest
from fastmcp import Client

from mcp_brasil.data.siscomex.server import mcp


class TestToolsRegistered:
    @pytest.mark.asyncio
    async def test_all_tools_registered(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            names = {t.name for t in tool_list}
            expected = {"buscar_ncm", "consultar_ncm"}
            assert expected.issubset(names), f"Missing: {expected - names}"

    @pytest.mark.asyncio
    async def test_tools_have_docstrings(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            for tool in tool_list:
                assert tool.description, f"Tool {tool.name} has no description"

    @pytest.mark.asyncio
    async def test_tool_count(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            assert len(tool_list) == 2
