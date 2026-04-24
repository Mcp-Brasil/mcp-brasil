"""Integration tests for bcb_olinda."""

from __future__ import annotations

import httpx
import pytest
import respx
from fastmcp import Client

from mcp_brasil.data.bcb_olinda import FEATURE_META
from mcp_brasil.data.bcb_olinda.constants import EXPECT_BASE, PTAX_BASE
from mcp_brasil.data.bcb_olinda.server import mcp


def test_feature_meta() -> None:
    assert FEATURE_META.name == "bcb_olinda"


@pytest.mark.asyncio
async def test_lists_tools() -> None:
    async with Client(mcp) as c:
        tools = await c.list_tools()
    names = {t.name for t in tools}
    assert {"ptax_dolar", "focus_anual", "focus_selic"} <= names


@pytest.mark.asyncio
@respx.mock
async def test_ptax_dolar() -> None:
    respx.get(f"{PTAX_BASE}/CotacaoDolarDia(dataCotacao=@dataCotacao)").mock(
        return_value=httpx.Response(
            200,
            json={
                "value": [
                    {
                        "cotacaoCompra": 5.0,
                        "cotacaoVenda": 5.01,
                        "dataHoraCotacao": "2026-04-23 13:03:26.089",
                    }
                ]
            },
        )
    )
    async with Client(mcp) as c:
        r = await c.call_tool("ptax_dolar", {"data": "04-23-2026"})
    data = getattr(r, "data", None) or str(r)
    assert "5.0" in data


@pytest.mark.asyncio
@respx.mock
async def test_focus_selic() -> None:
    respx.get(f"{EXPECT_BASE}/ExpectativasMercadoSelic").mock(
        return_value=httpx.Response(
            200,
            json={
                "value": [
                    {
                        "Indicador": "Selic",
                        "Data": "2026-04-17",
                        "Reuniao": "R2/2028",
                        "Media": 10.74,
                        "Mediana": 10.5,
                        "DesvioPadrao": 1.16,
                        "Minimo": 7.0,
                        "Maximo": 13.5,
                        "numeroRespondentes": 69,
                    }
                ]
            },
        )
    )
    async with Client(mcp) as c:
        r = await c.call_tool("focus_selic", {})
    data = getattr(r, "data", None) or str(r)
    assert "R2/2028" in data
    assert "10.5" in data


@pytest.mark.asyncio
async def test_listar_indicadores_offline() -> None:
    async with Client(mcp) as c:
        r = await c.call_tool("listar_indicadores_focus", {})
    data = getattr(r, "data", None) or str(r)
    assert "IPCA" in data
    assert "Selic" in data


@pytest.mark.asyncio
async def test_focus_indicador_invalido() -> None:
    async with Client(mcp) as c:
        r = await c.call_tool("focus_anual", {"indicador": "XYZ_INEXISTENTE"})
    data = getattr(r, "data", None) or str(r)
    assert "não reconhecido" in data
