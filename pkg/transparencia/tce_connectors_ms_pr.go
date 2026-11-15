package transparencia

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultTCEPRBaseURL = "https://siope.tce.pr.gov.br"
	defaultTCEMSBaseURL = "https://sip-barramento.tce.ms.gov.br/barramento"
	defaultTCESEBaseURL = "https://www.tce.se.gov.br/visualizadorRelatorios"
)

type tceMSPRConfig struct {
	mu        sync.Mutex
	prBaseURL string
	msBaseURL string
	seBaseURL string
}

var (
	msPRCfgMu sync.RWMutex
	msPRCfgs  = make(map[*TCEClient]*tceMSPRConfig)
)

func (c *TCEClient) msPRCfg() *tceMSPRConfig {
	msPRCfgMu.RLock()
	cfg := msPRCfgs[c]
	msPRCfgMu.RUnlock()
	if cfg != nil {
		return cfg
	}
	msPRCfgMu.Lock()
	defer msPRCfgMu.Unlock()
	if cfg = msPRCfgs[c]; cfg == nil {
		cfg = &tceMSPRConfig{
			prBaseURL: defaultTCEPRBaseURL,
			msBaseURL: defaultTCEMSBaseURL,
			seBaseURL: defaultTCESEBaseURL,
		}
		msPRCfgs[c] = cfg
	}
	return cfg
}

// SetTCEPRBaseURL configura a URL base da API SIOPE do TCE-PR (útil para testes).
func (c *TCEClient) SetTCEPRBaseURL(u string) {
	cfg := c.msPRCfg()
	cfg.mu.Lock()
	cfg.prBaseURL = strings.TrimRight(u, "/")
	cfg.mu.Unlock()
}

// SetTCEMSBaseURL configura a URL base do Barramento de Dados Abertos do TCE-MS (útil para testes).
func (c *TCEClient) SetTCEMSBaseURL(u string) {
	cfg := c.msPRCfg()
	cfg.mu.Lock()
	cfg.msBaseURL = strings.TrimRight(u, "/")
	cfg.mu.Unlock()
}

// SetTCESEBaseURL configura a URL base da API de Dados Abertos do TCE-SE (útil para testes).
func (c *TCEClient) SetTCESEBaseURL(u string) {
	cfg := c.msPRCfg()
	cfg.mu.Lock()
	cfg.seBaseURL = strings.TrimRight(u, "/")
	cfg.mu.Unlock()
}

func (c *TCEClient) getTCEPRBaseURL() string {
	cfg := c.msPRCfg()
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	return cfg.prBaseURL
}

func (c *TCEClient) getTCEMSBaseURL() string {
	cfg := c.msPRCfg()
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	return cfg.msBaseURL
}

func (c *TCEClient) getTCESEBaseURL() string {
	cfg := c.msPRCfg()
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	return cfg.seBaseURL
}

// ============================================================================
// TCE-PR (Paraná - 399 municípios) — API SIOPE
// Swagger: https://siope.tce.pr.gov.br/swagger/SIOPE/swagger.json
// ============================================================================

type tcePRDespesaResp struct {
	DatConsulta string `json:"dat_consulta"`
	Ente        []struct {
		SglUF        string `json:"sgl_uf"`
		CodMunicipio int    `json:"cod_municipio"`
		NumAno       int    `json:"num_ano"`
		NumPeriodo   int    `json:"num_periodo"`
		Despesa      []struct {
			CodFonte            int     `json:"cod_fonte"`
			DesElementoDespesa  string  `json:"des_elemento_despesa"`
			VlrDespesaEmpenhada float64 `json:"vlr_despesa_empenhada"`
			VlrDespesaLiquidada float64 `json:"vlr_despesa_liquidada"`
			VlrDespesaPaga      float64 `json:"vlr_despesa_paga"`
		} `json:"despesa"`
	} `json:"ente"`
}

