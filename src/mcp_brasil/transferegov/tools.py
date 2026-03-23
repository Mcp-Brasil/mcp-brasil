"""Tool functions for the TransfereGov feature.

Rules (ADR-001):
    - tools.py NEVER makes HTTP directly — delegates to client.py
    - Returns formatted strings for LLM consumption
"""

from __future__ import annotations

from mcp_brasil._shared.formatting import format_brl, markdown_table

from . import client
from .constants import DEFAULT_PAGE_SIZE
from .schemas import TransferenciaEspecial


def _pagination_hint(count: int, pagina: int) -> str:
    """Return a pagination hint string based on result count and current page."""
    if count >= DEFAULT_PAGE_SIZE:
        return f"\n\n> Use `pagina={pagina + 1}` para ver mais resultados."
    if pagina > 1 and count < DEFAULT_PAGE_SIZE:
        return "\n\n> Última página de resultados."
    return ""


def _format_rows(
    emendas: list[TransferenciaEspecial],
) -> list[tuple[str, ...]]:
    """Format a list of emendas into table rows."""
    return [
        (
            e.nr_emenda or "—",
            (e.autor_emenda or "—")[:40],
            e.tipo_emenda or "—",
            format_brl(e.valor_empenhado) if e.valor_empenhado else "—",
            format_brl(e.valor_pago) if e.valor_pago else "—",
            e.nm_municipio_beneficiario or "—",
            e.uf_beneficiario or "—",
        )
        for e in emendas
    ]


_HEADERS = ["Emenda", "Autor", "Tipo", "Empenhado", "Pago", "Município", "UF"]


async def buscar_emendas_pix(
    ano: int | None = None,
    uf: str | None = None,
    pagina: int = 1,
) -> str:
    """Lista transferências especiais (emendas pix) do TransfereGov.

    Consulta emendas parlamentares do tipo transferência especial
    (popularmente conhecidas como "emendas pix"), que são repasses
    diretos da União para estados e municípios sem convênio.

    Args:
        ano: Ano de exercício (ex: 2024).
        uf: UF do beneficiário (ex: PI, SP).
        pagina: Página de resultados (padrão: 1).

    Returns:
        Tabela com emendas pix encontradas.
    """
    emendas = await client.buscar_emendas_pix(ano=ano, uf=uf, pagina=pagina)
    if not emendas:
        return "Nenhuma emenda pix encontrada para os parâmetros informados."

    rows = _format_rows(emendas)
    header = f"Emendas pix (página {pagina}):\n\n"
    table = header + markdown_table(_HEADERS, rows)
    return table + _pagination_hint(len(emendas), pagina)


async def buscar_emenda_por_autor(
    nome_autor: str,
    ano: int | None = None,
    pagina: int = 1,
) -> str:
    """Busca emendas pix por nome do parlamentar autor.

    Pesquisa emendas parlamentares do tipo transferência especial
    pelo nome (ou parte do nome) do autor da emenda.

    Args:
        nome_autor: Nome ou parte do nome do parlamentar (ex: "Lira").
        ano: Ano de exercício (opcional, ex: 2024).
        pagina: Página de resultados (padrão: 1).

    Returns:
        Tabela com emendas do autor encontradas.
    """
    emendas = await client.buscar_emenda_por_autor(nome_autor, ano=ano, pagina=pagina)
    if not emendas:
        return f"Nenhuma emenda pix encontrada para o autor '{nome_autor}'."

    rows = _format_rows(emendas)
    header = f"Emendas pix do autor '{nome_autor}' (página {pagina}):\n\n"
    table = header + markdown_table(_HEADERS, rows)
    return table + _pagination_hint(len(emendas), pagina)


async def detalhe_emenda(id_transferencia: int) -> str:
    """Detalha uma emenda pix (transferência especial) por ID.

    Retorna informações completas de uma transferência especial,
    incluindo valores empenhados, liquidados e pagos.

    Args:
        id_transferencia: ID da transferência especial no TransfereGov.

    Returns:
        Detalhes da emenda.
    """
    emenda = await client.detalhe_emenda(id_transferencia)
    if not emenda:
        return f"Emenda pix com ID {id_transferencia} não encontrada."

    lines = [
        f"## Emenda Pix {emenda.nr_emenda or id_transferencia}\n",
        f"- **Autor:** {emenda.autor_emenda or '—'}",
        f"- **Tipo:** {emenda.tipo_emenda or '—'}",
        f"- **Ano:** {emenda.ano_exercicio or '—'}",
        f"- **Função:** {emenda.funcao or '—'}",
        f"- **Subfunção:** {emenda.subfuncao or '—'}",
        f"- **Objeto:** {emenda.objeto or '—'}",
        f"- **Valor Empenhado:** "
        f"{format_brl(emenda.valor_empenhado) if emenda.valor_empenhado else '—'}",
        f"- **Valor Liquidado:** "
        f"{format_brl(emenda.valor_liquidado) if emenda.valor_liquidado else '—'}",
        f"- **Valor Pago:** {format_brl(emenda.valor_pago) if emenda.valor_pago else '—'}",
        f"- **Município:** {emenda.nm_municipio_beneficiario or '—'}",
        f"- **UF:** {emenda.uf_beneficiario or '—'}",
        f"- **Entidade:** {emenda.nm_entidade_beneficiaria or '—'}",
    ]
    return "\n".join(lines)


async def emendas_por_municipio(
    nome_municipio: str,
    ano: int | None = None,
    pagina: int = 1,
) -> str:
    """Busca emendas pix destinadas a um município específico.

    Pesquisa transferências especiais (emendas pix) pelo nome
    do município beneficiário.

    Args:
        nome_municipio: Nome ou parte do nome do município (ex: "Teresina").
        ano: Ano de exercício (opcional, ex: 2024).
        pagina: Página de resultados (padrão: 1).

    Returns:
        Tabela com emendas destinadas ao município.
    """
    emendas = await client.emendas_por_municipio(nome_municipio, ano=ano, pagina=pagina)
    if not emendas:
        return f"Nenhuma emenda pix encontrada para o município '{nome_municipio}'."

    rows = _format_rows(emendas)
    header = f"Emendas pix para '{nome_municipio}' (página {pagina}):\n\n"
    table = header + markdown_table(_HEADERS, rows)
    return table + _pagination_hint(len(emendas), pagina)


async def resumo_emendas_ano(ano: int, pagina: int = 1) -> str:
    """Lista emendas pix de um ano para visão geral.

    Retorna uma visão geral das transferências especiais (emendas pix)
    realizadas em um determinado ano de exercício.

    Args:
        ano: Ano de exercício (ex: 2024).
        pagina: Página de resultados (padrão: 1).

    Returns:
        Tabela com emendas do ano.
    """
    emendas = await client.resumo_emendas_ano(ano, pagina=pagina)
    if not emendas:
        return f"Nenhuma emenda pix encontrada para o ano {ano}."

    rows = _format_rows(emendas)
    header = f"Emendas pix do ano {ano} (página {pagina}):\n\n"
    table = header + markdown_table(_HEADERS, rows)
    return table + _pagination_hint(len(emendas), pagina)
