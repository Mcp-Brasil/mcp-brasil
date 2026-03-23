"""Static reference data for the TCE-SP feature."""

from __future__ import annotations

import json


def endpoints_tce_sp() -> str:
    """Endpoints disponíveis na API de Transparência do TCE-SP.

    Descreve os 3 endpoints JSON com parâmetros e formato de dados.
    """
    endpoints = [
        {
            "endpoint": "municipios",
            "descricao": "Lista dos 645 municípios paulistas com slug e nome",
            "parametros": [],
        },
        {
            "endpoint": "despesas/{municipio}/{exercicio}/{mes}",
            "descricao": "Despesas municipais: empenhos, pagamentos, liquidações e anulações",
            "parametros": ["municipio (slug)", "exercicio (2014+)", "mes (1-12)"],
            "eventos": ["Empenhado", "Valor Pago", "Valor Liquidado", "Anulação"],
        },
        {
            "endpoint": "receitas/{municipio}/{exercicio}/{mes}",
            "descricao": "Receitas municipais por fonte de recurso e classificação",
            "parametros": ["municipio (slug)", "exercicio (2014+)", "mes (1-12)"],
        },
    ]
    return json.dumps(endpoints, ensure_ascii=False, indent=2)
