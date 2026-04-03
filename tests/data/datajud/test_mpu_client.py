"""Tests for the DataJud MPU HTTP client functions."""

import httpx
import pytest
import respx

from mcp_brasil.data.datajud import client
from mcp_brasil.data.datajud.constants import (
    DATAJUD_API_BASE,
    MPU_CLASSE_CRIMINAL,
    MPU_MOV_CONCEDIDA,
    MPU_MOV_PRORROGADA,
)

TJSP_URL = f"{DATAJUD_API_BASE}tjsp/_search"


def _es_response(hits: list | None = None, total: int | None = None, aggs: dict | None = None):
    """Helper to build Elasticsearch responses."""
    data: dict = {
        "hits": {
            "hits": hits or [],
        },
    }
    if total is not None:
        data["hits"]["total"] = {"value": total, "relation": "eq"}
    if aggs:
        data["aggregations"] = aggs
    return httpx.Response(200, json=data)


def _mpu_hit(
    numero: str = "0001234",
    classe_codigo: int = MPU_CLASSE_CRIMINAL,
    classe_nome: str = "MPU (Maria da Penha) — Criminal",
    tribunal: str = "TJSP",
    orgao: str = "1ª Vara Criminal",
    data_ajuizamento: str = "2024-06-15",
    movimentos: list | None = None,
) -> dict:
    """Build a fake Elasticsearch hit for MPU."""
    return {
        "_source": {
            "numeroProcesso": numero,
            "classe": {"codigo": classe_codigo, "nome": classe_nome},
            "assuntos": [{"codigo": 11529, "nome": "Violência Doméstica"}],
            "tribunal": tribunal,
            "orgaoJulgador": {"nome": orgao},
            "dataAjuizamento": data_ajuizamento,
            "movimentos": movimentos or [],
        }
    }


class TestBuscarMedidasProtetivas:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_parsed_mpus(self) -> None:
        respx.post(TJSP_URL).mock(return_value=_es_response([_mpu_hit()]))
        result = await client.buscar_medidas_protetivas(tribunal="tjsp")
        assert len(result) == 1
        assert result[0].numero == "0001234"
        assert result[0].classe_codigo == MPU_CLASSE_CRIMINAL

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty(self) -> None:
        respx.post(TJSP_URL).mock(return_value=_es_response([]))
        result = await client.buscar_medidas_protetivas(tribunal="tjsp")
        assert result == []

    @pytest.mark.asyncio
    @respx.mock
    async def test_filters_by_lei(self) -> None:
        route = respx.post(TJSP_URL).mock(return_value=_es_response([]))
        await client.buscar_medidas_protetivas(tribunal="tjsp", lei="maria_penha")
        assert route.called
        body = route.calls[0].request.content
        assert b"1268" in body  # MPU_CLASSE_CRIMINAL


class TestBuscarMpuConcedidas:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_concedidas(self) -> None:
        hit = _mpu_hit(
            movimentos=[
                {
                    "codigo": MPU_MOV_CONCEDIDA,
                    "dataHora": "2024-06-16",
                    "nome": "Concedida medida protetiva",
                    "complementosTabelados": [],
                }
            ]
        )
        respx.post(TJSP_URL).mock(return_value=_es_response([hit]))
        result = await client.buscar_mpu_concedidas(tribunal="tjsp")
        assert len(result) == 1
        assert result[0].movimentos is not None
        assert len(result[0].movimentos) == 1
        assert result[0].movimentos[0].codigo == MPU_MOV_CONCEDIDA

    @pytest.mark.asyncio
    @respx.mock
    async def test_filters_by_destinatario(self) -> None:
        route = respx.post(TJSP_URL).mock(return_value=_es_response([]))
        await client.buscar_mpu_concedidas(tribunal="tjsp", destinatario="mulher")
        assert route.called
        body = route.calls[0].request.content
        assert b"124" in body  # MPU_DEST_MULHER


class TestBuscarMpuPorTipo:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_by_tipo(self) -> None:
        respx.post(TJSP_URL).mock(return_value=_es_response([_mpu_hit()]))
        result = await client.buscar_mpu_por_tipo(
            tribunal="tjsp",
            tipo_medida="afastamento_lar",
        )
        assert len(result) == 1

    @pytest.mark.asyncio
    async def test_invalid_tipo_raises(self) -> None:
        with pytest.raises(ValueError, match="não reconhecido"):
            await client.buscar_mpu_por_tipo(tipo_medida="tipo_invalido")


