"""Tool functions for the TCM-GO feature.

Rules (ADR-001):
    - tools.py NEVER makes HTTP directly — delegates to client.py
    - Returns formatted strings for LLM consumption
    - Uses Context for structured logging
"""

from __future__ import annotations

from fastmcp import Context

from mcp_brasil.exceptions import HttpClientError

from . import client
from .constants import OPLOG_VALIDOS

_MAX_ITENS = 20


async def listar_valores_campo(
    ctx: Context,
    campo: str,
    municipio: str | None = None,
) -> str:
    """Lista os valores válidos de um campo do eSiproc (TCM-GO).

    Use antes de filtrar para descobrir a grafia exata aceita pela API
    (municípios exigem CAIXA ALTA COM ACENTO). Campos úteis incluem
    `municipio.keyword`, `unidade_gestora.keyword`, `relator.keyword`,
    `regiao.keyword`, `tipo_protocolo.keyword`, `ano_competencia` e
    `finalizado.keyword`.

    Args:
        ctx: Contexto MCP.
        campo: Nome do campo a agregar (ex: "municipio.keyword").
        municipio: Município (nome oficial) para escopar a agregação.

    Returns:
        Tabela de valores distintos com a contagem de processos de cada um.
    """
    await ctx.info(f"Listando valores do campo {campo} no eSiproc (TCM-GO)...")
    try:
        valores = await client.listar_valores_campo(campo=campo, municipio=municipio)
    except HttpClientError as exc:
        return f"Não foi possível consultar o eSiproc (TCM-GO) agora: {exc}"

    if not valores:
        return f"Nenhum valor retornado para o campo `{campo}` para estes filtros."

    lines: list[str] = [f"**Campo `{campo}` — {len(valores)} valor(es):**\n"]
    lines.append("| Valor | Qtd |")
    lines.append("|---|---|")
    for v in valores[:_MAX_ITENS]:
        lines.append(f"| {v.valor or '—'} | {v.quantidade if v.quantidade is not None else '—'} |")
    if len(valores) > _MAX_ITENS:
        lines.append(f"\n*Mostrando {_MAX_ITENS} de {len(valores)} valores.*")
    return "\n".join(lines)


async def buscar_processos(
    ctx: Context,
    municipio: str | None = None,
    unidade_gestora: str | None = None,
    tipo_protocolo: str | None = None,
    ano: int | None = None,
    relator: str | None = None,
    finalizado: str | None = None,
    numero: str | None = None,
    pagina: int = 0,
) -> str:
    """Busca processos eletrônicos no eSiproc do TCM-GO.

    Consulta os metadados de trâmite dos processos por município, unidade
    gestora, tipo de protocolo, ano de competência, relator e situação.
    Não indexa o full-text do mérito. Use `listar_valores_campo` para
    descobrir a grafia exata dos filtros.

    Args:
        ctx: Contexto MCP.
        municipio: Município (CAIXA ALTA COM ACENTO, ex: "ANÁPOLIS").
        unidade_gestora: Unidade gestora (ex: "PREFEITURA DE ANÁPOLIS").
        tipo_protocolo: Tipo de protocolo (ex: "Contas de Gestão").
        ano: Ano de competência (ex: 2024).
        relator: Conselheiro relator.
        finalizado: "SIM" ou "NÃO".
        numero: Trecho do número do processo no TCM.
        pagina: Página (0-indexada, ~10 processos por página).

    Returns:
        Lista de processos com município, unidade, tipo, relator e fase.
    """
    await ctx.info("Buscando processos no eSiproc (TCM-GO)...")
    try:
        resultado = await client.buscar_processos(
            municipio=municipio,
            unidade_gestora=unidade_gestora,
            tipo_protocolo=tipo_protocolo,
            ano=ano,
            relator=relator,
            finalizado=finalizado,
            numero=numero,
            pagina=pagina,
        )
    except HttpClientError as exc:
        return f"Não foi possível consultar o eSiproc (TCM-GO) agora: {exc}"

    if not resultado.processos:
        return (
            "Nenhum processo encontrado no eSiproc para estes filtros. "
            "Confira a grafia do município (exige CAIXA ALTA COM ACENTO) ou "
            "remova algum filtro."
        )

    lines: list[str] = [
        f"**{resultado.total} processo(s) no eSiproc — página {resultado.pagina} "
        f"({len(resultado.processos)} nesta página):**\n"
    ]
    for p in resultado.processos[:_MAX_ITENS]:
        lines.append(f"### {p.numero or '—'}")
        lines.append(f"- **Município/UG:** {p.municipio or '—'} / {p.unidade_gestora or '—'}")
        lines.append(
            f"- **Tipo:** {p.tipo_protocolo or '—'} | "
            f"**Competência:** {p.ano_competencia or '—'} ({p.periodo_competencia or '—'})"
        )
        lines.append(
            f"- **Relator:** {p.relator or '—'} | "
            f"**Fase:** {p.st_fase or '—'} | **Finalizado:** {p.finalizado or '—'}"
        )
        lines.append(
            f"- **Protocolo:** {p.data_protocolo or '—'} | "
            f"**Última tramitação:** {p.data_ultima_tramitacao or '—'}"
        )
        if p.decisoes:
            lines.append(
                "- **Decisões:** "
                + "; ".join(
                    f"{d.nome or '—'} ({d.arquivo or '—'}, {d.data or '—'})" for d in p.decisoes
                )
            )
        if p.envolvidos:
            lines.append(
                "- **Envolvidos:** "
                + "; ".join(
                    f"{e.nome or '—'} [{e.tipo_envolvimento or '—'}]" for e in p.envolvidos
                )
            )
        lines.append("")

    if resultado.total > len(resultado.processos[:_MAX_ITENS]):
        lines.append(f"*Use pagina={pagina + 1} para a próxima página.*")
    return "\n".join(lines)


