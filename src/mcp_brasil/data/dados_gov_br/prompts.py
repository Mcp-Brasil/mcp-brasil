"""Analysis prompts for the dados_gov_br feature."""

from __future__ import annotations


def explorar_dados(tema: str) -> str:
    """Explora dados abertos sobre um tema específico.

    Args:
        tema: Tema de interesse (ex: saúde, educação, meio ambiente).
    """
    return (
        f"Explore os dados abertos disponíveis sobre '{tema}'.\n\n"
        "Passos:\n"
        f"1. Use buscar_conjuntos(nome='{tema}') para encontrar datasets\n"
        "2. Para cada dataset relevante, use detalhar_conjunto(conjunto_id=...) "
        "para ver detalhes incluindo recursos\n"
        "3. Analise os formatos disponíveis e links de download\n\n"
        "Apresente um relatório com:\n"
        "- Datasets mais relevantes\n"
        "- Organizações publicadoras\n"
        "- Formatos disponíveis\n"
        "- Frequência de atualização"
    )


def listar_dados_orgao(orgao: str) -> str:
    """Lista datasets publicados por um órgão específico.

    Args:
        orgao: Nome do órgão (ex: IBGE, Ministério da Saúde).
    """
    return (
        f"Liste os conjuntos de dados publicados por '{orgao}'.\n\n"
        "Passos:\n"
        f"1. Use listar_organizacoes(nome='{orgao}') para encontrar o ID do órgão\n"
        "2. Use buscar_conjuntos(id_organizacao=...) para listar seus datasets\n"
        "3. Para cada dataset relevante, mostre título, formatos e atualização"
    )


def panorama_portal() -> str:
    """Apresenta um panorama geral do Portal de Dados Abertos."""
    return (
        "Apresente um panorama do Portal Brasileiro de Dados Abertos.\n\n"
        "Passos:\n"
        "1. Use listar_temas() para mostrar os grupos temáticos\n"
        "2. Use listar_organizacoes() para listar os principais órgãos publicadores\n"
        "3. Use listar_formatos() para mostrar os formatos disponíveis\n\n"
        "Apresente um resumo executivo com os números do portal."
    )


def descobrir_fonte(assunto: str) -> str:
    """Descobre fontes de dados abertas para um assunto.

    Args:
        assunto: Assunto ou tema de pesquisa.
    """
    return (
        f"Descubra fontes de dados abertos sobre '{assunto}'.\n\n"
        "Passos:\n"
        f"1. Use buscar_conjuntos(nome='{assunto}') para buscar datasets\n"
        "2. Identifique as organizações publicadoras\n"
        "3. Use detalhar_conjunto() nos mais relevantes para ver os recursos\n"
        "4. Sugira qual ferramenta do mcp-brasil usar para acesso direto "
        "(ex: ibge_*, bacen_*, opendatasus_*)\n\n"
        "Apresente:\n"
        "- Datasets encontrados com URLs de download\n"
        "- Organização responsável\n"
        "- Feature do mcp-brasil para acesso direto (se existir)"
    )
