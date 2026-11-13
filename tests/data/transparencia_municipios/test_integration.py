"""Integration tests for transparencia_municipios using fastmcp.Client."""

import pytest
from fastmcp import Client

from mcp_brasil.data.transparencia_municipios.server import mcp


@pytest.mark.asyncio
async def test_tools_registered() -> None:
    async with Client(mcp) as c:
        tools = await c.list_tools()
        names = {t.name for t in tools}
        expected = {
            "buscar_municipios",
            "buscar_por_cnpj",
            "listar_estados",
            "conectar_municipio",
            "conectar_estado",
            "descobrir_portal_municipio",
            "consultar_servidores_municipio",
            "consultar_despesas_municipio",
            "consultar_receitas_municipio",
            "consultar_licitacoes_municipio",
            "consultar_contratos_municipio",
            "consultar_transparencia_completa",
            "consultar_fiscal",
            "consultar_entregas",
            "pesquisar_transparencia",
            "comparar_transparencia",
            "estatisticas_nacionais",
            "listar_municipios_megasoft",
            "consultar_diarias_municipio",
            "consultar_obras_municipio",
            "consultar_convenios_municipio",
            "consultar_legislacao_municipio",
            "consultar_concursos_municipio",
            "consultar_detalhe_servidor_municipio",
        }
        assert expected.issubset(names), f"Missing: {expected - names}"
        assert len(names) == 24, f"Expected 24 tools, found {len(names)}: {names}"


@pytest.mark.asyncio
async def test_tools_have_docstrings() -> None:
    async with Client(mcp) as c:
        tools = await c.list_tools()
        for t in tools:
            assert t.description, f"Tool {t.name} has no description"


@pytest.mark.asyncio
async def test_resources_registered() -> None:
    async with Client(mcp) as c:
        resources = await c.list_resources()
        uris = {str(r.uri) for r in resources}
        expected = {
            "data://estados",
            "data://estatisticas",
            "data://capitais",
            "data://megasoft-municipios",
        }
        assert expected.issubset(uris), f"Missing: {expected - uris}"


@pytest.mark.asyncio
async def test_prompts_registered() -> None:
    async with Client(mcp) as c:
        prompts = await c.list_prompts()
        names = {p.name for p in prompts}
        expected = {
            "auditoria_transparencia_municipio",
            "comparativo_transparencia_regional",
        }
        assert expected.issubset(names), f"Missing: {expected - names}"


@pytest.mark.asyncio
async def test_e2e_buscar_municipios_call() -> None:
    async with Client(mcp) as c:
        result = await c.call_tool("buscar_municipios", {"query": "Belo Horizonte", "limit": 1})
        text = str(result.content[0].text if hasattr(result.content[0], "text") else result)
        assert "Belo Horizonte" in text
        assert "3106200" in text


@pytest.mark.asyncio
async def test_e2e_conectar_municipio_call() -> None:
    async with Client(mcp) as c:
        result = await c.call_tool("conectar_municipio", {"municipio": "3550308"})
        text = str(result.content[0].text if hasattr(result.content[0], "text") else result)
        assert "São Paulo" in text
        assert "TCM-SP" in text
        assert "transparencia.sao-paulo.sp.gov.br" in text


@pytest.mark.asyncio
async def test_e2e_conectar_estado_call() -> None:
    async with Client(mcp) as c:
        result = await c.call_tool("conectar_estado", {"uf": "RS"})
        text = str(result.content[0].text if hasattr(result.content[0], "text") else result)
        assert "Rio Grande do Sul" in text
        assert "TCE-RS" in text
        assert "transparencia.rs.gov.br" in text


@pytest.mark.asyncio
async def test_e2e_buscar_por_cnpj_call() -> None:
    async with Client(mcp) as c:
        result = await c.call_tool("buscar_por_cnpj", {"cnpj": "01613940000119"})
        text = str(result.content[0].text if hasattr(result.content[0], "text") else result)
        assert "Abadia de Goiás" in text
        assert "01613940000119" in text
        assert "TCM-GO" in text


@pytest.mark.asyncio
async def test_e2e_consultar_fiscal_call() -> None:
    async with Client(mcp) as c:
        result = await c.call_tool(
            "consultar_fiscal",
            {"municipio_ou_ibge": "3550308", "ano": 2023, "tipo": "dca", "limite": 5},
        )
        text = str(result.content[0].text if hasattr(result.content[0], "text") else result)
        assert "SICONFI" in text
        assert "3550308" in text
        assert "R$" in text


@pytest.mark.asyncio
async def test_e2e_serra_da_saudade_servidores_e_transparencia() -> None:
    async with Client(mcp) as c:
        res_serv = await c.call_tool(
            "consultar_servidores_municipio",
            {"municipio": "Serra da Saudade", "uf": "MG", "limite": 3},
        )
        c0 = res_serv.content[0]
        text_serv = str(c0.text if hasattr(c0, "text") else res_serv)
        assert "Serra da Saudade" in text_serv
        assert "Quasar" in text_serv
        assert "R$" in text_serv

        res_full = await c.call_tool(
            "consultar_transparencia_completa",
            {"municipio": "Serra da Saudade", "uf": "MG", "limite": 2},
        )
        f0 = res_full.content[0]
        text_full = str(f0.text if hasattr(f0, "text") else res_full)
        assert "Auditoria Completa de Transparência: Serra da Saudade (MG)" in text_full
        assert "serradasaudade.mg.gov.br" in text_full


@pytest.mark.asyncio
async def test_e2e_megasoft_tools_call() -> None:
    async with Client(mcp) as c:
        res = await c.call_tool("listar_municipios_megasoft", {"uf": "MS"})
        c0 = res.content[0]
        text = str(c0.text if hasattr(c0, "text") else res)
        assert "Tacuru" in text
        assert "5007950" in text
        assert "megasofttransparencia.com.br" in text
