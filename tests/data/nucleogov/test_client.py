"""Tests for the NucleoGov HTTP client."""

import json
import urllib.parse

import httpx
import pytest
import respx

from mcp_brasil.data.nucleogov import client
from mcp_brasil.exceptions import HttpClientError

PORTAL = "https://acessoainformacao.cidade.go.gov.br"
DESPESAS_URL = f"{PORTAL}/api/despesas_frl/listar"
CONTRATOS_URL = f"{PORTAL}/api/contratos_frl/listar"
LICITACOES_URL = f"{PORTAL}/api/licitacoes_frl/listar"
ATOS_URL = f"{PORTAL}/api/atos_pf_cidade/listar"


def _envelope(dados: list[dict[str, object]], total: int | None = None) -> dict[str, object]:
    return {
        "0-mcp": {
            "dados": dados,
            "total": total if total is not None else len(dados),
        }
    }


# ---------------------------------------------------------------------------
# buscar_despesas
# ---------------------------------------------------------------------------


class TestBuscarDespesas:
    @pytest.mark.asyncio
    @respx.mock
    async def test_parses_despesas(self) -> None:
        respx.post(DESPESAS_URL).mock(
            return_value=httpx.Response(
                200,
                json=_envelope(
                    [
                        {
                            "id": "1",
                            "orgao": "PREFEITURA MUNICIPAL DE CIDADE",
                            "unidade_nome": "SECRETARIA DE SAUDE",
                            "numero_processo": "1000",
                            "numero_empenho": "42",
                            "data": "31/12/2024",
                            "fornecedor": "Empresa Exemplo Ltda",
                            "cpf_cnpj": "12.345.678/0001-90",
                            "historico": "Aquisicao de material",
                            "valor_empenho": "500,00",
                            "valor_pago": "0,00",
                        }
                    ]
                ),
            )
        )
        resultado = await client.buscar_despesas(
            PORTAL, data_inicio="2024-01-01", data_fim="2024-12-31"
        )
        assert resultado.total == 1
        assert len(resultado.despesas) == 1
        despesa = resultado.despesas[0]
        assert despesa.fornecedor == "Empresa Exemplo Ltda"
        assert despesa.unidade == "SECRETARIA DE SAUDE"
        assert despesa.valor_empenho == 500.0
        assert despesa.valor_pago == 0.0

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_page(self) -> None:
        respx.post(DESPESAS_URL).mock(
            return_value=httpx.Response(200, json=_envelope([], total=0))
        )
        resultado = await client.buscar_despesas(PORTAL)
        assert resultado.despesas == []
        assert resultado.total == 0

    @pytest.mark.asyncio
    @respx.mock
    async def test_filters_by_cnpj_locally(self) -> None:
        respx.post(DESPESAS_URL).mock(
            return_value=httpx.Response(
                200,
                json=_envelope(
                    [
                        {"id": "1", "cpf_cnpj": "12.345.678/0001-90"},
                        {"id": "2", "cpf_cnpj": "00000000000000"},
                    ]
                ),
            )
        )
        resultado = await client.buscar_despesas(PORTAL, cnpj="12345678000190")
        assert len(resultado.despesas) == 1
        assert resultado.despesas[0].id == "1"

    @pytest.mark.asyncio
    @respx.mock
    async def test_normalizes_portal_without_scheme(self) -> None:
        respx.post(DESPESAS_URL).mock(
            return_value=httpx.Response(200, json=_envelope([], total=0))
        )
        resultado = await client.buscar_despesas("acessoainformacao.cidade.go.gov.br")
        assert resultado.total == 0

    @pytest.mark.asyncio
    @respx.mock
    async def test_converts_period_to_brazilian_format(self) -> None:
        route = respx.post(DESPESAS_URL).mock(
            return_value=httpx.Response(200, json=_envelope([], total=0))
        )
        await client.buscar_despesas(PORTAL, data_inicio="2024-03", data_fim="31/12/2024")
        body = urllib.parse.parse_qs(route.calls.last.request.content.decode("utf-8"))
        params = json.loads(body["params"][0])
        inner = params["0-mcp"]
        assert inner["datamin"] == "01/03/2024"
        assert inner["datamax"] == "31/12/2024"
        assert inner["acao"] == "despesas_frl/listar"

    @pytest.mark.asyncio
    async def test_invalid_date_raises(self) -> None:
        with pytest.raises(ValueError, match="data_inicio"):
            await client.buscar_despesas(PORTAL, data_inicio="2024/01/01")

    @pytest.mark.asyncio
    @respx.mock
    async def test_string_error_becomes_http_client_error(self) -> None:
        respx.post(DESPESAS_URL).mock(
            return_value=httpx.Response(200, json={"0-mcp": "Acao nao encontrada"})
        )
        with pytest.raises(HttpClientError, match="recusou"):
            await client.buscar_despesas(PORTAL)

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_object_becomes_http_client_error(self) -> None:
        respx.post(DESPESAS_URL).mock(return_value=httpx.Response(200, json={}))
        with pytest.raises(HttpClientError, match="não reconhecida"):
            await client.buscar_despesas(PORTAL)

    @pytest.mark.asyncio
    @respx.mock
    async def test_403_becomes_http_client_error(self) -> None:
        respx.post(DESPESAS_URL).mock(return_value=httpx.Response(403, text="Forbidden"))
        with pytest.raises(HttpClientError, match="403"):
            await client.buscar_despesas(PORTAL)

    @pytest.mark.asyncio
    @respx.mock
    async def test_500_becomes_http_client_error(self) -> None:
        respx.post(DESPESAS_URL).mock(return_value=httpx.Response(500, text="boom"))
        with pytest.raises(HttpClientError, match="HTTP 500"):
            await client.buscar_despesas(PORTAL)

    @pytest.mark.asyncio
    @respx.mock
    async def test_month_end_covers_whole_month(self) -> None:
        route = respx.post(DESPESAS_URL).mock(
            return_value=httpx.Response(200, json=_envelope([], total=0))
        )
        await client.buscar_despesas(PORTAL, data_inicio="2024-03", data_fim="2024-03")
        body = urllib.parse.parse_qs(route.calls.last.request.content.decode("utf-8"))
        inner = json.loads(body["params"][0])["0-mcp"]
        assert inner["datamin"] == "01/03/2024"
        assert inner["datamax"] == "31/03/2024"

    @pytest.mark.asyncio
    @respx.mock
    async def test_month_end_handles_leap_february(self) -> None:
        route = respx.post(DESPESAS_URL).mock(
            return_value=httpx.Response(200, json=_envelope([], total=0))
        )
        await client.buscar_despesas(PORTAL, data_fim="2024-02")
        inner = json.loads(
            urllib.parse.parse_qs(route.calls.last.request.content.decode("utf-8"))["params"][0]
        )["0-mcp"]
        assert inner["datamax"] == "29/02/2024"

        await client.buscar_despesas(PORTAL, data_fim="2023-02")
        inner = json.loads(
            urllib.parse.parse_qs(route.calls.last.request.content.decode("utf-8"))["params"][0]
        )["0-mcp"]
        assert inner["datamax"] == "28/02/2023"

    @pytest.mark.asyncio
    async def test_invalid_calendar_date_raises(self) -> None:
        with pytest.raises(ValueError, match="data_fim"):
            await client.buscar_despesas(PORTAL, data_fim="2024-13")
        with pytest.raises(ValueError, match="data_inicio"):
            await client.buscar_despesas(PORTAL, data_inicio="31/02/2024")

    @pytest.mark.asyncio
    @respx.mock
    async def test_structured_error_envelope_never_becomes_empty_list(self) -> None:
        respx.post(DESPESAS_URL).mock(
            return_value=httpx.Response(200, json={"0-mcp": {"erro": "falha"}})
        )
        with pytest.raises(HttpClientError, match="recusou"):
            await client.buscar_despesas(PORTAL)

    @pytest.mark.asyncio
    @respx.mock
    async def test_unexpected_envelope_never_becomes_empty_list(self) -> None:
        respx.post(DESPESAS_URL).mock(
            return_value=httpx.Response(200, json={"0-mcp": {"algo": "inesperado"}})
        )
        with pytest.raises(HttpClientError, match="inesperada"):
            await client.buscar_despesas(PORTAL)

    @pytest.mark.asyncio
    @respx.mock
    async def test_metadata_only_envelope_is_legitimate_empty(self) -> None:
        respx.post(DESPESAS_URL).mock(
            return_value=httpx.Response(200, json={"0-mcp": {"total": 0}})
        )
        resultado = await client.buscar_despesas(PORTAL)
        assert resultado.despesas == []
        assert resultado.total == 0

    @pytest.mark.asyncio
    @respx.mock
    async def test_cnpj_uses_remote_filter(self) -> None:
        route = respx.post(DESPESAS_URL).mock(
            return_value=httpx.Response(200, json=_envelope([], total=0))
        )
        await client.buscar_despesas(PORTAL, cnpj="12.345.678/0001-90")
        inner = json.loads(
            urllib.parse.parse_qs(route.calls.last.request.content.decode("utf-8"))["params"][0]
        )["0-mcp"]
        assert inner["cpf_cnpj"] == "12.345.678/0001-90"

    @pytest.mark.asyncio
    @respx.mock
    async def test_cnpj_match_outside_first_page_is_not_lost(self) -> None:
        respx.post(DESPESAS_URL).mock(
            return_value=httpx.Response(
                200,
                json=_envelope(
                    [
                        {"id": "1", "cpf_cnpj": "12.345.678/0001-90"},
                        {"id": "2", "cpf_cnpj": "00000000000000"},
                    ],
                    total=5,
                ),
            )
        )
        resultado = await client.buscar_despesas(PORTAL, cnpj="12345678000190", limite=2)
        assert [d.id for d in resultado.despesas] == ["1"]
        assert resultado.total == 5

    @pytest.mark.asyncio
    @respx.mock
    async def test_does_not_follow_redirect_outside_allowlist(self) -> None:
        respx.post(DESPESAS_URL).mock(
            return_value=httpx.Response(
                302, headers={"Location": "https://evil.example.com/steal"}
            )
        )
        externo = respx.post("https://evil.example.com/steal").mock(
            return_value=httpx.Response(200, json=_envelope([]))
        )
        with pytest.raises(HttpClientError, match="redirecionou"):
            await client.buscar_despesas(PORTAL)
        assert externo.call_count == 0


