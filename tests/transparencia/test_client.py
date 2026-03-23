"""Tests for the transparencia HTTP client."""

import logging
from unittest.mock import patch

import httpx
import pytest
import respx

from mcp_brasil.exceptions import AuthError
from mcp_brasil.transparencia import client
from mcp_brasil.transparencia.constants import (
    ACORDOS_LENIENCIA_URL,
    AUTH_ENV_VAR,
    BENEFICIOS_CIDADAO_URL,
    BOLSA_FAMILIA_MUNICIPIO_URL,
    BOLSA_FAMILIA_NIS_URL,
    CARTOES_URL,
    CONTRATO_DETALHE_URL,
    CONTRATOS_URL,
    CONVENIOS_URL,
    DESPESAS_URL,
    EMENDAS_URL,
    LICITACOES_URL,
    NOTAS_FISCAIS_URL,
    PEP_URL,
    PESSOAS_FISICAS_URL,
    PESSOAS_JURIDICAS_URL,
    SANCOES_DATABASES,
    SERVIDOR_DETALHE_URL,
    SERVIDORES_URL,
    VIAGENS_URL,
)


@pytest.fixture(autouse=True)
def _set_api_key(monkeypatch: pytest.MonkeyPatch) -> None:
    """Set a fake API key for all tests."""
    monkeypatch.setenv(AUTH_ENV_VAR, "test-api-key-123")


# ---------------------------------------------------------------------------
# Auth
# ---------------------------------------------------------------------------


class TestAuth:
    def test_get_api_key_success(self) -> None:
        key = client._get_api_key()
        assert key == "test-api-key-123"

    def test_get_api_key_missing(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.delenv(AUTH_ENV_VAR)
        with pytest.raises(AuthError, match=AUTH_ENV_VAR):
            client._get_api_key()

    def test_auth_headers(self) -> None:
        headers = client._auth_headers()
        assert headers == {"chave-api-dados": "test-api-key-123"}


class TestCleanCpfCnpj:
    def test_digits_only(self) -> None:
        assert client._clean_cpf_cnpj("12345678000190") == "12345678000190"

    def test_formatted_cnpj(self) -> None:
        assert client._clean_cpf_cnpj("12.345.678/0001-90") == "12345678000190"

    def test_formatted_cpf(self) -> None:
        assert client._clean_cpf_cnpj("123.456.789-09") == "12345678909"


# ---------------------------------------------------------------------------
# buscar_contratos
# ---------------------------------------------------------------------------


class TestBuscarContratos:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_contracts(self) -> None:
        respx.get(CONTRATOS_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "id": 1,
                        "numero": "CT-001",
                        "objeto": "Serviço de TI",
                        "valorInicial": 100000.0,
                        "valorFinal": 120000.0,
                        "dataInicioVigencia": "01/01/2024",
                        "dataFimVigencia": "31/12/2024",
                        "unidadeGestora": {"nome": "MEC"},
                        "fornecedor": {"nome": "Empresa XYZ"},
                    }
                ],
            )
        )
        result = await client.buscar_contratos("12345678000190")
        assert len(result) == 1
        assert result[0].numero == "CT-001"
        assert result[0].valor_final == 120000.0
        assert result[0].orgao == "MEC"
        assert result[0].fornecedor == "Empresa XYZ"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(CONTRATOS_URL).mock(return_value=httpx.Response(200, json=[]))
        result = await client.buscar_contratos("00000000000000")
        assert result == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_non_list_response(self) -> None:
        respx.get(CONTRATOS_URL).mock(
            return_value=httpx.Response(200, json={"message": "not found"})
        )
        result = await client.buscar_contratos("00000000000000")
        assert result == []


# ---------------------------------------------------------------------------
# consultar_despesas
# ---------------------------------------------------------------------------


class TestConsultarDespesas:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_expenses(self) -> None:
        respx.get(DESPESAS_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "ano": 2024,
                        "mes": 6,
                        "valor": 50000.0,
                        "nomeFavorecido": "João Silva",
                        "nomeOrgao": "Ministério da Saúde",
                        "uf": "DF",
                    }
                ],
            )
        )
        result = await client.consultar_despesas("01/2024", "06/2024")
        assert len(result) == 1
        assert result[0].valor == 50000.0
        assert result[0].favorecido_nome == "João Silva"

    @pytest.mark.asyncio
    @respx.mock
    async def test_with_favorecido(self) -> None:
        route = respx.get(DESPESAS_URL).mock(return_value=httpx.Response(200, json=[]))
        await client.consultar_despesas("01/2024", "06/2024", "12345678000190")
        assert "codigoFavorecido" in str(route.calls[0].request.url)


