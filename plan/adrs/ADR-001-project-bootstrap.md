# ADR-001: Bootstrap do Projeto mcp-brasil

## Status

**Aceito** — 2026-03-21

## Contexto

Estamos iniciando o projeto `mcp-brasil`, uma coleção de MCP servers que conecta AI agents a APIs públicas brasileiras (IBGE, Banco Central, Portal da Transparência, Câmara, Senado, DataJud, entre outras).

O projeto precisa definir:
1. Linguagem e framework MCP
2. Organização do código (package by feature vs. package by layer)
3. Tooling e DX (developer experience)
4. Padrões de código e qualidade
5. Estratégia de monorepo vs. multi-package
6. Modelo de contribuição open-source

O objetivo é criar um projeto que seja referência para a comunidade brasileira de desenvolvedores AI, com código limpo, bem documentado, fácil de contribuir.

---

## Decisão 1: Python + FastMCP (Prefect) como framework

### Opções Consideradas

| Opção | Prós | Contras |
|-------|------|---------|
| **FastMCP (Python)** | 23.9k stars, padrão de facto (70% dos MCP servers), API Pythonic com decorators, suporte a stdio + HTTP, CLI embutida, 1M downloads/dia | Menos natural para devs TS-only |
| TypeScript SDK oficial | SDK oficial Anthropic, ecossistema npm amplo | Mais verboso, menos abstrações, comunidade menor |
| FastMCP (TypeScript - punkpeye) | Boa DX, TypeScript-native | Menor adoção, ecossistema separado |

### Escolha: FastMCP (Python) da Prefect

**Justificativa:**
- FastMCP v1 foi incorporado ao SDK oficial do MCP Python em 2024 — é literalmente o padrão
- API via decorators (`@mcp.tool`, `@mcp.resource`) é a mais Pythonic e concisa possível
- Suporte nativo a async/await, type hints, validação automática via Pydantic
- CLI embutida (`fastmcp run`, `fastmcp dev`, `fastmcp inspect`) elimina boilerplate
- Deploy gratuito via Prefect Horizon ou self-hosted
- Comunidade ativa (Discord, 7.9k projetos dependentes, 89 releases)
- Ecossistema Python alinha com stack de data science / ML brasileiro

**Consequências:**
- Contribuidores precisam saber Python 3.10+
- Dependência direta de `fastmcp>=3.0` como framework core
- Seguiremos os design principles do FastMCP: Fast, Simple, Pythonic, Complete

---

## Decisão 2: Package by Feature (não por camada)

### Opções Consideradas

| Opção | Estrutura | Trade-off |
|-------|-----------|-----------|
| **Package by Feature** | `src/mcp_brasil/ibge/`, `src/mcp_brasil/bacen/` | Alta coesão por domínio, cada feature é auto-contida |
| Package by Layer | `src/tools/`, `src/clients/`, `src/schemas/` | Familiar, mas baixa coesão — mudar uma API toca N diretórios |
| Monorepo multi-package | `packages/ibge/`, `packages/bacen/` | Independência total, mas overhead de publicação e CI |

### Escolha: Package by Feature dentro de um único pacote

Cada API governamental é uma **feature** auto-contida. Dentro de cada feature, os arquivos seguem a mesma estrutura interna:

```
src/mcp_brasil/
├── __init__.py              # Exports do pacote principal
├── server.py                # FastMCP server principal que compõe as features
├── settings.py              # Configuração global (env vars, defaults)
├── exceptions.py            # Exceções do projeto
│
├── _shared/                 # Módulo privado: utilitários compartilhados
│   ├── __init__.py
│   ├── http_client.py       # httpx async client com retry, cache, rate-limit
│   ├── cache.py             # LRU cache com TTL
│   ├── formatting.py        # Formatação de respostas para LLMs
│   └── types.py             # Tipos compartilhados (TypedDict, Protocols)
│
├── ibge/                    # Feature: IBGE
│   ├── __init__.py          # Re-exports público da feature
│   ├── server.py            # FastMCP sub-server com tools registradas
│   ├── tools.py             # Funções das tools (@mcp.tool)
│   ├── client.py            # Client HTTP para API do IBGE
│   ├── schemas.py           # Pydantic models (input/output)
│   └── constants.py         # URLs, códigos de agregados, enums
│
├── bacen/                   # Feature: Banco Central
│   ├── __init__.py
│   ├── server.py
│   ├── tools.py
│   ├── client.py
│   ├── schemas.py
│   └── constants.py
│
├── transparencia/           # Feature: Portal da Transparência
│   ├── __init__.py
│   ├── server.py
│   ├── tools.py
│   ├── client.py
│   ├── schemas.py
│   └── constants.py
│
├── camara/                  # Feature: Câmara dos Deputados
├── senado/                  # Feature: Senado Federal
├── dados_abertos/           # Feature: Portal dados.gov.br
├── datajud/                 # Feature: DataJud (CNJ)
└── diario_oficial/          # Feature: Querido Diário
```

