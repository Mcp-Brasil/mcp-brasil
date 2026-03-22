"""Tests for the Compras tool functions.

Tools are tested by mocking client functions (never HTTP).
Context is mocked via MagicMock with async methods.
"""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from mcp_brasil.compras import tools
from mcp_brasil.compras.schemas import (
    AtaRegistroPreco,
    AtaResultado,
    Contratacao,
    ContratacaoResultado,
    Contrato,
    ContratoResultado,
)

CLIENT_MODULE = "mcp_brasil.compras.client"


def _mock_ctx() -> MagicMock:
    """Create a mock Context with async log methods."""
    ctx = MagicMock()
    ctx.info = AsyncMock()
    ctx.warning = AsyncMock()
    return ctx


# ---------------------------------------------------------------------------
# buscar_contratacoes
# ---------------------------------------------------------------------------


class TestBuscarContratacoes:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = ContratacaoResultado(
            total=1,
            contratacoes=[
                Contratacao(
                    orgao_cnpj="00394460000141",
                    orgao_nome="Ministério da Educação",
                    objeto="Aquisição de computadores",
                    modalidade_id=1,
                    modalidade_nome="Pregão eletrônico",
                    situacao_nome="Publicada",
                    valor_estimado=500000.0,
                    valor_homologado=480000.0,
                    data_publicacao="2024-03-15",
                    municipio="Brasília",
                    uf="DF",
                    esfera="Federal",
                    link_pncp="https://pncp.gov.br/app/editais/123",
                ),
            ],
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_contratacoes",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_contratacoes("computadores", ctx)
        assert "Aquisição de computadores" in result
        assert "Ministério da Educação" in result
        assert "00394460000141" in result
        assert "Pregão eletrônico" in result
        assert "Publicada" in result
        assert "R$ 500.000,00" in result
        assert "R$ 480.000,00" in result
        assert "Brasília" in result
        assert "DF" in result
        assert "Federal" in result
        assert "Ver no PNCP" in result
        assert "1 contratações" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        mock_data = ContratacaoResultado(total=0, contratacoes=[])
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_contratacoes",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_contratacoes("xyzinexistente", ctx)
        assert "Nenhuma contratação encontrada" in result
        assert "xyzinexistente" in result


# ---------------------------------------------------------------------------
# buscar_contratos
# ---------------------------------------------------------------------------


class TestBuscarContratos:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = ContratoResultado(
            total=1,
            contratos=[
                Contrato(
                    orgao_cnpj="00394460000141",
                    orgao_nome="Ministério da Saúde",
                    numero_contrato="2024/001",
                    objeto="Fornecimento de medicamentos",
                    fornecedor_cnpj="12345678000199",
                    fornecedor_nome="Empresa Pharma LTDA",
                    valor_inicial=100000.0,
                    valor_final=95000.0,
                    vigencia_inicio="2024-01-01",
                    vigencia_fim="2024-12-31",
                    situacao="Vigente",
                ),
            ],
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_contratos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_contratos(ctx, texto="medicamentos")
        assert "Fornecimento de medicamentos" in result
        assert "Ministério da Saúde" in result
        assert "Empresa Pharma LTDA" in result
        assert "12345678000199" in result
        assert "2024/001" in result
        assert "R$ 95.000,00" in result
        assert "2024-01-01" in result
        assert "2024-12-31" in result
        assert "Vigente" in result
        assert "1 contratos" in result

    @pytest.mark.asyncio
    async def test_missing_filter_validation(self) -> None:
        ctx = _mock_ctx()
        result = await tools.buscar_contratos(ctx)
        assert "Informe pelo menos um filtro" in result


# ---------------------------------------------------------------------------
# buscar_atas
# ---------------------------------------------------------------------------


class TestBuscarAtas:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = AtaResultado(
            total=1,
            atas=[
                AtaRegistroPreco(
                    orgao_cnpj="00394460000141",
                    orgao_nome="Universidade Federal",
                    numero_ata="2024/010",
                    objeto="Registro de preços para material de escritório",
                    fornecedor_cnpj="98765432000155",
                    fornecedor_nome="Papelaria Central LTDA",
                    valor_total=250000.0,
                    vigencia_inicio="2024-06-01",
                    vigencia_fim="2025-05-31",
                    situacao="Vigente",
                ),
            ],
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_atas",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_atas(ctx, texto="escritório")
        assert "material de escritório" in result
        assert "Universidade Federal" in result
        assert "Papelaria Central LTDA" in result
        assert "98765432000155" in result
        assert "2024/010" in result
        assert "R$ 250.000,00" in result
        assert "2024-06-01" in result
        assert "2025-05-31" in result
        assert "Vigente" in result
        assert "1 atas" in result

    @pytest.mark.asyncio
    async def test_missing_filter_validation(self) -> None:
        ctx = _mock_ctx()
        result = await tools.buscar_atas(ctx)
        assert "Informe pelo menos um filtro" in result
