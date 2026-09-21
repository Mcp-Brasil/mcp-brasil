"""Tests for the Megasoft HTTP client."""

from __future__ import annotations

from collections.abc import Iterator

import httpx
import pytest
import respx

from mcp_brasil.data.megasoft import client
from mcp_brasil.data.megasoft.constants import (
    CONFIG_PATH,
    EMPENHO_PATH,
    ORDEM_PAGAMENTO_PATH,
    ORDEM_PAGAMENTO_TOTAIS_PATH,
    RECEITA_PATH,
)
from mcp_brasil.exceptions import HttpClientError

PORTAL = "https://exemplo.megasofttransparencia.com.br"
CONFIG_URL = f"{PORTAL}/api/{CONFIG_PATH}"
EMPENHO_URL = f"{PORTAL}/api/{EMPENHO_PATH}"
ORDEM_URL = f"{PORTAL}/api/{ORDEM_PAGAMENTO_PATH}"
TOTAIS_URL = f"{PORTAL}/api/{ORDEM_PAGAMENTO_TOTAIS_PATH}"
RECEITA_URL = f"{PORTAL}/api/{RECEITA_PATH}"

TOKEN = "token-de-teste"


@pytest.fixture(autouse=True)
def _limpar_cache() -> Iterator[None]:
    client.limpar_cache_token()
    yield
    client.limpar_cache_token()


def _mock_config(json_body: dict[str, object] | None = None) -> None:
    respx.get(CONFIG_URL).mock(
        return_value=httpx.Response(200, json=json_body or {"tokenPortal": TOKEN})
    )


# ---------------------------------------------------------------------------
# buscar_empenhos
# ---------------------------------------------------------------------------


class TestBuscarEmpenhos:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_empenhos(self) -> None:
        _mock_config()
        respx.get(EMPENHO_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "total": 1,
                    "registros": [
                        {
                            "codigo": 1,
                            "numeroDoTcm": 1001,
                            "data": "15/03/2024",
                            "nomeDoFornecedor": "Empresa Exemplo Ltda",
                            "cnpjENomeDoFornecedor": "12.345.678/0001-90/Empresa Exemplo Ltda",
                            "valorDoEmpenho": 1000.0,
                            "valorDaLiquidacao": 800.0,
                            "valorDoPagamento": 500.0,
                            "codigoENomeDoOrgao": "1 - Prefeitura Municipal",
                            "historico": "Aquisição de material de escritório",
                        }
                    ],
                },
            )
        )
        resultado = await client.buscar_empenhos(
            "exemplo", data_inicial="01/01/2024", data_final="31/12/2024"
        )
        assert resultado.total == 1
        assert resultado.pagina == 1
        assert len(resultado.empenhos) == 1
        emp = resultado.empenhos[0]
        assert emp.numero_tcm == 1001
        assert emp.fornecedor == "Empresa Exemplo Ltda"
        assert emp.valor_empenho == 1000.0
        assert emp.orgao == "1 - Prefeitura Municipal"

    @pytest.mark.asyncio
    @respx.mock
    async def test_sends_required_headers_and_params(self) -> None:
        _mock_config()
        route = respx.get(EMPENHO_URL).mock(
            return_value=httpx.Response(200, json={"total": 0, "registros": []})
        )
        await client.buscar_empenhos(
            "exemplo", data_inicial="01/01/2024", data_final="31/12/2024", pagina=2, tamanho=50
        )
        request = route.calls.last.request
        assert request.headers["Authorization"] == f"Bearer {TOKEN}"
        assert request.headers["cliente-integrado"] == "megasoft-portal-da-transparencia"
        assert request.url.params["dataInicial"] == "01/01/2024"
        assert request.url.params["dataFinal"] == "31/12/2024"
        assert request.url.params["pagina"] == "2"
        assert request.url.params["tamanhoDaPagina"] == "50"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        _mock_config()
        respx.get(EMPENHO_URL).mock(
            return_value=httpx.Response(200, json={"total": 0, "registros": []})
        )
        resultado = await client.buscar_empenhos(
            "exemplo", data_inicial="01/01/2024", data_final="31/12/2024"
        )
        assert resultado.empenhos == []
        assert resultado.total == 0


