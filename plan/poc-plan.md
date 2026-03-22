# 🇧🇷 mcp-brasil — Plano da POC: Agregação de Todos os MCP Servers Brasileiros

**Data:** 2026-03-21
**Versão:** 1.0
**Baseado em:** Análise de 16 repos clonados + Roadmap v3 + ADR-001 + ADR-002

---

## 1. Inventário Completo — O Que Já Existe

### 1.1 Todos os Tools Mapeados por Domínio (92 tools únicos)

#### IBGE (4 repos, ~8 tools únicos)
| Tool Original | Repo Origem | API Real | → Feature mcp-brasil |
|--------------|-------------|----------|---------------------|
| `ibge_localidades` | mcp-dadosbr | servicodados.ibge.gov.br | `ibge/` |
| `consultar_ibge` (municipios/estados) | brasil-api-mcp | brasilapi.com.br/api/ibge | `ibge/` |
| Localidades, estados, municipios | mcp-brasil-api | brasilapi.com.br/api/ibge | `ibge/` |
| **Novos:** populacao, pib, nomes, malha, agregados, cnae, inflacao | — | servicodados.ibge.gov.br | `ibge/` |

#### Banco Central (1 repo dedicado, ~8 tools)
| Tool Original | Repo Origem | API Real | → Feature mcp-brasil |
|--------------|-------------|----------|---------------------|
| `bcb_serie_valores` | bcb-br-mcp | api.bcb.gov.br/dados/serie | `bacen/` |
| `bcb_serie_ultimos` | bcb-br-mcp | api.bcb.gov.br/dados/serie | `bacen/` |
| `bcb_serie_metadados` | bcb-br-mcp | api.bcb.gov.br/dados/serie | `bacen/` |
| `bcb_series_populares` | bcb-br-mcp | catálogo local 150+ séries | `bacen/` |
| `bcb_buscar_serie` | bcb-br-mcp | api.bcb.gov.br | `bacen/` |
| `bcb_indicadores_atuais` | bcb-br-mcp | api.bcb.gov.br | `bacen/` |
| `bcb_variacao` | bcb-br-mcp | api.bcb.gov.br | `bacen/` |
| `bcb_comparar` | bcb-br-mcp | api.bcb.gov.br | `bacen/` |
| `bacen_taxas` | mcp-dadosbr | brasilapi.com.br/api/taxas | `bacen/` |
| **Novos:** expectativas_focus, estatisticas_pix, ptax | — | olinda.bcb.gov.br | `bacen/` |

#### Portal da Transparência (2 repos, ~4 tools)
| Tool Original | Repo Origem | API Real | → Feature mcp-brasil |
|--------------|-------------|----------|---------------------|
| `transparencia_lookup` | mcp-dadosbr | portaldatransparencia.gov.br | `transparencia/` |
| `ceis_cnep_lookup` | mcp-dadosbr | portaldatransparencia.gov.br | `transparencia/` |
| **Novos:** contratos, despesas, servidores, licitacoes, bolsa_familia | — | portaldatransparencia.gov.br | `transparencia/` |

#### Senado Federal (1 repo dedicado, 33 tools)
| Grupo | Tools | → Feature mcp-brasil |
|-------|-------|---------------------|
| Senadores | listar, obter_detalhes, buscar_por_nome, votacoes | `senado/` |
| Matérias | buscar, obter_detalhes, tramitacao, textos, emendas | `senado/` |
| Votações | listar, detalhes, votos | `senado/` |
| Comissões | listar, detalhes, membros, reunioes | `senado/` |
| Agenda | sessoes_plenarias, pautas | `senado/` |
| e-Cidadania | ideias, consultas, sugestoes, eventos | `senado/` |
| Auxiliares | legislaturas, partidos, blocos, tipos_materia | `senado/` |

#### Câmara dos Deputados (NOVO — nenhum repo dedicado)
| Tools Planejados | API | → Feature mcp-brasil |
|-----------------|-----|---------------------|
| buscar_deputado, listar_deputados | dadosabertos.camara.leg.br/api/v2 | `camara/` |
| buscar_proposicao, tramitacao | dadosabertos.camara.leg.br/api/v2 | `camara/` |
| buscar_votacao, votos_nominais | dadosabertos.camara.leg.br/api/v2 | `camara/` |
| despesas_parlamentares | dadosabertos.camara.leg.br/api/v2 | `camara/` |
| agenda_legislativa, eventos | dadosabertos.camara.leg.br/api/v2 | `camara/` |

#### Judiciário / Legal (3 repos, ~8 tools)
| Tool Original | Repo Origem | API Real | → Feature mcp-brasil |
|--------------|-------------|----------|---------------------|
| `datajud_processos` | mcp-dadosbr | api-publica.datajud.cnj.jus.br | `datajud/` |
| `oab_advogado` | mcp-dadosbr | scraping OAB | `datajud/` |
| `bnmp_mandados` | mcp-dadosbr | portalbnmp.cnj.jus.br | `datajud/` |
| `procurados_lookup` | mcp-dadosbr | policiafederal.gov.br | `datajud/` |
| `lista_suja_lookup` | mcp-dadosbr | reporterbrasil.org.br | `datajud/` |
| STJ precedentes (busca, paginação) | brlaw_mcp_server | stj.jus.br | `jurisprudencia/` |
| TST precedentes | brlaw_mcp_server | tst.jus.br | `jurisprudencia/` |
| STF precedentes | brlaw_mcp_server | stf.jus.br (scraping) | `jurisprudencia/` |

