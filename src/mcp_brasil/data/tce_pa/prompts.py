"""Analysis prompts for the TCE-PA feature."""

from __future__ import annotations


def explorar_conteudo_pa(query: str = "", ano: int | None = None) -> str:
    """Exploração de conteúdo informativo do TCE-PA.

    Guia o LLM para buscar notícias, acervo bibliográfico, ações educacionais,
    vídeos do YouTube e banco de imagens do TCE-PA.

    Args:
        query: Termo de busca (ex: "licitação", "capacitação", "fiscalização").
        ano: Ano de referência (opcional).
    """
    ano_param = f", ano={ano}" if ano else ""
    query_param = f", query='{query}'" if query else ""

    return (
        "Explore o conteúdo informativo do TCE-PA"
        + (f" sobre '{query}'" if query else "")
        + (f" de {ano}" if ano else "")
        + ".\n\n"
        "Siga estes passos:\n\n"
        f"1. Use `buscar_conteudo_pa` com base='noticias'{query_param}{ano_param}"
        " para buscar notícias do portal do TCE-PA.\n"
        f"2. Use `buscar_conteudo_pa` com base='educacao'{query_param}{ano_param}"
        " para ações educacionais e cursos oferecidos.\n"
        f"3. Use `buscar_conteudo_pa` com base='acervo'{query_param}{ano_param}"
        " para publicações e obras do acervo bibliográfico.\n"
        f"4. Use `buscar_conteudo_pa` com base='youtube'{query_param}{ano_param}"
        " para vídeos no canal oficial do TCE-PA.\n"
        f"5. Use `buscar_conteudo_pa` com base='imagens'{query_param}{ano_param}"
        " para o banco de imagens institucional.\n"
        "6. Para paginar, use o parâmetro pagina (padrão: 1, 20 por página).\n\n"
        "Apresente um resumo com:\n"
        "- Notícias e comunicados recentes relevantes ao tema\n"
        "- Cursos e eventos educacionais disponíveis\n"
        "- Publicações do acervo relacionadas ao tema\n"
        "- Links diretos para vídeos e materiais encontrados\n"
    )


def analisar_jurisprudencia_pa(query: str = "", ano: int | None = None) -> str:
    """Pesquisa de jurisprudência do TCE-PA.

    Guia o LLM para buscar acórdãos, resoluções, portarias, prejulgados
    e informativos do Tribunal de Contas do Estado do Pará.

    Args:
        query: Termo de busca (ex: "licitação", "dispensa", "pessoal").
        ano: Ano de referência (opcional).
    """
    ano_info = f" de {ano}" if ano else ""
    query_info = f" sobre '{query}'" if query else ""
    ano_param = f", ano={ano}" if ano else ""
    query_param = f", query='{query}'" if query else ""

    return (
        f"Pesquise jurisprudência{query_info}{ano_info} no TCE-PA.\n\n"
        "Siga estes passos:\n\n"
        f"1. Use `buscar_jurisprudencia_pa` com base='acordaos'{query_param}{ano_param}"
        " para buscar acórdãos do plenário.\n"
        f"2. Use `buscar_jurisprudencia_pa` com base='acordaos-virtual'{query_param}{ano_param}"
        " para acórdãos do plenário virtual.\n"
        f"3. Use `buscar_jurisprudencia_pa` com base='resolucoes'{query_param}{ano_param}"
        " para resoluções normativas.\n"
        f"4. Use `buscar_jurisprudencia_pa` com base='prejulgados'{query_param}{ano_param}"
        " para prejulgados (teses consolidadas).\n"
        f"5. Use `buscar_jurisprudencia_pa` com base='portarias'{query_param}{ano_param}"
        " para portarias do TCE-PA.\n"
        "6. Para paginar resultados, use o parâmetro pagina (padrão: 1, 20 por página).\n"
        "7. Os documentos PDF estão disponíveis via url_documento de cada resultado.\n\n"
        "Apresente um resumo com:\n"
        "- Acórdãos mais relevantes encontrados e suas decisões\n"
        "- Resoluções e portarias aplicáveis ao tema pesquisado\n"
        "- Prejulgados que consolidam entendimento sobre o tema\n"
        "- Links diretos para os PDFs dos documentos principais\n"
    )


def analisar_diario_oficial_pa(ano: int = 2024, mes: int | None = None) -> str:
    """Análise de publicações do Diário Oficial do TCE-PA.

    Guia o LLM para buscar e analisar atos publicados no Diário Oficial
    do Tribunal de Contas do Estado do Pará em um determinado período.

    Args:
        ano: Ano de referência (padrão: 2024, mínimo: 2018).
        mes: Mês específico (1-12, opcional — omita para o ano inteiro).
    """
    periodo = f"{mes:02d}/{ano}" if mes else str(ano)
    mes_param = f", mes={mes}" if mes else ""

    return (
        f"Analise as publicações do Diário Oficial do TCE-PA em {periodo}.\n\n"
        "Siga estes passos:\n\n"
        f"1. Use `buscar_diario_oficial_pa` com ano={ano}{mes_param}"
        " para obter todas as publicações.\n"
        "2. Use `buscar_diario_oficial_pa` com"
        " tipo_ato='Contratos' para filtrar contratos.\n"
        "3. Use `buscar_diario_oficial_pa` com"
        " tipo_ato='Licitações' para filtrar licitações.\n"
        "4. Use `buscar_diario_oficial_pa` com"
        " tipo_ato='Atos e Normas' para normas e resoluções.\n\n"
        "Apresente um resumo com:\n"
        "- Total de publicações por tipo de ato\n"
        "- Principais contratos e licitações publicados\n"
        "- Atos normativos relevantes (resoluções, portarias)\n"
        "- Atos de pessoal em destaque\n"
    )


def analisar_sessoes_plenarias_pa(ano: int = 2024) -> str:
    """Análise de sessões plenárias do TCE-PA.

    Guia o LLM para buscar e analisar sessões plenárias do Tribunal de
    Contas do Estado do Pará, incluindo pautas, atas e vídeos.

    Args:
        ano: Ano de referência (padrão: 2024).
    """
    return (
        f"Analise as sessões plenárias do TCE-PA em {ano}.\n\n"
        "Siga estes passos:\n\n"
        f"1. Use `buscar_sessoes_plenarias_pa` com tipo='sessoes', ano={ano}"
        " para listar todas as sessões do ano.\n"
        f"2. Use `buscar_sessoes_plenarias_pa` com tipo='pautas', ano={ano}"
        " para obter pautas com links para download dos PDFs.\n"
        f"3. Use `buscar_sessoes_plenarias_pa` com tipo='atas', ano={ano}"
        " para obter atas com links para download dos PDFs.\n"
        f"4. Use `buscar_sessoes_plenarias_pa` com tipo='videos', ano={ano}"
        " para obter links dos vídeos no YouTube.\n"
        "5. Para buscar sessões específicas, use o parâmetro query (ex: query='extraordinária').\n"
        "6. Para navegar além da primeira página, use o parâmetro pagina.\n\n"
        "Apresente um resumo com:\n"
        "- Total de sessões realizadas e distribuição por tipo\n"
        "  (ordinária, extraordinária, plenário virtual)\n"
        "- Frequência das sessões ao longo do ano\n"
        "- Disponibilidade de pautas, atas e vídeos por sessão\n"
    )
