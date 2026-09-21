"""Pydantic schemas for the Megasoft municipal transparency portal feature."""

from __future__ import annotations

from pydantic import BaseModel, Field


class Empenho(BaseModel):
    """Empenho (despesa empenhada) de um município."""

    codigo: int | None = Field(default=None, description="Código interno do empenho")
    numero_tcm: int | None = Field(default=None, description="Número do empenho no TCM")
    data: str | None = Field(default=None, description="Data do empenho (DD/MM/AAAA)")
    fornecedor: str | None = Field(default=None, description="Nome do fornecedor")
    cnpj_fornecedor: str | None = Field(
        default=None, description="CNPJ/CPF e nome do fornecedor, como publicado"
    )
    classificacao_orcamentaria: str | None = Field(
        default=None, description="Classificação orçamentária da despesa"
    )
    valor_empenho: float | None = Field(default=None, description="Valor empenhado")
    valor_liquidacao: float | None = Field(default=None, description="Valor liquidado")
    valor_pagamento: float | None = Field(default=None, description="Valor pago")
    orgao: str | None = Field(default=None, description="Órgão responsável")
    unidade: str | None = Field(default=None, description="Unidade gestora")
    funcao: str | None = Field(default=None, description="Função de governo")
    natureza_despesa: str | None = Field(default=None, description="Natureza da despesa")
    fonte: str | None = Field(default=None, description="Fonte de recurso")
    historico: str | None = Field(default=None, description="Histórico do empenho")


class ResultadoEmpenhos(BaseModel):
    """Resultado paginado de empenhos."""

    empenhos: list[Empenho] = []
    total: int = 0
    pagina: int = 1


class OrdemPagamento(BaseModel):
    """Ordem de pagamento de um município."""

    codigo: int | None = Field(default=None, description="Código interno da ordem")
    numero: int | None = Field(default=None, description="Número da ordem de pagamento")
    orgao: str | None = Field(default=None, description="Órgão responsável")
    data_pagamento: str | None = Field(default=None, description="Data do pagamento (DD/MM/AAAA)")
    fornecedor: str | None = Field(default=None, description="Nome do fornecedor")
    cpf_cnpj_fornecedor: str | None = Field(default=None, description="CPF ou CNPJ do fornecedor")
    numero_empenho: int | None = Field(default=None, description="Número do empenho de origem")
    numero_liquidacao: int | None = Field(
        default=None, description="Número da liquidação de origem"
    )
    valor_pago: float | None = Field(default=None, description="Valor pago")
    valor_anulacao: float | None = Field(default=None, description="Valor anulado")
    valor_total: float | None = Field(default=None, description="Valor total da ordem")
    valor_retencao: float | None = Field(default=None, description="Valor retido")
    fase: str | None = Field(default=None, description="Fase da despesa")


class ResultadoOrdensPagamento(BaseModel):
    """Resultado paginado de ordens de pagamento."""

    ordens: list[OrdemPagamento] = []
    total: int = 0
    pagina: int = 1


class TotaisOrdensPagamento(BaseModel):
    """Totais financeiros consolidados das ordens de pagamento de um período."""

    valor_pago: float | None = Field(default=None, description="Total pago no período")
    valor_anulacao: float | None = Field(default=None, description="Total anulado no período")
    valor_total: float | None = Field(default=None, description="Total líquido no período")
    valor_retencao: float | None = Field(default=None, description="Total retido no período")


class Receita(BaseModel):
    """Receita orçamentária de um município."""

    codigo_receita: int | None = Field(default=None, description="Código da receita")
    ano: int | None = Field(default=None, description="Exercício da receita")
    mes: str | None = Field(default=None, description="Mês de referência")
    codigo_orgao: int | None = Field(default=None, description="Código do órgão")
    fonte: str | None = Field(default=None, description="Fonte da receita")
    receita: str | None = Field(default=None, description="Descrição da receita")
    natureza: str | None = Field(default=None, description="Natureza da receita")
    origem_recurso: str | None = Field(default=None, description="Origem do recurso")
    valor_orcado: float | None = Field(default=None, description="Valor orçado")
    arrecadacao: float | None = Field(default=None, description="Valor arrecadado")
    valor_acumulado: float | None = Field(default=None, description="Valor acumulado")


class ResultadoReceitas(BaseModel):
    """Resultado paginado de receitas."""

    receitas: list[Receita] = []
    total: int = 0
    pagina: int = 1
