"""Analysis prompts for the TCE-SP feature."""

from __future__ import annotations


def analisar_financas_municipio_sp(municipio: str, exercicio: int) -> str:
    """Análise financeira de um município paulista no TCE-SP.

    Cruza despesas e receitas mensais para avaliar a saúde financeira.

    Args:
        municipio: Slug do município (ex: "campinas").
        exercicio: Ano fiscal (ex: 2025).
    """
    return (
        f"Analise as finanças do município {municipio} no exercício {exercicio}:\n\n"
        "1. Use `consultar_receitas_sp` para cada mês e totalize por fonte de recurso\n"
        "2. Use `consultar_despesas_sp` para cada mês e totalize por evento\n"
        "3. Compare receitas vs despesas mês a mês\n"
        "4. Identifique:\n"
        "   - Meses com maior gasto\n"
        "   - Principais fornecedores por volume\n"
        "   - Proporção empenho vs pagamento efetivo\n"
        "   - Fontes de receita mais relevantes\n"
    )
