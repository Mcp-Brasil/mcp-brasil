"""Tests for the TCE-CE tool functions."""

from unittest.mock import AsyncMock, MagicMock, patch

from mcp_brasil.data.tce_ce import tools
from mcp_brasil.data.tce_ce.schemas import (
    Contrato,
    ContratoResultado,
    Empenho,
    EmpenhoResultado,
    Licitacao,
    LicitacaoResultado,
    Municipio,
)

CLIENT_MODULE = "mcp_brasil.data.tce_ce.client"


def _mock_ctx() -> MagicMock:
    ctx = MagicMock()
    ctx.info = AsyncMock()
    ctx.warning = AsyncMock()
    return ctx


class TestListarMunicipiosCe:
    async def test_formats_results(self) -> None:
        mock_data = [
            Municipio(
                codigo_municipio="057", nome_municipio="FORTALEZA", codigo_municipio_ibge="2304400"
            ),
            Municipio(codigo_municipio="134", nome_municipio="PENTECOSTE"),
        ]
        with patch(
            f"{CLIENT_MODULE}.listar_municipios", new_callable=AsyncMock, return_value=mock_data
        ):
            result = await tools.listar_municipios_ce(_mock_ctx())
        assert "FORTALEZA" in result
        assert "`057`" in result
        assert "IBGE 2304400" in result
        assert "2 municípios" in result

    async def test_passa_filtro_de_nome(self) -> None:
        with patch(
            f"{CLIENT_MODULE}.listar_municipios", new_callable=AsyncMock, return_value=[]
        ) as m:
            result = await tools.listar_municipios_ce(_mock_ctx(), nome_municipio="PENTECOSTE")
        m.assert_awaited_once_with(nome_municipio="PENTECOSTE")
        assert "Nenhum município" in result


class TestBuscarLicitacoesCe:
    async def test_formats_results(self) -> None:
        mock_data = LicitacaoResultado(
            licitacoes=[
                Licitacao(
                    numero_licitacao="09.002/2026-AD",
                    data_realizacao="2026-08-05",
                    modalidade_licitacao="9",
                    objeto="ADESAO A ATA",
                    valor_orcado_estimado=265141.5,
                    data_homologacao="2026-08-06",
                )
            ],
            total=1,
        )
        with patch(
            f"{CLIENT_MODULE}.buscar_licitacoes", new_callable=AsyncMock, return_value=mock_data
        ):
            result = await tools.buscar_licitacoes_ce(_mock_ctx(), "134", "2026")
        assert "09.002/2026-AD" in result
        assert "265.141,50" in result
        assert "2026-08-06" in result
        assert "deslocamento=" not in result

    async def test_indica_proxima_pagina(self) -> None:
        mock_data = LicitacaoResultado(
            licitacoes=[Licitacao(numero_licitacao=str(i)) for i in range(100)],
            total=100,
            tem_mais=True,
        )
        with patch(
            f"{CLIENT_MODULE}.buscar_licitacoes", new_callable=AsyncMock, return_value=mock_data
        ):
            result = await tools.buscar_licitacoes_ce(_mock_ctx(), "134", "2026", deslocamento=100)
        assert "deslocamento=200" in result
        assert "nesta página (há mais páginas)" in result

    async def test_empty(self) -> None:
        with patch(
            f"{CLIENT_MODULE}.buscar_licitacoes",
            new_callable=AsyncMock,
            return_value=LicitacaoResultado(),
        ):
            result = await tools.buscar_licitacoes_ce(_mock_ctx(), "057", "2026")
        assert "Nenhuma licitação" in result


class TestBuscarContratosCe:
    async def test_avisa_cruzamento_incompleto(self) -> None:
        mock_data = ContratoResultado(
            contratos=[Contrato(numero_contrato="1")], total=1, contratado_incompleto=True
        )
        with patch(
            f"{CLIENT_MODULE}.buscar_contratos", new_callable=AsyncMock, return_value=mock_data
        ):
            result = await tools.buscar_contratos_ce(_mock_ctx(), "057", "2020_2026")
        assert "NÃO significa contratado ausente" in result

    async def test_mostra_contratado(self) -> None:
        mock_data = ContratoResultado(
            contratos=[
                Contrato(
                    numero_contrato="20260826003",
                    data_contrato="2026-08-31",
                    tipo_contrato="L",
                    objeto="Aquisicao de 04 ambulancias",
                    valor_total_contrato=185590.0,
                    data_inicio_vigencia="2026-08-31",
                    data_fim_vigencia="2027-08-31",
                    numero_processo_adm="11.012/2026-PE",
                    nome_contratado="TERRA DO SOL COMERCIO DE VEICULOS LTDA",
                    documento_contratado="17161125000104",
                )
            ],
            total=1,
        )
        with patch(
            f"{CLIENT_MODULE}.buscar_contratos", new_callable=AsyncMock, return_value=mock_data
        ):
            result = await tools.buscar_contratos_ce(_mock_ctx(), "134", "2026")
        assert "TERRA DO SOL" in result
        assert "17161125000104" in result
        assert "185.590,00" in result
        assert "2026-08-31 a 2027-08-31" in result
        assert "11.012/2026-PE" in result

    async def test_empty(self) -> None:
        with patch(
            f"{CLIENT_MODULE}.buscar_contratos",
            new_callable=AsyncMock,
            return_value=ContratoResultado(),
        ):
            result = await tools.buscar_contratos_ce(_mock_ctx(), "057", "2026")
        assert "Nenhum contrato" in result


class TestBuscarEmpenhosCe:
    async def test_formats_results(self) -> None:
        mock_data = EmpenhoResultado(
            empenhos=[
                Empenho(
                    numero_empenho="03080033",
                    data_emissao="2026-08-03",
                    valor_empenho=7586.0,
                    nome_negociante="I3 SOLUCOES LTDA",
                    numero_documento_negociante="03307395000168",
                    historico="locacao de sistema",
                )
            ],
            total=1,
        )
        with patch(
            f"{CLIENT_MODULE}.buscar_empenhos", new_callable=AsyncMock, return_value=mock_data
        ) as m:
            result = await tools.buscar_empenhos_ce(_mock_ctx(), "134", 202608)
        m.assert_awaited_once_with(
            codigo_municipio="134", data_referencia=202608, codigo_orgao=None, deslocamento=0
        )
        assert "I3 SOLUCOES LTDA (03307395000168)" in result
        assert "7.586,00" in result

    async def test_empty(self) -> None:
        with patch(
            f"{CLIENT_MODULE}.buscar_empenhos",
            new_callable=AsyncMock,
            return_value=EmpenhoResultado(),
        ):
            result = await tools.buscar_empenhos_ce(_mock_ctx(), "134", 202608)
        assert "Nenhum empenho" in result
