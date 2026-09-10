"""Testes das tools da CPF.CNPJ (mock do client, nunca HTTP).

Cobrem: validação local antes de chamar o client (assert_not_called),
mascaramento de CPF de sócio/responsável por padrão e liberação via
MCP_BRASIL_LGPD_ALLOW_PII, e ausência de strings malformadas com campo None.
"""

from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from mcp_brasil import settings
from mcp_brasil.data.cpfcnpj import tools
from mcp_brasil.data.cpfcnpj.schemas import (
    Cnae,
    EmpresaCnpj,
    Ibge,
    IbgeCidade,
    IbgeEstado,
    InscricaoEstadual,
    MatrizEndereco,
    Porte,
    Qualificacao,
    Saldo,
    Situacao,
    Socio,
)

CLIENT = "mcp_brasil.data.cpfcnpj.client"


def _ctx() -> MagicMock:
    ctx = MagicMock()
    ctx.info = AsyncMock()
    ctx.warning = AsyncMock()
    return ctx


def _empresa_completa() -> EmpresaCnpj:
    return EmpresaCnpj(
        status=1,
        cnpj="11.222.333/0001-81",
        tipo="Matriz",
        razao="EMPRESA EXEMPLO LTDA",
        fantasia="EXEMPLO",
        matrizEndereco=MatrizEndereco(
            logradouro="DAS FLORES", numero="100", cidade="SAO PAULO", uf="SP"
        ),
        ibge=Ibge(estado=IbgeEstado(sigla="SP"), cidade=IbgeCidade(ibge_id=3550308)),
        inicioAtividade="01/01/2010",
        responsavel="Joao Jose",
        responsavelCpf="111.111.111-11",
        situacao=Situacao(nome="ATIVA", data="01/01/2010"),
        porte=Porte(descricao="DEMAIS"),
        cnae=Cnae(fiscal="6202300", descricao="Programas sob encomenda"),
        socios=[
            Socio(
                cpf_cnpj_socio="111.111.111-11",
                nome="Socio Teste",
                data_entrada="2018-11-14",
                qualificacao_socio=Qualificacao(id=49, descricao="Sócio-Administrador"),
            ),
            Socio(
                cpf_cnpj_socio="11.111.111/0001-11",
                nome="CORPORATION, INC.",
                data_entrada="2015-09-17",
                qualificacao_socio="SOCIO PESSOA JURIDICA",
            ),
        ],
    )


class TestValidarDocumento:
    @pytest.mark.asyncio
    async def test_cnpj_valido(self) -> None:
        result = await tools.validar_documento("11.222.333/0001-81", _ctx())
        assert "CNPJ" in result
        assert "sim" in result
        assert "11222333000181" in result

    @pytest.mark.asyncio
    async def test_cnpj_alfanumerico(self) -> None:
        result = await tools.validar_documento("12ABC34501DE35", _ctx())
        assert "Alfanumérico:** sim" in result

    @pytest.mark.asyncio
    async def test_cpf(self) -> None:
        result = await tools.validar_documento("111.444.777-35", _ctx())
        assert "CPF" in result
        assert "11144477735" in result

    @pytest.mark.asyncio
    async def test_desconhecido(self) -> None:
        result = await tools.validar_documento("123", _ctx())
        assert "não reconhecido" in result


class TestValidacaoAntesDoClient:
    @pytest.mark.asyncio
    async def test_cnpj_invalido_nao_chama_client(self) -> None:
        with patch(f"{CLIENT}.consultar_cnpj", new_callable=AsyncMock) as mock:
            result = await tools.consultar_cnpj("00000000000000", ctx=_ctx())
        mock.assert_not_called()
        assert "inválido" in result

    @pytest.mark.asyncio
    async def test_cnpj_com_espaco_nao_chama_client(self) -> None:
        with patch(f"{CLIENT}.consultar_cnpj", new_callable=AsyncMock) as mock:
            result = await tools.situacao_cnpj("33 000 167 0001 01", _ctx())
        mock.assert_not_called()
        assert "inválido" in result

    @pytest.mark.asyncio
    async def test_qsa_invalido_nao_chama_client(self) -> None:
        with patch(f"{CLIENT}.consultar_cnpj", new_callable=AsyncMock) as mock:
            await tools.qsa_cnpj("123", _ctx())
        mock.assert_not_called()

    @pytest.mark.asyncio
    async def test_inscricoes_invalido_nao_chama_client(self) -> None:
        with patch(f"{CLIENT}.inscricoes_estaduais", new_callable=AsyncMock) as mock:
            await tools.inscricoes_estaduais("123", _ctx())
        mock.assert_not_called()


