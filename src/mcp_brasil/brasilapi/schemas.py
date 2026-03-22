"""Pydantic schemas for the BrasilAPI feature."""

from __future__ import annotations

from pydantic import BaseModel, Field


class Endereco(BaseModel):
    """Endereço retornado pela consulta de CEP."""

    cep: str
    state: str = Field(description="Sigla do estado (UF)")
    city: str
    neighborhood: str | None = None
    street: str | None = None
    service: str | None = None


class EmpresaCnpj(BaseModel):
    """Dados de empresa retornados pela consulta de CNPJ."""

    cnpj: str
    razao_social: str | None = None
    nome_fantasia: str | None = None
    situacao_cadastral: int | None = None
    descricao_situacao_cadastral: str | None = None
    data_situacao_cadastral: str | None = None
    porte: str | None = None
    natureza_juridica: str | None = None
    cnae_fiscal: int | None = None
    cnae_fiscal_descricao: str | None = None
    logradouro: str | None = None
    numero: str | None = None
    complemento: str | None = None
    bairro: str | None = None
    cep: str | None = None
    uf: str | None = None
    municipio: str | None = None
    ddd_telefone_1: str | None = None
    email: str | None = None
    capital_social: float | None = None


class DddInfo(BaseModel):
    """Informações de um DDD (código de área)."""

    state: str
    cities: list[str]


class Banco(BaseModel):
    """Dados de um banco brasileiro."""

    ispb: str | None = None
    name: str | None = None
    code: int | None = None
    fullName: str | None = None


class Moeda(BaseModel):
    """Moeda disponível para câmbio."""

    simbolo: str
    nome_formatado: str
    tipo_moeda: str | None = None


class Cotacao(BaseModel):
    """Cotação de câmbio."""

    moeda: str
    data: str
    valor_compra: float | None = None
    valor_venda: float | None = None


class Feriado(BaseModel):
    """Feriado nacional brasileiro."""

    date: str
    name: str
    type: str


class TaxaOficial(BaseModel):
    """Taxa/índice oficial da economia."""

    nome: str
    valor: float | None = None


class FipeTabela(BaseModel):
    """Tabela de referência FIPE."""

    codigo: int
    mes: str


class FipeMarca(BaseModel):
    """Marca de veículo na tabela FIPE."""

    nome: str
    valor: str


class FipeVeiculo(BaseModel):
    """Veículo na tabela FIPE."""

    valor: str
    marca: str | None = None
    modelo: str | None = None
    ano_modelo: int | None = None
    combustivel: str | None = None
    codigo_fipe: str | None = None
    mes_referencia: str | None = None


class Livro(BaseModel):
    """Dados de livro retornados pela consulta ISBN."""

    isbn: str | None = None
    title: str | None = None
    subtitle: str | None = None
    authors: list[str] | None = None
    publisher: str | None = None
    year: int | str | None = None
    format: str | None = None
    page_count: int | None = None
    subjects: list[str] | None = None
    provider: str | None = None


class NcmItem(BaseModel):
    """Item da Nomenclatura Comum do Mercosul."""

    codigo: str
    descricao: str
    data_inicio: str | None = None
    data_fim: str | None = None
    tipo_ato: str | None = None
    numero_ato: str | None = None
    ano_ato: str | None = None


class PixParticipante(BaseModel):
    """Participante do sistema PIX."""

    ispb: str | None = None
    nome: str | None = None
    nome_reduzido: str | None = None
    modalidade_participacao: str | None = None
    tipo_participacao: str | None = None
    inicio_operacao: str | None = None


class RegistroBrDominio(BaseModel):
    """Status de domínio no Registro.br."""

    status_code: int | None = None
    status: str | None = None
    fqdn: str | None = None
    hosts: list[str] | None = None
    publication_status: str | None = None
    expires_at: str | None = None
