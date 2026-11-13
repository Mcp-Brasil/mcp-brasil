package transparencia

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	defaultTCEPIBaseURL = "https://sistemas.tce.pi.gov.br/api/portaldacidadania"
	defaultTCECEBaseURL = "https://api-dados-abertos.tce.ce.gov.br/sim"
	defaultTCESPBaseURL = "https://transparencia.tce.sp.gov.br/api/json"
)

// PIPrefeitura representa um município cadastrado na API do TCE-PI.
type PIPrefeitura struct {
	ID            int    `json:"id"`
	Nome          string `json:"nome"`
	CodIBGE       string `json:"codIBGE"`
	URLPrefeitura string `json:"urlPrefeitura"`
	URLCamara     string `json:"urlCamara"`
}

type ceMunicipio struct {
	CodigoMunicipio     string `json:"codigo_municipio"`
	NomeMunicipio       string `json:"nome_municipio"`
	CodigoMunicipioIBGE any    `json:"codigo_municipio_ibge"`
}

type spMunicipio struct {
	Slug string `json:"municipio"`
	Nome string `json:"municipio_extenso"`
}

// TCEClient centraliza conectores para APIs abertas estaduais de Tribunais de Contas (TCE-PI, TCE-CE, TCE-SP).
type TCEClient struct {
	client       *http.Client
	tcePIBaseURL string
	tceCEBaseURL string
	tceSPBaseURL string

	mu         sync.Mutex
	piPrefs    []PIPrefeitura
	ceMunis    []ceMunicipio
	spMunis    []spMunicipio
}

// NewTCEClient cria um cliente para os TCEs estaduais com API pública estruturada.
func NewTCEClient(httpClient *http.Client) *TCEClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 12 * time.Second}
	}
	return &TCEClient{
		client:       httpClient,
		tcePIBaseURL: defaultTCEPIBaseURL,
		tceCEBaseURL: defaultTCECEBaseURL,
		tceSPBaseURL: defaultTCESPBaseURL,
	}
}

func (c *TCEClient) SetTCEPIBaseURL(u string) { c.tcePIBaseURL = strings.TrimRight(u, "/") }
func (c *TCEClient) SetTCECEBaseURL(u string) { c.tceCEBaseURL = strings.TrimRight(u, "/") }
func (c *TCEClient) SetTCESPBaseURL(u string) { c.tceSPBaseURL = strings.TrimRight(u, "/") }

