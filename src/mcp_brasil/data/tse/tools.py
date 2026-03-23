"""Tool functions for the TSE (Tribunal Superior Eleitoral) feature.

Rules (ADR-001):
    - tools.py NEVER makes HTTP directly — delegates to client.py
    - Returns formatted strings for LLM consumption
"""

from __future__ import annotations

from fastmcp import Context

from mcp_brasil._shared.formatting import format_brl, format_number_br, markdown_table

from . import client
from .constants import CARGO_CODES_CDN


async def anos_eleitorais() -> str:
    """Lista os anos com dados eleitorais disponíveis no TSE.

    Retorna todos os anos em que houve eleições com dados registrados.

    Returns:
        Lista de anos eleitorais disponíveis.
    """
    anos = await client.anos_eleitorais()
    if not anos:
        return "Nenhum ano eleitoral disponível."
    return "Anos eleitorais disponíveis: " + ", ".join(str(a) for a in sorted(anos))


async def listar_eleicoes() -> str:
    """Lista todas as eleições ordinárias registradas no TSE.

    Inclui eleições municipais, estaduais e federais de todos os anos.

    Returns:
        Tabela com eleições disponíveis.
    """
    eleicoes = await client.listar_eleicoes()
    if not eleicoes:
        return "Nenhuma eleição encontrada."

    rows = [
        (
            str(e.id or "—"),
            str(e.ano or "—"),
            (e.nome or "—")[:40],
            e.tipo or "—",
            e.tipo_abrangencia or "—",
            e.data_eleicao or "—",
        )
        for e in eleicoes
    ]
    return f"Eleições ordinárias ({len(eleicoes)} registros):\n\n" + markdown_table(
        ["ID", "Ano", "Nome", "Tipo", "Abrangência", "Data"], rows
    )


async def listar_eleicoes_suplementares(ano: int, uf: str) -> str:
    """Lista eleições suplementares de um estado em um ano específico.

    Eleições suplementares ocorrem quando eleições regulares são anuladas
    ou quando há vacância de cargo eletivo.

    Args:
        ano: Ano da eleição (ex: 2020, 2022).
        uf: Sigla do estado (ex: SP, RJ, MG).

    Returns:
        Tabela com eleições suplementares encontradas.
    """
    eleicoes = await client.listar_eleicoes_suplementares(ano, uf)
    if not eleicoes:
        return f"Nenhuma eleição suplementar encontrada para {uf.upper()} em {ano}."

    rows = [
        (
            str(e.id or "—"),
            str(e.ano or "—"),
            (e.nome or "—")[:40],
            e.tipo or "—",
            e.data_eleicao or "—",
        )
        for e in eleicoes
    ]
    return f"Eleições suplementares {uf.upper()} {ano} ({len(eleicoes)}):\n\n" + markdown_table(
        ["ID", "Ano", "Nome", "Tipo", "Data"], rows
    )


async def listar_estados_suplementares(ano: int) -> str:
    """Lista estados que tiveram eleições suplementares em um ano.

    Útil para descobrir quais UFs tiveram eleições suplementares antes
    de consultar os detalhes com listar_eleicoes_suplementares().

    Args:
        ano: Ano para consulta (ex: 2020, 2022).

    Returns:
        Lista de siglas de estados com eleições suplementares.
    """
    estados = await client.listar_estados_suplementares(ano)
    if not estados:
        return f"Nenhum estado com eleição suplementar em {ano}."
    return f"Estados com eleições suplementares em {ano}: {', '.join(sorted(estados))}"


