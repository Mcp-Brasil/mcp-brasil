# Mapeamento completo de APIs públicas brasileiras para o projeto mcp-brasil

O ecossistema de APIs públicas brasileiras compreende **mais de 80 APIs documentadas** distribuídas pelos três poderes e múltiplos níveis de governo, das quais aproximadamente 45 são REST JSON abertas e prontas para integração via MCP servers. **Já existem 8 MCP servers brasileiros no GitHub** — liderados pelo `mcp-dadosbr` com 23 tools — mas lacunas significativas permanecem em áreas como Base dos Dados, CKAN dados.gov.br, SAPL/Interlegis, NFS-e Nacional e APIs ambientais. O projeto mcp-brasil pode se diferenciar com uma abordagem Python-first, modular e sem dependência de API keys pagas externas.

---

## 1. Poder Executivo Federal — o núcleo mais rico de APIs abertas

### IBGE — 6 APIs cobrindo todo o território nacional

O IBGE opera o conjunto mais maduro de APIs abertas do governo federal, todas sem autenticação e em REST JSON.

| API | URL Base | Dados | Rate Limits |
|-----|----------|-------|-------------|
| **Agregados (SIDRA)** | `servicodados.ibge.gov.br/api/v3/agregados` | Censos, PNAD, PIB, PIM — todas as pesquisas estatísticas | Não documentados |
| **Localidades** | `servicodados.ibge.gov.br/api/v1/localidades` | Regiões, UFs, municípios, distritos, subdistritos | Não documentados |
| **Malhas** | `servicodados.ibge.gov.br/api/v3/malhas` | Geometrias em GeoJSON, TopoJSON e SVG | Não documentados |
| **Nomes** | `servicodados.ibge.gov.br/api/v2/censos/nomes` | Frequência de nomes por década e UF | Não documentados |
| **SIDRA direta** | `apisidra.ibge.gov.br/values/t/{tabela}/...` | Acesso direto às tabelas do SIDRA | Não documentados |
| **Metadados** | Via Conecta gov.br | Operações estatísticas, variáveis, ODS | Tempo recuperação: 6h |

**MCP server existente:** `fmmarmello-servicodados-mcp` (cobre apenas Agregados). **Potencial:** Tools para `consultar_populacao`, `buscar_municipio`, `obter_malha_geografica`, `ranking_nomes`, `indicadores_economicos_por_municipio`.

### Banco Central — SGS + Plataforma Olinda com 18.000+ séries temporais

O BCB oferece dois sistemas complementares. O **SGS** (Sistema Gerenciador de Séries Temporais) disponibiliza **18.000+ séries** via `api.bcb.gov.br/dados/serie/bcdata.sgs.{codigo}/dados` em JSON e CSV, sem autenticação. A partir de março de 2025, séries diárias têm limite de **10 anos por consulta** e máximo de 20 registros para últimos N. As séries mais importantes incluem SELIC (432), IPCA (433), IGP-M (189), CDI (12), câmbio USD (1) e PIB (1207).

A **Plataforma Olinda** (`olinda.bcb.gov.br/olinda/servico/{Servico}/versao/v1/odata/`) expõe dados via protocolo OData em JSON, XML e CSV, com Swagger individual por serviço. Os serviços incluem **Expectativas de Mercado (Focus)** com expectativas anuais, mensais, trimestrais e Top5; **SPI (PIX)** com estatísticas diárias; **BcBase** com dados cadastrais de entidades supervisionadas; dados de instituições em funcionamento, agências, correspondentes, CADIP e dinheiro em circulação. Existe ainda a interface SOAP legada via WSDL em `www3.bcb.gov.br`.

**MCP server existente:** `sidneybissoli/bcb-br-mcp` (150+ séries catalogadas em 12 categorias). **Potencial adicional:** Tools para `expectativas_focus`, `estatisticas_pix`, `instituicoes_financeiras`.

### Portal da Transparência (CGU) — a API mais abrangente de gastos públicos

- **URL:** `portaldatransparencia.gov.br/api-de-dados/`
- **Swagger:** `api.portaldatransparencia.gov.br/swagger-ui/index.html`
- **Autenticação:** API Key gratuita (cadastro por e-mail)
- **Formato:** REST JSON — **Status:** Ativa (v1.30.0)
- **Rate limits:** Limite por minuto por chave; bloqueio temporário se excedido

