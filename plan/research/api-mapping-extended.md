# Mapeamento exaustivo de APIs públicas brasileiras para o projeto mcp-brasil

**Foram identificadas 90+ fontes adicionais de dados públicos brasileiros** que não constam no mapeamento atual do projeto mcp-brasil. A pesquisa cobre 17 setores temáticos e revela um ecossistema rico porém fragmentado: apenas ~25% das fontes oferecem APIs REST verdadeiras, enquanto a maioria opera via portais CKAN, downloads CSV ou interfaces web. As fontes com maior potencial imediato de integração são **ONS** (energia, API REST + AWS S3), **SICONFI/Tesouro** (finanças públicas, REST JSON sem auth), **Comex Stat** (comércio exterior, REST JSON sem auth), **SALIC/Lei Rouanet** (cultura, REST JSON sem auth), **IpeaData** (séries macro, OData v4), **Embrapa AgroAPI** (agricultura, REST com token freemium) e **Atlas da Violência** (segurança, OData REST).

---

## 1. Agências reguladoras — oito portais, quatro com APIs CKAN

As oito principais agências reguladoras publicam dados abertos, mas em níveis de maturidade muito distintos. **ANEEL** e **ANTT** se destacam com portais CKAN maduros e APIs com suporte a SQL.

| Agência | Portal | API | Datasets | Auth | Formato |
|---------|--------|-----|----------|------|---------|
| **ANEEL** | dadosabertos.aneel.gov.br | ✅ CKAN + SQL | 69 | Aberto | CSV, JSON, XML |
| **ANTT** | dados.antt.gov.br | ✅ CKAN + SQL | 105 | Aberto | CSV, JSON, KMZ, SHP |
| **ANTAQ** | dadosabertos.antaq.gov.br | ✅ CKAN | 6 | Aberto | CSV, TXT |
| **ANVISA** | api.anvisa.gov.br | ✅ REST dedicada | Vários | Bearer Token | JSON |
| **ANATEL** | gov.br/anatel/dados-abertos | ❌ Somente downloads | ~50+ | Aberto | CSV, XLS |
| **ANAC** | gov.br/anac/dados-abertos | ❌ Somente downloads | ~40+ | Aberto | CSV |
| **ANP** | gov.br/anp/dados-abertos | ❌ Somente downloads | ~30+ | Aberto | CSV, XLS |
| **ANS** | dadosabertos.ans.gov.br/FTP | ❌ FTP/HTTP | 30+ dirs | Aberto | CSV, DBC |

**ANEEL** (`dadosabertos.aneel.gov.br/api/3/action/`) permite consultas via `datastore_search` e `datastore_search_sql` com dados de **geração distribuída** (solar, eólica), **tarifas por distribuidora**, usinas do SIGA, fiscalização e reclamações. A **ANTT** cobre **105 datasets** incluindo RNTRC (transportadores de carga), volume de tráfego em praças de pedágio, rodovias concedidas e dados ferroviários. A **ANVISA** é a única com API REST dedicada em `api.anvisa.gov.br`, mas exige cadastro e Bearer Token para consultar registros de medicamentos, dispositivos médicos e cosméticos.

**ANATEL** publica dados de cobertura 4G/5G, reclamações por operadora, acessos de banda larga e ERBs em CSV — sem API. **ANP** tem dados semanais de preços de combustíveis em todos os postos do país. **ANAC** oferece VRA (atrasos/cancelamentos), tarifas aéreas e RAB (registro de aeronaves). **ANS** disponibiliza 30+ diretórios via FTP com beneficiários, reclamações, IDSS e faixas de preço de planos de saúde.

SDKs existentes: `ckanapi` (Python) funciona nativamente com ANEEL, ANTT e ANTAQ. `basedosdados` oferece datasets tratados de ANATEL, ANEEL, ANAC e ANP via BigQuery. `prf-api` no PyPI para dados da PRF. Wrappers ANVISA foram descontinuados após exigência de autenticação.

Perguntas típicas para AI agents: "Qual a tarifa de energia na distribuidora X?", "Preço médio da gasolina em SP esta semana?", "Cobertura 5G no município Y?", "Operadoras com mais reclamações na ANS?"

