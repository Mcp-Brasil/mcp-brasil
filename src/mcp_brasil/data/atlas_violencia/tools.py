"""Tool functions for the Atlas da Violência feature.

Rules (ADR-001):
    - tools.py NEVER makes HTTP directly — delegates to client.py
    - Returns formatted strings for LLM consumption
"""

from __future__ import annotations

from fastmcp import Context

from mcp_brasil._shared.formatting import markdown_table

from . import client
from .constants import ABRANGENCIAS, TEMAS_CONHECIDOS


async def listar_temas_violencia(ctx: Context) -> str:
    """Lista todos os temas disponíveis no Atlas da Violência.

    Retorna os temas (grupos temáticos) com seus IDs para uso em
    consultas de séries e dados. Inclui homicídios, violência por
    gênero, raça, armas de fogo, trânsito e mais.

    Returns:
        Lista de temas com ID e título.
    """
    await ctx.info("Listando temas do Atlas da Violência...")
    temas = await client.listar_temas()
    await ctx.info(f"{len(temas)} temas encontrados")

    if not temas:
        return "Nenhum tema encontrado no Atlas da Violência."

    rows = [(str(t.id), t.titulo) for t in temas]
    return f"**{len(temas)} temas** no Atlas da Violência:\n\n" + markdown_table(
        ["ID", "Tema"], rows
    )


async def listar_series_tema(
    tema_id: int,
    ctx: Context,
) -> str:
    """Lista as séries de dados disponíveis para um tema do Atlas.

    Cada tema contém múltiplas séries (ex: "Homicídios" tem séries para
    total, por sexo, por raça, taxa por 100mil, etc.). Use o ID da série
    para consultar os valores com consultar_valores_violencia.

    Args:
        tema_id: ID do tema (obtido via listar_temas_violencia).

    Returns:
        Lista de séries com ID e título.
    """
    tema_nome = TEMAS_CONHECIDOS.get(tema_id, f"Tema {tema_id}")
    await ctx.info(f"Listando séries do tema '{tema_nome}'...")
    series = await client.listar_series(tema_id)
    await ctx.info(f"{len(series)} séries encontradas")

    if not series:
        return f"Nenhuma série encontrada para o tema {tema_id} ({tema_nome})."

    rows = [(str(s.id), s.titulo) for s in series]
    header = f"**{len(series)} séries** no tema *{tema_nome}*:\n\n"
    return header + markdown_table(["ID Série", "Título"], rows)


async def consultar_valores_violencia(
    serie_id: int,
    ctx: Context,
    abrangencia: int = 3,
) -> str:
    """Consulta valores de uma série temporal do Atlas da Violência.

    Retorna os dados (contagens, taxas) da série para a abrangência
    geográfica escolhida. Para séries de homicídio, retorna valores
    por ano para cada UF, região ou município.

    Args:
        serie_id: ID da série (obtido via listar_series_tema).
        abrangencia: Escopo geográfico: 1=País, 2=Região, 3=UF, 4=Município.

    Returns:
        Tabela com dados da série por localidade e período.
    """
    scope_name = ABRANGENCIAS.get(abrangencia, f"Abrangência {abrangencia}")
    await ctx.info(f"Consultando série {serie_id} ({scope_name})...")

    if abrangencia == 4:
        return (
            "Abrangência municipal (4) retorna dados muito grandes. "
            "Use consultar_valores_por_regiao com códigos IBGE específicos."
        )

    valores = await client.consultar_valores(serie_id, abrangencia)
    await ctx.info(f"{len(valores)} registros obtidos")

    if not valores:
        return f"Nenhum dado encontrado para a série {serie_id} ({scope_name})."

    rows = [(v.sigla, v.periodo or "N/A", v.valor or "N/A") for v in valores]
    header = f"**Série {serie_id}** — {scope_name} ({len(valores)} registros):\n\n"
    return header + markdown_table(["Local", "Período", "Valor"], rows)


