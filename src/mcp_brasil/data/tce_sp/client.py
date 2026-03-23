"""HTTP client for the TCE-SP API.

Endpoints:
    - /json/municipios                           → listar_municipios
    - /json/despesas/{municipio}/{exercicio}/{mes} → buscar_despesas
    - /json/receitas/{municipio}/{exercicio}/{mes} → buscar_receitas
"""

from __future__ import annotations

from typing import Any

from mcp_brasil._shared.http_client import http_get

from .constants import DESPESAS_URL, MUNICIPIOS_URL, RECEITAS_URL
from .schemas import Despesa, Municipio, Receita


def _parse_brl_string(value: str | None) -> float | None:
    """Parse Brazilian number format ('1.234,56') to float."""
    if not value:
        return None
    try:
        return float(value.replace(".", "").replace(",", "."))
    except (ValueError, AttributeError):
        return None


async def listar_municipios() -> list[Municipio]:
    """Lista todos os 645 municípios paulistas."""
    data: list[dict[str, Any]] = await http_get(MUNICIPIOS_URL)
    return [
        Municipio(
            municipio=item.get("municipio", ""),
            municipio_extenso=item.get("municipio_extenso", ""),
        )
        for item in data
    ]


async def buscar_despesas(
    municipio: str,
    exercicio: int,
    mes: int,
) -> list[Despesa]:
    """Busca despesas de um município paulista por mês."""
    url = f"{DESPESAS_URL}/{municipio}/{exercicio}/{mes}"
    data: list[dict[str, Any]] = await http_get(url)
    return [
        Despesa(
            orgao=item.get("orgao"),
            mes=item.get("mes"),
            evento=item.get("evento"),
            nr_empenho=item.get("nr_empenho"),
            id_fornecedor=item.get("id_fornecedor"),
            nm_fornecedor=item.get("nm_fornecedor"),
            dt_emissao_despesa=item.get("dt_emissao_despesa"),
            vl_despesa=_parse_brl_string(item.get("vl_despesa")),
        )
        for item in data
    ]


async def buscar_receitas(
    municipio: str,
    exercicio: int,
    mes: int,
) -> list[Receita]:
    """Busca receitas de um município paulista por mês."""
    url = f"{RECEITAS_URL}/{municipio}/{exercicio}/{mes}"
    data: list[dict[str, Any]] = await http_get(url)
    return [
        Receita(
            orgao=item.get("orgao"),
            mes=item.get("mes"),
            ds_fonte_recurso=item.get("ds_fonte_recurso"),
            ds_cd_aplicacao_fixo=item.get("ds_cd_aplicacao_fixo"),
            ds_alinea=item.get("ds_alinea"),
            ds_subalinea=item.get("ds_subalinea"),
            vl_arrecadacao=_parse_brl_string(item.get("vl_arrecadacao")),
        )
        for item in data
    ]
