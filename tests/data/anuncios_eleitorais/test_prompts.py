"""Tests for anuncios_eleitorais prompts — unit + Client integration."""

from __future__ import annotations

import pytest
from fastmcp import Client

from mcp_brasil.data.anuncios_eleitorais.prompts import (
    analise_candidato,
    comparar_candidatos,
    panorama_eleitoral,
)
from mcp_brasil.data.anuncios_eleitorais.server import mcp


class TestAnaliseCandidato:
    """Tests for analise_candidato prompt."""

    def test_retorna_instrucoes(self) -> None:
        result = analise_candidato("João Silva")
        assert isinstance(result, str)
        assert "João Silva" in result

    def test_usa_busca_por_termo(self) -> None:
        result = analise_candidato("João Silva")
        assert "buscar_anuncios_eleitorais" in result

    def test_usa_busca_por_pagina_quando_id_fornecido(self) -> None:
        result = analise_candidato("João Silva", pagina_id="123456")
        assert "buscar_anuncios_por_pagina" in result
        assert "123456" in result

    def test_inclui_analise_demografica(self) -> None:
        result = analise_candidato("João Silva")
        assert "analisar_demografia_anuncios" in result


class TestPanoramaEleitoral:
    """Tests for panorama_eleitoral prompt."""

    def test_retorna_instrucoes(self) -> None:
        result = panorama_eleitoral()
        assert isinstance(result, str)
        assert "Brasil" in result

    def test_com_estado(self) -> None:
        result = panorama_eleitoral(estado="São Paulo")
        assert "São Paulo" in result
        assert "buscar_anuncios_por_regiao" in result

    def test_sem_estado(self) -> None:
        result = panorama_eleitoral()
        assert "buscar_anuncios_eleitorais" in result

    def test_com_periodo(self) -> None:
        result = panorama_eleitoral(periodo_inicio="2024-01-01", periodo_fim="2024-12-31")
        assert "2024-01-01" in result
        assert "2024-12-31" in result


class TestCompararCandidatos:
    """Tests for comparar_candidatos prompt."""

    def test_retorna_instrucoes(self) -> None:
        result = comparar_candidatos("Candidato A", "Candidato B")
        assert isinstance(result, str)
        assert "Candidato A" in result
        assert "Candidato B" in result

    def test_inclui_ferramentas(self) -> None:
        result = comparar_candidatos("A", "B")
        assert "buscar_anuncios_eleitorais" in result
        assert "analisar_demografia_anuncios" in result


class TestPromptRegistration:
    """Test prompts are properly registered via Client."""

    @pytest.mark.asyncio
    async def test_prompts_registrados(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = [p.name for p in prompts]
            assert "analise_candidato" in names
            assert "panorama_eleitoral" in names
            assert "comparar_candidatos" in names

    @pytest.mark.asyncio
    async def test_chamar_analise_candidato(self) -> None:
        async with Client(mcp) as c:
            result = await c.get_prompt(
                "analise_candidato",
                arguments={"nome_candidato": "Teste"},
            )
            assert len(result.messages) > 0
            text = result.messages[0].content.text  # type: ignore[union-attr]
            assert "Teste" in text

    @pytest.mark.asyncio
    async def test_chamar_comparar_candidatos(self) -> None:
        async with Client(mcp) as c:
            result = await c.get_prompt(
                "comparar_candidatos",
                arguments={"candidato_a": "A", "candidato_b": "B"},
            )
            assert len(result.messages) > 0
