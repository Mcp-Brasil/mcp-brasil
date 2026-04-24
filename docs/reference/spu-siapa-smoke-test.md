# Rodada de consultas — `spu_siapa` (ADR-004)

**Executado em:** 2026-04-24 (local)
**Dataset:** `spu_siapa` · 812.868 linhas · 45.8 MB (DuckDB) · fonte: SPU/MGI Nextcloud
**Ambiente:** `MCP_BRASIL_DATASETS=spu_siapa`, cache local já aquecido

20 perguntas reais executadas via `fastmcp.Client`, contra o DuckDB local. Tempo
**médio: 13ms/query · total: 0,27s**. Nenhuma chamada externa após o primeiro load.

---

## P1 — Panorama nacional por UF

**Tool:** `resumo_uf_siapa()` · 41ms

Resumido: RJ lidera com **136.138** imóveis (131.369 dominiais, 4.769 uso
especial); PE e SP fecham o pódio com 131.895 e 122.271. DF tem inversão
interessante — de 13.192 imóveis apenas 1.512 são dominiais e 11.680 uso
especial (sede da APF). AM e PA têm poucos imóveis (7.551 e 35.465) mas
**áreas gigantescas** (742 bi e 565 bi de m² respectivamente — grandes
glebas da União em terra firme amazônica).

## P2 — Regimes de utilização (Brasil)

**Tool:** `resumo_regime_siapa()` · 13ms

Top 5 regimes: (saída condensada)

| Regime | Imóveis |
|---|---|
| Inscrição de Ocupação | ~388.000 |
| Aforamento | ~337.000 |
| Sem Destinação Definida | ~30.000 |
| Entrega - Adm. Federal Direta | ~12.000 |
| Cessão de uso gratuita | ~10.000 |

Ocupação + Aforamento dominam (clássicos terrenos de marinha com foreiros
e ocupantes privados).

## P3 — Conceituação no RJ

**Tool:** `resumo_conceituacao_siapa(uf="RJ")` · 15ms

RJ é quase todo marinha: dos 136k imóveis, a grande maioria está em
"Marinha", "Acrescido de Marinha" ou "Marinha com Acrescido".

## P4 — Top 10 municípios de PE

**Tool:** `top_municipios_siapa(uf="PE", top=10)` · 12ms

Concentração costeira esperada: Recife, Olinda, Cabo de Santo Agostinho,
Ipojuca, Jaboatão — todos municípios litorâneos com forte presença de
terrenos de marinha.

## P5 — Aforamento no Rio de Janeiro (15 imóveis)

**Tool:** `buscar_imoveis_siapa(uf="RJ", municipio="Rio de Janeiro", regime="Aforamento")` · 11ms

15 RIPs retornados — todos dominiais em marinha, com RIPs
`600101…` (município 6001 = RJ capital no padrão SIAPA).

## P6 — Ocupações em Salvador-BA

**Tool:** `buscar_imoveis_siapa(uf="BA", municipio="Salvador", regime="Ocupação")` · 9ms

Retornou imóveis em regime de "Inscrição de Ocupação" (variante do
regime Ocupação) no centro histórico e bairros litorâneos.

## P7 — Terrenos de marinha em Santos

**Tool:** `buscar_imoveis_siapa(uf="SP", municipio="Santos", conceituacao="Marinha")` · 8ms

10 imóveis, todos em "Acrescido de Marinha" com áreas pequenas
(~240m²). Padrão de loteamento residencial pós-1940 em terrenos
aterrados na baixada santista.

## P8 — Uso especial em Brasília

**Tool:** `buscar_imoveis_siapa(uf="DF", municipio="Brasília", classe="Uso Especial")` · 14ms

**Retornou 0**. Achado: no SIAPA o município aparece como "Brasilia"
(sem acento). Com `municipio="Brasilia"` a query funciona. Bom exemplo
de por que o filtro usa ILIKE (substring) — o LLM pode tentar novamente
sem acento.

## P9 — Regimes em Pernambuco

**Tool:** `resumo_regime_siapa(uf="PE")` · 12ms

17 regimes distintos. **65.984** inscrições de ocupação + **62.426**
aforamentos formam 97% do total. Há 2 CUEM (Concessão de Uso Especial
para fins de Moradia) — REURB em ação.

## P10 — Terrenos de marinha no Brasil

**Tool:** `resumo_conceituacao_siapa()` · 12ms

| Conceituação | Imóveis |
|---|---|
| Marinha | **285.869** |
| Acrescido de Marinha | **221.781** |
| Nacional Interior | 149.157 |
| Marinha com Acrescido | 104.190 |
| Marginal de Rio | 30.386 |
| Terra Indígena | 722 (1.079 bilhões de m²!) |

Total de terrenos de marinha (todas as variantes): **~611.000 imóveis**
em 18 bilhões de m². Mais de 75% do patrimônio dominial da União.

## P11 — Vizinhança de um RIP conhecido

