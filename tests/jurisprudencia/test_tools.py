"""Tests for the Jurisprudência tool functions.

Tools are tested by mocking client functions (never HTTP).
"""

from unittest.mock import AsyncMock, patch

import pytest

from mcp_brasil.jurisprudencia import tools
from mcp_brasil.jurisprudencia.schemas import Jurisprudencia, RepercussaoGeral, Sumula

MODULE = "mcp_brasil.jurisprudencia.client"


class TestBuscarJurisprudenciaSTF:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = [
            Jurisprudencia(
                tribunal="STF",
                ementa="Direito à privacidade. Constitucional.",
                relator="Min. Fulano",
                numero_processo="RE 123456",
                classe="RE",
                data_julgamento="2024-03-15",
            )
        ]
        with patch(f"{MODULE}.buscar_stf", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.buscar_jurisprudencia_stf("privacidade")
        assert "privacidade" in result.lower()
        assert "Min. Fulano" in result
        assert "RE 123456" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.buscar_stf", new_callable=AsyncMock, return_value=[]):
            result = await tools.buscar_jurisprudencia_stf("inexistente")
        assert "Nenhum acórdão" in result


class TestBuscarJurisprudenciaSTJ:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = [
            Jurisprudencia(
                tribunal="STJ",
                ementa="Consumidor. Dano moral. Indenização.",
                relator="Min. Cicrano",
                numero_processo="REsp 789012",
                classe="REsp",
                data_julgamento="2024-06-20",
            )
        ]
        with patch(f"{MODULE}.buscar_stj", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.buscar_jurisprudencia_stj("consumidor dano moral")
        assert "Min. Cicrano" in result
        assert "REsp 789012" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.buscar_stj", new_callable=AsyncMock, return_value=[]):
            result = await tools.buscar_jurisprudencia_stj("xyzabc")
        assert "Nenhum acórdão" in result


class TestBuscarJurisprudenciaTST:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = [
            Jurisprudencia(
                tribunal="TST",
                ementa="Horas extras. Intervalo intrajornada.",
                relator="Min. Beltrano",
                numero_processo="RR 456789",
                classe="RR",
                data_julgamento="2024-09-10",
            )
        ]
        with patch(f"{MODULE}.buscar_tst", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.buscar_jurisprudencia_tst("horas extras")
        assert "Min. Beltrano" in result
        assert "RR 456789" in result


class TestBuscarSumulas:
    @pytest.mark.asyncio
    async def test_formats_sumulas(self) -> None:
        mock_data = [
            Sumula(
                tribunal="STF",
                numero=11,
                enunciado="Não cabe mandado de segurança contra ato de particular.",
                vinculante=True,
            )
        ]
        with patch(f"{MODULE}.buscar_sumulas_stf", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.buscar_sumulas("stf", "mandado de segurança")
        assert "Súmula 11" in result
        assert "VINCULANTE" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.buscar_sumulas_stf", new_callable=AsyncMock, return_value=[]):
            result = await tools.buscar_sumulas("stf", "inexistente")
        assert "Nenhuma súmula" in result

    @pytest.mark.asyncio
    async def test_unsupported_tribunal(self) -> None:
        result = await tools.buscar_sumulas("trt1")
        assert "disponível apenas" in result


class TestBuscarRepercussaoGeral:
    @pytest.mark.asyncio
    async def test_formats_themes(self) -> None:
        mock_data = [
            RepercussaoGeral(
                numero_tema=793,
                titulo="ICMS na base de cálculo do PIS/COFINS",
                relator="Min. Presidente",
                situacao="Julgado",
                tese="O ICMS não compõe a base de cálculo...",
            )
        ]
        with patch(
            f"{MODULE}.buscar_repercussao_geral",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_repercussao_geral(query="ICMS PIS COFINS")
        assert "Tema 793" in result
        assert "ICMS" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(
            f"{MODULE}.buscar_repercussao_geral",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.buscar_repercussao_geral(query="inexistente")
        assert "Nenhum tema" in result


class TestBuscarInformativos:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = [
            Jurisprudencia(
                tribunal="STF",
                ementa="Informativo 1100. Direito Constitucional.",
                relator="Min. Teste",
                numero_processo="ADI 1234",
                classe="ADI",
                data_julgamento="2024-10-01",
            )
        ]
        with patch(f"{MODULE}.buscar_stf", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.buscar_informativos("stf", "direito constitucional")
        assert "Informativos STF" in result
        assert "ADI 1234" in result

    @pytest.mark.asyncio
    async def test_unsupported_tribunal(self) -> None:
        result = await tools.buscar_informativos("trf1")
        assert "não suportado" in result
