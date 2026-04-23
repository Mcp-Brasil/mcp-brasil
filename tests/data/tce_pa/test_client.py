"""Tests for the TCE-PA HTTP client."""

import httpx
import pytest
import respx

from mcp_brasil.data.tce_pa import client
from mcp_brasil.data.tce_pa.constants import DIARIO_OFICIAL_URL, SESSOES_SEARCH_URL

_HTML_HEADERS = {"content-type": "text/html"}

# ---------------------------------------------------------------------------
# HTML fixtures
# ---------------------------------------------------------------------------

_SESSAO_HTML = """
<html><body>
<div class="resultado-organico">
  <h2 class="resultado-organico-titulo">
    <a class="titulo"
       href="/pesquisaintegrada/sessoes-plenarias/codigo/42/sessao-01-2024">
      Sessão Ordinária nº 01/2024
    </a>
  </h2>
  <div class="resultado-organico-campo">
    <dfn class="resultado-organico-campo-nome">Tipo de Sessão</dfn>
    <span class="resultado-organico-campo-valor">Ordinária</span>
  </div>
  <div class="resultado-organico-campo">
    <dfn class="resultado-organico-campo-nome">Data da Sessão</dfn>
    <span class="resultado-organico-campo-valor">15/01/2024</span>
  </div>
  <div class="resultado-organico-campo">
    <dfn class="resultado-organico-campo-nome">Ano</dfn>
    <span class="resultado-organico-campo-valor">2024</span>
  </div>
</div>
</body></html>
"""

_RESULTADO_HTML = """
<html><body>
<div class="resultado-organico">
  <h2 class="resultado-organico-titulo">
    <a class="titulo"
       href="/pesquisaintegrada/acordaos/codigo/99/acordao-1234-2024">
      Acórdão nº 1234/2024
    </a>
  </h2>
  <div class="resultado-organico-campo">
    <dfn class="resultado-organico-campo-nome">Data da sessão plenária</dfn>
    <span class="resultado-organico-campo-valor">20/03/2024</span>
  </div>
  <div class="resultado-organico-campo">
    <dfn class="resultado-organico-campo-nome">Fonte</dfn>
    <span class="resultado-organico-campo-valor">TCE-PA</span>
  </div>
</div>
</body></html>
"""

_EMPTY_HTML = "<html><body><p>Nenhum resultado</p></body></html>"


# ---------------------------------------------------------------------------
# buscar_diario_oficial
# ---------------------------------------------------------------------------


class TestBuscarDiarioOficial:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_publicacoes(self) -> None:
        respx.get(DIARIO_OFICIAL_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "data": [
                        {
                            "NumeroPublicacao": 42,
                            "DataPublicacao": "2024-03-15",
                            "TipoAto": "Contratos",
                            "Publicacao": "Contrato nº 001/2024",
                        }
                    ]
                },
            )
        )
        result = await client.buscar_diario_oficial(ano=2024)
        assert len(result) == 1
        assert result[0].numero_publicacao == 42
        assert result[0].tipo_ato == "Contratos"
        assert result[0].data_publicacao == "2024-03-15"

    @pytest.mark.asyncio
    @respx.mock
    async def test_strips_html_from_publicacao(self) -> None:
        respx.get(DIARIO_OFICIAL_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "data": [
                        {
                            "NumeroPublicacao": 1,
                            "DataPublicacao": "2024-01-01",
                            "TipoAto": "Atos e Normas",
                            "Publicacao": "<b>Resolução</b> nº <i>001/2024</i>",
                        }
                    ]
                },
            )
        )
        result = await client.buscar_diario_oficial(ano=2024)
        assert "<b>" not in result[0].publicacao
        assert "<i>" not in result[0].publicacao
        assert "Resolução" in result[0].publicacao
        assert "001/2024" in result[0].publicacao

    @pytest.mark.asyncio
    @respx.mock
    async def test_handles_flat_list_response(self) -> None:
        respx.get(DIARIO_OFICIAL_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "NumeroPublicacao": 7,
                        "DataPublicacao": "2024-06-10",
                        "TipoAto": "Licitações",
                        "Publicacao": "Pregão Eletrônico nº 001/2024",
                    }
                ],
            )
        )
        result = await client.buscar_diario_oficial(ano=2024)
        assert len(result) == 1
        assert result[0].numero_publicacao == 7

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_data_envelope(self) -> None:
        respx.get(DIARIO_OFICIAL_URL).mock(return_value=httpx.Response(200, json={"data": []}))
        result = await client.buscar_diario_oficial(ano=2024)
        assert result == []


