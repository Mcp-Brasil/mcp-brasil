# 🇧🇷 mcp-brasil — Roadmap Técnico v3

## MCP Servers para APIs Públicas Brasileiras

> **Stack:** Python 3.10+ · FastMCP v3 (Prefect) · httpx · Pydantic · uv  
> **Arquitetura:** Package by Feature · Auto-Registry · Clean Code · Async-first  
> **ADRs:** `001-project-bootstrap.md` · `002-auto-registry-pattern.md`

---

## 1. Visão

Uma única instalação, um comando, dados reais de 10+ APIs governamentais — com **zero configuração manual** para adicionar novas features.

```bash
uv pip install mcp-brasil
fastmcp run mcp_brasil.server:mcp
# INFO: Registered feature 'ibge' v0.1.0
# INFO: Registered feature 'bacen' v0.1.0
# INFO: Feature 'transparencia' requires TRANSPARENCIA_API_KEY (not set), skipping.
# INFO: mcp-brasil — 2 feature(s) active, 1 skipped
```

---

## 2. Arquitetura — Auto-Registry + Package by Feature

### Como funciona o Auto-Registry

O `server.py` raiz **nunca é editado**. Ele usa `FeatureRegistry` para:

1. **Escanear** `src/mcp_brasil/` via `pkgutil.iter_modules()` (mesmo padrão do Flask, pytest, Django)
2. **Validar** cada subpacote: tem `FEATURE_META`? Tem `server.mcp`? Auth disponível?
3. **Montar** automaticamente via `FastMCP.mount()` em `/{feature_name}`

```python
# src/mcp_brasil/server.py — este arquivo NUNCA muda
from fastmcp import FastMCP
from ._shared.feature import FeatureRegistry

mcp = FastMCP("mcp-brasil 🇧🇷")
registry = FeatureRegistry()
registry.discover()
registry.mount_all(mcp)
```

### Como adicionar uma feature (zero-touch)

```bash
# Basta criar o diretório com a convenção:
mkdir -p src/mcp_brasil/inep
```

```python
# src/mcp_brasil/inep/__init__.py
from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="inep",
    description="Dados educacionais: ENEM, IDEB, censo escolar",
    api_base="https://api.inep.gov.br",
)
```

```python
# src/mcp_brasil/inep/server.py
from fastmcp import FastMCP
from .tools import buscar_escolas, consultar_ideb

mcp = FastMCP("mcp-brasil-inep")
mcp.tool(buscar_escolas)
mcp.tool(consultar_ideb)
```

Pronto — na próxima execução, o INEP é descoberto e montado automaticamente.

### Estrutura do diretório

```
src/mcp_brasil/
├── __init__.py
├── server.py              # NUNCA MUDA — usa FeatureRegistry
├── settings.py
├── exceptions.py
│
├── _shared/               # Privado (prefixo _)
│   ├── __init__.py
│   ├── feature.py         # ★ FeatureMeta + FeatureRegistry
│   ├── http_client.py     # httpx com retry + cache + rate-limit
│   ├── cache.py           # LRU com TTL
│   └── formatting.py      # Formatação para LLMs
│
├── ibge/                  # ★ Auto-descoberto
│   ├── __init__.py        # → FEATURE_META (obrigatório)
│   ├── server.py          # → mcp: FastMCP (obrigatório)
│   ├── tools.py           # Lógica das tools
│   ├── client.py          # HTTP async para API
│   ├── schemas.py         # Pydantic models
│   └── constants.py       # URLs, IDs fixos
│
├── bacen/                 # ★ Auto-descoberto
├── transparencia/         # ★ Auto-descoberto (requer API key)
├── camara/                # ★ Auto-descoberto
├── senado/                # ★ Auto-descoberto
└── ...                    # Crie um diretório → auto-registrado
```

### Convenção obrigatória para discovery

| Requisito | Onde | O que |
|-----------|------|-------|
| `FEATURE_META` | `__init__.py` | Instância de `FeatureMeta` |
| `mcp` | `server.py` | Instância de `FastMCP` |
| Nome do diretório | Raiz de `mcp_brasil/` | Sem prefixo `_` |
| Auth (se necessário) | Env var | Definida no `auth_env_var` |

### Fluxo de dependência dentro de cada feature (Clean Code)

```
server.py  →  tools.py  →  client.py  →  schemas.py
   ↓              ↓            ↓             (dados puros)
registra      orquestra     faz HTTP
tools         lógica        async
```

Regras:
- `server.py` nunca tem lógica de negócio
- `tools.py` nunca faz HTTP direto
- `client.py` nunca formata para LLM
- `schemas.py` nunca tem lógica — só Pydantic models

---

## 3. FeatureRegistry — API Completa

