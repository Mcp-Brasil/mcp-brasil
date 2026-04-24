"""MCP tools for the SICONFI feature.

Delega todo acesso HTTP para ``client.py`` (ADR-001 rule #2).
"""

from __future__ import annotations

from mcp_brasil._shared.formatting import format_brl, markdown_table

from . import client
from .constants import (
    ANEXOS_RGF_POPULARES,
    ANEXOS_RREO_POPULARES,
    ESFERAS,
    PERIODOS_RREO_BIMESTRAIS,
)
from .schemas import ItemDeclaracao


async def listar_entes(uf: str | None = None, tipo: str | None = None) -> str:
    """Lista entes federados declarantes do SICONFI (União, estados, DF, municípios).

    Retorna nome, código IBGE, UF, esfera e população. Use esse cód. IBGE como
    ``id_ente`` nas demais tools (consultar_rreo, consultar_rgf, consultar_dca).

    Args:
        uf: Filtra por UF (ex: "SP"). Opcional.
        tipo: Filtra por esfera — "U" União, "E" Estado, "M" Município, "D" DF.
    """
    entes = await client.listar_entes()
    if uf:
        entes = [e for e in entes if e.uf.upper() == uf.upper()]
    if tipo:
        entes = [e for e in entes if e.esfera.upper() == tipo.upper()]
    if not entes:
        return "Nenhum ente encontrado com os filtros fornecidos."

    rows = [
        [e.cod_ibge, e.ente, e.uf, ESFERAS.get(e.esfera, e.esfera), e.populacao or ""]
        for e in entes[:500]
    ]
    header = f"{len(entes)} entes encontrados (exibindo até 500).\n\n"
    return header + markdown_table(["cod_ibge", "ente", "uf", "esfera", "populacao"], rows)


def _format_itens(itens: list[ItemDeclaracao], limit: int = 100) -> str:
    if not itens:
        return "Nenhum item retornado. Verifique parâmetros (ano, período, ente, anexo)."
    rows = []
    for i in itens[:limit]:
        valor = format_brl(i.valor) if i.valor is not None else ""
        rows.append(
            [
                i.anexo or "",
                i.conta or i.rotulo or "",
                i.coluna or "",
                valor,
            ]
        )
    total = len(itens)
    header = f"{total} linhas" + (f" (exibindo {limit})" if total > limit else "") + "\n\n"
    return header + markdown_table(["anexo", "conta", "coluna", "valor"], rows)


async def consultar_rreo(
    exercicio: int,
    periodo: int,
    ente_id: int,
    anexo: str | None = None,
    simplificado: bool = False,
    esfera: str | None = None,
) -> str:
    """Consulta RREO (Relatório Resumido da Execução Orçamentária) de um ente.

    RREO é publicado bimestralmente (6 períodos/ano) e consolida receitas e
    despesas. Use para análises de execução orçamentária municipal/estadual.

    Args:
        exercicio: Ano do relatório (ex: 2024).
        periodo: Bimestre (1-6). Ex: 1=Jan-Fev, 6=Nov-Dez.
        ente_id: Código IBGE do ente (obter via listar_entes).
        anexo: Anexo específico (ex: "RREO-Anexo 03"). Omita para retornar todos.
        simplificado: Se True, usa "RREO Simplificado" (prefeituras < 50k hab).
        esfera: Filtra por esfera (U/E/M/D).
    """
    tipo = "RREO Simplificado" if simplificado else "RREO"
    itens = await client.consultar_rreo(
        an_exercicio=exercicio,
        nr_periodo=periodo,
        co_tipo_demonstrativo=tipo,
        id_ente=ente_id,
        no_anexo=anexo,
        co_esfera=esfera,
    )
    periodo_lbl = PERIODOS_RREO_BIMESTRAIS.get(periodo, str(periodo))
    header = f"RREO {exercicio} / {periodo_lbl} / ente {ente_id}\n\n"
    return header + _format_itens(itens)


