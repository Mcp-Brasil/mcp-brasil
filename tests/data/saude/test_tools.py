"""Tests for the Saúde tool functions.

Tools are tested by mocking client functions (never HTTP).
Context is mocked via MagicMock with async methods.
"""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from mcp_brasil.data.saude import tools
from mcp_brasil.data.saude.schemas import (
    Estabelecimento,
    EstabelecimentoDetalhe,
    Leito,
    TipoEstabelecimento,
)
from mcp_brasil.exceptions import HttpClientError

CLIENT_MODULE = "mcp_brasil.data.saude.client"


def _mock_ctx() -> MagicMock:
    """Create a mock Context with async log methods."""
    ctx = MagicMock()
    ctx.info = AsyncMock()
    ctx.warning = AsyncMock()
    return ctx


# ---------------------------------------------------------------------------
# buscar_estabelecimentos
# ---------------------------------------------------------------------------


class TestBuscarEstabelecimentos:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Estabelecimento(
                codigo_cnes="1234567",
                nome_fantasia="UBS Central",
                descricao_tipo="Central de Regulação",
                tipo_gestao="Municipal",
                endereco="Rua ABC, 123",
            ),
            Estabelecimento(
                codigo_cnes="7654321",
                nome_fantasia="Hospital Geral",
                descricao_tipo="Hospital Geral",
                tipo_gestao="Estadual",
                endereco="Av XYZ, 456",
            ),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_estabelecimentos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_estabelecimentos(ctx, codigo_municipio="355030")
        assert "UBS Central" in result
        assert "1234567" in result
        assert "Hospital Geral" in result
        assert "2 resultados" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_estabelecimentos",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.buscar_estabelecimentos(ctx)
        assert "Nenhum estabelecimento" in result

    @pytest.mark.asyncio
    async def test_missing_fields_use_dash(self) -> None:
        mock_data = [Estabelecimento()]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_estabelecimentos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_estabelecimentos(ctx)
        assert "—" in result


# ---------------------------------------------------------------------------
# buscar_profissionais (deprecated — returns info message)
# ---------------------------------------------------------------------------


class TestBuscarProfissionais:
    @pytest.mark.asyncio
    async def test_returns_deprecation_message(self) -> None:
        ctx = _mock_ctx()
        result = await tools.buscar_profissionais(ctx, cnes="1234567")
        assert "descontinuado" in result


# ---------------------------------------------------------------------------
# listar_tipos_estabelecimento
# ---------------------------------------------------------------------------


class TestListarTiposEstabelecimento:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            TipoEstabelecimento(codigo="01", descricao="Central de Regulação"),
            TipoEstabelecimento(codigo="02", descricao="Hospital Geral"),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_tipos_estabelecimento",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.listar_tipos_estabelecimento(ctx)
        assert "Central de Regulação" in result
        assert "Hospital Geral" in result
        assert "2 tipos" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_tipos_estabelecimento",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.listar_tipos_estabelecimento(ctx)
        assert "Nenhum tipo" in result


# ---------------------------------------------------------------------------
# consultar_leitos
# ---------------------------------------------------------------------------


