package transparencia

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	defaultTCEPEBaseURL = "https://sistemas.tce.pe.gov.br/DadosAbertos"
	defaultTCERJBaseURL = "https://dados.tcerj.tc.br/api/v1"
	defaultTCERNBaseURL = "https://apidadosabertos.tce.rn.gov.br/api"
	defaultTCESCBaseURL = "https://servicos.tcesc.tc.br/endpoints-portal-transparencia"
	defaultTCETOBaseURL = "https://api.tceto.tc.br/econtas/api"
	defaultTCEPABaseURL = "https://sistemas.tcepa.tc.br/dadosabertos/api"
	defaultTCERSBaseURL = "https://dados.tce.rs.gov.br"
)

type tceExtraConfig struct {
	peBaseURL string
	rjBaseURL string
	rnBaseURL string
	scBaseURL string
	toBaseURL string
	paBaseURL string
	rsBaseURL string

	mu      sync.Mutex
	rnJuris []RNJurisdicionado
	scMunis []SCMunicipio
	rsMunis []RSMunicipio
}

var (
	extraCfgMu sync.RWMutex
	extraCfgs  = make(map[*TCEClient]*tceExtraConfig)
)

func (c *TCEClient) extraCfg() *tceExtraConfig {
	extraCfgMu.RLock()
	cfg := extraCfgs[c]
	extraCfgMu.RUnlock()
	if cfg != nil {
		return cfg
	}
	extraCfgMu.Lock()
	defer extraCfgMu.Unlock()
	if cfg = extraCfgs[c]; cfg == nil {
		cfg = &tceExtraConfig{
			peBaseURL: defaultTCEPEBaseURL,
			rjBaseURL: defaultTCERJBaseURL,
			rnBaseURL: defaultTCERNBaseURL,
			scBaseURL: defaultTCESCBaseURL,
			toBaseURL: defaultTCETOBaseURL,
			paBaseURL: defaultTCEPABaseURL,
			rsBaseURL: defaultTCERSBaseURL,
		}
		extraCfgs[c] = cfg
	}
	return cfg
}

// SetTCEPEBaseURL configura a URL base do TCE-PE (útil para testes).
func (c *TCEClient) SetTCEPEBaseURL(u string) {
	cfg := c.extraCfg()
	cfg.mu.Lock()
	cfg.peBaseURL = strings.TrimRight(u, "/")
	cfg.mu.Unlock()
}

// SetTCERJBaseURL configura a URL base do TCE-RJ (útil para testes).
func (c *TCEClient) SetTCERJBaseURL(u string) {
	cfg := c.extraCfg()
	cfg.mu.Lock()
	cfg.rjBaseURL = strings.TrimRight(u, "/")
	cfg.mu.Unlock()
}

// SetTCERNBaseURL configura a URL base do TCE-RN (útil para testes).
func (c *TCEClient) SetTCERNBaseURL(u string) {
	cfg := c.extraCfg()
	cfg.mu.Lock()
	cfg.rnBaseURL = strings.TrimRight(u, "/")
	cfg.rnJuris = nil
	cfg.mu.Unlock()
}

// SetTCESCBaseURL configura a URL base do TCE-SC (útil para testes).
func (c *TCEClient) SetTCESCBaseURL(u string) {
	cfg := c.extraCfg()
	cfg.mu.Lock()
	cfg.scBaseURL = strings.TrimRight(u, "/")
	cfg.scMunis = nil
	cfg.mu.Unlock()
}

// SetTCETOBaseURL configura a URL base do TCE-TO (útil para testes).
func (c *TCEClient) SetTCETOBaseURL(u string) {
	cfg := c.extraCfg()
	cfg.mu.Lock()
	cfg.toBaseURL = strings.TrimRight(u, "/")
	cfg.mu.Unlock()
}

// SetTCEPABaseURL configura a URL base do TCE-PA (útil para testes).
func (c *TCEClient) SetTCEPABaseURL(u string) {
	cfg := c.extraCfg()
	cfg.mu.Lock()
	cfg.paBaseURL = strings.TrimRight(u, "/")
	cfg.mu.Unlock()
}

// SetTCERSBaseURL configura a URL base do TCE-RS (útil para testes).
func (c *TCEClient) SetTCERSBaseURL(u string) {
	cfg := c.extraCfg()
	cfg.mu.Lock()
	cfg.rsBaseURL = strings.TrimRight(u, "/")
	cfg.rsMunis = nil
	cfg.mu.Unlock()
}

func (c *TCEClient) getPEBaseURL() string {
	cfg := c.extraCfg()
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	return cfg.peBaseURL
}

func (c *TCEClient) getRJBaseURL() string {
	cfg := c.extraCfg()
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	return cfg.rjBaseURL
}

func (c *TCEClient) getRNBaseURL() string {
	cfg := c.extraCfg()
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	return cfg.rnBaseURL
}

func (c *TCEClient) getSCBaseURL() string {
	cfg := c.extraCfg()
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	return cfg.scBaseURL
}

func (c *TCEClient) getTOBaseURL() string {
	cfg := c.extraCfg()
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	return cfg.toBaseURL
}

func (c *TCEClient) getPABaseURL() string {
	cfg := c.extraCfg()
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	return cfg.paBaseURL
}

func (c *TCEClient) getRSBaseURL() string {
	cfg := c.extraCfg()
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	return cfg.rsBaseURL
}

// ---------------------------------------------------------------------------
// Helpers compartilhados
// ---------------------------------------------------------------------------

// iso88591ToUTF8 converte bytes ISO-8859-1 para UTF-8 quando necessário.
// A API do TCE-PE retorna respostas codificadas em ISO-8859-1.
func iso88591ToUTF8(b []byte) []byte {
	if utf8.Valid(b) {
		return b
	}
	var sb strings.Builder
	sb.Grow(len(b) + len(b)/4)
	for _, ch := range b {
		sb.WriteRune(rune(ch))
	}
	return []byte(sb.String())
}

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

func stripHTMLTags(s string) string {
	cleaned := htmlTagRe.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(cleaned), " ")
}

func digitsOnly(s string) string {
	var sb strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// ---------------------------------------------------------------------------
// TCE-PE (https://sistemas.tce.pe.gov.br/DadosAbertos)
// Padrão: {API_BASE}/{Entity}!json?{params}
// Wrapper: {"resposta": {"status": "OK", "conteudo": [...]}}
// ---------------------------------------------------------------------------

// PEUnidade representa uma unidade jurisdicionada no TCE-PE.
type PEUnidade struct {
	Codigo          any    `json:"Codigo"`
	Nome            string `json:"Nome"`
	Natureza        string `json:"Natureza"`
	Municipio       string `json:"Municipio"`
	CodigoMunicipio string `json:"CodigoMunicipio"`
}

type tcePEResponse struct {
	Resposta struct {
		Status   string            `json:"status"`
		Entidade string            `json:"entidade"`
		Conteudo []json.RawMessage `json:"conteudo"`
	} `json:"resposta"`
}

func (c *TCEClient) fetchTCEPE(ctx context.Context, entity string, params url.Values) ([]json.RawMessage, error) {
	u := fmt.Sprintf("%s/%s!json", c.getPEBaseURL(), entity)
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	body, status, err := doJSONGet(ctx, c.client, u)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("TCE-PE %s retornou HTTP %d", entity, status)
	}
	utf8Body := iso88591ToUTF8(body)
	var wrap tcePEResponse
	if err := json.Unmarshal(utf8Body, &wrap); err != nil {
		return nil, err
	}
	if wrap.Resposta.Status != "OK" {
		return nil, fmt.Errorf("TCE-PE %s retornou status %q", entity, wrap.Resposta.Status)
	}
	return wrap.Resposta.Conteudo, nil
}

