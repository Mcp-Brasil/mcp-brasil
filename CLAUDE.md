# CLAUDE.md — Instruções para o Claude Code

## IMPORTANTE: Antes de implementar qualquer coisa

**Sempre leia todos os ADRs** em `plan/adrs/` antes de começar a implementar:
- `plan/adrs/ADR-001-project-bootstrap.md` — Stack, package-by-feature, convenções
- `plan/adrs/ADR-002-auto-registry-pattern.md` — FeatureRegistry, convenção de discovery
- `plan/adrs/ADR-003-redator-oficial.md` — Padrão de agentes (Prompt + Resource + Tool)

Os ADRs são a fonte de verdade para todas as decisões de arquitetura.

## Projeto

**mcp-brasil** — MCP servers para APIs públicas brasileiras.
Pacote Python que conecta AI agents a dados governamentais (IBGE, Banco Central, Portal da Transparência, Câmara, Senado, DataJud e mais).

## Stack

- **Linguagem:** Python 3.10+
- **Framework MCP:** FastMCP v3 (Prefect) — `@mcp.tool`, `@mcp.resource`, `@mcp.prompt`
- **HTTP:** httpx (async)
- **Schemas:** Pydantic v2
- **Package manager:** uv
- **Task runner:** just (justfile)
- **Lint/Format:** ruff (line-length 99)
- **Types:** mypy (strict)
- **Testes:** pytest + pytest-asyncio + respx

## Comandos

```bash
just install        # uv sync
just test           # pytest -v
just test-feature ibge  # pytest tests/ibge/ -v
just lint           # ruff check + format check
just fix            # ruff check --fix + format
just types          # mypy
just run            # fastmcp run (stdio)
just serve          # fastmcp run (HTTP :8000)
just inspect        # fastmcp inspect
just ci             # lint + types + test
```

## Arquitetura

### Auto-Registry (ADR-002)

O `server.py` raiz **nunca é editado manualmente**. Ele usa `FeatureRegistry` para auto-descobrir features:

```python
mcp = FastMCP("mcp-brasil")
registry = FeatureRegistry()
registry.discover()
registry.mount_all(mcp)
```

Para adicionar uma feature, basta criar o diretório com a convenção. Nenhum import manual.

### Package by Feature (ADR-001)

Cada API governamental é uma feature auto-contida:

```
src/mcp_brasil/{feature}/
├── __init__.py     # FEATURE_META (obrigatório para auto-discovery)
├── server.py       # mcp: FastMCP (obrigatório)
├── tools.py        # Funções das tools
├── client.py       # HTTP async para a API
├── schemas.py      # Pydantic models
└── constants.py    # URLs, enums, códigos
```

### Fluxo de dependência dentro de cada feature

```
server.py → tools.py → client.py → schemas.py
  registra    orquestra   faz HTTP     dados puros
```

## Regras invioláveis

1. **`server.py` raiz nunca muda** — auto-registry cuida de tudo
2. **`tools.py` nunca faz HTTP** — delega para `client.py`
3. **`client.py` nunca formata para LLM** — retorna Pydantic models
4. **`schemas.py` zero lógica** — apenas BaseModel/dataclass
5. **`server.py` da feature apenas registra** — zero lógica de negócio
6. **`constants.py` zero imports** de outros módulos do projeto
7. **Toda tool tem docstring** — usada pelo LLM para decidir quando chamar
8. **Async everywhere** — `async def` em tools e clients
9. **Type hints completos** em todas as funções

## Convenções de código

| Escopo | Convenção | Exemplo |
|--------|-----------|---------|
| Módulos | snake_case | `client.py` |
| Classes | PascalCase | `class Estado(BaseModel)` |
| Funções/tools | snake_case, verbo | `buscar_localidades()` |
| Constantes | UPPER_SNAKE | `IBGE_API_BASE` |
| Privados | `_prefixo` | `_shared/`, `_cache` |

## Commits

Conventional Commits em português/inglês:

```
feat(ibge): add tool consultar_populacao
fix(bacen): handle empty response from SGS
test(ibge): add integration tests for localidades
docs: update README with bacen feature
```

## Estrutura de testes

Testes espelham `src/`:

```
tests/{feature}/
├── test_tools.py        # Mock client, testa lógica
├── test_client.py       # respx mock HTTP
└── test_integration.py  # fastmcp.Client e2e
```

## Documentação de referência

- `plan/adrs/` — Decisões de arquitetura (ADR-001, ADR-002, ADR-003)
- `plan/roadmap.md` — Roadmap técnico
- `plan/poc-plan.md` — Plano da POC com inventário de APIs
- `plan/mapa-agentes.md` — Mapa de agentes escaláveis
- `plan/research/` — Mapeamento de APIs públicas brasileiras
- `plan/refs/registry/` — Código de referência original (feature.py, server.py)
