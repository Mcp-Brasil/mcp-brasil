"""Tool functions for the BrasilAPI feature.

Rules (ADR-001):
    - tools.py NEVER makes HTTP directly — delegates to client.py
    - Returns formatted strings for LLM consumption
    - Uses Context for structured logging and progress reporting
"""

from __future__ import annotations

from fastmcp import Context

from mcp_brasil._shared.formatting import (
    format_brl,
    format_number_br,
    markdown_table,
    truncate_list,
)

from . import client
from .constants import TAXAS_CONHECIDAS, TIPOS_VEICULO


async def consultar_cep(cep: str, ctx: Context) -> str:
    """Consulta endereço completo a partir de um CEP.

    Retorna logradouro, bairro, cidade e estado.
    Aceita CEP com ou sem hífen (ex: 01001-000 ou 01001000).

    Args:
        cep: CEP com 8 dígitos (ex: 01001000 ou 01001-000).

    Returns:
        Dados do endereço correspondente ao CEP.
    """
    await ctx.info(f"Consultando CEP {cep}...")
    endereco = await client.consultar_cep(cep)
    lines = [
        f"**CEP:** {endereco.cep}",
        f"**Logradouro:** {endereco.street or 'N/A'}",
        f"**Bairro:** {endereco.neighborhood or 'N/A'}",
        f"**Cidade:** {endereco.city}",
        f"**UF:** {endereco.state}",
    ]
    return "\n".join(lines)


async def consultar_cnpj(cnpj: str, ctx: Context) -> str:
    """Consulta dados cadastrais de uma empresa pelo CNPJ.

    Retorna razão social, nome fantasia, situação cadastral, endereço,
    atividade econômica (CNAE) e capital social.
    Aceita CNPJ com ou sem formatação.

    Args:
        cnpj: CNPJ com 14 dígitos (ex: 00000000000191 ou 00.000.000/0001-91).

    Returns:
        Dados cadastrais da empresa.
    """
    await ctx.info(f"Consultando CNPJ {cnpj}...")
    emp = await client.consultar_cnpj(cnpj)
    capital = format_brl(emp.capital_social) if emp.capital_social else "N/A"
    lines = [
        f"**CNPJ:** {emp.cnpj}",
        f"**Razão Social:** {emp.razao_social or 'N/A'}",
        f"**Nome Fantasia:** {emp.nome_fantasia or 'N/A'}",
        f"**Situação:** {emp.descricao_situacao_cadastral or 'N/A'}",
        f"**Porte:** {emp.porte or 'N/A'}",
        f"**Natureza Jurídica:** {emp.natureza_juridica or 'N/A'}",
        f"**CNAE:** {emp.cnae_fiscal} — {emp.cnae_fiscal_descricao or 'N/A'}",
        f"**Endereço:** {emp.logradouro or ''}, {emp.numero or ''} "
        f"{emp.complemento or ''} — {emp.bairro or ''}",
        f"**Cidade/UF:** {emp.municipio or 'N/A'}/{emp.uf or 'N/A'} — CEP {emp.cep or 'N/A'}",
        f"**Telefone:** {emp.ddd_telefone_1 or 'N/A'}",
        f"**Email:** {emp.email or 'N/A'}",
        f"**Capital Social:** {capital}",
    ]
    return "\n".join(lines)


async def consultar_ddd(ddd: str, ctx: Context) -> str:
    """Consulta quais cidades e estado pertencem a um código DDD.

    Útil para identificar a localização geográfica de um número de telefone.

    Args:
        ddd: Código DDD com 2 dígitos (ex: 11, 21, 61).

    Returns:
        Estado e lista de cidades do DDD.
    """
    await ctx.info(f"Consultando DDD {ddd}...")
    info = await client.consultar_ddd(ddd)
    header = f"**DDD {ddd}** — Estado: {info.state}\n\n"
    header += f"**Cidades ({len(info.cities)}):**\n\n"
    return header + truncate_list(sorted(info.cities), max_items=50)


async def listar_bancos(ctx: Context) -> str:
    """Lista todos os bancos brasileiros registrados no Banco Central.

    Retorna código, nome e ISPB de cada banco.
    Útil para identificar bancos por código ou buscar informações bancárias.

    Returns:
        Tabela com todos os bancos brasileiros.
    """
    await ctx.info("Buscando lista de bancos...")
    bancos = await client.listar_bancos()
    await ctx.info(f"{len(bancos)} bancos encontrados")
    rows = [
        (str(b.code or "—"), b.name or "N/A", b.ispb or "N/A")
        for b in bancos
        if b.code is not None
    ]
    rows.sort(key=lambda r: int(r[0]) if r[0] != "—" else 99999)
    return markdown_table(["Código", "Nome", "ISPB"], rows)


