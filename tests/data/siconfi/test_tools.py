"""Tests for SICONFI tool functions (client mocked)."""

from __future__ import annotations

from unittest.mock import AsyncMock, patch

import pytest

from mcp_brasil.data.siconfi import tools
from mcp_brasil.data.siconfi.schemas import AnexoRelatorio, Ente, ItemDeclaracao


@pytest.mark.asyncio
async def test_listar_entes_filters_uf() -> None:
    fake = [
        Ente(cod_ibge=1, ente="A", uf="SP", esfera="M"),
        Ente(cod_ibge=2, ente="B", uf="RJ", esfera="M"),
    ]
    with patch(
        "mcp_brasil.data.siconfi.client.listar_entes",
        new=AsyncMock(return_value=fake),
    ):
        out = await tools.listar_entes(uf="SP")
    assert "| 1 | A " in out
    assert "| 2 |" not in out


@pytest.mark.asyncio
async def test_listar_entes_empty_message() -> None:
    with patch(
        "mcp_brasil.data.siconfi.client.listar_entes",
        new=AsyncMock(return_value=[]),
    ):
        out = await tools.listar_entes()
    assert "Nenhum ente" in out


@pytest.mark.asyncio
async def test_consultar_rreo_formats() -> None:
    itens = [
        ItemDeclaracao(
            exercicio=2024,
            conta="Receita Corrente Líquida",
            coluna="Até o Bimestre",
            valor=1_000_000.0,
            anexo="RREO-Anexo 03",
        )
    ]
    with patch(
        "mcp_brasil.data.siconfi.client.consultar_rreo",
        new=AsyncMock(return_value=itens),
    ):
        out = await tools.consultar_rreo(
            exercicio=2024, periodo=6, ente_id=3550308, anexo="RREO-Anexo 03"
        )
    assert "RREO 2024" in out
    assert "Receita Corrente Líquida" in out
    assert "R$" in out


@pytest.mark.asyncio
async def test_consultar_rgf_uses_simplificado_flag() -> None:
    captured: dict[str, object] = {}

    async def fake(**kwargs: object) -> list[ItemDeclaracao]:
        captured.update(kwargs)
        return []

    with patch("mcp_brasil.data.siconfi.client.consultar_rgf", new=fake):
        await tools.consultar_rgf(exercicio=2024, periodo=3, ente_id=1, simplificado=True)
    assert captured["co_tipo_demonstrativo"] == "RGF Simplificado"


@pytest.mark.asyncio
async def test_extrato_entregas_empty() -> None:
    with patch(
        "mcp_brasil.data.siconfi.client.consultar_extrato_entregas",
        new=AsyncMock(return_value=[]),
    ):
        out = await tools.extrato_entregas(ente_id=1, ano=2024)
    assert "Nenhuma entrega" in out


@pytest.mark.asyncio
async def test_listar_anexos_relatorios_table() -> None:
    anexos = [
        AnexoRelatorio(
            esfera="M", co_tipo_demonstrativo="RREO", no_anexo="RREO-Anexo 03", de_anexo="RCL"
        )
    ]
    with patch(
        "mcp_brasil.data.siconfi.client.listar_anexos",
        new=AsyncMock(return_value=anexos),
    ):
        out = await tools.listar_anexos_relatorios()
    assert "RREO-Anexo 03" in out


@pytest.mark.asyncio
async def test_anexos_populares_runs_offline() -> None:
    out = await tools.anexos_populares()
    assert "RREO-Anexo 01" in out
    assert "RGF-Anexo 01" in out
