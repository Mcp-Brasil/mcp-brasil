"""Tests for the DataJud MPU tool functions.

Tools are tested by mocking client functions (never HTTP).
"""

from unittest.mock import AsyncMock, patch

import pytest

from mcp_brasil.data.datajud import tools
from mcp_brasil.data.datajud.schemas import Movimentacao, MPUEstatisticas, MPUProcesso

MODULE = "mcp_brasil.data.datajud.client"


class TestBuscarMedidasProtetivas:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            MPUProcesso(
                numero="0001234-56.2024.8.26.0100",
                classe_codigo=1268,
                classe_nome="MPU (Maria da Penha) — Criminal",
                tribunal="TJSP",
                orgao_julgador="1ª Vara Criminal",
                data_ajuizamento="2024-06-15",
            )
        ]
        with patch(
            f"{MODULE}.buscar_medidas_protetivas",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_medidas_protetivas(tribunal="tjsp")
        assert "0001234" in result
        assert "Maria da Penha" in result
        assert "TJSP" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(
            f"{MODULE}.buscar_medidas_protetivas",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.buscar_medidas_protetivas(tribunal="tjpi")
        assert "Nenhuma MPU" in result

    @pytest.mark.asyncio
    async def test_passes_lei_filter(self) -> None:
        with patch(
            f"{MODULE}.buscar_medidas_protetivas",
            new_callable=AsyncMock,
            return_value=[],
        ) as mocked:
            await tools.buscar_medidas_protetivas(
                tribunal="tjsp",
                lei="maria_penha",
                data_inicio="2024-01-01",
            )
        mocked.assert_awaited_once_with(
            tribunal="tjsp",
            data_inicio="2024-01-01",
            data_fim=None,
            lei="maria_penha",
            size=10,
        )


class TestBuscarMpuConcedidas:
    @pytest.mark.asyncio
    async def test_formats_table_with_movimentos(self) -> None:
        mock_data = [
            MPUProcesso(
                numero="0005678",
                classe_codigo=1268,
                classe_nome="MPU (Maria da Penha) — Criminal",
                orgao_julgador="2ª Vara",
                data_ajuizamento="2024-07-01",
                movimentos=[
                    Movimentacao(
                        data="2024-07-02",
                        nome="Concedida medida protetiva",
                        codigo=15486,
                    )
                ],
            )
        ]
        with patch(
            f"{MODULE}.buscar_mpu_concedidas",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_mpu_concedidas(
                tribunal="tjsp",
                destinatario="mulher",
            )
        assert "0005678" in result
        assert "Concedidas" in result
        assert "mulher" in result
        assert "Concedida medida protetiva" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(
            f"{MODULE}.buscar_mpu_concedidas",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.buscar_mpu_concedidas(tribunal="tjpi")
        assert "Nenhuma MPU concedida" in result


class TestBuscarMpuPorTipo:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            MPUProcesso(
                numero="0009999",
                orgao_julgador="3ª Vara",
                data_ajuizamento="2024-08-01",
                movimentos=[
                    Movimentacao(
                        data="2024-08-02",
                        nome="Concedida medida protetiva",
                        codigo=15486,
                    )
                ],
            )
        ]
        with patch(
            f"{MODULE}.buscar_mpu_por_tipo",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_mpu_por_tipo(
                tribunal="tjsp",
                tipo_medida="afastamento_lar",
            )
        assert "0009999" in result
        assert "Afastamento Lar" in result

    @pytest.mark.asyncio
    async def test_invalid_tipo(self) -> None:
        result = await tools.buscar_mpu_por_tipo(tipo_medida="tipo_invalido")
        assert "não reconhecido" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(
            f"{MODULE}.buscar_mpu_por_tipo",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.buscar_mpu_por_tipo(
                tribunal="tjsp",
                tipo_medida="proibicao_aproximacao",
            )
        assert "Nenhuma MPU do tipo" in result


class TestEstatisticasMpu:
    @pytest.mark.asyncio
    async def test_formats_statistics(self) -> None:
        mock_stats = MPUEstatisticas(
            total=150,
            tribunal="tjsp",
            periodo="2024",
            por_classe={
                "MPU (Maria da Penha) — Criminal": 120,
                "MPU (Maria da Penha) — Cível": 30,
            },
            por_decisao={
                "Concedida medida protetiva": 100,
                "Não concedida medida protetiva": 50,
            },
        )
        with patch(
            f"{MODULE}.estatisticas_mpu",
            new_callable=AsyncMock,
            return_value=mock_stats,
        ):
            result = await tools.estatisticas_mpu(tribunal="tjsp", ano=2024)
        assert "150" in result
        assert "TJSP" in result
        assert "2024" in result
        assert "Criminal" in result
        assert "Concedida" in result

    @pytest.mark.asyncio
    async def test_empty_stats(self) -> None:
        mock_stats = MPUEstatisticas(total=0, tribunal="tjpi", periodo="2024")
        with patch(
            f"{MODULE}.estatisticas_mpu",
            new_callable=AsyncMock,
            return_value=mock_stats,
        ):
            result = await tools.estatisticas_mpu(tribunal="tjpi", ano=2024)
        assert "0" in result
        assert "Nenhuma MPU encontrada" in result


class TestTimelineMpu:
    @pytest.mark.asyncio
    async def test_formats_timeline(self) -> None:
        mock_data = [
            Movimentacao(data="2024-06-01", nome="Concedida medida protetiva", codigo=15486),
            Movimentacao(data="2024-09-01", nome="Prorrogada a medida protetiva", codigo=15490),
        ]
        with patch(
            f"{MODULE}.timeline_mpu",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.timeline_mpu("0001234", tribunal="tjsp")
        assert "Concedida" in result
        assert "Prorrogada" in result
        assert "Timeline MPU" in result
        assert "2 movimentos" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(
            f"{MODULE}.timeline_mpu",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.timeline_mpu("9999999", tribunal="tjsp")
        assert "Nenhum movimento de MPU" in result


class TestFormatMpuMovimentos:
    def test_with_movimentos(self) -> None:
        movs = [
            Movimentacao(nome="Concedida medida protetiva", codigo=15486),
            Movimentacao(nome="Prorrogada a medida protetiva", codigo=15490),
        ]
        result = tools._format_mpu_movimentos(movs)
        assert "Concedida" in result
        assert "Prorrogada" in result

    def test_empty(self) -> None:
        assert tools._format_mpu_movimentos(None) == "—"
        assert tools._format_mpu_movimentos([]) == "—"
