# Catalogo de Features

41 features · 326 tools · 80 resources · 62 prompts

---

## Economico

### `ibge` — IBGE (9 tools)

Instituto Brasileiro de Geografia e Estatistica — estados, municipios, nomes, agregados estatisticos, CNAE, malhas geograficas.

| Tool | Descricao |
|------|-----------|
| `ibge_listar_estados` | Lista os 27 estados com codigo, nome, sigla e regiao |
| `ibge_buscar_municipios` | Municipios por codigo do estado |
| `ibge_listar_regioes` | Lista as 5 regioes do Brasil |
| `ibge_consultar_nome` | Frequencia de nomes pelo censo do IBGE |
| `ibge_ranking_nomes` | Nomes mais comuns por estado/municipio |
| `ibge_consultar_agregado` | Agregados estatisticos (populacao, PIB, area, series de pesquisa) |
| `ibge_listar_pesquisas` | Lista programas de pesquisa do IBGE |
| `ibge_obter_malha` | Malhas geograficas (GeoJSON) de estados/municipios |
| `ibge_buscar_cnae` | Busca codigos CNAE de atividades economicas |

**Chave:** Nenhuma

---

### `bacen` — Banco Central do Brasil (9 tools)

Selic, IPCA, cambio, PIB, +190 series do SGS, Boletim Focus.

| Tool | Descricao |
|------|-----------|
| `bacen_consultar_serie` | Consulta qualquer serie BCB/SGS por codigo |
| `bacen_ultimos_valores` | Ultimos N valores de uma serie |
| `bacen_metadados_serie` | Metadados (nome, unidade, periodicidade) |
| `bacen_series_populares` | Lista curada de series populares (Selic, IPCA, dolar, etc.) |
| `bacen_buscar_serie` | Busca series por palavra-chave |
| `bacen_indicadores_atuais` | Busca paralela: Selic, IPCA, IPCA-12m, dolar, IBC-Br |
| `bacen_calcular_variacao` | Variacao percentual entre duas datas |
| `bacen_comparar_series` | Compara 2-5 series lado a lado |
| `bacen_expectativas_focus` | Expectativas do mercado (Boletim Focus) |

**Chave:** Nenhuma

---

### `bndes` — BNDES (4 tools)

Dados abertos do BNDES: operacoes de financiamento, exportacao, desembolsos, instituicoes credenciadas e datasets via CKAN API.

| Tool | Descricao |
|------|-----------|
| `bndes_buscar_datasets_bndes` | Busca datasets no portal de dados abertos do BNDES |
| `bndes_detalhar_dataset_bndes` | Detalha um dataset do BNDES com seus recursos |
| `bndes_consultar_operacoes_bndes` | Consulta operacoes de financiamento do BNDES |
| `bndes_listar_instituicoes_bndes` | Lista instituicoes financeiras credenciadas pelo BNDES |

**Chave:** Nenhuma

---

## Legislativo

### `camara` — Camara dos Deputados (11 tools)

Deputados, proposicoes, votacoes, despesas, comissoes, frentes parlamentares.

| Tool | Descricao |
|------|-----------|
| `camara_listar_deputados` | Lista deputados com filtros (nome, partido, UF) |
| `camara_buscar_deputado` | Detalhes de um deputado por ID |
| `camara_buscar_proposicao` | Projetos de lei por tipo, ano, tema, autor ou deputado autor |
| `camara_detalhar_proposicao` | Detalhes completos de uma proposicao por ID |
| `camara_consultar_tramitacao` | Historico de tramitacao de uma proposicao |
| `camara_buscar_votacao` | Sessoes de votacao |
| `camara_votos_nominais` | Votos nominais (como cada deputado votou) |
| `camara_despesas_deputado` | Relatorio de despesas de deputados |
| `camara_agenda_legislativa` | Calendario legislativo |
| `camara_buscar_comissoes` | Comissoes da Camara |
| `camara_frentes_parlamentares` | Frentes parlamentares |

**Chave:** Nenhuma

---

### `senado` — Senado Federal (26 tools)

Senadores, materias, votacoes, comissoes, agenda, liderancas.