type tcePRReceitaResp struct {
	DatConsulta string `json:"dat_consulta"`
	Ente        []struct {
		SglUF        string `json:"sgl_uf"`
		CodMunicipio int    `json:"cod_municipio"`
		NumAno       int    `json:"num_ano"`
		NumPeriodo   int    `json:"num_periodo"`
		Receita      []struct {
			CodConta            string  `json:"cod_conta"`
			VlrReceitaRealizada float64 `json:"vlr_receita_realizada"`
		} `json:"receita"`
	} `json:"ente"`
}

type tcePRFuncao12Resp struct {
	DatConsulta string `json:"dat_consulta"`
	Ente        []struct {
		SglUF        string `json:"sgl_uf"`
		CodMunicipio int    `json:"cod_municipio"`
		NumAno       int    `json:"num_ano"`
		NumPeriodo   int    `json:"num_periodo"`
		Valores      []struct {
			CodSubfuncao        int     `json:"cod_subfuncao"`
			VlrDespesaEmpenhada float64 `json:"vlr_despesa_empenhada"`
			VlrDespesaLiquidada float64 `json:"vlr_despesa_liquidada"`
			VlrDespesaPaga      float64 `json:"vlr_despesa_paga"`
		} `json:"valores"`
	} `json:"ente"`
}

// FetchTCEPRDespesas consulta despesas municipais do Paraná via API SIOPE do TCE-PR (/consulta/despesa).
// Cobre os 399 municípios do Paraná usando o código IBGE em cod_municipio.
func (c *TCEClient) FetchTCEPRDespesas(ctx context.Context, m *Municipio, ano, periodo, limit int) (*DespesasResult, error) {
	if m == nil || strings.TrimSpace(m.CodigoIBGE) == "" {
		return nil, fmt.Errorf("município com código IBGE é obrigatório para TCE-PR")
	}
	baseURL := c.getTCEPRBaseURL()
	limit = normalizeLimit(limit)
	years := []int{ano}
	if ano <= 0 {
		nowY := time.Now().Year()
		years = []int{nowY - 1, nowY - 2, nowY}
	}

	for _, y := range years {
		q := url.Values{}
		q.Set("sgl_uf", "PR")
		q.Set("cod_municipio", strings.TrimSpace(m.CodigoIBGE))
		q.Set("num_ano", strconv.Itoa(y))
		if periodo > 0 {
			q.Set("num_periodo", strconv.Itoa(periodo))
		}
		u := fmt.Sprintf("%s/consulta/despesa?%s", baseURL, q.Encode())
		body, status, err := doJSONGet(ctx, c.client, u)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			continue
		}
		var resp tcePRDespesaResp
		if err := json.Unmarshal(body, &resp); err != nil {
			continue
		}
		var items []DespesaItem
		total := 0
		for _, ente := range resp.Ente {
			total += len(ente.Despesa)
			for _, d := range ente.Despesa {
				if len(items) >= limit {
					continue
				}
				unidade := ""
				if d.CodFonte > 0 {
					unidade = fmt.Sprintf("Fonte %d", d.CodFonte)
				}
				dataRef := ""
				if ente.NumPeriodo > 0 {
					dataRef = fmt.Sprintf("%d/P%d", ente.NumAno, ente.NumPeriodo)
				}
				items = append(items, DespesaItem{
					Ano:            ente.NumAno,
					Data:           dataRef,
					Unidade:        unidade,
					Elemento:       strings.TrimSpace(d.DesElementoDespesa),
					ValorEmpenhado: d.VlrDespesaEmpenhada,
					ValorLiquidado: d.VlrDespesaLiquidada,
					ValorPago:      d.VlrDespesaPaga,
					Fonte:          "TCE-PR SIOPE",
				})
			}
		}
		if total > 0 {
			return &DespesasResult{
				Municipio:      m,
				Fonte:          "TCE-PR SIOPE",
				Plataforma:     "tce_pr_siope",
				PortalURL:      baseURL,
				TotalRegistros: total,
				Despesas:       items,
				Disponivel:     true,
			}, nil
		}
	}

	return &DespesasResult{
		Municipio:  m,
		Fonte:      "TCE-PR SIOPE",
		Plataforma: "tce_pr_siope",
		PortalURL:  baseURL,
		Despesas:   []DespesaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma despesa retornada pelo TCE-PR SIOPE para o período.",
	}, nil
}

