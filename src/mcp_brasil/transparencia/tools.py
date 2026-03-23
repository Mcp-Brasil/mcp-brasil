"""Tool functions for the Transparência feature.

Rules (ADR-001):
    - tools.py NEVER makes HTTP directly — delegates to client.py
    - Returns formatted strings for LLM consumption
"""

from __future__ import annotations

from mcp_brasil._shared.formatting import format_brl, markdown_table, truncate_list

from . import client
from .constants import DEFAULT_PAGE_SIZE


def _pagination_hint(count: int, pagina: int) -> str:
    """Return a pagination hint string based on result count and current page."""
    if count >= DEFAULT_PAGE_SIZE:
        return f"\n\n> Use `pagina={pagina + 1}` para ver mais resultados."
    if pagina > 1 and count < DEFAULT_PAGE_SIZE:
        return "\n\n> Última página de resultados."
    return ""


async def buscar_contratos(cpf_cnpj: str, pagina: int = 1) -> str:
    """Busca contratos federais por CPF ou CNPJ do fornecedor.

    Consulta o Portal da Transparência para listar contratos firmados
    com o governo federal por um fornecedor específico.

    Args:
        cpf_cnpj: CPF ou CNPJ do fornecedor (aceita com ou sem formatação).
        pagina: Página de resultados (padrão: 1).

    Returns:
        Tabela com contratos encontrados.
    """
    contratos = await client.buscar_contratos(cpf_cnpj, pagina)
    if not contratos:
        return f"Nenhum contrato encontrado para o CPF/CNPJ '{cpf_cnpj}'."

    rows = [
        (
            c.numero or "—",
            (c.objeto or "—")[:80],
            format_brl(c.valor_final) if c.valor_final else "—",
            c.data_inicio or "—",
            c.data_fim or "—",
            (c.orgao or "—")[:40],
        )
        for c in contratos
    ]
    header = f"Contratos do fornecedor {cpf_cnpj} (página {pagina}):\n\n"
    table = header + markdown_table(
        ["Número", "Objeto", "Valor Final", "Início", "Fim", "Órgão"], rows
    )
    return table + _pagination_hint(len(contratos), pagina)


async def consultar_despesas(
    mes_ano_inicio: str,
    mes_ano_fim: str,
    codigo_favorecido: str | None = None,
    pagina: int = 1,
) -> str:
    """Consulta despesas e recursos recebidos por favorecido.

    Mostra pagamentos realizados pelo governo federal a um favorecido
    (pessoa física ou jurídica) em um período.

    Args:
        mes_ano_inicio: Mês/ano de início no formato MM/AAAA (ex: 01/2024).
        mes_ano_fim: Mês/ano de fim no formato MM/AAAA (ex: 12/2024).
        codigo_favorecido: CPF ou CNPJ do favorecido (opcional).
        pagina: Página de resultados (padrão: 1).

    Returns:
        Tabela com despesas encontradas.
    """
    despesas = await client.consultar_despesas(
        mes_ano_inicio, mes_ano_fim, codigo_favorecido, pagina
    )
    if not despesas:
        return "Nenhuma despesa encontrada para os parâmetros informados."

    rows = [
        (
            f"{d.mes or '—'}/{d.ano or '—'}",
            (d.favorecido_nome or "—")[:50],
            format_brl(d.valor) if d.valor else "—",
            (d.orgao_nome or "—")[:40],
            d.uf or "—",
        )
        for d in despesas
    ]
    header = f"Despesas de {mes_ano_inicio} a {mes_ano_fim} (página {pagina}):\n\n"
    table = header + markdown_table(["Período", "Favorecido", "Valor", "Órgão", "UF"], rows)
    return table + _pagination_hint(len(despesas), pagina)