Os endpoints cobrem **20+ temas**: Bolsa Família, BPC, Auxílio Emergencial, servidores públicos (dados funcionais e remuneração), despesas (empenhos, liquidações, pagamentos), contratos, licitações, convênios, emendas parlamentares, viagens a serviço, cartões de pagamento corporativo, CEAF, CEIS, CNEP, CEPIM, Garantia-Safra, PETI e Seguro Defeso.

**MCP server existente:** `dutradotdev/mcp-portal-transparencia` (geração dinâmica de tools via Swagger). Também parcialmente coberto por `mcp-dadosbr`.

### Receita Federal/SERPRO — CNPJ e CPF com múltiplas vias de acesso

A via **oficial** é a API SERPRO (`gateway.apiserpro.serpro.gov.br/consulta-cnpj-df/v2/`) que oferece três níveis de consulta CNPJ (Básica, QSA, Empresa) com OAuth2 e certificado digital, custando a partir de R$ 662/mês com trial de 3.000 consultas. Para **acesso gratuito**, existem alternativas comunitárias robustas: **BrasilAPI** (`brasilapi.com.br/api/cnpj/v1/{cnpj}`), **Minha Receita** (`minhareceita.org/{cnpj}`), **ReceitaWS** (3 consultas/min gratuitas) e **CNPJá**. Todas extraem dados dos CSVs públicos da RFB.

**Potencial MCP:** `consultar_cnpj`, `validar_empresa`, `buscar_socios`, `verificar_situacao_cadastral`.

### dados.gov.br — portal CKAN como hub de descoberta

O portal opera a API padrão CKAN v3 (`dados.gov.br/api/3/action/`) com endpoints para `package_list`, `package_search`, `package_show`, `organization_list` e `datastore_search` (quando DataStore habilitado). Sem autenticação para leitura, REST JSON. Contém **metadados e links** para milhares de datasets de todos os órgãos federais — não os dados em si, mas o catálogo para descobri-los.

**Potencial MCP:** `buscar_datasets`, `listar_organizacoes`, `consultar_recurso` — funciona como tool de descoberta de dados governamentais.

### Correios — APIs oficiais agora requerem contrato

Desde setembro de 2023, as APIs abertas dos Correios foram **descontinuadas**. A API oficial (`api.correios.com.br/`) agora requer Bearer Token via OAuth com contrato comercial, cobrindo preço, prazo, rastro, CEP e coleta. Para **consulta de CEP gratuita**, as alternativas são: **ViaCEP** (`viacep.com.br/ws/{cep}/json/`), **BrasilAPI** (`brasilapi.com.br/api/cep/v2/{cep}`) e **OpenCEP**.

### INEP — sem API REST, apenas microdados para download

O INEP **não possui API REST oficial**. Dados de ENEM, Censo Escolar, IDEB, ENADE e SAEB são disponibilizados como **arquivos CSV/ZIP** em `gov.br/inep/pt-br/acesso-a-informacao/dados-abertos`. O acesso programático é viável via **Base dos Dados** (BigQuery) ou a API comunitária **Educação Inteligente** (`educacao.dadosabertosbr.org/api`).

### DOU (Diário Oficial da União) — INLABS para download programático

O **INLABS** (`inlabs.in.gov.br`) permite download automatizado de edições do DOU desde 2020 em PDF e XML, com cadastro gratuito. Não é uma API REST convencional, mas oferece acesso programático via scripts. O **Ro-DOU** (`gestaogovbr.github.io/Ro-dou/`) é uma ferramenta de clipping que consome uma API interna do `in.gov.br`. Para **publicação** no DOU, existe o WS-INCom (SOAP XML, restrito a órgãos federais).

### Compras.gov.br e PNCP — licitações e contratos

Duas APIs complementares cobrem compras públicas. O **Compras.gov.br** (`compras.dados.gov.br/{modulo}/v1/{metodo}.{formato}`) é aberto, sem cadastro, com módulos para licitações, contratos, fornecedores, materiais (CATMAT) e serviços (CATSER) em JSON, XML e CSV. O **PNCP** (`pncp.gov.br/api/consulta/`) é mais recente (desde janeiro 2024), com Swagger documentado, cobrindo PCAs, editais, atas de registro de preço e contratos da Nova Lei de Licitações (14.133/2021).

**Potencial MCP:** `buscar_licitacoes`, `consultar_contratos`, `pesquisar_fornecedores`, `buscar_materiais`.

