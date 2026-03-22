"""Analysis prompts for the Diário Oficial feature."""

from __future__ import annotations


def investigar_empresa(nome_empresa: str, cidade: str = "") -> str:
    """Investiga menções de uma empresa em diários oficiais municipais.

    Args:
        nome_empresa: Nome da empresa ou CNPJ para investigar.
        cidade: Nome da cidade para filtrar (opcional).
    """
    passos = f"Investigue a empresa '{nome_empresa}' nos diários oficiais municipais.\n\nPassos:\n"
    if cidade:
        passos += (
            f"1. Use buscar_cidades(nome='{cidade}') para obter o código IBGE\n"
            f"2. Use buscar_diarios(texto='{nome_empresa}', territorio_id=<código>) "
            "para buscar menções\n"
        )
    else:
        passos += f"1. Use buscar_diarios(texto='{nome_empresa}') para buscar menções\n"
    passos += (
        "\nAnalise os resultados procurando:\n"
        "- Contratos e licitações\n"
        "- Sanções e penalidades\n"
        "- Nomeações e exonerações\n"
        "- Licenças e alvarás\n\n"
        "Apresente um relatório com os achados mais relevantes."
    )
    return passos
