"""Tool functions for the tce_pa feature.

Rules (ADR-001):
    - tools.py NEVER makes HTTP directly — delegates to client.py
    - Returns formatted strings for LLM consumption
"""

from __future__ import annotations

from fastmcp import Context

from mcp_brasil._shared.formatting import markdown_table

from . import client
from .constants import BASES_CONTEUDO, BASES_JURISPRUDENCIA, TIPO_ATO, TIPO_SESSAO


async def buscar_diario_oficial_pa(
    ctx: Context,
    ano: int = 2018,
    mes: int | None = None,
    tipo_ato: str | None = None,
    numero_publicacao: int | None = None,
) -> str:
    """Busca publicações no Diário Oficial do TCE-PA.

    Pesquisa atos publicados no Diário Oficial do Tribunal de Contas
    do Estado do Pará. Os dados estão disponíveis a partir de 2018.

    Args:
        ano: Ano da publicação (padrão: 2018, mínimo: 2018).
        mes: Mês (1-12, opcional).
        tipo_ato: Tipo de ato para filtrar (opcional). Valores válidos:
            "Atos de Pessoal para Fins de Registro",
            "Atos e Normas", "Contratos",
            "Convênios e Instrumentos Congêneres",
            "Licitações", "Outros Atos de Pessoal".
        numero_publicacao: Número específico da publicação (opcional).

    Returns:
        Tabela com publicações encontradas.
    """
    await ctx.info(f"Buscando Diário Oficial do TCE-PA ({ano})...")
    publicacoes = await client.buscar_diario_oficial(
        ano=ano,
        mes=mes,
        numero_publicacao=numero_publicacao,
        tipo_ato=tipo_ato,
    )

    if not publicacoes:
        return "Nenhuma publicação encontrada para os filtros informados."

    await ctx.info(f"{len(publicacoes)} publicação(ões) encontrada(s)")

    tipos_validos = "\n".join(f"  - {v}" for v in TIPO_ATO.values())
    header = (
        f"Diário Oficial do TCE-PA — {len(publicacoes)} publicação(ões) encontrada(s):\n\n"
        f"**Tipos de ato disponíveis:**\n{tipos_validos}\n\n"
    )

    rows = [
        (
            str(p.numero_publicacao) if p.numero_publicacao else "—",
            p.data_publicacao or "—",
            p.tipo_ato or "—",
            (p.publicacao[:80] + "...")
            if p.publicacao and len(p.publicacao) > 80
            else (p.publicacao or "—"),
        )
        for p in publicacoes
    ]

    return header + markdown_table(
        ["Nº Publicação", "Data", "Tipo de Ato", "Conteúdo (resumo)"],
        rows,
    )


async def buscar_sessoes_plenarias_pa(
    ctx: Context,
    tipo: str = "sessoes",
    query: str = "",
    ano: int | None = None,
    mes: int | None = None,
    pagina: int = 1,
) -> str:
    """Busca sessões plenárias do TCE-PA via Pesquisa Integrada.

    Pesquisa sessões plenárias, pautas, atas/extratos ou vídeos do
    Tribunal de Contas do Estado do Pará.

    Args:
        tipo: Tipo de dado — "sessoes", "pautas", "atas" ou "videos".
        query: Texto de busca (opcional).
        ano: Filtro por ano (opcional).
        mes: Filtro por mês 1-12 (opcional, busca textual aproximada).
        pagina: Página de resultados (padrão: 1, 20 resultados/página).

    Returns:
        Tabela com sessões encontradas e links para documentos.
    """
    tipos_validos = list(TIPO_SESSAO)
    if tipo not in tipos_validos:
        return f"Tipo inválido: '{tipo}'. Use um de: {tipos_validos}"

    await ctx.info(f"Buscando {tipo} de sessões plenárias do TCE-PA (página {pagina})...")
    sessoes = await client.buscar_sessoes_plenarias(
        tipo=tipo,
        query=query,
        ano=ano,
        mes=mes,
        pagina=pagina,
    )

    if not sessoes:
        return "Nenhuma sessão encontrada para os filtros informados."

    await ctx.info(f"{len(sessoes)} resultado(s) encontrado(s)")

    tipos_info = ", ".join(tipos_validos)
    header = (
        f"**Sessões Plenárias TCE-PA — {tipo}**\n"
        f"Página {pagina} | {len(sessoes)} resultado(s)\n"
        f"Tipos disponíveis: {tipos_info}\n\n"
    )

    if tipo == "sessoes":
        rows = [
            (
                s.data_sessao or "—",
                (s.titulo or "—")[:55],
                s.tipo_sessao or "—",
                str(s.ano or "—"),
                s.url_pagina or "—",
            )
            for s in sessoes
        ]
        return header + markdown_table(
            ["Data", "Título", "Tipo", "Ano", "URL"],
            rows,
        )

    rows_doc = [
        (
            s.data_sessao or "—",
            (s.titulo or "—")[:55],
            s.tipo_sessao or "—",
            s.url_documento or s.url_pagina or "—",
        )
        for s in sessoes
    ]
    col_doc = "Vídeo (YouTube)" if tipo == "videos" else "Download PDF"
    return header + markdown_table(
        ["Data", "Título", "Tipo", col_doc],
        rows_doc,
    )