#### BrasilAPI / Utilidades (4 repos, ~16 tools)
| Tool Original | Repo Origem | API Real | → Feature mcp-brasil |
|--------------|-------------|----------|---------------------|
| `cep_lookup` | mcp-dadosbr + 3 repos | brasilapi.com.br/api/cep | `brasilapi/` |
| `cnpj_lookup` | mcp-dadosbr + 3 repos | brasilapi.com.br/api/cnpj | `brasilapi/` |
| `cpf_validate` | mcp-dadosbr | validação local | `brasilapi/` |
| `fipe_veiculos` | mcp-dadosbr + mcp-brasil-api | brasilapi.com.br/api/fipe | `brasilapi/` |
| `consultar_ddd` | brasil-api-mcp + mcp-brasil-api | brasilapi.com.br/api/ddd | `brasilapi/` |
| `consultar_banco` | brasil-api-mcp-server + mcp-brasil-api | brasilapi.com.br/api/banks | `brasilapi/` |
| `consultar_cambio` | brasil-api-mcp + mcp-brasil-api | brasilapi.com.br/api/cambio | `brasilapi/` |
| `consultar_feriados` | mcp-brasil-api | brasilapi.com.br/api/feriados | `brasilapi/` |
| `consultar_taxa` | mcp-brasil-api | brasilapi.com.br/api/taxas | `brasilapi/` |
| `isbn_lookup` | brasil-api-mcp-server | brasilapi.com.br/api/isbn | `brasilapi/` |
| `domain_whois` | mcp-dadosbr | brasilapi.com.br/api/registrobr | `brasilapi/` |
| `pix_participantes` | — (BrasilAPI tem) | brasilapi.com.br/api/pix | `brasilapi/` |
| `ncm_lookup` | — (BrasilAPI tem) | brasilapi.com.br/api/ncm | `brasilapi/` |
| `cptec_previsao` | — (BrasilAPI tem) | brasilapi.com.br/api/cptec | `brasilapi/` |

#### Compras Públicas (1 tool existente + PNCP novo)
| Tool Original | Repo Origem | API Real | → Feature mcp-brasil |
|--------------|-------------|----------|---------------------|
| `pncp_licitacoes` | mcp-dadosbr | pncp.gov.br/api | `compras/` |
| **Novos:** buscar_contratos, fornecedores, materiais | — | compras.dados.gov.br | `compras/` |

#### Diários Oficiais (1 tool + wrapper existente)
| Tool Original | Repo Origem | API Real | → Feature mcp-brasil |
|--------------|-------------|----------|---------------------|
| `querido_diario` | mcp-dadosbr | queridodiario.ok.org.br | `diario_oficial/` |
| Wrapper Python completo | querido-diario-api-wrapper | queridodiario.ok.org.br | `diario_oficial/` |

#### Saúde (1 tool existente)
| Tool Original | Repo Origem | API Real | → Feature mcp-brasil |
|--------------|-------------|----------|---------------------|
| `cnes_saude` | mcp-dadosbr | cnes.datasus.gov.br | `saude/` |
| **Novos:** opendatasus busca, vacinacao | — | opendatasus.saude.gov.br | `saude/` |

#### OSINT (1 tool existente)
| Tool Original | Repo Origem | API Real | → Feature mcp-brasil |
|--------------|-------------|----------|---------------------|
| `strategic_osint_prompt` | mcp-dadosbr | prompt generator | `osint/` (não priorizado) |
| `consumidor_reclamacoes` | mcp-dadosbr | scraping | `osint/` |
| `company_deep_profile` | mcp-dadosbr | aggregate | `osint/` |

#### NOVOS — Sem MCP Server Existente
| Feature Planejada | API | Prioridade |
|------------------|-----|-----------|
| `camara/` | dadosabertos.camara.leg.br/api/v2 | 🔴 ALTA |
| `tse/` | dadosabertos.tse.jus.br + divulgacandcontas | 🔴 ALTA |
| `sapl/` | {instancia}/api/ (genérico Django REST) | 🔴 ALTA |
| `dados_abertos/` | dados.gov.br/api/3/ (CKAN) | 🟡 MÉDIA |
| `inpe/` | terrabrasilis.dpi.inpe.br + queimadas | 🟡 MÉDIA |
| `ana/` | snirh.gov.br/hidroweb/rest/api | 🟡 MÉDIA |
| `receita/` | minhareceita.org (CNPJ gratuito) | 🟡 MÉDIA |
| `lexml/` | lexml.gov.br/busca/SRU | 🟢 BAIXA |

---

## 2. Arquitetura da POC — Como Reescrever Tudo

### 2.1 Princípio: Reescrita, Não Fork

Cada repo será **estudado** (tools, APIs, lógica) e **reescrito do zero** em Python/FastMCP seguindo ADR-001 + ADR-002. Motivos:

