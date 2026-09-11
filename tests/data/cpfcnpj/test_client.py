"""Testes do cliente HTTP da CPF.CNPJ (respx).

Fixtures anonimizadas, fiéis ao JSON real (token público de testes devolve
documentos fictícios). O token nunca aparece em log, exceção ou fixture.
"""

from typing import Any

import httpx
import pytest
import respx

from mcp_brasil.data.cpfcnpj import client
from mcp_brasil.data.cpfcnpj.constants import CPFCNPJ_BASE

TOKEN = "testtoken"


def _url(pacote: str, documento: str) -> str:
    return f"{CPFCNPJ_BASE}/{TOKEN}/{pacote}/{documento}"


CNPJ_5: dict[str, Any] = {
    "status": 1,
    "cnpj": "11.222.333/0001-81",
    "tipo": "Matriz",
    "razao": "EMPRESA EXEMPLO LTDA",
    "fantasia": "EXEMPLO",
    "matrizEndereco": {
        "cep": "01001000",
        "tipo": "AVENIDA",
        "logradouro": "DAS FLORES",
        "numero": "100",
        "complemento": "SALA 1",
        "bairro": "CENTRO",
        "cidade": "SAO PAULO",
        "uf": "SP",
    },
    "ibge": {
        "estado": {"id": 35, "nome": "São Paulo", "sigla": "SP", "ibge_id": 35},
        "cidade": {"id": 3550308, "nome": "São Paulo", "ibge_id": 3550308, "siafi_id": 7107},
    },
    "pacoteUsado": 5,
    "saldo": 123,
    "consultaID": "11bb22cc33dd44ee",
    "delay": 0.3,
}

CNPJ_6: dict[str, Any] = {
    "status": 1,
    "cnpj": "11.222.333/0001-81",
    "tipo": "Matriz",
    "razao": "EMPRESA EXEMPLO LTDA",
    "fantasia": "EXEMPLO",
    "inicioAtividade": "01/01/2010",
    "email": "contato@exemplo.com.br",
    "responsavel": "Joao Jose",
    "responsavelCpf": "111.111.111-11",
    "situacao": {
        "id": 2,
        "nome": "ATIVA",
        "data": "01/01/2010",
        "motivo": {"id": 0, "descricao": "SEM MOTIVO"},
    },
    "naturezaJuridica": {"codigo": "2062", "descricao": "Sociedade Empresária Limitada"},
    "porte": {"id": 3, "descricao": "DEMAIS"},
    "cnae": {
        "fiscal": "6202300",
        "descricao": "Desenvolvimento de programas de computador sob encomenda",
        "secundarias": [
            {"id": 6203100, "descricao": "Desenvolvimento de programas customizáveis"}
        ],
    },
    "simplesNacional": {"optante": False, "inicio": None, "fim": None, "mei": False},
    "regimesTributarios": [
        {"ano": 2023, "regime_tributario": "LUCRO PRESUMIDO", "forma_de_tributacao": "-"}
    ],
    "socios": [
        {
            "cpf_cnpj_socio": "111.111.111-11",
            "nome": "Socio Teste",
            "tipo": "PESSOA FISICA",
            "data_entrada": "2018-11-14",
            "qualificacao_socio": {"id": 49, "descricao": "Sócio-Administrador"},
        },
        {
            "cpf_cnpj_socio": "11.111.111/0001-11",
            "nome": "CORPORATION, INC.",
            "tipo": "PESSOA JURIDICA",
            "data_entrada": "2015-09-17",
            "qualificacao_socio": {"id": 22, "descricao": "Sócio Pessoa Jurídica"},
        },
    ],
    "pacoteUsado": 6,
    "saldo": 123,
    "consultaID": "11bb22cc33dd44ee",
}

CNPJ_16: dict[str, Any] = {
    "status": 1,
    "cnpj": "11.222.333/0001-81",
    "razao": "EMPRESA EXEMPLO LTDA",
    "inscricoesEstaduais": [
        {
            "inscricao_estadual": "111111111111",
            "ativo": True,
            "atualizado_em": "2026-01-01T00:00:00Z",
            "estado": {"id": 35, "nome": "São Paulo", "sigla": "SP", "ibge_id": 35},
        },
        {
            "inscricao_estadual": "222222222",
            "ativo": False,
            "atualizado_em": "2026-01-01T00:00:00Z",
            "estado": {"id": 33, "nome": "Rio de Janeiro", "sigla": "RJ", "ibge_id": 33},
        },
    ],
    "pacoteUsado": 16,
    "saldo": 123,
    "consultaID": "11bb22cc33dd44ee",
}