async def buscar_servidores(
    cpf: str | None = None,
    nome: str | None = None,
    pagina: int = 1,
) -> str:
    """Busca servidores públicos federais por CPF ou nome.

    Consulta a base de servidores do Portal da Transparência.
    Informe CPF ou nome (pelo menos um é obrigatório).

    Args:
        cpf: CPF do servidor (opcional se nome fornecido).
        nome: Nome do servidor (opcional se CPF fornecido).
        pagina: Página de resultados (padrão: 1).

    Returns:
        Tabela com servidores encontrados.
    """
    if not cpf and not nome:
        return "Informe CPF ou nome do servidor para a busca."

    servidores = await client.buscar_servidores(cpf=cpf, nome=nome, pagina=pagina)
    if not servidores:
        busca = cpf or nome
        return f"Nenhum servidor encontrado para '{busca}'."

    rows = [
        (
            s.cpf or "—",
            (s.nome or "—")[:50],
            s.tipo_servidor or "—",
            s.situacao or "—",
            (s.orgao or "—")[:40],
        )
        for s in servidores
    ]
    busca = cpf or nome
    header = f"Servidores encontrados para '{busca}' (página {pagina}):\n\n"
    table = header + markdown_table(["CPF", "Nome", "Tipo", "Situação", "Órgão"], rows)
    return table + _pagination_hint(len(servidores), pagina)


async def buscar_licitacoes(
    codigo_orgao: str | None = None,
    data_inicial: str | None = None,
    data_final: str | None = None,
    pagina: int = 1,
) -> str:
    """Busca licitações federais por órgão e/ou período.

    Consulta processos licitatórios do governo federal.
    Pelo menos um filtro (órgão ou datas) é recomendado.

    Args:
        codigo_orgao: Código SIAFI do órgão (ex: "26246" para UFPI).
        data_inicial: Data inicial no formato DD/MM/AAAA.
        data_final: Data final no formato DD/MM/AAAA.
        pagina: Página de resultados (padrão: 1).

    Returns:
        Tabela com licitações encontradas.
    """
    licitacoes = await client.buscar_licitacoes(
        codigo_orgao=codigo_orgao,
        data_inicial=data_inicial,
        data_final=data_final,
        pagina=pagina,
    )
    if not licitacoes:
        return "Nenhuma licitação encontrada para os parâmetros informados."

    rows = [
        (
            lc.numero or "—",
            (lc.objeto or "—")[:60],
            lc.modalidade or "—",
            lc.situacao or "—",
            format_brl(lc.valor_estimado) if lc.valor_estimado else "—",
            lc.data_abertura or "—",
        )
        for lc in licitacoes
    ]
    header = f"Licitações encontradas (página {pagina}):\n\n"
    table = header + markdown_table(
        ["Número", "Objeto", "Modalidade", "Situação", "Valor Est.", "Abertura"], rows
    )
    return table + _pagination_hint(len(licitacoes), pagina)


async def consultar_bolsa_familia(
    mes_ano: str,
    codigo_ibge: str | None = None,
    nis: str | None = None,
    pagina: int = 1,
) -> str:
    """Consulta dados do Novo Bolsa Família por município ou NIS.

    Informe código IBGE do município OU NIS do beneficiário.
    Retorna dados de pagamento do programa de transferência de renda.

    Args:
        mes_ano: Mês/ano de referência no formato AAAAMM (ex: 202401).
        codigo_ibge: Código IBGE do município (ex: 3550308 para São Paulo).
        nis: NIS (Número de Identificação Social) do beneficiário.
        pagina: Página de resultados (padrão: 1).

    Returns:
        Dados do Bolsa Família encontrados.
    """
    if not codigo_ibge and not nis:
        return "Informe o código IBGE do município ou o NIS do beneficiário."

    if nis:
        sacados = await client.consultar_bolsa_familia_nis(mes_ano, nis, pagina)
        if not sacados:
            return f"Nenhum dado encontrado para NIS '{nis}' em {mes_ano}."
        rows = [
            (
                s.nis or "—",
                (s.nome or "—")[:50],
                s.municipio or "—",
                s.uf or "—",
                format_brl(s.valor) if s.valor else "—",
            )
            for s in sacados
        ]
        table = f"Bolsa Família — NIS {nis} ({mes_ano}):\n\n" + markdown_table(
            ["NIS", "Nome", "Município", "UF", "Valor"], rows
        )
        return table + _pagination_hint(len(sacados), pagina)

    assert codigo_ibge is not None
    municipios = await client.consultar_bolsa_familia_municipio(mes_ano, codigo_ibge, pagina)
    if not municipios:
        return f"Nenhum dado encontrado para município {codigo_ibge} em {mes_ano}."
    rows = [
        (
            m.municipio or "—",
            m.uf or "—",
            str(m.quantidade) if m.quantidade else "—",
            format_brl(m.valor) if m.valor else "—",
            m.data_referencia or "—",
        )
        for m in municipios
    ]
    table = f"Bolsa Família — Município {codigo_ibge} ({mes_ano}):\n\n" + markdown_table(
        ["Município", "UF", "Beneficiados", "Valor", "Referência"], rows
    )
    return table + _pagination_hint(len(municipios), pagina)