# ---------------------------------------------------------------------------
# buscar_servidores
# ---------------------------------------------------------------------------


class TestBuscarServidores:
    @pytest.mark.asyncio
    @respx.mock
    async def test_by_cpf(self) -> None:
        respx.get(SERVIDORES_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "id": 42,
                        "cpf": "***123456**",
                        "nome": "Maria Santos",
                        "tipoServidor": "Civil",
                        "situacao": "Ativo",
                        "orgaoServidorExercicio": {"nome": "INSS"},
                    }
                ],
            )
        )
        result = await client.buscar_servidores(cpf="12345678900")
        assert len(result) == 1
        assert result[0].nome == "Maria Santos"
        assert result[0].orgao == "INSS"

    @pytest.mark.asyncio
    @respx.mock
    async def test_by_nome(self) -> None:
        respx.get(SERVIDORES_URL).mock(return_value=httpx.Response(200, json=[]))
        result = await client.buscar_servidores(nome="Carlos")
        assert result == []


# ---------------------------------------------------------------------------
# buscar_licitacoes
# ---------------------------------------------------------------------------


class TestBuscarLicitacoes:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_bids(self) -> None:
        respx.get(LICITACOES_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "id": 10,
                        "numero": "PE-2024/001",
                        "objeto": "Aquisição de computadores",
                        "modalidadeLicitacao": "Pregão Eletrônico",
                        "situacao": "Aberta",
                        "valorEstimado": 500000.0,
                        "dataAbertura": "15/03/2024",
                        "unidadeGestora": {"nome": "UFPI"},
                    }
                ],
            )
        )
        result = await client.buscar_licitacoes(codigo_orgao="26246")
        assert len(result) == 1
        assert result[0].modalidade == "Pregão Eletrônico"
        assert result[0].orgao == "UFPI"


# ---------------------------------------------------------------------------
# consultar_bolsa_familia
# ---------------------------------------------------------------------------


class TestBolsaFamilia:
    @pytest.mark.asyncio
    @respx.mock
    async def test_municipio(self) -> None:
        respx.get(BOLSA_FAMILIA_MUNICIPIO_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "municipio": {"nomeIBGE": "São Paulo", "uf": {"sigla": "SP"}},
                        "quantidadeBeneficiados": 100000,
                        "valor": 25000000.0,
                        "dataReferencia": "202401",
                    }
                ],
            )
        )
        result = await client.consultar_bolsa_familia_municipio("202401", "3550308")
        assert len(result) == 1
        assert result[0].municipio == "São Paulo"
        assert result[0].uf == "SP"
        assert result[0].quantidade == 100000

    @pytest.mark.asyncio
    @respx.mock
    async def test_nis(self) -> None:
        respx.get(BOLSA_FAMILIA_NIS_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "nis": "12345678901",
                        "nome": "Ana Lima",
                        "municipio": {"nomeIBGE": "Teresina", "uf": {"sigla": "PI"}},
                        "valor": 600.0,
                    }
                ],
            )
        )
        result = await client.consultar_bolsa_familia_nis("202401", "12345678901")
        assert len(result) == 1
        assert result[0].nome == "Ana Lima"
        assert result[0].valor == 600.0


# ---------------------------------------------------------------------------
# buscar_sancoes
# ---------------------------------------------------------------------------


