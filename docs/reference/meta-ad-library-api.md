# API da Biblioteca de Anuncios da Meta

Documentacao de referencia da API usada pela feature `anuncios_eleitorais`.

- **Fonte:** [facebook.com/ads/library/api](https://www.facebook.com/ads/library/api/)
- **Base URL:** `https://graph.facebook.com/v25.0/ads_archive`
- **Autenticacao:** Token de acesso (`META_ACCESS_TOKEN`)
- **Rate limit:** Nao documentado oficialmente

---

## Visao geral

A API da Biblioteca de Anuncios permite pesquisar:

- **Anuncios politicos/eleitorais** — sobre temas sociais, eleicoes ou politica veiculados em qualquer lugar do mundo nos ultimos 7 anos
- **Todos os anuncios** — veiculados no Reino Unido e na Uniao Europeia no ultimo ano

No contexto do mcp-brasil, usamos apenas anuncios politicos/eleitorais com foco no Brasil (`ad_reached_countries=['BR']`).

## Como obter acesso

1. Confirme identidade em [facebook.com/ID](https://www.facebook.com/ID)
2. Crie conta em [Meta for Developers](https://developers.facebook.com/)
3. Crie um app e gere um token de acesso no [Graph API Explorer](https://developers.facebook.com/tools/explorer/)
4. Configure: `META_ACCESS_TOKEN=seu-token`

## Exemplo de consulta

```bash
curl -G \
  -d "search_terms='california'" \
  -d "ad_type=POLITICAL_AND_ISSUE_ADS" \
  -d "ad_reached_countries=['US']" \
  -d "access_token=<ACCESS_TOKEN>" \
  "https://graph.facebook.com/v25.0/ads_archive"
```

Resposta:

```json
{
  "data": [
    {
      "page_id": "123",
      "page_name": "Example Page",
      "ad_snapshot_url": "https://www.facebook.com/ads/archive/render_ad/?id=123"
    }
  ],
  "paging": {
    "cursors": { "before": "MAZDZD", "after": "MAZDZD" },
    "next": "https://graph.facebook.com/v25.0/ads_archive?...&after=MAZDZD"
  }
}
```

---

## Parametros de pesquisa

| Parametro | Tipo | Descricao |
|-----------|------|-----------|
| `ad_reached_countries` | `array<string>` | **Obrigatorio.** Codigos ISO de paises (ex: `['BR']`). |
| `search_terms` | `string` | Termos de busca (max 100 chars). Espaco = AND logico. |
| `search_type` | `enum` | `KEYWORD_UNORDERED` (padrao) ou `KEYWORD_EXACT_PHRASE`. |
| `ad_type` | `enum` | `ALL`, `POLITICAL_AND_ISSUE_ADS`, `EMPLOYMENT_ADS`, `HOUSING_ADS`, `FINANCIAL_PRODUCTS_AND_SERVICES_ADS`. Padrao: `ALL`. |
| `ad_active_status` | `enum` | `ACTIVE` (padrao), `INACTIVE`, `ALL`. |
| `ad_delivery_date_min` | `string` | Data minima de veiculacao (`YYYY-MM-DD`). |
| `ad_delivery_date_max` | `string` | Data maxima de veiculacao (`YYYY-MM-DD`). |
| `bylines` | `array<string>` | Financiador ("pago por"). Apenas para `POLITICAL_AND_ISSUE_ADS`. |
| `delivery_by_region` | `array<string>` | Regiao de alcance (ex: `['Sao Paulo']`). Apenas para `POLITICAL_AND_ISSUE_ADS`. |
| `estimated_audience_size_min` | `int64` | Audiencia minima estimada. Valores: 100, 1000, 5000, 10000, 50000, 100000, 500000, 1000000. |
| `estimated_audience_size_max` | `int64` | Audiencia maxima estimada. Mesmos valores acima. |
| `languages` | `array<string>` | Idiomas ISO 639-1 (ex: `['pt']`). |
| `media_type` | `enum` | `ALL`, `IMAGE`, `MEME`, `VIDEO`, `NONE`. |
| `publisher_platforms` | `array<enum>` | `FACEBOOK`, `INSTAGRAM`, `MESSENGER`, `WHATSAPP`, `AUDIENCE_NETWORK`, `THREADS`. |
| `search_page_ids` | `array<int64>` | IDs de paginas do Facebook (max 10). |
| `unmask_removed_content` | `boolean` | Mostrar conteudo removido por violacao. Padrao: `false`. |

---

## Campos de resposta

### Campos padrao (todos os anuncios)

| Campo | Tipo | Descricao |
|-------|------|-----------|
| `id` | `string` | ID da biblioteca do anuncio. |
| `ad_creation_time` | `string` | Data/hora UTC de criacao. |
| `ad_delivery_start_time` | `string` | Data/hora UTC de inicio da veiculacao. |
| `ad_delivery_stop_time` | `string` | Data/hora UTC de fim da veiculacao. |
| `ad_snapshot_url` | `string` | URL do snapshot do anuncio. |
| `page_id` | `string` | ID da pagina do Facebook. |
| `page_name` | `string` | Nome da pagina. |
| `publisher_platforms` | `list<enum>` | Plataformas onde apareceu. |
| `languages` | `list<string>` | Idiomas do anuncio (ISO 639-1). |

### Campos de criativo

| Campo | Tipo | Descricao |
|-------|------|-----------|
| `ad_creative_bodies` | `list<string>` | Textos do anuncio (um por card/versao). |
| `ad_creative_link_captions` | `list<string>` | Legendas do link CTA. |
| `ad_creative_link_descriptions` | `list<string>` | Descricoes do link CTA. |
| `ad_creative_link_titles` | `list<string>` | Titulos do link CTA. |

### Campos politicos/eleitorais (`POLITICAL_AND_ISSUE_ADS`)

| Campo | Tipo | Descricao |
|-------|------|-----------|
| `bylines` | `string` | Financiador do anuncio ("pago por"). |
| `currency` | `string` | Moeda (ISO, ex: `BRL`). |
| `spend` | `InsightsRangeValue` | Gasto total (faixas: <100, 100-499, 500-999, 1K-5K, etc.). |
| `impressions` | `InsightsRangeValue` | Impressoes (faixas: <1000, 1K-5K, 5K-10K, etc.). |
| `demographic_distribution` | `list<AudienceDistribution>` | Distribuicao por idade/genero do alcance. |
| `delivery_by_region` | `list<AudienceDistribution>` | Distribuicao regional do alcance. |
| `estimated_audience_size` | `InsightsRangeValue` | Tamanho estimado da audiencia. |

### Campos especificos do Brasil

| Campo | Tipo | Descricao |
|-------|------|-----------|
| `br_total_reach` | `int32` | Alcance estimado no Brasil. |
| `age_country_gender_reach_breakdown` | `list` | Demograficos detalhados (idade/genero/pais). |
| `target_ages` | `list<string>` | Faixas etarias de segmentacao. |
| `target_gender` | `enum` | Genero de segmentacao: `Women`, `Men`, `All`. |
| `target_locations` | `list<TargetLocation>` | Localizacoes de segmentacao. |
| `total_reach_by_location` | `list<KeyValue>` | Alcance por localizacao (BR, EU, UK). |

---

## Tools do mcp-brasil

A feature `anuncios_eleitorais` expoe 6 tools:

| Tool | Descricao |
|------|-----------|
| `buscar_anuncios_eleitorais` | Busca anuncios politicos por palavras-chave com filtros |
| `buscar_anuncios_por_pagina` | Busca anuncios de paginas especificas por ID |
| `buscar_anuncios_por_financiador` | Filtra por financiador (campo byline) |
| `buscar_anuncios_por_regiao` | Busca anuncios com alcance em estados brasileiros |
| `analisar_demografia_anuncios` | Analisa distribuicao demografica e regional |
| `buscar_anuncios_frase_exata` | Busca por frase exata (match exato) |

### Configuracao

```bash
META_ACCESS_TOKEN=seu-token-aqui
```

---

## Links uteis

- [Biblioteca de Anuncios](https://www.facebook.com/ads/library/?ad_type=political_and_issue_ads) — Interface web de busca
- [Relatorio da Biblioteca](https://www.facebook.com/ads/library/report/) — Relatorios agregados
- [Graph API Explorer](https://developers.facebook.com/tools/explorer/) — Testar consultas
- [Repositorio de scripts](https://github.com/facebookresearch/Ad-Library-API-Script-Repository) — Exemplos oficiais da Meta
- [Notas de versao](https://www.facebook.com/ads/library/api/releasenotes) — Changelog da API
- [Status do sistema](https://metastatus.com/ads-transparency) — Disponibilidade da API
