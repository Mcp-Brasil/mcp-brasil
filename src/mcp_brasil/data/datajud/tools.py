"""Tool functions for the DataJud (CNJ) feature.

Rules (ADR-001):
    - tools.py NEVER makes HTTP directly — delegates to client.py
    - Returns formatted strings for LLM consumption
"""

from __future__ import annotations

import contextlib

from mcp_brasil._shared.formatting import markdown_table

from . import client
from .constants import DEFAULT_PAGE_SIZE, MPU_MOV_NOMES, MPU_TIPOS_MEDIDA
from .schemas import Movimentacao


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


async def buscar_processos_avancado(
    tribunal: str = "tjsp",
    classe_codigo: int | None = None,
    orgao_codigo: int | None = None,
    tamanho: int = DEFAULT_PAGE_SIZE,
    search_after: str | None = None,
) -> str:
    """Busca avançada de processos com filtros por código e paginação.

    Permite combinar filtros (classe processual + órgão julgador) e paginar
    resultados grandes usando search_after (cursor do Elasticsearch).

    Use as classes processuais do resource data://classes-processuais
    e os códigos de órgão retornados pelo campo orgaoJulgador.codigo.

    Para paginar: passe o valor de search_after retornado na resposta anterior.

    Args:
        tribunal: Sigla do tribunal (ex: tjsp, trf1, tjdft). Default: tjsp.
        classe_codigo: Código da classe processual (ex: 1116 = Execução Fiscal).
        orgao_codigo: Código do órgão julgador (ex: 13597).
        tamanho: Quantidade de resultados por página (1-10000). Default: 10.
        search_after: Token de paginação retornado pela consulta anterior.

    Returns:
        Tabela com processos e token para próxima página.
    """
    token: list[int] | None = None
    if search_after is not None:
        with contextlib.suppress(ValueError, TypeError):
            token = [int(search_after)]

    processos, next_token = await client.buscar_processos_avancado(
        tribunal=tribunal,
        classe_codigo=classe_codigo,
        orgao_codigo=orgao_codigo,
        tamanho=tamanho,
        search_after=token,
    )

    if not processos:
        return f"Nenhum processo encontrado no {tribunal.upper()} com os filtros informados."

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
    header = f"Processos — {tribunal.upper()} ({len(processos)} resultados):\n\n"
    table = markdown_table(["Número", "Classe", "Assunto", "Órgão Julgador", "Ajuizamento"], rows)

    pagination = ""
    if next_token:
        pagination = f'\n\n**Próxima página:** use search_after="{next_token[0]}"'

    return header + table + pagination


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


# --- MPU (Medidas Protetivas de Urgência) ---


async def buscar_medidas_protetivas(
    tribunal: str = "tjsp",
    data_inicio: str | None = None,
    data_fim: str | None = None,
    lei: str = "ambas",
    size: int = DEFAULT_PAGE_SIZE,
) -> str:
    """Busca Medidas Protetivas de Urgência (MPU) na base do DataJud.

    Pesquisa processos de MPU da Lei Maria da Penha (11.340/2006) e/ou
    Lei Henry Borel (14.344/2022) por tribunal e período.

    A API pública do DataJud NÃO expõe dados de partes (nomes, CPFs).
    Os resultados são estatísticos/processuais.

    Args:
        tribunal: Sigla do tribunal (ex: tjsp, tjpi, tjrj). Default: tjsp.
        data_inicio: Data inicial no formato AAAA-MM-DD.
        data_fim: Data final no formato AAAA-MM-DD.
        lei: Filtro por lei: maria_penha, henry_borel ou ambas. Default: ambas.
        size: Quantidade máxima de resultados (1-100). Default: 10.

    Returns:
        Tabela com MPUs encontradas.
    """
    mpus = await client.buscar_medidas_protetivas(
        tribunal=tribunal,
        data_inicio=data_inicio,
        data_fim=data_fim,
        lei=lei,
        size=size,
    )
    if not mpus:
        return f"Nenhuma MPU encontrada no {tribunal.upper()} para os filtros informados."

    rows = [
        (
            (m.numero or "—")[:25],
            (m.classe_nome or "—")[:40],
            (m.orgao_julgador or "—")[:25],
            (m.data_ajuizamento or "—")[:10],
        )
        for m in mpus
    ]
    header = f"Medidas Protetivas de Urgência — {tribunal.upper()} ({len(mpus)} resultados):\n\n"
    return header + markdown_table(["Número", "Classe", "Órgão Julgador", "Ajuizamento"], rows)


