"""Pydantic schemas for the BNDES CKAN API."""

from __future__ import annotations

from pydantic import BaseModel


class DatasetResource(BaseModel):
    """Resource within a CKAN dataset."""

    id: str | None = None
    name: str | None = None
    description: str | None = None
    format: str | None = None
    url: str | None = None


class Dataset(BaseModel):
    """CKAN dataset metadata."""

    name: str | None = None
    title: str | None = None
    notes: str | None = None
    num_resources: int | None = None
    resources: list[DatasetResource] = []


class DatastoreResult(BaseModel):
    """Result from a CKAN datastore_search query."""

    total: int = 0
    fields: list[str] = []
    records: list[dict[str, object]] = []


class OperacaoFinanciamento(BaseModel):
    """Operação de financiamento do BNDES."""

    cliente: str | None = None
    cnpj: str | None = None
    uf: str | None = None
    municipio: str | None = None
    setor_cnae: str | None = None
    subsetor_cnae: str | None = None
    porte_do_cliente: str | None = None
    produto: str | None = None
    instrumento_financeiro: str | None = None
    valor_da_operacao_em_reais: float | None = None
    custo_financeiro: str | None = None
    juros: float | None = None
    situacao: str | None = None
    data_da_contratacao: str | None = None


class InstituicaoCredenciada(BaseModel):
    """Instituição financeira credenciada pelo BNDES."""

    cnpj: str | None = None
    nome: str | None = None
    pagina_na_internet: str | None = None