// FetchTCEPRReceitas consulta receitas municipais do Paraná via API SIOPE do TCE-PR (/consulta/receita).
// Cobre os 399 municípios do Paraná usando o código IBGE em cod_municipio.
func (c *TCEClient) FetchTCEPRReceitas(ctx context.Context, m *Municipio, ano, periodo, limit int) (*ReceitasResult, error) {
	if m == nil || strings.TrimSpace(m.CodigoIBGE) == "" {
		return nil, fmt.Errorf("município com código IBGE é obrigatório para TCE-PR")
	}
	baseURL := c.getTCEPRBaseURL()
	limit = normalizeLimit(limit)
	years := []int{ano}
	if ano <= 0 {
		nowY := time.Now().Year()
		years = []int{nowY - 1, nowY - 2, nowY}
	}

	for _, y := range years {
		q := url.Values{}
		q.Set("sgl_uf", "PR")
		q.Set("cod_municipio", strings.TrimSpace(m.CodigoIBGE))
		q.Set("num_ano", strconv.Itoa(y))
		if periodo > 0 {
			q.Set("num_periodo", strconv.Itoa(periodo))
		}
		u := fmt.Sprintf("%s/consulta/receita?%s", baseURL, q.Encode())
		body, status, err := doJSONGet(ctx, c.client, u)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			continue
		}
		var resp tcePRReceitaResp
		if err := json.Unmarshal(body, &resp); err != nil {
			continue
		}
		var items []ReceitaItem
		total := 0
		for _, ente := range resp.Ente {
			total += len(ente.Receita)
			for _, r := range ente.Receita {
				if len(items) >= limit {
					continue
				}
				cod := strings.TrimSpace(r.CodConta)
				desc := "Receita SIOPE"
				if cod != "" {
					desc = "Conta " + cod
				}
				dataRef := ""
				if ente.NumPeriodo > 0 {
					dataRef = fmt.Sprintf("%d/P%d", ente.NumAno, ente.NumPeriodo)
				}
				items = append(items, ReceitaItem{
					Codigo:          cod,
					Descricao:       desc,
					Data:            dataRef,
					ValorArrecadado: r.VlrReceitaRealizada,
					Fonte:           "TCE-PR SIOPE",
				})
			}
		}
		if total > 0 {
			return &ReceitasResult{
				Municipio:      m,
				Fonte:          "TCE-PR SIOPE",
				Plataforma:     "tce_pr_siope",
				PortalURL:      baseURL,
				TotalRegistros: total,
				Receitas:       items,
				Disponivel:     true,
			}, nil
		}
	}

	return &ReceitasResult{
		Municipio:  m,
		Fonte:      "TCE-PR SIOPE",
		Plataforma: "tce_pr_siope",
		PortalURL:  baseURL,
		Receitas:   []ReceitaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma receita retornada pelo TCE-PR SIOPE para o período.",
	}, nil
}