// ResolvePEUnidades consulta unidades jurisdicionadas (prefeituras) no TCE-PE.
func (c *TCEClient) ResolvePEUnidades(ctx context.Context, municipio string) ([]PEUnidade, error) {
	params := url.Values{"NATUREZA": {"prefeitura"}}
	if strings.TrimSpace(municipio) != "" {
		params.Set("MUNICIPIO", strings.TrimSpace(municipio))
	}
	raw, err := c.fetchTCEPE(ctx, "UnidadesJurisdicionadas", params)
	if err != nil {
		return nil, err
	}
	out := make([]PEUnidade, 0, len(raw))
	for _, r := range raw {
		var u PEUnidade
		if json.Unmarshal(r, &u) == nil {
			out = append(out, u)
		}
	}
	return out, nil
}

// FetchTCEPEDespesas consulta despesas municipais no TCE-PE (SAGRES).
func (c *TCEClient) FetchTCEPEDespesas(ctx context.Context, m *Municipio, ano, mes, limit int) (*DespesasResult, error) {
	baseURL := c.getPEBaseURL()
	limit = normalizeLimit(limit)
	years := []int{ano}
	if ano <= 0 {
		nowY := time.Now().Year()
		years = []int{nowY, nowY - 1, nowY - 2}
	}

	type peDespesa struct {
		NumeroEmpenho   string  `json:"NUMEROEMPENHO"`
		AnoReferencia   int     `json:"ANOREFERENCIA"`
		MesReferencia   int     `json:"MESREFERENCIA"`
		Fornecedor      string  `json:"FORNECEDOR"`
		CPFCNPJ         string  `json:"CPF_CNPJ"`
		Historico       string  `json:"HISTORICO"`
		ValorEmpenhado  float64 `json:"VALOREMPENHADO"`
		ValorLiquidado  float64 `json:"VALORLIQUIDADO"`
		ValorPago       float64 `json:"VALORPAGO"`
		Funcao          string  `json:"FUNCAO"`
		ElementoDespesa string  `json:"ELEMENTODESPESA"`
		UnidadeGestora  string  `json:"NOMEUNIDADEGESTORA"`
	}

	for _, y := range years {
		params := url.Values{"ANOREFERENCIA": {fmt.Sprintf("%d", y)}}
		if mes > 0 && mes <= 12 {
			params.Set("MESREFERENCIA", fmt.Sprintf("%d", mes))
		}
		if m != nil && strings.TrimSpace(m.Nome) != "" {
			params.Set("MUNICIPIO", strings.TrimSpace(m.Nome))
		}
		raw, err := c.fetchTCEPE(ctx, "DespesasMunicipais", params)
		if err != nil || len(raw) == 0 {
			continue
		}
		items := make([]DespesaItem, 0, minInt(limit, len(raw)))
		for _, r := range raw {
			var d peDespesa
			if json.Unmarshal(r, &d) != nil {
				continue
			}
			dataStr := ""
			if d.MesReferencia > 0 && d.AnoReferencia > 0 {
				dataStr = fmt.Sprintf("%02d/%04d", d.MesReferencia, d.AnoReferencia)
			}
			items = append(items, DespesaItem{
				NumeroEmpenho:       strings.TrimSpace(d.NumeroEmpenho),
				Ano:                 d.AnoReferencia,
				Data:                dataStr,
				Fornecedor:          strings.TrimSpace(d.Fornecedor),
				DocumentoFornecedor: maskIfCPF(d.CPFCNPJ),
				Orgao:               strings.TrimSpace(d.UnidadeGestora),
				Funcao:              strings.TrimSpace(d.Funcao),
				Elemento:            strings.TrimSpace(d.ElementoDespesa),
				Historico:           strings.TrimSpace(d.Historico),
				ValorEmpenhado:      d.ValorEmpenhado,
				ValorLiquidado:      d.ValorLiquidado,
				ValorPago:           d.ValorPago,
				Fonte:               "TCE-PE Dados Abertos (SAGRES)",
			})
			if len(items) >= limit {
				break
			}
		}
		return &DespesasResult{
			Municipio:      m,
			Fonte:          "TCE-PE Dados Abertos (SAGRES)",
			Plataforma:     "tce_pe",
			PortalURL:      baseURL,
			TotalRegistros: len(raw),
			Despesas:       items,
			Disponivel:     true,
		}, nil
	}

	return &DespesasResult{
		Municipio:  m,
		Fonte:      "TCE-PE Dados Abertos (SAGRES)",
		Plataforma: "tce_pe",
		PortalURL:  baseURL,
		Despesas:   []DespesaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma despesa retornada pelo TCE-PE para o período.",
	}, nil
}

// FetchTCEPELicitacoes consulta licitações municipais no TCE-PE (LICON).
func (c *TCEClient) FetchTCEPELicitacoes(ctx context.Context, m *Municipio, ano, limit int) (*LicitacoesResult, error) {
	baseURL := c.getPEBaseURL()
	limit = normalizeLimit(limit)
	years := []int{ano}
	if ano <= 0 {
		nowY := time.Now().Year()
		years = []int{nowY, nowY - 1, nowY - 2}
	}

	type peLicitacao struct {
		NumeroLicitacao string  `json:"NUMEROLICITACAO"`
		AnoLicitacao    int     `json:"ANOLICITACAO"`
		Modalidade      string  `json:"MODALIDADE"`
		Objeto          string  `json:"OBJETO"`
		ValorEstimado   float64 `json:"VALORESTIMADO"`
		Situacao        string  `json:"SITUACAO"`
	}

	for _, y := range years {
		params := url.Values{"ANOLICITACAO": {fmt.Sprintf("%d", y)}}
		if m != nil && strings.TrimSpace(m.Nome) != "" {
			params.Set("MUNICIPIO", strings.TrimSpace(m.Nome))
		}
		raw, err := c.fetchTCEPE(ctx, "LicitacaoUG", params)
		if err != nil || len(raw) == 0 {
			continue
		}
		items := make([]LicitacaoItem, 0, minInt(limit, len(raw)))
		for _, r := range raw {
			var l peLicitacao
			if json.Unmarshal(r, &l) != nil {
				continue
			}
			items = append(items, LicitacaoItem{
				Numero:        strings.TrimSpace(l.NumeroLicitacao),
				Ano:           l.AnoLicitacao,
				Modalidade:    strings.TrimSpace(l.Modalidade),
				Situacao:      strings.TrimSpace(l.Situacao),
				Objeto:        strings.TrimSpace(l.Objeto),
				ValorEstimado: l.ValorEstimado,
				Fonte:         "TCE-PE Dados Abertos (LICON)",
			})
			if len(items) >= limit {
				break
			}
		}
		return &LicitacoesResult{
			Municipio:      m,
			Fonte:          "TCE-PE Dados Abertos (LICON)",
			Plataforma:     "tce_pe",
			PortalURL:      baseURL,
			TotalRegistros: len(raw),
			Licitacoes:     items,
			Disponivel:     true,
		}, nil
	}

	return &LicitacoesResult{
		Municipio:  m,
		Fonte:      "TCE-PE Dados Abertos (LICON)",
		Plataforma: "tce_pe",
		PortalURL:  baseURL,
		Licitacoes: []LicitacaoItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma licitação retornada pelo TCE-PE para o período.",
	}, nil
}

