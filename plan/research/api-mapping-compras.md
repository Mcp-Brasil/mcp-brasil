# Mapa completo de APIs de compras públicas do Brasil para o mcp-brasil

O Brasil possui **mais de 40 APIs e fontes de dados públicos** para compras governamentais, distribuídas entre sistemas federais, estaduais, municipais e de controle. O **PNCP** (Portal Nacional de Contratações Públicas), obrigatório pela Lei 14.133/2021, emerge como a fonte central que unifica dados de todos os níveis de governo — e não possui MCP server. Apenas um projeto existente (`mcp-portal-transparencia`) cobre parcialmente o ecossistema brasileiro. Este mapa identifica cada fonte, suas APIs, formatos, autenticação e status, fornecendo a base técnica completa para o projeto mcp-brasil.

---

## 1. APIs federais: o núcleo do ecossistema

O governo federal mantém as APIs mais maduras e bem documentadas do ecossistema de compras públicas brasileiro. São **seis sistemas principais** com acesso programático.

### PNCP — Portal Nacional de Contratações Públicas

O PNCP é o **sistema mais importante para o mcp-brasil**, pois centraliza dados de todas as esferas governamentais (federal, estadual, municipal e distrital) conforme mandato da Lei 14.133/2021. Oferece **duas camadas de API**:

**API de Consulta (pública, sem autenticação):**
- Base URL: `https://pncp.gov.br/api/consulta`
- Swagger: `https://pncp.gov.br/api/consulta/swagger-ui/index.html`
- Formato: JSON (REST/HTTP 1.1)
- Endpoints principais: contratações por data de publicação, contratos, atas de registro de preços, itens de PCA (Plano de Contratações Anual), propostas em aberto
- Filtros por esfera: F (Federal), E (Estadual), M (Municipal), D (Distrital)

**API de Integração (escrita, requer JWT):**
- Base URL: `https://pncp.gov.br/api/pncp`
- Swagger: `https://pncp.gov.br/api/pncp/swagger-ui/index.html`
- Autenticação: JWT Bearer token via `POST /v1/usuarios/login` (validade 1 hora)
- Manual de Integração v2.3.10: disponível em `gov.br/pncp/pt-br/central-de-conteudo/manuais/`
- Capacidades: CRUD completo para contratações, itens, documentos (até 30MB), atas, contratos e PCA

Um relatório técnico da Transparência Brasil (junho 2024) identificou limitações: endpoints que retornam valores únicos, nomes de campos inconsistentes entre endpoints e discrepâncias na documentação.

### ComprasNet / Compras.gov.br — três gerações de APIs

O sistema federal de compras possui **três camadas de API** ativas simultaneamente:

**API Legacy (compras.dados.gov.br)** — a mais documentada, operacional para dados até 2020:
- Base URL: `http://compras.dados.gov.br/{modulo}/v1/{metodo}.{formato}`
- REST com HATEOAS, sem autenticação, licença ODBL
- Formatos: JSON, XML, CSV, HTML, RDF
- Módulos: Licitações, Contratos, Fornecedores, Materiais (CATMAT), Serviços (CATSER), Pregões, Compras sem Licitação, PGC
- Limite: **500 registros por requisição**
- O workspace do usuário no Notion já referencia esta API com exemplos de uso

**Nova API de Dados Abertos (dadosabertos.compras.gov.br)** — substituta moderna:
- Swagger: `https://dadosabertos.compras.gov.br/swagger-ui/index.html`
- Postman Docs: `https://documenter.getpostman.com/view/13166820/2sA3XJjPpR`
- Formatos: JSON, CSV
- Cobre dados antigos (SIASG) e novos (Compras.gov.br pós-2021)

**API de Contratos.gov.br** — contratos a partir de 2021:
- Documentação: `https://contratos.comprasnet.gov.br/api/docs`
- REST, JSON, licença GPL
- Ferramenta oficial de importação: `https://github.com/cgugovbr/comprasgovbr-crawler` (PHP/Laravel)

