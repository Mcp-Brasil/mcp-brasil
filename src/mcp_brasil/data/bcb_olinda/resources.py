"""Resources for bcb_olinda."""

from __future__ import annotations

import json

from .constants import INDICADORES_FOCUS


def catalogo_indicadores_focus() -> str:
    return json.dumps(INDICADORES_FOCUS, ensure_ascii=False)
