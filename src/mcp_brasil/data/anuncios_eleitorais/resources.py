"""Static reference data for the Anuncios Eleitorais feature.

Resources are read-only data sources that clients can pull.
They provide context to LLMs without requiring tool calls.

Resources are registered with data:// URIs (without the feature namespace —
mount() adds the namespace prefix automatically).
"""

from __future__ import annotations

import json

from .constants import (
    ESTADOS_BRASILEIROS,
    FAIXAS_AUDIENCIA,
    PLATAFORMAS,
)


def estados_brasileiros() -> str:
    """Lista dos 27 estados brasileiros (sigla e nome) para filtros regionais."""
    estados = [
        {"sigla": sigla, "nome": nome} for sigla, nome in sorted(ESTADOS_BRASILEIROS.items())
    ]
    return json.dumps(estados, ensure_ascii=False, indent=2)


def parametros_busca() -> str:
    """Referência dos parâmetros de busca disponíveis na API de anúncios eleitorais."""
    params = {
        "search_terms": {
            "descricao": "Termos de busca (max 100 chars). Espaço = AND.",
            "exemplo": "educação saúde",
        },
        "search_page_ids": {
            "descricao": "IDs de páginas do Facebook (até 10).",
            "exemplo": ["123456789"],
        },
        "ad_active_status": {
            "descricao": "Status do anúncio.",
            "valores": ["ACTIVE", "INACTIVE", "ALL"],
        },
        "ad_delivery_date_min": {
            "descricao": "Data mínima de veiculação.",
            "formato": "YYYY-mm-dd",
        },
        "ad_delivery_date_max": {
            "descricao": "Data máxima de veiculação.",
            "formato": "YYYY-mm-dd",
        },
        "bylines": {
            "descricao": "Financiadores (campo 'Pago por'). Texto exato.",
            "exemplo": ["Partido X"],
        },
        "delivery_by_region": {
            "descricao": "Regiões/estados de entrega. Use nome completo.",
            "exemplo": ["São Paulo", "Rio de Janeiro"],
        },
        "estimated_audience_size_min": {
            "descricao": "Tamanho mínimo da audiência estimada.",
            "valores_permitidos": FAIXAS_AUDIENCIA,
        },
        "estimated_audience_size_max": {
            "descricao": "Tamanho máximo da audiência estimada.",
            "valores_permitidos": FAIXAS_AUDIENCIA,
        },
        "media_type": {
            "descricao": "Tipo de mídia do anúncio.",
            "valores": ["ALL", "IMAGE", "MEME", "VIDEO", "NONE"],
        },
        "publisher_platforms": {
            "descricao": "Plataformas onde o anúncio apareceu.",
            "valores": PLATAFORMAS,
        },
        "search_type": {
            "descricao": "Tipo de busca textual.",
            "valores": ["KEYWORD_UNORDERED", "KEYWORD_EXACT_PHRASE"],
        },
    }
    return json.dumps(params, ensure_ascii=False, indent=2)


def campos_disponiveis() -> str:
    """Referência dos campos de dados disponíveis nos resultados de anúncios eleitorais."""
    campos = {
        "basicos": {
            "id": "ID da biblioteca do anúncio",
            "page_id": "ID da página do Facebook",
            "page_name": "Nome da página do Facebook",
            "ad_creation_time": "Data/hora de criação (UTC)",
            "ad_delivery_start_time": "Início da veiculação",
            "ad_delivery_stop_time": "Fim da veiculação",
            "ad_snapshot_url": "URL para visualizar o anúncio",
            "ad_creative_bodies": "Textos do criativo",
            "ad_creative_link_titles": "Títulos dos links",
            "ad_creative_link_descriptions": "Descrições dos links",
            "ad_creative_link_captions": "Legendas dos links",
            "languages": "Idiomas do anúncio",
            "publisher_platforms": "Plataformas (Facebook, Instagram, etc.)",
        },
        "politicos_brasil": {
            "bylines": "Financiador (campo 'Pago por')",
            "currency": "Moeda do gasto (BRL)",
            "spend": "Gasto total (faixa: <100, 100-499, 500-999, ...)",
            "impressions": "Impressões (faixa: <1000, 1K-5K, ...)",
            "demographic_distribution": "Distribuição por idade e gênero (%)",
            "delivery_by_region": "Distribuição por estado (%)",
            "estimated_audience_size": "Tamanho estimado da audiência",
            "br_total_reach": "Alcance estimado no Brasil",
            "target_ages": "Faixas etárias do direcionamento",
            "target_gender": "Gênero do direcionamento",
            "target_locations": "Localizações do direcionamento",
            "age_country_gender_reach_breakdown": "Detalhamento alcance por idade/gênero",
        },
    }
    return json.dumps(campos, ensure_ascii=False, indent=2)
