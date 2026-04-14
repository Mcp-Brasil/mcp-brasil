"""Tool functions for the dados_gov_br feature."""

from __future__ import annotations

from fastmcp import Context

from mcp_brasil._shared.formatting import markdown_table

from . import client


async def buscar_conjuntos(
    ctx: Context,
    pagina: int = 1,
    nome: str | None = None,
    id_organizacao: str | None = None,
) -> str:
    """Busca conjuntos de dados abertos do governo federal.

    Pesquisa no catálogo do Portal Brasileiro de Dados Abertos (dados.gov.br).
    Inclui datasets de saúde, educação, meio ambiente, economia e mais.

    Args:
        pagina: Página de resultados (padrão 1).
        nome: Filtrar por nome do dataset.
        id_organizacao: Filtrar por ID da organização publicadora.

    Returns:
        Lista de datasets encontrados com título, organização e temas.
    """
    filtro = nome or id_organizacao or "todos"
    await ctx.info(f"Buscando conjuntos de dados (filtro: {filtro})...")
    resultado = await client.buscar_conjuntos(
        pagina=pagina,
        nome=nome,
        id_organizacao=id_organizacao,
    )
    await ctx.info(f"{resultado.total} conjuntos encontrados")

    if not resultado.conjuntos:
        return "Nenhum conjunto de dados encontrado."

    lines = [f"**Total:** {resultado.total} conjuntos de dados\n"]
    for i, c in enumerate(resultado.conjuntos, 1):
        titulo = c.title or c.nome or "Sem título"
        org = c.organizationName or c.organizationTitle or "N/A"
        lines.extend(
            [
                f"### {i}. {titulo}",
                f"**ID:** `{c.id}`" if c.id else "",
                f"**Organização:** {org}",
                f"**Temas:** {c.temas or 'N/A'}",
                f"**Recursos:** {c.quantidadeRecursos or 'N/A'}",
                f"**Downloads:** {c.quantidadeDownloads or 'N/A'}",
                "",
            ]
        )

    if resultado.total > len(resultado.conjuntos):
        lines.append(f"*Use pagina={pagina + 1} para mais resultados.*")
    return "\n".join(lines)


async def detalhar_conjunto(conjunto_id: str, ctx: Context) -> str:
    """Obtém detalhes completos de um conjunto de dados do Portal Dados Abertos.

    Retorna título, descrição, organização, temas, tags, licença, recursos
    e datas de atualização. Recursos (arquivos CSV, JSON, etc.) já vêm incluídos.

    Use buscar_conjuntos() para encontrar o ID do dataset.

    Args:
        conjunto_id: ID do conjunto de dados.

    Returns:
        Detalhes completos do dataset com seus recursos.
    """
    await ctx.info(f"Detalhando conjunto {conjunto_id}...")
    conjunto = await client.detalhar_conjunto(conjunto_id)

    if not conjunto:
        return f"Conjunto de dados '{conjunto_id}' não encontrado."

    temas = (
        ", ".join(t.get("title", t.get("name", "")) for t in conjunto.temas)
        if conjunto.temas
        else "N/A"
    )
    tags = (
        ", ".join(t.display_name or t.name or "" for t in conjunto.tags)
        if conjunto.tags
        else "N/A"
    )
    desc = (conjunto.descricao or "Sem descrição")[:500]

    lines = [
        f"**{conjunto.titulo or 'Sem título'}**",
        f"\n{desc}",
        f"\n**Organização:** {conjunto.organizacao or 'N/A'}",
        f"**Licença:** {conjunto.licenca or 'N/A'}",
        f"**Periodicidade:** {conjunto.periodicidade or 'N/A'}",
        f"**Temas:** {temas}",
        f"**Tags:** {tags}",
        f"**Cobertura espacial:** {conjunto.coberturaEspacial or 'N/A'}",
        f"**Versão:** {conjunto.versao or 'N/A'}",
        f"**Última atualização (metadados):** {conjunto.dataUltimaAtualizacaoMetadados or 'N/A'}",
        f"**Última atualização (dados):** {conjunto.dataUltimaAtualizacaoArquivo or 'N/A'}",
    ]

    if conjunto.recursos:
        lines.append(f"\n### Recursos ({len(conjunto.recursos)})\n")
        for j, r in enumerate(conjunto.recursos, 1):
            lines.append(f"**{j}. {r.titulo or 'Sem título'}**")
            lines.append(f"  Formato: {r.formato or 'N/A'} | Tipo: {r.tipo or 'N/A'}")
            if r.link:
                lines.append(f"  [Download]({r.link})")
            if r.descricao:
                lines.append(f"  {r.descricao[:200]}")
            lines.append("")

    return "\n".join(lines)


