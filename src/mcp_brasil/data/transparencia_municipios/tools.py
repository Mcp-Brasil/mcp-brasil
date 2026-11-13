"""MCP tools for Brazilian municipal and state transparency."""

from __future__ import annotations

from mcp_brasil._shared.formatting import markdown_table

from . import client


async def buscar_municipios(
    query: str = "",
    uf: str | None = None,
    limit: int = 20,
) -> str:
    """Busca instantânea entre todos os 5.570 municípios brasileiros e UFs.

    Pesquisa por nome (com ou sem acento), código IBGE de 7 dígitos ou código SIAFI.
    O motor nativo em Go priorizado garante resposta em tempo recorde (< 1ms).

    Args:
        query: Termo de busca (ex: "São Paulo", "Campinas", "3550308", "7107").
        uf: Filtrar pela sigla da UF (ex: "SP", "RJ", "MG"). Opcional.
        limit: Quantidade máxima de resultados (padrão: 20).
    """
    res = await client.buscar_municipios(query=query, uf=uf, limit=limit)
    if not res.municipios:
        return f"Nenhum município encontrado para a busca: '{query}' (UF: {uf or 'todas'})."

    headers = [
        "Cód. IBGE",
        "Município",
        "UF",
        "Região",
        "Capital",
        "População",
        "CNPJ Prefeitura",
        "Portal Transparência",
    ]
    rows = []
    for m in res.municipios:
        portal = f"https://transparencia.{m.slug}.{m.uf.lower()}.gov.br"
        pop_str = f"{m.populacao:,}" if m.populacao else "N/D"
        rows.append(
            [
                m.codigo_ibge,
                m.nome,
                m.uf,
                m.regiao,
                "Sim" if m.capital else "Não",
                pop_str,
                m.cnpj or "N/D",
                portal,
            ]
        )

    header_text = f"**{res.total_encontrado} municípios encontrados** (exibindo até {limit}):\n\n"
    return header_text + markdown_table(headers, rows)


async def listar_estados() -> str:
    """Lista todas as 27 Unidades Federativas do Brasil e seus portais de transparência e TCEs.

    Retorna tabela com as 26 UFs + Distrito Federal, número de municípios,
    Portal da Transparência oficial do Estado e Tribunal de Contas responsável.
    """
    estados = await client.listar_estados()
    if not estados:
        return "Nenhuma UF encontrada."

    headers = [
        "UF",
        "Estado",
        "Região",
        "Capital",
        "Municípios",
        "Portal Transparência",
        "Tribunal de Contas",
    ]
    rows = []
    for e in estados:
        rows.append(
            [
                e.uf,
                e.nome,
                e.regiao,
                e.capital,
                str(e.total_municipios),
                e.portal_transparencia,
                e.tribunal_contas_nome,
            ]
        )

    header_text = (
        f"**Brasil — {len(estados)} Unidades Federativas e Portais de Transparência**:\n\n"
    )
    return header_text + markdown_table(headers, rows)


async def conectar_municipio(
    municipio: str,
    uf: str | None = None,
) -> str:
    """Conecta a qualquer um dos 5.570 municípios brasileiros e gera dossiê de transparência.

    Mapeia os links oficiais de transparência: Prefeitura, Câmara Municipal, Tribunal de Contas
    competente (TCE ou TCM), Radar ATRICON, SICONFI (dados fiscais), PNCP e CGU.

    Args:
        municipio: Nome do município ou código IBGE (ex: "São Paulo", "Campinas", "3550308").
        uf: Sigla do estado (ex: "SP", "MG"). Recomendado para cidades homônimas.
    """
    try:
        dos = await client.conectar_municipio(municipio=municipio, uf=uf)
    except ValueError as err:
        return str(err)

    m = dos.municipio
    cap_str = "Sim" if m.capital else "Não"
    pop_str = f"{m.populacao:,} habitantes" if m.populacao else "N/D"
    cnpj_str = m.cnpj or "N/D"

    pncp_link = (
        f"[{dos.pncp_cnpj_url}]({dos.pncp_cnpj_url})" if dos.pncp_cnpj_url else dos.pncp_url
    )

    return f"""### 🇧🇷 Dossiê de Transparência: {m.nome} ({m.uf})

**Identificação Territorial & Fiscal:**
- **Código IBGE:** `{m.codigo_ibge}` | **Código SIAFI:** `{m.siafi_id}`
- **CNPJ da Prefeitura:** `{cnpj_str}`
- **População Estimada:** {pop_str}
- **Região:** {m.regiao} | **DDD:** {m.ddd} | **Capital:** {cap_str}
- **Coordenadas:** Lat `{m.latitude}`, Long `{m.longitude}`

**Portais Oficiais do Município:**
- 🏛️ **Portal da Transparência (Prefeitura):** {dos.portal_transparencia_prefeitura}
- 🌐 **Site Oficial da Prefeitura:** {dos.portal_prefeitura}
- 📜 **Câmara Municipal de Vereadores:** {dos.portal_camara}

**Controle Externo & Avaliação:**
- ⚖️ **Tribunal de Contas:** [{dos.tribunal_contas_competente}]({dos.tribunal_contas_url})
- 🏅 **Radar Nacional (ATRICON):** [Acessar Radar]({dos.radar_transparencia_url})

**Fontes Nacionais Integradas:**
- 💰 **Dados Fiscais SICONFI:** [Declarações DCA/RREO/RGF]({dos.siconfi_url})
- 🛒 **Contratações Públicas (PNCP):** {pncp_link}
- 🤝 **Transferências Federais (Transferegov):** [Repasses da União]({dos.transferegov_url})
- 🔍 **Transparência Federal CGU:** [Benefícios e Convênios]({dos.transparencia_federal_url})
"""


async def conectar_estado(uf: str) -> str:
    """Conecta a qualquer uma das 27 Unidades Federativas e gera seu dossiê de transparência.

    Mapeia os links do Portal da Transparência Estadual, Portal do Governo, Tribunal de Contas
    do Estado (TCE), Assembleia Legislativa, SICONFI estadual e PNCP.

    Args:
        uf: Sigla da UF com 2 letras (ex: "SP", "RJ", "MG", "BA", "RS", "DF").
    """
    try:
        dos = await client.conectar_estado(uf=uf)
    except ValueError as err:
        return str(err)

    est = dos.estado
    principais_md = ""
    if dos.municipios_principais:
        m_list = [f"`{m.nome}` (IBGE: {m.codigo_ibge})" for m in dos.municipios_principais[:8]]
        principais_md = "\n**Municípios em Destaque:** " + ", ".join(m_list) + "\n"

    return f"""### 🇧🇷 Dossiê de Transparência Estadual: {est.nome} ({est.uf})

**Identificação:**
- **Código IBGE da UF:** `{est.codigo_uf}`
- **Região:** {est.regiao} | **Capital:** {est.capital}
- **Total de Municípios:** {dos.total_municipios}
{principais_md}
**Portais e Órgãos Estaduais:**
- 🏛️ **Portal da Transparência do Estado:** {dos.portal_transparencia}
- 🌐 **Portal Oficial do Governo Estadual:** {dos.portal_governo}
- ⚖️ **Tribunal de Contas do Estado:** [{dos.tribunal_contas_nome}]({dos.tribunal_contas_url})
- 📜 **Poder Legislativo:** [{dos.assembleia_nome}]({dos.assembleia_url})


**Fontes Nacionais Integradas:**
- 🏅 **Radar Nacional (ATRICON):** [Avaliação de Transparência]({dos.radar_transparencia_url})
- 💰 **Dados Fiscais SICONFI:** [Demonstrativos Fiscais do Estado]({dos.siconfi_url})
- 🛒 **Contratações Públicas (PNCP):** [Licitações e Contratos no PNCP]({dos.pncp_url})
"""


