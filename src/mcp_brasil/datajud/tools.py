"""Tool functions for the DataJud (CNJ) feature.

Rules (ADR-001):
    - tools.py NEVER makes HTTP directly — delegates to client.py
    - Returns formatted strings for LLM consumption
"""

from __future__ import annotations

from mcp_brasil._shared.formatting import markdown_table

from . import client
from .constants import DEFAULT_PAGE_SIZE


async def buscar_processos(
    query: str,
    tribunal: str = "tjsp",
    tamanho: int = DEFAULT_PAGE_SIZE,
) -> str:
    """Busca processos judiciais na API pública do DataJud (CNJ).

    Pesquisa por texto livre: CPF, CNPJ, nome da parte, número do processo
    ou qualquer termo relacionado ao processo.

    Requer a variável de ambiente DATAJUD_API_KEY configurada.
    Cadastre-se em: https://datajud.cnj.jus.br

    Args:
        query: Termo de busca (CPF, CNPJ, nome, número do processo).
        tribunal: Sigla do tribunal (ex: tjsp, trf1, stj). Default: tjsp.
        tamanho: Quantidade máxima de resultados (1-100). Default: 10.

    Returns:
        Tabela com processos encontrados.
    """
    processos = await client.buscar_processos(query, tribunal, tamanho)
    if not processos:
        return f"Nenhum processo encontrado para '{query}' no {tribunal.upper()}."

    rows = [
        (
            (p.numero or "—")[:25],
            (p.classe or "—")[:30],
            (p.assunto or "—")[:30],
            (p.orgao_julgador or "—")[:25],
            (p.data_ajuizamento or "—")[:10],
        )
        for p in processos
    ]
    header = f"Processos encontrados no {tribunal.upper()} ({len(processos)} resultados):\n\n"
    return header + markdown_table(
        ["Número", "Classe", "Assunto", "Órgão Julgador", "Ajuizamento"], rows
    )


async def buscar_processo_por_numero(
    numero_processo: str,
    tribunal: str = "tjsp",
) -> str:
    """Busca um processo específico pelo número unificado (NPU).

    Retorna detalhes completos incluindo partes, assuntos e movimentações.

    Args:
        numero_processo: Número do processo (formato livre, ex: 0001234-56.2024.8.26.0100).
        tribunal: Sigla do tribunal (ex: tjsp, trf1, stj). Default: tjsp.

    Returns:
        Detalhes do processo com partes e movimentações.
    """
    detalhe = await client.buscar_processo_por_numero(numero_processo, tribunal)
    if detalhe is None:
        return f"Processo '{numero_processo}' não encontrado no {tribunal.upper()}."

    lines = [
        f"**Processo:** {detalhe.numero or '—'}",
        f"**Classe:** {detalhe.classe or '—'}",
        f"**Tribunal:** {detalhe.tribunal or tribunal.upper()}",
        f"**Órgão Julgador:** {detalhe.orgao_julgador or '—'}",
        f"**Ajuizamento:** {detalhe.data_ajuizamento or '—'}",
        f"**Última atualização:** {detalhe.data_ultima_atualizacao or '—'}",
        f"**Grau:** {detalhe.grau or '—'}",
    ]

    # Assuntos
    if detalhe.assuntos:
        assuntos = [a.nome or "—" for a in detalhe.assuntos]
        lines.append(f"\n**Assuntos:** {', '.join(assuntos)}")

    # Partes
    if detalhe.partes:
        lines.append("\n**Partes:**")
        for parte in detalhe.partes[:20]:
            lines.append(f"  - [{parte.polo or '—'}] {parte.nome or '—'}")

    # Movimentações (últimas 10)
    if detalhe.movimentacoes:
        lines.append(f"\n**Últimas movimentações** ({len(detalhe.movimentacoes)}):")
        for mov in detalhe.movimentacoes[:10]:
            data = (mov.data or "—")[:10]
            nome = mov.nome or "—"
            lines.append(f"  - {data}: {nome}")

    return "\n".join(lines)


