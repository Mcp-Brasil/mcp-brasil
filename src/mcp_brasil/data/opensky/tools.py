"""Tools for the OpenSky Network feature."""

from __future__ import annotations

import time as _time
from datetime import datetime, timezone

from fastmcp import Context

from mcp_brasil._shared.formatting import markdown_table

from . import client
from .constants import BRASIL_BBOX

UTC = timezone.utc


def _epoch_utc(epoch: int | None) -> str:
    if not epoch:
        return "—"
    return datetime.fromtimestamp(epoch, tz=UTC).strftime("%Y-%m-%d %H:%M UTC")


def _mps_kmh(mps: float | None) -> str:
    return f"{mps * 3.6:.0f} km/h" if mps is not None else "—"


def _m_ft(m: float | None) -> str:
    return f"{m * 3.28084:.0f} ft" if m is not None else "—"


async def info_opensky(ctx: Context) -> str:
    """Metadados do feature OpenSky (bbox, rate limits, cobertura)."""
    await ctx.info("Info OpenSky...")
    return (
        "**OpenSky Network — feature mcp-brasil**\n\n"
        "- **API**: https://opensky-network.org/api (anonymous, 400 req/dia)\n"
        "- **Resolução**: 10s (/states/all)\n"
        f"- **Bounding box Brasil**: lat {BRASIL_BBOX['lamin']}..{BRASIL_BBOX['lamax']}, "
        f"lon {BRASIL_BBOX['lomin']}..{BRASIL_BBOX['lomax']}\n"
        "- **Cobertura**: aviação comercial quase completa; aviação geral "
        "parcial; militares raramente visíveis (ADS-B desligado).\n"
    )


async def voos_ao_vivo_brasil(ctx: Context, limite: int = 50) -> str:
    """Aeronaves ativas sobre o Brasil agora (ADS-B live).

    Args:
        limite: Máx linhas (padrão 50, máximo 200).

    Returns:
        Tabela callsign x país x posição x altitude x velocidade.
    """
    limite = max(1, min(limite, 200))
    await ctx.info("Consultando aeronaves sobre o Brasil...")
    states = await client.states_brasil()
    if not states:
        return "Nenhuma aeronave visível no bbox brasileiro agora."
    ordered = sorted(states, key=lambda s: s.baro_altitude or 0, reverse=True)[:limite]
    rows = []
    for s in ordered:
        pos = (
            f"{s.latitude:.2f}, {s.longitude:.2f}"
            if s.latitude is not None and s.longitude is not None
            else "—"
        )
        rows.append(
            (
                (s.callsign or s.icao24)[:12],
                (s.origin_country or "—")[:15],
                pos,
                _m_ft(s.baro_altitude),
                _mps_kmh(s.velocity),
                "solo" if s.on_ground else "voo",
            )
        )
    return (
        f"**OpenSky — {len(states)} aeronave(s) no BR agora** "
        f"(top {len(rows)} por altitude)\n\n"
        + markdown_table(
            ["Callsign", "País", "Lat,Lon", "Altitude", "Velocidade", "Situação"],
            rows,
        )
    )


async def rastrear_aeronave(ctx: Context, icao24: str) -> str:
    """Trajetória live de uma aeronave.

    Args:
        icao24: Endereço ICAO 24-bit (6 chars hex).

    Returns:
        Tabela com pontos da trajetória.
    """
    await ctx.info(f"Rastreando {icao24}...")
    pts = await client.track_aircraft(icao24, time_s=0)
    if not pts:
        return f"Sem trajetória para icao24='{icao24}'. Fora de cobertura ADS-B ou em solo."
    rows = []
    for p in pts[-30:]:
        pos = (
            f"{p.latitude:.3f}, {p.longitude:.3f}"
            if p.latitude is not None and p.longitude is not None
            else "—"
        )
        rows.append(
            (
                _epoch_utc(p.time),
                pos,
                _m_ft(p.baro_altitude),
                f"{p.true_track:.0f}°" if p.true_track is not None else "—",
                "solo" if p.on_ground else "voo",
            )
        )
    return (
        f"**Trajetória {icao24}** — {len(pts)} ponto(s), "
        f"exibindo últimos {len(rows)}\n\n"
        + markdown_table(["Tempo (UTC)", "Lat,Lon", "Altitude", "Proa", "Situação"], rows)
    )