// ResolvePIPrefeitura busca o ID e URLs oficiais no TCE-PI pelo código IBGE de 7 dígitos.
func (c *TCEClient) ResolvePIPrefeitura(ctx context.Context, codIBGE string) (*PIPrefeitura, error) {
	c.mu.Lock()
	cached := c.piPrefs
	c.mu.Unlock()

	if len(cached) == 0 {
		body, status, err := doJSONGet(ctx, c.client, c.tcePIBaseURL+"/prefeituras")
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			return nil, fmt.Errorf("TCE-PI /prefeituras retornou HTTP %d", status)
		}
		if err := json.Unmarshal(body, &cached); err != nil {
			return nil, err
		}
		c.mu.Lock()
		c.piPrefs = cached
		c.mu.Unlock()
	}

	target := strings.TrimSpace(codIBGE)
	for i := range cached {
		if strings.TrimSpace(cached[i].CodIBGE) == target {
			cp := cached[i]
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("município IBGE %s não encontrado no TCE-PI", codIBGE)
}

// FetchTCEPIServidores consulta servidores públicos municipais do Piauí via TCE-PI Portal da Cidadania.
func (c *TCEClient) FetchTCEPIServidores(ctx context.Context, m *Municipio, ano int, filtro string, limit int) (*ServidoresResult, error) {
	pref, err := c.ResolvePIPrefeitura(ctx, m.CodigoIBGE)
	if err != nil {
		return nil, err
	}
	limit = normalizeLimit(limit)

	years := []int{ano}
	if ano <= 0 {
		nowY := time.Now().Year()
		years = []int{nowY, nowY - 1, nowY - 2}
	}

	type piServ struct {
		Nome    string `json:"nome"`
		CPF     string `json:"cpf"`
		Cargo   string `json:"cargo"`
		Lotacao string `json:"lotacao"`
		Vinculo string `json:"vinculo"`
	}

	for _, y := range years {
		u := fmt.Sprintf("%s/servidores/%d/%d/lista?pagina=1&qtdePorPagina=%d&valorFiltro=%s",
			c.tcePIBaseURL, pref.ID, y, limit, url.QueryEscape(strings.TrimSpace(filtro)))
		body, status, err := doJSONGet(ctx, c.client, u)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			continue
		}
		var raw []piServ
		if err := json.Unmarshal(body, &raw); err != nil {
			continue
		}
		if len(raw) == 0 {
			continue
		}
		items := make([]ServidorItem, 0, len(raw))
		for _, r := range raw {
			items = append(items, ServidorItem{
				Nome:        strings.TrimSpace(r.Nome),
				CPFParcial:  strings.TrimSpace(r.CPF),
				Cargo:       strings.TrimSpace(r.Cargo),
				Lotacao:     strings.TrimSpace(r.Lotacao),
				Vinculo:     strings.TrimSpace(r.Vinculo),
				Competencia: fmt.Sprintf("%d", y),
				Fonte:       "TCE-PI Portal da Cidadania",
				PortalURL:   pref.URLPrefeitura,
			})
		}
		return &ServidoresResult{
			Municipio:      m,
			Fonte:          "TCE-PI Portal da Cidadania",
			Plataforma:     "tce_pi",
			PortalURL:      c.tcePIBaseURL,
			TotalRegistros: len(items),
			Servidores:     items,
			Disponivel:     true,
		}, nil
	}

	return &ServidoresResult{
		Municipio:  m,
		Fonte:      "TCE-PI Portal da Cidadania",
		Plataforma: "tce_pi",
		PortalURL:  c.tcePIBaseURL,
		Servidores: []ServidorItem{},
		Disponivel: true,
		Mensagem:   "Nenhum servidor retornado pelo TCE-PI para os anos consultados.",
	}, nil
}

// FetchTCEPIDespesas consulta despesas por função no TCE-PI.
func (c *TCEClient) FetchTCEPIDespesas(ctx context.Context, m *Municipio, ano int, limit int) (*DespesasResult, error) {
	pref, err := c.ResolvePIPrefeitura(ctx, m.CodigoIBGE)
	if err != nil {
		return nil, err
	}
	limit = normalizeLimit(limit)
	years := []int{ano}
	if ano <= 0 {
		nowY := time.Now().Year()
		years = []int{nowY - 1, nowY - 2}
	}

	for _, y := range years {
		u := fmt.Sprintf("%s/despesas/%d/%d/porFuncao", c.tcePIBaseURL, pref.ID, y)
		body, status, err := doJSONGet(ctx, c.client, u)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			continue
		}
		rows, err := decodeFlexibleRecords(body)
		if err != nil || len(rows) == 0 {
			continue
		}
		items := make([]DespesaItem, 0, minInt(limit, len(rows)))
		for _, r := range rows {
			items = append(items, DespesaItem{
				Ano:            y,
				Funcao:         mapGetStr(r, "funcao", "nomeFuncao", "descricao", "nome"),
				ValorEmpenhado: mapGetFloat(r, "valorEmpenhado", "empenhado", "valor"),
				ValorLiquidado: mapGetFloat(r, "valorLiquidado", "liquidado"),
				ValorPago:      mapGetFloat(r, "valorPago", "pago"),
				Fonte:          "TCE-PI Portal da Cidadania",
			})
			if len(items) >= limit {
				break
			}
		}
		return &DespesasResult{
			Municipio:      m,
			Fonte:          "TCE-PI Portal da Cidadania",
			Plataforma:     "tce_pi",
			PortalURL:      c.tcePIBaseURL,
			TotalRegistros: len(rows),
			Despesas:       items,
			Disponivel:     true,
		}, nil
	}

	return &DespesasResult{
		Municipio:  m,
		Fonte:      "TCE-PI Portal da Cidadania",
		Plataforma: "tce_pi",
		PortalURL:  c.tcePIBaseURL,
		Despesas:   []DespesaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma despesa retornada pelo TCE-PI para o período.",
	}, nil
}