// FetchTCEPRFuncao12Despesas consulta despesas da Função 12 (Educação) por subfunção via TCE-PR SIOPE (/consulta/funcao12).
func (c *TCEClient) FetchTCEPRFuncao12Despesas(ctx context.Context, m *Municipio, ano, periodo, limit int) (*DespesasResult, error) {
	if m == nil || strings.TrimSpace(m.CodigoIBGE) == "" {
		return nil, fmt.Errorf("município com código IBGE é obrigatório para TCE-PR")
	}
	baseURL := c.getTCEPRBaseURL()
	limit = normalizeLimit(limit)
	years := []int{ano}
	if ano <= 0 {
		nowY := time.Now().Year()
		years = []int{nowY - 1, nowY - 2, nowY}
	}

	for _, y := range years {
		q := url.Values{}
		q.Set("sgl_uf", "PR")
		q.Set("cod_municipio", strings.TrimSpace(m.CodigoIBGE))
		q.Set("num_ano", strconv.Itoa(y))
		if periodo > 0 {
			q.Set("num_periodo", strconv.Itoa(periodo))
		}
		u := fmt.Sprintf("%s/consulta/funcao12?%s", baseURL, q.Encode())
		body, status, err := doJSONGet(ctx, c.client, u)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			continue
		}
		var resp tcePRFuncao12Resp
		if err := json.Unmarshal(body, &resp); err != nil {
			continue
		}
		var items []DespesaItem
		total := 0
		for _, ente := range resp.Ente {
			total += len(ente.Valores)
			for _, v := range ente.Valores {
				if len(items) >= limit {
					continue
				}
				dataRef := ""
				if ente.NumPeriodo > 0 {
					dataRef = fmt.Sprintf("%d/P%d", ente.NumAno, ente.NumPeriodo)
				}
				items = append(items, DespesaItem{
					Ano:            ente.NumAno,
					Data:           dataRef,
					Funcao:         fmt.Sprintf("Educação (Subfunção %d)", v.CodSubfuncao),
					ValorEmpenhado: v.VlrDespesaEmpenhada,
					ValorLiquidado: v.VlrDespesaLiquidada,
					ValorPago:      v.VlrDespesaPaga,
					Fonte:          "TCE-PR SIOPE (Função 12)",
				})
			}
		}
		if total > 0 {
			return &DespesasResult{
				Municipio:      m,
				Fonte:          "TCE-PR SIOPE (Função 12)",
				Plataforma:     "tce_pr_siope",
				PortalURL:      baseURL,
				TotalRegistros: total,
				Despesas:       items,
				Disponivel:     true,
			}, nil
		}
	}

	return &DespesasResult{
		Municipio:  m,
		Fonte:      "TCE-PR SIOPE (Função 12)",
		Plataforma: "tce_pr_siope",
		PortalURL:  baseURL,
		Despesas:   []DespesaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma despesa da Função 12 retornada pelo TCE-PR SIOPE para o período.",
	}, nil
}

// ============================================================================
// TCE-MS (Mato Grosso do Sul) — Barramento Dados Abertos
// Swagger: https://sip-barramento.tce.ms.gov.br/barramento/swagger/dados-abertos/swagger.json
// Nota: Esta API expõe exclusivamente dados institucionais do próprio TCE-MS
// (Contracheque Único de membros e Repasses/Duodécimos de UGs 30101 e 30901),
// não expondo despesas/receitas/licitações dos 79 municípios sul-mato-grossenses.
// ============================================================================

type tceMSTRUMembro struct {
	RemuneracaoID         int64   `json:"remuneracaoId"`
	Matricula             int     `json:"matricula"`
	Nome                  string  `json:"nome"`
	CPF                   string  `json:"cpf"`
	CargoDescr            string  `json:"cargoDescr"`
	LotacaoDescr          string  `json:"lotacaoDescr"`
	MesAno                string  `json:"mesAno"`
	DataAdmissao          string  `json:"dataAdmissao"`
	TRU1001               float64 `json:"tru1001"`
	TRU1009               float64 `json:"tru1009"`
	TRU1010               float64 `json:"tru1010"`
	TRU1014               float64 `json:"tru1014"`
	TRU2001               float64 `json:"tru2001"`
	TRU2007               float64 `json:"tru2007"`
	TRU2010               float64 `json:"tru2010"`
	TRU8002               float64 `json:"tru8002"`
	TRU9002               float64 `json:"tru9002"`
	TRU9003               float64 `json:"tru9003"`
	MargemConsignavel     float64 `json:"margemConsignavel"`
	BaseCalculoRppsMsprev float64 `json:"baseCalculoRppsMsprev"`
	BaseCalculoIrrf       float64 `json:"baseCalculoIrrf"`
	RemuneracaoOutroOrgao float64 `json:"remuneracaoOutroOrgao"`
	BaseCalculoPrevcom    float64 `json:"baseCalculoPrevcom"`
}

