"""Integration tests for the TCE-PA feature using fastmcp.Client."""

import json
from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil.data.tce_pa.schemas import DiarioOficial, ResultadoPesquisa, SessaoPlenaria
from mcp_brasil.data.tce_pa.server import mcp

CLIENT_MODULE = "mcp_brasil.data.tce_pa.client"


class TestToolsRegistered:
    @pytest.mark.asyncio
    async def test_all_4_tools_registered(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            names = {t.name for t in tool_list}
            expected = {
                "buscar_diario_oficial_pa",
                "buscar_sessoes_plenarias_pa",
                "buscar_jurisprudencia_pa",
                "buscar_conteudo_pa",
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
    async def test_endpoints_resource_registered(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            assert "data://endpoints" in uris, f"URIs found: {uris}"

    @pytest.mark.asyncio
    async def test_endpoints_resource_content(self) -> None:
        async with Client(mcp) as c:
            content = await c.read_resource("data://endpoints")
            text = content[0].text if isinstance(content, list) else str(content)
            data = json.loads(text)
            endpoints = [e["endpoint"] for e in data]
            assert "/v1/diario_oficial" in endpoints
            assert "/pesquisa/resultados" in endpoints

    @pytest.mark.asyncio
    async def test_endpoints_resource_has_all_bases(self) -> None:
        async with Client(mcp) as c:
            content = await c.read_resource("data://endpoints")
            text = content[0].text if isinstance(content, list) else str(content)
            assert "acordaos" in text
            assert "resolucoes" in text
            assert "prejulgados" in text
            assert "noticias" in text
            assert "sessoes-plenarias" in text


class TestPromptsRegistered:
    @pytest.mark.asyncio
    async def test_all_4_prompts_registered(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            expected = {
                "analisar_jurisprudencia_pa",
                "analisar_diario_oficial_pa",
                "analisar_sessoes_plenarias_pa",
                "explorar_conteudo_pa",
            }
            assert expected.issubset(names), f"Missing: {expected - names}"


class TestToolExecution:
    @pytest.mark.asyncio
    async def test_buscar_diario_oficial_pa_e2e(self) -> None:
        mock_data = [
            DiarioOficial(
                numero_publicacao=1,
                data_publicacao="2024-01-01",
                tipo_ato="Contratos",
                publicacao="Contrato teste",
            )
        ]
        with patch(
            f"{CLIENT_MODULE}.buscar_diario_oficial",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_diario_oficial_pa", {"ano": 2024})
                assert "Contratos" in result.data

    @pytest.mark.asyncio
    async def test_buscar_diario_oficial_pa_empty_e2e(self) -> None:
        with patch(
            f"{CLIENT_MODULE}.buscar_diario_oficial",
            new_callable=AsyncMock,
            return_value=[],
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_diario_oficial_pa", {"ano": 2024})
                assert "Nenhuma publicação encontrada" in result.data

    @pytest.mark.asyncio
    async def test_buscar_sessoes_plenarias_pa_e2e(self) -> None:
        mock_data = [
            SessaoPlenaria(
                titulo="Sessão Ordinária nº 01/2024",
                data_sessao="15/01/2024",
                tipo_sessao="Ordinária",
                ano=2024,
                url_pagina="https://www.tcepa.tc.br/sessoes/codigo/1/titulo",
            )
        ]
        with patch(
            f"{CLIENT_MODULE}.buscar_sessoes_plenarias",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_sessoes_plenarias_pa", {"tipo": "sessoes"})
                assert "Ordinária" in result.data

    @pytest.mark.asyncio
    async def test_buscar_jurisprudencia_pa_e2e(self) -> None:
        mock_data = [
            ResultadoPesquisa(
                titulo="Acórdão nº 1/2024",
                url_pagina="https://www.tcepa.tc.br/acordaos/codigo/1/titulo",
                url_documento="https://www.tcepa.tc.br/acordaos/codigo/1/titulo/download",
                campos={"Data da sessão plenária": "01/01/2024"},
            )
        ]
        with patch(
            f"{CLIENT_MODULE}.buscar_pesquisa_integrada",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_jurisprudencia_pa", {"base": "acordaos"})
                assert "Acórdão" in result.data

    @pytest.mark.asyncio
    async def test_buscar_conteudo_pa_e2e(self) -> None:
        mock_data = [
            ResultadoPesquisa(
                titulo="TCE-PA em ação",
                url_pagina="https://www.tcepa.tc.br/noticias/codigo/1/titulo",
                url_documento="https://www.tcepa.tc.br/noticias/codigo/1/titulo/conteudo-original",
                campos={"Data de publicação": "01/04/2024"},
            )
        ]
        with patch(
            f"{CLIENT_MODULE}.buscar_pesquisa_integrada",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_conteudo_pa", {"base": "noticias"})
                assert "TCE-PA em ação" in result.data
