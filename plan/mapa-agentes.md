# 🇧🇷 mcp-brasil — Mapa de Agentes Escaláveis

## O Padrão: Prompt + Resource + Tool + Dados

Cada agente segue a mesma fórmula do Redator Oficial:

```
PROMPT (papel)     +  RESOURCE (contexto)    +  TOOL (ação)         +  DADOS (features)
@mcp.prompt           @mcp.resource             @mcp.tool              /ibge, /bacen...
"Sou o redator de..." "template://despacho"     formatar_data()        consultar_pib()
instrui o LLM         carrega normas/templates   executa funções        dados em tempo real
```

O **auto-registry** (ADR-002) descobre cada agente automaticamente — basta criar o diretório com `FEATURE_META` e `server.py`.

---

## Tier 1 — Administrativo 📋

### 1.1 Redator Oficial ✅ (ADR-003)
> Despacho, memorando, ofício, portaria, parecer, nota técnica, ata

| Componente | Detalhes |
|-----------|---------|
| **Prompts** | `redator_despacho`, `redator_memorando`, `redator_oficio`, `redator_portaria`, `redator_parecer`, `redator_nota_tecnica` |
| **Tools** | `formatar_data_extenso()`, `gerar_numeracao()`, `consultar_pronome_tratamento()`, `validar_documento()` |
| **Resources** | `template://despacho`, `template://memorando`, `template://oficio`, `normas://manual_redacao`, `normas://pronomes` |
| **Dados** | Qualquer feature disponível (IBGE para nota técnica, Bacen para dados econômicos) |
| **Esforço** | 2-3 semanas |

**Por que começa aqui:** Todo servidor público precisa redigir documentos. É o use case mais universal e de adoção imediata.

---

### 1.2 Analista de Licitações
> Análise de editais, composição de preços, pareceres, impugnações

| Componente | Detalhes |
|-----------|---------|
| **Prompts** | `analisar_edital` — analisa um edital e lista pontos críticos |
| | `compor_precos` — monta planilha de composição de custos |
| | `redigir_impugnacao` — redige impugnação fundamentada |
| | `parecer_licitacao` — parecer técnico sobre proposta |
| **Tools** | `extrair_itens_edital()` — extrai itens de um edital em formato estruturado |
| | `calcular_bdi()` — calcula BDI (Bonificação e Despesas Indiretas) |
| | `verificar_certidoes()` — verifica status de certidões |
| | `comparar_precos_ata()` — compara preços com atas de registro |
| **Resources** | `template://parecer_licitacao`, `normas://lei_14133` (Nova Lei de Licitações) |
| **Dados** | `/transparencia` — contratos e atas de registro de preços |
| **Esforço** | 3-4 semanas |

**O poder da composição:** O agente analisa um edital e automaticamente puxa preços médios de contratos similares do Portal da Transparência para validar se os valores estão dentro do mercado.

---

### 1.3 Gestor de Contratos
> Termos aditivos, notificações, acompanhamento de vigência, apostilamentos

| Componente | Detalhes |
|-----------|---------|
| **Prompts** | `redigir_aditivo`, `notificar_contratado`, `relatorio_execucao`, `apostilamento` |
| **Tools** | `calcular_reajuste_igpm()` — calcula reajuste por IGP-M/IPCA |
| | `verificar_vigencia()` — verifica se contrato está na vigência |
| | `gerar_cronograma_fiscal()` — gera cronograma de entregas |
| **Resources** | `template://termo_aditivo`, `template://notificacao`, `normas://lei_14133_gestao` |
| **Dados** | `/bacen` — índices IGP-M, IPCA para reajuste; `/transparencia` — contratos vigentes |
| **Esforço** | 3-4 semanas |

**Cenário real:** "Gere um termo aditivo de reajuste do contrato X aplicando o IPCA acumulado dos últimos 12 meses." → O agente consulta o IPCA no Bacen, calcula o reajuste e gera o documento.