async def consultar_rgf(
    exercicio: int,
    periodo: int,
    ente_id: int,
    poder: str = "E",
    periodicidade: str = "Q",
    anexo: str | None = None,
    simplificado: bool = False,
    esfera: str | None = None,
) -> str:
    """Consulta RGF (Relatório de Gestão Fiscal — LRF) de um ente.

    RGF verifica limites da LRF: despesa de pessoal, dívida consolidada,
    garantias e operações de crédito.

    Args:
        exercicio: Ano do relatório.
        periodo: Quadrimestre (1-3) ou semestre (1-2).
        ente_id: Código IBGE do ente.
        poder: E=Executivo (padrão), L=Legislativo, J=Judiciário, M=MP, D=Defensoria.
        periodicidade: Q=quadrimestral (padrão), S=semestral.
        anexo: Anexo específico (ex: "RGF-Anexo 01" para despesa com pessoal).
        simplificado: Se True, usa "RGF Simplificado".
        esfera: Filtra por esfera.
    """
    tipo = "RGF Simplificado" if simplificado else "RGF"
    itens = await client.consultar_rgf(
        an_exercicio=exercicio,
        in_periodicidade=periodicidade,
        nr_periodo=periodo,
        co_tipo_demonstrativo=tipo,
        co_poder=poder,
        id_ente=ente_id,
        no_anexo=anexo,
        co_esfera=esfera,
    )
    header = f"RGF {exercicio} / {periodicidade}{periodo} / ente {ente_id} / poder {poder}\n\n"
    return header + _format_itens(itens)


async def consultar_dca(
    exercicio: int,
    ente_id: int,
    anexo: str | None = None,
) -> str:
    """Consulta DCA (Declaração de Contas Anuais) de um ente.

    DCA é a prestação anual detalhada — receitas, despesas por função,
    ativos, passivos, patrimônio.

    Args:
        exercicio: Ano de referência.
        ente_id: Código IBGE do ente.
        anexo: Anexo específico (ex: "DCA-Anexo I-AB" para balanço orçamentário).
    """
    itens = await client.consultar_dca(an_exercicio=exercicio, id_ente=ente_id, no_anexo=anexo)
    header = f"DCA {exercicio} / ente {ente_id}\n\n"
    return header + _format_itens(itens)


async def extrato_entregas(ente_id: int, ano: int) -> str:
    """Lista situação das entregas declaratórias (RREO, RGF, DCA) de um ente no ano.

    Útil para checar quais relatórios já foram declarados antes de consultá-los.

    Args:
        ente_id: Código IBGE do ente.
        ano: Exercício de referência.
    """
    itens = await client.consultar_extrato_entregas(id_ente=ente_id, an_referencia=ano)
    if not itens:
        return f"Nenhuma entrega registrada para ente {ente_id} em {ano}."
    rows = [
        [i.periodo or "", i.periodicidade or "", i.rotulo or "", i.conta or ""]
        for i in itens[:200]
    ]
    header = f"{len(itens)} entregas para ente {ente_id} em {ano}\n\n"
    return header + markdown_table(["periodo", "periodicidade", "rotulo", "status"], rows)


async def listar_anexos_relatorios() -> str:
    """Catálogo de anexos disponíveis (RREO/RGF/DCA) por esfera — referência."""
    anexos = await client.listar_anexos()
    if not anexos:
        return "Catálogo vazio."
    rows = [
        [a.esfera, a.co_tipo_demonstrativo, a.no_anexo, a.de_anexo or ""] for a in anexos[:500]
    ]
    return markdown_table(["esfera", "tipo", "anexo", "descricao"], rows)


async def anexos_populares() -> str:
    """Lista os anexos RREO e RGF mais consultados (referência rápida, offline)."""
    out = ["## Anexos RREO populares\n"]
    for code, desc in ANEXOS_RREO_POPULARES.items():
        out.append(f"- `{code}` — {desc}")
    out.append("\n## Anexos RGF populares\n")
    for code, desc in ANEXOS_RGF_POPULARES.items():
        out.append(f"- `{code}` — {desc}")
    return "\n".join(out)
