"""Tests for the Saúde HTTP client."""

import httpx
import pytest
import respx

from mcp_brasil.data.saude import client
from mcp_brasil.data.saude.constants import (
    ESTABELECIMENTOS_URL,
    LEITOS_URL,
    TIPOS_URL,
)
from mcp_brasil.exceptions import HttpClientError

# ---------------------------------------------------------------------------
# buscar_estabelecimentos
# ---------------------------------------------------------------------------


class TestBuscarEstabelecimentos:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_establishments(self) -> None:
        respx.get(ESTABELECIMENTOS_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "codigo_cnes": "1234567",
                        "nome_fantasia": "UBS Central",
                        "nome_razao_social": "Unidade Básica de Saúde Central",
                        "natureza_organizacao_entidade": "Administração Pública",
                        "tipo_gestao": "Municipal",
                        "codigo_tipo_unidade": "01",
                        "descricao_turno_atendimento": "Central de Regulação",
                        "codigo_municipio": "355030",
                        "codigo_uf": "35",
                        "endereco_estabelecimento": "Rua ABC, 123",
                    }
                ],
            )
        )
        result = await client.buscar_estabelecimentos(codigo_municipio="355030")
        assert len(result) == 1
        assert result[0].codigo_cnes == "1234567"
        assert result[0].nome_fantasia == "UBS Central"
        assert result[0].codigo_municipio == "355030"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(ESTABELECIMENTOS_URL).mock(return_value=httpx.Response(200, json=[]))
        result = await client.buscar_estabelecimentos()
        assert result == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_sends_query_params(self) -> None:
        route = respx.get(ESTABELECIMENTOS_URL).mock(return_value=httpx.Response(200, json=[]))
        await client.buscar_estabelecimentos(
            codigo_municipio="355030", codigo_uf="35", status=1, limit=10, offset=5
        )
        req_url = str(route.calls[0].request.url)
        assert "codigo_municipio=355030" in req_url
        assert "codigo_uf=35" in req_url
        assert "status=1" in req_url
        assert "limit=10" in req_url
        assert "offset=5" in req_url

    @pytest.mark.asyncio
    @respx.mock
    async def test_limit_capped_at_max(self) -> None:
        route = respx.get(ESTABELECIMENTOS_URL).mock(return_value=httpx.Response(200, json=[]))
        await client.buscar_estabelecimentos(limit=999)
        req_url = str(route.calls[0].request.url)
        assert "limit=20" in req_url


# ---------------------------------------------------------------------------
# listar_tipos_estabelecimento
# ---------------------------------------------------------------------------


class TestListarTiposEstabelecimento:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_types(self) -> None:
        respx.get(TIPOS_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "codigo_tipo_unidade": "01",
                        "descricao_tipo_unidade": "Central de Regulação",
                    },
                    {
                        "codigo_tipo_unidade": "02",
                        "descricao_tipo_unidade": "Hospital Geral",
                    },
                ],
            )
        )
        result = await client.listar_tipos_estabelecimento()
        assert len(result) == 2
        assert result[0].codigo == "01"
        assert result[0].descricao == "Central de Regulação"
        assert result[1].codigo == "02"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(TIPOS_URL).mock(return_value=httpx.Response(200, json=[]))
        result = await client.listar_tipos_estabelecimento()
        assert result == []


# ---------------------------------------------------------------------------
# consultar_leitos
# ---------------------------------------------------------------------------