// FetchTCEPEContratos consulta contratos municipais no TCE-PE (LICON).
func (c *TCEClient) FetchTCEPEContratos(ctx context.Context, m *Municipio, ano, limit int) (*ContratosResult, error) {
	baseURL := c.getPEBaseURL()
	limit = normalizeLimit(limit)
	years := []int{ano}
	if ano <= 0 {
		nowY := time.Now().Year()
		years = []int{nowY, nowY - 1, nowY - 2}
	}

	type peContrato struct {
		NumeroContrato string  `json:"NUMEROCONTRATO"`
		AnoReferencia  int     `json:"ANOREFERENCIA"`
		Objeto         string  `json:"OBJETO"`
		ValorContrato  float64 `json:"VALORCONTRATO"`
		Fornecedor     string  `json:"FORNECEDOR"`
		CPFCNPJ        string  `json:"CPFCNPJ"`
	}

	for _, y := range years {
		params := url.Values{"ANOREFERENCIA": {fmt.Sprintf("%d", y)}}
		if m != nil && strings.TrimSpace(m.Nome) != "" {
			params.Set("MUNICIPIO", strings.TrimSpace(m.Nome))
		}
		raw, err := c.fetchTCEPE(ctx, "Contratos", params)
		if err != nil || len(raw) == 0 {
			continue
		}
		items := make([]ContratoItem, 0, minInt(limit, len(raw)))
		for _, r := range raw {
			var ct peContrato
			if json.Unmarshal(r, &ct) != nil {
				continue
			}
			items = append(items, ContratoItem{
				Numero:              strings.TrimSpace(ct.NumeroContrato),
				Ano:                 ct.AnoReferencia,
				Fornecedor:          strings.TrimSpace(ct.Fornecedor),
				DocumentoFornecedor: maskIfCPF(ct.CPFCNPJ),
				Objeto:              strings.TrimSpace(ct.Objeto),
				Valor:               ct.ValorContrato,
				Fonte:               "TCE-PE Dados Abertos (LICON)",
			})
			if len(items) >= limit {
				break
			}
		}
		return &ContratosResult{
			Municipio:      m,
			Fonte:          "TCE-PE Dados Abertos (LICON)",
			Plataforma:     "tce_pe",
			PortalURL:      baseURL,
			TotalRegistros: len(raw),
			Contratos:      items,
			Disponivel:     true,
		}, nil
	}

	return &ContratosResult{
		Municipio:  m,
		Fonte:      "TCE-PE Dados Abertos (LICON)",
		Plataforma: "tce_pe",
		PortalURL:  baseURL,
		Contratos:  []ContratoItem{},
		Disponivel: true,
		Mensagem:   "Nenhum contrato retornado pelo TCE-PE para o período.",
	}, nil
}

// ---------------------------------------------------------------------------
// TCE-RJ (https://dados.tcerj.tc.br/api/v1)
// Endpoints: /licitacoes, /contratos_municipio, /obras_paralisadas
// ---------------------------------------------------------------------------

// FetchTCERJLicitacoes consulta licitações municipais no TCE-RJ.
func (c *TCEClient) FetchTCERJLicitacoes(ctx context.Context, m *Municipio, ano, limit int) (*LicitacoesResult, error) {
	baseURL := c.getRJBaseURL()
	limit = normalizeLimit(limit)

	params := url.Values{
		"inicio": {"0"},
		"limite": {fmt.Sprintf("%d", limit)},
	}
	if ano > 0 {
		params.Set("ano", fmt.Sprintf("%d", ano))
	}
	if m != nil && strings.TrimSpace(m.Nome) != "" {
		params.Set("municipio", strings.TrimSpace(m.Nome))
	}

	u := baseURL + "/licitacoes?" + params.Encode()
	body, status, err := doJSONGet(ctx, c.client, u)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("TCE-RJ /licitacoes retornou HTTP %d", status)
	}

	type rjLic struct {
		Ente                  string  `json:"Ente"`
		Unidade               string  `json:"Unidade"`
		Ano                   int     `json:"Ano"`
		Modalidade            string  `json:"Modalidade"`
		ProcessoLicitatorio   string  `json:"ProcessoLicitatorio"`
		NumeroEdital          string  `json:"NumeroEdital"`
		Objeto                string  `json:"Objeto"`
		ValorEstimado         float64 `json:"ValorEstimado"`
		DataPublicacaoOficial string  `json:"DataPublicacaoOficial"`
	}

	var rows []rjLic
	total := 0
	var wrapped struct {
		Licitacoes []rjLic `json:"Licitacoes"`
		Count      int     `json:"Count"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil && wrapped.Licitacoes != nil {
		rows = wrapped.Licitacoes
		total = wrapped.Count
	} else if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}
	if total == 0 {
		total = len(rows)
	}

	items := make([]LicitacaoItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		num := strings.TrimSpace(r.NumeroEdital)
		if num == "" {
			num = strings.TrimSpace(r.ProcessoLicitatorio)
		}
		items = append(items, LicitacaoItem{
			Numero:        num,
			Ano:           r.Ano,
			Modalidade:    strings.TrimSpace(r.Modalidade),
			DataAbertura:  strings.TrimSpace(r.DataPublicacaoOficial),
			Objeto:        strings.TrimSpace(r.Objeto),
			ValorEstimado: r.ValorEstimado,
			Fonte:         "TCE-RJ Dados Abertos",
		})
		if len(items) >= limit {
			break
		}
	}

	return &LicitacoesResult{
		Municipio:      m,
		Fonte:          "TCE-RJ Dados Abertos",
		Plataforma:     "tce_rj",
		PortalURL:      baseURL,
		TotalRegistros: total,
		Licitacoes:     items,
		Disponivel:     true,
	}, nil
}

// FetchTCERJContratos consulta contratos municipais no TCE-RJ.
func (c *TCEClient) FetchTCERJContratos(ctx context.Context, m *Municipio, ano, limit int) (*ContratosResult, error) {
	baseURL := c.getRJBaseURL()
	limit = normalizeLimit(limit)

	params := url.Values{
		"inicio": {"0"},
		"limite": {fmt.Sprintf("%d", limit)},
	}
	if ano > 0 {
		params.Set("ano", fmt.Sprintf("%d", ano))
	}
	if m != nil && strings.TrimSpace(m.Nome) != "" {
		params.Set("municipio", strings.TrimSpace(m.Nome))
	}

	u := baseURL + "/contratos_municipio?" + params.Encode()
	body, status, err := doJSONGet(ctx, c.client, u)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("TCE-RJ /contratos_municipio retornou HTTP %d", status)
	}

	type rjContrato struct {
		Ente                   string  `json:"Ente"`
		NumeroContrato         string  `json:"NumeroContrato"`
		AnoContrato            int     `json:"AnoContrato"`
		Contratado             string  `json:"Contratado"`
		CNPJCPFContratado      string  `json:"CNPJCPFContratado"`
		Objeto                 string  `json:"Objeto"`
		ValorContrato          float64 `json:"ValorContrato"`
		DataAssinaturaContrato string  `json:"DataAssinaturaContrato"`
		DataVencimentoContrato string  `json:"DataVencimentoContrato"`
	}

	var rows []rjContrato
	total := 0
	var wrapped struct {
		Contratos []rjContrato `json:"Contratos"`
		Count     int          `json:"Count"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil && wrapped.Contratos != nil {
		rows = wrapped.Contratos
		total = wrapped.Count
	} else if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}
	if total == 0 {
		total = len(rows)
	}

	items := make([]ContratoItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		items = append(items, ContratoItem{
			Numero:              strings.TrimSpace(r.NumeroContrato),
			Ano:                 r.AnoContrato,
			Fornecedor:          strings.TrimSpace(r.Contratado),
			DocumentoFornecedor: maskIfCPF(r.CNPJCPFContratado),
			Objeto:              strings.TrimSpace(r.Objeto),
			DataAssinatura:      strings.TrimSpace(r.DataAssinaturaContrato),
			VigenciaFim:         strings.TrimSpace(r.DataVencimentoContrato),
			Valor:               r.ValorContrato,
			Fonte:               "TCE-RJ Dados Abertos",
		})
		if len(items) >= limit {
			break
		}
	}

	return &ContratosResult{
		Municipio:      m,
		Fonte:          "TCE-RJ Dados Abertos",
		Plataforma:     "tce_rj",
		PortalURL:      baseURL,
		TotalRegistros: total,
		Contratos:      items,
		Disponivel:     true,
	}, nil
}

