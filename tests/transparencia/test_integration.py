"""Integration tests for the transparencia feature using fastmcp.Client.

These tests verify the full pipeline: server → tools → client (mocked HTTP).
"""

from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil.transparencia.schemas import ContratoFornecedor, Sancao, Servidor
from mcp_brasil.transparencia.server import mcp

CLIENT_MODULE = "mcp_brasil.transparencia.client"


@pytest.fixture(autouse=True)
def _set_api_key(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("TRANSPARENCIA_API_KEY", "test-key")


class TestToolsRegistered:
    @pytest.mark.asyncio
    async def test_all_18_tools_registered(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            names = {t.name for t in tool_list}
            expected = {
                "buscar_contratos",
                "consultar_despesas",
                "buscar_servidores",
                "buscar_licitacoes",
                "consultar_bolsa_familia",
                "buscar_sancoes",
                "buscar_emendas",
                "consultar_viagens",
                "buscar_convenios",
                "buscar_cartoes_pagamento",
                "buscar_pep",
                "buscar_acordos_leniencia",
                "buscar_notas_fiscais",
                "consultar_beneficio_social",
                "consultar_cpf",
                "consultar_cnpj",
                "detalhar_contrato",
                "detalhar_servidor",
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
    async def test_all_4_resources_registered(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            expected = {
                "data://endpoints",
                "data://bases-sancoes",
                "data://info-api",
                "data://categorias-beneficios",
            }
            assert expected.issubset(uris), f"Missing: {expected - uris}"


class TestPromptsRegistered:
    @pytest.mark.asyncio
    async def test_all_3_prompts_registered(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            expected = {"auditoria_fornecedor", "analise_despesas", "verificacao_compliance"}
            assert expected.issubset(names), f"Missing: {expected - names}"


class TestToolExecution:
    @pytest.mark.asyncio
    async def test_buscar_contratos_e2e(self) -> None:
        mock_data = [
            ContratoFornecedor(
                numero="CT-E2E",
                objeto="Teste E2E",
                valor_final=99000.0,
                orgao="Órgão Teste",
            )
        ]
        with patch(
            f"{CLIENT_MODULE}.buscar_contratos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_contratos", {"cpf_cnpj": "12345678000190"})
                assert "CT-E2E" in result.data

    @pytest.mark.asyncio
    async def test_buscar_servidores_e2e(self) -> None:
        mock_data = [Servidor(nome="Servidor E2E", tipo_servidor="Civil", orgao="INSS")]
        with patch(
            f"{CLIENT_MODULE}.buscar_servidores",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_servidores", {"nome": "Servidor E2E"})
                assert "Servidor E2E" in result.data

    @pytest.mark.asyncio
    async def test_buscar_sancoes_e2e(self) -> None:
        mock_data = [
            Sancao(
                fonte="CEIS",
                nome="Sancionado E2E",
                cpf_cnpj="12.345.678/0001-90",
                orgao="CGU",
            )
        ]
        with patch(
            f"{CLIENT_MODULE}.buscar_sancoes",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_sancoes", {"consulta": "12345678000190"})
                assert "Sancionado E2E" in result.data

    @pytest.mark.asyncio
    async def test_buscar_servidores_no_params(self) -> None:
        """Tool should return validation message without hitting client."""
        async with Client(mcp) as c:
            result = await c.call_tool("buscar_servidores", {})
            assert "Informe CPF ou nome" in result.data
