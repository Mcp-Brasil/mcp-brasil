"""Tests for the DOU (federal) tool functions.

Tools are tested by mocking client_dou functions (never HTTP).
"""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from mcp_brasil.data.diario_oficial import tools_dou
from mcp_brasil.data.diario_oficial.schemas import PublicacaoDOU, ResultadoDOU
from mcp_brasil.exceptions import HttpClientError

CLIENT_MODULE = "mcp_brasil.data.diario_oficial.client_dou"


def _mock_ctx() -> MagicMock:
    """Create a mock Context with async log methods."""
    ctx = MagicMock()
    ctx.info = AsyncMock()
    ctx.warning = AsyncMock()
    return ctx


def _sample_pub(**overrides: object) -> PublicacaoDOU:
    """Create a sample DOU publication with defaults."""
    defaults = {
        "titulo": "Portaria nº 123",
        "resumo": "Dispõe sobre medidas administrativas",
        "url_titulo": "portaria-123-2024",
        "orgao": "Ministério da Saúde",
        "tipo_publicacao": "Portaria",
        "secao": "DO1",
        "data_publicacao": "2024-01-15",
        "edicao": "42",
        "pagina": "10",
        "assinante": "João Silva",
        "cargo_assinante": "Ministro",
    }
    defaults.update(overrides)
    return PublicacaoDOU(**defaults)


# ---------------------------------------------------------------------------
# dou_buscar
# ---------------------------------------------------------------------------


class TestDouBuscar:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = ResultadoDOU(total=1, publicacoes=[_sample_pub()])
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_dou",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools_dou.dou_buscar("portaria", ctx)
        assert "Portaria nº 123" in result
        assert "Ministério da Saúde" in result
        assert "2024-01-15" in result
        assert "João Silva" in result
        assert "1 publicações" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        mock_data = ResultadoDOU(total=0, publicacoes=[])
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_dou",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools_dou.dou_buscar("inexistente", ctx)
        assert "Nenhuma publicação encontrada" in result

    @pytest.mark.asyncio
    async def test_pagination_hint(self) -> None:
        pubs = [_sample_pub(titulo=f"Pub {i}") for i in range(20)]
        mock_data = ResultadoDOU(total=50, publicacoes=pubs)
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_dou",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools_dou.dou_buscar("teste", ctx)
        assert "pagina=1" in result
        assert "50" in result

    @pytest.mark.asyncio
    async def test_source_error_is_not_empty_result(self) -> None:
        """Client HTTP errors surface as outage, not "no results" (issue #22)."""
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_dou",
            new_callable=AsyncMock,
            side_effect=HttpClientError("Busca DOU indisponível (HTTP 404 em ...)"),
        ):
            result = await tools_dou.dou_buscar("portaria", ctx)
        assert "indisponível" in result
        assert "Nenhuma publicação encontrada" not in result


# ---------------------------------------------------------------------------
# dou_ler_publicacao
# ---------------------------------------------------------------------------


class TestDouLerPublicacao:
    @pytest.mark.asyncio
    async def test_formats_article(self) -> None:
        pub = _sample_pub(conteudo="Texto completo da portaria para leitura")
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.ler_publicacao_dou",
            new_callable=AsyncMock,
            return_value=pub,
        ):
            result = await tools_dou.dou_ler_publicacao("portaria-123", ctx)
        assert "Portaria nº 123" in result
        assert "Texto completo" in result
        assert "João Silva" in result
        assert "Ministro" in result

    @pytest.mark.asyncio
    async def test_not_found(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.ler_publicacao_dou",
            new_callable=AsyncMock,
            return_value=None,
        ):
            result = await tools_dou.dou_ler_publicacao("nao-existe", ctx)
        assert "não encontrada" in result


# ---------------------------------------------------------------------------
# dou_edicao_do_dia
# ---------------------------------------------------------------------------


class TestDouEdicaoDoDia:
    @pytest.mark.asyncio
    async def test_formats_edition(self) -> None:
        pubs = [
            _sample_pub(titulo="Portaria A", tipo_publicacao="Portaria", orgao="Órgão A"),
            _sample_pub(titulo="Decreto B", tipo_publicacao="Decreto", orgao="Órgão B"),
        ]
        mock_data = ResultadoDOU(total=2, publicacoes=pubs)
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_dou",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools_dou.dou_edicao_do_dia("2024-01-15", ctx)
        assert "2024-01-15" in result
        assert "Portaria A" in result
        assert "Decreto B" in result
        assert "2 publicações" in result

    @pytest.mark.asyncio
    async def test_empty_edition(self) -> None:
        mock_data = ResultadoDOU(total=0, publicacoes=[])
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_dou",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools_dou.dou_edicao_do_dia("2024-12-25", ctx)
        assert "Nenhuma publicação" in result

    @pytest.mark.asyncio
    async def test_source_error_is_not_empty_result(self) -> None:
        """Client HTTP errors surface as outage, not "no results" (issue #22)."""
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_dou",
            new_callable=AsyncMock,
            side_effect=HttpClientError("Busca DOU indisponível (HTTP 404 em ...)"),
        ):
            result = await tools_dou.dou_edicao_do_dia("2024-01-15", ctx)
        assert "indisponível" in result
        assert "Nenhuma publicação" not in result


# ---------------------------------------------------------------------------
# dou_buscar_por_orgao
# ---------------------------------------------------------------------------


class TestDouBuscarPorOrgao:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        pubs = [_sample_pub(titulo="Portaria X", orgao="ANVISA")]
        mock_data = ResultadoDOU(total=1, publicacoes=pubs)
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_dou",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools_dou.dou_buscar_por_orgao("ANVISA", ctx)
        assert "ANVISA" in result
        assert "Portaria X" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        mock_data = ResultadoDOU(total=0, publicacoes=[])
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_dou",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools_dou.dou_buscar_por_orgao("Órgão X", ctx)
        assert "Nenhuma publicação" in result


# ---------------------------------------------------------------------------
# dou_buscar_avancado
# ---------------------------------------------------------------------------


class TestDouBuscarAvancado:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        pubs = [_sample_pub()]
        mock_data = ResultadoDOU(total=1, publicacoes=pubs)
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_dou",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools_dou.dou_buscar_avancado(
                ctx, termo="teste", orgao="IBAMA", tipo="Portaria"
            )
        assert "Portaria nº 123" in result
        assert "1 resultados" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        mock_data = ResultadoDOU(total=0, publicacoes=[])
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_dou",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools_dou.dou_buscar_avancado(ctx, termo="inexistente")
        assert "Nenhuma publicação encontrada" in result

    @pytest.mark.asyncio
    async def test_date_range_uses_personalizado(self) -> None:
        mock_data = ResultadoDOU(total=0, publicacoes=[])
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_dou",
            new_callable=AsyncMock,
            return_value=mock_data,
        ) as mock_client:
            await tools_dou.dou_buscar_avancado(
                ctx, data_inicio="2024-01-01", data_fim="2024-06-30"
            )
        call_kwargs = mock_client.call_args.kwargs
        assert call_kwargs["periodo"] == "PERSONALIZADO"
        assert call_kwargs["data_inicio"] == "2024-01-01"
        assert call_kwargs["data_fim"] == "2024-06-30"
