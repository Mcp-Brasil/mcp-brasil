"""Tests for the NucleoGov tool functions."""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from mcp_brasil.data.nucleogov import tools
from mcp_brasil.data.nucleogov.schemas import (
    AtoNormativo,
    Contrato,
    Despesa,
    Licitacao,
    ResultadoAtos,
    ResultadoContratos,
    ResultadoDespesas,
    ResultadoLicitacoes,
)
from mcp_brasil.exceptions import HttpClientError

CLIENT_MODULE = "mcp_brasil.data.nucleogov.client"
PORTAL = "acessoainformacao.cidade.go.gov.br"


def _mock_ctx() -> MagicMock:
    ctx = MagicMock()
    ctx.info = AsyncMock()
    return ctx


# ---------------------------------------------------------------------------
# buscar_despesas
# ---------------------------------------------------------------------------


class TestBuscarDespesas:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = ResultadoDespesas(
            despesas=[
                Despesa(
                    numero_empenho="42",
                    data="31/12/2024",
                    orgao="PREFEITURA MUNICIPAL DE CIDADE",
                    fornecedor="Empresa Exemplo Ltda",
                    cpf_cnpj="12.345.678/0001-90",
                    historico="Aquisicao de material",
                    valor_empenho=500.0,
                    valor_pago=0.0,
                )
            ],
            total=1,
        )
        with patch(
            f"{CLIENT_MODULE}.buscar_despesas",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_despesas(_mock_ctx(), PORTAL)
        assert "Empenho 42" in result
        assert "Empresa Exemplo Ltda" in result
        assert "R$ 500,00" in result
        assert PORTAL in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        mock_data = ResultadoDespesas(despesas=[], total=0)
        with patch(
            f"{CLIENT_MODULE}.buscar_despesas",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_despesas(_mock_ctx(), PORTAL)
        assert "Nenhum resultado para estes filtros" in result
        assert "não existe" not in result

    @pytest.mark.asyncio
    async def test_client_error_is_reported(self) -> None:
        with patch(
            f"{CLIENT_MODULE}.buscar_despesas",
            new_callable=AsyncMock,
            side_effect=HttpClientError("portal fora do ar"),
        ):
            result = await tools.buscar_despesas(_mock_ctx(), PORTAL)
        assert "Não foi possível consultar" in result
        assert "portal fora do ar" in result

    @pytest.mark.asyncio
    async def test_empty_page_with_total_positive_is_not_nonexistence(self) -> None:
        mock_data = ResultadoDespesas(despesas=[], total=3)
        with patch(
            f"{CLIENT_MODULE}.buscar_despesas",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_despesas(_mock_ctx(), PORTAL, cnpj="12345678000190")
        assert "Nenhum resultado" not in result
        assert "declara 3 registros" in result

    @pytest.mark.asyncio
    async def test_malicious_portal_is_reported_without_request(self) -> None:
        with patch(
            f"{CLIENT_MODULE}.buscar_despesas",
            new_callable=AsyncMock,
            side_effect=ValueError("o portal deve ser um domínio gov.br"),
        ) as mock_call:
            result = await tools.buscar_despesas(_mock_ctx(), "https://evil.example.com")
        assert "Não foi possível consultar" in result
        assert mock_call.await_count == 1


# ---------------------------------------------------------------------------
# buscar_contratos
# ---------------------------------------------------------------------------


class TestBuscarContratos:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = ResultadoContratos(
            contratos=[
                Contrato(
                    numero="001/2024",
                    ano=2024,
                    orgao="PREFEITURA MUNICIPAL DE CIDADE",
                    contratado_nome="Empresa Exemplo Ltda",
                    objeto="Locacao de veiculos",
                    valor=10000.0,
                    situacao="Ativo",
                )
            ],
            total=1,
        )
        with patch(
            f"{CLIENT_MODULE}.buscar_contratos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_contratos(_mock_ctx(), PORTAL)
        assert "001/2024" in result
        assert "Empresa Exemplo Ltda" in result
        assert "R$ 10.000,00" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        mock_data = ResultadoContratos(contratos=[], total=0)
        with patch(
            f"{CLIENT_MODULE}.buscar_contratos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_contratos(_mock_ctx(), PORTAL)
        assert "Nenhum resultado para estes filtros" in result


# ---------------------------------------------------------------------------
# buscar_licitacoes / buscar_dispensas
# ---------------------------------------------------------------------------


class TestBuscarLicitacoes:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = ResultadoLicitacoes(
            licitacoes=[
                Licitacao(
                    numero="001/2024",
                    modalidade="PREGÃO ELETRÔNICO",
                    orgao="PREFEITURA MUNICIPAL DE CIDADE",
                    descricao="Aquisicao de equipamentos",
                    valor_estimado=100000.0,
                    situacao="Aberta",
                )
            ],
            total=1,
        )
        with patch(
            f"{CLIENT_MODULE}.buscar_licitacoes",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_licitacoes(_mock_ctx(), PORTAL)
        assert "001/2024" in result
        assert "PREGÃO ELETRÔNICO" in result
        assert "R$ 100.000,00" in result

    @pytest.mark.asyncio
    async def test_dispensas_passes_flag(self) -> None:
        mock_data = ResultadoLicitacoes(licitacoes=[Licitacao(numero="1")], total=1)
        with patch(
            f"{CLIENT_MODULE}.buscar_licitacoes",
            new_callable=AsyncMock,
            return_value=mock_data,
        ) as mock_call:
            result = await tools.buscar_dispensas(_mock_ctx(), PORTAL)
        assert mock_call.await_args.kwargs["dispensas"] is True
        assert "dispensas/inexigibilidades" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        mock_data = ResultadoLicitacoes(licitacoes=[], total=0)
        with patch(
            f"{CLIENT_MODULE}.buscar_licitacoes",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_dispensas(_mock_ctx(), PORTAL)
        assert "Nenhum resultado para estes filtros" in result


# ---------------------------------------------------------------------------
# buscar_legislacao
# ---------------------------------------------------------------------------


class TestBuscarLegislacao:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = ResultadoAtos(
            atos=[
                AtoNormativo(
                    tipo="DECRETO",
                    numero="100",
                    ementa="Dispoe sobre a estrutura administrativa",
                    data_publicacao="01/02/2024",
                )
            ],
            total=1,
        )
        with patch(
            f"{CLIENT_MODULE}.buscar_legislacao",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_legislacao(_mock_ctx(), PORTAL, municipio="Cidade")
        assert "DECRETO 100" in result
        assert "estrutura administrativa" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        mock_data = ResultadoAtos(atos=[], total=0)
        with patch(
            f"{CLIENT_MODULE}.buscar_legislacao",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_legislacao(_mock_ctx(), PORTAL, municipio="Cidade")
        assert "Nenhum resultado para estes filtros" in result

    @pytest.mark.asyncio
    async def test_client_error_is_reported(self) -> None:
        with patch(
            f"{CLIENT_MODULE}.buscar_legislacao",
            new_callable=AsyncMock,
            side_effect=HttpClientError("acao nao encontrada"),
        ):
            result = await tools.buscar_legislacao(_mock_ctx(), PORTAL, municipio="Cidade")
        assert "Não foi possível consultar" in result
