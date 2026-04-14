# Changelog

Todas as mudanças notáveis do mcp-brasil estão documentadas neste arquivo.

## [0.12.0] - 2026-04-14

### Bug Fixes

- **deps:** Upgrade fastmcp to 3.2.3 for oauth-protected-resource endpoint
- **docker:** Allow logo.png through dockerignore

### Documentation

- **foundry:** Rewrite Teams guide with full setup from scratch
- **guide:** Add Claude.ai web connector guide

### Features

- **server:** Add logo to consent page and MCP server metadata
- **foundry:** Add agent setup script and Teams deployment guide
- **auth:** Add multi mode for OAuth + static token simultaneously
- **dados_gov_br:** Replace dados_abertos with expanded dados.gov.br integration
- **educação:** Add inep and fnde education features
- **saude:** Add epidemiology tools (InfoDengue, InfoGripe, SINAN, DATASUS catalog)

## [0.11.0] - 2026-04-10

### Documentation

- Update tool counts to 363 and expand transparencia catalog

### Refactoring

- **auth:** Introduce configurable auth strategy (none|static|oauth)

## [0.10.0] - 2026-04-09

### Features

- **transparencia:** Port missing endpoints and fix API parsers

## [0.9.0] - 2026-04-03

### Bug Fixes

- **farmacia_popular:** Handle CNES API envelope and address fields

### Features