---

## 2. Poder Legislativo Federal — as APIs mais bem documentadas do Brasil

### Câmara dos Deputados — referência em dados abertos legislativos

- **URL:** `dadosabertos.camara.leg.br/api/v2/`
- **Swagger:** `dadosabertos.camara.leg.br/swagger/api.html`
- **Autenticação:** Aberta — **Formato:** REST JSON/XML
- **Cobertura:** Dados desde 1826 (parcial), completo desde ~1934
- **Status:** Ativa, atualização diária, **a melhor API legislativa do Brasil**

A API expõe **40+ endpoints** organizados em: **deputados** (perfil, despesas CEAP desde 2008, discursos, eventos, órgãos, frentes, ocupações); **proposições** (projetos de lei, PECs, MPVs com autores, tramitações, temas, votações, relacionadas); **votações** (nominais e simbólicas com votos individuais e orientações de bancada); **eventos** (seminários, sessões, audiências com pautas e participantes); **órgãos** (comissões, CPIs, Mesa Diretora com membros e votações); **partidos** (membros, líderes); **blocos** e **frentes parlamentares**; **legislaturas** (desde a 1ª, com mesa diretora). Também disponibiliza **arquivos bulk** para download em CSV, JSON, XML, XLSX e ODS.

**Potencial MCP:** `buscar_deputado`, `buscar_proposicao`, `buscar_votacao`, `monitorar_tramitacao`, `gastos_parlamentares`, `agenda_legislativa`.

### Senado Federal — modernizado em 2025 com OpenAPI

- **URL:** `legis.senado.leg.br/dadosabertos/`
- **Swagger (novo):** `legis.senado.leg.br/dadosabertos/api-docs/swagger-ui/index.html`
- **Autenticação:** Aberta — **Formato:** REST, padrão XML com suporte JSON e CSV
- **Status:** Ativa, recentemente modernizada (2025)

Cobre senadores (em exercício, por legislatura, mandatos, filiações, votações, lideranças), matérias/proposições (busca, tramitação, autores, relatorias, emendas, textos), votações (plenário, votos nominais, orientações), comissões (permanentes, temporárias, reuniões), plenário (sessões, pautas, presenças), legislação (normas por código/tipo/número/ano), orçamento (emendas parlamentares) e blocos parlamentares. O Catálogo de Dados Abertos em `senado.leg.br/dados-abertos/catalogo-de-dados-abertos` inclui dados administrativos como remuneração e licitações.

**MCP server existente:** `SidneyBissoli/senado-br-mcp` (33 tools). **Potencial adicional:** Integração cruzada com API da Câmara para visão completa do Congresso.

### SAPL/Interlegis — centenas de câmaras municipais com API padronizada

O SAPL (Sistema de Apoio ao Processo Legislativo) é um sistema Django mantido pelo Programa Interlegis do Senado (`github.com/interlegis/sapl`, 91 stars) usado por **centenas de câmaras municipais e assembleias legislativas**. Cada instância expõe automaticamente uma **API REST JSON** em `{instância}/api/` com Swagger auto-gerado em `{instância}/api/schema/swagger-ui/`.

Os endpoints cobrem: **parlamentares** (vereadores/deputados estaduais, mandatos, filiações), **matérias** (projetos de lei, tramitações, documentos acessórios, anexadas), **normas** (leis, decretos, resoluções aprovadas), **sessões** (plenárias, registros de votação, ordem do dia), **comissões** e **protocolo administrativo**. Instâncias confirmadas incluem Jataí-GO, Acre (assembleia), Piauí (assembleia) e dezenas de outras.

**Potencial MCP (altíssimo):** Um único adapter genérico que aceite a URL da instância SAPL como parâmetro permitiria consultas em **centenas de câmaras simultaneamente** — `buscar_materia_municipal(instancia, query)`, `buscar_vereador(instancia, nome)`, `buscar_norma_municipal(instancia, tipo, ano)`.

### LexML — busca transversal de legislação em todos os níveis

- **URL:** `lexml.gov.br/busca/SRU` (protocolo SRU/CQL)
- **Autenticação:** Aberta — **Formato:** SRU/XML
- **Cobertura:** Legislação, jurisprudência, proposições e doutrina de todos os níveis (federal, estadual, municipal) desde 1556