### Portal da Transparência (CGU)

A API mais abrangente para dados federais, incluindo **sanções, contratos e licitações**:
- Swagger: `https://api.portaldatransparencia.gov.br/swagger-ui/index.html`
- OpenAPI: `https://api.portaldatransparencia.gov.br/v3/api-docs`
- Formato: JSON
- **Autenticação obrigatória**: chave API gratuita via registro de email
- Rate limits: **90 req/min** (06h–23h59) e **300 req/min** (00h–05h59)
- Endpoints de compras: Licitações, Contratos, Convênios do Poder Executivo Federal
- Endpoints de sanções: CEIS, CNEP, CEPIM, CEAF (detalhados na seção 5)
- Download em massa: CSV disponível em `portaldatransparencia.gov.br/download-de-dados`

### CATMAT/CATSER — Catálogos de materiais e serviços

Acessíveis por **duas vias simultâneas**:
- Via API legacy: `compras.dados.gov.br/materiais/v1/` e `compras.dados.gov.br/servicos/v1/`
- Via nova API: `dadosabertos.compras.gov.br`
- Interface web: `https://catalogo.compras.gov.br/`
- CATMAT cobre grupos de materiais 10–99; CATSER cobre grupos de serviços 1–9

### Sistemas descontinuados ou em transição

**Painel de Preços**: descontinuado em **4 de julho de 2025**. Substituído pelo módulo "Pesquisa de Preços" dentro do Compras.gov.br. Sem API pública REST documentada.

**Painel de Compras**: em processo de substituição pelo repositório de dados abertos do Compras.gov.br.

**SICAF Legacy** (`api.comprasnet.gov.br/sicaf`): **desativado**. Dados incorporados ao dataset "Compras Públicas do Governo Federal". O SICAF Digital atual opera em `https://www.gov.br/compras/pt-br/sistemas/conheca-o-compras/sicaf-digital`, com dados acessíveis via API consolidada.

---

## 2. Tabela mestre de APIs federais

| Sistema | Base URL da API | REST | Auth | Formatos | Status |
|---------|----------------|------|------|----------|--------|
| **PNCP Consulta** | `pncp.gov.br/api/consulta` | Sim | Nenhuma | JSON | ✅ Ativa |
| **PNCP Integração** | `pncp.gov.br/api/pncp` | Sim | JWT | JSON | ✅ Ativa |
| **ComprasNet Legacy** | `compras.dados.gov.br` | Sim (HATEOAS) | Nenhuma | JSON/XML/CSV/HTML | ✅ Ativa (dados até 2020) |
| **Dados Abertos Compras** | `dadosabertos.compras.gov.br` | Sim | Nenhuma | JSON/CSV | ✅ Ativa (nova) |
| **Contratos.gov.br** | `contratos.comprasnet.gov.br/api` | Sim | Nenhuma (consulta) | JSON | ✅ Ativa |
| **Portal Transparência** | `api.portaldatransparencia.gov.br` | Sim | API Key (grátis) | JSON | ✅ Ativa |
| **CATMAT/CATSER** | Via compras.dados.gov.br + dadosabertos | Sim | Nenhuma | JSON/XML/CSV | ✅ Ativa |
| **SICAF Legacy** | `api.comprasnet.gov.br/sicaf` | Sim | Nenhuma | JSON/XML/RDF | ❌ Desativada |
| **Painel de Preços** | N/A (web) | Não | N/A | Dashboard | ⚠️ Descontinuado (jul/2025) |

---

## 3. Portais estaduais e municipais com APIs

A disponibilidade de APIs varia drasticamente entre estados. **São Paulo e Minas Gerais** lideram em maturidade de APIs, enquanto a maioria dos estados depende exclusivamente do PNCP e portais web sem acesso programático.

### Estados com APIs dedicadas

