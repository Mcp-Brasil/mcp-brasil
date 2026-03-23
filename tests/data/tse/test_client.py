"""Tests for the TSE HTTP client."""

import httpx
import pytest
import respx

from mcp_brasil.data.tse import client
from mcp_brasil.data.tse.constants import (
    CANDIDATURA_URL,
    ELEICAO_URL,
    PRESTADOR_URL,
    RESULTADOS_CDN_BASE,
)


class TestAnosEleitorais:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_years(self) -> None:
        respx.get(f"{ELEICAO_URL}/anos-eleitorais").mock(
            return_value=httpx.Response(200, json=[2016, 2018, 2020, 2022])
        )
        result = await client.anos_eleitorais()
        assert result == [2016, 2018, 2020, 2022]

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty(self) -> None:
        respx.get(f"{ELEICAO_URL}/anos-eleitorais").mock(return_value=httpx.Response(200, json=[]))
        result = await client.anos_eleitorais()
        assert result == []


class TestListarEleicoes:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_elections(self) -> None:
        respx.get(f"{ELEICAO_URL}/ordinarias").mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "id": 2030402020,
                        "siglaUF": None,
                        "ano": 2020,
                        "codigo": "2030402020",
                        "nomeEleicao": "Eleições Municipais 2020",
                        "tipoEleicao": "Municipal",
                        "turno": "1",
                        "tipoAbrangencia": "Municipal",
                        "dataEleicao": "15/11/2020",
                        "descricaoEleicao": "Eleições Municipais 2020 - 1º Turno",
                    }
                ],
            )
        )
        result = await client.listar_eleicoes()
        assert len(result) == 1
        assert result[0].ano == 2020
        assert result[0].tipo == "Municipal"


class TestListarEleicoesSupplementares:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed(self) -> None:
        respx.get(f"{ELEICAO_URL}/suplementares/2021/SP").mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "id": 2040402021,
                        "ano": 2021,
                        "nomeEleicao": "Eleição Suplementar Mauá",
                        "tipoEleicao": "Suplementar",
                        "dataEleicao": "06/06/2021",
                    }
                ],
            )
        )
        result = await client.listar_eleicoes_suplementares(2021, "SP")
        assert len(result) == 1
        assert result[0].ano == 2021

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty(self) -> None:
        respx.get(f"{ELEICAO_URL}/suplementares/2021/SP").mock(
            return_value=httpx.Response(200, json=[])
        )
        result = await client.listar_eleicoes_suplementares(2021, "SP")
        assert result == []


class TestListarEstadosSupplementares:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_states(self) -> None:
        respx.get(f"{ELEICAO_URL}/estados/2021/ano").mock(
            return_value=httpx.Response(
                200,
                json=[{"uf": "SP"}, {"uf": "RJ"}],
            )
        )
        result = await client.listar_estados_suplementares(2021)
        assert result == ["SP", "RJ"]

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty(self) -> None:
        respx.get(f"{ELEICAO_URL}/estados/2021/ano").mock(
            return_value=httpx.Response(200, json=[])
        )
        result = await client.listar_estados_suplementares(2021)
        assert result == []


class TestListarCargos:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_positions(self) -> None:
        url = f"{ELEICAO_URL}/listar/municipios/2030402020/35157/cargos"
        respx.get(url).mock(
            return_value=httpx.Response(
                200,
                json={
                    "unidadeEleitoralDTO": {"sigla": "BA"},
                    "cargos": [{"codigo": 11, "nome": "Prefeito", "titular": True, "contagem": 5}],
                },
            )
        )
        result = await client.listar_cargos(2030402020, 35157)
        assert len(result) == 1
        assert result[0].nome == "Prefeito"
        assert result[0].contagem == 5


class TestListarCandidatos:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_candidates(self) -> None:
        url = f"{CANDIDATURA_URL}/listar/2020/35157/2030402020/11/candidatos"
        respx.get(url).mock(
            return_value=httpx.Response(
                200,
                json={
                    "unidadeEleitoral": {},
                    "cargo": {},
                    "candidatos": [
                        {
                            "id": 50000867342,
                            "nomeUrna": "CANDIDATO TESTE",
                            "numero": 44000,
                            "partido": {"sigla": "PT"},
                            "descricaoSituacao": "Deferido",
                        }
                    ],
                },
            )
        )
        result = await client.listar_candidatos(2020, 35157, 2030402020, 11)
        assert len(result) == 1
        assert result[0].nome_urna == "CANDIDATO TESTE"
        assert result[0].partido == "PT"