async def listar_cargos(eleicao_id: int, municipio: int) -> str:
    """Lista os cargos disponíveis em um município para uma eleição.

    Use o ID da eleição obtido com listar_eleicoes() e o código
    do município (código IBGE ou TSE).

    Args:
        eleicao_id: ID da eleição (ex: 2030402020).
        municipio: Código do município (ex: 35157 para Feira de Santana).

    Returns:
        Tabela com cargos e quantidade de candidatos.
    """
    cargos = await client.listar_cargos(eleicao_id, municipio)
    if not cargos:
        return "Nenhum cargo encontrado para esta eleição/município."

    rows = [
        (
            str(c.codigo or "—"),
            c.nome or "—",
            "Sim" if c.titular else "Não",
            str(c.contagem or 0),
        )
        for c in cargos
    ]
    return f"Cargos disponíveis ({len(cargos)}):\n\n" + markdown_table(
        ["Código", "Cargo", "Titular", "Candidatos"], rows
    )


async def listar_candidatos(
    ano: int,
    municipio: int,
    eleicao_id: int,
    cargo: int,
) -> str:
    """Lista candidatos para um cargo em um município.

    Requer os IDs de eleição e cargo obtidos com listar_eleicoes()
    e listar_cargos().

    Args:
        ano: Ano da eleição (ex: 2020, 2022).
        municipio: Código do município.
        eleicao_id: ID da eleição.
        cargo: Código do cargo (ex: 11=Prefeito, 13=Vereador).

    Returns:
        Tabela com candidatos e suas situações.
    """
    candidatos = await client.listar_candidatos(ano, municipio, eleicao_id, cargo)
    if not candidatos:
        return "Nenhum candidato encontrado para os filtros informados."

    rows = [
        (
            str(c.id or "—"),
            (c.nome_urna or "—")[:30],
            str(c.numero or "—"),
            c.partido or "—",
            c.situacao or "—",
        )
        for c in candidatos
    ]
    return f"Candidatos ({len(candidatos)}):\n\n" + markdown_table(
        ["ID", "Nome de Urna", "Número", "Partido", "Situação"], rows
    )


async def buscar_candidato(
    ano: int,
    municipio: int,
    eleicao_id: int,
    candidato_id: int,
) -> str:
    """Busca detalhes completos de um candidato.

    Retorna informações pessoais, eleitorais, bens declarados
    e situação da candidatura.

    Args:
        ano: Ano da eleição.
        municipio: Código do município.
        eleicao_id: ID da eleição.
        candidato_id: ID do candidato (obtido com listar_candidatos).

    Returns:
        Ficha completa do candidato.
    """
    cand = await client.buscar_candidato(ano, municipio, eleicao_id, candidato_id)
    if cand is None:
        return "Candidato não encontrado."

    lines = [
        f"**Nome de urna:** {cand.nome_urna or '—'}",
        f"**Nome completo:** {cand.nome_completo or '—'}",
        f"**Número:** {cand.numero or '—'}",
        f"**Partido:** {cand.partido or '—'}",
        f"**Coligação:** {cand.coligacao or '—'}",
        f"**Situação:** {cand.situacao or '—'}",
        f"**Situação do candidato:** {cand.situacao_candidato or '—'}",
    ]

    # Totalização
    if cand.descricao_totalizacao:
        lines.append(f"**Resultado:** {cand.descricao_totalizacao}")
    if cand.total_votos is not None:
        lines.append(f"**Total de votos:** {format_number_br(cand.total_votos, 0)}")

    # Dados pessoais
    lines.append("\n**Dados pessoais:**")
    lines.append(f"  Sexo: {cand.sexo or '—'}")
    lines.append(f"  Cor/Raça: {cand.cor_raca or '—'}")
    lines.append(f"  Estado civil: {cand.estado_civil or '—'}")
    lines.append(f"  Escolaridade: {cand.grau_instrucao or '—'}")
    lines.append(f"  Ocupação: {cand.ocupacao or '—'}")
    lines.append(f"  Naturalidade: {cand.municipio_nascimento or '—'}/{cand.uf_nascimento or '—'}")

    # Patrimônio
    if cand.total_bens is not None:
        lines.append(f"\n**Total de bens declarados:** {format_brl(cand.total_bens)}")

    if cand.gasto_campanha is not None and cand.gasto_campanha > 0:
        lines.append(f"**Gasto de campanha:** {format_brl(cand.gasto_campanha)}")

    # Ficha limpa
    if cand.candidato_inapto:
        lines.append("\n**CANDIDATO INAPTO**")
    if cand.motivo_ficha_limpa:
        lines.append("**Motivo Ficha Limpa aplicado**")

    return "\n".join(lines)


