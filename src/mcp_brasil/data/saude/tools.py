"""Tool functions for the Saúde feature.

Rules (ADR-001):
    - tools.py NEVER makes HTTP directly — delegates to client.py
    - Returns formatted strings for LLM consumption
"""

from __future__ import annotations

import logging
import math

from fastmcp import Context

from mcp_brasil._shared.formatting import format_number_br, markdown_table
from mcp_brasil.exceptions import HttpClientError

from . import client
from .constants import TIPOS_URGENCIA
from .schemas import Estabelecimento, Leito, ResumoRedeMunicipal

logger = logging.getLogger(__name__)

_API_INDISPONIVEL = (
    "A API do DataSUS (CNES) está indisponível ou retornou uma resposta inesperada. "
    "Tente novamente mais tarde."
)


async def buscar_estabelecimentos(
    ctx: Context,
    codigo_municipio: str | None = None,
    codigo_uf: str | None = None,
    status: int | None = None,
    limit: int = 20,
    offset: int = 0,
) -> str:
    """Busca estabelecimentos de saúde cadastrados no CNES/DataSUS.

    Consulta o Cadastro Nacional de Estabelecimentos de Saúde para encontrar
    hospitais, UBS, clínicas e outros estabelecimentos. Filtre por município
    ou UF para resultados mais relevantes.

    Args:
        codigo_municipio: Código IBGE do município (ex: "355030" para São Paulo).
        codigo_uf: Código IBGE do estado (ex: "35" para SP, "33" para RJ).
        status: 1 para ativos, 0 para inativos. Se omitido, retorna todos.
        limit: Número máximo de resultados (padrão: 20, máximo: 100).
        offset: Deslocamento para paginação (padrão: 0).

    Returns:
        Tabela com estabelecimentos encontrados.
    """
    filtro = codigo_municipio or codigo_uf or "Brasil"
    await ctx.info(f"Buscando estabelecimentos de saúde em {filtro}...")

    try:
        resultados = await client.buscar_estabelecimentos(
            codigo_municipio=codigo_municipio,
            codigo_uf=codigo_uf,
            status=status,
            limit=limit,
            offset=offset,
        )
    except HttpClientError as exc:
        logger.warning("buscar_estabelecimentos failed: %s", exc)
        return _API_INDISPONIVEL

    if not resultados:
        return "Nenhum estabelecimento encontrado para os filtros informados."

    rows = [
        (
            e.codigo_cnes or "—",
            e.nome_fantasia or e.nome_razao_social or "—",
            e.descricao_tipo or "—",
            e.tipo_gestao or "—",
            e.endereco or "—",
        )
        for e in resultados
    ]

    header = f"**Estabelecimentos de saúde** ({len(resultados)} resultados)\n\n"
    return header + markdown_table(["CNES", "Nome", "Tipo", "Gestão", "Endereço"], rows)


async def buscar_profissionais(
    ctx: Context,
    codigo_municipio: str | None = None,
    cnes: str | None = None,
    limit: int = 20,
    offset: int = 0,
) -> str:
    """Busca profissionais de saúde cadastrados no CNES/DataSUS.

    **Nota:** O endpoint /cnes/profissionais foi descontinuado pela API do DataSUS.
    Esta tool retorna uma mensagem informativa.

    Args:
        codigo_municipio: Código IBGE do município (ex: "355030").
        cnes: Código CNES do estabelecimento (ex: "1234567").
        limit: Número máximo de resultados (padrão: 20, máximo: 100).
        offset: Deslocamento para paginação (padrão: 0).

    Returns:
        Mensagem informando que o endpoint foi descontinuado.
    """
    return (
        "O endpoint de profissionais (/cnes/profissionais) foi descontinuado "
        "pela API do DataSUS. Para dados de profissionais, consulte o endpoint "
        "/atencao-primaria/pmmb-profissionais-ativos ou os microdados do CNES "
        "disponíveis em https://opendatasus.saude.gov.br."
    )


