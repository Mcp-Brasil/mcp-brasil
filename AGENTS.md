# AGENTS.md — Agentes de Desenvolvimento do mcp-brasil

Este arquivo define 3 agentes especializados para o desenvolvimento do projeto.
Cada agente tem um papel, responsabilidades e checklists claros.

## IMPORTANTE: Pré-requisito para todos os agentes

**Antes de implementar qualquer coisa, leia todos os ADRs:**
- `plan/adrs/ADR-001-project-bootstrap.md` — Stack, package-by-feature, convenções
- `plan/adrs/ADR-002-auto-registry-pattern.md` — FeatureRegistry, convenção de discovery
- `plan/adrs/ADR-003-redator-oficial.md` — Padrão de agentes (Prompt + Resource + Tool)

Os ADRs são a fonte de verdade. Nenhuma implementação deve contradizê-los.

---

## Tech Lead

**Papel:** Guardião da arquitetura e das decisões técnicas.

### Responsabilidades

- Revisa todas as decisões de arquitetura contra os ADRs (001, 002, 003)
- Valida que novas features seguem o padrão package-by-feature
- Garante que o auto-registry (ADR-002) não é violado:
  - `src/mcp_brasil/server.py` **nunca recebe imports manuais de features**
  - Toda feature tem `FEATURE_META` no `__init__.py`
  - Toda feature tem `mcp: FastMCP` no `server.py`
- Revisa PRs e valida qualidade antes do merge
- Define prioridade de features conforme roadmap (`plan/roadmap.md`)

### Checklist de review

Ao revisar uma nova feature ou PR, verificar:

- [ ] **FEATURE_META** presente no `__init__.py` com todos os campos obrigatórios (`name`, `description`)
- [ ] **Separação de responsabilidades** respeitada:
  - `tools.py` não faz HTTP direto
  - `client.py` não formata para LLM
  - `schemas.py` não tem lógica
  - `server.py` apenas registra tools (zero lógica)
  - `constants.py` sem imports de outros módulos
- [ ] **Imports corretos** — nenhum import circular, nenhum import de `_shared` fora do pacote
- [ ] **Auto-registry intacto** — `server.py` raiz não foi modificado
- [ ] **Type hints** completos em todas as funções
- [ ] **Docstrings** em todas as tools (usadas pelo LLM)
- [ ] **Testes** presentes: `test_tools.py`, `test_client.py`
- [ ] **Commits** seguem Conventional Commits

### Referências

- `plan/adrs/ADR-001-project-bootstrap.md` — Decisões de stack e organização
- `plan/adrs/ADR-002-auto-registry-pattern.md` — Padrão de auto-discovery
- `plan/adrs/ADR-003-redator-oficial.md` — Padrão de agentes com Prompts + Resources + Tools
- `plan/roadmap.md` — Roadmap e prioridades
- `plan/poc-plan.md` — Inventário de APIs e plano de implementação

---

## Developer

**Papel:** Implementa features seguindo os padrões estabelecidos.

### Workflow para criar uma nova feature

Ordem de implementação dentro de cada feature:

```
1. constants.py  → URLs, enums, códigos fixos
2. schemas.py    → Pydantic models (input/output)
3. client.py     → HTTP async para a API
4. tools.py      → Lógica das tools (orquestra client + formata)
5. server.py     → Registra tools no FastMCP
6. __init__.py   → FEATURE_META + exports
```

### Templates de código

#### `constants.py`
```python
"""Constantes da feature {nome}."""

{NOME}_API_BASE = "https://api.exemplo.gov.br"
```

#### `schemas.py`
```python
"""Schemas Pydantic da feature {nome}."""

from pydantic import BaseModel, Field


class ExemploInput(BaseModel):
    campo: str = Field(description="Descrição do campo")


class ExemploOutput(BaseModel):
    id: int
    nome: str
```

#### `client.py`
```python
"""HTTP client para a API {nome}."""

import httpx

from .constants import {NOME}_API_BASE
from .schemas import ExemploOutput


async def buscar_exemplo(param: str) -> list[ExemploOutput]:
    async with httpx.AsyncClient() as client:
        response = await client.get(f"{{NOME}_API_BASE}/endpoint", params={"q": param})
        response.raise_for_status()
        data = response.json()
        return [ExemploOutput(**item) for item in data]
```

#### `tools.py`
```python
"""Tools da feature {nome}."""

from .client import buscar_exemplo


async def buscar_{nome}(query: str) -> str:
    """Busca dados de {nome} na API governamental.

    Use esta tool para consultar {descrição do que faz}.

    Args:
        query: Termo de busca.

    Returns:
        Resultados formatados.
    """
    resultados = await buscar_exemplo(query)
    return "\n".join(f"{r.id} — {r.nome}" for r in resultados)
```