Suporta pesquisa por URN (`urn:lex:br:...`), data, tipo de documento, localidade e autoridade. Protocolo OAI-PMH disponível para coleta de metadados. **Wrapper Python:** `py-lexml-acervo` (`github.com/netoferraz/py-lexml-acervo`).

### Assembleias Legislativas Estaduais

A **ALMG** (Minas Gerais) possui a melhor API estadual: `dadosabertos.almg.gov.br/api/v2/` em REST JSON/XML, aberta, com rate limit de **máximo 2 requisições simultâneas e mínimo 1 segundo entre requests**. Cobre deputados, comissões, proposições, pronunciamentos, verbas e legislação mineira. A **ALESP** (São Paulo) oferece dados abertos em `al.sp.gov.br/dados-abertos/catalogo` mas principalmente como downloads XML, não API REST. A **ALERJ** (Rio de Janeiro) **não possui API pública documentada**.

---

## 3. Poder Judiciário — DataJud como porta de entrada unificada

### DataJud (CNJ) — a API mais abrangente do Judiciário brasileiro

- **URL:** `api-publica.datajud.cnj.jus.br/`
- **Documentação:** `datajud-wiki.cnj.jus.br/api-publica/`
- **Autenticação:** API Key pública (fornecida na Wiki do CNJ)
- **Formato:** REST JSON baseado em **Elasticsearch** (POST `_search` com query DSL)
- **Status:** Ativa desde 2023

O DataJud é revolucionário por cobrir **~90 tribunais em um único sistema**: STJ, TRFs 1-6, TRTs 1-24, todos os TJs estaduais, TREs, TST e STM. Cada tribunal tem seu endpoint no padrão `api_publica_{tribunal}/_search`. Os dados incluem metadados de processos (capas processuais), movimentações, classes, assuntos e órgãos julgadores. A paginação usa `search_after` do Elasticsearch.

**MCP server parcialmente existente:** `mcp-dadosbr` inclui DataJud. **Potencial:** `consultar_processo(numero, tribunal)`, `buscar_processos_por_classe`, `buscar_por_orgao_julgador`, `obter_movimentacoes`, `estatisticas_tribunal`.

### TSE — ecossistema triplo para dados eleitorais

O TSE opera três sistemas complementares:

1. **Portal CKAN** (`dadosabertos.tse.jus.br/api/3/`) — datasets eleitorais desde 1945: candidatos, resultados, prestação de contas, eleitorado, boletins de urna, partidos, filiação partidária. REST JSON, aberto.

2. **DivulgaCandContas** (`divulgacandcontas.tse.jus.br/divulga/rest/`) — dados de candidaturas em tempo real: registros, fotos, propostas de governo, situação. REST JSON, aberto mas **não oficialmente documentado** (documentação comunitária em `github.com/augusto-herrmann/divulgacandcontas-doc`).

3. **CDN de Resultados** (`resultados.tse.jus.br`) — resultados em tempo real durante eleições via JSON estático servido por CDN. Ativo apenas durante janela eleitoral.

**API acadêmica complementar:** CEPESP-FGV (`github.com/Cepesp-Fgv/cepesp-rest`) oferece dados eleitorais pós-processados desde 1945 em REST JSON aberto.

### STJ e STF — portais CKAN e interfaces web

O **STJ** opera um portal CKAN (`dadosabertos.web.stj.jus.br/api/3/`) com 14 datasets incluindo movimentação processual, acórdãos, precedentes qualificados e sessões de julgamento. REST JSON, aberto.

O **STF não possui API pública REST documentada**. A pesquisa de jurisprudência em `jurisprudencia.stf.jus.br` usa uma API interna não documentada para uso externo. Dados do STF estão parcialmente acessíveis via DataJud (embora o STF esteja **ausente da lista oficial de aliases** da API pública).

### PJe/MNI — acesso profundo via SOAP complexo

O **Modelo Nacional de Interoperabilidade** (MNI) expõe dados processuais completos via **SOAP XML** em `pje.{tribunal}.jus.br/{instancia}/intercomunicacao?wsdl`. Operações incluem `consultarProcesso`, `consultarAvisosPendentes`, `entregarManifestacaoProcessual` e `consultarTeorComunicacao`. Requer CPF + senha no PJe ou certificado digital. A disponibilidade **varia significativamente** entre tribunais.

### Tribunais regionais com APIs próprias