async def buscar_sancoes(
    consulta: str,
    bases: list[str] | None = None,
    pagina: int = 1,
) -> str:
    """Busca sanções em bases federais (CEIS, CNEP, CEPIM, CEAF).

    Consulta simultânea nas bases de sanções do governo federal.
    Útil para due diligence, compliance e verificação anticorrupção.

    Bases disponíveis:
    - CEIS: Empresas Inidôneas e Suspensas
    - CNEP: Empresas Punidas (Lei Anticorrupção 12.846)
    - CEPIM: Entidades sem Fins Lucrativos Impedidas
    - CEAF: Expulsões da Administração Federal

    Args:
        consulta: CPF, CNPJ ou nome da pessoa/empresa a pesquisar.
        bases: Lista de bases (ex: ["ceis", "cnep"]). Padrão: todas.
        pagina: Página de resultados (padrão: 1).

    Returns:
        Sanções encontradas agrupadas por base.
    """
    sancoes = await client.buscar_sancoes(consulta, bases, pagina)
    if not sancoes:
        bases_str = ", ".join(bases) if bases else "CEIS, CNEP, CEPIM, CEAF"
        return f"Nenhuma sanção encontrada para '{consulta}' nas bases: {bases_str}."

    items: list[str] = []
    for s in sancoes:
        parts = [f"**{s.nome or '—'}** ({s.cpf_cnpj or '—'})"]
        parts.append(f"  Fonte: {s.fonte or '—'}")
        if s.tipo:
            parts.append(f"  Tipo: {s.tipo}")
        if s.orgao:
            parts.append(f"  Órgão sancionador: {s.orgao}")
        if s.data_inicio or s.data_fim:
            parts.append(f"  Período: {s.data_inicio or '—'} a {s.data_fim or '—'}")
        if s.fundamentacao:
            parts.append(f"  Fundamentação: {s.fundamentacao}")
        items.append("\n".join(parts))

    header = f"Sanções encontradas para '{consulta}' ({len(sancoes)} resultado(s)):\n\n"
    result = header + truncate_list(items, max_items=30)
    return result + _pagination_hint(len(sancoes), pagina)


async def buscar_emendas(
    ano: int | None = None,
    nome_autor: str | None = None,
    pagina: int = 1,
) -> str:
    """Busca emendas parlamentares por ano e/ou autor.

    Consulta emendas individuais e de bancada ao orçamento federal.

    Args:
        ano: Ano da emenda (ex: 2024).
        nome_autor: Nome do parlamentar autor da emenda.
        pagina: Página de resultados (padrão: 1).

    Returns:
        Tabela com emendas encontradas.
    """
    emendas = await client.buscar_emendas(ano=ano, nome_autor=nome_autor, pagina=pagina)
    if not emendas:
        return "Nenhuma emenda encontrada para os parâmetros informados."

    rows = [
        (
            e.numero or "—",
            (e.autor or "—")[:40],
            e.tipo or "—",
            (e.localidade or "—")[:30],
            format_brl(e.valor_empenhado) if e.valor_empenhado else "—",
            format_brl(e.valor_pago) if e.valor_pago else "—",
        )
        for e in emendas
    ]
    header = f"Emendas parlamentares (página {pagina}):\n\n"
    table = header + markdown_table(
        ["Número", "Autor", "Tipo", "Localidade", "Empenhado", "Pago"], rows
    )
    return table + _pagination_hint(len(emendas), pagina)


