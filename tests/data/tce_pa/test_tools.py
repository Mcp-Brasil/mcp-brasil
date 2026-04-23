"""Tests for the TCE-PA tool functions."""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from mcp_brasil.data.tce_pa import tools
from mcp_brasil.data.tce_pa.schemas import DiarioOficial, ResultadoPesquisa, SessaoPlenaria

CLIENT_MODULE = "mcp_brasil.data.tce_pa.client"


def _mock_ctx() -> MagicMock:
    ctx = MagicMock()
    ctx.info = AsyncMock()
    ctx.warning = AsyncMock()
    return ctx


# ---------------------------------------------------------------------------
# buscar_diario_oficial_pa
# ---------------------------------------------------------------------------


class TestBuscarDiarioOficialPa:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = [
            DiarioOficial(
                numero_publicacao=42,
                data_publicacao="2024-03-15",
                tipo_ato="Contratos",
                publicacao="Contrato nº 001/2024 firmado entre TCE-PA e Empresa X",
            )
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_diario_oficial",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_diario_oficial_pa(ctx, ano=2024)
        assert "42" in result
        assert "2024-03-15" in result
        assert "Contratos" in result
        assert "Contrato nº 001/2024" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_diario_oficial",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.buscar_diario_oficial_pa(ctx, ano=2024)
        assert "Nenhuma publicação encontrada" in result

    @pytest.mark.asyncio
    async def test_truncates_long_publicacao(self) -> None:
        long_text = "A" * 200
        mock_data = [
            DiarioOficial(
                numero_publicacao=1,
                data_publicacao="2024-01-01",
                tipo_ato="Atos e Normas",
                publicacao=long_text,
            )
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_diario_oficial",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_diario_oficial_pa(ctx, ano=2024)
        assert "..." in result

    @pytest.mark.asyncio
    async def test_handles_none_fields(self) -> None:
        mock_data = [DiarioOficial()]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_diario_oficial",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_diario_oficial_pa(ctx, ano=2024)
        assert "—" in result


# ---------------------------------------------------------------------------
# buscar_sessoes_plenarias_pa
# ---------------------------------------------------------------------------


class TestBuscarSessoesPlenariasPa:
    @pytest.mark.asyncio
    async def test_formats_sessoes(self) -> None:
        mock_data = [
            SessaoPlenaria(
                codigo=10,
                titulo="Sessão Ordinária nº 01/2024",
                data_sessao="15/01/2024",
                tipo_sessao="Ordinária",
                ano=2024,
                url_pagina="https://www.tcepa.tc.br/sessoes/codigo/10/titulo",
            )
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_sessoes_plenarias",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_sessoes_plenarias_pa(ctx, tipo="sessoes")
        assert "15/01/2024" in result
        assert "Ordinária" in result
        assert "2024" in result

    @pytest.mark.asyncio
    async def test_formats_pautas_with_pdf_col(self) -> None:
        mock_data = [
            SessaoPlenaria(
                titulo="Pauta nº 01/2024",
                data_sessao="15/01/2024",
                tipo_sessao="Ordinária",
                url_documento="https://www.tcepa.tc.br/pautas/codigo/5/titulo/download",
            )
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_sessoes_plenarias",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_sessoes_plenarias_pa(ctx, tipo="pautas")
        assert "Download PDF" in result

    @pytest.mark.asyncio
    async def test_formats_videos_with_youtube_col(self) -> None:
        mock_data = [
            SessaoPlenaria(
                titulo="Vídeo Sessão Ordinária 01/2024",
                data_sessao="15/01/2024",
                tipo_sessao="Ordinária",
                url_documento="https://youtu.be/abc123",
            )
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_sessoes_plenarias",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_sessoes_plenarias_pa(ctx, tipo="videos")
        assert "Vídeo (YouTube)" in result

    @pytest.mark.asyncio
    async def test_invalid_tipo_returns_error(self) -> None:
        ctx = _mock_ctx()
        result = await tools.buscar_sessoes_plenarias_pa(ctx, tipo="invalido")
        assert "Tipo inválido" in result
        assert "invalido" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_sessoes_plenarias",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.buscar_sessoes_plenarias_pa(ctx, tipo="sessoes")
        assert "Nenhuma sessão encontrada" in result


# ---------------------------------------------------------------------------
# buscar_jurisprudencia_pa
# ---------------------------------------------------------------------------


class TestBuscarJurisprudenciaPa:
    @pytest.mark.asyncio
    async def test_formats_acordaos(self) -> None:
        mock_data = [
            ResultadoPesquisa(
                titulo="Acórdão nº 1234/2024",
                url_pagina="https://www.tcepa.tc.br/acordaos/codigo/99/titulo",
                url_documento="https://www.tcepa.tc.br/acordaos/codigo/99/titulo/download",
                campos={"Data da sessão plenária": "20/03/2024", "Decisões": "Aprovado"},
            )
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_pesquisa_integrada",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_jurisprudencia_pa(ctx, base="acordaos")
        assert "Acórdão nº 1234/2024" in result
        assert "20/03/2024" in result
        assert "/download" in result

    @pytest.mark.asyncio
    async def test_invalid_base_returns_error(self) -> None:
        ctx = _mock_ctx()
        result = await tools.buscar_jurisprudencia_pa(ctx, base="invalida")
        assert "Base inválida" in result
        assert "invalida" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_pesquisa_integrada",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.buscar_jurisprudencia_pa(ctx, base="resolucoes")
        assert "Nenhum resultado encontrado" in result

    @pytest.mark.asyncio
    async def test_all_valid_bases_accepted(self) -> None:
        bases = [
            "acordaos",
            "acordaos-virtual",
            "resolucoes",
            "portarias",
            "atos",
            "informativos",
            "prejulgados",
        ]
        ctx = _mock_ctx()
        for base in bases:
            with patch(
                f"{CLIENT_MODULE}.buscar_pesquisa_integrada",
                new_callable=AsyncMock,
                return_value=[],
            ):
                result = await tools.buscar_jurisprudencia_pa(ctx, base=base)
            assert "Base inválida" not in result


# ---------------------------------------------------------------------------
# buscar_conteudo_pa
# ---------------------------------------------------------------------------


class TestBuscarConteudoPa:
    @pytest.mark.asyncio
    async def test_formats_noticias(self) -> None:
        mock_data = [
            ResultadoPesquisa(
                titulo="TCE-PA lança sistema de fiscalização",
                url_pagina="https://www.tcepa.tc.br/noticias/codigo/55/titulo",
                url_documento="https://www.tcepa.tc.br/noticias/codigo/55/conteudo-original",
                campos={"Data de publicação": "10/04/2024", "Resumo": "O TCE-PA anunciou..."},
            )
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_pesquisa_integrada",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_conteudo_pa(ctx, base="noticias")
        assert "TCE-PA lança sistema" in result
        assert "10/04/2024" in result

    @pytest.mark.asyncio
    async def test_youtube_col_label(self) -> None:
        mock_data = [
            ResultadoPesquisa(
                titulo="Sessão ao vivo",
                url_documento="https://youtu.be/xyz",
                campos={},
            )
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_pesquisa_integrada",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_conteudo_pa(ctx, base="youtube")
        assert "Vídeo (YouTube)" in result

    @pytest.mark.asyncio
    async def test_invalid_base_returns_error(self) -> None:
        ctx = _mock_ctx()
        result = await tools.buscar_conteudo_pa(ctx, base="invalida")
        assert "Base inválida" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_pesquisa_integrada",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.buscar_conteudo_pa(ctx, base="noticias")
        assert "Nenhum resultado encontrado" in result

    @pytest.mark.asyncio
    async def test_all_valid_bases_accepted(self) -> None:
        bases = ["noticias", "acervo", "educacao", "youtube", "imagens"]
        ctx = _mock_ctx()
        for base in bases:
            with patch(
                f"{CLIENT_MODULE}.buscar_pesquisa_integrada",
                new_callable=AsyncMock,
                return_value=[],
            ):
                result = await tools.buscar_conteudo_pa(ctx, base=base)
            assert "Base inválida" not in result