async def consultar_valores_por_regiao(
    serie_id: int,
    regioes: str,
    ctx: Context,
    abrangencia: int = 3,
) -> str:
    """Consulta valores de uma série filtrados por regiões específicas.

    Ideal para comparar estados ou municípios específicos. Para
    municípios, use abrangencia=4 e códigos IBGE (ex: "3304557" para
    Rio de Janeiro, "3550308" para São Paulo).

    Args:
        serie_id: ID da série (obtido via listar_series_tema).
        regioes: Códigos separados por vírgula (ex: "33,35" para RJ e SP).
        abrangencia: Escopo: 1=País, 2=Região, 3=UF, 4=Município.

    Returns:
        Tabela com dados filtrados por região.
    """
    scope_name = ABRANGENCIAS.get(abrangencia, f"Abrangência {abrangencia}")
    await ctx.info(f"Consultando série {serie_id} para regiões {regioes} ({scope_name})...")
    valores = await client.consultar_valores_por_regioes(serie_id, abrangencia, regioes)
    await ctx.info(f"{len(valores)} registros obtidos")

    if not valores:
        return f"Nenhum dado encontrado para série {serie_id}, regiões {regioes}."

    rows = [(v.sigla, v.periodo or "N/A", v.valor or "N/A") for v in valores]
    header = (
        f"**Série {serie_id}** — Regiões: {regioes} ({scope_name}, {len(valores)} registros):\n\n"
    )
    return header + markdown_table(["Local", "Período", "Valor"], rows)


async def consultar_serie_violencia(
    serie_id: int,
    ctx: Context,
) -> str:
    """Consulta metadados de uma série específica do Atlas.

    Retorna o nome/título da série. Útil para confirmar o que uma
    série representa antes de consultar seus valores.

    Args:
        serie_id: ID da série.

    Returns:
        Metadados da série.
    """
    await ctx.info(f"Consultando série {serie_id}...")
    serie = await client.consultar_serie(serie_id)
    if not serie:
        return f"Série {serie_id} não encontrada."
    return f"**Série {serie.id}:** {serie.titulo}"


async def listar_fontes_violencia(ctx: Context) -> str:
    """Lista as fontes de dados do Atlas da Violência.

    Retorna as fontes primárias (SIM/DataSUS, IBGE, etc.) de onde
    os dados do Atlas são extraídos.

    Returns:
        Lista de fontes com ID e nome.
    """
    await ctx.info("Listando fontes de dados...")
    fontes = await client.listar_fontes()
    await ctx.info(f"{len(fontes)} fontes encontradas")

    if not fontes:
        return "Nenhuma fonte encontrada."

    rows = [(str(f.id), f.titulo) for f in fontes]
    return markdown_table(["ID", "Fonte"], rows)


async def listar_metadados_violencia(ctx: Context) -> str:
    """Lista unidades de medida e periodicidades do Atlas da Violência.

    Retorna as unidades (taxa por 100mil, quantidade, proporção) e
    periodicidades (anual, mensal, etc.) disponíveis nas séries.

    Returns:
        Unidades e periodicidades disponíveis.
    """
    await ctx.info("Listando metadados do Atlas...")
    unidades = await client.listar_unidades()
    periodicidades = await client.listar_periodicidades()

    lines: list[str] = ["## Unidades de Medida\n"]
    if unidades:
        rows_u = [(str(u.id), u.titulo) for u in unidades]
        lines.append(markdown_table(["ID", "Unidade"], rows_u))
    else:
        lines.append("Nenhuma unidade encontrada.")

    lines.append("\n## Periodicidades\n")
    if periodicidades:
        rows_p = [(str(p.id), p.titulo) for p in periodicidades]
        lines.append(markdown_table(["ID", "Periodicidade"], rows_p))
    else:
        lines.append("Nenhuma periodicidade encontrada.")

    return "\n".join(lines)