type tceMSDuodecimosResp struct {
	Itens []struct {
		UnidadeGestora string  `json:"unidadeGestora"`
		Ano            int     `json:"ano"`
		Mes            int     `json:"mes"`
		Previsto       float64 `json:"previsto"`
		Recebido       float64 `json:"recebido"`
	} `json:"itens"`
	Total struct {
		Previsto float64 `json:"previsto"`
		Recebido float64 `json:"recebido"`
	} `json:"total"`
}

// FetchTCEMSServidores consulta membros e remunerações (Contracheque Único) publicados no Barramento TCE-MS.
func (c *TCEClient) FetchTCEMSServidores(ctx context.Context, m *Municipio, ano, mes int, filtro string, limit int) (*ServidoresResult, error) {
	baseURL := c.getTCEMSBaseURL()
	limit = normalizeLimit(limit)
	years := []int{ano}
	if ano <= 0 {
		nowY := time.Now().Year()
		years = []int{nowY, nowY - 1, nowY - 2}
	}

	for _, y := range years {
		q := url.Values{}
		q.Set("Ano", strconv.Itoa(y))
		if mes > 0 && mes <= 12 {
			q.Set("Mes", strconv.Itoa(mes))
		}
		if strings.TrimSpace(filtro) != "" {
			q.Set("Nome", strings.TrimSpace(filtro))
		}
		u := fmt.Sprintf("%s/api/Transparencia/GestaoPessoas/TRUMembros?%s", baseURL, q.Encode())
		body, status, err := doJSONGet(ctx, c.client, u)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			continue
		}
		var rows []tceMSTRUMembro
		if err := json.Unmarshal(body, &rows); err != nil || len(rows) == 0 {
			continue
		}
		items := make([]ServidorItem, 0, minInt(limit, len(rows)))
		for _, r := range rows {
			bruto := r.TRU1001 + r.TRU1009 + r.TRU1010 + r.TRU1014
			desc := r.TRU2001 + r.TRU2007 + r.TRU2010
			liq := r.TRU9003
			if liq <= 0 && bruto > 0 {
				liq = bruto - desc
				if liq < 0 {
					liq = 0
				}
			}
			items = append(items, ServidorItem{
				Nome:           strings.TrimSpace(r.Nome),
				Matricula:      strconv.Itoa(r.Matricula),
				CPFParcial:     strings.TrimSpace(r.CPF),
				Cargo:          strings.TrimSpace(r.CargoDescr),
				Lotacao:        strings.TrimSpace(r.LotacaoDescr),
				Competencia:    strings.TrimSpace(r.MesAno),
				DataAdmissao:   strings.TrimSpace(r.DataAdmissao),
				SalarioBruto:   bruto,
				Descontos:      desc,
				SalarioLiquido: liq,
				Orgao:          "TCE-MS",
				Fonte:          "TCE-MS Dados Abertos (Contracheque Único)",
				PortalURL:      baseURL,
			})
			if len(items) >= limit {
				break
			}
		}
		return &ServidoresResult{
			Municipio:      m,
			Fonte:          "TCE-MS Dados Abertos (Contracheque Único)",
			Plataforma:     "tce_ms_barramento",
			PortalURL:      baseURL,
			TotalRegistros: len(rows),
			Servidores:     items,
			Disponivel:     true,
		}, nil
	}

	return &ServidoresResult{
		Municipio:  m,
		Fonte:      "TCE-MS Dados Abertos (Contracheque Único)",
		Plataforma: "tce_ms_barramento",
		PortalURL:  baseURL,
		Servidores: []ServidorItem{},
		Disponivel: true,
		Mensagem:   "Nenhum registro retornado pelo Barramento TCE-MS para o período.",
	}, nil
}

