"""Tool functions for the TCE-CE feature.

Rules (ADR-001):
    - tools.py NEVER makes HTTP directly — delegates to client.py
    - Returns formatted strings for LLM consumption
    - Uses Context for structured logging and progress reporting
"""

from __future__ import annotations

from fastmcp import Context

from mcp_brasil._shared.formatting import format_brl

from . import client


def _cabecalho(total: int, tem_mais: bool, rotulo: str) -> str:
    if tem_mais:
        return f"**{total} {rotulo} no TCE-CE nesta página (há mais páginas):**\n"
    return f"**{total} {rotulo} no TCE-CE:**\n"


def _rodape_paginacao(mostrados: int, total: int, tem_mais: bool, deslocamento: int) -> str | None:
    if tem_mais:
        return (
            f"*Página com {total} registros (mostrando {mostrados}). Há mais resultados: "
            f"use deslocamento={deslocamento + total}.*"
        )
    if total > mostrados:
        return f"*Mostrando {mostrados} de {total}.*"
    return None


async def listar_municipios_ce(ctx: Context, nome_municipio: str | None = None) -> str:
    """Lista os municípios cearenses sob jurisdição do TCE-CE.

    Retorna código SIM, nome e código IBGE de cada município. O código SIM
    (3 dígitos, ex: "057" Fortaleza, "134" Pentecoste) é usado nas demais consultas.

    Args:
        ctx: Contexto MCP.
        nome_municipio: Nome exato do município para filtrar (ex: "PENTECOSTE").

    Returns:
        Lista de municípios com código e nome.
    """
    await ctx.info("Listando municípios do TCE-CE...")
    municipios = await client.listar_municipios(nome_municipio=nome_municipio)

    if not municipios:
        return "Nenhum município encontrado no TCE-CE."

    limite = 200
    lines: list[str] = [f"**{len(municipios)} municípios no TCE-CE:**\n"]
    for m in municipios[:limite]:
        ibge = f", IBGE {m.codigo_municipio_ibge}" if m.codigo_municipio_ibge else ""
        lines.append(f"- **{m.nome_municipio or '—'}** (código: `{m.codigo_municipio}`{ibge})")
    if len(municipios) > limite:
        lines.append(f"\n*Mostrando {limite} de {len(municipios)} municípios.*")
    return "\n".join(lines)


async def buscar_licitacoes_ce(
    ctx: Context,
    codigo_municipio: str,
    data_realizacao: str,
    deslocamento: int = 0,
) -> str:
    """Busca processos de contratação de um município cearense por período.

    Licitações, dispensas, inexigibilidades e adesões a atas registradas no SIM
    do TCE-CE, com objeto, valor estimado e homologação.

    Args:
        ctx: Contexto MCP.
        codigo_municipio: Código SIM do município (ex: "057" para Fortaleza).
            Use listar_municipios_ce para obter códigos.
        data_realizacao: Período: ano ("2026"), dia ("2026-08-05") ou intervalo
            ("2026-01-01_2026-12-31").
        deslocamento: Offset para paginação.

    Returns:
        Lista de processos com objeto, modalidade e valores.
    """
    await ctx.info(f"Buscando licitações no TCE-CE (município {codigo_municipio})...")
    resultado = await client.buscar_licitacoes(
        codigo_municipio=codigo_municipio,
        data_realizacao=data_realizacao,
        deslocamento=deslocamento,
    )

    if not resultado.licitacoes:
        return "Nenhuma licitação encontrada no TCE-CE para o período."

    limite = 20
    lines: list[str] = [
        _cabecalho(resultado.total, resultado.tem_mais, "processos de contratação")
    ]
    for lic in resultado.licitacoes[:limite]:
        valor = format_brl(lic.valor_orcado_estimado) if lic.valor_orcado_estimado else "—"
        lines.append(f"### {lic.numero_licitacao or '—'}")
        lines.append(f"- **Data:** {lic.data_realizacao or '—'}")
        lines.append(f"- **Modalidade:** {lic.modalidade_licitacao or '—'}")
        lines.append(f"- **Objeto:** {(lic.objeto or '—')[:300]}")
        lines.append(f"- **Valor estimado:** {valor}")
        if lic.data_homologacao:
            lines.append(f"- **Homologação:** {lic.data_homologacao}")
        lines.append("")

    rodape = _rodape_paginacao(
        min(limite, resultado.total), resultado.total, resultado.tem_mais, deslocamento
    )
    if rodape:
        lines.append(rodape)
    return "\n".join(lines)