class TestBuscarSancoes:
    @pytest.mark.asyncio
    @respx.mock
    async def test_single_base_by_cnpj(self) -> None:
        db = SANCOES_DATABASES["ceis"]
        respx.get(db["url"]).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "sancionado": {
                            "nome": "Empresa Sancionada",
                            "codigoFormatado": "12.345.678/0001-90",
                        },
                        "orgaoSancionador": {"nome": "CGU"},
                        "tipoSancao": "Inidoneidade",
                        "dataInicioSancao": "01/01/2023",
                        "dataFimSancao": "01/01/2028",
                        "fundamentacaoLegal": "Lei 8666/93",
                    }
                ],
            )
        )
        result = await client.buscar_sancoes("12345678000190", bases=["ceis"])
        assert len(result) == 1
        assert result[0].nome == "Empresa Sancionada"
        assert result[0].fonte == SANCOES_DATABASES["ceis"]["nome"]

    @pytest.mark.asyncio
    @respx.mock
    async def test_all_bases_parallel(self) -> None:
        for db in SANCOES_DATABASES.values():
            respx.get(db["url"]).mock(return_value=httpx.Response(200, json=[]))
        result = await client.buscar_sancoes("Empresa Teste")
        assert result == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_partial_failure(self) -> None:
        """One base fails, others succeed — no exception raised."""
        ceis = SANCOES_DATABASES["ceis"]
        cnep = SANCOES_DATABASES["cnep"]
        cepim = SANCOES_DATABASES["cepim"]
        ceaf = SANCOES_DATABASES["ceaf"]

        respx.get(ceis["url"]).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "sancionado": {"nome": "Teste"},
                        "orgaoSancionador": {},
                    }
                ],
            )
        )
        respx.get(cnep["url"]).mock(return_value=httpx.Response(500, text="Error"))
        respx.get(cepim["url"]).mock(return_value=httpx.Response(200, json=[]))
        respx.get(ceaf["url"]).mock(return_value=httpx.Response(200, json=[]))

        with patch("mcp_brasil._shared.http_client.asyncio.sleep"):
            result = await client.buscar_sancoes("12345678000190", pagina=1)
        assert len(result) == 1


# ---------------------------------------------------------------------------
# buscar_emendas
# ---------------------------------------------------------------------------


class TestBuscarEmendas:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_amendments(self) -> None:
        respx.get(EMENDAS_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "codigoEmenda": "EMD-001",
                        "autor": {"nome": "Dep. Fulano"},
                        "tipoEmenda": "Individual",
                        "localidadeDoGasto": {"nome": "Teresina"},
                        "valorEmpenhado": 1000000.0,
                        "valorPago": 500000.0,
                        "ano": 2024,
                    }
                ],
            )
        )
        result = await client.buscar_emendas(ano=2024)
        assert len(result) == 1
        assert result[0].autor == "Dep. Fulano"
        assert result[0].valor_empenhado == 1000000.0


# ---------------------------------------------------------------------------
# consultar_viagens
# ---------------------------------------------------------------------------


class TestConsultarViagens:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_trips(self) -> None:
        respx.get(VIAGENS_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "id": 99,
                        "cpf": "***123456**",
                        "nome": "Pedro Almeida",
                        "cargo": "Analista",
                        "nomeOrgao": "MRE",
                        "destinos": "Brasília/DF",
                        "dataInicio": "01/03/2024",
                        "dataFim": "05/03/2024",
                        "valorPassagens": 1500.0,
                        "valorDiarias": 2000.0,
                    }
                ],
            )
        )
        result = await client.consultar_viagens("12345678900")
        assert len(result) == 1
        assert result[0].nome == "Pedro Almeida"
        assert result[0].valor_diarias == 2000.0
        assert result[0].destino == "Brasília/DF"


# ---------------------------------------------------------------------------
# Edge cases: _safe_parse_list + non-list responses + warning logging
# ---------------------------------------------------------------------------


class TestSafeParseListLogging:
    @pytest.mark.asyncio
    @respx.mock
    async def test_non_list_despesas_logs_warning(self, caplog: pytest.LogCaptureFixture) -> None:
        respx.get(DESPESAS_URL).mock(return_value=httpx.Response(200, json={"error": "invalid"}))
        with caplog.at_level(logging.WARNING, logger="mcp_brasil.transparencia.client"):
            result = await client.consultar_despesas("01/2024", "06/2024")
        assert result == []
        assert "Resposta inesperada" in caplog.text

    @pytest.mark.asyncio
    @respx.mock
    async def test_non_list_servidores_logs_warning(
        self, caplog: pytest.LogCaptureFixture
    ) -> None:
        respx.get(SERVIDORES_URL).mock(return_value=httpx.Response(200, json="plain string"))
        with caplog.at_level(logging.WARNING, logger="mcp_brasil.transparencia.client"):
            result = await client.buscar_servidores(nome="Test")
        assert result == []
        assert "Resposta inesperada" in caplog.text

    @pytest.mark.asyncio
    @respx.mock
    async def test_non_list_licitacoes(self) -> None:
        respx.get(LICITACOES_URL).mock(return_value=httpx.Response(200, json={"status": "error"}))
        result = await client.buscar_licitacoes()
        assert result == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_non_list_bolsa_municipio(self) -> None:
        respx.get(BOLSA_FAMILIA_MUNICIPIO_URL).mock(
            return_value=httpx.Response(200, json={"error": "not found"})
        )
        result = await client.consultar_bolsa_familia_municipio("202401", "3550308")
        assert result == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_non_list_bolsa_nis(self) -> None:
        respx.get(BOLSA_FAMILIA_NIS_URL).mock(return_value=httpx.Response(200, json=42))
        result = await client.consultar_bolsa_familia_nis("202401", "12345678901")
        assert result == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_non_list_emendas(self) -> None:
        respx.get(EMENDAS_URL).mock(return_value=httpx.Response(200, json={"msg": "err"}))
        result = await client.buscar_emendas(ano=2024)
        assert result == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_non_list_viagens(self) -> None:
        respx.get(VIAGENS_URL).mock(return_value=httpx.Response(200, json="oops"))
        result = await client.consultar_viagens("12345678900")
        assert result == []


