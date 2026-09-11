"""Schemas Pydantic da feature CPF.CNPJ.

Mapeiam o retorno real da API (``RespostaCNPJ``) por nome de chave, tolerando
campos ausentes/nulos. ``qualificacao_socio`` é polimórfico: objeto ``{id,
descricao}`` no pacote 6 e string no pacote 25; ambos são aceitos.
"""

from __future__ import annotations

from pydantic import BaseModel, Field


class Qualificacao(BaseModel):
    """Qualificação (de sócio, responsável ou representante)."""

    id: int | None = None
    descricao: str | None = None


class Motivo(BaseModel):
    """Motivo da situação cadastral."""

    id: int | None = None
    descricao: str | None = None


class MatrizEndereco(BaseModel):
    """Endereço da matriz."""

    cep: str | None = None
    tipo: str | None = None
    logradouro: str | None = None
    numero: str | None = None
    complemento: str | None = None
    bairro: str | None = None
    cidade: str | None = None
    uf: str | None = None


class IbgeEstado(BaseModel):
    """Estado com identificadores oficiais do IBGE."""

    id: int | None = None
    nome: str | None = None
    sigla: str | None = None
    ibge_id: int | None = None


class IbgeCidade(BaseModel):
    """Município com identificadores oficiais do IBGE."""

    id: int | None = None
    nome: str | None = None
    ibge_id: int | None = None
    siafi_id: int | None = None


class Ibge(BaseModel):
    """Bloco de identificadores IBGE (estado e cidade)."""

    estado: IbgeEstado | None = None
    cidade: IbgeCidade | None = None


class Situacao(BaseModel):
    """Situação cadastral com data e motivo."""

    id: int | None = None
    nome: str | None = None
    data: str | None = None
    motivo: Motivo | None = None


class Porte(BaseModel):
    """Porte da empresa."""

    id: int | None = None
    descricao: str | None = None


class NaturezaJuridica(BaseModel):
    """Natureza jurídica."""

    codigo: str | None = None
    descricao: str | None = None


class CnaeSecundaria(BaseModel):
    """CNAE secundária."""

    id: int | None = None
    descricao: str | None = None


class Cnae(BaseModel):
    """CNAE principal (fiscal) e secundárias."""

    fiscal: str | None = None
    descricao: str | None = None
    secundarias: list[CnaeSecundaria] = Field(default_factory=list)


class SimplesNacional(BaseModel):
    """Situação no Simples Nacional / MEI.

    A API pode devolver ``optante``/``mei`` como bool ou como string
    ("Sim"/"Não"); ambos os formatos são tolerados.
    """

    optante: bool | str | None = None
    inicio: str | None = None
    fim: str | None = None
    mei: bool | str | None = None


class RegimeTributario(BaseModel):
    """Regime tributário por ano."""

    ano: int | None = None
    regime_tributario: str | None = None
    forma_de_tributacao: str | None = None


class Socio(BaseModel):
    """Sócio do quadro societário (QSA)."""

    cpf_cnpj_socio: str | None = None
    nome: str | None = None
    tipo: str | None = None
    data_entrada: str | None = None
    percentual: float | None = None
    qualificacao_socio: Qualificacao | str | None = None


class InscricaoEstadual(BaseModel):
    """Inscrição estadual por UF."""

    inscricao_estadual: str | None = None
    ativo: bool | None = None
    atualizado_em: str | None = None
    estado: IbgeEstado | None = None


class EmpresaCnpj(BaseModel):
    """Resposta consolidada de CNPJ (superconjunto dos pacotes 5, 6 e 16)."""

    status: int | None = None
    cnpj: str | None = None
    tipo: str | None = None
    razao: str | None = None
    fantasia: str | None = None
    matrizEndereco: MatrizEndereco | None = None
    ibge: Ibge | None = None
    inicioAtividade: str | None = None
    email: str | None = None
    responsavel: str | None = None
    responsavelCpf: str | None = None
    responsavelQualificacao: Qualificacao | None = None
    situacao: Situacao | None = None
    naturezaJuridica: NaturezaJuridica | None = None
    porte: Porte | None = None
    cnae: Cnae | None = None
    simplesNacional: SimplesNacional | None = None
    regimesTributarios: list[RegimeTributario] = Field(default_factory=list)
    socios: list[Socio] = Field(default_factory=list)
    inscricoesEstaduais: list[InscricaoEstadual] = Field(default_factory=list)
    pacoteUsado: int | None = None
    saldo: int | None = None
    consultaID: str | None = None


class Saldo(BaseModel):
    """Saldo de créditos de um pacote."""

    id: int | None = None
    nome: str | None = None
    saldo: int | None = None
