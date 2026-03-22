# ADR-002: Auto-Registry de Features via Convention-Based Discovery

## Status

**Aceito** — 2026-03-21

## Contexto

No ADR-001, definimos Package by Feature como organização do projeto. Cada feature (ibge, bacen, transparencia, etc.) é um diretório auto-contido com um `server.py` que exporta um objeto `mcp` do tipo `FastMCP`.

O problema: o `server.py` raiz precisa importar e montar cada feature manualmente:

```python
# ❌ Acoplamento manual — precisa editar a cada nova feature
from .ibge.server import mcp as ibge
from .bacen.server import mcp as bacen
from .transparencia.server import mcp as transparencia

mcp = FastMCP("mcp-brasil 🇧🇷")
mcp.mount("/ibge", ibge)
mcp.mount("/bacen", bacen)
mcp.mount("/transparencia", transparencia)
```

Isso viola o Open-Closed Principle: para adicionar uma feature, você precisa modificar um arquivo existente. Queremos que adicionar uma nova feature seja **zero-touch** no server raiz — basta criar o diretório com a convenção correta e ele é descoberto automaticamente.

---

## Opções Consideradas

### Opção A: Import manual (status quo)
```python
from .ibge.server import mcp as ibge
mcp.mount("/ibge", ibge)
```
- **Prós:** Explícito, simples, sem mágica
- **Contras:** Viola Open-Closed, precisa editar `server.py` a cada feature, propenso a esquecer

### Opção B: Lista declarativa (array de nomes)
```python
FEATURES = ["ibge", "bacen", "transparencia"]
for name in FEATURES:
    module = importlib.import_module(f".{name}.server", package="mcp_brasil")
    mcp.mount(f"/{name}", module.mcp)
```
- **Prós:** Centralizado, fácil de desabilitar
- **Contras:** Ainda precisa editar a lista

### Opção C: `pkgutil.iter_modules()` auto-discovery (padrão Flask/pytest)
```python
# Descobre automaticamente todos os subpacotes com server.py
for _, name, ispkg in pkgutil.iter_modules(package.__path__):
    if ispkg and not name.startswith("_"):
        module = importlib.import_module(f".{name}.server", package="mcp_brasil")
        mcp.mount(f"/{name}", module.mcp)
```
- **Prós:** Zero-touch, padrão da comunidade Python (Flask, pytest, Django), Open-Closed compliant
- **Contras:** "Mágica" implícita, precisa de convenção clara

### Opção D: Entry points (setuptools)
```toml
[project.entry-points."mcp_brasil.features"]
ibge = "mcp_brasil.ibge.server:mcp"
```
- **Prós:** Padrão PyPA oficial, suporta plugins de terceiros
- **Contras:** Overengineering para um pacote único, requer install para funcionar

### Opção E: Híbrido — Auto-discovery + Feature Protocol + Registry class
Combina C com um protocolo explícito e metadados por feature:
```python
# Cada feature declara seus metadados em __init__.py
FEATURE_META = FeatureMeta(
    name="ibge",
    description="Instituto Brasileiro de Geografia e Estatística",
    version="0.1.0",
    api_base="https://servicodados.ibge.gov.br/api",
    requires_auth=False,
)
```
- **Prós:** Auto-discovery + metadados ricos + validação em tempo de import + introspection para docs
- **Contras:** Mais complexo que C puro

---

## Decisão: Opção E — Híbrido com Feature Protocol

Usamos `pkgutil.iter_modules()` para discovery automático, combinado com um **Protocol** (typing) que cada feature deve implementar, e uma **classe Registry** que valida, registra, e expõe metadados.

**Por quê:**
1. Auto-discovery elimina edição manual (Open-Closed Principle)
2. O Protocol garante que cada feature tem a estrutura correta (fail-fast)
3. Os metadados permitem gerar docs, CLI inspect, e health checks automaticamente
4. É o padrão usado por Flask (blueprints), Django (apps), pytest (plugins), e FastAPI (routers)
5. Um contribuidor cria o diretório, segue a convenção, e é descoberto automaticamente

