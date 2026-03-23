"""Tests for the TCE-SP HTTP client."""

import httpx
import pytest
import respx

from mcp_brasil.data.tce_sp import client
from mcp_brasil.data.tce_sp.constants import DESPESAS_URL, MUNICIPIOS_URL, RECEITAS_URL

# ---------------------------------------------------------------------------
# _parse_brl_string
# ---------------------------------------------------------------------------


class TestParseBrlString:
    def test_simple_value(self) -> None:
        assert client._parse_brl_string("5034,11") == 5034.11

    def test_thousands_separator(self) -> None:
        assert client._parse_brl_string("924.000,00") == 924000.0

    def test_negative_value(self) -> None:
        assert client._parse_brl_string("-266.629,43") == -266629.43

    def test_none_returns_none(self) -> None:
        assert client._parse_brl_string(None) is None

    def test_empty_string_returns_none(self) -> None:
        assert client._parse_brl_string("") is None

    def test_small_value(self) -> None:
        assert client._parse_brl_string("3,11") == 3.11


# ---------------------------------------------------------------------------
# listar_municipios
# ---------------------------------------------------------------------------


class TestListarMunicipios:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_municipios(self) -> None:
        respx.get(MUNICIPIOS_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {"municipio": "campinas", "municipio_extenso": "Campinas"},
                    {"municipio": "sao-paulo", "municipio_extenso": "São Paulo"},
                ],
            )
        )
        result = await client.listar_municipios()
        assert len(result) == 2
        assert result[0].municipio == "campinas"
        assert result[1].municipio_extenso == "São Paulo"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(MUNICIPIOS_URL).mock(return_value=httpx.Response(200, json=[]))
        result = await client.listar_municipios()
        assert result == []


# ---------------------------------------------------------------------------
# buscar_despesas
# ---------------------------------------------------------------------------


class TestBuscarDespesas:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_despesas(self) -> None:
        respx.get(f"{DESPESAS_URL}/balsamo/2025/1").mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "orgao": "PREFEITURA MUNICIPAL DE BÁLSAMO",
                        "mes": "Janeiro",
                        "evento": "Empenhado",
                        "nr_empenho": "110-2025",
                        "id_fornecedor": "PESSOA JURÍDICA - 12345678000199",
                        "nm_fornecedor": "EMPRESA X LTDA",
                        "dt_emissao_despesa": "02/01/2025",
                        "vl_despesa": "5.034,11",
                    }
                ],
            )
        )
        result = await client.buscar_despesas("balsamo", 2025, 1)
        assert len(result) == 1
        assert result[0].evento == "Empenhado"
        assert result[0].vl_despesa == 5034.11
        assert result[0].nm_fornecedor == "EMPRESA X LTDA"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(f"{DESPESAS_URL}/inexistente/2025/1").mock(
            return_value=httpx.Response(200, json=[])
        )
        result = await client.buscar_despesas("inexistente", 2025, 1)
        assert result == []


# ---------------------------------------------------------------------------
# buscar_receitas
# ---------------------------------------------------------------------------


class TestBuscarReceitas:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_receitas(self) -> None:
        respx.get(f"{RECEITAS_URL}/balsamo/2025/1").mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "orgao": "PREFEITURA MUNICIPAL DE BÁLSAMO",
                        "mes": "Janeiro",
                        "ds_fonte_recurso": "01 - TESOURO",
                        "ds_cd_aplicacao_fixo": "110 - GERAL",
                        "ds_alinea": "13210000 - Juros e Correções Monetárias",
                        "ds_subalinea": "13210011 - Remuneração de Depósitos Bancários",
                        "vl_arrecadacao": "2.314,94",
                    }
                ],
            )
        )
        result = await client.buscar_receitas("balsamo", 2025, 1)
        assert len(result) == 1
        assert result[0].vl_arrecadacao == 2314.94
        assert result[0].ds_fonte_recurso == "01 - TESOURO"

    @pytest.mark.asyncio
    @respx.mock
    async def test_negative_values(self) -> None:
        respx.get(f"{RECEITAS_URL}/balsamo/2025/1").mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "orgao": "PREFEITURA",
                        "mes": "Janeiro",
                        "ds_fonte_recurso": "05 - TRANSFERÊNCIAS",
                        "ds_cd_aplicacao_fixo": "110",
                        "ds_alinea": "17215100 - IPVA",
                        "ds_subalinea": "",
                        "vl_arrecadacao": "-266.629,43",
                    }
                ],
            )
        )
        result = await client.buscar_receitas("balsamo", 2025, 1)
        assert len(result) == 1
        assert result[0].vl_arrecadacao == -266629.43