# ---------------------------------------------------------------------------
# Edge cases: parser null/empty fields
# ---------------------------------------------------------------------------


class TestParserEdgeCases:
    def test_contrato_all_none(self) -> None:
        result = client._parse_contrato({})
        assert result.numero is None
        assert result.orgao is None
        assert result.fornecedor is None

    def test_contrato_nested_empty(self) -> None:
        result = client._parse_contrato({"fornecedor": {}, "unidadeGestora": {}})
        assert result.fornecedor is None
        assert result.orgao is None

    def test_recurso_all_none(self) -> None:
        result = client._parse_recurso({})
        assert result.ano is None
        assert result.valor is None

    def test_servidor_all_none(self) -> None:
        result = client._parse_servidor({})
        assert result.nome is None
        assert result.orgao is None

    def test_licitacao_all_none(self) -> None:
        result = client._parse_licitacao({})
        assert result.numero is None
        assert result.orgao is None

    def test_bolsa_municipio_string_municipio(self) -> None:
        """When API returns municipio as a string instead of dict."""
        result = client._parse_bolsa_municipio({"municipio": "São Paulo"})
        assert result.municipio == "São Paulo"
        assert result.uf is None

    def test_bolsa_municipio_none_municipio(self) -> None:
        result = client._parse_bolsa_municipio({})
        assert result.municipio is None
        assert result.uf is None

    def test_bolsa_sacado_string_municipio(self) -> None:
        """When API returns municipio as a string instead of dict."""
        result = client._parse_bolsa_sacado({"municipio": "Teresina"})
        assert result.municipio is None  # String doesn't have .get("nomeIBGE")
        assert result.uf is None

    def test_bolsa_sacado_none_municipio(self) -> None:
        result = client._parse_bolsa_sacado({})
        assert result.municipio is None
        assert result.uf is None

    def test_sancao_all_none(self) -> None:
        result = client._parse_sancao({}, "CEIS")
        assert result.fonte == "CEIS"
        assert result.nome is None

    def test_emenda_autor_string(self) -> None:
        """When autor is a string instead of dict."""
        result = client._parse_emenda({"autor": "Dep. Fulano"})
        assert result.autor == "Dep. Fulano"

    def test_emenda_all_none(self) -> None:
        result = client._parse_emenda({})
        assert result.numero is None
        assert result.autor is None

    def test_viagem_all_none(self) -> None:
        result = client._parse_viagem({})
        assert result.nome is None
        assert result.destino is None

    def test_convenio_all_none(self) -> None:
        result = client._parse_convenio({})
        assert result.numero is None
        assert result.orgao is None

    def test_cartao_all_none(self) -> None:
        result = client._parse_cartao({})
        assert result.portador is None
        assert result.valor is None

    def test_pep_all_none(self) -> None:
        result = client._parse_pep({})
        assert result.nome is None
        assert result.funcao is None

    def test_acordo_leniencia_all_none(self) -> None:
        result = client._parse_acordo_leniencia({})
        assert result.empresa is None
        assert result.cnpj is None

    def test_nota_fiscal_all_none(self) -> None:
        result = client._parse_nota_fiscal({})
        assert result.numero is None
        assert result.emitente is None

    def test_beneficio_social_all_none(self) -> None:
        result = client._parse_beneficio_social({})
        assert result.tipo is None
        assert result.valor is None

    def test_pessoa_fisica_all_none(self) -> None:
        result = client._parse_pessoa_fisica({})
        assert result.cpf is None
        assert result.nome is None

    def test_pessoa_juridica_all_none(self) -> None:
        result = client._parse_pessoa_juridica({})
        assert result.cnpj is None
        assert result.razao_social is None

    def test_contrato_detalhe_all_none(self) -> None:
        result = client._parse_contrato_detalhe({})
        assert result.numero is None
        assert result.modalidade is None

    def test_servidor_detalhe_all_none(self) -> None:
        result = client._parse_servidor_detalhe({})
        assert result.nome is None
        assert result.remuneracao_basica is None


