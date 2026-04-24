"""HTTP client for IPEADATA OData v4 API."""

from __future__ import annotations

from typing import Any

from mcp_brasil._shared.http_client import http_get

from .constants import IPEADATA_API_BASE
from .schemas import SerieMetadado, SerieValor


async def _get_odata(path: str, params: dict[str, str] | None = None) -> list[dict[str, Any]]:
    """Low-level GET on an OData endpoint — returns the ``value`` list."""
    url = f"{IPEADATA_API_BASE}{path}"
    data = await http_get(url, params=params or {})
    if isinstance(data, dict):
        vals = data.get("value")
        if isinstance(vals, list):
            return [v for v in vals if isinstance(v, dict)]
    return []


async def listar_series(
    *,
    filtro_nome: str | None = None,
    tema_codigo: int | None = None,
    top: int = 100,
) -> list[SerieMetadado]:
    """GET /Metadados — lista metadados de séries.

    Args:
        filtro_nome: Substring a buscar no SERNOME (OData ``contains``).
        tema_codigo: Filtra por TEMCODIGO.
        top: Limite de resultados (padrão 100).
    """
    params: dict[str, str] = {"$top": str(top), "$orderby": "SERNOME"}
    filters: list[str] = []
    if filtro_nome:
        # IPEADATA OData server only supports startswith — contains/substringof
        # are rejected with HTTP 400.
        esc = filtro_nome.replace("'", "''")
        filters.append(f"startswith(SERNOME, '{esc}')")
    if tema_codigo is not None:
        filters.append(f"TEMCODIGO eq {tema_codigo}")
    if filters:
        params["$filter"] = " and ".join(filters)
    rows = await _get_odata("/Metadados", params)
    return [SerieMetadado.model_validate(r) for r in rows]


async def metadados_serie(codigo: str) -> SerieMetadado | None:
    """GET metadados de uma série específica."""
    esc = codigo.replace("'", "''")
    rows = await _get_odata("/Metadados", {"$filter": f"SERCODIGO eq '{esc}'", "$top": "1"})
    if not rows:
        return None
    return SerieMetadado.model_validate(rows[0])


async def valores_serie(codigo: str, *, top: int = 500) -> list[SerieValor]:
    """GET /ValoresSerie(SERCODIGO='...') — valores históricos, mais recentes primeiro.

    Nota: IPEADATA ignora ``$orderby`` em ValoresSerie e sempre retorna asc.
    Buscamos todos os registros (sem top) e cortamos os últimos ``top`` no
    lado cliente — caro mas correto. Para séries pesadas, use ``ultimo_valor``.

    Args:
        codigo: SERCODIGO.
        top: Número de pontos mais recentes (padrão 500, máx 5000).
    """
    esc = codigo.replace("'", "''")
    path = f"/ValoresSerie(SERCODIGO='{esc}')"
    rows = await _get_odata(path, {})
    # Rows chegam em ordem asc. Pega os últimos `top` e inverte.
    top = max(1, min(top, 5000))
    tail = rows[-top:] if len(rows) > top else rows
    return [SerieValor.model_validate(r) for r in reversed(tail)]


async def listar_temas() -> list[dict[str, Any]]:
    """GET /Temas — lista de temas (categorização)."""
    return await _get_odata("/Temas", {"$top": "100"})