async def pesquisar_transparencia(
    municipio_ou_estado: str,
    uf: str | None = None,
    tipo: str = "todas",
) -> str:
    """Pesquisa nativamente a transparência de um município ou estado.

    Gera os conectores e acessos para auditoria de receitas, despesas, contratações,
    relatórios da LRF (RREO, RGF, DCA), transferências federais e tribunal de contas.

    Args:
        municipio_ou_estado: Nome ou código IBGE ou sigla da UF (ex: "Curitiba", "SP", "3550308").
        uf: Sigla da UF caso seja município homônimo (ex: "PR", "MG"). Opcional.
        tipo: Tipo de transparência desejada — "fiscal", "compras", "transferencias" ou "todas".
    """
    alvo = municipio_ou_estado.strip()

    if len(alvo) == 2 and (not uf or uf == alvo):
        try:
            dos_est = await client.conectar_estado(alvo)
            est = dos_est.estado
            return f"""### 📊 Pesquisa de Transparência: Estado de {est.nome} ({est.uf})

**Tipo de Consulta:** {tipo.upper()}

**Canais e Fontes Diretas:**
1. **Transparência Direta Estadual:** {est.portal_transparencia}
2. **Tribunal de Contas (TCE):** {est.tribunal_contas_url}
3. **Execução Orçamentária e Balanços (SICONFI):**
   - Declarações Fiscais: {dos_est.siconfi_url}
   - Endpoint API SICONFI: `{dos_est.siconfi_api}`
4. **Compras Públicas e Licitações (PNCP):**
   - Contratos e Editais: {dos_est.pncp_url}
5. **Avaliação no Radar ATRICON:**
   - Índice de Transparência: {est.radar_transparencia_url}
"""
        except ValueError:
            pass

    try:
        dos_muni = await client.conectar_municipio(municipio=alvo, uf=uf)
    except ValueError as err:
        return str(err)

    m = dos_muni.municipio
    return f"""### 📊 Pesquisa de Transparência: {m.nome} ({m.uf})

**Tipo de Consulta:** {tipo.upper()}
**Código IBGE:** `{m.codigo_ibge}` | **Código SIAFI:** `{m.siafi_id}`

**Canais e Fontes Diretas:**
1. **Portal da Transparência da Prefeitura:** {dos_muni.portal_transparencia_prefeitura}
2. **Tribunal de Contas:** [{dos_muni.tribunal_contas_competente}]({dos_muni.tribunal_contas_url})
3. **Execução Orçamentária (SICONFI / Tesouro Nacional):**
   - Declarações DCA / RREO / RGF: {dos_muni.siconfi_url}
   - Endpoint API: `{dos_muni.siconfi_api}`
4. **Contratações e Licitações Públicas (PNCP):**
   - Consulta Nacional PNCP: {dos_muni.pncp_url}
5. **Repasses e Convênios Federais:**
   - Transferegov: {dos_muni.transferegov_url}
   - Portal da Transparência Federal: {dos_muni.transparencia_federal_url}
6. **Avaliação no Radar ATRICON:**
   - Selo de Transparência Pública: {dos_muni.radar_transparencia_url}
"""


async def comparar_transparencia(alvos: list[str]) -> str:
    """Compara estrutura de transparência e canais de fiscalização entre municípios e estados.

    Args:
        alvos: Lista de municípios ou UFs (ex: ["SP", "RJ", "Belo Horizonte", "Curitiba"]).
    """
    if not alvos:
        return "Informe pelo menos um município ou UF para comparar."

    rel = await client.comparar_transparencia(alvos)
    if not rel.itens:
        return f"Nenhum município ou estado encontrado para os alvos: {alvos}"

    headers = [
        "Tipo",
        "Ente",
        "UF",
        "IBGE",
        "Região",
        "Portal Transparência",
        "Tribunal de Contas",
        "PNCP Compras",
    ]
    rows = []
    for item in rel.itens:
        rows.append(
            [
                item.tipo.capitalize(),
                item.nome,
                item.uf,
                item.codigo_ibge,
                item.regiao,
                item.portal_transparencia,
                item.tribunal_contas_nome,
                item.pncp_url,
            ]
        )

    header_text = f"**Comparativo de Estrutura de Transparência ({rel.total_itens} entes)**:\n\n"
    return header_text + markdown_table(headers, rows)


async def estatisticas_nacionais() -> str:
    """Retorna estatísticas consolidadas nacionais sobre municípios e estados brasileiros."""
    stats = await client.obter_estatisticas()

    res = f"""### 🇧🇷 Panorama Nacional de Municípios e Transparência Pública

- **Total de Municípios e Unidades Territoriais:** {stats.total_municipios:,}
- **Total de Unidades Federativas (UFs):** {stats.total_ufs} (26 estados + Distrito Federal)
- **Capitais Conectadas:** {len(stats.capitais)}
"""
    if stats.populacao_total > 0:
        res += f"- **População Total Estimada (IBGE):** {stats.populacao_total:,} habitantes\n"

    res += "\n**Distribuição de Municípios por Região Geográfica:**\n"
    for reg, count in sorted(stats.municipios_por_regiao.items()):
        res += f"- **{reg}:** {count} municípios\n"

    res += "\n**Top Estados por Quantidade de Municípios:**\n"
    sorted_ufs = sorted(stats.municipios_por_uf.items(), key=lambda x: x[1], reverse=True)
    for uf, count in sorted_ufs[:10]:
        res += f"- **{uf}:** {count} municípios\n"

    return res


async def buscar_por_cnpj(cnpj: str) -> str:
    """Busca dossiê completo de transparência de um município pelo CNPJ oficial da Prefeitura.

    Pesquisa instantânea entre todas as 5.570 prefeituras municipais do Brasil.

    Args:
        cnpj: CNPJ da prefeitura com 14 dígitos ou formatado (ex: "01613940000119").
    """
    try:
        dos = await client.buscar_por_cnpj(cnpj=cnpj)
    except ValueError as err:
        return str(err)

    m = dos.municipio
    cap_str = "Sim" if m.capital else "Não"
    pop_str = f"{m.populacao:,} habitantes" if m.populacao else "N/D"
    pncp_link = (
        f"[{dos.pncp_cnpj_url}]({dos.pncp_cnpj_url})" if dos.pncp_cnpj_url else dos.pncp_url
    )

    return f"""### 🇧🇷 Município Localizado por CNPJ: {m.nome} ({m.uf})

**Identificação Oficial:**
- **CNPJ da Prefeitura:** `{m.cnpj}`
- **Código IBGE:** `{m.codigo_ibge}` | **SIAFI:** `{m.siafi_id}`
- **População Estimada:** {pop_str}
- **Região:** {m.regiao} | **DDD:** {m.ddd} | **Capital:** {cap_str}

**Portais de Transparência e Controle:**
- 🏛️ **Portal da Transparência:** {dos.portal_transparencia_prefeitura}
- 🌐 **Site da Prefeitura:** {dos.portal_prefeitura}
- 📜 **Câmara de Vereadores:** {dos.portal_camara}
- ⚖️ **Tribunal de Contas:** [{dos.tribunal_contas_competente}]({dos.tribunal_contas_url})
- 🛒 **Contratações no PNCP:** {pncp_link}
- 💰 **Balanço Contábil SICONFI:** [Acessar SICONFI]({dos.siconfi_url})
"""


