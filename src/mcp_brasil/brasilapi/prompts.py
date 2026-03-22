"""Analysis prompts for the BrasilAPI feature.

Prompts are reusable templates that guide LLM interactions.
They instruct the LLM on which tools to call and how to analyze the data.
"""

from __future__ import annotations


def analise_empresa(cnpj: str) -> str:
    """Análise completa de uma empresa brasileira pelo CNPJ.

    Gera um relatório com dados cadastrais, endereço e situação.

    Args:
        cnpj: CNPJ da empresa (com ou sem formatação).
    """
    return (
        f"Faça uma análise completa da empresa com CNPJ {cnpj}.\n\n"
        "Passos:\n"
        f"1. Use consultar_cnpj(cnpj='{cnpj}') para obter os dados cadastrais\n"
        "2. Com o CEP retornado, use consultar_cep para confirmar o endereço\n"
        "3. Com o DDD do telefone, use consultar_ddd para identificar a região\n\n"
        "Apresente um relatório organizado com:\n"
        "- Dados da empresa (razão social, fantasia, CNAE, porte)\n"
        "- Situação cadastral\n"
        "- Endereço completo\n"
        "- Capital social"
    )


def panorama_economico() -> str:
    """Panorama das principais taxas e indicadores econômicos."""
    return (
        "Monte um panorama econômico atual do Brasil.\n\n"
        "Passos:\n"
        "1. Use consultar_taxa(sigla='SELIC') para a taxa básica de juros\n"
        "2. Use consultar_taxa(sigla='CDI') para o CDI\n"
        "3. Use consultar_taxa(sigla='IPCA') para a inflação\n"
        "4. Use consultar_cotacao(moeda='USD', data=<ontem>) para o dólar\n"
        "5. Use consultar_cotacao(moeda='EUR', data=<ontem>) para o euro\n\n"
        "Apresente uma tabela comparativa com os indicadores e uma breve análise."
    )
