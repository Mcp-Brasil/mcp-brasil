"""Tests for the DOU (federal) HTTP client."""

import json

import httpx
import pytest
import respx

from mcp_brasil.data.diario_oficial import client_dou
from mcp_brasil.data.diario_oficial.constants import DOU_ARTICLE_URL, DOU_SEARCH_URL


def _mock_search_response(total: int = 1, items: list | None = None) -> dict:
    """Build a mock DOU search API response (JSON payload)."""
    if items is None:
        items = [
            {
                "title": "Portaria nº 123",
                "abstract": "Dispõe sobre medidas administrativas",
                "urlTitle": "portaria-123-2024",
                "pubName": "Ministério da Saúde",
                "artType": "Portaria",
                "pubType": "DO1",
                "pubDate": "2024-01-15",
                "numberPage": "42",
                "pageNumber": "10",
                "content": "Texto completo da portaria",
                "assina": "João Silva",
                "cargo": "Ministro de Estado da Saúde",
            }
        ]
    return {"jsonArray": items, "total": total}


def _wrap_in_html(payload: dict) -> str:
    """Wrap a JSON payload in HTML mimicking the DOU search page."""
    tag_id = "_br_com_seatecnologia_in_buscadou_BuscaDouPortlet_params"
    json_str = json.dumps(payload, ensure_ascii=False)
    return (
        "<html><body>"
        f'<script id="{tag_id}" type="application/json">'
        f"{json_str}"
        "</script>"
        "</body></html>"
    )


# ---------------------------------------------------------------------------
# buscar_dou
# ---------------------------------------------------------------------------


class TestBuscarDou:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_results(self) -> None:
        html = _wrap_in_html(_mock_search_response())
        respx.get(DOU_SEARCH_URL).mock(
            return_value=httpx.Response(
                200, text=html, headers={"content-type": "text/html; charset=UTF-8"}
            )
        )
        result = await client_dou.buscar_dou(termo="portaria")
        assert result.total == 1
        assert len(result.publicacoes) == 1
        pub = result.publicacoes[0]
        assert pub.titulo == "Portaria nº 123"
        assert pub.orgao == "Ministério da Saúde"
        assert pub.tipo_publicacao == "Portaria"
        assert pub.data_publicacao == "2024-01-15"
        assert pub.assinante == "João Silva"

    @pytest.mark.asyncio
    @respx.mock
    async def test_search_params_sent(self) -> None:
        html = _wrap_in_html(_mock_search_response(0, []))
        route = respx.get(DOU_SEARCH_URL).mock(
            return_value=httpx.Response(
                200, text=html, headers={"content-type": "text/html; charset=UTF-8"}
            )
        )
        await client_dou.buscar_dou(termo="decreto", secao="SECAO_1", periodo="SEMANA")
        req_url = str(route.calls[0].request.url)
        assert "q=decreto" in req_url
        # Section names are mapped to DOU codes
        assert "s=do1" in req_url

    @pytest.mark.asyncio
    @respx.mock
    async def test_orgao_filter(self) -> None:
        html = _wrap_in_html(_mock_search_response(0, []))
        route = respx.get(DOU_SEARCH_URL).mock(
            return_value=httpx.Response(
                200, text=html, headers={"content-type": "text/html; charset=UTF-8"}
            )
        )
        await client_dou.buscar_dou(termo="teste", orgao="IBAMA")
        req_url = str(route.calls[0].request.url)
        assert "orgPrin=IBAMA" in req_url

    @pytest.mark.asyncio
    @respx.mock
    async def test_date_range_params(self) -> None:
        html = _wrap_in_html(_mock_search_response(0, []))
        route = respx.get(DOU_SEARCH_URL).mock(
            return_value=httpx.Response(
                200, text=html, headers={"content-type": "text/html; charset=UTF-8"}
            )
        )
        await client_dou.buscar_dou(
            termo="teste",
            data_inicio="2024-01-01",
            data_fim="2024-01-31",
        )
        req_url = str(route.calls[0].request.url)
        # Dates are converted to dd-mm-yyyy format for the DOU API
        assert "publishFrom=01-01-2024" in req_url
        assert "publishTo=31-01-2024" in req_url

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        html = _wrap_in_html({"jsonArray": [], "total": 0})
        respx.get(DOU_SEARCH_URL).mock(
            return_value=httpx.Response(
                200, text=html, headers={"content-type": "text/html; charset=UTF-8"}
            )
        )
        result = await client_dou.buscar_dou(termo="inexistente")
        assert result.total == 0
        assert result.publicacoes == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_tipo_publicacao_filter(self) -> None:
        html = _wrap_in_html(_mock_search_response(0, []))
        route = respx.get(DOU_SEARCH_URL).mock(
            return_value=httpx.Response(
                200, text=html, headers={"content-type": "text/html; charset=UTF-8"}
            )
        )
        await client_dou.buscar_dou(termo="teste", tipo_publicacao="Decreto")
        req_url = str(route.calls[0].request.url)
        assert "artType=Decreto" in req_url

    @pytest.mark.asyncio
    @respx.mock
    async def test_missing_script_tag_returns_empty(self) -> None:
        """When the HTML has no script tag, return empty result."""
        respx.get(DOU_SEARCH_URL).mock(
            return_value=httpx.Response(
                200,
                text="<html><body></body></html>",
                headers={"content-type": "text/html; charset=UTF-8"},
            )
        )
        result = await client_dou.buscar_dou(termo="teste")
        assert result.total == 0
        assert result.publicacoes == []


# ---------------------------------------------------------------------------
# ler_publicacao_dou
# ---------------------------------------------------------------------------


class TestLerPublicacaoDou:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_article(self) -> None:
        url = f"{DOU_ARTICLE_URL}/portaria-123-2024"
        respx.get(url).mock(
            return_value=httpx.Response(
                200,
                json={
                    "title": "Portaria nº 123",
                    "content": "Texto completo da portaria...",
                    "pubName": "Ministério da Saúde",
                    "artType": "Portaria",
                    "pubDate": "2024-01-15",
                    "assina": "João Silva",
                    "cargo": "Ministro",
                },
            )
        )
        result = await client_dou.ler_publicacao_dou("portaria-123-2024")
        assert result is not None
        assert result.titulo == "Portaria nº 123"
        assert result.conteudo == "Texto completo da portaria..."
        assert result.assinante == "João Silva"

    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_none_on_error(self) -> None:
        url = f"{DOU_ARTICLE_URL}/nao-existe"
        respx.get(url).mock(return_value=httpx.Response(404, json={}))
        result = await client_dou.ler_publicacao_dou("nao-existe")
        assert result is None