async def resultado_eleicao(
    ano: int,
    municipio: int,
    eleicao_id: int,
    cargo: int,
) -> str:
    """Mostra o resultado de uma eleição com candidatos rankeados por votos.

    Retorna a totalização de votos para todos os candidatos de um cargo
    em um município, ordenados do mais votado ao menos votado.

    Args:
        ano: Ano da eleição (ex: 2020, 2022).
        municipio: Código do município.
        eleicao_id: ID da eleição.
        cargo: Código do cargo (ex: 11=Prefeito, 13=Vereador).

    Returns:
        Tabela com ranking de candidatos por votos.
    """
    resultados = await client.resultado_eleicao(ano, municipio, eleicao_id, cargo)
    if not resultados:
        return "Nenhum resultado encontrado para os filtros informados."

    rows = [
        (
            str(i),
            (r.nome_urna or "—")[:30],
            str(r.numero or "—"),
            r.partido or "—",
            format_number_br(r.total_votos, 0) if r.total_votos is not None else "—",
            r.percentual or "—",
            (r.descricao_totalizacao or "—")[:20],
        )
        for i, r in enumerate(resultados, 1)
    ]
    return f"Resultado da eleição ({len(resultados)} candidatos):\n\n" + markdown_table(
        ["#", "Nome", "Nº", "Partido", "Votos", "%", "Resultado"], rows
    )


async def consultar_prestacao_contas(
    eleicao_id: int,
    ano: int,
    municipio: int,
    cargo: int,
    candidato_id: int,
) -> str:
    """Consulta a prestação de contas de campanha de um candidato.

    Retorna receitas, despesas, doadores, fornecedores e limites de gastos.

    Args:
        eleicao_id: ID da eleição.
        ano: Ano da eleição.
        municipio: Código do município.
        cargo: Código do cargo.
        candidato_id: ID do candidato.

    Returns:
        Resumo financeiro da campanha.
    """
    contas = await client.consultar_prestacao_contas(
        eleicao_id, ano, municipio, cargo, candidato_id
    )
    if contas is None:
        return "Prestação de contas não encontrada para este candidato."

    lines = [
        f"**Candidato:** {contas.nome or '—'}",
        f"**Partido:** {contas.partido or '—'}",
        f"**CNPJ campanha:** {contas.cnpj or '—'}",
    ]

    lines.append("\n**Receitas:**")
    lines.append(f"  Total recebido: {format_brl(contas.total_recebido or 0)}")
    lines.append(f"  Pessoa física: {format_brl(contas.total_receita_pf or 0)}")
    lines.append(f"  Pessoa jurídica: {format_brl(contas.total_receita_pj or 0)}")
    lines.append(f"  Fundo partidário: {format_brl(contas.total_fundo_partidario or 0)}")
    lines.append(f"  Fundo especial: {format_brl(contas.total_fundo_especial or 0)}")

    lines.append("\n**Despesas:**")
    lines.append(f"  Total despesas: {format_brl(contas.total_despesas or 0)}")
    lines.append(f"  Limite de gastos: {format_brl(contas.limite_gastos or 0)}")

    if contas.divida_campanha:
        lines.append(f"\n**Dívida de campanha:** {contas.divida_campanha}")
    if contas.sobra_financeira:
        lines.append(f"**Sobra financeira:** {contas.sobra_financeira}")

    return "\n".join(lines)


# ---------------------------------------------------------------------------
# CDN de Resultados — votação por região (país, estado, município)
# ---------------------------------------------------------------------------