**Tool:** `buscar_imoveis_siapa(rip="6001013067")` · 22ms

Retornou 10 RIPs contíguos na Zona Portuária do Rio (RIP 6001…),
misturando aforamento e ocupação. Mostra que a numeração sequencial
do SIAPA agrupa imóveis por quadra/logradouro.

## P12 — Cessões no DF

**Tool:** `buscar_imoveis_siapa(uf="DF", regime="Cessão")` · 16ms

Retornou concessões de direito real de uso (ex: 1.265.327 m² — uma
gleba) e cessões de uso gratuita em Brasília. Mostra o padrão do DF:
grandes áreas institucionais cedidas a órgãos e fundações.

## P13 — Top 10 municípios de SP

**Tool:** `top_municipios_siapa(uf="SP", top=10)` · 13ms

Surpresa: **Barueri lidera** com 38.529 imóveis (Alphaville + adjacências,
loteamentos dominiais). Santos (29k), São Vicente (18k), Santana de
Parnaíba (10k) completam. Zona litorânea vs zona de expansão metropolitana.

## P14 — Imóveis em Recife

**Tool:** `buscar_imoveis_siapa(uf="PE", municipio="Recife")` · 9ms

Mix de inscrição de ocupação e aforamento em marinha / marinha com
acrescido — padrão do centro histórico do Recife.

## P15 — Aforamento em marinha em SP

**Tool:** `buscar_imoveis_siapa(uf="SP", regime="Aforamento", conceituacao="Marinha")` · 8ms

15 RIPs em Santos com áreas pequenas (~230-260m²) — loteamentos
residenciais dominiais beira-mar da baixada santista.

## P16 — Estado do cache

**Tool:** `info_spu_siapa()` · 2ms

- Cached: sim
- Linhas: **812.868**
- Tamanho DuckDB: 45.8 MB (vs 220MB CSV original — 79% de redução)
- Idade: 0.00 dias (fresh)
- TTL: 30 dias

## P17 — Top municípios do AM

**Tool:** `top_municipios_siapa(uf="AM", top=10)` · 11ms

| Município | Imóveis |
|---|---|
| Careiro da Várzea | 1.362 |
| Boca do Acre | 950 |
| Manicoré | 763 |
| Manaus | 759 |
| Iranduba | 565 |

Padrão ribeirinho — imóveis em terreno marginal de rios federais.

## P18 — Conceituação no Pará

**Tool:** `resumo_conceituacao_siapa(uf="PA")` · 11ms

PA é inverso do RJ: **14.693 marginais de rio** lidera, marinha só 8.714.
Terra Indígena tem apenas 67 imóveis mas com **283 bilhões de m²** (médio
de 4 bilhões m² por registro — grandes demarcações).

## P19 — Uso especial em MG

**Tool:** `buscar_imoveis_siapa(uf="MG", classe="Uso Especial")` · 12ms

10 imóveis em 10 municípios diferentes — distribuição dispersa de
postos e instalações federais pelo interior mineiro (Diamantina, Caxambu,
Coimbra, Leopoldina).

## P20 — Ilhas federais

**Tool:** `buscar_imoveis_siapa(conceituacao="Ilha")` · 20ms

**Retornou 0**. No SIAPA não existe conceituação literal "Ilha" —
ilhas aparecem sob "Nacional Interior" ou na camada geoespacial
`spu_geo.ilha_federal`. Achado: docs da tool precisam mencionar que
`conceituacao` usa os valores do SIAPA, e para ilhas usar `spu_geo`.

---

## Resumo de latências

| Estatística | Valor |
|---|---|
| Queries executadas | **20** |
| Total | **0,27s** |
| Média | **13ms** |
| Mínima | 2ms (`info_spu_siapa`) |
| Máxima | 41ms (`resumo_uf_siapa` — full-scan + GROUP BY) |

Comparação: o dataset Raio-X APF (17MB, 54k linhas, CSV em memória Python)
levava ~7s para carregar na primeira tool e rodava cada filtro com linear
scan em Python puro. DuckDB carregou 15× mais dados e responde cada query
em 1/1000 do tempo, com SQL completo (GROUP BY, window, joins).

---

## Achados para iteração futura

1. **P8/P20**: filtros literais falharam onde a base tem grafia diferente
   (`Brasilia` sem acento; `Ilha` não é um valor válido de conceituação).
   A docstring da tool deveria listar valores canônicos — ou adicionar uma
   tool `valores_distintos_siapa(coluna)` que faz `SELECT DISTINCT`.
2. **P10/P18**: "Terra Indígena" tem áreas gigantescas (médio 4 bi m²) —
   vale criar tool dedicada para jurisdições especiais (TI, UC, Quilombola).
3. **P13**: Barueri lidera SP com 38k imóveis — provável concentração
   cartográfica específica que merece investigação (loteamentos Alphaville
   historicamente dominiais?).
4. **P16**: Dataset tem 812k linhas no banco vs 787k documentados no ADR —
   atualizar ADR-004 para o número real.
