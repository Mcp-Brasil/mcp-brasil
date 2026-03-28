"""Integration tests for the TCU feature using fastmcp.Client."""

from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil.data.tcu.schemas import Acordao
from mcp_brasil.data.tcu.server import mcp

CLIENT_MODULE = "mcp_brasil.data.tcu.client"


class TestToolsRegistered:
    @pytest.mark.asyncio
    async def test_all_8_tools_registered(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            names = {t.name for t in tool_list}
            expected = {
                "consultar_acordaos",
                "consultar_inabilitados",
                "consultar_inidoneos",
                "consultar_certidoes",
                "calcular_debito",
                "consultar_pedidos_congresso",
                "consultar_termos_contratuais",
                "consultar_cadirreg",
            }
            assert expected.issubset(names), f"Missing: {expected - names}"

    @pytest.mark.asyncio
    async def test_tools_have_docstrings(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            for tool in tool_list:
                assert tool.description, f"Tool {tool.name} has no description"


class TestResourcesRegistered:
    @pytest.mark.asyncio
    async def test_resources_registered(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            assert "data://catalogo-endpoints" in uris
            assert "data://colegiados" in uris
            assert "data://situacoes-acordao" in uris


class TestPromptsRegistered:
    @pytest.mark.asyncio
    async def test_prompts_registered(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            assert "analise_acordaos" in names
            assert "verificar_empresa" in names


class TestToolExecution:
    @pytest.mark.asyncio
    async def test_consultar_acordaos_e2e(self) -> None:
        mock_data = [
            Acordao(
                key="ACORDAO-COMPLETO-123",
                anoAcordao="2026",
                titulo="ACORDAO 100/2026 - PLENARIO",
                numeroAcordao="100",
                colegiado="Plenário",
                relator="BRUNO DANTAS",
                dataSessao="18/03/2026",
                sumario="Embargos de declaração",
            ),
        ]
        with patch(
            f"{CLIENT_MODULE}.consultar_acordaos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("consultar_acordaos", {})
                text = result[0].text if isinstance(result, list) else str(result)
                assert "100" in text
                assert "BRUNO DANTAS" in text

    @pytest.mark.asyncio
    async def test_consultar_inabilitados_empty(self) -> None:
        with patch(
            f"{CLIENT_MODULE}.consultar_inabilitados",
            new_callable=AsyncMock,
            return_value=[],
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("consultar_inabilitados", {"cpf": "99999999999"})
                text = result[0].text if isinstance(result, list) else str(result)
                assert "Nenhum inabilitado" in text
