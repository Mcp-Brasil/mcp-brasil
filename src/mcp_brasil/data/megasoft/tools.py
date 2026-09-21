"""Tool functions for the Megasoft municipal transparency portal feature.

Rules (ADR-001):
    - tools.py NEVER makes HTTP directly — delegates to client.py
    - Returns formatted strings for LLM consumption
    - Uses Context for structured logging and progress reporting
"""

from __future__ import annotations

from fastmcp import Context

from mcp_brasil._shared.formatting import format_brl, markdown_table
from mcp_brasil.exceptions import HttpClientError

from . import client
from .schemas import Empenho, OrdemPagamento, Receita

_MAX_ITENS = 20

_ERRO_PORTAL = (
    "Não foi possível consultar o portal Megasoft: {erro}. "
    "Verifique o subdomínio do município e tente novamente."
)


def _valor(valor: float | None) -> str:
    return format_brl(valor) if valor is not None else "—"


def _formatar_empenho(emp: Empenho) -> str:
    linhas = [f"### Empenho {emp.numero_tcm or emp.codigo or '—'} ({emp.data or '—'})"]
    linhas.append(f"- **Fornecedor:** {emp.fornecedor or '—'}")
    linhas.append(f"- **Valor empenhado:** {_valor(emp.valor_empenho)}")
    linhas.append(f"- **Valor liquidado:** {_valor(emp.valor_liquidacao)}")
    linhas.append(f"- **Valor pago:** {_valor(emp.valor_pagamento)}")
    linhas.append(f"- **Órgão:** {emp.orgao or '—'}")
    linhas.append(f"- **Natureza da despesa:** {emp.natureza_despesa or '—'}")
    if emp.historico:
        linhas.append(f"- **Histórico:** {emp.historico[:300]}")
    linhas.append("")
    return "\n".join(linhas)


def _formatar_ordem(ordem: OrdemPagamento) -> str:
    linhas = [f"### Ordem {ordem.numero or ordem.codigo or '—'} ({ordem.data_pagamento or '—'})"]
    linhas.append(f"- **Fornecedor:** {ordem.fornecedor or '—'}")
    linhas.append(f"- **CPF/CNPJ:** {ordem.cpf_cnpj_fornecedor or '—'}")
    linhas.append(f"- **Valor pago:** {_valor(ordem.valor_pago)}")
    linhas.append(f"- **Valor total:** {_valor(ordem.valor_total)}")
    linhas.append(f"- **Órgão:** {ordem.orgao or '—'}")
    linhas.append(f"- **Fase:** {ordem.fase or '—'}")
    linhas.append("")
    return "\n".join(linhas)


def _formatar_receita(rec: Receita) -> str:
    linhas = [f"### {rec.receita or '—'}"]
    linhas.append(f"- **Fonte:** {rec.fonte or '—'}")
    linhas.append(f"- **Natureza:** {rec.natureza or '—'}")
    linhas.append(f"- **Origem do recurso:** {rec.origem_recurso or '—'}")
    linhas.append(f"- **Valor orçado:** {_valor(rec.valor_orcado)}")
    linhas.append(f"- **Arrecadado:** {_valor(rec.arrecadacao)}")
    linhas.append(f"- **Acumulado:** {_valor(rec.valor_acumulado)}")
    linhas.append("")
    return "\n".join(linhas)


async def buscar_empenhos(
    ctx: Context,
    municipio: str,
    data_inicial: str,
    data_final: str,
    pagina: int = 1,
    tamanho: int = 100,
) -> str:
    """Busca empenhos (despesas empenhadas) de um município em um período.

    Use quando precisar listar despesas empenhadas, seus fornecedores e
    valores de um município que publica o Portal da Transparência na
    plataforma Megasoft.

    Args:
        ctx: Contexto MCP.
        municipio: Subdomínio do portal Megasoft, sem o domínio
            (ex: "serranopolis"), ou a URL completa do portal.
        data_inicial: Início do período no formato DD/MM/AAAA.
        data_final: Fim do período no formato DD/MM/AAAA.
        pagina: Número da página (começa em 1).
        tamanho: Registros por página (máximo 100).

    Returns:
        Lista de empenhos com fornecedor, valores e órgão responsável.
    """
    await ctx.info("Buscando empenhos no portal Megasoft...")
    try:
        resultado = await client.buscar_empenhos(
            municipio,
            data_inicial=data_inicial,
            data_final=data_final,
            pagina=pagina,
            tamanho=tamanho,
        )
    except ValueError as exc:
        return f"Parâmetro inválido: {exc}"
    except HttpClientError as exc:
        return _ERRO_PORTAL.format(erro=exc)
    except RuntimeError as exc:
        return f"Falha ao obter o token do portal Megasoft: {exc}"

    if not resultado.empenhos:
        return (
            "Nenhum empenho para estes filtros "
            f"(município '{municipio}', {data_inicial} a {data_final})."
        )

    linhas = [
        f"**{resultado.total} empenhos no período {data_inicial} a {data_final} "
        f"(página {resultado.pagina}):**\n"
    ]
    for emp in resultado.empenhos[:_MAX_ITENS]:
        linhas.append(_formatar_empenho(emp))
    if resultado.total > len(resultado.empenhos[:_MAX_ITENS]):
        linhas.append(
            f"*Mostrando {min(_MAX_ITENS, len(resultado.empenhos))} de {resultado.total}. "
            f"Use pagina={pagina + 1} para a próxima página.*"
        )
    return "\n".join(linhas)


