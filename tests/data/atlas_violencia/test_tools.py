"""Tests for the Atlas da Violência tool functions."""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from mcp_brasil.data.atlas_violencia import tools
from mcp_brasil.data.atlas_violencia.schemas import (
    Fonte,
    Periodicidade,
    Serie,
    Tema,
    Unidade,
    ValorSerie,
)

CLIENT_MODULE = "mcp_brasil.data.atlas_violencia.client"


def _mock_ctx() -> MagicMock:
    ctx = MagicMock()
    ctx.info = AsyncMock()
    ctx.warning = AsyncMock()
    return ctx


class TestListarTemasViolencia:
    @pytest.mark.asyncio
    async def test_formats_temas(self) -> None:
        mock_data = [
            Tema(id=1, titulo="Homicidios"),
            Tema(id=5, titulo="Obitos por Armas de Fogo"),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_temas", new_callable=AsyncMock, return_value=mock_data
        ):
            result = await tools.listar_temas_violencia(ctx)
        assert "2 temas" in result
        assert "Homicidios" in result
        assert "Armas de Fogo" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        ctx = _mock_ctx()
        with patch(f"{CLIENT_MODULE}.listar_temas", new_callable=AsyncMock, return_value=[]):
            result = await tools.listar_temas_violencia(ctx)
        assert "Nenhum tema" in result


class TestListarSeriesTema:
    @pytest.mark.asyncio
    async def test_formats_series(self) -> None:
        mock_data = [
            Serie(id=328, titulo="Homicidios"),
            Serie(id=329, titulo="Homicidios - Masculino"),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_series", new_callable=AsyncMock, return_value=mock_data
        ):
            result = await tools.listar_series_tema(1, ctx)
        assert "2 séries" in result
        assert "328" in result
        assert "Homicidios" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        ctx = _mock_ctx()
        with patch(f"{CLIENT_MODULE}.listar_series", new_callable=AsyncMock, return_value=[]):
            result = await tools.listar_series_tema(99, ctx)
        assert "Nenhuma série" in result


class TestConsultarValoresViolencia:
    @pytest.mark.asyncio
    async def test_formats_values(self) -> None:
        mock_data = [
            ValorSerie(cod="33", sigla="RJ", valor="5000", periodo="2023-01-15"),
            ValorSerie(cod="35", sigla="SP", valor="8000", periodo="2023-01-15"),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_valores", new_callable=AsyncMock, return_value=mock_data
        ):
            result = await tools.consultar_valores_violencia(328, ctx, abrangencia=3)
        assert "RJ" in result
        assert "SP" in result
        assert "5000" in result

    @pytest.mark.asyncio
    async def test_rejects_municipal(self) -> None:
        ctx = _mock_ctx()
        result = await tools.consultar_valores_violencia(328, ctx, abrangencia=4)
        assert "municipal" in result.lower()

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        ctx = _mock_ctx()
        with patch(f"{CLIENT_MODULE}.consultar_valores", new_callable=AsyncMock, return_value=[]):
            result = await tools.consultar_valores_violencia(328, ctx)
        assert "Nenhum dado" in result


class TestConsultarValoresPorRegiao:
    @pytest.mark.asyncio
    async def test_formats_filtered(self) -> None:
        mock_data = [
            ValorSerie(cod="33", sigla="RJ", valor="5000", periodo="2023-01-15"),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_valores_por_regioes",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_valores_por_regiao(328, "33", ctx)
        assert "RJ" in result
        assert "5000" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_valores_por_regioes",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.consultar_valores_por_regiao(328, "99", ctx)
        assert "Nenhum dado" in result


class TestConsultarSerieViolencia:
    @pytest.mark.asyncio
    async def test_returns_metadata(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_serie",
            new_callable=AsyncMock,
            return_value=Serie(id=328, titulo="Homicidios"),
        ):
            result = await tools.consultar_serie_violencia(328, ctx)
        assert "328" in result
        assert "Homicidios" in result

    @pytest.mark.asyncio
    async def test_not_found(self) -> None:
        ctx = _mock_ctx()
        with patch(f"{CLIENT_MODULE}.consultar_serie", new_callable=AsyncMock, return_value=None):
            result = await tools.consultar_serie_violencia(999, ctx)
        assert "não encontrada" in result


class TestListarFontesViolencia:
    @pytest.mark.asyncio
    async def test_formats_fontes(self) -> None:
        mock_data = [Fonte(id=1, titulo="IPEA"), Fonte(id=2, titulo="SIM/DataSUS")]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_fontes", new_callable=AsyncMock, return_value=mock_data
        ):
            result = await tools.listar_fontes_violencia(ctx)
        assert "IPEA" in result
        assert "SIM/DataSUS" in result


class TestListarMetadadosViolencia:
    @pytest.mark.asyncio
    async def test_formats_metadata(self) -> None:
        mock_unidades = [Unidade(id=7, titulo="Quantidade")]
        mock_periodicidades = [Periodicidade(id=1, titulo="Anual")]
        ctx = _mock_ctx()
        with (
            patch(
                f"{CLIENT_MODULE}.listar_unidades",
                new_callable=AsyncMock,
                return_value=mock_unidades,
            ),
            patch(
                f"{CLIENT_MODULE}.listar_periodicidades",
                new_callable=AsyncMock,
                return_value=mock_periodicidades,
            ),
        ):
            result = await tools.listar_metadados_violencia(ctx)
        assert "Quantidade" in result
        assert "Anual" in result
        assert "Unidades" in result
        assert "Periodicidades" in result
