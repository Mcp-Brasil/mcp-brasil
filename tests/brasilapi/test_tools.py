"""Tests for the BrasilAPI tool functions.

Tools are tested by mocking client functions (never HTTP).
Context is mocked via MagicMock with async methods.
"""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from mcp_brasil.brasilapi import tools
from mcp_brasil.brasilapi.schemas import (
    Banco,
    DddInfo,
    EmpresaCnpj,
    Endereco,
    Feriado,
    FipeMarca,
    Livro,
    NcmItem,
    RegistroBrDominio,
    TaxaOficial,
)

CLIENT_MODULE = "mcp_brasil.brasilapi.client"


def _mock_ctx() -> MagicMock:
    """Create a mock Context with async log methods."""
    ctx = MagicMock()
    ctx.info = AsyncMock()
    ctx.warning = AsyncMock()
    return ctx


# ---------------------------------------------------------------------------
# consultar_cep
# ---------------------------------------------------------------------------


class TestConsultarCep:
    @pytest.mark.asyncio
    async def test_formats_address(self) -> None:
        mock_data = Endereco(
            cep="01001-000",
            state="SP",
            city="São Paulo",
            neighborhood="Sé",
            street="Praça da Sé",
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_cep",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_cep("01001000", ctx)
        assert "01001-000" in result
        assert "Praça da Sé" in result
        assert "Sé" in result
        assert "São Paulo" in result
        assert "SP" in result

    @pytest.mark.asyncio
    async def test_missing_optional_fields(self) -> None:
        mock_data = Endereco(
            cep="69900-000",
            state="AC",
            city="Rio Branco",
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_cep",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_cep("69900000", ctx)
        assert "N/A" in result
        assert "Rio Branco" in result
        assert "AC" in result


# ---------------------------------------------------------------------------
# consultar_cnpj
# ---------------------------------------------------------------------------


class TestConsultarCnpj:
    @pytest.mark.asyncio
    async def test_formats_company(self) -> None:
        mock_data = EmpresaCnpj(
            cnpj="00000000000191",
            razao_social="BANCO DO BRASIL SA",
            nome_fantasia="BANCO DO BRASIL",
            descricao_situacao_cadastral="ATIVA",
            porte="DEMAIS",
            natureza_juridica="Sociedade de Economia Mista",
            cnae_fiscal=6422100,
            cnae_fiscal_descricao="Bancos múltiplos",
            logradouro="SAUN Q 5 L B",
            numero="S/N",
            complemento="ANDAR 1 A 16",
            bairro="ASA NORTE",
            cep="70040912",
            uf="DF",
            municipio="BRASILIA",
            ddd_telefone_1="6134939002",
            email="di.gerat@bb.com.br",
            capital_social=120000000000.0,
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_cnpj",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_cnpj("00000000000191", ctx)
        assert "BANCO DO BRASIL SA" in result
        assert "ATIVA" in result
        assert "DF" in result
        assert "BRASILIA" in result
        assert "R$" in result

    @pytest.mark.asyncio
    async def test_missing_optional_fields(self) -> None:
        mock_data = EmpresaCnpj(cnpj="12345678000100")
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_cnpj",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_cnpj("12345678000100", ctx)
        assert "12345678000100" in result
        assert "N/A" in result


# ---------------------------------------------------------------------------
# consultar_ddd
# ---------------------------------------------------------------------------


class TestConsultarDdd:
    @pytest.mark.asyncio
    async def test_formats_ddd(self) -> None:
        mock_data = DddInfo(
            state="SP",
            cities=["São Paulo", "Guarulhos", "Osasco"],
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_ddd",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_ddd("11", ctx)
        assert "SP" in result
        assert "3" in result  # count of cities
        assert "Guarulhos" in result
        assert "São Paulo" in result

    @pytest.mark.asyncio
    async def test_empty_cities(self) -> None:
        mock_data = DddInfo(state="AC", cities=[])
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_ddd",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_ddd("68", ctx)
        assert "AC" in result
        assert "0" in result


# ---------------------------------------------------------------------------
# listar_bancos
# ---------------------------------------------------------------------------


class TestListarBancos:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Banco(ispb="00000000", name="BCO DO BRASIL S.A.", code=1, fullName="Banco do Brasil"),
            Banco(ispb="60701190", name="ITAÚ UNIBANCO S.A.", code=341, fullName="Itaú"),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_bancos",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.listar_bancos(ctx)
        assert "BCO DO BRASIL" in result
        assert "ITAÚ" in result
        assert "341" in result

    @pytest.mark.asyncio
    async def test_empty_list(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_bancos",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.listar_bancos(ctx)
        assert isinstance(result, str)


# ---------------------------------------------------------------------------
# consultar_feriados
# ---------------------------------------------------------------------------


class TestConsultarFeriados:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            Feriado(date="2024-01-01", name="Confraternização Universal", type="national"),
            Feriado(date="2024-04-21", name="Tiradentes", type="national"),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_feriados",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_feriados(2024, ctx)
        assert "Confraternização Universal" in result
        assert "Tiradentes" in result
        assert "2024-01-01" in result

    @pytest.mark.asyncio
    async def test_empty_year(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_feriados",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.consultar_feriados(1800, ctx)
        assert isinstance(result, str)


# ---------------------------------------------------------------------------
# consultar_taxa
# ---------------------------------------------------------------------------


class TestConsultarTaxa:
    @pytest.mark.asyncio
    async def test_formats_rate(self) -> None:
        mock_data = TaxaOficial(nome="Selic", valor=13.75)
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_taxa",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_taxa("SELIC", ctx)
        assert "Selic" in result
        assert "%" in result
        assert "taxa básica de juros" in result.lower() or "Selic" in result

    @pytest.mark.asyncio
    async def test_unknown_rate(self) -> None:
        mock_data = TaxaOficial(nome="XYZ", valor=1.23)
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_taxa",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_taxa("XYZ", ctx)
        assert "XYZ" in result
        assert "%" in result
        # Unknown rate should NOT have the "Descrição" line
        assert "Descrição" not in result


# ---------------------------------------------------------------------------
# listar_marcas_fipe
# ---------------------------------------------------------------------------


class TestListarMarcasFipe:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            FipeMarca(nome="Fiat", valor="21"),
            FipeMarca(nome="Volkswagen", valor="59"),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.listar_marcas_fipe",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.listar_marcas_fipe("carros", ctx)
        assert "Fiat" in result
        assert "Volkswagen" in result

    @pytest.mark.asyncio
    async def test_invalid_type(self) -> None:
        ctx = _mock_ctx()
        result = await tools.listar_marcas_fipe("bicicletas", ctx)
        assert "Tipo inválido" in result
        assert "bicicletas" in result
        assert "carros" in result
        assert "caminhoes" in result
        assert "motos" in result


# ---------------------------------------------------------------------------
# consultar_isbn
# ---------------------------------------------------------------------------


class TestConsultarIsbn:
    @pytest.mark.asyncio
    async def test_formats_book(self) -> None:
        mock_data = Livro(
            isbn="9788535902778",
            title="Grande Sertão: Veredas",
            subtitle="Romance",
            authors=["João Guimarães Rosa"],
            publisher="Nova Fronteira",
            year=2001,
            page_count=624,
            subjects=["Ficção brasileira", "Romance"],
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_isbn",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_isbn("9788535902778", ctx)
        assert "Grande Sertão: Veredas" in result
        assert "João Guimarães Rosa" in result
        assert "Nova Fronteira" in result
        assert "624" in result
        assert "Subtítulo" in result
        assert "Romance" in result
        assert "Ficção brasileira" in result

    @pytest.mark.asyncio
    async def test_minimal_book(self) -> None:
        mock_data = Livro(isbn="1234567890")
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_isbn",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_isbn("1234567890", ctx)
        assert "1234567890" in result
        assert "N/A" in result
        # No subtitle or subjects sections
        assert "Subtítulo" not in result
        assert "Assuntos" not in result


# ---------------------------------------------------------------------------
# buscar_ncm
# ---------------------------------------------------------------------------


class TestBuscarNcm:
    @pytest.mark.asyncio
    async def test_formats_table(self) -> None:
        mock_data = [
            NcmItem(codigo="01012100", descricao="Cavalos reprodutores de raça pura"),
            NcmItem(codigo="01012900", descricao="Outros cavalos vivos"),
        ]
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_ncm",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.buscar_ncm("cavalo", ctx)
        assert "01012100" in result
        assert "Cavalos reprodutores" in result

    @pytest.mark.asyncio
    async def test_empty_results(self) -> None:
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.buscar_ncm",
            new_callable=AsyncMock,
            return_value=[],
        ):
            result = await tools.buscar_ncm("xyzabc123", ctx)
        assert "Nenhum NCM encontrado" in result
        assert "xyzabc123" in result


# ---------------------------------------------------------------------------
# consultar_registro_br
# ---------------------------------------------------------------------------


class TestConsultarRegistroBr:
    @pytest.mark.asyncio
    async def test_registered_domain(self) -> None:
        mock_data = RegistroBrDominio(
            status_code=2,
            status="REGISTERED",
            fqdn="google.com.br",
            hosts=["ns1.google.com", "ns2.google.com"],
            expires_at="2025-12-01",
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_registro_br",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_registro_br("google.com.br", ctx)
        assert "google.com.br" in result
        assert "Registrado" in result
        assert "2025-12-01" in result
        assert "ns1.google.com" in result

    @pytest.mark.asyncio
    async def test_available_domain(self) -> None:
        # Note: status_code=0 is falsy in Python, so `0 or -1` == -1
        # status_code=0 maps to "Disponível para registro" in the status_map.
        mock_data = RegistroBrDominio(
            status_code=0,
            status="AVAILABLE",
            fqdn="meudominioinexistente.com.br",
        )
        ctx = _mock_ctx()
        with patch(
            f"{CLIENT_MODULE}.consultar_registro_br",
            new_callable=AsyncMock,
            return_value=mock_data,
        ):
            result = await tools.consultar_registro_br("meudominioinexistente.com.br", ctx)
        assert "meudominioinexistente.com.br" in result
        assert "Disponível para registro" in result
        # No expiry or DNS for available domains
        assert "Expira em" not in result
        assert "DNS" not in result