async def consultar_fiscal(
    municipio_ou_ibge: str,
    uf: str | None = None,
    ano: int = 0,
    tipo: str = "dca",
    periodo: int = 1,
    limite: int = 25,
) -> str:
    """Consulta dados fiscais e contábeis oficiais no SICONFI/Tesouro Nacional.

    Acessa a Declaração de Contas Anuais (DCA), o Relatório Resumido da Execução
    Orçamentária bimestral (RREO) ou o Relatório de Gestão Fiscal (RGF - despesa com pessoal LRF).

    Args:
        municipio_ou_ibge: Nome do município ou código IBGE (ex: "Curitiba", "3550308").
        uf: Sigla do estado (ex: "PR", "SP"). Opcional.
        ano: Ano de exercício (padrão: 2023 para DCA, 2024 para RREO/RGF).
        tipo: "dca" (balanço anual), "rreo" (execução bimestral) ou "rgf" (gastos com pessoal LRF).
        periodo: Bimestre (1 a 6) para RREO ou quadrimestre/semestre (1 a 3) para RGF. Padrão: 1.
        limite: Quantidade máxima de linhas a exibir na tabela (padrão: 25).
    """
    alvo = municipio_ou_ibge.strip()
    nome_exibicao = alvo
    cod_ibge = alvo

    if not alvo.isdigit() or len(alvo) != 7:
        res = await client.buscar_municipios(query=alvo, uf=uf, limit=1)
        if not res.municipios:
            return f"Município não encontrado para a consulta: '{alvo}' (UF: {uf or 'todas'})."
        m = res.municipios[0]
        cod_ibge = m.codigo_ibge
        nome_exibicao = f"{m.nome} ({m.uf})"

    try:
        items = await client.consultar_fiscal(
            cod_ibge=cod_ibge,
            ano=ano,
            tipo=tipo,
            periodo=periodo,
        )
    except Exception as err:
        return f"Falha ao consultar dados fiscais no SICONFI: {err}"

    if not items:
        return (
            f"Nenhum dado fiscal encontrado no SICONFI para {nome_exibicao} "
            f"(IBGE: {cod_ibge}) no exercício {ano or 'padrão'} ({tipo.upper()})."
        )

    headers = [
        "Exercício",
        "Cód. Conta",
        "Conta / Descrição",
        "Coluna / Período",
        "Valor (R$)",
    ]
    rows = []
    for it in items[:limite]:
        rows.append(
            [
                str(it.exercicio),
                it.cod_conta,
                it.conta[:45] + "..." if len(it.conta) > 45 else it.conta,
                it.coluna,
                f"R$ {it.valor:,.2f}",
            ]
        )

    tipo_low = tipo.lower().strip()
    if tipo_low == "dca":
        tipo_nome = "Balanço e Contas Anuais (DCA)"
    elif tipo_low == "rgf":
        tipo_nome = f"Relatório de Gestão Fiscal / Pessoal LRF (RGF - Período {periodo})"
    else:
        tipo_nome = f"Execução Orçamentária (RREO - Bimestre {periodo})"
    res_text = (
        f"### 💰 SICONFI — {tipo_nome}: {nome_exibicao}\n\n"
        f"**Código IBGE:** `{cod_ibge}` | **Total de Contas:** {len(items)}\n\n"
        + markdown_table(headers, rows)
    )
    if len(items) > limite:
        res_text += f"\n\n*Exibindo primeiras {limite} de {len(items)} linhas contábeis.*"
    return res_text


async def consultar_entregas(
    municipio_ou_ibge: str,
    uf: str | None = None,
    ano: int = 0,
) -> str:
    """Consulta o status de cumprimento das entregas de relatórios da LRF no SICONFI.

    Verifica se o município enviou no prazo os relatórios exigidos pela Lei de Responsabilidade
    Fiscal (MSC, DCA, RREO, RGF) para o Tesouro Nacional.

    Args:
        municipio_ou_ibge: Nome do município ou código IBGE (ex: "3550308", "São Paulo").
        uf: Sigla do estado (ex: "SP"). Opcional.
        ano: Ano de referência (padrão: ano corrente/2024).
    """
    alvo = municipio_ou_ibge.strip()
    nome_exibicao = alvo
    cod_ibge = alvo

    if not alvo.isdigit() or len(alvo) != 7:
        res = await client.buscar_municipios(query=alvo, uf=uf, limit=1)
        if not res.municipios:
            return f"Município não encontrado para a consulta: '{alvo}' (UF: {uf or 'todas'})."
        m = res.municipios[0]
        cod_ibge = m.codigo_ibge
        nome_exibicao = f"{m.nome} ({m.uf})"

    try:
        items = await client.consultar_entregas(cod_ibge=cod_ibge, ano=ano)
    except Exception as err:
        return f"Falha ao consultar extrato de entregas no SICONFI: {err}"

    if not items:
        return f"Nenhuma entrega registrada no SICONFI para {nome_exibicao} no ano {ano or 2024}."

    headers = [
        "Exercício",
        "Entregável",
        "Período",
        "Data Envio / Status",
        "Formato",
        "Órgão / Instituição",
    ]
    rows = []
    for it in items:
        data_str = it.data_status[:10] if len(it.data_status) >= 10 else it.data_status
        periodo_str = f"{it.periodo} ({it.periodicidade})"
        inst_str = it.instituicao[:30] + "..." if len(it.instituicao) > 30 else it.instituicao
        rows.append(
            [
                str(it.exercicio),
                it.entregavel,
                periodo_str,
                data_str,
                it.forma_envio,
                inst_str,
            ]
        )

    res_text = (
        f"### 📋 Cumprimento da LRF (SICONFI): {nome_exibicao}\n\n"
        f"**Código IBGE:** `{cod_ibge}` | **Total de Entregas Registradas:** {len(items)}\n\n"
        + markdown_table(headers, rows)
    )
    return res_text


def _trunc(text: str, max_len: int = 50) -> str:
    text = text.strip()
    if len(text) <= max_len:
        return text
    return text[: max_len - 3] + "..."


async def descobrir_portal_municipio(
    municipio: str,
    uf: str | None = None,
    url: str | None = None,
) -> str:
    """Descobre dinamicamente o portal real e a plataforma de transparência de qualquer município.

    Resolve variações de domínio (slug compacto, com hífen, sem preposições, raspagem da
    prefeitura e base do TCE-PI), identifica a tecnologia do portal (Quasar, Fiorilli SCPI,
    IPM/Atende.Net, Betha, Sonner, EL, HTML ou proteção anti-bot) e verifica suporte a APIs.

    Args:
        municipio: Nome do município ou código IBGE (ex: "Serra da Saudade", "Araxá", "3166600").
        uf: Sigla da UF (ex: "MG", "SP", "CE"). Recomendado para cidades homônimas.
        url: URL explícita do portal da transparência para inspeção direta (opcional).
    """
    try:
        res = await client.descobrir_portal_municipio(municipio=municipio, uf=uf, url=url)
    except Exception as err:
        return f"Erro ao descobrir portal municipal: {err}"

    m = res.municipio
    suporte = (
        "Sim (consulta direta em tempo real)"
        if res.servidores_suportado
        else "Não (usar TCE/SICONFI ou acesso web)"
    )
    entidade = res.entidade_nome or "N/D"
    cod_adm = str(res.cod_administracao) if res.cod_administracao else "N/D"
    metodo = res.fonte_descoberta or "automático"

    lines = [
        f"### 🔎 Descoberta de Portal da Transparência: {m.nome} ({m.uf})",
        "",
        f"- **Código IBGE:** `{m.codigo_ibge}` | **CNPJ:** `{m.cnpj or 'N/D'}`",
        f"- **URL Resolvida:** {res.url_resolvida or 'Não identificada automaticamente'}",
        f"- **Plataforma / Tecnologia:** `{res.plataforma}`"
        + (f" (versão `{res.versao}`)" if res.versao else ""),
        f"- **Entidade / Cód. Administração:** {entidade} (`{cod_adm}`)",
        f"- **API de Servidores Suportada:** {suporte}",
        f"- **Método de Descoberta:** `{metodo}` ({res.tempo_ms} ms)",
    ]
    if res.site_prefeitura:
        lines.append(f"- **Site da Prefeitura:** {res.site_prefeitura}")
    if res.portal_camara:
        lines.append(f"- **Portal da Câmara:** {res.portal_camara}")
    if res.observacoes:
        lines.append(f"- **Observações:** {res.observacoes}")
    if res.candidatos_testados:
        lines.append(
            f"- **Candidatos Testados ({len(res.candidatos_testados)}):** "
            + ", ".join(f"`{c}`" for c in res.candidatos_testados[:6])
        )
    return "\n".join(lines)