def _cargos_disponiveis() -> str:
    """Return comma-separated list of available cargo names."""
    return ", ".join(CARGO_CODES_CDN.keys())


async def resultado_nacional(
    ano: int,
    cargo: str,
    ctx: Context,
    turno: int = 1,
) -> str:
    """Mostra o resultado nacional de uma eleição com todos os candidatos.

    Retorna a totalização de votos em nível nacional: total de eleitores,
    comparecimento, abstenções e ranking de candidatos por votos.

    Cargos disponíveis: presidente, governador, senador, deputado_federal,
    deputado_estadual, prefeito, vereador.

    Eleições disponíveis: 2022 (federal) e 2024 (municipal).

    Args:
        ano: Ano da eleição (ex: 2022, 2024).
        cargo: Nome do cargo (ex: "presidente", "prefeito").
        turno: Turno da eleição (1 ou 2). Default: 1.

    Returns:
        Tabela com ranking nacional de candidatos por votos.
    """
    await ctx.info(f"Buscando resultado nacional {cargo} {ano} T{turno}...")

    try:
        resultado = await client.resultado_simplificado(ano, cargo, "br", turno)
    except ValueError as e:
        return str(e)

    if resultado is None or not resultado.candidatos:
        return f"Resultado não encontrado para {cargo} {ano} turno {turno}."

    header_lines = [
        f"**Resultado Nacional — {cargo.replace('_', ' ').title()} {ano} (T{turno})**\n",
        f"Apuração: {resultado.pct_apurado}% das seções",
    ]
    if resultado.total_eleitores:
        header_lines.append(f"Eleitores: {format_number_br(resultado.total_eleitores, 0)}")
    if resultado.total_comparecimento:
        header_lines.append(
            f"Comparecimento: {format_number_br(resultado.total_comparecimento, 0)}"
        )
    if resultado.total_abstencoes:
        header_lines.append(f"Abstenções: {format_number_br(resultado.total_abstencoes, 0)}")

    rows = [
        (
            str(i),
            (c.nome or "—")[:25],
            c.numero or "—",
            format_number_br(c.votos, 0) if c.votos else "—",
            f"{c.percentual}%" if c.percentual else "—",
            (c.situacao or "—")[:15],
        )
        for i, c in enumerate(resultado.candidatos, 1)
    ]

    return (
        "\n".join(header_lines)
        + "\n\n"
        + markdown_table(["#", "Candidato", "Nº", "Votos", "%", "Situação"], rows)
    )


async def resultado_por_estado(
    ano: int,
    cargo: str,
    uf: str,
    ctx: Context,
    turno: int = 1,
) -> str:
    """Mostra o resultado de uma eleição em um estado específico.

    Retorna a totalização de votos de cada candidato naquele estado.

    Args:
        ano: Ano da eleição (ex: 2022, 2024).
        cargo: Nome do cargo (ex: "presidente", "governador").
        uf: Sigla do estado (ex: "SP", "RJ", "PI").
        turno: Turno da eleição (1 ou 2). Default: 1.

    Returns:
        Tabela com ranking de candidatos no estado.
    """
    await ctx.info(f"Buscando resultado {cargo} {ano} em {uf.upper()}...")

    try:
        resultado = await client.resultado_simplificado(ano, cargo, uf, turno)
    except ValueError as e:
        return str(e)

    if resultado is None or not resultado.candidatos:
        return f"Resultado não encontrado para {cargo} {ano} T{turno} em {uf.upper()}."

    header_lines = [
        f"**Resultado {uf.upper()} — {cargo.replace('_', ' ').title()} {ano} (T{turno})**\n",
        f"Apuração: {resultado.pct_apurado}% das seções",
    ]
    if resultado.total_eleitores:
        header_lines.append(f"Eleitores: {format_number_br(resultado.total_eleitores, 0)}")

    rows = [
        (
            str(i),
            (c.nome or "—")[:25],
            c.numero or "—",
            format_number_br(c.votos, 0) if c.votos else "—",
            f"{c.percentual}%" if c.percentual else "—",
        )
        for i, c in enumerate(resultado.candidatos, 1)
    ]

    return (
        "\n".join(header_lines)
        + "\n\n"
        + markdown_table(["#", "Candidato", "Nº", "Votos", "%"], rows)
    )


