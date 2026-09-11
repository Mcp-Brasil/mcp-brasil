"""Pydantic schemas for the TCU feature."""

from __future__ import annotations

from typing import Any, Union, get_args, get_origin

from pydantic import BaseModel, Field, model_validator


def _accepts_none(annotation: Any) -> bool:
    """Return True when the annotation allows None (Optional/Union with None)."""
    if annotation is None or annotation is type(None):
        return True
    if get_origin(annotation) is Union:
        return type(None) in get_args(annotation)
    # PEP 604 unions (str | None) surface as types.UnionType at runtime.
    return type(None) in get_args(annotation)


class TcuModel(BaseModel):
    """Base model that treats an explicit JSON ``null`` as an absent field.

    Several TCU endpoints send ``null`` instead of omitting a value - for
    example ``nomeFantasia`` for a company with no trade name. Pydantic
    rejects that for non-optional fields, which made the whole response fail
    to parse. Dropping those nulls lets the declared default apply, while
    fields typed as optional keep their explicit ``None``.
    """

    @model_validator(mode="before")
    @classmethod
    def _null_means_absent(cls, data: Any) -> Any:
        if not isinstance(data, dict):
            return data
        nullable: set[str] = set()
        for name, field in cls.model_fields.items():
            if _accepts_none(field.annotation):
                nullable.add(name)
                if field.alias:
                    nullable.add(field.alias)
        return {k: v for k, v in data.items() if v is not None or k in nullable}


class Acordao(TcuModel):
    """Acórdão (decisão colegiada) do TCU."""

    key: str = Field(description="Identificador único (ex: ACORDAO-COMPLETO-2745491)")
    tipo: str = ""
    ano_acordao: str = Field(default="", alias="anoAcordao")
    titulo: str = ""
    numero_acordao: str = Field(default="", alias="numeroAcordao")
    numero_ata: str = Field(default="", alias="numeroAta")
    colegiado: str = ""
    data_sessao: str = Field(default="", alias="dataSessao")
    relator: str = ""
    situacao: str = ""
    sumario: str | None = ""
    url_acordao: str = Field(default="", alias="urlAcordao")

    model_config = {"populate_by_name": True}


class Inabilitado(TcuModel):
    """Pessoa inabilitada para exercer cargo/função pública pelo TCU."""

    nome: str = ""
    cpf: str = ""
    processo: str = ""
    deliberacao: str = ""
    data_transito_julgado: str = ""
    data_final: str = ""
    data_acordao: str = ""
    uf: str = ""
    municipio: str = ""


class Inidoneo(TcuModel):
    """Licitante declarado inidôneo pelo TCU."""

    nome: str = ""
    cpf_cnpj: str = ""
    processo: str = ""
    deliberacao: str = ""
    data_transito_julgado: str = ""
    data_final: str = ""
    data_acordao: str = ""
    uf: str = ""
    municipio: str | None = None


class CertidaoItem(TcuModel):
    """Certidão individual dentro da consulta consolidada."""

    emissor: str = ""
    tipo: str = ""
    descricao: str = ""
    situacao: str = ""
    observacao: str | None = None
    data_hora_emissao: str = Field(default="", alias="dataHoraEmissao")

    model_config = {"populate_by_name": True}


class CertidaoConsolidada(TcuModel):
    """Resultado consolidado de certidões de pessoa jurídica."""

    razao_social: str = Field(default="", alias="razaoSocial")
    nome_fantasia: str = Field(default="", alias="nomeFantasia")
    cnpj: str = ""
    certidoes: list[CertidaoItem] = []
    se_cnpj_encontrado: bool = Field(default=False, alias="seCnpjEncontradoNaBaseTcu")

    model_config = {"populate_by_name": True}


class PedidoCongresso(TcuModel):
    """Solicitação do Congresso Nacional ao TCU."""

    tipo: str = ""
    numero: int = 0
    data_aprovacao: str = ""
    assunto: str = ""
    autor: str | None = None
    processo_scn: str = ""
    link_proposicao: str = ""


class ParcelaDebito(TcuModel):
    """Parcela de débito para cálculo."""

    data_fato: str = Field(description="Data do fato gerador (DD/MM/AAAA)")
    indicativo: str = Field(default="D", description="D para débito, C para crédito")
    valor_original: float = Field(description="Valor original da parcela")


class ResultadoDebito(TcuModel):
    """Resultado do cálculo de débito."""

    data: str = ""
    saldo_debito: float = Field(default=0.0, alias="saldoDebito")
    saldo_variacao_selic: float = Field(default=0.0, alias="saldoVariacaoSelic")
    saldo_juros: float = Field(default=0.0, alias="saldoJuros")
    saldo_total: float = Field(default=0.0, alias="saldoTotal")

    model_config = {"populate_by_name": True}


class TermoContratual(TcuModel):
    """Termo contratual firmado pelo TCU."""

    numero: int = 0
    ano: int = 0
    tipo_contratacao: str = Field(default="", alias="tipoContratacao")
    nome_fornecedor: str = Field(default="", alias="nomeFornecedor")
    cnpj_fornecedor: str = Field(default="", alias="cnpjFornecedor")
    objeto: str = ""
    valor_inicial: float = Field(default=0.0, alias="valorInicial")
    valor_atualizado: float = Field(default=0.0, alias="valorAtualizado")
    data_assinatura: str = Field(default="", alias="dataAssinatura")
    data_termino_vigencia: str = Field(default="", alias="dataTerminoVigencia")
    modalidade_licitacao: str = Field(default="", alias="modalidadeLicitacao")
    numero_processo: str = Field(default="", alias="numeroProcesso")
    unidade_gestora: str = Field(default="", alias="unidadeGestora")

    model_config = {"populate_by_name": True}


class PautaSessao(TcuModel):
    """Pauta de sessão de julgamento do TCU."""

    nome_colegiado: str = Field(default="", alias="nomeColegiado")
    sigla_colegiado: str = Field(default="", alias="siglaColegiado")
    data_sessao: str = Field(default="", alias="dataSessao")
    sigla_relator: str = Field(default="", alias="siglaRelator")
    nome_relator: str = Field(default="", alias="nomeRelator")
    numero_processo: str = Field(default="", alias="numeroProcesso")
    natureza_processo: str = Field(default="", alias="naturezaProcesso")
    tipo_processo: str = Field(default="", alias="tipoProcesso")

    model_config = {"populate_by_name": True}


class PessoaCadirreg(TcuModel):
    """Pessoa com contas julgadas irregulares pelo TCU (CADIRREG)."""

    nome_responsavel: str = Field(default="", alias="nomeResponsavel")
    num_cpf: str = Field(default="", alias="numCPF")
    num_processo: str = Field(default="", alias="numProcesso")
    ano_processo: str = Field(default="", alias="anoProcesso")
    julgamento: str = ""
    unidade_tecnica_processo: str = Field(default="", alias="unidadeTecnicaProcesso")
    se_detentor_cargo: str = Field(default="", alias="seDetentorCargoFuncaoPublica")
    se_falecido: str = Field(default="", alias="seFalecido")

    model_config = {"populate_by_name": True}