async def consultar_servidores_municipio(
    municipio: str,
    uf: str | None = None,
    nome: str = "",
    inicio: str = "",
    fim: str = "",
    ano: int = 0,
    mes: int = 0,
    limite: int = 20,
    url: str | None = None,
) -> str:
    """Consulta servidores públicos municipais, cargos, lotação e remuneração em tempo real.

    Conecta automaticamente via melhor rota disponível: portal municipal estruturado
    (Quasar `/publico/servidor`, Fiorilli SCPI) ou API aberta de Tribunal de Contas estadual
    (TCE-PI Portal da Cidadania para os 224 municípios do Piauí; TCE-CE Dados Abertos SIM + folha
    para os 184 municípios do Ceará). CPFs são sempre mascarados em conformidade com a LGPD.

    Args:
        municipio: Nome do município ou código IBGE (ex: "Serra da Saudade", "Araxá", "Teresina").
        uf: Sigla da UF (ex: "MG", "PI", "CE"). Recomendado.
        nome: Filtro por nome do servidor (ex: "Maria", "Silva"). Opcional.
        inicio: Data/competência inicial (ex: "2025-01-01" ou "202501"). Opcional.
        fim: Data/competência final (ex: "2025-12-31"). Opcional.
        ano: Ano de referência (opcional; busca automático do mais recente se omitido).
        mes: Mês de competência (1 a 12). Opcional.
        limite: Quantidade máxima de servidores na tabela (padrão: 20).
        url: URL explícita do portal municipal, caso conhecida (opcional).
    """
    try:
        res = await client.consultar_servidores_municipio(
            municipio=municipio,
            uf=uf,
            nome=nome,
            inicio=inicio,
            fim=fim,
            ano=ano,
            mes=mes,
            limit=limite,
            url=url,
        )
    except Exception as err:
        return f"Erro ao consultar servidores de '{municipio}': {err}"

    m = res.municipio
    if not res.disponivel:
        return (
            f"### 👥 Servidores Públicos — {m.nome} ({m.uf})\n\n"
            f"- **Fonte:** {res.fonte} | **Plataforma:** `{res.plataforma}`\n"
            f"- **Portal:** {res.portal_url or 'N/D'}\n"
            f"- **Status:** {res.mensagem}\n"
        )

    if not res.servidores:
        return (
            f"### 👥 Servidores Públicos — {m.nome} ({m.uf})\n\n"
            f"- **Fonte:** {res.fonte} (`{res.plataforma}`)\n"
            f"- **Portal:** {res.portal_url}\n"
            f"- **Resultado:** Nenhum servidor retornado para os filtros informados."
            + (f" ({res.mensagem})" if res.mensagem else "")
        )

    headers = [
        "Nome",
        "CPF (LGPD)",
        "Cargo / Função",
        "Lotação / Vínculo",
        "Comp.",
        "Salário Bruto",
        "Salário Líquido",
    ]
    rows = []
    for s in res.servidores[:limite]:
        cargo_fn = _trunc(s.cargo or s.funcao or "N/D", 32)
        lot_vinc = _trunc(s.lotacao or s.orgao or s.vinculo or "N/D", 28)
        bruto = f"R$ {s.salario_bruto:,.2f}" if s.salario_bruto else "N/D"
        liq = f"R$ {s.salario_liquido:,.2f}" if s.salario_liquido else "N/D"
        rows.append(
            [
                _trunc(s.nome, 34),
                s.cpf_parcial or "N/D",
                cargo_fn,
                lot_vinc,
                s.competencia or "N/D",
                bruto,
                liq,
            ]
        )

    header_md = (
        f"### 👥 Servidores Públicos — {m.nome} ({m.uf})\n\n"
        f"- **Fonte:** {res.fonte} (`{res.plataforma}`)\n"
        f"- **Portal:** {res.portal_url}\n"
        f"- **Total Encontrado:** {res.total_registros:,} registros "
        f"(exibindo {len(rows)})\n\n"
    )
    return header_md + markdown_table(headers, rows)


async def consultar_despesas_municipio(
    municipio: str,
    uf: str | None = None,
    nome: str = "",
    inicio: str = "",
    fim: str = "",
    ano: int = 0,
    mes: int = 0,
    limite: int = 20,
    url: str | None = None,
) -> str:
    """Consulta despesas e empenhos detalhados de qualquer município brasileiro.

    Conecta aos portais municipais (Quasar/Fiorilli) ou às APIs abertas dos Tribunais de Contas
    (TCE-SP para os 645 municípios paulistas e TCE-PI para os 224 municípios piauienses).

    Args:
        municipio: Nome do município ou código IBGE (ex: "Serra da Saudade", "Adamantina").
        uf: Sigla da UF (ex: "MG", "SP", "PI"). Opcional.
        nome: Filtro por fornecedor/credor ou histórico (opcional).
        inicio: Data inicial `YYYY-MM-DD` (opcional).
        fim: Data final `YYYY-MM-DD` (opcional).
        ano: Ano de exercício (opcional).
        mes: Mês (1 a 12). Opcional.
        limite: Quantidade máxima de despesas exibidas (padrão: 20).
        url: URL explícita do portal municipal (opcional).
    """
    try:
        res = await client.consultar_despesas_municipio(
            municipio=municipio,
            uf=uf,
            nome=nome,
            inicio=inicio,
            fim=fim,
            ano=ano,
            mes=mes,
            limit=limite,
            url=url,
        )
    except Exception as err:
        return f"Erro ao consultar despesas de '{municipio}': {err}"

    m = res.municipio
    if not res.disponivel or not res.despesas:
        return (
            f"### 💸 Despesas Municipais — {m.nome} ({m.uf})\n\n"
            f"- **Fonte:** {res.fonte} (`{res.plataforma}`)\n"
            f"- **Status:** {res.mensagem or 'Nenhum registro de despesa encontrado.'}\n"
        )

    headers = [
        "Empenho",
        "Data",
        "Fornecedor / Credor",
        "Função / Histórico",
        "Empenhado",
        "Pago",
    ]
    rows = []
    for d in res.despesas[:limite]:
        rows.append(
            [
                d.numero_empenho or str(d.ano or "N/D"),
                d.data or (str(d.ano) if d.ano else "N/D"),
                _trunc(d.fornecedor or d.orgao or "N/D", 30),
                _trunc(d.historico or d.funcao or d.elemento or "N/D", 35),
                f"R$ {d.valor_empenhado:,.2f}" if d.valor_empenhado else "N/D",
                f"R$ {d.valor_pago:,.2f}" if d.valor_pago else "N/D",
            ]
        )

    header_md = (
        f"### 💸 Despesas e Empenhos — {m.nome} ({m.uf})\n\n"
        f"- **Fonte:** {res.fonte} (`{res.plataforma}`)\n"
        f"- **Total de Registros:** {res.total_registros:,} (exibindo {len(rows)})\n\n"
    )
    return header_md + markdown_table(headers, rows)


async def consultar_receitas_municipio(
    municipio: str,
    uf: str | None = None,
    inicio: str = "",
    fim: str = "",
    ano: int = 0,
    mes: int = 0,
    limite: int = 20,
    url: str | None = None,
) -> str:
    """Consulta receitas públicas arrecadadas de qualquer município brasileiro.

    Conecta aos portais municipais (Quasar/Fiorilli) ou APIs abertas de TCEs (TCE-SP e TCE-PI).

    Args:
        municipio: Nome do município ou código IBGE (ex: "Serra da Saudade", "Campinas").
        uf: Sigla da UF (ex: "MG", "SP"). Opcional.
        inicio: Data inicial `YYYY-MM-DD` (opcional).
        fim: Data final `YYYY-MM-DD` (opcional).
        ano: Ano de exercício (opcional).
        mes: Mês (1 a 12). Opcional.
        limite: Quantidade máxima de receitas exibidas (padrão: 20).
        url: URL explícita do portal municipal (opcional).
    """
    try:
        res = await client.consultar_receitas_municipio(
            municipio=municipio,
            uf=uf,
            inicio=inicio,
            fim=fim,
            ano=ano,
            mes=mes,
            limit=limite,
            url=url,
        )
    except Exception as err:
        return f"Erro ao consultar receitas de '{municipio}': {err}"

    m = res.municipio
    if not res.disponivel or not res.receitas:
        return (
            f"### 📈 Receitas Municipais — {m.nome} ({m.uf})\n\n"
            f"- **Fonte:** {res.fonte} (`{res.plataforma}`)\n"
            f"- **Status:** {res.mensagem or 'Nenhum registro de receita encontrado.'}\n"
        )

    headers = ["Código", "Descrição", "Categoria / Fonte", "Data", "Previsto", "Arrecadado"]
    rows = []
    for r in res.receitas[:limite]:
        rows.append(
            [
                r.codigo or "N/D",
                _trunc(r.descricao or "N/D", 38),
                _trunc(r.categoria or "N/D", 22),
                r.data or "N/D",
                f"R$ {r.valor_previsto:,.2f}" if r.valor_previsto else "N/D",
                f"R$ {r.valor_arrecadado:,.2f}" if r.valor_arrecadado else "N/D",
            ]
        )

    header_md = (
        f"### 📈 Receitas Públicas — {m.nome} ({m.uf})\n\n"
        f"- **Fonte:** {res.fonte} (`{res.plataforma}`)\n"
        f"- **Total de Registros:** {res.total_registros:,} (exibindo {len(rows)})\n\n"
    )
    return header_md + markdown_table(headers, rows)


