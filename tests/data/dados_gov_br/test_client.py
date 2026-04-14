"""Tests for the dados_gov_br HTTP client."""

from unittest.mock import patch

import httpx
import pytest
import respx

from mcp_brasil.data.dados_gov_br import client
from mcp_brasil.data.dados_gov_br.constants import (
    CONJUNTOS_URL,
    ORGANIZACAO_URL,
    REUSO_URL,
    REUSOS_URL,
    TAGS_URL,
    TEMAS_URL,
)

# All tests mock the API key to avoid AuthError
API_KEY_PATCH = patch.dict("os.environ", {"DADOS_GOV_BR_API_KEY": "test-key"})


def setup_module() -> None:
    API_KEY_PATCH.start()


def teardown_module() -> None:
    API_KEY_PATCH.stop()


# ---------------------------------------------------------------------------
# buscar_conjuntos
# ---------------------------------------------------------------------------


class TestBuscarConjuntos:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_conjuntos_from_wrapper(self) -> None:
        respx.get(CONJUNTOS_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "totalRegistros": 1,
                    "registros": [
                        {
                            "id": "abc-123",
                            "title": "Dados de Saúde",
                            "organizationName": "Ministério da Saúde",
                            "temas": "Saúde",
                            "dataCriacao": "2023-01-15",
                        }
                    ],
                },
            )
        )
        result = await client.buscar_conjuntos(pagina=1)
        assert result.total == 1
        assert len(result.conjuntos) == 1
        c = result.conjuntos[0]
        assert c.id == "abc-123"
        assert c.title == "Dados de Saúde"
        assert c.organizationName == "Ministério da Saúde"

    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_conjuntos_from_list(self) -> None:
        respx.get(CONJUNTOS_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "id": "abc-123",
                        "title": "Dados de Saúde",
                        "organizationName": "Ministério da Saúde",
                    }
                ],
            )
        )
        result = await client.buscar_conjuntos(pagina=1)
        assert result.total == 1
        assert len(result.conjuntos) == 1

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(CONJUNTOS_URL).mock(
            return_value=httpx.Response(
                200,
                json={"totalRegistros": 0, "registros": []},
            )
        )
        result = await client.buscar_conjuntos(pagina=1)
        assert result.total == 0
        assert result.conjuntos == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_with_filters(self) -> None:
        route = respx.get(CONJUNTOS_URL).mock(return_value=httpx.Response(200, json=[]))
        await client.buscar_conjuntos(
            pagina=1,
            nome="educação",
            id_organizacao="org-1",
            dados_abertos=True,
        )
        assert route.called
        request = route.calls[0].request
        assert b"nomeConjuntoDados" in request.url.query
        assert b"idOrganizacao" in request.url.query


# ---------------------------------------------------------------------------
# detalhar_conjunto
# ---------------------------------------------------------------------------


class TestDetalharConjunto:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_detail(self) -> None:
        respx.get(f"{CONJUNTOS_URL}/abc-123").mock(
            return_value=httpx.Response(
                200,
                json={
                    "id": "abc-123",
                    "titulo": "Dados de Educação",
                    "descricao": "Indicadores educacionais",
                    "organizacao": "MEC",
                    "licenca": "cc-by",
                    "periodicidade": "ANUAL",
                    "temas": [{"name": "educacao", "title": "Educação"}],
                    "tags": [{"id": "t1", "name": "inep"}],
                    "recursos": [
                        {
                            "id": "r1",
                            "titulo": "CSV",
                            "formato": "CSV",
                            "link": "https://download.csv",
                        }
                    ],
                },
            )
        )
        result = await client.detalhar_conjunto("abc-123")
        assert result is not None
        assert result.id == "abc-123"
        assert result.titulo == "Dados de Educação"
        assert result.organizacao == "MEC"
        assert len(result.temas) == 1
        assert len(result.tags) == 1
        assert len(result.recursos) == 1
        assert result.recursos[0].formato == "CSV"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response_returns_none(self) -> None:
        respx.get(f"{CONJUNTOS_URL}/nao-existe").mock(return_value=httpx.Response(200, json={}))
        result = await client.detalhar_conjunto("nao-existe")
        assert result is None


# ---------------------------------------------------------------------------
# listar_tags_conjunto
# ---------------------------------------------------------------------------


class TestListarTagsConjunto:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_tags(self) -> None:
        respx.get(f"{CONJUNTOS_URL}/abc-123/tag").mock(
            return_value=httpx.Response(
                200,
                json=[
                    {"id": "t1", "name": "saude", "display_name": "Saúde"},
                    {"id": "t2", "name": "sus", "display_name": "SUS"},
                ],
            )
        )
        result = await client.listar_tags_conjunto("abc-123")
        assert len(result) == 2
        assert result[0].name == "saude"
        assert result[1].display_name == "SUS"


# ---------------------------------------------------------------------------
# listar_formatos
# ---------------------------------------------------------------------------


class TestListarFormatos:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_formats(self) -> None:
        respx.get(f"{CONJUNTOS_URL}/formatos").mock(
            return_value=httpx.Response(200, json=["CSV", "JSON", "XML"])
        )
        result = await client.listar_formatos()
        assert result == ["CSV", "JSON", "XML"]


# ---------------------------------------------------------------------------
# listar_organizacoes
# ---------------------------------------------------------------------------