// FetchTCEPIReceitas consulta receitas no TCE-PI.
func (c *TCEClient) FetchTCEPIReceitas(ctx context.Context, m *Municipio, ano int, limit int) (*ReceitasResult, error) {
	pref, err := c.ResolvePIPrefeitura(ctx, m.CodigoIBGE)
	if err != nil {
		return nil, err
	}
	limit = normalizeLimit(limit)
	years := []int{ano}
	if ano <= 0 {
		nowY := time.Now().Year()
		years = []int{nowY - 1, nowY - 2}
	}

	for _, y := range years {
		u := fmt.Sprintf("%s/receitas/%d/%d", c.tcePIBaseURL, pref.ID, y)
		body, status, err := doJSONGet(ctx, c.client, u)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			continue
		}
		rows, err := decodeFlexibleRecords(body)
		if err != nil || len(rows) == 0 {
			continue
		}
		items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
		for _, r := range rows {
			items = append(items, ReceitaItem{
				Codigo:          mapGetStr(r, "codigo", "codReceita"),
				Descricao:       mapGetStr(r, "descricao", "nome", "especificacao"),
				ValorPrevisto:   mapGetFloat(r, "valorPrevisto", "previsto", "orcado"),
				ValorArrecadado: mapGetFloat(r, "valorArrecadado", "arrecadado", "valor"),
				Fonte:           "TCE-PI Portal da Cidadania",
			})
			if len(items) >= limit {
				break
			}
		}
		return &ReceitasResult{
			Municipio:      m,
			Fonte:          "TCE-PI Portal da Cidadania",
			Plataforma:     "tce_pi",
			PortalURL:      c.tcePIBaseURL,
			TotalRegistros: len(rows),
			Receitas:       items,
			Disponivel:     true,
		}, nil
	}

	return &ReceitasResult{
		Municipio:  m,
		Fonte:      "TCE-PI Portal da Cidadania",
		Plataforma: "tce_pi",
		PortalURL:  c.tcePIBaseURL,
		Receitas:   []ReceitaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma receita retornada pelo TCE-PI para o período.",
	}, nil
}

// ResolveCECodigoMunicipio resolve o código interno de 3 dígitos do município no TCE-CE SIM.
func (c *TCEClient) ResolveCECodigoMunicipio(ctx context.Context, m *Municipio) (string, error) {
	c.mu.Lock()
	cached := c.ceMunis
	c.mu.Unlock()

	if len(cached) == 0 {
		body, status, err := doJSONGet(ctx, c.client, c.tceCEBaseURL+"/municipios")
		if err != nil {
			return "", err
		}
		if status != http.StatusOK {
			return "", fmt.Errorf("TCE-CE /municipios retornou HTTP %d", status)
		}
		var wrapper struct {
			Elements []ceMunicipio `json:"elements"`
		}
		if err := json.Unmarshal(body, &wrapper); err != nil {
			return "", err
		}
		cached = wrapper.Elements
		c.mu.Lock()
		c.ceMunis = cached
		c.mu.Unlock()
	}

	targetNorm := RemoveAccents(m.Nome)
	for _, cm := range cached {
		if strings.TrimSpace(anyToString(cm.CodigoMunicipioIBGE)) == m.CodigoIBGE {
			return cm.CodigoMunicipio, nil
		}
	}
	for _, cm := range cached {
		if RemoveAccents(strings.TrimSpace(cm.NomeMunicipio)) == targetNorm {
			return cm.CodigoMunicipio, nil
		}
	}
	return "", fmt.Errorf("município %s (%s) não mapeado no TCE-CE SIM", m.Nome, m.CodigoIBGE)
}

