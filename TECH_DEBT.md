# TECH_DEBT.md — Interactive TODO for Technical Debt

> Track bugs, incompatibilities, mocks, and incomplete implementations.
> Update this file whenever you find something that needs attention later.

## Legend

- `[ ]` — Open (needs work)
- `[~]` — In progress
- `[x]` — Resolved

---

## Bootstrap Phase

- [x] **mount() API mismatch** — `feature.py` used `mount("/path", server)` instead of FastMCP v3's `mount(server, namespace=name)`. Fixed.
- [x] **list_tools() accessed private API** — `_tool_manager._tools` is private FastMCP internals. Removed method to avoid mypy strict failures.
- [x] **_shared/http_client.py** — `create_client()` + `http_get()` with retry + exponential backoff + 429/5xx.
- [x] **_shared/formatting.py** — `markdown_table`, `format_brl`, `format_number_br`, `format_percent`, `truncate_list`.
- [x] **_shared/cache.py** — `TTLCache` class + `@ttl_cache(ttl=300)` decorator for async functions.
- [x] **settings.py** — Env var overrides: `HTTP_TIMEOUT`, `HTTP_MAX_RETRIES`, `HTTP_BACKOFF_BASE`, `USER_AGENT`.
- [x] **pyproject.toml dependency-groups** — Migrated from `[project.optional-dependencies]` to `[dependency-groups]`. `make dev` uses `uv sync --group dev`.
- [x] **justfile removed** — Replaced by Makefile.

## Core — Open

- [x] **Response size limiting for LLM context** — APIs can return huge payloads (e.g., 5000+ municipios). Need a strategy to truncate/summarize responses to avoid blowing LLM context windows. See `_shared/formatting.py:truncate_list` as starting point.

## Transparência Feature

- [x] **Resources e prompts faltando** — Feature tinha apenas tools/client/schemas. Adicionados resources.py (endpoints, bases de sanções, info da API) e prompts.py (auditoria_fornecedor, analise_despesas, verificacao_compliance) + server.py atualizado + 27 testes novos.
- [x] **API response shapes unverified** — Resolvido. Adicionado `_safe_parse_list()` com logging de warning para respostas inesperadas, guards em `_parse_bolsa_*` contra strings no lugar de dicts, e 20+ testes de edge cases (non-list, null fields, string fields).
- [x] **Rate limiting not enforced client-side** — Resolvido. Adicionado `_shared/rate_limiter.py` (sliding window 80 req/min) aplicado via `_get()`. `buscar_sancoes` refatorado para usar `_get()` ao invés de `http_get()` direto.
- [x] **Pagination not automatic** — Resolvido. Adicionado `_pagination_hint()` em tools.py que mostra "Use pagina=N+1" quando resultados >= DEFAULT_PAGE_SIZE e "Última página" quando < PAGE_SIZE em pagina > 1.
- [x] **Pre-existing mypy errors in lifespan.py and ibge/client.py** — Resolvido. mypy passa limpo em todos os 35+ arquivos.
- [x] **Cobertura limitada a 8 tools** — Expandido de 8 para 18 tools. Adicionados: convênios, cartões corporativos, PEP, acordos de leniência, notas fiscais, benefícios sociais, consulta CPF/CNPJ, detalhe de contrato/servidor. Resource `categorias-beneficios` adicionada. 148 testes passam.
- [ ] **API endpoints unverified against production** — 10 novos endpoints adicionados baseados na documentação da API. Os nomes de parâmetros e shapes de resposta precisam ser validados contra a API real do Portal da Transparência.

## Câmara Feature

- [x] **Envelope extraction** — API wraps all responses in `{"dados": [...], "links": [...]}`. Handled by `_get()` helper that auto-extracts `dados` field. Tested with empty/missing dados.
- [x] **No client-side rate limiting** — Resolvido. RateLimiter(60 req/min) aplicado via `_get()`.
- [x] **Pagination is server-controlled** — By design. `_pagination_hint()` provides LLM-facing hints ("Use pagina=N+1"). Auto-pagination is not needed for MCP tools since the LLM controls iteration.

## Senado Feature

- [x] **Deeply nested JSON responses** — API returns structures like `data.ListaParlamentarEmExercicio.Parlamentares.Parlamentar`. Handled by `_deep_get(*keys)` helper with safe navigation.
- [x] **Single result as dict instead of list** — When only 1 result, API returns `{}` instead of `[{}]`. Handled by `_ensure_list()` coercion in all parsers.
- [x] **JSON via Accept header** — API requires `Accept: application/json` header. `JSON_HEADERS` constant passed through all requests.
- [x] **No client-side rate limiting** — Resolvido. RateLimiter(60 req/min) aplicado via `_get()`.
- [x] **No pagination support** — By design. `_pagination_hint()` suggests refining filters. Senado API returns full datasets; LLM-facing hints guide users to narrow queries.
- [x] **Votação nominal endpoint may vary** — Resolvido. Old plenário endpoint (`/plenario/lista/votacao`) deprecated and deactivated 2026-02-01. Migrated `listar_votacoes`, `obter_votacao`, `votacoes_recentes` to new `/votacao` API (flat JSON, camelCase). Parsers handle both old PascalCase and new camelCase formats. `votos_materia` still uses `/materia/votacoes/{id}` which remains active.
- [ ] **E-Cidadania tools not implemented** — Plan includes 9 web-scraping tools for e-Cidadania. Deferred to future sessions.
- [ ] **dados_abertos auxiliary tools not implemented** — Plan includes 4 additional tools. Deferred to future sessions.

