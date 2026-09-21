"""Static reference data for the TCM-GO feature."""

from __future__ import annotations

import json


def endpoints_disponiveis() -> str:
    """Superfícies públicas de busca do TCM-GO disponíveis nesta feature.

    Descreve o eSiproc (processos eletrônicos, metadados) e o TCMJuris
    (acórdãos em full-text, com link do PDF) e os filtros de cada um.
    """
    endpoints = [
        {
            "superficie": "eSiproc",
            "descricao": "Processos eletrônicos do TCM-GO (metadados de trâmite)",
            "tools": ["buscar_processos", "listar_valores_campo"],
            "filtros": [
                "municipio",
                "unidade_gestora",
                "tipo_protocolo",
                "ano",
                "relator",
                "finalizado",
                "numero",
            ],
            "paginacao": True,
            "full_text": False,
        },
        {
            "superficie": "TCMJuris",
            "descricao": "Acórdãos publicados pelo TCM-GO (inteiro teor, com link do PDF)",
            "tools": ["buscar_acordaos"],
            "filtros": [
                "pesquisa",
                "municipio",
                "assunto",
                "relator",
                "numero_processo",
                "data_sessao_ini",
                "data_sessao_fin",
            ],
            "paginacao": False,
            "full_text": True,
        },
    ]
    return json.dumps(endpoints, ensure_ascii=False, indent=2)
