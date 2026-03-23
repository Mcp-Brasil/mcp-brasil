"""Pydantic schemas for the TCE-SP feature."""

from __future__ import annotations

from pydantic import BaseModel


class Municipio(BaseModel):
    """Município sob jurisdição do TCE-SP."""

    municipio: str  # slug (ex: "campinas")
    municipio_extenso: str  # nome completo (ex: "Campinas")


class Despesa(BaseModel):
    """Despesa municipal registrada no TCE-SP."""

    orgao: str | None = None
    mes: str | None = None
    evento: str | None = None  # Empenhado, Valor Pago, Valor Liquidado, Anulação
    nr_empenho: str | None = None
    id_fornecedor: str | None = None
    nm_fornecedor: str | None = None
    dt_emissao_despesa: str | None = None
    vl_despesa: float | None = None  # parsed from Brazilian format


class Receita(BaseModel):
    """Receita municipal registrada no TCE-SP."""

    orgao: str | None = None
    mes: str | None = None
    ds_fonte_recurso: str | None = None
    ds_cd_aplicacao_fixo: str | None = None
    ds_alinea: str | None = None
    ds_subalinea: str | None = None
    vl_arrecadacao: float | None = None  # parsed from Brazilian format