async def mapa_resultado_estados(
    ano: int,
    cargo: str,
    ctx: Context,
    turno: int = 1,
) -> str:
    """Mostra quem venceu em cada estado — mapa eleitoral completo.

    Faz consulta paralela em todos os 27 estados e retorna o candidato
    mais votado em cada UF com votos e percentual.

    Args:
        ano: Ano da eleição (ex: 2022).
        cargo: Nome do cargo (ex: "presidente").
        turno: Turno da eleição (1 ou 2). Default: 1.

    Returns:
        Tabela com vencedor de cada estado.
    """
    await ctx.info(f"Buscando mapa eleitoral {cargo} {ano} T{turno} (27 UFs)...")

    try:
        resultados = await client.resultado_todos_estados(ano, cargo, turno)
    except ValueError as e:
        return str(e)

    if not resultados:
        return f"Nenhum resultado encontrado para {cargo} {ano} turno {turno}."

    rows = []
    for r in sorted(resultados, key=lambda x: (x.uf or "").upper()):
        if not r.candidatos:
            continue
        vencedor = r.candidatos[0]
        rows.append(
            (
                (r.uf or "—").upper(),
                (vencedor.nome or "—")[:20],
                vencedor.numero or "—",
                format_number_br(vencedor.votos, 0) if vencedor.votos else "—",
                f"{vencedor.percentual}%" if vencedor.percentual else "—",
                f"{r.pct_apurado}%" if r.pct_apurado else "—",
            )
        )

    header = (
        f"**Mapa Eleitoral — {cargo.replace('_', ' ').title()} {ano} (T{turno})**\n"
        f"{len(rows)} estados com dados\n"
    )
    return (
        header + "\n" + markdown_table(["UF", "Mais votado", "Nº", "Votos", "%", "Apuração"], rows)
    )


async def listar_municipios_eleitorais(
    ano: int,
    uf: str,
    ctx: Context,
    turno: int = 1,
) -> str:
    """Lista municípios eleitorais de uma UF com códigos TSE e IBGE.

    Retorna os municípios disponíveis para consulta de resultados por município.
    Disponível apenas para eleições municipais (2024).

    Use o codigo_tse retornado como parâmetro para resultado_por_municipio().

    Args:
        ano: Ano da eleição (ex: 2024).
        uf: Sigla do estado (ex: "SP", "RJ").
        turno: Turno da eleição (1 ou 2). Default: 1.

    Returns:
        Tabela com municípios e seus códigos TSE/IBGE.
    """
    await ctx.info(f"Buscando municípios eleitorais de {uf.upper()} {ano}...")

    try:
        municipios = await client.listar_municipios_eleitorais(ano, uf, turno)
    except ValueError as e:
        return str(e)

    if not municipios:
        return f"Nenhum município encontrado para {uf.upper()} {ano} turno {turno}."

    rows = [
        (
            m.codigo_tse or "—",
            m.codigo_ibge or "—",
            (m.nome or "—")[:30],
            "Sim" if m.capital else "Não",
        )
        for m in municipios
    ]
    return f"Municípios eleitorais {uf.upper()} {ano} ({len(municipios)}):\n\n" + markdown_table(
        ["Cód. TSE", "Cód. IBGE", "Nome", "Capital"], rows
    )