async def listar_organizacoes(
    ctx: Context,
    pagina: int = 1,
    nome: str | None = None,
) -> str:
    """Lista organizações que publicam dados no Portal Dados Abertos.

    Retorna ministérios, autarquias e órgãos federais que disponibilizam
    datasets abertos. Pode filtrar por nome.

    Args:
        pagina: Página de resultados (padrão 1).
        nome: Filtrar por nome da organização.

    Returns:
        Lista de organizações com nome e quantidade de datasets.
    """
    await ctx.info("Listando organizações...")
    resultado = await client.listar_organizacoes(pagina=pagina, nome=nome)
    await ctx.info(f"{resultado.total} organizações encontradas")

    if not resultado.organizacoes:
        return "Nenhuma organização encontrada."

    rows = [
        (
            o.titulo or o.nome or "N/A",
            str(o.qtdConjuntoDeDados or "0"),
            o.organizationEsfera or "N/A",
        )
        for o in resultado.organizacoes
    ]
    header = f"**Total:** {resultado.total} organizações\n\n"
    table = markdown_table(["Organização", "Datasets", "Esfera"], rows)

    footer = ""
    if resultado.total > len(resultado.organizacoes):
        footer = f"\n\n*Use pagina={pagina + 1} para mais resultados.*"
    return header + table + footer


async def detalhar_organizacao(organizacao_id: str, ctx: Context) -> str:
    """Obtém detalhes de um órgão que publica dados no Portal Dados Abertos.

    Retorna nome completo, descrição, imagem institucional e contagem
    de datasets publicados pelo órgão.

    Use listar_organizacoes() para encontrar o ID do órgão.

    Args:
        organizacao_id: ID da organização.

    Returns:
        Detalhes completos da organização.
    """
    await ctx.info(f"Detalhando organização {organizacao_id}...")
    org = await client.detalhar_organizacao(organizacao_id)

    if not org:
        return f"Organização '{organizacao_id}' não encontrada."

    nome = org.displayName or org.nome or org.name or "Sem nome"
    lines = [
        f"**{nome}**",
        f"\n{org.descricao or 'Sem descrição'}",
        f"\n**ID:** `{org.id}`",
        f"**Slug:** {org.name or 'N/A'}",
        f"**Datasets:** {org.quantidadeConjuntoDados or 'N/A'}",
        f"**Seguidores:** {org.quantidadeSeguidores or 'N/A'}",
        f"**Ativo:** {org.ativo or 'N/A'}",
    ]
    return "\n".join(lines)


async def listar_temas(ctx: Context) -> str:
    """Lista temas (grupos temáticos) do Portal Dados Abertos.

    Retorna categorias como Educação, Saúde, Meio Ambiente, etc.
    Cada tema agrupa conjuntos de dados relacionados.

    Returns:
        Lista de temas com nome e contagem de datasets.
    """
    await ctx.info("Listando temas...")
    temas = await client.listar_temas()
    await ctx.info(f"{len(temas)} temas encontrados")

    if not temas:
        return "Nenhum tema encontrado."

    rows = [
        (
            t.title or t.displayName or t.name or "N/A",
            str(t.packageCount or 0),
            (t.description or "")[:80],
        )
        for t in temas
    ]
    return markdown_table(["Tema", "Datasets", "Descrição"], rows)


async def buscar_tags(nome: str, ctx: Context) -> str:
    """Busca tags (etiquetas) usadas para classificar datasets no portal.

    Tags são palavras-chave associadas a conjuntos de dados para
    facilitar a descoberta e filtragem.

    Args:
        nome: Nome ou parte do nome da tag a buscar.

    Returns:
        Lista de tags encontradas.
    """
    await ctx.info(f"Buscando tags '{nome}'...")
    tags = await client.buscar_tags(nome)
    await ctx.info(f"{len(tags)} tags encontradas")

    if not tags:
        return f"Nenhuma tag encontrada para '{nome}'."

    rows = [(t.display_name or t.name or "N/A", t.id or "N/A") for t in tags]
    return markdown_table(["Tag", "ID"], rows)


async def listar_formatos(ctx: Context) -> str:
    """Lista formatos de arquivo disponíveis no Portal Dados Abertos.

    Retorna os tipos de formato (CSV, JSON, XML, etc.) usados nos
    recursos dos conjuntos de dados.

    Returns:
        Lista de formatos disponíveis.
    """
    await ctx.info("Listando formatos disponíveis...")
    formatos = await client.listar_formatos()
    await ctx.info(f"{len(formatos)} formatos encontrados")

    if not formatos:
        return "Nenhum formato encontrado."

    lines = ["**Formatos disponíveis no portal:**\n"]
    for f in sorted(formatos):
        lines.append(f"- {f}")
    return "\n".join(lines)