---

## 2. Mercado financeiro e Tesouro — CVM e SICONFI lideram em maturidade

O setor financeiro oferece algumas das fontes mais ricas para AI agents, com destaque para três APIs totalmente abertas e sem autenticação.

**CVM** (`dados.cvm.gov.br`) opera em CKAN com catálogo extenso de fundos de investimento (cadastro, informe diário com PL/cota/captação, composição de carteira), companhias abertas (FCA, FRE, DFP, ITR, IPE) e ofertas de distribuição. Acesso aberto, CSV/JSON, sem auth. Potencial excepcional para análise de performance de fundos, scoring de risco e NLP em documentos societários.

**SICONFI — Tesouro Nacional** (`apidatalake.tesouro.gov.br/docs/siconfi/`) é a API mais poderosa para finanças públicas: retorna dados fiscais de **5.500+ municípios, 27 estados e União** em JSON puro, sem autenticação. Cobre RREO, RGF, Matriz de Saldos Contábeis, CAPAG e operações de crédito. SDK em R: `rsiconfi`; Python: `siconfi_api`.

**BNDES** (`dadosabertos.bndes.gov.br`) usa CKAN com dados de todas as operações de financiamento, desembolsos por setor/região/porte, e aplicação de recursos do Tesouro. GitHub oficial com exemplos Python/R.

**Tesouro Direto** tem dados de taxas, preços e volumes via CKAN do Tesouro Transparente (CSV). **ANBIMA** (`developers.anbima.com.br`) oferece APIs de títulos públicos e curvas de juros (requer cadastro). **brapi.dev** agrega cotações B3, dividendos, balanços e indicadores em JSON REST freemium. **Dados de Mercado** (`dadosdemercado.com.br/api/docs`) oferece bolsa, títulos e macro em REST freemium.

**B3** (`developers.b3.com.br`) tem APIs oficiais, mas acesso é **B2B pago** com contrato formal — inviável para MCP aberto. Alternativas comunitárias como brapi.dev e `b3-api-dados-historicos` (GitHub) suprem essa lacuna.

---

## 3. Comércio exterior e tributário — Comex Stat é destaque absoluto

**Comex Stat** (`api-comexstat.mdic.gov.br/docs`) é a API REST mais completa para comércio exterior brasileiro: exportações e importações por NCM, país, UF, município, bloco econômico e via de transporte, sem autenticação, em JSON. Swagger UI disponível. SDK R: `ComexstatR`. AliceWeb foi descontinuado e migrado para cá. SISCOMEX não tem dados abertos.

No lado tributário, **Receita Federal** (`gov.br/receitafederal/dados-abertos`) publica séries de arrecadação por estado/município/CNAE, bases completas do CNPJ em CSV bulk, CAEPF, CNO, CAFIR, estudos de carga tributária, IRPF (grandes números) e mercadorias apreendidas — tudo em download sem auth. **PGFN** (`gov.br/pgfn/dados-abertos`) disponibiliza trimestralmente a base completa de **dívida ativa da União** em CSV: identificação do devedor, valor, situação e origem. Potencial altíssimo para scoring de crédito. **Simples Nacional** não tem API oficial (apenas consulta web com captcha) — APIs de terceiros (SintegraWS, CNPJá) são pagas.

---

## 4. Educação, trabalho e previdência — CAPES e FNDE com CKAN, Lattes segue fechado

**CAPES** (`dadosabertos.capes.gov.br`) opera portal CKAN com API funcional: produção intelectual de programas de pós-graduação, teses e dissertações, dados da UAB e execução orçamentária. **FNDE** (`fnde.gov.br/dadosabertos`) também em CKAN: repasses do PNAE (alimentação escolar), FUNDEB, PDDE, PNLD (livros didáticos) e PNATE (transporte).

**MEC** publica no `dadosabertos.mec.gov.br` dados em CSV de **e-MEC** (IES e cursos), **SISU** (notas de corte por edição), **PROUNI** (bolsas concedidas com perfil do beneficiário) e **FIES** (contratos e valores). Nenhum desses tem API REST — apenas bulk CSV. Wrapper comunitário `e-MEC-API` existe em PHP.