async def consultar_viagens(cpf: str, pagina: int = 1) -> str:
    """Consulta viagens a serviço de servidor federal por CPF.

    Mostra viagens realizadas a serviço, incluindo diárias e passagens.

    Args:
        cpf: CPF do servidor (aceita com ou sem formatação).
        pagina: Página de resultados (padrão: 1).

    Returns:
        Tabela com viagens encontradas.
    """
    viagens = await client.consultar_viagens(cpf, pagina)
    if not viagens:
        return f"Nenhuma viagem encontrada para o CPF '{cpf}'."

    rows = [
        (
            (v.nome or "—")[:40],
            v.cargo or "—",
            (v.orgao or "—")[:30],
            v.destino or "—",
            f"{v.data_inicio or '—'} a {v.data_fim or '—'}",
            format_brl(v.valor_diarias) if v.valor_diarias else "—",
            format_brl(v.valor_passagens) if v.valor_passagens else "—",
        )
        for v in viagens
    ]
    header = f"Viagens do servidor CPF {cpf} (página {pagina}):\n\n"
    table = header + markdown_table(
        ["Nome", "Cargo", "Órgão", "Destino", "Período", "Diárias", "Passagens"], rows
    )
    return table + _pagination_hint(len(viagens), pagina)


async def buscar_convenios(
    orgao: str | None = None,
    convenente: str | None = None,
    pagina: int = 1,
) -> str:
    """Busca convênios e transferências voluntárias do governo federal.

    Consulta convênios celebrados entre órgãos federais e entidades
    (estados, municípios, ONGs) para repasse de recursos.

    Args:
        orgao: Código do órgão concedente (ex: "26246").
        convenente: Nome ou CNPJ do convenente.
        pagina: Página de resultados (padrão: 1).

    Returns:
        Tabela com convênios encontrados.
    """
    convenios = await client.buscar_convenios(orgao=orgao, convenente=convenente, pagina=pagina)
    if not convenios:
        return "Nenhum convênio encontrado para os parâmetros informados."

    rows = [
        (
            c.numero or "—",
            (c.objeto or "—")[:60],
            c.situacao or "—",
            format_brl(c.valor_convenio) if c.valor_convenio else "—",
            format_brl(c.valor_liberado) if c.valor_liberado else "—",
            (c.orgao or "—")[:30],
            (c.convenente or "—")[:30],
        )
        for c in convenios
    ]
    header = f"Convênios encontrados (página {pagina}):\n\n"
    table = header + markdown_table(
        ["Número", "Objeto", "Situação", "Valor", "Liberado", "Órgão", "Convenente"], rows
    )
    return table + _pagination_hint(len(convenios), pagina)


async def buscar_cartoes_pagamento(
    cpf_portador: str | None = None,
    codigo_orgao: str | None = None,
    mes_ano_inicio: str | None = None,
    mes_ano_fim: str | None = None,
    pagina: int = 1,
) -> str:
    """Busca pagamentos com cartão corporativo (suprimento de fundos).

    Consulta gastos realizados com cartão de pagamento do governo federal,
    incluindo cartão corporativo e suprimento de fundos.

    Args:
        cpf_portador: CPF do portador do cartão (opcional).
        codigo_orgao: Código do órgão (opcional).
        mes_ano_inicio: Mês/ano de início no formato MM/AAAA (ex: 01/2024).
        mes_ano_fim: Mês/ano de fim no formato MM/AAAA (ex: 12/2024).
        pagina: Página de resultados (padrão: 1).

    Returns:
        Tabela com pagamentos encontrados.
    """
    cartoes = await client.buscar_cartoes_pagamento(
        cpf_portador=cpf_portador,
        codigo_orgao=codigo_orgao,
        mes_ano_inicio=mes_ano_inicio,
        mes_ano_fim=mes_ano_fim,
        pagina=pagina,
    )
    if not cartoes:
        return "Nenhum pagamento com cartão encontrado para os parâmetros informados."

    rows = [
        (
            (c.portador or "—")[:40],
            (c.orgao or "—")[:30],
            format_brl(c.valor) if c.valor else "—",
            c.data or "—",
            c.tipo or "—",
            (c.estabelecimento or "—")[:30],
        )
        for c in cartoes
    ]
    header = f"Pagamentos com cartão (página {pagina}):\n\n"
    table = header + markdown_table(
        ["Portador", "Órgão", "Valor", "Data", "Tipo", "Estabelecimento"], rows
    )
    return table + _pagination_hint(len(cartoes), pagina)


