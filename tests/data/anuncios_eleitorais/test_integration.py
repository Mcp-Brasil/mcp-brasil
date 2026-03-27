"""Integration tests for anuncios_eleitorais — fastmcp.Client e2e."""

from __future__ import annotations

import os
from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil.data.anuncios_eleitorais.schemas import (
    AnuncioEleitoral,
    FaixaValor,
    RespostaAnuncios,
)
from mcp_brasil.data.anuncios_eleitorais.server import mcp


def _mock_response() -> RespostaAnuncios:
    return RespostaAnuncios(
        data=[
            AnuncioEleitoral(
                id="999888777",
                page_id="111222333",
                page_name="Integração Teste",
                ad_delivery_start_time="2024-01-15T00:00:00+0000",
                ad_snapshot_url="https://www.facebook.com/ads/archive/render_ad/?id=999888777",
                ad_creative_bodies=["Anúncio de integração teste"],
                bylines="Financiador Teste",
                currency="BRL",
                spend=FaixaValor(lower_bound="500", upper_bound="999"),
                impressions=FaixaValor(lower_bound="5000", upper_bound="10000"),
                br_total_reach=25000,
                publisher_platforms=["facebook"],
            )
        ]
    )


@pytest.fixture(autouse=True)
def _set_token() -> None:
    """Set META_ACCESS_TOKEN for all tests."""
    with patch.dict(os.environ, {"META_ACCESS_TOKEN": "test_token"}):
        yield  # type: ignore[misc]


class TestToolRegistration:
    """Test that all tools are properly registered."""

    @pytest.mark.asyncio
    async def test_tools_registradas(self) -> None:
        async with Client(mcp) as c:
            tools = await c.list_tools()
            names = [t.name for t in tools]
            assert "buscar_anuncios_eleitorais" in names
            assert "buscar_anuncios_por_pagina" in names
            assert "buscar_anuncios_por_financiador" in names
            assert "buscar_anuncios_por_regiao" in names
            assert "analisar_demografia_anuncios" in names
            assert "buscar_anuncios_frase_exata" in names

    @pytest.mark.asyncio
    async def test_total_tools(self) -> None:
        async with Client(mcp) as c:
            tools = await c.list_tools()
            assert len(tools) == 6


class TestToolExecution:
    """Test tool execution via Client."""

    @pytest.mark.asyncio
    async def test_buscar_anuncios_eleitorais(self) -> None:
        with patch(
            "mcp_brasil.data.anuncios_eleitorais.client.buscar_anuncios",
            new_callable=AsyncMock,
            return_value=_mock_response(),
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "buscar_anuncios_eleitorais",
                    {"search_terms": "educação"},
                )
                text = result.data  # type: ignore[union-attr]
                assert "Integração Teste" in str(text)

    @pytest.mark.asyncio
    async def test_buscar_anuncios_por_pagina(self) -> None:
        with patch(
            "mcp_brasil.data.anuncios_eleitorais.client.buscar_anuncios",
            new_callable=AsyncMock,
            return_value=_mock_response(),
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "buscar_anuncios_por_pagina",
                    {"search_page_ids": ["111222333"]},
                )
                text = result.data  # type: ignore[union-attr]
                assert "Integração Teste" in str(text)

    @pytest.mark.asyncio
    async def test_buscar_anuncios_frase_exata(self) -> None:
        with patch(
            "mcp_brasil.data.anuncios_eleitorais.client.buscar_anuncios",
            new_callable=AsyncMock,
            return_value=_mock_response(),
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "buscar_anuncios_frase_exata",
                    {"frase": "governo federal"},
                )
                text = result.data  # type: ignore[union-attr]
                assert "Integração Teste" in str(text)


class TestResourceExecution:
    """Test resources via Client."""

    @pytest.mark.asyncio
    async def test_total_resources(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            assert len(resources) == 3

    @pytest.mark.asyncio
    async def test_ler_estados(self) -> None:
        async with Client(mcp) as c:
            result = await c.read_resource("data://estados-brasileiros")
            assert len(result) > 0


class TestPromptExecution:
    """Test prompts via Client."""

    @pytest.mark.asyncio
    async def test_total_prompts(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            assert len(prompts) == 3

    @pytest.mark.asyncio
    async def test_chamar_analise_candidato(self) -> None:
        async with Client(mcp) as c:
            result = await c.get_prompt(
                "analise_candidato",
                arguments={"nome_candidato": "Teste"},
            )
            assert len(result.messages) > 0
