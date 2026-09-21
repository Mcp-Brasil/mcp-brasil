"""Tests for the TCM-GO HTTP client."""

import json

import httpx
import pytest
import respx

from mcp_brasil.data.tcm_go import client
from mcp_brasil.data.tcm_go.constants import (
    ESIPROC_LISTAR_CAMPO_URL,
    ESIPROC_PESQUISAR_URL,
    TCMJURIS_URL,
)
from mcp_brasil.exceptions import HttpClientError

# ---------------------------------------------------------------------------
# Fixtures (synthetic content only)
# ---------------------------------------------------------------------------

ACORDAOS_HTML = """
<html><body>
<table>
<tr><th>Seq</th><th>Acórdão</th><th>Processo</th><th>Julgamento</th>
<th>Relator</th><th>Município</th><th>Assunto</th></tr>
<tr>
  <td>1</td>
  <td><a href="/ecs/arquivo/acordao-0001-2024.pdf">0001/2024</a></td>
  <td>0000001/2024</td>
  <td>01/01/2024</td>
  <td>Cons. Fulano de Tal</td>
  <td>ANÁPOLIS</td>
  <td><font>DENUNCIAS</font></td>
</tr>
<tr>
  <td>2</td>
  <td><a href="https://www.tcmgo.tc.br/ecs/arquivo/acordao-0002-2024.pdf">0002/2024</a></td>
  <td>0000002/2024</td>
  <td>02/02/2024</td>
  <td>Cons. Beltrana de Tal</td>
  <td>GOIÂNIA</td>
  <td>CONTAS DE GESTÃO</td>
</tr>
</table>
</body></html>
"""


def _processos_payload() -> dict[str, object]:
    return {
        "hits": {
            "total": {"value": 2, "relation": "eq"},
            "hits": [
                {
                    "_source": {
                        "numero": "00000000.2024.000.0.00000.0001",
                        "municipio": "ANÁPOLIS",
                        "unidade_gestora": "PREFEITURA DE ANÁPOLIS",
                        "tipo_protocolo": "Contas de Gestão",
                        "ano_competencia": 2024,
                        "periodo_competencia": "Período 1/2024",
                        "relator": "Cons. Fulano de Tal",
                        "st_fase": "ARQUIVADO",
                        "finalizado": "SIM",
                        "data_protocolo": "2024-01-01T00:00Z",
                        "data_ultima_tramitacao": "2024-01-01T00:00Z",
                        "envolvidos": [
                            {
                                "nome": "Fulano de Tal",
                                "tipo_envolvimento": "RELATOR",
                            }
                        ],
                        "decisoes": [
                            {
                                "nome": "Dar Ciência",
                                "nome_arquivo": "Relatório Conclusivo",
                                "data_criacao": "2024-01-01T00:00Z",
                            }
                        ],
                    }
                },
                {"_source": {"numero": "00000000.2024.000.0.00000.0002", "municipio": "ANÁPOLIS"}},
            ],
        }
    }


# ---------------------------------------------------------------------------
# listar_valores_campo
# ---------------------------------------------------------------------------


class TestListarValoresCampo:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_values(self) -> None:
        respx.post(ESIPROC_LISTAR_CAMPO_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "aggregations": {
                        "distinct": {
                            "buckets": [
                                {"key": "Contas de Gestão", "doc_count": 100},
                                {"key": "Pensão", "doc_count": 200},
                            ]
                        }
                    }
                },
            )
        )
        valores = await client.listar_valores_campo(campo="tipo_protocolo.keyword")
        assert len(valores) == 2
        assert valores[0].valor == "Contas de Gestão"
        assert valores[0].quantidade == 100

    @pytest.mark.asyncio
    @respx.mock
    async def test_sends_municipio_scope(self) -> None:
        route = respx.post(ESIPROC_LISTAR_CAMPO_URL).mock(
            return_value=httpx.Response(200, json={"aggregations": {"distinct": {"buckets": []}}})
        )
        await client.listar_valores_campo(campo="relator.keyword", municipio="ANÁPOLIS")
        body = json.loads(route.calls[0].request.content)
        assert body["campo"] == "relator.keyword"
        assert body["filtros"]["municipio.keyword"] == "ANÁPOLIS"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_aggregation(self) -> None:
        respx.post(ESIPROC_LISTAR_CAMPO_URL).mock(
            return_value=httpx.Response(200, json={"aggregations": {"distinct": {"buckets": []}}})
        )
        valores = await client.listar_valores_campo(campo="area.keyword")
        assert valores == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_http_error_raises(self) -> None:
        respx.post(ESIPROC_LISTAR_CAMPO_URL).mock(return_value=httpx.Response(400))
        with pytest.raises(HttpClientError):
            await client.listar_valores_campo(campo="area.keyword")

    @pytest.mark.asyncio
    @respx.mock
    async def test_missing_aggregations_raises(self) -> None:
        respx.post(ESIPROC_LISTAR_CAMPO_URL).mock(
            return_value=httpx.Response(200, json={"foo": "bar"})
        )
        with pytest.raises(HttpClientError):
            await client.listar_valores_campo(campo="area.keyword")

    @pytest.mark.asyncio
    @respx.mock
    async def test_missing_buckets_raises(self) -> None:
        respx.post(ESIPROC_LISTAR_CAMPO_URL).mock(
            return_value=httpx.Response(200, json={"aggregations": {"distinct": {}}})
        )
        with pytest.raises(HttpClientError):
            await client.listar_valores_campo(campo="area.keyword")