// FetchTCERJObrasParalisadas consulta obras públicas paralisadas no TCE-RJ, filtrando pelo município quando informado.
func (c *TCEClient) FetchTCERJObrasParalisadas(ctx context.Context, m *Municipio, limit int) (*ObrasResult, error) {
	baseURL := c.getRJBaseURL()
	limit = normalizeLimit(limit)

	body, status, err := doJSONGet(ctx, c.client, baseURL+"/obras_paralisadas")
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("TCE-RJ /obras_paralisadas retornou HTTP %d", status)
	}

	type rjObra struct {
		Ente               string  `json:"Ente"`
		TipoEnte           string  `json:"TipoEnte"`
		Nome               string  `json:"Nome"`
		FuncaoGoverno      string  `json:"FuncaoGoverno"`
		ValorTotalContrato float64 `json:"ValorTotalContrato"`
		MotivoParalisacao  string  `json:"MotivoParalisacao"`
		DataInicioObra     string  `json:"DataInicioObra"`
		DataParalisacao    string  `json:"DataParalisacao"`
		StatusContrato     string  `json:"StatusContrato"`
	}

	var rows []rjObra
	var wrapped struct {
		Obras []rjObra `json:"Obras"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil && wrapped.Obras != nil {
		rows = wrapped.Obras
	} else if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}

	targetNorm := ""
	if m != nil && strings.TrimSpace(m.Nome) != "" {
		targetNorm = RemoveAccents(m.Nome)
	}

	items := make([]ObraItem, 0, minInt(limit, len(rows)))
	matched := 0
	for _, r := range rows {
		if targetNorm != "" && !strings.Contains(RemoveAccents(r.Ente), targetNorm) {
			continue
		}
		matched++
		if len(items) < limit {
			sit := strings.TrimSpace(r.StatusContrato)
			if r.MotivoParalisacao != "" {
				if sit != "" {
					sit += " — " + strings.TrimSpace(r.MotivoParalisacao)
				} else {
					sit = strings.TrimSpace(r.MotivoParalisacao)
				}
			}
			items = append(items, ObraItem{
				Titulo:      strings.TrimSpace(r.Nome),
				Orgao:       strings.TrimSpace(r.Ente),
				DataInicial: strings.TrimSpace(r.DataInicioObra),
				DataFinal:   strings.TrimSpace(r.DataParalisacao),
				Situacao:    sit,
				Valor:       r.ValorTotalContrato,
				Fonte:       "TCE-RJ Dados Abertos",
			})
		}
	}

	return &ObrasResult{
		Municipio:      m,
		Fonte:          "TCE-RJ Dados Abertos",
		Plataforma:     "tce_rj",
		PortalURL:      baseURL,
		TotalRegistros: matched,
		Obras:          items,
		Disponivel:     true,
	}, nil
}

// ---------------------------------------------------------------------------
// TCE-RN (https://apidadosabertos.tce.rn.gov.br/api)
// Path-param only, flat JSON arrays
// ---------------------------------------------------------------------------

// RNJurisdicionado representa uma entidade jurisdicionada no TCE-RN.
type RNJurisdicionado struct {
	IdentificadorUnidade int    `json:"identificadorUnidade"`
	CodigoOrgao          string `json:"codigoOrgao"`
	NomeOrgao            string `json:"nomeOrgao"`
	CNPJ                 string `json:"cnpj"`
}

// ResolveRNIdUnidade resolve o identificadorUnidade da prefeitura no TCE-RN.
func (c *TCEClient) ResolveRNIdUnidade(ctx context.Context, m *Municipio) (int, error) {
	if m == nil {
		return 0, fmt.Errorf("município obrigatório para TCE-RN")
	}
	cfg := c.extraCfg()
	cfg.mu.Lock()
	cached := cfg.rnJuris
	baseURL := cfg.rnBaseURL
	cfg.mu.Unlock()

	if len(cached) == 0 {
		body, status, err := doJSONGet(ctx, c.client, baseURL+"/InformacoesBasicasApi/JurisdicionadosTCE/Json")
		if err != nil {
			return 0, err
		}
		if status != http.StatusOK {
			return 0, fmt.Errorf("TCE-RN JurisdicionadosTCE retornou HTTP %d", status)
		}
		if err := json.Unmarshal(body, &cached); err != nil {
			return 0, err
		}
		cfg.mu.Lock()
		cfg.rnJuris = cached
		cfg.mu.Unlock()
	}

	targetCNPJ := digitsOnly(m.CNPJ)
	if targetCNPJ != "" {
		for _, j := range cached {
			if digitsOnly(j.CNPJ) == targetCNPJ {
				return j.IdentificadorUnidade, nil
			}
		}
	}

	targetNorm := RemoveAccents(m.Nome)
	for _, j := range cached {
		norm := RemoveAccents(j.NomeOrgao)
		if strings.Contains(norm, "prefeitura") && strings.Contains(norm, targetNorm) {
			return j.IdentificadorUnidade, nil
		}
	}
	for _, j := range cached {
		if strings.Contains(RemoveAccents(j.NomeOrgao), targetNorm) {
			return j.IdentificadorUnidade, nil
		}
	}
	return 0, fmt.Errorf("município %s (%s) não encontrado no TCE-RN", m.Nome, m.CodigoIBGE)
}

// FetchTCERNDespesas consulta despesas orçamentárias (balanço orçamentário) no TCE-RN.
func (c *TCEClient) FetchTCERNDespesas(ctx context.Context, m *Municipio, ano, bimestre, limit int) (*DespesasResult, error) {
	idUnidade, err := c.ResolveRNIdUnidade(ctx, m)
	if err != nil {
		return nil, err
	}
	baseURL := c.getRNBaseURL()
	limit = normalizeLimit(limit)

	years := []int{ano}
	if ano <= 0 {
		nowY := time.Now().Year()
		years = []int{nowY - 1, nowY - 2, nowY}
	}
	bimestres := []int{bimestre}
	if bimestre <= 0 || bimestre > 6 {
		bimestres = []int{6, 4, 2, 1}
	}

	type rnDespesa struct {
		CategoriaEconomica string  `json:"descricaoCategoriaEconomica"`
		GrupoDespesa       string  `json:"descricaoGrupoDespesa"`
		ElementoDespesa    string  `json:"descricaoElementoDespesa"`
		ValorEmpenhado     float64 `json:"valorEmpenhoAtePeriodo"`
		ValorLiquidado     float64 `json:"valorLiquidacaoAtePeriodo"`
		ValorPago          float64 `json:"valorPagoAtePeriodo"`
	}

	for _, y := range years {
		for _, b := range bimestres {
			u := fmt.Sprintf("%s/BalancoOrcamentarioApi/Despesa/Json/%d/%d/%d", baseURL, y, b, idUnidade)
			body, status, err := doJSONGet(ctx, c.client, u)
			if err != nil {
				return nil, err
			}
			if status != http.StatusOK {
				continue
			}
			var rows []rnDespesa
			if json.Unmarshal(body, &rows) != nil || len(rows) == 0 {
				continue
			}
			items := make([]DespesaItem, 0, minInt(limit, len(rows)))
			for _, r := range rows {
				items = append(items, DespesaItem{
					Ano:            y,
					Data:           fmt.Sprintf("%dº bimestre/%d", b, y),
					Elemento:       strings.TrimSpace(r.ElementoDespesa),
					Historico:      strings.TrimSpace(r.CategoriaEconomica + " — " + r.GrupoDespesa),
					ValorEmpenhado: r.ValorEmpenhado,
					ValorLiquidado: r.ValorLiquidado,
					ValorPago:      r.ValorPago,
					Fonte:          "TCE-RN Dados Abertos",
				})
				if len(items) >= limit {
					break
				}
			}
			return &DespesasResult{
				Municipio:      m,
				Fonte:          "TCE-RN Dados Abertos",
				Plataforma:     "tce_rn",
				PortalURL:      baseURL,
				TotalRegistros: len(rows),
				Despesas:       items,
				Disponivel:     true,
			}, nil
		}
	}

	return &DespesasResult{
		Municipio:  m,
		Fonte:      "TCE-RN Dados Abertos",
		Plataforma: "tce_rn",
		PortalURL:  baseURL,
		Despesas:   []DespesaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma despesa retornada pelo TCE-RN para o período.",
	}, nil
}

// FetchTCERNReceitas consulta receitas orçamentárias (balanço orçamentário) no TCE-RN.
func (c *TCEClient) FetchTCERNReceitas(ctx context.Context, m *Municipio, ano, bimestre, limit int) (*ReceitasResult, error) {
	idUnidade, err := c.ResolveRNIdUnidade(ctx, m)
	if err != nil {
		return nil, err
	}
	baseURL := c.getRNBaseURL()
	limit = normalizeLimit(limit)

	years := []int{ano}
	if ano <= 0 {
		nowY := time.Now().Year()
		years = []int{nowY - 1, nowY - 2, nowY}
	}
	bimestres := []int{bimestre}
	if bimestre <= 0 || bimestre > 6 {
		bimestres = []int{6, 4, 2, 1}
	}

	type rnReceita struct {
		Descricao               string  `json:"descricaoReceita"`
		CodNatureza             string  `json:"codNaturezaReceita"`
		ValorPrevistoInicial    float64 `json:"valorPrevistoInicial"`
		ValorPrevistoAtualizado float64 `json:"valorPrevistoAtualizado"`
		ValorRealizadoTypo      float64 `json:"valorRealizadoNoExecicio"`
		ValorRealizado          float64 `json:"valorRealizadoNoExercicio"`
	}

	for _, y := range years {
		for _, b := range bimestres {
			u := fmt.Sprintf("%s/BalancoOrcamentarioApi/Receita/Json/%d/%d/%d", baseURL, y, b, idUnidade)
			body, status, err := doJSONGet(ctx, c.client, u)
			if err != nil {
				return nil, err
			}
			if status != http.StatusOK {
				continue
			}
			var rows []rnReceita
			if json.Unmarshal(body, &rows) != nil || len(rows) == 0 {
				continue
			}
			items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
			for _, r := range rows {
				previsto := r.ValorPrevistoAtualizado
				if previsto == 0 {
					previsto = r.ValorPrevistoInicial
				}
				realizado := r.ValorRealizadoTypo
				if realizado == 0 {
					realizado = r.ValorRealizado
				}
				items = append(items, ReceitaItem{
					Codigo:          strings.TrimSpace(r.CodNatureza),
					Descricao:       strings.TrimSpace(r.Descricao),
					Data:            fmt.Sprintf("%dº bimestre/%d", b, y),
					ValorPrevisto:   previsto,
					ValorArrecadado: realizado,
					Fonte:           "TCE-RN Dados Abertos",
				})
				if len(items) >= limit {
					break
				}
			}
			return &ReceitasResult{
				Municipio:      m,
				Fonte:          "TCE-RN Dados Abertos",
				Plataforma:     "tce_rn",
				PortalURL:      baseURL,
				TotalRegistros: len(rows),
				Receitas:       items,
				Disponivel:     true,
			}, nil
		}
	}

	return &ReceitasResult{
		Municipio:  m,
		Fonte:      "TCE-RN Dados Abertos",
		Plataforma: "tce_rn",
		PortalURL:  baseURL,
		Receitas:   []ReceitaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma receita retornada pelo TCE-RN para o período.",
	}, nil
}

// FetchTCERNLicitacoes consulta licitações públicas no TCE-RN.
func (c *TCEClient) FetchTCERNLicitacoes(ctx context.Context, m *Municipio, ano, limit int) (*LicitacoesResult, error) {
	idUnidade, err := c.ResolveRNIdUnidade(ctx, m)
	if err != nil {
		return nil, err
	}
	baseURL := c.getRNBaseURL()
	limit = normalizeLimit(limit)

	years := []int{ano}
	if ano <= 0 {
		nowY := time.Now().Year()
		years = []int{nowY - 1, nowY - 2, nowY}
	}

	type rnLicitacao struct {
		NumeroLicitacao  string  `json:"numeroLicitacao"`
		AnoLicitacao     any     `json:"anoLicitacao"`
		Modalidade       string  `json:"modalidade"`
		DescricaoObjeto  string  `json:"descricaoObjeto"`
		ValorTotalOrcado float64 `json:"valorTotalOrcado"`
		Situacao         string  `json:"situacaoProcedimentoLicitacao"`
	}

	for _, y := range years {
		dataInicio := fmt.Sprintf("%04d-01-01", y)
		dataFim := fmt.Sprintf("%04d-12-31", y)
		u := fmt.Sprintf("%s/ProcedimentosLicitatoriosApi/LicitacaoPublica/Json/%d/%s/%s", baseURL, idUnidade, dataInicio, dataFim)
		body, status, err := doJSONGet(ctx, c.client, u)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			continue
		}
		var rows []rnLicitacao
		if json.Unmarshal(body, &rows) != nil || len(rows) == 0 {
			continue
		}
		items := make([]LicitacaoItem, 0, minInt(limit, len(rows)))
		for _, r := range rows {
			items = append(items, LicitacaoItem{
				Numero:        strings.TrimSpace(r.NumeroLicitacao),
				Ano:           y,
				Modalidade:    strings.TrimSpace(r.Modalidade),
				Situacao:      strings.TrimSpace(r.Situacao),
				Objeto:        strings.TrimSpace(r.DescricaoObjeto),
				ValorEstimado: r.ValorTotalOrcado,
				Fonte:         "TCE-RN Dados Abertos",
			})
			if len(items) >= limit {
				break
			}
		}
		return &LicitacoesResult{
			Municipio:      m,
			Fonte:          "TCE-RN Dados Abertos",
			Plataforma:     "tce_rn",
			PortalURL:      baseURL,
			TotalRegistros: len(rows),
			Licitacoes:     items,
			Disponivel:     true,
		}, nil
	}

	return &LicitacoesResult{
		Municipio:  m,
		Fonte:      "TCE-RN Dados Abertos",
		Plataforma: "tce_rn",
		PortalURL:  baseURL,
		Licitacoes: []LicitacaoItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma licitação retornada pelo TCE-RN para o período.",
	}, nil
}

// FetchTCERNContratos consulta contratos administrativos no TCE-RN.
func (c *TCEClient) FetchTCERNContratos(ctx context.Context, m *Municipio, limit int) (*ContratosResult, error) {
	idUnidade, err := c.ResolveRNIdUnidade(ctx, m)
	if err != nil {
		return nil, err
	}
	baseURL := c.getRNBaseURL()
	limit = normalizeLimit(limit)

	u := fmt.Sprintf("%s/ContratosApi/Contratos/Json/%d/false", baseURL, idUnidade)
	body, status, err := doJSONGet(ctx, c.client, u)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("TCE-RN Contratos retornou HTTP %d", status)
	}

	type rnContrato struct {
		NumeroContrato      string  `json:"numeroContrato"`
		AnoContrato         int     `json:"anoContrato"`
		ObjetoContrato      string  `json:"objetoContrato"`
		ValorContrato       float64 `json:"valorContrato"`
		NomeContratado      string  `json:"nomeContratado"`
		CPFCNPJContratado   string  `json:"cpfcnpjContratado"`
		DataInicioVigencia  string  `json:"dataInicioVigencia"`
		DataTerminoVigencia string  `json:"dataTerminoVigencia"`
	}

	var rows []rnContrato
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}

	items := make([]ContratoItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		items = append(items, ContratoItem{
			Numero:              strings.TrimSpace(r.NumeroContrato),
			Ano:                 r.AnoContrato,
			Fornecedor:          strings.TrimSpace(r.NomeContratado),
			DocumentoFornecedor: maskIfCPF(r.CPFCNPJContratado),
			Objeto:              strings.TrimSpace(r.ObjetoContrato),
			VigenciaInicio:      strings.TrimSpace(r.DataInicioVigencia),
			VigenciaFim:         strings.TrimSpace(r.DataTerminoVigencia),
			Valor:               r.ValorContrato,
			Fonte:               "TCE-RN Dados Abertos",
		})
		if len(items) >= limit {
			break
		}
	}

	return &ContratosResult{
		Municipio:      m,
		Fonte:          "TCE-RN Dados Abertos",
		Plataforma:     "tce_rn",
		PortalURL:      baseURL,
		TotalRegistros: len(rows),
		Contratos:      items,
		Disponivel:     true,
	}, nil
}

// ---------------------------------------------------------------------------
// TCE-SC (https://servicos.tcesc.tc.br/endpoints-portal-transparencia)
// Apenas dois endpoints abertos sem autenticação:
//   - /municipios.php
//   - /unidades-gestoras.php
// ---------------------------------------------------------------------------

// SCMunicipio representa um município cadastrado no TCE-SC.
type SCMunicipio struct {
	CodigoMunicipio int    `json:"codigo_municipio"`
	NomeMunicipio   string `json:"nome_municipio"`
}

// SCUnidadeGestora representa uma unidade gestora no TCE-SC.
type SCUnidadeGestora struct {
	CodigoUnidade int    `json:"codigo_unidade"`
	NomeUnidade   string `json:"nome_unidade"`
	SiglaUnidade  string `json:"sigla_unidade"`
	NomeMunicipio string `json:"nome_municipio"`
}

// ResolveSCMunicipio resolve o código do município no TCE-SC pelo nome.
func (c *TCEClient) ResolveSCMunicipio(ctx context.Context, m *Municipio) (*SCMunicipio, error) {
	if m == nil {
		return nil, fmt.Errorf("município obrigatório para TCE-SC")
	}
	cfg := c.extraCfg()
	cfg.mu.Lock()
	cached := cfg.scMunis
	baseURL := cfg.scBaseURL
	cfg.mu.Unlock()

	if len(cached) == 0 {
		body, status, err := doJSONGet(ctx, c.client, baseURL+"/municipios.php")
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			return nil, fmt.Errorf("TCE-SC /municipios.php retornou HTTP %d", status)
		}
		if err := json.Unmarshal(body, &cached); err != nil {
			return nil, err
		}
		cfg.mu.Lock()
		cfg.scMunis = cached
		cfg.mu.Unlock()
	}

	targetNorm := RemoveAccents(m.Nome)
	for i := range cached {
		if RemoveAccents(cached[i].NomeMunicipio) == targetNorm {
			cp := cached[i]
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("município %s não encontrado no TCE-SC", m.Nome)
}

// FetchTCESCUnidadesGestoras lista unidades gestoras de um município catarinense no TCE-SC.
func (c *TCEClient) FetchTCESCUnidadesGestoras(ctx context.Context, m *Municipio) ([]SCUnidadeGestora, error) {
	baseURL := c.getSCBaseURL()
	body, status, err := doJSONGet(ctx, c.client, baseURL+"/unidades-gestoras.php")
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("TCE-SC /unidades-gestoras.php retornou HTTP %d", status)
	}
	var all []SCUnidadeGestora
	if err := json.Unmarshal(body, &all); err != nil {
		return nil, err
	}
	if m == nil || strings.TrimSpace(m.Nome) == "" {
		return all, nil
	}
	targetNorm := RemoveAccents(m.Nome)
	var out []SCUnidadeGestora
	for _, ug := range all {
		if RemoveAccents(ug.NomeMunicipio) == targetNorm {
			out = append(out, ug)
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// TCE-TO (https://api.tceto.tc.br/econtas/api)
// Requer header Accept: application/json (já enviado por doJSONGet).
// Endpoints: /pessoas, /pautas, /processo/{numero}/{ano}
// ---------------------------------------------------------------------------

// TCETOProcessoResumo representa um resumo de processo vinculado a uma pessoa no TCE-TO.
type TCETOProcessoResumo struct {
	NumeroAno               string `json:"numero_ano"`
	Assunto                 string `json:"assunto"`
	ClasseAssunto           string `json:"classe_assunto"`
	EntidadeOrigem          string `json:"entidade_origem"`
	EntidadeOrigemMunicipio string `json:"entidade_origem_municipio"`
	DataEntrada             string `json:"data_entrada"`
	DepartamentoAtual       string `json:"departamento_atual"`
}

// TCETOPessoa representa uma pessoa com processos no TCE-TO.
type TCETOPessoa struct {
	ID        int                   `json:"id"`
	Nome      string                `json:"nome"`
	Codigo    string                `json:"codigo"`
	Processos []TCETOProcessoResumo `json:"processos"`
}

// TCETOPauta representa uma pauta de sessão do TCE-TO.
type TCETOPauta struct {
	Data   string `json:"data"`
	Hora   string `json:"hora"`
	Tipo   string `json:"tipo"`
	Origem string `json:"origem"`
	URL    string `json:"url"`
}

// TCETOProcesso representa um processo detalhado no TCE-TO.
type TCETOProcesso struct {
	NumeroAno               string `json:"numero_ano"`
	Assunto                 string `json:"assunto"`
	ClasseAssunto           string `json:"classe_assunto"`
	EntidadeOrigem          string `json:"entidade_origem"`
	EntidadeOrigemMunicipio string `json:"entidade_origem_municipio"`
	EntidadeOrigemCNPJ      string `json:"entidade_origem_cnpj"`
	DataEntrada             string `json:"data_entrada"`
	DepartamentoAtual       string `json:"departamento_atual"`
	ErrorMessage            string `json:"error_message,omitempty"`
}

// FetchTCETOPessoas busca pessoas com processos no TCE-TO e-Contas.
func (c *TCEClient) FetchTCETOPessoas(ctx context.Context, nome string, limit int) ([]TCETOPessoa, error) {
	baseURL := c.getTOBaseURL()
	limit = normalizeLimit(limit)
	if limit > 50 {
		limit = 50
	}
	params := url.Values{
		"pagina":  {"1"},
		"tamanho": {fmt.Sprintf("%d", limit)},
	}
	if strings.TrimSpace(nome) != "" {
		params.Set("nome", strings.TrimSpace(nome))
	}
	u := baseURL + "/pessoas?" + params.Encode()
	body, status, err := doJSONGet(ctx, c.client, u)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("TCE-TO /pessoas retornou HTTP %d", status)
	}
	var out []TCETOPessoa
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// FetchTCETOPautas lista pautas de sessão do TCE-TO e-Contas.
func (c *TCEClient) FetchTCETOPautas(ctx context.Context, limit int) ([]TCETOPauta, error) {
	baseURL := c.getTOBaseURL()
	limit = normalizeLimit(limit)
	if limit > 50 {
		limit = 50
	}
	u := fmt.Sprintf("%s/pautas?ordem=DESC&tamanho=%d", baseURL, limit)
	body, status, err := doJSONGet(ctx, c.client, u)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("TCE-TO /pautas retornou HTTP %d", status)
	}
	var wrap struct {
		Pautas []TCETOPauta `json:"pautas"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		return nil, err
	}
	return wrap.Pautas, nil
}