async def buscar_processos_por_classe(
    classe: str,
    tribunal: str = "tjsp",
    tamanho: int = DEFAULT_PAGE_SIZE,
) -> str:
    """Busca processos por classe processual.

    Exemplos de classes: Ação Civil Pública, Mandado de Segurança,
    Habeas Corpus, Execução Fiscal, Recurso Extraordinário.

    Args:
        classe: Nome da classe processual (ex: Mandado de Segurança).
        tribunal: Sigla do tribunal. Default: tjsp.
        tamanho: Quantidade máxima de resultados (1-100). Default: 10.

    Returns:
        Tabela com processos da classe informada.
    """
    processos = await client.buscar_processos_por_classe(classe, tribunal, tamanho)
    if not processos:
        return f"Nenhum processo da classe '{classe}' encontrado no {tribunal.upper()}."

    rows = [
        (
            (p.numero or "—")[:25],
            (p.assunto or "—")[:35],
            (p.orgao_julgador or "—")[:25],
            (p.data_ajuizamento or "—")[:10],
        )
        for p in processos
    ]
    header = f"Processos — {classe} — {tribunal.upper()} ({len(processos)} resultados):\n\n"
    return header + markdown_table(["Número", "Assunto", "Órgão Julgador", "Ajuizamento"], rows)


async def buscar_processos_por_assunto(
    assunto: str,
    tribunal: str = "tjsp",
    tamanho: int = DEFAULT_PAGE_SIZE,
) -> str:
    """Busca processos por assunto/tema.

    Exemplos: Direito do Consumidor, Direito Ambiental, Dano Moral,
    Execução de Título Extrajudicial.

    Args:
        assunto: Assunto ou tema do processo.
        tribunal: Sigla do tribunal. Default: tjsp.
        tamanho: Quantidade máxima de resultados (1-100). Default: 10.

    Returns:
        Tabela com processos do assunto informado.
    """
    processos = await client.buscar_processos_por_assunto(assunto, tribunal, tamanho)
    if not processos:
        return f"Nenhum processo sobre '{assunto}' encontrado no {tribunal.upper()}."

    rows = [
        (
            (p.numero or "—")[:25],
            (p.classe or "—")[:25],
            (p.orgao_julgador or "—")[:25],
            (p.data_ajuizamento or "—")[:10],
        )
        for p in processos
    ]
    header = (
        f"Processos — assunto: {assunto} — {tribunal.upper()} ({len(processos)} resultados):\n\n"
    )
    return header + markdown_table(["Número", "Classe", "Órgão Julgador", "Ajuizamento"], rows)


async def buscar_processos_por_orgao(
    orgao_julgador: str,
    tribunal: str = "tjsp",
    tamanho: int = DEFAULT_PAGE_SIZE,
) -> str:
    """Busca processos por órgão julgador (vara, câmara, turma).

    Exemplos: 1ª Vara Cível, 3ª Câmara de Direito Privado,
    1ª Turma Recursal.

    Args:
        orgao_julgador: Nome do órgão julgador.
        tribunal: Sigla do tribunal. Default: tjsp.
        tamanho: Quantidade máxima de resultados (1-100). Default: 10.

    Returns:
        Tabela com processos do órgão informado.
    """
    processos = await client.buscar_processos_por_orgao(orgao_julgador, tribunal, tamanho)
    if not processos:
        return f"Nenhum processo encontrado no órgão '{orgao_julgador}' do {tribunal.upper()}."

    rows = [
        (
            (p.numero or "—")[:25],
            (p.classe or "—")[:25],
            (p.assunto or "—")[:30],
            (p.data_ajuizamento or "—")[:10],
        )
        for p in processos
    ]
    header = (
        f"Processos — {orgao_julgador} — {tribunal.upper()} ({len(processos)} resultados):\n\n"
    )
    return header + markdown_table(["Número", "Classe", "Assunto", "Ajuizamento"], rows)


async def consultar_movimentacoes(
    numero_processo: str,
    tribunal: str = "tjsp",
) -> str:
    """Consulta movimentações de um processo judicial.

    Retorna o histórico de andamentos (despachos, decisões, audiências, etc.).

    Args:
        numero_processo: Número do processo (formato livre).
        tribunal: Sigla do tribunal. Default: tjsp.

    Returns:
        Lista cronológica de movimentações.
    """
    movimentacoes = await client.consultar_movimentacoes(numero_processo, tribunal)
    if not movimentacoes:
        return (
            f"Nenhuma movimentação encontrada para o processo '{numero_processo}' "
            f"no {tribunal.upper()}."
        )

    rows = [
        (
            (m.data or "—")[:10],
            (m.nome or "—")[:40],
            (m.complemento or "—")[:40],
        )
        for m in movimentacoes
    ]
    header = (
        f"Movimentações do processo {numero_processo} — {tribunal.upper()} "
        f"({len(movimentacoes)} movimentações):\n\n"
    )
    return header + markdown_table(["Data", "Movimentação", "Complemento"], rows)