1. **7 de 8 repos são TypeScript** — precisam ser portados para Python
2. **Nenhum segue nossa convenção** (Package by Feature, Auto-Registry)
3. **APIs duplicadas** entre repos (CEP, CNPJ, IBGE aparecem em 4+ repos)
4. **Código acoplado** — mcp-dadosbr mistura OSINT com governo
5. **Dependências pagas** — mcp-dadosbr requer Tavily/Perplexity; nós não

### 2.2 Mapping: Repo Original → Feature mcp-brasil

> **Repos clonados em:** `.claude/.tmp/projects/`

```
mcp-dadosbr (TS, 22 tools)                    → .claude/.tmp/projects/mcp-dadosbr/
├── core.ts (cnpj, cep)           → brasilapi/
├── financial.ts (bacen, fipe)    → bacen/ + brasilapi/
├── government.ts (ibge, transp)  → ibge/ + transparencia/ + compras/ + diario_oficial/
├── legal.ts (datajud, oab)       → datajud/
├── health.ts (cnes)              → saude/
├── company.ts (cpf, whois)       → brasilapi/
└── osint.ts                      → [descartado na POC]

senado-br-mcp (TS, 33 tools)                  → .claude/.tmp/projects/senado-br-mcp/
└── tools/* (senadores, materias)  → senado/ (port direto, maior feature)

bcb-br-mcp (TS, 8 tools)                      → .claude/.tmp/projects/bcb-br-mcp/
└── tools.ts (séries, catálogo)    → bacen/ (merge com bacen_taxas do mcp-dadosbr)

brasil-api-mcp-server (TS, 5 tools)            → .claude/.tmp/projects/brasil-api-mcp-server/
└── tools/* (banks, cep, cnpj)     → brasilapi/ (absorvido)

brasil-api-mcp (TS, 7 tools)                   → .claude/.tmp/projects/brasil-api-mcp/
└── tools/* (cep, cambio, ddd)     → brasilapi/ (absorvido)

mcp-brasil-api (Python, 12 tools)              → .claude/.tmp/projects/mcp-brasil-api/
└── tools/* (cep, cnpj, fipe)      → brasilapi/ (referência Python direta!)

brlaw_mcp_server (Python, 3 tools)             → .claude/.tmp/projects/brlaw_mcp_server/
└── domain/* (stj, tst, stf)       → jurisprudencia/ (port direto, já Python!)

querido-diario-api-wrapper (Python)            → .claude/.tmp/projects/querido-diario-api-wrapper/
└── wrapper completo               → diario_oficial/ (usar como client base)

BrasilAPI (Next.js, 17 endpoints)              → .claude/.tmp/projects/BrasilAPI/
└── pages/api/*                    → brasilapi/ (referência para completar endpoints)

sapl (Django, API REST)                        → .claude/.tmp/projects/sapl/
└── api/ views + serializers       → sapl/ (entender schema para adapter genérico)

divulgacandcontas-doc (Swagger)                → .claude/.tmp/projects/divulgacandcontas-doc/
└── swagger.yaml                   → tse/ (usar como spec para implementar client)
```

### Repos adicionais clonados (sem mapping direto)

```
.claude/.tmp/projects/ANA-hidroweb/            → ana/ (referência para hidrologia)
.claude/.tmp/projects/agrobr-mcp/              → referência Python MCP agrícola
.claude/.tmp/projects/brasil-api/              → fonte original BrasilAPI
.claude/.tmp/projects/brasilapi-php/           → wrapper PHP (referência de endpoints)
.claude/.tmp/projects/censo-querido-diario/    → scraper do Querido Diário
.claude/.tmp/projects/cepesp-rest/             → API CEPESP-FGV dados eleitorais
.claude/.tmp/projects/esocial/                 → referência eSocial
.claude/.tmp/projects/fastmcp/                 → código-fonte do framework FastMCP
.claude/.tmp/projects/py-lexml-acervo/         → wrapper Python LexML
.claude/.tmp/projects/querido-diario/          → scraper principal
.claude/.tmp/projects/querido-diario-api/      → API FastAPI do Querido Diário
```

### 2.3 Estrutura Final do Projeto