class TestConsultarCnpj:
    @pytest.mark.asyncio
    async def test_pacote_5_default(self) -> None:
        with patch(
            f"{CLIENT}.consultar_cnpj", new_callable=AsyncMock, return_value=_empresa_completa()
        ) as mock:
            result = await tools.consultar_cnpj("11222333000181", ctx=_ctx())
        mock.assert_awaited_once_with("11222333000181", 5)
        assert "EMPRESA EXEMPLO LTDA" in result
        assert "SP" in result

    @pytest.mark.asyncio
    async def test_pacote_6_completo(self) -> None:
        with patch(
            f"{CLIENT}.consultar_cnpj", new_callable=AsyncMock, return_value=_empresa_completa()
        ) as mock:
            result = await tools.consultar_cnpj("11222333000181", completo=True, ctx=_ctx())
        mock.assert_awaited_once_with("11222333000181", 6)
        assert "ATIVA" in result
        assert "Quadro de Sócios" in result


class TestMascaramentoLgpd:
    @pytest.mark.asyncio
    async def test_cpf_socio_mascarado_por_padrao(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setattr(settings, "DATASETS_ALLOW_PII", [])
        with patch(
            f"{CLIENT}.consultar_cnpj", new_callable=AsyncMock, return_value=_empresa_completa()
        ):
            result = await tools.qsa_cnpj("11222333000181", _ctx())
        assert "111.111.111-11" not in result  # CPF do sócio mascarado
        assert "***" in result
        assert "11.111.111/0001-11" in result  # CNPJ de sócio PJ sai inteiro

    @pytest.mark.asyncio
    async def test_cpf_socio_inteiro_com_pii_liberada(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        monkeypatch.setattr(settings, "DATASETS_ALLOW_PII", ["cpfcnpj"])
        with patch(
            f"{CLIENT}.consultar_cnpj", new_callable=AsyncMock, return_value=_empresa_completa()
        ):
            result = await tools.qsa_cnpj("11222333000181", _ctx())
        assert "111.111.111-11" in result

    @pytest.mark.asyncio
    async def test_responsavel_cpf_mascarado(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setattr(settings, "DATASETS_ALLOW_PII", [])
        with patch(
            f"{CLIENT}.consultar_cnpj", new_callable=AsyncMock, return_value=_empresa_completa()
        ):
            result = await tools.consultar_cnpj("11222333000181", completo=True, ctx=_ctx())
        assert "Joao Jose" in result
        assert "111.111.111-11" not in result


class TestSemStringMalformada:
    @pytest.mark.asyncio
    async def test_campos_none_nao_geram_none_none(self) -> None:
        vazia = EmpresaCnpj(status=1, cnpj="11.222.333/0001-81", razao="EMPRESA X")
        with patch(f"{CLIENT}.consultar_cnpj", new_callable=AsyncMock, return_value=vazia):
            result = await tools.consultar_cnpj("11222333000181", completo=True, ctx=_ctx())
        assert "(None)" not in result
        assert "None None" not in result
        assert "None," not in result


class TestSituacaoEInscricoes:
    @pytest.mark.asyncio
    async def test_situacao(self) -> None:
        with patch(
            f"{CLIENT}.consultar_cnpj", new_callable=AsyncMock, return_value=_empresa_completa()
        ):
            result = await tools.situacao_cnpj("11222333000181", _ctx())
        assert "ATIVA" in result
        assert "DEMAIS" in result

    @pytest.mark.asyncio
    async def test_inscricoes(self) -> None:
        emp = EmpresaCnpj(
            status=1,
            razao="EMPRESA X",
            inscricoesEstaduais=[
                InscricaoEstadual(
                    inscricao_estadual="111", ativo=True, estado=IbgeEstado(sigla="SP")
                ),
                InscricaoEstadual(
                    inscricao_estadual="222", ativo=False, estado=IbgeEstado(sigla="RJ")
                ),
            ],
        )
        with patch(f"{CLIENT}.inscricoes_estaduais", new_callable=AsyncMock, return_value=emp):
            result = await tools.inscricoes_estaduais("11222333000181", _ctx())
        assert "SP" in result
        assert "ativa" in result
        assert "inativa" in result


class TestSaldo:
    @pytest.mark.asyncio
    async def test_saldo(self) -> None:
        with patch(
            f"{CLIENT}.saldo",
            new_callable=AsyncMock,
            return_value=Saldo(id=5, nome="CNPJ B", saldo=123),
        ):
            result = await tools.saldo(5, _ctx())
        assert "CNPJ B" in result
        assert "123" in result