async def buscar_jurisprudencia_pa(
    ctx: Context,
    base: str = "acordaos",
    query: str = "",
    ano: int | None = None,
    mes: int | None = None,
    pagina: int = 1,
) -> str:
    """Busca jurisprudência e normas do TCE-PA via Pesquisa Integrada.

    Pesquisa acórdãos, resoluções, portarias, atos, prejulgados e informativos
    do Tribunal de Contas do Estado do Pará.

    Args:
        base: Base de dados — "acordaos", "acordaos-virtual", "resolucoes",
              "portarias", "atos", "informativos", "prejulgados".
        query: Texto de busca (opcional).
        ano: Filtro por ano (opcional).
        mes: Filtro por mês 1-12 (opcional, busca textual aproximada).
        pagina: Página de resultados (padrão: 1, 20 resultados/página).

    Returns:
        Tabela com resultados e links para documentos PDF.
    """
    bases_validas = list(BASES_JURISPRUDENCIA)
    if base not in bases_validas:
        return f"Base inválida: '{base}'. Use: {bases_validas}"

    slug = BASES_JURISPRUDENCIA[base]
    await ctx.info(f"Buscando {base} no TCE-PA (página {pagina})...")
    resultados = await client.buscar_pesquisa_integrada(
        slug=slug, query=query, ano=ano, mes=mes, pagina=pagina
    )

    if not resultados:
        return "Nenhum resultado encontrado para os filtros informados."

    await ctx.info(f"{len(resultados)} resultado(s) encontrado(s)")

    bases_info = ", ".join(bases_validas)
    header = (
        f"**TCE-PA Jurisprudência — {base}**\n"
        f"Página {pagina} | {len(resultados)} resultado(s)\n"
        f"Bases disponíveis: {bases_info}\n\n"
    )

    # Pick most informative campos per base type
    if base in ("acordaos", "acordaos-virtual"):
        campo_data = "Data da sessão plenária"
        campo_extra = "Decisões"
    else:
        campo_data = "Data"
        campo_extra = "Tipo"

    rows = []
    for r in resultados:
        data_val = r.campos.get(campo_data, r.campos.get("Data", "—"))[:30]
        extra_val = r.campos.get(campo_extra, "—")[:40]
        rows.append(
            (
                data_val,
                (r.titulo or "—")[:55],
                extra_val,
                r.url_documento or r.url_pagina or "—",
            )
        )

    return header + markdown_table(["Data", "Título", "Info", "Documento PDF"], rows)


async def buscar_conteudo_pa(
    ctx: Context,
    base: str = "noticias",
    query: str = "",
    ano: int | None = None,
    mes: int | None = None,
    pagina: int = 1,
) -> str:
    """Busca conteúdo informativo do TCE-PA via Pesquisa Integrada.

    Pesquisa notícias, acervo bibliográfico, ações educacionais,
    canal YouTube e banco de imagens do TCE-PA.

    Args:
        base: Base de dados — "noticias", "acervo", "educacao",
              "youtube", "imagens".
        query: Texto de busca (opcional).
        ano: Filtro por ano (opcional).
        mes: Filtro por mês 1-12 (opcional, busca textual aproximada).
        pagina: Página de resultados (padrão: 1, 20 resultados/página).

    Returns:
        Tabela com resultados e links para conteúdo.
    """
    bases_validas = list(BASES_CONTEUDO)
    if base not in bases_validas:
        return f"Base inválida: '{base}'. Use: {bases_validas}"

    slug = BASES_CONTEUDO[base]
    await ctx.info(f"Buscando {base} no TCE-PA (página {pagina})...")
    resultados = await client.buscar_pesquisa_integrada(
        slug=slug, query=query, ano=ano, mes=mes, pagina=pagina
    )

    if not resultados:
        return "Nenhum resultado encontrado para os filtros informados."

    await ctx.info(f"{len(resultados)} resultado(s) encontrado(s)")

    bases_info = ", ".join(bases_validas)
    header = (
        f"**TCE-PA Conteúdo — {base}**\n"
        f"Página {pagina} | {len(resultados)} resultado(s)\n"
        f"Bases disponíveis: {bases_info}\n\n"
    )

    # Pick most informative date/summary campo per base
    data_campo = "Data de publicação" if base == "noticias" else "Data"
    resumo_campo = "Resumo" if base == "noticias" else "Descrição"

    col_link = "Vídeo (YouTube)" if base == "youtube" else "Link"

    rows = []
    for r in resultados:
        data_val = r.campos.get(data_campo, r.campos.get("Ano de publicação", "—"))[:20]
        resumo_val = r.campos.get(resumo_campo, "—")[:60]
        rows.append(
            (
                data_val,
                (r.titulo or "—")[:55],
                resumo_val,
                r.url_documento or r.url_pagina or "—",
            )
        )

    return header + markdown_table(["Data", "Título", "Resumo", col_link], rows)