**Senadores (4):** `senado_listar_senadores`, `senado_buscar_senador`, `senado_buscar_senador_por_nome`, `senado_votacoes_senador`

**Materias (5):** `senado_buscar_materia`, `senado_detalhe_materia`, `senado_consultar_tramitacao_materia`, `senado_textos_materia`, `senado_votos_materia`

**Votacoes (3):** `senado_listar_votacoes`, `senado_detalhe_votacao`, `senado_votacoes_recentes`

**Comissoes (4):** `senado_listar_comissoes`, `senado_detalhe_comissao`, `senado_membros_comissao`, `senado_reunioes_comissao`

**Agenda (2):** `senado_agenda_plenario`, `senado_agenda_comissoes`

**Auxiliar (6):** `senado_legislatura_atual`, `senado_partidos_senado`, `senado_ufs_senado`, `senado_tipos_materia`, `senado_emendas_materia`, `senado_listar_blocos`

**Extra (2):** `senado_listar_liderancas`, `senado_relatorias_senador`

**Chave:** Nenhuma

---

## Transparencia / Fiscal

### `transparencia` — Portal da Transparencia (18 tools)

Contratos federais, despesas, servidores, sancoes, bolsa familia, emendas, viagens, cartoes de pagamento.

| Tool | Descricao |
|------|-----------|
| `transparencia_buscar_contratos` | Contratos federais |
| `transparencia_consultar_despesas` | Despesas por funcao/UF/ano |
| `transparencia_buscar_servidores` | Servidores publicos federais |
| `transparencia_buscar_licitacoes` | Processos licitatorios |
| `transparencia_consultar_bolsa_familia` | Beneficiarios do Bolsa Familia |
| `transparencia_buscar_sancoes` | Sancoes (CEIS/CNEP/CEPIM/CEAF) |
| `transparencia_buscar_emendas` | Emendas parlamentares |
| `transparencia_consultar_viagens` | Viagens a servico do governo |
| `transparencia_buscar_convenios` | Convenios |
| `transparencia_buscar_cartoes_pagamento` | Transacoes de cartoes de pagamento |
| `transparencia_buscar_pep` | Pessoas politicamente expostas |
| `transparencia_buscar_acordos_leniencia` | Acordos de leniencia |
| `transparencia_buscar_notas_fiscais` | Notas fiscais |
| `transparencia_consultar_beneficio_social` | Beneficios sociais |
| `transparencia_consultar_cpf` | Consulta CPF |
| `transparencia_consultar_cnpj` | Consulta CNPJ |
| `transparencia_detalhar_contrato` | Detalhes de um contrato |
| `transparencia_detalhar_servidor` | Detalhes de um servidor |