class TestConsultarLeitos:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_beds(self) -> None:
        respx.get(LEITOS_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "nome_do_hospital": "Hospital Central",
                        "descricao_do_tipo_da_unidade": "Cirúrgico",
                        "descricao_da_natureza_juridica_do_hosptial": "Cirurgia Geral",
                        "quantidade_total_de_leitos_do_hosptial": 20,
                        "quantidade_total_de_leitos_sus_do_hosptial": 15,
                    }
                ],
            )
        )
        result = await client.consultar_leitos()
        assert len(result) == 1
        assert result[0].tipo_leito == "Cirúrgico"
        assert result[0].existente == 20
        assert result[0].sus == 15

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(LEITOS_URL).mock(return_value=httpx.Response(200, json=[]))
        result = await client.consultar_leitos()
        assert result == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_sends_query_params(self) -> None:
        route = respx.get(LEITOS_URL).mock(return_value=httpx.Response(200, json=[]))
        await client.consultar_leitos(limit=50, offset=10)
        req_url = str(route.calls[0].request.url)
        assert "limit=50" in req_url
        assert "offset=10" in req_url


# ---------------------------------------------------------------------------
# Parse functions
# ---------------------------------------------------------------------------


class TestParseEstabelecimento:
    def test_handles_missing_fields(self) -> None:
        result = client._parse_estabelecimento({})
        assert result.codigo_cnes == ""
        assert result.nome_fantasia is None

    def test_converts_numeric_codes_to_str(self) -> None:
        result = client._parse_estabelecimento(
            {"codigo_cnes": 1234567, "codigo_municipio": 355030, "codigo_uf": 35}
        )
        assert result.codigo_cnes == "1234567"
        assert result.codigo_municipio == "355030"
        assert result.codigo_uf == "35"


class TestParseTipo:
    def test_handles_missing_fields(self) -> None:
        result = client._parse_tipo({})
        assert result.codigo == ""
        assert result.descricao is None


class TestParseLeito:
    def test_handles_missing_fields(self) -> None:
        result = client._parse_leito({})
        assert result.codigo_cnes == ""
        assert result.existente is None
        assert result.sus is None


# ---------------------------------------------------------------------------
# buscar_estabelecimento_por_cnes
# ---------------------------------------------------------------------------


class TestBuscarEstabelecimentoPorCnes:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_detail(self) -> None:
        respx.get(f"{ESTABELECIMENTOS_URL}/1234567").mock(
            return_value=httpx.Response(
                200,
                json={
                    "codigo_cnes": "1234567",
                    "nome_fantasia": "Hospital São Paulo",
                    "nome_razao_social": "Hospital São Paulo Ltda",
                    "natureza_organizacao_entidade": "Administração Pública",
                    "tipo_gestao": "Estadual",
                    "codigo_tipo_unidade": "05",
                    "descricao_turno_atendimento": "Hospital Geral",
                    "codigo_municipio": "355030",
                    "codigo_uf": "35",
                    "endereco_estabelecimento": "Rua Napoleão de Barros, 715",
                    "bairro_estabelecimento": "Vila Clementino",
                    "codigo_cep_estabelecimento": "04024-002",
                    "numero_telefone_estabelecimento": "(11) 5576-4000",
                    "latitude_estabelecimento_decimo_grau": -23.5989,
                    "longitude_estabelecimento_decimo_grau": -46.6423,
                    "numero_cnpj": "12.345.678/0001-90",
                    "data_atualizacao": "2024-01-15",
                },
            )
        )
        result = await client.buscar_estabelecimento_por_cnes("1234567")
        assert result is not None
        assert result.codigo_cnes == "1234567"
        assert result.nome_fantasia == "Hospital São Paulo"
        assert result.telefone == "(11) 5576-4000"
        assert result.latitude == -23.5989

    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_none_for_empty(self) -> None:
        respx.get(f"{ESTABELECIMENTOS_URL}/0000000").mock(
            return_value=httpx.Response(200, json={})
        )
        result = await client.buscar_estabelecimento_por_cnes("0000000")
        assert result is None


# ---------------------------------------------------------------------------
# buscar_estabelecimentos_por_tipo
# ---------------------------------------------------------------------------


