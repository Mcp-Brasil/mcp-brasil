"""Pydantic schemas for the ComexStat feature."""

from __future__ import annotations

from pydantic import BaseModel


class ComexItem(BaseModel):
    """Single trade flow record from ComexStat."""

    periodo: str
    pais: str | None = None
    ncm: str | None = None
    fob_usd: float
    kg_liquido: float


class BalancaItem(BaseModel):
    """Trade balance record for a period."""

    periodo: str
    exportacoes_fob: float
    importacoes_fob: float
    saldo_fob: float


class PaisItem(BaseModel):
    """Country entry from ComexStat filters."""

    id: str
    nome: str
