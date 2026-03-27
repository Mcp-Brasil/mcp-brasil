"""Pydantic schemas for the TCE-ES feature."""

from __future__ import annotations

from pydantic import BaseModel


class Licitacao(BaseModel):
    """Processo licitatório do próprio TCEES."""

    Modalidade: str | None = None
    NumeroEdital: str | None = None
    AnoEdital: str | None = None
    Objeto: str | None = None
    DataAbertura: str | None = None
    ValorReferencia: str | None = None
    ValorHomologado: str | None = None
    Situacao: str | None = None


class Contrato(BaseModel):
    """Contrato celebrado pelo TCEES."""

    ContratoNumero: str | None = None
    ContratoAno: str | None = None
    Modalidade: str | None = None
    ResumoObjeto: str | None = None
    FornecedorNome: str | None = None
    FornecedorDocumento: str | None = None
    TermoOriginalValorGlobal: str | None = None
    TermoOriginalDataInicioVigencia: str | None = None
    TermoOriginalDataFimVigencia: str | None = None
    VigenciaAtualValorGlobal: str | None = None
    Setor: str | None = None
    UrlPortalTransparencia: str | None = None


class ContratacaoMunicipio(BaseModel):
    """Contratação de município ou órgão capixaba monitorado pelo TCE-ES."""

    NomeUnidadeGestoraReferencia: str | None = None
    NomeEsferaAdministrativa: str | None = None
    ObjetoContratacao: str | None = None
    ModalidadeLicitacao: str | None = None
    ValorEstimado: str | None = None
    ValorTotalContratacao: str | None = None
    AnoReferencia: str | None = None
    SituacaoContratacao: str | None = None
    NomePoder: str | None = None


class Obra(BaseModel):
    """Obra pública cadastrada no TCE-ES."""

    Licitacao: str | None = None
    Contrato: str | None = None
    DataAssinaturaContrato: str | None = None
    Empresa: str | None = None
    EmpresaCNPJ: str | None = None
    ValorInicial: str | None = None
    Situacao: str | None = None