async def consultar_licitacoes_municipio(
    municipio: str,
    uf: str | None = None,
    nome: str = "",
    inicio: str = "",
    fim: str = "",
    ano: int = 0,
    limite: int = 20,
    url: str | None = None,
) -> str:
    """Consulta licitações e editais de qualquer um dos 5.570 municípios brasileiros.

    Busca diretamente no portal municipal (quando estruturado) com fallback universal para
    a API oficial do PNCP (Portal Nacional de Contratações Públicas) via CNPJ da Prefeitura.

    Args:
        municipio: Nome do município ou código IBGE (ex: "Serra da Saudade", "São Paulo").
        uf: Sigla da UF (ex: "MG", "SP"). Opcional.
        nome: Termo de filtro no objeto da licitação (opcional).
        inicio: Data inicial `YYYY-MM-DD` (opcional; padrão últimos 90 dias no PNCP).
        fim: Data final `YYYY-MM-DD` (opcional).
        ano: Ano de exercício (opcional).
        limite: Quantidade máxima de licitações exibidas (padrão: 20).
        url: URL explícita do portal municipal (opcional).
    """
    try:
        res = await client.consultar_licitacoes_municipio(
            municipio=municipio,
            uf=uf,
            nome=nome,
            inicio=inicio,
            fim=fim,
            ano=ano,
            limit=limite,
            url=url,
        )
    except Exception as err:
        return f"Erro ao consultar licitações de '{municipio}': {err}"

    m = res.municipio
    if not res.disponivel or not res.licitacoes:
        status_msg = res.mensagem or "Nenhuma licitação encontrada no período consultado."
        return (
            f"### ⚖️ Licitações e Editais — {m.nome} ({m.uf})\n\n"
            f"- **Fonte:** {res.fonte} (`{res.plataforma}`)\n"
            f"- **Status:** {status_msg}\n"
        )

    headers = ["Número", "Modalidade", "Situação", "Abertura", "Objeto", "Valor Estimado"]
    rows = []
    for lic in res.licitacoes[:limite]:
        val = lic.valor_homologado or lic.valor_estimado
        rows.append(
            [
                lic.numero or str(lic.ano or "N/D"),
                _trunc(lic.modalidade or "N/D", 22),
                _trunc(lic.situacao or "N/D", 18),
                lic.data_abertura or "N/D",
                _trunc(lic.objeto or "N/D", 42),
                f"R$ {val:,.2f}" if val else "N/D",
            ]
        )

    header_md = (
        f"### ⚖️ Licitações e Contratações — {m.nome} ({m.uf})\n\n"
        f"- **Fonte:** {res.fonte} (`{res.plataforma}`)\n"
        f"- **Total Encontrado:** {res.total_registros:,} (exibindo {len(rows)})\n\n"
    )
    return header_md + markdown_table(headers, rows)


async def consultar_contratos_municipio(
    municipio: str,
    uf: str | None = None,
    nome: str = "",
    inicio: str = "",
    fim: str = "",
    ano: int = 0,
    limite: int = 20,
    url: str | None = None,
) -> str:
    """Consulta contratos administrativos firmados por qualquer município brasileiro.

    Conecta diretamente ao portal municipal ou consulta a API oficial do PNCP por CNPJ.

    Args:
        municipio: Nome do município ou código IBGE (ex: "Serra da Saudade", "Curitiba").
        uf: Sigla da UF (ex: "MG", "PR"). Opcional.
        nome: Filtro por fornecedor ou objeto do contrato (opcional).
        inicio: Data inicial `YYYY-MM-DD` (opcional).
        fim: Data final `YYYY-MM-DD` (opcional).
        ano: Ano de exercício (opcional).
        limite: Quantidade máxima de contratos exibidos (padrão: 20).
        url: URL explícita do portal municipal (opcional).
    """
    try:
        res = await client.consultar_contratos_municipio(
            municipio=municipio,
            uf=uf,
            nome=nome,
            inicio=inicio,
            fim=fim,
            ano=ano,
            limit=limite,
            url=url,
        )
    except Exception as err:
        return f"Erro ao consultar contratos de '{municipio}': {err}"

    m = res.municipio
    if not res.disponivel or not res.contratos:
        status_msg = res.mensagem or "Nenhum contrato encontrado no período consultado."
        return (
            f"### 📄 Contratos Administrativos — {m.nome} ({m.uf})\n\n"
            f"- **Fonte:** {res.fonte} (`{res.plataforma}`)\n"
            f"- **Status:** {status_msg}\n"
        )

    headers = ["Número", "Fornecedor", "Assinatura", "Situação", "Objeto", "Valor Global"]
    rows = []
    for c in res.contratos[:limite]:
        rows.append(
            [
                c.numero or str(c.ano or "N/D"),
                _trunc(c.fornecedor or c.documento_fornecedor or "N/D", 28),
                c.data_assinatura or "N/D",
                _trunc(c.situacao or "N/D", 16),
                _trunc(c.objeto or "N/D", 38),
                f"R$ {c.valor:,.2f}" if c.valor else "N/D",
            ]
        )

    header_md = (
        f"### 📄 Contratos Administrativos — {m.nome} ({m.uf})\n\n"
        f"- **Fonte:** {res.fonte} (`{res.plataforma}`)\n"
        f"- **Total Encontrado:** {res.total_registros:,} (exibindo {len(rows)})\n\n"
    )
    return header_md + markdown_table(headers, rows)