**BEC/SP (Bolsa Eletrônica de Compras — São Paulo)**:
- Portal: `https://www.bec.sp.gov.br` e `https://compras.sp.gov.br`
- API: Web Service (SOAP/REST) documentado em `portal.fazenda.sp.gov.br/acessoinformacao/Paginas/Webservice-BEC.aspx`
- Manual técnico: download em formato .doc (Maio 2018)
- Open Data: `catalogo.governoaberto.sp.gov.br/dataset/702-bec-bolsa-eletronica-de-compras`
- Cobertura: Administração Direta, Autarquias, Universidades, Fundações, Estatais e Municípios Paulistas conveniados
- Certificação ISO 9001

**Portal de Compras MG (Minas Gerais)**:
- Plataforma de APIs PRODEMGE: `https://api.prodemge.gov.br/store/` (WSO2 API Manager)
- Transparência API v1.0.0 com dados de compras, fornecedores, itens/serviços
- Open Data CKAN: `https://dados.mg.gov.br` com DataStore API para queries diretas
- Autenticação: registro necessário na API Store
- GitHub: `https://github.com/transparencia-mg` com repositórios portal_contratos e portal_licitacoes

**Goiás**: CKAN DataStore API em `https://dadosabertos.go.gov.br/api/3/action/datastore_search` com documentação em `transparencia.go.gov.br/api-de-dados-abertos/`.

**Rio Grande do Sul**: Portal de APIs em `rs.gov.br/apis-disponiveis` e CKAN em `dados.rs.gov.br`.

**Santa Catarina**: CKAN em `dados.sc.gov.br` com API padrão.

### Estados sem APIs dedicadas (dados via PNCP)

Paraná (ComprasParaná — apenas web), Rio de Janeiro (SIGA — apenas web, mas o TCE-RJ tem API), Bahia, Pernambuco, Ceará e Distrito Federal publicam dados primariamente via PNCP e Compras.gov.br.

### Municípios com APIs

**São Paulo Capital** é o município mais avançado:
- APILIB (Vitrine de APIs): `https://apilib.prefeitura.sp.gov.br/store/` — inclui Licitações API v1 (WSO2)
- Open Data CKAN: `https://dados.prefeitura.sp.gov.br/dataset/base-de-compras-e-licitacoes`

Outros municípios com CKAN DataStore API: **Rio de Janeiro** (data.rio), **Belo Horizonte** (dados.pbh.gov.br), **Recife** (dados.recife.pe.gov.br), **Porto Alegre** (dados.portoalegre.rs.gov.br), **Curitiba** (curitiba.pr.gov.br/dadosabertos).

---

## 4. Estatais e Licitações-e

A maioria das empresas estatais **não oferece APIs públicas**, com exceção do BNDES. Seus dados federais, porém, são acessíveis via Compras.gov.br usando códigos UASG.

**Licitações-e (Banco do Brasil)**: plataforma operacional em `https://licitacoes-e2.bb.com.br/` usada por milhares de entidades públicas. **Sem API pública** — é exclusivamente plataforma de pregão eletrônico.

**Petrobras**: portal em `https://transparencia.petrobras.com.br/licitacoes-e-contratos`. Downloads em Excel/CSV disponíveis, mas **sem API**. A Petrobras explicitamente declara isenção do Plano de Dados Abertos (Decreto 8.777/2016) por operar em condições de mercado competitivo. Fornecedores usam Petronect (SAP Ariba).

**BNDES** — a exceção positiva: portal de dados abertos CKAN em `https://dadosabertos.bndes.gov.br/` com API e exemplos Python/R em `https://github.com/bndes/dados-abertos-exemplos`.

**Caixa Econômica Federal**: portal em `https://licitacoes.caixa.gov.br/consultapublica/`. Sem API pública.

**Eletrobras e Correios**: sem APIs dedicadas. Dados acessíveis via Compras.gov.br (códigos UASG) e, no caso dos Correios, via Licitações-e.

