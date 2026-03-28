"""Tests for the BNDES tool functions."""

from unittest.mock import AsyncMock

import pytest

from mcp_brasil.data.bndes import tools
from mcp_brasil.data.bndes.schemas import (
    Dataset,
    DatasetResource,
    DatastoreResult,
    InstituicaoCredenciada,
)


def _make_ctx() -> AsyncMock:
    ctx = AsyncMock()
    ctx.info = AsyncMock()
    return ctx


class TestBuscarDatasetsBndes:
    @pytest.mark.asyncio
    async def test_returns_formatted(self, monkeypatch: pytest.MonkeyPatch) -> None:
        ds = Dataset(
            name="operacoes-financiamento",
            title="Operações de Financiamento",
            notes="Descrição das operações do BNDES",
            num_resources=4,
            resources=[],
        )
        monkeypatch.setattr(
            "mcp_brasil.data.bndes.tools.client.buscar_datasets",
            AsyncMock(return_value=[ds]),
        )
        result = await tools.buscar_datasets_bndes("financiamento", _make_ctx())
        assert "Operações de Financiamento" in result
        assert "4 recursos" in result

    @pytest.mark.asyncio
    async def test_empty(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setattr(
            "mcp_brasil.data.bndes.tools.client.buscar_datasets",
            AsyncMock(return_value=[]),
        )
        result = await tools.buscar_datasets_bndes("xyz", _make_ctx())
        assert "Nenhum dataset" in result


class TestDetalharDatasetBndes:
    @pytest.mark.asyncio
    async def test_returns_details(self, monkeypatch: pytest.MonkeyPatch) -> None:
        ds = Dataset(
            name="desembolsos",
            title="Desembolsos",
            notes="Dados de desembolsos",
            num_resources=22,
            resources=[
                DatasetResource(
                    id="abc-123", name="CSV principal", format="CSV", description="Dados"
                )
            ],
        )
        monkeypatch.setattr(
            "mcp_brasil.data.bndes.tools.client.detalhar_dataset",
            AsyncMock(return_value=ds),
        )
        result = await tools.detalhar_dataset_bndes("desembolsos", _make_ctx())
        assert "Desembolsos" in result
        assert "abc-123" in result
        assert "CSV principal" in result


class TestConsultarOperacoesBndes:
    @pytest.mark.asyncio
    async def test_returns_formatted(self, monkeypatch: pytest.MonkeyPatch) -> None:
        dr = DatastoreResult(
            total=1000,
            fields=["cliente", "uf", "valor_da_operacao_em_reais"],
            records=[
                {
                    "cliente": "Empresa ABC",
                    "uf": " SP",
                    "municipio": "SÃO PAULO",
                    "valor_da_operacao_em_reais": 150000.0,
                    "porte_do_cliente": "GRANDE",
                    "setor_cnae": "INDÚSTRIA",
                    "situacao_da_operacao": "Ativa",
                    "data_da_contratacao": "2023-01-15",
                }
            ],
        )
        monkeypatch.setattr(
            "mcp_brasil.data.bndes.tools.client.consultar_operacoes",
            AsyncMock(return_value=dr),
        )
        result = await tools.consultar_operacoes_bndes(_make_ctx(), uf="SP")
        assert "Empresa ABC" in result
        assert "R$ 150,000.00" in result
        assert "GRANDE" in result
        assert "offset=" in result  # pagination hint

    @pytest.mark.asyncio
    async def test_empty(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setattr(
            "mcp_brasil.data.bndes.tools.client.consultar_operacoes",
            AsyncMock(return_value=DatastoreResult()),
        )
        result = await tools.consultar_operacoes_bndes(_make_ctx(), uf="XX")
        assert "Nenhuma operação" in result


class TestListarInstituicoesBndes:
    @pytest.mark.asyncio
    async def test_returns_formatted(self, monkeypatch: pytest.MonkeyPatch) -> None:
        inst = [
            InstituicaoCredenciada(
                cnpj="00.000.000/0001-91",
                nome="Banco do Brasil",
                pagina_na_internet="https://bb.com.br",
            )
        ]
        monkeypatch.setattr(
            "mcp_brasil.data.bndes.tools.client.listar_instituicoes_credenciadas",
            AsyncMock(return_value=inst),
        )
        result = await tools.listar_instituicoes_bndes(_make_ctx())
        assert "Banco do Brasil" in result
        assert "00.000.000/0001-91" in result