# ---------------------------------------------------------------------------
# buscar_ordens_pagamento
# ---------------------------------------------------------------------------


class TestBuscarOrdensPagamento:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_ordens(self) -> None:
        _mock_config()
        respx.get(ORDEM_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "total": 1,
                    "registros": [
                        {
                            "codigo": 7,
                            "numero": 2024007,
                            "nomeDoOrgao": "2 - Secretaria de Saúde",
                            "dataPagamento": "20/03/2024",
                            "nomeDoFornecedor": "Empresa Exemplo Ltda",
                            "cpfOuCnpjDoFornecedor": "12.345.678/0001-90",
                            "numeroDoEmpenho": 1001,
                            "numeroDaLiquidacao": 2001,
                            "valorPago": 500.0,
                            "valorTotal": 500.0,
                            "fase": "Pagamento",
                        }
                    ],
                },
            )
        )
        resultado = await client.buscar_ordens_pagamento(
            "exemplo", data_inicial="01/01/2024", data_final="31/12/2024"
        )
        assert resultado.total == 1
        ordem = resultado.ordens[0]
        assert ordem.numero == 2024007
        assert ordem.cpf_cnpj_fornecedor == "12.345.678/0001-90"
        assert ordem.valor_pago == 500.0
        assert ordem.fase == "Pagamento"

    @pytest.mark.asyncio
    @respx.mock
    async def test_handles_list_envelope(self) -> None:
        _mock_config()
        respx.get(ORDEM_URL).mock(
            return_value=httpx.Response(200, json=[{"codigo": 1, "valorPago": 10.0}])
        )
        resultado = await client.buscar_ordens_pagamento(
            "exemplo", data_inicial="01/01/2024", data_final="31/12/2024"
        )
        assert resultado.total == 1
        assert resultado.ordens[0].valor_pago == 10.0


# ---------------------------------------------------------------------------
# totais_ordens_pagamento
# ---------------------------------------------------------------------------


class TestTotaisOrdensPagamento:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_totais(self) -> None:
        _mock_config()
        respx.get(TOTAIS_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "valorPago": 100000.0,
                    "valorAnulacao": 2000.0,
                    "valorTotal": 98000.0,
                    "valorRetencao": 5000.0,
                },
            )
        )
        totais = await client.totais_ordens_pagamento(
            "exemplo", data_inicial="01/01/2024", data_final="31/12/2024"
        )
        assert totais.valor_pago == 100000.0
        assert totais.valor_anulacao == 2000.0
        assert totais.valor_total == 98000.0
        assert totais.valor_retencao == 5000.0


# ---------------------------------------------------------------------------
# buscar_receitas
# ---------------------------------------------------------------------------


class TestBuscarReceitas:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_receitas(self) -> None:
        _mock_config()
        respx.get(RECEITA_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "total": 1,
                    "registros": [
                        {
                            "chave": {
                                "codigoDaReceita": 5478,
                                "ano": 2024,
                                "mes": "12",
                                "codigoDoOrgao": 22,
                            },
                            "fonteDaReceita": "1.02.000 - Receitas de Impostos",
                            "receita": "IPTU - Principal",
                            "naturezaDaReceita": "1112.50.0.1",
                            "origemDoRecurso": "Rendas Locais do Municipio",
                            "valorOrcado": 200000.0,
                            "arrecadacao": 150000.0,
                            "valorAcumulado": 150000.0,
                        }
                    ],
                },
            )
        )
        resultado = await client.buscar_receitas(
            "exemplo", data_inicial="01/01/2024", data_final="31/12/2024"
        )
        assert resultado.total == 1
        rec = resultado.receitas[0]
        assert rec.codigo_receita == 5478
        assert rec.ano == 2024
        assert rec.mes == "12"
        assert rec.receita == "IPTU - Principal"
        assert rec.arrecadacao == 150000.0