```python
from mcp_brasil._shared.feature import FeatureRegistry, FeatureMeta

# Criar e descobrir
registry = FeatureRegistry()
registry.discover()                         # Escaneia mcp_brasil/

# Consultar
registry.features                           # dict[str, RegisteredFeature]
registry.skipped                            # dict[str, str] — nome: motivo
registry.get_feature("ibge")                # RegisteredFeature | None
registry.list_tools()                       # ["/ibge/buscar_localidades", ...]
registry.summary()                          # Resumo formatado para logs

# Montar no server
registry.mount_all(mcp)                     # Monta todas no root server

# Chaining
registry.discover().mount_all(mcp)          # Uma linha
```

### FeatureMeta — campos

```python
FeatureMeta(
    name="transparencia",                   # Identificador da feature
    description="Contratos e despesas...",   # Descrição para docs e LLMs
    version="0.1.0",                        # SemVer da feature
    api_base="https://api...",              # URL base da API
    requires_auth=True,                     # Precisa de credencial?
    auth_env_var="TRANSPARENCIA_API_KEY",   # Nome da env var
    enabled=True,                           # Feature flag
    tags=["governo", "contratos"],          # Tags para busca/docs
)
```

---

## 4. APIs por Tier

### Tier 1 — MVP (Semanas 1-2)

| Feature | Auth | Tools |
|---------|------|-------|
| `ibge` | 🔓 Nenhuma | localidades, populacao, pib, nomes, malha, agregados, cnae, inflacao |
| `bacen` | 🔓 Nenhuma | cambio, selic, ipca, expectativas, pix, ptax |
| `transparencia` | 🔑 API key | contratos, despesas, servidores, licitacoes, bolsa_familia, ceis |

### Tier 2 — Expansão (Semanas 3-4)

| Feature | Auth | Tools |
|---------|------|-------|
| `camara` | 🔓 Nenhuma | deputados, proposicoes, votacoes, despesas |
| `senado` | 🔓 Nenhuma | senadores, materias, votacoes, comissoes |
| `dados_abertos` | 🔑 Token | buscar_datasets, detalhar_dataset, listar_organizacoes |

### Tier 3 — Vertical (Semanas 5-8)

| Feature | Auth | Tools |
|---------|------|-------|
| `datajud` | 🔑 API key CNJ | buscar_processos, estatisticas, movimentacoes |
| `diario_oficial` | 🔓 Nenhuma | buscar_publicacoes, extrair_texto |
| `receita` | 🔓 Nenhuma | consultar_cnpj, situacao_cadastral |

---

## 5. Timeline

### Semana 0 — Bootstrap (2 dias)

- [ ] Criar repo, pyproject.toml, justfile
- [ ] Implementar `_shared/feature.py` (FeatureRegistry)
- [ ] Implementar `server.py` raiz (3 linhas)
- [ ] ADR-001 + ADR-002 commitados
- [ ] CI: ruff → mypy → pytest
- [ ] AGENTS.md, CLAUDE.md, CONTRIBUTING.md

### Semana 1 — Feature IBGE (5 dias)

- [ ] 8 tools (localidades, populacao, pib, nomes, malha, agregados, cnae, inflacao)
- [ ] `_shared/http_client.py` com retry + backoff
- [ ] Tests: unit + mock + integration
- [ ] GIF demo no README
- [ ] Publicar `mcp-brasil==0.1.0` no PyPI

### Semana 2 — Features Bacen + Transparência (5 dias)

- [ ] 6 tools Bacen + 6 tools Transparência
- [ ] `_shared/cache.py` com TTL
- [ ] Validar que Transparência é pulada sem API key (registry behavior)
- [ ] Publicar `mcp-brasil==0.2.0`

### Semana 3 — Launch (5 dias)

- [ ] Features Câmara + Senado
- [ ] DIA DE LANÇAMENTO: Reddit, HN, Dev.to, X, LinkedIn
- [ ] Artigo técnico

### Semana 4+ — Comunidade + Escala

- [ ] Features Tier 3
- [ ] Integração OpenClaw / n8n
- [ ] 500+ stars

---

## 6. Configuração

### Claude Desktop

```json
{
  "mcpServers": {
    "brasil": {
      "command": "uv",
      "args": ["run", "fastmcp", "run", "mcp_brasil.server:mcp"],
      "env": {
        "TRANSPARENCIA_API_KEY": "sua-chave-aqui"
      }
    }
  }
}
```

### justfile

```just
install:
    uv sync
test:
    uv run pytest -v
test-feature feature:
    uv run pytest tests/{{feature}}/ -v
lint:
    uv run ruff check src/ tests/ && uv run ruff format src/ tests/
types:
    uv run mypy src/mcp_brasil/
run:
    uv run fastmcp run mcp_brasil.server:mcp
serve:
    uv run fastmcp run mcp_brasil.server:mcp --transport http --port 8000
inspect:
    uv run fastmcp inspect mcp_brasil.server:mcp
ci: lint types test
```

---

*Roadmap v3 — 2026-03-21 — Python + FastMCP v3 + Auto-Registry*