| Estatal | Portal | API | Acesso alternativo |
|---------|--------|-----|--------------------|
| **BNDES** | dadosabertos.bndes.gov.br | ✅ CKAN API | GitHub com exemplos |
| **Petrobras** | transparencia.petrobras.com.br | ❌ Downloads CSV/XLS | Petronect (fornecedores) |
| **Caixa** | licitacoes.caixa.gov.br | ❌ Web only | — |
| **Eletrobras** | gov.br/mme (subsidiárias) | ❌ Web only | Compras.gov.br (UASG) |
| **Correios** | editais.correios.com.br | ❌ Web only | Licitações-e (BB) |
| **Licitações-e (BB)** | licitacoes-e2.bb.com.br | ❌ Web only | — |

---

## 5. Tribunais de Contas e bases de sanções

O TCU possui o **ecossistema de APIs mais rico** entre os órgãos de controle, com mais de 10 endpoints REST públicos. Os TCEs estaduais variam enormemente em maturidade.

### TCU — APIs REST públicas (sem autenticação)

| Endpoint | URL Base | Dados |
|----------|----------|-------|
| **Acórdãos** | `dados-abertos.apps.tcu.gov.br/api/acordao/` | Deliberações, relator, colegiado, resumo, downloads DOC/PDF |
| **Sanções (Inabilitados)** | `contas.tcu.gov.br/ords/condenacao/consulta/inabilitados` | CPF, processo, deliberação, data trânsito, UF |
| **Certidões PJ** | `certidoes-apf.apps.tcu.gov.br/api/rest/publico/certidoes/{cnpj}` | Certidões consolidadas (JSON + PDF) |
| **Atos Normativos** | `dados-abertos.apps.tcu.gov.br/api/atonormativo/` | INs, portarias, resoluções |
| **Pautas de Sessão** | `dados-abertos.apps.tcu.gov.br/api/pautassessao` | Agenda de julgamentos |
| **Solicitações do Congresso** | `contas.tcu.gov.br/ords/api/publica/scn/pedidos_congresso` | Pedidos de auditoria |
| **Termos Contratuais** | `contas.tcu.gov.br/contrata2RS/api/publico/termos-contratuais` | Contratos do próprio TCU |
| **Cálculo de Débito** | `divida.apps.tcu.gov.br/api/publico/calculadora/` | Cálculo com variação Selic |
| **Jurisprudência** | `sites.tcu.gov.br/dados-abertos/jurisprudencia/` | Download CSV (5 bases) |

Todos os endpoints retornam JSON, exceto jurisprudência (CSV) e licitações do TCU (XML). A documentação está em `https://sites.tcu.gov.br/dados-abertos/webservices-tcu/`. Nota: serviços podem ficar indisponíveis entre 20h e 21h diariamente.

### Bases de sanções (via Portal da Transparência API)

As quatro bases de sanções são acessíveis via API do Portal da Transparência (requer API Key):

- **CEIS** (Cadastro de Empresas Inidôneas e Suspensas): empresas impedidas de participar de licitações. Consulta por CNPJ/CPF, órgão sancionador, período.
- **CNEP** (Cadastro Nacional de Empresas Punidas): empresas punidas pela Lei Anticorrupção 12.846/2013.
- **CEPIM** (Entidades Privadas Sem Fins Lucrativos Impedidas): entidades impedidas de firmar convênios. Fonte: SIAFI.
- **CEAF** (Cadastro de Expulsões da Administração Federal): servidores expulsos da administração federal.

O **Banco de Sanções** (`ceiscadastro.cgu.gov.br`) é o sistema unificado que alimenta CEIS, CNEP e CEAF, operado pela CGU.

### TCEs estaduais com APIs

