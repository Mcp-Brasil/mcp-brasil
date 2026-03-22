"""Tests for IBGE resources — static reference data."""

import json

import pytest

from mcp_brasil.ibge.resources import estados_brasileiros, niveis_territoriais, regioes_brasileiras


class TestEstadosBrasileiros:
    def test_returns_valid_json(self) -> None:
        data = json.loads(estados_brasileiros())
        assert isinstance(data, list)

    def test_has_27_estados(self) -> None:
        data = json.loads(estados_brasileiros())
        assert len(data) == 27

    def test_each_estado_has_required_fields(self) -> None:
        data = json.loads(estados_brasileiros())
        for estado in data:
            assert "sigla" in estado
            assert "nome" in estado
            assert "regiao" in estado

    def test_contains_sp_and_rj(self) -> None:
        data = json.loads(estados_brasileiros())
        siglas = {e["sigla"] for e in data}
        assert "SP" in siglas
        assert "RJ" in siglas


class TestRegioesBrasileiras:
    def test_returns_valid_json(self) -> None:
        data = json.loads(regioes_brasileiras())
        assert isinstance(data, list)

    def test_has_5_regioes(self) -> None:
        data = json.loads(regioes_brasileiras())
        assert len(data) == 5

    def test_regioes_have_required_fields(self) -> None:
        data = json.loads(regioes_brasileiras())
        for r in data:
            assert "id" in r
            assert "sigla" in r
            assert "nome" in r
            assert "estados" in r


class TestNiveisTerritoriais:
    def test_returns_valid_json(self) -> None:
        data = json.loads(niveis_territoriais())
        assert isinstance(data, list)

    def test_contains_estado_and_municipio(self) -> None:
        data = json.loads(niveis_territoriais())
        niveis = {item["nivel"] for item in data}
        assert "estado" in niveis
        assert "municipio" in niveis


class TestResourcesIntegration:
    @pytest.mark.asyncio
    async def test_resources_accessible_via_client(self) -> None:
        from fastmcp import Client

        from mcp_brasil.ibge.server import mcp

        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            assert "data://estados" in uris
            assert "data://regioes" in uris
            assert "data://niveis-territoriais" in uris

    @pytest.mark.asyncio
    async def test_read_estados_resource(self) -> None:
        from fastmcp import Client

        from mcp_brasil.ibge.server import mcp

        async with Client(mcp) as c:
            contents = await c.read_resource("data://estados")
            text = contents[0].text if hasattr(contents[0], "text") else str(contents[0])
            assert "SP" in text
            assert "São Paulo" in text
