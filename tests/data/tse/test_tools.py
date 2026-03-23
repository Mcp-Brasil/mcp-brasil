"""Tests for the TSE tool functions.

Tools are tested by mocking client functions (never HTTP).
"""

from unittest.mock import AsyncMock, patch

import pytest

from mcp_brasil.data.tse import tools
from mcp_brasil.data.tse.schemas import (
    Candidato,
    CandidatoResumo,
    Cargo,
    Eleicao,
    MunicipioEleitoral,
    PrestaContas,
    ResultadoCandidato,
    ResultadoCDN,
    ResultadoRegiao,
)

MODULE = "mcp_brasil.data.tse.client"


class TestAnosEleitorais:
    @pytest.mark.asyncio
    async def test_formats_years(self) -> None:
        with patch(f"{MODULE}.anos_eleitorais", new_callable=AsyncMock, return_value=[2020, 2022]):
            result = await tools.anos_eleitorais()
        assert "2020" in result
        assert "2022" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.anos_eleitorais", new_callable=AsyncMock, return_value=[]):
            result = await tools.anos_eleitorais()
        assert "Nenhum ano" in result


class TestListarEleicoes:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Eleicao(
                id=2030402020,
                ano=2020,
                nome="Eleições Municipais 2020",
                tipo="Municipal",
                tipo_abrangencia="Municipal",
                data_eleicao="15/11/2020",
            )
        ]
        with patch(f"{MODULE}.listar_eleicoes", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.listar_eleicoes()
        assert "2020" in result
        assert "Municipal" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.listar_eleicoes", new_callable=AsyncMock, return_value=[]):
            result = await tools.listar_eleicoes()
        assert "Nenhuma eleição" in result


class TestListarEleicoesSupplementares:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Eleicao(
                id=2040402021,
                ano=2021,
                nome="Eleição Suplementar Mauá",
                tipo="Suplementar",
                data_eleicao="06/06/2021",
            )
        ]
        with patch(
            f"{MODULE}.listar_eleicoes_suplementares",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.listar_eleicoes_suplementares(2021, "SP")
        assert "Suplementar" in result
        assert "SP" in result
        assert "2021" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(
            f"{MODULE}.listar_eleicoes_suplementares",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.listar_eleicoes_suplementares(2021, "SP")
        assert "Nenhuma eleição suplementar" in result


class TestListarEstadosSupplementares:
    @pytest.mark.asyncio
    async def test_formats_states(self) -> None:
        with patch(
            f"{MODULE}.listar_estados_suplementares",
            new_callable=AsyncMock,
            return_value=["SP", "RJ", "MG"],
        ):
            result = await tools.listar_estados_suplementares(2021)
        assert "SP" in result
        assert "RJ" in result
        assert "MG" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(
            f"{MODULE}.listar_estados_suplementares",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.listar_estados_suplementares(2021)
        assert "Nenhum estado" in result


class TestListarCargos:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Cargo(codigo=11, nome="Prefeito", titular=True, contagem=5),
            Cargo(codigo=13, nome="Vereador", titular=True, contagem=200),
        ]
        with patch(f"{MODULE}.listar_cargos", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.listar_cargos(2030402020, 35157)
        assert "Prefeito" in result
        assert "Vereador" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.listar_cargos", new_callable=AsyncMock, return_value=[]):
            result = await tools.listar_cargos(999, 999)
        assert "Nenhum cargo" in result


class TestListarCandidatos:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            CandidatoResumo(
                id=50000867342,
                nome_urna="CANDIDATO TESTE",
                numero=44000,
                partido="PT",
                situacao="Deferido",
            )
        ]
        with patch(f"{MODULE}.listar_candidatos", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.listar_candidatos(2020, 35157, 2030402020, 11)
        assert "CANDIDATO TESTE" in result
        assert "PT" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.listar_candidatos", new_callable=AsyncMock, return_value=[]):
            result = await tools.listar_candidatos(2020, 999, 999, 999)
        assert "Nenhum candidato" in result


class TestBuscarCandidato:
    @pytest.mark.asyncio
    async def test_formats_profile(self) -> None:
        mock_data = Candidato(
            id=50000867342,
            nome_urna="CANDIDATO TESTE",
            nome_completo="Candidato Teste da Silva",
            numero=44000,
            partido="PT",
            situacao="Deferido",
            sexo="Masculino",
            cor_raca="Parda",
            grau_instrucao="Superior completo",
            total_bens=150000.0,
            gasto_campanha=50000.0,
        )
        with patch(f"{MODULE}.buscar_candidato", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.buscar_candidato(2020, 35157, 2030402020, 50000867342)
        assert "CANDIDATO TESTE" in result
        assert "Candidato Teste da Silva" in result
        assert "R$" in result
        assert "Superior completo" in result

    @pytest.mark.asyncio
    async def test_not_found(self) -> None:
        with patch(f"{MODULE}.buscar_candidato", new_callable=AsyncMock, return_value=None):
            result = await tools.buscar_candidato(2020, 999, 999, 999)
        assert "não encontrado" in result


class TestBuscarCandidatoTotalizacao:
    @pytest.mark.asyncio
    async def test_shows_totalizacao(self) -> None:
        mock_data = Candidato(
            id=123,
            nome_urna="CANDIDATO X",
            partido="PT",
            situacao="Deferido",
            descricao_totalizacao="Eleito",
            total_votos=25000,
        )
        with patch(f"{MODULE}.buscar_candidato", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.buscar_candidato(2020, 35157, 2030402020, 123)
        assert "Eleito" in result
        assert "25.000" in result


class TestResultadoEleicao:
    @pytest.mark.asyncio
    async def test_formats_ranked_table(self) -> None:
        mock_data = [
            ResultadoCandidato(
                nome_urna="CANDIDATO A",
                numero=44,
                partido="PT",
                total_votos=10000,
                percentual="60,00%",
                descricao_totalizacao="Eleito",
            ),
            ResultadoCandidato(
                nome_urna="CANDIDATO B",
                numero=15,
                partido="MDB",
                total_votos=5000,
                percentual="30,00%",
                descricao_totalizacao="Não eleito",
            ),
        ]
        with patch(f"{MODULE}.resultado_eleicao", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.resultado_eleicao(2020, 35157, 2030402020, 11)
        assert "CANDIDATO A" in result
        assert "CANDIDATO B" in result
        assert "10.000" in result
        assert "60,00%" in result
        assert "Eleito" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.resultado_eleicao", new_callable=AsyncMock, return_value=[]):
            result = await tools.resultado_eleicao(2020, 999, 999, 11)
        assert "Nenhum resultado" in result


class TestConsultarPrestacaoContas:
    @pytest.mark.asyncio
    async def test_formats_finances(self) -> None:
        mock_data = PrestaContas(
            candidato_id="50000867342",
            nome="Candidato Teste",
            partido="PT",
            cnpj="12.345.678/0001-99",
            total_recebido=100000.0,
            total_despesas=80000.0,
            limite_gastos=200000.0,
        )
        with patch(
            f"{MODULE}.consultar_prestacao_contas",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_prestacao_contas(
                2030402020, 2020, 35157, 11, 50000867342
            )
        assert "Candidato Teste" in result
        assert "R$" in result
        assert "PT" in result

    @pytest.mark.asyncio
    async def test_not_found(self) -> None:
        with patch(
            f"{MODULE}.consultar_prestacao_contas",
            new_callable=AsyncMock,
            return_value=None,
        ):
            result = await tools.consultar_prestacao_contas(999, 2020, 999, 999, 999)
        assert "não encontrada" in result


# --- CDN de Resultados ---


def _mock_ctx() -> AsyncMock:
    """Create a mock FastMCP Context."""
    ctx = AsyncMock()
    ctx.info = AsyncMock()
    ctx.warning = AsyncMock()
    return ctx


def _make_resultado_regiao(
    uf: str = "BR",
    candidatos: list[ResultadoCDN] | None = None,
) -> ResultadoRegiao:
    """Build a ResultadoRegiao for testing."""
    if candidatos is None:
        candidatos = [
            ResultadoCDN(
                nome="LULA",
                numero="13",
                votos=60345999,
                percentual="50.90",
                eleito=True,
                situacao="Eleito",
            ),
            ResultadoCDN(
                nome="JAIR BOLSONARO",
                numero="22",
                votos=58206354,
                percentual="49.10",
                eleito=False,
                situacao="2 turno",
            ),
        ]
    return ResultadoRegiao(
        codigo=uf,
        tipo="br" if uf == "BR" else "uf",
        uf=uf,
        pct_apurado="100.00",
        total_secoes=472075,
        total_eleitores=156454011,
        total_comparecimento=124252796,
        total_abstencoes=32201215,
        candidatos=candidatos,
    )


class TestResultadoNacional:
    @pytest.mark.asyncio
    async def test_formats_national_result(self) -> None:
        ctx = _mock_ctx()
        resultado = _make_resultado_regiao("BR")
        with patch(
            f"{MODULE}.resultado_simplificado",
            new_callable=AsyncMock,
            return_value=resultado,
        ):
            result = await tools.resultado_nacional(2022, "presidente", ctx, 1)
        assert "Resultado Nacional" in result
        assert "Presidente" in result
        assert "LULA" in result
        assert "JAIR BOLSONARO" in result
        assert "100.00%" in result

    @pytest.mark.asyncio
    async def test_not_found(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{MODULE}.resultado_simplificado",
            new_callable=AsyncMock,
            return_value=None,
        ):
            result = await tools.resultado_nacional(2022, "presidente", ctx)
        assert "não encontrado" in result.lower()

    @pytest.mark.asyncio
    async def test_invalid_election_returns_error(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{MODULE}.resultado_simplificado",
            new_callable=AsyncMock,
            side_effect=ValueError("Eleição 1990 turno 1 não mapeada"),
        ):
            result = await tools.resultado_nacional(1990, "presidente", ctx)
        assert "não mapeada" in result


class TestResultadoPorEstado:
    @pytest.mark.asyncio
    async def test_formats_state_result(self) -> None:
        ctx = _mock_ctx()
        resultado = _make_resultado_regiao("SP")
        with patch(
            f"{MODULE}.resultado_simplificado",
            new_callable=AsyncMock,
            return_value=resultado,
        ):
            result = await tools.resultado_por_estado(2022, "presidente", "SP", ctx, 1)
        assert "SP" in result
        assert "Presidente" in result
        assert "LULA" in result

    @pytest.mark.asyncio
    async def test_not_found(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{MODULE}.resultado_simplificado",
            new_callable=AsyncMock,
            return_value=None,
        ):
            result = await tools.resultado_por_estado(2022, "presidente", "XX", ctx)
        assert "não encontrado" in result.lower()


class TestMapaResultadoEstados:
    @pytest.mark.asyncio
    async def test_formats_map(self) -> None:
        ctx = _mock_ctx()
        resultados = [
            _make_resultado_regiao("SP"),
            _make_resultado_regiao("RJ"),
        ]
        with patch(
            f"{MODULE}.resultado_todos_estados",
            new_callable=AsyncMock,
            return_value=resultados,
        ):
            result = await tools.mapa_resultado_estados(2022, "presidente", ctx, 1)
        assert "Mapa Eleitoral" in result
        assert "SP" in result
        assert "RJ" in result
        assert "2 estados" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{MODULE}.resultado_todos_estados",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.mapa_resultado_estados(2022, "presidente", ctx)
        assert "Nenhum resultado" in result


class TestApuracaoStatus:
    @pytest.mark.asyncio
    async def test_formats_status(self) -> None:
        ctx = _mock_ctx()
        resultado = _make_resultado_regiao("BR")
        with patch(
            f"{MODULE}.resultado_simplificado",
            new_callable=AsyncMock,
            return_value=resultado,
        ):
            result = await tools.apuracao_status(2022, "presidente", ctx, "br", 1)
        assert "Status da Apuração" in result
        assert "Nacional" in result
        assert "100.00%" in result

    @pytest.mark.asyncio
    async def test_state_level(self) -> None:
        ctx = _mock_ctx()
        resultado = _make_resultado_regiao("SP")
        with patch(
            f"{MODULE}.resultado_simplificado",
            new_callable=AsyncMock,
            return_value=resultado,
        ):
            result = await tools.apuracao_status(2022, "presidente", ctx, "SP", 1)
        assert "SP" in result
        assert "Status da Apuração" in result

    @pytest.mark.asyncio
    async def test_not_found(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{MODULE}.resultado_simplificado",
            new_callable=AsyncMock,
            return_value=None,
        ):
            result = await tools.apuracao_status(2022, "presidente", ctx)
        assert "não encontrados" in result.lower()


class TestListarMunicipiosEleitoraisTool:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        ctx = _mock_ctx()
        mock_data = [
            MunicipioEleitoral(
                codigo_tse="71072",
                codigo_ibge="3550308",
                nome="SÃO PAULO",
                capital=True,
                uf="SP",
            ),
            MunicipioEleitoral(
                codigo_tse="62910",
                codigo_ibge="3509502",
                nome="CAMPINAS",
                capital=False,
                uf="SP",
            ),
        ]
        with patch(
            f"{MODULE}.listar_municipios_eleitorais",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.listar_municipios_eleitorais(2024, "SP", ctx, 1)
        assert "SÃO PAULO" in result
        assert "71072" in result
        assert "3550308" in result
        assert "CAMPINAS" in result
        assert "2 " in result or "(2)" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{MODULE}.listar_municipios_eleitorais",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.listar_municipios_eleitorais(2024, "AM", ctx)
        assert "Nenhum município" in result

    @pytest.mark.asyncio
    async def test_invalid_election(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{MODULE}.listar_municipios_eleitorais",
            new_callable=AsyncMock,
            side_effect=ValueError("Eleição 1990 turno 1 não mapeada"),
        ):
            result = await tools.listar_municipios_eleitorais(1990, "SP", ctx)
        assert "não mapeada" in result


class TestResultadoPorMunicipio:
    @pytest.mark.asyncio
    async def test_formats_result(self) -> None:
        ctx = _mock_ctx()
        resultado = ResultadoRegiao(
            codigo="71072",
            tipo="mu",
            uf="71072",
            pct_apurado="100.00",
            total_eleitores=8000000,
            candidatos=[
                ResultadoCDN(
                    nome="CANDIDATO A",
                    numero="15",
                    votos=3500000,
                    percentual="53.85",
                    eleito=True,
                    situacao="Eleito",
                ),
                ResultadoCDN(
                    nome="CANDIDATO B",
                    numero="13",
                    votos=3000000,
                    percentual="46.15",
                    eleito=False,
                    situacao="Não eleito",
                ),
            ],
        )
        with patch(
            f"{MODULE}.resultado_municipio",
            new_callable=AsyncMock,
            return_value=resultado,
        ):
            result = await tools.resultado_por_municipio(2024, "prefeito", "SP", "71072", ctx, 1)
        assert "71072" in result
        assert "SP" in result
        assert "Prefeito" in result
        assert "CANDIDATO A" in result
        assert "CANDIDATO B" in result
        assert "100.00%" in result

    @pytest.mark.asyncio
    async def test_not_found(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{MODULE}.resultado_municipio",
            new_callable=AsyncMock,
            return_value=None,
        ):
            result = await tools.resultado_por_municipio(2024, "prefeito", "SP", "99999", ctx)
        assert "não encontrado" in result.lower()

    @pytest.mark.asyncio
    async def test_invalid_election(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{MODULE}.resultado_municipio",
            new_callable=AsyncMock,
            side_effect=ValueError("Eleição 1990 turno 1 não mapeada"),
        ):
            result = await tools.resultado_por_municipio(1990, "prefeito", "SP", "71072", ctx)
        assert "não mapeada" in result
