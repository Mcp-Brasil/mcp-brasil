"""Tool functions for the TCE-ES feature.

Rules (ADR-001):
    - tools.py NEVER makes HTTP directly — delegates to client.py
    - Returns formatted strings for LLM consumption
    - Uses Context for structured logging and progress reporting
"""

from __future__ import annotations

from fastmcp import Context

from . import client


async def buscar_licitacoes_es(
    ctx: Context,
    q: str | None = None,
    ano: int | None = None,
    situacao: str | None = None,
    deslocamento: int = 0,
) -> str:
    """Busca licitações do próprio TCE-ES (Tribunal de Contas do Espírito Santo).

    Dados do portal de Aquisições do TCE-ES via dados.es.gov.br.
    Inclui modalidade, objeto, valores de referência e homologado.

    Args:
        ctx: Contexto MCP.
        q: Texto livre para busca no objeto da licitação (opcional).
        ano: Filtro por ano do edital, ex: 2024 (opcional).
        situacao: Filtro por situação, ex: "Homologado", "Em Andamento" (opcional).
        deslocamento: Offset para paginação (default: 0).

    Returns:
        Lista de licitações com modalidade, objeto e valores.
    """
    await ctx.info("Buscando licitações do TCE-ES...")
    licitacoes, total = await client.buscar_licitacoes(
        q=q,
        ano=ano,
        situacao=situacao,
        offset=deslocamento,
    )

    if not licitacoes:
        return "Nenhuma licitação encontrada no TCE-ES."

    lines: list[str] = [f"**{total} licitações no TCE-ES:**\n"]
    for lic in licitacoes[:20]:
        valor_ref = lic.ValorReferencia or "—"
        valor_hom = lic.ValorHomologado or "—"
        objeto = (lic.Objeto or "—")[:200]
        lines.append(f"### Edital {lic.NumeroEdital or '—'}/{lic.AnoEdital or '—'}")
        lines.append(f"- **Modalidade:** {lic.Modalidade or '—'}")
        lines.append(f"- **Objeto:** {objeto}")
        lines.append(f"- **Valor referência:** {valor_ref}")
        lines.append(f"- **Valor homologado:** {valor_hom}")
        lines.append(f"- **Situação:** {lic.Situacao or '—'}")
        if lic.DataAbertura:
            lines.append(f"- **Abertura:** {lic.DataAbertura}")
        lines.append("")

    if total > deslocamento + 20:
        lines.append(
            f"*Mostrando 20 de {total}. Use deslocamento={deslocamento + 20} para próxima página.*"
        )
    return "\n".join(lines)


async def buscar_contratos_es(
    ctx: Context,
    q: str | None = None,
    ano: int | None = None,
    deslocamento: int = 0,
) -> str:
    """Busca contratos celebrados pelo TCE-ES (Tribunal de Contas do Espírito Santo).

    Dados do portal de Aquisições do TCE-ES via dados.es.gov.br.
    Inclui fornecedor, objeto, vigência e valores.

    Args:
        ctx: Contexto MCP.
        q: Texto livre para busca no objeto/fornecedor (opcional).
        ano: Filtro por ano do contrato, ex: 2024 (opcional).
        deslocamento: Offset para paginação (default: 0).

    Returns:
        Lista de contratos com fornecedor, objeto e valores.
    """
    await ctx.info("Buscando contratos do TCE-ES...")
    contratos, total = await client.buscar_contratos(
        q=q,
        ano=ano,
        offset=deslocamento,
    )

    if not contratos:
        return "Nenhum contrato encontrado no TCE-ES."

    lines: list[str] = [f"**{total} contratos no TCE-ES:**\n"]
    for c in contratos[:20]:
        objeto = (c.ResumoObjeto or "—")[:200]
        valor = c.VigenciaAtualValorGlobal or c.TermoOriginalValorGlobal or "—"
        lines.append(f"### Contrato {c.ContratoNumero or '—'}/{c.ContratoAno or '—'}")
        lines.append(f"- **Modalidade:** {c.Modalidade or '—'}")
        lines.append(f"- **Objeto:** {objeto}")
        lines.append(f"- **Fornecedor:** {c.FornecedorNome or '—'}")
        lines.append(f"- **Valor atual:** {valor}")
        if c.TermoOriginalDataFimVigencia:
            lines.append(f"- **Vigência até:** {c.TermoOriginalDataFimVigencia}")
        if c.Setor:
            lines.append(f"- **Setor:** {c.Setor}")
        lines.append("")

    if total > deslocamento + 20:
        lines.append(
            f"*Mostrando 20 de {total}. Use deslocamento={deslocamento + 20} para próxima página.*"
        )
    return "\n".join(lines)