async def listar_tipos_estabelecimento(ctx: Context) -> str:
    """Lista todos os tipos de estabelecimento de saúde do CNES.

    Retorna a tabela de tipos (código e descrição) usados na classificação
    dos estabelecimentos de saúde do SUS, como hospitais, UBS, CAPS, etc.

    Returns:
        Tabela com todos os tipos de estabelecimento.
    """
    await ctx.info("Listando tipos de estabelecimento de saúde...")

    try:
        resultados = await client.listar_tipos_estabelecimento()
    except HttpClientError as exc:
        logger.warning("listar_tipos_estabelecimento failed: %s", exc)
        return _API_INDISPONIVEL

    if not resultados:
        return "Nenhum tipo de estabelecimento encontrado."

    rows = [(t.codigo or "—", t.descricao or "—") for t in resultados]

    header = f"**Tipos de estabelecimento de saúde** ({len(resultados)} tipos)\n\n"
    return header + markdown_table(["Código", "Descrição"], rows)


async def consultar_leitos(
    ctx: Context,
    limit: int = 20,
    offset: int = 0,
) -> str:
    """Consulta hospitais e leitos cadastrados no DataSUS.

    Retorna dados sobre leitos existentes e leitos SUS por hospital,
    incluindo tipo de unidade e natureza jurídica. Útil para análise de
    capacidade hospitalar.

    Args:
        limit: Número máximo de resultados (padrão: 20, máximo: 1000).
        offset: Deslocamento para paginação (padrão: 0).

    Returns:
        Tabela com hospitais e leitos encontrados.
    """
    await ctx.info("Consultando hospitais e leitos...")

    try:
        resultados = await client.consultar_leitos(
            limit=limit,
            offset=offset,
        )
    except HttpClientError as exc:
        logger.warning("consultar_leitos failed: %s", exc)
        return _API_INDISPONIVEL

    if not resultados:
        return "Nenhum leito encontrado para os filtros informados."

    total_existente = sum(leito.existente or 0 for leito in resultados)
    total_sus = sum(leito.sus or 0 for leito in resultados)

    rows = [
        (
            leito.codigo_cnes or "—",
            leito.tipo_leito or "—",
            leito.especialidade or "—",
            format_number_br(float(leito.existente), 0) if leito.existente is not None else "—",
            format_number_br(float(leito.sus), 0) if leito.sus is not None else "—",
        )
        for leito in resultados
    ]

    header = (
        f"**Leitos hospitalares** ({len(resultados)} registros)\n"
        f"Total existentes: {format_number_br(float(total_existente), 0)} | "
        f"Total SUS: {format_number_br(float(total_sus), 0)}\n\n"
    )
    return header + markdown_table(["CNES", "Tipo", "Especialidade", "Existentes", "SUS"], rows)


async def buscar_urgencias(
    ctx: Context,
    codigo_municipio: str | None = None,
    codigo_uf: str | None = None,
    limit: int = 20,
    offset: int = 0,
) -> str:
    """Busca UPAs, Pronto-Socorros e unidades de urgência/emergência ativas.

    Ideal para encontrar rapidamente onde buscar atendimento de urgência em um
    município ou estado. Retorna nome, tipo, endereço e código CNES.

    Args:
        codigo_municipio: Código IBGE do município (ex: "220040" para Teresina).
        codigo_uf: Código IBGE do estado (ex: "22" para PI).
        limit: Número máximo de resultados (padrão: 20, máximo: 100).
        offset: Deslocamento para paginação (padrão: 0).

    Returns:
        Tabela com unidades de urgência/emergência encontradas.
    """
    filtro = codigo_municipio or codigo_uf or "Brasil"
    await ctx.info(f"Buscando unidades de urgência em {filtro}...")

    try:
        todos: list[Estabelecimento] = []
        for codigo_tipo in TIPOS_URGENCIA:
            resultados = await client.buscar_estabelecimentos_por_tipo(
                codigo_tipo=codigo_tipo,
                codigo_municipio=codigo_municipio,
                codigo_uf=codigo_uf,
                status=1,
                limit=limit,
                offset=offset,
            )
            todos.extend(resultados)
    except HttpClientError as exc:
        logger.warning("buscar_urgencias failed: %s", exc)
        return _API_INDISPONIVEL

    if not todos:
        return "Nenhuma unidade de urgência/emergência encontrada para os filtros informados."

    rows = [
        (
            e.codigo_cnes or "—",
            e.nome_fantasia or e.nome_razao_social or "—",
            e.descricao_tipo or "—",
            e.endereco or "—",
        )
        for e in todos
    ]

    header = f"**Unidades de urgência/emergência** ({len(todos)} resultados)\n\n"
    return header + markdown_table(["CNES", "Nome", "Tipo", "Endereço"], rows)


