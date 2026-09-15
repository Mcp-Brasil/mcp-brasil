"""Tool functions for the ANA feature.

Rules (ADR-001):
    - tools.py NEVER makes HTTP directly — delegates to client.py
    - Returns formatted strings for LLM consumption
"""

from __future__ import annotations

from fastmcp import Context

from mcp_brasil._shared.formatting import format_number_br, markdown_table
from mcp_brasil.exceptions import HttpClientError

from . import client
from .constants import AUTH_ENV_VAR

_UNAUTHORIZED_MSG = (
    "A API do Hidroweb/SNIRH (ANA) está exigindo autenticação (HTTP 401) "
    "neste endpoint. Se você possui um token de acesso, configure a variável "
    f"de ambiente {AUTH_ENV_VAR}. Sem token, este endpoint pode estar "
    "temporariamente restrito — tente novamente mais tarde."
)


def _erro_api_ana(exc: HttpClientError) -> str:
    """Format an ANA API failure for LLM consumption."""
    if "401" in str(exc):
        return f"{_UNAUTHORIZED_MSG} Detalhe: {exc}"
    return f"A API da ANA está indisponível no momento. Detalhe: {exc}"


async def buscar_estacoes(
    ctx: Context,
    codigo_estacao: str | None = None,
    codigo_rio: str | None = None,
    codigo_bacia: str | None = None,
    codigo_sub_bacia: str | None = None,
    nome_estacao: str | None = None,
    tipo_estacao: int | None = None,
) -> str:
    """Busca estações hidrológicas da ANA no sistema Hidroweb.

    Permite pesquisar estações fluviométricas e pluviométricas por
    código, nome, rio, bacia ou sub-bacia hidrográfica.

    Args:
        codigo_estacao: Código da estação (ex: "60435000").
        codigo_rio: Código do rio para filtrar.
        codigo_bacia: Código da bacia hidrográfica.
        codigo_sub_bacia: Código da sub-bacia.
        nome_estacao: Nome da estação (busca parcial).
        tipo_estacao: Tipo da estação (1=Fluviométrica, 2=Pluviométrica).

    Returns:
        Tabela com as estações encontradas.
    """
    await ctx.info("Buscando estações hidrológicas na ANA...")
    try:
        estacoes = await client.buscar_estacoes(
            codigo_estacao=codigo_estacao,
            codigo_rio=codigo_rio,
            codigo_bacia=codigo_bacia,
            codigo_sub_bacia=codigo_sub_bacia,
            nome_estacao=nome_estacao,
            tipo_estacao=tipo_estacao,
        )
    except HttpClientError as exc:
        return _erro_api_ana(exc)

    if not estacoes:
        return "Nenhuma estação encontrada para os filtros informados."

    await ctx.info(f"{len(estacoes)} estação(ões) encontrada(s)")

    rows = [
        (
            e.codigo_estacao,
            e.nome_estacao,
            e.nome_rio or "—",
            e.municipio or "—",
            e.estado or "—",
            e.tipo_estacao or "—",
        )
        for e in estacoes
    ]
    header = f"Estações hidrológicas ANA ({len(estacoes)} encontradas):\n\n"
    return header + markdown_table(["Código", "Nome", "Rio", "Município", "UF", "Tipo"], rows)


async def consultar_telemetria(
    ctx: Context,
    codigo_estacao: str,
    data_inicio: str | None = None,
    data_fim: str | None = None,
) -> str:
    """Consulta dados telemétricos de uma estação hidrológica da ANA.

    Retorna leituras de nível da água, vazão e precipitação (chuva)
    de uma estação. Os dados são coletados automaticamente por
    sensores em tempo real.

    Args:
        codigo_estacao: Código da estação (ex: "60435000").
        data_inicio: Data inicial no formato dd/MM/yyyy (opcional).
        data_fim: Data final no formato dd/MM/yyyy (opcional).

    Returns:
        Tabela com os dados telemétricos.
    """
    await ctx.info(f"Consultando telemetria da estação {codigo_estacao}...")
    try:
        dados = await client.consultar_telemetria(
            codigo_estacao=codigo_estacao,
            data_inicio=data_inicio,
            data_fim=data_fim,
        )
    except HttpClientError as exc:
        return _erro_api_ana(exc)

    if not dados:
        return f"Nenhum dado telemétrico encontrado para a estação {codigo_estacao}."

    await ctx.info(f"{len(dados)} leitura(s) encontrada(s)")

    rows = [
        (
            d.data_hora,
            format_number_br(d.nivel, 2) if d.nivel is not None else "—",
            format_number_br(d.vazao, 1) if d.vazao is not None else "—",
            format_number_br(d.chuva, 1) if d.chuva is not None else "—",
        )
        for d in dados
    ]
    header = f"Telemetria da estação {codigo_estacao} ({len(dados)} leituras):\n\n"
    return header + markdown_table(["Data/Hora", "Nível (cm)", "Vazão (m³/s)", "Chuva (mm)"], rows)


async def monitorar_reservatorios(
    ctx: Context,
    codigo_reservatorio: str | None = None,
    data_inicio: str | None = None,
    data_fim: str | None = None,
) -> str:
    """Monitora reservatórios do Sistema de Acompanhamento de Reservatórios (SAR) da ANA.

    Retorna dados de volume útil, cota, vazão afluente e defluente
    dos principais reservatórios brasileiros. Útil para acompanhar
    a situação hídrica e o nível dos reservatórios de hidrelétricas.

    Args:
        codigo_reservatorio: Código do reservatório (opcional).
        data_inicio: Data inicial no formato dd/MM/yyyy (opcional).
        data_fim: Data final no formato dd/MM/yyyy (opcional).

    Returns:
        Tabela com os dados dos reservatórios.
    """
    await ctx.info("Consultando dados de reservatórios no SAR/ANA...")
    try:
        reservatorios = await client.monitorar_reservatorios(
            codigo_reservatorio=codigo_reservatorio,
            data_inicio=data_inicio,
            data_fim=data_fim,
        )
    except HttpClientError as exc:
        return _erro_api_ana(exc)

    if not reservatorios:
        return "Nenhum dado de reservatório encontrado para os filtros informados."

    await ctx.info(f"{len(reservatorios)} registro(s) de reservatório(s) encontrado(s)")

    rows = [
        (
            r.nome_reservatorio,
            r.rio or "—",
            r.estado or "—",
            r.data or "—",
            f"{format_number_br(r.volume_util, 1)}%" if r.volume_util is not None else "—",
            format_number_br(r.vazao_afluente, 1) if r.vazao_afluente is not None else "—",
            format_number_br(r.vazao_defluente, 1) if r.vazao_defluente is not None else "—",
        )
        for r in reservatorios
    ]
    header = f"Reservatórios SAR/ANA ({len(reservatorios)} registros):\n\n"
    return header + markdown_table(
        ["Reservatório", "Rio", "UF", "Data", "Vol. Útil", "Afluente (m³/s)", "Defluente (m³/s)"],
        rows,
    )
