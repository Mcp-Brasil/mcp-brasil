"""Integration tests for the Diário Oficial feature using fastmcp.Client.

These tests verify the full pipeline: server -> tools -> client (mocked HTTP).
"""

from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil.diario_oficial.schemas import (
    CidadeQueridoDiario,
    DiarioOficial,
    DiarioResultado,
)
from mcp_brasil.diario_oficial.server import mcp

CLIENT_MODULE = "mcp_brasil.diario_oficial.client"


class TestToolsRegistered:
    @pytest.mark.asyncio
    async def test_all_3_tools_registered(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            names = {t.name for t in tool_list}
            expected = {
                "buscar_diarios",
                "buscar_cidades",
                "listar_territorios",
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
    async def test_capitais_cobertas_resource(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            assert "data://capitais-cobertas" in uris, f"Missing resource. Found: {uris}"


class TestPromptsRegistered:
    @pytest.mark.asyncio
    async def test_investigar_empresa_prompt(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            assert "investigar_empresa" in names, f"Missing prompt. Found: {names}"


class TestToolExecution:
    @pytest.mark.asyncio
    async def test_buscar_diarios_e2e(self) -> None:
        mock_data = DiarioResultado(
            total_gazettes=1,
            gazettes=[
                DiarioOficial(
                    territory_id="3550308",
                    territory_name="São Paulo",
                    state_code="SP",
                    date="2024-01-15",
                    edition_number="1234",
                    is_extra_edition=False,
                    txt_url="https://example.com/gazette.txt",
                    excerpts=["Contrato de licitação"],
                ),
            ],
        )
        with patch(
            f"{CLIENT_MODULE}.buscar_diarios",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_diarios", {"texto": "licitação"})
                assert "São Paulo" in result.data
                assert "2024-01-15" in result.data

    @pytest.mark.asyncio
    async def test_buscar_cidades_e2e(self) -> None:
        mock_data = [
            CidadeQueridoDiario(
                territory_id="3550308",
                territory_name="São Paulo",
                state_code="SP",
            )
        ]
        with patch(
            f"{CLIENT_MODULE}.buscar_cidades",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_cidades", {"nome": "São Paulo"})
                assert "São Paulo" in result.data
                assert "3550308" in result.data

    @pytest.mark.asyncio
    async def test_listar_territorios_e2e(self) -> None:
        mock_data = [
            CidadeQueridoDiario(
                territory_id="2408102",
                territory_name="Natal",
                state_code="RN",
            )
        ]
        with patch(
            f"{CLIENT_MODULE}.listar_cidades",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("listar_territorios", {})
                assert "Natal" in result.data
                assert "RN" in result.data
