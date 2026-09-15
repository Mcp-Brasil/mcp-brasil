"""Tool functions for DOU (Diário Oficial da União) — federal.

Rules (ADR-001):
    - tools.py NEVER makes HTTP directly — delegates to client_dou.py
    - Returns formatted strings for LLM consumption
"""

from __future__ import annotations

from fastmcp import Context

from mcp_brasil.exceptions import HttpClientError

from . import client_dou
from .constants import DOU_ARTICLE_URL, DOU_SECTIONS_INFO


def _erro_fonte_indisponivel(exc: HttpClientError) -> str:
    """Format a source-outage error for LLM consumption.

    Distinct from "no results" messages: an outage must never look like
    an empty search, otherwise endpoint removals go unnoticed (issue #22).
    """
    return (
        "A fonte DOU (portal in.gov.br) está indisponível no momento. "
        f"Detalhe: {exc} "
        "Isso indica instabilidade ou mudança no portal da Imprensa Nacional, "
        "não ausência de resultados. Tente novamente mais tarde."
    )


async def dou_buscar(
    termo: str,
    ctx: Context,
    secao: str = "TODOS",
    periodo: str = "MES",
    orgao: str | None = None,
    tipo: str | None = None,
    campo: str = "TUDO",
    pagina: int = 0,
) -> str:
    """Busca publicações no Diário Oficial da União (federal).

    Pesquisa em atos oficiais federais: leis, decretos, portarias,
    nomeações, licitações, contratos e demais atos de todos os
    poderes e órgãos do governo federal.

    Args:
        termo: Texto para buscar (nome, CNPJ, assunto, número de lei).
        secao: Seção do DOU — SECAO_1, SECAO_2, SECAO_3, EDICAO_EXTRA, TODOS.
        periodo: Período — DIA, SEMANA, MES, ANO.
        orgao: Nome do órgão publicador (opcional, ex: "Ministério da Saúde").
        tipo: Tipo de publicação (opcional, ex: "Portaria", "Decreto").
        campo: Campo de busca — TUDO, TITULO, CONTEUDO.
        pagina: Página de resultados (0-indexada).

    Returns:
        Lista de publicações do DOU com resumo e metadados.
    """
    await ctx.info(f"Buscando no DOU: '{termo}' (seção={secao}, período={periodo})")
    try:
        resultado = await client_dou.buscar_dou(
            termo=termo,
            secao=secao,
            periodo=periodo,
            orgao=orgao,
            tipo_publicacao=tipo,
            campo=campo,
            pagina=pagina,
        )
    except HttpClientError as exc:
        return _erro_fonte_indisponivel(exc)
    await ctx.info(f"{resultado.total} publicações encontradas no DOU")

    if not resultado.publicacoes:
        return f"Nenhuma publicação encontrada no DOU para '{termo}'."

    lines = [f"**Total:** {resultado.total} publicações no DOU\n"]
    for i, p in enumerate(resultado.publicacoes[:20], 1):
        lines.append(f"### {i}. {p.titulo or 'Sem título'}")
        meta = []
        if p.secao:
            meta.append(f"Seção: {p.secao}")
        if p.data_publicacao:
            meta.append(f"Data: {p.data_publicacao}")
        if p.orgao:
            meta.append(f"Órgão: {p.orgao}")
        if p.tipo_publicacao:
            meta.append(f"Tipo: {p.tipo_publicacao}")
        if meta:
            lines.append(f"**{' | '.join(meta)}**")
        if p.resumo:
            lines.append(f"\n> {p.resumo[:400]}...")
        if p.assinante:
            lines.append(f"\n*Assinado por: {p.assinante}*")
        if p.url_titulo:
            lines.append(f"\n[Ler publicação completa]({DOU_ARTICLE_URL}/{p.url_titulo})")
        lines.append("")

    if resultado.total > 20:
        lines.append(
            f"\n*Mostrando 20 de {resultado.total}. Use pagina={pagina + 1} para mais resultados.*"
        )
    return "\n".join(lines)


