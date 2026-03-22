"""Tests for the Câmara tool functions.

Tools are tested by mocking client functions (never HTTP).
"""

from unittest.mock import AsyncMock, patch

import pytest

from mcp_brasil.camara import tools
from mcp_brasil.camara.constants import DEFAULT_PAGE_SIZE
from mcp_brasil.camara.schemas import (
    Deputado,
    DespesaDeputado,
    Evento,
    FrenteParlamentar,
    Orgao,
    Proposicao,
    Tramitacao,
    Votacao,
    VotoNominal,
)

MODULE = "mcp_brasil.camara.client"


# ---------------------------------------------------------------------------
# listar_deputados
# ---------------------------------------------------------------------------


class TestListarDeputados:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Deputado(
                id=204554,
                nome="Fulano da Silva",
                sigla_partido="PT",
                sigla_uf="SP",
                email="dep.fulano@camara.leg.br",
            )
        ]
        with patch(f"{MODULE}.listar_deputados", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.listar_deputados()
        assert "Fulano da Silva" in result
        assert "PT" in result
        assert "SP" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.listar_deputados", new_callable=AsyncMock, return_value=[]):
            result = await tools.listar_deputados()
        assert "Nenhum deputado" in result


# ---------------------------------------------------------------------------
# buscar_deputado
# ---------------------------------------------------------------------------


class TestBuscarDeputado:
    @pytest.mark.asyncio
    async def test_formats_profile(self) -> None:
        mock_data = Deputado(
            id=204554,
            nome="Fulano da Silva",
            sigla_partido="PT",
            sigla_uf="SP",
            email="dep.fulano@camara.leg.br",
            foto="https://foto.example.com/204554.jpg",
            legislatura=57,
        )
        with patch(f"{MODULE}.obter_deputado", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.buscar_deputado(204554)
        assert "Fulano da Silva" in result
        assert "PT" in result
        assert "foto" in result.lower()

    @pytest.mark.asyncio
    async def test_not_found(self) -> None:
        with patch(f"{MODULE}.obter_deputado", new_callable=AsyncMock, return_value=None):
            result = await tools.buscar_deputado(999999)
        assert "não encontrado" in result


# ---------------------------------------------------------------------------
# buscar_proposicao
# ---------------------------------------------------------------------------


class TestBuscarProposicao:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Proposicao(
                sigla_tipo="PL",
                numero=1234,
                ano=2024,
                ementa="Dispõe sobre educação",
                data_apresentacao="2024-03-15",
                situacao="Tramitando",
            )
        ]
        with patch(f"{MODULE}.buscar_proposicoes", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.buscar_proposicao(sigla_tipo="PL", ano=2024)
        assert "PL" in result
        assert "1234" in result
        assert "educação" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.buscar_proposicoes", new_callable=AsyncMock, return_value=[]):
            result = await tools.buscar_proposicao()
        assert "Nenhuma proposição" in result


# ---------------------------------------------------------------------------
# consultar_tramitacao
# ---------------------------------------------------------------------------


class TestConsultarTramitacao:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Tramitacao(
                data="2024-03-15",
                descricao="Apresentação",
                orgao="PLEN",
                situacao="Tramitando",
            )
        ]
        with patch(f"{MODULE}.obter_tramitacoes", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.consultar_tramitacao(2300001)
        assert "Apresentação" in result
        assert "PLEN" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.obter_tramitacoes", new_callable=AsyncMock, return_value=[]):
            result = await tools.consultar_tramitacao(999999)
        assert "Nenhuma tramitação" in result


# ---------------------------------------------------------------------------
# buscar_votacao
# ---------------------------------------------------------------------------


class TestBuscarVotacao:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Votacao(
                id="vot-123",
                data="2024-06-01",
                descricao="Votação em turno único",
                aprovacao=True,
            )
        ]
        with patch(f"{MODULE}.listar_votacoes", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.buscar_votacao()
        assert "vot-123" in result
        assert "Sim" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.listar_votacoes", new_callable=AsyncMock, return_value=[]):
            result = await tools.buscar_votacao()
        assert "Nenhuma votação" in result


# ---------------------------------------------------------------------------
# votos_nominais
# ---------------------------------------------------------------------------


class TestVotosNominais:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            VotoNominal(
                deputado_nome="Fulano",
                partido="PT",
                uf="SP",
                voto="Sim",
            )
        ]
        with patch(f"{MODULE}.obter_votos", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.votos_nominais("vot-123")
        assert "Fulano" in result
        assert "PT" in result
        assert "Sim" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.obter_votos", new_callable=AsyncMock, return_value=[]):
            result = await tools.votos_nominais("vot-999")
        assert "Nenhum voto" in result


# ---------------------------------------------------------------------------
# despesas_deputado
# ---------------------------------------------------------------------------


class TestDespesasDeputado:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            DespesaDeputado(
                tipo_despesa="PASSAGENS AÉREAS",
                fornecedor="GOL",
                valor_liquido=1200.0,
                data_documento="2024-03-15",
                mes=3,
                ano=2024,
            )
        ]
        with patch(f"{MODULE}.listar_despesas", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.despesas_deputado(204554)
        assert "PASSAGENS" in result
        assert "R$ 1.200,00" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.listar_despesas", new_callable=AsyncMock, return_value=[]):
            result = await tools.despesas_deputado(999999)
        assert "Nenhuma despesa" in result


# ---------------------------------------------------------------------------
# agenda_legislativa
# ---------------------------------------------------------------------------


class TestAgendaLegislativa:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Evento(
                data_inicio="2024-03-15T09:00:00",
                titulo="Sessão Deliberativa",
                descricao="Ordem do Dia",
                situacao="Encerrada",
                orgaos="PLEN",
            )
        ]
        with patch(f"{MODULE}.listar_eventos", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.agenda_legislativa()
        assert "Sessão Deliberativa" in result
        assert "Encerrada" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.listar_eventos", new_callable=AsyncMock, return_value=[]):
            result = await tools.agenda_legislativa()
        assert "Nenhum evento" in result


# ---------------------------------------------------------------------------
# buscar_comissoes
# ---------------------------------------------------------------------------


class TestBuscarComissoes:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Orgao(
                sigla="CCJC",
                nome="Comissão de Constituição e Justiça e de Cidadania",
                tipo="Comissão Permanente",
                situacao="Ativa",
            )
        ]
        with patch(f"{MODULE}.listar_orgaos", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.buscar_comissoes()
        assert "CCJC" in result
        assert "Comissão Permanente" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.listar_orgaos", new_callable=AsyncMock, return_value=[]):
            result = await tools.buscar_comissoes()
        assert "Nenhuma comissão" in result


# ---------------------------------------------------------------------------
# frentes_parlamentares
# ---------------------------------------------------------------------------


class TestFrentesParlamentares:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            FrenteParlamentar(
                id=55001,
                titulo="Frente Parlamentar da Educação",
                legislatura=57,
                coordenador="Dep. Coordenador",
            )
        ]
        with patch(f"{MODULE}.listar_frentes", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.frentes_parlamentares()
        assert "Educação" in result
        assert "Dep. Coordenador" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.listar_frentes", new_callable=AsyncMock, return_value=[]):
            result = await tools.frentes_parlamentares()
        assert "Nenhuma frente" in result


# ---------------------------------------------------------------------------
# Pagination hints
# ---------------------------------------------------------------------------


class TestPaginationHints:
    @pytest.mark.asyncio
    async def test_shows_next_page_hint(self) -> None:
        data = [Deputado(id=i, nome=f"Dep {i}") for i in range(DEFAULT_PAGE_SIZE)]
        with patch(f"{MODULE}.listar_deputados", new_callable=AsyncMock, return_value=data):
            result = await tools.listar_deputados(pagina=1)
        assert "pagina=2" in result

    @pytest.mark.asyncio
    async def test_no_hint_below_page_size(self) -> None:
        data = [Deputado(id=1, nome="Dep 1")]
        with patch(f"{MODULE}.listar_deputados", new_callable=AsyncMock, return_value=data):
            result = await tools.listar_deputados(pagina=1)
        assert "pagina=" not in result
        assert "Última página" not in result

    @pytest.mark.asyncio
    async def test_last_page_hint(self) -> None:
        data = [Deputado(id=1, nome="Dep 1")]
        with patch(f"{MODULE}.listar_deputados", new_callable=AsyncMock, return_value=data):
            result = await tools.listar_deputados(pagina=2)
        assert "Última página" in result
