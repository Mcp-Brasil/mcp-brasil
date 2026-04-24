"""Pydantic schemas for SICONFI API responses.

Zero business logic (ADR-001 rule #4).
"""

from __future__ import annotations

from pydantic import BaseModel, ConfigDict, Field


class Ente(BaseModel):
    """Federation entity declarant (município, estado, União, DF)."""

    model_config = ConfigDict(populate_by_name=True, extra="allow")

    cod_ibge: int = Field(alias="cod_ibge")
    ente: str
    capital: int = 0
    regiao: str = ""
    uf: str = ""
    esfera: str = ""
    exercicio: int | None = None
    populacao: int | None = None
    cnpj: str | None = None


class ItemDeclaracao(BaseModel):
    """Generic line item from RREO, RGF, DCA, or MSC response."""

    model_config = ConfigDict(populate_by_name=True, extra="allow")

    exercicio: int | None = None
    periodo: int | None = None
    periodicidade: str | None = None
    instituicao: str | None = None
    cod_ibge: int | None = None
    populacao: int | None = None
    uf: str | None = None
    rotulo: str | None = None
    conta: str | None = None
    cod_conta: str | None = None
    coluna: str | None = None
    anexo: str | None = None
    esfera: str | None = None
    poder: str | None = None
    valor: float | None = None


class AnexoRelatorio(BaseModel):
    """Catalog entry for available report attachments."""

    model_config = ConfigDict(populate_by_name=True, extra="allow")

    esfera: str
    co_tipo_demonstrativo: str
    no_anexo: str
    de_anexo: str | None = None
