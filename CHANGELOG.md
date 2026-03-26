# Changelog

Todas as mudanças notáveis do mcp-brasil estão documentadas neste arquivo.

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
