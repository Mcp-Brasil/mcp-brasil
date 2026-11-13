package transparencia

// Estado representa uma Unidade Federativa brasileira (26 estados + DF).
type Estado struct {
	CodigoUF             int     `json:"codigo_uf"`
	UF                   string  `json:"uf"`
	Nome                 string  `json:"nome"`
	Latitude             float64 `json:"latitude"`
	Longitude            float64 `json:"longitude"`
	Regiao               string  `json:"regiao"`
	Capital              string  `json:"capital"`
	TotalMunicipios      int     `json:"total_municipios"`
	PortalTransparencia  string  `json:"portal_transparencia"`
	PortalGoverno        string  `json:"portal_governo"`
	TribunalContasNome   string  `json:"tribunal_contas_nome"`
	TribunalContasURL    string  `json:"tribunal_contas_url"`
	AssembleiaNome       string  `json:"assembleia_nome"`
	AssembleiaURL        string  `json:"assembleia_url"`
	SiconfiEnteID        string  `json:"siconfi_ente_id"`
	RadarTransparenciaURL string `json:"radar_transparencia_url"`
	PNCPURL              string  `json:"pncp_url"`
}

// Municipio representa um município brasileiro (5.570 municípios + DF + Fernando de Noronha).
type Municipio struct {
	CodigoIBGE  string  `json:"codigo_ibge"`
	Nome        string  `json:"nome"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Capital     bool    `json:"capital"`
	CodigoUF    int     `json:"codigo_uf"`
	UF          string  `json:"uf"`
	Regiao      string  `json:"regiao"`
	SiafiID     string  `json:"siafi_id"`
	DDD         string  `json:"ddd"`
	FusoHorario string  `json:"fuso_horario"`
	Slug        string  `json:"slug"`
	CNPJ        string  `json:"cnpj"`
	Populacao   int     `json:"populacao"`
}

// MunicipioDossier consolida a identidade de transparência de um município.
type MunicipioDossier struct {
	Municipio                     *Municipio `json:"municipio"`
	Estado                        *Estado    `json:"estado"`
	CNPJ                          string     `json:"cnpj"`
	Populacao                     int        `json:"populacao"`
	PortalTransparenciaPrefeitura string     `json:"portal_transparencia_prefeitura"`
	PortalPrefeitura              string     `json:"portal_prefeitura"`
	PortalCamara                  string     `json:"portal_camara"`
	TribunalContasCompetente      string     `json:"tribunal_contas_competente"`
	TribunalContasURL             string     `json:"tribunal_contas_url"`
	RadarTransparenciaURL         string     `json:"radar_transparencia_url"`
	SiconfiURL                    string     `json:"siconfi_url"`
	SiconfiAPI                    string     `json:"siconfi_api"`
	PNCPURL                       string     `json:"pncp_url"`
	PNCPCNPJURL                   string     `json:"pncp_cnpj_url,omitempty"`
	TransferegovURL               string     `json:"transferegov_url"`
	TransparenciaFederalURL       string     `json:"transparencia_federal_url"`
	SiconfiDCAEndpoint            string     `json:"siconfi_dca_endpoint,omitempty"`
	SiconfiRREOEndpoint           string     `json:"siconfi_rreo_endpoint,omitempty"`
	SiconfiEntregasEndpoint       string     `json:"siconfi_entregas_endpoint,omitempty"`
}

// EstadoDossier consolida a identidade de transparência de uma UF.
type EstadoDossier struct {
	Estado               *Estado      `json:"estado"`
	TotalMunicipios      int          `json:"total_municipios"`
	MunicipiosPrincipais []*Municipio `json:"municipios_principais,omitempty"`
	PortalTransparencia  string       `json:"portal_transparencia"`
	PortalGoverno        string       `json:"portal_governo"`
	TribunalContasNome   string       `json:"tribunal_contas_nome"`
	TribunalContasURL    string       `json:"tribunal_contas_url"`
	AssembleiaNome       string       `json:"assembleia_nome"`
	AssembleiaURL        string       `json:"assembleia_url"`
	RadarTransparenciaURL string      `json:"radar_transparencia_url"`
	SiconfiURL           string       `json:"siconfi_url"`
	SiconfiAPI           string       `json:"siconfi_api"`
	PNCPURL              string       `json:"pncp_url"`
}

// FiscalItem representa uma linha de dado contábil/fiscal do SICONFI (DCA ou RREO).
type FiscalItem struct {
	Exercicio   int     `json:"exercicio"`
	Instituicao string  `json:"instituicao"`
	CodIBGE     int     `json:"cod_ibge"`
	UF          string  `json:"uf"`
	Anexo       string  `json:"anexo"`
	Conta       string  `json:"conta"`
	CodConta    string  `json:"cod_conta"`
	Coluna      string  `json:"coluna"`
	Valor       float64 `json:"valor"`
	Populacao   int     `json:"populacao"`
}

// EntregaItem representa o status de entrega de demonstrativos da Lei de Responsabilidade Fiscal.
type EntregaItem struct {
	Exercicio       int    `json:"exercicio"`
	CodIBGE         int    `json:"cod_ibge"`
	Instituicao     string `json:"instituicao"`
	Entregavel      string `json:"entregavel"`
	Periodo         int    `json:"periodo"`
	Periodicidade   string `json:"periodicidade"`
	StatusRelatorio string `json:"status_relatorio,omitempty"`
	DataStatus      string `json:"data_status"`
	FormaEnvio      string `json:"forma_envio"`
}

// SearchResult representa o resultado de uma busca nativa.
type SearchResult struct {
	Query           string       `json:"query"`
	UF              string       `json:"uf,omitempty"`
	TotalEncontrado int          `json:"total_encontrado"`
	Municipios      []*Municipio `json:"municipios"`
	Estados         []*Estado    `json:"estados,omitempty"`
}

// Stats reúne métricas dos municípios e estados brasileiros.
type Stats struct {
	TotalMunicipios     int            `json:"total_municipios"`
	TotalUFs            int            `json:"total_ufs"`
	PopulacaoTotal      int64          `json:"populacao_total"`
	MunicipiosPorRegiao map[string]int `json:"municipios_por_regiao"`
	MunicipiosPorUF     map[string]int `json:"municipios_por_uf"`
	Capitais            []string       `json:"capitais"`
}

// ComparativoItem é um item no relatório de comparação de transparência.
type ComparativoItem struct {
	Tipo                 string  `json:"tipo"` // "municipio" ou "estado"
	Nome                 string  `json:"nome"`
	UF                   string  `json:"uf"`
	CodigoIBGE           string  `json:"codigo_ibge"`
	SiafiID              string  `json:"siafi_id,omitempty"`
	CNPJ                 string  `json:"cnpj,omitempty"`
	Populacao            int     `json:"populacao,omitempty"`
	Regiao               string  `json:"regiao"`
	Capital              bool    `json:"capital"`
	PortalTransparencia  string  `json:"portal_transparencia"`
	TribunalContasNome   string  `json:"tribunal_contas_nome"`
	TribunalContasURL    string  `json:"tribunal_contas_url"`
	RadarTransparenciaURL string `json:"radar_transparencia_url"`
	SiconfiURL           string  `json:"siconfi_url"`
	PNCPURL              string  `json:"pncp_url"`
}

// ComparativoRelatorio agrega múltiplos itens comparados.
type ComparativoRelatorio struct {
	TotalItens int                `json:"total_itens"`
	Itens      []*ComparativoItem `json:"itens"`
}

// ProbeResult representa o resultado do teste de conectividade de um portal.
type ProbeResult struct {
	Alvo       string `json:"alvo"`
	URL        string `json:"url"`
	StatusHTTP int    `json:"status_http,omitempty"`
	TempoMs    int64  `json:"tempo_ms"`
	Online     bool   `json:"online"`
	Erro       string `json:"erro,omitempty"`
}

// PortalDiscoveryResult representa o resultado da descoberta inteligente do portal municipal.
type PortalDiscoveryResult struct {
	Municipio           *Municipio `json:"municipio"`
	URLResolvida        string     `json:"url_resolvida,omitempty"`
	SitePrefeitura      string     `json:"site_prefeitura,omitempty"`
	PortalCamara        string     `json:"portal_camara,omitempty"`
	Plataforma          string     `json:"plataforma"`
	Versao              string     `json:"versao,omitempty"`
	CodAdministracao    int        `json:"cod_administracao,omitempty"`
	TenantID            string     `json:"tenant_id,omitempty"`
	EntidadeNome        string     `json:"entidade_nome,omitempty"`
	FonteDescoberta     string     `json:"fonte_descoberta,omitempty"`
	ServidoresSuportado bool       `json:"servidores_suportado"`
	CandidatosTestados  []string   `json:"candidatos_testados,omitempty"`
	TempoMs             int64      `json:"tempo_ms"`
	Observacoes         string     `json:"observacoes,omitempty"`
}

// ServidorItem representa um registro de servidor público e remuneração.
type ServidorItem struct {
	Nome          string  `json:"nome"`
	Matricula     string  `json:"matricula,omitempty"`
	CPFParcial    string  `json:"cpf_parcial,omitempty"`
	Cargo         string  `json:"cargo,omitempty"`
	Funcao        string  `json:"funcao,omitempty"`
	Lotacao       string  `json:"lotacao,omitempty"`
	LocalTrabalho string  `json:"local_trabalho,omitempty"`
	Vinculo       string  `json:"vinculo,omitempty"`
	Situacao      string  `json:"situacao,omitempty"`
	SalarioBase   float64 `json:"salario_base,omitempty"`
	SalarioBruto  float64 `json:"salario_bruto,omitempty"`
	Descontos     float64 `json:"descontos,omitempty"`
	SalarioLiquido float64 `json:"salario_liquido,omitempty"`
	Competencia   string  `json:"competencia,omitempty"`
	CargaHoraria  string  `json:"carga_horaria,omitempty"`
	DataAdmissao  string  `json:"data_admissao,omitempty"`
	Orgao         string  `json:"orgao,omitempty"`
	Fonte         string  `json:"fonte"`
	PortalURL     string  `json:"portal_url,omitempty"`
}

// ServidoresResult consolida a consulta de servidores públicos de um município.
type ServidoresResult struct {
	Municipio      *Municipio     `json:"municipio"`
	Fonte          string         `json:"fonte"`
	Plataforma     string         `json:"plataforma"`
	PortalURL      string         `json:"portal_url,omitempty"`
	TotalRegistros int            `json:"total_registros"`
	Servidores     []ServidorItem `json:"servidores"`
	Disponivel     bool           `json:"disponivel"`
	Mensagem       string         `json:"mensagem,omitempty"`
}

// DespesaItem representa um empenho ou despesa pública municipal.
type DespesaItem struct {
	NumeroEmpenho       string  `json:"numero_empenho,omitempty"`
	Ano                 int     `json:"ano,omitempty"`
	Data                string  `json:"data,omitempty"`
	Fornecedor          string  `json:"fornecedor,omitempty"`
	DocumentoFornecedor string  `json:"documento_fornecedor,omitempty"`
	Orgao               string  `json:"orgao,omitempty"`
	Unidade             string  `json:"unidade,omitempty"`
	Funcao              string  `json:"funcao,omitempty"`
	Elemento            string  `json:"elemento,omitempty"`
	Historico           string  `json:"historico,omitempty"`
	ValorEmpenhado      float64 `json:"valor_empenhado,omitempty"`
	ValorLiquidado      float64 `json:"valor_liquidado,omitempty"`
	ValorPago           float64 `json:"valor_pago,omitempty"`
	Fonte               string  `json:"fonte"`
}

// DespesasResult consolida a consulta de despesas municipais.
type DespesasResult struct {
	Municipio      *Municipio    `json:"municipio"`
	Fonte          string        `json:"fonte"`
	Plataforma     string        `json:"plataforma"`
	PortalURL      string        `json:"portal_url,omitempty"`
	TotalRegistros int           `json:"total_registros"`
	Despesas       []DespesaItem `json:"despesas"`
	Disponivel     bool          `json:"disponivel"`
	Mensagem       string        `json:"mensagem,omitempty"`
}

// ReceitaItem representa uma receita pública municipal.
type ReceitaItem struct {
	Codigo          string  `json:"codigo,omitempty"`
	Descricao       string  `json:"descricao"`
	Categoria       string  `json:"categoria,omitempty"`
	Data            string  `json:"data,omitempty"`
	ValorPrevisto   float64 `json:"valor_previsto,omitempty"`
	ValorArrecadado float64 `json:"valor_arrecadado,omitempty"`
	Fonte           string  `json:"fonte"`
}

// ReceitasResult consolida a consulta de receitas municipais.
type ReceitasResult struct {
	Municipio      *Municipio    `json:"municipio"`
	Fonte          string        `json:"fonte"`
	Plataforma     string        `json:"plataforma"`
	PortalURL      string        `json:"portal_url,omitempty"`
	TotalRegistros int           `json:"total_registros"`
	Receitas       []ReceitaItem `json:"receitas"`
	Disponivel     bool          `json:"disponivel"`
	Mensagem       string        `json:"mensagem,omitempty"`
}

// LicitacaoItem representa um processo licitatório ou contratação publicada.
type LicitacaoItem struct {
	Numero          string  `json:"numero,omitempty"`
	Ano             int     `json:"ano,omitempty"`
	Modalidade      string  `json:"modalidade,omitempty"`
	Situacao        string  `json:"situacao,omitempty"`
	DataAbertura    string  `json:"data_abertura,omitempty"`
	Objeto          string  `json:"objeto"`
	ValorEstimado   float64 `json:"valor_estimado,omitempty"`
	ValorHomologado float64 `json:"valor_homologado,omitempty"`
	Link            string  `json:"link,omitempty"`
	Fonte           string  `json:"fonte"`
}

// LicitacoesResult consolida a consulta de licitações municipais.
type LicitacoesResult struct {
	Municipio      *Municipio      `json:"municipio"`
	Fonte          string          `json:"fonte"`
	Plataforma     string          `json:"plataforma"`
	PortalURL      string          `json:"portal_url,omitempty"`
	TotalRegistros int             `json:"total_registros"`
	Licitacoes     []LicitacaoItem `json:"licitacoes"`
	Disponivel     bool            `json:"disponivel"`
	Mensagem       string          `json:"mensagem,omitempty"`
}

// ContratoItem representa um contrato administrativo municipal.
type ContratoItem struct {
	Numero              string  `json:"numero,omitempty"`
	Ano                 int     `json:"ano,omitempty"`
	Fornecedor          string  `json:"fornecedor,omitempty"`
	DocumentoFornecedor string  `json:"documento_fornecedor,omitempty"`
	Objeto              string  `json:"objeto"`
	Situacao            string  `json:"situacao,omitempty"`
	DataAssinatura      string  `json:"data_assinatura,omitempty"`
	VigenciaInicio      string  `json:"vigencia_inicio,omitempty"`
	VigenciaFim         string  `json:"vigencia_fim,omitempty"`
	Valor               float64 `json:"valor,omitempty"`
	Fonte               string  `json:"fonte"`
}

// ContratosResult consolida a consulta de contratos municipais.
type ContratosResult struct {
	Municipio      *Municipio     `json:"municipio"`
	Fonte          string         `json:"fonte"`
	Plataforma     string         `json:"plataforma"`
	PortalURL      string         `json:"portal_url,omitempty"`
	TotalRegistros int            `json:"total_registros"`
	Contratos      []ContratoItem `json:"contratos"`
	Disponivel     bool           `json:"disponivel"`
	Mensagem       string         `json:"mensagem,omitempty"`
}

// DiariaItem representa um pagamento de diária ou passagem a servidor/agente público.
type DiariaItem struct {
	Beneficiario  string  `json:"beneficiario"`
	Matricula     string  `json:"matricula,omitempty"`
	Cargo         string  `json:"cargo,omitempty"`
	Destino       string  `json:"destino,omitempty"`
	Motivo        string  `json:"motivo,omitempty"`
	Data          string  `json:"data,omitempty"`
	DataSaida     string  `json:"data_saida,omitempty"`
	DataChegada   string  `json:"data_chegada,omitempty"`
	NumeroDiarias float64 `json:"numero_diarias,omitempty"`
	Valor         float64 `json:"valor,omitempty"`
	Fonte         string  `json:"fonte"`
}

// DiariasResult consolida a consulta de diárias de servidores/agentes públicos.
type DiariasResult struct {
	Municipio      *Municipio   `json:"municipio"`
	Fonte          string       `json:"fonte"`
	Plataforma     string       `json:"plataforma"`
	PortalURL      string       `json:"portal_url,omitempty"`
	TotalRegistros int          `json:"total_registros"`
	Diarias        []DiariaItem `json:"diarias"`
	Disponivel     bool         `json:"disponivel"`
	Mensagem       string       `json:"mensagem,omitempty"`
}

// ObraItem representa uma obra pública municipal.
type ObraItem struct {
	Codigo      string  `json:"codigo,omitempty"`
	Titulo      string  `json:"titulo"`
	Objeto      string  `json:"objeto,omitempty"`
	Orgao       string  `json:"orgao,omitempty"`
	Endereco    string  `json:"endereco,omitempty"`
	Bairro      string  `json:"bairro,omitempty"`
	DataInicial string  `json:"data_inicial,omitempty"`
	DataFinal   string  `json:"data_final,omitempty"`
	Situacao    string  `json:"situacao,omitempty"`
	Valor       float64 `json:"valor,omitempty"`
	Fonte       string  `json:"fonte"`
}

// ObrasResult consolida a consulta de obras públicas municipais.
type ObrasResult struct {
	Municipio      *Municipio `json:"municipio"`
	Fonte          string     `json:"fonte"`
	Plataforma     string     `json:"plataforma"`
	PortalURL      string     `json:"portal_url,omitempty"`
	TotalRegistros int        `json:"total_registros"`
	Obras          []ObraItem `json:"obras"`
	Disponivel     bool       `json:"disponivel"`
	Mensagem       string     `json:"mensagem,omitempty"`
}

// ConvenioItem representa um convênio ou transferência de recurso municipal.
type ConvenioItem struct {
	Numero        string  `json:"numero,omitempty"`
	Concedente    string  `json:"concedente,omitempty"`
	Convenente    string  `json:"convenente,omitempty"`
	Categoria     string  `json:"categoria,omitempty"`
	Tipo          string  `json:"tipo,omitempty"`
	Situacao      string  `json:"situacao,omitempty"`
	Data          string  `json:"data,omitempty"`
	Vigencia      string  `json:"vigencia,omitempty"`
	Descricao     string  `json:"descricao"`
	Valor         float64 `json:"valor,omitempty"`
	ValorLiberado float64 `json:"valor_liberado,omitempty"`
	Fonte         string  `json:"fonte"`
}

// ConveniosResult consolida a consulta de convênios municipais.
type ConveniosResult struct {
	Municipio      *Municipio     `json:"municipio"`
	Fonte          string         `json:"fonte"`
	Plataforma     string         `json:"plataforma"`
	PortalURL      string         `json:"portal_url,omitempty"`
	TotalRegistros int            `json:"total_registros"`
	Convenios      []ConvenioItem `json:"convenios"`
	Disponivel     bool           `json:"disponivel"`
	Mensagem       string         `json:"mensagem,omitempty"`
}

// LegislacaoItem representa uma lei, decreto, portaria ou ato normativo municipal.
type LegislacaoItem struct {
	Codigo         string `json:"codigo,omitempty"`
	TipoDocumento  string `json:"tipo_documento,omitempty"`
	Numero         string `json:"numero,omitempty"`
	Ano            int    `json:"ano,omitempty"`
	DataPublicacao string `json:"data_publicacao,omitempty"`
	Situacao       string `json:"situacao,omitempty"`
	Nome           string `json:"nome,omitempty"`
	Descricao      string `json:"descricao"`
	Fonte          string `json:"fonte"`
}

// LegislacaoResult consolida a consulta de legislação e atos normativos municipais.
type LegislacaoResult struct {
	Municipio      *Municipio       `json:"municipio"`
	Fonte          string           `json:"fonte"`
	Plataforma     string           `json:"plataforma"`
	PortalURL      string           `json:"portal_url,omitempty"`
	TotalRegistros int              `json:"total_registros"`
	Atos           []LegislacaoItem `json:"atos"`
	Disponivel     bool             `json:"disponivel"`
	Mensagem       string           `json:"mensagem,omitempty"`
}

// ConcursoItem representa um aprovado/convocado em concurso público ou processo seletivo.
type ConcursoItem struct {
	Edital         string `json:"edital,omitempty"`
	DataPublicacao string `json:"data_publicacao,omitempty"`
	Nome           string `json:"nome"`
	Cargo          string `json:"cargo,omitempty"`
	Inscricao      string `json:"inscricao,omitempty"`
	Classificacao  int    `json:"classificacao,omitempty"`
	Situacao       string `json:"situacao,omitempty"`
	Fonte          string `json:"fonte"`
}

// ConcursosResult consolida a consulta de concursos e processos seletivos municipais.
type ConcursosResult struct {
	Municipio      *Municipio     `json:"municipio"`
	Fonte          string         `json:"fonte"`
	Plataforma     string         `json:"plataforma"`
	PortalURL      string         `json:"portal_url,omitempty"`
	TotalRegistros int            `json:"total_registros"`
	Aprovados      []ConcursoItem `json:"aprovados"`
	Disponivel     bool           `json:"disponivel"`
	Mensagem       string         `json:"mensagem,omitempty"`
}

// VerbaItem representa uma rubrica individual no detalhamento do contracheque (Megasoft).
type VerbaItem struct {
	Descricao string  `json:"descricao"`
	Valor     float64 `json:"valor"`
}

// ServidorDetalheResult representa o contracheque detalhado de um servidor (LGPD: CPF mascarado).
type ServidorDetalheResult struct {
	Municipio      *Municipio  `json:"municipio"`
	Nome           string      `json:"nome"`
	Matricula      string      `json:"matricula"`
	CPFParcial     string      `json:"cpf_parcial,omitempty"`
	Cargo          string      `json:"cargo,omitempty"`
	Orgao          string      `json:"orgao,omitempty"`
	Departamento   string      `json:"departamento,omitempty"`
	Vinculo        string      `json:"vinculo,omitempty"`
	Situacao       string      `json:"situacao,omitempty"`
	Competencia    string      `json:"competencia,omitempty"`
	DataAdmissao   string      `json:"data_admissao,omitempty"`
	CargaHoraria   string      `json:"carga_horaria,omitempty"`
	SalarioBruto   float64     `json:"salario_bruto"`
	Descontos      float64     `json:"descontos"`
	SalarioLiquido float64     `json:"salario_liquido"`
	Vencimentos    []VerbaItem `json:"vencimentos,omitempty"`
	VerbasDesconto []VerbaItem `json:"verbas_desconto,omitempty"`
	Ferias         []VerbaItem `json:"ferias,omitempty"`
	DecimoTerceiro []VerbaItem `json:"decimo_terceiro,omitempty"`
	PortalURL      string      `json:"portal_url,omitempty"`
	Fonte          string      `json:"fonte"`
	Disponivel     bool        `json:"disponivel"`
	Mensagem       string      `json:"mensagem,omitempty"`
}

// MegasoftMunicipio representa um município com portal Megasoft verificado no catálogo nacional.
type MegasoftMunicipio struct {
	CodigoIBGE     string `json:"codigo_ibge"`
	Nome           string `json:"nome"`
	UF             string `json:"uf"`
	PortalURL      string `json:"portal_url"`
	PortalCamara   string `json:"portal_camara,omitempty"`
	CodigoOrgao    int    `json:"codigo_orgao,omitempty"`
	SitePrefeitura string `json:"site_prefeitura,omitempty"`
}

// PortalCatalogMunicipio representa um município com portal/API municipal verificado no catálogo nacional.
type PortalCatalogMunicipio struct {
	CodigoIBGE   string `json:"codigo_ibge"`
	Nome         string `json:"nome"`
	UF           string `json:"uf"`
	Plataforma   string `json:"plataforma"`
	PortalURL    string `json:"portal_url"`
	TenantID     string `json:"tenant_id,omitempty"`
	EntidadeNome string `json:"entidade_nome,omitempty"`
}

// TransparenciaCompletaReport consolida toda a transparência disponível para um município.
type TransparenciaCompletaReport struct {
	Dossier           *MunicipioDossier      `json:"dossier"`
	Discovery         *PortalDiscoveryResult `json:"discovery,omitempty"`
	Servidores        *ServidoresResult      `json:"servidores,omitempty"`
	Diarias           *DiariasResult         `json:"diarias,omitempty"`
	Despesas          *DespesasResult        `json:"despesas,omitempty"`
	Receitas          *ReceitasResult        `json:"receitas,omitempty"`
	Licitacoes        *LicitacoesResult      `json:"licitacoes,omitempty"`
	Contratos         *ContratosResult       `json:"contratos,omitempty"`
	Obras             *ObrasResult           `json:"obras,omitempty"`
	Convenios         *ConveniosResult       `json:"convenios,omitempty"`
	Legislacao        *LegislacaoResult      `json:"legislacao,omitempty"`
	Concursos         *ConcursosResult       `json:"concursos,omitempty"`
	SiconfiDCA        []FiscalItem           `json:"siconfi_dca,omitempty"`
	SiconfiRGF        []FiscalItem           `json:"siconfi_rgf,omitempty"`
	SiconfiEntregas   []EntregaItem          `json:"siconfi_entregas,omitempty"`
	FontesConsultadas []string               `json:"fontes_consultadas"`
	Limitacoes        []string               `json:"limitacoes,omitempty"`
	TempoTotalMs      int64                  `json:"tempo_total_ms"`
}