async def buscar_ordens_pagamento(
    ctx: Context,
    municipio: str,
    data_inicial: str,
    data_final: str,
    pagina: int = 1,
    tamanho: int = 100,
) -> str:
    """Busca ordens de pagamento de um município em um período.

    Use quando precisar verificar pagamentos efetivamente realizados a
    fornecedores (com data de pagamento, empenho de origem e valores).

    Args:
        ctx: Contexto MCP.
        municipio: Subdomínio do portal Megasoft, sem o domínio
            (ex: "serranopolis"), ou a URL completa do portal.
        data_inicial: Início do período no formato DD/MM/AAAA.
        data_final: Fim do período no formato DD/MM/AAAA.
        pagina: Número da página (começa em 1).
        tamanho: Registros por página (máximo 100).

    Returns:
        Lista de ordens de pagamento com fornecedor, data e valores.
    """
    await ctx.info("Buscando ordens de pagamento no portal Megasoft...")
    try:
        resultado = await client.buscar_ordens_pagamento(
            municipio,
            data_inicial=data_inicial,
            data_final=data_final,
            pagina=pagina,
            tamanho=tamanho,
        )
    except ValueError as exc:
        return f"Parâmetro inválido: {exc}"
    except HttpClientError as exc:
        return _ERRO_PORTAL.format(erro=exc)
    except RuntimeError as exc:
        return f"Falha ao obter o token do portal Megasoft: {exc}"

    if not resultado.ordens:
        return (
            "Nenhuma ordem de pagamento para estes filtros "
            f"(município '{municipio}', {data_inicial} a {data_final})."
        )

    linhas = [
        f"**{resultado.total} ordens de pagamento no período {data_inicial} a {data_final} "
        f"(página {resultado.pagina}):**\n"
    ]
    for ordem in resultado.ordens[:_MAX_ITENS]:
        linhas.append(_formatar_ordem(ordem))
    if resultado.total > len(resultado.ordens[:_MAX_ITENS]):
        linhas.append(
            f"*Mostrando {min(_MAX_ITENS, len(resultado.ordens))} de {resultado.total}. "
            f"Use pagina={pagina + 1} para a próxima página.*"
        )
    return "\n".join(linhas)


async def totais_ordens_pagamento(
    ctx: Context,
    municipio: str,
    data_inicial: str,
    data_final: str,
) -> str:
    """Busca os totais financeiros das ordens de pagamento de um município.

    Use quando precisar do consolidado do período (total pago, anulado,
    retido e líquido), sem listar cada ordem individualmente.

    Args:
        ctx: Contexto MCP.
        municipio: Subdomínio do portal Megasoft, sem o domínio
            (ex: "serranopolis"), ou a URL completa do portal.
        data_inicial: Início do período no formato DD/MM/AAAA.
        data_final: Fim do período no formato DD/MM/AAAA.

    Returns:
        Totais financeiros consolidados do período.
    """
    await ctx.info("Buscando totais de ordens de pagamento no portal Megasoft...")
    try:
        totais = await client.totais_ordens_pagamento(
            municipio,
            data_inicial=data_inicial,
            data_final=data_final,
        )
    except ValueError as exc:
        return f"Parâmetro inválido: {exc}"
    except HttpClientError as exc:
        return _ERRO_PORTAL.format(erro=exc)
    except RuntimeError as exc:
        return f"Falha ao obter o token do portal Megasoft: {exc}"

    linhas = [
        f"**Totais de ordens de pagamento — {municipio} ({data_inicial} a {data_final}):**\n",
        markdown_table(
            ["Métrica", "Valor"],
            [
                ["Valor pago", _valor(totais.valor_pago)],
                ["Valor anulado", _valor(totais.valor_anulacao)],
                ["Valor retido", _valor(totais.valor_retencao)],
                ["Valor total (líquido)", _valor(totais.valor_total)],
            ],
        ),
    ]
    return "\n".join(linhas)


async def buscar_receitas(
    ctx: Context,
    municipio: str,
    data_inicial: str,
    data_final: str,
    pagina: int = 1,
    tamanho: int = 100,
) -> str:
    """Busca receitas (arrecadação) de um município em um período.

    Use quando precisar analisar a arrecadação municipal por receita,
    natureza e fonte de recurso.

    Args:
        ctx: Contexto MCP.
        municipio: Subdomínio do portal Megasoft, sem o domínio
            (ex: "serranopolis"), ou a URL completa do portal.
        data_inicial: Início do período no formato DD/MM/AAAA.
        data_final: Fim do período no formato DD/MM/AAAA.
        pagina: Número da página (começa em 1).
        tamanho: Registros por página (máximo 100).

    Returns:
        Lista de receitas com valores orçado, arrecadado e acumulado.
    """
    await ctx.info("Buscando receitas no portal Megasoft...")
    try:
        resultado = await client.buscar_receitas(
            municipio,
            data_inicial=data_inicial,
            data_final=data_final,
            pagina=pagina,
            tamanho=tamanho,
        )
    except ValueError as exc:
        return f"Parâmetro inválido: {exc}"
    except HttpClientError as exc:
        return _ERRO_PORTAL.format(erro=exc)
    except RuntimeError as exc:
        return f"Falha ao obter o token do portal Megasoft: {exc}"

    if not resultado.receitas:
        return (
            "Nenhuma receita para estes filtros "
            f"(município '{municipio}', {data_inicial} a {data_final})."
        )

    linhas = [
        f"**{resultado.total} receitas no período {data_inicial} a {data_final} "
        f"(página {resultado.pagina}):**\n"
    ]
    for rec in resultado.receitas[:_MAX_ITENS]:
        linhas.append(_formatar_receita(rec))
    if resultado.total > len(resultado.receitas[:_MAX_ITENS]):
        linhas.append(
            f"*Mostrando {min(_MAX_ITENS, len(resultado.receitas))} de {resultado.total}. "
            f"Use pagina={pagina + 1} para a próxima página.*"
        )
    return "\n".join(linhas)