---

## Implementação

### 1. Feature Metadata — `src/mcp_brasil/_shared/feature.py`

```python
"""Feature metadata and protocol for auto-registry."""

from __future__ import annotations

import importlib
import logging
import pkgutil
from dataclasses import dataclass, field
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from fastmcp import FastMCP

logger = logging.getLogger(__name__)


@dataclass(frozen=True)
class FeatureMeta:
    """Metadados declarativos de uma feature.

    Cada feature declara um FEATURE_META no seu __init__.py.
    O registry usa esses metadados para discovery, validação e docs.
    """

    name: str
    description: str
    version: str = "0.1.0"
    api_base: str = ""
    requires_auth: bool = False
    auth_env_var: str | None = None
    enabled: bool = True
    tags: list[str] = field(default_factory=list)


@dataclass
class RegisteredFeature:
    """Feature descoberta e validada pelo registry."""

    meta: FeatureMeta
    server: FastMCP
    module_path: str


class FeatureRegistry:
    """Auto-registry que descobre, valida e monta features.

    Usa pkgutil.iter_modules() para escanear subpacotes de mcp_brasil,
    importar os que seguem a convenção (têm FEATURE_META + server.mcp),
    e montar no server raiz via FastMCP.mount().

    Padrão inspirado em: Flask blueprints, Django app registry,
    pytest plugin discovery, FastAPI router auto-include.

    Usage:
        from fastmcp import FastMCP
        from mcp_brasil._shared.feature import FeatureRegistry

        mcp = FastMCP("mcp-brasil 🇧🇷")
        registry = FeatureRegistry()
        registry.discover()
        registry.mount_all(mcp)
    """

    def __init__(self) -> None:
        self._features: dict[str, RegisteredFeature] = {}

    @property
    def features(self) -> dict[str, RegisteredFeature]:
        """Features descobertas e registradas."""
        return dict(self._features)

    def discover(self, package_name: str = "mcp_brasil") -> None:
        """Descobre todas as features no pacote.

        Escaneia subpacotes de `package_name` que:
        1. Não começam com '_' (ignora _shared, __pycache__)
        2. Têm um __init__.py com FEATURE_META: FeatureMeta
        3. Têm um server.py com `mcp`: FastMCP

        Args:
            package_name: Pacote base para escanear. Default: "mcp_brasil".
        """
        package = importlib.import_module(package_name)

        for finder, name, ispkg in pkgutil.iter_modules(
            package.__path__, package.__name__ + "."
        ):
            # Pula módulos privados e não-pacotes
            short_name = name.rsplit(".", 1)[-1]
            if not ispkg or short_name.startswith("_"):
                continue

            try:
                self._try_register(name, short_name)
            except Exception as exc:
                logger.warning(
                    "Feature '%s' ignorada: %s",
                    short_name,
                    exc,
                )

    def _try_register(self, module_path: str, short_name: str) -> None:
        """Tenta importar e registrar uma feature."""
        # 1. Importar __init__.py da feature
        feature_module = importlib.import_module(module_path)

        # 2. Verificar FEATURE_META
        meta = getattr(feature_module, "FEATURE_META", None)
        if meta is None:
            raise ValueError(f"Sem FEATURE_META em {module_path}.__init__")

        if not isinstance(meta, FeatureMeta):
            raise TypeError(
                f"FEATURE_META em {module_path} não é FeatureMeta"
            )

        # 3. Feature desabilitada?
        if not meta.enabled:
            logger.info("Feature '%s' está desabilitada, pulando.", short_name)
            return

        # 4. Verificar auth se necessária
        if meta.requires_auth and meta.auth_env_var:
            import os

            if not os.environ.get(meta.auth_env_var):
                logger.warning(
                    "Feature '%s' requer %s (não definida), pulando.",
                    short_name,
                    meta.auth_env_var,
                )
                return

        # 5. Importar server.py e pegar o objeto mcp
        server_module = importlib.import_module(f"{module_path}.server")
        server = getattr(server_module, "mcp", None)

        if server is None:
            raise ValueError(f"Sem `mcp` em {module_path}.server")

        # 6. Registrar
        self._features[short_name] = RegisteredFeature(
            meta=meta,
            server=server,
            module_path=module_path,
        )
        logger.info(
            "Feature '%s' v%s registrada (%d tools)",
            meta.name,
            meta.version,
            len(server._tool_manager._tools) if hasattr(server, '_tool_manager') else 0,
        )

    def mount_all(self, root_server: FastMCP) -> None:
        """Monta todas as features descobertas no server raiz.

        Cada feature é montada em /{feature_name}.

        Args:
            root_server: O FastMCP server raiz.
        """
        for name, feature in sorted(self._features.items()):
            root_server.mount(f"/{name}", feature.server)
            logger.info("Montada: /%s — %s", name, feature.meta.description)

    def summary(self) -> str:
        """Retorna um resumo das features registradas (útil para logs e docs)."""
        lines = [f"mcp-brasil — {len(self._features)} features registradas:\n"]
        for name, feat in sorted(self._features.items()):
            auth = "🔑" if feat.meta.requires_auth else "🔓"
            lines.append(
                f"  /{name:<20} {auth} {feat.meta.description}"
            )
        return "\n".join(lines)
```