class TestBuscarEstabelecimentosPorTipo:
    @pytest.mark.asyncio
    @respx.mock
    async def test_sends_tipo_param(self) -> None:
        route = respx.get(ESTABELECIMENTOS_URL).mock(return_value=httpx.Response(200, json=[]))
        await client.buscar_estabelecimentos_por_tipo(
            codigo_tipo="73",
            codigo_municipio="355030",
        )
        req_url = str(route.calls[0].request.url)
        assert "codigo_tipo_unidade=73" in req_url
        assert "codigo_municipio=355030" in req_url
        assert "status=1" in req_url

    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_list(self) -> None:
        respx.get(ESTABELECIMENTOS_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "codigo_cnes": "9876543",
                        "nome_fantasia": "UPA 24h",
                        "codigo_tipo_unidade": "73",
                        "descricao_turno_atendimento": "Pronto Atendimento",
                        "codigo_municipio": "220040",
                        "codigo_uf": "22",
                    }
                ],
            )
        )
        result = await client.buscar_estabelecimentos_por_tipo(codigo_tipo="73")
        assert len(result) == 1
        assert result[0].codigo_cnes == "9876543"
        assert result[0].descricao_tipo == "Pronto Atendimento"


# ---------------------------------------------------------------------------
# Parse detail function
# ---------------------------------------------------------------------------


class TestParseEstabelecimentoDetalhe:
    def test_handles_missing_fields(self) -> None:
        result = client._parse_estabelecimento_detalhe({})
        assert result.codigo_cnes == ""
        assert result.telefone is None
        assert result.latitude is None

    def test_parses_all_fields(self) -> None:
        result = client._parse_estabelecimento_detalhe(
            {
                "codigo_cnes": 1234567,
                "nome_fantasia": "Hospital X",
                "bairro_estabelecimento": "Centro",
                "codigo_cep_estabelecimento": "01000-000",
                "numero_telefone_estabelecimento": "1199999999",
                "latitude_estabelecimento_decimo_grau": -23.55,
                "longitude_estabelecimento_decimo_grau": -46.63,
                "numero_cnpj": "12345678000190",
                "data_atualizacao": "2024-06-01",
            }
        )
        assert result.codigo_cnes == "1234567"
        assert result.bairro == "Centro"
        assert result.latitude == -23.55


# ---------------------------------------------------------------------------
# Malformed API responses (type validation)
# ---------------------------------------------------------------------------


class TestMalformedResponses:
    """Test that client functions raise HttpClientError on unexpected response types."""

    @pytest.mark.asyncio
    @respx.mock
    async def test_estabelecimentos_string_response(self) -> None:
        respx.get(ESTABELECIMENTOS_URL).mock(
            return_value=httpx.Response(200, json="Service Unavailable")
        )
        with pytest.raises(HttpClientError, match="expected list"):
            await client.buscar_estabelecimentos()

    @pytest.mark.asyncio
    @respx.mock
    async def test_estabelecimentos_dict_without_list_returns_empty(self) -> None:
        """Dict response without list-valued keys returns empty list (API wraps in dict)."""
        respx.get(ESTABELECIMENTOS_URL).mock(
            return_value=httpx.Response(200, json={"error": "not found"})
        )
        result = await client.buscar_estabelecimentos()
        assert result == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_leitos_string_response(self) -> None:
        respx.get(LEITOS_URL).mock(return_value=httpx.Response(200, json="404 Not Found"))
        with pytest.raises(HttpClientError, match="expected list"):
            await client.consultar_leitos()

    @pytest.mark.asyncio
    @respx.mock
    async def test_tipos_string_response(self) -> None:
        respx.get(TIPOS_URL).mock(return_value=httpx.Response(200, json="error"))
        with pytest.raises(HttpClientError, match="expected list"):
            await client.listar_tipos_estabelecimento()

    @pytest.mark.asyncio
    @respx.mock
    async def test_estabelecimento_por_cnes_string_response(self) -> None:
        respx.get(f"{ESTABELECIMENTOS_URL}/1234567").mock(
            return_value=httpx.Response(200, json="not found")
        )
        with pytest.raises(HttpClientError, match="expected dict"):
            await client.buscar_estabelecimento_por_cnes("1234567")
