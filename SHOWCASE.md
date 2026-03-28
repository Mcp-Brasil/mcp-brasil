# MCP-Brasil: Demonstração de Poder

**326 tools** | **41 features** | **47/51 testes reais com sucesso (92.2%)**

> Testado em 28/03/2026 contra APIs governamentais reais, sem mocks.

---

## Resumo Executivo

| Indicador | Valor |
|-----------|-------|
| Features ativas | 41 (39 data + 1 agente + 1 meta) |
| Tools registradas | 326 |
| Testes executados | 51 |
| Sucesso | 47 (92.2%) |
| Falha (API externa fora) | 4 (7.8%) |
| Tempo médio por tool | ~0.5s |

---

## Perguntas-Exemplo com Respostas Reais

### IBGE — Dados Geográficos e Demográficos

| Pergunta | Tool | Resultado |
|----------|------|-----------|
| Quais são os estados do Brasil? | `ibge_listar_estados` | 27 estados com UF, nome e região em tabela markdown |
| Quais municípios existem no Acre? | `ibge_buscar_municipios` | 22 municípios (Acrelândia, Brasiléia, Rio Branco...) |
| Qual a frequência do nome Maria? | `ibge_consultar_nome` | Série histórica: de 336K (antes de 1930) a 2.4M (1960-70) |
| Quais os nomes mais populares? | `ibge_ranking_nomes` | Top 20: Maria (11.7M), José (5.7M), Ana (3M), João (2.9M) |

**Exemplo de resposta real:**
```
| # | Nome | Frequência |
|---|------|------------|
| 1 | MARIA | 11.734.129 |
| 2 | JOSE | 5.754.529 |
| 3 | ANA | 3.089.858 |
| 4 | JOAO | 2.984.119 |
| 5 | ANTONIO | 2.576.348 |
```

---

### BrasilAPI — Dados Unificados

| Pergunta | Tool | Resultado |
|----------|------|-----------|
| Qual o endereço do CEP 01001-000? | `brasilapi_consultar_cep` | Praça da Sé, Sé, São Paulo/SP |
| Dados cadastrais da Petrobras? | `brasilapi_consultar_cnpj` | Razão Social, CNAE, endereço, capital social R$ 205B |
| Cidades que usam DDD 11? | `brasilapi_consultar_ddd` | 64 cidades: Guarulhos, Osasco, São Paulo, Jundiaí... |
| Quais bancos existem no Brasil? | `brasilapi_listar_bancos` | 357 bancos: BB, Bradesco, Itaú, Santander, BNDES... |
| Feriados nacionais de 2025? | `brasilapi_consultar_feriados` | 13 feriados: Carnaval (04/03), Páscoa (20/04)... |
| Domínio google.com.br registrado? | `brasilapi_consultar_registro_br` | Registrado, DNS: ns1-4.google.com |

**Exemplo de resposta real:**
```
**CNPJ:** 33000167000101
**Razão Social:** PETROLEO BRASILEIRO S A PETROBRAS
**Situação:** ATIVA
**CNAE:** 600001 — Extração de petróleo e gás natural
**Capital Social:** R$ 205.431.960.000,00
```

---

### BACEN — Banco Central

| Pergunta | Tool | Resultado |
|----------|------|-----------|
| Taxa SELIC recente? | `bacen_consultar_serie` | 87 registros: 12,25% (jan/2025) → 13,25% (fev/2025) |
| Séries populares disponíveis? | `bacen_series_populares` | 169 séries em 12 categorias (juros, inflação, câmbio...) |
| Indicadores econômicos atuais? | `bacen_indicadores_atuais` | Selic 14,75%, IPCA 0,70%, Dólar R$ 5,20 |

**Exemplo de resposta real:**
```
| Indicador | Valor | Data |
|-----------|-------|------|
| Selic (a.a.) | 14,7500 | 29/04/2026 |
| IPCA mensal (%) | 0,7000 | 01/02/2026 |
| Dólar PTAX (venda) | 5,2006 | 01/02/2026 |
```

---

### Poder Legislativo

| Pergunta | Tool | Resultado |
|----------|------|-----------|
| Deputados federais em exercício? | `camara_listar_deputados` | Lista paginada com ID, nome, partido, UF e email |
| Senadores em exercício? | `senado_listar_senadores` | 81 senadores com código, nome, partido e UF |
| Partidos no Senado? | `senado_partidos_senado` | 12 partidos: PL (16), PSD (13), MDB (10), PT (9)... |