# ---------------------------------------------------------------------------
# buscar_convenios
# ---------------------------------------------------------------------------


class TestBuscarConvenios:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed(self) -> None:
        respx.get(CONVENIOS_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "numero": "CV-001",
                        "objeto": "Construção de escola",
                        "situacao": "Vigente",
                        "valorConvenio": 500000.0,
                        "valorLiberado": 250000.0,
                        "orgaoConcedente": {"nome": "MEC"},
                        "convenente": {"nome": "Prefeitura de Teresina"},
                        "dataInicioVigencia": "01/01/2024",
                        "dataFimVigencia": "31/12/2025",
                    }
                ],
            )
        )
        result = await client.buscar_convenios()
        assert len(result) == 1
        assert result[0].numero == "CV-001"
        assert result[0].orgao == "MEC"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty(self) -> None:
        respx.get(CONVENIOS_URL).mock(return_value=httpx.Response(200, json=[]))
        result = await client.buscar_convenios()
        assert result == []


# ---------------------------------------------------------------------------
# buscar_cartoes_pagamento
# ---------------------------------------------------------------------------


class TestBuscarCartoes:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed(self) -> None:
        respx.get(CARTOES_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "portador": "João Silva",
                        "cpfPortador": "***123***",
                        "nomeOrgao": "MRE",
                        "valorTransacao": 1500.0,
                        "dataTransacao": "15/03/2024",
                        "tipoCartao": "Suprimento de Fundos",
                        "nomeEstabelecimento": "Hotel Central",
                    }
                ],
            )
        )
        result = await client.buscar_cartoes_pagamento()
        assert len(result) == 1
        assert result[0].portador == "João Silva"
        assert result[0].valor == 1500.0


# ---------------------------------------------------------------------------
# buscar_pep
# ---------------------------------------------------------------------------


class TestBuscarPep:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed(self) -> None:
        respx.get(PEP_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "cpf": "***123***",
                        "nome": "Maria Política",
                        "orgao": {"nome": "Senado Federal"},
                        "descricaoFuncao": "Senadora",
                        "dataInicioExercicio": "01/02/2023",
                        "dataFimExercicio": None,
                    }
                ],
            )
        )
        result = await client.buscar_pep(nome="Maria")
        assert len(result) == 1
        assert result[0].nome == "Maria Política"
        assert result[0].funcao == "Senadora"


# ---------------------------------------------------------------------------
# buscar_acordos_leniencia
# ---------------------------------------------------------------------------


class TestBuscarAcordosLeniencia:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed(self) -> None:
        respx.get(ACORDOS_LENIENCIA_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "pessoa": {"nome": "Construtora XYZ", "cnpj": "12345678000190"},
                        "orgaoResponsavel": {"nome": "CGU"},
                        "situacao": "Vigente",
                        "dataInicioAcordo": "01/06/2020",
                        "valorMulta": 10000000.0,
                    }
                ],
            )
        )
        result = await client.buscar_acordos_leniencia()
        assert len(result) == 1
        assert result[0].empresa == "Construtora XYZ"
        assert result[0].valor == 10000000.0


# ---------------------------------------------------------------------------
# buscar_notas_fiscais
# ---------------------------------------------------------------------------


class TestBuscarNotasFiscais:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed(self) -> None:
        respx.get(NOTAS_FISCAIS_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "numero": "NF-001",
                        "serie": "1",
                        "emitente": {"nome": "Fornecedor ABC", "cnpj": "99888777000111"},
                        "valor": 5000.0,
                        "dataEmissao": "10/03/2024",
                    }
                ],
            )
        )
        result = await client.buscar_notas_fiscais()
        assert len(result) == 1
        assert result[0].numero == "NF-001"
        assert result[0].emitente == "Fornecedor ABC"


# ---------------------------------------------------------------------------
# consultar_beneficio_social
# ---------------------------------------------------------------------------


