"""Pydantic models for Câmara dos Deputados API responses."""

from __future__ import annotations

from pydantic import BaseModel


class Deputado(BaseModel):
    """Deputado federal."""

    id: int | None = None
    nome: str | None = None
    sigla_partido: str | None = None
    sigla_uf: str | None = None
    email: str | None = None
    foto: str | None = None
    legislatura: int | None = None


class Proposicao(BaseModel):
    """Proposição legislativa (PL, PEC, MPV, etc.)."""

    id: int | None = None
    sigla_tipo: str | None = None
    numero: int | None = None
    ano: int | None = None
    ementa: str | None = None
    data_apresentacao: str | None = None
    situacao: str | None = None
    autor: str | None = None


class Tramitacao(BaseModel):
    """Evento de tramitação de uma proposição."""

    data: str | None = None
    descricao: str | None = None
    orgao: str | None = None
    situacao: str | None = None
    despacho: str | None = None


class Votacao(BaseModel):
    """Votação em plenário ou comissão."""

    id: str | None = None
    data: str | None = None
    descricao: str | None = None
    aprovacao: bool | None = None
    proposicao_id: int | None = None
    proposicao_descricao: str | None = None


class VotoNominal(BaseModel):
    """Voto individual de deputado em uma votação."""

    deputado_id: int | None = None
    deputado_nome: str | None = None
    partido: str | None = None
    uf: str | None = None
    voto: str | None = None


class DespesaDeputado(BaseModel):
    """Despesa de cota parlamentar (CEAP)."""

    deputado_id: int | None = None
    deputado_nome: str | None = None
    tipo_despesa: str | None = None
    fornecedor: str | None = None
    cnpj_cpf: str | None = None
    valor_documento: float | None = None
    valor_liquido: float | None = None
    data_documento: str | None = None
    mes: int | None = None
    ano: int | None = None


class Evento(BaseModel):
    """Evento legislativo (sessão, audiência, reunião)."""

    id: int | None = None
    data_inicio: str | None = None
    data_fim: str | None = None
    titulo: str | None = None
    descricao: str | None = None
    local: str | None = None
    situacao: str | None = None
    orgaos: str | None = None


class Orgao(BaseModel):
    """Órgão legislativo (comissão, CPI, etc.)."""

    id: int | None = None
    sigla: str | None = None
    nome: str | None = None
    tipo: str | None = None
    situacao: str | None = None


class FrenteParlamentar(BaseModel):
    """Frente parlamentar."""

    id: int | None = None
    titulo: str | None = None
    legislatura: int | None = None
    coordenador: str | None = None
    situacao: str | None = None
