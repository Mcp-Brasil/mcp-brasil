"""MCP Prompts for auditing and analyzing Brazilian municipal and state transparency."""

from __future__ import annotations


def auditoria_transparencia_municipio(municipio: str, uf: str = "") -> str:
    """Gera um roteiro completo de auditoria de transparência para um município brasileiro.

    Args:
        municipio: Nome do município ou código IBGE (ex: "Campinas", "3550308").
        uf: Sigla do estado (ex: "SP").
    """
    uf_str = f"({uf})" if uf else ""
    return f"""Você é um auditor especialista em Transparência Pública no Brasil.
Execute a auditoria dos dados de transparência do município de {municipio} {uf_str}.

Passos a executar:
1. Use `transparencia_municipios_conectar_municipio` para obter o dossiê de transparência.
2. Identifique o Tribunal de Contas responsável (TCE ou TCM) e os portais municipais.
3. Consulte a situação fiscal no SICONFI (relatórios RREO, RGF e balanço anual DCA).
4. Verifique editais e contratos recentes no PNCP (Compras Públicas).
5. Verifique repasses federais e transferências voluntárias no Transferegov.
6. Avalie o cumprimento da Lei de Acesso à Informação (LAI) e LRF (LC 101/2000).
"""


def comparativo_transparencia_regional(alvos: str) -> str:
    """Gera roteiro de análise comparativa de transparência entre estados ou municípios.

    Args:
        alvos: Lista de municípios ou UFs separados por vírgula (ex: "SP, RJ, MG").
    """
    return f"""Atue como analista de políticas públicas e transparência federativa no Brasil.
Compare os entes governamentais solicitados: {alvos}.

Passos a executar:
1. Use `transparencia_municipios_comparar` passando os entes informados.
2. Compare os portais de transparência adotados e os órgãos fiscalizadores (TCEs/TCMs).
3. Avalie a facilidade de acesso a dados abertos em cada ente.
4. Identifique boas práticas e pontos de atenção com base no Radar Nacional (ATRICON).

"""