```
mcp-brasil/
├── src/mcp_brasil/
│   ├── __init__.py
│   ├── server.py                    # Auto-Registry (NUNCA MUDA)
│   ├── settings.py
│   ├── exceptions.py
│   │
│   ├── _shared/                     # Kernel compartilhado
│   │   ├── feature.py               # FeatureRegistry + FeatureMeta
│   │   ├── http_client.py           # httpx + retry + cache + rate-limit
│   │   ├── cache.py                 # LRU com TTL
│   │   ├── formatting.py            # Markdown tables, BRL format
│   │   └── validators.py            # CPF, CNPJ, CEP validators
│   │
│   │  # ═══════ TIER 1: MVP (Semanas 1-2) ═══════
│   │
│   ├── ibge/                        # 🔓 8 tools — SEM AUTH
│   │   ├── __init__.py              # FEATURE_META
│   │   ├── server.py                # mcp: FastMCP
│   │   ├── tools.py                 # buscar_localidades, consultar_populacao,
│   │   │                            # consultar_pib, ranking_nomes, obter_malha,
│   │   │                            # consultar_agregado, buscar_cnae, consultar_inflacao
│   │   ├── client.py                # IBGEClient → servicodados.ibge.gov.br
│   │   ├── schemas.py               # Pydantic models
│   │   └── constants.py             # URLs, códigos de agregados
│   │
│   ├── bacen/                       # 🔓 12 tools — SEM AUTH
│   │   ├── __init__.py              # FEATURE_META
│   │   ├── server.py
│   │   ├── tools.py                 # consultar_serie, ultimos_valores,
│   │   │                            # metadados_serie, series_populares,
│   │   │                            # buscar_serie, indicadores_atuais,
│   │   │                            # calcular_variacao, comparar_series,
│   │   │                            # consultar_cambio, consultar_ptax,
│   │   │                            # expectativas_focus, estatisticas_pix
│   │   ├── client.py                # BCBClient → api.bcb.gov.br + olinda.bcb.gov.br
│   │   ├── catalog.py               # Catálogo 150+ séries (port do bcb-br-mcp)
│   │   ├── schemas.py
│   │   └── constants.py
│   │
│   ├── transparencia/               # 🔑 8 tools — API KEY GRATUITA
│   │   ├── __init__.py              # FEATURE_META(requires_auth=True,
│   │   │                            #   auth_env_var="TRANSPARENCIA_API_KEY")
│   │   ├── server.py
│   │   ├── tools.py                 # buscar_contratos, consultar_despesas,
│   │   │                            # buscar_servidores, buscar_licitacoes,
│   │   │                            # consultar_bolsa_familia, buscar_ceis_cnep,
│   │   │                            # buscar_emendas, consultar_viagens
│   │   ├── client.py                # TransparenciaClient → api.portaldatransparencia
│   │   ├── schemas.py
│   │   └── constants.py
│   │
│   │  # ═══════ TIER 2: Legislativo (Semanas 3-4) ═══════
│   │
│   ├── camara/                      # 🔓 10 tools — SEM AUTH ★ NOVO
│   │   ├── __init__.py
│   │   ├── server.py
│   │   ├── tools.py                 # buscar_deputado, listar_deputados,
│   │   │                            # buscar_proposicao, consultar_tramitacao,
│   │   │                            # buscar_votacao, votos_nominais,
│   │   │                            # despesas_deputado, agenda_legislativa,
│   │   │                            # buscar_comissoes, frentes_parlamentares
│   │   ├── client.py                # CamaraClient → dadosabertos.camara.leg.br/api/v2
│   │   ├── schemas.py
│   │   └── constants.py
│   │
│   ├── senado/                      # 🔓 20 tools — SEM AUTH (port do senado-br-mcp)
│   │   ├── __init__.py
│   │   ├── server.py
│   │   ├── tools_senadores.py       # listar, detalhes, buscar, votacoes
│   │   ├── tools_materias.py        # buscar, detalhes, tramitacao, textos, emendas
│   │   ├── tools_votacoes.py        # listar, detalhes, votos
│   │   ├── tools_comissoes.py       # listar, detalhes, membros, reunioes
│   │   ├── tools_ecidadania.py      # ideias, consultas, sugestoes, eventos
│   │   ├── tools_auxiliares.py      # legislaturas, partidos, blocos
│   │   ├── client.py                # SenadoClient → legis.senado.leg.br/dadosabertos
│   │   ├── schemas.py
│   │   └── constants.py
│   │
│   ├── dados_abertos/               # 🔓 4 tools — SEM AUTH ★ NOVO
│   │   ├── __init__.py
│   │   ├── server.py
│   │   ├── tools.py                 # buscar_datasets, listar_organizacoes,
│   │   │                            # detalhar_dataset, consultar_recurso
│   │   ├── client.py                # CKANClient → dados.gov.br/api/3
│   │   ├── schemas.py
│   │   └── constants.py
│   │
│   │  # ═══════ TIER 3: Judiciário + Eleitoral (Semanas 5-6) ═══════
│   │
│   ├── datajud/                     # 🔑 6 tools — API KEY CNJ
│   │   ├── __init__.py              # FEATURE_META(requires_auth=True,
│   │   │                            #   auth_env_var="DATAJUD_API_KEY")
│   │   ├── server.py
│   │   ├── tools.py                 # consultar_processo, buscar_por_classe,
│   │   │                            # buscar_por_assunto, buscar_por_orgao,
│   │   │                            # obter_movimentacoes, estatisticas_tribunal
│   │   ├── client.py                # DataJudClient → api-publica.datajud.cnj.jus.br
│   │   ├── schemas.py               # Elasticsearch query builders
│   │   └── constants.py             # aliases dos ~90 tribunais
│   │
│   ├── jurisprudencia/              # 🔓 6 tools — SEM AUTH (port do brlaw_mcp_server)
│   │   ├── __init__.py
│   │   ├── server.py
│   │   ├── tools.py                 # buscar_stj, buscar_tst, buscar_stf,
│   │   │                            # detalhes_acordao, temas_repetitivos,
│   │   │                            # sumulas_vinculantes
│   │   ├── client_stj.py            # STJClient
│   │   ├── client_tst.py            # TSTClient
│   │   ├── client_stf.py            # STFClient
│   │   ├── schemas.py
│   │   └── constants.py
│   │
│   ├── tse/                         # 🔓 6 tools — SEM AUTH ★ NOVO
│   │   ├── __init__.py
│   │   ├── server.py
│   │   ├── tools.py                 # buscar_candidato, resultados_eleicao,
│   │   │                            # buscar_eleitorado, consultar_partido,
│   │   │                            # prestacao_contas, buscar_datasets_tse
│   │   ├── client_ckan.py           # TSE CKAN → dadosabertos.tse.jus.br/api/3
│   │   ├── client_divulga.py        # DivulgaCandContas REST
│   │   ├── schemas.py
│   │   └── constants.py
│   │
│   │  # ═══════ TIER 4: BrasilAPI + Diários + Compras (Semanas 7-8) ═══════
│   │
│   ├── brasilapi/                   # 🔓 16 tools — SEM AUTH
│   │   ├── __init__.py
│   │   ├── server.py
│   │   ├── tools.py                 # consultar_cep, consultar_cnpj,
│   │   │                            # validar_cpf, consultar_ddd,
│   │   │                            # listar_bancos, consultar_banco,
│   │   │                            # consultar_cambio, consultar_feriados,
│   │   │                            # buscar_fipe, consultar_isbn,
│   │   │                            # consultar_ncm, previsao_tempo,
│   │   │                            # consultar_taxas, participantes_pix,
│   │   │                            # consultar_cvm, consultar_registrobr
│   │   ├── client.py                # BrasilAPIClient → brasilapi.com.br
│   │   ├── schemas.py
│   │   └── constants.py
│   │
│   ├── diario_oficial/              # 🔓 4 tools — SEM AUTH
│   │   ├── __init__.py
│   │   ├── server.py
│   │   ├── tools.py                 # buscar_publicacoes, buscar_por_municipio,
│   │   │                            # extrair_texto, listar_municipios_cobertos
│   │   ├── client.py                # QDClient (port do wrapper Python existente)
│   │   ├── schemas.py
│   │   └── constants.py
│   │
│   ├── compras/                     # 🔓 6 tools — SEM AUTH ★ NOVO
│   │   ├── __init__.py
│   │   ├── server.py
│   │   ├── tools.py                 # buscar_licitacoes_pncp, buscar_contratos_pncp,
│   │   │                            # buscar_comprasnet, consultar_fornecedor,
│   │   │                            # buscar_materiais, buscar_servicos
│   │   ├── client_pncp.py           # PNCPClient → pncp.gov.br/api
│   │   ├── client_comprasnet.py     # ComprasClient → compras.dados.gov.br
│   │   ├── schemas.py
│   │   └── constants.py
│   │
│   │  # ═══════ TIER 5: Especializados (Semanas 9-12) ═══════
│   │
│   ├── saude/                       # 🔓 4 tools — SEM AUTH
│   │   ├── __init__.py
│   │   ├── server.py
│   │   ├── tools.py                 # buscar_estabelecimento_cnes,
│   │   │                            # buscar_datasets_saude, consultar_vacinacao,
│   │   │                            # estatisticas_epidemiologicas
│   │   ├── client.py                # OpenDataSUSClient (CKAN)
│   │   ├── schemas.py
│   │   └── constants.py
│   │
│   ├── sapl/                        # 🔓 5 tools — SEM AUTH ★ NOVO ÚNICO
│   │   ├── __init__.py
│   │   ├── server.py
│   │   ├── tools.py                 # buscar_materia, buscar_parlamentar,
│   │   │                            # buscar_norma, buscar_sessao,
│   │   │                            # listar_instancias_conhecidas
│   │   ├── client.py                # SAPLClient (genérico, aceita URL base)
│   │   ├── registry.py              # Catálogo de instâncias SAPL conhecidas
│   │   ├── schemas.py
│   │   └── constants.py             # URLs de 50+ câmaras municipais
│   │
│   ├── inpe/                        # 🔓 4 tools — SEM AUTH ★ NOVO
│   │   ├── __init__.py
│   │   ├── server.py
│   │   ├── tools.py                 # consultar_desmatamento, buscar_queimadas,
│   │   │                            # dados_satelite, alertas_deter
│   │   ├── client.py                # INPEClient → terrabrasilis
│   │   ├── schemas.py
│   │   └── constants.py
│   │
│   └── ana/                         # 🔓 3 tools — SEM AUTH ★ NOVO
│       ├── __init__.py
│       ├── server.py
│       ├── tools.py                 # buscar_estacao, consultar_dados_hidrologicos,
│       │                            # monitorar_reservatorios
│       ├── client.py                # ANAClient → snirh.gov.br/hidroweb/rest/api
│       ├── schemas.py
│       └── constants.py
│
├── tests/                           # Espelha features
│   ├── conftest.py                  # Fixtures globais
│   ├── ibge/
│   ├── bacen/
│   ├── senado/
│   ├── camara/
│   └── ...
│
├── plan/decisions/
│   ├── ADR-001-project-bootstrap.md
│   └── ADR-002-auto-registry-pattern.md
│
├── examples/
│   ├── claude_desktop_config.json
│   └── quick_demo.py
│
├── pyproject.toml
├── justfile
├── AGENTS.md
├── CLAUDE.md
├── CONTRIBUTING.md
├── CODE_OF_CONDUCT.md
└── README.md
```

