"""Integration tests for the TCE-ES feature using fastmcp.Client."""

from __future__ import annotations

from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil.data.tce_es.schemas import ContratacaoMunicipio, Licitacao
from mcp_brasil.data.tce_es.server import mcp

CLIENT_MODULE = "mcp_brasil.data.tce_es.client"


class TestToolsRegistered:
    @pytest.mark.asyncio
    async def test_all_4_tools_registered(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            names = {t.name for t in tool_list}
            expected = {
                "buscar_licitacoes_es",
                "buscar_contratos_es",
                "buscar_contratacoes_municipios_es",
                "buscar_obras_es",
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
    async def test_datasets_resource(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            assert "data://datasets" in uris, f"URIs: {uris}"

    @pytest.mark.asyncio
    async def test_datasets_content(self) -> None:
        async with Client(mcp) as c:
            content = await c.read_resource("data://datasets")
            text = content[0].text if isinstance(content, list) else str(content)
            assert "licitacoes" in text.lower() or "Licitações" in text
            assert "contratos" in text.lower() or "Contratos" in text
            assert "resource_id" in text


class TestPromptsRegistered:
    @pytest.mark.asyncio
    async def test_analisar_gestao_prompt(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            assert "analisar_gestao_es" in names, f"Prompts: {names}"


class TestToolExecution:
    @pytest.mark.asyncio
    async def test_buscar_licitacoes_empty_e2e(self) -> None:
        with patch(
            f"{CLIENT_MODULE}.buscar_licitacoes",
            new_callable=AsyncMock,
            return_value=([], 0),
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_licitacoes_es", {})
                assert "Nenhuma licitação" in result.data

    @pytest.mark.asyncio
    async def test_buscar_licitacoes_with_results_e2e(self) -> None:
        mock_data = [
            Licitacao(
                Modalidade="Pregão Eletrônico",
                NumeroEdital="90001",
                AnoEdital="2024",
                Objeto="Aquisição de material",
                Situacao="Homologado",
            )
        ]
        with patch(
            f"{CLIENT_MODULE}.buscar_licitacoes",
            new_callable=AsyncMock,
            return_value=(mock_data, 1),
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_licitacoes_es", {"ano": 2024})
                assert "90001" in result.data
                assert "Homologado" in result.data

    @pytest.mark.asyncio
    async def test_buscar_contratacoes_municipios_e2e(self) -> None:
        mock_data = [
            ContratacaoMunicipio(
                NomeUnidadeGestoraReferencia="PREFEITURA DE VITÓRIA",
                NomeEsferaAdministrativa="Municipal",
                ObjetoContratacao="Reforma de escola",
                AnoReferencia="2024",
            )
        ]
        with patch(
            f"{CLIENT_MODULE}.buscar_contratacoes_municipios",
            new_callable=AsyncMock,
            return_value=(mock_data, 1),
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "buscar_contratacoes_municipios_es",
                    {"q": "Vitória", "ano_referencia": 2024},
                )
                assert "PREFEITURA DE VITÓRIA" in result.data