async def resultado_por_municipio(
    ano: int,
    cargo: str,
    uf: str,
    cod_tse: str,
    ctx: Context,
    turno: int = 1,
) -> str:
    """Mostra o resultado de uma eleição em um município específico.

    Retorna a totalização de votos de cada candidato no município.
    Disponível apenas para eleições municipais (2024 — prefeito, vereador).

    Use listar_municipios_eleitorais() para obter o código TSE do município.

    Cargos disponíveis: prefeito, vereador.
    Eleições disponíveis: 2024 (municipal).

    Args:
        ano: Ano da eleição (ex: 2024).
        cargo: Nome do cargo (ex: "prefeito", "vereador").
        uf: Sigla do estado (ex: "SP", "RJ").
        cod_tse: Código TSE do município (5 dígitos, ex: "71072" para São Paulo).
        turno: Turno da eleição (1 ou 2). Default: 1.

    Returns:
        Tabela com ranking de candidatos por votos no município.
    """
    await ctx.info(f"Buscando resultado {cargo} {ano} em {uf.upper()} (município {cod_tse})...")

    try:
        resultado = await client.resultado_municipio(ano, cargo, uf, cod_tse, turno)
    except ValueError as e:
        return str(e)

    if resultado is None or not resultado.candidatos:
        return (
            f"Resultado não encontrado para {cargo} {ano} T{turno} "
            f"em {uf.upper()} município {cod_tse}."
        )

    header_lines = [
        f"**Resultado Município {cod_tse} ({uf.upper()}) — "
        f"{cargo.replace('_', ' ').title()} {ano} (T{turno})**\n",
        f"Apuração: {resultado.pct_apurado}% das seções",
    ]
    if resultado.total_eleitores:
        header_lines.append(f"Eleitores: {format_number_br(resultado.total_eleitores, 0)}")

    rows = [
        (
            str(i),
            (c.nome or "—")[:25],
            c.numero or "—",
            format_number_br(c.votos, 0) if c.votos else "—",
            f"{c.percentual}%" if c.percentual else "—",
        )
        for i, c in enumerate(resultado.candidatos, 1)
    ]

    return (
        "\n".join(header_lines)
        + "\n\n"
        + markdown_table(["#", "Candidato", "Nº", "Votos", "%"], rows)
    )


async def apuracao_status(
    ano: int,
    cargo: str,
    ctx: Context,
    uf: str = "br",
    turno: int = 1,
) -> str:
    """Mostra o status da apuração de uma eleição.

    Retorna percentual de seções apuradas, total de eleitores,
    comparecimento e abstenções.

    Args:
        ano: Ano da eleição.
        cargo: Nome do cargo.
        uf: Sigla do estado ou "br" para nacional. Default: "br".
        turno: Turno da eleição. Default: 1.

    Returns:
        Resumo do status da apuração.
    """
    regiao_label = "Nacional" if uf.lower() == "br" else uf.upper()
    await ctx.info(f"Consultando apuração {cargo} {ano} ({regiao_label})...")

    try:
        resultado = await client.resultado_simplificado(ano, cargo, uf, turno)
    except ValueError as e:
        return str(e)

    if resultado is None:
        return f"Dados de apuração não encontrados para {cargo} {ano} T{turno}."

    lines = [
        f"**Status da Apuração — {cargo.replace('_', ' ').title()} {ano} (T{turno})**",
        f"**Região:** {regiao_label}",
        f"**Seções apuradas:** {resultado.pct_apurado}%"
        + (f" de {format_number_br(resultado.total_secoes, 0)}" if resultado.total_secoes else ""),
    ]

    if resultado.total_eleitores:
        lines.append(f"**Eleitores:** {format_number_br(resultado.total_eleitores, 0)}")
    if resultado.total_comparecimento:
        lines.append(f"**Comparecimento:** {format_number_br(resultado.total_comparecimento, 0)}")
    if resultado.total_abstencoes and resultado.total_eleitores:
        pct_abs = resultado.total_abstencoes / resultado.total_eleitores * 100
        lines.append(
            f"**Abstenções:** {format_number_br(resultado.total_abstencoes, 0)} ({pct_abs:.1f}%)"
        )

    return "\n".join(lines)