---

## Tier 2 — Fiscal e Financeiro 💰

### 2.1 Analista Fiscal
> Consulta CNPJ, situação cadastral, certidões, validação de NF-e

| Componente | Detalhes |
|-----------|---------|
| **Prompts** | `verificar_regularidade` — verifica regularidade fiscal de empresa |
| | `analisar_nfe` — analisa notas fiscais eletrônicas |
| | `relatorio_fiscal` — gera relatório de situação fiscal |
| **Tools** | `consultar_cnpj()`, `validar_nfe()`, `verificar_cnd()`, `consultar_simples_nacional()` |
| **Resources** | `normas://obrigacoes_acessorias`, `template://relatorio_fiscal` |
| **Dados** | `/receita` — CNPJ, situação cadastral (Tier 3 do roadmap) |
| **Esforço** | 3-4 semanas |

---

### 2.2 Analista Econômico
> Relatórios macroeconômicos, projeções, análise de indicadores

| Componente | Detalhes |
|-----------|---------|
| **Prompts** | `relatorio_conjuntura` — relatório de conjuntura econômica atual |
| | `analise_inflacao` — análise da evolução inflacionária |
| | `projecao_pib` — projeção de PIB com fundamentação |
| | `comparativo_regional` — compara indicadores entre UFs |
| **Tools** | `gerar_grafico_serie()`, `calcular_variacao()`, `projetar_tendencia()` |
| **Resources** | `template://relatorio_conjuntura`, `template://nota_tecnica_economica` |
| **Dados** | `/bacen` — Selic, IPCA, câmbio, expectativas; `/ibge` — PIB, emprego, IPCA |
| **Esforço** | 2-3 semanas |

**O agente mais poderoso:** Usa TODAS as features de dados para gerar relatórios econômicos com números reais. Nenhum outro MCP server no mundo faz isso para a economia brasileira.

---

### 2.3 Auditor de Despesas
> Análise de gastos públicos, detecção de anomalias, relatórios de auditoria

| Componente | Detalhes |
|-----------|---------|
| **Prompts** | `auditar_despesas` — audita despesas de um órgão/período |
| | `detectar_anomalias` — identifica gastos fora do padrão |
| | `relatorio_auditoria` — gera relatório formal de auditoria |
| | `parecer_auditoria` — parecer técnico de auditoria |
| **Tools** | `classificar_despesa()`, `comparar_media_historica()`, `identificar_fracionamento()` |
| **Resources** | `template://relatorio_auditoria`, `normas://tcu_jurisprudencia` |
| **Dados** | `/transparencia` — despesas, contratos, servidores, CEIS/CNEP |
| **Esforço** | 4-5 semanas |

**Cenário:** "Analise as despesas do Ministério X nos últimos 6 meses e identifique possíveis fracionamentos." → Consulta Transparência, classifica despesas, detecta padrões suspeitos.

---

## Tier 3 — Legislativo e Jurídico ⚖️

### 3.1 Monitor Legislativo
> Acompanhamento de PLs, resumos de votações, alertas

| Componente | Detalhes |
|-----------|---------|
| **Prompts** | `resumir_pl` — resume projeto de lei em linguagem acessível |
| | `analisar_votacao` — analisa resultado de votação com partidos |
| | `alerta_tramitacao` — lista mudanças recentes em tramitação |
| | `comparar_versoes` — compara versões de um PL (original vs. substitutivo) |
| **Tools** | `buscar_proposicoes()`, `resumir_ementa()`, `listar_autores()`, `timeline_tramitacao()` |
| **Resources** | `template://resumo_legislativo`, `template://nota_tecnica_pl` |
| **Dados** | `/camara` — proposições, votações, despesas; `/senado` — matérias, comissões |
| **Esforço** | 2-3 semanas |

**Cenário:** "Quais PLs sobre IA foram apresentados na Câmara em 2026? Resuma cada um." → Consulta API da Câmara, filtra por tema, gera resumos formatados.