## DataJud Feature

- [x] **Elasticsearch POST API** — DataJud uses POST with JSON body (not GET). Client uses `httpx.AsyncClient.post()` directly instead of shared `http_get()`. Rate limited at 30 req/min.
- [ ] **API key required** — Feature requires `DATAJUD_API_KEY` env var. Auto-discovery skips the feature if not set. Registration: https://datajud.cnj.jus.br
- [x] **Elasticsearch query DSL limited** — Resolvido. `buscar_processos` migrado de `_all` (deprecated) para `multi_match`. Adicionado `buscar_processos_avancado` com `bool.must` query por códigos (classe.codigo + orgaoJulgador.codigo).
- [x] **No pagination (offset/from)** — Resolvido. Implementado `search_after` com `sort` por `@timestamp` (padrão recomendado pela Wiki DataJud). Token de paginação retornado para o LLM.
- [x] **Tribunais incompletos** — Resolvido. Adicionados 27 TREs (tre-ac a tre-to) e 3 TJMs (tjmmg, tjmrs, tjmsp). Total: 91 tribunais cobrindo toda a documentação oficial.

## TSE Feature

- [x] **REST API without CORS** — TSE DivulgaCandContas API blocks browser requests but works fine with httpx. No auth required.
- [x] **Nested response shapes** — Candidatos endpoint wraps results in `{"candidatos": [...]}`, cargos in `{"cargos": [...]}`. Handled by extracting nested fields.
- [ ] **API is unofficial** — DivulgaCandContas API is reverse-engineered. No official documentation. Endpoints may change without notice.
- [ ] **No result totalization** — TSE API returns election results (votos) but `buscar_candidato` only shows `descricaoTotalizacao`. A dedicated tool for election results could be added.

## Jurisprudência Feature

- [x] **REST APIs for STF/STJ/TST** — Implemented using httpx GET requests to court search APIs. Browser automation (Playwright) not needed.
- [ ] **API response shapes unverified against real data** — STF/STJ/TST APIs are reverse-engineered from web UIs. Real response formats may differ from what parsers expect. Graceful fallback returns empty lists on errors.
- [ ] **STJ SCON may return HTML** — STJ's search endpoint (`pesquisar.jsp`) may return HTML instead of JSON depending on parameters. Current implementation assumes JSON; needs real API validation.
- [ ] **TST backend API undocumented** — TST jurisprudence backend endpoint may not accept the parameters currently used. Needs real API testing.
- [ ] **Súmulas search limited to STF** — Only STF súmulas are implemented. STJ and TST súmula APIs need investigation.
- [ ] **No informativo-specific endpoint** — `buscar_informativos` currently delegates to the main search with "informativo" as query. A dedicated informativo API would give better results.

## BrasilAPI Feature

- [x] **Falsy `status_code=0` bug** — `status_map.get(info.status_code or -1, ...)` treated `status_code=0` as `-1` because `0` is falsy in Python. Fixed to use explicit `is not None` check.
- [ ] **FIPE endpoints may be slow/unreliable** — BrasilAPI proxies FIPE data; upstream can be slow. No retry/fallback implemented beyond shared `http_get()` retries.
- [ ] **CNPJ rate limiting** — BrasilAPI has aggressive rate limiting on CNPJ endpoint. Shared rate limiter not applied (uses shared `http_get()` only).

## Diário Oficial Feature

- [ ] **Limited to 3 tools** — Original plan called for 4 tools. The 4th (buscar por CNPJs em lote) was deferred; Querido Diário API doesn't support batch CNPJ search natively.
- [ ] **No excerpt highlighting** — Querido Diário returns raw text excerpts with `<em>` tags. Current implementation truncates at 500 chars but doesn't strip HTML tags.

## Compras Feature

- [ ] **Limited to 3 tools** — Original plan called for 6 tools (including CEIS/CNEP sanctions and Comprasnet). CEIS/CNEP covered by transparência feature. Comprasnet deferred.
- [ ] **PNCP API response format unverified** — Response parsing uses `data.resultado` fallback. Real API response shape needs validation against live PNCP endpoint.

## TransfereGov Feature

- [x] **Feature criada do zero** — 5 tools (buscar_emendas_pix, buscar_emenda_por_autor, detalhe_emenda, emendas_por_municipio, resumo_emendas_ano) + resource + prompt + 36 testes. API PostgREST pública sem auth.
- [ ] **API endpoints unverified against production** — TransfereGov PostgREST API pode ter colunas/nomes diferentes dos usados. Precisa validação com API real.
- [ ] **Sem agregação server-side** — `resumo_emendas_ano` retorna registros individuais paginados, não um resumo agregado. PostgREST suporta RPC functions para agregação, mas não as usamos.

## Known Limitations

- [x] **No CONTRIBUTING.md** — Resolvido. CONTRIBUTING.md criado com getting started, estrutura, como adicionar features, convenções, testes e PR guidelines.

---

*Last updated: 2026-03-22*