async def buscar_mpu_concedidas(
    tribunal: str = "tjsp",
    data_inicio: str | None = None,
    data_fim: str | None = None,
    destinatario: str = "todos",
    size: int = DEFAULT_PAGE_SIZE,
) -> str:
    """Busca MPUs com decisão de concessão (total ou parcial).

    Filtra processos de Medidas Protetivas que tiveram movimentos de
    concessão. Aceita filtro por destinatário: mulher, idoso, criança ou todos.

    Inclui tanto códigos de movimentos novos (Nov/2024) quanto legados.

    Args:
        tribunal: Sigla do tribunal (ex: tjsp, tjpi). Default: tjsp.
        data_inicio: Data inicial (AAAA-MM-DD).
        data_fim: Data final (AAAA-MM-DD).
        destinatario: Filtro: mulher, idoso, crianca ou todos. Default: todos.
        size: Quantidade máxima de resultados. Default: 10.

    Returns:
        Tabela com MPUs concedidas e movimentos relevantes.
    """
    mpus = await client.buscar_mpu_concedidas(
        tribunal=tribunal,
        data_inicio=data_inicio,
        data_fim=data_fim,
        destinatario=destinatario,
        size=size,
    )
    if not mpus:
        return (
            f"Nenhuma MPU concedida encontrada no {tribunal.upper()} para os filtros informados."
        )

    rows = [
        (
            (m.numero or "—")[:25],
            (m.classe_nome or "—")[:35],
            (m.orgao_julgador or "—")[:25],
            (m.data_ajuizamento or "—")[:10],
            _format_mpu_movimentos(m.movimentos),
        )
        for m in mpus
    ]
    dest_label = f" (destinatário: {destinatario})" if destinatario != "todos" else ""
    header = f"MPUs Concedidas — {tribunal.upper()}{dest_label} ({len(mpus)} resultados):\n\n"
    return header + markdown_table(
        ["Número", "Classe", "Órgão Julgador", "Ajuizamento", "Decisão"],
        rows,
    )


async def buscar_mpu_por_tipo(
    tribunal: str = "tjsp",
    tipo_medida: str = "afastamento_lar",
    data_inicio: str | None = None,
    data_fim: str | None = None,
    size: int = DEFAULT_PAGE_SIZE,
) -> str:
    """Busca MPUs por tipo de medida protetiva.

    Tipos disponíveis: afastamento_lar, proibicao_aproximacao,
    proibicao_contato, proibicao_frequentar, restricao_visitas,
    alimentos_provisorios, reabilitacao_agressor,
    monitoramento_eletronico, outras.

    Args:
        tribunal: Sigla do tribunal. Default: tjsp.
        tipo_medida: Tipo da medida protetiva. Default: afastamento_lar.
        data_inicio: Data inicial (AAAA-MM-DD).
        data_fim: Data final (AAAA-MM-DD).
        size: Quantidade máxima de resultados. Default: 10.

    Returns:
        Tabela com MPUs do tipo informado.
    """
    if tipo_medida not in MPU_TIPOS_MEDIDA:
        return (
            f"Tipo de medida '{tipo_medida}' não reconhecido. "
            f"Opções: {', '.join(sorted(MPU_TIPOS_MEDIDA.keys()))}"
        )

    mpus = await client.buscar_mpu_por_tipo(
        tribunal=tribunal,
        tipo_medida=tipo_medida,
        data_inicio=data_inicio,
        data_fim=data_fim,
        size=size,
    )
    if not mpus:
        return (
            f"Nenhuma MPU do tipo '{tipo_medida}' encontrada no {tribunal.upper()} "
            "para os filtros informados."
        )

    tipo_label = tipo_medida.replace("_", " ").title()
    rows = [
        (
            (m.numero or "—")[:25],
            (m.orgao_julgador or "—")[:25],
            (m.data_ajuizamento or "—")[:10],
            _format_mpu_movimentos(m.movimentos),
        )
        for m in mpus
    ]
    header = f"MPUs — {tipo_label} — {tribunal.upper()} ({len(mpus)} resultados):\n\n"
    return header + markdown_table(["Número", "Órgão Julgador", "Ajuizamento", "Decisão"], rows)