async def dou_ler_publicacao(url_titulo: str, ctx: Context) -> str:
    """Lê o conteúdo completo de uma publicação do DOU.

    Use o url_titulo retornado pela busca (dou_buscar) para ler
    o texto integral de uma publicação específica.

    Args:
        url_titulo: Identificador da publicação (campo url_titulo da busca).

    Returns:
        Conteúdo completo da publicação com metadados.
    """
    await ctx.info(f"Lendo publicação DOU: {url_titulo}")
    pub = await client_dou.ler_publicacao_dou(url_titulo)

    if not pub:
        return f"Publicação não encontrada: {url_titulo}"

    lines = [f"# {pub.titulo or 'Sem título'}\n"]
    meta = []
    if pub.orgao:
        meta.append(f"**Órgão:** {pub.orgao}")
    if pub.tipo_publicacao:
        meta.append(f"**Tipo:** {pub.tipo_publicacao}")
    if pub.secao:
        meta.append(f"**Seção:** {pub.secao}")
    if pub.data_publicacao:
        meta.append(f"**Data:** {pub.data_publicacao}")
    if pub.edicao:
        meta.append(f"**Edição:** {pub.edicao}")
    if pub.pagina:
        meta.append(f"**Página:** {pub.pagina}")
    if meta:
        lines.append(" | ".join(meta))
        lines.append("")

    if pub.conteudo:
        lines.append("---\n")
        lines.append(pub.conteudo)
    elif pub.resumo:
        lines.append("---\n")
        lines.append(pub.resumo)
    else:
        lines.append("*Conteúdo não disponível.*")

    if pub.assinante:
        lines.append(f"\n\n**Assinado por:** {pub.assinante}")
        if pub.cargo_assinante:
            lines.append(f"**Cargo:** {pub.cargo_assinante}")

    return "\n".join(lines)


async def dou_edicao_do_dia(
    data: str,
    ctx: Context,
    secao: str = "TODOS",
) -> str:
    """Lista publicações de uma edição do DOU por data e seção.

    Retorna todas as publicações de um dia específico, útil para
    monitoramento diário do DOU.

    Args:
        data: Data da edição no formato YYYY-MM-DD.
        secao: Seção do DOU (SECAO_1, SECAO_2, SECAO_3, TODOS).

    Returns:
        Lista de publicações da edição.
    """
    await ctx.info(f"Buscando edição do DOU de {data} (seção={secao})")
    try:
        resultado = await client_dou.buscar_dou(
            termo="*",
            secao=secao,
            periodo="PERSONALIZADO",
            data_inicio=data,
            data_fim=data,
            tamanho=50,
        )
    except HttpClientError as exc:
        return _erro_fonte_indisponivel(exc)
    await ctx.info(f"{resultado.total} publicações na edição de {data}")

    if not resultado.publicacoes:
        return f"Nenhuma publicação encontrada na edição de {data}."

    secao_desc = DOU_SECTIONS_INFO.get(secao.lower(), secao)
    lines = [f"**Edição de {data}** — {secao_desc}\n"]
    lines.append(f"**Total:** {resultado.total} publicações\n")

    for i, p in enumerate(resultado.publicacoes[:50], 1):
        tipo = f"[{p.tipo_publicacao}] " if p.tipo_publicacao else ""
        orgao = f" — {p.orgao}" if p.orgao else ""
        lines.append(f"{i}. {tipo}**{p.titulo or 'Sem título'}**{orgao}")

    return "\n".join(lines)


async def dou_buscar_por_orgao(
    orgao: str,
    ctx: Context,
    periodo: str = "MES",
    secao: str = "TODOS",
) -> str:
    """Busca publicações de um órgão específico no DOU.

    Lista todos os atos publicados por um órgão federal em um período.
    Útil para monitorar atividade de ministérios, agências e autarquias.

    Args:
        orgao: Nome do órgão (ex: "Ministério da Saúde", "IBAMA", "ANVISA").
        periodo: Período — DIA, SEMANA, MES, ANO.
        secao: Seção do DOU (SECAO_1, SECAO_2, SECAO_3, TODOS).

    Returns:
        Publicações do órgão no período.
    """
    await ctx.info(f"Buscando publicações de '{orgao}' no DOU ({periodo})")
    try:
        resultado = await client_dou.buscar_dou(
            termo="*",
            secao=secao,
            periodo=periodo,
            orgao=orgao,
            tamanho=30,
        )
    except HttpClientError as exc:
        return _erro_fonte_indisponivel(exc)
    await ctx.info(f"{resultado.total} publicações de '{orgao}'")

    if not resultado.publicacoes:
        return f"Nenhuma publicação de '{orgao}' encontrada no período."

    lines = [f"**{orgao}** — {resultado.total} publicações ({periodo})\n"]
    for i, p in enumerate(resultado.publicacoes[:30], 1):
        tipo = f"[{p.tipo_publicacao}] " if p.tipo_publicacao else ""
        data = f" ({p.data_publicacao})" if p.data_publicacao else ""
        lines.append(f"{i}. {tipo}**{p.titulo or 'Sem título'}**{data}")

    return "\n".join(lines)