class TestBuscarCandidato:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_full_candidate(self) -> None:
        url = f"{CANDIDATURA_URL}/buscar/2020/35157/2030402020/candidato/50000867342"
        respx.get(url).mock(
            return_value=httpx.Response(
                200,
                json={
                    "id": 50000867342,
                    "nomeUrna": "CANDIDATO TESTE",
                    "nomeCompleto": "Candidato Teste da Silva",
                    "numero": 44000,
                    "partido": {"sigla": "PT"},
                    "descricaoSexo": "Masculino",
                    "grauInstrucao": "Superior completo",
                    "totalDeBens": 150000.0,
                    "descricaoSituacao": "Deferido",
                    "emails": ["teste@email.com"],
                    "sites": [],
                },
            )
        )
        result = await client.buscar_candidato(2020, 35157, 2030402020, 50000867342)
        assert result is not None
        assert result.nome_completo == "Candidato Teste da Silva"
        assert result.total_bens == 150000.0

    @pytest.mark.asyncio
    @respx.mock
    async def test_not_found(self) -> None:
        url = f"{CANDIDATURA_URL}/buscar/2020/999/999/candidato/999"
        respx.get(url).mock(return_value=httpx.Response(200, json={}))
        result = await client.buscar_candidato(2020, 999, 999, 999)
        assert result is None


class TestResultadoEleicao:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_ranked_results(self) -> None:
        url = f"{CANDIDATURA_URL}/listar/2020/35157/2030402020/11/candidatos"
        respx.get(url).mock(
            return_value=httpx.Response(
                200,
                json={
                    "candidatos": [
                        {
                            "nomeUrna": "CANDIDATO B",
                            "numero": 15,
                            "partido": {"sigla": "MDB"},
                            "totalVotos": 5000,
                            "percentual": "30,00%",
                            "descricaoTotalizacao": "Não eleito",
                        },
                        {
                            "nomeUrna": "CANDIDATO A",
                            "numero": 44,
                            "partido": {"sigla": "PT"},
                            "totalVotos": 10000,
                            "percentual": "60,00%",
                            "descricaoTotalizacao": "Eleito",
                        },
                    ]
                },
            )
        )
        result = await client.resultado_eleicao(2020, 35157, 2030402020, 11)
        assert len(result) == 2
        assert result[0].nome_urna == "CANDIDATO A"
        assert result[0].total_votos == 10000
        assert result[1].nome_urna == "CANDIDATO B"
        assert result[1].total_votos == 5000

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty(self) -> None:
        url = f"{CANDIDATURA_URL}/listar/2020/999/999/11/candidatos"
        respx.get(url).mock(return_value=httpx.Response(200, json={"candidatos": []}))
        result = await client.resultado_eleicao(2020, 999, 999, 11)
        assert result == []


class TestBuscarCandidatoEnriched:
    @pytest.mark.asyncio
    @respx.mock
    async def test_includes_totalizacao(self) -> None:
        url = f"{CANDIDATURA_URL}/buscar/2020/35157/2030402020/candidato/123"
        respx.get(url).mock(
            return_value=httpx.Response(
                200,
                json={
                    "id": 123,
                    "nomeUrna": "CANDIDATO X",
                    "partido": {"sigla": "PT"},
                    "descricaoTotalizacao": "Eleito",
                    "totalVotos": 25000,
                },
            )
        )
        result = await client.buscar_candidato(2020, 35157, 2030402020, 123)
        assert result is not None
        assert result.descricao_totalizacao == "Eleito"
        assert result.total_votos == 25000


class TestConsultarPrestacaoContas:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_accounts(self) -> None:
        url = f"{PRESTADOR_URL}/consulta/2030402020/2020/35157/11/90/90/50000867342"
        respx.get(url).mock(
            return_value=httpx.Response(
                200,
                json={
                    "idCandidato": "50000867342",
                    "nomeCandidato": "Candidato Teste",
                    "siglaPartido": "PT",
                    "cnpj": "12.345.678/0001-99",
                    "dadosConsolidados": {
                        "totalRecebido": 100000,
                        "totalReceitaPF": 50000,
                        "totalReceitaPJ": 30000,
                        "totalPartidos": 20000,
                        "totalDoacaoFcc": 0,
                    },
                    "despesas": {
                        "totalDespesasPagas": 80000,
                        "valorLimiteDeGastos": 200000,
                    },
                },
            )
        )
        result = await client.consultar_prestacao_contas(2030402020, 2020, 35157, 11, 50000867342)
        assert result is not None
        assert result.total_recebido == 100000
        assert result.total_despesas == 80000


