"""Prompts for bcb_olinda."""

from __future__ import annotations


def panorama_focus() -> str:
    """Panorama das expectativas Focus — inflação, juros, câmbio, PIB."""
    return (
        "Dê um panorama das expectativas de mercado (Focus).\n\n"
        "1. focus_anual('IPCA') — inflação anual projetada\n"
        "2. focus_selic() — trajetória esperada da Selic\n"
        "3. focus_anual('PIB Total') — crescimento projetado\n"
        "4. focus_anual('Câmbio') — R$/US$ projetado\n\n"
        "Sintetize: quem revisou? Para cima ou baixo? Principais mudanças vs. semana passada."
    )
