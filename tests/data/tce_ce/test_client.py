"""Tests for the TCE-CE HTTP client (SIM 2.0).

Os payloads reproduzem respostas reais medidas em 14/10/2026
(municipio 134 = Pentecoste).
"""

import httpx
import pytest
import respx

from mcp_brasil.data.tce_ce import client
from mcp_brasil.data.tce_ce.constants import (
    CONTRATADOS_URL,
    CONTRATOS_URL,
    EMPENHOS_URL,
    MUNICIPIOS_URL,
    PROCESSOS_URL,
)

NEXT = [{"rel": "next", "title": "Next interval", "href": "?$start_index=1"}]

# ---------------------------------------------------------------------------
# helpers
# ---------------------------------------------------------------------------


class TestHelpers:
    def test_codigo_municipio_zero_padded(self) -> None:
        assert client.normalizar_codigo_municipio("57") == "057"
        assert client.normalizar_codigo_municipio(134) == "134"

    def test_codigo_municipio_invalido(self) -> None:
        with pytest.raises(ValueError):
            client.normalizar_codigo_municipio("abc")

    @pytest.mark.parametrize(
        ("periodo", "esperado"),
        [
            ("2026", ("20260101", "20261231")),
            ("2026-08-31", ("20260831", "20260831")),
            ("2026-01-01_2026-08-31", ("20260101", "20260831")),
            ("01/01/2026_31/08/2026", ("20260101", "20260831")),
            ("20260101_20260831", ("20260101", "20260831")),
        ],
    )
    def test_parse_periodo(self, periodo: str, esperado: tuple[str, str]) -> None:
        assert client.parse_periodo(periodo) == esperado

    def test_parse_periodo_invalido(self) -> None:
        with pytest.raises(ValueError):
            client.parse_periodo("agosto")


# ---------------------------------------------------------------------------
# listar_municipios
# ---------------------------------------------------------------------------


class TestListarMunicipios:
    @respx.mock
    async def test_returns_parsed_municipios(self) -> None:
        respx.get(MUNICIPIOS_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "name": "dv_municipios",
                    "elements": [
                        {
                            "codigo_municipio": "057",
                            "nome_municipio": "FORTALEZA",
                            "codigo_municipio_ibge": "2304400",
                        },
                        {"codigo_municipio": "134", "nome_municipio": "PENTECOSTE"},
                    ],
                },
            )
        )
        result = await client.listar_municipios()
        assert [m.codigo_municipio for m in result] == ["057", "134"]
        assert result[0].codigo_municipio_ibge == "2304400"

    @respx.mock
    async def test_filtro_por_nome_vai_em_maiusculas(self) -> None:
        route = respx.get(MUNICIPIOS_URL).mock(
            return_value=httpx.Response(200, json={"name": "dv_municipios", "elements": []})
        )
        await client.listar_municipios(nome_municipio="Pentecoste")
        assert route.calls[0].request.url.params["nome_municipio"] == "PENTECOSTE"

    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(MUNICIPIOS_URL).mock(return_value=httpx.Response(200, json={"elements": []}))
        assert await client.listar_municipios() == []


# ---------------------------------------------------------------------------
# buscar_licitacoes
# ---------------------------------------------------------------------------


class TestBuscarLicitacoes:
    @respx.mock
    async def test_parametros_e_parse(self) -> None:
        route = respx.get(PROCESSOS_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "name": "dv_processos_administrativos_contratacoes",
                    "elements": [
                        {
                            "codigo_municipio": "134",
                            "data_realizacao_autuacao_licitacao": "2026-08-05T00:00:00",
                            "numero_licitacao": "09.002/2026-AD",
                            "modalidade_licitacao": "9",
                            "descricao_objeto_licitacao": "ADESAO A ATA DE REGISTRO DE PRECOS",
                            "valor_orcado_estimado": 265141.5,
                            "data_homologacao": "2026-08-06T00:00:00",
                            "nome_responsavel_homologacao": "JOSE DENILSON ALVES SALDANHA",
                            "modalidade_processo_administrativo": "R",
                        }
                    ],
                    "links": NEXT,
                },
            )
        )
        result = await client.buscar_licitacoes(
            codigo_municipio="134", data_realizacao="2026-08-01_2026-08-31"
        )
        params = route.calls[0].request.url.params
        assert params["data_inicio"] == "20260801"
        assert params["data_fim"] == "20260831"
        assert params["codigo_municipio"] == "134"
        assert result.tem_mais is True
        lic = result.licitacoes[0]
        assert lic.numero_licitacao == "09.002/2026-AD"
        assert lic.data_realizacao == "2026-08-05"
        assert lic.valor_orcado_estimado == 265141.5
        assert lic.data_homologacao == "2026-08-06"

    @respx.mock
    async def test_empty(self) -> None:
        respx.get(PROCESSOS_URL).mock(return_value=httpx.Response(200, json={"elements": []}))
        result = await client.buscar_licitacoes(codigo_municipio="057", data_realizacao="2026")
        assert result.licitacoes == []
        assert result.tem_mais is False


