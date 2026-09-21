"""Static reference data for the Megasoft feature."""

from __future__ import annotations

import json


def endpoints_disponiveis() -> str:
    """Endpoints disponíveis nos portais municipais Megasoft.

    Lista os módulos de receitas e despesas expostos pela API do portal,
    com os filtros aceitos e o tipo de resposta.
    """
    endpoints = [
        {
            "modulo": "Empenhos",
            "descricao": "Despesas empenhadas por fornecedor, órgão e natureza",
            "filtros": ["dataInicial", "dataFinal", "pagina", "tamanhoDaPagina"],
            "paginacao": True,
        },
        {
            "modulo": "Ordens de Pagamento",
            "descricao": "Pagamentos realizados a fornecedores, com empenho de origem",
            "filtros": ["dataInicial", "dataFinal", "pagina", "tamanhoDaPagina"],
            "paginacao": True,
        },
        {
            "modulo": "Totais de Ordens de Pagamento",
            "descricao": "Consolidado financeiro do período (pago, anulado, retido, líquido)",
            "filtros": ["dataInicial", "dataFinal"],
            "paginacao": False,
        },
        {
            "modulo": "Receitas",
            "descricao": "Arrecadação municipal por receita, natureza e fonte de recurso",
            "filtros": ["dataInicial", "dataFinal", "pagina", "tamanhoDaPagina"],
            "paginacao": True,
        },
    ]
    return json.dumps(endpoints, ensure_ascii=False, indent=2)