// FetchTCEMSDuodecimos consulta repasses de duodécimos do TCE-MS (/api/Transparencia/Repasse/Duodecimos).
// Quando ugID <= 0, consulta a UG padrão 30101 (Tribunal de Contas de MS).
func (c *TCEClient) FetchTCEMSDuodecimos(ctx context.Context, m *Municipio, ugID, ano, mes, limit int) (*ReceitasResult, error) {
	baseURL := c.getTCEMSBaseURL()
	limit = normalizeLimit(limit)
	if ugID <= 0 {
		ugID = 30101
	}
	years := []int{ano}
	if ano <= 0 {
		nowY := time.Now().Year()
		years = []int{nowY - 1, nowY - 2, nowY}
	}

	for _, y := range years {
		q := url.Values{}
		q.Set("UnidadeGestoraId", strconv.Itoa(ugID))
		q.Set("Ano", strconv.Itoa(y))
		if mes > 0 && mes <= 12 {
			q.Set("Mes", strconv.Itoa(mes))
		}
		u := fmt.Sprintf("%s/api/Transparencia/Repasse/Duodecimos?%s", baseURL, q.Encode())
		body, status, err := doJSONGet(ctx, c.client, u)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			continue
		}
		var resp tceMSDuodecimosResp
		if err := json.Unmarshal(body, &resp); err != nil || len(resp.Itens) == 0 {
			continue
		}
		items := make([]ReceitaItem, 0, minInt(limit, len(resp.Itens)))
		for _, it := range resp.Itens {
			items = append(items, ReceitaItem{
				Codigo:          strconv.Itoa(ugID),
				Descricao:       strings.TrimSpace(it.UnidadeGestora),
				Categoria:       "Duodécimo",
				Data:            fmt.Sprintf("%02d/%04d", it.Mes, it.Ano),
				ValorPrevisto:   it.Previsto,
				ValorArrecadado: it.Recebido,
				Fonte:           "TCE-MS Dados Abertos (Repasse Duodécimos)",
			})
			if len(items) >= limit {
				break
			}
		}
		return &ReceitasResult{
			Municipio:      m,
			Fonte:          "TCE-MS Dados Abertos (Repasse Duodécimos)",
			Plataforma:     "tce_ms_barramento",
			PortalURL:      baseURL,
			TotalRegistros: len(resp.Itens),
			Receitas:       items,
			Disponivel:     true,
		}, nil
	}

	return &ReceitasResult{
		Municipio:  m,
		Fonte:      "TCE-MS Dados Abertos (Repasse Duodécimos)",
		Plataforma: "tce_ms_barramento",
		PortalURL:  baseURL,
		Receitas:   []ReceitaItem{},
		Disponivel: true,
		Mensagem:   "Nenhum repasse retornado pelo Barramento TCE-MS para o período.",
	}, nil
}

// ============================================================================
// TCE-SE (Sergipe) — Visualizador de Relatórios / Dados Abertos
// Swagger: https://www.tce.se.gov.br/visualizadorRelatorios/swagger/docs/v1
// Nota: API pública sem autenticação (HTTP 200), expondo dados da UG 021011 (TCE-SE).
// ============================================================================

type tceSEEmpenho struct {
	Exercicio              string  `json:"Exercicio"`
	UgCodigo               string  `json:"UgCodigo"`
	Ug                     string  `json:"Ug"`
	Empenho                string  `json:"Empenho"`
	DataEmpenho            string  `json:"DataEmpenho"`
	FuncionalProgramatica  string  `json:"FuncionalProgramatica"`
	NaturezaDespesaElem    string  `json:"NaturezaDespesaElemento"`
	NaturezaDespesaSubelem string  `json:"NaturezaDespesaSubelemento"`
	FonteRecurso           string  `json:"FonteRecurso"`
	ModalidadeLicitacao    string  `json:"ModalidadeLicitacao"`
	Licitacao              string  `json:"Licitacao"`
	Credor                 string  `json:"Credor"`
	ValorEmpenhado         float64 `json:"ValorEmpenhado"`
	ValorLiquidado         float64 `json:"ValorLiquidado"`
	ValorPago              float64 `json:"ValorPago"`
}

type tceSEReceita struct {
	Exercicio       int     `json:"Exercicio"`
	NaturezaReceita string  `json:"NaturezaReceita"`
	FonteRecurso    string  `json:"FonteRecurso"`
	ValorPrevisto   float64 `json:"ValorPrevisto"`
	ValorArrecadado float64 `json:"ValorArrecadado"`
}