// FetchTCECEServidores consulta agentes públicos municipais do Ceará via API Open Data SIM do TCE-CE.
func (c *TCEClient) FetchTCECEServidores(ctx context.Context, m *Municipio, ano, mes int, filtro string, limit int) (*ServidoresResult, error) {
	codCE, err := c.ResolveCECodigoMunicipio(ctx, m)
	if err != nil {
		return nil, err
	}
	limit = normalizeLimit(limit)
	fetchCount := limit
	if filtro != "" && fetchCount < 100 {
		fetchCount = 100
	}

	years := []int{ano}
	if ano <= 0 {
		nowY := time.Now().Year()
		years = []int{nowY - 1, nowY - 2, nowY}
	}
	if mes <= 0 || mes > 12 {
		mes = 1
	}

	type ceAgente struct {
		NomeServidor      string `json:"nome_servidor"`
		CPFSrv            string `json:"cpf_servidor"`
		Matricula         string `json:"numero_matricula"`
		CargaHoraria      any    `json:"valor_carga_horaria"`
		DataPosse         string `json:"data_posse"`
		CodigoVinculo     string `json:"codigo_vinculo"`
		SituacaoFuncional string `json:"situacao_funcional"`
		NomeTipoCargo     string `json:"nm_tipo_cargo"`
		DataReferenciaDoc any    `json:"data_referencia_doc"`
		CodigoOrgao       string `json:"codigo_orgao"`
	}

	type ceFolhaItem struct {
		CPFAgente string  `json:"cpf_agente"`
		ValorItem float64 `json:"valor_item"`
		Dedutivel bool    `json:"codigo_detutibilidade_item"`
	}

	filtroLow := strings.ToLower(strings.TrimSpace(filtro))

	for _, y := range years {
		exOrc := fmt.Sprintf("%04d00", y)
		refDoc := fmt.Sprintf("%04d%02d", y, mes)
		u := fmt.Sprintf("%s/agentes_publicos_municipais?codigo_municipio=%s&exercicio_orcamento=%s&data_referencia_doc=%s&$count=%d&$start_index=0",
			c.tceCEBaseURL, url.QueryEscape(codCE), exOrc, refDoc, fetchCount)
		body, status, err := doJSONGet(ctx, c.client, u)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			continue
		}
		var wrapper struct {
			Elements []ceAgente `json:"elements"`
		}
		if err := json.Unmarshal(body, &wrapper); err != nil {
			continue
		}
		if len(wrapper.Elements) == 0 {
			continue
		}

		// Enriquece remuneração via /sim/agentes_publicos_folha (best-effort)
		type folhaSum struct{ bruto, desc float64 }
		folhaMap := make(map[string]folhaSum)
		uFolha := fmt.Sprintf("%s/agentes_publicos_folha?codigo_municipio=%s&exercicio_orcamento=%s&data_referencia_doc=%s&$count=500&$start_index=0",
			c.tceCEBaseURL, url.QueryEscape(codCE), exOrc, refDoc)
		if fBody, fStatus, fErr := doJSONGet(ctx, c.client, uFolha); fErr == nil && fStatus == http.StatusOK {
			var fWrap struct {
				Elements []ceFolhaItem `json:"elements"`
			}
			if json.Unmarshal(fBody, &fWrap) == nil {
				for _, fi := range fWrap.Elements {
					cur := folhaMap[fi.CPFAgente]
					if fi.Dedutivel {
						cur.desc += fi.ValorItem
					} else {
						cur.bruto += fi.ValorItem
					}
					folhaMap[fi.CPFAgente] = cur
				}
			}
		}

		items := make([]ServidorItem, 0, minInt(limit, len(wrapper.Elements)))
		for _, r := range wrapper.Elements {
			nome := strings.TrimSpace(r.NomeServidor)
			if filtroLow != "" && !strings.Contains(strings.ToLower(nome), filtroLow) {
				continue
			}
			fs := folhaMap[r.CPFSrv]
			liq := fs.bruto - fs.desc
			if liq < 0 {
				liq = 0
			}
			items = append(items, ServidorItem{
				Nome:           nome,
				Matricula:      strings.TrimSpace(r.Matricula),
				Cargo:          strings.TrimSpace(r.NomeTipoCargo),
				Vinculo:        strings.TrimSpace(r.CodigoVinculo),
				Situacao:       strings.TrimSpace(r.SituacaoFuncional),
				SalarioBruto:   fs.bruto,
				Descontos:      fs.desc,
				SalarioLiquido: liq,
				CargaHoraria:   anyToString(r.CargaHoraria),
				DataAdmissao:   strings.TrimSpace(r.DataPosse),
				Orgao:          "Órgão " + strings.TrimSpace(r.CodigoOrgao),
				Competencia:    anyToString(r.DataReferenciaDoc),
				Fonte:          "TCE-CE Dados Abertos (SIM)",
				PortalURL:      c.tceCEBaseURL,
			})
			if len(items) >= limit {
				break
			}
		}
		return &ServidoresResult{
			Municipio:      m,
			Fonte:          "TCE-CE Dados Abertos (SIM)",
			Plataforma:     "tce_ce_sim",
			PortalURL:      c.tceCEBaseURL,
			TotalRegistros: len(items),
			Servidores:     items,
			Disponivel:     true,
		}, nil
	}

	return &ServidoresResult{
		Municipio:  m,
		Fonte:      "TCE-CE Dados Abertos (SIM)",
		Plataforma: "tce_ce_sim",
		PortalURL:  c.tceCEBaseURL,
		Servidores: []ServidorItem{},
		Disponivel: true,
		Mensagem:   "Nenhum agente público retornado pelo TCE-CE SIM para o período.",
	}, nil
}