---

## 3. Plano de Execução — 12 Semanas

### Fase 0: Bootstrap (2 dias)

**Objetivo:** Repo funcional com Auto-Registry, zero features.

| # | Tarefa | Arquivo |
|---|--------|---------|
| 0.1 | Criar repo GitHub `mcp-brasil` | — |
| 0.2 | `pyproject.toml` com FastMCP, httpx, Pydantic | `pyproject.toml` |
| 0.3 | `justfile` completo (install, test, lint, run, serve) | `justfile` |
| 0.4 | `_shared/feature.py` — FeatureRegistry + FeatureMeta | `src/mcp_brasil/_shared/` |
| 0.5 | `_shared/http_client.py` — httpx + retry + backoff | `src/mcp_brasil/_shared/` |
| 0.6 | `_shared/cache.py` — LRU com TTL | `src/mcp_brasil/_shared/` |
| 0.7 | `_shared/formatting.py` — Markdown tables, BRL | `src/mcp_brasil/_shared/` |
| 0.8 | `_shared/validators.py` — CPF, CNPJ, CEP | `src/mcp_brasil/_shared/` |
| 0.9 | `server.py` raiz (3 linhas, nunca muda) | `src/mcp_brasil/server.py` |
| 0.10 | CI: GitHub Actions (ruff → mypy → pytest) | `.github/workflows/` |
| 0.11 | AGENTS.md + CLAUDE.md + CONTRIBUTING.md | raiz |
| 0.12 | ADRs commitados | `plan/decisions/` |

