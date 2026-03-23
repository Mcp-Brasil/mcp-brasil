"""Resources for the TransfereGov feature — static reference data for LLM context."""

from __future__ import annotations

import json


def info_api() -> str:
    """Informações sobre a API do TransfereGov e como usar."""
    data = {
        "nome": "API TransfereGov (PostgREST)",
        "url_base": "https://api.transferegov.gestao.gov.br",
        "autenticacao": "Nenhuma (API pública)",
        "formato": "JSON array (sem wrapper)",
        "paginacao": "limit/offset via query params",
        "filtros": {
            "descricao": "PostgREST: column=operator.value",
            "operadores": ["eq", "neq", "gt", "lt", "gte", "lte", "like", "ilike", "in"],
            "exemplos": [
                "ano_exercicio=eq.2024",
                "autor_emenda=ilike.*nome*",
                "uf_beneficiario=eq.PI",
            ],
        },
        "endpoint_principal": "/transferenciasespeciais",
        "colunas_principais": [
            "id_transferencia_especial",
            "ano_exercicio",
            "nr_emenda",
            "autor_emenda",
            "tipo_emenda",
            "valor_empenhado",
            "valor_liquidado",
            "valor_pago",
            "nm_municipio_beneficiario",
            "uf_beneficiario",
        ],
    }
    return json.dumps(data, ensure_ascii=False)
