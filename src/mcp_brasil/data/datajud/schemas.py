"""Pydantic models for DataJud (CNJ) API responses."""

from __future__ import annotations

from pydantic import BaseModel


class Processo(BaseModel):
    """Processo judicial retornado pela API DataJud."""

    numero: str | None = None
    classe: str | None = None
    assunto: str | None = None
    tribunal: str | None = None
    orgao_julgador: str | None = None
    data_ajuizamento: str | None = None
    data_ultima_atualizacao: str | None = None
    grau: str | None = None
    nivel_sigilo: int | None = None
    formato_numero: str | None = None


class Movimentacao(BaseModel):
    """Movimentação de um processo judicial."""

    data: str | None = None
    nome: str | None = None
    codigo: int | None = None
    complemento: str | None = None


class Assunto(BaseModel):
    """Assunto processual."""

    codigo: int | None = None
    nome: str | None = None


class Parte(BaseModel):
    """Parte de um processo (autor, réu, etc.)."""

    nome: str | None = None
    tipo: str | None = None
    polo: str | None = None
    documento: str | None = None


class ProcessoDetalhe(BaseModel):
    """Detalhes completos de um processo judicial."""

    numero: str | None = None
    classe: str | None = None
    assuntos: list[Assunto] | None = None
    tribunal: str | None = None
    orgao_julgador: str | None = None
    data_ajuizamento: str | None = None
    data_ultima_atualizacao: str | None = None
    grau: str | None = None
    partes: list[Parte] | None = None
    movimentacoes: list[Movimentacao] | None = None
    nivel_sigilo: int | None = None


class MPUProcesso(BaseModel):
    """Processo de Medida Protetiva de Urgência."""

    numero: str | None = None
    classe_codigo: int | None = None
    classe_nome: str | None = None
    tribunal: str | None = None
    orgao_julgador: str | None = None
    data_ajuizamento: str | None = None
    movimentos: list[Movimentacao] | None = None


class MPUEstatisticas(BaseModel):
    """Estatísticas agregadas de MPUs."""

    total: int = 0
    tribunal: str | None = None
    periodo: str | None = None
    por_decisao: dict[str, int] | None = None
    por_classe: dict[str, int] | None = None