// FetchTCETOProcesso consulta detalhes de um processo específico no TCE-TO e-Contas.
func (c *TCEClient) FetchTCETOProcesso(ctx context.Context, numero, ano int) (*TCETOProcesso, error) {
	baseURL := c.getTOBaseURL()
	u := fmt.Sprintf("%s/processo/%d/%d", baseURL, numero, ano)
	body, status, err := doJSONGet(ctx, c.client, u)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("TCE-TO /processo retornou HTTP %d", status)
	}
	var proc TCETOProcesso
	if err := json.Unmarshal(body, &proc); err != nil {
		return nil, err
	}
	if proc.ErrorMessage != "" {
		return nil, fmt.Errorf("TCE-TO /processo: %s", proc.ErrorMessage)
	}
	return &proc, nil
}

// ---------------------------------------------------------------------------
// TCE-PA (https://sistemas.tcepa.tc.br/dadosabertos/api)
// Endpoint JSON: /v1/diario_oficial?ano=...&mes=...&tipo_ato=...
// ---------------------------------------------------------------------------

// TCEPADiarioItem representa uma publicação no Diário Oficial do TCE-PA.
type TCEPADiarioItem struct {
	NumeroPublicacao int    `json:"NumeroPublicacao"`
	DataPublicacao   string `json:"DataPublicacao"`
	TipoAto          string `json:"TipoAto"`
	Publicacao       string `json:"Publicacao"`
}