async def consultar_transparencia_completa(
    municipio: str,
    uf: str | None = None,
    ano: int = 0,
    limite: int = 10,
    url: str | None = None,
) -> str:
    """Executa auditoria completa e paralela de toda a transparência de um município brasileiro.

    Orquestra no motor Go: descoberta do portal municipal + servidores públicos (com CPFs
    mascarados pela LGPD) + despesas + receitas + licitações + contratos (portal local ou PNCP)
    + SICONFI DCA, RGF (gastos de pessoal LRF) e entregas obrigatórias da LRF.

    Args:
        municipio: Nome do município ou código IBGE (ex: "Serra da Saudade", "Araxá", "3166600").
        uf: Sigla da UF (ex: "MG", "SP", "PI", "CE"). Recomendado.
        ano: Ano fiscal de referência (opcional; seleciona o mais recente automaticamente).
        limite: Limite de amostras exibidas por seção (padrão: 10).
        url: URL explícita do portal municipal, caso conhecida (opcional).
    """
    try:
        rep = await client.consultar_transparencia_completa(
            municipio=municipio,
            uf=uf,
            ano=ano,
            limit=limite,
            url=url,
        )
    except Exception as err:
        return f"Erro ao executar transparência completa de '{municipio}': {err}"

    m = rep.dossier.municipio
    disc = rep.discovery
    plat = disc.plataforma if disc else "nao_identificado"
    url_real = (
        disc.url_resolvida
        if disc and disc.url_resolvida
        else rep.dossier.portal_transparencia_prefeitura
    )
    tce_nome = rep.dossier.tribunal_contas_competente
    tce_url = rep.dossier.tribunal_contas_url
    fontes_str = ", ".join(rep.fontes_consultadas) if rep.fontes_consultadas else "N/D"

    sections: list[str] = [
        f"### 🏛️ Auditoria Completa de Transparência: {m.nome} ({m.uf})",
        "",
        f"- **IBGE:** `{m.codigo_ibge}` | **CNPJ:** `{m.cnpj or 'N/D'}` | "
        f"**População:** {m.populacao:,}",
        f"- **Portal Resolvido:** {url_real} (`{plat}`)",
        f"- **Tribunal de Contas:** [{tce_nome}]({tce_url})",
        f"- **Tempo Total de Coleta Paralela:** {rep.tempo_total_ms} ms",
        f"- **Fontes Ativas:** {fontes_str}",
        "",
    ]

    if rep.servidores and rep.servidores.servidores:
        rows = [
            [
                _trunc(s.nome, 30),
                _trunc(s.cargo or s.funcao or "N/D", 25),
                s.competencia or "N/D",
                f"R$ {s.salario_bruto:,.2f}" if s.salario_bruto else "N/D",
                f"R$ {s.salario_liquido:,.2f}" if s.salario_liquido else "N/D",
            ]
            for s in rep.servidores.servidores[:limite]
        ]
        serv_title = (
            f"#### 👥 Servidores Públicos ({rep.servidores.total_registros:,} "
            f"total — {rep.servidores.fonte})"
        )
        sections.append(
            f"{serv_title}\n\n"
            + markdown_table(["Nome", "Cargo", "Comp.", "Bruto", "Líquido"], rows)
        )

    if rep.despesas and rep.despesas.despesas:
        rows = [
            [
                d.numero_empenho or str(d.ano or "N/D"),
                _trunc(d.fornecedor or d.orgao or "N/D", 28),
                _trunc(d.historico or d.funcao or "N/D", 30),
                f"R$ {(d.valor_pago or d.valor_empenhado):,.2f}",
            ]
            for d in rep.despesas.despesas[:limite]
        ]
        sections.append(
            f"#### 💸 Despesas ({rep.despesas.total_registros:,} total — {rep.despesas.fonte})\n\n"
            + markdown_table(["Empenho", "Fornecedor/Órgão", "Descrição", "Valor"], rows)
        )

    if rep.receitas and rep.receitas.receitas:
        rows = [
            [
                _trunc(r.descricao or "N/D", 36),
                r.data or "N/D",
                f"R$ {r.valor_arrecadado:,.2f}",
            ]
            for r in rep.receitas.receitas[:limite]
        ]
        sections.append(
            f"#### 📈 Receitas ({rep.receitas.total_registros:,} total — {rep.receitas.fonte})\n\n"
            + markdown_table(["Descrição", "Data", "Arrecadado"], rows)
        )

    if rep.licitacoes and rep.licitacoes.licitacoes:
        rows = [
            [
                lic.numero or "N/D",
                _trunc(lic.modalidade or "N/D", 20),
                _trunc(lic.objeto or "N/D", 34),
                f"R$ {(lic.valor_homologado or lic.valor_estimado):,.2f}",
            ]
            for lic in rep.licitacoes.licitacoes[:limite]
        ]
        lic_title = (
            f"#### ⚖️ Licitações ({rep.licitacoes.total_registros:,} "
            f"total — {rep.licitacoes.fonte})"
        )
        sections.append(
            f"{lic_title}\n\n" + markdown_table(["Número", "Modalidade", "Objeto", "Valor"], rows)
        )

    if rep.contratos and rep.contratos.contratos:
        rows = [
            [
                c.numero or "N/D",
                _trunc(c.fornecedor or "N/D", 26),
                _trunc(c.objeto or "N/D", 32),
                f"R$ {c.valor:,.2f}",
            ]
            for c in rep.contratos.contratos[:limite]
        ]
        cont_title = (
            f"#### 📄 Contratos ({rep.contratos.total_registros:,} total — {rep.contratos.fonte})"
        )
        sections.append(
            f"{cont_title}\n\n" + markdown_table(["Número", "Fornecedor", "Objeto", "Valor"], rows)
        )

    if rep.diarias and rep.diarias.diarias:
        rows = [
            [
                _trunc(d.beneficiario, 30),
                _trunc(d.cargo or "N/D", 22),
                _trunc(d.destino or "N/D", 20),
                f"R$ {d.valor:,.2f}",
            ]
            for d in rep.diarias.diarias[:limite]
        ]
        d_title = (
            f"#### ✈️ Diárias e Viagens ({rep.diarias.total_registros:,} "
            f"total — {rep.diarias.fonte})\n\n"
        )
        sections.append(
            d_title + markdown_table(["Beneficiário", "Cargo", "Destino", "Valor"], rows)
        )

    if rep.obras and rep.obras.obras:
        rows = [
            [
                _trunc(o.titulo or o.objeto, 34),
                _trunc(o.situacao or "N/D", 18),
                f"R$ {o.valor:,.2f}" if o.valor else "N/D",
            ]
            for o in rep.obras.obras[:limite]
        ]
        o_title = (
            f"#### 🏗️ Obras Públicas ({rep.obras.total_registros:,} total — {rep.obras.fonte})\n\n"
        )
        sections.append(o_title + markdown_table(["Obra / Título", "Situação", "Valor"], rows))

    if rep.convenios and rep.convenios.convenios:
        rows = [
            [
                c.numero or "N/D",
                _trunc(c.concedente or "N/D", 24),
                _trunc(c.descricao, 34),
                f"R$ {c.valor:,.2f}" if c.valor else "N/D",
            ]
            for c in rep.convenios.convenios[:limite]
        ]
        conv_title = (
            f"#### 🤝 Convênios ({rep.convenios.total_registros:,} "
            f"total — {rep.convenios.fonte})\n\n"
        )
        sections.append(
            conv_title + markdown_table(["Número", "Concedente", "Objeto", "Valor"], rows)
        )

    if rep.legislacao and rep.legislacao.atos:
        rows = [
            [
                a.numero or str(a.ano or "N/D"),
                _trunc(a.tipo_documento or "N/D", 20),
                _trunc(a.descricao, 40),
            ]
            for a in rep.legislacao.atos[:limite]
        ]
        leg_title = (
            f"#### 📜 Legislação Municipal ({rep.legislacao.total_registros:,} "
            f"total — {rep.legislacao.fonte})\n\n"
        )
        sections.append(leg_title + markdown_table(["Número", "Tipo", "Ementa / Descrição"], rows))

    if rep.concursos and rep.concursos.aprovados:
        rows = [
            [
                _trunc(ap.nome, 30),
                _trunc(ap.cargo or "N/D", 22),
                str(ap.classificacao) if ap.classificacao else "N/D",
                ap.situacao or "N/D",
            ]
            for ap in rep.concursos.aprovados[:limite]
        ]
        conc_title = (
            f"#### 🎓 Concursos e Processos Seletivos ({rep.concursos.total_registros:,} "
            f"total — {rep.concursos.fonte})\n\n"
        )
        sections.append(
            conc_title + markdown_table(["Nome", "Cargo", "Classificação", "Situação"], rows)
        )

    if rep.siconfi_rgf:
        rows = [
            [str(it.exercicio), _trunc(it.conta, 42), it.coluna or "N/D", f"R$ {it.valor:,.2f}"]
            for it in rep.siconfi_rgf[:limite]
        ]
        sections.append(
            "#### 💼 SICONFI — RGF / Gastos com Pessoal (LRF)\n\n"
            + markdown_table(["Exercício", "Conta LRF", "Coluna", "Valor"], rows)
        )

    if rep.siconfi_dca:
        rows = [
            [str(it.exercicio), _trunc(it.conta, 42), it.coluna or "N/D", f"R$ {it.valor:,.2f}"]
            for it in rep.siconfi_dca[:limite]
        ]
        sections.append(
            "#### 💰 SICONFI — DCA (Balanço Anual)\n\n"
            + markdown_table(["Exercício", "Conta", "Coluna", "Valor"], rows)
        )

    if rep.siconfi_entregas:
        sections.append(
            f"#### 📋 SICONFI — Entregas da LRF\n\n"
            f"- **{len(rep.siconfi_entregas)} demonstrativos registrados** "
            f"(ex: {', '.join(e.entregavel for e in rep.siconfi_entregas[:5])})."
        )

    if rep.limitacoes:
        sections.append(
            "#### ⚠️ Observações e Limitações de Coleta\n\n"
            + "\n".join(f"- {lim}" for lim in rep.limitacoes)
        )

    return "\n\n".join(sections)


