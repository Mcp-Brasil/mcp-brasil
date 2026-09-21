"""Analysis prompts for the Megasoft feature."""

from __future__ import annotations


def analisar_municipio_megasoft(municipio: str, exercicio: str) -> str:
    """Análise de receitas e despesas de um município no portal Megasoft.

    Consolida empenhos, ordens de pagamento, totais financeiros e receitas
    de um exercício para apoiar a leitura das contas públicas municipais.

    Args:
        municipio: Subdomínio do portal Megasoft (ex: "serranopolis").
        exercicio: Ano do exercício (ex: "2024").
    """
    return (
        f"Analise as contas públicas do município {municipio} no exercício {exercicio}:\n\n"
        f"1. Use `totais_ordens_pagamento` com data_inicial='01/01/{exercicio}' e "
        f"data_final='31/12/{exercicio}' para o consolidado financeiro\n"
        f"2. Use `buscar_empenhos` para listar as despesas empenhadas\n"
        f"3. Use `buscar_ordens_pagamento` para verificar os pagamentos realizados\n"
        f"4. Use `buscar_receitas` para analisar a arrecadação municipal\n\n"
        "Apresente um resumo consolidado com:\n"
        "- Total empenhado, pago e a diferença entre ambos\n"
        "- Principais fornecedores e órgãos por volume de despesa\n"
        "- Composição da arrecadação por natureza e fonte de recurso\n"
    )
