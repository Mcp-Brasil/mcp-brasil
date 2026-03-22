"""Prompts for the Câmara feature — analysis templates for LLMs."""

from __future__ import annotations


def acompanhar_proposicao(sigla_tipo: str, numero: int, ano: int) -> str:
    """Gera um acompanhamento completo de uma proposição legislativa.

    Cria um template que orienta o LLM a consultar dados da proposição,
    tramitação e votações na Câmara dos Deputados.

    Args:
        sigla_tipo: Tipo da proposição (ex: PL, PEC, MPV).
        numero: Número da proposição.
        ano: Ano da proposição.
    """
    prop = f"{sigla_tipo} {numero}/{ano}"
    return (
        f"Faça um acompanhamento completo da proposição {prop} "
        "usando os dados da Câmara dos Deputados.\n\n"
        "Passos:\n"
        f"1. Use buscar_proposicao(sigla_tipo='{sigla_tipo}', numero={numero}, "
        f"ano={ano}) para encontrar a proposição\n"
        "2. Com o ID da proposição, use consultar_tramitacao(proposicao_id=ID) "
        "para ver o histórico de tramitação\n"
        "3. Use buscar_votacao(proposicao_id=ID) para verificar se houve votações\n"
        "4. Se houver votação, use votos_nominais(votacao_id=ID) "
        "para ver como cada deputado votou\n\n"
        "Apresente:\n"
        f"- Resumo da {prop}: ementa e situação atual\n"
        "- Histórico de tramitação (principais eventos)\n"
        "- Resultado das votações (se houver)\n"
        "- Placar por partido (se houver votação nominal)\n"
        "- Próximos passos previstos na tramitação"
    )


def perfil_deputado(deputado_id: int) -> str:
    """Gera um perfil completo de um deputado federal.

    Cria um template com dados pessoais, votações e despesas do deputado.

    Args:
        deputado_id: ID do deputado na API da Câmara.
    """
    return (
        f"Monte um perfil completo do deputado ID {deputado_id} "
        "usando os dados da Câmara dos Deputados.\n\n"
        "Passos:\n"
        f"1. Use buscar_deputado(deputado_id={deputado_id}) "
        "para obter os dados básicos\n"
        f"2. Use despesas_deputado(deputado_id={deputado_id}) "
        "para verificar os gastos de cota parlamentar\n\n"
        "Apresente:\n"
        "- Dados do deputado: nome, partido, UF, legislatura\n"
        "- Gastos de cota parlamentar: total, principais categorias\n"
        "- Principais fornecedores de serviços\n"
        "- Análise dos gastos: compatível com a média?"
    )


def analise_votacao(votacao_id: str) -> str:
    """Gera uma análise detalhada de uma votação na Câmara.

    Cria um template que analisa os votos nominais por partido e região.

    Args:
        votacao_id: ID da votação na API da Câmara.
    """
    return (
        f"Analise detalhadamente a votação {votacao_id} "
        "na Câmara dos Deputados.\n\n"
        "Passos:\n"
        f"1. Use votos_nominais(votacao_id='{votacao_id}') "
        "para obter todos os votos individuais\n\n"
        "Apresente:\n"
        "- Resultado geral: aprovada ou rejeitada, total de votos Sim/Não/Abstenção\n"
        "- Análise por partido: como cada partido votou (% Sim vs Não)\n"
        "- Análise por região: como cada UF/região votou\n"
        "- Destaques: votos dissidentes (deputados que votaram contra o partido)\n"
        "- Governismo vs oposição: qual foi o comportamento de cada bloco?"
    )