#### `server.py`
```python
"""Server da feature {nome}."""

from fastmcp import FastMCP

from .tools import buscar_{nome}

mcp = FastMCP("mcp-brasil-{nome}")

mcp.tool(buscar_{nome})
```

#### `__init__.py`
```python
"""Feature {nome} — {descrição}."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="{nome}",
    description="{descrição da feature}",
    version="0.1.0",
    api_base="{url_base}",
    requires_auth=False,
)
```

### Regras invioláveis

1. **`tools.py` nunca faz HTTP** — sempre delega para `client.py`
2. **`client.py` nunca formata para LLM** — retorna dados tipados (Pydantic models)
3. **`schemas.py` zero lógica** — apenas `BaseModel` com `Field`
4. **`server.py` apenas registra** — `mcp.tool(fn)`, nenhuma lógica de negócio
5. **`constants.py` zero imports** de outros módulos do projeto
6. **Toda tool tem docstring** — ela é usada pelo LLM para decidir quando chamar
7. **Async everywhere** — `async def` em todas as tools e funções de client
8. **Type hints completos** — mypy strict deve passar
9. **Nunca editar `src/mcp_brasil/server.py`** — auto-registry cuida

### Referências

- `plan/adrs/ADR-001-project-bootstrap.md` — Anatomia de uma feature (Decisão 3)
- `plan/refs/registry/feature.py` — Código de referência do FeatureRegistry
- `plan/refs/registry/server.py` — Código de referência do server raiz

---

## QA

**Papel:** Escreve e mantém testes, valida qualidade do código.

### Estrutura de testes

Testes espelham a estrutura de `src/`:

```
tests/
├── conftest.py                 # Fixtures globais
├── {feature}/
│   ├── test_tools.py           # Testa lógica, mock no client
│   ├── test_client.py          # Testa HTTP com respx
│   └── test_integration.py    # Testa via fastmcp.Client (e2e)
└── _shared/
    └── test_feature.py         # Testa FeatureRegistry
```

### Padrões de teste

#### `test_tools.py` — Mock no client, testa lógica
```python
"""Testes das tools de {feature}."""

from unittest.mock import AsyncMock, patch

import pytest

from mcp_brasil.{feature}.tools import buscar_{feature}


@pytest.mark.asyncio
async def test_buscar_{feature}_retorna_formatado():
    mock_data = [...]  # dados esperados
    with patch("mcp_brasil.{feature}.tools.buscar_exemplo", new_callable=AsyncMock) as mock:
        mock.return_value = mock_data
        resultado = await buscar_{feature}("query")
        assert "esperado" in resultado
```

#### `test_client.py` — Mock HTTP com respx
```python
"""Testes do client HTTP de {feature}."""

import httpx
import pytest
import respx

from mcp_brasil.{feature}.client import buscar_exemplo
from mcp_brasil.{feature}.constants import {FEATURE}_API_BASE


@pytest.mark.asyncio
@respx.mock
async def test_buscar_exemplo_sucesso():
    respx.get(f"{{{FEATURE}_API_BASE}}/endpoint").mock(
        return_value=httpx.Response(200, json=[{"id": 1, "nome": "Teste"}])
    )
    resultado = await buscar_exemplo("query")
    assert len(resultado) == 1
    assert resultado[0].nome == "Teste"
```

#### `test_integration.py` — End-to-end com fastmcp.Client
```python
"""Testes de integração da feature {feature}."""

import pytest
from fastmcp import Client

from mcp_brasil.{feature}.server import mcp


@pytest.mark.asyncio
async def test_tool_{feature}_via_mcp_client():
    async with Client(mcp) as client:
        result = await client.call_tool("buscar_{feature}", {"query": "teste"})
        assert result is not None
```

### Checklist de qualidade

Antes de aprovar qualquer PR:

- [ ] **Lint passa:** `just lint` sem erros
- [ ] **Types passam:** `just types` sem erros
- [ ] **Testes passam:** `just test` sem falhas
- [ ] **Cobertura:** toda tool tem pelo menos 1 teste
- [ ] **Mock HTTP:** `test_client.py` usa `respx`, nunca faz requisição real
- [ ] **Sem secrets:** nenhum token ou API key hardcoded
- [ ] **Docstrings:** todas as tools têm docstring descritiva
- [ ] **Nomes consistentes:** tools em snake_case com verbo (buscar_, consultar_, listar_)

### Comandos

```bash
just test                    # Rodar todos os testes
just test-feature ibge       # Rodar testes de uma feature
just lint                    # Verificar lint
just types                   # Verificar types
just ci                      # Pipeline completa (lint + types + test)
```
