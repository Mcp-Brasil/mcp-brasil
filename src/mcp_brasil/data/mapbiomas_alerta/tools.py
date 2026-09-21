"""Tool functions for the MapBiomas Alerta feature.

Regras (ADR-001):
    - tools.py NUNCA faz HTTP direto — delega para client.py
    - devolve texto formatado para consumo por LLM
"""

from __future__ import annotations

from fastmcp import Context

from mcp_brasil._shared.formatting import format_number_br, markdown_table
from mcp_brasil.exceptions import AuthError, HttpClientError

from . import client
from .constants import DEFAULT_LIMIT
from .schemas import Alerta

_WKT_LIMITE = 2000

_NOTA_AUSENCIA = (
    "A ausência de alerta não prova ausência de supressão: há área mínima de "
    "validação e defasagem entre a detecção e a publicação. Registre sempre o "
    "período consultado e a data da consulta."
)


def _formatar_erro(exc: Exception) -> str:
    """Mensagem clara para falha de credencial/HTTP (nunca resultado vazio)."""
    return f"Não foi possível consultar o MapBiomas Alerta: {exc}"


def _ha(valor: float | None) -> str:
    return f"{format_number_br(valor)} ha" if valor is not None else "—"


def _localizacao(alerta: Alerta) -> str:
    partes = list(alerta.municipios[:2])
    ufs = alerta.estados[:2]
    if ufs:
        partes.append("/".join(ufs) if len(ufs) > 1 else ufs[0])
    return ", ".join(partes) if partes else "—"


async def listar_territorios(
    ctx: Context,
    categoria: str | None = None,
    busca: str | None = None,
    limite: int = 50,
) -> str:
    """Lista territórios do MapBiomas Alerta e seus códigos.

    Use para descobrir o ``territorio_id`` antes de chamar ``buscar_alertas``.
    Categorias comuns: ``city`` (município), ``state`` (estado),
    ``conservation_unit`` (Unidade de Conservação) e ``indigenous_land``
    (Terra Indígena).

    Args:
        ctx: Contexto MCP.
        categoria: Filtra por categoria (ex: city, state, conservation_unit).
        busca: Trecho do nome do território (sem diferenciar acento/caixa).
        limite: Máximo de territórios devolvidos (default 50).

    Returns:
        Tabela com código, nome e categoria de cada território.
    """
    await ctx.info("Listando territórios do MapBiomas Alerta...")
    try:
        territorios = await client.listar_territorios(
            categoria=categoria, busca=busca, limite=limite
        )
    except (AuthError, HttpClientError) as exc:
        return _formatar_erro(exc)

    if not territorios:
        return "Nenhum território encontrado para estes filtros."

    rows = [(str(t.codigo), t.nome, t.categoria_nome or t.categoria) for t in territorios]
    header = f"Territórios do MapBiomas Alerta ({len(territorios)} resultados):\n\n"
    return header + markdown_table(["Código", "Nome", "Categoria"], rows)