**Justificativa:**
- **Coesão:** Tudo sobre IBGE está em `ibge/`. Para adicionar uma nova tool, você toca apenas um diretório.
- **Navegação:** Um contribuidor novo sabe exatamente onde olhar. Quer entender o Bacen? Abra `bacen/`.
- **Testabilidade:** Testes espelham a estrutura: `tests/ibge/`, `tests/bacen/`.
- **Composição:** O `server.py` raiz compõe os sub-servers via `mcp.mount()` do FastMCP.
- **Clean Code:** Cada feature tem responsabilidade única (SRP). O `client.py` faz HTTP, o `tools.py` expõe tools, o `schemas.py` define contratos.

**Consequências:**
- Novas features são adicionadas criando um diretório com a mesma estrutura interna
- Cada feature pode ser desenvolvida e testada isoladamente
- O `_shared/` é privado (prefixo `_`) — não faz parte da API pública

---

## Decisão 3: Anatomia de uma Feature (Clean Code)

### Convenção: cada arquivo tem uma responsabilidade clara

```
feature/
├── server.py      → Registro de tools no FastMCP (composição)
├── tools.py       → Lógica de negócio das tools (funções puras quando possível)
├── client.py      → Comunicação HTTP com a API externa (I/O isolado)
├── schemas.py     → Modelos Pydantic para input/output (contratos)
└── constants.py   → Valores imutáveis (URLs, enums, códigos)
```

**Regras:**
1. `tools.py` **nunca** faz HTTP diretamente — delega para `client.py`
2. `client.py` **nunca** formata resposta para LLM — retorna dados tipados
3. `schemas.py` contém apenas Pydantic models — zero lógica
4. `server.py` apenas registra tools — zero lógica de negócio
5. `constants.py` contém apenas valores literais — zero imports de outros módulos

**Exemplo — IBGE tool:**

```python
# ibge/schemas.py
from pydantic import BaseModel, Field

class LocalidadeInput(BaseModel):
    tipo: str = Field(description="'estados' ou 'municipios'")
    uf: str | None = Field(default=None, description="Sigla do estado (ex: PI)")

class Estado(BaseModel):
    id: int
    sigla: str
    nome: str
    regiao: str

# ibge/client.py
import httpx
from .constants import IBGE_API_BASE
from .schemas import Estado

async def listar_estados() -> list[Estado]:
    async with httpx.AsyncClient() as client:
        response = await client.get(f"{IBGE_API_BASE}/v1/localidades/estados")
        response.raise_for_status()
        data = response.json()
        return [
            Estado(
                id=e["id"],
                sigla=e["sigla"],
                nome=e["nome"],
                regiao=e["regiao"]["nome"],
            )
            for e in data
        ]

# ibge/tools.py
from .client import listar_estados, listar_municipios
from .schemas import LocalidadeInput

async def buscar_localidades(tipo: str, uf: str | None = None) -> str:
    """Busca estados e municípios do Brasil via IBGE."""
    if tipo == "estados":
        estados = await listar_estados()
        return "\n".join(f"{e.sigla} — {e.nome} ({e.regiao})" for e in estados)
    elif tipo == "municipios" and uf:
        municipios = await listar_municipios(uf)
        return "\n".join(f"{m.codigo} — {m.nome}" for m in municipios)
    return "Informe tipo='estados' ou tipo='municipios' com uf."

# ibge/server.py
from fastmcp import FastMCP
from .tools import buscar_localidades, consultar_agregado, consultar_nomes

mcp = FastMCP("mcp-brasil-ibge")

mcp.tool(buscar_localidades)
mcp.tool(consultar_agregado)
mcp.tool(consultar_nomes)
```

**Composição no server raiz:**

