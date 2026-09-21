"""Analysis prompts for the TCM-GO feature."""

from __future__ import annotations


def analisar_municipio_goias(municipio: str) -> str:
    """Análise de um município goiano nas superfícies públicas do TCM-GO.

    Orienta a consultar processos no eSiproc e acórdãos no TCMJuris de um
    município de Goiás, consolidando o resultado.

    Args:
        municipio: Nome do município (ex: "ANÁPOLIS").
    """
    return (
        f"Analise a situação do município {municipio} no TCM-GO:\n\n"
        "1. Use `listar_valores_campo` com campo `tipo_protocolo.keyword` para "
        "conhecer os tipos de processo disponíveis\n"
        f'2. Use `buscar_processos` com municipio="{municipio}" para listar os '
        "processos eletrônicos recentes\n"
        f'3. Use `buscar_acordaos` com municipio="{municipio}" para localizar os '
        "acórdãos publicados, com o link do PDF de cada um\n\n"
        "Apresente um resumo consolidado com:\n"
        "- Volume e tipos de processos do município\n"
        "- Processos finalizados e em andamento\n"
        "- Acórdãos publicados e seus assuntos\n"
    )