# ---------------------------------------------------------------------------
# buscar_contratos (+ cruzamento com /contratados)
# ---------------------------------------------------------------------------

CONTRATO_AMBULANCIAS = {
    "codigo_municipio": "134",
    "data_contrato": "2026-08-31T00:00:00",
    "numero_contrato": "20260826003",
    "tipo_contrato": "L",
    "modalide_contrato": "OR",
    "descricao_objeto_contrato": "Aquisicao de 04 ambulancias e 02 veiculos de passeio",
    "valor_total_contrato": 185590.0,
    "data_inicio_vigencia_contrato": "2026-08-31T00:00:00",
    "data_fim_vigencia_contrato": "2027-08-31T00:00:00",
    "numero_processo_adm": " 11.012/2026-PE",
    "nome_fiscal_contrato": "MARIA ZULEIDE AMORIM MUNIZ",
}

CONTRATADO_AMBULANCIAS = {
    "codigo_municipio": "134",
    "data_contrato": "2026-08-31T00:00:00",
    "numero_contrato": "20260826003",
    "numero_documento_negociante": "17161125000104",
    "nome_negociante": "TERRA DO SOL COMERCIO DE VEICULOS LTDA",
    "nome_municipio_negociante": "Quixada",
    "codigo_uf": "CE",
}