// ResolveSPSlug resolve o identificador de município na API pública do TCE-SP.
func (c *TCEClient) ResolveSPSlug(ctx context.Context, m *Municipio) (string, error) {
	c.mu.Lock()
	cached := c.spMunis
	c.mu.Unlock()

	if len(cached) == 0 {
		body, status, err := doJSONGet(ctx, c.client, c.tceSPBaseURL+"/municipios")
		if err != nil {
			return "", err
		}
		if status != http.StatusOK {
			return "", fmt.Errorf("TCE-SP /municipios retornou HTTP %d", status)
		}
		if err := json.Unmarshal(body, &cached); err != nil {
			return "", err
		}
		c.mu.Lock()
		c.spMunis = cached
		c.mu.Unlock()
	}

	norm := RemoveAccents(m.Nome)
	for _, sm := range cached {
		if RemoveAccents(strings.TrimSpace(sm.Nome)) == norm {
			return sm.Slug, nil
		}
	}
	return m.Slug, nil
}

// FetchTCESPDespesas consulta despesas municipais paulistas via API do TCE-SP.
func (c *TCEClient) FetchTCESPDespesas(ctx context.Context, m *Municipio, ano, mes, limit int) (*DespesasResult, error) {
	slug, err := c.ResolveSPSlug(ctx, m)
	if err != nil {
		slug = m.Slug
	}
	limit = normalizeLimit(limit)
	if mes <= 0 || mes > 12 {
		mes = 1
	}
	years := []int{ano}
	if ano <= 0 {
		nowY := time.Now().Year()
		years = []int{nowY - 1, nowY - 2, nowY}
	}

	type spDesp struct {
		Orgao       string `json:"orgao"`
		Mes         string `json:"mes"`
		Evento      string `json:"evento"`
		NrEmpenho   string `json:"nr_empenho"`
		IDFornec    string `json:"id_fornecedor"`
		NmFornec    string `json:"nm_fornecedor"`
		DataEmissao string `json:"dt_emissao_despesa"`
		ValorStr    string `json:"vl_despesa"`
	}

	for _, y := range years {
		u := fmt.Sprintf("%s/despesas/%s/%d/%d", c.tceSPBaseURL, url.PathEscape(slug), y, mes)
		body, status, err := doJSONGet(ctx, c.client, u)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			continue
		}
		var rows []spDesp
		if err := json.Unmarshal(body, &rows); err != nil || len(rows) == 0 {
			continue
		}
		items := make([]DespesaItem, 0, minInt(limit, len(rows)))
		for _, r := range rows {
			val, _ := parseBRFloat(r.ValorStr)
			evLow := strings.ToLower(r.Evento)
			it := DespesaItem{
				NumeroEmpenho:       strings.TrimSpace(r.NrEmpenho),
				Ano:                 y,
				Data:                strings.TrimSpace(r.DataEmissao),
				Fornecedor:          strings.TrimSpace(r.NmFornec),
				DocumentoFornecedor: strings.TrimSpace(r.IDFornec),
				Orgao:               strings.TrimSpace(r.Orgao),
				Historico:           strings.TrimSpace(r.Evento),
				Fonte:               "TCE-SP Transparência",
			}
			switch {
			case strings.Contains(evLow, "pago"):
				it.ValorPago = val
			case strings.Contains(evLow, "liquid"):
				it.ValorLiquidado = val
			default:
				it.ValorEmpenhado = val
			}
			items = append(items, it)
			if len(items) >= limit {
				break
			}
		}
		return &DespesasResult{
			Municipio:      m,
			Fonte:          "TCE-SP Transparência",
			Plataforma:     "tce_sp",
			PortalURL:      c.tceSPBaseURL,
			TotalRegistros: len(rows),
			Despesas:       items,
			Disponivel:     true,
		}, nil
	}

	return &DespesasResult{
		Municipio:  m,
		Fonte:      "TCE-SP Transparência",
		Plataforma: "tce_sp",
		PortalURL:  c.tceSPBaseURL,
		Despesas:   []DespesaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma despesa retornada pelo TCE-SP para o período.",
	}, nil
}