async def buscar_pep(
    cpf: str | None = None,
    nome: str | None = None,
    pagina: int = 1,
) -> str:
    """Busca Pessoas Expostas Politicamente (PEP).

    Consulta a base de PEPs do governo federal — pessoas que ocupam ou
    ocuparam cargos, empregos ou funções públicas relevantes.

    Args:
        cpf: CPF da pessoa (opcional se nome fornecido).
        nome: Nome da pessoa (opcional se CPF fornecido).
        pagina: Página de resultados (padrão: 1).

    Returns:
        Tabela com PEPs encontrados.
    """
    if not cpf and not nome:
        return "Informe CPF ou nome para buscar Pessoas Expostas Politicamente."

    peps = await client.buscar_pep(cpf=cpf, nome=nome, pagina=pagina)
    if not peps:
        busca = cpf or nome
        return f"Nenhuma PEP encontrada para '{busca}'."

    rows = [
        (
            p.cpf or "—",
            (p.nome or "—")[:40],
            (p.orgao or "—")[:30],
            p.funcao or "—",
            p.data_inicio or "—",
            p.data_fim or "—",
        )
        for p in peps
    ]
    busca = cpf or nome
    header = f"PEPs encontradas para '{busca}' (página {pagina}):\n\n"
    table = header + markdown_table(["CPF", "Nome", "Órgão", "Função", "Início", "Fim"], rows)
    return table + _pagination_hint(len(peps), pagina)


async def buscar_acordos_leniencia(
    nome_empresa: str | None = None,
    cnpj: str | None = None,
    pagina: int = 1,
) -> str:
    """Busca acordos de leniência (anticorrupção).

    Consulta acordos firmados com empresas envolvidas em atos ilícitos
    contra a administração pública (Lei Anticorrupção 12.846/2013).

    Args:
        nome_empresa: Nome da empresa (opcional).
        cnpj: CNPJ da empresa (opcional).
        pagina: Página de resultados (padrão: 1).

    Returns:
        Tabela com acordos encontrados.
    """
    acordos = await client.buscar_acordos_leniencia(
        nome_empresa=nome_empresa, cnpj=cnpj, pagina=pagina
    )
    if not acordos:
        return "Nenhum acordo de leniência encontrado para os parâmetros informados."

    rows = [
        (
            (a.empresa or "—")[:40],
            a.cnpj or "—",
            (a.orgao or "—")[:30],
            a.situacao or "—",
            a.data_inicio or "—",
            format_brl(a.valor) if a.valor else "—",
        )
        for a in acordos
    ]
    header = f"Acordos de leniência (página {pagina}):\n\n"
    table = header + markdown_table(
        ["Empresa", "CNPJ", "Órgão", "Situação", "Início", "Valor Multa"], rows
    )
    return table + _pagination_hint(len(acordos), pagina)


