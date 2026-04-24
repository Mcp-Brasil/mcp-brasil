"""Tests for the SICONFI HTTP client (respx-mocked)."""

from __future__ import annotations

import httpx
import pytest
import respx

from mcp_brasil.data.siconfi import client
from mcp_brasil.data.siconfi.constants import SICONFI_API_BASE


@pytest.mark.asyncio
@respx.mock
async def test_listar_entes_parses_items() -> None:
    url = f"{SICONFI_API_BASE}/entes"
    respx.get(url).mock(
        return_value=httpx.Response(
            200,
            json={
                "items": [
                    {
                        "cod_ibge": 3550308,
                        "ente": "São Paulo",
                        "capital": 1,
                        "regiao": "Sudeste",
                        "uf": "SP",
                        "esfera": "M",
                        "populacao": 12000000,
                    }
                ],
                "count": 1,
            },
        )
    )
    entes = await client.listar_entes()
    assert len(entes) == 1
    assert entes[0].ente == "São Paulo"
    assert entes[0].cod_ibge == 3550308


@pytest.mark.asyncio
@respx.mock
async def test_consultar_rreo_passes_params() -> None:
    url = f"{SICONFI_API_BASE}/rreo"
    route = respx.get(url).mock(
        return_value=httpx.Response(
            200,
            json={
                "items": [
                    {
                        "exercicio": 2024,
                        "conta": "Receita",
                        "valor": 1000.0,
                        "anexo": "RREO-Anexo 03",
                    }
                ]
            },
        )
    )
    itens = await client.consultar_rreo(
        an_exercicio=2024,
        nr_periodo=6,
        co_tipo_demonstrativo="RREO",
        id_ente=3550308,
        no_anexo="RREO-Anexo 03",
    )
    assert len(itens) == 1
    assert itens[0].valor == 1000.0
    # Confirm params went through
    assert route.called
    called_url = str(route.calls[0].request.url)
    assert "an_exercicio=2024" in called_url
    assert "id_ente=3550308" in called_url


@pytest.mark.asyncio
@respx.mock
async def test_get_items_handles_non_list_payload() -> None:
    url = f"{SICONFI_API_BASE}/entes"
    respx.get(url).mock(return_value=httpx.Response(200, json={"error": "no items"}))
    entes = await client.listar_entes()
    assert entes == []


@pytest.mark.asyncio
@respx.mock
async def test_listar_anexos() -> None:
    url = f"{SICONFI_API_BASE}/anexos-relatorios"
    respx.get(url).mock(
        return_value=httpx.Response(
            200,
            json={
                "items": [
                    {
                        "esfera": "M",
                        "co_tipo_demonstrativo": "RREO",
                        "no_anexo": "RREO-Anexo 01",
                        "de_anexo": "Balanço",
                    }
                ]
            },
        )
    )
    anexos = await client.listar_anexos()
    assert anexos[0].no_anexo == "RREO-Anexo 01"
