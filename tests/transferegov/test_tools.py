"""Tests for the transferegov tool functions.

Tools are tested by mocking client functions (never HTTP).
"""

from unittest.mock import AsyncMock, patch

import pytest

from mcp_brasil.transferegov import tools
from mcp_brasil.transferegov.constants import DEFAULT_PAGE_SIZE
from mcp_brasil.transferegov.schemas import TransferenciaEspecial

MODULE = "mcp_brasil.transferegov.client"


def _make_emenda(**kwargs: object) -> TransferenciaEspecial:
    defaults: dict[str, object] = {
        "id_transferencia_especial": 1,
        "ano_exercicio": 2024,
        "nr_emenda": "EMD-PIX-001",
        "autor_emenda": "Dep. Fulano",
        "tipo_emenda": "Individual",
        "valor_empenhado": 500000.0,
        "valor_pago": 300000.0,
        "nm_municipio_beneficiario": "Teresina",
        "uf_beneficiario": "PI",
    }
    defaults.update(kwargs)
    return TransferenciaEspecial(**defaults)  # type: ignore[arg-type]


# ---------------------------------------------------------------------------
# buscar_emendas_pix
# ---------------------------------------------------------------------------


class TestBuscarEmendasPix:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [_make_emenda()]
        with patch(f"{MODULE}.buscar_emendas_pix", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.buscar_emendas_pix(ano=2024)
        assert "EMD-PIX-001" in result
        assert "Dep. Fulano" in result
        assert "R$ 500.000,00" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.buscar_emendas_pix", new_callable=AsyncMock, return_value=[]):
            result = await tools.buscar_emendas_pix()
        assert "Nenhuma emenda pix" in result


# ---------------------------------------------------------------------------
# buscar_emenda_por_autor
# ---------------------------------------------------------------------------


class TestBuscarEmendaPorAutor:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [_make_emenda()]
        with patch(
            f"{MODULE}.buscar_emenda_por_autor", new_callable=AsyncMock, return_value=mock_data
        ):
            result = await tools.buscar_emenda_por_autor("Fulano")
        assert "Dep. Fulano" in result
        assert "Teresina" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.buscar_emenda_por_autor", new_callable=AsyncMock, return_value=[]):
            result = await tools.buscar_emenda_por_autor("Inexistente")
        assert "Nenhuma emenda pix" in result


# ---------------------------------------------------------------------------
# detalhe_emenda
# ---------------------------------------------------------------------------


class TestDetalheEmenda:
    @pytest.mark.asyncio
    async def test_formats_detail(self) -> None:
        mock_data = _make_emenda(
            funcao="Saúde",
            subfuncao="Atenção Básica",
            objeto="Construção de UBS",
            valor_liquidado=400000.0,
            nm_entidade_beneficiaria="Prefeitura de Teresina",
        )
        with patch(f"{MODULE}.detalhe_emenda", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.detalhe_emenda(1)
        assert "EMD-PIX-001" in result
        assert "R$ 500.000,00" in result
        assert "Saúde" in result
        assert "Construção de UBS" in result

    @pytest.mark.asyncio
    async def test_not_found(self) -> None:
        with patch(f"{MODULE}.detalhe_emenda", new_callable=AsyncMock, return_value=None):
            result = await tools.detalhe_emenda(999)
        assert "não encontrada" in result


# ---------------------------------------------------------------------------
# emendas_por_municipio
# ---------------------------------------------------------------------------


class TestEmendasPorMunicipio:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [_make_emenda()]
        with patch(
            f"{MODULE}.emendas_por_municipio", new_callable=AsyncMock, return_value=mock_data
        ):
            result = await tools.emendas_por_municipio("Teresina")
        assert "Teresina" in result
        assert "R$ 300.000,00" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.emendas_por_municipio", new_callable=AsyncMock, return_value=[]):
            result = await tools.emendas_por_municipio("Inexistente")
        assert "Nenhuma emenda pix" in result


# ---------------------------------------------------------------------------
# resumo_emendas_ano
# ---------------------------------------------------------------------------


class TestResumoEmendasAno:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [_make_emenda()]
        with patch(f"{MODULE}.resumo_emendas_ano", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.resumo_emendas_ano(2024)
        assert "2024" in result
        assert "EMD-PIX-001" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.resumo_emendas_ano", new_callable=AsyncMock, return_value=[]):
            result = await tools.resumo_emendas_ano(2025)
        assert "Nenhuma emenda pix" in result


# ---------------------------------------------------------------------------
# Pagination hints
# ---------------------------------------------------------------------------


class TestPaginationHints:
    @pytest.mark.asyncio
    async def test_shows_next_page_hint(self) -> None:
        data = [_make_emenda(nr_emenda=f"EMD-{i}") for i in range(DEFAULT_PAGE_SIZE)]
        with patch(f"{MODULE}.buscar_emendas_pix", new_callable=AsyncMock, return_value=data):
            result = await tools.buscar_emendas_pix(ano=2024)
        assert "pagina=2" in result

    @pytest.mark.asyncio
    async def test_no_hint_below_page_size(self) -> None:
        data = [_make_emenda()]
        with patch(f"{MODULE}.buscar_emendas_pix", new_callable=AsyncMock, return_value=data):
            result = await tools.buscar_emendas_pix(ano=2024, pagina=1)
        assert "pagina=" not in result

    @pytest.mark.asyncio
    async def test_last_page_hint(self) -> None:
        data = [_make_emenda()]
        with patch(f"{MODULE}.buscar_emendas_pix", new_callable=AsyncMock, return_value=data):
            result = await tools.buscar_emendas_pix(ano=2024, pagina=2)
        assert "Última página" in result