async def partidas_aeroporto(ctx: Context, icao_aeroporto: str, horas: int = 6) -> str:
    """Partidas de um aeroporto nas últimas N horas.

    Args:
        icao_aeroporto: Código ICAO (ex: SBGR, SBKP, SBGL, SBBR).
        horas: Janela no passado (padrão 6, máximo 168).

    Returns:
        Tabela callsign x destino x horários.
    """
    horas = max(1, min(horas, 168))
    end = int(_time.time())
    begin = end - horas * 3600
    await ctx.info(f"Partidas de {icao_aeroporto} nas últimas {horas}h...")
    flights = await client.flights_departure(icao_aeroporto, begin, end)
    if not flights:
        return f"Sem partidas de {icao_aeroporto} nas últimas {horas}h."
    rows = [
        (
            (f.callsign or f.icao24)[:12],
            (f.est_arrival_airport or "?")[:6],
            _epoch_utc(f.first_seen),
            _epoch_utc(f.last_seen),
        )
        for f in flights[:100]
    ]
    return f"**Partidas de {icao_aeroporto}** — {len(flights)} voo(s)\n\n" + markdown_table(
        ["Callsign", "Destino ICAO", "Primeira detecção", "Última detecção"],
        rows,
    )


async def chegadas_aeroporto(ctx: Context, icao_aeroporto: str, horas: int = 6) -> str:
    """Chegadas em um aeroporto nas últimas N horas.

    Args:
        icao_aeroporto: Código ICAO.
        horas: Janela (padrão 6, máximo 168).

    Returns:
        Tabela callsign x origem x horários.
    """
    horas = max(1, min(horas, 168))
    end = int(_time.time())
    begin = end - horas * 3600
    await ctx.info(f"Chegadas em {icao_aeroporto} nas últimas {horas}h...")
    flights = await client.flights_arrival(icao_aeroporto, begin, end)
    if not flights:
        return f"Sem chegadas em {icao_aeroporto} nas últimas {horas}h."
    rows = [
        (
            (f.callsign or f.icao24)[:12],
            (f.est_departure_airport or "?")[:6],
            _epoch_utc(f.first_seen),
            _epoch_utc(f.last_seen),
        )
        for f in flights[:100]
    ]
    return f"**Chegadas em {icao_aeroporto}** — {len(flights)} voo(s)\n\n" + markdown_table(
        ["Callsign", "Origem ICAO", "Primeira detecção", "Última detecção"],
        rows,
    )


async def voos_aeronave(ctx: Context, icao24: str, dias: int = 7) -> str:
    """Histórico de voos de uma aeronave nos últimos N dias.

    Args:
        icao24: Endereço ICAO 24-bit (6 chars hex).
        dias: Janela (padrão 7, máximo 30).

    Returns:
        Tabela com origem/destino/horários.
    """
    dias = max(1, min(dias, 30))
    end = int(_time.time())
    begin = end - dias * 86400
    await ctx.info(f"Voos de {icao24} nos últimos {dias} dias...")
    flights = await client.flights_aircraft(icao24, begin, end)
    if not flights:
        return f"Sem voos para {icao24} nos últimos {dias} dias."
    rows = [
        (
            (f.callsign or "—")[:12],
            (f.est_departure_airport or "?")[:6],
            (f.est_arrival_airport or "?")[:6],
            _epoch_utc(f.first_seen),
            _epoch_utc(f.last_seen),
        )
        for f in flights[:60]
    ]
    return f"**Voos de {icao24}** — {len(flights)} voo(s) em {dias} dia(s)\n\n" + markdown_table(
        ["Callsign", "Origem", "Destino", "Decolagem", "Pouso"], rows
    )