# ---------------------------------------------------------------------------
# buscar_processos
# ---------------------------------------------------------------------------


class TestBuscarProcessos:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_processos(self) -> None:
        respx.post(ESIPROC_PESQUISAR_URL).mock(
            return_value=httpx.Response(200, json=_processos_payload())
        )
        resultado = await client.buscar_processos(municipio="ANÁPOLIS")
        assert resultado.total == 2
        assert resultado.pagina == 0
        assert len(resultado.processos) == 2
        primeiro = resultado.processos[0]
        assert primeiro.numero == "00000000.2024.000.0.00000.0001"
        assert primeiro.unidade_gestora == "PREFEITURA DE ANÁPOLIS"
        assert primeiro.ano_competencia == 2024
        assert primeiro.envolvidos[0].tipo_envolvimento == "RELATOR"
        assert primeiro.decisoes[0].arquivo == "Relatório Conclusivo"

    @pytest.mark.asyncio
    @respx.mock
    async def test_payload_keeps_all_filter_keys(self) -> None:
        route = respx.post(ESIPROC_PESQUISAR_URL).mock(
            return_value=httpx.Response(200, json={"hits": {"total": {"value": 0}, "hits": []}})
        )
        await client.buscar_processos(ano=2024, finalizado="SIM", pagina=1)
        body = json.loads(route.calls[0].request.content)
        assert body["filtros"]["ano_competencia"] == 2024
        assert body["filtros"]["finalizado.keyword"] == "SIM"
        assert body["pagina"] == 1
        assert "municipio.keyword" in body["filtros"]
        assert body["filtros"]["municipio.keyword"] is None

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.post(ESIPROC_PESQUISAR_URL).mock(
            return_value=httpx.Response(200, json={"hits": {"total": {"value": 0}, "hits": []}})
        )
        resultado = await client.buscar_processos(municipio="ANÁPOLIS")
        assert resultado.processos == []
        assert resultado.total == 0

    @pytest.mark.asyncio
    @respx.mock
    async def test_http_error_raises(self) -> None:
        respx.post(ESIPROC_PESQUISAR_URL).mock(return_value=httpx.Response(400))
        with pytest.raises(HttpClientError):
            await client.buscar_processos()

    @pytest.mark.asyncio
    @respx.mock
    async def test_missing_hits_raises(self) -> None:
        respx.post(ESIPROC_PESQUISAR_URL).mock(
            return_value=httpx.Response(200, json={"foo": "bar"})
        )
        with pytest.raises(HttpClientError):
            await client.buscar_processos()

    @pytest.mark.asyncio
    @respx.mock
    async def test_missing_total_raises(self) -> None:
        respx.post(ESIPROC_PESQUISAR_URL).mock(
            return_value=httpx.Response(200, json={"hits": {"hits": []}})
        )
        with pytest.raises(HttpClientError):
            await client.buscar_processos()


# ---------------------------------------------------------------------------
# buscar_acordaos
# ---------------------------------------------------------------------------


