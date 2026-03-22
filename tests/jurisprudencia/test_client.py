"""Tests for the Jurisprudência HTTP client."""

import httpx
import pytest
import respx

from mcp_brasil.jurisprudencia import client
from mcp_brasil.jurisprudencia.constants import STF_API_BASE, STJ_API_BASE, TST_API_BASE


class TestBuscarSTF:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_results(self) -> None:
        respx.get(STF_API_BASE).mock(
            return_value=httpx.Response(
                200,
                json={
                    "result": [
                        {
                            "ementa": "Direito à privacidade. Art. 5º, X, CF.",
                            "ministro": "Min. Fulano",
                            "incidente": "RE 123456",
                            "classe": "RE",
                            "dataJulgamento": "15/03/2024",
                            "orgaoJulgador": "Plenário",
                        }
                    ]
                },
            )
        )
        result = await client.buscar_stf("privacidade")
        assert len(result) == 1
        assert result[0].tribunal == "STF"
        assert result[0].ementa is not None
        assert "privacidade" in result[0].ementa

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty(self) -> None:
        respx.get(STF_API_BASE).mock(return_value=httpx.Response(200, json={"result": []}))
        result = await client.buscar_stf("inexistente")
        assert result == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_handles_error(self) -> None:
        respx.get(STF_API_BASE).mock(
            return_value=httpx.Response(500, text="Internal Server Error")
        )
        result = await client.buscar_stf("erro")
        assert result == []


class TestBuscarSTJ:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_results(self) -> None:
        respx.get(STJ_API_BASE).mock(
            return_value=httpx.Response(
                200,
                json={
                    "documentos": [
                        {
                            "ementa": "Consumidor. Dano moral. Banco.",
                            "relator": "Min. Cicrano",
                            "processo": "REsp 789012",
                            "siglaClasse": "REsp",
                            "dtJulgamento": "20/06/2024",
                        }
                    ]
                },
            )
        )
        result = await client.buscar_stj("consumidor dano moral")
        assert len(result) == 1
        assert result[0].tribunal == "STJ"
        assert "Consumidor" in (result[0].ementa or "")

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty(self) -> None:
        respx.get(STJ_API_BASE).mock(return_value=httpx.Response(200, json={"documentos": []}))
        result = await client.buscar_stj("inexistente")
        assert result == []


class TestBuscarTST:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_results(self) -> None:
        respx.get(TST_API_BASE).mock(
            return_value=httpx.Response(
                200,
                json={
                    "documentos": [
                        {
                            "ementa": "Horas extras. Intervalo intrajornada.",
                            "relator": "Min. Beltrano",
                            "processo": "RR 456789",
                            "siglaClasse": "RR",
                            "dtJulgamento": "10/09/2024",
                        }
                    ]
                },
            )
        )
        result = await client.buscar_tst("horas extras")
        assert len(result) == 1
        assert result[0].tribunal == "TST"
        assert "Horas extras" in (result[0].ementa or "")


class TestBuscarSumulasSTF:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_sumulas(self) -> None:
        respx.get(STF_API_BASE).mock(
            return_value=httpx.Response(
                200,
                json={
                    "result": [
                        {
                            "numero": "11",
                            "enunciado": "Não cabe mandado de segurança contra ato particular.",
                            "situacao": "Vigente",
                            "vinculante": True,
                        }
                    ]
                },
            )
        )
        result = await client.buscar_sumulas_stf("mandado de segurança")
        assert len(result) == 1
        assert result[0].numero == 11
        assert result[0].vinculante is True


class TestBuscarRepercussaoGeral:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_themes(self) -> None:
        respx.get(STF_API_BASE).mock(
            return_value=httpx.Response(
                200,
                json={
                    "result": [
                        {
                            "numeroTema": "793",
                            "titulo": "ICMS na base de cálculo do PIS/COFINS",
                            "relator": "Min. Presidente",
                            "situacao": "Julgado",
                            "tese": "O ICMS não compõe a base de cálculo...",
                        }
                    ]
                },
            )
        )
        result = await client.buscar_repercussao_geral(query="ICMS")
        assert len(result) == 1
        assert result[0].numero_tema == 793
        assert "ICMS" in (result[0].titulo or "")


class TestParserEdgeCases:
    def test_parse_stf_resultado_empty(self) -> None:
        result = client._parse_stf_resultado({})
        assert result.tribunal == "STF"
        assert result.ementa is None

    def test_parse_stj_resultado_empty(self) -> None:
        result = client._parse_stj_resultado({})
        assert result.tribunal == "STJ"
        assert result.ementa is None

    def test_parse_tst_resultado_empty(self) -> None:
        result = client._parse_tst_resultado({})
        assert result.tribunal == "TST"
        assert result.ementa is None

    def test_parse_sumula_numero_as_string(self) -> None:
        result = client._parse_sumula({"numero": "42"}, "STF")
        assert result.numero == 42

    def test_parse_sumula_numero_invalid(self) -> None:
        result = client._parse_sumula({"numero": "abc"}, "STF")
        assert result.numero is None

    def test_parse_repercussao_empty(self) -> None:
        result = client._parse_repercussao({})
        assert result.numero_tema is None
        assert result.titulo is None