class TestParserEdgeCases:
    def test_eleicao_empty(self) -> None:
        result = client._parse_eleicao({})
        assert result.id is None
        assert result.ano is None

    def test_cargo_empty(self) -> None:
        result = client._parse_cargo({})
        assert result.codigo is None
        assert result.nome is None

    def test_candidato_resumo_partido_string(self) -> None:
        result = client._parse_candidato_resumo({"partido": "PT"})
        assert result.partido == "PT"

    def test_candidato_resumo_partido_dict(self) -> None:
        result = client._parse_candidato_resumo({"partido": {"sigla": "PL"}})
        assert result.partido == "PL"

    def test_resultado_candidato_empty(self) -> None:
        result = client._parse_resultado_candidato({})
        assert result.nome_urna is None
        assert result.total_votos is None
        assert result.partido is None

    def test_resultado_candidato_partido_string(self) -> None:
        result = client._parse_resultado_candidato({"partido": "PSOL"})
        assert result.partido == "PSOL"

    def test_presta_contas_empty(self) -> None:
        result = client._parse_presta_contas({})
        assert result.total_recebido is None
        assert result.total_despesas is None


# --- CDN de Resultados ---

CDN_RESULT_JSON = {
    "ele": 544,
    "tpabr": "uf",
    "cdabr": "SP",
    "dt": "02/10/2022",
    "s": 101073,
    "e": 34684927,
    "c": 27189714,
    "a": 7495213,
    "pst": "100.00",
    "cand": [
        {
            "seq": "0001",
            "n": "22",
            "nm": "JAIR BOLSONARO",
            "cc": "PL",
            "nv": "BRAGA NETTO",
            "e": "s",
            "st": "2 turno",
            "dvt": "Válido",
            "vap": "12239989",
            "pvap": "47.71",
        },
        {
            "seq": "0002",
            "n": "13",
            "nm": "LULA",
            "cc": "PT",
            "nv": "GERALDO ALCKMIN",
            "e": "s",
            "st": "2 turno",
            "dvt": "Válido",
            "vap": "10490032",
            "pvap": "40.89",
        },
    ],
}


class TestResolveEleicao:
    def test_known_election(self) -> None:
        ciclo, padded, unpadded = client._resolve_eleicao(2022, 1)
        assert ciclo == "ele2022"
        assert padded == "000544"
        assert unpadded == "544"

    def test_known_election_2024(self) -> None:
        ciclo, padded, unpadded = client._resolve_eleicao(2024, 1)
        assert ciclo == "ele2024"
        assert padded == "000619"
        assert unpadded == "619"

    def test_unknown_election_raises(self) -> None:
        with pytest.raises(ValueError, match="não mapeada"):
            client._resolve_eleicao(1990, 1)


class TestResolveCargo:
    def test_known_cargo(self) -> None:
        assert client._resolve_cargo("presidente") == "0001"
        assert client._resolve_cargo("governador") == "0003"
        assert client._resolve_cargo("prefeito") == "0011"

    def test_unknown_cargo_raises(self) -> None:
        with pytest.raises(ValueError, match="não mapeado"):
            client._resolve_cargo("papa")


class TestParseResultadoCDN:
    def test_parses_candidate(self) -> None:
        raw = CDN_RESULT_JSON["cand"][0]
        result = client._parse_resultado_cdn(raw)
        assert result.nome == "JAIR BOLSONARO"
        assert result.numero == "22"
        assert result.votos == 12239989
        assert result.percentual == "47.71"
        assert result.eleito is True

    def test_parses_not_elected(self) -> None:
        raw = {"nm": "TESTE", "n": "99", "e": "n", "vap": "100"}
        result = client._parse_resultado_cdn(raw)
        assert result.eleito is False
        assert result.votos == 100