# ---------------------------------------------------------------------------
# SSRF — nenhum host fora da allowlist recebe requisição
# ---------------------------------------------------------------------------

#: Placeholder credentials (never real) for the credential-rejection test.
#: Built from parts so the fixture does not look like an e-mail address.
_CREDENCIAIS = "user" + ":" + "secret" + "@"

PORTAL_MALICIOSO = [
    "http://acessoainformacao.cidade.go.gov.br",
    "ftp://acessoainformacao.cidade.go.gov.br",
    "https://127.0.0.1:9999",
    "https://169.254.169.254/latest/meta-data",
    "https://[::1]/",
    "https://localhost",
    "https://acessoainformacao.cidade.go.gov.br:8443",
    "https://" + _CREDENCIAIS + "acessoainformacao.cidade.go.gov.br",
    "https://evil.example.com",
    "https://acessoainformacao.cidade.go.gov.br.evil.com",
    "https://acessoainformacao.cidade.go.gov.br?x=1",
]

ENTRADAS_CLIENT = {
    "buscar_despesas": lambda portal: client.buscar_despesas(portal),
    "buscar_contratos": lambda portal: client.buscar_contratos(portal),
    "buscar_licitacoes": lambda portal: client.buscar_licitacoes(portal),
    "buscar_legislacao": lambda portal: client.buscar_legislacao(portal, municipio="Cidade"),
}