async def buscar_alertas(
    ctx: Context,
    territorio_id: int | None = None,
    de: str | None = None,
    ate: str | None = None,
    min_ha: float | None = None,
    max_ha: float | None = None,
    em_area_embargada: bool | None = None,
    com_autorizacao: bool | None = None,
    intersecta_car: bool | None = None,
    pagina: int = 1,
    limite: int = DEFAULT_LIMIT,
) -> str:
    """Busca alertas de supressão de vegetação nativa no MapBiomas Alerta.

    Filtra por território (código obtido em ``listar_territorios``), período de
    detecção (AAAA-MM-DD), faixa de área em hectares e presença de embargo,
    autorização de supressão ou CAR cruzado. Requer as variáveis de ambiente
    ``MAPBIOMAS_EMAIL`` e ``MAPBIOMAS_PASSWORD``.

    O filtro de território é uma **união (OR)** — informe um único território
    por consulta. As sobreposições do retorno não devem ser somadas.

    Args:
        ctx: Contexto MCP.
        territorio_id: Código do território (município, estado, UC, TI etc.).
        de: Data inicial de detecção (AAAA-MM-DD).
        ate: Data final de detecção (AAAA-MM-DD).
        min_ha: Área mínima do alerta em hectares.
        max_ha: Área máxima do alerta em hectares.
        em_area_embargada: Restringe a alertas em área embargada.
        com_autorizacao: Filtra por existência de autorização de supressão.
        intersecta_car: Restringe a alertas que cruzam algum imóvel do CAR.
        pagina: Página da listagem (a partir de 1).
        limite: Itens por página (1 a 200, default 20).

    Returns:
        Tabela de alertas com área, data, localização e cruzamentos principais.
    """
    await ctx.info("Buscando alertas no MapBiomas Alerta...")
    try:
        resultado = await client.buscar_alertas(
            territorio_id=territorio_id,
            de=de,
            ate=ate,
            min_ha=min_ha,
            max_ha=max_ha,
            em_area_embargada=em_area_embargada,
            com_autorizacao=com_autorizacao,
            intersecta_car=intersecta_car,
            pagina=pagina,
            limite=limite,
        )
    except (AuthError, HttpClientError) as exc:
        return _formatar_erro(exc)

    if not resultado.alertas:
        return f"Nenhum alerta encontrado para estes filtros.\n\n> {_NOTA_AUSENCIA}"

    rows = [
        (
            str(a.codigo or "—"),
            _ha(a.area_ha),
            (a.detectado_em or "—")[:10],
            _localizacao(a),
            _ha(a.app_area_ha),
            _ha(a.reserva_legal_area_ha),
            str(a.car_total) if a.car_total is not None else "—",
        )
        for a in resultado.alertas
    ]
    header = (
        f"Alertas do MapBiomas Alerta ({len(resultado.alertas)} de "
        f"{resultado.total} no total — página {resultado.pagina}):\n\n"
    )
    tabela = markdown_table(
        ["Código", "Área", "Detectado em", "Localização", "APP", "Reserva Legal", "CAR"],
        rows,
    )

    rodape = ""
    if resultado.total_paginas and resultado.pagina < resultado.total_paginas:
        rodape = (
            f"\n\n**Próxima página:** página={resultado.pagina + 1} de {resultado.total_paginas}."
        )
    return header + tabela + rodape


async def consultar_alerta(ctx: Context, codigo: int) -> str:
    """Dossiê completo de um alerta do MapBiomas Alerta pelo código.

    Traz área em hectares, datas de detecção/publicação, imagens antes e depois
    e a sobreposição com APP, Reserva Legal, Unidade de Conservação, Terra
    Indígena, quilombo, assentamento, bacia e nascente, além de embargos,
    autorizações de supressão, manejo florestal e os códigos de CAR cruzados.
    Devolve também a geometria (WKT) quando disponível na resposta.

    Args:
        ctx: Contexto MCP.
        codigo: Código numérico do alerta (ex: 123456).

    Returns:
        Dossiê formatado do alerta.
    """
    await ctx.info(f"Consultando o alerta {codigo} no MapBiomas Alerta...")
    try:
        alerta = await client.consultar_alerta(codigo)
    except (AuthError, HttpClientError) as exc:
        return _formatar_erro(exc)

    if alerta is None:
        return (
            f"Nenhum alerta encontrado para o código {codigo}. "
            "Confirme o código ou consulte a listagem por território."
        )

    linhas = [
        f"**Alerta MapBiomas {alerta.codigo or codigo}**",
        f"- **Área:** {_ha(alerta.area_ha)}",
        f"- **Detectado em:** {(alerta.detectado_em or '—')[:10]}",
        f"- **Publicado em:** {(alerta.publicado_em or '—')[:10]}",
        f"- **Status:** {alerta.status or '—'}",
        f"- **Fontes:** {', '.join(alerta.fontes) or '—'}",
        f"- **Classes de vegetação:** {', '.join(alerta.classes_desmatamento) or '—'}",
        f"- **Biomas:** {', '.join(alerta.biomas) or '—'}",
        f"- **Localização:** {_localizacao(alerta)}",
    ]
    if alerta.imagem_antes_em or alerta.imagem_depois_em:
        linhas.append(
            f"- **Imagem antes / depois:** {(alerta.imagem_antes_em or '—')[:10]} / "
            f"{(alerta.imagem_depois_em or '—')[:10]}"
        )

    linhas.append("\n**Sobreposições (não somar as áreas):**")
    linhas.append(f"- **APP:** {_ha(alerta.app_area_ha)}")
    linhas.append(f"- **Reserva Legal:** {_ha(alerta.reserva_legal_area_ha)}")
    _linha_opcional(
        linhas,
        "Unidade de Conservação",
        alerta.unidades_conservacao_area_ha,
        alerta.unidades_conservacao,
    )
    _linha_opcional(
        linhas, "Terra Indígena", alerta.terras_indigenas_area_ha, alerta.terras_indigenas
    )
    _linha_opcional(linhas, "Quilombo", alerta.quilombos_area_ha, alerta.quilombos)
    _linha_opcional(linhas, "Assentamento", alerta.assentamentos_area_ha, alerta.assentamentos)
    _linha_opcional(linhas, "Bacia", alerta.bacias_area_ha, alerta.bacias)
    if alerta.nascentes_area_ha:
        linhas.append(f"- **Nascentes:** {_ha(alerta.nascentes_area_ha)}")

    linhas.append("\n**Embargos e autorizações:**")
    linhas.append(f"- **Embargos:** {alerta.embargos_total or 0} ({_ha(alerta.embargos_area_ha)})")
    linhas.append(f"- **Autorização de supressão:** {_ha(alerta.autorizacoes_area_ha)}")
    if alerta.autorizacoes_atividades or alerta.autorizacoes_categorias:
        linhas.append(
            "  - Detalhe: "
            + ", ".join(alerta.autorizacoes_atividades or alerta.autorizacoes_categorias)
        )
    if alerta.manejo_florestal_area_ha or alerta.manejo_florestal_atividades:
        detalhe = (
            f" — {', '.join(alerta.manejo_florestal_atividades)}"
            if alerta.manejo_florestal_atividades
            else ""
        )
        linhas.append(f"- **Manejo florestal:** {_ha(alerta.manejo_florestal_area_ha)}{detalhe}")

    if alerta.car_codigos:
        linhas.append(f"\n**CAR cruzado ({alerta.car_total or len(alerta.car_codigos)}):**")
        for codigo_car in alerta.car_codigos[:10]:
            linhas.append(f"- {codigo_car}")

    if alerta.geometria_wkt:
        recorte = alerta.geometria_wkt[:_WKT_LIMITE]
        sufixo = "" if len(alerta.geometria_wkt) <= _WKT_LIMITE else " (truncado)"
        linhas.append(f"\n**Geometria (WKT{sufixo}, datum {alerta.datum or '—'}):**\n{recorte}")
    else:
        linhas.append("\n**Geometria:** não disponível na resposta deste alerta.")

    linhas.append(f"\n> {_NOTA_AUSENCIA}")
    return "\n".join(linhas)