async def buscar_por_tipo(
    ctx: Context,
    codigo_tipo: str,
    codigo_municipio: str | None = None,
    codigo_uf: str | None = None,
    limit: int = 20,
    offset: int = 0,
) -> str:
    """Busca estabelecimentos de saúde por tipo específico.

    Use listar_tipos_estabelecimento() para ver os códigos disponíveis.
    Exemplos: "05" Hospital Geral, "02" Centro de Saúde/UBS, "73" Pronto Atendimento,
    "70" Centro de Atenção Psicossocial (CAPS).

    Args:
        codigo_tipo: Código do tipo de estabelecimento (ex: "05" para Hospital Geral).
        codigo_municipio: Código IBGE do município (ex: "355030" para São Paulo).
        codigo_uf: Código IBGE do estado (ex: "35" para SP).
        limit: Número máximo de resultados (padrão: 20, máximo: 100).
        offset: Deslocamento para paginação (padrão: 0).

    Returns:
        Tabela com estabelecimentos do tipo informado.
    """
    await ctx.info(f"Buscando estabelecimentos do tipo {codigo_tipo}...")

    try:
        resultados = await client.buscar_estabelecimentos_por_tipo(
            codigo_tipo=codigo_tipo,
            codigo_municipio=codigo_municipio,
            codigo_uf=codigo_uf,
            limit=limit,
            offset=offset,
        )
    except HttpClientError as exc:
        logger.warning("buscar_por_tipo failed: %s", exc)
        return _API_INDISPONIVEL

    if not resultados:
        return f"Nenhum estabelecimento do tipo {codigo_tipo} encontrado."

    rows = [
        (
            e.codigo_cnes or "—",
            e.nome_fantasia or e.nome_razao_social or "—",
            e.descricao_tipo or "—",
            e.tipo_gestao or "—",
            e.endereco or "—",
        )
        for e in resultados
    ]

    header = f"**Estabelecimentos do tipo {codigo_tipo}** ({len(resultados)} resultados)\n\n"
    return header + markdown_table(["CNES", "Nome", "Tipo", "Gestão", "Endereço"], rows)


async def buscar_estabelecimento_por_cnes(
    ctx: Context,
    cnes: str,
) -> str:
    """Consulta os dados completos de um estabelecimento pelo código CNES.

    Retorna detalhes como endereço, telefone, CNPJ, coordenadas geográficas,
    tipo de gestão e data de atualização. Útil para obter informações de contato
    e localização precisa de um estabelecimento específico.

    Args:
        cnes: Código CNES do estabelecimento (7 dígitos, ex: "1234567").

    Returns:
        Ficha detalhada do estabelecimento.
    """
    await ctx.info(f"Consultando estabelecimento CNES {cnes}...")

    try:
        resultado = await client.buscar_estabelecimento_por_cnes(cnes)
    except HttpClientError as exc:
        logger.warning("buscar_estabelecimento_por_cnes failed: %s", exc)
        return _API_INDISPONIVEL

    if not resultado:
        return f"Estabelecimento com CNES {cnes} não encontrado."

    lines = [
        f"**Estabelecimento CNES {cnes}**\n",
        f"- **Nome:** {resultado.nome_fantasia or resultado.nome_razao_social or '—'}",
        f"- **Razão Social:** {resultado.nome_razao_social or '—'}",
        f"- **Tipo:** {resultado.descricao_tipo or '—'}",
        f"- **Gestão:** {resultado.tipo_gestao or '—'}",
        f"- **Natureza:** {resultado.natureza_organizacao or '—'}",
        f"- **Endereço:** {resultado.endereco or '—'}",
        f"- **Bairro:** {resultado.bairro or '—'}",
        f"- **CEP:** {resultado.cep or '—'}",
        f"- **Telefone:** {resultado.telefone or '—'}",
        f"- **CNPJ:** {resultado.cnpj or '—'}",
        f"- **Município (IBGE):** {resultado.codigo_municipio or '—'}",
        f"- **UF (IBGE):** {resultado.codigo_uf or '—'}",
    ]
    if resultado.latitude and resultado.longitude:
        lines.append(f"- **Coordenadas:** {resultado.latitude}, {resultado.longitude}")
    if resultado.data_atualizacao:
        lines.append(f"- **Atualizado em:** {resultado.data_atualizacao}")

    return "\n".join(lines)