class TestBuscarAcordaos:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_acordaos(self) -> None:
        respx.get(TCMJURIS_URL).mock(
            return_value=httpx.Response(
                200, text=ACORDAOS_HTML, headers={"Content-Type": "text/html; charset=utf-8"}
            )
        )
        resultado = await client.buscar_acordaos(pesquisa="denuncias")
        assert resultado.total_servidor == 2
        assert resultado.total_apos_refiltro == 2
        assert resultado.truncado is False
        primeiro = resultado.acordaos[0]
        assert primeiro.numero_acordao == "0001/2024"
        assert primeiro.processo == "0000001/2024"
        assert primeiro.relator == "Cons. Fulano de Tal"
        assert primeiro.municipio == "ANÁPOLIS"
        assert primeiro.assunto == "DENUNCIAS"
        assert primeiro.url_pdf == "https://www.tcmgo.tc.br/ecs/arquivo/acordao-0001-2024.pdf"

    @pytest.mark.asyncio
    @respx.mock
    async def test_absolute_pdf_link_is_kept(self) -> None:
        respx.get(TCMJURIS_URL).mock(return_value=httpx.Response(200, text=ACORDAOS_HTML))
        resultado = await client.buscar_acordaos(pesquisa="gestao")
        assert resultado.acordaos[1].url_pdf == (
            "https://www.tcmgo.tc.br/ecs/arquivo/acordao-0002-2024.pdf"
        )

    @pytest.mark.asyncio
    @respx.mock
    async def test_refilters_municipio_exactly(self) -> None:
        respx.get(TCMJURIS_URL).mock(return_value=httpx.Response(200, text=ACORDAOS_HTML))
        resultado = await client.buscar_acordaos(municipio="ANAPOLIS")
        assert resultado.total_servidor == 2
        assert resultado.total_apos_refiltro == 1
        assert resultado.acordaos[0].municipio == "ANÁPOLIS"

    @pytest.mark.asyncio
    @respx.mock
    async def test_refilters_assunto_exactly(self) -> None:
        respx.get(TCMJURIS_URL).mock(return_value=httpx.Response(200, text=ACORDAOS_HTML))
        resultado = await client.buscar_acordaos(assunto="CONTAS DE GESTAO")
        assert resultado.total_apos_refiltro == 1
        assert resultado.acordaos[0].assunto == "CONTAS DE GESTÃO"

    @pytest.mark.asyncio
    @respx.mock
    async def test_sends_guest_params(self) -> None:
        route = respx.get(TCMJURIS_URL).mock(return_value=httpx.Response(200, text=ACORDAOS_HTML))
        await client.buscar_acordaos(pesquisa="pensão", oplog="OR")
        params = route.calls[0].request.url.params
        assert params["guest"] == "true"
        assert params["oplog"] == "OR"
        assert params["pesquisa"] == "pensão"

    @pytest.mark.asyncio
    @respx.mock
    async def test_truncated_flag_at_1000_rows(self) -> None:
        row = (
            "<tr><td>1</td><td><a href='/x.pdf'>0001/2024</a></td><td>0000001/2024</td>"
            "<td>01/01/2024</td><td>Cons. Fulano de Tal</td><td>ANÁPOLIS</td>"
            "<td>DENUNCIAS</td></tr>"
        )
        html = f"<html><body><table>{row * 1000}</table></body></html>"
        respx.get(TCMJURIS_URL).mock(return_value=httpx.Response(200, text=html))
        resultado = await client.buscar_acordaos(pesquisa="denuncias")
        assert resultado.total_servidor == 1000
        assert resultado.truncado is True

    @pytest.mark.asyncio
    @respx.mock
    async def test_explicit_no_results_marker_is_empty(self) -> None:
        html = "<html><body><p>Nenhum registro encontrado para a busca.</p></body></html>"
        respx.get(TCMJURIS_URL).mock(return_value=httpx.Response(200, text=html))
        resultado = await client.buscar_acordaos(pesquisa="inexistente")
        assert resultado.acordaos == []
        assert resultado.total_servidor == 0

    @pytest.mark.asyncio
    @respx.mock
    async def test_results_table_without_rows_is_empty(self) -> None:
        html = (
            "<html><body><table>"
            "<tr><th>Seq</th><th>Acórdão</th><th>Processo</th><th>Julgamento</th>"
            "<th>Relator</th><th>Município</th><th>Assunto</th></tr>"
            "</table></body></html>"
        )
        respx.get(TCMJURIS_URL).mock(return_value=httpx.Response(200, text=html))
        resultado = await client.buscar_acordaos(pesquisa="sem linhas")
        assert resultado.acordaos == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_unexpected_html_raises(self) -> None:
        html = "<html><body><p>Página fora do ar ou layout inesperado.</p></body></html>"
        respx.get(TCMJURIS_URL).mock(return_value=httpx.Response(200, text=html))
        with pytest.raises(HttpClientError):
            await client.buscar_acordaos(pesquisa="denuncias")

    @pytest.mark.asyncio
    @respx.mock
    async def test_http_error_raises(self) -> None:
        respx.get(TCMJURIS_URL).mock(return_value=httpx.Response(400))
        with pytest.raises(HttpClientError):
            await client.buscar_acordaos(pesquisa="denuncias")
