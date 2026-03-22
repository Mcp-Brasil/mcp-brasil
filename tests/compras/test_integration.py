"""Integration tests for the Compras feature using fastmcp.Client.

These tests verify the full pipeline: server -> tools -> client (mocked HTTP).
"""

from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil.compras.schemas import Contratacao, ContratacaoResultado
from mcp_brasil.compras.server import mcp

CLIENT_MODULE = "mcp_brasil.compras.client"


class TestToolsRegistered:
    @pytest.mark.asyncio
    async def test_all_3_tools_registered(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            names = {t.name for t in tool_list}
            expected = {
                "buscar_contratacoes",
                "buscar_contratos",
                "buscar_atas",
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
    async def test_1_resource_registered(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            expected = {"data://modalidades"}
            assert expected.issubset(uris), f"Missing: {expected - uris}"


class TestPromptsRegistered:
    @pytest.mark.asyncio
    async def test_1_prompt_registered(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            expected = {"investigar_fornecedor"}
            assert expected.issubset(names), f"Missing: {expected - names}"


class TestToolExecution:
    @pytest.mark.asyncio
    async def test_buscar_contratacoes_e2e(self) -> None:
        mock_data = ContratacaoResultado(
            total=1,
            contratacoes=[
                Contratacao(
                    orgao_nome="Ministério da Educação",
                    objeto="Aquisição de computadores",
                    modalidade_id=1,
                    situacao_nome="Publicada",
                    valor_estimado=500000.0,
                ),
            ],
        )
        with patch(
            f"{CLIENT_MODULE}.buscar_contratacoes",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "buscar_contratacoes",
                    {"texto": "computadores"},
                )
                assert "Aquisição de computadores" in result.data
                assert "Ministério da Educação" in result.data

    @pytest.mark.asyncio
    async def test_buscar_contratos_no_filter(self) -> None:
        async with Client(mcp) as c:
            result = await c.call_tool("buscar_contratos", {})
            assert "Informe pelo menos um filtro" in result.data

    @pytest.mark.asyncio
    async def test_buscar_atas_no_filter(self) -> None:
        async with Client(mcp) as c:
            result = await c.call_tool("buscar_atas", {})
            assert "Informe pelo menos um filtro" in result.data