async def consultar_banco(codigo: int, ctx: Context) -> str:
    """Consulta dados de um banco específico pelo código.

    Args:
        codigo: Código do banco (ex: 1 para Banco do Brasil, 341 para Itaú).

    Returns:
        Dados completos do banco.
    """
    await ctx.info(f"Consultando banco {codigo}...")
    banco = await client.consultar_banco(codigo)
    lines = [
        f"**Código:** {banco.code}",
        f"**Nome:** {banco.name or 'N/A'}",
        f"**Nome Completo:** {banco.fullName or 'N/A'}",
        f"**ISPB:** {banco.ispb or 'N/A'}",
    ]
    return "\n".join(lines)


async def listar_moedas(ctx: Context) -> str:
    """Lista todas as moedas disponíveis para consulta de câmbio.

    Retorna símbolo e nome de cada moeda para uso em consultar_cotacao.

    Returns:
        Tabela com as moedas disponíveis.
    """
    await ctx.info("Buscando moedas disponíveis...")
    moedas = await client.listar_moedas()
    await ctx.info(f"{len(moedas)} moedas encontradas")
    rows = [(m.simbolo, m.nome_formatado, m.tipo_moeda or "—") for m in moedas]
    return markdown_table(["Símbolo", "Nome", "Tipo"], rows)


async def consultar_cotacao(moeda: str, data: str, ctx: Context) -> str:
    """Consulta a cotação de uma moeda em relação ao Real (BRL) em uma data.

    Use listar_moedas para ver moedas disponíveis.
    Dados disponíveis a partir de 1984-11-28. Não é possível consultar datas futuras.

    Args:
        moeda: Símbolo da moeda (ex: USD, EUR, GBP).
        data: Data no formato YYYY-MM-DD (ex: 2024-01-15).

    Returns:
        Cotação de compra e venda da moeda.
    """
    await ctx.info(f"Consultando cotação {moeda} em {data}...")
    cotacao = await client.consultar_cotacao(moeda, data)
    compra = format_number_br(cotacao.valor_compra, 4) if cotacao.valor_compra else "N/A"
    venda = format_number_br(cotacao.valor_venda, 4) if cotacao.valor_venda else "N/A"
    lines = [
        f"**Moeda:** {cotacao.moeda}",
        f"**Data:** {cotacao.data}",
        f"**Compra:** R$ {compra}",
        f"**Venda:** R$ {venda}",
    ]
    return "\n".join(lines)


async def consultar_feriados(ano: int, ctx: Context) -> str:
    """Lista todos os feriados nacionais de um ano.

    Retorna data, nome e tipo de cada feriado.

    Args:
        ano: Ano com 4 dígitos (ex: 2024, 2025).

    Returns:
        Tabela com os feriados nacionais do ano.
    """
    await ctx.info(f"Consultando feriados de {ano}...")
    feriados = await client.consultar_feriados(ano)
    await ctx.info(f"{len(feriados)} feriados encontrados")
    rows = [(f.date, f.name, f.type) for f in feriados]
    return markdown_table(["Data", "Feriado", "Tipo"], rows)


async def consultar_taxa(sigla: str, ctx: Context) -> str:
    """Consulta uma taxa ou índice oficial da economia brasileira.

    Taxas disponíveis: SELIC, CDI, IPCA, TR, INPC, IGP-M, entre outras.

    Args:
        sigla: Sigla da taxa (ex: SELIC, CDI, IPCA).

    Returns:
        Nome e valor atual da taxa.
    """
    await ctx.info(f"Consultando taxa {sigla.upper()}...")
    taxa = await client.consultar_taxa(sigla)
    valor = format_number_br(taxa.valor, 2) if taxa.valor is not None else "N/A"
    desc = TAXAS_CONHECIDAS.get(sigla.upper(), "")
    lines = [
        f"**Taxa:** {taxa.nome}",
        f"**Valor:** {valor}%",
    ]
    if desc:
        lines.append(f"**Descrição:** {desc}")
    return "\n".join(lines)


async def listar_tabelas_fipe(ctx: Context) -> str:
    """Lista as tabelas de referência FIPE disponíveis.

    A tabela mais recente é a primeira da lista.
    Use o código da tabela em listar_marcas_fipe para filtrar por período.

    Returns:
        Lista das tabelas de referência FIPE com código e mês.
    """
    await ctx.info("Buscando tabelas FIPE...")
    tabelas = await client.listar_tabelas_fipe()
    await ctx.info(f"{len(tabelas)} tabelas encontradas")
    rows = [(str(t.codigo), t.mes) for t in tabelas]
    return markdown_table(["Código", "Mês/Ano"], rows[:24])


async def listar_marcas_fipe(tipo_veiculo: str, ctx: Context) -> str:
    """Lista as marcas de veículos na tabela FIPE por tipo.

    Args:
        tipo_veiculo: Tipo de veículo: carros, caminhoes ou motos.

    Returns:
        Lista de marcas com nome e código para consulta de veículos.
    """
    if tipo_veiculo not in TIPOS_VEICULO:
        return f"Tipo inválido: {tipo_veiculo}. Use: {', '.join(sorted(TIPOS_VEICULO))}"
    await ctx.info(f"Buscando marcas FIPE ({tipo_veiculo})...")
    marcas = await client.listar_marcas_fipe(tipo_veiculo)
    await ctx.info(f"{len(marcas)} marcas encontradas")
    rows = [(m.valor, m.nome) for m in marcas]
    return markdown_table(["Código", "Marca"], rows)


