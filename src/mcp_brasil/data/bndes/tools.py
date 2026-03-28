"""Tool functions for the BNDES feature.

Rules (ADR-001):
    - tools.py NEVER makes HTTP directly — delegates to client.py
    - Returns formatted strings for LLM consumption
    - Uses Context for structured logging and progress reporting
"""

from __future__ import annotations

from fastmcp import Context

from . import client


async def buscar_datasets_bndes(
    query: str,
    ctx: Context,
    limite: int = 10,
) -> str:
    """Busca datasets no portal de dados abertos do BNDES.

    O BNDES disponibiliza 46 datasets com dados sobre financiamentos,
    desembolsos, exportações, instituições credenciadas e mais.

    Args:
        query: Termo de busca (ex: 'financiamento', 'exportação', 'desembolso').
        limite: Máximo de resultados (padrão 10).

    Returns:
        Lista de datasets encontrados com título e descrição.
    """
    await ctx.info(f"Buscando datasets BNDES para '{query}'...")
    datasets = await client.buscar_datasets(query, limite)

    if not datasets:
        return f"Nenhum dataset encontrado para '{query}'."

    lines = [f"**{len(datasets)} datasets encontrados**\n"]
    for i, ds in enumerate(datasets, 1):
        desc = (ds.notes or "")[:150]
        if len(ds.notes or "") > 150:
            desc += "..."
        resources_info = f"{ds.num_resources} recursos" if ds.num_resources else "N/A"
        lines.extend(
            [
                f"### {i}. {ds.title or ds.name}",
                f"**Nome:** `{ds.name}` | **Recursos:** {resources_info}",
                f"{desc}" if desc else "",
                "",
            ]
        )
    return "\n".join(lines)


async def detalhar_dataset_bndes(
    nome: str,
    ctx: Context,
) -> str:
    """Detalha um dataset do BNDES com seus recursos disponíveis.

    Mostra metadados e lista de recursos (CSVs, PDFs) com IDs para
    consulta via datastore_search.

    Args:
        nome: Nome do dataset (ex: 'operacoes-financiamento', 'desembolsos').

    Returns:
        Detalhes do dataset e lista de recursos.
    """
    await ctx.info(f"Detalhando dataset '{nome}'...")
    ds = await client.detalhar_dataset(nome)

    if not ds:
        return f"Dataset '{nome}' não encontrado."

    lines = [
        f"## {ds.title or ds.name}",
        f"{ds.notes or 'Sem descrição.'}\n",
        f"**Recursos ({ds.num_resources or len(ds.resources)}):**\n",
    ]
    for i, r in enumerate(ds.resources, 1):
        desc = r.description or "Sem descrição"
        lines.append(f"{i}. **{r.name or 'N/A'}** ({r.format or 'N/A'}) — {desc}")
        if r.id:
            lines.append(f"   `resource_id: {r.id}`")
    return "\n".join(lines)


async def consultar_operacoes_bndes(
    ctx: Context,
    uf: str | None = None,
    setor: str | None = None,
    porte: str | None = None,
    busca: str | None = None,
    nao_automaticas: bool = False,
    limite: int = 20,
    offset: int = 0,
) -> str:
    """Consulta operações de financiamento do BNDES.

    Pesquisa no banco com 2.3M+ operações de financiamento (automáticas)
    ou 23K operações não-automáticas (com descrição de projeto).

    Args:
        uf: Sigla da UF (ex: SP, RJ, MG). Opcional.
        setor: Setor CNAE (ex: 'INDÚSTRIA DE TRANSFORMAÇÃO'). Opcional.
        porte: Porte do cliente (ex: 'MICRO', 'PEQUENA', 'MÉDIA', 'GRANDE'). Opcional.
        busca: Busca textual livre em todos os campos. Opcional.
        nao_automaticas: Se True, busca operações não-automáticas (com projeto). Padrão False.
        limite: Máximo de resultados (padrão 20, máximo 100).
        offset: Deslocamento para paginação. Padrão 0.

    Returns:
        Operações encontradas com cliente, valor, UF e setor.
    """
    tipo = "não-automáticas" if nao_automaticas else "automáticas"
    await ctx.info(f"Consultando operações {tipo} do BNDES...")
    resultado = await client.consultar_operacoes(
        uf=uf,
        setor=setor,
        porte=porte,
        busca=busca,
        limite=limite,
        offset=offset,
        nao_automaticas=nao_automaticas,
    )

    if not resultado.records:
        return "Nenhuma operação encontrada com os filtros informados."

    n = len(resultado.records)
    lines = [f"**{resultado.total} operações {tipo} encontradas** (mostrando {n})\n"]
    for i, rec in enumerate(resultado.records, 1):
        cliente = rec.get("cliente", "N/A")
        uf_val = str(rec.get("uf", "N/A")).strip()
        municipio = rec.get("municipio", "")
        valor = rec.get("valor_da_operacao_em_reais")
        valor_fmt = f"R$ {valor:,.2f}" if isinstance(valor, (int, float)) else str(valor or "N/A")
        setor_val = rec.get("setor_cnae", "N/A")
        porte_val = rec.get("porte_do_cliente", "N/A")
        situacao = rec.get("situacao_da_operacao") or rec.get("situacao_do_contrato", "N/A")
        data = rec.get("data_da_contratacao", "N/A")
        lines.extend(
            [
                f"### {i}. {cliente}",
                f"**UF:** {uf_val}/{municipio}" if municipio else f"**UF:** {uf_val}",
                f"**Valor:** {valor_fmt} | **Porte:** {porte_val}",
                f"**Setor:** {setor_val}",
                f"**Situação:** {situacao} | **Data:** {data}",
                "",
            ]
        )

    if resultado.total > offset + len(resultado.records):
        next_offset = offset + len(resultado.records)
        lines.append(f"*Use offset={next_offset} para mais resultados.*")
    return "\n".join(lines)


async def listar_instituicoes_bndes(
    ctx: Context,
) -> str:
    """Lista instituições financeiras credenciadas pelo BNDES.

    Mostra os bancos e instituições autorizados a operar com
    recursos do BNDES em operações indiretas de financiamento.

    Returns:
        Lista de instituições credenciadas com CNPJ e site.
    """
    await ctx.info("Listando instituições credenciadas...")
    instituicoes = await client.listar_instituicoes_credenciadas()

    if not instituicoes:
        return "Nenhuma instituição credenciada encontrada."

    lines = [f"**{len(instituicoes)} instituições credenciadas pelo BNDES**\n"]
    for i, inst in enumerate(instituicoes, 1):
        site = inst.pagina_na_internet or "N/A"
        lines.append(f"{i}. **{inst.nome or 'N/A'}** — CNPJ: {inst.cnpj or 'N/A'} | Site: {site}")
    return "\n".join(lines)