```python
# src/mcp_brasil/server.py
from fastmcp import FastMCP
from .ibge.server import mcp as ibge_server
from .bacen.server import mcp as bacen_server
from .transparencia.server import mcp as transparencia_server

mcp = FastMCP("mcp-brasil 🇧🇷")

mcp.mount("/ibge", ibge_server)
mcp.mount("/bacen", bacen_server)
mcp.mount("/transparencia", transparencia_server)

if __name__ == "__main__":
    mcp.run()
```

---

## Decisão 4: Tooling e DX

### Stack de desenvolvimento

| Ferramenta | Papel | Justificativa |
|-----------|-------|--------------|
| **uv** | Package manager + venv | Padrão do FastMCP, mais rápido que pip/poetry |
| **ruff** | Linting + formatting | Substitui flake8+black+isort em um único tool Rust |
| **pytest** | Testes | Padrão Python, suporte nativo a async |
| **pytest-asyncio** | Testes async | Tools usam async/await |
| **mypy** ou **ty** | Type checking | Garante consistência de types |
| **prek** (pre-commit) | Git hooks | Roda ruff+types antes de cada commit |
| **just** | Task runner | Mesmo que FastMCP usa, cross-platform |
| **httpx** | HTTP client | Async-first, melhor que requests para I/O concorrente |
| **pydantic** | Schemas | Validação + serialização + docs automáticas |
| **GitHub Actions** | CI/CD | Lint → Type check → Test → Publish |

### pyproject.toml (configuração unificada)

```toml
[project]
name = "mcp-brasil"
version = "0.1.0"
description = "MCP servers para APIs públicas brasileiras"
requires-python = ">=3.10"
license = "MIT"
dependencies = [
    "fastmcp>=3.0",
    "httpx>=0.27",
    "pydantic>=2.0",
]

[project.optional-dependencies]
dev = [
    "pytest>=8.0",
    "pytest-asyncio>=0.24",
    "ruff>=0.8",
    "mypy>=1.13",
    "prek>=0.2",
]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[tool.hatch.build.targets.wheel]
packages = ["src/mcp_brasil"]

[tool.ruff]
target-version = "py310"
line-length = 99

[tool.ruff.lint]
select = ["E", "F", "I", "UP", "B", "SIM", "RUF"]

[tool.pytest.ini_options]
testpaths = ["tests"]
asyncio_mode = "auto"
```

**Consequências:**
- Usa `src/` layout (não flat) — padrão recomendado por PyPA e FastMCP
- Um único `pyproject.toml` sem `setup.py` ou `setup.cfg`
- `uv sync` instala tudo, dev e prod

---

## Decisão 5: Estrutura de Testes (espelha features)

```
tests/
├── conftest.py              # Fixtures globais (mock HTTP, FastMCP test client)
├── ibge/
│   ├── test_tools.py        # Testa lógica das tools
│   ├── test_client.py       # Testa client HTTP (com mock/vcr)
│   └── test_integration.py  # Testa tool via FastMCP client (e2e)
├── bacen/
│   ├── test_tools.py
│   ├── test_client.py
│   └── test_integration.py
├── transparencia/
└── _shared/
    ├── test_cache.py
    └── test_http_client.py
```

**Padrões de teste:**
- `test_tools.py` — testa lógica pura, mocka o client
- `test_client.py` — testa HTTP com `respx` (mock httpx)
- `test_integration.py` — usa `fastmcp.Client` para testar tool end-to-end

**Consequências:**
- Cada feature tem seus próprios testes isolados
- CI roda `pytest tests/ibge/ -v` para PRs que só tocam IBGE

---

## Decisão 6: Estratégia de distribuição

### Opções Consideradas

| Opção | Prós | Contras |
|-------|------|---------|
| Pacote único `mcp-brasil` | Simples, um `pip install`, composição via mount | Instala todas as features mesmo se quer só uma |
| Multi-package (`mcp-brasil-ibge`, `mcp-brasil-bacen`) | Granular, install apenas o necessário | Overhead de CI/CD, versionamento, publicação |
| **Pacote único com extras** | `pip install mcp-brasil[ibge,bacen]` | Melhor dos dois mundos |

### Escolha: Pacote único com todas as features incluídas

Para o MVP, publicar como um único pacote `mcp-brasil` que inclui todos os servers. Quando atingir 10+ features, avaliar migração para extras opcionais.

**Uso:**

```bash
# Instalar
uv pip install mcp-brasil

# Rodar server completo (todas as APIs)
fastmcp run mcp_brasil.server:mcp

# Rodar apenas IBGE
fastmcp run mcp_brasil.ibge.server:mcp

# Rodar via HTTP
fastmcp run mcp_brasil.server:mcp --transport http --port 8000
```