**CNPq/Lattes** é a fonte mais restritiva: **não existe API pública** para os 7M+ currículos. O Extrator Lattes requer credenciamento institucional formal. Wrappers comunitários (`scriptLattes`, `getLattes`) fazem scraping.

No setor trabalho, **MTE** publica dados de seguro-desemprego por UF/município/CNAE, lista de trabalho escravo (CSV) e CBO (2.600+ ocupações). **SINE/Emprega Brasil não tem API pública** de vagas. **SmartLab** (`smartlabbr.org`) da MPT/OIT é uma joia subutilizada: Flask REST API open source no GitHub (`datahub-api`) agregando dados de trabalho escravo, segurança do trabalho, trabalho infantil e diversidade.

**DATAPREV** (`dadosabertos.dataprev.gov.br`) tem portal CKAN com AEPS (Anuário Estatístico de Previdência) e AEAT (acidentes do trabalho). APIs ricas de benefícios previdenciários e relações trabalhistas existem via Conecta gov.br, mas são **restritas a órgãos federais**.

---

## 5. Segurança, defesa e social — Atlas da Violência com API REST exemplar

**Atlas da Violência** (`ipea.gov.br/atlasviolencia/api`) do IPEA/FBSP oferece API REST OData v4 sem autenticação: séries históricas de homicídios, violência por perfil demográfico e geográfico, com endpoints para fontes, indicadores, temas e valores por série/abrangência (país, região, UF, município).

**PRF** publica um dos conjuntos mais ricos do governo: acidentes em rodovias federais desde 2007 em CSV com data, hora, UF, BR, km, tipo, causa, vítimas. SDK Python: `prf-api` (PyPI). **SINESP/MJSP** (`dados.mj.gov.br`) opera CKAN com 28 indicadores criminais por município/UF desde 2015. **DEPEN/SISDEPEN** tem microdados do censo penitenciário (755 mil+ presos, perfil completo). **Polícia Federal** publica estatísticas de inquéritos, crimes cibernéticos e fluxos migratórios em CSV e painéis BI.

No setor social, **Censo SUAS** disponibiliza microdados abertos anuais de CRAS, CREAS, Centros POP e gestão municipal. **CadÚnico** tem dados agregados públicos via CECAD e VIS DATA (painéis Qlik Sense), mas microdados identificados exigem cessão formal. **Bolsa Família** tem dados extras via VIS DATA 3 do MDS além do Portal da Transparência.

---

## 6. Agricultura e energia — Embrapa AgroAPI e ONS são referência

**Embrapa AgroAPI** (`agroapi.cnptia.embrapa.br/store/`) é o ecossistema de API mais maduro da agricultura brasileira: **8 APIs REST/JSON com OAuth2**, Swagger documentado, modelo freemium (90 dias grátis, 3.000 req). Inclui Agritec (zoneamento de risco climático, cultivares aptas para 12 culturas, previsão de produtividade), AGROFIT (agrotóxicos registrados), SATVeg (séries NDVI/EVI via satélite), ClimAPI (dados agrometeorológicos), BovTrace (rastreabilidade bovina) e Bioinsumos.

**MAPA** (`dados.agricultura.gov.br`) opera CKAN com Agrostat (comércio exterior do agro desde 1997), AGROFIT, certificação orgânica e SISLEGIS. **CONAB** publica preços de 130+ produtos agrícolas, safras e custos de produção, mas **não tem portal CKAN nem API** — dados apenas via interface web e XLS/PDF. **SFB/SICAR** oferece shapefiles de 7M+ imóveis rurais cadastrados no CAR.

**ONS** (`dados.ons.org.br`) é excepcional: portal CKAN + APIs REST customizadas (`apicarga.ons.org.br`) + dados no **AWS S3** (Open Data Registry). **60+ datasets** com carga de energia semi-horária, geração por fonte (hídrica/térmica/eólica/solar), intercâmbio entre subsistemas, CMO, reservatórios, fator de capacidade e indicadores de confiabilidade. Séries desde ~2000. Comunidade ativa no GitHub com notebooks Python.