| TCE | URL da API | Tipo | Formatos | Dados principais |
|-----|-----------|------|----------|------------------|
| **TCE-SP** | `transparencia.tce.sp.gov.br/api/{json\|xml}/` | REST | JSON, XML | Despesas, receitas municipais (2014-2019+) |
| **TCE-RJ** | `dados.tcerj.tc.br/api/v1/` | REST | JSON | Licitações, contratos, compras diretas, obras paralisadas |
| **TCE-CE** | `api.tce.ce.gov.br/sim/1_0/` | REST | XML/JSON/CSV | Licitações, despesas, negociantes municipais |
| **TCE-RS** | `dados.tce.rs.gov.br/api/3` | CKAN | CSV/JSON | LicitaCon (licitações e contratos) |
| **TCE-PE** | `sistemas.tce.pe.gov.br/DadosAbertos/` | REST | JSON | Licitações, contratos, fornecedores, obras |
| **TCE-SC** | `tcesc.tc.br/apis-dados-abertos` | REST (Swagger) | JSON/XML | Diversos |
| **TCE-RN** | `apidadosabertos.tce.rn.gov.br/` | REST | CSV/JSON | Dados do SIAI |
| **TCE-TO** | `api.tceto.tc.br/econtas/api` | REST | JSON | Processos, decisões |
| **TCE-PI** | `sistemas.tce.pi.gov.br/api/portaldacidadania/docs/` | REST (Swagger) | JSON | Portal da Cidadania |
| **TCE-MG** | `dadosabertos.tce.mg.gov.br/` | Downloads | CSV | Sem API REST formal |

O **TCE-RJ** destaca-se como a melhor fonte programática para dados de compras de **91 municípios fluminenses** (exceto a capital). O **CNJ DataJud** (`api-publica.datajud.cnj.jus.br`) também oferece API REST (Elasticsearch) para consultar processos judiciais relacionados a licitações, requerendo API Key.

---

## 6. Dados abertos, preços e referências legais

### Portais de dados abertos

O **dados.gov.br** é o catálogo central, com o dataset "Compras Públicas do Governo Federal" consolidando dados do SIASG/ComprasNet. O **Banco de Preços em Saúde (BPS)** merece atenção especial: opera em `https://bps.saude.gov.br` com dashboard em `infoms.saude.gov.br` e dumps CSV anuais via OPENDATASUS (`dadosabertos.saude.gov.br/dataset/bps`). Cobre preços de medicamentos e dispositivos médicos de todos os níveis de governo usando códigos CATMAT. **Sem API REST pública** — acesso via dashboard e downloads CSV.

O repositório GitHub `dadosgovbr/catalogos-dados-brasil` mapeia todos os portais de dados abertos estaduais e municipais, servindo como referência para descoberta de novas fontes.

### Modelos e referências legais da AGU

A AGU mantém modelos oficiais atualizados em `gov.br/agu/pt-br/composicao/cgu/cgu/modelos/licitacoesecontratos`, incluindo editais, termos de referência, contratos e atas para a Lei 14.133/2021. O **Ger@AGU** gera automaticamente editais a partir dos modelos. A última revisão major (abril 2025) incorporou mais de **300 contribuições** de consulta pública.

As **Instruções Normativas SEGES/MGI** regulamentam aspectos específicos: IN 65/2021 (pesquisa de preços), IN 81/2022 (Termo de Referência e TR Digital), IN 58/2022 (ETP Digital), IN 98/2022 (contratação de serviços indiretos), entre outras. O sistema **Contrata+Brasil** (`gov.br/contratamaisbrasil/`) é a nova plataforma de credenciamento com regras definidas pela IN 460/2025.

---

## 7. Projetos open-source existentes e arquiteturas de referência

**Nenhum MCP server completo para compras públicas brasileiras existe hoje.** O projeto mais próximo é o `dutradotdev/mcp-portal-transparencia` (TypeScript/Node.js), que cobre a API do Portal da Transparência mas não é específico para compras. Outros MCP servers brasileiros como `lucianfialho/mcp-brasil-api` cobrem dados gerais (CNPJ, CEP) sem foco em procurement.

### MCP servers de referência internacional

