"""Pydantic schemas for the TCM-GO feature."""

from __future__ import annotations

from pydantic import BaseModel

# ---------------------------------------------------------------------------
# eSiproc — valores de campo
# ---------------------------------------------------------------------------


class ValorCampo(BaseModel):
    """Valor distinto de um campo do eSiproc, com a contagem de processos."""

    valor: str | None = None
    quantidade: int | None = None


# ---------------------------------------------------------------------------
# eSiproc — processos
# ---------------------------------------------------------------------------


class Envolvido(BaseModel):
    """Pessoa ou órgão envolvido em um processo do eSiproc."""

    nome: str | None = None
    tipo_envolvimento: str | None = None


class Decisao(BaseModel):
    """Decisão registrada em um processo do eSiproc."""

    nome: str | None = None
    arquivo: str | None = None
    data: str | None = None


class Processo(BaseModel):
    """Processo eletrônico do eSiproc (metadados de trâmite)."""

    numero: str | None = None
    municipio: str | None = None
    unidade_gestora: str | None = None
    poder: str | None = None
    tipo_protocolo: str | None = None
    ano_competencia: int | None = None
    periodo_competencia: str | None = None
    relator: str | None = None
    procurador_contas: str | None = None
    st_fase: str | None = None
    finalizado: str | None = None
    passo: str | None = None
    data_protocolo: str | None = None
    data_ultima_tramitacao: str | None = None
    data_finalizacao: str | None = None
    envolvidos: list[Envolvido] = []
    decisoes: list[Decisao] = []


class ProcessoResultado(BaseModel):
    """Resultado paginado da busca de processos no eSiproc."""

    total: int = 0
    pagina: int = 0
    processos: list[Processo] = []


# ---------------------------------------------------------------------------
# TCMJuris — acórdãos
# ---------------------------------------------------------------------------


class Acordao(BaseModel):
    """Acórdão publicado pelo TCM-GO, com link do PDF do julgado."""

    numero_acordao: str | None = None
    processo: str | None = None
    data_julgamento: str | None = None
    relator: str | None = None
    municipio: str | None = None
    assunto: str | None = None
    url_pdf: str | None = None


class AcordaoResultado(BaseModel):
    """Resultado da busca de acórdãos no TCMJuris."""

    total_servidor: int = 0
    total_apos_refiltro: int = 0
    truncado: bool = False
    acordaos: list[Acordao] = []
