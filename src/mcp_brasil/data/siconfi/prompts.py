"""Prompts for the SICONFI feature — fiscal analysis templates."""

from __future__ import annotations


def analise_fiscal_municipio(municipio: str, exercicio: int = 2024) -> str:
    """Gera uma análise fiscal resumida de um município com base em RREO/RGF.

    Args:
        municipio: Nome do município (ex: "São Paulo").
        exercicio: Ano da análise.
    """
    return (
        f"Faça uma análise fiscal do município {municipio} para {exercicio}.\n\n"
        "Passos:\n"
        f"1. listar_entes(uf='...') para obter o cod_ibge do município '{municipio}'\n"
        f"2. consultar_rreo(exercicio={exercicio}, periodo=6, ente_id=<cod>, "
        f"anexo='RREO-Anexo 03') — Receita Corrente Líquida\n"
        f"3. consultar_rgf(exercicio={exercicio}, periodo=3, ente_id=<cod>, "
        f"poder='E', anexo='RGF-Anexo 01') — Despesa com Pessoal vs. limite LRF\n"
        f"4. consultar_rgf(exercicio={exercicio}, periodo=3, ente_id=<cod>, "
        f"poder='E', anexo='RGF-Anexo 02') — Dívida Consolidada\n"
        "5. Calcule a % de despesa de pessoal sobre RCL (limite prudencial LRF: "
        "51,3% Executivo municipal)\n\n"
        "Apresente: RCL, despesa de pessoal (R$ e %), dívida/RCL, alertas de limite."
    )


def comparar_entes(entes_ids: str, exercicio: int = 2024) -> str:
    """Compara indicadores fiscais entre entes.

    Args:
        entes_ids: Códigos IBGE separados por vírgula (ex: "3550308,3304557").
        exercicio: Ano de referência.
    """
    return (
        f"Compare fiscalmente os entes {entes_ids} para {exercicio}.\n\n"
        "Para cada ente:\n"
        f"- consultar_rreo(exercicio={exercicio}, periodo=6, ente_id=..., "
        "anexo='RREO-Anexo 03') — RCL\n"
        f"- consultar_rgf(exercicio={exercicio}, periodo=3, ente_id=..., "
        "poder='E', anexo='RGF-Anexo 01') — despesa com pessoal\n"
        f"- consultar_rgf(exercicio={exercicio}, periodo=3, ente_id=..., "
        "poder='E', anexo='RGF-Anexo 02') — dívida consolidada\n\n"
        "Consolide em uma tabela comparativa e destaque quem está mais próximo "
        "dos limites prudenciais da LRF."
    )