async def listar_municipios_megasoft(
    uf: str | None = None,
) -> str:
    """Lista todos os 268 municípios brasileiros atendidos pela tecnologia Megasoft (MegaAdmWeb).

    Permite filtrar por UF (ex: "GO", "TO", "AC", "MA", "MS", "PA").
    Retorna URLs diretas dos portais da transparência da Prefeitura e da Câmara Municipal.

    Args:
        uf: Sigla da UF para filtro (ex: "GO", "TO", "MS"). Opcional.
    """
    try:
        munis = await client.listar_municipios_megasoft(uf=uf)
    except Exception as err:
        return f"Erro ao listar municípios Megasoft: {err}"

    if not munis:
        return f"Nenhum município Megasoft encontrado para o filtro UF: '{uf or 'todas'}'."

    headers = [
        "Cód. IBGE",
        "Município",
        "UF",
        "Portal Transparência Prefeitura",
        "Portal Câmara Municipal",
        "Site Prefeitura",
    ]
    rows = []
    for m in munis:
        rows.append(
            [
                m.codigo_ibge,
                m.nome,
                m.uf,
                m.portal_url,
                m.portal_camara or "N/D",
                m.site_prefeitura or "N/D",
            ]
        )

    title = (
        f"### 🏛️ Municípios com Tecnologia Megasoft (MegaAdmWeb)\n\n"
        f"- **Total Cadastrado e Verificado:** {len(munis)} municípios (UF: {uf or 'todas'})\n"
        f"- **Módulos Disponíveis:** Servidores, Contracheque Detalhado, Diárias, Obras, "
        f"Convênios, Legislação, Concursos, Licitações, Contratos, Despesas e Receitas.\n\n"
    )
    return title + markdown_table(headers, rows)


async def consultar_diarias_municipio(
    municipio: str,
    uf: str | None = None,
    nome: str = "",
    inicio: str = "",
    fim: str = "",
    ano: int = 0,
    limite: int = 20,
    url: str | None = None,
) -> str:
    """Consulta pagamentos de diárias e viagens concedidas a servidores e agentes públicos.

    Args:
        municipio: Nome do município ou código IBGE (ex: "Tacuru", "Abadia de Goiás").
        uf: Sigla da UF (ex: "MS", "GO"). Opcional.
        nome: Filtrar por nome do beneficiário (opcional).
        inicio: Data inicial `YYYY-MM-DD` (opcional).
        fim: Data final `YYYY-MM-DD` (opcional).
        ano: Ano de referência (opcional).
        limite: Quantidade máxima de registros exibidos (padrão: 20).
        url: URL base explícita do portal municipal (opcional).
    """
    try:
        res = await client.consultar_diarias_municipio(
            municipio=municipio,
            uf=uf,
            nome=nome,
            inicio=inicio,
            fim=fim,
            ano=ano,
            limit=limite,
            url=url,
        )
    except Exception as err:
        return f"Erro ao consultar diárias de '{municipio}': {err}"

    m = res.municipio
    if not res.disponivel or not res.diarias:
        status_msg = res.mensagem or "Nenhuma diária encontrada no período consultado."
        return (
            f"### ✈️ Diárias e Viagens — {m.nome} ({m.uf})\n\n"
            f"- **Fonte:** {res.fonte} (`{res.plataforma}`)\n"
            f"- **Status:** {status_msg}\n"
        )

    headers = ["Beneficiário", "Cargo", "Destino", "Data", "Valor", "Motivo"]
    rows = []
    for d in res.diarias[:limite]:
        rows.append(
            [
                _trunc(d.beneficiario, 30),
                _trunc(d.cargo or "N/D", 22),
                _trunc(d.destino or "N/D", 20),
                d.data or d.data_saida or "N/D",
                f"R$ {d.valor:,.2f}",
                _trunc(d.motivo or "N/D", 34),
            ]
        )

    header_md = (
        f"### ✈️ Diárias e Viagens de Servidores — {m.nome} ({m.uf})\n\n"
        f"- **Fonte:** {res.fonte} (`{res.plataforma}`)\n"
        f"- **Total de Registros:** {res.total_registros:,} (exibindo {len(rows)})\n\n"
    )
    return header_md + markdown_table(headers, rows)


async def consultar_obras_municipio(
    municipio: str,
    uf: str | None = None,
    ano: int = 0,
    limite: int = 20,
    url: str | None = None,
) -> str:
    """Consulta obras públicas municipais com situação, prazos, localização e valores.

    Args:
        municipio: Nome do município ou código IBGE.
        uf: Sigla da UF. Opcional.
        ano: Ano de exercício (opcional).
        limite: Quantidade máxima de obras exibidas (padrão: 20).
        url: URL base explícita do portal municipal (opcional).
    """
    try:
        res = await client.consultar_obras_municipio(
            municipio=municipio,
            uf=uf,
            ano=ano,
            limit=limite,
            url=url,
        )
    except Exception as err:
        return f"Erro ao consultar obras de '{municipio}': {err}"

    m = res.municipio
    if not res.disponivel or not res.obras:
        status_msg = res.mensagem or "Nenhuma obra pública encontrada."
        return (
            f"### 🏗️ Obras Públicas — {m.nome} ({m.uf})\n\n"
            f"- **Fonte:** {res.fonte} (`{res.plataforma}`)\n"
            f"- **Status:** {status_msg}\n"
        )

    headers = ["Código", "Título / Objeto", "Situação", "Início", "Término Previsto", "Valor"]
    rows = []
    for o in res.obras[:limite]:
        rows.append(
            [
                o.codigo or "N/D",
                _trunc(o.titulo or o.objeto, 36),
                _trunc(o.situacao or "N/D", 18),
                o.data_inicial or "N/D",
                o.data_final or "N/D",
                f"R$ {o.valor:,.2f}" if o.valor else "N/D",
            ]
        )

    header_md = (
        f"### 🏗️ Obras Públicas Municipais — {m.nome} ({m.uf})\n\n"
        f"- **Fonte:** {res.fonte} (`{res.plataforma}`)\n"
        f"- **Total de Obras:** {res.total_registros:,} (exibindo {len(rows)})\n\n"
    )
    return header_md + markdown_table(headers, rows)


async def consultar_convenios_municipio(
    municipio: str,
    uf: str | None = None,
    ano: int = 0,
    limite: int = 20,
    url: str | None = None,
) -> str:
    """Consulta convênios, parcerias e transferências voluntárias do município.

    Args:
        municipio: Nome do município ou código IBGE.
        uf: Sigla da UF. Opcional.
        ano: Ano de referência (opcional).
        limite: Quantidade máxima de convênios exibidos (padrão: 20).
        url: URL base explícita do portal municipal (opcional).
    """
    try:
        res = await client.consultar_convenios_municipio(
            municipio=municipio,
            uf=uf,
            ano=ano,
            limit=limite,
            url=url,
        )
    except Exception as err:
        return f"Erro ao consultar convênios de '{municipio}': {err}"

    m = res.municipio
    if not res.disponivel or not res.convenios:
        status_msg = res.mensagem or "Nenhum convênio encontrado."
        return (
            f"### 🤝 Convênios e Parcerias — {m.nome} ({m.uf})\n\n"
            f"- **Fonte:** {res.fonte} (`{res.plataforma}`)\n"
            f"- **Status:** {status_msg}\n"
        )

    headers = ["Número", "Concedente", "Convenente", "Objeto", "Valor", "Liberado"]
    rows = []
    for c in res.convenios[:limite]:
        rows.append(
            [
                c.numero or "N/D",
                _trunc(c.concedente or "N/D", 22),
                _trunc(c.convenente or "N/D", 20),
                _trunc(c.descricao, 34),
                f"R$ {c.valor:,.2f}" if c.valor else "N/D",
                f"R$ {c.valor_liberado:,.2f}" if c.valor_liberado else "N/D",
            ]
        )

    header_md = (
        f"### 🤝 Convênios e Transferências Voluntárias — {m.nome} ({m.uf})\n\n"
        f"- **Fonte:** {res.fonte} (`{res.plataforma}`)\n"
        f"- **Total de Registros:** {res.total_registros:,} (exibindo {len(rows)})\n\n"
    )
    return header_md + markdown_table(headers, rows)


