"""Integration tests for the dados_gov_br feature using fastmcp.Client.

These tests verify the full pipeline: server -> tools -> client (mocked HTTP).
"""

from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil.data.dados_gov_br.schemas import (
    ConjuntoIndice,
    ConjuntoResultado,
    OrganizacaoDetalhe,
    OrganizacaoIndice,
    OrganizacaoResultado,
    Tema,
)
from mcp_brasil.data.dados_gov_br.server import mcp

CLIENT_MODULE = "mcp_brasil.data.dados_gov_br.client"


class TestToolsRegistered:
    @pytest.mark.asyncio
    async def test_all_11_tools_registered(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            names = {t.name for t in tool_list}
            expected = {
                "buscar_conjuntos",
                "detalhar_conjunto",
                "listar_organizacoes",
                "detalhar_organizacao",
                "listar_temas",
                "buscar_tags",
                "listar_formatos",
                "listar_ods",
                "listar_observancia_legal",
                "listar_reusos",
                "detalhar_reuso",
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
    async def test_3_resources_registered(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            expected = {
                "data://formatos",
                "data://documentacao",
                "data://legislacao",
            }
            assert expected.issubset(uris), f"Missing: {expected - uris}"


class TestPromptsRegistered:
    @pytest.mark.asyncio
    async def test_4_prompts_registered(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            expected = {
                "explorar_dados",
                "listar_dados_orgao",
                "panorama_portal",
                "descobrir_fonte",
            }
            assert expected.issubset(names), f"Missing: {expected - names}"


class TestToolExecution:
    @pytest.mark.asyncio
    async def test_buscar_conjuntos_e2e(self) -> None:
        mock_data = ConjuntoResultado(
            total=1,
            conjuntos=[
                ConjuntoIndice(
                    id="abc-123",
                    title="Dados de Saúde Pública",
                    organizationName="Ministério da Saúde",
                    temas="Saúde",
                ),
            ],
        )
        with patch(
            f"{CLIENT_MODULE}.buscar_conjuntos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "buscar_conjuntos",
                    {"nome": "saúde"},
                )
                assert "Dados de Saúde Pública" in result.data
                assert "Ministério da Saúde" in result.data

    @pytest.mark.asyncio
    async def test_detalhar_conjunto_not_found(self) -> None:
        with patch(
            f"{CLIENT_MODULE}.detalhar_conjunto",
            new_callable=AsyncMock,
            return_value=None,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "detalhar_conjunto",
                    {"conjunto_id": "nao-existe"},
                )
                assert "não encontrado" in result.data

    @pytest.mark.asyncio
    async def test_listar_temas_e2e(self) -> None:
        mock_data = [
            Tema(id="t1", name="saude", title="Saúde", packageCount=200),
        ]
        with patch(
            f"{CLIENT_MODULE}.listar_temas",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("listar_temas", {})
                assert "Saúde" in result.data

    @pytest.mark.asyncio
    async def test_listar_organizacoes_e2e(self) -> None:
        mock_data = OrganizacaoResultado(
            total=1,
            organizacoes=[
                OrganizacaoIndice(
                    id="org-1",
                    titulo="IBGE",
                    qtdConjuntoDeDados="80",
                ),
            ],
        )
        with patch(
            f"{CLIENT_MODULE}.listar_organizacoes",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("listar_organizacoes", {})
                assert "IBGE" in result.data

    @pytest.mark.asyncio
    async def test_detalhar_organizacao_e2e(self) -> None:
        mock_data = OrganizacaoDetalhe(
            id="org-1",
            displayName="IBGE",
            descricao="Estatísticas",
            quantidadeConjuntoDados=80,
        )
        with patch(
            f"{CLIENT_MODULE}.detalhar_organizacao",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "detalhar_organizacao",
                    {"organizacao_id": "org-1"},
                )
                assert "IBGE" in result.data
