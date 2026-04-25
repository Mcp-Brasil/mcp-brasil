<div align="center">

<img src="docs/assets/logo.png" alt="mcp-brasil logo" width="100">

# mcp-brasil

**MCP Server para 70 fontes de dados públicas brasileiras**


[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

533 tools · 131 resources · 102 prompts · 70 features · 15 áreas temáticas

Conecte AI agents (Claude, GPT, Copilot, etc.) a dados governamentais do Brasil — economia, legislação, transparência, judiciário, eleições, meio ambiente, saúde, educação, segurança pública, fiscal subnacional, aviação e mais.

**66 APIs não requerem chave** · 4 usam chaves gratuitas (cadastro em 1 min)

[Quick Start](#quick-start) · [Fontes de dados](#fontes-de-dados) · [Documentação](#documentação) · [Desenvolvimento](#desenvolvimento)

</div>

---

## Features

- **533 tools** em 70 features cobrindo 15 áreas — economia, legislativo, transparência, judiciário, eleitoral, ambiental, saúde, educação, segurança pública, compras públicas, fiscal subnacional, aviação, energia e mais
- **Datasets grandes com cache local** — SIAPA (~813k imóveis), TSE 2014-2024 (candidatos, bens, votação, redes sociais, FEFC), ANP preços de combustíveis, INEP Censo Escolar e ENEM, ISP-RJ segurança pública, ANAC aeronaves e voos regulares — SQL via DuckDB embedded, opt-in via env
- **Cross-referencing** com `planejar_consulta` — cria planos de execução combinando múltiplas APIs (ex: gastos de um deputado + votações + proposições)
- **Execução em lote** com `executar_lote` — dispara consultas em paralelo numa única chamada
- **Smart discovery** — BM25 search transform filtra 533 tools para só mostrar as relevantes ao contexto
- **Auto-registry** — adicionar uma feature é criar uma pasta; zero configuração manual
- **Async everywhere** — httpx async + Pydantic v2 + rate limiting com backoff

## Quick Start

### Instalar

```bash
pip install mcp-brasil
```

```bash
uv add mcp-brasil
```

### Claude Desktop

Adicione ao `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "mcp-brasil": {
      "command": "uvx",
      "args": ["--from", "mcp-brasil", "python", "-m", "mcp_brasil.server"],
      "env": {
        "TRANSPARENCIA_API_KEY": "sua-chave-aqui",
        "DATAJUD_API_KEY": "sua-chave-aqui",
        "META_ACCESS_TOKEN": "seu-token-aqui"
      }
    }
  }
}
```

> As chaves são opcionais — sem elas, as 36 APIs restantes funcionam normalmente.

### VS Code / Cursor

Crie `.vscode/mcp.json` na raiz do projeto:

```json
{
  "servers": {
    "mcp-brasil": {
      "command": "uvx",
      "args": ["--from", "mcp-brasil", "python", "-m", "mcp_brasil.server"],
      "env": {
        "TRANSPARENCIA_API_KEY": "sua-chave-aqui",
        "DATAJUD_API_KEY": "sua-chave-aqui",
        "META_ACCESS_TOKEN": "seu-token-aqui"
      }
    }
  }
}
```

### Claude Code

```bash
claude mcp add mcp-brasil -- uvx --from mcp-brasil python -m mcp_brasil.server
```

### HTTP (outros clientes)

```bash
fastmcp run mcp_brasil.server:mcp --transport http --port 8000
# Server disponível em http://localhost:8000/mcp
```

## Exemplos

Conecte o server e faça perguntas em linguagem natural:

> **Legislativo:** "Quais projetos de lei sobre inteligência artificial tramitaram na Câmara em 2024? Quem foram os autores?"

> **Econômico:** "Qual a tendência da taxa Selic nos últimos 12 meses? Compare com a inflação (IPCA) no mesmo período."

> **Transparência:** "Quais os 10 maiores contratos do governo federal em 2024? Quem são os fornecedores?"

> **Cross-reference:** "Compare os gastos per capita com saúde em São Paulo e Minas Gerais cruzando dados do TCE-SP e IBGE."

> **Judiciário:** "Busque processos sobre licitação irregular no TCU. Quais foram as penalidades aplicadas?"

> **Eleitoral:** "Quais os maiores doadores da campanha do candidato X? Qual o total arrecadado?"

## Fontes de dados

### Economia e Finanças

| Feature | Fonte | Tools |
|---------|-------|:-----:|
| `bacen` | Banco Central — Selic, IPCA, câmbio, PIB e +190 séries temporais (SGS) | 9 |
| `bcb_olinda` | Banco Central — PTAX (câmbio oficial), Expectativas Focus, taxas de juros bancárias | 8 |
| `bndes` | BNDES — operações de financiamento, desembolsos, instituições credenciadas | 4 |
| `ipeadata` | IPEADATA/Ipea — séries macro, regionais e sociais históricas (OData) | 5 |

### Geografia e Estatística

| Feature | Fonte | Tools |
|---------|-------|:-----:|
| `ibge` | IBGE — estados, municípios, nomes, agregados estatísticos | 9 |

### Legislativo e Executivo

| Feature | Fonte | Tools |
|---------|-------|:-----:|
| `camara` | Câmara dos Deputados — deputados, proposições, votações, despesas | 11 |
| `senado` | Senado Federal — senadores, matérias, votações, comissões | 26 |
| `governadores` | Governadores de todos os estados em exercício (nome, partido, mandato) | 4 |

### Transparência e Fiscalização

| Feature | Fonte | Tools |
|---------|-------|:-----:|
| `transparencia` | Portal da Transparência — contratos, despesas, servidores, sanções, imóveis funcionais, renúncias fiscais, órgãos, COVID-19 | 54 |
| `tcu` | Tribunal de Contas da União — acórdãos, inidôneos, débitos, pautas | 9 |
| `tce_sp` | TCE-SP — despesas e receitas de 645 municípios paulistas | 3 |
| `tce_rj` | TCE-RJ — licitações, contratos, obras, penalidades, concessões | 7 |
| `tce_rs` | TCE-RS — educação, saúde, gestão fiscal (LRF) | 5 |
| `tce_pe` | TCE-PE — licitações, contratos, despesas, fornecedores | 5 |
| `tce_ce` | TCE-CE — licitações, contratos, empenhos | 4 |
| `tce_es` | TCE-ES — licitações, contratos, obras públicas | 4 |
| `tce_rn` | TCE-RN — jurisdicionados, licitações, contratos | 5 |
| `tce_pi` | TCE-PI — prefeituras, despesas, receitas | 5 |
| `tce_sc` | TCE-SC — municípios e unidades gestoras | 2 |
| `tce_to` | TCE-TO — processos, pautas de sessões | 3 |
| `tce_pa` | TCE-PA — Diário Oficial, sessões plenárias, jurisprudência (acórdãos, resoluções, portarias, prejulgados) e conteúdo informativo | 4 |
| `spu_geo` | SPU GeoPortal — terrenos de marinha, acrescidos, marginais, ilhas federais, praias, manguezais e localização de imóveis da União | 4 |
| `spu_imoveis` | Imóveis da União (Raio-X APF / Gov360) — busca por órgão, UF, município, regime e agregações institucionais | 4 |
| `siconfi` | SICONFI/Tesouro Nacional — RREO, RGF, DCA e MSC de municípios/estados (LRF, LC 101) | 7 |
| `mj` | MJSP (CKAN) — SINESP, INFOPEN, PROCONs/Sindec, armas, sistema prisional | 4 |

### Judiciário

| Feature | Fonte | Tools |
|---------|-------|:-----:|
| `datajud` | DataJud/CNJ — processos judiciais, movimentações | 7 |
| `jurisprudencia` | STF, STJ e TST — acórdãos, súmulas, decisões | 6 |

### Eleitoral

| Feature | Fonte | Tools |
|---------|-------|:-----:|
| `tse` | TSE — eleições, candidatos, prestação de contas | 15 |
| `anuncios_eleitorais` | Biblioteca de Anúncios da Meta — propaganda eleitoral na internet | 6 |

### Meio Ambiente

| Feature | Fonte | Tools |
|---------|-------|:-----:|
| `inpe` | INPE — focos de queimadas, desmatamento DETER/PRODES | 4 |
| `ana` | ANA — estações hidrológicas, telemetria, reservatórios | 3 |
| `ibama` | IBAMA (CKAN) — autos de infração, CTF/APP, TCFA, biomas | 4 |

### Energia e Infraestrutura

| Feature | Fonte | Tools |
|---------|-------|:-----:|
| `aneel` | ANEEL (CKAN) — SIGA, geração distribuída, tarifas, bandeiras | 4 |
| `antt` | ANTT (CKAN) — rodovias concedidas, acidentes, cargas, passageiros | 4 |

### Aviação

| Feature | Fonte | Tools |
|---------|-------|:-----:|
| `opensky` | OpenSky Network — voos ao vivo via ADS-B (rastreamento, tracks) | 6 |

### Saúde

| Feature | Fonte | Tools |
|---------|-------|:-----:|
| `saude` | CNES/DataSUS — estabelecimentos, profissionais, leitos | 10 |
| `opendatasus` | OpenDataSUS — datasets de saúde pública (CKAN) | 7 |
| `anvisa` | ANVISA — bulário, medicamentos, preços CMED, registros | 10 |
| `denasus` | DENASUS — auditorias do SUS | 5 |
| `imunizacao` | SI-PNI — vacinação, calendário, cobertura vacinal, SRAG | 10 |
| `bps` | BPS — preços de medicamentos e dispositivos médicos no SUS | 3 |
| `farmacia_popular` | Farmácia Popular — medicamentos gratuitos, farmácias credenciadas | 8 |
| `rename` | RENAME — medicamentos essenciais do SUS por princípio ativo | 5 |

### Segurança Pública

| Feature | Fonte | Tools |
|---------|-------|:-----:|
| `atlas_violencia` | Atlas da Violência (IPEA/FBSP) — homicídios, violência por gênero/raça, armas de fogo | 7 |
| `sinesp` | SINESP/MJSP — datasets de segurança pública, sistema penitenciário | 6 |
| `forum_seguranca` | Fórum Brasileiro de Segurança Pública — publicações, Anuário | 4 |

### Educação

| Feature | Fonte | Tools |
|---------|-------|:-----:|
| `inep` | INEP — índices educacionais (IDEB, indicadores) e pesquisas | 4 |
| `fnde` | FNDE — repasses, merenda escolar, PNATE, transporte | 4 |

### Compras Públicas

| Feature | Fonte | Tools |
|---------|-------|:-----:|
| `compras` | PNCP + ComprasNet/SIASG — licitações, contratos, pregões, CATMAT | 29 |
| `transferegov` | TransfereGov — emendas parlamentares PIX | 5 |

### Mercado Financeiro e Notícias

| Feature | Fonte | Tools |
|---------|-------|:-----:|
| `b3` | Bolsa B3 (brapi.dev) — cotações ações/FIIs/BDRs, históricos, índices | 6 |
| `noticias` | RSS agregado — Câmara, Senado, Agência Brasil, BCB | 4 |

### Dados Abertos e Utilidades

| Feature | Fonte | Tools |
|---------|-------|:-----:|
| `brasilapi` | BrasilAPI — CEP, CNPJ, DDD, bancos, câmbio, FIPE, PIX | 16 |
| `dados_abertos` | Dados Abertos (dados.gov.br) — catálogo de datasets federais | 4 |
| `diario_oficial` | Querido Diário + DOU — diários oficiais de 5.000+ cidades e da União | 11 |
| `tabua_mares` | Tábua de Marés — previsão de marés para portos do litoral | 7 |

### Datasets locais (opt-in via env)

Features que baixam CSVs/ZIPs grandes (~100MB–1,6GB) para **DuckDB embedded local**
e expõem SQL via tools canned. Ativadas apenas quando listadas em `MCP_BRASIL_DATASETS`.
Primeira carga: minutos (download + ingest); subsequentes: ms.

| Feature | Fonte | Período | Tools |
|---------|-------|---------|:-----:|
| `spu_siapa` | SPU — imóveis da União (SIAPA completo, 813k imóveis com dominiais + uso especial) | 2026 snapshot | 8 |
| `tse_candidatos` | TSE — candidatos de todas as eleições (~4M registros) | 2014-2024 | 8 |
| `tse_bens` | TSE — bens declarados por candidatos, join via `sq_candidato` | 2014-2024 | 5 |
| `tse_votacao` | TSE — votos por candidato × município × zona | 2014-2024 | 6 |
| `tse_redes_sociais` | TSE — URLs Instagram/Facebook/Twitter dos candidatos | 2018-2024 | 4 |
| `tse_fefc` | TSE — Fundo Eleitoral Especial (distribuição partido × gênero) | 2020, 2024 | 4 |
| `anp_precos` | ANP — preços semanais de combustíveis (gasolina/etanol, diesel/GNV, GLP) em postos | 2024 integral | 7 |
| `inep_censo_escolar` | INEP — Censo Escolar da Educação Básica (~180k escolas, infraestrutura, matrículas) | 2023 | 7 |
| `inep_enem` | INEP — Microdados ENEM (~3,9M inscritos, notas, perfil socioeconômico) | 2023 | 6 |
| `isp_rj` | ISP-RJ — estatísticas criminais mensais por CISP do Rio de Janeiro | 1991-present | 6 |
| `anac_rab` | ANAC — Registro Aeronáutico Brasileiro (~75k aeronaves civis) | snapshot semanal | 6 |
| `anac_vra` | ANAC — Voos Regulares Ativos (pontualidade, atrasos, origem/destino) | 2020-present | — |
| `cvm_fundos` | CVM — Cadastro de Fundos de Investimento (~40k fundos, PL, taxa adm, gestor) | snapshot diário | 6 |

Ative com:
```bash
# .env
MCP_BRASIL_DATASETS=tse_candidatos,tse_bens,tse_votacao
```

Ver [guia de Datasets locais](docs/guide/datasets.md) para detalhes de uso.

> O server raiz também expõe 5 meta-tools: `listar_features`, `recomendar_tools`,
> `planejar_consulta`, `executar_lote` e `listar_datasets_disponiveis`.

## Chaves de API

| API | Obrigatória? | Como obter |
|-----|-------------|------------|
| Portal da Transparência | Opcional | [Cadastro gratuito](https://portaldatransparencia.gov.br/api-de-dados/cadastrar-email) |
| DataJud/CNJ | Opcional | [Cadastro gratuito](https://datajud-wiki.cnj.jus.br/api-publica/acesso) |
| Anúncios Eleitorais (Meta) | Opcional | [Meta Ad Library API](https://www.facebook.com/ads/library/api/) |
| brapi.dev (B3) | Opcional | [Token gratuito](https://brapi.dev/dashboard) para índices e múltiplas ações |
| Todas as outras (62) | Nenhuma chave | — |

Configure via variáveis de ambiente ou `.env`:

```bash
TRANSPARENCIA_API_KEY=sua-chave
DATAJUD_API_KEY=sua-chave
META_ACCESS_TOKEN=seu-token
```

## Configuração

| Variável | Default | Descrição |
|----------|---------|-----------|
| `TRANSPARENCIA_API_KEY` | — | Chave do Portal da Transparência |
| `DATAJUD_API_KEY` | — | Chave do DataJud/CNJ |
| `META_ACCESS_TOKEN` | — | Token da [Meta Ad Library API](https://www.facebook.com/ads/library/api/) |
| `MCP_BRASIL_TOOL_SEARCH` | `bm25` | Modo de discovery: `bm25`, `code_mode` ou `none` |
| `MCP_BRASIL_HTTP_TIMEOUT` | `30.0` | Timeout HTTP em segundos |
| `MCP_BRASIL_HTTP_MAX_RETRIES` | `3` | Máximo de retentativas HTTP |
| `MCP_BRASIL_DATASETS` | — | Lista CSV de datasets locais a ativar. Ex: `tse_candidatos,tse_bens` |
| `MCP_BRASIL_DATASET_CACHE_DIR` | `~/.cache/mcp-brasil` | Diretório raiz do cache DuckDB |
| `MCP_BRASIL_DATASET_REFRESH` | `auto` | `auto` (TTL), `never` (só cache) ou `force` (sempre baixar) |
| `MCP_BRASIL_DATASET_TIMEOUT` | `600` | Timeout (s) do download de datasets grandes |
| `MCP_BRASIL_LGPD_ALLOW_PII` | — | Lista CSV de datasets com PII liberada (ex: `tse_candidatos`) |

## Documentação

| Página | Descrição |
|--------|-----------|
| [Quick Start](docs/guide/quickstart.md) | Instalação e configuração em 2 minutos |
| [Arquitetura](docs/concepts/architecture.md) | Como o projeto funciona por dentro |
| [Catálogo de Features](docs/reference/features.md) | Todas as 69 features e 525 tools |
| [Datasets locais (DuckDB)](docs/guide/datasets.md) | SIAPA + TSE 2014-2024 via SQL embedded |
| [Smart Tools](docs/reference/smart-tools.md) | Meta-tools: planner, batch, discovery |
| [Adicionando Features](docs/guide/adding-features.md) | Guia para contribuir com novas APIs |
| [Configuração](docs/reference/configuration.md) | Variáveis de ambiente e opções |
| [Meta Ad Library API](docs/reference/meta-ad-library-api.md) | Referência da API de anúncios eleitorais da Meta |
| [Code Mode](docs/reference/code-mode.md) | Discovery programático + sandbox Python (experimental) |
| [Desenvolvimento](docs/guide/development.md) | Setup de dev, testes, lint, CI |
| [Deploy Azure (datasets)](docs/deploy/azure-datasets.md) | Container Apps + Azure Files + warmup automático dos datasets |

## Desenvolvimento

```bash
git clone https://github.com/jxnxts/mcp-brasil.git
cd mcp-brasil
make dev              # Instalar dependências (prod + dev)
make test             # Rodar todos os testes
make test-feature F=ibge  # Testes de uma feature
make lint             # Lint + format check
make ruff             # Auto-fix lint + format
make types            # mypy strict
make ci               # lint + types + test
make run              # Server stdio
make serve            # Server HTTP :8000
make inspect          # Listar tools/resources/prompts
```

## Arquitetura

O projeto usa **Package by Feature** com **Auto-Registry** — cada feature é uma pasta auto-contida:

```
src/mcp_brasil/
├── server.py              # Auto-registry (nunca editado manualmente)
├── _shared/               # Utilitários compartilhados
│   └── datasets/          # Infra DuckDB local
├── data/                  # 51 features — REST passthrough
│   ├── ibge/
│   │   ├── __init__.py    # FEATURE_META
│   │   ├── server.py      # FastMCP instance
│   │   ├── tools.py       # Lógica das tools
│   │   ├── client.py      # HTTP async
│   │   ├── schemas.py     # Pydantic models
│   │   └── constants.py   # URLs, códigos
│   ├── bacen/
│   └── ...
├── datasets/              # 12 features — cache local DuckDB (opt-in via env)
│   ├── spu_siapa/         # SIAPA 813k imóveis
│   ├── tse_candidatos/    # TSE candidatos 2014-2024
│   └── ...
└── agentes/               # 1 feature — agentes inteligentes
    └── redator/
```

Três modalidades de feature coexistem:

- **`data/`** — REST passthrough: HTTP async → Pydantic → tool formatada
- **`datasets/`** — DuckDB embedded local: CSVs/ZIPs grandes com SQL,
  gated por `MCP_BRASIL_DATASETS`, cache em `~/.cache/mcp-brasil/`
- **`agentes/`** — Agentes inteligentes: tools + prompts + resources
  compondo fluxos complexos (ex: redação oficial)

Para adicionar uma nova feature, basta criar o diretório seguindo a convenção — o registry descobre automaticamente.

## Contribuindo

1. Fork o repositório
2. Crie uma feature em `src/mcp_brasil/data/{feature}/` ou `agentes/{feature}/`
3. Exporte `FEATURE_META` no `__init__.py` e `mcp: FastMCP` no `server.py`
4. Adicione testes em `tests/data/{feature}/`
5. Rode `make ci` e abra um PR

## Disclaimer

Este projeto integra um número significativo de APIs governamentais brasileiras, muitas com documentação inconsistente ou incompleta. Embora todo esforço tenha sido feito para garantir precisão, alguns endpoints podem retornar resultados inesperados ou ter cobertura parcial de parâmetros.

Este é um projeto open-source da comunidade — se encontrar algo quebrado ou que possa ser melhorado, **abra uma issue ou envie um PR**. O objetivo é tornar dados públicos brasileiros acessíveis via IA, juntos.

Todos os dados vêm de APIs oficiais do governo brasileiro — o server não gera, modifica ou editorializa nenhum dado.

## Licença

MIT

## Star History

<a href="https://www.star-history.com/?repos=jxnxts%2Fmcp-brasil&type=date&legend=top-left">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=jxnxts/mcp-brasil&type=date&theme=dark&legend=top-left" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=jxnxts/mcp-brasil&type=date&legend=top-left" />
   <img alt="Star History Chart" src="https://api.star-history.com/chart?repos=jxnxts/mcp-brasil&type=date&legend=top-left" />
 </picture>
</a>
