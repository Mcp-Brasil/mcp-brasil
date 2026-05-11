"""Pydantic schemas for the Siscomex feature."""

from __future__ import annotations

from pydantic import BaseModel


class NcmItem(BaseModel):
    """NCM entry from the official Siscomex nomenclature table."""

    codigo: str
    descricao: str
    dataUltimaAlteracao: str | None = None
    dataInicio: str | None = None
    dataFim: str | None = None
    tipoOrgaoAtoIni: str | None = None
    numeroAtoIni: str | None = None
    anoAtoIni: int | None = None