async def buscar_contratacoes_municipios_es(
    ctx: Context,
    q: str | None = None,
    ano_referencia: int | None = None,
    esfera: str | None = None,
    deslocamento: int = 0,
) -> str:
    """Busca contratações de municípios e órgãos capixabas monitorados pelo TCE-ES.

    Base do Controle Externo com 127 mil registros cobrindo todo o ES.
    Inclui modalidade, objeto, unidade gestora e valores estimados.

    Args:
        ctx: Contexto MCP.
        q: Texto livre (objeto, unidade gestora, etc.) (opcional).
        ano_referencia: Ano de referência, ex: 2024 (opcional).
        esfera: Esfera administrativa, ex: "Municipal", "Estadual" (opcional).
        deslocamento: Offset para paginação (default: 0).

    Returns:
        Lista de contratações com unidade gestora, objeto e valores.
    """
    await ctx.info("Buscando contratações de municípios ES no TCE-ES...")
    contratacoes, total = await client.buscar_contratacoes_municipios(
        q=q,
        ano_referencia=ano_referencia,
        esfera=esfera,
        offset=deslocamento,
    )

    if not contratacoes:
        return "Nenhuma contratação encontrada no TCE-ES."

    lines: list[str] = [f"**{total} contratações no TCE-ES:**\n"]
    for c in contratacoes[:20]:
        objeto = (c.ObjetoContratacao or "—")[:200]
        valor_est = c.ValorEstimado or "—"
        valor_total = c.ValorTotalContratacao or "—"
        lines.append(f"### {c.NomeUnidadeGestoraReferencia or '—'}")
        lines.append(f"- **Esfera:** {c.NomeEsferaAdministrativa or '—'}")
        lines.append(f"- **Modalidade:** {c.ModalidadeLicitacao or '—'}")
        lines.append(f"- **Objeto:** {objeto}")
        lines.append(f"- **Valor estimado:** {valor_est}")
        lines.append(f"- **Valor total:** {valor_total}")
        lines.append(f"- **Situação:** {c.SituacaoContratacao or '—'}")
        if c.AnoReferencia:
            lines.append(f"- **Ano:** {c.AnoReferencia}")
        lines.append("")

    if total > deslocamento + 20:
        lines.append(
            f"*Mostrando 20 de {total}. Use deslocamento={deslocamento + 20} para próxima página.*"
        )
    return "\n".join(lines)


async def buscar_obras_es(
    ctx: Context,
    q: str | None = None,
    situacao: str | None = None,
    deslocamento: int = 0,
) -> str:
    """Busca obras públicas cadastradas no TCE-ES.

    Base de Obras Públicas do Controle Externo do TCE-ES.
    Inclui empresa contratada, valor inicial e situação da obra.

    Args:
        ctx: Contexto MCP.
        q: Texto livre (empresa, município, objeto) (opcional).
        situacao: Filtro por situação da obra (opcional).
        deslocamento: Offset para paginação (default: 0).

    Returns:
        Lista de obras com empresa, valores e situação.
    """
    await ctx.info("Buscando obras públicas no TCE-ES...")
    obras, total = await client.buscar_obras(
        q=q,
        situacao=situacao,
        offset=deslocamento,
    )

    if not obras:
        return "Nenhuma obra encontrada no TCE-ES."

    lines: list[str] = [f"**{total} obras no TCE-ES:**\n"]
    for o in obras[:20]:
        valor = o.ValorInicial or "—"
        lines.append(f"### Contrato {o.Contrato or '—'}")
        lines.append(f"- **Licitação:** {o.Licitacao or '—'}")
        lines.append(f"- **Empresa:** {o.Empresa or '—'}")
        if o.EmpresaCNPJ:
            lines.append(f"- **CNPJ:** {o.EmpresaCNPJ}")
        lines.append(f"- **Valor inicial:** {valor}")
        lines.append(f"- **Situação:** {o.Situacao or '—'}")
        if o.DataAssinaturaContrato:
            lines.append(f"- **Assinatura:** {o.DataAssinaturaContrato}")
        lines.append("")

    if total > deslocamento + 20:
        lines.append(
            f"*Mostrando 20 de {total}. Use deslocamento={deslocamento + 20} para próxima página.*"
        )
    return "\n".join(lines)
