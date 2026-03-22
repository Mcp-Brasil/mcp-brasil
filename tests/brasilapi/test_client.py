"""Tests for the BrasilAPI HTTP client.

Uses respx to mock HTTP requests and verify response parsing.
"""

import httpx
import pytest
import respx

from mcp_brasil.brasilapi import client
from mcp_brasil.brasilapi.constants import (
    BANKS_URL,
    CEP_URL,
    CNPJ_URL,
    FERIADOS_URL,
)

# ---------------------------------------------------------------------------
# consultar_cep
# ---------------------------------------------------------------------------


class TestConsultarCep:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_address(self) -> None:
        respx.get(f"{CEP_URL}/01001000").mock(
            return_value=httpx.Response(
                200,
                json={
                    "cep": "01001000",
                    "state": "SP",
                    "city": "São Paulo",
                    "neighborhood": "Sé",
                    "street": "Praça da Sé",
                    "service": "viacep",
                },
            )
        )
        result = await client.consultar_cep("01001000")
        assert result.cep == "01001000"
        assert result.state == "SP"
        assert result.city == "São Paulo"
        assert result.neighborhood == "Sé"
        assert result.street == "Praça da Sé"

    @pytest.mark.asyncio
    @respx.mock
    async def test_strips_hyphen(self) -> None:
        route = respx.get(f"{CEP_URL}/01001000").mock(
            return_value=httpx.Response(
                200,
                json={
                    "cep": "01001000",
                    "state": "SP",
                    "city": "São Paulo",
                },
            )
        )
        await client.consultar_cep("01001-000")
        assert route.called

    @pytest.mark.asyncio
    @respx.mock
    async def test_optional_fields_none(self) -> None:
        respx.get(f"{CEP_URL}/69900000").mock(
            return_value=httpx.Response(
                200,
                json={
                    "cep": "69900000",
                    "state": "AC",
                    "city": "Rio Branco",
                },
            )
        )
        result = await client.consultar_cep("69900000")
        assert result.neighborhood is None
        assert result.street is None


# ---------------------------------------------------------------------------
# consultar_cnpj
# ---------------------------------------------------------------------------


class TestConsultarCnpj:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_company(self) -> None:
        respx.get(f"{CNPJ_URL}/00000000000191").mock(
            return_value=httpx.Response(
                200,
                json={
                    "cnpj": "00000000000191",
                    "razao_social": "BANCO DO BRASIL SA",
                    "nome_fantasia": "BANCO DO BRASIL",
                    "descricao_situacao_cadastral": "ATIVA",
                    "porte": "DEMAIS",
                    "natureza_juridica": "Soc. Economia Mista",
                    "cnae_fiscal": 6422100,
                    "cnae_fiscal_descricao": "Bancos múltiplos",
                    "logradouro": "SAUN Q 5 L B",
                    "numero": "S/N",
                    "bairro": "ASA NORTE",
                    "cep": "70040912",
                    "uf": "DF",
                    "municipio": "BRASILIA",
                    "capital_social": 120000000000.0,
                },
            )
        )
        result = await client.consultar_cnpj("00000000000191")
        assert result.cnpj == "00000000000191"
        assert result.razao_social == "BANCO DO BRASIL SA"
        assert result.capital_social == 120000000000.0
        assert result.uf == "DF"

    @pytest.mark.asyncio
    @respx.mock
    async def test_strips_formatting(self) -> None:
        route = respx.get(f"{CNPJ_URL}/00000000000191").mock(
            return_value=httpx.Response(
                200,
                json={"cnpj": "00000000000191"},
            )
        )
        await client.consultar_cnpj("00.000.000/0001-91")
        assert route.called


# ---------------------------------------------------------------------------
# listar_bancos
# ---------------------------------------------------------------------------


class TestListarBancos:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_banks(self) -> None:
        respx.get(BANKS_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "ispb": "00000000",
                        "name": "BCO DO BRASIL S.A.",
                        "code": 1,
                        "fullName": "Banco do Brasil S.A.",
                    },
                    {
                        "ispb": "60701190",
                        "name": "ITAÚ UNIBANCO S.A.",
                        "code": 341,
                        "fullName": "Itaú Unibanco S.A.",
                    },
                ],
            )
        )
        result = await client.listar_bancos()
        assert len(result) == 2
        assert result[0].code == 1
        assert result[0].name == "BCO DO BRASIL S.A."
        assert result[1].code == 341

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(BANKS_URL).mock(return_value=httpx.Response(200, json=[]))
        result = await client.listar_bancos()
        assert result == []


# ---------------------------------------------------------------------------
# consultar_feriados
# ---------------------------------------------------------------------------


class TestConsultarFeriados:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_holidays(self) -> None:
        respx.get(f"{FERIADOS_URL}/2024").mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "date": "2024-01-01",
                        "name": "Confraternização Universal",
                        "type": "national",
                    },
                    {
                        "date": "2024-04-21",
                        "name": "Tiradentes",
                        "type": "national",
                    },
                    {
                        "date": "2024-12-25",
                        "name": "Natal",
                        "type": "national",
                    },
                ],
            )
        )
        result = await client.consultar_feriados(2024)
        assert len(result) == 3
        assert result[0].name == "Confraternização Universal"
        assert result[1].date == "2024-04-21"
        assert result[2].type == "national"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(f"{FERIADOS_URL}/1800").mock(return_value=httpx.Response(200, json=[]))
        result = await client.consultar_feriados(1800)
        assert result == []