**Chave:** Opcional — [cadastro gratuito](http://portaldatransparencia.gov.br/api-de-dados/cadastrar-email)

---

### `tcu` — Tribunal de Contas da Uniao (9 tools)

Acordaos, licitantes inabilitados/inidoneos, certidoes APF, debitos, contratos, pautas de sessao.

| Tool | Descricao |
|------|-----------|
| `tcu_consultar_acordaos` | Acordaos do TCU com filtros de busca |
| `tcu_consultar_inabilitados` | Pessoas inabilitadas |
| `tcu_consultar_inidoneos` | Empresas inidoneas |
| `tcu_consultar_certidoes` | Certidoes APF (TCU + CNJ + CGU) |
| `tcu_calcular_debito` | Correcao de debito via SELIC |
| `tcu_consultar_pedidos_congresso` | Pedidos do Congresso ao TCU |
| `tcu_consultar_termos_contratuais` | Contratos do TCU |
| `tcu_consultar_pautas_sessao` | Pautas de sessoes de julgamento |
| `tcu_consultar_cadirreg` | Registro de irregularidades (CADIRREG) |

**Chave:** Nenhuma

---

### `transferegov` — TransfereGov (5 tools)

Emendas parlamentares PIX (transferencias especiais).

| Tool | Descricao |
|------|-----------|
| `transferegov_buscar_emendas_pix` | Emendas PIX por ano/estado |
| `transferegov_buscar_emenda_por_autor` | Emendas por nome do parlamentar |
| `transferegov_detalhe_emenda` | Detalhes por ID do plano de acao |
| `transferegov_emendas_por_municipio` | Emendas para um municipio |
| `transferegov_resumo_emendas_ano` | Resumo anual de emendas PIX |

**Chave:** Nenhuma

---

### Tribunais de Contas Estaduais (10 features)

| Feature | UF | Tools | Cobertura |
|---------|-----|-------|-----------|
| `tce_sp` | Sao Paulo | 3 | Despesas, receitas, municipios |
| `tce_rj` | Rio de Janeiro | 7 | Licitacoes, contratos, obras, penalidades |
| `tce_rs` | Rio Grande do Sul | 5 | Educacao, saude, gestao fiscal (LRF) |
| `tce_sc` | Santa Catarina | 2 | Municipios, unidades gestoras |
| `tce_pe` | Pernambuco | 5 | Licitacoes, contratos, despesas, fornecedores |
| `tce_ce` | Ceara | 4 | Municipios, licitacoes, contratos, empenhos |
| `tce_rn` | Rio Grande do Norte | 5 | Jurisdicionados, licitacoes, contratos |
| `tce_pi` | Piaui | 5 | Prefeituras, despesas, receitas |
| `tce_to` | Tocantins | 3 | Processos, pautas de sessoes |
| `tce_es` | Espirito Santo | 4 | Licitacoes, contratos, contratacoes, obras |

**Chave:** Nenhuma (todas)

---

## Judiciario

### `datajud` — DataJud / CNJ (7 tools)

Processos judiciais de todos os tribunais brasileiros, movimentacoes, busca avancada.

| Tool | Descricao |
|------|-----------|
| `datajud_buscar_processos` | Processos com filtros diversos |
| `datajud_buscar_processo_por_numero` | Processo por numero (formato CNJ) |
| `datajud_buscar_processos_por_classe` | Por codigo de classe processual |
| `datajud_buscar_processos_por_assunto` | Por codigo de assunto |
| `datajud_buscar_processos_por_orgao` | Por tribunal/orgao julgador |
| `datajud_buscar_processos_avancado` | Busca avancada com paginacao `search_after` |
| `datajud_consultar_movimentacoes` | Movimentacoes de um processo |

**Chave:** Opcional — [cadastro gratuito](https://datajud-wiki.cnj.jus.br/api-publica/acesso)

---

### `jurisprudencia` — STF, STJ, TST (6 tools)

Decisoes dos tribunais superiores, sumulas, repercussao geral, informativos.

| Tool | Descricao |
|------|-----------|
| `jurisprudencia_buscar_jurisprudencia_stf` | Decisoes do STF |
| `jurisprudencia_buscar_jurisprudencia_stj` | Decisoes do STJ |
| `jurisprudencia_buscar_jurisprudencia_tst` | Decisoes trabalhistas do TST |
| `jurisprudencia_buscar_sumulas` | Sumulas de todos os tribunais |
| `jurisprudencia_buscar_repercussao_geral` | Temas de repercussao geral (STF) |
| `jurisprudencia_buscar_informativos` | Boletins informativos dos tribunais |

**Chave:** Nenhuma

---

## Eleitoral

### `tse` — Tribunal Superior Eleitoral (15 tools)

Eleicoes, candidatos, resultados, prestacao de contas, apuracao.

| Tool | Descricao |
|------|-----------|
| `tse_anos_eleitorais` | Anos com eleicoes disponiveis |
| `tse_listar_eleicoes` | Eleicoes por ano |
| `tse_listar_eleicoes_suplementares` | Eleicoes suplementares |
| `tse_listar_estados_suplementares` | Estados com eleicoes suplementares |
| `tse_listar_cargos` | Cargos em disputa |
| `tse_listar_candidatos` | Candidatos com filtros |
| `tse_buscar_candidato` | Detalhes de um candidato |
| `tse_resultado_eleicao` | Resultado de uma eleicao |
| `tse_consultar_prestacao_contas` | Prestacao de contas de candidatos |
| `tse_resultado_nacional` | Resultado nacional via CDN |
| `tse_resultado_por_estado` | Resultado por estado |
| `tse_mapa_resultado_estados` | Mapa de resultados (27 estados em paralelo) |
| `tse_listar_municipios_eleitorais` | Municipios eleitorais |
| `tse_resultado_por_municipio` | Resultado por municipio |
| `tse_apuracao_status` | Status da apuracao |

**Chave:** Nenhuma

---

### `anuncios_eleitorais` — Biblioteca de Anuncios da Meta (6 tools)

Busca e analise de anuncios eleitorais e sobre temas sociais, eleicoes ou politica veiculados no Brasil via Meta Ad Library API.

| Tool | Descricao |
|------|-----------|
| `anuncios_eleitorais_buscar_anuncios_eleitorais` | Busca anuncios eleitorais e politicos por termos |
| `anuncios_eleitorais_buscar_anuncios_por_pagina` | Anuncios de paginas especificas do Facebook |
| `anuncios_eleitorais_buscar_anuncios_por_financiador` | Anuncios pelo nome do financiador |
| `anuncios_eleitorais_buscar_anuncios_por_regiao` | Anuncios com alcance em uma regiao/estado |
| `anuncios_eleitorais_analisar_demografia_anuncios` | Distribuicao demografica e regional dos anuncios |
| `anuncios_eleitorais_buscar_anuncios_frase_exata` | Busca por frase exata |

**Chave:** Obrigatoria — `META_ACCESS_TOKEN` ([cadastro Meta for Developers](https://developers.facebook.com/))

---

## Ambiental

### `inpe` — INPE (4 tools)

Focos de queimadas, desmatamento (PRODES e DETER), dados de satelite.

| Tool | Descricao |
|------|-----------|
| `inpe_buscar_focos_queimadas` | Focos ativos de incendio |
| `inpe_consultar_desmatamento` | Dados de desmatamento (PRODES) |
| `inpe_alertas_deter` | Alertas de desmatamento (DETER) |
| `inpe_dados_satelite` | Dados por sensor de satelite |

**Chave:** Nenhuma

---

### `ana` — Agencia Nacional de Aguas (3 tools)

Estacoes hidrologicas, telemetria (chuva, vazao, nivel), reservatorios.

| Tool | Descricao |
|------|-----------|
| `ana_buscar_estacoes` | Estacoes de monitoramento hidrologico |
| `ana_consultar_telemetria` | Leituras de telemetria |
| `ana_monitorar_reservatorios` | Monitoramento de reservatorios (SAR/ANA) |

**Chave:** Nenhuma

---

### `tabua_mares` — Tabua de Mares (7 tools)

Previsao de mares para portos do litoral brasileiro.

| Tool | Descricao |
|------|-----------|
| `tabua_mares_listar_estados_costeiros` | 17 estados costeiros com portos disponiveis |
| `tabua_mares_listar_portos` | Portos disponiveis em um estado costeiro |
| `tabua_mares_buscar_portos` | Informacoes detalhadas de portos pelo ID |
| `tabua_mares_consultar_tabua_mare` | Tabua de mares de um porto para dias especificos |
| `tabua_mares_porto_mais_proximo` | Porto mais proximo de uma coordenada em um estado |
| `tabua_mares_porto_mais_proximo_geral` | Porto mais proximo independente do estado |
| `tabua_mares_tabua_mare_por_geolocalizacao` | Tabua de mares do porto mais proximo via coordenadas |

**Chave:** Nenhuma

---

## Saude

### `saude` — CNES / DataSUS (10 tools)

Estabelecimentos de saude, profissionais, leitos, urgencias, tipos, coordenadas, resumo municipal e comparacao.

| Tool | Descricao |
|------|-----------|
| `saude_buscar_estabelecimentos` | Estabelecimentos de saude (CNES) |
| `saude_buscar_profissionais` | Profissionais de saude |
| `saude_listar_tipos_estabelecimento` | Tipos de estabelecimento |
| `saude_consultar_leitos` | Disponibilidade de leitos |
| `saude_buscar_urgencias` | Unidades de urgencia e emergencia |
| `saude_buscar_por_tipo` | Estabelecimentos por tipo especifico |
| `saude_buscar_estabelecimento_por_cnes` | Detalhes de um estabelecimento pelo CNES |
| `saude_buscar_por_coordenadas` | Estabelecimentos proximos a uma coordenada |
| `saude_resumo_rede_municipal` | Resumo da rede de saude de um municipio |
| `saude_comparar_municipios` | Comparacao de redes de saude entre municipios |

**Chave:** Nenhuma

---

### `anvisa` — ANVISA (10 tools)

Bulario eletronico, bulas, categorias regulatorias, genericos, principios ativos, registros e empresas.

| Tool | Descricao |
|------|-----------|
| `anvisa_buscar_medicamento` | Busca medicamentos no Bulario Eletronico |
| `anvisa_buscar_por_principio_ativo` | Busca por principio ativo |
| `anvisa_consultar_bula` | Bulas de um medicamento por numero de processo |
| `anvisa_listar_categorias` | Categorias regulatorias (generico, similar, referencia) |
| `anvisa_informacoes_bula` | Secoes padrao de uma bula no Brasil |
| `anvisa_buscar_por_categoria` | Medicamentos por categoria regulatoria |
| `anvisa_buscar_genericos` | Medicamentos genericos |
| `anvisa_verificar_registro` | Registro de medicamento na ANVISA |
| `anvisa_buscar_por_empresa` | Medicamentos de uma empresa especifica |
| `anvisa_resumo_regulatorio` | Resumo regulatorio de um medicamento |

**Chave:** Nenhuma

---

### `farmacia_popular` — Farmacia Popular (8 tools)

Medicamentos gratuitos, farmacias credenciadas, indicacoes terapeuticas, elegibilidade.

| Tool | Descricao |
|------|-----------|
| `farmacia_popular_buscar_farmacias` | Farmacias credenciadas por municipio/UF |
| `farmacia_popular_listar_medicamentos` | Medicamentos disponiveis no programa |
| `farmacia_popular_verificar_medicamento` | Verifica se um medicamento esta no programa |
| `farmacia_popular_buscar_por_indicacao` | Medicamentos por indicacao terapeutica |
| `farmacia_popular_estatisticas_programa` | Estatisticas consolidadas do programa |
| `farmacia_popular_verificar_elegibilidade` | Requisitos para retirar medicamentos |
| `farmacia_popular_municipios_atendidos` | Municipios com farmacias credenciadas |
| `farmacia_popular_farmacia_mais_proxima` | Farmacia mais proxima por coordenadas |

**Chave:** Nenhuma

---

### `rename` — RENAME (5 tools)

Relacao Nacional de Medicamentos Essenciais do SUS — medicamentos por nome, principio ativo ou grupo terapeutico.

| Tool | Descricao |
|------|-----------|
| `rename_buscar_medicamento_rename` | Busca medicamentos no catalogo RENAME |
| `rename_listar_por_grupo_terapeutico` | Medicamentos por grupo terapeutico |
| `rename_verificar_disponibilidade_sus` | Verifica disponibilidade no SUS |
| `rename_listar_grupos_terapeuticos` | Grupos terapeuticos da RENAME |
| `rename_estatisticas_rename` | Estatisticas consolidadas da RENAME |

**Chave:** Nenhuma

---

### `bps` — Banco de Precos em Saude (3 tools)

Precos de medicamentos e dispositivos medicos comprados pelo governo em todas as esferas.

| Tool | Descricao |
|------|-----------|
| `bps_consultar_precos_saude` | Compras registradas no BPS |
| `bps_buscar_medicamento_bps` | Precos de medicamentos por descricao |
| `bps_buscar_catmat_bps` | Precos por codigo CATMAT |

**Chave:** Nenhuma

---

### `opendatasus` — OpenDataSUS (7 tools)

Portal de dados abertos do SUS via CKAN API — hospitais, leitos, vacinacao, SRAG, qualidade da agua.

| Tool | Descricao |
|------|-----------|
| `opendatasus_buscar_datasets` | Busca datasets no OpenDataSUS |
| `opendatasus_detalhar_dataset` | Detalhes de um dataset |
| `opendatasus_consultar_datastore` | Registros de um recurso DataStore |
| `opendatasus_listar_datasets_conhecidos` | Datasets mais importantes do OpenDataSUS |
| `opendatasus_buscar_com_filtro` | Busca com filtros avancados |
| `opendatasus_consultar_vacinacao` | Dados de vacinacao |
| `opendatasus_consultar_srag` | Dados de SRAG (sindrome respiratoria aguda grave) |

**Chave:** Nenhuma

---

### `imunizacao` — PNI / Programa Nacional de Imunizacoes (10 tools)

Registros de vacinacao, calendario nacional, vacinas do SUS, metas de cobertura vacinal.

| Tool | Descricao |
|------|-----------|
| `imunizacao_buscar_vacinacao` | Busca registros de vacinacao |
| `imunizacao_estatisticas_por_vacina` | Estatisticas por tipo de vacina |
| `imunizacao_estatisticas_por_faixa_etaria` | Estatisticas por faixa etaria |
| `imunizacao_buscar_datasets_pni` | Datasets do PNI no OpenDataSUS |
| `imunizacao_consultar_doses_dataset` | Dados de um dataset PNI especifico |
| `imunizacao_calendario_vacinacao` | Calendario Nacional de Vacinacao completo |
| `imunizacao_listar_vacinas_sus` | Vacinas disponiveis no SUS |
| `imunizacao_consultar_vacina` | Detalhes de uma vacina especifica |
| `imunizacao_verificar_esquema_vacinal` | Vacinas por faixa etaria |
| `imunizacao_metas_cobertura` | Metas de cobertura vacinal |

**Chave:** Nenhuma

---

### `denasus` — DENASUS (5 tools)

Departamento Nacional de Auditoria do SUS — atividades de auditoria, relatorios anuais e planos de auditoria interna.

| Tool | Descricao |
|------|-----------|
| `denasus_buscar_auditorias` | Atividades de auditoria do DENASUS |
| `denasus_listar_relatorios_anuais` | Relatorios anuais de atividades |
| `denasus_listar_planos` | Planos anuais de auditoria interna |
| `denasus_verificar_municipio` | Verifica se municipio foi alvo de auditoria |
| `denasus_informacoes_sna` | Sistema Nacional de Auditoria do SUS (SNA) |

**Chave:** Nenhuma

---

## Compras Publicas

### `compras/pncp` — PNCP (14 tools)

Portal Nacional de Contratacoes Publicas (Lei 14.133/2021).

| Tool | Descricao |
|------|-----------|
| `compras_pncp_buscar_contratacoes` | Contratacoes por texto, CNPJ, data |
| `compras_pncp_buscar_contratos` | Contratos por texto, CNPJ do fornecedor |
| `compras_pncp_buscar_atas` | Atas de registro de preco |
| `compras_pncp_consultar_fornecedor` | Dados do fornecedor por CNPJ |
| `compras_pncp_consultar_orgao` | Orgaos contratantes |
| `compras_pncp_buscar_contratacoes_abertas` | Licitacoes com prazo de proposta aberto |
| `compras_pncp_buscar_contratacoes_atualizadas` | Contratacoes atualizadas por periodo |
| `compras_pncp_buscar_contratos_atualizados` | Contratos atualizados por periodo |
| `compras_pncp_buscar_atas_atualizadas` | Atas atualizadas por periodo |
| `compras_pncp_consultar_contratacao_detalhe` | Detalhes de uma contratacao especifica |
| `compras_pncp_buscar_pca` | Plano de contratacoes anual |
| `compras_pncp_buscar_pca_atualizacao` | PCAs atualizados por periodo |
| `compras_pncp_buscar_pca_usuario` | Itens de PCA por usuario responsavel |
| `compras_pncp_buscar_instrumentos_cobranca` | Instrumentos de cobranca (notas fiscais) |

**Chave:** Nenhuma

---

### `compras/dadosabertos` — ComprasNet / SIASG (8 tools)

Dados legados de compras publicas (ate ~2020).

| Tool | Descricao |
|------|-----------|
| `compras_dadosabertos_buscar_licitacoes` | Processos licitatorios por data |
| `compras_dadosabertos_buscar_pregoes` | Pregoes eletronicos |
| `compras_dadosabertos_buscar_dispensas` | Dispensas/inexigibilidades |
| `compras_dadosabertos_buscar_contratos` | Contratos por vigencia |
| `compras_dadosabertos_consultar_fornecedor` | Fornecedores por CNPJ/CPF |
| `compras_dadosabertos_buscar_material_catmat` | Catalogo CATMAT (materiais) |
| `compras_dadosabertos_buscar_servico_catser` | Catalogo CATSER (servicos) |
| `compras_dadosabertos_buscar_uasg` | Codigos UASG de orgaos |

**Chave:** Nenhuma

---

### `compras/contratosgovbr` — Contratos.gov.br (7 tools)

Contratos federais pos-2021 via API Contratos.gov.br — empenhos, faturas, aditivos, itens e terceirizados.

| Tool | Descricao |
|------|-----------|
| `compras_contratosgovbr_listar_contratos_unidade` | Contratos ativos de uma Unidade Gestora |
| `compras_contratosgovbr_consultar_contrato_id` | Contrato especifico por ID |
| `compras_contratosgovbr_consultar_empenhos_contrato` | Empenhos (compromissos orcamentarios) |
| `compras_contratosgovbr_consultar_faturas_contrato` | Faturas (notas fiscais) |
| `compras_contratosgovbr_consultar_historico_contrato` | Termos aditivos e apostilamentos |
| `compras_contratosgovbr_consultar_itens_contrato` | Itens (materiais e servicos) |
| `compras_contratosgovbr_consultar_terceirizados_contrato` | Trabalhadores terceirizados |

**Chave:** Nenhuma

---

## Seguranca Publica

### `atlas_violencia` — Atlas da Violencia / IPEA (7 tools)

Series historicas de homicidios, violencia por genero/raca, suicidios, armas de fogo, mortes no transito e intervencao policial.

| Tool | Descricao |
|------|-----------|
| `atlas_violencia_listar_temas_violencia` | Temas disponiveis no Atlas da Violencia |
| `atlas_violencia_listar_series_tema` | Series de dados disponiveis para um tema |
| `atlas_violencia_consultar_valores_violencia` | Valores de uma serie temporal |
| `atlas_violencia_consultar_valores_por_regiao` | Valores filtrados por regioes especificas |
| `atlas_violencia_consultar_serie_violencia` | Metadados de uma serie especifica |
| `atlas_violencia_listar_fontes_violencia` | Fontes de dados do Atlas |
| `atlas_violencia_listar_metadados_violencia` | Unidades de medida e periodicidades |

**Chave:** Nenhuma

---

### `sinesp` — SINESP / MJSP (6 tools)

Datasets de seguranca publica (homicidios, estupros, roubos, furtos, trafico, sistema penitenciario) via CKAN do Ministerio da Justica.

| Tool | Descricao |
|------|-----------|
| `sinesp_listar_datasets_mjsp` | Todos os datasets do portal MJSP |
| `sinesp_buscar_datasets_mjsp` | Busca datasets por palavra-chave |
| `sinesp_detalhar_dataset_mjsp` | Detalhes de um dataset |
| `sinesp_listar_organizacoes_mjsp` | Organizacoes do portal MJSP |
| `sinesp_listar_datasets_organizacao` | Datasets publicados por uma organizacao |
| `sinesp_listar_datasets_grupo_seguranca` | Datasets do grupo de seguranca publica |

**Chave:** Nenhuma

---

### `forum_seguranca` — Forum Brasileiro de Seguranca Publica (4 tools)

Publicacoes sobre seguranca publica, violencia, sistema prisional, Atlas da Violencia e Anuario via DSpace API.

| Tool | Descricao |
|------|-----------|
| `forum_seguranca_buscar_publicacoes_seguranca` | Publicacoes no repositorio FBSP |
| `forum_seguranca_listar_temas_seguranca` | Comunidades tematicas |
| `forum_seguranca_detalhar_publicacao_seguranca` | Detalhes de uma publicacao por UUID |
| `forum_seguranca_buscar_por_tema_seguranca` | Publicacoes dentro de uma comunidade tematica |

**Chave:** Nenhuma

---

## Utilidades

### `brasilapi` — BrasilAPI (16 tools)

CEP, CNPJ, DDD, bancos, cambio, FIPE, feriados, PIX, ISBN, NCM.

| Tool | Descricao |
|------|-----------|
| `brasilapi_consultar_cep` | Consulta CEP |
| `brasilapi_consultar_cnpj` | Consulta CNPJ |
| `brasilapi_consultar_ddd` | Consulta DDD |
| `brasilapi_listar_bancos` | Lista bancos |
| `brasilapi_consultar_banco` | Detalhes de um banco |
| `brasilapi_listar_moedas` | Lista moedas disponiveis |
| `brasilapi_consultar_cotacao` | Cotacao de moeda |
| `brasilapi_consultar_feriados` | Feriados nacionais por ano |
| `brasilapi_consultar_taxa` | Taxas (SELIC/CDI/IPCA/TR) |
| `brasilapi_listar_tabelas_fipe` | Tabelas FIPE |
| `brasilapi_listar_marcas_fipe` | Marcas na FIPE |
| `brasilapi_buscar_veiculos_fipe` | Veiculos na FIPE |
| `brasilapi_consultar_isbn` | Consulta ISBN de livros |
| `brasilapi_buscar_ncm` | Nomenclatura Comum do Mercosul |
| `brasilapi_consultar_pix_participantes` | Participantes do PIX |
| `brasilapi_consultar_registro_br` | Dominios .br |

**Chave:** Nenhuma

---

### `dados_abertos` — Portal Dados Abertos (4 tools)

Catalogo de datasets de dados.gov.br.

| Tool | Descricao |
|------|-----------|
| `dados_abertos_buscar_conjuntos` | Busca datasets |
| `dados_abertos_detalhar_conjunto` | Detalhes de um dataset |
| `dados_abertos_listar_organizacoes` | Organizacoes publicadoras |
| `dados_abertos_buscar_recursos` | Recursos/arquivos dentro de um dataset |

**Chave:** Nenhuma

---

### `diario_oficial` — Querido Diario + DOU (11 tools)

Diarios oficiais de 5.000+ municipios (Querido Diario) e Diario Oficial da Uniao (DOU federal).

**Querido Diario — municipios (6):**

| Tool | Descricao |
|------|-----------|
| `diario_oficial_buscar_diarios` | Busca full-text em diarios oficiais municipais |
| `diario_oficial_buscar_diarios_regiao` | Busca em diarios de uma regiao |
| `diario_oficial_buscar_cidades` | Busca municipios por nome (codigo IBGE) |
| `diario_oficial_listar_territorios` | Territorios com diarios disponiveis |
| `diario_oficial_listar_diarios_recentes` | Publicacoes recentes de um municipio |
| `diario_oficial_buscar_diario_unificado` | Busca unificada (municipios + DOU federal) |

**DOU — Diario Oficial da Uniao (5):**

| Tool | Descricao |
|------|-----------|
| `diario_oficial_dou_buscar` | Busca publicacoes no DOU por termo, secao, periodo |
| `diario_oficial_dou_ler_publicacao` | Le conteudo completo de uma publicacao do DOU |
| `diario_oficial_dou_edicao_do_dia` | Publicacoes de uma edicao do DOU por data e secao |
| `diario_oficial_dou_buscar_por_orgao` | Publicacoes de um orgao especifico no DOU |
| `diario_oficial_dou_buscar_avancado` | Busca avancada com todos os filtros combinados |

**Chave:** Nenhuma

---

## Agentes IA

### `redator` — Redator Oficial (5 tools + 5 prompts + 10 resources)

Agente inteligente para redacao oficial brasileira — oficio, despacho, memorando, portaria, parecer, nota tecnica.

**Tools:**

| Tool | Descricao |
|------|-----------|
| `redator_formatar_data_extenso` | Formata data por extenso em portugues |
| `redator_gerar_numeracao` | Gera numeracao oficial de documentos |
| `redator_consultar_pronome_tratamento` | Pronome de tratamento correto por cargo |
| `redator_validar_documento` | Valida CPF/CNPJ |
| `redator_listar_tipos_documento` | Lista tipos de documento suportados |

**Prompts:** `redator_oficio`, `redator_despacho`, `redator_portaria`, `redator_parecer`, `redator_nota_tecnica`

**Resources:** 7 templates de documentos + 3 documentos normativos (manual de redacao, pronomes, fechos)

**Chave:** Nenhuma
