"""Tests for the TCM-GO tool functions."""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from mcp_brasil.data.tcm_go import tools
from mcp_brasil.data.tcm_go.schemas import (
    Acordao,
    AcordaoResultado,
    Decisao,
    Envolvido,
    Processo,
    ProcessoResultado,
    ValorCampo,
)
from mcp_brasil.exceptions import HttpClientError

CLIENT_MODULE = "mcp_brasil.data.tcm_go.client"


def _mock_ctx() -> MagicMock:
    ctx = MagicMock()
    ctx.info = AsyncMock()
    ctx.warning = AsyncMock()
    return ctx


# ---------------------------------------------------------------------------
# listar_valores_campo
# ---------------------------------------------------------------------------


class TestListarValoresCampo:
    @pytest.mark.asyncio
    async def test_formats_values(self) -> None:
        mock_data = [
            ValorCampo(valor="Contas de Gestão", quantidade=100),
            ValorCampo(valor="Pensão", quantidade=200),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_valores_campo",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.listar_valores_campo(ctx, campo="tipo_protocolo.keyword")
        assert "tipo_protocolo.keyword" in result
        assert "Contas de Gestão" in result
        assert "200" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_valores_campo",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.listar_valores_campo(ctx, campo="area.keyword")
        assert "Nenhum valor retornado" in result

    @pytest.mark.asyncio
    async def test_http_error_is_reported(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_valores_campo",
            new_callable=AsyncMock,
            side_effect=HttpClientError("HTTP 400"),
        ):
            result = await tools.listar_valores_campo(ctx, campo="area.keyword")
        assert "Não foi possível consultar" in result
        assert "HTTP 400" in result


# ---------------------------------------------------------------------------
# buscar_processos
# ---------------------------------------------------------------------------


class TestBuscarProcessos:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = ProcessoResultado(
            total=2,
            pagina=0,
            processos=[
                Processo(
                    numero="00000000.2024.000.0.00000.0001",
                    municipio="ANÁPOLIS",
                    unidade_gestora="PREFEITURA DE ANÁPOLIS",
                    tipo_protocolo="Contas de Gestão",
                    ano_competencia=2024,
                    periodo_competencia="Período 1/2024",
                    relator="Cons. Fulano de Tal",
                    st_fase="ARQUIVADO",
                    finalizado="SIM",
                    envolvidos=[Envolvido(nome="Fulano de Tal", tipo_envolvimento="RELATOR")],
                    decisoes=[Decisao(nome="Dar Ciência", arquivo="Relatório Conclusivo")],
                )
            ],
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_processos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_processos(ctx, municipio="ANÁPOLIS")
        assert "00000000.2024.000.0.00000.0001" in result
        assert "PREFEITURA DE ANÁPOLIS" in result
        assert "Cons. Fulano de Tal" in result
        assert "Relatório Conclusivo" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        mock_data = ProcessoResultado(total=0, pagina=0, processos=[])
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_processos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_processos(ctx, municipio="ANÁPOLIS")
        assert "Nenhum processo encontrado" in result
        assert "não existe" not in result

    @pytest.mark.asyncio
    async def test_http_error_is_reported(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_processos",
            new_callable=AsyncMock,
            side_effect=HttpClientError("timeout"),
        ):
            result = await tools.buscar_processos(ctx)
        assert "Não foi possível consultar" in result
        assert "timeout" in result


# ---------------------------------------------------------------------------
# buscar_acordaos
# ---------------------------------------------------------------------------


class TestBuscarAcordaos:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = AcordaoResultado(
            total_servidor=2,
            total_apos_refiltro=1,
            acordaos=[
                Acordao(
                    numero_acordao="0001/2024",
                    processo="0000001/2024",
                    data_julgamento="01/01/2024",
                    relator="Cons. Fulano de Tal",
                    municipio="ANÁPOLIS",
                    assunto="DENUNCIAS",
                    url_pdf="https://www.tcmgo.tc.br/ecs/arquivo/acordao-0001-2024.pdf",
                )
            ],
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_acordaos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_acordaos(ctx, municipio="ANÁPOLIS")
        assert "0001/2024" in result
        assert "DENUNCIAS" in result
        assert "acordao-0001-2024.pdf" in result
        assert "após refiltro exato 1" in result

    @pytest.mark.asyncio
    async def test_flags_truncated_result(self) -> None:
        mock_data = AcordaoResultado(
            total_servidor=1000,
            total_apos_refiltro=1,
            truncado=True,
            acordaos=[Acordao(numero_acordao="0001/2024")],
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_acordaos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_acordaos(ctx, pesquisa="denuncias")
        assert "capado em 1000" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        mock_data = AcordaoResultado()
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_acordaos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_acordaos(ctx, pesquisa="inexistente")
        assert "Nenhum acórdão encontrado" in result
        assert "não existe" not in result

    @pytest.mark.asyncio
    async def test_http_error_is_reported(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_acordaos",
            new_callable=AsyncMock,
            side_effect=HttpClientError("HTTP 502"),
        ):
            result = await tools.buscar_acordaos(ctx, pesquisa="denuncias")
        assert "Não foi possível consultar" in result
        assert "HTTP 502" in result

    @pytest.mark.asyncio
    async def test_invalid_oplog(self) -> None:
        ctx = _mock_ctx()
        result = await tools.buscar_acordaos(ctx, pesquisa="denuncias", oplog="XOR")
        assert "Operador inválido" in result

    @pytest.mark.asyncio
    async def test_requires_at_least_one_criterion(self) -> None:
        ctx = _mock_ctx()
        result = await tools.buscar_acordaos(ctx)
        assert "ao menos um critério de busca" in result