// FetchTCEPADiarioOficial consulta publicações do Diário Oficial do TCE-PA.
func (c *TCEClient) FetchTCEPADiarioOficial(ctx context.Context, ano, mes int, tipoAto string, limit int) ([]TCEPADiarioItem, error) {
	baseURL := c.getPABaseURL()
	limit = normalizeLimit(limit)
	if ano <= 0 {
		ano = time.Now().Year()
	}

	params := url.Values{"ano": {fmt.Sprintf("%d", ano)}}
	if mes > 0 && mes <= 12 {
		params.Set("mes", fmt.Sprintf("%d", mes))
	}
	if strings.TrimSpace(tipoAto) != "" {
		params.Set("tipo_ato", strings.TrimSpace(tipoAto))
	}

	u := baseURL + "/v1/diario_oficial?" + params.Encode()
	body, status, err := doJSONGet(ctx, c.client, u)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("TCE-PA /v1/diario_oficial retornou HTTP %d", status)
	}

	var items []TCEPADiarioItem
	var wrap struct {
		Data []TCEPADiarioItem `json:"data"`
	}
	if err := json.Unmarshal(body, &wrap); err == nil && wrap.Data != nil {
		items = wrap.Data
	} else if err := json.Unmarshal(body, &items); err != nil {
		return nil, err
	}

	if len(items) > limit {
		items = items[:limit]
	}
	for i := range items {
		items[i].Publicacao = stripHTMLTags(items[i].Publicacao)
	}
	return items, nil
}

