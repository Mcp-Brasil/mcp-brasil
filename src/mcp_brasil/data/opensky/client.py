"""HTTP client for OpenSky Network public API.

API docs: https://openskynetwork.github.io/opensky-api/rest.html
Anonymous rate limit: ~400 req/day per IP.
"""

from __future__ import annotations

from typing import Any

from mcp_brasil._shared.http_client import create_client
from mcp_brasil.exceptions import HttpClientError

from .constants import BRASIL_BBOX, OPENSKY_API_BASE
from .schemas import Flight, StateVector, TrackPoint

_STATE_FIELDS = (
    "icao24",
    "callsign",
    "origin_country",
    "time_position",
    "last_contact",
    "longitude",
    "latitude",
    "baro_altitude",
    "on_ground",
    "velocity",
    "true_track",
    "vertical_rate",
    "sensors",
    "geo_altitude",
    "squawk",
)


def _row_to_state(row: list[Any]) -> StateVector:
    d: dict[str, Any] = {}
    for i, key in enumerate(_STATE_FIELDS):
        if i < len(row) and key in StateVector.model_fields:
            d[key] = row[i]
    callsign = d.get("callsign")
    if isinstance(callsign, str):
        d["callsign"] = callsign.strip() or None
    return StateVector.model_validate(d)


async def states_bbox(lamin: float, lomin: float, lamax: float, lomax: float) -> list[StateVector]:
    """Fetch all aircraft states in a bbox."""
    params = {"lamin": lamin, "lomin": lomin, "lamax": lamax, "lomax": lomax}
    async with create_client(base_url=OPENSKY_API_BASE) as c:
        try:
            r = await c.get("/states/all", params=params)
            r.raise_for_status()
        except Exception as exc:
            raise HttpClientError(f"OpenSky /states/all failed: {exc}") from exc
        data = r.json() or {}
    return [_row_to_state(row) for row in (data.get("states") or [])]


async def states_brasil() -> list[StateVector]:
    return await states_bbox(**BRASIL_BBOX)


async def track_aircraft(icao24: str, time_s: int = 0) -> list[TrackPoint]:
    """Trajectory of one aircraft. time=0 for live."""
    params: dict[str, str | int] = {"icao24": icao24.lower().strip(), "time": time_s}
    async with create_client(base_url=OPENSKY_API_BASE) as c:
        try:
            r = await c.get("/tracks/all", params=params)
            if r.status_code == 404:
                return []
            r.raise_for_status()
        except Exception as exc:
            raise HttpClientError(f"OpenSky /tracks/all failed: {exc}") from exc
        data = r.json() or {}
    pts = []
    for p in data.get("path") or []:
        if len(p) >= 6:
            pts.append(
                TrackPoint(
                    time=p[0],
                    latitude=p[1],
                    longitude=p[2],
                    baro_altitude=p[3],
                    true_track=p[4],
                    on_ground=p[5],
                )
            )
    return pts


async def _fetch_flights(path: str, params: dict[str, Any]) -> list[Flight]:
    async with create_client(base_url=OPENSKY_API_BASE, timeout=30.0) as c:
        try:
            r = await c.get(path, params=params)
            if r.status_code == 404:
                return []
            r.raise_for_status()
        except Exception as exc:
            raise HttpClientError(f"OpenSky {path} failed: {exc}") from exc
        data = r.json() or []
    if not isinstance(data, list):
        return []
    out: list[Flight] = []
    for f in data:
        try:
            out.append(Flight.model_validate(f))
        except Exception:
            continue
    return out


async def flights_aircraft(icao24: str, begin: int, end: int) -> list[Flight]:
    return await _fetch_flights(
        "/flights/aircraft",
        {"icao24": icao24.lower().strip(), "begin": begin, "end": end},
    )


async def flights_departure(airport_icao: str, begin: int, end: int) -> list[Flight]:
    return await _fetch_flights(
        "/flights/departure",
        {"airport": airport_icao.upper().strip(), "begin": begin, "end": end},
    )


async def flights_arrival(airport_icao: str, begin: int, end: int) -> list[Flight]:
    return await _fetch_flights(
        "/flights/arrival",
        {"airport": airport_icao.upper().strip(), "begin": begin, "end": end},
    )
