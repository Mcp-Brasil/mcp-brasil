"""Pydantic schemas for the Anuncios Eleitorais feature."""

from __future__ import annotations

from pydantic import BaseModel, Field


class FaixaValor(BaseModel):
    """Faixa de valor (min/max) usada para impressões e gastos."""

    lower_bound: str | None = Field(default=None, description="Limite inferior da faixa")
    upper_bound: str | None = Field(default=None, description="Limite superior da faixa")


class DistribuicaoDemografica(BaseModel):
    """Distribuição demográfica por idade e gênero."""

    age: str | None = Field(default=None, description="Faixa etária (ex: 18-24, 25-34)")
    gender: str | None = Field(default=None, description="Gênero (Male, Female, Unknown)")
    percentage: str | None = Field(default=None, description="Percentual do alcance")


class DistribuicaoRegional(BaseModel):
    """Distribuição regional do alcance do anúncio."""

    region: str | None = Field(default=None, description="Nome da região/estado")
    percentage: str | None = Field(default=None, description="Percentual do alcance")


class LocalizacaoAlvo(BaseModel):
    """Localização incluída ou excluída do direcionamento."""

    name: str | None = Field(default=None, description="Nome da localização")
    num_obfuscated: int | None = Field(
        default=None, description="Número de localizações ofuscadas"
    )
    excluded: bool | None = Field(default=None, description="Se é uma exclusão")


class AnuncioEleitoral(BaseModel):
    """Anúncio eleitoral/político da Biblioteca de Anúncios da Meta."""

    id: str = Field(description="ID da biblioteca do anúncio")
    ad_creation_time: str | None = Field(
        default=None, description="Data/hora de criação do anúncio (UTC)"
    )
    ad_creative_bodies: list[str] | None = Field(
        default=None, description="Textos do criativo do anúncio"
    )
    ad_creative_link_captions: list[str] | None = Field(
        default=None, description="Legendas dos links"
    )
    ad_creative_link_descriptions: list[str] | None = Field(
        default=None, description="Descrições dos links"
    )
    ad_creative_link_titles: list[str] | None = Field(
        default=None, description="Títulos dos links"
    )
    ad_delivery_start_time: str | None = Field(
        default=None, description="Data/hora de início da veiculação"
    )
    ad_delivery_stop_time: str | None = Field(
        default=None, description="Data/hora de parada da veiculação"
    )
    ad_snapshot_url: str | None = Field(
        default=None, description="URL para visualizar o anúncio arquivado"
    )
    bylines: str | None = Field(default=None, description="Nome do financiador do anúncio")
    currency: str | None = Field(default=None, description="Moeda (ISO)")
    spend: FaixaValor | None = Field(default=None, description="Gasto total (faixa)")
    impressions: FaixaValor | None = Field(default=None, description="Impressões (faixa)")
    demographic_distribution: list[DistribuicaoDemografica] | None = Field(
        default=None, description="Distribuição demográfica (idade/gênero)"
    )
    delivery_by_region: list[DistribuicaoRegional] | None = Field(
        default=None, description="Distribuição regional do alcance"
    )
    estimated_audience_size: FaixaValor | None = Field(
        default=None, description="Tamanho estimado da audiência"
    )
    br_total_reach: int | None = Field(default=None, description="Alcance estimado no Brasil")
    languages: list[str] | None = Field(default=None, description="Idiomas do anúncio")
    page_id: str | None = Field(default=None, description="ID da página do Facebook")
    page_name: str | None = Field(default=None, description="Nome da página do Facebook")
    publisher_platforms: list[str] | None = Field(
        default=None, description="Plataformas onde o anúncio apareceu"
    )
    target_ages: list[str] | None = Field(
        default=None, description="Faixas etárias do direcionamento"
    )
    target_gender: str | None = Field(
        default=None, description="Gênero do direcionamento (Women, Men, All)"
    )
    target_locations: list[LocalizacaoAlvo] | None = Field(
        default=None, description="Localizações do direcionamento"
    )
    age_country_gender_reach_breakdown: list[dict[str, object]] | None = Field(
        default=None, description="Detalhamento do alcance por idade/país/gênero"
    )


class CursorPaginacao(BaseModel):
    """Cursores para paginação da API."""

    before: str | None = None
    after: str | None = None


class Paginacao(BaseModel):
    """Dados de paginação da resposta."""

    cursors: CursorPaginacao | None = None
    next: str | None = Field(default=None, description="URL da próxima página")


class RespostaAnuncios(BaseModel):
    """Resposta paginada da API de anúncios."""

    data: list[AnuncioEleitoral] = Field(default_factory=list)
    paging: Paginacao | None = None