class TestParseResultadoRegiao:
    def test_parses_region(self) -> None:
        result = client._parse_resultado_regiao(CDN_RESULT_JSON)
        assert result.uf == "SP"
        assert result.total_secoes == 101073
        assert result.total_eleitores == 34684927
        assert result.pct_apurado == "100.00"
        assert len(result.candidatos) == 2
        # Should be sorted by votes (descending)
        assert result.candidatos[0].nome == "JAIR BOLSONARO"
        assert result.candidatos[1].nome == "LULA"


class TestResultadoSimplificado:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_result(self) -> None:
        url = f"{RESULTADOS_CDN_BASE}/ele2022/544/dados-simplificados/sp/sp-c0001-e000544-r.json"
        respx.get(url).mock(return_value=httpx.Response(200, json=CDN_RESULT_JSON))
        result = await client.resultado_simplificado(2022, "presidente", "sp", 1)
        assert result is not None
        assert result.uf == "SP"
        assert len(result.candidatos) == 2

    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_none_on_404(self) -> None:
        url = f"{RESULTADOS_CDN_BASE}/ele2022/544/dados-simplificados/xx/xx-c0001-e000544-r.json"
        respx.get(url).mock(return_value=httpx.Response(404))
        result = await client.resultado_simplificado(2022, "presidente", "xx", 1)
        assert result is None

    @pytest.mark.asyncio
    @respx.mock
    async def test_nacional(self) -> None:
        url = f"{RESULTADOS_CDN_BASE}/ele2022/544/dados-simplificados/br/br-c0001-e000544-r.json"
        br_json = {**CDN_RESULT_JSON, "cdabr": "BR", "tpabr": "br"}
        respx.get(url).mock(return_value=httpx.Response(200, json=br_json))
        result = await client.resultado_simplificado(2022, "presidente", "br", 1)
        assert result is not None
        assert result.uf == "BR"


class TestResultadoTodosEstados:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_list(self) -> None:
        # Mock SP and RJ only, rest 404
        sp_url = (
            f"{RESULTADOS_CDN_BASE}/ele2022/544/dados-simplificados/sp/sp-c0001-e000544-r.json"
        )
        rj_url = (
            f"{RESULTADOS_CDN_BASE}/ele2022/544/dados-simplificados/rj/rj-c0001-e000544-r.json"
        )
        sp_json = {**CDN_RESULT_JSON, "cdabr": "SP"}
        rj_json = {**CDN_RESULT_JSON, "cdabr": "RJ"}
        respx.get(sp_url).mock(return_value=httpx.Response(200, json=sp_json))
        respx.get(rj_url).mock(return_value=httpx.Response(200, json=rj_json))
        # All other UFs get 404
        respx.get(url__regex=r".*/dados-simplificados/(?!sp|rj).*").mock(
            return_value=httpx.Response(404)
        )
        result = await client.resultado_todos_estados(2022, "presidente", 1)
        assert len(result) == 2
        ufs = {r.uf for r in result}
        assert "SP" in ufs
        assert "RJ" in ufs


# --- Município (CDN formato unificado -u.json) ---

CDN_UNIFIED_JSON = {
    "cdabr": "71072",
    "tpabr": "mu",
    "dt": "06/10/2024",
    "s": {"ts": 15000, "pst": "100.00"},
    "e": {"te": 8000000, "c": 6500000, "a": 1500000},
    "carg": [
        {
            "cd": "0011",
            "agr": [
                {
                    "par": [
                        {
                            "cand": [
                                {
                                    "seq": "0001",
                                    "n": "15",
                                    "nm": "CANDIDATO PREFEITO A",
                                    "nmu": "CAND A",
                                    "cc": "MDB",
                                    "e": "s",
                                    "st": "Eleito",
                                    "dvt": "Válido",
                                    "vap": "3500000",
                                    "pvap": "53.85",
                                },
                            ]
                        },
                        {
                            "cand": [
                                {
                                    "seq": "0002",
                                    "n": "13",
                                    "nm": "CANDIDATO PREFEITO B",
                                    "nmu": "CAND B",
                                    "cc": "PT",
                                    "e": "n",
                                    "st": "Não eleito",
                                    "dvt": "Válido",
                                    "vap": "3000000",
                                    "pvap": "46.15",
                                },
                            ]
                        },
                    ]
                }
            ],
        }
    ],
}

