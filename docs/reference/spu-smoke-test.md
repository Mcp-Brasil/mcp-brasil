# Smoke test das novas tools SPU

**Executado em:** 2026-04-23T19:34:50+00:00

**Escopo:** 8 tools novas (`spu_geo` + `spu_imoveis`) + 1 prompt (`compras/pncp.alienacoes_imoveis_spu`) contra as APIs públicas reais. Todas as chamadas abaixo foram feitas via `fastmcp.Client`.

## 1. `spu_geo` — GeoPortal SPUnet (WMS)

Fonte: `https://geoportal-spunet.gestao.gov.br/geoserver/ows` · Protocolo: WMS 1.1.1 GetFeatureInfo · Zero-auth

### ✅ `listar_camadas_spu()` _(⏱ 0.01s)_

```
**GeoPortal SPU — 13 camadas disponíveis**

| ID | Título | Geometria | Descrição |
| --- | --- | --- | --- |
| terreno_marinha | Trechos de Terreno de Marinha | MultiPolygon | Faixa de 33m a partir da linha preamar média de 1831 (DL 9.760/1946, a… |
| acrescido_marinha | Terrenos Acrescidos de Marinha | MultiPolygon | Terrenos formados natural ou artificialmente além dos terrenos de mari… |
| terreno_marginal | Trechos de Terreno Marginal | MultiPolygon | Terrenos marginais de rios federais (faixa de 15m). |
| acrescido_marginal | Terrenos Acrescidos Marginais | MultiPolygon | Acrescidos em terrenos marginais. |
| ilha_federal | Ilhas Federais | MultiPolygon | Ilhas de propriedade da União. |
| praia_federal | Praias Federais | MultiPolygon | Praias (bem de uso comum do povo sob administração da União). |
| manguezal_federal | Manguezais Federais | MultiPolygon | Áreas de manguezal em terras da União. |
| espelho_dagua_federal | Espelhos d'Água Federais | MultiPolygon | Corpos d'água federais (rios, lagos, reservatórios). |
| imovel_localizacao | Localização de Imóveis da União | Point | Pontos geocodificados de imóveis cadastrados no SPUnet, com RIP, tipo,… |
| linha_preamar_media | Linha de Preamar Média (LPM) | MultiLineString | Linha de Preamar Média de 1831 (referência para terrenos de marinha). |
| linha_terreno_marinha | Linha do Terreno de Marinha (LTM) | MultiLineString | Limite dos 33m da faixa de terreno de marinha. |
| limite_lltm | Limite da Linha do Terreno de Marinha (LLTM) | MultiLineString | Limite superior da demarcação de terrenos de marinha. |
| lmeo | Linha Média das Enchentes Ordinárias (LMEO) | MultiLineString | Referência para terrenos marginais em rios federais. |
```

### ✅ `detalhar_camada_spu(camada_id='terreno_marinha')` _(⏱ 0.00s)_

**Camada: Trechos de Terreno de Marinha** (`terreno_marinha`)

- **GeoServer typename**: `spunet:vw_app_trecho_terreno_marinha_a`
- **Tipo de geometria**: MultiPolygon
- **Descrição**: Faixa de 33m a partir da linha preamar média de 1831 (DL 9.760/1946, art. 2º).

Acessível via WMS GetFeatureInfo em `https://geoportal-spunet.gestao.gov.br/geoserver/ows`.

### ✅ `consultar_ponto_spu(lat=-22.896, lon=-43.2013)` — Rio de Janeiro, Zona Portuária _(⏱ 1.29s)_

**Ponto (-22.896, -43.2013) está em terras da União**

Camadas atingidas: terreno_marinha

| Camada | Nome/Endereço | UF | Área | Situação |
| --- | --- | --- | --- | --- |
| terreno_marinha | Rua Santo Cristo | RJ | 13.700 m² | Aprovado |

### ✅ `consultar_ponto_spu(lat=-15.7939, lon=-47.8828)` — Brasília (Planalto) _(⏱ 1.06s)_

O ponto (-15.7939, -47.8828) **não está em nenhuma** das camadas públicas de terras da União verificadas (terreno_marinha, acrescido_marinha, terreno_marginal, acrescido_marginal, ilha_federal, praia_federal, manguezal_federal).

