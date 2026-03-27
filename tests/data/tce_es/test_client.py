"""Tests for the TCE-ES HTTP client."""

from __future__ import annotations

import json

import httpx
import pytest
import respx

from mcp_brasil.data.tce_es import client
from mcp_brasil.data.tce_es.constants import CKAN_BASE

CKAN_URL = CKAN_BASE

# ---------------------------------------------------------------------------
# buscar_licitacoes
# ---------------------------------------------------------------------------


class TestBuscarLicitacoes:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_licitacoes(self) -> None:
        respx.get(CKAN_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "success": True,
                    "result": {
                        "total": 1,
                        "records": [
                            {
                                "_id": 1,
                                "Modalidade": "Pregão Eletrônico",
                                "NumeroEdital": "90001",
                                "AnoEdital": 2024,
                                "Objeto": "Aquisição de material de TI",
                                "DataAbertura": "2024-03-01T00:00:00",
                                "ValorReferencia": "R$ 50.000,00",
                                "ValorHomologado": "R$ 48.000,00",
                                "Situacao": "Homologado",
                            }
                        ],
                    },
                },
            )
        )
        licitacoes, total = await client.buscar_licitacoes(q="material")
        assert total == 1
        assert len(licitacoes) == 1
        lic = licitacoes[0]
        assert lic.Modalidade == "Pregão Eletrônico"
        assert lic.NumeroEdital == "90001"
        assert lic.Situacao == "Homologado"

    @pytest.mark.asyncio
    @respx.mock
    async def test_filters_by_ano(self) -> None:
        respx.get(CKAN_URL).mock(
            return_value=httpx.Response(
                200,
                json={"success": True, "result": {"total": 0, "records": []}},
            )
        )
        licitacoes, total = await client.buscar_licitacoes(ano=2024)
        assert licitacoes == []
        assert total == 0
        # Verifica que o parâmetro filters foi passado com AnoEdital
        request = respx.calls[0].request
        params = dict(request.url.params)
        assert "filters" in params
        filters = json.loads(params["filters"])
        assert filters["AnoEdital"] == 2024

    @pytest.mark.asyncio
    @respx.mock
    async def test_skips_id_field(self) -> None:
        """O campo _id do CKAN não deve ser passado ao schema."""
        respx.get(CKAN_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "success": True,
                    "result": {
                        "total": 1,
                        "records": [{"_id": 99, "Modalidade": "Dispensa", "Objeto": "Serviços"}],
                    },
                },
            )
        )
        licitacoes, _ = await client.buscar_licitacoes()
        assert licitacoes[0].Modalidade == "Dispensa"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(CKAN_URL).mock(
            return_value=httpx.Response(
                200,
                json={"success": True, "result": {"total": 0, "records": []}},
            )
        )
        licitacoes, total = await client.buscar_licitacoes()
        assert licitacoes == []
        assert total == 0


# ---------------------------------------------------------------------------
# buscar_contratos
# ---------------------------------------------------------------------------


class TestBuscarContratos:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_contratos(self) -> None:
        respx.get(CKAN_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "success": True,
                    "result": {
                        "total": 2,
                        "records": [
                            {
                                "_id": 1,
                                "ContratoNumero": "1",
                                "ContratoAno": 2024,
                                "Modalidade": "Pregão",
                                "ResumoObjeto": "Fornecimento de material",
                                "FornecedorNome": "EMPRESA XYZ LTDA",
                                "FornecedorDocumento": "12345678000199",
                                "TermoOriginalValorGlobal": "100000.00",
                                "VigenciaAtualValorGlobal": "120000.00",
                            }
                        ],
                    },
                },
            )
        )
        contratos, total = await client.buscar_contratos(ano=2024)
        assert total == 2
        assert len(contratos) == 1
        c = contratos[0]
        assert c.ContratoNumero == "1"
        assert c.FornecedorNome == "EMPRESA XYZ LTDA"
        assert c.VigenciaAtualValorGlobal == "120000.00"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(CKAN_URL).mock(
            return_value=httpx.Response(
                200,
                json={"success": True, "result": {"total": 0, "records": []}},
            )
        )
        contratos, total = await client.buscar_contratos()
        assert contratos == []
        assert total == 0


# ---------------------------------------------------------------------------
# buscar_contratacoes_municipios
# ---------------------------------------------------------------------------


class TestBuscarContratacoesMunicipios:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_contratacoes(self) -> None:
        respx.get(CKAN_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "success": True,
                    "result": {
                        "total": 127297,
                        "records": [
                            {
                                "_id": 1,
                                "NomeUnidadeGestoraReferencia": "PREFEITURA DE VITÓRIA",
                                "NomeEsferaAdministrativa": "Municipal",
                                "ObjetoContratacao": "Reforma de escola",
                                "ModalidadeLicitacao": "Tomada de Preços",
                                "ValorEstimado": "500000.00",
                                "ValorTotalContratacao": "480000.00",
                                "AnoReferencia": 2024,
                                "SituacaoContratacao": "Concluído",
                            }
                        ],
                    },
                },
            )
        )
        contratacoes, total = await client.buscar_contratacoes_municipios(ano_referencia=2024)
        assert total == 127297
        assert len(contratacoes) == 1
        c = contratacoes[0]
        assert c.NomeUnidadeGestoraReferencia == "PREFEITURA DE VITÓRIA"
        assert c.AnoReferencia == "2024"

    @pytest.mark.asyncio
    @respx.mock
    async def test_filters_by_esfera(self) -> None:
        respx.get(CKAN_URL).mock(
            return_value=httpx.Response(
                200,
                json={"success": True, "result": {"total": 0, "records": []}},
            )
        )
        _, _ = await client.buscar_contratacoes_municipios(esfera="Municipal")
        request = respx.calls[0].request
        params = dict(request.url.params)
        filters = json.loads(params["filters"])
        assert filters["NomeEsferaAdministrativa"] == "Municipal"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(CKAN_URL).mock(
            return_value=httpx.Response(
                200,
                json={"success": True, "result": {"total": 0, "records": []}},
            )
        )
        contratacoes, total = await client.buscar_contratacoes_municipios()
        assert contratacoes == []
        assert total == 0


# ---------------------------------------------------------------------------
# buscar_obras
# ---------------------------------------------------------------------------


class TestBuscarObras:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_obras(self) -> None:
        respx.get(CKAN_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "success": True,
                    "result": {
                        "total": 1250,
                        "records": [
                            {
                                "_id": 1,
                                "Licitacao": "PE 001/2023",
                                "Contrato": "CT-001/2023",
                                "DataAssinaturaContrato": "2023-03-15",
                                "Empresa": "CONSTRUTORA ABC LTDA",
                                "EmpresaCNPJ": "12345678000199",
                                "ValorInicial": "1.282.499,39",
                                "Situacao": "Em Execução",
                            }
                        ],
                    },
                },
            )
        )
        obras, total = await client.buscar_obras(q="ABC")
        assert total == 1250
        assert len(obras) == 1
        o = obras[0]
        assert o.Empresa == "CONSTRUTORA ABC LTDA"
        assert o.Situacao == "Em Execução"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(CKAN_URL).mock(
            return_value=httpx.Response(
                200,
                json={"success": True, "result": {"total": 0, "records": []}},
            )
        )
        obras, total = await client.buscar_obras()
        assert obras == []
        assert total == 0