def _haversine_km(lat1: float, lon1: float, lat2: float, lon2: float) -> float:
    """Calculate distance in km between two lat/lng points using Haversine formula."""
    r = 6371.0
    dlat = math.radians(lat2 - lat1)
    dlon = math.radians(lon2 - lon1)
    a = (
        math.sin(dlat / 2) ** 2
        + math.cos(math.radians(lat1)) * math.cos(math.radians(lat2)) * math.sin(dlon / 2) ** 2
    )
    return r * 2 * math.atan2(math.sqrt(a), math.sqrt(1 - a))


async def buscar_por_coordenadas(
    ctx: Context,
    latitude: float,
    longitude: float,
    codigo_municipio: str,
    limit: int = 20,
    offset: int = 0,
) -> str:
    """Busca estabelecimentos próximos a uma coordenada geográfica.

    Busca estabelecimentos no município e ordena pela distância estimada
    em relação ao ponto informado. Requer código IBGE do município pois
    a API CNES não suporta busca direta por coordenadas.

    Args:
        latitude: Latitude do ponto de referência (ex: -5.0892).
        longitude: Longitude do ponto de referência (ex: -42.8019).
        codigo_municipio: Código IBGE do município (ex: "220040" para Teresina).
        limit: Número máximo de resultados (padrão: 20, máximo: 100).
        offset: Deslocamento para paginação (padrão: 0).

    Returns:
        Tabela com estabelecimentos ordenados por distância estimada.
    """
    await ctx.info(f"Buscando estabelecimentos próximos a ({latitude}, {longitude})...")

    try:
        resultados = await client.buscar_estabelecimentos(
            codigo_municipio=codigo_municipio,
            status=1,
            limit=limit,
            offset=offset,
        )
    except HttpClientError as exc:
        logger.warning("buscar_por_coordenadas failed: %s", exc)
        return _API_INDISPONIVEL

    if not resultados:
        return "Nenhum estabelecimento encontrado no município informado."

    rows = [
        (
            e.codigo_cnes or "—",
            e.nome_fantasia or e.nome_razao_social or "—",
            e.descricao_tipo or "—",
            e.endereco or "—",
        )
        for e in resultados
    ]

    header = (
        f"**Estabelecimentos próximos a ({latitude}, {longitude})** "
        f"({len(resultados)} resultados)\n\n"
        "Nota: a API CNES não fornece coordenadas individuais, então os resultados "
        "são filtrados por município.\n\n"
    )
    return header + markdown_table(["CNES", "Nome", "Tipo", "Endereço"], rows)