# ---------------------------------------------------------------------------
# buscar_sessoes_plenarias (HTML scraping)
# ---------------------------------------------------------------------------


class TestBuscarSessoesPlenarias:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_sessoes(self) -> None:
        respx.get(SESSOES_SEARCH_URL).mock(
            return_value=httpx.Response(200, text=_SESSAO_HTML, headers=_HTML_HEADERS)
        )
        result = await client.buscar_sessoes_plenarias(tipo="sessoes")
        assert len(result) == 1
        assert result[0].titulo == "Sessão Ordinária nº 01/2024"
        assert result[0].codigo == 42
        assert result[0].tipo_sessao == "Ordinária"
        assert result[0].data_sessao == "15/01/2024"
        assert result[0].ano == 2024

    @pytest.mark.asyncio
    @respx.mock
    async def test_doc_url_pautas(self) -> None:
        respx.get(SESSOES_SEARCH_URL).mock(
            return_value=httpx.Response(200, text=_SESSAO_HTML, headers=_HTML_HEADERS)
        )
        result = await client.buscar_sessoes_plenarias(tipo="pautas")
        assert result[0].url_documento is not None
        assert result[0].url_documento.endswith("/download")

    @pytest.mark.asyncio
    @respx.mock
    async def test_doc_url_atas(self) -> None:
        respx.get(SESSOES_SEARCH_URL).mock(
            return_value=httpx.Response(200, text=_SESSAO_HTML, headers=_HTML_HEADERS)
        )
        result = await client.buscar_sessoes_plenarias(tipo="atas")
        assert result[0].url_documento is not None
        assert result[0].url_documento.endswith("/download")

    @pytest.mark.asyncio
    @respx.mock
    async def test_doc_url_videos(self) -> None:
        respx.get(SESSOES_SEARCH_URL).mock(
            return_value=httpx.Response(200, text=_SESSAO_HTML, headers=_HTML_HEADERS)
        )
        result = await client.buscar_sessoes_plenarias(tipo="videos")
        assert result[0].url_documento is not None
        assert result[0].url_documento.endswith("/conteudo-original")

    @pytest.mark.asyncio
    @respx.mock
    async def test_doc_url_sessoes_is_none(self) -> None:
        respx.get(SESSOES_SEARCH_URL).mock(
            return_value=httpx.Response(200, text=_SESSAO_HTML, headers=_HTML_HEADERS)
        )
        result = await client.buscar_sessoes_plenarias(tipo="sessoes")
        assert result[0].url_documento is None

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_html_returns_empty_list(self) -> None:
        respx.get(SESSOES_SEARCH_URL).mock(
            return_value=httpx.Response(200, text=_EMPTY_HTML, headers=_HTML_HEADERS)
        )
        result = await client.buscar_sessoes_plenarias(tipo="sessoes")
        assert result == []


# ---------------------------------------------------------------------------
# buscar_pesquisa_integrada (HTML scraping)
# ---------------------------------------------------------------------------