async def buscar_veiculos_fipe(tipo_veiculo: str, codigo_marca: str, ctx: Context) -> str:
    """Busca modelos de veículos na tabela FIPE por tipo e marca.

    Use listar_marcas_fipe para obter o código da marca.

    Args:
        tipo_veiculo: Tipo de veículo: carros, caminhoes ou motos.
        codigo_marca: Código da marca (obtido de listar_marcas_fipe).

    Returns:
        Lista de modelos com preço FIPE.
    """
    if tipo_veiculo not in TIPOS_VEICULO:
        return f"Tipo inválido: {tipo_veiculo}. Use: {', '.join(sorted(TIPOS_VEICULO))}"
    await ctx.info(f"Buscando veículos FIPE ({tipo_veiculo}, marca {codigo_marca})...")
    veiculos = await client.buscar_veiculos_fipe(tipo_veiculo, codigo_marca)
    await ctx.info(f"{len(veiculos)} veículos encontrados")
    rows = [(v.codigo_fipe or "—", v.modelo or v.valor, v.valor) for v in veiculos]
    return markdown_table(["Código FIPE", "Modelo", "Valor"], rows[:50])


async def consultar_isbn(isbn: str, ctx: Context) -> str:
    """Consulta dados de um livro pelo ISBN.

    Busca em múltiplos provedores: CBL, Google Books, Mercado Editorial, Open Library.

    Args:
        isbn: ISBN-10 ou ISBN-13 (com ou sem hífens).

    Returns:
        Dados do livro (título, autor, editora, ano, páginas).
    """
    await ctx.info(f"Consultando ISBN {isbn}...")
    livro = await client.consultar_isbn(isbn)
    autores = ", ".join(livro.authors) if livro.authors else "N/A"
    lines = [
        f"**ISBN:** {livro.isbn or isbn}",
        f"**Título:** {livro.title or 'N/A'}",
    ]
    if livro.subtitle:
        lines.append(f"**Subtítulo:** {livro.subtitle}")
    lines.extend(
        [
            f"**Autor(es):** {autores}",
            f"**Editora:** {livro.publisher or 'N/A'}",
            f"**Ano:** {livro.year or 'N/A'}",
            f"**Páginas:** {livro.page_count or 'N/A'}",
        ]
    )
    if livro.subjects:
        lines.append(f"**Assuntos:** {', '.join(livro.subjects)}")
    return "\n".join(lines)


async def buscar_ncm(busca: str, ctx: Context) -> str:
    """Busca códigos NCM (Nomenclatura Comum do Mercosul) por descrição ou código.

    NCM é o código usado para classificar mercadorias em operações de
    comércio exterior e nota fiscal eletrônica.

    Args:
        busca: Texto de busca (descrição do produto ou código NCM parcial).

    Returns:
        Lista de códigos NCM encontrados.
    """
    await ctx.info(f"Buscando NCM '{busca}'...")
    itens = await client.buscar_ncm(busca)
    await ctx.info(f"{len(itens)} NCMs encontrados")
    if not itens:
        return f"Nenhum NCM encontrado para '{busca}'."
    rows = [(n.codigo, n.descricao) for n in itens]
    return markdown_table(["Código", "Descrição"], rows[:30])


async def consultar_pix_participantes(ctx: Context) -> str:
    """Lista todas as instituições participantes do sistema PIX.

    Retorna ISPB, nome e tipo de participação de cada instituição.

    Returns:
        Tabela com os participantes do PIX.
    """
    await ctx.info("Buscando participantes do PIX...")
    participantes = await client.listar_pix_participantes()
    await ctx.info(f"{len(participantes)} participantes encontrados")
    rows = [
        (p.ispb or "—", p.nome_reduzido or p.nome or "N/A", p.tipo_participacao or "—")
        for p in participantes
    ]
    return markdown_table(["ISPB", "Nome", "Tipo"], rows[:50])


async def consultar_registro_br(dominio: str, ctx: Context) -> str:
    """Consulta a disponibilidade de um domínio .br no Registro.br.

    Args:
        dominio: Nome de domínio (ex: meusite.com.br).

    Returns:
        Status de disponibilidade do domínio.
    """
    await ctx.info(f"Consultando domínio {dominio}...")
    info = await client.consultar_registro_br(dominio)
    status_map = {
        0: "Disponível para registro",
        1: "Disponível com processo de liberação",
        2: "Registrado",
        3: "Indisponível",
        4: "Processo de registro em andamento",
        5: "Domínio em processo de liberação (aguardando)",
    }
    code = info.status_code if info.status_code is not None else -1
    status_desc = status_map.get(code, info.status or "Desconhecido")
    lines = [
        f"**Domínio:** {info.fqdn or dominio}",
        f"**Status:** {status_desc}",
    ]
    if info.expires_at:
        lines.append(f"**Expira em:** {info.expires_at}")
    if info.hosts:
        lines.append(f"**DNS:** {', '.join(info.hosts)}")
    return "\n".join(lines)