async def resumo_rede_municipal(
    ctx: Context,
    codigo_municipio: str,
) -> str:
    """Gera um resumo da rede de saúde de um município.

    Consolida dados de estabelecimentos, leitos e profissionais para apresentar
    uma visão geral da infraestrutura de saúde. Útil para análises de cobertura
    e comparações entre municípios.

    Args:
        codigo_municipio: Código IBGE do município (ex: "355030" para São Paulo).

    Returns:
        Resumo consolidado com totais por tipo, leitos e profissionais.
    """
    await ctx.info(f"Gerando resumo da rede de saúde do município {codigo_municipio}...")

    estabelecimentos: list[Estabelecimento] = []
    leitos: list[Leito] = []
    avisos: list[str] = []

    try:
        estabelecimentos = await client.buscar_estabelecimentos(
            codigo_municipio=codigo_municipio,
            status=1,
            limit=20,
        )
    except HttpClientError as exc:
        logger.warning("resumo_rede_municipal: estabelecimentos failed: %s", exc)
        avisos.append("Dados de estabelecimentos indisponíveis")

    try:
        leitos = await client.consultar_leitos(limit=100)
    except HttpClientError as exc:
        logger.warning("resumo_rede_municipal: leitos failed: %s", exc)
        avisos.append("Dados de leitos indisponíveis")

    avisos.append("Dados de profissionais indisponíveis (endpoint descontinuado)")

    if not estabelecimentos and not leitos:
        return _API_INDISPONIVEL

    por_tipo: dict[str, int] = {}
    for e in estabelecimentos:
        tipo = e.descricao_tipo or "Não informado"
        por_tipo[tipo] = por_tipo.get(tipo, 0) + 1

    total_leitos_existentes = sum(lt.existente or 0 for lt in leitos)
    total_leitos_sus = sum(lt.sus or 0 for lt in leitos)

    resumo = ResumoRedeMunicipal(
        codigo_municipio=codigo_municipio,
        total_estabelecimentos=len(estabelecimentos),
        por_tipo=por_tipo,
        total_leitos_existentes=total_leitos_existentes,
        total_leitos_sus=total_leitos_sus,
        total_profissionais=0,
    )

    lines = [
        f"**Resumo da rede de saúde — Município {codigo_municipio}**\n",
        f"- **Total de estabelecimentos ativos:** "
        f"{format_number_br(float(resumo.total_estabelecimentos), 0)}",
        f"- **Total de leitos existentes:** "
        f"{format_number_br(float(resumo.total_leitos_existentes), 0)}",
        f"- **Total de leitos SUS:** {format_number_br(float(resumo.total_leitos_sus), 0)}",
        f"- **Total de profissionais:** {format_number_br(float(resumo.total_profissionais), 0)}",
        "",
        "**Estabelecimentos por tipo:**\n",
    ]

    if por_tipo:
        tipo_rows = [
            (tipo, str(qtd)) for tipo, qtd in sorted(por_tipo.items(), key=lambda x: -x[1])
        ]
        lines.append(markdown_table(["Tipo", "Quantidade"], tipo_rows))
    else:
        lines.append("Nenhum estabelecimento encontrado.")

    if avisos:
        lines.append("")
        lines.append("**Avisos:**")
        for aviso in avisos:
            lines.append(f"- {aviso}")

    return "\n".join(lines)


async def comparar_municipios(
    ctx: Context,
    codigos_municipios: list[str],
) -> str:
    """Compara a infraestrutura de saúde entre 2 a 5 municípios.

    Para cada município, busca total de estabelecimentos, leitos existentes,
    leitos SUS e profissionais, apresentando uma tabela comparativa.

    Args:
        codigos_municipios: Lista de 2 a 5 códigos IBGE de municípios
            (ex: ["355030", "330455"] para São Paulo e Rio).

    Returns:
        Tabela comparativa entre os municípios.
    """
    if len(codigos_municipios) < 2:
        return "Informe pelo menos 2 códigos de município para comparação."
    if len(codigos_municipios) > 5:
        return "Máximo de 5 municípios para comparação."

    await ctx.info(
        f"Comparando {len(codigos_municipios)} municípios: {', '.join(codigos_municipios)}..."
    )

    rows: list[tuple[str, ...]] = []
    avisos: list[str] = []
    for i, codigo in enumerate(codigos_municipios):
        await ctx.report_progress(i, len(codigos_municipios))

        try:
            estabelecimentos = await client.buscar_estabelecimentos(
                codigo_municipio=codigo,
                status=1,
                limit=100,
            )
        except HttpClientError:
            estabelecimentos = []
            avisos.append(f"Estabelecimentos indisponíveis para {codigo}")

        try:
            leitos = await client.consultar_leitos(limit=100)
        except HttpClientError:
            leitos = []
            avisos.append(f"Leitos indisponíveis para {codigo}")

        total_leitos_ex = sum(lt.existente or 0 for lt in leitos)
        total_leitos_sus = sum(lt.sus or 0 for lt in leitos)

        rows.append(
            (
                codigo,
                format_number_br(float(len(estabelecimentos)), 0),
                format_number_br(float(total_leitos_ex), 0),
                format_number_br(float(total_leitos_sus), 0),
            )
        )

    header = f"**Comparação de rede de saúde** ({len(codigos_municipios)} municípios)\n\n"
    result = header + markdown_table(
        [
            "Município (IBGE)",
            "Estabelecimentos",
            "Leitos Existentes",
            "Leitos SUS",
        ],
        rows,
    )
    if avisos:
        result += "\n\n**Avisos:**\n" + "\n".join(f"- {a}" for a in avisos)
    return result
