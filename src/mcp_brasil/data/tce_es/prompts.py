"""Analysis prompts for the TCE-ES feature."""

from __future__ import annotations


def analisar_gestao_es(unidade_gestora: str, ano: int) -> str:
    """Análise da gestão de uma unidade gestora capixaba no TCE-ES.

    Cruza contratações, licitações e obras para avaliar
    a gestão de compras públicas da unidade no período.

    Args:
        unidade_gestora: Nome da unidade gestora ou município
            (ex: "Vitória", "PREFEITURA DE CACHOEIRO").
        ano: Ano de referência (ex: 2024).
    """
    return (
        f"Analise a gestão de compras públicas de '{unidade_gestora}' no ano {ano}:\n\n"
        f"1. Use `buscar_contratacoes_municipios_es` com "
        f"q='{unidade_gestora}' e ano_referencia={ano}\n"
        f"2. Use `buscar_obras_es` com q='{unidade_gestora}' para mapear obras públicas\n"
        f"3. Apresente um resumo com:\n"
        "   - Número total de contratações e volume financeiro\n"
        "   - Distribuição por modalidade licitatória\n"
        "   - Principais fornecedores e objetos contratados\n"
        "   - Obras ativas com empresa e valor\n"
        "   - Alertas para contratações diretas de alto valor\n"
    )
