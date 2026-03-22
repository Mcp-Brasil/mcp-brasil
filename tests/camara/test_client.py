"""Tests for the Câmara HTTP client."""

import logging

import httpx
import pytest
import respx

from mcp_brasil.camara import client
from mcp_brasil.camara.constants import (
    DEPUTADOS_URL,
    EVENTOS_URL,
    FRENTES_URL,
    ORGAOS_URL,
    PROPOSICOES_URL,
    VOTACOES_URL,
)

# ---------------------------------------------------------------------------
# _get envelope extraction
# ---------------------------------------------------------------------------


class TestGetEnvelope:
    @pytest.mark.asyncio
    @respx.mock
    async def test_extracts_dados_field(self) -> None:
        respx.get(DEPUTADOS_URL).mock(
            return_value=httpx.Response(
                200, json={"dados": [{"id": 1, "nome": "Dep. Teste"}], "links": []}
            )
        )
        data = await client._get(DEPUTADOS_URL)
        assert isinstance(data, list)
        assert data[0]["nome"] == "Dep. Teste"

    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_raw_when_no_dados(self) -> None:
        respx.get(DEPUTADOS_URL).mock(
            return_value=httpx.Response(200, json={"id": 1, "nome": "Dep. Detalhe"})
        )
        data = await client._get(DEPUTADOS_URL)
        assert isinstance(data, dict)
        assert data["nome"] == "Dep. Detalhe"


# ---------------------------------------------------------------------------
# listar_deputados
# ---------------------------------------------------------------------------


class TestListarDeputados:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_deputies(self) -> None:
        respx.get(DEPUTADOS_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "dados": [
                        {
                            "id": 204554,
                            "nome": "Fulano da Silva",
                            "siglaPartido": "PT",
                            "siglaUf": "SP",
                            "email": "dep.fulano@camara.leg.br",
                            "urlFoto": "https://foto.example.com/204554.jpg",
                            "idLegislatura": 57,
                        }
                    ],
                    "links": [],
                },
            )
        )
        result = await client.listar_deputados()
        assert len(result) == 1
        assert result[0].nome == "Fulano da Silva"
        assert result[0].sigla_partido == "PT"
        assert result[0].sigla_uf == "SP"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(DEPUTADOS_URL).mock(
            return_value=httpx.Response(200, json={"dados": [], "links": []})
        )
        result = await client.listar_deputados()
        assert result == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_non_list_response(self, caplog: pytest.LogCaptureFixture) -> None:
        respx.get(DEPUTADOS_URL).mock(
            return_value=httpx.Response(200, json={"dados": "invalid", "links": []})
        )
        with caplog.at_level(logging.WARNING, logger="mcp_brasil.camara.client"):
            result = await client.listar_deputados()
        assert result == []
        assert "Resposta inesperada" in caplog.text


# ---------------------------------------------------------------------------
# obter_deputado
# ---------------------------------------------------------------------------


class TestObterDeputado:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_deputy_detail(self) -> None:
        respx.get(f"{DEPUTADOS_URL}/204554").mock(
            return_value=httpx.Response(
                200,
                json={
                    "dados": {
                        "id": 204554,
                        "nomeCivil": "Fulano da Silva Santos",
                        "siglaPartido": "PT",
                        "siglaUf": "SP",
                        "email": "dep.fulano@camara.leg.br",
                    },
                    "links": [],
                },
            )
        )
        result = await client.obter_deputado(204554)
        assert result is not None
        assert result.nome == "Fulano da Silva Santos"


# ---------------------------------------------------------------------------
# buscar_proposicoes
# ---------------------------------------------------------------------------