class TestBuscarPesquisaIntegrada:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_resultados(self) -> None:
        respx.get(SESSOES_SEARCH_URL).mock(
            return_value=httpx.Response(200, text=_RESULTADO_HTML, headers=_HTML_HEADERS)
        )
        result = await client.buscar_pesquisa_integrada(slug="acordaos")
        assert len(result) == 1
        assert result[0].titulo == "Acórdão nº 1234/2024"
        assert "Data da sessão plenária" in result[0].campos
        assert result[0].campos["Data da sessão plenária"] == "20/03/2024"

    @pytest.mark.asyncio
    @respx.mock
    async def test_fonte_campo_filtered_out(self) -> None:
        respx.get(SESSOES_SEARCH_URL).mock(
            return_value=httpx.Response(200, text=_RESULTADO_HTML, headers=_HTML_HEADERS)
        )
        result = await client.buscar_pesquisa_integrada(slug="acordaos")
        assert "Fonte" not in result[0].campos

    @pytest.mark.asyncio
    @respx.mock
    async def test_pdf_url_for_bases_pdf_download(self) -> None:
        respx.get(SESSOES_SEARCH_URL).mock(
            return_value=httpx.Response(200, text=_RESULTADO_HTML, headers=_HTML_HEADERS)
        )
        result = await client.buscar_pesquisa_integrada(slug="acordaos")
        assert result[0].url_documento is not None
        assert result[0].url_documento.endswith("/download")

    @pytest.mark.asyncio
    @respx.mock
    async def test_conteudo_original_url_for_other_bases(self) -> None:
        respx.get(SESSOES_SEARCH_URL).mock(
            return_value=httpx.Response(200, text=_RESULTADO_HTML, headers=_HTML_HEADERS)
        )
        result = await client.buscar_pesquisa_integrada(slug="noticias-portal-internet")
        assert result[0].url_documento is not None
        assert result[0].url_documento.endswith("/conteudo-original")

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_html_returns_empty_list(self) -> None:
        respx.get(SESSOES_SEARCH_URL).mock(
            return_value=httpx.Response(200, text=_EMPTY_HTML, headers=_HTML_HEADERS)
        )
        result = await client.buscar_pesquisa_integrada(slug="resolucoes")
        assert result == []


# ---------------------------------------------------------------------------
# _parse_sessoes_html (pure function — unit tested directly)
# ---------------------------------------------------------------------------


class TestParseSessoesHtml:
    def test_skips_card_without_titulo_anchor(self) -> None:
        html = """
        <div class="resultado-organico">
          <h2 class="resultado-organico-titulo"><span>Sem link</span></h2>
        </div>
        """
        result = client._parse_sessoes_html(html, tipo="sessoes")
        assert result == []

    def test_extracts_codigo_from_url(self) -> None:
        html = """
        <div class="resultado-organico">
          <h2 class="resultado-organico-titulo">
            <a class="titulo"
               href="/pesquisaintegrada/sessoes/codigo/777/titulo">Título</a>
          </h2>
        </div>
        """
        result = client._parse_sessoes_html(html, tipo="sessoes")
        assert result[0].codigo == 777

    def test_strips_query_params_from_href(self) -> None:
        html = """
        <div class="resultado-organico">
          <h2 class="resultado-organico-titulo">
            <a class="titulo"
               href="/pesquisaintegrada/sessoes/codigo/1/titulo?foo=bar">Título</a>
          </h2>
        </div>
        """
        result = client._parse_sessoes_html(html, tipo="sessoes")
        assert "?" not in result[0].url_pagina

    def test_removes_conteudo_original_suffix(self) -> None:
        html = """
        <div class="resultado-organico">
          <h2 class="resultado-organico-titulo">
            <a class="titulo"
               href="/pesquisaintegrada/sessoes/codigo/1/titulo/conteudo-original">
              Título
            </a>
          </h2>
        </div>
        """
        result = client._parse_sessoes_html(html, tipo="sessoes")
        assert not result[0].url_pagina.endswith("/conteudo-original")


# ---------------------------------------------------------------------------
# _parse_resultados_html (pure function — unit tested directly)
# ---------------------------------------------------------------------------