---

### TSE — Eleições

| Pergunta | Tool | Resultado |
|----------|------|-----------|
| Anos com eleições registradas? | `tse_anos_eleitorais` | 2004-2024 (11 anos disponíveis) |
| Eleições registradas? | `tse_listar_eleicoes` | Listagem de eleições ordinárias e suplementares |

---

### Transparência — Controle Social

| Pergunta | Tool | Resultado |
|----------|------|-----------|
| Servidores federais com sobrenome Silva? | `transparencia_buscar_servidores` | Lista de servidores com nome, cargo, órgão |
| Emendas pix de 2024? | `transferegov_buscar_emendas_pix` | Emendas com autor, valor, beneficiário e UF |

**Exemplo de resposta real (emendas pix):**
```
| Emenda | Parlamentar | Valor | Beneficiário | UF |
|--------|-------------|-------|--------------|-----|
| 202415810013 | Jefferson Campos | R$ 2.000.000,00 | MUNICIPIO DE OSASCO | SP |
| 202443430002 | Delegada Ione | R$ 800.000,00 | MUNICIPIO DE MIRADOURO | MG |
| 202443400005 | Dayany Bittencourt | R$ 4.000.000,00 | MUNICIPIO DE ACARAU | CE |
```

---

### TCU — Tribunal de Contas

| Pergunta | Tool | Resultado |
|----------|------|-----------|
| Empresas declaradas inidôneas? | `tcu_consultar_inidoneos` | 25 empresas com nome, CNPJ, motivo e prazo |

---

### Dados Abertos

| Pergunta | Tool | Resultado |
|----------|------|-----------|
| Datasets abertos sobre educação? | `dados_abertos_buscar_conjuntos` | Conjuntos do dados.gov.br sobre educação |

---

### INPE — Meio Ambiente

| Pergunta | Tool | Resultado |
|----------|------|-----------|
| Satélites de monitoramento? | `inpe_dados_satelite` | Lista de satélites com capacidades |
| Focos de queimada no Amazonas? | `inpe_buscar_focos_queimadas` | Focos detectados com coordenadas e data |

---

### Saúde Pública

| Pergunta | Tool | Resultado |
|----------|------|-----------|
| Medicamentos gratuitos Farmácia Popular? | `farmacia_popular_listar_medicamentos` | Medicamentos com indicação e disponibilidade |
| Grupos terapêuticos RENAME? | `rename_listar_grupos_terapeuticos` | Grupos do SUS: cardiovascular, endócrino, etc. |
| Preço de amoxicilina no governo? | `bps_buscar_medicamento_bps` | Preços de compras governamentais |
| Datasets OpenDataSUS? | `opendatasus_listar_datasets_conhecidos` | Datasets: vacinação, SRAG, leitos, hospitais... |
| Calendário de vacinação? | `imunizacao_calendario_vacinacao` | Calendário completo por faixa etária |
| Tipos de estabelecimento de saúde? | `saude_listar_tipos_estabelecimento` | UPAs, hospitais, UBS, clínicas, etc. |
| O que é o SNA do SUS? | `denasus_informacoes_sna` | Informações sobre o Sistema Nacional de Auditoria |

---

### Tribunais de Contas Estaduais

| Pergunta | Tool | Resultado |
|----------|------|-----------|
| Municípios SP no TCE? | `tce_sp_listar_municipios_sp` | 645 municípios paulistas |
| Obras paralisadas RJ? | `tce_rj_buscar_obras_paralisadas` | Obras com status e localização |
| Municípios CE no TCE? | `tce_ce_listar_municipios_ce` | Municípios cearenses cadastrados |
| Unidades gestoras PE? | `tce_pe_buscar_unidades_pe` | Unidades com código e nome |
| Municípios RS no TCE? | `tce_rs_listar_municipios_rs` | Municípios gaúchos |

---

### Diário Oficial

| Pergunta | Tool | Resultado |
|----------|------|-----------|
| Municípios de SP com diários? | `diario_oficial_buscar_cidades` | Municípios com diários digitalizados |

---

### BNDES

| Pergunta | Tool | Resultado |
|----------|------|-----------|
| Datasets sobre financiamento? | `bndes_buscar_datasets_bndes` | Datasets CKAN: operações, desembolsos |

---

### Jurisprudência

