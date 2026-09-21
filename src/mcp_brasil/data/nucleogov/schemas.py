"""Pydantic schemas for the NucleoGov feature."""

from __future__ import annotations

from pydantic import BaseModel

# ---------------------------------------------------------------------------
# Despesas / empenhos
# ---------------------------------------------------------------------------


class Despesa(BaseModel):
    """Empenho (despesa) publicado por um portal municipal NucleoGov."""

    id: str | None = None
    orgao: str | None = None
    unidade: str | None = None
    numero_processo: str | None = None
    numero_empenho: str | None = None
    tipo: str | None = None
    data: str | None = None
    fornecedor: str | None = None
    cpf_cnpj: str | None = None
    historico: str | None = None
    funcao: str | None = None
    subfuncao: str | None = None
    programa: str | None = None
    natureza: str | None = None
    fonte: str | None = None
    elemento: str | None = None
    modalidade: str | None = None
    numero_licitacao: str | None = None
    valor_empenho: float | None = None
    valor_pago: float | None = None
    valor_saldo: float | None = None


class ResultadoDespesas(BaseModel):
    """Resultado paginado de despesas."""

    despesas: list[Despesa] = []
    total: int = 0


# ---------------------------------------------------------------------------
# Contratos
# ---------------------------------------------------------------------------


class Contrato(BaseModel):
    """Contrato (ou aditivo) publicado por um portal municipal NucleoGov."""

    id: str | None = None
    numero: str | None = None
    numero_detalhado: str | None = None
    licitacao: str | None = None
    ano: int | None = None
    orgao: str | None = None
    situacao: str | None = None
    tipo: str | None = None
    data_publicacao: str | None = None
    data_firmatura: str | None = None
    inicio_vigencia: str | None = None
    fim_vigencia: str | None = None
    contratado_nome: str | None = None
    contratado_documento: str | None = None
    fiscal_contrato: str | None = None
    objeto: str | None = None
    valor: float | None = None


class ResultadoContratos(BaseModel):
    """Resultado paginado de contratos."""

    contratos: list[Contrato] = []
    total: int = 0


# ---------------------------------------------------------------------------
# Licitações / dispensas
# ---------------------------------------------------------------------------


class Licitacao(BaseModel):
    """Processo licitatório ou contratação direta publicado pelo portal."""

    id: str | None = None
    numero: str | None = None
    numero_processo: str | None = None
    modalidade: str | None = None
    modalidade_tce: str | None = None
    situacao: str | None = None
    orgao: str | None = None
    ano: int | None = None
    anobase: int | None = None
    data_abertura: str | None = None
    data_edital: str | None = None
    descricao: str | None = None
    valor_estimado: float | None = None


class ResultadoLicitacoes(BaseModel):
    """Resultado paginado de licitações (ou dispensas/inexigibilidades)."""

    licitacoes: list[Licitacao] = []
    total: int = 0


# ---------------------------------------------------------------------------
# Legislação
# ---------------------------------------------------------------------------


class AtoNormativo(BaseModel):
    """Lei, decreto, portaria ou outro ato publicado pelo portal."""

    id: str | None = None
    tipo: str | None = None
    numero: str | None = None
    ementa: str | None = None
    orgao: str | None = None
    data_publicacao: str | None = None
    url: str | None = None


class ResultadoAtos(BaseModel):
    """Resultado paginado de atos normativos."""

    atos: list[AtoNormativo] = []
    total: int = 0
