"""Resources for Brazilian municipal and state transparency."""

from __future__ import annotations

import json

from . import client


async def estados_brasileiros() -> str:
    """Retorna a lista completa das 27 Unidades Federativas em formato JSON."""
    estados = await client.listar_estados()
    return json.dumps([e.model_dump() for e in estados], ensure_ascii=False, indent=2)


async def estatisticas_nacionais_resource() -> str:
    """Retorna estatísticas consolidadas dos municípios e estados brasileiros em JSON."""
    stats = await client.obter_estatisticas()
    return json.dumps(stats.model_dump(), ensure_ascii=False, indent=2)


async def capitais_brasileiras() -> str:
    """Retorna as 27 capitais brasileiras com códigos IBGE e SIAFI em JSON."""
    res = await client.buscar_municipios(query="", limit=6000)
    capitais = [m.model_dump() for m in res.municipios if m.capital]
    return json.dumps(capitais, ensure_ascii=False, indent=2)


async def megasoft_municipios_resource() -> str:
    """Retorna os 268 municípios brasileiros atendidos pela tecnologia Megasoft em JSON."""
    munis = await client.listar_municipios_megasoft()
    return json.dumps([m.model_dump() for m in munis], ensure_ascii=False, indent=2)
