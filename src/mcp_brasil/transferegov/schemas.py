"""Pydantic models for TransfereGov API responses."""

from __future__ import annotations

from pydantic import BaseModel


class TransferenciaEspecial(BaseModel):
    """Transferência especial (emenda pix)."""

    id_transferencia_especial: int | None = None
    ano_exercicio: int | None = None
    nr_emenda: str | None = None
    autor_emenda: str | None = None
    tipo_emenda: str | None = None
    funcao: str | None = None
    subfuncao: str | None = None
    valor_empenhado: float | None = None
    valor_liquidado: float | None = None
    valor_pago: float | None = None
    nm_municipio_beneficiario: str | None = None
    uf_beneficiario: str | None = None
    nm_entidade_beneficiaria: str | None = None
    objeto: str | None = None