// ---------------------------------------------------------------------------
// TCE-RS (https://dados.tce.rs.gov.br)
// Arquivos JSON diretos (/dados/...) e CKAN (/api/3/action/package_search)
// ---------------------------------------------------------------------------

// RSMunicipio representa um município no arquivo auxiliar do TCE-RS.
type RSMunicipio struct {
	Codigo     int    `json:"COD_MUNICIPIO"`
	Nome       string `json:"NOME_MUNICIPIO"`
	UF         string `json:"UF"`
	CodigoIBGE int    `json:"CD_MUNICIPIO_IBGE"`
}

// TCERSDataset representa um dataset retornado pela API CKAN do TCE-RS.
type TCERSDataset struct {
	Name         string `json:"name"`
	Title        string `json:"title"`
	Grupo        string `json:"grupo,omitempty"`
	Notas        string `json:"notas,omitempty"`
	URL          string `json:"url"`
	NumResources int    `json:"num_resources"`
}

// extractTCERSItems extrai o slice de objetos JSON de uma resposta do TCE-RS
// (que pode ser um array direto ou um objeto com chave raiz contendo o array).
func extractTCERSItems(body []byte) ([]map[string]any, error) {
	var arr []map[string]any
	if err := json.Unmarshal(body, &arr); err == nil {
		return arr, nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, err
	}
	for _, raw := range obj {
		var list []map[string]any
		if json.Unmarshal(raw, &list) == nil {
			return list, nil
		}
	}
	return nil, fmt.Errorf("TCE-RS JSON inesperado: nenhuma lista encontrada")
}

