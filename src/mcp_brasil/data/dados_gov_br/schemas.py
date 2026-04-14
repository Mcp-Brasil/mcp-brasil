"""Pydantic schemas for the dados_gov_br feature.

Models based on the OpenAPI spec at https://dados.gov.br/v3/api-docs.
"""

from __future__ import annotations

from pydantic import BaseModel, ConfigDict

# --- Tags ---


class Tag(BaseModel):
    """Etiqueta temática de um dataset."""

    model_config = ConfigDict(extra="ignore")

    id: str | None = None
    name: str | None = None
    display_name: str | None = None


# --- Recursos ---


class Recurso(BaseModel):
    """Recurso (arquivo/API) dentro de um conjunto de dados."""

    model_config = ConfigDict(extra="ignore")

    id: str | None = None
    titulo: str | None = None
    descricao: str | None = None
    link: str | None = None
    formato: str | None = None
    tipo: str | None = None
    tamanho: int | None = None
    nomeArquivo: str | None = None
    quantidadeDownloads: int | None = None
    dataUltimaAtualizacaoArquivo: str | None = None
    dataCatalogacao: str | None = None


# --- Conjuntos de dados ---


class ConjuntoIndice(BaseModel):
    """Item resumido de um dataset na listagem.

    Schema: ApiResultadoListaConjuntoDados / IndiceConjuntoDados.
    """

    model_config = ConfigDict(extra="ignore")

    id: str | None = None
    title: str | None = None
    nome: str | None = None
    notes: str | None = None
    organizationName: str | None = None
    organizationTitle: str | None = None
    temas: str | None = None
    dataCriacao: str | None = None
    dataAtualizacao: str | None = None
    quantidadeRecursos: int | None = None
    quantidadeDownloads: int | None = None
    licenca: str | None = None
    dadosAbertos: bool | None = None


class ConjuntoDetalhe(BaseModel):
    """Detalhe completo de um dataset.

    Schema: ConjuntoDadosApiView.
    """

    model_config = ConfigDict(extra="ignore")

    id: str | None = None
    titulo: str | None = None
    nome: str | None = None
    descricao: str | None = None
    organizacao: str | None = None
    licenca: str | None = None
    periodicidade: str | None = None
    responsavel: str | None = None
    emailResponsavel: str | None = None
    versao: str | None = None
    visibilidade: str | None = None
    coberturaEspacial: str | None = None
    coberturaTemporalInicio: str | None = None
    coberturaTemporalFim: str | None = None
    temas: list[dict[str, str]] = []
    tags: list[Tag] = []
    recursos: list[Recurso] = []
    dadosAbertos: str | None = None
    descontinuado: bool | None = None
    dataUltimaAtualizacaoMetadados: str | None = None
    dataUltimaAtualizacaoArquivo: str | None = None
    dataCatalogacao: str | None = None
    observanciaLegal: str | None = None
    ods: list[int] = []


class ConjuntoResultado(BaseModel):
    """Resultado paginado de listagem de conjuntos."""

    total: int = 0
    conjuntos: list[ConjuntoIndice] = []


# --- Organizações ---


class OrganizacaoIndice(BaseModel):
    """Item resumido de uma organização na listagem.

    Schema: IndiceOrganizacao.
    """

    model_config = ConfigDict(extra="ignore")

    id: str | None = None
    titulo: str | None = None
    nome: str | None = None
    descricao: str | None = None
    urlImagem: str | None = None
    qtdConjuntoDeDados: str | None = None
    qtdSeguidores: str | None = None
    organizationEsfera: str | None = None
    organizationUf: str | None = None


class OrganizacaoDetalhe(BaseModel):
    """Detalhe completo de uma organização.

    Schema: OrganizacaoApiDTO.
    """

    model_config = ConfigDict(extra="ignore")

    id: str | None = None
    name: str | None = None
    displayName: str | None = None
    nome: str | None = None
    descricao: str | None = None
    urlFoto: str | None = None
    quantidadeSeguidores: int | None = None
    quantidadeConjuntoDados: int | None = None
    ativo: str | None = None


class OrganizacaoResultado(BaseModel):
    """Resultado paginado de organizações."""

    total: int = 0
    organizacoes: list[OrganizacaoIndice] = []


# --- Temas ---


class Tema(BaseModel):
    """Grupo temático de datasets.

    Schema: TemaDTO.
    """

    model_config = ConfigDict(extra="ignore")

    id: str | None = None
    name: str | None = None
    title: str | None = None
    description: str | None = None
    displayName: str | None = None
    packageCount: int | None = None


# --- Reusos ---


class ReusoIndice(BaseModel):
    """Item resumido de um reuso na listagem.

    Schema: ApiResultadoListaReuso.
    """

    model_config = ConfigDict(extra="ignore")

    id: str | None = None
    nome: str | None = None
    autor: str | None = None
    organizacao: str | None = None
    dataAtualizacao: str | None = None
    situacao: str | None = None


class ReusoDetalhe(BaseModel):
    """Detalhe completo de um reuso.

    Schema: ApiResultadoReuso.
    """

    model_config = ConfigDict(extra="ignore")

    id: str | None = None
    nome: str | None = None
    descricao: str | None = None
    url: str | None = None
    organizacao: str | None = None
    responsavel: str | None = None
    autor: str | None = None
    situacao: str | None = None
    dataAtualizacao: str | None = None
    dataLancamento: str | None = None
    temas: list[str] = []
    tipos: list[str] = []
    palavrasChave: list[str] = []
    conjuntoDados: list[str] = []


# --- Auxiliares ---


class ODS(BaseModel):
    """Objetivo de Desenvolvimento Sustentável."""

    model_config = ConfigDict(extra="ignore")

    id: int | None = None
    descricao: str | None = None


class ObservanciaLegal(BaseModel):
    """Opção de observância legal."""

    model_config = ConfigDict(extra="ignore")

    id: int | None = None
    descricao: str | None = None