O **`blencorp/capture-mcp-server`** para procurement americano (SAM.gov, USASpending.gov) e o **`lzinga/us-gov-open-data-mcp`** com 300+ tools e carregamento modular seletivo são as melhores arquiteturas de referência para o mcp-brasil. O `switchr24/mcp-india-tenders` (Índia, compatível OCDS) também oferece padrões relevantes.

### Clientes e scrapers existentes

| Projeto | Linguagem | Fonte integrada | Status |
|---------|-----------|-----------------|--------|
| `thiagosy/PNCP` | Python (Jupyter) | PNCP API | Atualizado jun/2024 |
| `powerandcontrol/PNCP` | Python | PNCP API (coleta paginada → Excel) | Ativo |
| `SHJordan/api-pncp-php` | PHP | PNCP API (OpenAPI-generated) | Mantido |
| `codevance/python-comprasnet` | Python | ComprasNet (scraping) | Inativo desde 2018 |
| `marcoarthur/compras-gov` | Perl | compras.dados.gov.br API | Mantido |
| `cgugovbr/comprasgovbr-crawler` | PHP/Laravel | Contratos.gov.br API (oficial CGU) | Oficial |
| `gitlab.com/comprasnet/api-comprasnet` | — | Contratos ComprasNet (oficial) | Oficial (GitLab) |
| `jfalves/licitacao_publica` | Python/Pentaho | compras.dados.gov.br → PostgreSQL | Acadêmico |
| `leopiccionia/LicitaSP` | Python (Scrapy) | SP capital (scraping) | Antigo |

---

## 8. Arquitetura recomendada para o mcp-brasil

Com base no mapeamento completo, o mcp-brasil deve ser estruturado em **módulos independentes** seguindo a arquitetura modular do `us-gov-open-data-mcp`. A priorização deve seguir esta ordem de impacto:

**Prioridade 1 — Fontes federais com API aberta (sem auth)**:
PNCP Consulta, ComprasNet Legacy, Dados Abertos Compras.gov.br, CATMAT/CATSER

**Prioridade 2 — Fontes federais com auth simples**:
Portal da Transparência (API Key), CEIS/CNEP/CEPIM/CEAF (mesma API Key), Contratos.gov.br

**Prioridade 3 — Tribunais de Contas**:
TCU (10+ endpoints), TCE-RJ, TCE-CE, TCE-SP, TCE-PE, TCE-RS

**Prioridade 4 — Fontes estaduais/municipais**:
BEC/SP, PRODEMGE MG, APILIB SP Capital, portais CKAN municipais

**Prioridade 5 — Fontes complementares**:
BNDES CKAN, BPS Saúde (CSV), CNJ DataJud

O mapa identifica **25 APIs REST ativas**, **8 portais CKAN com API**, **4 fontes apenas CSV/download** e **6 portais operacionais sem API** (Licitações-e, Petrobras, Caixa, Eletrobras, Correios, ComprasParaná). O gap mais crítico é a ausência de qualquer MCP server para o PNCP — a fonte que centraliza dados de **todas as esferas de governo** e é obrigatória pela nova lei de licitações.

---

## Conclusão

O ecossistema de dados de compras públicas do Brasil está em plena transição: o PNCP consolida progressivamente dados de todos os níveis governamentais, enquanto sistemas legados (ComprasNet, Painel de Preços) são descontinuados ou absorvidos. Para o mcp-brasil, a estratégia ótima é construir primeiro sobre o **PNCP + Portal da Transparência + ComprasNet** (cobrindo ~80% das necessidades), e então expandir modularmente para TCUs e portais estaduais. A existência de projetos como `mcp-portal-transparencia` e `capture-mcp-server` (US procurement) fornece tanto código reutilizável quanto padrões arquiteturais validados. O fato de que nenhum MCP server integra hoje as fontes de compras públicas brasileiras representa tanto o principal gap quanto a principal oportunidade do projeto.