async def buscar_contratos_ce(
    ctx: Context,
    codigo_municipio: str,
    data_contrato: str,
    deslocamento: int = 0,
) -> str:
    """Busca contratos de um município cearense por período, com o contratado.

    Dados do SIM do TCE-CE: objeto, valor, vigência, tipo e o contratado
    (nome e CNPJ/CPF, cruzados com o cadastro de contratados do SIM).

    Args:
        ctx: Contexto MCP.
        codigo_municipio: Código SIM do município (ex: "057" para Fortaleza).
        data_contrato: Período: ano ("2026"), dia ("2026-08-31") ou intervalo
            ("2026-01-01_2026-12-31").
        deslocamento: Offset para paginação.

    Returns:
        Lista de contratos com contratado, objeto, valor e vigência.
    """
    await ctx.info(f"Buscando contratos no TCE-CE (município {codigo_municipio})...")
    resultado = await client.buscar_contratos(
        codigo_municipio=codigo_municipio,
        data_contrato=data_contrato,
        deslocamento=deslocamento,
    )

    if not resultado.contratos:
        return "Nenhum contrato encontrado no TCE-CE para o período."

    limite = 20
    lines: list[str] = [_cabecalho(resultado.total, resultado.tem_mais, "contratos")]
    for c in resultado.contratos[:limite]:
        valor = format_brl(c.valor_total_contrato) if c.valor_total_contrato else "—"
        lines.append(f"### {c.numero_contrato or '—'}")
        lines.append(f"- **Data:** {c.data_contrato or '—'}")
        if c.nome_contratado:
            doc = f" ({c.documento_contratado})" if c.documento_contratado else ""
            lines.append(f"- **Contratado:** {c.nome_contratado}{doc}")
        lines.append(f"- **Tipo:** {c.tipo_contrato or '—'}")
        lines.append(f"- **Objeto:** {(c.objeto or '—')[:300]}")
        lines.append(f"- **Valor:** {valor}")
        if c.data_inicio_vigencia or c.data_fim_vigencia:
            lines.append(
                f"- **Vigência:** {c.data_inicio_vigencia or '—'} a {c.data_fim_vigencia or '—'}"
            )
        if c.numero_processo_adm:
            lines.append(f"- **Processo administrativo:** {c.numero_processo_adm}")
        lines.append("")

    if resultado.contratado_incompleto:
        lines.append(
            "*Atenção: o cruzamento com o cadastro de contratados parou no limite de leitura. "
            "Contrato sem contratado aqui NÃO significa contratado ausente no SIM; "
            "consulte um período menor.*"
        )
    rodape = _rodape_paginacao(
        min(limite, resultado.total), resultado.total, resultado.tem_mais, deslocamento
    )
    if rodape:
        lines.append(rodape)
    return "\n".join(lines)


async def buscar_empenhos_ce(
    ctx: Context,
    codigo_municipio: str,
    data_referencia: int,
    codigo_orgao: str | None = None,
    deslocamento: int = 0,
) -> str:
    """Busca notas de empenho de um município cearense em um mês.

    Empenhos são o primeiro estágio da despesa pública. Dados do SIM do TCE-CE,
    com credor (nome e CNPJ/CPF), valor e histórico.

    Args:
        ctx: Contexto MCP.
        codigo_municipio: Código SIM do município (ex: "057" para Fortaleza).
        data_referencia: Ano-mês no formato AAAAMM (ex: 202608 para ago/2026).
        codigo_orgao: Código do órgão no SIM (opcional; varia por município).
        deslocamento: Offset para paginação.

    Returns:
        Lista de empenhos com credor, valor e descrição.
    """
    await ctx.info(f"Buscando empenhos no TCE-CE (município {codigo_municipio})...")
    resultado = await client.buscar_empenhos(
        codigo_municipio=codigo_municipio,
        data_referencia=data_referencia,
        codigo_orgao=codigo_orgao,
        deslocamento=deslocamento,
    )

    if not resultado.empenhos:
        return "Nenhum empenho encontrado no TCE-CE para o mês."

    limite = 20
    lines: list[str] = [_cabecalho(resultado.total, resultado.tem_mais, "empenhos")]
    for e in resultado.empenhos[:limite]:
        valor = format_brl(e.valor_empenho) if e.valor_empenho else "—"
        doc = f" ({e.numero_documento_negociante})" if e.numero_documento_negociante else ""
        lines.append(f"### Empenho {e.numero_empenho or '—'}")
        lines.append(f"- **Data:** {e.data_emissao or '—'}")
        lines.append(f"- **Credor:** {e.nome_negociante or '—'}{doc}")
        lines.append(f"- **Valor:** {valor}")
        lines.append(f"- **Descrição:** {(e.historico or '—')[:200]}")
        lines.append("")

    rodape = _rodape_paginacao(
        min(limite, resultado.total), resultado.total, resultado.tem_mais, deslocamento
    )
    if rodape:
        lines.append(rodape)
    return "\n".join(lines)