// FetchTCESPReceitas consulta receitas municipais paulistas via API do TCE-SP.
func (c *TCEClient) FetchTCESPReceitas(ctx context.Context, m *Municipio, ano, mes, limit int) (*ReceitasResult, error) {
	slug, err := c.ResolveSPSlug(ctx, m)
	if err != nil {
		slug = m.Slug
	}
	limit = normalizeLimit(limit)
	if mes <= 0 || mes > 12 {
		mes = 1
	}
	years := []int{ano}
	if ano <= 0 {
		nowY := time.Now().Year()
		years = []int{nowY - 1, nowY - 2, nowY}
	}

	type spRec struct {
		Orgao         string `json:"orgao"`
		Mes           string `json:"mes"`
		FonteRecurso  string `json:"ds_fonte_recurso"`
		Alinea        string `json:"ds_alinea"`
		SubAlinea     string `json:"ds_subalinea"`
		ValorStr      string `json:"vl_arrecadacao"`
	}

	for _, y := range years {
		u := fmt.Sprintf("%s/receitas/%s/%d/%d", c.tceSPBaseURL, url.PathEscape(slug), y, mes)
		body, status, err := doJSONGet(ctx, c.client, u)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			continue
		}
		var rows []spRec
		if err := json.Unmarshal(body, &rows); err != nil || len(rows) == 0 {
			continue
		}
		items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
		for _, r := range rows {
			val, _ := parseBRFloat(r.ValorStr)
			desc := strings.TrimSpace(r.Alinea)
			if r.SubAlinea != "" {
				desc = desc + " — " + strings.TrimSpace(r.SubAlinea)
			}
			items = append(items, ReceitaItem{
				Descricao:       desc,
				Categoria:       strings.TrimSpace(r.FonteRecurso),
				Data:            fmt.Sprintf("%s/%d", strings.TrimSpace(r.Mes), y),
				ValorArrecadado: val,
				Fonte:           "TCE-SP Transparência",
			})
			if len(items) >= limit {
				break
			}
		}
		return &ReceitasResult{
			Municipio:      m,
			Fonte:          "TCE-SP Transparência",
			Plataforma:     "tce_sp",
			PortalURL:      c.tceSPBaseURL,
			TotalRegistros: len(rows),
			Receitas:       items,
			Disponivel:     true,
		}, nil
	}

	return &ReceitasResult{
		Municipio:  m,
		Fonte:      "TCE-SP Transparência",
		Plataforma: "tce_sp",
		PortalURL:  c.tceSPBaseURL,
		Receitas:   []ReceitaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma receita retornada pelo TCE-SP para o período.",
	}, nil
}