async def consultar_legislacao_municipio(
    municipio: str,
    uf: str | None = None,
    ano: int = 0,
    limite: int = 20,
    url: str | None = None,
) -> str:
    """Consulta legislação municipal: leis ordinárias, complementares, decretos e portarias.

    Args:
        municipio: Nome do município ou código IBGE.
        uf: Sigla da UF. Opcional.
        ano: Ano de publicação (opcional).
        limite: Quantidade máxima de atos exibidos (padrão: 20).
        url: URL base explícita do portal municipal (opcional).
    """
    try:
        res = await client.consultar_legislacao_municipio(
            municipio=municipio,
            uf=uf,
            ano=ano,
            limit=limite,
            url=url,
        )
    except Exception as err:
        return f"Erro ao consultar legislação de '{municipio}': {err}"

    m = res.municipio
    if not res.disponivel or not res.atos:
        status_msg = res.mensagem or "Nenhum ato legislativo encontrado."
        return (
            f"### 📜 Legislação Municipal — {m.nome} ({m.uf})\n\n"
            f"- **Fonte:** {res.fonte} (`{res.plataforma}`)\n"
            f"- **Status:** {status_msg}\n"
        )

    headers = ["Número", "Ano", "Tipo de Documento", "Publicação", "Ementa / Descrição"]
    rows = []
    for a in res.atos[:limite]:
        rows.append(
            [
                a.numero or "N/D",
                str(a.ano) if a.ano else "N/D",
                _trunc(a.tipo_documento or "N/D", 22),
                a.data_publicacao or "N/D",
                _trunc(a.descricao, 48),
            ]
        )

    header_md = (
        f"### 📜 Legislação e Atos Normativos Municipais — {m.nome} ({m.uf})\n\n"
        f"- **Fonte:** {res.fonte} (`{res.plataforma}`)\n"
        f"- **Total de Atos:** {res.total_registros:,} (exibindo {len(rows)})\n\n"
    )
    return header_md + markdown_table(headers, rows)


async def consultar_concursos_municipio(
    municipio: str,
    uf: str | None = None,
    ano: int = 0,
    limite: int = 20,
    url: str | None = None,
) -> str:
    """Consulta aprovados e convocados em concursos públicos e processos seletivos do município.

    Args:
        municipio: Nome do município ou código IBGE.
        uf: Sigla da UF. Opcional.
        ano: Ano do edital ou homologação (opcional).
        limite: Quantidade máxima de registros exibidos (padrão: 20).
        url: URL base explícita do portal municipal (opcional).
    """
    try:
        res = await client.consultar_concursos_municipio(
            municipio=municipio,
            uf=uf,
            ano=ano,
            limit=limite,
            url=url,
        )
    except Exception as err:
        return f"Erro ao consultar concursos de '{municipio}': {err}"

    m = res.municipio
    if not res.disponivel or not res.aprovados:
        status_msg = res.mensagem or "Nenhum aprovado ou processo seletivo registrado."
        return (
            f"### 🎓 Concursos e Processos Seletivos — {m.nome} ({m.uf})\n\n"
            f"- **Fonte:** {res.fonte} (`{res.plataforma}`)\n"
            f"- **Status:** {status_msg}\n"
        )

    headers = ["Candidato(a)", "Cargo", "Classificação", "Situação", "Edital", "Publicação"]
    rows = []
    for ap in res.aprovados[:limite]:
        rows.append(
            [
                _trunc(ap.nome, 32),
                _trunc(ap.cargo or "N/D", 24),
                str(ap.classificacao) if ap.classificacao else "N/D",
                ap.situacao or "N/D",
                ap.edital or "N/D",
                ap.data_publicacao or "N/D",
            ]
        )

    header_md = (
        f"### 🎓 Concursos Públicos e Processos Seletivos — {m.nome} ({m.uf})\n\n"
        f"- **Fonte:** {res.fonte} (`{res.plataforma}`)\n"
        f"- **Total de Convocados/Aprovados:** {res.total_registros:,} (exibindo {len(rows)})\n\n"
    )
    return header_md + markdown_table(headers, rows)


async def consultar_detalhe_servidor_municipio(
    municipio: str,
    matricula: str,
    uf: str | None = None,
    ano: int = 0,
    mes: int = 0,
    url: str | None = None,
) -> str:
    """Consulta contracheque detalhado de um servidor público municipal por matrícula.

    Exibe rubrica a rubrica (vencimento básico, adicionais, quinquênios, descontos,
    previdência e IRRF), com total conformidade LGPD (CPF com mascaramento central).

    Args:
        municipio: Nome do município ou código IBGE.
        matricula: Número da matrícula funcional do servidor.
        uf: Sigla da UF. Opcional.
        ano: Ano de referência da folha (opcional).
        mes: Mês de referência (1-12) (opcional).
        url: URL base explícita do portal municipal (opcional).
    """
    try:
        det = await client.consultar_detalhe_servidor_municipio(
            municipio=municipio,
            matricula=matricula,
            uf=uf,
            ano=ano,
            mes=mes,
            url=url,
        )
    except Exception as err:
        return f"Erro ao consultar contracheque detalhado do servidor '{matricula}': {err}"

    m = det.municipio
    sections: list[str] = []
    sections.append(
        f"### 👤 Contracheque Detalhado — {det.nome} ({m.nome} - {m.uf})\n\n"
        f"- **Cargo / Função:** {det.cargo or 'N/D'}\n"
        f"- **Matrícula:** `{det.matricula}`\n"
        f"- **CPF (LGPD):** `{det.cpf_parcial or '***.XXX.XXX-**'}`\n"
        f"- **Órgão / Lotação:** {det.orgao or 'N/D'} | {det.departamento or 'N/D'}\n"
        f"- **Vínculo / Situação:** {det.vinculo or 'N/D'} ({det.situacao or 'N/D'})\n"
        f"- **Competência:** {det.competencia or 'Atual'}\n"
        f"- **Carga Horária:** {det.carga_horaria or 'N/D'}\n"
        f"- **Data de Admissão:** {det.data_admissao or 'N/D'}\n"
        f"- **Fonte:** {det.fonte}\n\n"
        f"#### 💵 Totais Financeiros\n"
        f"- **Remuneração Bruta:** `R$ {det.salario_bruto:,.2f}`\n"
        f"- **Total de Descontos:** `R$ {det.descontos:,.2f}`\n"
        f"- **Remuneração Líquida:** `R$ {det.salario_liquido:,.2f}`\n"
    )

    if det.vencimentos:
        rows_venc = [[_trunc(v.descricao, 48), f"R$ {v.valor:,.2f}"] for v in det.vencimentos]
        sections.append(
            "#### Proventos e Vencimentos (+)\n\n"
            + markdown_table(["Rubrica / Provento", "Valor"], rows_venc)
        )

    if det.verbas_desconto:
        rows_desc = [[_trunc(d.descricao, 48), f"R$ {d.valor:,.2f}"] for d in det.verbas_desconto]
        sections.append(
            "#### Descontos e Retenções (-)\n\n"
            + markdown_table(["Rubrica / Desconto", "Valor"], rows_desc)
        )

    if det.ferias:
        rows_fer = [[_trunc(f.descricao, 48), f"R$ {f.valor:,.2f}"] for f in det.ferias]
        sections.append(
            "#### 🌴 Férias\n\n" + markdown_table(["Rubrica de Férias", "Valor"], rows_fer)
        )

    if det.decimo_terceiro:
        rows_13 = [[_trunc(d.descricao, 48), f"R$ {d.valor:,.2f}"] for d in det.decimo_terceiro]
        sections.append(
            "#### 🎁 13º Salário\n\n" + markdown_table(["Rubrica de 13º", "Valor"], rows_13)
        )

    return "\n\n".join(sections)