def _linha_opcional(linhas: list[str], rotulo: str, area: float | None, nomes: list[str]) -> None:
    """Anexa uma linha de sobreposição apenas quando há dado a mostrar."""
    if not area and not nomes:
        return
    detalhe = f" — {', '.join(nomes)}" if nomes else ""
    linhas.append(f"- **{rotulo}:** {_ha(area)}{detalhe}")


async def buscar_alertas_por_car(ctx: Context, codigo_car: str) -> str:
    """Lista os alertas do MapBiomas Alerta associados a um imóvel rural (CAR).

    Use o código do CAR (ex: GO-5211909-XXXXXXXX). O vínculo do alerta com o
    imóvel é um cotejo espacial — não atribui automaticamente a área do alerta
    a nenhum responsável.

    Args:
        ctx: Contexto MCP.
        codigo_car: Código do imóvel rural no CAR.

    Returns:
        Dados do imóvel e tabela dos alertas cruzados.
    """
    await ctx.info(f"Buscando alertas do CAR {codigo_car} no MapBiomas Alerta...")
    try:
        imovel = await client.buscar_alertas_por_car(codigo_car)
    except (AuthError, HttpClientError) as exc:
        return _formatar_erro(exc)

    if imovel is None:
        return (
            f"Nenhum imóvel encontrado para o CAR {codigo_car} nesta base. "
            "Confirme o código informado."
        )

    linhas = [
        f"**Imóvel rural {imovel.codigo or codigo_car}**",
        f"- **Tipo:** {imovel.tipo or '—'}",
        f"- **UF:** {imovel.uf or '—'}",
        f"- **Área do imóvel:** {_ha(imovel.area_ha)}",
        f"- **CAR atualizado em:** {(imovel.car_atualizado_em or '—')[:10]}",
        f"- **Alertas cruzados:** {len(imovel.alertas)}",
        f"- **Área alertada somada:** {_ha(imovel.area_alertada_ha)}",
    ]

    if not imovel.alertas:
        linhas.append("")
        linhas.append("Nenhum alerta encontrado para este imóvel no período disponível.")
        linhas.append(f"> {_NOTA_AUSENCIA}")
        return "\n".join(linhas)

    rows = [
        (
            str(a.codigo or "—"),
            _ha(a.area_ha),
            (a.detectado_em or "—")[:10],
            (a.publicado_em or "—")[:10],
            _ha(a.app_area_ha),
        )
        for a in imovel.alertas
    ]
    linhas.append("")
    linhas.append(markdown_table(["Código", "Área", "Detectado em", "Publicado em", "APP"], rows))
    return "\n".join(linhas)