class TestBuscarProposicoes:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_proposals(self) -> None:
        respx.get(PROPOSICOES_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "dados": [
                        {
                            "id": 2300001,
                            "siglaTipo": "PL",
                            "numero": 1234,
                            "ano": 2024,
                            "ementa": "Dispõe sobre educação",
                            "dataApresentacao": "2024-03-15",
                        }
                    ],
                    "links": [],
                },
            )
        )
        result = await client.buscar_proposicoes(sigla_tipo="PL", ano=2024)
        assert len(result) == 1
        assert result[0].sigla_tipo == "PL"
        assert result[0].numero == 1234

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty(self) -> None:
        respx.get(PROPOSICOES_URL).mock(
            return_value=httpx.Response(200, json={"dados": [], "links": []})
        )
        result = await client.buscar_proposicoes()
        assert result == []


# ---------------------------------------------------------------------------
# obter_tramitacoes
# ---------------------------------------------------------------------------


class TestObterTramitacoes:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_tramitacoes(self) -> None:
        respx.get(f"{PROPOSICOES_URL}/2300001/tramitacoes").mock(
            return_value=httpx.Response(
                200,
                json={
                    "dados": [
                        {
                            "dataHora": "2024-03-15T10:00:00",
                            "descricaoTramitacao": "Apresentação",
                            "orgao": {"sigla": "PLEN"},
                            "descricaoSituacao": "Tramitando",
                            "despacho": "Encaminhe-se à CCJC",
                        }
                    ],
                    "links": [],
                },
            )
        )
        result = await client.obter_tramitacoes(2300001)
        assert len(result) == 1
        assert result[0].orgao == "PLEN"
        assert result[0].despacho == "Encaminhe-se à CCJC"


# ---------------------------------------------------------------------------
# listar_votacoes
# ---------------------------------------------------------------------------


class TestListarVotacoes:
    @pytest.mark.asyncio
    @respx.mock
    async def test_by_proposicao(self) -> None:
        respx.get(f"{PROPOSICOES_URL}/2300001/votacoes").mock(
            return_value=httpx.Response(
                200,
                json={
                    "dados": [
                        {
                            "id": "vot-123",
                            "dataHoraRegistro": "2024-06-01T14:00:00",
                            "descricao": "Votação em turno único",
                            "aprovacao": True,
                        }
                    ],
                    "links": [],
                },
            )
        )
        result = await client.listar_votacoes(proposicao_id=2300001)
        assert len(result) == 1
        assert result[0].id == "vot-123"
        assert result[0].aprovacao is True

    @pytest.mark.asyncio
    @respx.mock
    async def test_by_period(self) -> None:
        respx.get(VOTACOES_URL).mock(
            return_value=httpx.Response(200, json={"dados": [], "links": []})
        )
        result = await client.listar_votacoes(data_inicio="2024-01-01", data_fim="2024-12-31")
        assert result == []


# ---------------------------------------------------------------------------
# obter_votos
# ---------------------------------------------------------------------------


class TestObterVotos:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_votes(self) -> None:
        respx.get(f"{VOTACOES_URL}/vot-123/votos").mock(
            return_value=httpx.Response(
                200,
                json={
                    "dados": [
                        {
                            "deputado_": {
                                "id": 204554,
                                "nome": "Fulano",
                                "siglaPartido": "PT",
                                "siglaUf": "SP",
                            },
                            "tipoVoto": "Sim",
                        }
                    ],
                    "links": [],
                },
            )
        )
        result = await client.obter_votos("vot-123")
        assert len(result) == 1
        assert result[0].deputado_nome == "Fulano"
        assert result[0].voto == "Sim"


# ---------------------------------------------------------------------------
# listar_despesas
# ---------------------------------------------------------------------------


class TestListarDespesas:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_expenses(self) -> None:
        respx.get(f"{DEPUTADOS_URL}/204554/despesas").mock(
            return_value=httpx.Response(
                200,
                json={
                    "dados": [
                        {
                            "tipoDespesa": "PASSAGENS AÉREAS",
                            "nomeFornecedor": "GOL",
                            "cnpjCpfFornecedor": "07575651000159",
                            "valorDocumento": 1500.0,
                            "valorLiquido": 1200.0,
                            "dataDocumento": "2024-03-15",
                            "mes": 3,
                            "ano": 2024,
                        }
                    ],
                    "links": [],
                },
            )
        )
        result = await client.listar_despesas(204554)
        assert len(result) == 1
        assert result[0].tipo_despesa == "PASSAGENS AÉREAS"
        assert result[0].valor_liquido == 1200.0


