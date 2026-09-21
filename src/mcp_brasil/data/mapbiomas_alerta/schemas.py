"""Pydantic models for the MapBiomas Alerta feature."""

from __future__ import annotations

from pydantic import BaseModel


class Alerta(BaseModel):
    """Alerta validado de supressão de vegetação nativa.

    Um alerta é uma detecção por satélite validada por equipe técnica.
    As áreas de cruzamento (APP, Reserva Legal, UC, TI etc.) medem a
    sobreposição do polígono do alerta com cada camada e **não devem ser
    somadas** entre si.
    """

    codigo: int | None = None
    area_ha: float | None = None
    detectado_em: str | None = None
    publicado_em: str | None = None
    status: str | None = None
    status_em: str | None = None
    fontes: list[str] = []
    classes_desmatamento: list[str] = []
    velocidade_desmatamento: str | None = None
    republicado: bool | None = None

    biomas: list[str] = []
    biomas_area_ha: float | None = None
    estados: list[str] = []
    estados_area_ha: float | None = None
    municipios: list[str] = []
    municipios_area_ha: float | None = None

    app_area_ha: float | None = None
    app_sobreposicoes: int | None = None
    reserva_legal_area_ha: float | None = None
    reserva_legal_sobreposicoes: int | None = None
    unidades_conservacao: list[str] = []
    unidades_conservacao_area_ha: float | None = None
    terras_indigenas: list[str] = []
    terras_indigenas_area_ha: float | None = None
    quilombos: list[str] = []
    quilombos_area_ha: float | None = None
    assentamentos: list[str] = []
    assentamentos_area_ha: float | None = None
    bacias: list[str] = []
    bacias_area_ha: float | None = None
    nascentes_area_ha: float | None = None

    embargos_total: int | None = None
    embargos_area_ha: float | None = None
    autorizacoes_area_ha: float | None = None
    autorizacoes_categorias: list[str] = []
    autorizacoes_atividades: list[str] = []
    manejo_florestal_area_ha: float | None = None
    manejo_florestal_atividades: list[str] = []

    car_codigos: list[str] = []
    car_total: int | None = None

    imagem_antes_em: str | None = None
    imagem_depois_em: str | None = None
    datum: str | None = None
    bbox: str | None = None
    geometria_wkt: str | None = None


class ResultadoAlertas(BaseModel):
    """Página de alertas devolvida pela busca."""

    alertas: list[Alerta] = []
    total: int = 0
    total_paginas: int = 0
    pagina: int = 1


class Territorio(BaseModel):
    """Território (município, estado, UC, TI etc.) com o código usado nos filtros."""

    codigo: int
    nome: str
    categoria: str
    categoria_nome: str


class ImovelRural(BaseModel):
    """Imóvel rural do CAR e os alertas a ele associados."""

    codigo: str | None = None
    tipo: str | None = None
    uf: str | None = None
    area_ha: float | None = None
    car_atualizado_em: str | None = None
    versao: int | None = None
    alertas: list[Alerta] = []
    area_alertada_ha: float = 0.0