Alguns tribunais mantêm APIs independentes: **TJPR** (`portal.tjpr.jus.br`) com endpoints para atos normativos, magistratura, plantões e PROJUDI (SOAP); **TJDFT** com API de jurisprudência em REST JSON; **TRT4** com APIs para unidades, sessões e licitações. O **BNMP 3.0** (Banco Nacional de Medidas Penais e Prisões) do CNJ opera em `portalbnmp.cnj.jus.br` com consulta pública limitada, acesso completo restrito a magistrados e servidores.

---

## 4. APIs temáticas — saúde, meio ambiente, trabalho e fiscal

### Saúde: OpenDataSUS e o legado DATASUS

O **OpenDataSUS** (`opendatasus.saude.gov.br/api/3/`) opera como portal CKAN com datasets sobre vacinação COVID-19, SINAN, SRAG e vigilância epidemiológica, aberto e em REST JSON. Os sistemas legados do DATASUS (SIH, SIM, SINASC, SI-PNI, SIA, SINAN, CNES) disponibilizam dados via **FTP público** em formato DBC proprietário, convertível via **PySUS** (`pip install pysus` — a principal biblioteca Python para dados do SUS). O **TABNET** (`tabnet.datasus.gov.br`) oferece tabulação cruzada em interface web sem API REST. A **RNDS** (Rede Nacional de Dados em Saúde) segue padrão HL7 FHIR mas requer certificado digital ICP-Brasil.

### Meio ambiente: INPE, IBAMA e ANA

O **INPE** opera o **TerraBrasilis** (`terrabrasilis.dpi.inpe.br`) com dados de desmatamento (PRODES anual, DETER alertas diários) via GeoServer WMS/WFS em GeoJSON e Shapefiles, aberto. O **BDQueimadas** fornece focos de queimadas atualizados a cada 3 horas, com séries desde 1998. O portal **data.inpe.br** oferece imagens de satélite via STAC (experimental).

O **IBAMA** mantém portal CKAN (`dadosabertos.ibama.gov.br/api/3/`) com autos de infração, embargos, apreensões e dados do CAR, aberto. A **ANA** (Agência Nacional de Águas) opera o **HidroWeb** (`snirh.gov.br/hidroweb/rest/api/`) com dados hidrológicos (vazão, nível, chuva) de estações fluviométricas e pluviométricas, REST JSON, aberto, além do SAR com monitoramento de **713+ reservatórios**.

### Trabalho e emprego: microdados sem API REST

RAIS e CAGED/Novo CAGED são disponibilizados como **microdados em TXT** via FTP (`ftp.mtps.gov.br/pdet/microdados/`). Dados anonimizados são abertos; identificados requerem acordo de cooperação com o MTE. O **CNIS não possui API pública** — acesso apenas via Meu INSS ou Conecta gov.br. A melhor alternativa para acesso programático é a **Base dos Dados** (`basedosdados.org`), que disponibiliza RAIS, CAGED e dezenas de outros datasets tratados via BigQuery com SDK Python.

O **INSS** opera portal CKAN (`dadosabertos.inss.gov.br/api/3/`) com estatísticas de benefícios concedidos/indeferidos/emitidos e acidentes de trabalho. A API de Benefícios Previdenciários via Conecta gov.br é **restrita a órgãos governamentais**.

### Fiscal: eSocial e NF-e em SOAP XML com certificado digital

O **eSocial** opera via **SOAP 1.1 XML** (`webservices.esocial.gov.br`) com certificado digital ICP-Brasil obrigatório. A **NF-e** usa web services SEFAZ por UF autorizadora, também SOAP XML com mTLS. A **NFS-e Nacional** (`nfse.gov.br`) é mais moderna (JSON nas rotas, XML para documentos fiscais) e **obrigatória desde janeiro 2026** para todos os municípios. A via SERPRO oferece consulta NF-e em REST JSON mas é **paga** (a partir de R$ 662/mês).

### Trânsito: WSDenatran restrito

O **WSDenatran** (`wsdenatran.estaleiro.serpro.gov.br`) oferece 52 tipos de consulta (RENAVAM, RENACH, RENAINF) em REST JSON via SERPRO, mas requer certificado digital + termo de autorização do DENATRAN + contrato comercial.

---

## 5. Plataformas agregadoras — BrasilAPI, Querido Diário e Base dos Dados

### BrasilAPI — a API mais versátil para desenvolvedores