**Validação:** `fastmcp run mcp_brasil.server:mcp` roda sem features.

### Fase 1: IBGE + Bacen + Transparência (Semana 1-2)

**Fontes primárias de estudo:**
- `.claude/.tmp/projects/mcp-dadosbr/lib/tools/government.ts` → lógica ibge_localidades
- `.claude/.tmp/projects/mcp-dadosbr/lib/tools/financial.ts` → lógica bacen_taxas
- `.claude/.tmp/projects/bcb-br-mcp/src/tools.ts` → catálogo 150+ séries, 8 tools BCB
- `.claude/.tmp/projects/mcp-brasil-api/server.py` → referência Python FastMCP
- `.claude/.tmp/projects/mcp-dadosbr/lib/tools/government.ts` → transparencia_lookup, ceis_cnep

| Semana | Feature | Tools | Fonte de Referência |
|--------|---------|-------|-------------------|
| 1 | `ibge/` | 8 tools | API direta IBGE + mcp-dadosbr |
| 1 | `_shared/` | http_client, cache | mcp-dadosbr/lib/infrastructure/ |
| 2 | `bacen/` | 12 tools | **bcb-br-mcp** (port TS→Python) |
| 2 | `transparencia/` | 8 tools | mcp-dadosbr + API Swagger direta |

**Entregável:** `mcp-brasil==0.1.0` no PyPI com ~28 tools.

### Fase 2: Legislativo Federal (Semana 3-4)

**Fontes primárias de estudo:**
- `.claude/.tmp/projects/senado-br-mcp/src/tools/*` → 33 tools TS a portar
- `.claude/.tmp/projects/senado-br-mcp/src/api/endpoints.ts` → mapeamento completo
- `.claude/.tmp/projects/senado-br-mcp/src/schemas/*` → tipos de dados
- API Swagger Câmara: `dadosabertos.camara.leg.br/swagger/api.html`

| Semana | Feature | Tools | Fonte de Referência |
|--------|---------|-------|-------------------|
| 3 | `senado/` | 20 tools (curados dos 33) | **senado-br-mcp** (port TS→Python) |
| 3 | `camara/` | 10 tools | API Swagger direta ★ NOVO |
| 4 | `dados_abertos/` | 4 tools | API CKAN dados.gov.br ★ NOVO |

**Entregável:** `mcp-brasil==0.2.0` — ~62 tools. **DIA DE LANÇAMENTO.**

### Fase 3: Judiciário + Eleitoral (Semana 5-6)

**Fontes primárias de estudo:**
- `.claude/.tmp/projects/mcp-dadosbr/lib/tools/legal.ts` → datajud_processos, bnmp
- `.claude/.tmp/projects/brlaw_mcp_server/src/domain/*.py` → STJ/TST/STF (já Python!)
- `.claude/.tmp/projects/divulgacandcontas-doc/swagger.yaml` → spec TSE
- Wiki DataJud: `datajud-wiki.cnj.jus.br/api-publica/`

| Semana | Feature | Tools | Fonte de Referência |
|--------|---------|-------|-------------------|
| 5 | `datajud/` | 6 tools | mcp-dadosbr + Wiki CNJ |
| 5 | `jurisprudencia/` | 6 tools | **brlaw_mcp_server** (port direto Python!) |
| 6 | `tse/` | 6 tools | **divulgacandcontas-doc** + CKAN TSE ★ NOVO |

**Entregável:** `mcp-brasil==0.3.0` — ~80 tools.

### Fase 4: BrasilAPI + Diários + Compras (Semana 7-8)