# ---------------------------------------------------------------------------
# listar_eventos
# ---------------------------------------------------------------------------


class TestListarEventos:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_events(self) -> None:
        respx.get(EVENTOS_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "dados": [
                        {
                            "id": 70001,
                            "dataHoraInicio": "2024-03-15T09:00:00",
                            "dataHoraFim": "2024-03-15T12:00:00",
                            "descricaoTipo": "Sessão Deliberativa",
                            "descricao": "Ordem do Dia",
                            "situacao": "Encerrada",
                            "orgaos": [{"sigla": "PLEN"}],
                        }
                    ],
                    "links": [],
                },
            )
        )
        result = await client.listar_eventos()
        assert len(result) == 1
        assert result[0].titulo == "Sessão Deliberativa"
        assert result[0].orgaos == "PLEN"


# ---------------------------------------------------------------------------
# listar_orgaos
# ---------------------------------------------------------------------------


class TestListarOrgaos:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_organs(self) -> None:
        respx.get(ORGAOS_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "dados": [
                        {
                            "id": 2001,
                            "sigla": "CCJC",
                            "nome": "Comissão de Constituição e Justiça e de Cidadania",
                            "tipoOrgao": "Comissão Permanente",
                            "situacao": "Ativa",
                        }
                    ],
                    "links": [],
                },
            )
        )
        result = await client.listar_orgaos()
        assert len(result) == 1
        assert result[0].sigla == "CCJC"
        assert result[0].tipo == "Comissão Permanente"


# ---------------------------------------------------------------------------
# listar_frentes
# ---------------------------------------------------------------------------


class TestListarFrentes:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_fronts(self) -> None:
        respx.get(FRENTES_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "dados": [
                        {
                            "id": 55001,
                            "titulo": "Frente Parlamentar da Educação",
                            "idLegislatura": 57,
                            "coordenador": {"nome": "Dep. Coordenador"},
                            "situacao": "Em exercício",
                        }
                    ],
                    "links": [],
                },
            )
        )
        result = await client.listar_frentes()
        assert len(result) == 1
        assert result[0].titulo == "Frente Parlamentar da Educação"
        assert result[0].coordenador == "Dep. Coordenador"


# ---------------------------------------------------------------------------
# Parser edge cases
# ---------------------------------------------------------------------------


class TestParserEdgeCases:
    def test_deputado_empty(self) -> None:
        result = client._parse_deputado({})
        assert result.nome is None
        assert result.sigla_partido is None

    def test_proposicao_empty(self) -> None:
        result = client._parse_proposicao({})
        assert result.sigla_tipo is None
        assert result.situacao is None

    def test_tramitacao_empty(self) -> None:
        result = client._parse_tramitacao({})
        assert result.data is None
        assert result.orgao is None

    def test_tramitacao_orgao_not_dict(self) -> None:
        result = client._parse_tramitacao({"orgao": "PLEN"})
        assert result.orgao is None

    def test_votacao_empty(self) -> None:
        result = client._parse_votacao({})
        assert result.id is None
        assert result.proposicao_id is None

    def test_voto_empty(self) -> None:
        result = client._parse_voto({})
        assert result.deputado_nome is None
        assert result.voto is None

    def test_despesa_empty(self) -> None:
        result = client._parse_despesa({})
        assert result.tipo_despesa is None
        assert result.valor_liquido is None

    def test_evento_empty(self) -> None:
        result = client._parse_evento({})
        assert result.titulo is None
        assert result.orgaos is None

    def test_orgao_empty(self) -> None:
        result = client._parse_orgao({})
        assert result.sigla is None

    def test_frente_empty(self) -> None:
        result = client._parse_frente({})
        assert result.titulo is None
        assert result.coordenador is None

    def test_frente_coordenador_not_dict(self) -> None:
        result = client._parse_frente({"coordenador": "Dep. Nome"})
        assert result.coordenador is None
