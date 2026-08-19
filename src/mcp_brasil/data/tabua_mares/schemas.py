"""Pydantic schemas for the Tabua Mares feature."""

from __future__ import annotations

from pydantic import BaseModel, Field


class GeoLocalizacao(BaseModel):
    """Coordenadas geográficas de um porto."""

    lat: str
    lng: str
    decimal_lat: str = Field(description="Latitude em formato graus/minutos (ex: 09° 41' S)")
    decimal_lng: str = Field(description="Longitude em formato graus/minutos (ex: 35° 43'.5 W)")
    lat_direction: str
    lng_direction: str


class Porto(BaseModel):
    """Informações detalhadas de um porto."""

    id: str = Field(description="ID alfanumérico do porto (ex: 'pb01')")
    harbor_name: str
    state: str
    timezone: str
    card: str
    geo_location: list[GeoLocalizacao] = Field(default_factory=list)
    mean_level: float | None = Field(default=None, description="Nível médio do mar em metros")


class PortoResumo(BaseModel):
    """Resumo de um porto retornado pela listagem por estado."""

    id: str = Field(description="ID alfanumérico do porto (ex: 'pb01')")
    year: int
    harbor_name: str
    data_collection_institution: str


class HoraMare(BaseModel):
    """Registro de maré em uma hora específica."""

    hour: str = Field(description="Horário no formato HH:MM:SS")
    level: float = Field(description="Nível da maré em metros")


class DiaMare(BaseModel):
    """Dados de maré para um dia."""

    weekday_name: str
    day: int
    hours: list[HoraMare] = Field(default_factory=list)


class MesMare(BaseModel):
    """Dados de maré para um mês."""

    month_name: str
    month: int
    days: list[DiaMare] = Field(default_factory=list)


class TabuaMare(BaseModel):
    """Tábua de marés completa para um porto."""

    year: int
    harbor_name: str
    state: str
    timezone: str
    card: str
    data_collection_institution: str
    mean_level: float
    months: list[MesMare] = Field(default_factory=list)
