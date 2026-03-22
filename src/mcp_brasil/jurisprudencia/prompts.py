"""Prompts for the Jurisprudência feature — analysis templates for LLMs."""

from __future__ import annotations


def pesquisa_jurisprudencial(tema: str, tribunais: str = "stf,stj,tst") -> str:
    """Gera uma pesquisa jurisprudencial completa sobre um tema.

    Orienta o LLM a buscar decisões em múltiplos tribunais superiores.

    Args:
        tema: Tema jurídico para pesquisar.
        tribunais: Tribunais para consultar (separados por vírgula). Default: stf,stj,tst.
    """
    tribs = [t.strip().upper() for t in tribunais.split(",")]
    steps = []
    step_num = 1
    for t in tribs:
        t_lower = t.lower()
        steps.append(
            f"{step_num}. Use buscar_jurisprudencia_{t_lower}(query='{tema}') para buscar no {t}"
        )
        step_num += 1

    steps.append(
        f"{step_num}. Use buscar_sumulas(tribunal='stf', query='{tema}') "
        "para verificar súmulas relacionadas"
    )
    step_num += 1
    steps.append(
        f"{step_num}. Use buscar_repercussao_geral(query='{tema}') para temas de repercussão geral"
    )

    return (
        f"Faça uma pesquisa jurisprudencial completa sobre '{tema}' "
        f"nos tribunais {', '.join(tribs)}.\n\n"
        "Passos:\n" + "\n".join(steps) + "\n\n"
        "Apresente:\n"
        "- Entendimento predominante em cada tribunal\n"
        "- Súmulas aplicáveis (se houver)\n"
        "- Temas de repercussão geral relacionados\n"
        "- Divergências entre tribunais (se houver)\n"
        "- Evolução jurisprudencial do tema"
    )


def analise_tema(numero_tema: int) -> str:
    """Gera uma análise de um tema de repercussão geral do STF.

    Args:
        numero_tema: Número do tema de repercussão geral.
    """
    return (
        f"Analise o Tema {numero_tema} de Repercussão Geral do STF.\n\n"
        "Passos:\n"
        f"1. Use buscar_repercussao_geral(tema={numero_tema}) "
        "para obter detalhes do tema\n"
        "2. Use buscar_jurisprudencia_stf(query='tema {numero_tema} "
        "repercussão geral') para decisões relacionadas\n\n"
        "Apresente:\n"
        "- Questão constitucional discutida\n"
        "- Tese fixada pelo STF\n"
        "- Leading case (processo paradigma)\n"
        "- Situação atual (pendente, julgado, em revisão)\n"
        "- Impacto prático da decisão"
    )