**CCEE** (`dadosabertos.ccee.org.br`) lançou portal CKAN em julho/2023 com **132 datasets**: PLD horário, geração por usina, consumo por classe, contratos, leilões e encargos. **EPE** publica o Balanço Energético Nacional (séries desde 1970) em XLSX tidyverse. **MME** (`dadosabertos.mme.gov.br`) tem CKAN com SIE Brasil (oferta/demanda municipal) e Luz Para Todos.

---

## 7. Infraestrutura e transportes — DNIT e SENATRAN com bases extensas

**DNIT** (`servicos.dnit.gov.br/dadosabertos/`) opera CKAN com SNV (cadastro completo da malha rodoviária federal), VGeo (WMS/WFS geoespacial) e PNCT (dados contínuos de tráfego por segmento). **EPL/INFRA S.A.** mantém o ONTL com painéis de movimentação multimodal de cargas, Anuário Estatístico de Transportes e cenários PNL 2035/2050.

**SENATRAN** (`dados.transportes.gov.br`) publica frota de veículos mensal por UF/município/tipo em CSV (100M+ veículos). Painel SERPRO com dados detalhados por marca/modelo/combustível é pago. **INFRAERO** oferece movimentação mensal de ~66 aeroportos em XLS, mas ANAC tem dados mais abrangentes incluindo aeroportos concedidos.

**ANM/CPRM** (`geo.anm.gov.br`) opera ArcGIS REST Services para processos minerários georreferenciados, com shapefiles por UF em `app.anm.gov.br/dadosabertos/SIGMINE/PROCESSOS_MINERARIOS/{UF}.zip`.

---

## 8. Plataformas de dados e legislação — IpeaData e TCU com APIs prontas

**IpeaData** (`ipeadata.gov.br/api/odata4/`) oferece milhares de séries macroeconômicas e sociais via **OData v4 sem autenticação**. SDKs maduros: `ipeadatapy` (Python/PyPI), `ipeadatar` (R/CRAN). Entidades: Metadados, Valores, Territórios, Temas.

**TCU** (`dados-abertos.apps.tcu.gov.br`) tem APIs REST para acórdãos, certidões APF (consulta por CNPJ), atos normativos e jurisprudência. Sem auth. **CGU** (`portaldatransparencia.gov.br/api-de-dados`) complementa o Portal da Transparência com API Swagger para CEIS (empresas inidôneas), CNEP (punidas), CEPIM, CEAF, contratos e licitações — requer token por email.

**SALIC/Lei Rouanet** (`api.salic.cultura.gov.br/api/v1/`) é a API mais madura do setor cultural: REST JSON sem auth, endpoints para projetos, proponentes, incentivadores e fornecedores desde 1991. Documentação Swagger completa.

**MapBiomas** publica cobertura do solo (1985–2024) via Google Earth Engine e API GraphQL para alertas de desmatamento. **Brasil.io** (`api.brasil.io/v1/`) oferece datasets "libertados" em JSON (requer token gratuito). **Jarbas/Serenata** tem REST sem auth para reembolsos parlamentares com flag de suspeita por IA. SDK: `tapioca-jarbas` (Python).

**Imprensa Nacional/DOU** publica base mensal em XML no dados.gov.br (desde 1990). Ferramenta Python `Ro-DOU` (GitHub: gestaogovbr) automatiza pesquisas. **Portal da Legislação do Planalto** não tem API oficial — apenas crawlers comunitários.

**CNES** (`apidadosabertos.saude.gov.br/v1/`) tem API REST para 400 mil+ estabelecimentos de saúde (tipo, regime, serviços, profissionais, leitos). Complementa DATASUS.

---

## 9. Portais geoespaciais — INDE centraliza, GeoSampa e SICAR oferecem WMS/WFS

**INDE** (`inde.gov.br`) é o ponto central que agrega geosserviços WMS/WFS/WCS de dezenas de instituições. Catálogo CSW para busca de metadados. SDK: `OWSLib` (Python). **CAR/SICAR** (`car.gov.br`) tem WMS/WFS públicos em `geoserver.car.gov.br` com 6M+ imóveis rurais, APPs e Reserva Legal. SDK não-oficial: `SICAR` (Python).