---

### 3.2 Analista Jurídico
> Pesquisa de jurisprudência, análise de legislação, pareceres jurídicos

| Componente | Detalhes |
|-----------|---------|
| **Prompts** | `pesquisar_jurisprudencia`, `analisar_legislacao`, `parecer_juridico`, `minuta_contrato` |
| **Tools** | `buscar_processos()`, `extrair_decisao()`, `comparar_legislacao()`, `verificar_vigencia_lei()` |
| **Resources** | `template://parecer_juridico`, `normas://constituicao_resumo`, `normas://cpc_resumo` |
| **Dados** | `/datajud` — processos judiciais CNJ; `/diario_oficial` — publicações DOU |
| **Esforço** | 4-6 semanas |

---

## Tier 4 — Dados e Pesquisa 📊

### 4.1 Pesquisador IBGE
> Análises demográficas, perfis municipais, comparativos regionais

| Componente | Detalhes |
|-----------|---------|
| **Prompts** | `perfil_municipio` — gera perfil completo de um município |
| | `analise_demografica` — analisa dados demográficos de região |
| | `comparativo_ufs` — compara indicadores entre estados |
| | `serie_indicador` — evolução temporal de um indicador |
| **Tools** | `cruzar_indicadores()`, `rankear_municipios()`, `gerar_mapa_tematico()` |
| **Resources** | `template://relatorio_pesquisa`, `template://perfil_municipal` |
| **Dados** | `/ibge` — todas as 8 tools (população, PIB, censo, malhas, nomes, agregados, CNAE, inflação) |
| **Esforço** | 2-3 semanas |

**Cenário:** "Gere o perfil completo de Teresina com dados demográficos, econômicos e comparação com capitais do Nordeste." → Puxa população, PIB, índices do IBGE e monta relatório.

---

### 4.2 Jornalista de Dados
> Investigação de dados públicos, cruzamentos, geração de pautas

| Componente | Detalhes |
|-----------|---------|
| **Prompts** | `investigar_gastos` — investiga padrões de gastos de um órgão |
| | `cruzar_dados` — cruza dados de múltiplas fontes |
| | `gerar_pauta` — sugere pautas investigativas baseadas em dados |
| | `factcheck_declaracao` — verifica declaração de político com dados reais |
| **Tools** | `cruzar_servidores_contratos()`, `ranking_despesas()`, `timeline_eventos()`, `exportar_dados()` |
| **Resources** | `template://pauta_investigativa`, `template://factcheck` |
| **Dados** | **TODOS** — Transparência + Câmara + Senado + IBGE + Bacen |
| **Esforço** | 3-4 semanas |

**O agente-chave para viralização:** Jornalistas usando IA para investigar dados públicos é uma narrativa muito forte. Gera mídia espontânea.

---

## Tier 5 — Segurança Pública 🛡️

### 5.1 Analista OSINT
> Investigação em fontes abertas, correlação de dados, relatórios de inteligência

| Componente | Detalhes |
|-----------|---------|
| **Prompts** | `investigar_entidade`, `correlacionar_dados`, `relatorio_inteligencia`, `perfil_osint` |
| **Tools** | `buscar_fontes_abertas()`, `correlacionar_cpf_cnpj()`, `gerar_grafo_relacoes()` |
| **Resources** | `template://relatorio_osint`, `normas://lgpd_limites` |
| **Dados** | `/transparencia`, `/receita`, `/datajud`, `/diario_oficial` |
| **Esforço** | 4-6 semanas |

### 5.2 Analista de Ocorrências
> Análise de boletins, padrões criminais, mapas de calor

| Componente | Detalhes |
|-----------|---------|
| **Prompts** | `analisar_bo`, `mapa_ocorrencias`, `relatorio_operacional`, `tendencia_criminal` |
| **Tools** | `classificar_ocorrencia()`, `gerar_mapa_calor()`, `detectar_padrao()` |
| **Resources** | `template://relatorio_operacional`, `template://analise_criminal` |
| **Dados** | Dados de SSP estaduais (feature futura) |
| **Esforço** | 5-8 semanas |

