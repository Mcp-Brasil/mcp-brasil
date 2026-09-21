"""Integration tests for the Megasoft feature using fastmcp.Client."""

from __future__ import annotations

from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil.data.megasoft.schemas import (
    ResultadoEmpenhos,
    ResultadoOrdensPagamento,
    ResultadoReceitas,
    TotaisOrdensPagamento,
)
from mcp_brasil.data.megasoft.server import mcp

CLIENT_MODULE = "mcp_brasil.data.megasoft.client"


class TestToolsRegistered:
    @pytest.mark.asyncio
    async def test_all_4_tools_registered(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            names = {t.name for t in tool_list}
            expected = {
                "buscar_empenhos",
                "buscar_ordens_pagamento",
                "totais_ordens_pagamento",
                "buscar_receitas",
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
            assert "Empenhos" in text
            assert "Ordens de Pagamento" in text
            assert "Receitas" in text


class TestPromptsRegistered:
    @pytest.mark.asyncio
    async def test_analisar_municipio_prompt(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            assert "analisar_municipio_megasoft" in names, f"Prompts: {names}"


class TestToolExecution:
    @pytest.mark.asyncio
    async def test_buscar_empenhos_e2e(self) -> None:
        mock_data = ResultadoEmpenhos(total=0, empenhos=[])
        with patch(
            f"{CLIENT_MODULE}.buscar_empenhos", new_callable=AsyncMock, return_value=mock_data
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "buscar_empenhos",
                    {
                        "municipio": "exemplo",
                        "data_inicial": "01/01/2024",
                        "data_final": "31/12/2024",
                    },
                )
                assert "Nenhum empenho" in result.data

    @pytest.mark.asyncio
    async def test_totais_e2e(self) -> None:
        mock_data = TotaisOrdensPagamento(valor_pago=100000.0, valor_total=98000.0)
        with patch(
            f"{CLIENT_MODULE}.totais_ordens_pagamento",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "totais_ordens_pagamento",
                    {
                        "municipio": "exemplo",
                        "data_inicial": "01/01/2024",
                        "data_final": "31/12/2024",
                    },
                )
                assert "R$ 100.000,00" in result.data

    @pytest.mark.asyncio
    async def test_buscar_receitas_e2e(self) -> None:
        mock_data = ResultadoReceitas(total=0, receitas=[])
        with patch(
            f"{CLIENT_MODULE}.buscar_receitas", new_callable=AsyncMock, return_value=mock_data
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "buscar_receitas",
                    {
                        "municipio": "exemplo",
                        "data_inicial": "01/01/2024",
                        "data_final": "31/12/2024",
                    },
                )
                assert "Nenhuma receita" in result.data

    @pytest.mark.asyncio
    async def test_ordens_e2e(self) -> None:
        mock_data = ResultadoOrdensPagamento(total=0, ordens=[])
        with patch(
            f"{CLIENT_MODULE}.buscar_ordens_pagamento",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "buscar_ordens_pagamento",
                    {
                        "municipio": "exemplo",
                        "data_inicial": "01/01/2024",
                        "data_final": "31/12/2024",
                    },
                )
                assert "Nenhuma ordem de pagamento" in result.data