class TestSsrf:
    @pytest.mark.asyncio
    @pytest.mark.parametrize("portal", PORTAL_MALICIOSO)
    @respx.mock
    async def test_rejects_portal_outside_allowlist(self, portal: str) -> None:
        route = respx.route().mock(return_value=httpx.Response(200, json={}))
        with pytest.raises(ValueError):
            await client.buscar_despesas(portal)
        assert route.call_count == 0, f"houve requisição para {portal!r}"

    @pytest.mark.asyncio
    @pytest.mark.parametrize("nome", sorted(ENTRADAS_CLIENT))
    @respx.mock
    async def test_every_entrypoint_rejects_outside_allowlist(self, nome: str) -> None:
        route = respx.route().mock(return_value=httpx.Response(200, json={}))
        with pytest.raises(ValueError):
            await ENTRADAS_CLIENT[nome]("https://evil.example.com")
        assert route.call_count == 0, f"{nome} fez requisição fora da allowlist"

    @pytest.mark.asyncio
    @respx.mock
    async def test_accepts_gov_br_portal(self) -> None:
        respx.post(DESPESAS_URL).mock(
            return_value=httpx.Response(200, json=_envelope([], total=0))
        )
        resultado = await client.buscar_despesas(PORTAL)
        assert resultado.total == 0


# ---------------------------------------------------------------------------
# buscar_contratos
# ---------------------------------------------------------------------------