Isso não exclui a possibilidade de o imóvel estar em regime de aforamento/ocupação sem demarcação geoespacial oficial.

### ✅ `consultar_ponto_spu(lat=-13.0064, lon=-38.5336)` — Salvador, Porto da Barra _(⏱ 1.06s)_

O ponto (-13.0064, -38.5336) **não está em nenhuma** das camadas públicas de terras da União verificadas (terreno_marinha, acrescido_marinha, terreno_marginal, acrescido_marginal, ilha_federal, praia_federal, manguezal_federal).

Isso não exclui a possibilidade de o imóvel estar em regime de aforamento/ocupação sem demarcação geoespacial oficial.

### ✅ `buscar_imoveis_area_spu(bbox='-43.22,-22.92,-43.18,-22.88', limite=10)` — Rio _(⏱ 0.21s)_

**Imóveis da União em bbox=-43.22,-22.92,-43.18,-22.88** — 10 encontrado(s)

| RIP | Tipo | UF | Município | Endereço |
| --- | --- | --- | --- | --- |
| 00005569 | Gleba/Terreno/Lote sem edificação | RJ | Rio de Janeiro | Praça SANTO CRISTO |
| 00010285 | Gleba/Terreno/Lote com edificação | RJ | Rio de Janeiro | Rua DA GAMBOA |
| 00010439 | Gleba/Terreno/Lote sem edificação | RJ | Rio de Janeiro | Avenida CIDADE DE LIMA |
| 00011343 | Gleba/Terreno/Lote com edificação | RJ | Rio de Janeiro | Rua DA AMERICA |
| 00011345 | Gleba/Terreno/Lote com edificação | RJ | Rio de Janeiro | Rua DA AMERICA |
| 00011346 | Gleba/Terreno/Lote com edificação | RJ | Rio de Janeiro | Rua DA AMERICA |
| 00011347 | Gleba/Terreno/Lote com edificação | RJ | Rio de Janeiro | Rua DA AMERICA |
| 00011348 | Gleba/Terreno/Lote com edificação | RJ | Rio de Janeiro | Rua DA AMERICA |
| 00011349 | Gleba/Terreno/Lote com edificação | RJ | Rio de Janeiro | Rua DA AMERICA |
| 00011351 | Gleba/Terreno/Lote com edificação | RJ | Rio de Janeiro | Rua DA AMERICA |

### ✅ `buscar_imoveis_area_spu(bbox='-34.92,-8.12,-34.88,-8.05', limite=5)` — Recife (Boa Viagem) _(⏱ 0.17s)_

**Imóveis da União em bbox=-34.92,-8.12,-34.88,-8.05** — 1 encontrado(s)

| RIP | Tipo | UF | Município | Endereço |
| --- | --- | --- | --- | --- |
| 00022057 | Gleba/Terreno/Lote sem edificação | PE | Recife | Rua GENERAL ESTILAC LEAL |


## 2. `spu_imoveis` — Raio-X APF / Patrimônio da União

Fonte: `repositorio.dados.gov.br/seges/raio-x/patrimonio-uniao.csv` · Zero-auth · Dataset consolidado SPUnet/SIAPA pelo MGI

### ✅ `info_dataset_spu()` _(⏱ 7.07s)_

**Raio-X APF — Patrimônio da União**

- Total de imóveis: **54.405**
- Meses de referência: 202405
- CSV: https://repositorio.dados.gov.br/seges/raio-x/patrimonio-uniao.csv
- Datapackage: https://repositorio.dados.gov.br/seges/raio-x/datapackage.json
- Cache atualizado em: 2026-04-23T19:34:54.767316+00:00


### ✅ `resumo_orgaos_spu(top=10)` _(⏱ 0.03s)_

**Patrimônio da União por Órgão Superior — Top 10**

