"""Tests for Bacen resources — catalog reference data."""

import json

import pytest

from mcp_brasil.bacen.resources import catalogo_series, categorias_series, indicadores_chave


class TestCatalogoSeries:
    def test_returns_valid_json(self) -> None:
        data = json.loads(catalogo_series())
        assert isinstance(data, list)

    def test_has_169_series(self) -> None:
        data = json.loads(catalogo_series())
        assert len(data) == 169

    def test_each_entry_has_required_fields(self) -> None:
        data = json.loads(catalogo_series())
        for entry in data[:5]:
            assert "codigo" in entry
            assert "nome" in entry
            assert "categoria" in entry
            assert "periodicidade" in entry

    def test_contains_selic(self) -> None:
        data = json.loads(catalogo_series())
        codigos = {e["codigo"] for e in data}
        assert 432 in codigos  # Selic


class TestCategoriasSeries:
    def test_returns_valid_json(self) -> None:
        data = json.loads(categorias_series())
        assert isinstance(data, list)

    def test_has_12_categorias(self) -> None:
        data = json.loads(categorias_series())
        assert len(data) == 12

    def test_each_has_nome_and_count(self) -> None:
        data = json.loads(categorias_series())
        for cat in data:
            assert "categoria" in cat
            assert "quantidade" in cat
            assert cat["quantidade"] > 0


class TestIndicadoresChave:
    def test_returns_valid_json(self) -> None:
        data = json.loads(indicadores_chave())
        assert isinstance(data, list)

    def test_has_5_indicadores(self) -> None:
        data = json.loads(indicadores_chave())
        assert len(data) == 5

    def test_contains_selic_and_ipca(self) -> None:
        data = json.loads(indicadores_chave())
        nomes = {i["nome"] for i in data}
        assert any("Selic" in n for n in nomes)
        assert any("IPCA" in n for n in nomes)


class TestResourcesIntegration:
    @pytest.mark.asyncio
    async def test_resources_accessible_via_client(self) -> None:
        from fastmcp import Client

        from mcp_brasil.bacen.server import mcp

        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            assert "data://catalogo" in uris
            assert "data://categorias" in uris
            assert "data://indicadores-chave" in uris

    @pytest.mark.asyncio
    async def test_read_indicadores_resource(self) -> None:
        from fastmcp import Client

        from mcp_brasil.bacen.server import mcp

        async with Client(mcp) as c:
            contents = await c.read_resource("data://indicadores-chave")
            text = contents[0].text if hasattr(contents[0], "text") else str(contents[0])
            assert "Selic" in text