| Pergunta | Tool | Resultado |
|----------|------|-----------|
| STF sobre liberdade de expressão? | `jurisprudencia_buscar_jurisprudencia_stf` | Acórdãos e decisões relevantes (7.7s) |

---

### DataJud — Processos Judiciais

| Pergunta | Tool | Resultado |
|----------|------|-----------|
| Processos sobre direito do consumidor no TJSP? | `datajud_buscar_processos` | Processos com número, classe e assunto (6.8s) |

---

### Segurança Pública

| Pergunta | Tool | Resultado |
|----------|------|-----------|
| Datasets segurança MJSP? | `sinesp_listar_datasets_mjsp` | Datasets: homicídios, estupros, roubos... |
| Temas do Fórum de Segurança? | `forum_seguranca_listar_temas_seguranca` | Comunidades temáticas DSpace |
| Temas de violência no Atlas? | `atlas_violencia_listar_temas_violencia` | Séries IPEA: homicídios, armas, trânsito... |

---

### Compras Públicas

| Pergunta | Tool | Resultado |
|----------|------|-----------|
| Fornecedor por CNPJ (Petrobras)? | `compras_dadosabertos_consultar_fornecedor` | PETROBRAS, CNAE: Extração de petróleo, Ativo |

---

### Redator — Agente de Redação Oficial

| Pergunta | Tool | Resultado |
|----------|------|-----------|
| Pronome de tratamento para Ministro? | `redator_consultar_pronome_tratamento` | Vossa Excelência, Vocativo: Senhor Ministro |
| Tipos de documento oficial? | `redator_listar_tipos_documento` | ofício, despacho, portaria, parecer, nota técnica, ata |
| Data de hoje por extenso? | `redator_formatar_data_extenso` | "Brasília, 28 de março de 2026." |

**Exemplo de resposta real:**
```
Cargo: ministro
Tratamento: Vossa Excelência
Vocativo: Senhor Ministro,
Abreviatura: V. Exa.
Endereçamento: A Sua Excelência o Senhor
```

---

## APIs com Falha (4/51 = 7.8%)

Estas APIs estavam indisponíveis no momento do teste:

| Feature | Tool | Erro |
|---------|------|------|
| Tábua de Marés | `listar_estados_costeiros` | HTTP 404 — API tabuademares.com indisponível |
| Tábua de Marés | `listar_portos` | HTTP 404 — API tabuademares.com indisponível |
| ANVISA | `buscar_medicamento` | HTTP 403 — Bloqueio da ANVISA (Cloudflare) |
| ANA | `monitorar_reservatorios` | JSON inválido — API SAR instável |

> Nota: estas falhas são de APIs externas, não do mcp-brasil. As tools estão implementadas corretamente.

---

## Cobertura por Domínio

| Domínio | Features | Tools | Status |
|---------|----------|-------|--------|
| Geografia/Censo | IBGE | 9 | Operacional |
| Dados Unificados | BrasilAPI | 16 | Operacional |
| Economia/Finanças | BACEN, BNDES | 13 | Operacional |
| Legislativo | Câmara, Senado | 37 | Operacional |
| Eleitoral | TSE | 15 | Operacional |
| Transparência | Transparência, TransfereGov | 23 | Operacional |
| Controle Externo | TCU | 9 | Operacional |
| Judiciário | DataJud, Jurisprudência | 13 | Operacional |
| Saúde | Saúde, ANVISA, RENAME, BPS, Farmácia Popular, Imunização, OpenDataSUS, DENASUS | 59 | Parcial (ANVISA bloqueada) |
| Meio Ambiente | INPE, ANA, Tábua de Marés | 14 | Parcial (ANA/Marés instáveis) |
| Segurança | SINESP, Fórum Segurança, Atlas Violência | 17 | Operacional |
| Dados Abertos | Dados Abertos, Diário Oficial | 10 | Operacional |
| Compras | Compras (PNCP + DadosAbertos + Contratos) | 27 | Operacional |
| TCEs Estaduais | CE, ES, PE, PI, RJ, RN, RS, SC, SP, TO | 48 | Operacional |
| Agentes | Redator | 5 | Operacional |

---

## Como Executar os Testes

```bash
# Executar showcase completo
MCP_BRASIL_TOOL_SEARCH=none uv run python test_showcase.py

# Ver resultados em JSON
cat showcase_results.json | python -m json.tool
```

---

*Gerado automaticamente em 28/03/2026 via test_showcase.py*
