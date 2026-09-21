"""Integration tests for the NucleoGov feature using fastmcp.Client."""

from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil.data.nucleogov.schemas import (
    Despesa,
    ResultadoDespesas,
    ResultadoLicitacoes,
)
from mcp_brasil.data.nucleogov.server import mcp

CLIENT_MODULE = "mcp_brasil.data.nucleogov.client"


class TestToolsRegistered:
    @pytest.mark.asyncio
    async def test_all_5_tools_registered(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            names = {t.name for t in tool_list}
            expected = {
                "buscar_despesas",
                "buscar_contratos",
                "buscar_licitacoes",
                "buscar_dispensas",
                "buscar_legislacao",
            }
            assert expected.issubset(names), f"Missing: {expected - names}"

    @pytest.mark.asyncio
    async def test_tools_have_docstrings(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            for tool in tool_list:
                assert tool.description, f"Tool {tool.name} has no description"


class TestToolExecution:
    @pytest.mark.asyncio
    async def test_buscar_despesas_e2e(self) -> None:
        mock_data = ResultadoDespesas(
            despesas=[Despesa(numero_empenho="42", fornecedor="Empresa Exemplo Ltda")],
            total=1,
        )
        with patch(
            f"{CLIENT_MODULE}.buscar_despesas",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "buscar_despesas", {"portal": "acessoainformacao.cidade.go.gov.br"}
                )
                assert "Empresa Exemplo Ltda" in result.data

    @pytest.mark.asyncio
    async def test_buscar_dispensas_empty_e2e(self) -> None:
        mock_data = ResultadoLicitacoes(licitacoes=[], total=0)
        with patch(
            f"{CLIENT_MODULE}.buscar_licitacoes",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "buscar_dispensas", {"portal": "acessoainformacao.cidade.go.gov.br"}
                )
                assert "Nenhum resultado para estes filtros" in result.data