class TestListarOrganizacoes:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_organizacoes(self) -> None:
        respx.get(ORGANIZACAO_URL).mock(
            return_value=httpx.Response(
                200,
                json={
                    "totalRegistros": 2,
                    "registros": [
                        {
                            "id": "org-1",
                            "titulo": "Ministério da Saúde",
                            "nome": "ministerio-da-saude",
                            "qtdConjuntoDeDados": "150",
                        },
                        {
                            "id": "org-2",
                            "titulo": "IBGE",
                            "nome": "ibge",
                            "qtdConjuntoDeDados": "80",
                        },
                    ],
                },
            )
        )
        result = await client.listar_organizacoes()
        assert result.total == 2
        assert len(result.organizacoes) == 2
        assert result.organizacoes[0].titulo == "Ministério da Saúde"
        assert result.organizacoes[1].titulo == "IBGE"

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_response(self) -> None:
        respx.get(ORGANIZACAO_URL).mock(
            return_value=httpx.Response(
                200,
                json={"totalRegistros": 0, "registros": []},
            )
        )
        result = await client.listar_organizacoes()
        assert result.total == 0
        assert result.organizacoes == []


# ---------------------------------------------------------------------------
# detalhar_organizacao
# ---------------------------------------------------------------------------


class TestDetalharOrganizacao:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_detail(self) -> None:
        respx.get(f"{ORGANIZACAO_URL}/org-1").mock(
            return_value=httpx.Response(
                200,
                json={
                    "id": "org-1",
                    "name": "ministerio-da-saude",
                    "displayName": "Ministério da Saúde",
                    "descricao": "Saúde pública",
                    "quantidadeConjuntoDados": 150,
                    "quantidadeSeguidores": 500,
                },
            )
        )
        result = await client.detalhar_organizacao("org-1")
        assert result is not None
        assert result.displayName == "Ministério da Saúde"
        assert result.quantidadeConjuntoDados == 150


# ---------------------------------------------------------------------------
# listar_temas
# ---------------------------------------------------------------------------


class TestListarTemas:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_temas(self) -> None:
        respx.get(TEMAS_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "id": "t1",
                        "name": "saude",
                        "title": "Saúde",
                        "packageCount": 200,
                    },
                    {
                        "id": "t2",
                        "name": "educacao",
                        "title": "Educação",
                        "packageCount": 150,
                    },
                ],
            )
        )
        result = await client.listar_temas()
        assert len(result) == 2
        assert result[0].title == "Saúde"
        assert result[0].packageCount == 200


# ---------------------------------------------------------------------------
# buscar_tags
# ---------------------------------------------------------------------------


class TestBuscarTags:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_tags(self) -> None:
        respx.get(TAGS_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {"id": "tag-1", "name": "saude", "display_name": "Saúde"},
                ],
            )
        )
        result = await client.buscar_tags("saude")
        assert len(result) == 1
        assert result[0].name == "saude"
        assert result[0].display_name == "Saúde"


# ---------------------------------------------------------------------------
# listar_ods
# ---------------------------------------------------------------------------


class TestListarOds:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_ods(self) -> None:
        respx.get(f"{CONJUNTOS_URL}/objetivos-desenvolvimento-sustentavel").mock(
            return_value=httpx.Response(
                200,
                json=[
                    {"id": 1, "descricao": "Erradicação da Pobreza"},
                    {"id": 3, "descricao": "Saúde e Bem-Estar"},
                ],
            )
        )
        result = await client.listar_ods()
        assert len(result) == 2
        assert result[0].descricao == "Erradicação da Pobreza"


# ---------------------------------------------------------------------------
# listar_observancia_legal
# ---------------------------------------------------------------------------


class TestListarObservanciaLegal:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed(self) -> None:
        respx.get(f"{CONJUNTOS_URL}/observancia-legal").mock(
            return_value=httpx.Response(
                200,
                json=[
                    {"id": 1, "descricao": "Lei de Acesso à Informação"},
                ],
            )
        )
        result = await client.listar_observancia_legal()
        assert len(result) == 1
        assert result[0].descricao == "Lei de Acesso à Informação"


# ---------------------------------------------------------------------------
# listar_reusos
# ---------------------------------------------------------------------------


class TestListarReusos:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_reusos(self) -> None:
        respx.get(REUSOS_URL).mock(
            return_value=httpx.Response(
                200,
                json=[
                    {
                        "id": "r1",
                        "nome": "Dashboard COVID",
                        "autor": "João",
                        "organizacao": "Fiocruz",
                        "situacao": "HOMOLOGADO",
                    }
                ],
            )
        )
        result = await client.listar_reusos()
        assert len(result) == 1
        assert result[0].nome == "Dashboard COVID"


# ---------------------------------------------------------------------------
# detalhar_reuso
# ---------------------------------------------------------------------------


class TestDetalharReuso:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_detail(self) -> None:
        respx.get(f"{REUSO_URL}/r1").mock(
            return_value=httpx.Response(
                200,
                json={
                    "id": "r1",
                    "nome": "Dashboard COVID",
                    "descricao": "Painel de monitoramento",
                    "url": "https://example.com",
                    "autor": "João",
                    "temas": ["Saúde"],
                    "tipos": ["Dashboard"],
                },
            )
        )
        result = await client.detalhar_reuso("r1")
        assert result is not None
        assert result.nome == "Dashboard COVID"
        assert result.temas == ["Saúde"]

    @pytest.mark.asyncio
    @respx.mock
    async def test_not_found(self) -> None:
        respx.get(f"{REUSO_URL}/nao-existe").mock(return_value=httpx.Response(200, json={}))
        result = await client.detalhar_reuso("nao-existe")
        assert result is None


# ---------------------------------------------------------------------------
# Auth
# ---------------------------------------------------------------------------


class TestAuth:
    @pytest.mark.asyncio
    async def test_missing_api_key_raises_auth_error(self) -> None:
        from mcp_brasil.exceptions import AuthError

        with (
            patch.dict("os.environ", {"DADOS_GOV_BR_API_KEY": ""}),
            pytest.raises(AuthError, match="DADOS_GOV_BR_API_KEY"),
        ):
            client._get_api_key()
