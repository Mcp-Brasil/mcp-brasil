"""Pydantic schemas for the Compras feature."""

from __future__ import annotations

from pydantic import BaseModel


class Contratacao(BaseModel):
    """Contratação pública retornada pelo PNCP."""

    orgao_cnpj: str | None = None
    orgao_nome: str | None = None
    ano: int | None = None
    numero_sequencial: int | None = None
    numero_controle_pncp: str | None = None
    objeto: str | None = None
    modalidade_id: int | None = None
    modalidade_nome: str | None = None
    situacao_id: int | None = None
    situacao_nome: str | None = None
    valor_estimado: float | None = None
    valor_homologado: float | None = None
    data_publicacao: str | None = None
    data_abertura: str | None = None
    uf: str | None = None
    municipio: str | None = None
    esfera: str | None = None
    link_pncp: str | None = None


class ContratacaoResultado(BaseModel):
    """Resultado paginado de busca de contratações."""

    total: int = 0
    contratacoes: list[Contratacao] = []


class Contrato(BaseModel):
    """Contrato público."""

    orgao_cnpj: str | None = None
    orgao_nome: str | None = None
    numero_contrato: str | None = None
    objeto: str | None = None
    fornecedor_cnpj: str | None = None
    fornecedor_nome: str | None = None
    valor_inicial: float | None = None
    valor_final: float | None = None
    vigencia_inicio: str | None = None
    vigencia_fim: str | None = None
    data_publicacao: str | None = None
    situacao: str | None = None


class ContratoResultado(BaseModel):
    """Resultado paginado de busca de contratos."""

    total: int = 0
    contratos: list[Contrato] = []


class AtaRegistroPreco(BaseModel):
    """Ata de registro de preço."""

    orgao_cnpj: str | None = None
    orgao_nome: str | None = None
    numero_ata: str | None = None
    objeto: str | None = None
    fornecedor_cnpj: str | None = None
    fornecedor_nome: str | None = None
    valor_total: float | None = None
    vigencia_inicio: str | None = None
    vigencia_fim: str | None = None
    situacao: str | None = None


class AtaResultado(BaseModel):
    """Resultado paginado de busca de atas."""

    total: int = 0
    atas: list[AtaRegistroPreco] = []
