"""Tests for the Megasoft tool functions."""

from __future__ import annotations

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from mcp_brasil.data.megasoft import client, tools
from mcp_brasil.data.megasoft.schemas import (
    Empenho,
    OrdemPagamento,
    Receita,
    ResultadoEmpenhos,
    ResultadoOrdensPagamento,
    ResultadoReceitas,
    TotaisOrdensPagamento,
)
from mcp_brasil.exceptions import HttpClientError

CLIENT_MODULE = "mcp_brasil.data.megasoft.client"


def _mock_ctx() -> MagicMock:
    ctx = MagicMock()
    ctx.info = AsyncMock()
    ctx.warning = AsyncMock()
    return ctx


# ---------------------------------------------------------------------------
# buscar_empenhos
# ---------------------------------------------------------------------------


class TestBuscarEmpenhos:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = ResultadoEmpenhos(
            empenhos=[
                Empenho(
                    numero_tcm=1001,
                    data="15/03/2024",
                    fornecedor="Empresa Exemplo Ltda",
                    valor_empenho=1000.0,
                    valor_liquidacao=800.0,
                    valor_pagamento=500.0,
                    orgao="1 - Prefeitura Municipal",
                    natureza_despesa="339030 - Material de Consumo",
                )
            ],
            total=1,
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_empenhos", new_callable=AsyncMock, return_value=mock_data
        ):
            result = await tools.buscar_empenhos(ctx, "exemplo", "01/01/2024", "31/12/2024")
        assert "1001" in result
        assert "Empresa Exemplo Ltda" in result
        assert "R$ 1.000,00" in result
        assert "1 - Prefeitura Municipal" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        mock_data = ResultadoEmpenhos(empenhos=[], total=0)
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_empenhos", new_callable=AsyncMock, return_value=mock_data
        ):
            result = await tools.buscar_empenhos(ctx, "exemplo", "01/01/2024", "31/12/2024")
        assert "Nenhum empenho para estes filtros" in result

    @pytest.mark.asyncio
    async def test_http_error_message(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_empenhos",
            new_callable=AsyncMock,
            side_effect=HttpClientError("HTTP 403"),
        ):
            result = await tools.buscar_empenhos(ctx, "exemplo", "01/01/2024", "31/12/2024")
        assert "Não foi possível consultar o portal Megasoft" in result

    @pytest.mark.asyncio
    async def test_unexpected_envelope_message(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_empenhos",
            new_callable=AsyncMock,
            side_effect=client.RespostaInesperadaError("Resposta inesperada de empenho"),
        ):
            result = await tools.buscar_empenhos(ctx, "exemplo", "01/01/2024", "31/12/2024")
        assert "Não foi possível consultar o portal Megasoft" in result
        assert "Resposta inesperada" in result

    @pytest.mark.asyncio
    async def test_invalid_param_message(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_empenhos",
            new_callable=AsyncMock,
            side_effect=ValueError("Data inválida"),
        ):
            result = await tools.buscar_empenhos(ctx, "exemplo", "x", "31/12/2024")
        assert "Parâmetro inválido" in result


# ---------------------------------------------------------------------------
# buscar_ordens_pagamento
# ---------------------------------------------------------------------------


class TestBuscarOrdensPagamento:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = ResultadoOrdensPagamento(
            ordens=[
                OrdemPagamento(
                    numero=2024007,
                    data_pagamento="20/03/2024",
                    fornecedor="Empresa Exemplo Ltda",
                    cpf_cnpj_fornecedor="12.345.678/0001-90",
                    valor_pago=500.0,
                    valor_total=500.0,
                    orgao="2 - Secretaria de Saúde",
                    fase="Pagamento",
                )
            ],
            total=1,
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_ordens_pagamento",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_ordens_pagamento(
                ctx, "exemplo", "01/01/2024", "31/12/2024"
            )
        assert "2024007" in result
        assert "12.345.678/0001-90" in result
        assert "R$ 500,00" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        mock_data = ResultadoOrdensPagamento(ordens=[], total=0)
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_ordens_pagamento",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_ordens_pagamento(
                ctx, "exemplo", "01/01/2024", "31/12/2024"
            )
        assert "Nenhuma ordem de pagamento para estes filtros" in result


# ---------------------------------------------------------------------------
# totais_ordens_pagamento
# ---------------------------------------------------------------------------


class TestTotaisOrdensPagamento:
    @pytest.mark.asyncio
    async def test_formats_totais(self) -> None:
        mock_data = TotaisOrdensPagamento(
            valor_pago=100000.0,
            valor_anulacao=2000.0,
            valor_total=98000.0,
            valor_retencao=5000.0,
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.totais_ordens_pagamento",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.totais_ordens_pagamento(
                ctx, "exemplo", "01/01/2024", "31/12/2024"
            )
        assert "R$ 100.000,00" in result
        assert "R$ 98.000,00" in result
        assert "R$ 5.000,00" in result


# ---------------------------------------------------------------------------
# buscar_receitas
# ---------------------------------------------------------------------------


class TestBuscarReceitas:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = ResultadoReceitas(
            receitas=[
                Receita(
                    codigo_receita=5478,
                    ano=2024,
                    mes="12",
                    receita="IPTU - Principal",
                    natureza="1112.50.0.1",
                    origem_recurso="Rendas Locais do Municipio",
                    valor_orcado=200000.0,
                    arrecadacao=150000.0,
                    valor_acumulado=150000.0,
                )
            ],
            total=1,
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_receitas", new_callable=AsyncMock, return_value=mock_data
        ):
            result = await tools.buscar_receitas(ctx, "exemplo", "01/01/2024", "31/12/2024")
        assert "IPTU - Principal" in result
        assert "R$ 200.000,00" in result
        assert "R$ 150.000,00" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        mock_data = ResultadoReceitas(receitas=[], total=0)
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_receitas", new_callable=AsyncMock, return_value=mock_data
        ):
            result = await tools.buscar_receitas(ctx, "exemplo", "01/01/2024", "31/12/2024")
        assert "Nenhuma receita para estes filtros" in result
