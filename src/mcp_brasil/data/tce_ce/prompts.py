"""Analysis prompts for the TCE-CE feature."""

from __future__ import annotations


def analisar_municipio_ce(codigo_municipio: str, ano: int) -> str:
    """Análise de um município cearense no TCE-CE.

    Cruza licitações, contratos e empenhos para avaliar
    a gestão de compras e despesas do município.

    Args:
        codigo_municipio: Código SIM do município (ex: "057").
        ano: Ano de referência (ex: 2026).
    """
    return (
        f"Analise a gestão do município {codigo_municipio} no ano {ano}:\n\n"
        f'1. Use `buscar_licitacoes_ce` com data_realizacao="{ano}"\n'
        f'2. Use `buscar_contratos_ce` com data_contrato="{ano}" (traz o contratado)\n'
        f"3. Use `buscar_empenhos_ce` para meses-chave (ex: {ano}01, {ano}06, {ano}12)\n"
        "4. Apresente um resumo com:\n"
        "   - Número de licitações e contratos no período\n"
        "   - Principais fornecedores por volume\n"
        "   - Proporção de contratações diretas vs licitações\n"
        "   - Volume total empenhado\n"
        "O SIM recebe os dados mês a mês: o mês corrente costuma estar incompleto.\n"
    )