| Sigla | Órgão | Imóveis | Área (km²) | Valor total |
| --- | --- | --- | --- | --- |
| MGI | MINISTÉRIO DA GESTÃO E DA INOVAÇÃO EM SERVIÇO | 18.268 | 181.936,87 km² | R$ 1.709.715.118.457,89 |
| MD | MINISTÉRIO DA DEFESA | 11.709 | 52.716,78 km² | R$ 653.808.284.365,18 |
| MDA | MINISTÉRIO DO DESENVOLVIMENTO AGRÁRIO E AGRIC | 9.671 | 850.550,09 km² | R$ 784.085.875.849,72 |
| MEC | MINISTÉRIO DA EDUCAÇÃO | 6.423 | 8.950,68 km² | R$ 1.001.268.383.434,77 |
| MS | MINISTÉRIO DA SAÚDE | 1.526 | 17,70 km² | R$ 11.398.870.976,80 |
| MT | MINISTÉRIO DOS TRANSPORTES | 1.258 | 33,44 km² | R$ 49.672.750.427,52 |
| MMA | MINISTÉRIO DO MEIO AMBIENTE E MUDANÇA DO CLIM | 1.107 | 67.488,71 km² | R$ 31.271.879.785,12 |
| MIDR | MINISTÉRIO DA INTEGRAÇÃO E DO DESENVOLVIMENTO | 847 | 22.921,88 km² | R$ 3.235.114.271,48 |
| MPI | MINISTÉRIO DOS POVOS INDÍGENAS | 762 | 1.065.285,98 km² | R$ 260.237.102.133,09 |
| MJSP | MINISTÉRIO DA JUSTIÇA E SEGURANÇA PÚBLICA | 668 | 120,35 km² | R$ 7.242.808.360,58 |

### ✅ `resumo_ufs_spu()` _(⏱ 0.03s)_

**Patrimônio da União por UF**

| UF | Imóveis | Área (km²) | Valor total | Aluguel anual |
| --- | --- | --- | --- | --- |
| DF | 10.379 | 4.332,89 km² | R$ 423.235.624.131,99 | R$ 22.965.054,46 |
| SP | 6.942 | 2.932,31 km² | R$ 252.644.253.174,61 | R$ 3.724.790,31 |
| RJ | 4.178 | 3.136,93 km² | R$ 1.999.507.090.901,06 | R$ 6.905.561,33 |
| MG | 3.559 | 12.703,84 km² | R$ 91.907.805.096,20 | R$ 4.816.493,61 |
| RS | 2.658 | 5.780,26 km² | R$ 60.175.704.533,24 | R$ 1.522.391,30 |
| PA | 2.217 | 560.776,35 km² | R$ 267.879.847.010,64 | R$ 4.602.524,12 |
| PE | 2.202 | 10.322,31 km² | R$ 19.753.939.522,73 | R$ 1.412.833,33 |
| PR | 2.110 | 6.450,40 km² | R$ 63.574.489.485,63 | R$ 1.992.514,75 |
| CE | 1.904 | 14.102,78 km² | R$ 122.918.239.876,66 | R$ 1.190.955,00 |
| BA | 1.764 | 21.758,66 km² | R$ 50.167.578.879,42 | R$ 1.116.631,76 |
| MA | 1.755 | 92.888,17 km² | R$ 55.364.422.518,08 | R$ 1.067.358,48 |
| SC | 1.751 | 2.854,10 km² | R$ 84.855.529.885,82 | R$ 932.670,84 |
| MT | 1.501 | 195.733,80 km² | R$ 248.712.520.280,16 | R$ 543.698,31 |
| AM | 1.484 | 730.728,59 km² | R$ 423.641.452.603,24 | R$ 3.224.994,18 |
| MS | 1.439 | 14.570,42 km² | R$ 42.937.616.484,16 | R$ 558.603,30 |
| GO | 1.179 | 12.763,08 km² | R$ 38.484.146.253,43 | R$ 552.420,51 |
| TO | 1.048 | 48.910,19 km² | R$ 24.985.344.917,65 | R$ 1.325.519,22 |
| RN | 968 | 5.671,80 km² | R$ 22.278.024.671,80 | R$ 388.626,30 |
| RO | 948 | 187.197,76 km² | R$ 175.328.428.924,44 | R$ 935.618,86 |
| PI | 772 | 22.339,99 km² | R$ 5.148.283.175,98 | R$ 2.579.839,59 |
| PB | 762 | 3.181,16 km² | R$ 8.479.392.635,74 | R$ 291.782,03 |
| SE | 717 | 1.445,12 km² | R$ 5.022.102.869,02 | R$ 1.802.727,88 |
| AL | 537 | 1.216,98 km² | R$ 5.936.953.175,65 | R$ 167.115,46 |
| AC | 445 | 53.290,92 km² | R$ 41.335.887.211,60 | R$ 158.562,89 |
| ES | 419 | 1.249,64 km² | R$ 37.994.039.951,51 | R$ 281.640,97 |
| AP | 324 | 55.594,54 km² | R$ 28.279.780.949,55 | R$ 234.343,55 |
| RR | 289 | 183.329,80 km² | R$ 93.760.323.373,58 | R$ 701.115,30 |
| EX | 154 | 0,50 km² | R$ 4.531.137.262,06 | R$ 153.276.247.117,99 |