---

## Tier 6 — Educação e Social 🎓

### 6.1 Analista Educacional
> Dados do ENEM, IDEB, censo escolar

| Componente | Detalhes |
|-----------|---------|
| **Prompts** | `perfil_escola`, `analise_enem`, `comparar_ideb`, `diagnostico_educacional` |
| **Tools** | `buscar_escola()`, `rankear_municipios_ideb()`, `serie_enem()` |
| **Dados** | `/inep` — ENEM, IDEB, censo escolar |
| **Esforço** | 3-4 semanas |

### 6.2 Analista Social
> Programas sociais, Bolsa Família, BPC, indicadores de vulnerabilidade

| Componente | Detalhes |
|-----------|---------|
| **Prompts** | `diagnostico_social`, `mapa_vulnerabilidade`, `relatorio_programa_social` |
| **Tools** | `consultar_beneficiarios()`, `indicador_vulnerabilidade()`, `comparar_cobertura()` |
| **Dados** | `/transparencia` — Bolsa Família, BPC; `/ibge` — censo, indicadores sociais |
| **Esforço** | 3-4 semanas |

---

## Roadmap de Implementação

```
SEMANA   AGENTE                      DEPENDE DE
──────────────────────────────────────────────────
  1-3    Redator Oficial              core/ + templates
  3-5    Analista Econômico           /ibge + /bacen
  5-7    Monitor Legislativo          /camara + /senado
  7-9    Analista de Licitações       /transparencia
  9-11   Pesquisador IBGE             /ibge (completo)
 11-13   Gestor de Contratos          /bacen + /transparencia
 13-15   Analista Fiscal              /receita
 15-18   Auditor de Despesas          /transparencia (avançado)
 18-22   Jornalista de Dados          todos os dados
 22-26   Analista Jurídico            /datajud + /diario_oficial
 26-30   OSINT + Ocorrências          dados especializados
 30-34   Educação + Social            /inep + /transparencia
```

---

## Como contribuidores adicionam novos agentes

```bash
# 1. Criar diretório com a convenção
mkdir -p src/mcp_brasil/novo_agente/{templates,normas}

# 2. __init__.py com FEATURE_META
cat > src/mcp_brasil/novo_agente/__init__.py << 'EOF'
from mcp_brasil._shared.feature import FeatureMeta
FEATURE_META = FeatureMeta(
    name="novo_agente",
    description="Descrição do que o agente faz",
    tags=["dominio", "area"],
)
EOF

# 3. Criar prompts.py, tools.py, resources.py, server.py
# seguindo o padrão do Redator (ADR-003)

# 4. Rodar → auto-descoberto!
fastmcp run mcp_brasil.server:mcp
```

**Zero arquivos existentes modificados.** Cada agente é uma feature independente.

---

## A visão: Plataforma de Agentes GovTech

```
mcp-brasil v1.0 (hoje)
  └── Features de DADOS: IBGE, Bacen, Transparência, Câmara, Senado

mcp-brasil v2.0 (com agentes)
  ├── Features de DADOS: 10+ APIs governamentais
  ├── Features de AGENTES: 12+ agentes especializados
  └── Composição: Agentes usam dados em tempo real

mcp-brasil v3.0 (plataforma)
  ├── Marketplace de agentes (contribuidores publicam)
  ├── Templates por órgão (cada estado/prefeitura customiza)
  ├── Workflows multi-agente (Licitação → Contrato → Fiscalização)
  └── Integração SEI/e-Docs (protocolo digital)
```

O `mcp-brasil` deixa de ser "coleção de MCP servers" e se torna a **plataforma de agentes AI para o setor público brasileiro**.

---

*Mapa de Agentes v1 — 2026-03-22*
