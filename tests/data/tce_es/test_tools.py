"""Tests for the TCE-ES tool functions."""

from __future__ import annotations

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from mcp_brasil.data.tce_es import tools
from mcp_brasil.data.tce_es.schemas import (
    ContratacaoMunicipio,
    Contrato,
    Licitacao,
    Obra,
)

CLIENT_MODULE = "mcp_brasil.data.tce_es.client"


def _mock_ctx() -> MagicMock:
    ctx = MagicMock()
    ctx.info = AsyncMock()
    ctx.warning = AsyncMock()
    return ctx


# ---------------------------------------------------------------------------
# buscar_licitacoes_es
# ---------------------------------------------------------------------------


class TestBuscarLicitacoesEs:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = (
            [
                Licitacao(
                    Modalidade="Pregão Eletrônico",
                    NumeroEdital="90001",
                    AnoEdital="2024",
                    Objeto="Aquisição de material de TI",
                    ValorReferencia="R$ 50.000,00",
                    ValorHomologado="R$ 48.000,00",
                    Situacao="Homologado",
                    DataAbertura="2024-03-01",
                )
            ],
            1,
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_licitacoes",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_licitacoes_es(ctx, q="material")
        assert "90001" in result
        assert "Pregão Eletrônico" in result
        assert "material de TI" in result
        assert "Homologado" in result
        assert "1 licitações" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_licitacoes",
            new_callable=AsyncMock,
            return_value=([], 0),
        ):
            result = await tools.buscar_licitacoes_es(ctx)
        assert "Nenhuma licitação encontrada" in result

    @pytest.mark.asyncio
    async def test_pagination_hint(self) -> None:
        mock_licitacoes = [
            Licitacao(NumeroEdital=f"{i:05d}", AnoEdital="2024", Objeto=f"Objeto {i}")
            for i in range(20)
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_licitacoes",
            new_callable=AsyncMock,
            return_value=(mock_licitacoes, 50),
        ):
            result = await tools.buscar_licitacoes_es(ctx)
        assert "Mostrando 20 de 50" in result
        assert "deslocamento=" in result


# ---------------------------------------------------------------------------
# buscar_contratos_es
# ---------------------------------------------------------------------------


class TestBuscarContratosEs:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = (
            [
                Contrato(
                    ContratoNumero="1",
                    ContratoAno="2024",
                    Modalidade="Pregão",
                    ResumoObjeto="Fornecimento de material de escritório",
                    FornecedorNome="EMPRESA XYZ LTDA",
                    VigenciaAtualValorGlobal="120000.00",
                    TermoOriginalDataFimVigencia="2024-12-31",
                    Setor="Secretaria Administrativa",
                )
            ],
            1,
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_contratos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_contratos_es(ctx, ano=2024)
        assert "1/2024" in result
        assert "EMPRESA XYZ LTDA" in result
        assert "material de escritório" in result
        assert "2024-12-31" in result
        assert "Secretaria Administrativa" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_contratos",
            new_callable=AsyncMock,
            return_value=([], 0),
        ):
            result = await tools.buscar_contratos_es(ctx)
        assert "Nenhum contrato encontrado" in result

    @pytest.mark.asyncio
    async def test_pagination_hint(self) -> None:
        mock_contratos = [Contrato(ContratoNumero=str(i), ContratoAno="2024") for i in range(20)]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_contratos",
            new_callable=AsyncMock,
            return_value=(mock_contratos, 45),
        ):
            result = await tools.buscar_contratos_es(ctx)
        assert "Mostrando 20 de 45" in result
        assert "deslocamento=" in result


# ---------------------------------------------------------------------------
# buscar_contratacoes_municipios_es
# ---------------------------------------------------------------------------


class TestBuscarContratacoesMunicipiosEs:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = (
            [
                ContratacaoMunicipio(
                    NomeUnidadeGestoraReferencia="PREFEITURA DE VITÓRIA",
                    NomeEsferaAdministrativa="Municipal",
                    ObjetoContratacao="Reforma de escola municipal",
                    ModalidadeLicitacao="Tomada de Preços",
                    ValorEstimado="500000.00",
                    ValorTotalContratacao="480000.00",
                    AnoReferencia="2024",
                    SituacaoContratacao="Concluído",
                )
            ],
            127297,
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_contratacoes_municipios",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_contratacoes_municipios_es(ctx, ano_referencia=2024)
        assert "PREFEITURA DE VITÓRIA" in result
        assert "Municipal" in result
        assert "escola municipal" in result
        assert "Tomada de Preços" in result
        assert "127297 contratações" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_contratacoes_municipios",
            new_callable=AsyncMock,
            return_value=([], 0),
        ):
            result = await tools.buscar_contratacoes_municipios_es(ctx)
        assert "Nenhuma contratação encontrada" in result

    @pytest.mark.asyncio
    async def test_pagination_hint(self) -> None:
        mock_contratacoes = [
            ContratacaoMunicipio(NomeUnidadeGestoraReferencia=f"PREFEITURA {i}") for i in range(20)
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_contratacoes_municipios",
            new_callable=AsyncMock,
            return_value=(mock_contratacoes, 100),
        ):
            result = await tools.buscar_contratacoes_municipios_es(ctx)
        assert "Mostrando 20 de 100" in result
        assert "deslocamento=" in result


# ---------------------------------------------------------------------------
# buscar_obras_es
# ---------------------------------------------------------------------------


class TestBuscarObrasEs:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = (
            [
                Obra(
                    Licitacao="PE 001/2023",
                    Contrato="CT-001/2023",
                    Empresa="CONSTRUTORA ABC LTDA",
                    EmpresaCNPJ="12345678000199",
                    ValorInicial="1.282.499,39",
                    Situacao="Em Execução",
                    DataAssinaturaContrato="2023-03-15",
                )
            ],
            1250,
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_obras",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_obras_es(ctx, q="ABC")
        assert "CT-001/2023" in result
        assert "CONSTRUTORA ABC LTDA" in result
        assert "1.282.499,39" in result
        assert "Em Execução" in result
        assert "2023-03-15" in result
        assert "1250 obras" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_obras",
            new_callable=AsyncMock,
            return_value=([], 0),
        ):
            result = await tools.buscar_obras_es(ctx)
        assert "Nenhuma obra encontrada" in result

    @pytest.mark.asyncio
    async def test_pagination_hint(self) -> None:
        mock_obras = [Obra(Contrato=f"CT-{i}", Empresa=f"Empresa {i}") for i in range(20)]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_obras",
            new_callable=AsyncMock,
            return_value=(mock_obras, 200),
        ):
            result = await tools.buscar_obras_es(ctx)
        assert "Mostrando 20 de 200" in result
        assert "deslocamento=" in result
