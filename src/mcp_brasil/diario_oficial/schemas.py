"""Pydantic schemas for the Diário Oficial feature."""

from __future__ import annotations

from pydantic import BaseModel


class DiarioOficial(BaseModel):
    """Edição de diário oficial retornada pela API."""

    territory_id: str | None = None
    territory_name: str | None = None
    state_code: str | None = None
    date: str | None = None
    edition_number: str | None = None
    is_extra_edition: bool | None = None
    url: str | None = None
    txt_url: str | None = None
    excerpts: list[str] | None = None
    highlight_texts: list[str] | None = None


class DiarioResultado(BaseModel):
    """Resultado paginado da busca de diários."""

    total_gazettes: int = 0
    gazettes: list[DiarioOficial] = []


class CidadeQueridoDiario(BaseModel):
    """Cidade disponível na base do Querido Diário."""

    territory_id: str
    territory_name: str
    state_code: str
    publication_urls: list[str] | None = None
    level: str | None = None
