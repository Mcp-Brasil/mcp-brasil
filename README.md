# mcp-brasil

MCP servers para APIs públicas brasileiras.

Um único pacote Python que conecta AI agents (Claude, GPT, etc.) a dados governamentais do Brasil: IBGE, Banco Central, Portal da Transparência, Câmara dos Deputados, Senado Federal, DataJud e mais.

## Quick Start

```bash
# Instalar
uv pip install mcp-brasil

# Rodar server (todas as APIs)
fastmcp run mcp_brasil.server:mcp

# Rodar via HTTP
fastmcp run mcp_brasil.server:mcp --transport http --port 8000
```

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

## Arquitetura

O projeto usa **Package by Feature** com **Auto-Registry**:

- Cada API governamental e uma feature auto-contida em `src/mcp_brasil/{feature}/`
- O server raiz descobre e monta features automaticamente via `FeatureRegistry`
- Para adicionar uma nova feature, basta criar o diretorio com a convencao — zero edição no server raiz

```
src/mcp_brasil/
├── server.py              # Auto-registry (nunca muda)
├── _shared/               # Utilitarios compartilhados
│   └── feature.py         # FeatureMeta + FeatureRegistry
├── ibge/                  # Feature: IBGE
│   ├── __init__.py        # FEATURE_META
│   ├── server.py          # mcp: FastMCP
│   ├── tools.py           # Logica das tools
│   ├── client.py          # HTTP async
│   ├── schemas.py         # Pydantic models
│   └── constants.py       # URLs, codigos
├── bacen/                 # Feature: Banco Central
├── transparencia/         # Feature: Portal da Transparencia
└── ...
```

### Convenção de cada feature

```
server.py  →  tools.py  →  client.py  →  schemas.py
  registra     orquestra    faz HTTP      dados puros
```

Regras:
- `tools.py` nunca faz HTTP direto
- `client.py` nunca formata para LLM
- `schemas.py` apenas Pydantic models
- `server.py` apenas registra tools

## Features planejadas

| Feature | API | Auth | Status |
|---------|-----|------|--------|
| `ibge` | servicodados.ibge.gov.br | Nenhuma | Planejado |
| `bacen` | api.bcb.gov.br | Nenhuma | Planejado |
| `transparencia` | portaldatransparencia.gov.br | API key gratuita | Planejado |
| `camara` | dadosabertos.camara.leg.br | Nenhuma | Planejado |
| `senado` | legis.senado.leg.br | Nenhuma | Planejado |
| `datajud` | api-publica.datajud.cnj.jus.br | API key CNJ | Planejado |
| `brasilapi` | brasilapi.com.br | Nenhuma | Planejado |
| `diario_oficial` | queridodiario.ok.org.br | Nenhuma | Planejado |

## Desenvolvimento

```bash
# Clonar e instalar
git clone https://github.com/seu-usuario/mcp-brasil.git
cd mcp-brasil
just install

# Rodar testes
just test

# Lint + types + testes
just ci

# Inspecionar server
just inspect
```

## Como contribuir

1. Crie um diretorio em `src/mcp_brasil/{feature}/` seguindo a estrutura padrão
2. Exporte `FEATURE_META` no `__init__.py`
3. Exporte `mcp: FastMCP` no `server.py`
4. Adicione testes em `tests/{feature}/`
5. Rode `just ci` e abra um PR

Consulte `AGENTS.md` para detalhes dos padrões e templates de código.

## Documentação

- `plan/adrs/` — Decisões de arquitetura
- `plan/roadmap.md` — Roadmap técnico
- `plan/poc-plan.md` — Plano da POC com inventário de 90+ APIs
- `plan/research/` — Mapeamento completo de APIs públicas brasileiras

## Licença

MIT