async def listar_ods(ctx: Context) -> str:
    """Lista os Objetivos de Desenvolvimento Sustentável (ODS) do portal.

    Os ODS da ONU são usados para classificar datasets quanto ao seu
    alinhamento com metas globais de desenvolvimento sustentável.

    Returns:
        Lista de ODS com ID e descrição.
    """
    await ctx.info("Listando ODS...")
    ods_list = await client.listar_ods()
    await ctx.info(f"{len(ods_list)} ODS encontrados")

    if not ods_list:
        return "Nenhum ODS encontrado."

    rows = [(str(o.id or ""), o.descricao or "N/A") for o in ods_list]
    return markdown_table(["ODS", "Descrição"], rows)


async def listar_observancia_legal(ctx: Context) -> str:
    """Lista opções de observância legal para datasets.

    Retorna as bases legais que fundamentam a publicação de dados abertos,
    como a LAI e a LGPD.

    Returns:
        Lista de opções de observância legal.
    """
    await ctx.info("Listando observância legal...")
    items = await client.listar_observancia_legal()
    await ctx.info(f"{len(items)} opções encontradas")

    if not items:
        return "Nenhuma opção de observância legal encontrada."

    rows = [(str(o.id or ""), o.descricao or "N/A") for o in items]
    return markdown_table(["ID", "Descrição"], rows)


async def listar_reusos(
    ctx: Context,
    nome_reuso: str | None = None,
    nome_autor: str | None = None,
    id_organizacao: str | None = None,
) -> str:
    """Lista reusos de dados publicados no portal.

    Reusos são aplicações, dashboards, artigos e outros projetos
    que utilizam dados abertos do portal.

    Args:
        nome_reuso: Filtrar por nome do reuso.
        nome_autor: Filtrar por nome do autor.
        id_organizacao: Filtrar por ID da organização.

    Returns:
        Lista de reusos com nome, autor e situação.
    """
    await ctx.info("Listando reusos de dados...")
    reusos = await client.listar_reusos(
        nome_reuso=nome_reuso,
        nome_autor=nome_autor,
        id_organizacao=id_organizacao,
    )
    await ctx.info(f"{len(reusos)} reusos encontrados")

    if not reusos:
        return "Nenhum reuso encontrado."

    rows = [
        (
            r.nome or "N/A",
            r.autor or "N/A",
            r.organizacao or "N/A",
            r.situacao or "N/A",
        )
        for r in reusos
    ]
    return markdown_table(["Reuso", "Autor", "Organização", "Situação"], rows)


async def detalhar_reuso(reuso_id: str, ctx: Context) -> str:
    """Obtém detalhes de um reuso de dados do portal.

    Reusos são projetos que utilizam dados abertos: apps, dashboards,
    artigos científicos, etc.

    Use listar_reusos() para encontrar o ID do reuso.

    Args:
        reuso_id: ID do reuso.

    Returns:
        Detalhes completos do reuso.
    """
    await ctx.info(f"Detalhando reuso {reuso_id}...")
    reuso = await client.detalhar_reuso(reuso_id)

    if not reuso:
        return f"Reuso '{reuso_id}' não encontrado."

    temas = ", ".join(reuso.temas) if reuso.temas else "N/A"
    tipos = ", ".join(reuso.tipos) if reuso.tipos else "N/A"
    tags = ", ".join(reuso.palavrasChave) if reuso.palavrasChave else "N/A"
    datasets = ", ".join(reuso.conjuntoDados) if reuso.conjuntoDados else "N/A"

    lines = [
        f"**{reuso.nome or 'Sem nome'}**",
        f"\n{reuso.descricao or 'Sem descrição'}",
        f"\n**URL:** {reuso.url or 'N/A'}",
        f"**Autor:** {reuso.autor or 'N/A'}",
        f"**Organização:** {reuso.organizacao or 'N/A'}",
        f"**Responsável:** {reuso.responsavel or 'N/A'}",
        f"**Situação:** {reuso.situacao or 'N/A'}",
        f"**Tipos:** {tipos}",
        f"**Temas:** {temas}",
        f"**Palavras-chave:** {tags}",
        f"**Datasets utilizados:** {datasets}",
        f"**Atualizado em:** {reuso.dataAtualizacao or 'N/A'}",
        f"**Lançado em:** {reuso.dataLancamento or 'N/A'}",
    ]
    return "\n".join(lines)