**INCRA** oferece Acervo Fundiário com shapefiles de parcelas certificadas (SIGEF), assentamentos e territórios quilombolas. WMS/WFS disponíveis, mas requerem login GOV.BR para download. API SIGEF GEO via Conecta é restrita a órgãos.

---

## 10. Portais estaduais — São Paulo e Ceará com APIs REST, demais em CKAN

| Estado | Portal | API | Destaque |
|--------|--------|-----|----------|
| **São Paulo** | dadosabertos.sp.gov.br | ✅ SEADE API REST JSON | 200+ variáveis para 645 municípios |
| **Ceará** | api-dados-abertos.cearatransparente.ce.gov.br | ✅ REST + TCE-CE API | Dados de todos os municípios |
| **Minas Gerais** | dados.mg.gov.br | ✅ CKAN (Frictionless Data) | SIAFI, SISOR, saúde |
| **Pernambuco** | dados.pe.gov.br | ✅ CKAN + TCE-PE API | SAGRES, fiscais granulares |
| **Rio Grande do Sul** | dados.rs.gov.br | ✅ CKAN | FEPAM (BCRS25 1:25.000) |
| **Rio de Janeiro** | dadosabertos.rj.gov.br | ✅ CKAN | ISP-RJ: criminalidade desde 2003 |
| **Paraná** | dados.pr.gov.br | Portal próprio | IPARDES: 15M+ dados municipais |
| **Bahia** | dados.ba.gov.br | ✅ CKAN (limitado) | 14 datasets apenas |

**SEADE** (`api.seade.gov.br/`) destaca-se com APIs JSON sem autenticação cobrindo 200+ variáveis socioeconômicas de todos os municípios paulistas. **ISP-RJ** (`ispdados.rj.gov.br`) oferece séries históricas de criminalidade desde 2003 por delegacia — pacote R `ispdados` disponível.

---

## 11. Portais municipais — GeoSampa e SPTrans lideram em APIs

| Município | Portal | API | Destaque |
|-----------|--------|-----|----------|
| **São Paulo** | dados.prefeitura.sp.gov.br | ✅ CKAN + APILIB + SPTrans OlhoVivo + GeoSampa WMS/WFS | 400+ camadas geo; ônibus em tempo real |
| **Rio de Janeiro** | data.rio | ✅ CKAN + API v2 + ArcGIS | API própria documentada |
| **Curitiba** | dadosabertos.curitiba.pr.gov.br | ✅ CKAN + ArcGIS Hub | Central 156, alvarás, transporte |
| **Belo Horizonte** | dados.pbh.gov.br | ✅ CKAN | BHTrans/GTFS, saúde, educação |
| **Fortaleza** | dados.fortaleza.ce.gov.br | ✅ CKAN | Saúde (UBS geolocalizado) |
| **Porto Alegre** | dadosabertos.poa.br | ✅ CKAN | GTFS transporte público |
| **Recife** | dados.recife.pe.gov.br | ✅ CKAN | Licenciamento, trânsito |

**SPTrans OlhoVivo** (`api.olhovivo.sptrans.com.br/v2.1`) é uma das APIs em tempo real mais ricas do Brasil: posição de ~15.000 ônibus, previsão de chegada, linhas e paradas. Requer cadastro gratuito + token. SDKs: `sptrans` (R), `lib-api-olhoVivo` (PHP). **GeoSampa** oferece 400+ camadas via WMS/WFS OGC: zoneamento, equipamentos públicos, hidrografia, ortofotos.

---

## 12. Cultura e patrimônio — SALIC é a estrela, IPHAN tem shapefiles

**SALIC/Lei Rouanet** já detalhado acima. **ANCINE/OCA** (`gov.br/ancine/oca/dados-abertos`) publica obras audiovisuais, bilheteria semanal, market share e parque exibidor em CSV — sem API. **MinC** (`dados.cultura.gov.br`) agrega via CKAN. **IPHAN** disponibiliza 1.700+ bens tombados em planilhas e sítios arqueológicos georreferenciados (CNSA) em shapefiles. **Biblioteca Nacional** tem 3M+ documentos digitalizados mas sem API REST — catálogo usa protocolo Z39.50/SRU.

