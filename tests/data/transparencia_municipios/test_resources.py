"""Tests for resources and prompts in transparencia_municipios."""

import json

import pytest

from mcp_brasil.data.transparencia_municipios import prompts, resources


@pytest.mark.asyncio
async def test_estados_resource() -> None:
    data = await resources.estados_brasileiros()
    parsed = json.loads(data)
    assert len(parsed) == 27
    assert any(e["uf"] == "SP" for e in parsed)


@pytest.mark.asyncio
async def test_estatisticas_resource() -> None:
    data = await resources.estatisticas_nacionais_resource()
    parsed = json.loads(data)
    assert parsed["total_municipios"] == 5571
    assert parsed["total_ufs"] == 27


@pytest.mark.asyncio
async def test_capitais_resource() -> None:
    data = await resources.capitais_brasileiras()
    parsed = json.loads(data)
    assert len(parsed) == 27
    nomes = {c["nome"] for c in parsed}
    assert "São Paulo" in nomes
    assert "Brasília" in nomes


def test_prompts() -> None:
    p1 = prompts.auditoria_transparencia_municipio("Campinas", "SP")
    assert "Campinas" in p1
    assert "SICONFI" in p1

    p2 = prompts.comparativo_transparencia_regional("SP, RJ")
    assert "SP, RJ" in p2
    assert "Radar Nacional" in p2