CDN_MUN_CONFIG_JSON = {
    "dg": "06/10/2024",
    "hg": "20:00:00",
    "abr": [
        {
            "cd": "SP",
            "mu": [
                {"cd": "71072", "cdi": "3550308", "nm": "SÃO PAULO", "c": "S"},
                {"cd": "62910", "cdi": "3509502", "nm": "CAMPINAS", "c": "N"},
            ],
        },
        {
            "cd": "RJ",
            "mu": [
                {"cd": "60011", "cdi": "3304557", "nm": "RIO DE JANEIRO", "c": "S"},
            ],
        },
    ],
}


class TestParseResultadoUnificado:
    def test_parses_unified_format(self) -> None:
        result = client._parse_resultado_unificado(CDN_UNIFIED_JSON)
        assert result.codigo == "71072"
        assert result.tipo == "mu"
        assert result.total_secoes == 15000
        assert result.pct_apurado == "100.00"
        assert result.total_eleitores == 8000000
        assert result.total_comparecimento == 6500000
        assert result.total_abstencoes == 1500000
        assert len(result.candidatos) == 2
        # Sorted by votes descending
        assert result.candidatos[0].nome == "CANDIDATO PREFEITO A"
        assert result.candidatos[0].votos == 3500000
        assert result.candidatos[1].nome == "CANDIDATO PREFEITO B"
        assert result.candidatos[1].votos == 3000000

    def test_empty_carg(self) -> None:
        result = client._parse_resultado_unificado({"s": {}, "e": {}})
        assert result.candidatos == []
        assert result.total_secoes is None


class TestListarMunicipiosEleitorais:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_municipios_for_uf(self) -> None:
        url = f"{RESULTADOS_CDN_BASE}/ele2024/619/config/mun-e000619-cm.json"
        respx.get(url).mock(return_value=httpx.Response(200, json=CDN_MUN_CONFIG_JSON))
        result = await client.listar_municipios_eleitorais(2024, "SP", 1)
        assert len(result) == 2
        assert result[0].codigo_tse == "71072"
        assert result[0].codigo_ibge == "3550308"
        assert result[0].nome == "SÃO PAULO"
        assert result[0].capital is True
        assert result[0].uf == "SP"
        assert result[1].codigo_tse == "62910"
        assert result[1].capital is False

    @pytest.mark.asyncio
    @respx.mock
    async def test_case_insensitive_uf(self) -> None:
        url = f"{RESULTADOS_CDN_BASE}/ele2024/619/config/mun-e000619-cm.json"
        respx.get(url).mock(return_value=httpx.Response(200, json=CDN_MUN_CONFIG_JSON))
        result = await client.listar_municipios_eleitorais(2024, "sp", 1)
        assert len(result) == 2

    @pytest.mark.asyncio
    @respx.mock
    async def test_uf_not_found(self) -> None:
        url = f"{RESULTADOS_CDN_BASE}/ele2024/619/config/mun-e000619-cm.json"
        respx.get(url).mock(return_value=httpx.Response(200, json=CDN_MUN_CONFIG_JSON))
        result = await client.listar_municipios_eleitorais(2024, "AM", 1)
        assert result == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_empty_on_404(self) -> None:
        url = f"{RESULTADOS_CDN_BASE}/ele2024/619/config/mun-e000619-cm.json"
        respx.get(url).mock(return_value=httpx.Response(404))
        result = await client.listar_municipios_eleitorais(2024, "SP", 1)
        assert result == []


class TestResultadoMunicipio:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_result(self) -> None:
        url = f"{RESULTADOS_CDN_BASE}/ele2024/619/dados/sp/sp71072-c0011-e000619-u.json"
        respx.get(url).mock(return_value=httpx.Response(200, json=CDN_UNIFIED_JSON))
        result = await client.resultado_municipio(2024, "prefeito", "SP", "71072", 1)
        assert result is not None
        assert result.codigo == "71072"
        assert len(result.candidatos) == 2
        assert result.candidatos[0].votos == 3500000

    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_none_on_404(self) -> None:
        url = f"{RESULTADOS_CDN_BASE}/ele2024/619/dados/sp/sp99999-c0011-e000619-u.json"
        respx.get(url).mock(return_value=httpx.Response(404))
        result = await client.resultado_municipio(2024, "prefeito", "SP", "99999", 1)
        assert result is None