SALDO: dict[str, Any] = {
    "status": 1,
    "pacote": {"id": 5, "nome": "CNPJ B", "saldo": 123},
    "pacoteUsado": 5,
}


@pytest.fixture(autouse=True)
def _set_token(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("CPFCNPJ_TOKEN", TOKEN)


class TestRateLimiter:
    def test_config(self) -> None:
        assert client._rate_limiter._max_requests == 20
        assert client._rate_limiter._period == 1.0


class TestConsultarCnpj:
    @pytest.mark.asyncio
    @respx.mock
    async def test_pacote_5(self) -> None:
        respx.get(_url("5", "11222333000181")).mock(return_value=httpx.Response(200, json=CNPJ_5))
        emp = await client.consultar_cnpj("11222333000181", 5)
        assert emp.razao == "EMPRESA EXEMPLO LTDA"
        assert emp.matrizEndereco is not None
        assert emp.matrizEndereco.uf == "SP"
        assert emp.ibge is not None
        assert emp.ibge.cidade is not None
        assert emp.ibge.cidade.ibge_id == 3550308

    @pytest.mark.asyncio
    @respx.mock
    async def test_pacote_6(self) -> None:
        respx.get(_url("6", "11222333000181")).mock(return_value=httpx.Response(200, json=CNPJ_6))
        emp = await client.consultar_cnpj("11222333000181", 6)
        assert emp.situacao is not None
        assert emp.situacao.nome == "ATIVA"
        assert len(emp.socios) == 2
        assert emp.socios[0].qualificacao_socio is not None
        assert emp.cnae is not None
        assert emp.cnae.fiscal == "6202300"

    @pytest.mark.asyncio
    @respx.mock
    async def test_status_como_string(self) -> None:
        # A API pode devolver status como string ("1"); deve ser sucesso.
        resp = {**CNPJ_5, "status": "1"}
        respx.get(_url("5", "11222333000181")).mock(return_value=httpx.Response(200, json=resp))
        emp = await client.consultar_cnpj("11222333000181", 5)
        assert emp.razao == "EMPRESA EXEMPLO LTDA"

    @pytest.mark.asyncio
    @respx.mock
    async def test_alfanumerico(self) -> None:
        alfa = {**CNPJ_5, "cnpj": "12.ABC.345/01DE-35"}
        respx.get(_url("5", "12ABC34501DE35")).mock(return_value=httpx.Response(200, json=alfa))
        emp = await client.consultar_cnpj("12ABC34501DE35", 5)
        assert emp.cnpj == "12.ABC.345/01DE-35"


class TestInscricoesEstaduais:
    @pytest.mark.asyncio
    @respx.mock
    async def test_pacote_16(self) -> None:
        respx.get(_url("16", "11222333000181")).mock(
            return_value=httpx.Response(200, json=CNPJ_16)
        )
        emp = await client.inscricoes_estaduais("11222333000181")
        assert len(emp.inscricoesEstaduais) == 2
        assert emp.inscricoesEstaduais[0].estado is not None
        assert emp.inscricoesEstaduais[0].estado.sigla == "SP"


class TestSaldo:
    @pytest.mark.asyncio
    @respx.mock
    async def test_saldo(self) -> None:
        respx.get(_url("saldo", "5")).mock(return_value=httpx.Response(200, json=SALDO))
        saldo = await client.saldo(5)
        assert saldo.id == 5
        assert saldo.nome == "CNPJ B"
        assert saldo.saldo == 123


class TestErros:
    @pytest.mark.asyncio
    @respx.mock
    async def test_erro_negocio_201(self) -> None:
        respx.get(_url("5", "11222333000181")).mock(
            return_value=httpx.Response(
                400,
                json={
                    "status": 0,
                    "erro": "Informe um CNPJ!",
                    "pacoteUsado": 5,
                    "erroCodigo": 201,
                },
            )
        )
        with pytest.raises(client.CpfCnpjError) as exc:
            await client.consultar_cnpj("11222333000181", 5)
        assert exc.value.codigo == 201
        assert "inválido ou incompleto" in exc.value.mensagem

    @pytest.mark.asyncio
    @respx.mock
    async def test_erro_negocio_1001(self) -> None:
        respx.get(_url("5", "11222333000181")).mock(
            return_value=httpx.Response(
                400, json={"status": 0, "erro": "Sem crédito", "erroCodigo": 1001}
            )
        )
        with pytest.raises(client.CpfCnpjError) as exc:
            await client.consultar_cnpj("11222333000181", 5)
        assert exc.value.codigo == 1001
        assert "créditos" in exc.value.mensagem.lower()

    @pytest.mark.asyncio
    @respx.mock
    async def test_erro_negocio_codigo_string(self) -> None:
        # erroCodigo pode vir como string ("1001"); deve mapear sem crédito.
        respx.get(_url("5", "11222333000181")).mock(
            return_value=httpx.Response(
                400, json={"status": "0", "erro": "Sem crédito", "erroCodigo": "1001"}
            )
        )
        with pytest.raises(client.CpfCnpjError) as exc:
            await client.consultar_cnpj("11222333000181", 5)
        assert exc.value.codigo == 1001
        assert "créditos" in exc.value.mensagem.lower()

    @pytest.mark.asyncio
    @respx.mock
    async def test_erro_negocio_1007(self) -> None:
        respx.get(_url("5", "11222333000181")).mock(
            return_value=httpx.Response(
                400, json={"status": 0, "erro": "limite", "erroCodigo": 1007}
            )
        )
        with pytest.raises(client.CpfCnpjError) as exc:
            await client.consultar_cnpj("11222333000181", 5)
        assert exc.value.codigo == 1007

    @pytest.mark.asyncio
    @respx.mock
    async def test_erro_gateway(self) -> None:
        respx.get(_url("5", "11222333000181")).mock(
            return_value=httpx.Response(
                400,
                json={"status": "error", "code": 400, "message": "Incorrect parameters"},
            )
        )
        with pytest.raises(client.CpfCnpjError) as exc:
            await client.consultar_cnpj("11222333000181", 5)
        assert exc.value.codigo == 400
        assert "Incorrect parameters" in exc.value.mensagem

    @pytest.mark.asyncio
    @respx.mock
    async def test_corpo_html_nao_json(self) -> None:
        respx.get(_url("5", "11222333000181")).mock(
            return_value=httpx.Response(404, text="<html>404 not found</html>")
        )
        with pytest.raises(client.CpfCnpjError) as exc:
            await client.consultar_cnpj("11222333000181", 5)
        assert "/***/" in exc.value.mensagem

    @pytest.mark.asyncio
    @respx.mock
    async def test_token_nao_aparece_na_excecao(self) -> None:
        respx.get(_url("5", "11222333000181")).mock(
            return_value=httpx.Response(404, text="<html>erro</html>")
        )
        with pytest.raises(client.CpfCnpjError) as exc:
            await client.consultar_cnpj("11222333000181", 5)
        assert TOKEN not in str(exc.value)
        assert TOKEN not in exc.value.mensagem

    @pytest.mark.asyncio
    async def test_sem_token(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("CPFCNPJ_TOKEN", "")
        with pytest.raises(client.CpfCnpjError) as exc:
            await client.consultar_cnpj("11222333000181", 5)
        assert exc.value.codigo == 1000


class TestLimiterPorChamadaLogica:
    @pytest.mark.asyncio
    @respx.mock
    async def test_limiter_adquirido_uma_vez_por_chamada(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        # Desvio consciente do molde #142: o limiter é adquirido uma vez por
        # chamada lógica, mesmo quando há retry interno (503 seguido de 200).
        monkeypatch.setattr(client, "HTTP_BACKOFF_BASE", 0.0)
        acquires = 0
        original = client._rate_limiter.acquire

        async def _contar() -> None:
            nonlocal acquires
            acquires += 1
            await original()

        monkeypatch.setattr(client._rate_limiter, "acquire", _contar)

        route = respx.get(_url("5", "11222333000181"))
        route.side_effect = [
            httpx.Response(503),
            httpx.Response(200, json=CNPJ_5),
        ]
        emp = await client.consultar_cnpj("11222333000181", 5)
        assert emp.razao == "EMPRESA EXEMPLO LTDA"
        assert acquires == 1
        assert route.call_count == 2
