"""Integration tests for the TCM-GO feature using fastmcp.Client."""

from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil.data.tcm_go.schemas import Acordao, AcordaoResultado, Processo, ProcessoResultado
from mcp_brasil.data.tcm_go.server import mcp

CLIENT_MODULE = "mcp_brasil.data.tcm_go.client"


class TestToolsRegistered:
    @pytest.mark.asyncio
    async def test_all_3_tools_registered(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            names = {t.name for t in tool_list}
            expected = {"buscar_processos", "listar_valores_campo", "buscar_acordaos"}
            assert expected.issubset(names), f"Missing: {expected - names}"

    @pytest.mark.asyncio
    async def test_tools_have_docstrings(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            for tool in tool_list:
                assert tool.description, f"Tool {tool.name} has no description"


class TestResourcesRegistered:
    @pytest.mark.asyncio
    async def test_endpoints_disponiveis_resource(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            assert "data://endpoints-disponiveis" in uris, f"URIs: {uris}"

    @pytest.mark.asyncio
    async def test_endpoints_disponiveis_content(self) -> None:
        async with Client(mcp) as c:
            content = await c.read_resource("data://endpoints-disponiveis")
            text = content[0].text if isinstance(content, list) else str(content)
            assert "eSiproc" in text
            assert "TCMJuris" in text


class TestPromptsRegistered:
    @pytest.mark.asyncio
    async def test_analisar_municipio_prompt(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            assert "analisar_municipio_goias" in names, f"Prompts: {names}"


class TestToolExecution:
    @pytest.mark.asyncio
    async def test_buscar_processos_e2e(self) -> None:
        mock_data = ProcessoResultado(
            total=1,
            pagina=0,
            processos=[Processo(numero="00000000.2024.000.0.00000.0001", municipio="ANÁPOLIS")],
        )
        with patch(
            f"{CLIENT_MODULE}.buscar_processos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_processos", {"municipio": "ANÁPOLIS"})
                assert "00000000.2024.000.0.00000.0001" in result.data

    @pytest.mark.asyncio
    async def test_buscar_acordaos_e2e(self) -> None:
        mock_data = AcordaoResultado(
            total_servidor=1,
            total_apos_refiltro=1,
            acordaos=[Acordao(numero_acordao="0001/2024", municipio="ANÁPOLIS")],
        )
        with patch(
            f"{CLIENT_MODULE}.buscar_acordaos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_acordaos", {"pesquisa": "denuncias"})
                assert "0001/2024" in result.data
