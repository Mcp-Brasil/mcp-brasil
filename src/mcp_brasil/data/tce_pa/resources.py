"""Static reference data for the TCE-PA feature."""

from __future__ import annotations

import json

_PESQUISA_PARAMS = {
    "b": "string (obrigatório): slug da base de dados (ver abaixo)",
    "q": "string (opcional): busca textual, suporta ano:YYYY para filtrar por ano",
    "p": "integer (opcional): página (padrão: 1)",
    "rpp": "integer (opcional): resultados por página — 10, 15, 20 ou 25",
    "o": "string (opcional): relevancia | titulo | data",
    "or": "bool (opcional): True=decrescente, False=crescente",
}


def endpoints_tce_pa() -> str:
    """Catálogo de endpoints disponíveis na API de Dados Abertos e Pesquisa Integrada do TCE-PA."""
    endpoints = [
        {
            "endpoint": "/v1/diario_oficial",
            "base_url": "https://sistemas.tcepa.tc.br/dadosabertos/api",
            "descricao": "Publicações do Diário Oficial do TCE-PA a partir de 2018",
            "ferramenta": "buscar_diario_oficial_pa",
            "parametros": {
                "ano": "integer (obrigatório, padrão: 2018, mínimo: 2018)",
                "mes": "integer (opcional, 1-12)",
                "numero_publicacao": "integer (opcional)",
                "tipo_ato": "string (opcional)",
            },
            "tipo_ato_valores": [
                "Atos de Pessoal para Fins de Registro",
                "Atos e Normas",
                "Contratos",
                "Convênios e Instrumentos Congêneres",
                "Licitações",
                "Outros Atos de Pessoal",
            ],
        },
        {
            "endpoint": "/pesquisa/resultados",
            "base_url": "https://www.tcepa.tc.br/pesquisaintegrada",
            "descricao": "Pesquisa Integrada do TCE-PA — busca textual em múltiplas bases",
            "parametros": _PESQUISA_PARAMS,
            "bases": {
                "sessoes_plenarias": {
                    "ferramenta": "buscar_sessoes_plenarias_pa",
                    "slugs": {
                        "sessoes": "sessoes-plenarias",
                        "pautas": "pautas-sessoes-plenarias",
                        "atas": "atas-extratos-sessoes-plenarias",
                        "videos": "videos-sessoes-plenarias",
                    },
                    "documentos": {
                        "pautas": "URL do registro + /download → PDF",
                        "atas": "URL do registro + /download → PDF",
                        "videos": "URL do registro + /conteudo-original → YouTube",
                    },
                },
                "jurisprudencia": {
                    "ferramenta": "buscar_jurisprudencia_pa",
                    "slugs": {
                        "acordaos": "acordaos",
                        "acordaos-virtual": "acordaos-plenario-virtual",
                        "resolucoes": "resolucoes",
                        "portarias": "portarias-tcepa",
                        "atos": "atos",
                        "informativos": "informativos-jurisprudencia",
                        "prejulgados": "prejulgados",
                    },
                    "documentos": "URL do registro + /download → PDF",
                },
                "conteudo": {
                    "ferramenta": "buscar_conteudo_pa",
                    "slugs": {
                        "noticias": "noticias-portal-internet",
                        "acervo": "acervo-bibliografico",
                        "educacao": "acoes-educacionais",
                        "youtube": "canal-youtube-tce-pa",
                        "imagens": "banco-imagens",
                    },
                    "documentos": "URL do registro + /conteudo-original → conteúdo online",
                },
            },
        },
    ]
    return json.dumps(endpoints, ensure_ascii=False, indent=2)
