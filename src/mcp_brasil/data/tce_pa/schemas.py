"""Pydantic schemas for the TCE-PA feature."""

from __future__ import annotations

from pydantic import BaseModel


class DiarioOficial(BaseModel):
    """Publicação no Diário Oficial do TCE-PA."""

    numero_publicacao: int | None = None
    publicacao: str | None = None
    data_publicacao: str | None = None
    tipo_ato: str | None = None


class SessaoPlenaria(BaseModel):
    """Registro de sessão plenária do TCE-PA."""

    codigo: int | None = None
    titulo: str | None = None
    data_sessao: str | None = None
    tipo_sessao: str | None = None
    ano: int | None = None
    url_pagina: str | None = None
    url_documento: str | None = None


class ResultadoPesquisa(BaseModel):
    """Resultado genérico da Pesquisa Integrada do TCE-PA."""

    titulo: str | None = None
    url_pagina: str | None = None
    url_documento: str | None = None
    campos: dict[str, str] = {}