async def buscar_notas_fiscais(
    cnpj_emitente: str | None = None,
    data_emissao_de: str | None = None,
    data_emissao_ate: str | None = None,
    pagina: int = 1,
) -> str:
    """Busca notas fiscais eletrônicas vinculadas a gastos federais.

    Consulta notas fiscais eletrônicas relacionadas a despesas
    do governo federal.

    Args:
        cnpj_emitente: CNPJ do emitente da nota (opcional).
        data_emissao_de: Data de emissão inicial DD/MM/AAAA (opcional).
        data_emissao_ate: Data de emissão final DD/MM/AAAA (opcional).
        pagina: Página de resultados (padrão: 1).

    Returns:
        Tabela com notas fiscais encontradas.
    """
    notas = await client.buscar_notas_fiscais(
        cnpj_emitente=cnpj_emitente,
        data_emissao_de=data_emissao_de,
        data_emissao_ate=data_emissao_ate,
        pagina=pagina,
    )
    if not notas:
        return "Nenhuma nota fiscal encontrada para os parâmetros informados."

    rows = [
        (
            n.numero or "—",
            n.serie or "—",
            (n.emitente or "—")[:40],
            n.cnpj_emitente or "—",
            format_brl(n.valor) if n.valor else "—",
            n.data_emissao or "—",
        )
        for n in notas
    ]
    header = f"Notas fiscais (página {pagina}):\n\n"
    table = header + markdown_table(
        ["Número", "Série", "Emitente", "CNPJ", "Valor", "Emissão"], rows
    )
    return table + _pagination_hint(len(notas), pagina)


async def consultar_beneficio_social(
    cpf: str | None = None,
    nis: str | None = None,
    mes_ano: str | None = None,
    pagina: int = 1,
) -> str:
    """Consulta benefícios sociais (BPC, seguro-desemprego, etc.) por CPF ou NIS.

    Consulta programas sociais do governo federal além do Bolsa Família,
    como BPC (Benefício de Prestação Continuada) e seguro-desemprego.

    Args:
        cpf: CPF do beneficiário (opcional se NIS fornecido).
        nis: NIS do beneficiário (opcional se CPF fornecido).
        mes_ano: Mês/ano de referência no formato AAAAMM (ex: 202401).
        pagina: Página de resultados (padrão: 1).

    Returns:
        Tabela com benefícios encontrados.
    """
    if not cpf and not nis:
        return "Informe CPF ou NIS do beneficiário."

    beneficios = await client.consultar_beneficio_social(
        cpf=cpf, nis=nis, mes_ano=mes_ano, pagina=pagina
    )
    if not beneficios:
        busca = cpf or nis
        return f"Nenhum benefício social encontrado para '{busca}'."

    rows = [
        (
            b.tipo or "—",
            (b.nome_beneficiario or "—")[:40],
            format_brl(b.valor) if b.valor else "—",
            b.mes_referencia or "—",
            b.municipio or "—",
            b.uf or "—",
        )
        for b in beneficios
    ]
    busca = cpf or nis
    header = f"Benefícios sociais para '{busca}' (página {pagina}):\n\n"
    table = header + markdown_table(
        ["Tipo", "Beneficiário", "Valor", "Referência", "Município", "UF"], rows
    )
    return table + _pagination_hint(len(beneficios), pagina)


async def consultar_cpf(cpf: str, pagina: int = 1) -> str:
    """Consulta vínculos e benefícios de uma pessoa física por CPF.

    Mostra informações consolidadas sobre os vínculos de uma pessoa
    com o governo federal (servidores, beneficiários, fornecedores).

    Args:
        cpf: CPF da pessoa (aceita com ou sem formatação).
        pagina: Página de resultados (padrão: 1).

    Returns:
        Informações sobre vínculos encontrados.
    """
    vinculos = await client.consultar_cpf(cpf, pagina)
    if not vinculos:
        return f"Nenhum vínculo encontrado para o CPF '{cpf}'."

    items: list[str] = []
    for v in vinculos:
        parts = [f"**{v.nome or '—'}** (CPF: {v.cpf or '—'})"]
        if v.tipo_vinculo:
            parts.append(f"  Tipo: {v.tipo_vinculo}")
        if v.orgao:
            parts.append(f"  Órgão: {v.orgao}")
        if v.beneficios:
            parts.append(f"  Benefícios: {v.beneficios}")
        items.append("\n".join(parts))

    header = f"Vínculos do CPF {cpf} ({len(vinculos)} resultado(s), página {pagina}):\n\n"
    result = header + truncate_list(items, max_items=30)
    return result + _pagination_hint(len(vinculos), pagina)


