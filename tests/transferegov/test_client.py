"""Tests for the transferegov HTTP client."""

import httpx
import pytest
import respx

from mcp_brasil.transferegov import client
from mcp_brasil.transferegov.constants import TRANSFERENCIAS_ESPECIAIS_URL

_SAMPLE_EMENDA = {
    "id_transferencia_especial": 1,
    "ano_exercicio": 2024,
    "nr_emenda": "EMD-PIX-001",
    "autor_emenda": "Dep. Fulano da Silva",
    "tipo_emenda": "Individual",
    "funcao": "Saúde",
    "subfuncao": "Atenção Básica",
    "valor_empenhado": 500000.0,
    "valor_liquidado": 400000.0,
    "valor_pago": 300000.0,
    "nm_municipio_beneficiario": "Teresina",
    "uf_beneficiario": "PI",
    "nm_entidade_beneficiaria": "Prefeitura de Teresina",
    "objeto": "Construção de UBS",
}


# ---------------------------------------------------------------------------
# Helper: _build_query
# ---------------------------------------------------------------------------


class TestBuildQuery:
    def test_basic(self) -> None:
        result = client._build_query({"ano_exercicio": "eq.2024"})
        assert result["ano_exercicio"] == "eq.2024"
        assert result["limit"] == "15"
        assert result["offset"] == "0"

    def test_with_offset(self) -> None:
        result = client._build_query({}, limit=10, offset=20)
        assert result["limit"] == "10"
        assert result["offset"] == "20"

    def test_with_order(self) -> None:
        result = client._build_query({}, order="valor_pago.desc")
        assert result["order"] == "valor_pago.desc"


# ---------------------------------------------------------------------------
# Parser
# ---------------------------------------------------------------------------


class TestParseTransferencia:
    def test_full(self) -> None:
        result = client._parse_transferencia(_SAMPLE_EMENDA)
        assert result.nr_emenda == "EMD-PIX-001"
        assert result.autor_emenda == "Dep. Fulano da Silva"
        assert result.valor_pago == 300000.0
        assert result.nm_municipio_beneficiario == "Teresina"

    def test_empty(self) -> None:
        result = client._parse_transferencia({})
        assert result.nr_emenda is None
        assert result.valor_empenhado is None


# ---------------------------------------------------------------------------
# buscar_emendas_pix
# ---------------------------------------------------------------------------


class TestBuscarEmendasPix:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed(self) -> None:
        respx.get(TRANSFERENCIAS_ESPECIAIS_URL).mock(
            return_value=httpx.Response(200, json=[_SAMPLE_EMENDA])
        )
        result = await client.buscar_emendas_pix(ano=2024)
        assert len(result) == 1
        assert result[0].nr_emenda == "EMD-PIX-001"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty(self) -> None:
        respx.get(TRANSFERENCIAS_ESPECIAIS_URL).mock(return_value=httpx.Response(200, json=[]))
        result = await client.buscar_emendas_pix()
        assert result == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_non_list_response(self) -> None:
        respx.get(TRANSFERENCIAS_ESPECIAIS_URL).mock(
            return_value=httpx.Response(200, json={"error": "not found"})
        )
        result = await client.buscar_emendas_pix()
        assert result == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_with_uf_filter(self) -> None:
        route = respx.get(TRANSFERENCIAS_ESPECIAIS_URL).mock(
            return_value=httpx.Response(200, json=[])
        )
        await client.buscar_emendas_pix(ano=2024, uf="PI")
        assert "uf_beneficiario=eq.PI" in str(route.calls[0].request.url)


# ---------------------------------------------------------------------------
# buscar_emenda_por_autor
# ---------------------------------------------------------------------------


class TestBuscarEmendaPorAutor:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed(self) -> None:
        respx.get(TRANSFERENCIAS_ESPECIAIS_URL).mock(
            return_value=httpx.Response(200, json=[_SAMPLE_EMENDA])
        )
        result = await client.buscar_emenda_por_autor("Fulano")
        assert len(result) == 1
        assert result[0].autor_emenda == "Dep. Fulano da Silva"

    @pytest.mark.asyncio
    @respx.mock
    async def test_uses_ilike(self) -> None:
        route = respx.get(TRANSFERENCIAS_ESPECIAIS_URL).mock(
            return_value=httpx.Response(200, json=[])
        )
        await client.buscar_emenda_por_autor("Lira")
        url_str = str(route.calls[0].request.url)
        # URL-encoded: * → %2A
        assert "autor_emenda=ilike" in url_str
        assert "Lira" in url_str


# ---------------------------------------------------------------------------
# detalhe_emenda
# ---------------------------------------------------------------------------


class TestDetalheEmenda:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_detail(self) -> None:
        respx.get(TRANSFERENCIAS_ESPECIAIS_URL).mock(
            return_value=httpx.Response(200, json=[_SAMPLE_EMENDA])
        )
        result = await client.detalhe_emenda(1)
        assert result is not None
        assert result.nr_emenda == "EMD-PIX-001"

    @pytest.mark.asyncio
    @respx.mock
    async def test_not_found(self) -> None:
        respx.get(TRANSFERENCIAS_ESPECIAIS_URL).mock(return_value=httpx.Response(200, json=[]))
        result = await client.detalhe_emenda(999)
        assert result is None


# ---------------------------------------------------------------------------
# emendas_por_municipio
# ---------------------------------------------------------------------------


class TestEmendasPorMunicipio:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed(self) -> None:
        respx.get(TRANSFERENCIAS_ESPECIAIS_URL).mock(
            return_value=httpx.Response(200, json=[_SAMPLE_EMENDA])
        )
        result = await client.emendas_por_municipio("Teresina")
        assert len(result) == 1
        assert result[0].nm_municipio_beneficiario == "Teresina"

    @pytest.mark.asyncio
    @respx.mock
    async def test_uses_ilike(self) -> None:
        route = respx.get(TRANSFERENCIAS_ESPECIAIS_URL).mock(
            return_value=httpx.Response(200, json=[])
        )
        await client.emendas_por_municipio("São Paulo")
        url_str = str(route.calls[0].request.url)
        assert "ilike" in url_str


# ---------------------------------------------------------------------------
# resumo_emendas_ano
# ---------------------------------------------------------------------------


class TestResumoEmendasAno:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed(self) -> None:
        respx.get(TRANSFERENCIAS_ESPECIAIS_URL).mock(
            return_value=httpx.Response(200, json=[_SAMPLE_EMENDA])
        )
        result = await client.resumo_emendas_ano(2024)
        assert len(result) == 1

    @pytest.mark.asyncio
    @respx.mock
    async def test_filters_by_year(self) -> None:
        route = respx.get(TRANSFERENCIAS_ESPECIAIS_URL).mock(
            return_value=httpx.Response(200, json=[])
        )
        await client.resumo_emendas_ano(2025)
        assert "ano_exercicio=eq.2025" in str(route.calls[0].request.url)