- **camara:** Support id_deputado_autor in buscar_proposicao (#4)
- **datajud:** Add MPU tools for medidas protetivas de urgência

## [0.8.0] - 2026-03-29

### Documentation

- Update README and features catalog with 41 features and 326 tools

### Features

- **server:** Add optional Bearer token auth and health check endpoint
- **docker:** Add Dockerfile and docker-compose for HTTP deployment

### Refactoring

- **prompts:** Update prompt templates for citizen-focused queries

## [0.7.3] - 2026-03-29

### Documentation

- Update .env.example with META_ACCESS_TOKEN and current version
- Update features catalog and README with 41 features and 326 tools
- Remove showcase test file

### Miscellaneous

- Move showcase files to tests/

## [0.7.2] - 2026-03-28

### Documentation

- Add showcase with 51 real API tests (92.2% success rate)

## [0.7.1] - 2026-03-28

### Bug Fixes

- **saude:** Handle unexpected API response types from DataSUS
- Handle broken API endpoints across 6 features
- Handle broken API endpoints across 3 features

### Documentation

- **anuncios_eleitorais:** Add Meta Ad Library API reference
- Add Code Mode reference and update smart-tools docs

### Features

- **seguranca:** Add atlas_violencia and sinesp features
- **atlas_violencia:** Add IPEA violence statistics feature with 7 tools
- **sinesp:** Add MJSP CKAN portal feature with 6 tools

## [0.7.0] - 2026-03-28

### Bug Fixes

- **compras:** Handle nested API format and list responses in contratosgovbr
- **tcu:** Handle None sumario in mypy strict mode
- **imunizacao:** Fix mypy strict errors with proper Any typing

### Documentation

- Update README and docs with new features and tool counts

### Features

- **saude:** Add 6 new CNES tools for urgency, type search, detail, coordinates, municipal summary and comparison
- **farmacia_popular:** Add feature with 6 tools, resources and prompts
- **compras:** Add contratosgovbr submodule
- **anvisa:** Add bulario tools for medication search, bulas and categories
- **tcu:** Add certidoes, pautas and solicitations tools
- **tcu:** Add all 8 tools with full test coverage
- **rename:** Add RENAME essential medicines catalog with 5 tools
- **bndes:** Add CKAN integration with 4 tools for financing data
- **opendatasus:** Add CNES health facility tools
- **opendatasus:** Add OpenDataSUS CKAN API with 5 tools
- **tcu:** Add search filters to consultar_acordaos
- **denasus:** Add DENASUS audit scraping with 5 tools
- **anvisa:** Add 5 new tools (category, generics, registry, company, summary)
- **bps:** Add BPS health pricing tools
- **farmacia_popular:** Add municipios_atendidos and farmacia_mais_proxima tools
- **opendatasus:** Add consultar_vacinacao and consultar_srag dedicated tools
- **tcu:** Add consultar_pautas_sessao tool
- **imunizacao:** Add PNI vaccination feature with 10 tools
- **diario_oficial:** Add DOU federal, unified search, and fix QD bugs
- **diario_oficial:** Add listar_diarios_recentes tool and orgaos_federais_dou resource
- **pncp:** Add 9 new tools covering all PNCP API endpoints
- **forum_seguranca:** Add public safety publications feature via DSpace API

### Refactoring

- **tcu:** Simplify schemas, client and tools

### Styling

- **compras:** Apply ruff formatting

## [0.6.0] - 2026-03-28

### Bug Fixes

- **batch:** Replace AsyncMock with real function in ctx test

### Features

- **tce_es:** Add TCE-ES integration via dados.es.gov.br CKAN API (#2)

## [0.5.0] - 2026-03-27

### Bug Fixes

- **anuncios_eleitorais:** Align client with Meta Graph API format

### Documentation

- Remove ADR references and documentation section from CONTRIBUTING
- Update README with tabua_mares feature and correct counts
- Remove raio-x-parlamentar example

### Features

- **anuncios_eleitorais:** Add Meta Ad Library feature with 6 tools, 3 resources, and 3 prompts

## [0.4.0] - 2026-03-26

### Features

- **tabua_mares:** Add tide table feature with 7 tools, resource, and prompts

## [0.3.4] - 2026-03-26

### Features

- **camara:** Add detalhar_proposicao tool and improve buscar_proposicao

## [0.3.3] - 2026-03-26

### Bug Fixes

- **transparencia,dados_abertos,diario_oficial:** Address API limitations

## [0.3.2] - 2026-03-26

### Miscellaneous

- Add PyPI keywords and classifiers

## [0.3.1] - 2026-03-26

### Bug Fixes

- **code-mode:** Graceful fallback to BM25 when pydantic-monty missing
- **pncp:** Rewrite client to match real API spec
- **.gitignore:** Remove temporary files from Claude directory
- **batch:** Fix AsyncMock spec for ctx inspection in test

### Build

- **deps:** Add fastmcp[code-mode] extra to dependencies
- **deps:** Move anthropic to main dependencies

### Documentation

- Rewrite README for public launch and add MIT license
- **examples:** Add 11 use case guides for different professional contexts
- **readme:** Update tool count from 205 to 204

### Features

- **batch:** Add executar_lote tool for parallel multi-query execution
- **planner:** Add planejar_consulta tool with structured execution plans

### Miscellaneous

- Update build config and architecture docs
- Add logo, update README and gitignore
- Remove internal files from git tracking
- Add white logo variant

### Performance

- **tse:** Cache state data in _enrich_candidate_names

## [0.3.0] - 2026-03-23

### Bug Fixes

- **tse:** Resolve CDN election codes per cargo type
- **tests:** Set TOOL_SEARCH=none in conftest.py before any import

### Documentation

- **contributing:** Add release, CI/CD, testing patterns and stack info
- Add tool search and LLM discovery env vars to .env.example
- **tech-debt:** Add comprasnet deprecation and TCE features status

### Features

- **compras:** Add Dados Abertos Compras.gov.br with 8 tools
- **tse:** Add CDN election results with 4 new tools
- **tcu:** Add TCU with 8 tools, 1 resource, 1 prompt
- **tce_rj:** Add TCE-RJ with 7 tools, 1 resource, 1 prompt
- **tools:** Add semantic tags to all tool registrations
- **tools:** Add semantic tags to brasilapi and datajud tools
- **tools:** Add semantic tags to compras, jurisprudencia, tcu tools
- **tools:** Add semantic tags to pncp and transferegov tools
- **tools:** Add semantic tags to dados_abertos, diario_oficial, saude
- **tools:** Add semantic tags to redator, ana, inpe tools
- **tce_sp:** Add TCE-SP with 3 tools, 1 resource, 1 prompt
- **tse:** Add federal election results via CDN -v.json format
- **discovery:** Add BM25 search, code_mode, and recomendar_tools
- **tce_ce:** Add TCE-CE with 4 tools, 1 resource, 1 prompt
- **tce_pe:** Add TCE-PE with 5 tools, 1 resource, 1 prompt
- **tce_rs:** Add TCE-RS with 5 tools, 1 resource, 1 prompt
- **tce_sc:** Add TCE-SC with 2 tools, 1 resource, 1 prompt
- **tce_rn:** Add TCE-RN with 5 tools, 1 resource, 1 prompt
- **tce_to:** Add TCE-TO with 3 tools, 1 resource, 1 prompt
- **tce_pi:** Add TCE-PI with 5 tools, 1 resource, 1 prompt

## [0.2.2] - 2026-03-23

### Documentation

- **release:** Add release rules to CLAUDE.md and AGENTS.md

### Testing

- **compras:** Update tests for pncp subpackage

## [0.2.0] - 2026-03-23

### Bug Fixes

- **transparencia:** Add safe parsing, rate limiting, and bolsa guards
- **senado:** Migrate votação endpoints to new API
- **datajud:** Replace try-except-pass with contextlib.suppress
- **transferegov,transparencia:** Correct API mapping and number parsing
- **diario_oficial:** Strip HTML tags from excerpts
- **ibge:** Correct aggregate IDs for pib_per_capita and area_territorial
- **datajud,transparencia:** Remove STF from DataJud, fix PEP endpoint

### Documentation

- **tech-debt:** Update resolved items and fix header typo
- Add commit-on-green and tech-debt rules
- **adrs,skills:** Update patterns with resources, prompts, context, and middleware
- Add CONTRIBUTING.md

### Features

- **shared:** Add http client and formatting utilities
- **core:** Add TTL cache and migrate to dependency-groups
- **ibge,bacen:** Add ibge feature and bacen scaffold
- **ibge,bacen,transparencia:** Add malha, CNAE, bacen client, and transparencia feature
- Add lifespan, context, resources, prompts, and middleware
- **transparencia:** Add resources, prompts, and integration tests
- **_shared:** Add async RateLimiter with sliding window
- **transparencia:** Add pagination hints to tool responses
- **camara:** Add deputies and legislation tools
- **senado:** Add senators and legislation tools
- **legislativo:** Add rate limiting to camara and senado clients
- **senado:** Add partidos_senado and ufs_senado tools
- **judiciario:** Add datajud, tse and jurisprudencia features
- **phase4:** Add brasilapi, diario_oficial and compras features
- **tse:** Add supplementary elections tools
- **datajud:** Add TREs, TJMs, bool queries and search_after pagination
- **transparencia,transferegov:** Expand to 18 tools + new transferegov feature
- **senado:** Add 4 dados_abertos tools (emendas, blocos, liderancas, relatorias)
- **tse:** Add resultado_eleicao tool with vote totalization
- Complete mcp-brasil with 4 new features + expand 3 existing
- **tse:** Add CDN election results with 4 new tools
- **release:** Add release management infrastructure

### Miscellaneous

- Bootstrap project structure
- Add claude code skills (commit, fastmcp, skill-creator)
- **config:** Switch from justfile to Makefile
- Update Makefile, README, gitignore, and env example
- Mark all transparencia tech debt as resolved
- Add ibge tests, drop cursor config
- Resolve pagination tech debt as by-design
- **transparencia:** Add new schemas and constants
- Load .env file automatically via dotenv

### Refactoring

- Rename docs/ to plan/ and create empty docs/
- **registry:** Fix mount API and add core modules
- Reorganize features into data/ and agentes/ packages

### Testing

- **shared:** Add test suite for core and shared modules
- Add integration tests for resources, prompts, and full server
- **redator:** Add 3ª edição integration tests for ofício and pronomes

<!-- generated by git-cliff -->
