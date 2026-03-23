"""Prompts for the TransfereGov feature — analysis templates for LLMs."""

from __future__ import annotations


def analise_emendas_pix(ano: int, uf: str = "") -> str:
    """Analisa emendas pix (transferências especiais) de um ano.

    Cria um template que orienta o LLM a consultar e analisar
    emendas parlamentares do tipo transferência especial.

    Args:
        ano: Ano de exercício a analisar (ex: 2024).
        uf: UF para filtrar análise (opcional, ex: PI, SP).
    """
    filtro_uf = f" no estado {uf.upper()}" if uf else ""
    return (
        f"Analise as emendas pix (transferências especiais) do ano {ano}"
        f"{filtro_uf} usando o TransfereGov.\n\n"
        "Passos:\n"
        f"1. Use buscar_emendas_pix(ano={ano}"
        + (f', uf="{uf.upper()}"' if uf else "")
        + ") para listar as emendas\n"
        "2. Navegue pelas páginas para ter uma visão ampla\n"
        "3. Para emendas relevantes, use detalhe_emenda(id) para mais informações\n\n"
        "Apresente:\n"
        "- Volume total de emendas pix e valores\n"
        "- Principais autores (parlamentares com mais emendas/valor)\n"
        "- Distribuição por UF e município\n"
        "- Áreas temáticas (funções/subfunções) mais contempladas\n"
        "- Observações sobre concentração de recursos"
    )
