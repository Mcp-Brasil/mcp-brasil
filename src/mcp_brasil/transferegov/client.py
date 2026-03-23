"""HTTP client for the TransfereGov API (PostgREST).

PostgREST API — no auth, filters via query params (column=operator.value).
Pagination via limit/offset query params.

Endpoints:
    - /transferenciasespeciais → buscar_emendas_pix
"""

from __future__ import annotations

from typing import Any

from mcp_brasil._shared.http_client import http_get

from .constants import DEFAULT_PAGE_SIZE, TRANSFERENCIAS_ESPECIAIS_URL
from .schemas import TransferenciaEspecial


def _build_query(
    filters: dict[str, str],
    limit: int = DEFAULT_PAGE_SIZE,
    offset: int = 0,
    order: str | None = None,
) -> dict[str, str]:
    """Build PostgREST query params from a dict of column=operator.value filters."""
    params: dict[str, str] = dict(filters)
    params["limit"] = str(limit)
    params["offset"] = str(offset)
    if order:
        params["order"] = order
    return params


def _parse_transferencia(raw: dict[str, Any]) -> TransferenciaEspecial:
    """Parse a raw TransfereGov JSON into a TransferenciaEspecial model."""
    return TransferenciaEspecial(
        id_transferencia_especial=raw.get("id_transferencia_especial"),
        ano_exercicio=raw.get("ano_exercicio"),
        nr_emenda=raw.get("nr_emenda"),
        autor_emenda=raw.get("autor_emenda"),
        tipo_emenda=raw.get("tipo_emenda"),
        funcao=raw.get("funcao"),
        subfuncao=raw.get("subfuncao"),
        valor_empenhado=raw.get("valor_empenhado"),
        valor_liquidado=raw.get("valor_liquidado"),
        valor_pago=raw.get("valor_pago"),
        nm_municipio_beneficiario=raw.get("nm_municipio_beneficiario"),
        uf_beneficiario=raw.get("uf_beneficiario"),
        nm_entidade_beneficiaria=raw.get("nm_entidade_beneficiaria"),
        objeto=raw.get("objeto"),
    )


async def _get(filters: dict[str, str], pagina: int = 1) -> list[TransferenciaEspecial]:
    """Make a GET to the TransfereGov API with PostgREST filters."""
    offset = (pagina - 1) * DEFAULT_PAGE_SIZE
    params = _build_query(filters, limit=DEFAULT_PAGE_SIZE, offset=offset)
    data = await http_get(TRANSFERENCIAS_ESPECIAIS_URL, params=params)
    if isinstance(data, list):
        return [_parse_transferencia(item) for item in data]
    return []


async def buscar_emendas_pix(
    ano: int | None = None,
    uf: str | None = None,
    pagina: int = 1,
) -> list[TransferenciaEspecial]:
    """Lista transferências especiais (emendas pix).

    Args:
        ano: Ano de exercício.
        uf: UF do beneficiário.
        pagina: Número da página.
    """
    filters: dict[str, str] = {}
    if ano:
        filters["ano_exercicio"] = f"eq.{ano}"
    if uf:
        filters["uf_beneficiario"] = f"eq.{uf.upper()}"
    return await _get(filters, pagina)


async def buscar_emenda_por_autor(
    nome_autor: str,
    ano: int | None = None,
    pagina: int = 1,
) -> list[TransferenciaEspecial]:
    """Busca emendas pix por nome do parlamentar.

    Args:
        nome_autor: Nome (ou parte do nome) do autor da emenda.
        ano: Ano de exercício (opcional).
        pagina: Número da página.
    """
    filters: dict[str, str] = {"autor_emenda": f"ilike.*{nome_autor}*"}
    if ano:
        filters["ano_exercicio"] = f"eq.{ano}"
    return await _get(filters, pagina)


async def detalhe_emenda(id_transferencia: int) -> TransferenciaEspecial | None:
    """Busca detalhe de uma emenda pix específica.

    Args:
        id_transferencia: ID da transferência especial.
    """
    filters: dict[str, str] = {"id_transferencia_especial": f"eq.{id_transferencia}"}
    params = _build_query(filters, limit=1, offset=0)
    data = await http_get(TRANSFERENCIAS_ESPECIAIS_URL, params=params)
    if isinstance(data, list) and len(data) > 0:
        return _parse_transferencia(data[0])
    return None


async def emendas_por_municipio(
    nome_municipio: str,
    ano: int | None = None,
    pagina: int = 1,
) -> list[TransferenciaEspecial]:
    """Busca emendas pix destinadas a um município.

    Args:
        nome_municipio: Nome (ou parte do nome) do município beneficiário.
        ano: Ano de exercício (opcional).
        pagina: Número da página.
    """
    filters: dict[str, str] = {"nm_municipio_beneficiario": f"ilike.*{nome_municipio}*"}
    if ano:
        filters["ano_exercicio"] = f"eq.{ano}"
    return await _get(filters, pagina)


async def resumo_emendas_ano(
    ano: int,
    pagina: int = 1,
) -> list[TransferenciaEspecial]:
    """Lista emendas pix de um ano para visão geral.

    Args:
        ano: Ano de exercício.
        pagina: Número da página.
    """
    filters: dict[str, str] = {"ano_exercicio": f"eq.{ano}"}
    return await _get(filters, pagina)