### ✅ `buscar_imoveis_spu(orgao_sigla='MEC', uf='DF', limite=5)` _(⏱ 0.01s)_

**Imóveis da União — 5 resultado(s)**

| Órgão | UF | Município | Regime | Tipo | Endereço | Valor |
| --- | --- | --- | --- | --- | --- | --- |
| MEC | DF | BRASÍLIA | ENTREGA - ADMINISTRAÇÃO FEDERAL DIR | Edifício / Prédio | ED SGO QUADRA 1 s/n VIA N3 - Garagem do  | R$ 6.176.834,26 |
| MEC | DF | BRASÍLIA | ENTREGA - ADMINISTRAÇÃO FEDERAL DIR | Edifício / Prédio | ESP DOS MINISTERIOS 05 Ministério da Edu | R$ 51.150.999,90 |
| MEC | DF | BRASÍLIA | ENTREGA - ADMINISTRAÇÃO FEDERAL DIR | Edifício / Prédio | ST SGAS 607 607 Quadra 607, módulo 50 | R$ 10.502.081,02 |
| MEC | DF | BRASÍLIA | ENTREGA - ADMINISTRAÇÃO FEDERAL DIR | Edifício / Prédio | ST SGAS 604 28 Quadra 604 | R$ 2.839.764,19 |
| MEC | DF | BRASÍLIA | USO EM SERVIÇO PÚBLICO | Edifício / Prédio | ESP DOS MINISTERIOS 01 Bloco L - Ministé | R$ 40.701.075,54 |

### ✅ `buscar_imoveis_spu(uf='DF', municipio='brasília', regime='funcional', limite=5)` _(⏱ 0.01s)_

**Imóveis da União — 5 resultado(s)**

| Órgão | UF | Município | Regime | Tipo | Endereço | Valor |
| --- | --- | --- | --- | --- | --- | --- |
| PR | DF | BRASÍLIA | IMÓVEL FUNCIONAL | Apartamento | Q SQN 105 BLOCO G 502 Apartamento | R$ 263.586,56 |
| PR | DF | BRASÍLIA | IMÓVEL FUNCIONAL | Apartamento | Q SQN 106 BLOCO D 306 apartamento | R$ 1.323.936,74 |
| PR | DF | BRASÍLIA | IMÓVEL FUNCIONAL | Apartamento | Q SQN 106 BLOCO K 106 Bloco  K | R$ 465.152,89 |
| PR | DF | BRASÍLIA | IMÓVEL FUNCIONAL | Apartamento | Q SQN 112 BLOCO C 305 apartamento | R$ 1.261.168,72 |
| PR | DF | BRASÍLIA | IMÓVEL FUNCIONAL | Apartamento | Q SQN 304 BLOCO A 106 apartamento | R$ 1.544.526,30 |

### ✅ `buscar_imoveis_spu(orgao_sigla='MS', uf='SP', limite=5)` _(⏱ 0.01s)_

**Imóveis da União — 5 resultado(s)**