# ---------------------------------------------------------------------------
# Token, validação e erros
# ---------------------------------------------------------------------------


class TestToken:
    @pytest.mark.asyncio
    @respx.mock
    async def test_env_token_override_skips_config(self, monkeypatch: pytest.MonkeyPatch) -> None:
        monkeypatch.setenv("MEGASOFT_TOKEN", "token-do-ambiente")
        route = respx.get(EMPENHO_URL).mock(
            return_value=httpx.Response(200, json={"total": 0, "registros": []})
        )
        await client.buscar_empenhos("exemplo", data_inicial="01/01/2024", data_final="31/12/2024")
        assert route.calls.last.request.headers["Authorization"] == "Bearer token-do-ambiente"

    @pytest.mark.asyncio
    @respx.mock
    async def test_token_is_cached(self) -> None:
        route_config = respx.get(CONFIG_URL).mock(
            return_value=httpx.Response(200, json={"tokenPortal": TOKEN})
        )
        respx.get(EMPENHO_URL).mock(
            return_value=httpx.Response(200, json={"total": 0, "registros": []})
        )
        await client.buscar_empenhos("exemplo", data_inicial="01/01/2024", data_final="31/12/2024")
        await client.buscar_empenhos("exemplo", data_inicial="01/01/2024", data_final="31/12/2024")
        assert route_config.call_count == 1

    @pytest.mark.asyncio
    @respx.mock
    async def test_missing_token_raises(self) -> None:
        _mock_config(json_body={"tipo": 3})
        with pytest.raises(RuntimeError, match="tokenPortal"):
            await client.buscar_empenhos(
                "exemplo", data_inicial="01/01/2024", data_final="31/12/2024"
            )


class TestValidacao:
    @pytest.mark.asyncio
    async def test_invalid_date_raises(self) -> None:
        with pytest.raises(ValueError, match="DD/MM/AAAA"):
            await client.buscar_empenhos(
                "exemplo", data_inicial="2024-01-01", data_final="31/12/2024"
            )

    @pytest.mark.asyncio
    async def test_invalid_page_raises(self) -> None:
        with pytest.raises(ValueError, match="pagina"):
            await client.buscar_empenhos(
                "exemplo", data_inicial="01/01/2024", data_final="31/12/2024", pagina=0
            )

    @pytest.mark.asyncio
    async def test_empty_municipio_raises(self) -> None:
        with pytest.raises(ValueError, match="município"):
            await client.buscar_empenhos("  ", data_inicial="01/01/2024", data_final="31/12/2024")

    @pytest.mark.asyncio
    @respx.mock
    async def test_full_url_is_accepted(self) -> None:
        _mock_config()
        respx.get(f"{PORTAL}/api/{EMPENHO_PATH}").mock(
            return_value=httpx.Response(200, json={"total": 0, "registros": []})
        )
        resultado = await client.buscar_empenhos(
            PORTAL, data_inicial="01/01/2024", data_final="31/12/2024"
        )
        assert resultado.total == 0


