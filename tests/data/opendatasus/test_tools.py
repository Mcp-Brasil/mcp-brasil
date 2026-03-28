"""Tests for the OpenDataSUS tool functions."""

from unittest.mock import AsyncMock, patch

import pytest

from mcp_brasil.data.opendatasus import tools
from mcp_brasil.data.opendatasus.schemas import (
    DatasetOpenDataSUS,
    RecursoDataset,
    RegistroDataStore,
)

CLIENT_MODULE = "mcp_brasil.data.opendatasus.client"


@pytest.fixture
def ctx() -> AsyncMock:
    mock = AsyncMock()
    mock.info = AsyncMock()
    return mock


SAMPLE_DATASET = DatasetOpenDataSUS(
    id="abc-123",
    nome="hospitais-leitos",
    titulo="Hospitais e Leitos",
    descricao="Dados de hospitais e leitos no Brasil.",
    organizacao="Ministério da Saúde",
    tags=["leitos", "hospitais"],
    recursos=[
        RecursoDataset(id="res-1", nome="dados.csv", formato="CSV", url="https://example.com")
    ],
    total_recursos=1,
    data_criacao="2024-01-01",
    data_atualizacao="2024-06-01",
)


SAMPLE_RECORD = RegistroDataStore(campos={"uf": "SP", "municipio": "São Paulo", "leitos": 1000})


# ---------------------------------------------------------------------------
# buscar_datasets
# ---------------------------------------------------------------------------


class TestBuscarDatasets:
    @pytest.mark.asyncio
    async def test_returns_table(self, ctx: AsyncMock) -> None:
        with patch(
            f"{CLIENT_MODULE}.buscar_datasets",
            new_callable=AsyncMock,
            return_value=([SAMPLE_DATASET], 1),
        ):
            result = await tools.buscar_datasets(ctx, "leitos")
            assert "Hospitais e Leitos" in result
            assert "OpenDataSUS" in result

    @pytest.mark.asyncio
    async def test_no_results(self, ctx: AsyncMock) -> None:
        with patch(
            f"{CLIENT_MODULE}.buscar_datasets",
            new_callable=AsyncMock,
            return_value=([], 0),
        ):
            result = await tools.buscar_datasets(ctx, "inexistente")
            assert "Nenhum dataset encontrado" in result


# ---------------------------------------------------------------------------
# detalhar_dataset
# ---------------------------------------------------------------------------


class TestDetalharDataset:
    @pytest.mark.asyncio
    async def test_returns_details(self, ctx: AsyncMock) -> None:
        with patch(
            f"{CLIENT_MODULE}.detalhar_dataset",
            new_callable=AsyncMock,
            return_value=SAMPLE_DATASET,
        ):
            result = await tools.detalhar_dataset(ctx, "hospitais-leitos")
            assert "Hospitais e Leitos" in result
            assert "Ministério da Saúde" in result
            assert "dados.csv" in result

    @pytest.mark.asyncio
    async def test_not_found(self, ctx: AsyncMock) -> None:
        with patch(
            f"{CLIENT_MODULE}.detalhar_dataset",
            new_callable=AsyncMock,
            return_value=None,
        ):
            result = await tools.detalhar_dataset(ctx, "inexistente")
            assert "não encontrado" in result


# ---------------------------------------------------------------------------
# consultar_datastore
# ---------------------------------------------------------------------------


class TestConsultarDatastore:
    @pytest.mark.asyncio
    async def test_returns_table(self, ctx: AsyncMock) -> None:
        with patch(
            f"{CLIENT_MODULE}.consultar_datastore",
            new_callable=AsyncMock,
            return_value=([SAMPLE_RECORD], 100),
        ):
            result = await tools.consultar_datastore(ctx, "res-uuid")
            assert "DataStore" in result
            assert "SP" in result
            assert "São Paulo" in result

    @pytest.mark.asyncio
    async def test_no_records(self, ctx: AsyncMock) -> None:
        with patch(
            f"{CLIENT_MODULE}.consultar_datastore",
            new_callable=AsyncMock,
            return_value=([], 0),
        ):
            result = await tools.consultar_datastore(ctx, "res-uuid")
            assert "Nenhum registro" in result


# ---------------------------------------------------------------------------
# listar_datasets_conhecidos
# ---------------------------------------------------------------------------


class TestListarDatasetsConhecidos:
    @pytest.mark.asyncio
    async def test_returns_known_datasets(self, ctx: AsyncMock) -> None:
        result = await tools.listar_datasets_conhecidos(ctx)
        assert "Datasets conhecidos" in result
        assert "hospitais-leitos" in result
        assert "srag" in result


# ---------------------------------------------------------------------------
# buscar_com_filtro
# ---------------------------------------------------------------------------


class TestBuscarComFiltro:
    @pytest.mark.asyncio
    async def test_returns_filtered(self, ctx: AsyncMock) -> None:
        with patch(
            f"{CLIENT_MODULE}.consultar_datastore",
            new_callable=AsyncMock,
            return_value=([SAMPLE_RECORD], 50),
        ):
            result = await tools.buscar_com_filtro(ctx, "res-uuid", "uf", "SP")
            assert "filtrado" in result.lower()
            assert "uf=SP" in result

    @pytest.mark.asyncio
    async def test_no_results(self, ctx: AsyncMock) -> None:
        with patch(
            f"{CLIENT_MODULE}.consultar_datastore",
            new_callable=AsyncMock,
            return_value=([], 0),
        ):
            result = await tools.buscar_com_filtro(ctx, "res-uuid", "uf", "XX")
            assert "Nenhum registro" in result
