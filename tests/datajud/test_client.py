"""Tests for the DataJud HTTP client."""

import httpx
import pytest
import respx

from mcp_brasil.datajud import client
from mcp_brasil.datajud.constants import DATAJUD_API_BASE

TJSP_URL = f"{DATAJUD_API_BASE}tjsp/_search"


# ---------------------------------------------------------------------------
# buscar_processos
# ---------------------------------------------------------------------------


class TestBuscarProcessos:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_processes(self) -> None:
        respx.post(TJSP_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "hits": {
                        "hits": [
                            {
                                "_source": {
                                    "numeroProcesso": "0001234",
                                    "classe": {"nome": "Procedimento Comum Cível"},
                                    "assuntos": [{"nome": "Dano Moral"}],
                                    "tribunal": "TJSP",
                                    "orgaoJulgador": {"nome": "1ª Vara Cível"},
                                    "dataAjuizamento": "2024-03-15",
                                }
                            }
                        ]
                    }
                },
            )
        )
        result = await client.buscar_processos("dano moral", "tjsp")
        assert len(result) == 1
        assert result[0].numero == "0001234"
        assert result[0].classe == "Procedimento Comum Cível"
        assert result[0].assunto == "Dano Moral"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_hits(self) -> None:
        respx.post(TJSP_URL).mock(return_value=httpx.Response(200, json={"hits": {"hits": []}}))
        result = await client.buscar_processos("inexistente", "tjsp")
        assert result == []


# ---------------------------------------------------------------------------
# buscar_processo_por_numero
# ---------------------------------------------------------------------------


class TestBuscarProcessoPorNumero:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_detailed_process(self) -> None:
        respx.post(TJSP_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "hits": {
                        "hits": [
                            {
                                "_source": {
                                    "numeroProcesso": "0001234",
                                    "classe": {"nome": "Ação Civil Pública"},
                                    "assuntos": [{"codigo": 1116, "nome": "Meio Ambiente"}],
                                    "tribunal": "TJSP",
                                    "orgaoJulgador": {"nome": "2ª Vara"},
                                    "dataAjuizamento": "2024-01-10",
                                    "poloAtivo": [
                                        {"nome": "Ministério Público", "tipoPessoa": "JURIDICA"}
                                    ],
                                    "poloPassivo": [
                                        {"nome": "Empresa XYZ", "tipoPessoa": "JURIDICA"}
                                    ],
                                    "movimentos": [
                                        {
                                            "dataHora": "2024-01-10",
                                            "nome": "Distribuição",
                                            "codigo": 26,
                                        }
                                    ],
                                }
                            }
                        ]
                    }
                },
            )
        )
        result = await client.buscar_processo_por_numero("0001234", "tjsp")
        assert result is not None
        assert result.numero == "0001234"
        assert result.classe == "Ação Civil Pública"
        assert result.assuntos is not None
        assert len(result.assuntos) == 1
        assert result.partes is not None
        assert len(result.partes) == 2
        assert result.movimentacoes is not None
        assert len(result.movimentacoes) == 1

    @pytest.mark.asyncio
    @respx.mock
    async def test_not_found(self) -> None:
        respx.post(TJSP_URL).mock(return_value=httpx.Response(200, json={"hits": {"hits": []}}))
        result = await client.buscar_processo_por_numero("9999999", "tjsp")
        assert result is None


# ---------------------------------------------------------------------------
# Parser edge cases
# ---------------------------------------------------------------------------


class TestParserEdgeCases:
    def test_parse_processo_empty_source(self) -> None:
        result = client._parse_processo({"_source": {}})
        assert result.numero is None
        assert result.assunto == ""

    def test_parse_processo_classe_as_string(self) -> None:
        result = client._parse_processo({"_source": {"classe": "Ação Penal"}})
        assert result.classe == "Ação Penal"

    def test_parse_processo_no_assuntos(self) -> None:
        result = client._parse_processo({"_source": {"assuntos": []}})
        assert result.assunto == ""

    def test_parse_processo_detalhe_empty(self) -> None:
        result = client._parse_processo_detalhe({"_source": {}})
        assert result.numero is None
        assert result.assuntos == []
        assert result.partes == []
        assert result.movimentacoes == []


# ---------------------------------------------------------------------------
# Tribunal validation
# ---------------------------------------------------------------------------


class TestTribunalValidation:
    def test_valid_tribunal(self) -> None:
        url = client._tribunal_url("tjsp")
        assert "tjsp" in url

    def test_invalid_tribunal(self) -> None:
        with pytest.raises(ValueError, match="não suportado"):
            client._tribunal_url("tribunal_invalido")

    def test_case_insensitive(self) -> None:
        url = client._tribunal_url("TJSP")
        assert "tjsp" in url
