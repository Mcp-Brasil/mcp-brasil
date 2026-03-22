"""Tests for Câmara resources — static reference data."""

import json

import pytest

from mcp_brasil.camara.resources import info_api, legislaturas_recentes, tipos_proposicao


class TestTiposProposicao:
    def test_returns_valid_json(self) -> None:
        data = json.loads(tipos_proposicao())
        assert isinstance(data, list)

    def test_has_types(self) -> None:
        data = json.loads(tipos_proposicao())
        assert len(data) >= 8

    def test_each_type_has_required_fields(self) -> None:
        data = json.loads(tipos_proposicao())
        for t in data:
            assert "sigla" in t
            assert "nome" in t
            assert "descricao" in t

    def test_contains_pl_and_pec(self) -> None:
        data = json.loads(tipos_proposicao())
        siglas = {t["sigla"] for t in data}
        assert "PL" in siglas
        assert "PEC" in siglas


class TestLegislaturasRecentes:
    def test_returns_valid_json(self) -> None:
        data = json.loads(legislaturas_recentes())
        assert isinstance(data, list)

    def test_has_legislaturas(self) -> None:
        data = json.loads(legislaturas_recentes())
        assert len(data) >= 4

    def test_each_legislatura_has_required_fields(self) -> None:
        data = json.loads(legislaturas_recentes())
        for leg in data:
            assert "id" in leg
            assert "inicio" in leg
            assert "fim" in leg
            assert "descricao" in leg

    def test_contains_current_legislatura(self) -> None:
        data = json.loads(legislaturas_recentes())
        ids = {leg["id"] for leg in data}
        assert 57 in ids


class TestInfoApi:
    def test_returns_valid_json(self) -> None:
        data = json.loads(info_api())
        assert isinstance(data, dict)

    def test_has_required_fields(self) -> None:
        data = json.loads(info_api())
        assert "nome" in data
        assert "url_base" in data
        assert "autenticacao" in data
        assert "formato" in data

    def test_no_auth_required(self) -> None:
        data = json.loads(info_api())
        assert "Não requer" in data["autenticacao"]


class TestResourcesIntegration:
    @pytest.mark.asyncio
    async def test_resources_accessible_via_client(self) -> None:
        from fastmcp import Client

        from mcp_brasil.camara.server import mcp

        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            assert "data://tipos-proposicao" in uris
            assert "data://legislaturas" in uris
            assert "data://info-api" in uris

    @pytest.mark.asyncio
    async def test_read_tipos_proposicao_resource(self) -> None:
        from fastmcp import Client

        from mcp_brasil.camara.server import mcp

        async with Client(mcp) as c:
            contents = await c.read_resource("data://tipos-proposicao")
            text = contents[0].text if hasattr(contents[0], "text") else str(contents[0])
            assert "PL" in text
