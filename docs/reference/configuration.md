# Configuracao

## Variaveis de ambiente

### Chaves de API

| Variavel | Default | Descricao |
|----------|---------|-----------|
| `TRANSPARENCIA_API_KEY` | — | Chave do Portal da Transparencia (opcional, cadastro gratuito) |
| `DATAJUD_API_KEY` | — | Chave do DataJud/CNJ (opcional, cadastro gratuito) |
| `MAPBIOMAS_EMAIL` | — | E-mail da conta no MapBiomas Alerta (obrigatoria para a feature `mapbiomas_alerta`) |
| `MAPBIOMAS_PASSWORD` | — | Senha da conta no MapBiomas Alerta (obrigatoria para a feature `mapbiomas_alerta`) |
| `ANTHROPIC_API_KEY` | — | Necessaria para `recomendar_tools` e `planejar_consulta` |

### Configuracao do server

| Variavel | Default | Descricao |
|----------|---------|-----------|
| `MCP_BRASIL_TOOL_SEARCH` | `bm25` | Modo de discovery de tools |
| `MCP_BRASIL_HTTP_TIMEOUT` | `30.0` | Timeout HTTP em segundos |
| `MCP_BRASIL_HTTP_MAX_RETRIES` | `3` | Maximo de retentativas HTTP |

### Datasets locais

| Variavel | Default | Descricao |
|----------|---------|-----------|
| `MCP_BRASIL_DATASETS` | "" | Lista CSV de datasets locais a ativar. Ex: `spu_siapa,tse_candidatos,tse_bens` |
| `MCP_BRASIL_DATASET_CACHE_DIR` | `~/.cache/mcp-brasil` | Diretorio raiz do cache DuckDB |
| `MCP_BRASIL_DATASET_REFRESH` | `auto` | `auto` (respeita TTL), `never` (so cache), `force` (sempre baixar) |
| `MCP_BRASIL_DATASET_TIMEOUT` | `600` | Timeout (s) do download — aumente para datasets >500MB |
| `MCP_BRASIL_DATASET_MAX_CACHE_GB` | `20` | Limite soft do cache total em GB |
| `MCP_BRASIL_LGPD_ALLOW_PII` | "" | Lista CSV de datasets com PII liberada (default mascara CPF/email) |

Ver [guia de Datasets locais](../guide/datasets.md) para detalhes.

### `MCP_BRASIL_TOOL_SEARCH`

Controla como as 435 tools sao expostas ao LLM:

| Valor | Comportamento | Quando usar |
|-------|---------------|-------------|
| `bm25` | Filtra para top-10 mais relevantes por contexto; meta-tools sempre visiveis | Default — recomendado para a maioria dos usos |
| `none` | Todas as 435 tools visiveis sem filtragem | Debug, ou quando o LLM tem contexto grande |
| `code_mode` | Experimental — discovery programatico via `get_tags`, `search`, `GetSchemas` | Testes avancados (requer `pydantic-monty`) |

## Chaves de API

### Portal da Transparencia

Chave gratuita que aumenta o rate limit. Sem chave, a feature `transparencia` e desativada.

1. Acesse [portaldatransparencia.gov.br/api-de-dados/cadastrar-email](http://portaldatransparencia.gov.br/api-de-dados/cadastrar-email)
2. Cadastre seu email
3. Copie a chave recebida
4. Configure: `TRANSPARENCIA_API_KEY=sua-chave`

### DataJud / CNJ

Chave gratuita para acesso a processos judiciais. Sem chave, a feature `datajud` e desativada.

1. Acesse [datajud-wiki.cnj.jus.br/api-publica/acesso](https://datajud-wiki.cnj.jus.br/api-publica/acesso)
2. Siga o processo de cadastro
3. Configure: `DATAJUD_API_KEY=sua-chave`

### MapBiomas Alerta

Conta gratuita para acessar os alertas de supressao de vegetacao nativa. Sem as credenciais, a feature `mapbiomas_alerta` e desativada.

1. Acesse [plataforma.alerta.mapbiomas.org/sign-up](https://plataforma.alerta.mapbiomas.org/sign-up)
2. Crie a conta e confirme o cadastro
3. Configure: `MAPBIOMAS_EMAIL=seu-email` e `MAPBIOMAS_PASSWORD=sua-senha`

### Anthropic API

Necessaria apenas para as meta-tools `recomendar_tools` e `planejar_consulta`. Se nao configurada, essas tools retornam erro — as demais 201+ tools funcionam normalmente.

Configure: `ANTHROPIC_API_KEY=sua-chave`

## Configuracao por cliente

### Claude Desktop

```json
{
  "mcpServers": {
    "mcp-brasil": {
      "command": "uvx",
      "args": ["--from", "mcp-brasil", "python", "-m", "mcp_brasil.server"],
      "env": {
        "TRANSPARENCIA_API_KEY": "sua-chave",
        "DATAJUD_API_KEY": "sua-chave",
        "ANTHROPIC_API_KEY": "sua-chave",
        "MCP_BRASIL_TOOL_SEARCH": "bm25"
      }
    }
  }
}
```

### VS Code / Cursor

```json
{
  "servers": {
    "mcp-brasil": {
      "command": "uvx",
      "args": ["--from", "mcp-brasil", "python", "-m", "mcp_brasil.server"],
      "env": {
        "TRANSPARENCIA_API_KEY": "sua-chave",
        "DATAJUD_API_KEY": "sua-chave"
      }
    }
  }
}
```

### Claude Code

```bash
claude mcp add mcp-brasil -- uvx --from mcp-brasil python -m mcp_brasil.server
```

Para adicionar env vars:

```bash
claude mcp add mcp-brasil \
  -e TRANSPARENCIA_API_KEY=sua-chave \
  -e DATAJUD_API_KEY=sua-chave \
  -- uvx --from mcp-brasil python -m mcp_brasil.server
```

### HTTP

```bash
TRANSPARENCIA_API_KEY=xxx DATAJUD_API_KEY=yyy \
  fastmcp run mcp_brasil.server:mcp --transport http --port 8000
```

## HTTP Client

O httpx client compartilhado tem comportamento configuravel:

### Retry com backoff

Retenta automaticamente em:
- **429** — Too Many Requests (respeita `Retry-After` header)
- **5xx** — Server errors (502, 503, 504)
- **Timeout** — Quando a API demora mais que `MCP_BRASIL_HTTP_TIMEOUT`
- **Erros de conexao** — Network unreachable, DNS failure

Backoff exponencial: 1s → 2s → 4s (com jitter), ate `MCP_BRASIL_HTTP_MAX_RETRIES` tentativas.

### Rate limiting

Cada feature pode usar o `RateLimiter` do `_shared/rate_limiter.py`:

```python
limiter = RateLimiter(max_requests=5, period=1.0)  # 5 req/s

async with limiter:
    data = await http_get(url)
```