class TestParseResultadosHtml:
    def test_skips_card_without_titulo_anchor(self) -> None:
        html = """
        <div class="resultado-organico">
          <h2 class="resultado-organico-titulo"><span>Sem link</span></h2>
        </div>
        """
        result = client._parse_resultados_html(html, slug="acordaos")
        assert result == []

    def test_extracts_campos_from_card(self) -> None:
        html = """
        <div class="resultado-organico">
          <h2 class="resultado-organico-titulo">
            <a class="titulo" href="/pesquisaintegrada/acordaos/codigo/1/titulo">
              Acórdão nº 1/2024
            </a>
          </h2>
          <div class="resultado-organico-campo">
            <dfn class="resultado-organico-campo-nome">Data</dfn>
            <span class="resultado-organico-campo-valor">01/01/2024</span>
          </div>
        </div>
        """
        result = client._parse_resultados_html(html, slug="acordaos")
        assert len(result) == 1
        assert "Data" in result[0].campos
        assert result[0].campos["Data"] == "01/01/2024"

    def test_filters_fonte_campo(self) -> None:
        html = """
        <div class="resultado-organico">
          <h2 class="resultado-organico-titulo">
            <a class="titulo" href="/pesquisaintegrada/acordaos/codigo/1/titulo">Título</a>
          </h2>
          <div class="resultado-organico-campo">
            <dfn class="resultado-organico-campo-nome">Fonte</dfn>
            <span class="resultado-organico-campo-valor">TCE-PA</span>
          </div>
        </div>
        """
        result = client._parse_resultados_html(html, slug="acordaos")
        assert "Fonte" not in result[0].campos

    def test_pdf_url_for_pdf_bases(self) -> None:
        html = """
        <div class="resultado-organico">
          <h2 class="resultado-organico-titulo">
            <a class="titulo" href="/pesquisaintegrada/acordaos/codigo/1/titulo">Título</a>
          </h2>
        </div>
        """
        result = client._parse_resultados_html(html, slug="acordaos")
        assert result[0].url_documento is not None
        assert result[0].url_documento.endswith("/download")

    def test_conteudo_original_url_for_non_pdf_bases(self) -> None:
        html = """
        <div class="resultado-organico">
          <h2 class="resultado-organico-titulo">
            <a class="titulo" href="/pesquisaintegrada/noticias/codigo/1/titulo">Título</a>
          </h2>
        </div>
        """
        result = client._parse_resultados_html(html, slug="noticias-portal-internet")
        assert result[0].url_documento is not None
        assert result[0].url_documento.endswith("/conteudo-original")

    def test_strips_query_params_from_href(self) -> None:
        html = """
        <div class="resultado-organico">
          <h2 class="resultado-organico-titulo">
            <a class="titulo"
               href="/pesquisaintegrada/acordaos/codigo/1/titulo?q=teste">Título</a>
          </h2>
        </div>
        """
        result = client._parse_resultados_html(html, slug="acordaos")
        assert "?" not in result[0].url_pagina

    def test_truncates_long_campo_value(self) -> None:
        long_value = "X" * 300
        html = f"""
        <div class="resultado-organico">
          <h2 class="resultado-organico-titulo">
            <a class="titulo" href="/pesquisaintegrada/acordaos/codigo/1/titulo">Título</a>
          </h2>
          <div class="resultado-organico-campo">
            <dfn class="resultado-organico-campo-nome">Descrição</dfn>
            <span class="resultado-organico-campo-valor">{long_value}</span>
          </div>
        </div>
        """
        result = client._parse_resultados_html(html, slug="acordaos")
        assert len(result[0].campos["Descrição"]) == 200


# ---------------------------------------------------------------------------
# _build_query (pure function)
# ---------------------------------------------------------------------------


class TestBuildQuery:
    def test_empty_returns_empty_string(self) -> None:
        assert client._build_query("", None, None) == ""

    def test_query_only(self) -> None:
        assert client._build_query("licitação", None, None) == "licitação"

    def test_ano_appended(self) -> None:
        result = client._build_query("", 2024, None)
        assert "ano:2024" in result

    def test_mes_appended_zero_padded(self) -> None:
        result = client._build_query("", None, 3)
        assert "mes:03" in result

    def test_all_params_combined(self) -> None:
        result = client._build_query("contrato", 2024, 6)
        assert "contrato" in result
        assert "ano:2024" in result
        assert "mes:06" in result

    def test_no_query_with_ano(self) -> None:
        result = client._build_query("", 2023, None)
        assert result == "ano:2023"


# ---------------------------------------------------------------------------
# _strip_html (pure function)
# ---------------------------------------------------------------------------


class TestStripHtml:
    def test_removes_tags(self) -> None:
        assert client._strip_html("<b>Texto</b>") == "Texto"

    def test_removes_nested_tags(self) -> None:
        result = client._strip_html("<p><b>Texto</b><i>Extra</i></p>")
        assert "Texto" in result
        assert "Extra" in result
        assert "<" not in result

    def test_empty_string(self) -> None:
        assert client._strip_html("") == ""

    def test_no_tags(self) -> None:
        assert client._strip_html("Texto simples") == "Texto simples"