class TestEstatisticasMpu:
    @pytest.mark.asyncio
    @respx.mock
    async def test_returns_stats(self) -> None:
        aggs = {
            "por_classe": {
                "buckets": [
                    {"key": MPU_CLASSE_CRIMINAL, "doc_count": 100},
                ],
            },
            "por_movimento": {
                "mpu_movimentos": {
                    "por_tipo": {
                        "buckets": [
                            {"key": MPU_MOV_CONCEDIDA, "doc_count": 80},
                        ],
                    },
                },
            },
        }
        respx.post(TJSP_URL).mock(return_value=_es_response(total=100, aggs=aggs))
        result = await client.estatisticas_mpu(tribunal="tjsp", ano=2024)
        assert result.total == 100
        assert result.tribunal == "tjsp"
        assert result.periodo == "2024"
        assert result.por_classe is not None
        assert result.por_decisao is not None

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty_stats(self) -> None:
        respx.post(TJSP_URL).mock(return_value=_es_response(total=0))
        result = await client.estatisticas_mpu(tribunal="tjsp", ano=2024)
        assert result.total == 0


class TestTimelineMpu:
    @pytest.mark.asyncio
    @respx.mock
    async def test_filters_mpu_movements(self) -> None:
        hit = {
            "_source": {
                "numeroProcesso": "0001234",
                "classe": {"nome": "MPU Criminal"},
                "assuntos": [],
                "tribunal": "TJSP",
                "orgaoJulgador": {"nome": "1ª Vara"},
                "dataAjuizamento": "2024-06-15",
                "movimentos": [
                    {"codigo": MPU_MOV_CONCEDIDA, "dataHora": "2024-06-16", "nome": "Concedida"},
                    {"codigo": 26, "dataHora": "2024-06-15", "nome": "Distribuição"},
                    {"codigo": MPU_MOV_PRORROGADA, "dataHora": "2024-09-01", "nome": "Prorrogada"},
                ],
                "poloAtivo": [],
                "poloPassivo": [],
            }
        }
        respx.post(TJSP_URL).mock(return_value=_es_response([hit]))
        result = await client.timeline_mpu("0001234", "tjsp")
        assert len(result) == 2
        assert result[0].codigo == MPU_MOV_CONCEDIDA
        assert result[1].codigo == MPU_MOV_PRORROGADA

    @pytest.mark.asyncio
    @respx.mock
    async def test_empty(self) -> None:
        respx.post(TJSP_URL).mock(return_value=_es_response([]))
        result = await client.timeline_mpu("9999999", "tjsp")
        assert result == []


class TestBuildMpuQuery:
    def test_basic_query(self) -> None:
        query = client._build_mpu_query(classes=[1268, 12423])
        assert query["query"]["bool"]["must"][0]["terms"]["classe.codigo"] == [1268, 12423]
        assert query["size"] == 10

    def test_with_dates(self) -> None:
        query = client._build_mpu_query(
            classes=[1268],
            data_inicio="2024-01-01",
            data_fim="2024-12-31",
        )
        must = query["query"]["bool"]["must"]
        assert len(must) == 2
        assert "range" in must[1]

    def test_with_movimentos_and_destinatario(self) -> None:
        query = client._build_mpu_query(
            classes=[1268],
            movimentos=[15486],
            destinatario=124,
        )
        must = query["query"]["bool"]["must"]
        nested = must[1]
        assert "nested" in nested
        assert nested["nested"]["path"] == "movimentos"


class TestMpuClassesForLei:
    def test_maria_penha(self) -> None:
        assert client._mpu_classes_for_lei("maria_penha") == [1268, 12423, 15309]

    def test_henry_borel(self) -> None:
        assert client._mpu_classes_for_lei("henry_borel") == [15170, 15171]

    def test_ambas(self) -> None:
        result = client._mpu_classes_for_lei("ambas")
        assert 1268 in result
        assert 15170 in result


class TestParseMpuProcesso:
    def test_parses_basic_hit(self) -> None:
        hit = _mpu_hit()
        result = client._parse_mpu_processo(hit)
        assert result.numero == "0001234"
        assert result.classe_codigo == MPU_CLASSE_CRIMINAL
        assert result.classe_nome == "MPU (Maria da Penha) — Criminal"

    def test_filters_only_mpu_movimentos(self) -> None:
        hit = _mpu_hit(
            movimentos=[
                {"codigo": MPU_MOV_CONCEDIDA, "dataHora": "2024-06-16", "nome": "Concedida"},
                {"codigo": 26, "dataHora": "2024-06-15", "nome": "Distribuição"},
            ]
        )
        result = client._parse_mpu_processo(hit)
        assert result.movimentos is not None
        assert len(result.movimentos) == 1
        assert result.movimentos[0].codigo == MPU_MOV_CONCEDIDA

    def test_empty_source(self) -> None:
        result = client._parse_mpu_processo({"_source": {}})
        assert result.numero is None
        assert result.movimentos is None
