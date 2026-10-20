"""Tests for the BrasilAPI HTTP client.

Uses respx to mock HTTP requests and verify response parsing.
"""

import httpx
import pytest
import respx

from mcp_brasil._shared.rate_limiter import RateLimiter
from mcp_brasil.data.brasilapi import client
from mcp_brasil.data.brasilapi.constants import (
    BANKS_URL,
    CAMBIO_COTACAO_URL,
    CEP_URL,
    CNPJ_URL,
    FERIADOS_URL,
)

# ---------------------------------------------------------------------------
# Rate limiter
# ---------------------------------------------------------------------------


class TestRateLimiter:
    def test_rate_limiter_exists(self) -> None:
        assert hasattr(client, "_rate_limiter")
        assert isinstance(client._rate_limiter, RateLimiter)

    def test_rate_limiter_config(self) -> None:
        assert client._rate_limiter._max_requests == 60
        assert client._rate_limiter._period == 60.0


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


# ---------------------------------------------------------------------------
# consultar_cotacao
# ---------------------------------------------------------------------------


def _boletim(tipo: str, hora: str, compra: float, venda: float) -> dict[str, object]:
    return {
        "paridade_compra": 1,
        "paridade_venda": 1,
        "cotacao_compra": compra,
        "cotacao_venda": venda,
        "data_hora_cotacao": f"2026-10-09 {hora}",
        "tipo_boletim": tipo,
    }


# Real BrasilAPI response for USD on 2026-10-09: five bulletins, closing last.
BOLETINS_USD_2026_10_09 = [
    _boletim("ABERTURA", "10:10:14.047225", 5.0012, 5.0018),
    _boletim("INTERMEDIÁRIO", "11:04:12.553856", 4.9906, 4.9912),
    _boletim("INTERMEDIÁRIO", "12:02:23.4373", 4.9855, 4.9861),
    _boletim("INTERMEDIÁRIO", "13:07:30.550247", 4.977, 4.9776),
    _boletim("FECHAMENTO PTAX", "13:07:30.746597", 4.9886, 4.9892),
]


class TestConsultarCotacao:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_closing_ptax_bulletin(self) -> None:
        respx.get(f"{CAMBIO_COTACAO_URL}/USD/2026-10-09").mock(
            return_value=httpx.Response(
                200,
                json={"cotacoes": BOLETINS_USD_2026_10_09, "moeda": "USD", "data": "2026-10-09"},
            )
        )
        result = await client.consultar_cotacao("usd", "2026-10-09")
        assert result.valor_compra == 4.9886
        assert result.valor_venda == 4.9892
        assert result.tipo_boletim == "FECHAMENTO PTAX"
        assert result.data_hora == "2026-10-09 13:07:30.746597"

    @pytest.mark.asyncio
    @respx.mock
    async def test_closing_bulletin_is_found_in_any_position(self) -> None:
        boletins = list(reversed(BOLETINS_USD_2026_10_09))
        respx.get(f"{CAMBIO_COTACAO_URL}/USD/2026-10-09").mock(
            return_value=httpx.Response(200, json={"cotacoes": boletins})
        )
        result = await client.consultar_cotacao("USD", "2026-10-09")
        assert result.valor_venda == 4.9892

    @pytest.mark.asyncio
    @respx.mock
    async def test_without_closing_uses_latest_bulletin(self) -> None:
        # During the day the closing bulletin is not published yet.
        respx.get(f"{CAMBIO_COTACAO_URL}/USD/2026-10-09").mock(
            return_value=httpx.Response(200, json={"cotacoes": BOLETINS_USD_2026_10_09[:3]})
        )
        result = await client.consultar_cotacao("USD", "2026-10-09")
        assert result.valor_venda == 4.9861
        assert result.tipo_boletim == "INTERMEDIÁRIO"

    @pytest.mark.asyncio
    @respx.mock
    async def test_no_bulletins(self) -> None:
        respx.get(f"{CAMBIO_COTACAO_URL}/USD/2026-10-11").mock(
            return_value=httpx.Response(200, json={"cotacoes": []})
        )
        result = await client.consultar_cotacao("USD", "2026-10-11")
        assert result.moeda == "USD"
        assert result.data == "2026-10-11"
        assert result.valor_venda is None
        assert result.tipo_boletim is None
