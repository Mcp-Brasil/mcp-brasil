"""Analysis prompts for the TCU feature."""

from __future__ import annotations


def analise_acordaos() -> str:
    """Análise dos acórdãos mais recentes do TCU.

    Gera um resumo analítico das decisões recentes do tribunal.
    """
    return (
        "Faça uma análise dos acórdãos mais recentes do TCU:\n\n"
        "1. Use consultar_acordaos(quantidade=20) para obter os últimos acórdãos\n"
        "2. Agrupe por colegiado (Plenário, 1ª Câmara, 2ª Câmara)\n"
        "3. Identifique os relatores mais frequentes\n"
        "4. Resuma os principais temas dos sumários\n"
        "5. Apresente um panorama geral das decisões recentes"
    )


def verificar_empresa(cnpj: str) -> str:
    """Verificação completa de uma empresa nos cadastros do TCU.

    Consulta certidões consolidadas e lista de inidôneos.

    Args:
        cnpj: CNPJ da empresa (somente números).
    """
    return (
        f"Faça uma verificação completa da empresa CNPJ {cnpj} nos cadastros do TCU:\n\n"
        f"1. Use consultar_certidoes(cnpj='{cnpj}') para verificar situação "
        "em 4 cadastros (TCU, CNJ, CEIS, CNEP)\n"
        f"2. Use consultar_inidoneos(cpf_cnpj='{cnpj}') para verificar "
        "se está na lista de inidôneos\n"
        "3. Apresente um resumo consolidado da situação da empresa\n"
        "4. Destaque qualquer restrição encontrada"
    )
