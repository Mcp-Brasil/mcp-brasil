"""Integration tests for the BrasilAPI feature using fastmcp.Client.

These tests verify the full pipeline: server registration and tool execution.
"""

from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil.brasilapi.schemas import (
    Banco,
    DddInfo,
    Endereco,
    Feriado,
    NcmItem,
    RegistroBrDominio,
    TaxaOficial,
)
from mcp_brasil.brasilapi.server import mcp

CLIENT_MODULE = "mcp_brasil.brasilapi.client"


class TestToolsRegistered:
    @pytest.mark.asyncio
    async def test_all_16_tools_registered(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            names = {t.name for t in tool_list}
            expected = {
                "consultar_cep",
                "consultar_cnpj",
                "consultar_ddd",
                "listar_bancos",
                "consultar_banco",
                "listar_moedas",
                "consultar_cotacao",
                "consultar_feriados",
                "consultar_taxa",
                "listar_tabelas_fipe",
                "listar_marcas_fipe",
                "buscar_veiculos_fipe",
                "consultar_isbn",
                "buscar_ncm",
                "consultar_pix_participantes",
                "consultar_registro_br",
            }
            assert expected.issubset(names), f"Missing: {expected - names}"
            assert len(expected) == 16

    @pytest.mark.asyncio
    async def test_tools_have_docstrings(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            for tool in tool_list:
                assert tool.description, f"Tool {tool.name} has no description"


class TestResourcesRegistered:
    @pytest.mark.asyncio
    async def test_all_3_resources_registered(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            expected = {
                "data://taxas",
                "data://tipos-veiculo-fipe",
                "data://endpoints",
            }
            assert expected.issubset(uris), f"Missing: {expected - uris}"


class TestPromptsRegistered:
    @pytest.mark.asyncio
    async def test_all_2_prompts_registered(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            expected = {
                "analise_empresa",
                "panorama_economico",
            }
            assert expected.issubset(names), f"Missing: {expected - names}"


class TestToolExecution:
    @pytest.mark.asyncio
    async def test_consultar_cep_e2e(self) -> None:
        mock_data = Endereco(
            cep="01001-000",
            state="SP",
            city="São Paulo",
            neighborhood="Sé",
            street="Praça da Sé",
        )
        with patch(
            f"{CLIENT_MODULE}.consultar_cep",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("consultar_cep", {"cep": "01001000"})
                assert "São Paulo" in result.data
                assert "Praça da Sé" in result.data

    @pytest.mark.asyncio
    async def test_consultar_ddd_e2e(self) -> None:
        mock_data = DddInfo(state="SP", cities=["São Paulo", "Guarulhos"])
        with patch(
            f"{CLIENT_MODULE}.consultar_ddd",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("consultar_ddd", {"ddd": "11"})
                assert "SP" in result.data
                assert "São Paulo" in result.data

    @pytest.mark.asyncio
    async def test_listar_bancos_e2e(self) -> None:
        mock_data = [
            Banco(ispb="00000000", name="BCO DO BRASIL S.A.", code=1),
        ]
        with patch(
            f"{CLIENT_MODULE}.listar_bancos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("listar_bancos", {})
                assert "BCO DO BRASIL" in result.data

    @pytest.mark.asyncio
    async def test_consultar_feriados_e2e(self) -> None:
        mock_data = [
            Feriado(date="2024-01-01", name="Confraternização Universal", type="national"),
        ]
        with patch(
            f"{CLIENT_MODULE}.consultar_feriados",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("consultar_feriados", {"ano": 2024})
                assert "Confraternização Universal" in result.data

    @pytest.mark.asyncio
    async def test_consultar_taxa_e2e(self) -> None:
        mock_data = TaxaOficial(nome="Selic", valor=13.75)
        with patch(
            f"{CLIENT_MODULE}.consultar_taxa",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("consultar_taxa", {"sigla": "SELIC"})
                assert "Selic" in result.data

    @pytest.mark.asyncio
    async def test_listar_marcas_fipe_invalid_type_e2e(self) -> None:
        async with Client(mcp) as c:
            result = await c.call_tool("listar_marcas_fipe", {"tipo_veiculo": "aviao"})
            assert "Tipo inválido" in result.data

    @pytest.mark.asyncio
    async def test_buscar_ncm_empty_e2e(self) -> None:
        with patch(
            f"{CLIENT_MODULE}.buscar_ncm",
            new_callable=AsyncMock,
            return_value=[],
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_ncm", {"busca": "xyzabc"})
                assert "Nenhum NCM encontrado" in result.data

    @pytest.mark.asyncio
    async def test_buscar_ncm_with_results_e2e(self) -> None:
        mock_data = [
            NcmItem(codigo="01012100", descricao="Cavalos reprodutores de raça pura"),
        ]
        with patch(
            f"{CLIENT_MODULE}.buscar_ncm",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_ncm", {"busca": "cavalo"})
                assert "01012100" in result.data

    @pytest.mark.asyncio
    async def test_consultar_registro_br_e2e(self) -> None:
        mock_data = RegistroBrDominio(
            status_code=2,
            status="REGISTERED",
            fqdn="google.com.br",
            hosts=["ns1.google.com"],
        )
        with patch(
            f"{CLIENT_MODULE}.consultar_registro_br",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("consultar_registro_br", {"dominio": "google.com.br"})
                assert "google.com.br" in result.data
                assert "Registrado" in result.data
