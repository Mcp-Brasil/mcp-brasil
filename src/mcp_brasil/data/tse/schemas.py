"""Pydantic models for TSE (Tribunal Superior Eleitoral) API responses."""

from __future__ import annotations

from pydantic import BaseModel


class Eleicao(BaseModel):
    """Eleição (ordinária ou suplementar)."""

    id: int | None = None
    sigla_uf: str | None = None
    ano: int | None = None
    codigo: str | None = None
    nome: str | None = None
    tipo: str | None = None
    turno: str | None = None
    tipo_abrangencia: str | None = None
    data_eleicao: str | None = None
    descricao: str | None = None


class Cargo(BaseModel):
    """Cargo eletivo."""

    codigo: int | None = None
    sigla: str | None = None
    nome: str | None = None
    titular: bool | None = None
    contagem: int | None = None


class CandidatoResumo(BaseModel):
    """Candidato (resumo da listagem)."""

    id: int | None = None
    nome_urna: str | None = None
    numero: int | None = None
    partido: str | None = None
    situacao: str | None = None
    foto_url: str | None = None


class Candidato(BaseModel):
    """Candidato com detalhes completos."""

    id: int | None = None
    nome_urna: str | None = None
    nome_completo: str | None = None
    numero: int | None = None
    cpf: str | None = None
    data_nascimento: str | None = None
    sexo: str | None = None
    estado_civil: str | None = None
    cor_raca: str | None = None
    nacionalidade: str | None = None
    grau_instrucao: str | None = None
    ocupacao: str | None = None
    uf_nascimento: str | None = None
    municipio_nascimento: str | None = None
    partido: str | None = None
    situacao: str | None = None
    situacao_candidato: str | None = None
    coligacao: str | None = None
    composicao_coligacao: str | None = None
    descricao_totalizacao: str | None = None
    total_votos: int | None = None
    gasto_campanha: float | None = None
    total_bens: float | None = None
    emails: list[str] | None = None
    sites: list[str] | None = None
    foto_url: str | None = None
    candidato_inapto: bool | None = None
    motivo_ficha_limpa: bool | None = None


class ResultadoCandidato(BaseModel):
    """Candidato com resultado eleitoral (totalização de votos)."""

    nome_urna: str | None = None
    numero: int | None = None
    partido: str | None = None
    total_votos: int | None = None
    percentual: str | None = None
    descricao_totalizacao: str | None = None


class BemCandidato(BaseModel):
    """Bem declarado pelo candidato."""

    ordem: int | None = None
    descricao: str | None = None
    tipo: str | None = None
    valor: float | None = None


class ResultadoCDN(BaseModel):
    """Candidato com resultado do CDN de resultados do TSE."""

    sequencia: str | None = None
    nome: str | None = None
    numero: str | None = None
    nome_vice: str | None = None
    coligacao: str | None = None
    votos: int | None = None
    percentual: str | None = None
    eleito: bool = False
    situacao: str | None = None
    validade_voto: str | None = None


class ResultadoRegiao(BaseModel):
    """Resultado de uma eleição em uma região (país, estado ou município)."""

    codigo: str | None = None
    tipo: str | None = None  # "br", "uf", "mu"
    uf: str | None = None
    data_eleicao: str | None = None
    total_secoes: int | None = None
    pct_apurado: str | None = None
    total_eleitores: int | None = None
    total_comparecimento: int | None = None
    total_abstencoes: int | None = None
    candidatos: list[ResultadoCDN] = []


class MunicipioEleitoral(BaseModel):
    """Município com códigos eleitorais TSE e IBGE."""

    codigo_tse: str | None = None
    codigo_ibge: str | None = None
    nome: str | None = None
    capital: bool = False
    uf: str | None = None


class PrestaContas(BaseModel):
    """Resumo da prestação de contas de campanha."""

    candidato_id: str | None = None
    nome: str | None = None
    partido: str | None = None
    cnpj: str | None = None
    total_recebido: float | None = None
    total_despesas: float | None = None
    total_bens: float | None = None
    limite_gastos: float | None = None
    divida_campanha: str | None = None
    sobra_financeira: str | None = None
    total_receita_pf: float | None = None
    total_receita_pj: float | None = None
    total_fundo_partidario: float | None = None
    total_fundo_especial: float | None = None
