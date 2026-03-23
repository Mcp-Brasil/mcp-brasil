"""Integration tests for the TSE feature using fastmcp.Client."""

from unittest.mock import AsyncMock, patch

import pytest
from fastmcp import Client

from mcp_brasil.data.tse.server import mcp

CLIENT_MODULE = "mcp_brasil.data.tse.client"


class TestToolsRegistered:
    @pytest.mark.asyncio
    async def test_all_15_tools_registered(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            names = {t.name for t in tool_list}
            expected = {
                # DivulgaCandContas (9)
                "anos_eleitorais",
                "listar_eleicoes",
                "listar_eleicoes_suplementares",
                "listar_estados_suplementares",
                "listar_cargos",
                "listar_candidatos",
                "buscar_candidato",
                "resultado_eleicao",
                "consultar_prestacao_contas",
                # CDN de Resultados (6)
                "resultado_nacional",
                "resultado_por_estado",
                "resultado_por_municipio",
                "mapa_resultado_estados",
                "apuracao_status",
                "listar_municipios_eleitorais",
            }
            assert expected.issubset(names), f"Missing: {expected - names}"

    @pytest.mark.asyncio
    async def test_tools_have_docstrings(self) -> None:
        async with Client(mcp) as c:
            tool_list = await c.list_tools()
            for tool in tool_list:
                assert tool.description, f"Tool {tool.name} has no description"


class TestResourcesRegistered:
    @pytest.mark.asyncio
    async def test_all_2_resources_registered(self) -> None:
        async with Client(mcp) as c:
            resources = await c.list_resources()
            uris = {str(r.uri) for r in resources}
            expected = {"data://cargos-eleitorais", "data://info-api"}
            assert expected.issubset(uris), f"Missing: {expected - uris}"


class TestPromptsRegistered:
    @pytest.mark.asyncio
    async def test_all_2_prompts_registered(self) -> None:
        async with Client(mcp) as c:
            prompts = await c.list_prompts()
            names = {p.name for p in prompts}
            expected = {"analise_candidato", "comparativo_eleicao"}
            assert expected.issubset(names), f"Missing: {expected - names}"


class TestToolExecution:
    @pytest.mark.asyncio
    async def test_anos_eleitorais_e2e(self) -> None:
        with patch(
            f"{CLIENT_MODULE}.anos_eleitorais",
            new_callable=AsyncMock,
            return_value=[2020, 2022],
        ):
            async with Client(mcp) as c:
                result = await c.call_tool("anos_eleitorais", {})
                assert "2020" in result.data
                assert "2022" in result.data

    @pytest.mark.asyncio
    async def test_resultado_eleicao_e2e(self) -> None:
        from mcp_brasil.data.tse.schemas import ResultadoCandidato

        mock_data = [
            ResultadoCandidato(
                nome_urna="CANDIDATO A",
                numero=44,
                partido="PT",
                total_votos=10000,
                percentual="60,00%",
                descricao_totalizacao="Eleito",
            ),
        ]
        with patch(
            f"{CLIENT_MODULE}.resultado_eleicao",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "resultado_eleicao",
                    {"ano": 2020, "municipio": 35157, "eleicao_id": 2030402020, "cargo": 11},
                )
                assert "CANDIDATO A" in result.data
                assert "10.000" in result.data

    @pytest.mark.asyncio
    async def test_resultado_nacional_e2e(self) -> None:
        from mcp_brasil.data.tse.schemas import ResultadoCDN, ResultadoRegiao

        mock_data = ResultadoRegiao(
            uf="BR",
            pct_apurado="100.00",
            total_eleitores=156000000,
            candidatos=[
                ResultadoCDN(nome="LULA", numero="13", votos=60000000, percentual="50.90"),
            ],
        )
        with patch(
            f"{CLIENT_MODULE}.resultado_simplificado",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "resultado_nacional",
                    {"ano": 2022, "cargo": "presidente"},
                )
                assert "LULA" in result.data
                assert "Resultado Nacional" in result.data

    @pytest.mark.asyncio
    async def test_resultado_por_estado_e2e(self) -> None:
        from mcp_brasil.data.tse.schemas import ResultadoCDN, ResultadoRegiao

        mock_data = ResultadoRegiao(
            uf="SP",
            pct_apurado="100.00",
            total_eleitores=34000000,
            candidatos=[
                ResultadoCDN(nome="BOLSONARO", numero="22", votos=12000000, percentual="47.71"),
            ],
        )
        with patch(
            f"{CLIENT_MODULE}.resultado_simplificado",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "resultado_por_estado",
                    {"ano": 2022, "cargo": "presidente", "uf": "SP"},
                )
                assert "BOLSONARO" in result.data
                assert "SP" in result.data

    @pytest.mark.asyncio
    async def test_apuracao_status_e2e(self) -> None:
        from mcp_brasil.data.tse.schemas import ResultadoRegiao

        mock_data = ResultadoRegiao(
            uf="BR",
            pct_apurado="100.00",
            total_secoes=472075,
            total_eleitores=156000000,
            total_comparecimento=124000000,
            total_abstencoes=32000000,
        )
        with patch(
            f"{CLIENT_MODULE}.resultado_simplificado",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "apuracao_status",
                    {"ano": 2022, "cargo": "presidente"},
                )
                assert "Status da Apuração" in result.data
                assert "100.00%" in result.data

    @pytest.mark.asyncio
    async def test_listar_municipios_eleitorais_e2e(self) -> None:
        from mcp_brasil.data.tse.schemas import MunicipioEleitoral

        mock_data = [
            MunicipioEleitoral(
                codigo_tse="71072",
                codigo_ibge="3550308",
                nome="SÃO PAULO",
                capital=True,
                uf="SP",
            ),
        ]
        with patch(
            f"{CLIENT_MODULE}.listar_municipios_eleitorais",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "listar_municipios_eleitorais",
                    {"ano": 2024, "uf": "SP"},
                )
                assert "SÃO PAULO" in result.data
                assert "71072" in result.data

    @pytest.mark.asyncio
    async def test_resultado_por_municipio_e2e(self) -> None:
        from mcp_brasil.data.tse.schemas import ResultadoCDN, ResultadoRegiao

        mock_data = ResultadoRegiao(
            codigo="71072",
            tipo="mu",
            uf="71072",
            pct_apurado="100.00",
            total_eleitores=8000000,
            candidatos=[
                ResultadoCDN(
                    nome="CANDIDATO A",
                    numero="15",
                    votos=3500000,
                    percentual="53.85",
                    eleito=True,
                ),
            ],
        )
        with patch(
            f"{CLIENT_MODULE}.resultado_municipio",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            async with Client(mcp) as c:
                result = await c.call_tool(
                    "resultado_por_municipio",
                    {"ano": 2024, "cargo": "prefeito", "uf": "SP", "cod_tse": "71072"},
                )
                assert "CANDIDATO A" in result.data
                assert "71072" in result.data