class TestPortalBase:
    """The portal host must always stay under *.megasofttransparencia.com.br."""

    def test_bare_slug_builds_https_subdomain(self) -> None:
        assert (
            client._portal_base("serranopolis")
            == "https://serranopolis.megasofttransparencia.com.br"
        )

    def test_full_domain_and_url_accepted(self) -> None:
        esperado = "https://serranopolis.megasofttransparencia.com.br"
        assert client._portal_base("serranopolis.megasofttransparencia.com.br") == esperado
        assert client._portal_base(esperado) == esperado

    @pytest.mark.parametrize(
        "entrada",
        [
            "http://serranopolis.megasofttransparencia.com.br",
            "ftp://serranopolis.megasofttransparencia.com.br",
            "http://127.0.0.1:8765",
            "127.0.0.1",
            "https://evil.com",
            "evil.com",
            "https://serranopolis.evil.com",
            "https://serranopolis.megasofttransparencia.com.br:8443",
            "serranopolis.megasofttransparencia.com.br:8443",
            "https://megasofttransparencia.com.br",
            "https://a.b.megasofttransparencia.com.br",
            "https://serranopolis.megasofttransparencia.com.br/caminho",
            "https://serranopolis.megasofttransparencia.com.br?x=1",
            "https://serranopolis.megasofttransparencia.com.br#frag",
        ],
    )
    def test_rejects_hosts_outside_allowed_domain(self, entrada: str) -> None:
        with pytest.raises(ValueError):
            client._portal_base(entrada)

    @respx.mock
    def test_rejected_host_makes_no_request(self) -> None:
        route = respx.route().mock(return_value=httpx.Response(200, json={}))
        with pytest.raises(ValueError):
            client._portal_base("https://evil.com")
        assert not route.called


class TestErros:
    @pytest.mark.asyncio
    @respx.mock
    async def test_http_error_raises(self) -> None:
        _mock_config()
        respx.get(EMPENHO_URL).mock(return_value=httpx.Response(403, json={"erro": "negado"}))
        with pytest.raises(HttpClientError):
            await client.buscar_empenhos(
                "exemplo", data_inicial="01/01/2024", data_final="31/12/2024"
            )

    @pytest.mark.asyncio
    @respx.mock
    async def test_unexpected_200_envelope_raises(self) -> None:
        _mock_config()
        respx.get(EMPENHO_URL).mock(
            return_value=httpx.Response(200, json={"erro": "falha interna"})
        )
        with pytest.raises(client.RespostaInesperadaError):
            await client.buscar_empenhos(
                "exemplo", data_inicial="01/01/2024", data_final="31/12/2024"
            )

    @pytest.mark.asyncio
    @respx.mock
    async def test_scalar_200_envelope_raises(self) -> None:
        _mock_config()
        respx.get(EMPENHO_URL).mock(return_value=httpx.Response(200, json="erro"))
        with pytest.raises(client.RespostaInesperadaError):
            await client.buscar_empenhos(
                "exemplo", data_inicial="01/01/2024", data_final="31/12/2024"
            )

    @pytest.mark.asyncio
    @respx.mock
    async def test_malformed_json_raises(self) -> None:
        _mock_config()
        respx.get(EMPENHO_URL).mock(
            return_value=httpx.Response(
                200, content=b"<html>nao e json</html>", headers={"content-type": "text/html"}
            )
        )
        with pytest.raises(client.RespostaInesperadaError):
            await client.buscar_empenhos(
                "exemplo", data_inicial="01/01/2024", data_final="31/12/2024"
            )

    @pytest.mark.asyncio
    @respx.mock
    async def test_totais_unexpected_200_raises(self) -> None:
        _mock_config()
        respx.get(TOTAIS_URL).mock(return_value=httpx.Response(200, json={"erro": "falha"}))
        with pytest.raises(client.RespostaInesperadaError):
            await client.totais_ordens_pagamento(
                "exemplo", data_inicial="01/01/2024", data_final="31/12/2024"
            )

    @pytest.mark.asyncio
    @respx.mock
    async def test_totais_non_dict_200_raises(self) -> None:
        _mock_config()
        respx.get(TOTAIS_URL).mock(return_value=httpx.Response(200, json=[1, 2, 3]))
        with pytest.raises(client.RespostaInesperadaError):
            await client.totais_ordens_pagamento(
                "exemplo", data_inicial="01/01/2024", data_final="31/12/2024"
            )

    def test_resposta_inesperada_is_http_client_error(self) -> None:
        assert issubclass(client.RespostaInesperadaError, HttpClientError)