### 2. Feature exemplo — `src/mcp_brasil/ibge/__init__.py`

```python
"""Feature IBGE — Instituto Brasileiro de Geografia e Estatística."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="ibge",
    description="Dados do IBGE: localidades, população, PIB, nomes, malhas",
    version="0.1.0",
    api_base="https://servicodados.ibge.gov.br/api",
    requires_auth=False,
    tags=["geodados", "censo", "indicadores"],
)
```

### 3. Feature com auth — `src/mcp_brasil/transparencia/__init__.py`

```python
"""Feature Portal da Transparência — Governo Federal."""

from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="transparencia",
    description="Contratos, despesas, servidores e licitações do governo federal",
    version="0.1.0",
    api_base="https://api.portaldatransparencia.gov.br/api-de-dados",
    requires_auth=True,
    auth_env_var="TRANSPARENCIA_API_KEY",
    tags=["governo", "contratos", "despesas"],
)
```

### 4. Server raiz — `src/mcp_brasil/server.py`

```python
"""Server raiz do mcp-brasil.

Usa FeatureRegistry para auto-discovery e mount de features.
Zero imports manuais — basta criar o diretório com a convenção.
"""

import logging

from fastmcp import FastMCP

from ._shared.feature import FeatureRegistry

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

mcp = FastMCP("mcp-brasil 🇧🇷")

# Auto-discovery: escaneia todos os subpacotes com FEATURE_META
registry = FeatureRegistry()
registry.discover()
registry.mount_all(mcp)

# Log resumo
logger.info(registry.summary())

if __name__ == "__main__":
    mcp.run()
```

### 5. Como adicionar uma nova feature (zero-touch no server raiz)

```bash
# 1. Criar o diretório com a estrutura padrão
mkdir -p src/mcp_brasil/inep

# 2. Criar __init__.py com FEATURE_META
cat > src/mcp_brasil/inep/__init__.py << 'EOF'
from mcp_brasil._shared.feature import FeatureMeta

FEATURE_META = FeatureMeta(
    name="inep",
    description="Dados educacionais: ENEM, IDEB, censo escolar",
    version="0.1.0",
    api_base="https://api.inep.gov.br",
    requires_auth=False,
    tags=["educacao", "enem"],
)
EOF

# 3. Criar server.py, tools.py, client.py, schemas.py, constants.py
# ... (seguindo a convenção do ADR-001)

# 4. Rodar — a feature é descoberta automaticamente!
fastmcp run mcp_brasil.server:mcp
# INFO: Feature 'inep' v0.1.0 registrada
# INFO: Montada: /inep — Dados educacionais: ENEM, IDEB, censo escolar
```

**Nenhum arquivo existente é modificado.** O server raiz continua exatamente igual.

---

## Fluxo de Discovery (diagrama)