async def consultar_cnpj(cnpj: str, pagina: int = 1) -> str:
    """Consulta sanções e contratos de pessoa jurídica por CNPJ.

    Mostra informações consolidadas sobre uma empresa junto ao
    governo federal (contratos, sanções, pendências).

    Args:
        cnpj: CNPJ da empresa (aceita com ou sem formatação).
        pagina: Página de resultados (padrão: 1).

    Returns:
        Informações sobre vínculos encontrados.
    """
    vinculos = await client.consultar_cnpj(cnpj, pagina)
    if not vinculos:
        return f"Nenhum vínculo encontrado para o CNPJ '{cnpj}'."

    items: list[str] = []
    for v in vinculos:
        parts = [f"**{v.razao_social or '—'}** (CNPJ: {v.cnpj or '—'})"]
        if v.sancoes:
            parts.append(f"  Sanções: {v.sancoes}")
        if v.contratos:
            parts.append(f"  Contratos: {v.contratos}")
        items.append("\n".join(parts))

    header = f"Vínculos do CNPJ {cnpj} ({len(vinculos)} resultado(s), página {pagina}):\n\n"
    result = header + truncate_list(items, max_items=30)
    return result + _pagination_hint(len(vinculos), pagina)


async def detalhar_contrato(id_contrato: int) -> str:
    """Detalha um contrato federal específico por ID.

    Retorna informações completas de um contrato, incluindo
    modalidade, licitação, situação e valores.

    Args:
        id_contrato: ID do contrato no Portal da Transparência.

    Returns:
        Detalhes do contrato.
    """
    contrato = await client.detalhar_contrato(id_contrato)
    if not contrato:
        return f"Contrato com ID {id_contrato} não encontrado."

    lines = [
        f"## Contrato {contrato.numero or id_contrato}\n",
        f"- **Objeto:** {contrato.objeto or '—'}",
        f"- **Fornecedor:** {contrato.fornecedor or '—'}",
        f"- **Órgão:** {contrato.orgao or '—'}",
        f"- **Modalidade:** {contrato.modalidade or '—'}",
        f"- **Situação:** {contrato.situacao or '—'}",
        f"- **Valor Inicial:** "
        f"{format_brl(contrato.valor_inicial) if contrato.valor_inicial else '—'}",
        f"- **Valor Final:** {format_brl(contrato.valor_final) if contrato.valor_final else '—'}",
        f"- **Vigência:** {contrato.data_inicio or '—'} a {contrato.data_fim or '—'}",
        f"- **Licitação:** {contrato.licitacao or '—'}",
    ]
    return "\n".join(lines)


async def detalhar_servidor(id_servidor: int) -> str:
    """Detalha um servidor público federal por ID, incluindo remuneração.

    Retorna informações completas de um servidor, incluindo cargo,
    função e remuneração bruta e líquida.

    Args:
        id_servidor: ID do servidor no Portal da Transparência.

    Returns:
        Detalhes do servidor.
    """
    servidor = await client.detalhar_servidor(id_servidor)
    if not servidor:
        return f"Servidor com ID {id_servidor} não encontrado."

    lines = [
        f"## Servidor {servidor.nome or id_servidor}\n",
        f"- **CPF:** {servidor.cpf or '—'}",
        f"- **Tipo:** {servidor.tipo_servidor or '—'}",
        f"- **Situação:** {servidor.situacao or '—'}",
        f"- **Órgão:** {servidor.orgao or '—'}",
        f"- **Cargo:** {servidor.cargo or '—'}",
        f"- **Função:** {servidor.funcao or '—'}",
        f"- **Remuneração Básica:** "
        f"{format_brl(servidor.remuneracao_basica) if servidor.remuneracao_basica else '—'}",
        "- **Remuneração Líquida:** "
        + (
            format_brl(servidor.remuneracao_apos_deducoes)
            if servidor.remuneracao_apos_deducoes
            else "—"
        ),
    ]
    return "\n".join(lines)
