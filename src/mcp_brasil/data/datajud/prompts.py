"""Prompts for the DataJud feature — analysis templates for LLMs."""

from __future__ import annotations


def analise_processo(numero_processo: str, tribunal: str = "tjsp") -> str:
    """Gera uma análise completa de um processo judicial.

    Orienta o LLM a consultar dados do processo, partes e movimentações.

    Args:
        numero_processo: Número do processo (NPU).
        tribunal: Sigla do tribunal (ex: tjsp, trf1, stj).
    """
    return (
        f"Faça uma análise completa do processo {numero_processo} "
        f"no {tribunal.upper()} usando os dados do DataJud.\n\n"
        "Passos:\n"
        f"1. Use buscar_processo_por_numero(numero_processo='{numero_processo}', "
        f"tribunal='{tribunal}') para obter detalhes\n"
        f"2. Use consultar_movimentacoes(numero_processo='{numero_processo}', "
        f"tribunal='{tribunal}') para o histórico\n\n"
        "Apresente:\n"
        "- Resumo: classe, assunto, órgão julgador\n"
        "- Partes envolvidas (polo ativo e passivo)\n"
        "- Cronologia das movimentações relevantes\n"
        "- Situação atual do processo\n"
        "- Observações sobre prazos ou próximos passos"
    )


def pesquisa_juridica(tema: str, tribunal: str = "tjsp") -> str:
    """Gera uma pesquisa jurídica sobre um tema.

    Orienta o LLM a buscar processos relacionados a um tema específico.

    Args:
        tema: Tema jurídico para pesquisar.
        tribunal: Sigla do tribunal. Default: tjsp.
    """
    return (
        f"Faça uma pesquisa jurídica sobre '{tema}' "
        f"no {tribunal.upper()} usando o DataJud.\n\n"
        "Passos:\n"
        f"1. Use buscar_processos(query='{tema}', tribunal='{tribunal}') "
        "para encontrar processos relevantes\n"
        f"2. Use buscar_processos_por_assunto(assunto='{tema}', "
        f"tribunal='{tribunal}') para filtrar por assunto\n"
        "3. Para processos relevantes, use buscar_processo_por_numero() "
        "para obter detalhes\n\n"
        "Apresente:\n"
        "- Quantos processos foram encontrados\n"
        "- Principais classes processuais usadas\n"
        "- Órgãos julgadores mais frequentes\n"
        "- Resumo dos processos mais relevantes\n"
        "- Tendências observadas (se houver)"
    )


# --- Prompts MPU ---


def analisar_mpus(tribunal: str = "tjsp", periodo: str = "2024") -> str:
    """Análise de Medidas Protetivas de Urgência em um tribunal.

    Orienta o LLM a gerar um relatório analítico sobre MPUs, combinando
    estatísticas agregadas com detalhes processuais.

    Args:
        tribunal: Sigla do tribunal (ex: tjsp, tjpi, tjrj).
        periodo: Ano ou período para análise (ex: 2024).
    """
    return (
        f"Analise as Medidas Protetivas de Urgência no {tribunal.upper()} "
        f"para o período de {periodo}.\n\n"
        "Passos:\n"
        f"1. Use estatisticas_mpu(tribunal='{tribunal}', ano={periodo}) "
        "para obter totais e distribuições\n"
        f"2. Use buscar_mpu_concedidas(tribunal='{tribunal}', "
        f"data_inicio='{periodo}-01-01', data_fim='{periodo}-12-31', "
        "destinatario='mulher') para ver concessões\n"
        f"3. Use buscar_medidas_protetivas(tribunal='{tribunal}', "
        f"data_inicio='{periodo}-01-01', data_fim='{periodo}-12-31') "
        "para visão geral\n\n"
        "Apresente:\n"
        "- Volume total de MPUs e comparação com períodos anteriores\n"
        "- Taxa de concessão vs denegação\n"
        "- Tipos de medida mais comuns\n"
        "- Distribuição por varas/comarcas (se disponível)\n"
        "- Observações sobre padrões ou tendências\n\n"
        "**Nota:** A API pública do DataJud não expõe dados de partes. "
        "A análise é estatística/processual."
    )


def monitorar_mpu(tribunal: str = "tjsp", comarca: str | None = None) -> str:
    """Monitoramento de MPUs ativas em um tribunal ou comarca.

    Orienta o LLM a identificar MPUs recentes, revogações e padrões.

    Args:
        tribunal: Sigla do tribunal.
        comarca: Nome da comarca/vara (opcional).
    """
    filtro = f" na comarca '{comarca}'" if comarca else ""
    return (
        f"Monitore as Medidas Protetivas de Urgência ativas no "
        f"{tribunal.upper()}{filtro}.\n\n"
        "Passos:\n"
        f"1. Use buscar_medidas_protetivas(tribunal='{tribunal}') "
        "para MPUs recentes\n"
        f"2. Use buscar_mpu_concedidas(tribunal='{tribunal}', "
        "destinatario='mulher') para concessões a mulheres\n"
        f"3. Use estatisticas_mpu(tribunal='{tribunal}') para visão geral\n"
        "4. Para processos específicos, use timeline_mpu(numero_processo, "
        f"tribunal='{tribunal}') para ver a sequência de decisões\n\n"
        "Identifique:\n"
        "- MPUs concedidas recentemente\n"
        "- Revogações recentes (pode indicar resolução ou risco)\n"
        "- Prorrogações (medidas estendidas)\n"
        "- Padrões de reincidência (se detectáveis pela numeração)\n\n"
        "**Nota:** Dados de partes não são acessíveis pela API pública."
    )