// ResolveRSMunicipio resolve o código do município no TCE-RS pelo código IBGE ou nome.
func (c *TCEClient) ResolveRSMunicipio(ctx context.Context, m *Municipio) (*RSMunicipio, error) {
	if m == nil {
		return nil, fmt.Errorf("município obrigatório para TCE-RS")
	}
	cfg := c.extraCfg()
	cfg.mu.Lock()
	cached := cfg.rsMunis
	baseURL := cfg.rsBaseURL
	cfg.mu.Unlock()

	if len(cached) == 0 {
		body, status, err := doJSONGet(ctx, c.client, baseURL+"/dados/auxiliar/municipios.json")
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			return nil, fmt.Errorf("TCE-RS municipios.json retornou HTTP %d", status)
		}
		rows, err := extractTCERSItems(body)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			b, _ := json.Marshal(r)
			var rm RSMunicipio
			if json.Unmarshal(b, &rm) == nil {
				cached = append(cached, rm)
			}
		}
		cfg.mu.Lock()
		cfg.rsMunis = cached
		cfg.mu.Unlock()
	}

	targetIBGE := strings.TrimSpace(m.CodigoIBGE)
	for i := range cached {
		if targetIBGE != "" && fmt.Sprintf("%d", cached[i].CodigoIBGE) == targetIBGE {
			cp := cached[i]
			return &cp, nil
		}
	}
	targetNorm := RemoveAccents(m.Nome)
	for i := range cached {
		if RemoveAccents(cached[i].Nome) == targetNorm {
			cp := cached[i]
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("município %s (%s) não encontrado no TCE-RS", m.Nome, m.CodigoIBGE)
}

// FetchTCERSDespesas consulta indicadores de despesa municipal (LRF: pessoal, MDE, ASPS) no TCE-RS.
func (c *TCEClient) FetchTCERSDespesas(ctx context.Context, m *Municipio, ano, limit int) (*DespesasResult, error) {
	baseURL := c.getRSBaseURL()
	limit = normalizeLimit(limit)
	years := []int{ano}
	if ano <= 0 {
		nowY := time.Now().Year()
		years = []int{nowY - 1, nowY - 2, nowY}
	}

	targetNorm := ""
	if m != nil && strings.TrimSpace(m.Nome) != "" {
		targetNorm = RemoveAccents(m.Nome)
	}

	for _, y := range years {
		u := fmt.Sprintf("%s/dados/municipal/gastos-lrf-mde-asps/%d.json", baseURL, y)
		body, status, err := doJSONGet(ctx, c.client, u)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			continue
		}
		rows, err := extractTCERSItems(body)
		if err != nil || len(rows) == 0 {
			continue
		}

		items := make([]DespesaItem, 0, minInt(limit, len(rows)*3))
		for _, r := range rows {
			orgao := mapGetStr(r, "NM_Orgao")
			if targetNorm != "" && !strings.Contains(RemoveAccents(orgao), targetNorm) {
				continue
			}
			if v := mapGetFloat(r, "VL_DespesaPessoal"); v > 0 {
				items = append(items, DespesaItem{
					Ano:       y,
					Orgao:     orgao,
					Funcao:    "Despesa com Pessoal (LRF)",
					ValorPago: v,
					Fonte:     "TCE-RS Dados Abertos",
				})
			}
			if v := mapGetFloat(r, "VL_DespesaMDE"); v > 0 {
				items = append(items, DespesaItem{
					Ano:       y,
					Orgao:     orgao,
					Funcao:    "Educação (MDE)",
					ValorPago: v,
					Fonte:     "TCE-RS Dados Abertos",
				})
			}
			if v := mapGetFloat(r, "VL_DespesaASPS"); v > 0 {
				items = append(items, DespesaItem{
					Ano:       y,
					Orgao:     orgao,
					Funcao:    "Saúde (ASPS)",
					ValorPago: v,
					Fonte:     "TCE-RS Dados Abertos",
				})
			}
			if len(items) >= limit {
				items = items[:limit]
				break
			}
		}
		if len(items) > 0 {
			return &DespesasResult{
				Municipio:      m,
				Fonte:          "TCE-RS Dados Abertos",
				Plataforma:     "tce_rs",
				PortalURL:      baseURL,
				TotalRegistros: len(items),
				Despesas:       items,
				Disponivel:     true,
			}, nil
		}
	}

	return &DespesasResult{
		Municipio:  m,
		Fonte:      "TCE-RS Dados Abertos",
		Plataforma: "tce_rs",
		PortalURL:  baseURL,
		Despesas:   []DespesaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma despesa retornada pelo TCE-RS para o período.",
	}, nil
}

// FetchTCERSReceitas consulta indicadores de receita municipal (RCL, MDE, ASPS) no TCE-RS.
func (c *TCEClient) FetchTCERSReceitas(ctx context.Context, m *Municipio, ano, limit int) (*ReceitasResult, error) {
	baseURL := c.getRSBaseURL()
	limit = normalizeLimit(limit)
	years := []int{ano}
	if ano <= 0 {
		nowY := time.Now().Year()
		years = []int{nowY - 1, nowY - 2, nowY}
	}

	targetNorm := ""
	if m != nil && strings.TrimSpace(m.Nome) != "" {
		targetNorm = RemoveAccents(m.Nome)
	}

	for _, y := range years {
		u := fmt.Sprintf("%s/dados/municipal/gastos-lrf-mde-asps/%d.json", baseURL, y)
		body, status, err := doJSONGet(ctx, c.client, u)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			continue
		}
		rows, err := extractTCERSItems(body)
		if err != nil || len(rows) == 0 {
			continue
		}

		items := make([]ReceitaItem, 0, minInt(limit, len(rows)*3))
		for _, r := range rows {
			orgao := mapGetStr(r, "NM_Orgao")
			if targetNorm != "" && !strings.Contains(RemoveAccents(orgao), targetNorm) {
				continue
			}
			dataStr := fmt.Sprintf("%d", y)
			if v := mapGetFloat(r, "VL_ReceitaCorrenteLiquida"); v > 0 {
				items = append(items, ReceitaItem{
					Descricao:       "Receita Corrente Líquida (RCL)",
					Categoria:       orgao,
					Data:            dataStr,
					ValorArrecadado: v,
					Fonte:           "TCE-RS Dados Abertos",
				})
			}
			if v := mapGetFloat(r, "VL_ReceitaMDE"); v > 0 {
				items = append(items, ReceitaItem{
					Descricao:       "Receita Base Educação (MDE)",
					Categoria:       orgao,
					Data:            dataStr,
					ValorArrecadado: v,
					Fonte:           "TCE-RS Dados Abertos",
				})
			}
			if v := mapGetFloat(r, "VL_ReceitaASPS"); v > 0 {
				items = append(items, ReceitaItem{
					Descricao:       "Receita Base Saúde (ASPS)",
					Categoria:       orgao,
					Data:            dataStr,
					ValorArrecadado: v,
					Fonte:           "TCE-RS Dados Abertos",
				})
			}
			if len(items) >= limit {
				items = items[:limit]
				break
			}
		}
		if len(items) > 0 {
			return &ReceitasResult{
				Municipio:      m,
				Fonte:          "TCE-RS Dados Abertos",
				Plataforma:     "tce_rs",
				PortalURL:      baseURL,
				TotalRegistros: len(items),
				Receitas:       items,
				Disponivel:     true,
			}, nil
		}
	}

	return &ReceitasResult{
		Municipio:  m,
		Fonte:      "TCE-RS Dados Abertos",
		Plataforma: "tce_rs",
		PortalURL:  baseURL,
		Receitas:   []ReceitaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma receita retornada pelo TCE-RS para o período.",
	}, nil
}

// FetchTCERSDatasets pesquisa datasets no catálogo CKAN do TCE-RS.
func (c *TCEClient) FetchTCERSDatasets(ctx context.Context, query string, limit int) ([]TCERSDataset, error) {
	baseURL := c.getRSBaseURL()
	limit = normalizeLimit(limit)
	if limit > 50 {
		limit = 50
	}
	params := url.Values{
		"q":    {strings.TrimSpace(query)},
		"rows": {fmt.Sprintf("%d", limit)},
	}
	u := baseURL + "/api/3/action/package_search?" + params.Encode()
	body, status, err := doJSONGet(ctx, c.client, u)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("TCE-RS package_search retornou HTTP %d", status)
	}

	var resp struct {
		Result struct {
			Results []struct {
				Name         string `json:"name"`
				Title        string `json:"title"`
				Notes        string `json:"notes"`
				NumResources int    `json:"num_resources"`
				Groups       []struct {
					Title string `json:"title"`
				} `json:"groups"`
			} `json:"results"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	out := make([]TCERSDataset, 0, len(resp.Result.Results))
	for _, pkg := range resp.Result.Results {
		grupo := ""
		if len(pkg.Groups) > 0 {
			grupo = pkg.Groups[0].Title
		}
		out = append(out, TCERSDataset{
			Name:         pkg.Name,
			Title:        pkg.Title,
			Grupo:        grupo,
			Notas:        pkg.Notes,
			URL:          baseURL + "/dataset/" + pkg.Name,
			NumResources: pkg.NumResources,
		})
	}
	return out, nil
}
