"""Integration tests for the Saúde feature using fastmcp.Client.

These tests verify the full pipeline: server -> tools -> client (mocked HTTP).
"""

from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil.data.saude.schemas import (
    Estabelecimento,
    EstabelecimentoDetalhe,
    Leito,
    TipoEstabelecimento,
)
from mcp_brasil.data.saude.server import mcp

CLIENT_MODULE = "mcp_brasil.data.saude.client"


class TestToolsRegistered:
    @pytest.mark.asyncio
    async def test_all_10_tools_registered(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            names = {t.name for t in tool_list}
            expected = {
                "buscar_estabelecimentos",
                "buscar_profissionais",
                "listar_tipos_estabelecimento",
                "consultar_leitos",
                "buscar_urgencias",
                "buscar_por_tipo",
                "buscar_estabelecimento_por_cnes",
                "buscar_por_coordenadas",
                "resumo_rede_municipal",
                "comparar_municipios",
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
    async def test_resource_registered(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            assert "data://codigos-uf" in uris


class TestPromptsRegistered:
    @pytest.mark.asyncio
    async def test_prompt_registered(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            assert "analise_rede_saude" in names


class TestToolExecution:
    @pytest.mark.asyncio
    async def test_buscar_estabelecimentos_e2e(self) -> None:
        mock_data = [
            Estabelecimento(
                codigo_cnes="1234567",
                nome_fantasia="UBS Central",
                descricao_tipo="Central de Regulação",
                tipo_gestao="Municipal",
                endereco="Rua ABC, 123",
            )
        ]
        with patch(
            f"{CLIENT_MODULE}.buscar_estabelecimentos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "buscar_estabelecimentos",
                    {"codigo_municipio": "355030"},
                )
                assert "UBS Central" in result.data
                assert "1234567" in result.data

    @pytest.mark.asyncio
    async def test_buscar_profissionais_e2e(self) -> None:
        """buscar_profissionais now returns a deprecation message (no mock needed)."""
        async with Client(mcp) as c:
            result = await c.call_tool("buscar_profissionais", {"cnes": "1234567"})
            assert "descontinuado" in result.data

    @pytest.mark.asyncio
    async def test_listar_tipos_e2e(self) -> None:
        mock_data = [
            TipoEstabelecimento(codigo="01", descricao="Central de Regulação"),
        ]
        with patch(
            f"{CLIENT_MODULE}.listar_tipos_estabelecimento",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("listar_tipos_estabelecimento", {})
                assert "Central de Regulação" in result.data

    @pytest.mark.asyncio
    async def test_consultar_leitos_e2e(self) -> None:
        mock_data = [
            Leito(
                codigo_cnes="1234567",
                tipo_leito="Cirúrgico",
                especialidade="Cirurgia Geral",
                existente=20,
                sus=15,
            )
        ]
        with patch(
            f"{CLIENT_MODULE}.consultar_leitos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("consultar_leitos", {})
                assert "Cirúrgico" in result.data
                assert "Cirurgia Geral" in result.data

    @pytest.mark.asyncio
    async def test_buscar_estabelecimentos_empty(self) -> None:
        with patch(
            f"{CLIENT_MODULE}.buscar_estabelecimentos",
            new_callable=AsyncMock,
            return_value=[],
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_estabelecimentos", {})
                assert "Nenhum estabelecimento" in result.data

    @pytest.mark.asyncio
    async def test_buscar_urgencias_e2e(self) -> None:
        mock_data = [
            Estabelecimento(
                codigo_cnes="9876543",
                nome_fantasia="UPA 24h Norte",
                descricao_tipo="Pronto Atendimento",
                endereco="Av Norte, 100",
            )
        ]
        with patch(
            f"{CLIENT_MODULE}.buscar_estabelecimentos_por_tipo",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_urgencias", {"codigo_municipio": "220040"})
                assert "UPA 24h Norte" in result.data

    @pytest.mark.asyncio
    async def test_buscar_por_tipo_e2e(self) -> None:
        mock_data = [
            Estabelecimento(
                codigo_cnes="1111111",
                nome_fantasia="Hospital Regional",
                descricao_tipo="Hospital Geral",
            )
        ]
        with patch(
            f"{CLIENT_MODULE}.buscar_estabelecimentos_por_tipo",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_por_tipo", {"codigo_tipo": "05"})
                assert "Hospital Regional" in result.data

    @pytest.mark.asyncio
    async def test_buscar_estabelecimento_por_cnes_e2e(self) -> None:
        mock_data = EstabelecimentoDetalhe(
            codigo_cnes="1234567",
            nome_fantasia="Hospital São Paulo",
            descricao_tipo="Hospital Geral",
            telefone="(11) 5576-4000",
        )
        with patch(
            f"{CLIENT_MODULE}.buscar_estabelecimento_por_cnes",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("buscar_estabelecimento_por_cnes", {"cnes": "1234567"})
                assert "Hospital São Paulo" in result.data
                assert "(11) 5576-4000" in result.data

    @pytest.mark.asyncio
    async def test_resumo_rede_municipal_e2e(self) -> None:
        with (
            patch(
                f"{CLIENT_MODULE}.buscar_estabelecimentos",
                new_callable=AsyncMock,
                return_value=[Estabelecimento(descricao_tipo="Hospital Geral")],
            ),
            patch(
                f"{CLIENT_MODULE}.consultar_leitos",
                new_callable=AsyncMock,
                return_value=[Leito(existente=10, sus=8)],
            ),
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("resumo_rede_municipal", {"codigo_municipio": "355030"})
                assert "Resumo da rede de saúde" in result.data
                assert "Hospital Geral" in result.data

    @pytest.mark.asyncio
    async def test_comparar_municipios_e2e(self) -> None:
        with (
            patch(
                f"{CLIENT_MODULE}.buscar_estabelecimentos",
                new_callable=AsyncMock,
                return_value=[Estabelecimento()],
            ),
            patch(
                f"{CLIENT_MODULE}.consultar_leitos",
                new_callable=AsyncMock,
                return_value=[Leito(existente=5, sus=3)],
            ),
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "comparar_municipios",
                    {"codigos_municipios": ["355030", "330455"]},
                )
                assert "Comparação" in result.data
                assert "355030" in result.data