async def buscar_acordaos(
    ctx: Context,
    pesquisa: str | None = None,
    municipio: str | None = None,
    assunto: str | None = None,
    relator: str | None = None,
    numero_processo: str | None = None,
    data_sessao_ini: str | None = None,
    data_sessao_fin: str | None = None,
    oplog: str = "AND",
) -> str:
    """Busca acórdãos no TCMJuris (full-text) do TCM-GO.

    Pesquisa o inteiro teor dos julgados publicados pelo Tribunal desde 2013,
    por expressão livre e/ou por campos (município, assunto, relator, número do
    processo e data da sessão). Devolve o link do PDF de cada acórdão. Os
    filtros de município e assunto são refiltrados com igualdade exata.

    Args:
        ctx: Contexto MCP.
        pesquisa: Termo de busca livre (aceita E/OU e aspas).
        municipio: Município (igualdade exata, sem acento).
        assunto: Assunto do julgado (igualdade exata, sem acento).
        relator: Conselheiro relator.
        numero_processo: Número do processo (formato NNNNN/AA).
        data_sessao_ini: Data inicial da sessão (DD/MM/AAAA).
        data_sessao_fin: Data final da sessão (DD/MM/AAAA).
        oplog: Operador da busca livre: "AND" ou "OR".

    Returns:
        Lista de acórdãos com processo, julgamento, relator, município,
        assunto e link do PDF.
    """
    await ctx.info("Buscando acórdãos no TCMJuris (TCM-GO)...")
    if oplog not in OPLOG_VALIDOS:
        return f"Operador inválido: use um de {', '.join(OPLOG_VALIDOS)}."
    if not any(
        (pesquisa, municipio, assunto, relator, numero_processo, data_sessao_ini, data_sessao_fin)
    ):
        return (
            "Informe ao menos um critério de busca: pesquisa, município, assunto, "
            "relator, número do processo ou período de sessão."
        )
    try:
        resultado = await client.buscar_acordaos(
            pesquisa=pesquisa,
            municipio=municipio,
            assunto=assunto,
            relator=relator,
            numero_processo=numero_processo,
            data_sessao_ini=data_sessao_ini,
            data_sessao_fin=data_sessao_fin,
            oplog=oplog,
        )
    except HttpClientError as exc:
        return f"Não foi possível consultar o TCMJuris (TCM-GO) agora: {exc}"

    if not resultado.acordaos:
        return (
            "Nenhum acórdão encontrado no TCMJuris para estes filtros. "
            "Confira o termo ou remova algum filtro."
        )

    cabecalho = (
        f"**{resultado.total_apos_refiltro} acórdão(s) no TCMJuris** "
        f"(servidor devolveu {resultado.total_servidor}; após refiltro exato "
        f"{resultado.total_apos_refiltro})"
    )
    if resultado.truncado:
        cabecalho += " — ⚠ resultado capado em 1000 linhas, restrinja a busca"
    lines: list[str] = [cabecalho + ":\n"]
    lines.append("| Acórdão | Processo | Julgamento | Relator | Município | Assunto | PDF |")
    lines.append("|---|---|---|---|---|---|---|")
    for a in resultado.acordaos[:_MAX_ITENS]:
        link = f"[PDF]({a.url_pdf})" if a.url_pdf else "—"
        lines.append(
            f"| {a.numero_acordao or '—'} | {a.processo or '—'} | {a.data_julgamento or '—'} | "
            f"{a.relator or '—'} | {a.municipio or '—'} | {a.assunto or '—'} | {link} |"
        )
    if resultado.total_apos_refiltro > _MAX_ITENS:
        lines.append(f"\n*Mostrando {_MAX_ITENS} de {resultado.total_apos_refiltro} acórdãos.*")
    return "\n".join(lines)
