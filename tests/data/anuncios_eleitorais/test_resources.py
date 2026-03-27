"""Tests for anuncios_eleitorais resources — unit + Client integration."""

from __future__ import annotations

import json

import pytest
from fastmcp import Client

from mcp_brasil.data.anuncios_eleitorais.resources import (
    campos_disponiveis,
    estados_brasileiros,
    parametros_busca,
)
from mcp_brasil.data.anuncios_eleitorais.server import mcp


class TestEstadosBrasileiros:
    """Tests for estados_brasileiros resource."""

    def test_retorna_json_valido(self) -> None:
        result = json.loads(estados_brasileiros())
        assert isinstance(result, list)

    def test_todos_27_estados(self) -> None:
        result = json.loads(estados_brasileiros())
        assert len(result) == 27

    def test_estrutura_estado(self) -> None:
        result = json.loads(estados_brasileiros())
        for estado in result:
            assert "sigla" in estado
            assert "nome" in estado
            assert len(estado["sigla"]) == 2

    def test_contem_sp(self) -> None:
        result = json.loads(estados_brasileiros())
        siglas = [e["sigla"] for e in result]
        assert "SP" in siglas

    def test_ordenado_por_sigla(self) -> None:
        result = json.loads(estados_brasileiros())
        siglas = [e["sigla"] for e in result]
        assert siglas == sorted(siglas)


class TestParametrosBusca:
    """Tests for parametros_busca resource."""

    def test_retorna_json_valido(self) -> None:
        result = json.loads(parametros_busca())
        assert isinstance(result, dict)

    def test_contem_parametros_principais(self) -> None:
        result = json.loads(parametros_busca())
        assert "search_terms" in result
        assert "ad_active_status" in result
        assert "bylines" in result
        assert "delivery_by_region" in result

    def test_cada_parametro_tem_descricao(self) -> None:
        result = json.loads(parametros_busca())
        for key, param in result.items():
            assert "descricao" in param, f"Parâmetro '{key}' sem descrição"


class TestCamposDisponiveis:
    """Tests for campos_disponiveis resource."""

    def test_retorna_json_valido(self) -> None:
        result = json.loads(campos_disponiveis())
        assert isinstance(result, dict)

    def test_contem_categorias(self) -> None:
        result = json.loads(campos_disponiveis())
        assert "basicos" in result
        assert "politicos_brasil" in result

    def test_campos_basicos(self) -> None:
        result = json.loads(campos_disponiveis())
        basicos = result["basicos"]
        assert "id" in basicos
        assert "page_name" in basicos
        assert "ad_snapshot_url" in basicos

    def test_campos_politicos(self) -> None:
        result = json.loads(campos_disponiveis())
        politicos = result["politicos_brasil"]
        assert "spend" in politicos
        assert "impressions" in politicos
        assert "br_total_reach" in politicos


class TestResourceRegistration:
    """Test resources are properly registered via Client."""

    @pytest.mark.asyncio
    async def test_recursos_registrados(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = [str(r.uri) for r in resources]
            assert "data://estados-brasileiros" in uris
            assert "data://parametros-busca" in uris
            assert "data://campos-disponiveis" in uris

    @pytest.mark.asyncio
    async def test_ler_estados_brasileiros(self) -> None:
        async with Client(mcp) as c:
            result = await c.read_resource("data://estados-brasileiros")
            data = json.loads(result[0].text)  # type: ignore[union-attr]
            assert len(data) == 27

    @pytest.mark.asyncio
    async def test_ler_parametros_busca(self) -> None:
        async with Client(mcp) as c:
            result = await c.read_resource("data://parametros-busca")
            data = json.loads(result[0].text)  # type: ignore[union-attr]
            assert "search_terms" in data
