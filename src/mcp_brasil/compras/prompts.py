"""Analysis prompts for the Compras feature."""

from __future__ import annotations


def investigar_fornecedor(cnpj: str) -> str:
    """Investiga um fornecedor em contratações públicas.

    Args:
        cnpj: CNPJ do fornecedor a investigar.
    """
    return (
        f"Investigue o fornecedor com CNPJ {cnpj} em contratações públicas.\n\n"
        "Passos:\n"
        f"1. Use buscar_contratos(cnpj_fornecedor='{cnpj}') para ver contratos\n"
        f"2. Use buscar_contratacoes(texto='{cnpj}') para ver licitações\n\n"
        "Apresente um relatório com:\n"
        "- Total de contratos e valores\n"
        "- Órgãos contratantes\n"
        "- Objetos mais frequentes\n"
        "- Período de atuação"
    )
