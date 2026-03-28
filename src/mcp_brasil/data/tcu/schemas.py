"""Pydantic schemas for the TCU feature."""

from __future__ import annotations

from pydantic import BaseModel, Field


class Acordao(BaseModel):
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
    sumario: str = ""
    url_acordao: str = Field(default="", alias="urlAcordao")

    model_config = {"populate_by_name": True}


class Inabilitado(BaseModel):
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


class Inidoneo(BaseModel):
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


class CertidaoItem(BaseModel):
    """Certidão individual dentro da consulta consolidada."""

    emissor: str = ""
    tipo: str = ""
    descricao: str = ""
    situacao: str = ""
    observacao: str | None = None
    data_hora_emissao: str = Field(default="", alias="dataHoraEmissao")

    model_config = {"populate_by_name": True}


class CertidaoConsolidada(BaseModel):
    """Resultado consolidado de certidões de pessoa jurídica."""

    razao_social: str = Field(default="", alias="razaoSocial")
    nome_fantasia: str = Field(default="", alias="nomeFantasia")
    cnpj: str = ""
    certidoes: list[CertidaoItem] = []
    se_cnpj_encontrado: bool = Field(default=False, alias="seCnpjEncontradoNaBaseTcu")

    model_config = {"populate_by_name": True}


class PedidoCongresso(BaseModel):
    """Solicitação do Congresso Nacional ao TCU."""

    tipo: str = ""
    numero: int = 0
    data_aprovacao: str = ""
    assunto: str = ""
    autor: str | None = None
    processo_scn: str = ""
    link_proposicao: str = ""


class ParcelaDebito(BaseModel):
    """Parcela de débito para cálculo."""

    data_fato: str = Field(description="Data do fato gerador (DD/MM/AAAA)")
    indicativo: str = Field(default="D", description="D para débito, C para crédito")
    valor_original: float = Field(description="Valor original da parcela")


class ResultadoDebito(BaseModel):
    """Resultado do cálculo de débito."""

    data: str = ""
    saldo_debito: float = Field(default=0.0, alias="saldoDebito")
    saldo_variacao_selic: float = Field(default=0.0, alias="saldoVariacaoSelic")
    saldo_juros: float = Field(default=0.0, alias="saldoJuros")
    saldo_total: float = Field(default=0.0, alias="saldoTotal")

    model_config = {"populate_by_name": True}


class TermoContratual(BaseModel):
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


class PessoaCadirreg(BaseModel):
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
