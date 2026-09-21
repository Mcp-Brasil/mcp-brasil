"""Resources for the MapBiomas Alerta feature — static reference for LLM context."""

from __future__ import annotations

import json

from .constants import (
    AUTH_ENV_EMAIL,
    AUTH_ENV_PASSWORD,
    AUTH_SIGNUP_URL,
    DATE_TYPES,
    GRAPHQL_URL,
    MAX_LIMIT,
    SORT_DIRECTIONS,
    SORT_FIELDS,
    TERRITORY_CATEGORIES,
)


def info_api() -> str:
    """Informações gerais sobre a API do MapBiomas Alerta (autenticação, limites)."""
    data = {
        "nome": "MapBiomas Alerta — API GraphQL v2",
        "url_base": GRAPHQL_URL,
        "autenticacao": (
            "Requer login: mutation signIn com e-mail e senha, que devolve um "
            "token Bearer. Configure as variáveis de ambiente "
            f"{AUTH_ENV_EMAIL} e {AUTH_ENV_PASSWORD}."
        ),
        "cadastro": AUTH_SIGNUP_URL,
        "formato": "GraphQL (POST com query e variáveis)",
        "pagina_maxima": MAX_LIMIT,
        "campos_ordenacao": list(SORT_FIELDS),
        "direcoes_ordenacao": list(SORT_DIRECTIONS),
        "tipos_data": list(DATE_TYPES),
        "cobertura": "Alertas validados de supressão de vegetação nativa no Brasil",
        "aviso": (
            "Alerta é detecção por satélite validada por equipe técnica, não laudo "
            "pericial. As áreas de sobreposição não devem ser somadas."
        ),
    }
    return json.dumps(data, ensure_ascii=False)


def categorias_territorio() -> str:
    """Categorias de território aceitas no filtro de alertas, com rótulos."""
    data = [{"categoria": chave, "nome": nome} for chave, nome in TERRITORY_CATEGORIES.items()]
    return json.dumps(data, ensure_ascii=False)