- **URL:** `brasilapi.com.br/api/` — **GitHub:** 9.9k stars, 677 forks
- **Autenticação:** Nenhuma — **Formato:** REST JSON
- **Rate limits:** Uso razoável; sem crawling

Agrega **16+ endpoints**: CEP (v1/v2 com geolocalização), CNPJ, DDD, bancos, feriados, FIPE (marcas/preços/tabelas), IBGE (municípios/estados), ISBN, NCM, CPTEC (previsão do tempo), Registro.br, taxas de juros, corretoras CVM, câmbio e participantes PIX. É a API mais usada por desenvolvedores brasileiros.

### Querido Diário — diários oficiais municipais

- **API:** `api.queridodiario.ok.org.br/` — **GitHub:** 1.3k stars
- **Python wrapper:** `pip install querido_diario`
- **Autenticação:** Nenhuma — **Formato:** REST JSON
- **Cobertura:** Raspagem de diários oficiais de 5.570 municípios (cobertura parcial)

Endpoint principal: `GET /gazettes?territory_ids={IBGE_CODE}&querystring={KEYWORDS}`. Mantido pela Open Knowledge Brasil (OKBR), com ecossistema de scrapers Scrapy, API FastAPI e frontend dedicado.

### Base dos Dados — BigQuery com centenas de datasets tratados

- **URL:** `basedosdados.org` — **Python SDK:** `pip install basedosdados`
- **API GraphQL:** `api.basedosdados.org/api/v1/graphql`
- **Autenticação:** Google Cloud project (free tier 1TB/mês)

Datalake público com **centenas de datasets brasileiros padronizados** incluindo PIB, população, mortalidade, RAIS, CAGED, dados do INEP e dezenas de outros. Acesso via SQL direto no BigQuery ou via SDK Python (`bd.read_sql(query)` / `bd.read_table('br_ibge_pib', 'municipio')`).

### Conecta gov.br — gateway federal restrito

O Conecta (`gov.br/conecta/catalogo/`) opera como gateway OAuth 2.0 do SERPRO para interoperabilidade entre órgãos federais. Catálogo com **100+ APIs** (CPF Light, SIAPE, Cadastro Base do Cidadão, WSDenatran, benefícios previdenciários). **Acesso restrito exclusivamente a órgãos da Administração Pública Federal** — não viável para MCP servers públicos.

---

## 6. MCP servers brasileiros já existentes — 8 projetos mapeados

| Servidor | Linguagem | Escopo | Tools |
|----------|-----------|--------|-------|
| **mcp-dadosbr** (cristianoaredes) | TypeScript | OSINT completo — governo, jurídico, financeiro, saúde | **23** |
| **senado-br-mcp** (SidneyBissoli) | TypeScript | Senado Federal completo | **33** |
| **brasil-api-mcp-server** (mauricio-cantu) | TypeScript | Todos endpoints BrasilAPI | ~16 |
| **brasil-api-mcp** (guilhermelirio) | TypeScript | BrasilAPI parcial | ~6 |
| **mcp-brasil-api** (lucianfialho) | **Python** | BrasilAPI parcial | 6 |
| **brlaw_mcp_server** (pdmtt) | **Python** | Precedentes STJ/TST | 3 |
| **bcb-br-mcp** (SidneyBissoli) | TypeScript | BCB séries temporais | ~10 |
| **agrobr-mcp** (bruno-portfolio) | **Python** | Dados agrícolas (CEPEA, CONAB, INPE) | ~5 |

O `mcp-dadosbr` é o mais ambicioso mas **requer API keys pagas** (Tavily, Perplexity) além das governamentais. O `senado-br-mcp` é notavelmente completo com 33 tools dedicados ao Senado. A maioria é TypeScript — apenas 3 projetos são Python.

### Python SDKs existentes para APIs brasileiras

- **brazilcep** (v7.0.1) — CEP via múltiplos provedores, suporta async
- **PySUS** — Principal biblioteca para dados DATASUS (SIH, SIM, SINASC, PNI, CNES)
- **basedosdados** (v2.0.2) — SDK para BigQuery da Base dos Dados
- **validate-docbr** (484 stars) — Validação de CPF, CNPJ, CNH, CNS, RENAVAM
- **querido_diario** — Wrapper para API Querido Diário
- **py-lexml-acervo** — Consulta ao LexML
- **tapioca-jarbas** — Wrapper para API Jarbas/Serenata de Amor

---

## 7. Análise estratégica para o projeto mcp-brasil

