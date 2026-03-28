"""Tests for the OpenDataSUS HTTP client."""

import httpx
import pytest
import respx

from mcp_brasil.data.opendatasus import client
from mcp_brasil.data.opendatasus.constants import (
    DATASTORE_SEARCH_URL,
    PACKAGE_SEARCH_URL,
    PACKAGE_SHOW_URL,
)

# ---------------------------------------------------------------------------
# buscar_datasets
# ---------------------------------------------------------------------------


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
                                "id": "abc-123",
                                "name": "hospitais-leitos",
                                "title": "Hospitais e Leitos",
                                "notes": "Dados de hospitais.",
                                "organization": {"name": "ms", "title": "Ministério da Saúde"},
                                "tags": [{"name": "leitos", "display_name": "leitos"}],
                                "resources": [
                                    {
                                        "id": "res-1",
                                        "name": "dados.csv",
                                        "format": "CSV",
                                        "url": "https://example.com/dados.csv",
                                    }
                                ],
                                "metadata_created": "2024-01-01",
                                "metadata_modified": "2024-06-01",
                            }
                        ],
                    },
                },
            )
        )
        datasets, total = await client.buscar_datasets("leitos")
        assert total == 1
        assert len(datasets) == 1
        assert datasets[0].nome == "hospitais-leitos"
        assert datasets[0].titulo == "Hospitais e Leitos"
        assert datasets[0].organizacao == "Ministério da Saúde"
        assert len(datasets[0].tags) == 1
        assert datasets[0].total_recursos == 1

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_results(self) -> None:
        respx.get(PACKAGE_SEARCH_URL).mock(
            return_value=httpx.Response(
                200,
                json={"success": True, "result": {"count": 0, "results": []}},
            )
        )
        datasets, total = await client.buscar_datasets("inexistente")
        assert datasets == []
        assert total == 0

    @pytest.mark.asyncio
    @respx.mock
    async def test_sends_query_params(self) -> None:
        route = respx.get(PACKAGE_SEARCH_URL).mock(
            return_value=httpx.Response(
                200,
                json={"success": True, "result": {"count": 0, "results": []}},
            )
        )
        await client.buscar_datasets("vacinação", limite=5)
        req_url = str(route.calls[0].request.url)
        assert "q=" in req_url
        assert "rows=5" in req_url


# ---------------------------------------------------------------------------
# detalhar_dataset
# ---------------------------------------------------------------------------


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
                        "id": "abc-123",
                        "name": "srag",
                        "title": "SRAG",
                        "notes": "Vigilância de SRAG.",
                        "tags": [],
                        "resources": [],
                    },
                },
            )
        )
        ds = await client.detalhar_dataset("srag")
        assert ds is not None
        assert ds.nome == "srag"
        assert ds.titulo == "SRAG"

    @pytest.mark.asyncio
    @respx.mock
    async def test_not_found(self) -> None:
        respx.get(PACKAGE_SHOW_URL).mock(
            return_value=httpx.Response(
                200,
                json={"success": True, "result": {}},
            )
        )
        ds = await client.detalhar_dataset("inexistente")
        assert ds is None

    @pytest.mark.asyncio
    @respx.mock
    async def test_sends_id_param(self) -> None:
        route = respx.get(PACKAGE_SHOW_URL).mock(
            return_value=httpx.Response(
                200,
                json={"success": True, "result": {}},
            )
        )
        await client.detalhar_dataset("my-dataset")
        req_url = str(route.calls[0].request.url)
        assert "id=my-dataset" in req_url


# ---------------------------------------------------------------------------
# consultar_datastore
# ---------------------------------------------------------------------------


class TestConsultarDatastore:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_records(self) -> None:
        respx.get(DATASTORE_SEARCH_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "success": True,
                    "result": {
                        "total": 100,
                        "records": [
                            {"_id": 1, "uf": "SP", "municipio": "São Paulo", "valor": 42},
                            {"_id": 2, "uf": "RJ", "municipio": "Rio de Janeiro", "valor": 33},
                        ],
                    },
                },
            )
        )
        records, total = await client.consultar_datastore("res-uuid-123")
        assert total == 100
        assert len(records) == 2
        assert records[0].campos["uf"] == "SP"
        assert "_id" not in records[0].campos

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_datastore(self) -> None:
        respx.get(DATASTORE_SEARCH_URL).mock(
            return_value=httpx.Response(
                200,
                json={"success": True, "result": {"total": 0, "records": []}},
            )
        )
        records, total = await client.consultar_datastore("res-uuid-123")
        assert records == []
        assert total == 0

    @pytest.mark.asyncio
    @respx.mock
    async def test_with_query_and_filters(self) -> None:
        route = respx.get(DATASTORE_SEARCH_URL).mock(
            return_value=httpx.Response(
                200,
                json={"success": True, "result": {"total": 0, "records": []}},
            )
        )
        await client.consultar_datastore(
            "res-uuid",
            query="dengue",
            filtros={"uf": "SP"},
            limite=5,
            offset=10,
        )
        req_url = str(route.calls[0].request.url)
        assert "q=dengue" in req_url
        assert "limit=5" in req_url
        assert "offset=10" in req_url
        assert "filters=" in req_url


# ---------------------------------------------------------------------------
# Parse helpers
# ---------------------------------------------------------------------------


class TestParseHelpers:
    def test_extract_result_from_ckan(self) -> None:
        data = {"success": True, "result": {"count": 5}}
        assert client._extract_result(data) == {"count": 5}

    def test_extract_result_fallback(self) -> None:
        data = {"count": 5}
        assert client._extract_result(data) == {"count": 5}

    def test_extract_result_non_dict(self) -> None:
        assert client._extract_result([1, 2]) == {}

    def test_parse_dataset(self) -> None:
        raw = {
            "id": "123",
            "name": "test",
            "title": "Test Dataset",
            "notes": "Description",
            "organization": {"name": "org", "title": "Org Title"},
            "tags": [{"name": "tag1", "display_name": "Tag 1"}],
            "resources": [{"id": "r1", "name": "file.csv", "format": "CSV"}],
            "metadata_created": "2024-01-01",
            "metadata_modified": "2024-06-01",
        }
        ds = client._parse_dataset(raw)
        assert ds.nome == "test"
        assert ds.titulo == "Test Dataset"
        assert ds.organizacao == "Org Title"
        assert ds.total_recursos == 1
        assert ds.tags == ["Tag 1"]

    def test_parse_recurso(self) -> None:
        raw = {
            "id": "r1",
            "name": "dados.csv",
            "format": "CSV",
            "url": "https://example.com/dados.csv",
            "description": "Data file",
        }
        r = client._parse_recurso(raw)
        assert r.id == "r1"
        assert r.nome == "dados.csv"
        assert r.formato == "CSV"