```
fastmcp run mcp_brasil.server:mcp
          │
          ▼
   FeatureRegistry.discover()
          │
          ▼
   pkgutil.iter_modules(mcp_brasil)
          │
          ├── ibge/          → tem FEATURE_META? ✅ → tem server.mcp? ✅ → REGISTRA
          ├── bacen/         → tem FEATURE_META? ✅ → tem server.mcp? ✅ → REGISTRA
          ├── transparencia/ → tem FEATURE_META? ✅ → auth ok? ✅        → REGISTRA
          ├── camara/        → tem FEATURE_META? ✅ → tem server.mcp? ✅ → REGISTRA
          ├── _shared/       → começa com _? SIM → PULA
          └── datajud/       → tem FEATURE_META? ✅ → auth ok? ❌        → PULA (log warning)
          │
          ▼
   FeatureRegistry.mount_all(mcp)
          │
          ├── mcp.mount("/ibge", ibge_server)
          ├── mcp.mount("/bacen", bacen_server)
          ├── mcp.mount("/camara", camara_server)
          └── mcp.mount("/transparencia", transparencia_server)
          │
          ▼
   Server rodando com 4 features, ~30 tools
```

---

## Checklist de convenção (para CONTRIBUTING.md)

Para uma feature ser auto-descoberta, ela **precisa**:

1. ☐ Ser um subpacote de `src/mcp_brasil/` (diretório com `__init__.py`)
2. ☐ **Não** começar com `_` (underscore = privado)
3. ☐ Ter `FEATURE_META: FeatureMeta` exportado no `__init__.py`
4. ☐ Ter `mcp: FastMCP` exportado no `server.py`
5. ☐ Se `requires_auth=True`, ter `auth_env_var` definido

Para uma feature **não** ser descoberta:

- Setar `enabled=False` no `FEATURE_META`
- Ou não definir `FEATURE_META` no `__init__.py`
- Ou nomear o diretório com `_` prefix (ex: `_experimental/`)

---

## Benefícios para contribuidores open-source

| Benefício | Como o Registry ajuda |
|-----------|----------------------|
| **Onboarding rápido** | Cria diretório, segue template, funciona |
| **Zero conflitos de merge** | Nunca toca arquivo compartilhado |
| **Testável isolado** | `fastmcp run mcp_brasil.inep.server:mcp` |
| **Feature flags grátis** | `enabled=False` no FEATURE_META |
| **Auth segura** | Features com auth faltante são silenciosamente puladas |
| **Introspection** | `registry.summary()` lista tudo para docs e CI |

---

## Consequências

**Positivas:**
- Adicionar feature = criar diretório (Open-Closed Principle ✅)
- Server raiz nunca muda depois de escrito
- Features com auth faltante não quebram o server inteiro
- Metadados ricos para docs automáticas, CLI inspect, health checks
- Contribuidores podem trabalhar em features paralelas sem conflito

**Negativas:**
- Mais "mágica" que imports explícitos (mitigado pela documentação clara)
- Erro no `__init__.py` de uma feature pode ser silencioso (mitigado pelo logging)
- Ordem de mount é alfabética, não declarativa (aceitável para MCP)

**Trade-off aceito:** Preferimos convention over configuration. O custo de documentar a convenção é menor que o custo de manter imports manuais em um projeto com 10+ features.

---

## Referências

- [Python Packaging Guide: Creating and Discovering Plugins](https://packaging.python.org/en/latest/guides/creating-and-discovering-plugins/)
- [Flask Blueprint auto-discovery pattern](https://flask.palletsprojects.com/en/stable/blueprints/)
- [Django App Registry](https://docs.djangoproject.com/en/5.0/ref/applications/)
- [pytest plugin discovery via pkgutil](https://docs.pytest.org/en/latest/how-to/writing_plugins.html)
- [FastAPI auto-register routers (Medium)](https://medium.com/@bhagyarana80/how-i-built-a-plugin-driven-fastapi-backend-that-auto-registers-routes-e815a7298c29)
- [FastMCP mount() API](https://gofastmcp.com/servers/composition)