class TestBuscarContratos:
    @pytest.mark.asyncio
    @respx.mock
    async def test_parses_contratos(self) -> None:
        respx.post(CONTRATOS_URL).mock(
            return_value=httpx.Response(
                200,
                json=_envelope(
                    [
                        {
                            "id": "10",
                            "numero": "001/2024",
                            "ano": 2024,
                            "orgao": "PREFEITURA MUNICIPAL DE CIDADE",
                            "situacao": "Ativo",
                            "contratado_nome": "Empresa Exemplo Ltda",
                            "contratado_documento": "12.345.678/0001-90",
                            "objeto": "Locacao de veiculos",
                            "valor": "10.000,00",
                        }
                    ]
                ),
            )
        )
        resultado = await client.buscar_contratos(PORTAL, ano=2024)
        assert resultado.total == 1
        contrato = resultado.contratos[0]
        assert contrato.numero == "001/2024"
        assert contrato.ano == 2024
        assert contrato.valor == 10000.0

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_page(self) -> None:
        respx.post(CONTRATOS_URL).mock(
            return_value=httpx.Response(200, json=_envelope([], total=0))
        )
        resultado = await client.buscar_contratos(PORTAL)
        assert resultado.contratos == []


# ---------------------------------------------------------------------------
# buscar_licitacoes
# ---------------------------------------------------------------------------


class TestBuscarLicitacoes:
    @pytest.mark.asyncio
    @respx.mock
    async def test_parses_licitacoes(self) -> None:
        respx.post(LICITACOES_URL).mock(
            return_value=httpx.Response(
                200,
                json=_envelope(
                    [
                        {
                            "id": "20",
                            "numero": "001/2024",
                            "modalidade": "PREGÃO ELETRÔNICO",
                            "situacao": "Aberta",
                            "orgao": "PREFEITURA MUNICIPAL DE CIDADE",
                            "ano": 2024,
                            "descricao": "Aquisicao de equipamentos",
                            "valor_estimado": "100.000,00",
                        }
                    ]
                ),
            )
        )
        resultado = await client.buscar_licitacoes(PORTAL, ano=2024)
        assert resultado.total == 1
        licitacao = resultado.licitacoes[0]
        assert licitacao.modalidade == "PREGÃO ELETRÔNICO"
        assert licitacao.valor_estimado == 100000.0

    @pytest.mark.asyncio
    @respx.mock
    async def test_dispensas_filter(self) -> None:
        route = respx.post(LICITACOES_URL).mock(
            return_value=httpx.Response(200, json=_envelope([], total=0))
        )
        await client.buscar_licitacoes(PORTAL, dispensas=True)
        body = urllib.parse.parse_qs(route.calls.last.request.content.decode("utf-8"))
        inner = json.loads(body["params"][0])["0-mcp"]
        assert inner["dispensas"] == 1


# ---------------------------------------------------------------------------
# buscar_legislacao
# ---------------------------------------------------------------------------


class TestBuscarLegislacao:
    @pytest.mark.asyncio
    @respx.mock
    async def test_parses_atos(self) -> None:
        respx.post(ATOS_URL).mock(
            return_value=httpx.Response(
                200,
                json=_envelope(
                    [
                        {
                            "id": "30",
                            "tipo": "DECRETO",
                            "numero": "100",
                            "ementa": "Dispoe sobre a estrutura administrativa",
                            "data_publicacao": "01/02/2024",
                            "url": "/cidadao/legislacao/decreto/30",
                        }
                    ]
                ),
            )
        )
        resultado = await client.buscar_legislacao(PORTAL, municipio="Cidade", tipo="decreto")
        assert resultado.total == 1
        ato = resultado.atos[0]
        assert ato.tipo == "DECRETO"
        assert ato.ementa.startswith("Dispoe")

    @pytest.mark.asyncio
    @respx.mock
    async def test_slug_normalizes_accented_name(self) -> None:
        url = f"{PORTAL}/api/atos_pf_senador_canedo/listar"
        respx.post(url).mock(return_value=httpx.Response(200, json=_envelope([], total=0)))
        resultado = await client.buscar_legislacao(PORTAL, municipio="Senador Canedo")
        assert resultado.total == 0

    @pytest.mark.asyncio
    @respx.mock
    async def test_accepts_explicit_slug(self) -> None:
        url = f"{PORTAL}/api/atos_pf_senador_canedo/listar"
        respx.post(url).mock(return_value=httpx.Response(200, json=_envelope([], total=0)))
        resultado = await client.buscar_legislacao(PORTAL, municipio="senador_canedo")
        assert resultado.total == 0

    @pytest.mark.asyncio
    async def test_invalid_municipio_raises(self) -> None:
        with pytest.raises(ValueError, match="município"):
            await client.buscar_legislacao(PORTAL, municipio="   ")
