"""Integration tests for the DataJud MPU feature using fastmcp.Client."""

from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil.data.datajud.schemas import Movimentacao, MPUEstatisticas, MPUProcesso
from mcp_brasil.data.datajud.server import mcp

CLIENT_MODULE = "mcp_brasil.data.datajud.client"


class TestMpuToolsRegistered:
    @pytest.mark.asyncio
    async def test_all_5_mpu_tools_registered(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            names = {t.name for t in tool_list}
            expected = {
                "buscar_medidas_protetivas",
                "buscar_mpu_concedidas",
                "buscar_mpu_por_tipo",
                "estatisticas_mpu",
                "timeline_mpu",
            }
            assert expected.issubset(names), f"Missing: {expected - names}"

    @pytest.mark.asyncio
    async def test_mpu_tools_have_docstrings(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            mpu_tools = [t for t in tool_list if "mpu" in t.name or "medidas_protetivas" in t.name]
            assert len(mpu_tools) == 5
            for tool in mpu_tools:
                assert tool.description, f"Tool {tool.name} has no description"


class TestMpuResourcesRegistered:
    @pytest.mark.asyncio
    async def test_all_3_mpu_resources_registered(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            expected = {
                "data://mpu/classes",
                "data://mpu/movimentos",
                "data://mpu/complementos",
            }
            assert expected.issubset(uris), f"Missing: {expected - uris}"


class TestMpuPromptsRegistered:
    @pytest.mark.asyncio
    async def test_all_2_mpu_prompts_registered(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            expected = {"analisar_mpus", "monitorar_mpu"}
            assert expected.issubset(names), f"Missing: {expected - names}"


class TestMpuToolExecution:
    @pytest.mark.asyncio
    async def test_buscar_medidas_protetivas_e2e(self) -> None:
        mock_data = [
            MPUProcesso(
                numero="0001234",
                classe_nome="MPU Criminal",
                tribunal="TJSP",
            )
        ]
        with patch(
            f"{CLIENT_MODULE}.buscar_medidas_protetivas",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "buscar_medidas_protetivas",
                    {"tribunal": "tjsp", "lei": "maria_penha"},
                )
                assert "0001234" in result.data

    @pytest.mark.asyncio
    async def test_estatisticas_mpu_e2e(self) -> None:
        mock_stats = MPUEstatisticas(total=42, tribunal="tjpi", periodo="2024")
        with patch(
            f"{CLIENT_MODULE}.estatisticas_mpu",
            new_callable=AsyncMock,
            return_value=mock_stats,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "estatisticas_mpu",
                    {"tribunal": "tjpi", "ano": 2024},
                )
                assert "42" in result.data
                assert "TJPI" in result.data

    @pytest.mark.asyncio
    async def test_timeline_mpu_e2e(self) -> None:
        mock_data = [
            Movimentacao(data="2024-06-01", nome="Concedida medida protetiva", codigo=15486),
        ]
        with patch(
            f"{CLIENT_MODULE}.timeline_mpu",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "timeline_mpu",
                    {"numero_processo": "0001234", "tribunal": "tjsp"},
                )
                assert "Concedida" in result.data


class TestTotalRegistrations:
    """Verify total counts after MPU expansion."""

    @pytest.mark.asyncio
    async def test_total_12_tools(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            assert len(tool_list) == 12, f"Expected 12 tools, got {len(tool_list)}"

    @pytest.mark.asyncio
    async def test_total_6_resources(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            assert len(resources) == 6, f"Expected 6 resources, got {len(resources)}"

    @pytest.mark.asyncio
    async def test_total_4_prompts(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            assert len(prompts) == 4, f"Expected 4 prompts, got {len(prompts)}"
