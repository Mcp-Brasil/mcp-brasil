"""Tests for the Compras HTTP client."""

import httpx
import pytest
import respx

from mcp_brasil.compras import client
from mcp_brasil.compras.constants import ATAS_URL, CONTRATACOES_URL, CONTRATOS_URL

# ---------------------------------------------------------------------------
# buscar_contratacoes
# ---------------------------------------------------------------------------


class TestBuscarContratacoes:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_contratacoes(self) -> None:
        respx.get(CONTRATACOES_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "totalRegistros": 1,
                    "data": [
                        {
                            "orgaoEntidade": {
                                "cnpj": "00394460000141",
                                "razaoSocial": "Ministério da Educação",
                                "ufSigla": "DF",
                                "municipioNome": "Brasília",
                                "esferaNome": "Federal",
                            },
                            "anoCompra": 2024,
                            "sequencialCompra": 1,
                            "numeroControlePNCP": "00394460000141-1-000001/2024",
                            "objetoCompra": "Aquisição de computadores",
                            "modalidadeId": 1,
                            "modalidadeNome": "Pregão eletrônico",
                            "situacaoCompraId": 1,
                            "situacaoCompraNome": "Publicada",
                            "valorTotalEstimado": 500000.0,
                            "valorTotalHomologado": 480000.0,
                            "dataPublicacaoPncp": "2024-03-15",
                            "dataAberturaProposta": "2024-04-01",
                            "linkPncp": "https://pncp.gov.br/app/editais/123",
                        }
                    ],
                },
            )
        )
        result = await client.buscar_contratacoes(query="computadores")
        assert result.total == 1
        assert len(result.contratacoes) == 1
        c = result.contratacoes[0]
        assert c.orgao_cnpj == "00394460000141"
        assert c.orgao_nome == "Ministério da Educação"
        assert c.objeto == "Aquisição de computadores"
        assert c.modalidade_id == 1
        assert c.valor_estimado == 500000.0
        assert c.valor_homologado == 480000.0
        assert c.uf == "DF"
        assert c.municipio == "Brasília"
        assert c.link_pncp == "https://pncp.gov.br/app/editais/123"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(CONTRATACOES_URL).mock(
            return_value=httpx.Response(
                200,
                json={"totalRegistros": 0, "data": []},
            )
        )
        result = await client.buscar_contratacoes(query="inexistente")
        assert result.total == 0
        assert result.contratacoes == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_fallback_resultado_key(self) -> None:
        respx.get(CONTRATACOES_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "count": 1,
                    "resultado": [
                        {
                            "orgaoEntidade": {"cnpj": "11111111000100"},
                            "objetoCompra": "Teste fallback",
                        }
                    ],
                },
            )
        )
        result = await client.buscar_contratacoes(query="teste")
        assert result.total == 1
        assert len(result.contratacoes) == 1
        assert result.contratacoes[0].objeto == "Teste fallback"


# ---------------------------------------------------------------------------
# buscar_contratos
# ---------------------------------------------------------------------------