async def dou_buscar_avancado(
    ctx: Context,
    termo: str = "",
    secao: str = "TODOS",
    orgao: str | None = None,
    tipo: str | None = None,
    campo: str = "TUDO",
    data_inicio: str | None = None,
    data_fim: str | None = None,
    pagina: int = 0,
) -> str:
    """Busca avançada no DOU com todos os filtros combinados.

    Permite combinar texto, seção, órgão, tipo de publicação, campo
    de busca e intervalo de datas para buscas precisas no DOU.

    Args:
        termo: Texto para buscar (pode ser vazio para listar por filtros).
        secao: Seção do DOU — SECAO_1, SECAO_2, SECAO_3, EDICAO_EXTRA, TODOS.
        orgao: Nome do órgão publicador (opcional).
        tipo: Tipo de publicação (opcional, ex: "Portaria", "Decreto").
        campo: Campo de busca — TUDO, TITULO, CONTEUDO.
        data_inicio: Data inicial YYYY-MM-DD (opcional).
        data_fim: Data final YYYY-MM-DD (opcional).
        pagina: Página de resultados (0-indexada).

    Returns:
        Publicações filtradas com metadados completos.
    """
    filtros = []
    if termo:
        filtros.append(f"termo='{termo}'")
    if orgao:
        filtros.append(f"órgão='{orgao}'")
    if tipo:
        filtros.append(f"tipo='{tipo}'")
    if data_inicio:
        filtros.append(f"de {data_inicio}")
    if data_fim:
        filtros.append(f"até {data_fim}")
    desc = ", ".join(filtros) if filtros else "todos"
    await ctx.info(f"Busca avançada no DOU: {desc}")

    periodo = "PERSONALIZADO" if (data_inicio and data_fim) else "ANO"
    try:
        resultado = await client_dou.buscar_dou(
            termo=termo or "*",
            secao=secao,
            periodo=periodo,
            data_inicio=data_inicio,
            data_fim=data_fim,
            orgao=orgao,
            tipo_publicacao=tipo,
            campo=campo,
            pagina=pagina,
            tamanho=20,
        )
    except HttpClientError as exc:
        return _erro_fonte_indisponivel(exc)
    await ctx.info(f"{resultado.total} publicações encontradas")

    if not resultado.publicacoes:
        return f"Nenhuma publicação encontrada com os filtros: {desc}."

    lines = [f"**Busca avançada DOU** — {resultado.total} resultados\n"]
    for i, p in enumerate(resultado.publicacoes[:20], 1):
        lines.append(f"### {i}. {p.titulo or 'Sem título'}")
        meta = []
        if p.secao:
            meta.append(f"Seção: {p.secao}")
        if p.data_publicacao:
            meta.append(f"Data: {p.data_publicacao}")
        if p.orgao:
            meta.append(f"Órgão: {p.orgao}")
        if p.tipo_publicacao:
            meta.append(f"Tipo: {p.tipo_publicacao}")
        if meta:
            lines.append(f"**{' | '.join(meta)}**")
        if p.resumo:
            lines.append(f"\n> {p.resumo[:300]}...")
        if p.assinante:
            lines.append(f"\n*Assinado por: {p.assinante}*")
        lines.append("")

    if resultado.total > 20:
        lines.append(f"\n*Mostrando 20 de {resultado.total}. Use pagina={pagina + 1} para mais.*")
    return "\n".join(lines)