### APIs com maior potencial de impacto para MCP servers

As APIs que combinam **acesso aberto, boa documentação, dados ricos e alta demanda** são, em ordem de prioridade:

1. **Câmara dos Deputados** — A melhor API do governo brasileiro, REST JSON, 40+ endpoints, sem auth. Não existe MCP server dedicado.
2. **DataJud/CNJ** — Cobertura de ~90 tribunais via Elasticsearch. Parcialmente coberto por `mcp-dadosbr` mas merece server dedicado.
3. **SAPL/Interlegis** — Um único adapter genérico para centenas de câmaras municipais. **Nenhum MCP server existente**.
4. **TSE (CKAN + DivulgaCandContas)** — Dados eleitorais desde 1945. **Nenhum MCP server existente**.
5. **Base dos Dados** — Acesso unificado a centenas de datasets via BigQuery. **Nenhum MCP server existente**.
6. **PNCP** — Compras públicas sob Nova Lei de Licitações. **Nenhum MCP server dedicado**.
7. **OpenDataSUS** — Dados de saúde pública. **Nenhum MCP server dedicado**.
8. **INPE/TerraBrasilis** — Desmatamento e queimadas. Parcialmente coberto por `agrobr-mcp`.
9. **ANA/HidroWeb** — Dados hidrológicos. **Nenhum MCP server existente**.
10. **Querido Diário** — Diários oficiais municipais com busca textual. **Nenhum MCP server dedicado**.

### Lacunas significativas no ecossistema atual

Nenhum MCP server existente cobre: **SAPL/Interlegis** (legislativo municipal), **TSE** (dados eleitorais), **Base dos Dados** (BigQuery), **PNCP** (licitações), **OpenDataSUS** (saúde), **ANA** (hidrologia), **INPE** (desmatamento), **ALMG** (legislativo estadual) ou **Câmara dos Deputados** como server dedicado. Estas representam as oportunidades mais claras para o mcp-brasil.

### Limitações conhecidas por categoria

- **SOAP XML** (eSocial, NF-e, PJe/MNI, WSDenatran): Requerem certificado digital ICP-Brasil, assinatura XML e parsing complexo — alto custo de implementação
- **FTP/downloads** (DATASUS DBC, RAIS/CAGED TXT, INEP CSV): Sem API REST; requerem download + processamento local
- **Acesso restrito** (Conecta gov.br, SERPRO APIs pagas, BNMP, SISBAJUD): Limitados a órgãos governamentais ou contratos comerciais
- **Rate limits rígidos** (ALMG: 2 req simultâneas + 1s entre requests; BCB SGS: 10 anos por consulta; Portal da Transparência: limite por minuto)
- **Documentação precária** (STF: sem API pública; DivulgaCandContas TSE: não oficial; muitos TJs: apenas interface web)

### Diferenciação estratégica para mcp-brasil

O projeto pode se diferenciar de `mcp-dadosbr` e demais com quatro eixos: **Python-first** (alinhado com o ecossistema de IA); **arquitetura modular** com plugins por órgão; **zero dependência de API keys pagas externas** (usar apenas keys gratuitas como Portal da Transparência e DataJud); e **cobertura de lacunas** nos legislativos municipal/estadual (SAPL), judiciário (DataJud dedicado), eleições (TSE) e dados unificados (Base dos Dados). Uma arquitetura em que cada poder/órgão é um módulo independente — instalável separadamente — maximizaria a adoção pela comunidade.

## Conclusão

O ecossistema de APIs públicas brasileiras é surpreendentemente robusto, com destaque para a **Câmara dos Deputados** (melhor documentação), o **DataJud/CNJ** (maior cobertura unificada do Judiciário), o **BCB** (18.000+ séries temporais) e o **Portal da Transparência** (20+ temas de gastos). As maiores oportunidades para o mcp-brasil residem nas APIs que combinam alta qualidade técnica com ausência de MCP servers dedicados — notavelmente SAPL/Interlegis (centenas de câmaras), TSE (dados eleitorais), Base dos Dados (BigQuery), PNCP (licitações) e OpenDataSUS (saúde). A estratégia mais eficaz é uma arquitetura modular Python que cubra sistematicamente estas lacunas, começando pelas APIs de maior impacto e menor barreira de acesso (abertas, REST JSON, bem documentadas), para depois expandir para integrações mais complexas envolvendo SOAP, certificados digitais e processamento de microdados.