**Fontes primárias de estudo:**
- `.claude/.tmp/projects/mcp-brasil-api/server.py` → referência Python FastMCP completa
- `.claude/.tmp/projects/brasil-api-mcp-server/src/tools/*` → endpoints TS
- `.claude/.tmp/projects/BrasilAPI/pages/api/*` → 17 endpoints Next.js
- `.claude/.tmp/projects/querido-diario-api-wrapper/` → wrapper Python existente
- `.claude/.tmp/projects/mcp-dadosbr/lib/tools/government.ts` → pncp_licitacoes

| Semana | Feature | Tools | Fonte de Referência |
|--------|---------|-------|-------------------|
| 7 | `brasilapi/` | 16 tools | **mcp-brasil-api** + 3 repos BrasilAPI |
| 7 | `diario_oficial/` | 4 tools | **querido-diario-api-wrapper** (Python!) |
| 8 | `compras/` | 6 tools | mcp-dadosbr + APIs PNCP/Comprasnet ★ NOVO |

**Entregável:** `mcp-brasil==0.4.0` — ~106 tools.

### Fase 5: Especializados (Semana 9-12)

| Semana | Feature | Tools | Fonte de Referência |
|--------|---------|-------|-------------------|
| 9 | `saude/` | 4 tools | mcp-dadosbr (cnes) + OpenDataSUS |
| 9 | `sapl/` | 5 tools | **sapl/** repo Django (API REST genérica) ★ ÚNICO |
| 10 | `inpe/` | 4 tools | TerraBrasilis API ★ NOVO |
| 10 | `ana/` | 3 tools | HidroWeb + **ANA-hidroweb** repo ★ NOVO |
| 11-12 | Polish | — | Docs, vídeos, integrações |

**Entregável:** `mcp-brasil==1.0.0` — **~122 tools**, 16 features.

---

## 4. Mapa de Port — De Qual Repo Vem Cada Feature

```
┌─────────────────────────────────────────────────────────────────┐
│                        mcp-brasil v1.0                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  PORTADOS DE REPOS EXISTENTES:                                  │
│  ┌───────────────┐  ┌──────────────────┐  ┌──────────────────┐ │
│  │ bacen/        │  │ senado/          │  │ jurisprudencia/  │ │
│  │ PORT de       │  │ PORT de          │  │ PORT de          │ │
│  │ bcb-br-mcp    │  │ senado-br-mcp    │  │ brlaw_mcp_server │ │
│  │ (TS→Python)   │  │ (TS→Python)      │  │ (Python→Python)  │ │
│  │ 12 tools      │  │ 20 tools         │  │ 6 tools          │ │
│  └───────────────┘  └──────────────────┘  └──────────────────┘ │
│                                                                 │
│  ┌───────────────┐  ┌──────────────────┐  ┌──────────────────┐ │
│  │ brasilapi/    │  │ diario_oficial/  │  │ ibge/            │ │
│  │ MERGE de      │  │ PORT de          │  │ MERGE de         │ │
│  │ 4 repos       │  │ querido-diario   │  │ 2 repos +        │ │
│  │ (TS+Py→Py)    │  │ -api-wrapper     │  │ API direta       │ │
│  │ 16 tools      │  │ (Python→Python)  │  │ 8 tools          │ │
│  └───────────────┘  │ 4 tools          │  └──────────────────┘ │
│                     └──────────────────┘                        │
│  ABSORVIDOS PARCIAIS (tools individuais):                       │
│  ┌───────────────┐  ┌──────────────────┐  ┌──────────────────┐ │
│  │ transparencia/│  │ datajud/         │  │ saude/           │ │
│  │ de mcp-dadosbr│  │ de mcp-dadosbr   │  │ de mcp-dadosbr   │ │
│  │ + API direta  │  │ + Wiki CNJ       │  │ + OpenDataSUS    │ │
│  │ 8 tools       │  │ 6 tools          │  │ 4 tools          │ │
│  └───────────────┘  └──────────────────┘  └──────────────────┘ │
│                                                                 │
│  100% NOVOS (SEM REPO DE REFERÊNCIA):                           │
│  ┌───────────────┐  ┌──────────────────┐  ┌──────────────────┐ │
│  │ camara/       │  │ tse/             │  │ sapl/            │ │
│  │ ★ NOVO        │  │ ★ NOVO           │  │ ★ NOVO ÚNICO     │ │
│  │ API Swagger   │  │ CKAN + DivulgaCC │  │ Adapter genérico │ │
│  │ 10 tools      │  │ 6 tools          │  │ 5 tools          │ │
│  └───────────────┘  └──────────────────┘  └──────────────────┘ │
│  ┌───────────────┐  ┌──────────────────┐  ┌──────────────────┐ │
│  │ compras/      │  │ inpe/            │  │ ana/             │ │
│  │ ★ NOVO        │  │ ★ NOVO           │  │ ★ NOVO           │ │
│  │ PNCP+Compras  │  │ TerraBrasilis    │  │ HidroWeb         │ │
│  │ 6 tools       │  │ 4 tools          │  │ 3 tools          │ │
│  └───────────────┘  └──────────────────┘  └──────────────────┘ │
│  ┌───────────────┐                                              │
│  │ dados_abertos/│                                              │
│  │ ★ NOVO        │                                              │
│  │ CKAN dados.gov│                                              │
│  │ 4 tools       │                                              │
│  └───────────────┘                                              │
│                                                                 │
│  TOTAL: 16 features · ~122 tools · 3 poderes · 15+ órgãos      │
└─────────────────────────────────────────────────────────────────┘
```

---

## 5. Regras de Reescrita

### 5.1 De TypeScript para Python

| Pattern TS (repos originais) | Pattern Python (mcp-brasil) |
|-----------------------------|-----------------------------|
| `z.string().describe(...)` | `Field(description=...)` no Pydantic |
| `registerTool({ name, schema, handler })` | `@mcp.tool` decorator com type hints |
| `fetch(url, { headers })` | `httpx.AsyncClient.get(url, headers=)` |
| `try/catch + JSON.stringify` | `try/except + Pydantic .model_dump_json()` |
| `interface SerieValor { ... }` | `class SerieValor(BaseModel): ...` |
| `const CONFIG = { ... }` | `class Settings(BaseSettings): ...` |
| Circuit breaker manual | `tenacity` retry decorator |
| Rate limiter manual | `_shared/http_client.py` com semaphore |

### 5.2 Regras de Qualidade (Clean Code)

1. **Toda tool tem docstring** — é usada pelo LLM para decidir quando chamar
2. **tools.py nunca faz HTTP** — delega para client.py
3. **client.py nunca formata** — retorna Pydantic models
4. **schemas.py zero lógica** — apenas Pydantic BaseModels
5. **server.py apenas registra** — zero lógica de negócio
6. **Async everywhere** — async def em tools e clients
7. **Type hints completos** — mypy strict
8. **Retorno formatado para LLM** — Markdown tables, valores formatados em BRL

### 5.3 O Que NÃO Portar

| Componente | Repo | Motivo |
|-----------|------|--------|
| OSINT tools | mcp-dadosbr | Fora de escopo (dados gov, não investigação) |
| `cnpj_intelligence` | mcp-dadosbr | Requer Tavily/Perplexity (APIs pagas) |
| `company_deep_profile` | mcp-dadosbr | Requer Tavily/Perplexity (APIs pagas) |
| `strategic_osint_prompt` | mcp-dadosbr | Prompt engineering, não dados |
| `consumidor_reclamacoes` | mcp-dadosbr | Scraping instável |
| e-Cidadania scraping | senado-br-mcp | Scraping de páginas HTML (frágil) |
| Cloudflare Worker adapter | bcb-br-mcp | Não relevante para FastMCP Python |

---

## 6. Métricas da POC

| Milestone | Quando | Features | Tools | Stars Alvo |
|-----------|--------|----------|-------|-----------|
| v0.1.0-alpha | Semana 1 | 1 (ibge) | 8 | — |
| v0.1.0 | Semana 2 | 3 (ibge+bacen+transp) | 28 | — |
| v0.2.0 🚀 LAUNCH | Semana 4 | 6 (+camara+senado+dados) | 62 | 100+ |
| v0.3.0 | Semana 6 | 9 (+datajud+jurisp+tse) | 80 | 300+ |
| v0.4.0 | Semana 8 | 12 (+brasilapi+diario+compras) | 106 | 500+ |
| v1.0.0 | Semana 12 | 16 (todas) | 122 | 1000+ |

---

## 7. Próximo Passo Imediato

**Começar pela Fase 0 (Bootstrap)** — implementar:

1. `_shared/feature.py` (FeatureRegistry — já desenhado no ADR-002)
2. `_shared/http_client.py` (estudar `mcp-dadosbr/lib/infrastructure/`)
3. `ibge/` como primeira feature (API mais simples, sem auth)
4. Testes para ibge/ (unit + mock + integration)
5. Publicar `mcp-brasil==0.1.0-alpha` no PyPI

**Referências prontas nos repos clonados (`.claude/.tmp/projects/`):**
- `.claude/.tmp/projects/mcp-brasil-api/` → Python FastMCP pattern
- `.claude/.tmp/projects/brlaw_mcp_server/` → Python clean architecture
- `.claude/.tmp/projects/bcb-br-mcp/src/tools.ts` → catálogo séries BCB
- `.claude/.tmp/projects/senado-br-mcp/src/` → maior cobertura legislativa
- `.claude/.tmp/projects/BrasilAPI/pages/api/` → 17 endpoints de referência
- `.claude/.tmp/projects/fastmcp/` → código-fonte do framework FastMCP
- `.claude/.tmp/projects/ANA-hidroweb/` → referência para feature ana/
- `.claude/.tmp/projects/agrobr-mcp/` → referência Python MCP agrícola
- `.claude/.tmp/projects/querido-diario-api/` → API FastAPI do Querido Diário
- `.claude/.tmp/projects/cepesp-rest/` → API CEPESP-FGV dados eleitorais
- `.claude/.tmp/projects/py-lexml-acervo/` → wrapper Python LexML

---

*Documento gerado em 2026-03-21 · Atualizado em 2026-03-22 com caminhos locais dos repos*
*Repos clonados em: `.claude/.tmp/projects/` (21 repositórios)*
*Total de código analisado: ~106MB · 92+ tools únicos mapeados*