async def estatisticas_mpu(
    tribunal: str | None = None,
    ano: int | None = None,
) -> str:
    """Estatísticas agregadas de Medidas Protetivas de Urgência.

    Retorna totais de MPUs por tribunal, agrupados por classe processual
    e tipo de decisão (concedida, não concedida, revogada, prorrogada).

    Args:
        tribunal: Sigla do tribunal (opcional, default: tjsp).
        ano: Ano para filtrar (ex: 2024). Sem filtro = todos os anos.

    Returns:
        Resumo estatístico com totais e distribuições.
    """
    stats = await client.estatisticas_mpu(tribunal=tribunal, ano=ano)

    lines = [
        f"**Estatísticas de MPU — {(stats.tribunal or 'tjsp').upper()}**",
        f"**Período:** {stats.periodo or 'todos'}",
        f"**Total de processos:** {stats.total}",
    ]

    if stats.por_classe:
        lines.append("\n**Por classe processual:**")
        for classe, count in stats.por_classe.items():
            lines.append(f"  - {classe}: {count}")

    if stats.por_decisao:
        lines.append("\n**Por tipo de decisão:**")
        for decisao, count in stats.por_decisao.items():
            lines.append(f"  - {decisao}: {count}")

    if stats.total == 0:
        lines.append("\n> Nenhuma MPU encontrada. Verifique o tribunal e o período.")

    return "\n".join(lines)


async def timeline_mpu(
    numero_processo: str,
    tribunal: str = "tjsp",
) -> str:
    """Timeline de movimentos de MPU de um processo específico.

    Mostra a sequência de decisões sobre a medida protetiva:
    concessão → prorrogação → revogação, etc.

    Args:
        numero_processo: Número do processo (formato livre).
        tribunal: Sigla do tribunal. Default: tjsp.

    Returns:
        Cronologia dos movimentos de MPU do processo.
    """
    movimentos = await client.timeline_mpu(numero_processo, tribunal)
    if not movimentos:
        return (
            f"Nenhum movimento de MPU encontrado para o processo '{numero_processo}' "
            f"no {tribunal.upper()}."
        )

    rows = [
        (
            (m.data or "—")[:10],
            m.nome or MPU_MOV_NOMES.get(m.codigo or 0, "—"),
            (m.complemento or "—")[:50],
        )
        for m in movimentos
    ]
    header = (
        f"Timeline MPU — Processo {numero_processo} — {tribunal.upper()} "
        f"({len(movimentos)} movimentos):\n\n"
    )
    return header + markdown_table(["Data", "Decisão", "Complemento"], rows)


def _format_mpu_movimentos(movimentos: list[Movimentacao] | None) -> str:
    """Format MPU movements into a short summary string."""
    if not movimentos:
        return "—"
    nomes = [m.nome or MPU_MOV_NOMES.get(m.codigo or 0, "?") for m in movimentos[:3]]
    return "; ".join(nomes)[:60]