---

## 13. Correios e outros serviços — API oficial agora requer contrato

**Correios** (`cws.correios.com.br`) modernizou suas APIs em 2024: REST JSON com token. APIs de rastreamento agora são **restritas a objetos vinculados ao contrato do remetente** — o rastreamento público aberto foi descontinuado. Cálculo de frete e prazos disponível com login Meu Correios (gratuito). Comunidade GitHub muito ativa com wrappers em Node, Python e PHP.

---

## Classificação por prioridade de integração ao mcp-brasil

As fontes abaixo são as **top 20 mais viáveis** para se tornarem MCP tools, ranqueadas por acessibilidade da API, riqueza dos dados e demanda potencial:

| # | Fonte | API | Auth | Potencial |
|---|-------|-----|------|-----------|
| 1 | **SICONFI/Tesouro** | REST JSON | Nenhuma | Finanças de 5.500+ municípios |
| 2 | **Comex Stat** | REST JSON (Swagger) | Nenhuma | Comércio exterior completo |
| 3 | **ONS** | CKAN + REST + AWS S3 | Nenhuma | Energia em tempo quase-real |
| 4 | **SALIC/Lei Rouanet** | REST JSON (Swagger) | Nenhuma | Cultura/incentivos desde 1991 |
| 5 | **IpeaData** | OData v4 | Nenhuma | Milhares de séries macro/sociais |
| 6 | **CVM** | CKAN API | Nenhuma | Fundos e companhias abertas |
| 7 | **Atlas da Violência** | REST OData | Nenhuma | Segurança por município |
| 8 | **ANEEL** | CKAN + SQL | Nenhuma | 69 datasets energia |
| 9 | **ANTT** | CKAN + SQL | Nenhuma | 105 datasets transportes |
| 10 | **TCU** | REST JSON | Nenhuma | Acórdãos e jurisprudência |
| 11 | **CCEE** | CKAN API | Nenhuma | 132 datasets mercado energia |
| 12 | **BNDES** | CKAN API | Nenhuma | Operações de financiamento |
| 13 | **CNES** | REST JSON | Nenhuma | 400 mil+ estabelecimentos saúde |
| 14 | **Embrapa AgroAPI** | REST (Swagger) | Token freemium | 8 APIs agrícolas |
| 15 | **CAPES** | CKAN API | Nenhuma | Pós-graduação e pesquisa |
| 16 | **FNDE** | CKAN API | Nenhuma | Educação básica (PNAE, FUNDEB) |
| 17 | **SPTrans OlhoVivo** | REST JSON | Token gratuito | Ônibus SP tempo real |
| 18 | **Brasil.io** | REST JSON | Token gratuito | Datasets diversos |
| 19 | **SINESP/MJSP** | CKAN API | Nenhuma | Criminalidade municipal |
| 20 | **SEADE (SP)** | REST JSON | Nenhuma | 200+ variáveis por município SP |

---

## Conclusão: ecossistema rico mas que precisa de unificação

O mapeamento revela que o governo brasileiro já publica um volume impressionante de dados, mas a experiência de acesso é fragmentada entre **portais CKAN** (ANEEL, ANTT, ANTAQ, BNDES, CVM, CCEE, ONS, MME, DNIT, CAPES, FNDE, DATAPREV + dezenas de portais estaduais/municipais), **APIs REST dedicadas** (SICONFI, Comex Stat, SALIC, Atlas da Violência, TCU, CNES, Embrapa, ANVISA, CGU), **OData** (IpeaData), **downloads bulk** (Receita Federal, PGFN, PRF, ANP, ANS, MTE, DEPEN) e **geosserviços OGC** (INDE, GeoSampa, SICAR, ANM, DNIT). Nenhuma agência reguladora tem MCP server — essa é a principal lacuna e oportunidade. Um projeto mcp-brasil que unifique essas 90+ fontes em MCP tools padronizados criaria o maior hub de acesso programático a dados públicos do Brasil, habilitando AI agents a responder perguntas sobre finanças públicas, energia, comércio exterior, saúde, educação, segurança, agricultura e urbanismo com dados oficiais atualizados.