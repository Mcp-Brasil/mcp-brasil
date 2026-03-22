"""Static reference data for the Compras feature."""

from __future__ import annotations

import json

from .constants import MODALIDADES


def modalidades_licitacao() -> str:
    """Catálogo de modalidades de licitação conforme Lei 14.133/2021."""
    data = [{"id": k, "nome": v} for k, v in MODALIDADES.items()]
    return json.dumps(data, ensure_ascii=False, indent=2)