class TestConsultarBeneficioSocial:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed(self) -> None:
        respx.get(BENEFICIOS_CIDADAO_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "tipoBeneficio": "BPC",
                        "nomeBeneficiario": "José Lima",
                        "cpf": "***456***",
                        "nis": "98765432101",
                        "valor": 1412.0,
                        "mesReferencia": "202401",
                        "municipio": {"nomeIBGE": "Teresina", "uf": {"sigla": "PI"}},
                    }
                ],
            )
        )
        result = await client.consultar_beneficio_social(cpf="12345678900")
        assert len(result) == 1
        assert result[0].tipo == "BPC"
        assert result[0].valor == 1412.0


# ---------------------------------------------------------------------------
# consultar_cpf
# ---------------------------------------------------------------------------


class TestConsultarCpf:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed(self) -> None:
        respx.get(PESSOAS_FISICAS_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "cpf": "***123***",
                        "nome": "Ana Souza",
                        "tipoVinculo": "Servidor",
                        "nomeOrgao": "INSS",
                    }
                ],
            )
        )
        result = await client.consultar_cpf("12345678900")
        assert len(result) == 1
        assert result[0].nome == "Ana Souza"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty(self) -> None:
        respx.get(PESSOAS_FISICAS_URL).mock(return_value=httpx.Response(200, json=[]))
        result = await client.consultar_cpf("00000000000")
        assert result == []


# ---------------------------------------------------------------------------
# consultar_cnpj
# ---------------------------------------------------------------------------


class TestConsultarCnpj:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed(self) -> None:
        respx.get(PESSOAS_JURIDICAS_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "cnpj": "12345678000190",
                        "razaoSocial": "Empresa Teste LTDA",
                        "sancoes": "Nenhuma",
                        "contratos": "3 contratos ativos",
                    }
                ],
            )
        )
        result = await client.consultar_cnpj("12345678000190")
        assert len(result) == 1
        assert result[0].razao_social == "Empresa Teste LTDA"


# ---------------------------------------------------------------------------
# detalhar_contrato
# ---------------------------------------------------------------------------


class TestDetalharContrato:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_detail(self) -> None:
        respx.get(f"{CONTRATO_DETALHE_URL}/123").mock(
            return_value=httpx.Response(
                200,
                json={
                    "id": 123,
                    "numero": "CT-DETAIL",
                    "objeto": "Serviço completo",
                    "valorInicial": 100000.0,
                    "valorFinal": 150000.0,
                    "dataInicioVigencia": "01/01/2024",
                    "dataFimVigencia": "31/12/2024",
                    "unidadeGestora": {"nome": "MEC"},
                    "fornecedor": {"nome": "Fornecedor Detail"},
                    "modalidadeCompra": "Pregão",
                    "situacao": "Ativo",
                },
            )
        )
        result = await client.detalhar_contrato(123)
        assert result is not None
        assert result.numero == "CT-DETAIL"
        assert result.modalidade == "Pregão"

    @pytest.mark.asyncio
    @respx.mock
    async def test_not_found(self) -> None:
        respx.get(f"{CONTRATO_DETALHE_URL}/999").mock(return_value=httpx.Response(200, json=[]))
        result = await client.detalhar_contrato(999)
        assert result is None


# ---------------------------------------------------------------------------
# detalhar_servidor
# ---------------------------------------------------------------------------


class TestDetalharServidor:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_detail(self) -> None:
        respx.get(f"{SERVIDOR_DETALHE_URL}/42").mock(
            return_value=httpx.Response(
                200,
                json={
                    "id": 42,
                    "cpf": "***123***",
                    "nome": "Servidor Completo",
                    "tipoServidor": "Civil",
                    "situacao": "Ativo",
                    "orgaoServidorExercicio": {"nome": "INSS"},
                    "cargo": "Analista",
                    "funcao": "Coordenador",
                    "remuneracaoBasicaBruta": 12000.0,
                    "remuneracaoAposDeducoesObrigatorias": 9500.0,
                },
            )
        )
        result = await client.detalhar_servidor(42)
        assert result is not None
        assert result.nome == "Servidor Completo"
        assert result.remuneracao_basica == 12000.0
        assert result.remuneracao_apos_deducoes == 9500.0

    @pytest.mark.asyncio
    @respx.mock
    async def test_not_found(self) -> None:
        respx.get(f"{SERVIDOR_DETALHE_URL}/999").mock(return_value=httpx.Response(200, json=[]))
        result = await client.detalhar_servidor(999)
        assert result is None
