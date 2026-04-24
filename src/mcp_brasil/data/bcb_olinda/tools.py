"""MCP tools for BCB Olinda APIs."""

from __future__ import annotations

from typing import Any

from mcp_brasil._shared.formatting import markdown_table

from . import client
from .constants import INDICADORES_FOCUS


def _fmt_num(v: Any, decimals: int = 4) -> str:
    try:
        return f"{float(v):.{decimals}f}"
    except (TypeError, ValueError):
        return "—"


async def ptax_dolar(data: str) -> str:
    """Cotação PTAX do dólar em uma data específica.

    Args:
        data: Data no formato MM-DD-YYYY (ex: '04-23-2026').
    """
    rows = await client.cotacao_dolar_dia(data)
    if not rows:
        return f"Sem cotação para '{data}' (verifique formato MM-DD-YYYY e se é dia útil)."
    r = rows[0]
    return "\n".join(
        [
            f"**PTAX USD — {data}**",
            "",
            f"- Compra: R$ {_fmt_num(r.get('cotacaoCompra'))}",
            f"- Venda: R$ {_fmt_num(r.get('cotacaoVenda'))}",
            f"- Timestamp: {r.get('dataHoraCotacao') or '—'}",
        ]
    )


async def ptax_dolar_periodo(data_inicial: str, data_final: str) -> str:
    """Cotações PTAX do dólar em um período.

    Args:
        data_inicial: MM-DD-YYYY.
        data_final: MM-DD-YYYY.
    """
    rows = await client.cotacao_dolar_periodo(data_inicial, data_final)
    if not rows:
        return "Nenhuma cotação no período."
    table = [
        (
            (r.get("dataHoraCotacao") or "")[:19],
            _fmt_num(r.get("cotacaoCompra")),
            _fmt_num(r.get("cotacaoVenda")),
            r.get("tipoBoletim") or "—",
        )
        for r in rows[-50:]
    ]
    return (
        f"**PTAX USD — {data_inicial} a {data_final}** "
        f"({len(rows)} pontos; exibindo últimos 50)\n\n"
        + markdown_table(["timestamp", "compra", "venda", "tipo"], table)
    )


async def listar_moedas_ptax() -> str:
    """Lista todas as moedas com cotação PTAX disponível."""
    rows = await client.listar_moedas()
    if not rows:
        return "Nenhuma moeda."
    table = [
        (
            r.get("simbolo") or "—",
            r.get("nomeFormatado") or "—",
            r.get("tipoMoeda") or "—",
        )
        for r in rows
    ]
    return f"**{len(rows)} moedas PTAX**\n\n" + markdown_table(["sigla", "nome", "tipo"], table)


async def ptax_moeda(moeda: str, data: str) -> str:
    """Cotação PTAX de uma moeda específica.

    Args:
        moeda: Sigla (ex: 'EUR', 'GBP', 'JPY').
        data: MM-DD-YYYY.
    """
    rows = await client.cotacao_moeda_dia(moeda, data)
    if not rows:
        return f"Sem cotação para {moeda} em {data}."
    r = rows[0]
    return "\n".join(
        [
            f"**PTAX {moeda.upper()} — {data}**",
            "",
            f"- Compra: R$ {_fmt_num(r.get('cotacaoCompra'))}",
            f"- Venda: R$ {_fmt_num(r.get('cotacaoVenda'))}",
            f"- Paridade compra: {_fmt_num(r.get('paridadeCompra'))}",
            f"- Paridade venda: {_fmt_num(r.get('paridadeVenda'))}",
        ]
    )


async def focus_anual(indicador: str, top: int = 15) -> str:
    """Expectativas Focus anuais para um indicador.

    Args:
        indicador: Nome (ex: 'IPCA', 'Selic', 'PIB Total', 'Câmbio').
        top: Número de pontos (padrão 15, máx 100).
    """
    if indicador not in INDICADORES_FOCUS:
        return (
            f"Indicador '{indicador}' não reconhecido. Válidos: {list(INDICADORES_FOCUS.keys())}"
        )
    top = max(1, min(top, 100))
    rows = await client.expectativas_focus(indicador, top=top)
    if not rows:
        return f"Sem dados para '{indicador}'."
    table = [
        (
            (r.get("Data") or "")[:10],
            str(r.get("DataReferencia") or "")[:4],
            _fmt_num(r.get("Media")),
            _fmt_num(r.get("Mediana")),
            _fmt_num(r.get("DesvioPadrao"), 2),
            str(r.get("numeroRespondentes") or "—"),
        )
        for r in rows
    ]
    return f"**Focus — {indicador}** (top {len(rows)} recentes)\n\n" + markdown_table(
        ["data pub.", "ref. ano", "média", "mediana", "desv. pad.", "n"], table
    )


async def focus_mensal(indicador: str, top: int = 15) -> str:
    """Expectativas Focus mensais (para IPCA, IGP-M, etc. — horizonte mensal).

    Args:
        indicador: Nome do indicador.
        top: Máximo de pontos.
    """
    top = max(1, min(top, 100))
    rows = await client.expectativas_focus_mensais(indicador, top=top)
    if not rows:
        return f"Sem dados mensais para '{indicador}'."
    table = [
        (
            (r.get("Data") or "")[:10],
            str(r.get("DataReferencia") or "")[:7],
            _fmt_num(r.get("Media"), 2),
            _fmt_num(r.get("Mediana"), 2),
            str(r.get("numeroRespondentes") or "—"),
        )
        for r in rows
    ]
    return f"**Focus mensal — {indicador}**\n\n" + markdown_table(
        ["data pub.", "ref. mês", "média", "mediana", "n"], table
    )


async def focus_selic() -> str:
    """Expectativas Focus para Selic por reunião do Copom (próximas reuniões)."""
    rows = await client.expectativas_focus_selic(top=30)
    if not rows:
        return "Sem dados de Selic."
    table = [
        (
            (r.get("Data") or "")[:10],
            r.get("Reuniao") or "—",
            _fmt_num(r.get("Media"), 2),
            _fmt_num(r.get("Mediana"), 2),
            _fmt_num(r.get("Maximo"), 2),
            _fmt_num(r.get("Minimo"), 2),
            str(r.get("numeroRespondentes") or "—"),
        )
        for r in rows
    ]
    return "**Focus — Selic por reunião do Copom**\n\n" + markdown_table(
        ["data pub.", "reunião", "média", "mediana", "máx", "mín", "n"], table
    )


async def listar_indicadores_focus() -> str:
    """Catálogo dos indicadores disponíveis na pesquisa Focus."""
    rows = [(k, v) for k, v in INDICADORES_FOCUS.items()]
    return "**Indicadores Focus**\n\n" + markdown_table(["indicador", "descrição"], rows)
