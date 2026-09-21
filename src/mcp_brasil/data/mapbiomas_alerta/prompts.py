"""Prompts for the MapBiomas Alerta feature — analysis templates for LLMs."""

from __future__ import annotations


def analisar_alertas(territorio: str, periodo: str = "últimos 12 meses") -> str:
    """Orienta uma análise de alertas de supressão de vegetação nativa.

    Combina a listagem por território com o dossiê de cada alerta relevante.

    Args:
        territorio: Nome do município ou estado a analisar.
        periodo: Período de detecção a considerar (ex: "2024", "últimos 12 meses").
    """
    return (
        f"Analise os alertas de supressão de vegetação nativa em '{territorio}' "
        f"no período de {periodo}, usando os dados do MapBiomas Alerta.\n\n"
        "Passos:\n"
        f"1. Use listar_territorios(busca='{territorio}') para obter o codigo do território.\n"
        "2. Use buscar_alertas(territorio_id=...) com o período desejado para listar os alertas.\n"
        "3. Para cada alerta relevante, use consultar_alerta(codigo=...) para ver as "
        "sobreposições (APP, Reserva Legal, UC, TI) e os CAR cruzados.\n\n"
        "Apresente:\n"
        "- Total de alertas e soma das áreas no período\n"
        "- Distribuição por área e por data de detecção\n"
        "- Sobreposições com áreas protegidas e embargos\n"
        "- Imóveis do CAR cruzados\n\n"
        "**Lembrete:** alerta é detecção validada, não laudo pericial; as áreas de "
        "sobreposição não devem ser somadas; a ausência de alerta não prova ausência "
        "de supressão."
    )