**Claude Desktop config:**

```json
{
  "mcpServers": {
    "brasil": {
      "command": "fastmcp",
      "args": ["run", "mcp_brasil.server:mcp"]
    }
  }
}
```

---

## Decisão 7: Convenções de código (Clean Code)

### Naming

| Escopo | Convenção | Exemplo |
|--------|-----------|---------|
| Módulos | snake_case, singular | `ibge/client.py` |
| Classes | PascalCase | `class Estado(BaseModel)` |
| Funções/tools | snake_case, verbo | `buscar_localidades()` |
| Constantes | UPPER_SNAKE | `IBGE_API_BASE` |
| Variáveis privadas | `_prefixo` | `_cache = {}` |
| Módulos internos | `_prefixo` | `_shared/` |

### Docstrings

Toda tool **obrigatoriamente** tem docstring — ela é usada pelo LLM para entender quando chamar a tool:

```python
@mcp.tool
async def consultar_cambio(moeda: str = "USD", data: str | None = None) -> str:
    """Consulta a cotação de uma moeda no Banco Central do Brasil.

    Use esta tool para obter cotações de câmbio atuais ou históricas.
    Moedas suportadas: USD, EUR, GBP, JPY, ARS, entre outras.

    Args:
        moeda: Código ISO da moeda (ex: USD, EUR). Default: USD.
        data: Data da cotação no formato YYYY-MM-DD. Se omitido, retorna a mais recente.

    Returns:
        Cotação de compra e venda formatada.
    """
```

### Commits

Conventional Commits: `feat(ibge): add tool consultar_populacao`

Prefixos: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `ci`

---

## Decisão 8: AGENTS.md e documentação para AI

O projeto inclui `AGENTS.md` na raiz (padrão AAIF, adotado por 60k+ projetos) com instruções para AI agents que interagem com o código:

```markdown
# AGENTS.md

## Project: mcp-brasil
MCP servers for Brazilian government public APIs.

## Structure
- Package by feature: each API is a self-contained module in `src/mcp_brasil/{feature}/`
- Each feature has: server.py, tools.py, client.py, schemas.py, constants.py

## Code conventions
- Python 3.10+, async/await for all I/O
- Full type annotations on all functions
- Pydantic models for all inputs/outputs
- httpx for async HTTP
- ruff for formatting (line-length 99)

## Testing
- pytest with asyncio_mode=auto
- Tests mirror src structure: tests/{feature}/
- Use respx for HTTP mocking

## Adding a new feature
1. Create directory src/mcp_brasil/{feature}/
2. Add server.py, tools.py, client.py, schemas.py, constants.py
3. Mount in src/mcp_brasil/server.py
4. Add tests in tests/{feature}/
5. Update README with new tools
```

---

## Resumo das Decisões

| # | Decisão | Escolha |
|---|---------|---------|
| 1 | Framework | Python + FastMCP v3 (Prefect) |
| 2 | Organização | Package by Feature (single package, `src/` layout) |
| 3 | Anatomia de feature | server.py → tools.py → client.py → schemas.py → constants.py |
| 4 | Tooling | uv + ruff + pytest + mypy + prek + just + httpx + pydantic |
| 5 | Testes | Espelham features, 3 níveis (unit, mock HTTP, integration) |
| 6 | Distribuição | Pacote único `mcp-brasil` no PyPI |
| 7 | Código | Clean Code: SRP, type hints, docstrings, conventional commits |
| 8 | AI-ready | AGENTS.md + CLAUDE.md na raiz |

---

## Referências

- [FastMCP GitHub](https://github.com/PrefectHQ/fastmcp) — 23.9k stars, Apache-2.0
- [FastMCP Docs](https://gofastmcp.com) — Quickstart, Contributing, API Reference
- [FastMCP Design Principles](https://gofastmcp.com/development/contributing#design-principles) — Fast, Simple, Pythonic, Complete
- [MCP Specification](https://modelcontextprotocol.io) — Protocolo oficial
- [AAIF (Agentic AI Foundation)](https://aaif.io) — Governance do MCP
- [MADR Template](https://adr.github.io/madr/) — Template de ADR utilizado
- [APIs Gov BR](https://catalogodedadosabertos.com.br/Apis) — Catálogo de APIs públicas