class TestBuscarContratos:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_contratos(self) -> None:
        respx.get(CONTRATOS_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "totalRegistros": 1,
                    "data": [
                        {
                            "orgaoEntidade": {
                                "cnpj": "00394460000141",
                                "razaoSocial": "Ministério da Saúde",
                            },
                            "fornecedor": {
                                "cnpj": "12345678000199",
                                "razaoSocial": "Empresa Pharma LTDA",
                            },
                            "numeroContratoEmpenho": "2024/001",
                            "objetoContrato": "Fornecimento de medicamentos",
                            "valorInicial": 100000.0,
                            "valorFinal": 95000.0,
                            "dataVigenciaInicio": "2024-01-01",
                            "dataVigenciaFim": "2024-12-31",
                            "dataPublicacaoPncp": "2024-01-10",
                            "nomeStatus": "Vigente",
                        }
                    ],
                },
            )
        )
        result = await client.buscar_contratos(query="medicamentos")
        assert result.total == 1
        assert len(result.contratos) == 1
        c = result.contratos[0]
        assert c.orgao_cnpj == "00394460000141"
        assert c.orgao_nome == "Ministério da Saúde"
        assert c.fornecedor_cnpj == "12345678000199"
        assert c.fornecedor_nome == "Empresa Pharma LTDA"
        assert c.numero_contrato == "2024/001"
        assert c.objeto == "Fornecimento de medicamentos"
        assert c.valor_inicial == 100000.0
        assert c.valor_final == 95000.0
        assert c.vigencia_inicio == "2024-01-01"
        assert c.vigencia_fim == "2024-12-31"
        assert c.situacao == "Vigente"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(CONTRATOS_URL).mock(
            return_value=httpx.Response(
                200,
                json={"totalRegistros": 0, "data": []},
            )
        )
        result = await client.buscar_contratos(query="inexistente")
        assert result.total == 0
        assert result.contratos == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_fornecedor_cpfcnpj_fallback(self) -> None:
        respx.get(CONTRATOS_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "totalRegistros": 1,
                    "data": [
                        {
                            "orgaoEntidade": {},
                            "fornecedor": {
                                "cpfCnpj": "99988877000166",
                                "nomeRazaoSocial": "Fornecedor Alt",
                            },
                            "objetoContrato": "Teste fallback fornecedor",
                        }
                    ],
                },
            )
        )
        result = await client.buscar_contratos(query="teste")
        c = result.contratos[0]
        assert c.fornecedor_cnpj == "99988877000166"
        assert c.fornecedor_nome == "Fornecedor Alt"


# ---------------------------------------------------------------------------
# buscar_atas
# ---------------------------------------------------------------------------


class TestBuscarAtas:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_atas(self) -> None:
        respx.get(ATAS_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "totalRegistros": 1,
                    "data": [
                        {
                            "orgaoEntidade": {
                                "cnpj": "00394460000141",
                                "razaoSocial": "Universidade Federal",
                            },
                            "fornecedor": {
                                "cnpj": "98765432000155",
                                "razaoSocial": "Papelaria Central LTDA",
                            },
                            "numeroAtaRegistroPreco": "2024/010",
                            "objetoContrato": "Material de escritório",
                            "valorInicial": 250000.0,
                            "dataVigenciaInicio": "2024-06-01",
                            "dataVigenciaFim": "2025-05-31",
                            "nomeStatus": "Vigente",
                        }
                    ],
                },
            )
        )
        result = await client.buscar_atas(query="escritório")
        assert result.total == 1
        assert len(result.atas) == 1
        a = result.atas[0]
        assert a.orgao_cnpj == "00394460000141"
        assert a.orgao_nome == "Universidade Federal"
        assert a.fornecedor_cnpj == "98765432000155"
        assert a.fornecedor_nome == "Papelaria Central LTDA"
        assert a.numero_ata == "2024/010"
        assert a.objeto == "Material de escritório"
        assert a.valor_total == 250000.0
        assert a.vigencia_inicio == "2024-06-01"
        assert a.vigencia_fim == "2025-05-31"
        assert a.situacao == "Vigente"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(ATAS_URL).mock(
            return_value=httpx.Response(
                200,
                json={"totalRegistros": 0, "data": []},
            )
        )
        result = await client.buscar_atas(query="inexistente")
        assert result.total == 0
        assert result.atas == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_ata_fields_fallback(self) -> None:
        respx.get(ATAS_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "totalRegistros": 1,
                    "data": [
                        {
                            "orgaoEntidade": {},
                            "fornecedor": {
                                "cpfCnpj": "11122233000144",
                                "nomeRazaoSocial": "Fornecedor Ata Alt",
                            },
                            "numeroAta": "ATA-001",
                            "objetoAta": "Objeto via campo ata",
                            "valorTotal": 300000.0,
                        }
                    ],
                },
            )
        )
        result = await client.buscar_atas(query="ata")
        a = result.atas[0]
        assert a.fornecedor_cnpj == "11122233000144"
        assert a.fornecedor_nome == "Fornecedor Ata Alt"
        assert a.numero_ata == "ATA-001"
        assert a.objeto == "Objeto via campo ata"
        assert a.valor_total == 300000.0