| Órgão | UF | Município | Regime | Tipo | Endereço | Valor |
| --- | --- | --- | --- | --- | --- | --- |
| MS | SP | MIRACATU | LOCAÇÃO DE TERCEIROS | Base Militar | R Waldir de Azevedo 741 Sala | R$ 2.641,35 |
| MS | SP | PERUÍBE | LOCAÇÃO DE TERCEIROS | Base Militar | R MINISTRO GENÉSIO DE ALMEIDA MOURA 64 C | R$ 166.737,60 |
| MS | SP | BAURU | LOCAÇÃO DE TERCEIROS | Base Militar | R QUINTINO BOCAIUVA QUADRA 1417 Casa | R$ 304.445,68 |
| MS | SP | UBATUBA | LOCAÇÃO DE TERCEIROS | Base Militar | R Rio Grande do Sul 101 Casa | R$ 426.828,00 |
| MS | SP | REGISTRO | LOCAÇÃO DE TERCEIROS | Base Militar | R Seiji Sumida 106 Casa | R$ 25.344,59 |

### ✅ `buscar_imoveis_spu(orgao_sigla='MDA', tipo_destinacao='fazenda', limite=5)` _(⏱ 0.01s)_

**Imóveis da União — 5 resultado(s)**

| Órgão | UF | Município | Regime | Tipo | Endereço | Valor |
| --- | --- | --- | --- | --- | --- | --- |
| MDA | GO | FAZENDA NOVA | EM REGULARIZAÇÃO - ENTREGA | Fazenda | FAZ PALMEIRAS FAZENDA NOVA-GO s/n PROJET | R$ 94.021.278,27 |
| MDA | PE | SERRA TALHADA | EM REGULARIZAÇÃO - OUTROS | Fazenda | AV FAZENDA BOA VISTA S/N  | R$ 304.646,75 |
| MDA | PE | BELÉM DO SÃO FRANCISCO | EM REGULARIZAÇÃO - OUTROS | Fazenda | AV Fazenda Santana 806 GLEBA XIQUE XIQUE | R$ 378.449,36 |
| MDA | PE | CABROBÓ | EM REGULARIZAÇÃO - OUTROS | Fazenda | AV Fazenda Terra Nova (Bananeiras) S/N l | R$ 419.853,12 |
| MDA | PE | SERRA TALHADA | EM REGULARIZAÇÃO - OUTROS | Fazenda | COND BR-238 sentido Salgueiro percorre-s | R$ 1.096.819,04 |


## 3. `compras/pncp` — prompt `alienacoes_imoveis_spu`

Template de análise que o LLM recebe como instrução. Sem chamada HTTP.

### ✅ `alienacoes_imoveis_spu('20260101', '20260331', uf='DF')`

```
Levante as alienações de imóveis da União entre 20260101 e 20260331 na UF DF via PNCP:

1. **Leilão eletrônico** (modalidade 1):
   `buscar_contratacoes(data_inicial='20260101', data_final='20260331', modalidade=1, uf='DF')`

2. **Leilão presencial** (modalidade 13):
   `buscar_contratacoes(data_inicial='20260101', data_final='20260331', modalidade=13, uf='DF')`

3. Filtre localmente os resultados por termos como 'imóvel da União', 'terreno de marinha', 'SPU' ou pelo CNPJ da SPU (00.394.460/0024-32 — Secretaria do Patrimônio da União / MGI).

4. **Amparo legal Lei 9.636/1998** (confirmação em contratos):
   A API de consulta do PNCP não expõe o código de amparo como filtro, mas nos detalhes individuais de cada contrato (tool `consultar_contratacao_detalhe`) o campo `amparoLegal` identifica os códigos **55 (art. 11-C, I), 56 (art. 11-C, II), 57, 58 e 59 (art. 24-C)** — todos dispositivos da Lei 9.636/1998, que regulamenta a gestão do patrimônio imobiliário da União.

5. Para cada leilão relevante, informe: objeto, órgão, valor estimado/homologado, situação, UF/município e link do PNCP.

6. Se houver imóveis geocodificados no resultado, considere cruzar com `spu_geo_consultar_ponto_spu` (se lat/lon disponíveis) e `spu_imoveis_buscar_imoveis_spu` (por UF/município) para enriquecer o dossiê patrimonial.
```
