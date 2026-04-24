"""Resources for the SICONFI feature — reference catalogs."""

from __future__ import annotations

import json

from .constants import (
    ANEXOS_RGF_POPULARES,
    ANEXOS_RREO_POPULARES,
    ESFERAS,
    PERIODOS_RREO_BIMESTRAIS,
    TIPO_ENTE,
)


def catalogo_anexos() -> str:
    """Catálogo de anexos RREO e RGF mais consultados, com descrições."""
    return json.dumps(
        {
            "rreo": ANEXOS_RREO_POPULARES,
            "rgf": ANEXOS_RGF_POPULARES,
        },
        ensure_ascii=False,
    )


def referencia_codigos() -> str:
    """Códigos de esfera, tipo de ente e bimestres usados pela API."""
    return json.dumps(
        {
            "esferas": ESFERAS,
            "tipos_ente": TIPO_ENTE,
            "bimestres_rreo": PERIODOS_RREO_BIMESTRAIS,
            "periodicidade_rgf": {"Q": "Quadrimestral", "S": "Semestral"},
            "poderes": {
                "E": "Executivo",
                "L": "Legislativo",
                "J": "Judiciário",
                "M": "Ministério Público",
                "D": "Defensoria",
            },
        },
        ensure_ascii=False,
    )