class TestBuscarContratos:
    @respx.mock
    async def test_cruza_contratado_por_numero_e_data(self) -> None:
        route = respx.get(CONTRATOS_URL).mock(
            return_value=httpx.Response(200, json={"elements": [CONTRATO_AMBULANCIAS]})
        )
        rota_contratados = respx.get(CONTRATADOS_URL).mock(
            return_value=httpx.Response(200, json={"elements": [CONTRATADO_AMBULANCIAS]})
        )
        result = await client.buscar_contratos(codigo_municipio="134", data_contrato="2026")
        params = route.calls[0].request.url.params
        assert params["data_inicio"] == "20260101"
        assert params["$count"] == "100"
        # contratados: so o intervalo de datas da pagina, nao o ano inteiro
        p_contratados = rota_contratados.calls[0].request.url.params
        assert p_contratados["data_inicio"] == "20260831"
        assert p_contratados["data_fim"] == "20260831"
        assert result.contratado_incompleto is False
        c = result.contratos[0]
        assert c.numero_contrato == "20260826003"
        assert c.modalidade_contrato == "OR"
        assert c.data_fim_vigencia == "2027-08-31"
        assert c.numero_processo_adm == "11.012/2026-PE"
        assert c.nome_contratado == "TERRA DO SOL COMERCIO DE VEICULOS LTDA"
        assert c.documento_contratado == "17161125000104"

    @respx.mock
    async def test_mesmo_numero_em_outra_data_nao_casa(self) -> None:
        respx.get(CONTRATOS_URL).mock(
            return_value=httpx.Response(200, json={"elements": [CONTRATO_AMBULANCIAS]})
        )
        outro = {**CONTRATADO_AMBULANCIAS, "data_contrato": "2025-01-10T00:00:00"}
        respx.get(CONTRATADOS_URL).mock(
            return_value=httpx.Response(200, json={"elements": [outro]})
        )
        result = await client.buscar_contratos(codigo_municipio="134", data_contrato="2026")
        assert result.contratos[0].nome_contratado is None

    @respx.mock
    async def test_contratados_paginados(self) -> None:
        respx.get(CONTRATOS_URL).mock(
            return_value=httpx.Response(200, json={"elements": [CONTRATO_AMBULANCIAS]})
        )
        outro = {**CONTRATADO_AMBULANCIAS, "numero_contrato": "OUTRO"}
        rota = respx.get(CONTRATADOS_URL).mock(
            side_effect=[
                httpx.Response(200, json={"elements": [outro], "links": NEXT}),
                httpx.Response(200, json={"elements": [CONTRATADO_AMBULANCIAS]}),
            ]
        )
        result = await client.buscar_contratos(codigo_municipio="134", data_contrato="2026")
        assert rota.call_count == 2
        assert rota.calls[1].request.url.params["$start_index"] == "1"
        assert result.contratos[0].nome_contratado == "TERRA DO SOL COMERCIO DE VEICULOS LTDA"

    @respx.mock
    async def test_teto_de_contratados_e_declarado(self) -> None:
        """Freio atingido = truncado declarado, nunca corte silencioso."""
        outro = {**CONTRATADO_AMBULANCIAS, "numero_contrato": "OUTRO"}
        respx.get(CONTRATADOS_URL).mock(
            return_value=httpx.Response(200, json={"elements": [outro, outro], "links": NEXT})
        )
        contratados, truncado = await client.buscar_contratados(
            codigo_municipio="134", data_contrato="2026", limite=3
        )
        assert truncado is True
        assert len(contratados) == 4

    @respx.mock
    async def test_resultado_sinaliza_cruzamento_incompleto(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        respx.get(CONTRATOS_URL).mock(
            return_value=httpx.Response(200, json={"elements": [CONTRATO_AMBULANCIAS]})
        )
        respx.get(CONTRATADOS_URL).mock(
            return_value=httpx.Response(200, json={"elements": [], "links": NEXT})
        )
        # o freio e o default de `limite`; zero forca o caso "parou no teto"
        monkeypatch.setitem(client.buscar_contratados.__kwdefaults__, "limite", 0)
        result = await client.buscar_contratos(codigo_municipio="134", data_contrato="2026")
        assert result.contratado_incompleto is True
        assert result.contratos[0].nome_contratado is None

    @respx.mock
    async def test_sem_contratos_nao_consulta_contratados(self) -> None:
        respx.get(CONTRATOS_URL).mock(return_value=httpx.Response(200, json={"elements": []}))
        rota = respx.get(CONTRATADOS_URL).mock(
            return_value=httpx.Response(200, json={"elements": []})
        )
        result = await client.buscar_contratos(codigo_municipio="134", data_contrato="2026")
        assert result.contratos == []
        assert rota.call_count == 0

    @respx.mock
    async def test_quantidade_limitada_a_1000(self) -> None:
        route = respx.get(CONTRATOS_URL).mock(
            return_value=httpx.Response(200, json={"elements": []})
        )
        await client.buscar_contratos(
            codigo_municipio="134", data_contrato="2026", quantidade=5000, deslocamento=200
        )
        params = route.calls[0].request.url.params
        assert params["$count"] == "1000"
        assert params["$start_index"] == "200"


# ---------------------------------------------------------------------------
# buscar_empenhos
# ---------------------------------------------------------------------------


class TestBuscarEmpenhos:
    @respx.mock
    async def test_exercicio_e_referencia(self) -> None:
        route = respx.get(EMPENHOS_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "elements": [
                        {
                            "codigo_municipio": "134",
                            "codigo_orgao": "13",
                            "data_emissao_empenho": "2026-08-03T00:00:00",
                            "numero_empenho": "03080033",
                            "codigo_funcao": "04",
                            "descricao_historico_empenho": "locacao de sistema de gestao de frota",
                            "valor_empenhado": 7586.0,
                            "numero_documento_negociante": "03307395000168",
                            "nome_negociante": "I3 SOLUCOES LTDA",
                        }
                    ]
                },
            )
        )
        result = await client.buscar_empenhos(codigo_municipio=134, data_referencia=202608)
        params = route.calls[0].request.url.params
        assert params["exercicio_orcamento"] == "202600"
        assert params["data_referencia_doc"] == "202608"
        assert "codigo_orgao" not in params
        e = result.empenhos[0]
        assert e.valor_empenho == 7586.0
        assert e.nome_negociante == "I3 SOLUCOES LTDA"
        assert e.data_emissao == "2026-08-03"

    @respx.mock
    async def test_orgao_opcional_e_mes_ano(self) -> None:
        route = respx.get(EMPENHOS_URL).mock(
            return_value=httpx.Response(200, json={"elements": []})
        )
        await client.buscar_empenhos(
            codigo_municipio="57", data_referencia="08/2026", codigo_orgao="13"
        )
        params = route.calls[0].request.url.params
        assert params["codigo_municipio"] == "057"
        assert params["data_referencia_doc"] == "202608"
        assert params["codigo_orgao"] == "13"

    def test_referencia_invalida(self) -> None:
        with pytest.raises(ValueError):
            client._ano_mes("2026")

    @respx.mock
    async def test_cpf_cifrado_do_credor_vira_none(self) -> None:
        """O SIM cifra o CPF de pessoa fisica (base64): nao e documento exibivel."""
        respx.get(EMPENHOS_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "elements": [
                        {
                            "numero_empenho": "26080006",
                            "nome_negociante": "PESSOA FISICA",
                            "numero_documento_negociante": "h9pgitYKo+rsjgJeDXDBnw==",
                        }
                    ]
                },
            )
        )
        result = await client.buscar_empenhos(codigo_municipio="134", data_referencia=202608)
        assert result.empenhos[0].numero_documento_negociante is None
