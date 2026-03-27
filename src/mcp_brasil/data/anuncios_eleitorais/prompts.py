"""Analysis prompts for the Anuncios Eleitorais feature.

Prompts are reusable templates that guide LLM interactions.
They instruct the LLM on which tools to call and how to analyze the data.

In Claude Desktop, prompts appear as selectable options (similar to slash commands).
"""

from __future__ import annotations


def analise_candidato(nome_candidato: str, pagina_id: str = "") -> str:
    """Análise completa dos anúncios eleitorais de um candidato ou partido.

    Gera uma análise detalhada dos gastos, alcance e estratégia de
    anúncios eleitorais na Biblioteca de Anúncios da Meta.

    Args:
        nome_candidato: Nome do candidato ou partido para analisar.
        pagina_id: ID da página do Facebook (opcional, melhora precisão).
    """
    instrucoes = (
        f"Faça uma análise completa dos anúncios eleitorais de '{nome_candidato}' "
        "na Biblioteca de Anúncios da Meta:\n\n"
    )

    if pagina_id:
        instrucoes += (
            f"1. Use `buscar_anuncios_por_pagina` com o ID [{pagina_id}] "
            "para obter os anúncios da página\n"
        )
    else:
        instrucoes += (
            f"1. Use `buscar_anuncios_eleitorais` com o termo '{nome_candidato}' "
            "para encontrar anúncios relevantes\n"
        )

    instrucoes += (
        f"2. Use `analisar_demografia_anuncios` com '{nome_candidato}' para "
        "entender o público-alvo\n"
        "3. Apresente uma análise com:\n"
        "   - Quantidade de anúncios encontrados\n"
        "   - Faixa total de gastos\n"
        "   - Plataformas mais utilizadas (Facebook, Instagram, etc.)\n"
        "   - Perfil demográfico do público alcançado (idade e gênero)\n"
        "   - Estados com maior alcance\n"
        "   - Principais temas/mensagens dos anúncios\n"
        "   - Período de maior atividade\n"
        "4. Destaque padrões relevantes na estratégia de comunicação"
    )
    return instrucoes


def panorama_eleitoral(estado: str = "", periodo_inicio: str = "", periodo_fim: str = "") -> str:
    """Panorama dos anúncios eleitorais em um estado ou no Brasil.

    Gera uma visão geral dos anúncios políticos veiculados,
    incluindo maiores anunciantes e temas predominantes.

    Args:
        estado: Nome do estado para filtrar (ex: 'São Paulo'). Vazio = Brasil todo.
        periodo_inicio: Data de início no formato YYYY-mm-dd (opcional).
        periodo_fim: Data de fim no formato YYYY-mm-dd (opcional).
    """
    local = f"no estado de {estado}" if estado else "no Brasil"
    instrucoes = f"Gere um panorama dos anúncios eleitorais {local}:\n\n"

    if estado:
        instrucoes += f"1. Use `buscar_anuncios_por_regiao` com região ['{estado}'] "
    else:
        instrucoes += "1. Use `buscar_anuncios_eleitorais` com termos amplos "

    if periodo_inicio:
        instrucoes += f"filtrando a partir de {periodo_inicio} "
    if periodo_fim:
        instrucoes += f"até {periodo_fim} "

    instrucoes += (
        "para obter uma amostra de anúncios\n"
        "2. Analise os resultados e identifique:\n"
        "   - Páginas/candidatos com mais anúncios\n"
        "   - Temas e palavras-chave predominantes\n"
        "   - Faixas de gasto mais comuns\n"
        "   - Plataformas mais utilizadas\n"
        "3. Use `analisar_demografia_anuncios` para entender o perfil do público\n"
        "4. Apresente um resumo executivo com:\n"
        "   - Visão geral da atividade publicitária política\n"
        "   - Top anunciantes por volume e gasto\n"
        "   - Principais temas abordados\n"
        "   - Tendências observadas no período"
    )
    return instrucoes


def comparar_candidatos(candidato_a: str, candidato_b: str) -> str:
    """Comparação de estratégia de anúncios entre dois candidatos ou partidos.

    Analisa diferenças em gastos, alcance, público-alvo e mensagens.

    Args:
        candidato_a: Nome do primeiro candidato ou partido.
        candidato_b: Nome do segundo candidato ou partido.
    """
    return (
        f"Compare os anúncios eleitorais de '{candidato_a}' e '{candidato_b}':\n\n"
        f"1. Use `buscar_anuncios_eleitorais` com '{candidato_a}' (limit=50)\n"
        f"2. Use `buscar_anuncios_eleitorais` com '{candidato_b}' (limit=50)\n"
        f"3. Use `analisar_demografia_anuncios` para '{candidato_a}'\n"
        f"4. Use `analisar_demografia_anuncios` para '{candidato_b}'\n"
        "5. Compare e apresente:\n"
        "   - Quantidade de anúncios de cada um\n"
        "   - Comparação de gastos (faixas)\n"
        "   - Diferenças no público-alvo (idade, gênero)\n"
        "   - Diferenças na distribuição regional\n"
        "   - Plataformas preferidas por cada um\n"
        "   - Diferenças nos temas e tom das mensagens\n"
        "   - Qual tem maior alcance estimado\n"
        "6. Conclua com insights sobre as estratégias de cada candidato"
    )
