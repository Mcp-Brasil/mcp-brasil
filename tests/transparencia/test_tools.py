"""Tests for the transparencia tool functions.

Tools are tested by mocking client functions (never HTTP).
"""

from unittest.mock import AsyncMock, patch

import pytest

from mcp_brasil.transparencia import tools
from mcp_brasil.transparencia.constants import DEFAULT_PAGE_SIZE
from mcp_brasil.transparencia.schemas import (
    AcordoLeniencia,
    BeneficioSocial,
    BolsaFamiliaMunicipio,
    BolsaFamiliaSacado,
    CartaoPagamento,
    ContratoDetalhe,
    ContratoFornecedor,
    Convenio,
    Emenda,
    Licitacao,
    NotaFiscal,
    PessoaExpostaPoliticamente,
    PessoaFisicaVinculos,
    PessoaJuridicaVinculos,
    RecursoRecebido,
    Sancao,
    Servidor,
    ServidorDetalhe,
    Viagem,
)

MODULE = "mcp_brasil.transparencia.client"


# ---------------------------------------------------------------------------
# buscar_contratos
# ---------------------------------------------------------------------------


class TestBuscarContratos:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            ContratoFornecedor(
                numero="CT-001",
                objeto="Serviço de TI",
                valor_final=120000.0,
                data_inicio="01/01/2024",
                data_fim="31/12/2024",
                orgao="MEC",
                fornecedor="Empresa XYZ",
            )
        ]
        with patch(f"{MODULE}.buscar_contratos", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.buscar_contratos("12345678000190")
        assert "CT-001" in result
        assert "R$ 120.000,00" in result
        assert "Empresa XYZ" not in result  # fornecedor is in header, not table
        assert "MEC" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.buscar_contratos", new_callable=AsyncMock, return_value=[]):
            result = await tools.buscar_contratos("00000000000000")
        assert "Nenhum contrato" in result


# ---------------------------------------------------------------------------
# consultar_despesas
# ---------------------------------------------------------------------------


class TestConsultarDespesas:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            RecursoRecebido(
                ano=2024,
                mes=6,
                valor=50000.0,
                favorecido_nome="João Silva",
                orgao_nome="Min. Saúde",
                uf="DF",
            )
        ]
        with patch(f"{MODULE}.consultar_despesas", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.consultar_despesas("01/2024", "06/2024")
        assert "R$ 50.000,00" in result
        assert "João Silva" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.consultar_despesas", new_callable=AsyncMock, return_value=[]):
            result = await tools.consultar_despesas("01/2024", "06/2024")
        assert "Nenhuma despesa" in result


# ---------------------------------------------------------------------------
# buscar_servidores
# ---------------------------------------------------------------------------


class TestBuscarServidores:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Servidor(
                cpf="***123***",
                nome="Maria Santos",
                tipo_servidor="Civil",
                situacao="Ativo",
                orgao="INSS",
            )
        ]
        with patch(f"{MODULE}.buscar_servidores", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.buscar_servidores(cpf="12345678900")
        assert "Maria Santos" in result
        assert "INSS" in result

    @pytest.mark.asyncio
    async def test_no_params(self) -> None:
        result = await tools.buscar_servidores()
        assert "Informe CPF ou nome" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.buscar_servidores", new_callable=AsyncMock, return_value=[]):
            result = await tools.buscar_servidores(nome="Fulano")
        assert "Nenhum servidor" in result


# ---------------------------------------------------------------------------
# buscar_licitacoes
# ---------------------------------------------------------------------------


class TestBuscarLicitacoes:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Licitacao(
                numero="PE-001",
                objeto="Compra de PCs",
                modalidade="Pregão",
                situacao="Aberta",
                valor_estimado=500000.0,
                data_abertura="15/03/2024",
                orgao="UFPI",
            )
        ]
        with patch(f"{MODULE}.buscar_licitacoes", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.buscar_licitacoes(codigo_orgao="26246")
        assert "PE-001" in result
        assert "R$ 500.000,00" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.buscar_licitacoes", new_callable=AsyncMock, return_value=[]):
            result = await tools.buscar_licitacoes()
        assert "Nenhuma licitação" in result


# ---------------------------------------------------------------------------
# consultar_bolsa_familia
# ---------------------------------------------------------------------------


class TestConsultarBolsaFamilia:
    @pytest.mark.asyncio
    async def test_no_params(self) -> None:
        result = await tools.consultar_bolsa_familia("202401")
        assert "Informe" in result

    @pytest.mark.asyncio
    async def test_by_municipio(self) -> None:
        mock_data = [
            BolsaFamiliaMunicipio(
                municipio="São Paulo",
                uf="SP",
                quantidade=100000,
                valor=25000000.0,
                data_referencia="202401",
            )
        ]
        with patch(
            f"{MODULE}.consultar_bolsa_familia_municipio",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_bolsa_familia("202401", codigo_ibge="3550308")
        assert "São Paulo" in result
        assert "R$ 25.000.000,00" in result

    @pytest.mark.asyncio
    async def test_by_nis(self) -> None:
        mock_data = [
            BolsaFamiliaSacado(
                nis="12345678901",
                nome="Ana Lima",
                municipio="Teresina",
                uf="PI",
                valor=600.0,
            )
        ]
        with patch(
            f"{MODULE}.consultar_bolsa_familia_nis",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_bolsa_familia("202401", nis="12345678901")
        assert "Ana Lima" in result
        assert "R$ 600,00" in result

    @pytest.mark.asyncio
    async def test_nis_empty(self) -> None:
        with patch(
            f"{MODULE}.consultar_bolsa_familia_nis",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.consultar_bolsa_familia("202401", nis="99999999999")
        assert "Nenhum dado" in result


# ---------------------------------------------------------------------------
# buscar_sancoes
# ---------------------------------------------------------------------------


class TestBuscarSancoes:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = [
            Sancao(
                fonte="CEIS",
                tipo="Inidoneidade",
                nome="Empresa Má",
                cpf_cnpj="12.345.678/0001-90",
                orgao="CGU",
                data_inicio="01/01/2023",
                data_fim="01/01/2028",
                fundamentacao="Lei 8666/93",
            )
        ]
        with patch(f"{MODULE}.buscar_sancoes", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.buscar_sancoes("12345678000190")
        assert "Empresa Má" in result
        assert "CEIS" in result
        assert "CGU" in result
        assert "Lei 8666/93" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.buscar_sancoes", new_callable=AsyncMock, return_value=[]):
            result = await tools.buscar_sancoes("00000000000000")
        assert "Nenhuma sanção" in result


# ---------------------------------------------------------------------------
# buscar_emendas
# ---------------------------------------------------------------------------


class TestBuscarEmendas:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Emenda(
                numero="EMD-001",
                autor="Dep. Fulano",
                tipo="Individual",
                localidade="Teresina",
                valor_empenhado=1000000.0,
                valor_pago=500000.0,
                ano=2024,
            )
        ]
        with patch(f"{MODULE}.buscar_emendas", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.buscar_emendas(ano=2024)
        assert "EMD-001" in result
        assert "Dep. Fulano" in result
        assert "R$ 1.000.000,00" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.buscar_emendas", new_callable=AsyncMock, return_value=[]):
            result = await tools.buscar_emendas()
        assert "Nenhuma emenda" in result


# ---------------------------------------------------------------------------
# consultar_viagens
# ---------------------------------------------------------------------------


class TestConsultarViagens:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Viagem(
                nome="Pedro Almeida",
                cargo="Analista",
                orgao="MRE",
                destino="Brasília/DF",
                data_inicio="01/03/2024",
                data_fim="05/03/2024",
                valor_passagens=1500.0,
                valor_diarias=2000.0,
            )
        ]
        with patch(f"{MODULE}.consultar_viagens", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.consultar_viagens("12345678900")
        assert "Pedro Almeida" in result
        assert "R$ 2.000,00" in result
        assert "Brasília/DF" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.consultar_viagens", new_callable=AsyncMock, return_value=[]):
            result = await tools.consultar_viagens("00000000000")
        assert "Nenhuma viagem" in result


# ---------------------------------------------------------------------------
# Pagination hints
# ---------------------------------------------------------------------------


def _make_contratos(n: int) -> list[ContratoFornecedor]:
    return [ContratoFornecedor(numero=f"CT-{i}") for i in range(n)]


class TestPaginationHints:
    @pytest.mark.asyncio
    async def test_shows_next_page_hint(self) -> None:
        """When results >= DEFAULT_PAGE_SIZE, hint to load next page."""
        data = _make_contratos(DEFAULT_PAGE_SIZE)
        with patch(f"{MODULE}.buscar_contratos", new_callable=AsyncMock, return_value=data):
            result = await tools.buscar_contratos("12345678000190", pagina=1)
        assert "pagina=2" in result

    @pytest.mark.asyncio
    async def test_no_hint_below_page_size(self) -> None:
        """When results < DEFAULT_PAGE_SIZE on page 1, no hint."""
        data = _make_contratos(3)
        with patch(f"{MODULE}.buscar_contratos", new_callable=AsyncMock, return_value=data):
            result = await tools.buscar_contratos("12345678000190", pagina=1)
        assert "pagina=" not in result
        assert "Última página" not in result

    @pytest.mark.asyncio
    async def test_last_page_hint(self) -> None:
        """When results < DEFAULT_PAGE_SIZE on page > 1, show last page hint."""
        data = _make_contratos(3)
        with patch(f"{MODULE}.buscar_contratos", new_callable=AsyncMock, return_value=data):
            result = await tools.buscar_contratos("12345678000190", pagina=2)
        assert "Última página" in result

    @pytest.mark.asyncio
    async def test_despesas_hint(self) -> None:
        data = [RecursoRecebido(ano=2024, mes=1, valor=100.0)] * DEFAULT_PAGE_SIZE
        with patch(f"{MODULE}.consultar_despesas", new_callable=AsyncMock, return_value=data):
            result = await tools.consultar_despesas("01/2024", "06/2024")
        assert "pagina=2" in result

    @pytest.mark.asyncio
    async def test_servidores_hint(self) -> None:
        data = [Servidor(nome="Test")] * DEFAULT_PAGE_SIZE
        with patch(f"{MODULE}.buscar_servidores", new_callable=AsyncMock, return_value=data):
            result = await tools.buscar_servidores(nome="Test")
        assert "pagina=2" in result

    @pytest.mark.asyncio
    async def test_licitacoes_hint(self) -> None:
        data = [Licitacao(numero="PE-001")] * DEFAULT_PAGE_SIZE
        with patch(f"{MODULE}.buscar_licitacoes", new_callable=AsyncMock, return_value=data):
            result = await tools.buscar_licitacoes()
        assert "pagina=2" in result

    @pytest.mark.asyncio
    async def test_emendas_hint(self) -> None:
        data = [Emenda(numero="EMD-001")] * DEFAULT_PAGE_SIZE
        with patch(f"{MODULE}.buscar_emendas", new_callable=AsyncMock, return_value=data):
            result = await tools.buscar_emendas()
        assert "pagina=2" in result

    @pytest.mark.asyncio
    async def test_viagens_hint(self) -> None:
        data = [Viagem(nome="Test")] * DEFAULT_PAGE_SIZE
        with patch(f"{MODULE}.consultar_viagens", new_callable=AsyncMock, return_value=data):
            result = await tools.consultar_viagens("12345678900")
        assert "pagina=2" in result


# ---------------------------------------------------------------------------
# buscar_convenios
# ---------------------------------------------------------------------------


class TestBuscarConvenios:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Convenio(
                numero="CV-001",
                objeto="Construção de escola",
                situacao="Vigente",
                valor_convenio=500000.0,
                valor_liberado=250000.0,
                orgao="MEC",
                convenente="Prefeitura de Teresina",
            )
        ]
        with patch(f"{MODULE}.buscar_convenios", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.buscar_convenios()
        assert "CV-001" in result
        assert "R$ 500.000,00" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.buscar_convenios", new_callable=AsyncMock, return_value=[]):
            result = await tools.buscar_convenios()
        assert "Nenhum convênio" in result


# ---------------------------------------------------------------------------
# buscar_cartoes_pagamento
# ---------------------------------------------------------------------------


class TestBuscarCartoesPagamento:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            CartaoPagamento(
                portador="João Silva",
                orgao="MRE",
                valor=1500.0,
                data="15/03/2024",
                tipo="Suprimento",
                estabelecimento="Hotel Central",
            )
        ]
        with patch(
            f"{MODULE}.buscar_cartoes_pagamento",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_cartoes_pagamento()
        assert "João Silva" in result
        assert "R$ 1.500,00" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.buscar_cartoes_pagamento", new_callable=AsyncMock, return_value=[]):
            result = await tools.buscar_cartoes_pagamento()
        assert "Nenhum pagamento" in result


# ---------------------------------------------------------------------------
# buscar_pep
# ---------------------------------------------------------------------------


class TestBuscarPep:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            PessoaExpostaPoliticamente(
                cpf="***123***",
                nome="Maria Política",
                orgao="Senado Federal",
                funcao="Senadora",
                data_inicio="01/02/2023",
            )
        ]
        with patch(f"{MODULE}.buscar_pep", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.buscar_pep(nome="Maria")
        assert "Maria Política" in result
        assert "Senadora" in result

    @pytest.mark.asyncio
    async def test_no_params(self) -> None:
        result = await tools.buscar_pep()
        assert "Informe CPF ou nome" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.buscar_pep", new_callable=AsyncMock, return_value=[]):
            result = await tools.buscar_pep(nome="Inexistente")
        assert "Nenhuma PEP" in result


# ---------------------------------------------------------------------------
# buscar_acordos_leniencia
# ---------------------------------------------------------------------------


class TestBuscarAcordosLeniencia:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            AcordoLeniencia(
                empresa="Construtora XYZ",
                cnpj="12345678000190",
                orgao="CGU",
                situacao="Vigente",
                data_inicio="01/06/2020",
                valor=10000000.0,
            )
        ]
        with patch(
            f"{MODULE}.buscar_acordos_leniencia",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_acordos_leniencia()
        assert "Construtora XYZ" in result
        assert "R$ 10.000.000,00" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.buscar_acordos_leniencia", new_callable=AsyncMock, return_value=[]):
            result = await tools.buscar_acordos_leniencia()
        assert "Nenhum acordo" in result


# ---------------------------------------------------------------------------
# buscar_notas_fiscais
# ---------------------------------------------------------------------------


class TestBuscarNotasFiscais:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            NotaFiscal(
                numero="NF-001",
                serie="1",
                emitente="Fornecedor ABC",
                cnpj_emitente="99888777000111",
                valor=5000.0,
                data_emissao="10/03/2024",
            )
        ]
        with patch(
            f"{MODULE}.buscar_notas_fiscais", new_callable=AsyncMock, return_value=mock_data
        ):
            result = await tools.buscar_notas_fiscais()
        assert "NF-001" in result
        assert "R$ 5.000,00" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.buscar_notas_fiscais", new_callable=AsyncMock, return_value=[]):
            result = await tools.buscar_notas_fiscais()
        assert "Nenhuma nota fiscal" in result


# ---------------------------------------------------------------------------
# consultar_beneficio_social
# ---------------------------------------------------------------------------


class TestConsultarBeneficioSocial:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            BeneficioSocial(
                tipo="BPC",
                nome_beneficiario="José Lima",
                valor=1412.0,
                mes_referencia="202401",
                municipio="Teresina",
                uf="PI",
            )
        ]
        with patch(
            f"{MODULE}.consultar_beneficio_social",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_beneficio_social(cpf="12345678900")
        assert "BPC" in result
        assert "R$ 1.412,00" in result

    @pytest.mark.asyncio
    async def test_no_params(self) -> None:
        result = await tools.consultar_beneficio_social()
        assert "Informe CPF ou NIS" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(
            f"{MODULE}.consultar_beneficio_social", new_callable=AsyncMock, return_value=[]
        ):
            result = await tools.consultar_beneficio_social(cpf="00000000000")
        assert "Nenhum benefício" in result


# ---------------------------------------------------------------------------
# consultar_cpf
# ---------------------------------------------------------------------------


class TestConsultarCpf:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = [
            PessoaFisicaVinculos(
                cpf="***123***",
                nome="Ana Souza",
                tipo_vinculo="Servidor",
                orgao="INSS",
            )
        ]
        with patch(f"{MODULE}.consultar_cpf", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.consultar_cpf("12345678900")
        assert "Ana Souza" in result
        assert "Servidor" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.consultar_cpf", new_callable=AsyncMock, return_value=[]):
            result = await tools.consultar_cpf("00000000000")
        assert "Nenhum vínculo" in result


# ---------------------------------------------------------------------------
# consultar_cnpj
# ---------------------------------------------------------------------------


class TestConsultarCnpj:
    @pytest.mark.asyncio
    async def test_formats_results(self) -> None:
        mock_data = [
            PessoaJuridicaVinculos(
                cnpj="12345678000190",
                razao_social="Empresa Teste LTDA",
                sancoes="Nenhuma",
                contratos="3 contratos ativos",
            )
        ]
        with patch(f"{MODULE}.consultar_cnpj", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.consultar_cnpj("12345678000190")
        assert "Empresa Teste" in result
        assert "3 contratos" in result

    @pytest.mark.asyncio
    async def test_empty(self) -> None:
        with patch(f"{MODULE}.consultar_cnpj", new_callable=AsyncMock, return_value=[]):
            result = await tools.consultar_cnpj("00000000000000")
        assert "Nenhum vínculo" in result


# ---------------------------------------------------------------------------
# detalhar_contrato
# ---------------------------------------------------------------------------


class TestDetalharContrato:
    @pytest.mark.asyncio
    async def test_formats_detail(self) -> None:
        mock_data = ContratoDetalhe(
            id=123,
            numero="CT-DETAIL",
            objeto="Serviço completo",
            valor_inicial=100000.0,
            valor_final=150000.0,
            data_inicio="01/01/2024",
            data_fim="31/12/2024",
            orgao="MEC",
            fornecedor="Fornecedor Detail",
            modalidade="Pregão",
            situacao="Ativo",
        )
        with patch(f"{MODULE}.detalhar_contrato", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.detalhar_contrato(123)
        assert "CT-DETAIL" in result
        assert "R$ 150.000,00" in result
        assert "Pregão" in result

    @pytest.mark.asyncio
    async def test_not_found(self) -> None:
        with patch(f"{MODULE}.detalhar_contrato", new_callable=AsyncMock, return_value=None):
            result = await tools.detalhar_contrato(999)
        assert "não encontrado" in result


# ---------------------------------------------------------------------------
# detalhar_servidor
# ---------------------------------------------------------------------------


class TestDetalharServidor:
    @pytest.mark.asyncio
    async def test_formats_detail(self) -> None:
        mock_data = ServidorDetalhe(
            id=42,
            cpf="***123***",
            nome="Servidor Completo",
            tipo_servidor="Civil",
            situacao="Ativo",
            orgao="INSS",
            cargo="Analista",
            funcao="Coordenador",
            remuneracao_basica=12000.0,
            remuneracao_apos_deducoes=9500.0,
        )
        with patch(f"{MODULE}.detalhar_servidor", new_callable=AsyncMock, return_value=mock_data):
            result = await tools.detalhar_servidor(42)
        assert "Servidor Completo" in result
        assert "R$ 12.000,00" in result
        assert "Coordenador" in result

    @pytest.mark.asyncio
    async def test_not_found(self) -> None:
        with patch(f"{MODULE}.detalhar_servidor", new_callable=AsyncMock, return_value=None):
            result = await tools.detalhar_servidor(999)
        assert "não encontrado" in result