type tceSEContrato struct {
	IDContrato       int     `json:"IdContrato"`
	Fornecedor       string  `json:"Fornecedor"`
	NrProcesso       string  `json:"NrProcesso"`
	AnoProcesso      string  `json:"AnoProcesso"`
	DataInicio       string  `json:"DataInicio"`
	DataFim          string  `json:"DataFim"`
	SetorResponsavel string  `json:"SetorResponsavel"`
	ValorGlobal      float64 `json:"ValorGlobal"`
	Objeto           string  `json:"Objeto"`
	TipoProcesso     string  `json:"TipoProcesso"`
	QtdAditivos      int     `json:"QtdAditivos"`
}

// FetchTCESEDespesas consulta empenhos na API de Dados Abertos do TCE-SE (/api/Despesas/Empenhos).
func (c *TCEClient) FetchTCESEDespesas(ctx context.Context, m *Municipio, ano int, ug string, limit int) (*DespesasResult, error) {
	baseURL := c.getTCESEBaseURL()
	limit = normalizeLimit(limit)
	years := []int{ano}
	if ano <= 0 {
		nowY := time.Now().Year()
		years = []int{nowY - 1, nowY - 2, nowY}
	}

	for _, y := range years {
		q := url.Values{}
		q.Set("exercicio", strconv.Itoa(y))
		if strings.TrimSpace(ug) != "" {
			q.Set("ug", strings.TrimSpace(ug))
		}
		u := fmt.Sprintf("%s/api/Despesas/Empenhos?%s", baseURL, q.Encode())
		body, status, err := doJSONGet(ctx, c.client, u)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			continue
		}
		var rows []tceSEEmpenho
		if err := json.Unmarshal(body, &rows); err != nil || len(rows) == 0 {
			continue
		}
		items := make([]DespesaItem, 0, minInt(limit, len(rows)))
		for _, r := range rows {
			ex, _ := strconv.Atoi(strings.TrimSpace(r.Exercicio))
			if ex == 0 {
				ex = y
			}
			elem := strings.TrimSpace(r.NaturezaDespesaSubelem)
			if elem == "" {
				elem = strings.TrimSpace(r.NaturezaDespesaElem)
			}
			items = append(items, DespesaItem{
				NumeroEmpenho:  strings.TrimSpace(r.Empenho),
				Ano:            ex,
				Data:           strings.TrimSpace(r.DataEmpenho),
				Fornecedor:     strings.TrimSpace(r.Credor),
				Orgao:          strings.TrimSpace(r.Ug),
				Funcao:         strings.TrimSpace(r.FuncionalProgramatica),
				Elemento:       elem,
				Historico:      strings.TrimSpace(r.ModalidadeLicitacao),
				ValorEmpenhado: r.ValorEmpenhado,
				ValorLiquidado: r.ValorLiquidado,
				ValorPago:      r.ValorPago,
				Fonte:          "TCE-SE Dados Abertos",
			})
			if len(items) >= limit {
				break
			}
		}
		return &DespesasResult{
			Municipio:      m,
			Fonte:          "TCE-SE Dados Abertos",
			Plataforma:     "tce_se",
			PortalURL:      baseURL,
			TotalRegistros: len(rows),
			Despesas:       items,
			Disponivel:     true,
		}, nil
	}

	return &DespesasResult{
		Municipio:  m,
		Fonte:      "TCE-SE Dados Abertos",
		Plataforma: "tce_se",
		PortalURL:  baseURL,
		Despesas:   []DespesaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma despesa retornada pelo TCE-SE para o período.",
	}, nil
}

