"""Tests for the dados_gov_br tool functions.

Tools are tested by mocking client functions (never HTTP).
Context is mocked via MagicMock with async methods.
"""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from mcp_brasil.data.dados_gov_br import tools
from mcp_brasil.data.dados_gov_br.schemas import (
    ODS,
    ConjuntoDetalhe,
    ConjuntoIndice,
    ConjuntoResultado,
    ObservanciaLegal,
    OrganizacaoDetalhe,
    OrganizacaoIndice,
    OrganizacaoResultado,
    Recurso,
    ReusoDetalhe,
    ReusoIndice,
    Tag,
    Tema,
)

CLIENT_MODULE = "mcp_brasil.data.dados_gov_br.client"


def _mock_ctx() -> MagicMock:
    """Create a mock Context with async log methods."""
    ctx = MagicMock()
    ctx.info = AsyncMock()
    ctx.warning = AsyncMock()
    return ctx


# ---------------------------------------------------------------------------
# buscar_conjuntos
# ---------------------------------------------------------------------------


class TestBuscarConjuntos:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = ConjuntoResultado(
            total=1,
            conjuntos=[
                ConjuntoIndice(
                    id="abc-123",
                    title="Dados de Saúde Pública",
                    organizationName="Ministério da Saúde",
                    temas="Saúde",
                    quantidadeRecursos=5,
                ),
            ],
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_conjuntos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_conjuntos(ctx, nome="saúde")
        assert "Dados de Saúde Pública" in result
        assert "Ministério da Saúde" in result
        assert "abc-123" in result
        assert "1 conjuntos de dados" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        mock_data = ConjuntoResultado(total=0, conjuntos=[])
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_conjuntos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_conjuntos(ctx)
        assert "Nenhum conjunto de dados encontrado" in result

    @pytest.mark.asyncio
    async def test_pagination_hint(self) -> None:
        mock_data = ConjuntoResultado(
            total=25,
            conjuntos=[ConjuntoIndice(id=f"id-{i}", title=f"Dataset {i}") for i in range(10)],
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_conjuntos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_conjuntos(ctx, pagina=1)
        assert "pagina=2" in result


# ---------------------------------------------------------------------------
# detalhar_conjunto
# ---------------------------------------------------------------------------


class TestDetalharConjunto:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = ConjuntoDetalhe(
            id="abc-123",
            titulo="Dados de Educação",
            descricao="Indicadores educacionais do INEP",
            organizacao="Ministério da Educação",
            licenca="cc-by",
            periodicidade="ANUAL",
            temas=[{"name": "educacao", "title": "Educação"}],
            tags=[Tag(id="t1", name="inep", display_name="INEP")],
            recursos=[
                Recurso(
                    id="r1",
                    titulo="Dados CSV",
                    formato="CSV",
                    link="https://dados.gov.br/download/dados.csv",
                ),
            ],
            dataUltimaAtualizacaoMetadados="2024-03-01",
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.detalhar_conjunto",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.detalhar_conjunto("abc-123", ctx)
        assert "Dados de Educação" in result
        assert "Indicadores educacionais do INEP" in result
        assert "Ministério da Educação" in result
        assert "Educação" in result
        assert "INEP" in result
        assert "CSV" in result
        assert "https://dados.gov.br/download/dados.csv" in result

    @pytest.mark.asyncio
    async def test_not_found(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.detalhar_conjunto",
            new_callable=AsyncMock,
            return_value=None,
        ):
            result = await tools.detalhar_conjunto("nao-existe", ctx)
        assert "não encontrado" in result
        assert "nao-existe" in result


# ---------------------------------------------------------------------------
# listar_organizacoes
# ---------------------------------------------------------------------------


class TestListarOrganizacoes:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = OrganizacaoResultado(
            total=2,
            organizacoes=[
                OrganizacaoIndice(
                    id="org-1",
                    titulo="Ministério da Saúde",
                    qtdConjuntoDeDados="150",
                    organizationEsfera="Federal",
                ),
                OrganizacaoIndice(
                    id="org-2",
                    titulo="IBGE",
                    qtdConjuntoDeDados="80",
                    organizationEsfera="Federal",
                ),
            ],
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_organizacoes",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.listar_organizacoes(ctx)
        assert "Ministério da Saúde" in result
        assert "IBGE" in result
        assert "150" in result
        assert "80" in result
        assert "2 organizações" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        mock_data = OrganizacaoResultado(total=0, organizacoes=[])
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_organizacoes",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.listar_organizacoes(ctx)
        assert "Nenhuma organização encontrada" in result


# ---------------------------------------------------------------------------
# detalhar_organizacao
# ---------------------------------------------------------------------------


class TestDetalharOrganizacao:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = OrganizacaoDetalhe(
            id="org-1",
            name="ministerio-da-saude",
            displayName="Ministério da Saúde",
            descricao="Órgão responsável pela saúde pública",
            quantidadeConjuntoDados=150,
            quantidadeSeguidores=500,
            ativo="true",
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.detalhar_organizacao",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.detalhar_organizacao("org-1", ctx)
        assert "Ministério da Saúde" in result
        assert "saúde pública" in result
        assert "150" in result

    @pytest.mark.asyncio
    async def test_not_found(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.detalhar_organizacao",
            new_callable=AsyncMock,
            return_value=None,
        ):
            result = await tools.detalhar_organizacao("nao-existe", ctx)
        assert "não encontrada" in result


# ---------------------------------------------------------------------------
# listar_temas
# ---------------------------------------------------------------------------


class TestListarTemas:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = [
            Tema(id="t1", name="saude", title="Saúde", packageCount=200),
            Tema(id="t2", name="educacao", title="Educação", packageCount=150),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_temas",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.listar_temas(ctx)
        assert "Saúde" in result
        assert "Educação" in result
        assert "200" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_temas",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.listar_temas(ctx)
        assert "Nenhum tema encontrado" in result


# ---------------------------------------------------------------------------
# buscar_tags
# ---------------------------------------------------------------------------


class TestBuscarTags:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = [
            Tag(id="tag-1", name="saude", display_name="Saúde"),
            Tag(id="tag-2", name="sus", display_name="SUS"),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_tags",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_tags("saude", ctx)
        assert "Saúde" in result
        assert "SUS" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_tags",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.buscar_tags("xyz", ctx)
        assert "Nenhuma tag encontrada" in result


# ---------------------------------------------------------------------------
# listar_formatos
# ---------------------------------------------------------------------------


class TestListarFormatos:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = ["CSV", "JSON", "XML"]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_formatos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.listar_formatos(ctx)
        assert "CSV" in result
        assert "JSON" in result
        assert "XML" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_formatos",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.listar_formatos(ctx)
        assert "Nenhum formato encontrado" in result


# ---------------------------------------------------------------------------
# listar_ods
# ---------------------------------------------------------------------------


class TestListarOds:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = [
            ODS(id=1, descricao="Erradicação da Pobreza"),
            ODS(id=3, descricao="Saúde e Bem-Estar"),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_ods",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.listar_ods(ctx)
        assert "Erradicação da Pobreza" in result
        assert "Saúde e Bem-Estar" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_ods",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.listar_ods(ctx)
        assert "Nenhum ODS encontrado" in result


# ---------------------------------------------------------------------------
# listar_observancia_legal
# ---------------------------------------------------------------------------


class TestListarObservanciaLegal:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = [
            ObservanciaLegal(id=1, descricao="Lei de Acesso à Informação"),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_observancia_legal",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.listar_observancia_legal(ctx)
        assert "Lei de Acesso à Informação" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_observancia_legal",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.listar_observancia_legal(ctx)
        assert "Nenhuma opção" in result


# ---------------------------------------------------------------------------
# listar_reusos
# ---------------------------------------------------------------------------


class TestListarReusos:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = [
            ReusoIndice(
                id="r1",
                nome="Dashboard COVID",
                autor="João",
                organizacao="Fiocruz",
                situacao="HOMOLOGADO",
            ),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_reusos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.listar_reusos(ctx)
        assert "Dashboard COVID" in result
        assert "João" in result
        assert "Fiocruz" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_reusos",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.listar_reusos(ctx)
        assert "Nenhum reuso encontrado" in result


# ---------------------------------------------------------------------------
# detalhar_reuso
# ---------------------------------------------------------------------------


class TestDetalharReuso:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = ReusoDetalhe(
            id="r1",
            nome="Dashboard COVID",
            descricao="Painel de monitoramento",
            url="https://example.com/dashboard",
            autor="João",
            organizacao="Fiocruz",
            situacao="HOMOLOGADO",
            temas=["Saúde"],
            tipos=["Dashboard"],
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.detalhar_reuso",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.detalhar_reuso("r1", ctx)
        assert "Dashboard COVID" in result
        assert "monitoramento" in result
        assert "https://example.com/dashboard" in result
        assert "João" in result

    @pytest.mark.asyncio
    async def test_not_found(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.detalhar_reuso",
            new_callable=AsyncMock,
            return_value=None,
        ):
            result = await tools.detalhar_reuso("nao-existe", ctx)
        assert "não encontrado" in result
