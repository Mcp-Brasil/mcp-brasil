"""Tests for the BNDES CKAN HTTP client."""

import httpx
import pytest
import respx

from mcp_brasil.data.bndes import client
from mcp_brasil.data.bndes.constants import (
    DATASTORE_SEARCH_URL,
    PACKAGE_LIST_URL,
    PACKAGE_SEARCH_URL,
    PACKAGE_SHOW_URL,
)


class TestListarDatasets:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_dataset_names(self) -> None:
        respx.get(PACKAGE_LIST_URL).mock(
            return_value=httpx.Response(
                200,
                json={"success": True, "result": ["operacoes-financiamento", "desembolsos"]},
            )
        )
        result = await client.listar_datasets()
        assert result == ["operacoes-financiamento", "desembolsos"]

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_list(self) -> None:
        respx.get(PACKAGE_LIST_URL).mock(
            return_value=httpx.Response(200, json={"success": True, "result": []})
        )
        result = await client.listar_datasets()
        assert result == []


class TestBuscarDatasets:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_datasets(self) -> None:
        respx.get(PACKAGE_SEARCH_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "success": True,
                    "result": {
                        "count": 1,
                        "results": [
                            {
                                "name": "operacoes-financiamento",
                                "title": "Operações de Financiamento",
                                "notes": "Descrição das operações",
                                "num_resources": 4,
                                "resources": [
                                    {
                                        "id": "abc-123",
                                        "name": "Operações automáticas",
                                        "format": "CSV",
                                    }
                                ],
                            }
                        ],
                    },
                },
            )
        )
        result = await client.buscar_datasets("financiamento")
        assert len(result) == 1
        assert result[0].name == "operacoes-financiamento"
        assert result[0].title == "Operações de Financiamento"
        assert len(result[0].resources) == 1


class TestDetalharDataset:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_dataset(self) -> None:
        respx.get(PACKAGE_SHOW_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "success": True,
                    "result": {
                        "name": "desembolsos",
                        "title": "Desembolsos",
                        "notes": "Estatísticas de desembolsos",
                        "num_resources": 22,
                        "resources": [],
                    },
                },
            )
        )
        result = await client.detalhar_dataset("desembolsos")
        assert result is not None
        assert result.name == "desembolsos"
        assert result.num_resources == 22

    @pytest.mark.asyncio
    @respx.mock
    async def test_not_found(self) -> None:
        respx.get(PACKAGE_SHOW_URL).mock(
            return_value=httpx.Response(
                200, json={"success": False, "error": {"message": "Not found"}}
            )
        )
        result = await client.detalhar_dataset("nao-existe")
        assert result is None


class TestConsultarOperacoes:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_datastore_result(self) -> None:
        respx.get(DATASTORE_SEARCH_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "success": True,
                    "result": {
                        "total": 100,
                        "fields": [
                            {"id": "_id", "type": "int"},
                            {"id": "cliente", "type": "text"},
                            {"id": "uf", "type": "text"},
                        ],
                        "records": [
                            {"_id": 1, "cliente": "Empresa ABC", "uf": " SP"},
                        ],
                    },
                },
            )
        )
        result = await client.consultar_operacoes(uf="SP", limite=10)
        assert result.total == 100
        assert len(result.records) == 1
        assert result.records[0]["cliente"] == "Empresa ABC"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_result(self) -> None:
        respx.get(DATASTORE_SEARCH_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "success": True,
                    "result": {"total": 0, "fields": [], "records": []},
                },
            )
        )
        result = await client.consultar_operacoes(uf="XX")
        assert result.total == 0
        assert result.records == []


class TestListarInstituicoes:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_instituicoes(self) -> None:
        respx.get(DATASTORE_SEARCH_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "success": True,
                    "result": {
                        "total": 2,
                        "records": [
                            {
                                "cnpj": "00.000.000/0001-91",
                                "razao_social": "Banco do Brasil",
                                "site": "https://bb.com.br",
                            },
                            {
                                "cnpj": "60.746.948/0001-12",
                                "razao_social": "Bradesco",
                                "site": "https://bradesco.com.br",
                            },
                        ],
                    },
                },
            )
        )
        result = await client.listar_instituicoes_credenciadas()
        assert len(result) == 2
        assert result[0].nome == "Banco do Brasil"
        assert result[1].cnpj == "60.746.948/0001-12"