// FetchTCESEReceitas consulta receitas arrecadadas na API de Dados Abertos do TCE-SE (/api/Receitas/Arrecadadas).
func (c *TCEClient) FetchTCESEReceitas(ctx context.Context, m *Municipio, ano int, ug string, limit int) (*ReceitasResult, error) {
	baseURL := c.getTCESEBaseURL()
	limit = normalizeLimit(limit)
	years := []int{ano}
	if ano <= 0 {
		nowY := time.Now().Year()
		years = []int{nowY - 1, nowY - 2, nowY}
	}

	for _, y := range years {
		q := url.Values{}
		q.Set("exercicio", strconv.Itoa(y))
		if strings.TrimSpace(ug) != "" {
			q.Set("ug", strings.TrimSpace(ug))
		}
		u := fmt.Sprintf("%s/api/Receitas/Arrecadadas?%s", baseURL, q.Encode())
		body, status, err := doJSONGet(ctx, c.client, u)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			continue
		}
		var rows []tceSEReceita
		if err := json.Unmarshal(body, &rows); err != nil || len(rows) == 0 {
			continue
		}
		items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
		for _, r := range rows {
			items = append(items, ReceitaItem{
				Descricao:       strings.TrimSpace(r.NaturezaReceita),
				Categoria:       strings.TrimSpace(r.FonteRecurso),
				Data:            strconv.Itoa(r.Exercicio),
				ValorPrevisto:   r.ValorPrevisto,
				ValorArrecadado: r.ValorArrecadado,
				Fonte:           "TCE-SE Dados Abertos",
			})
			if len(items) >= limit {
				break
			}
		}
		return &ReceitasResult{
			Municipio:      m,
			Fonte:          "TCE-SE Dados Abertos",
			Plataforma:     "tce_se",
			PortalURL:      baseURL,
			TotalRegistros: len(rows),
			Receitas:       items,
			Disponivel:     true,
		}, nil
	}

	return &ReceitasResult{
		Municipio:  m,
		Fonte:      "TCE-SE Dados Abertos",
		Plataforma: "tce_se",
		PortalURL:  baseURL,
		Receitas:   []ReceitaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma receita retornada pelo TCE-SE para o período.",
	}, nil
}

// FetchTCESEContratos consulta contratos vigentes na API de Dados Abertos do TCE-SE (/api/Contratos/ContratosVigentes).
func (c *TCEClient) FetchTCESEContratos(ctx context.Context, m *Municipio, limit int) (*ContratosResult, error) {
	baseURL := c.getTCESEBaseURL()
	limit = normalizeLimit(limit)

	u := fmt.Sprintf("%s/api/Contratos/ContratosVigentes", baseURL)
	body, status, err := doJSONGet(ctx, c.client, u)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return &ContratosResult{
			Municipio:  m,
			Fonte:      "TCE-SE Dados Abertos",
			Plataforma: "tce_se",
			PortalURL:  baseURL,
			Contratos:  []ContratoItem{},
			Disponivel: true,
			Mensagem:   fmt.Sprintf("TCE-SE /api/Contratos/ContratosVigentes retornou HTTP %d", status),
		}, nil
	}
	var rows []tceSEContrato
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, err
	}
	items := make([]ContratoItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		anoProc, _ := strconv.Atoi(strings.TrimSpace(r.AnoProcesso))
		num := strconv.Itoa(r.IDContrato)
		if strings.TrimSpace(r.NrProcesso) != "" && strings.TrimSpace(r.AnoProcesso) != "" {
			num = fmt.Sprintf("%s/%s", strings.TrimSpace(r.NrProcesso), strings.TrimSpace(r.AnoProcesso))
		}
		items = append(items, ContratoItem{
			Numero:         num,
			Ano:            anoProc,
			Fornecedor:     strings.TrimSpace(r.Fornecedor),
			Objeto:         strings.TrimSpace(r.Objeto),
			Situacao:       strings.TrimSpace(r.TipoProcesso),
			VigenciaInicio: strings.TrimSpace(r.DataInicio),
			VigenciaFim:    strings.TrimSpace(r.DataFim),
			Valor:          r.ValorGlobal,
			Fonte:          "TCE-SE Dados Abertos",
		})
		if len(items) >= limit {
			break
		}
	}

	return &ContratosResult{
		Municipio:      m,
		Fonte:          "TCE-SE Dados Abertos",
		Plataforma:     "tce_se",
		PortalURL:      baseURL,
		TotalRegistros: len(rows),
		Contratos:      items,
		Disponivel:     true,
	}, nil
}