class TestConsultarLeitos:
    @pytest.mark.asyncio
    async def test_formats_table_with_totals(self) -> None:
        mock_data = [
            Leito(
                codigo_cnes="1234567",
                tipo_leito="Cirúrgico",
                especialidade="Cirurgia Geral",
                existente=20,
                sus=15,
            ),
            Leito(
                codigo_cnes="1234567",
                tipo_leito="Clínico",
                especialidade="Clínica Médica",
                existente=30,
                sus=25,
            ),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_leitos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_leitos(ctx)
        assert "Cirúrgico" in result
        assert "Clínico" in result
        assert "2 registros" in result
        # Check totals: 50 existentes, 40 SUS
        assert "50" in result
        assert "40" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_leitos",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.consultar_leitos(ctx)
        assert "Nenhum leito" in result

    @pytest.mark.asyncio
    async def test_none_values_handled(self) -> None:
        mock_data = [
            Leito(codigo_cnes="1234567", existente=None, sus=None),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_leitos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_leitos(ctx)
        assert "—" in result


# ---------------------------------------------------------------------------
# buscar_urgencias
# ---------------------------------------------------------------------------


class TestBuscarUrgencias:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Estabelecimento(
                codigo_cnes="9876543",
                nome_fantasia="UPA 24h Norte",
                descricao_tipo="Pronto Atendimento",
                endereco="Av Norte, 100",
            ),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_estabelecimentos_por_tipo",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_urgencias(ctx, codigo_municipio="220040")
        assert "UPA 24h Norte" in result
        assert "Pronto Atendimento" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_estabelecimentos_por_tipo",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.buscar_urgencias(ctx)
        assert "Nenhuma unidade de urgência" in result


# ---------------------------------------------------------------------------
# buscar_por_tipo
# ---------------------------------------------------------------------------


class TestBuscarPorTipo:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Estabelecimento(
                codigo_cnes="1111111",
                nome_fantasia="Hospital Regional",
                descricao_tipo="Hospital Geral",
                tipo_gestao="Estadual",
                endereco="Rua Hospital, 1",
            ),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_estabelecimentos_por_tipo",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_por_tipo(ctx, codigo_tipo="05")
        assert "Hospital Regional" in result
        assert "1 resultados" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_estabelecimentos_por_tipo",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.buscar_por_tipo(ctx, codigo_tipo="99")
        assert "Nenhum estabelecimento do tipo 99" in result


# ---------------------------------------------------------------------------
# buscar_estabelecimento_por_cnes
# ---------------------------------------------------------------------------


class TestBuscarEstabelecimentoPorCnes:
    @pytest.mark.asyncio
    async def test_formats_detail(self) -> None:
        mock_data = EstabelecimentoDetalhe(
            codigo_cnes="1234567",
            nome_fantasia="Hospital São Paulo",
            nome_razao_social="Hospital São Paulo Ltda",
            descricao_tipo="Hospital Geral",
            tipo_gestao="Estadual",
            natureza_organizacao="Administração Pública",
            endereco="Rua Napoleão de Barros, 715",
            bairro="Vila Clementino",
            cep="04024-002",
            telefone="(11) 5576-4000",
            cnpj="12.345.678/0001-90",
            codigo_municipio="355030",
            codigo_uf="35",
            latitude=-23.5989,
            longitude=-46.6423,
            data_atualizacao="2024-01-15",
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_estabelecimento_por_cnes",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_estabelecimento_por_cnes(ctx, cnes="1234567")
        assert "Hospital São Paulo" in result
        assert "(11) 5576-4000" in result
        assert "-23.5989" in result
        assert "2024-01-15" in result

    @pytest.mark.asyncio
    async def test_not_found(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_estabelecimento_por_cnes",
            new_callable=AsyncMock,
            return_value=None,
        ):
            result = await tools.buscar_estabelecimento_por_cnes(ctx, cnes="0000000")
        assert "não encontrado" in result


# ---------------------------------------------------------------------------
# buscar_por_coordenadas
# ---------------------------------------------------------------------------


class TestBuscarPorCoordenadas:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Estabelecimento(
                codigo_cnes="2222222",
                nome_fantasia="UBS Próxima",
                descricao_tipo="Centro de Saúde/UBS",
                endereco="Rua Perto, 50",
            ),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_estabelecimentos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_por_coordenadas(
                ctx,
                latitude=-5.0892,
                longitude=-42.8019,
                codigo_municipio="220040",
            )
        assert "UBS Próxima" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_estabelecimentos",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.buscar_por_coordenadas(
                ctx,
                latitude=-5.0,
                longitude=-42.0,
                codigo_municipio="220040",
            )
        assert "Nenhum estabelecimento" in result


# ---------------------------------------------------------------------------
# resumo_rede_municipal
# ---------------------------------------------------------------------------


class TestResumoRedeMunicipal:
    @pytest.mark.asyncio
    async def test_formats_summary(self) -> None:
        mock_estab = [
            Estabelecimento(descricao_tipo="Hospital Geral"),
            Estabelecimento(descricao_tipo="Hospital Geral"),
            Estabelecimento(descricao_tipo="Centro de Saúde/UBS"),
        ]
        mock_leitos = [
            Leito(existente=20, sus=15),
            Leito(existente=30, sus=25),
        ]
        ctx = _mock_ctx()
        with (
            patch(
                f"{CLIENT_MODULE}.buscar_estabelecimentos",
                new_callable=AsyncMock,
                return_value=mock_estab,
            ),
            patch(
                f"{CLIENT_MODULE}.consultar_leitos",
                new_callable=AsyncMock,
                return_value=mock_leitos,
            ),
        ):
            result = await tools.resumo_rede_municipal(ctx, codigo_municipio="355030")
        assert "Resumo da rede de saúde" in result
        assert "Hospital Geral" in result
        assert "Centro de Saúde/UBS" in result


# ---------------------------------------------------------------------------
# comparar_municipios
# ---------------------------------------------------------------------------


class TestCompararMunicipios:
    @pytest.mark.asyncio
    async def test_formats_comparison(self) -> None:
        ctx = _mock_ctx()
        ctx.report_progress = AsyncMock()
        mock_estab = [Estabelecimento()]
        mock_leitos = [Leito(existente=10, sus=8)]
        with (
            patch(
                f"{CLIENT_MODULE}.buscar_estabelecimentos",
                new_callable=AsyncMock,
                return_value=mock_estab,
            ),
            patch(
                f"{CLIENT_MODULE}.consultar_leitos",
                new_callable=AsyncMock,
                return_value=mock_leitos,
            ),
        ):
            result = await tools.comparar_municipios(
                ctx,
                codigos_municipios=["355030", "330455"],
            )
        assert "Comparação" in result
        assert "355030" in result
        assert "330455" in result

    @pytest.mark.asyncio
    async def test_less_than_2(self) -> None:
        ctx = _mock_ctx()
        result = await tools.comparar_municipios(ctx, codigos_municipios=["355030"])
        assert "pelo menos 2" in result

    @pytest.mark.asyncio
    async def test_more_than_5(self) -> None:
        ctx = _mock_ctx()
        result = await tools.comparar_municipios(
            ctx,
            codigos_municipios=["1", "2", "3", "4", "5", "6"],
        )
        assert "Máximo de 5" in result


# ---------------------------------------------------------------------------
# HttpClientError handling
# ---------------------------------------------------------------------------


class TestApiErrorHandling:
    """Test that tools return friendly messages when the API is unavailable."""

    @pytest.mark.asyncio
    async def test_buscar_estabelecimentos_api_error(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_estabelecimentos",
            new_callable=AsyncMock,
            side_effect=HttpClientError("HTTP 404"),
        ):
            result = await tools.buscar_estabelecimentos(ctx)
        assert "indisponível" in result

    @pytest.mark.asyncio
    async def test_consultar_leitos_api_error(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_leitos",
            new_callable=AsyncMock,
            side_effect=HttpClientError("HTTP 404"),
        ):
            result = await tools.consultar_leitos(ctx)
        assert "indisponível" in result

    @pytest.mark.asyncio
    async def test_buscar_estabelecimento_por_cnes_api_error(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_estabelecimento_por_cnes",
            new_callable=AsyncMock,
            side_effect=HttpClientError("HTTP 404"),
        ):
            result = await tools.buscar_estabelecimento_por_cnes(ctx, cnes="1234567")
        assert "indisponível" in result

    @pytest.mark.asyncio
    async def test_resumo_partial_failure(self) -> None:
        """resumo_rede_municipal handles partial API failures gracefully."""
        ctx = _mock_ctx()
        mock_estab = [Estabelecimento(descricao_tipo="UBS")]
        with (
            patch(
                f"{CLIENT_MODULE}.buscar_estabelecimentos",
                new_callable=AsyncMock,
                return_value=mock_estab,
            ),
            patch(
                f"{CLIENT_MODULE}.consultar_leitos",
                new_callable=AsyncMock,
                side_effect=HttpClientError("leitos offline"),
            ),
        ):
            result = await tools.resumo_rede_municipal(ctx, codigo_municipio="355030")
        assert "Resumo da rede" in result
        assert "leitos" in result.lower()

    @pytest.mark.asyncio
    async def test_resumo_total_failure(self) -> None:
        """resumo_rede_municipal returns error when all APIs fail."""
        ctx = _mock_ctx()
        with (
            patch(
                f"{CLIENT_MODULE}.buscar_estabelecimentos",
                new_callable=AsyncMock,
                side_effect=HttpClientError("offline"),
            ),
            patch(
                f"{CLIENT_MODULE}.consultar_leitos",
                new_callable=AsyncMock,
                side_effect=HttpClientError("offline"),
            ),
        ):
            result = await tools.resumo_rede_municipal(ctx, codigo_municipio="355030")
        assert "indisponível" in result

    @pytest.mark.asyncio
    async def test_comparar_municipios_partial_failure(self) -> None:
        """comparar_municipios handles partial failures per municipality."""
        ctx = _mock_ctx()
        ctx.report_progress = AsyncMock()
        with (
            patch(
                f"{CLIENT_MODULE}.buscar_estabelecimentos",
                new_callable=AsyncMock,
                return_value=[Estabelecimento()],
            ),
            patch(
                f"{CLIENT_MODULE}.consultar_leitos",
                new_callable=AsyncMock,
                side_effect=HttpClientError("offline"),
            ),
        ):
            result = await tools.comparar_municipios(ctx, codigos_municipios=["355030", "330455"])
        assert "Comparação" in result
        assert "indisponíveis" in result.lower()
