package transparencia

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// QueryOptions define parâmetros unificados para consulta de dados de transparência municipal.
type QueryOptions struct {
	UF               string
	OverrideURL      string
	NomeFiltro       string
	Inicio           string
	Fim              string
	Ano              int
	Mes              int
	Limit            int
	Timeout          time.Duration
	OrdemSalarioDesc bool
	OrdemValorDesc   bool
}

// Hub orquestra o motor territorial + descoberta de portais + conectores municipais, estaduais (TCEs) e federais (SICONFI/PNCP).
type Hub struct {
	Engine     *Engine
	HTTPClient *http.Client
	TCE        *TCEClient
	Siconfi    *SiconfiClient
	PNCP       *PNCPClient
}

// NewHub cria um Hub pronto para consultar todos os 5.570 municípios e 27 UFs.
func NewHub(engine *Engine) *Hub {
	hc := &http.Client{
		Timeout: 12 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     30 * time.Second,
		},
	}
	return &Hub{
		Engine:     engine,
		HTTPClient: hc,
		TCE:        NewTCEClient(hc),
		Siconfi:    NewSiconfiClient(),
		PNCP:       NewPNCPClient(),
	}
}

func (h *Hub) resolveMunicipio(query, uf string) (*Municipio, *Estado, error) {
	munis := h.Engine.SearchMunicipios(query, uf, 1)
	if len(munis) == 0 {
		return nil, nil, fmt.Errorf("nenhum município encontrado para '%s' (UF: %s)", query, uf)
	}
	m := munis[0]
	est, _ := h.Engine.GetEstado(m.UF)
	return m, est, nil
}

// DiscoverMunicipioPortal descobre o portal da transparência real e a plataforma de um município.
func (h *Hub) DiscoverMunicipioPortal(ctx context.Context, query string, opts QueryOptions) (*PortalDiscoveryResult, error) {
	m, _, err := h.resolveMunicipio(query, opts.UF)
	if err != nil {
		return nil, err
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return DiscoverPortal(ctx, m, DiscoverOptions{
		OverrideURL:  opts.OverrideURL,
		HTTPClient:   h.HTTPClient,
		TCEPIBaseURL: h.TCE.tcePIBaseURL,
		Timeout:      timeout,
	}), nil
}

// QueryServidores consulta servidores públicos municipais usando a melhor rota disponível (Portal Municipal -> TCE Estadual).
func (h *Hub) QueryServidores(ctx context.Context, query string, opts QueryOptions) (*ServidoresResult, error) {
	m, _, err := h.resolveMunicipio(query, opts.UF)
	if err != nil {
		return nil, err
	}

	disc := DiscoverPortal(ctx, m, DiscoverOptions{
		OverrideURL:  opts.OverrideURL,
		HTTPClient:   h.HTTPClient,
		TCEPIBaseURL: h.TCE.tcePIBaseURL,
		Timeout:      opts.Timeout,
	})
	return h.queryServidoresWithDiscovery(ctx, m, disc, opts)
}

const rankingFetchLimit = 1000

// htmlFallbackTargets retorna URLs candidatas de portal HTML municipal quando o conector estruturado
// falha, não suporta o módulo consultado ou aponta para um portal externo sem API JSON aberta.
func (h *Hub) htmlFallbackTargets(ctx context.Context, m *Municipio, disc *PortalDiscoveryResult, extraURL string) []string {
	isMockTest := strings.Contains(disc.URLResolvida, "127.0.0.1") ||
		strings.Contains(disc.URLResolvida, "localhost") ||
		hasActiveSaaSTestOverride() ||
		(h.TCE != nil && strings.Contains(h.TCE.tcePIBaseURL, "127.0.0.1"))

	var targets []string
	seen := make(map[string]bool)
	add := func(u string) {
		u = strings.TrimSpace(u)
		if u == "" || seen[u] {
			return
		}
		if isMockTest && !strings.Contains(u, "127.0.0.1") && !strings.Contains(u, "localhost") {
			return
		}
		seen[u] = true
		targets = append(targets, u)
	}
	if extraURL != "" && extraURL != disc.URLResolvida {
		add(extraURL)
	}
	if htmlRec, ok := LookupPortalHTMLByIBGE(m.CodigoIBGE); ok && htmlRec.PortalURL != disc.URLResolvida {
		add(htmlRec.PortalURL)
	}
	if disc.SitePrefeitura != "" && disc.SitePrefeitura != disc.URLResolvida {
		add(disc.SitePrefeitura)
	}
	if !isMockTest && len(targets) < 2 {
		cands := CandidatePrefeituraURLs(m)
		if len(cands) > 2 {
			cands = cands[:2]
		}
		subCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
		for _, l := range scrapeTransparencyLinks(subCtx, h.HTTPClient, cands) {
			if l == disc.URLResolvida {
				continue
			}
			lowL := strings.ToLower(l)
			for _, hubPrefix := range []string{"/portal-da-transparencia/", "/portal_da_transparencia/", "/transparencia/"} {
				if idx := strings.Index(lowL, hubPrefix); idx > 0 {
					add(l[:idx+len(hubPrefix)-1])
				}
			}
			add(l)
			if len(targets) >= 3 {
				break
			}
		}
		for _, cand := range cands {
			add(cand)
			if len(targets) >= 3 {
				break
			}
		}
		cancel()
	}
	if !isMockTest && len(targets) == 0 && disc.URLResolvida != "" && !IsHTMLPortalPlatform(disc.Plataforma) && disc.Plataforma != "anti_bot_protegido" {
		add(disc.URLResolvida)
	}
	return targets
}

// detectEmbeddedSaaSDiscovery verifica se uma URL descoberta dentro de um portal HTML
// (via iframe, redirect JS ou link externo) corresponde a uma plataforma SaaS estruturada.
func detectEmbeddedSaaSDiscovery(candidateURL string) (plataforma, resolvedURL, tenantID string, ok bool) {
	u := strings.TrimSpace(candidateURL)
	if u == "" {
		return "", "", "", false
	}
	low := strings.ToLower(u)
	if m := bethaHashRegex.FindStringSubmatch(u); len(m) >= 2 {
		return "betha_cloud", "https://transparencia.betha.cloud/#/" + m[1], m[1], true
	}
	if strings.Contains(low, ".portaltp.com.br") {
		return "el_portaltp", strings.TrimRight(u, "/"), "", true
	}
	if code := ExtractAspecEntityCode(u); code != "" && strings.Contains(low, "governotransparente.com.br") {
		return "aspec_gt", "https://www.governotransparente.com.br/" + code, code, true
	}
	if t, ent := ExtractMemoryTenant(u); t != "" {
		tid := t
		if ent != "" {
			tid = t + "/" + ent
		}
		return "memory_openknowledge", u, tid, true
	}
	return "", "", "", false
}

// fetchSiconfiDespesasFallback converte demonstrativos fiscais oficiais do SICONFI (DCA / RREO)
// em itens estruturados de DespesasResult quando o portal municipal está sob WAF ou sem tabelas diretas.
func (h *Hub) fetchSiconfiDespesasFallback(ctx context.Context, m *Municipio, disc *PortalDiscoveryResult, opts QueryOptions) (*DespesasResult, bool) {
	if h.Siconfi == nil {
		return nil, false
	}
	if strings.Contains(disc.URLResolvida, "127.0.0.1") || strings.Contains(disc.URLResolvida, "localhost") || hasActiveSaaSTestOverride() {
		if !strings.Contains(h.Siconfi.baseURL, "127.0.0.1") && !strings.Contains(h.Siconfi.baseURL, "localhost") {
			return nil, false
		}
	}
	ano := opts.Ano
	if ano <= 0 {
		ano = time.Now().Year() - 1
	}
	dca, err := h.Siconfi.FetchDCA(ctx, m.CodigoIBGE, ano)
	if (err != nil || len(dca) == 0) && opts.Ano <= 0 {
		ano--
		dca, err = h.Siconfi.FetchDCA(ctx, m.CodigoIBGE, ano)
	}
	if err != nil || len(dca) == 0 {
		return nil, false
	}

	limit := normalizeLimit(opts.Limit)
	filtro := strings.ToLower(strings.TrimSpace(opts.NomeFiltro))
	var items []DespesaItem
	for _, f := range dca {
		if f.Valor <= 0 {
			continue
		}
		hay := strings.ToLower(f.Conta + " " + f.Coluna + " " + f.Anexo)
		isExpense := strings.Contains(hay, "despesa") || strings.Contains(hay, "empenhad") || strings.Contains(hay, "liquidad") || strings.Contains(hay, "pag") || strings.Contains(hay, "dca-ab") || strings.Contains(hay, "anexo i-d") || strings.Contains(hay, "anexo i-e")
		if !isExpense {
			continue
		}
		if filtro != "" && !strings.Contains(hay, filtro) {
			continue
		}
		emp := f.Valor
		liq := 0.0
		pago := 0.0
		colLow := strings.ToLower(f.Coluna)
		if strings.Contains(colLow, "pago") || strings.Contains(colLow, "paga") {
			pago = f.Valor
		} else if strings.Contains(colLow, "liquidad") {
			liq = f.Valor
		}
		items = append(items, DespesaItem{
			NumeroEmpenho:  fmt.Sprintf("SICONFI-%d", f.Exercicio),
			Ano:            f.Exercicio,
			Data:           fmt.Sprintf("31/12/%d", f.Exercicio),
			Fornecedor:     "Execução Orçamentária Municipal (SICONFI)",
			Orgao:          f.Instituicao,
			Elemento:       f.CodConta,
			Historico:      strings.TrimSpace(f.Conta + " — " + f.Coluna),
			ValorEmpenhado: emp,
			ValorLiquidado: liq,
			ValorPago:      pago,
			Fonte:          "SICONFI DCA (Tesouro Nacional)",
		})
		if len(items) >= limit {
			break
		}
	}
	if len(items) == 0 {
		return nil, false
	}
	res := &DespesasResult{
		Municipio:      m,
		Fonte:          "SICONFI DCA (Tesouro Nacional — Contingência Estruturada)",
		Plataforma:     disc.Plataforma,
		PortalURL:      disc.URLResolvida,
		TotalRegistros: len(items),
		Despesas:       items,
		Disponivel:     true,
		Mensagem:       "Dados extraídos automaticamente do SICONFI (Tesouro Nacional) como contingência estruturada para o portal municipal.",
	}
	h.applyDespesasPostProcess(res, opts)
	return res, true
}

// fetchSiconfiReceitasFallback converte demonstrativos fiscais oficiais do SICONFI (DCA)
// em itens estruturados de ReceitasResult quando o portal municipal está sob WAF ou sem tabelas diretas.
func (h *Hub) fetchSiconfiReceitasFallback(ctx context.Context, m *Municipio, disc *PortalDiscoveryResult, opts QueryOptions) (*ReceitasResult, bool) {
	if h.Siconfi == nil {
		return nil, false
	}
	if strings.Contains(disc.URLResolvida, "127.0.0.1") || strings.Contains(disc.URLResolvida, "localhost") || hasActiveSaaSTestOverride() {
		if !strings.Contains(h.Siconfi.baseURL, "127.0.0.1") && !strings.Contains(h.Siconfi.baseURL, "localhost") {
			return nil, false
		}
	}
	ano := opts.Ano
	if ano <= 0 {
		ano = time.Now().Year() - 1
	}
	dca, err := h.Siconfi.FetchDCA(ctx, m.CodigoIBGE, ano)
	if (err != nil || len(dca) == 0) && opts.Ano <= 0 {
		ano--
		dca, err = h.Siconfi.FetchDCA(ctx, m.CodigoIBGE, ano)
	}
	if err != nil || len(dca) == 0 {
		return nil, false
	}

	limit := normalizeLimit(opts.Limit)
	filtro := strings.ToLower(strings.TrimSpace(opts.NomeFiltro))
	var items []ReceitaItem
	for _, f := range dca {
		if f.Valor <= 0 {
			continue
		}
		hay := strings.ToLower(f.Conta + " " + f.Coluna + " " + f.Anexo)
		isRev := strings.Contains(hay, "receita") || strings.Contains(hay, "arrecadad") || strings.Contains(hay, "tribut") || strings.Contains(hay, "transferencia") || strings.Contains(hay, "anexo i-c")
		if !isRev {
			continue
		}
		if filtro != "" && !strings.Contains(hay, filtro) {
			continue
		}
		items = append(items, ReceitaItem{
			Codigo:          f.CodConta,
			Descricao:       strings.TrimSpace(f.Conta + " — " + f.Coluna),
			Categoria:       f.Anexo,
			Data:            fmt.Sprintf("31/12/%d", f.Exercicio),
			ValorArrecadado: f.Valor,
			Fonte:           "SICONFI DCA (Tesouro Nacional)",
		})
		if len(items) >= limit {
			break
		}
	}
	if len(items) == 0 {
		return nil, false
	}
	return &ReceitasResult{
		Municipio:      m,
		Fonte:          "SICONFI DCA (Tesouro Nacional — Contingência Estruturada)",
		Plataforma:     disc.Plataforma,
		PortalURL:      disc.URLResolvida,
		TotalRegistros: len(items),
		Receitas:       items,
		Disponivel:     true,
		Mensagem:       "Dados extraídos automaticamente do SICONFI (Tesouro Nacional) como contingência estruturada para o portal municipal.",
	}, true
}

func (h *Hub) queryServidoresWithDiscovery(ctx context.Context, m *Municipio, disc *PortalDiscoveryResult, opts QueryOptions) (*ServidoresResult, error) {
	resolvedURL := disc.URLResolvida
	discObs := disc.Observacoes
	fetchLimit := opts.Limit
	if opts.OrdemSalarioDesc && fetchLimit < rankingFetchLimit {
		fetchLimit = rankingFetchLimit
	}
	p := PortalQueryParams{
		BaseURL:          resolvedURL,
		CodAdministracao: disc.CodAdministracao,
		TenantID:         disc.TenantID,
		NomeFiltro:       opts.NomeFiltro,
		Inicio:           opts.Inicio,
		Fim:              opts.Fim,
		Ano:              opts.Ano,
		Mes:              opts.Mes,
		Limit:            fetchLimit,
		OrdemSalarioDesc: opts.OrdemSalarioDesc,
		OrdemValorDesc:   opts.OrdemValorDesc,
	}

	var unavailableRes *ServidoresResult
	acceptServidores := func(res *ServidoresResult, err error) bool {
		if err != nil || res == nil {
			return false
		}
		if res.Disponivel {
			h.applyServidoresPostProcess(res, opts)
			return true
		}
		unavailableRes = res
		return false
	}

	// 1. Conector direto do portal municipal
	if resolvedURL != "" {
		switch disc.Plataforma {
		case "megasoft":
			if res, err := QueryMegasoftServidores(ctx, h.HTTPClient, m, p); acceptServidores(res, err) {
				return res, nil
			}
		case "quasar_publico":
			if res, err := QueryQuasarServidores(ctx, h.HTTPClient, m, p); acceptServidores(res, err) {
				return res, nil
			}
		case "fiorilli_scpi":
			if res, err := QueryFiorilliServidores(ctx, h.HTTPClient, m, p); acceptServidores(res, err) {
				return res, nil
			}
		case "elotech_oxy":
			if res, err := QueryElotechServidores(ctx, h.HTTPClient, m, p); acceptServidores(res, err) {
				return res, nil
			}
		case "betha_cloud":
			if res, err := QueryBethaServidores(ctx, h.HTTPClient, m, p); acceptServidores(res, err) {
				return res, nil
			}
		case "agili_blue":
			if res, err := QueryAgiliServidores(ctx, h.HTTPClient, m, p); acceptServidores(res, err) {
				return res, nil
			}
		case "ipm_atende":
			if res, err := QueryIPMServidores(ctx, h.HTTPClient, m, p); acceptServidores(res, err) {
				return res, nil
			}
		case "el_portaltp":
			if res, err := QueryPortalTPServidores(ctx, h.HTTPClient, m, p); acceptServidores(res, err) {
				return res, nil
			}
		case "sonner_grp":
			if res, err := QuerySonnerServidores(ctx, h.HTTPClient, m, p); acceptServidores(res, err) {
				return res, nil
			}
		case "epublica_portal":
			if res, err := QueryEPublicaServidores(ctx, h.HTTPClient, m, p); acceptServidores(res, err) {
				return res, nil
			}
		case "govbr_cidade360":
			if res, err := QueryGovBRServidores(ctx, h.HTTPClient, m, p, disc.TenantID); acceptServidores(res, err) {
				return res, nil
			}
		case "nucleogov":
			if res, err := QueryNucleoGovServidores(ctx, h.HTTPClient, m, p); acceptServidores(res, err) {
				return res, nil
			}
		case "diretoriodigital_sistp":
			if res, err := QueryDiretorioDigitalServidores(ctx, h.HTTPClient, m, p, disc.TenantID); acceptServidores(res, err) {
				return res, nil
			}
		case "infbr_netuse":
			if res, err := QueryInfBrServidores(ctx, h.HTTPClient, m, p); acceptServidores(res, err) {
				return res, nil
			}
		case "assesi_portal":
			if res, err := QueryAssesiServidores(ctx, h.HTTPClient, m, p); acceptServidores(res, err) {
				return res, nil
			}
		case "centi_api":
			if res, err := QueryCentiServidores(ctx, h.HTTPClient, m, p); acceptServidores(res, err) {
				return res, nil
			}
		case "fator_sistemas":
			if res, err := QueryFatorServidores(ctx, h.HTTPClient, m, p); acceptServidores(res, err) {
				return res, nil
			}
		case "smarapd_pai":
			if res, err := QuerySmarapdServidores(ctx, h.HTTPClient, m, p); acceptServidores(res, err) {
				return res, nil
			}
		case "memory_openknowledge":
			if res, err := QueryMemoryServidores(ctx, h.HTTPClient, m, p); acceptServidores(res, err) {
				return res, nil
			}
		case "facil_transparencia":
			if res, err := QueryFacilServidores(ctx, h.HTTPClient, m, p); acceptServidores(res, err) {
				return res, nil
			}
		case "publicsoft_api":
			if res, err := QueryPublicsoftServidores(ctx, h.HTTPClient, m, p); acceptServidores(res, err) {
				return res, nil
			}
		case "cittaweb_api":
			if res, err := QueryCittaWebServidores(ctx, h.HTTPClient, m, p); acceptServidores(res, err) {
				return res, nil
			}
		case "topsolutions_rn":
			if res, err := QueryTopSolutionsServidores(ctx, h.HTTPClient, m, p); acceptServidores(res, err) {
				return res, nil
			}
		default:
			if IsHTMLPortalPlatform(disc.Plataforma) {
				if res, err := QueryHTMLPortalServidores(ctx, h.HTTPClient, m, p, disc.Plataforma); err == nil && res != nil {
					h.applyServidoresPostProcess(res, opts)
					return res, nil
				} else if res != nil && res.PortalURL != "" {
					resolvedURL = res.PortalURL
					if res.Mensagem != "" {
						discObs = res.Mensagem
					}
				} else if err != nil && discObs == "" {
					discObs = err.Error()
				}
			}
		}

		// 1b. Fallback HTML universal quando conector estruturado falhou ou indicou folha externa
		if !IsHTMLPortalPlatform(disc.Plataforma) {
			extraURL := ""
			if unavailableRes != nil {
				extraURL = unavailableRes.PortalURL
			}
			for _, targetURL := range h.htmlFallbackTargets(ctx, m, disc, extraURL) {
				fp := p
				fp.BaseURL = targetURL
				if res, err := QueryHTMLPortalServidores(ctx, h.HTTPClient, m, fp, "html_portal"); err == nil && res != nil && res.Disponivel {
					h.applyServidoresPostProcess(res, opts)
					return res, nil
				} else if res != nil && res.Mensagem != "" && unavailableRes != nil && !strings.Contains(unavailableRes.Mensagem, res.Mensagem) {
					unavailableRes.Mensagem = strings.TrimSpace(unavailableRes.Mensagem + " " + res.Mensagem)
				}
			}
		}
	}

	// 2. Fallback / Rota estadual centralizada nos TCEs com API aberta (PI e CE)
	switch m.UF {
	case "PI":
		if res, err := h.TCE.FetchTCEPIServidores(ctx, m, opts.Ano, opts.NomeFiltro, fetchLimit); err == nil {
			h.applyServidoresPostProcess(res, opts)
			return res, nil
		}
	case "CE":
		if res, err := h.TCE.FetchTCECEServidores(ctx, m, opts.Ano, opts.Mes, opts.NomeFiltro, fetchLimit); err == nil {
			h.applyServidoresPostProcess(res, opts)
			return res, nil
		}
	}

	if unavailableRes != nil {
		return unavailableRes, nil
	}

	// 3. Caso portal não tenha API JSON aberta de folha, informa claramente o que foi descoberto
	dos := BuildMunicipioDossier(m, nil)
	portalLink := resolvedURL
	if portalLink == "" {
		portalLink = dos.PortalTransparenciaPrefeitura
	}
	msg := fmt.Sprintf(
		"Portal identificado (%s) não expõe API JSON padronizada de servidores sem autenticação/WAF. "+
			"Acesse diretamente %s ou consulte os gastos totais com pessoal no SICONFI (RGF).",
		disc.Plataforma, portalLink,
	)
	if discObs != "" {
		msg = discObs + " " + msg
	}
	return &ServidoresResult{
		Municipio:  m,
		Fonte:      "Indisponível via API aberta",
		Plataforma: disc.Plataforma,
		PortalURL:  portalLink,
		Servidores: []ServidorItem{},
		Disponivel: false,
		Mensagem:   msg,
	}, nil
}

// QueryDespesas consulta despesas municipais (Portal Municipal -> TCE-SP / TCE-PI).
func (h *Hub) QueryDespesas(ctx context.Context, query string, opts QueryOptions) (*DespesasResult, error) {
	m, _, err := h.resolveMunicipio(query, opts.UF)
	if err != nil {
		return nil, err
	}
	disc := DiscoverPortal(ctx, m, DiscoverOptions{
		OverrideURL:  opts.OverrideURL,
		HTTPClient:   h.HTTPClient,
		TCEPIBaseURL: h.TCE.tcePIBaseURL,
		Timeout:      opts.Timeout,
	})
	return h.queryDespesasWithDiscovery(ctx, m, disc, opts)
}

func (h *Hub) queryDespesasWithDiscovery(ctx context.Context, m *Municipio, disc *PortalDiscoveryResult, opts QueryOptions) (*DespesasResult, error) {
	fetchLimit := opts.Limit
	if opts.OrdemValorDesc && fetchLimit < rankingFetchLimit {
		fetchLimit = rankingFetchLimit
	}
	p := PortalQueryParams{
		BaseURL:          disc.URLResolvida,
		CodAdministracao: disc.CodAdministracao,
		TenantID:         disc.TenantID,
		NomeFiltro:       opts.NomeFiltro,
		Inicio:           opts.Inicio,
		Fim:              opts.Fim,
		Ano:              opts.Ano,
		Mes:              opts.Mes,
		Limit:            fetchLimit,
		OrdemSalarioDesc: opts.OrdemSalarioDesc,
		OrdemValorDesc:   opts.OrdemValorDesc,
	}

	var unavailableRes *DespesasResult
	acceptDespesas := func(res *DespesasResult, err error) bool {
		if err != nil || res == nil {
			return false
		}
		if res.Disponivel {
			h.applyDespesasPostProcess(res, opts)
			return true
		}
		unavailableRes = res
		return false
	}

	if disc.URLResolvida != "" {
		switch disc.Plataforma {
		case "megasoft":
			if res, err := QueryMegasoftDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "quasar_publico":
			if res, err := QueryQuasarDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "fiorilli_scpi":
			if res, err := QueryFiorilliDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "elotech_oxy":
			if res, err := QueryElotechDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "betha_cloud":
			if res, err := QueryBethaDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "agili_blue":
			if res, err := QueryAgiliDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "ipm_atende":
			if res, err := QueryIPMDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "equiplano_cloud":
			if res, err := QueryEquiplanoDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "el_portaltp":
			if res, err := QueryPortalTPDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "sonner_grp":
			if res, err := QuerySonnerDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "epublica_portal":
			if res, err := QueryEPublicaDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "coplan_gp":
			if res, err := QueryCoplanDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "thema_apigw":
			if res, err := QueryThemaDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "nucleogov":
			if res, err := QueryNucleoGovDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "aspec_gt":
			if res, err := QueryAspecDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "assesi_portal":
			if res, err := QueryAssesiDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "centi_api":
			if res, err := QueryCentiDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "fator_sistemas":
			if res, err := QueryFatorDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "smarapd_pai":
			if res, err := QuerySmarapdDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "memory_openknowledge":
			if res, err := QueryMemoryDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "facil_transparencia":
			if res, err := QueryFacilDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "publicsoft_api":
			if res, err := QueryPublicsoftDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "tomeconta_appbr":
			if res, err := QueryTomeContaDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "cittaweb_api":
			if res, err := QueryCittaWebDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "geosiap_api":
			if res, err := QueryGeoSiapDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		case "topsolutions_rn":
			if res, err := QueryTopSolutionsDespesas(ctx, h.HTTPClient, m, p); acceptDespesas(res, err) {
				return res, nil
			}
		default:
			if IsHTMLPortalPlatform(disc.Plataforma) {
				if res, err := QueryHTMLPortalDespesas(ctx, h.HTTPClient, m, p, disc.Plataforma); acceptDespesas(res, err) {
					return res, nil
				}
			}
		}

		if !IsHTMLPortalPlatform(disc.Plataforma) {
			extraURL := ""
			if unavailableRes != nil {
				extraURL = unavailableRes.PortalURL
			}
			for _, targetURL := range h.htmlFallbackTargets(ctx, m, disc, extraURL) {
				fp := p
				fp.BaseURL = targetURL
				if res, err := QueryHTMLPortalDespesas(ctx, h.HTTPClient, m, fp, "html_portal"); err == nil && res != nil && res.Disponivel {
					h.applyDespesasPostProcess(res, opts)
					return res, nil
				}
			}
		}
	}

	switch m.UF {
	case "SP":
		if res, err := h.TCE.FetchTCESPDespesas(ctx, m, opts.Ano, opts.Mes, fetchLimit); err == nil {
			h.applyDespesasPostProcess(res, opts)
			return res, nil
		}
	case "PI":
		if res, err := h.TCE.FetchTCEPIDespesas(ctx, m, opts.Ano, fetchLimit); err == nil {
			h.applyDespesasPostProcess(res, opts)
			return res, nil
		}
	case "PE":
		if res, err := h.TCE.FetchTCEPEDespesas(ctx, m, opts.Ano, opts.Mes, fetchLimit); err == nil {
			h.applyDespesasPostProcess(res, opts)
			return res, nil
		}
	case "RN":
		if res, err := h.TCE.FetchTCERNDespesas(ctx, m, opts.Ano, 0, fetchLimit); err == nil {
			h.applyDespesasPostProcess(res, opts)
			return res, nil
		}
	case "RS":
		if res, err := h.TCE.FetchTCERSDespesas(ctx, m, opts.Ano, fetchLimit); err == nil {
			h.applyDespesasPostProcess(res, opts)
			return res, nil
		}
	case "PR":
		if res, err := h.TCE.FetchTCEPRDespesas(ctx, m, opts.Ano, 0, fetchLimit); err == nil {
			h.applyDespesasPostProcess(res, opts)
			return res, nil
		}
	}

	if res, ok := h.fetchSiconfiDespesasFallback(ctx, m, disc, opts); ok {
		return res, nil
	}

	if unavailableRes != nil {
		return unavailableRes, nil
	}

	return &DespesasResult{
		Municipio:  m,
		Fonte:      "Indisponível via API municipal/TCE direta (use SICONFI DCA/RREO)",
		Plataforma: disc.Plataforma,
		PortalURL:  disc.URLResolvida,
		Despesas:   []DespesaItem{},
		Disponivel: false,
		Mensagem:   "Portal municipal sem API JSON aberta suportada; os dados orçamentários anuais/bimestrais estão disponíveis no SICONFI.",
	}, nil
}

func servidorRankingKey(s ServidorItem) string {
	norm := func(v string) string {
		return strings.ToUpper(strings.Join(strings.Fields(strings.TrimSpace(v)), " "))
	}
	nome := norm(s.Nome)
	if nome == "" {
		return ""
	}
	return nome + "|" + norm(s.Matricula) + "|" + norm(s.Cargo)
}

func (h *Hub) applyServidoresPostProcess(res *ServidoresResult, opts QueryOptions) {
	if res == nil || len(res.Servidores) == 0 {
		return
	}
	if opts.OrdemSalarioDesc {
		sort.SliceStable(res.Servidores, func(i, j int) bool {
			if res.Servidores[i].SalarioBruto != res.Servidores[j].SalarioBruto {
				return res.Servidores[i].SalarioBruto > res.Servidores[j].SalarioBruto
			}
			if res.Servidores[i].SalarioLiquido != res.Servidores[j].SalarioLiquido {
				return res.Servidores[i].SalarioLiquido > res.Servidores[j].SalarioLiquido
			}
			return res.Servidores[i].Competencia > res.Servidores[j].Competencia
		})
		if opts.Mes == 0 && opts.Inicio == "" && opts.Fim == "" {
			deduped := make([]ServidorItem, 0, len(res.Servidores))
			seen := make(map[string]struct{}, len(res.Servidores))
			for _, s := range res.Servidores {
				key := servidorRankingKey(s)
				if key != "" {
					if _, ok := seen[key]; ok {
						continue
					}
					seen[key] = struct{}{}
				}
				deduped = append(deduped, s)
			}
			res.Servidores = deduped
		}
	}
	targetLimit := opts.Limit
	if targetLimit <= 0 && opts.OrdemSalarioDesc {
		targetLimit = normalizeLimit(0)
	}
	if targetLimit > 0 && len(res.Servidores) > targetLimit {
		res.Servidores = res.Servidores[:targetLimit]
	}
}

func (h *Hub) applyDespesasPostProcess(res *DespesasResult, opts QueryOptions) {
	if res == nil || len(res.Despesas) == 0 {
		return
	}
	if opts.OrdemValorDesc {
		despesaValor := func(d DespesaItem) float64 {
			if d.ValorPago != 0 {
				return d.ValorPago
			}
			if d.ValorLiquidado != 0 {
				return d.ValorLiquidado
			}
			return d.ValorEmpenhado
		}
		sort.SliceStable(res.Despesas, func(i, j int) bool {
			vi := despesaValor(res.Despesas[i])
			vj := despesaValor(res.Despesas[j])
			if vi != vj {
				return vi > vj
			}
			return res.Despesas[i].ValorEmpenhado > res.Despesas[j].ValorEmpenhado
		})
	}
	targetLimit := opts.Limit
	if targetLimit <= 0 && opts.OrdemValorDesc {
		targetLimit = normalizeLimit(0)
	}
	if targetLimit > 0 && len(res.Despesas) > targetLimit {
		res.Despesas = res.Despesas[:targetLimit]
	}
}

// QueryReceitas consulta receitas municipais (Portal Municipal -> TCE-SP / TCE-PI).
func (h *Hub) QueryReceitas(ctx context.Context, query string, opts QueryOptions) (*ReceitasResult, error) {
	m, _, err := h.resolveMunicipio(query, opts.UF)
	if err != nil {
		return nil, err
	}
	disc := DiscoverPortal(ctx, m, DiscoverOptions{
		OverrideURL:  opts.OverrideURL,
		HTTPClient:   h.HTTPClient,
		TCEPIBaseURL: h.TCE.tcePIBaseURL,
		Timeout:      opts.Timeout,
	})
	return h.queryReceitasWithDiscovery(ctx, m, disc, opts)
}

func (h *Hub) queryReceitasWithDiscovery(ctx context.Context, m *Municipio, disc *PortalDiscoveryResult, opts QueryOptions) (*ReceitasResult, error) {
	p := PortalQueryParams{
		BaseURL:          disc.URLResolvida,
		CodAdministracao: disc.CodAdministracao,
		TenantID:         disc.TenantID,
		Inicio:           opts.Inicio,
		Fim:              opts.Fim,
		Ano:              opts.Ano,
		Mes:              opts.Mes,
		Limit:            opts.Limit,
	}

	var unavailableRes *ReceitasResult
	acceptReceitas := func(res *ReceitasResult, err error) bool {
		if err != nil || res == nil {
			return false
		}
		if res.Disponivel {
			return true
		}
		unavailableRes = res
		return false
	}

	if disc.URLResolvida != "" {
		switch disc.Plataforma {
		case "megasoft":
			if res, err := QueryMegasoftReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "quasar_publico":
			if res, err := QueryQuasarReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "fiorilli_scpi":
			if res, err := QueryFiorilliReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "elotech_oxy":
			if res, err := QueryElotechReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "betha_cloud":
			if res, err := QueryBethaReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "agili_blue":
			if res, err := QueryAgiliReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "ipm_atende":
			if res, err := QueryIPMReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "equiplano_cloud":
			if res, err := QueryEquiplanoReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "el_portaltp":
			if res, err := QueryPortalTPReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "sonner_grp":
			if res, err := QuerySonnerReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "epublica_portal":
			if res, err := QueryEPublicaReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "govbr_cidade360":
			if res, err := QueryGovBRReceitas(ctx, h.HTTPClient, m, p, disc.TenantID); acceptReceitas(res, err) {
				return res, nil
			}
		case "nucleogov":
			if res, err := QueryNucleoGovReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "aspec_gt":
			if res, err := QueryAspecReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "assesi_portal":
			if res, err := QueryAssesiReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "centi_api":
			if res, err := QueryCentiReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "fator_sistemas":
			if res, err := QueryFatorReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "smarapd_pai":
			if res, err := QuerySmarapdReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "memory_openknowledge":
			if res, err := QueryMemoryReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "facil_transparencia":
			if res, err := QueryFacilReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "publicsoft_api":
			if res, err := QueryPublicsoftReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "tomeconta_appbr":
			if res, err := QueryTomeContaReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "cittaweb_api":
			if res, err := QueryCittaWebReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "geosiap_api":
			if res, err := QueryGeoSiapReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "multi24h_xml":
			if res, err := QueryMulti24hReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		case "topsolutions_rn":
			if res, err := QueryTopSolutionsReceitas(ctx, h.HTTPClient, m, p); acceptReceitas(res, err) {
				return res, nil
			}
		default:
			if IsHTMLPortalPlatform(disc.Plataforma) {
				if res, err := QueryHTMLPortalReceitas(ctx, h.HTTPClient, m, p, disc.Plataforma); acceptReceitas(res, err) {
					return res, nil
				}
			}
		}

		if !IsHTMLPortalPlatform(disc.Plataforma) {
			extraURL := ""
			if unavailableRes != nil {
				extraURL = unavailableRes.PortalURL
			}
			for _, targetURL := range h.htmlFallbackTargets(ctx, m, disc, extraURL) {
				fp := p
				fp.BaseURL = targetURL
				if res, err := QueryHTMLPortalReceitas(ctx, h.HTTPClient, m, fp, "html_portal"); err == nil && res != nil && res.Disponivel {
					return res, nil
				}
			}
		}
	}

	switch m.UF {
	case "SP":
		if res, err := h.TCE.FetchTCESPReceitas(ctx, m, opts.Ano, opts.Mes, opts.Limit); err == nil {
			return res, nil
		}
	case "PI":
		if res, err := h.TCE.FetchTCEPIReceitas(ctx, m, opts.Ano, opts.Limit); err == nil {
			return res, nil
		}
	case "RN":
		if res, err := h.TCE.FetchTCERNReceitas(ctx, m, opts.Ano, 0, opts.Limit); err == nil {
			return res, nil
		}
	case "RS":
		if res, err := h.TCE.FetchTCERSReceitas(ctx, m, opts.Ano, opts.Limit); err == nil {
			return res, nil
		}
	case "PR":
		if res, err := h.TCE.FetchTCEPRReceitas(ctx, m, opts.Ano, 0, opts.Limit); err == nil {
			return res, nil
		}
	}

	if res, ok := h.fetchSiconfiReceitasFallback(ctx, m, disc, opts); ok {
		return res, nil
	}

	if unavailableRes != nil {
		return unavailableRes, nil
	}

	return &ReceitasResult{
		Municipio:  m,
		Fonte:      "Indisponível via API municipal/TCE direta (use SICONFI DCA/RREO)",
		Plataforma: disc.Plataforma,
		PortalURL:  disc.URLResolvida,
		Receitas:   []ReceitaItem{},
		Disponivel: false,
		Mensagem:   "Portal municipal sem API JSON aberta suportada; receitas consolidadas disponíveis via SICONFI.",
	}, nil
}

// QueryLicitacoes consulta licitações municipais (Portal Municipal -> PNCP Nacional por CNPJ).
func (h *Hub) QueryLicitacoes(ctx context.Context, query string, opts QueryOptions) (*LicitacoesResult, error) {
	m, _, err := h.resolveMunicipio(query, opts.UF)
	if err != nil {
		return nil, err
	}
	disc := DiscoverPortal(ctx, m, DiscoverOptions{
		OverrideURL:  opts.OverrideURL,
		HTTPClient:   h.HTTPClient,
		TCEPIBaseURL: h.TCE.tcePIBaseURL,
		Timeout:      opts.Timeout,
	})
	return h.queryLicitacoesWithDiscovery(ctx, m, disc, opts)
}

func (h *Hub) queryLicitacoesWithDiscovery(ctx context.Context, m *Municipio, disc *PortalDiscoveryResult, opts QueryOptions) (*LicitacoesResult, error) {
	if disc.URLResolvida != "" {
		p := PortalQueryParams{
			BaseURL:          disc.URLResolvida,
			CodAdministracao: disc.CodAdministracao,
			TenantID:         disc.TenantID,
			NomeFiltro:       opts.NomeFiltro,
			Inicio:           opts.Inicio,
			Fim:              opts.Fim,
			Ano:              opts.Ano,
			Limit:            opts.Limit,
		}
		switch disc.Plataforma {
		case "megasoft":
			if res, err := QueryMegasoftLicitacoes(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Licitacoes) > 0 {
				return res, nil
			}
		case "quasar_publico":
			if res, err := QueryQuasarLicitacoes(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Licitacoes) > 0 {
				return res, nil
			}
		case "elotech_oxy":
			if res, err := QueryElotechLicitacoes(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Licitacoes) > 0 {
				return res, nil
			}
		case "betha_cloud":
			if res, err := QueryBethaLicitacoes(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Licitacoes) > 0 {
				return res, nil
			}
		case "agili_blue":
			if res, err := QueryAgiliLicitacoes(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Licitacoes) > 0 {
				return res, nil
			}
		case "ipm_atende":
			if res, err := QueryIPMLicitacoes(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Licitacoes) > 0 {
				return res, nil
			}
		case "el_portaltp":
			if res, err := QueryPortalTPLicitacoes(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Licitacoes) > 0 {
				return res, nil
			}
		case "sonner_grp":
			if res, err := QuerySonnerLicitacoes(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Licitacoes) > 0 {
				return res, nil
			}
		case "epublica_portal":
			if res, err := QueryEPublicaLicitacoes(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Licitacoes) > 0 {
				return res, nil
			}
		case "nucleogov":
			if res, err := QueryNucleoGovLicitacoes(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Licitacoes) > 0 {
				return res, nil
			}
		case "aspec_gt":
			if res, err := QueryAspecLicitacoes(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Licitacoes) > 0 {
				return res, nil
			}
		case "assesi_portal":
			if res, err := QueryAssesiLicitacoes(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Licitacoes) > 0 {
				return res, nil
			}
		case "centi_api":
			if res, err := QueryCentiLicitacoes(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Licitacoes) > 0 {
				return res, nil
			}
		case "fator_sistemas":
			if res, err := QueryFatorLicitacoes(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Licitacoes) > 0 {
				return res, nil
			}
		case "smarapd_pai":
			if res, err := QuerySmarapdLicitacoes(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Licitacoes) > 0 {
				return res, nil
			}
		case "memory_openknowledge":
			if res, err := QueryMemoryLicitacoes(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Licitacoes) > 0 {
				return res, nil
			}
		case "facil_transparencia":
			if res, err := QueryFacilLicitacoes(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Licitacoes) > 0 {
				return res, nil
			}
		case "publicsoft_api":
			if res, err := QueryPublicsoftLicitacoes(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Licitacoes) > 0 {
				return res, nil
			}
		case "tomeconta_appbr":
			if res, err := QueryTomeContaLicitacoes(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Licitacoes) > 0 {
				return res, nil
			}
		case "cittaweb_api":
			if res, err := QueryCittaWebLicitacoes(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Licitacoes) > 0 {
				return res, nil
			}
		case "topsolutions_rn":
			if res, err := QueryTopSolutionsLicitacoes(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Licitacoes) > 0 {
				return res, nil
			}
		default:
			if IsHTMLPortalPlatform(disc.Plataforma) {
				if res, err := QueryHTMLPortalLicitacoes(ctx, h.HTTPClient, m, p, disc.Plataforma); err == nil && res != nil && len(res.Licitacoes) > 0 {
					return res, nil
				}
			}
		}

		if !IsHTMLPortalPlatform(disc.Plataforma) {
			for _, targetURL := range h.htmlFallbackTargets(ctx, m, disc, "") {
				fp := p
				fp.BaseURL = targetURL
				if res, err := QueryHTMLPortalLicitacoes(ctx, h.HTTPClient, m, fp, "html_portal"); err == nil && res != nil && len(res.Licitacoes) > 0 {
					return res, nil
				}
			}
		}
	}

	switch m.UF {
	case "PE":
		if res, err := h.TCE.FetchTCEPELicitacoes(ctx, m, opts.Ano, opts.Limit); err == nil && res != nil && len(res.Licitacoes) > 0 {
			return res, nil
		}
	case "RJ":
		if res, err := h.TCE.FetchTCERJLicitacoes(ctx, m, opts.Ano, opts.Limit); err == nil && res != nil && len(res.Licitacoes) > 0 {
			return res, nil
		}
	case "RN":
		if res, err := h.TCE.FetchTCERNLicitacoes(ctx, m, opts.Ano, opts.Limit); err == nil && res != nil && len(res.Licitacoes) > 0 {
			return res, nil
		}
	}

	// Universal fallback via PNCP (Portal Nacional de Contratações Públicas)
	if res, err := h.PNCP.FetchContratacoesByCNPJ(ctx, m, opts.Inicio, opts.Fim, opts.Ano, 0, opts.Limit); err == nil && res != nil {
		return res, nil
	} else {
		msg := "Portal municipal e PNCP indisponíveis para licitações no momento."
		if err != nil {
			msg = fmt.Sprintf("%s (%v)", msg, err)
		}
		return &LicitacoesResult{
			Municipio:  m,
			Fonte:      "Indisponível (Portal Municipal / PNCP)",
			Plataforma: disc.Plataforma,
			PortalURL:  disc.URLResolvida,
			Licitacoes: []LicitacaoItem{},
			Disponivel: false,
			Mensagem:   msg,
		}, nil
	}
}

// QueryContratos consulta contratos municipais (Portal Municipal -> PNCP Nacional por CNPJ).
func (h *Hub) QueryContratos(ctx context.Context, query string, opts QueryOptions) (*ContratosResult, error) {
	m, _, err := h.resolveMunicipio(query, opts.UF)
	if err != nil {
		return nil, err
	}
	disc := DiscoverPortal(ctx, m, DiscoverOptions{
		OverrideURL:  opts.OverrideURL,
		HTTPClient:   h.HTTPClient,
		TCEPIBaseURL: h.TCE.tcePIBaseURL,
		Timeout:      opts.Timeout,
	})
	return h.queryContratosWithDiscovery(ctx, m, disc, opts)
}

func (h *Hub) queryContratosWithDiscovery(ctx context.Context, m *Municipio, disc *PortalDiscoveryResult, opts QueryOptions) (*ContratosResult, error) {
	if disc.URLResolvida != "" {
		p := PortalQueryParams{
			BaseURL:          disc.URLResolvida,
			CodAdministracao: disc.CodAdministracao,
			TenantID:         disc.TenantID,
			NomeFiltro:       opts.NomeFiltro,
			Inicio:           opts.Inicio,
			Fim:              opts.Fim,
			Ano:              opts.Ano,
			Limit:            opts.Limit,
		}
		switch disc.Plataforma {
		case "megasoft":
			if res, err := QueryMegasoftContratos(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Contratos) > 0 {
				return res, nil
			}
		case "quasar_publico":
			if res, err := QueryQuasarContratos(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Contratos) > 0 {
				return res, nil
			}
		case "betha_cloud":
			if res, err := QueryBethaContratos(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Contratos) > 0 {
				return res, nil
			}
		case "agili_blue":
			if res, err := QueryAgiliContratos(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Contratos) > 0 {
				return res, nil
			}
		case "ipm_atende":
			if res, err := QueryIPMContratos(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Contratos) > 0 {
				return res, nil
			}
		case "equiplano_cloud":
			if res, err := QueryEquiplanoContratos(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Contratos) > 0 {
				return res, nil
			}
		case "el_portaltp":
			if res, err := QueryPortalTPContratos(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Contratos) > 0 {
				return res, nil
			}
		case "sonner_grp":
			if res, err := QuerySonnerContratos(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Contratos) > 0 {
				return res, nil
			}
		case "epublica_portal":
			if res, err := QueryEPublicaContratos(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Contratos) > 0 {
				return res, nil
			}
		case "nucleogov":
			if res, err := QueryNucleoGovContratos(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Contratos) > 0 {
				return res, nil
			}
		case "aspec_gt":
			if res, err := QueryAspecContratos(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Contratos) > 0 {
				return res, nil
			}
		case "assesi_portal":
			if res, err := QueryAssesiContratos(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Contratos) > 0 {
				return res, nil
			}
		case "centi_api":
			if res, err := QueryCentiContratos(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Contratos) > 0 {
				return res, nil
			}
		case "fator_sistemas":
			if res, err := QueryFatorContratos(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Contratos) > 0 {
				return res, nil
			}
		case "smarapd_pai":
			if res, err := QuerySmarapdContratos(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Contratos) > 0 {
				return res, nil
			}
		case "memory_openknowledge":
			if res, err := QueryMemoryContratos(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Contratos) > 0 {
				return res, nil
			}
		case "publicsoft_api":
			if res, err := QueryPublicsoftContratos(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Contratos) > 0 {
				return res, nil
			}
		case "tomeconta_appbr":
			if res, err := QueryTomeContaContratos(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Contratos) > 0 {
				return res, nil
			}
		case "cittaweb_api":
			if res, err := QueryCittaWebContratos(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Contratos) > 0 {
				return res, nil
			}
		case "geosiap_api":
			if res, err := QueryGeoSiapContratos(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Contratos) > 0 {
				return res, nil
			}
		case "topsolutions_rn":
			if res, err := QueryTopSolutionsContratos(ctx, h.HTTPClient, m, p); err == nil && res != nil && len(res.Contratos) > 0 {
				return res, nil
			}
		default:
			if IsHTMLPortalPlatform(disc.Plataforma) {
				if res, err := QueryHTMLPortalContratos(ctx, h.HTTPClient, m, p, disc.Plataforma); err == nil && res != nil && len(res.Contratos) > 0 {
					return res, nil
				}
			}
		}

		if !IsHTMLPortalPlatform(disc.Plataforma) {
			for _, targetURL := range h.htmlFallbackTargets(ctx, m, disc, "") {
				fp := p
				fp.BaseURL = targetURL
				if res, err := QueryHTMLPortalContratos(ctx, h.HTTPClient, m, fp, "html_portal"); err == nil && res != nil && len(res.Contratos) > 0 {
					return res, nil
				}
			}
		}
	}

	switch m.UF {
	case "PE":
		if res, err := h.TCE.FetchTCEPEContratos(ctx, m, opts.Ano, opts.Limit); err == nil && res != nil && len(res.Contratos) > 0 {
			return res, nil
		}
	case "RJ":
		if res, err := h.TCE.FetchTCERJContratos(ctx, m, opts.Ano, opts.Limit); err == nil && res != nil && len(res.Contratos) > 0 {
			return res, nil
		}
	case "RN":
		if res, err := h.TCE.FetchTCERNContratos(ctx, m, opts.Limit); err == nil && res != nil && len(res.Contratos) > 0 {
			return res, nil
		}
	}

	if res, err := h.PNCP.FetchContratosByCNPJ(ctx, m, opts.Inicio, opts.Fim, opts.Ano, opts.Limit); err == nil && res != nil {
		return res, nil
	} else {
		msg := "Portal municipal e PNCP indisponíveis para contratos no momento."
		if err != nil {
			msg = fmt.Sprintf("%s (%v)", msg, err)
		}
		return &ContratosResult{
			Municipio:  m,
			Fonte:      "Indisponível (Portal Municipal / PNCP)",
			Plataforma: disc.Plataforma,
			PortalURL:  disc.URLResolvida,
			Contratos:  []ContratoItem{},
			Disponivel: false,
			Mensagem:   msg,
		}, nil
	}
}

// QueryDiarias consulta diárias e passagens pagas a servidores e agentes políticos.
func (h *Hub) QueryDiarias(ctx context.Context, query string, opts QueryOptions) (*DiariasResult, error) {
	m, _, err := h.resolveMunicipio(query, opts.UF)
	if err != nil {
		return nil, err
	}
	disc := DiscoverPortal(ctx, m, DiscoverOptions{
		OverrideURL:  opts.OverrideURL,
		HTTPClient:   h.HTTPClient,
		TCEPIBaseURL: h.TCE.tcePIBaseURL,
		Timeout:      opts.Timeout,
	})
	return h.queryDiariasWithDiscovery(ctx, m, disc, opts)
}

func (h *Hub) queryDiariasWithDiscovery(ctx context.Context, m *Municipio, disc *PortalDiscoveryResult, opts QueryOptions) (*DiariasResult, error) {
	resolvedURL := disc.URLResolvida
	if resolvedURL != "" {
		p := PortalQueryParams{
			BaseURL:          resolvedURL,
			CodAdministracao: disc.CodAdministracao,
			TenantID:         disc.TenantID,
			NomeFiltro:       opts.NomeFiltro,
			Inicio:           opts.Inicio,
			Fim:              opts.Fim,
			Ano:              opts.Ano,
			Mes:              opts.Mes,
			Limit:            opts.Limit,
		}
		switch disc.Plataforma {
		case "megasoft":
			if res, err := QueryMegasoftDiarias(ctx, h.HTTPClient, m, p); err == nil && res != nil && res.Disponivel {
				return res, nil
			}
		case "betha_cloud":
			if res, err := QueryBethaDiarias(ctx, h.HTTPClient, m, p); err == nil && res != nil && res.Disponivel {
				return res, nil
			}
		case "el_portaltp":
			if res, err := QueryPortalTPDiarias(ctx, h.HTTPClient, m, p); err == nil && res != nil && res.Disponivel {
				return res, nil
			}
		case "publicsoft_api":
			if res, err := QueryPublicsoftDiarias(ctx, h.HTTPClient, m, p); err == nil && res != nil && res.Disponivel {
				return res, nil
			}
		default:
			if IsHTMLPortalPlatform(disc.Plataforma) {
				if res, err := QueryHTMLPortalDiarias(ctx, h.HTTPClient, m, p, disc.Plataforma); err == nil && res != nil {
					return res, nil
				} else if res != nil && res.PortalURL != "" {
					resolvedURL = res.PortalURL
				}
			}
		}

		if !IsHTMLPortalPlatform(disc.Plataforma) {
			for _, targetURL := range h.htmlFallbackTargets(ctx, m, disc, "") {
				fp := p
				fp.BaseURL = targetURL
				if res, err := QueryHTMLPortalDiarias(ctx, h.HTTPClient, m, fp, "html_portal"); err == nil && res != nil && res.Disponivel {
					return res, nil
				}
			}
		}
	}
	return &DiariasResult{
		Municipio:  m,
		Fonte:      "Indisponível via API direta",
		Plataforma: disc.Plataforma,
		PortalURL:  resolvedURL,
		Diarias:    []DiariaItem{},
		Disponivel: false,
		Mensagem:   fmt.Sprintf("Consulta direta de diárias via API está disponível para portais Megasoft, Betha Cloud, E&L PortalTP, Publicsoft e HTML Portal; plataforma detectada: %s.", disc.Plataforma),
	}, nil
}

// QueryObras consulta obras públicas municipais.
func (h *Hub) QueryObras(ctx context.Context, query string, opts QueryOptions) (*ObrasResult, error) {
	m, _, err := h.resolveMunicipio(query, opts.UF)
	if err != nil {
		return nil, err
	}
	disc := DiscoverPortal(ctx, m, DiscoverOptions{
		OverrideURL:  opts.OverrideURL,
		HTTPClient:   h.HTTPClient,
		TCEPIBaseURL: h.TCE.tcePIBaseURL,
		Timeout:      opts.Timeout,
	})
	return h.queryObrasWithDiscovery(ctx, m, disc, opts)
}

func (h *Hub) queryObrasWithDiscovery(ctx context.Context, m *Municipio, disc *PortalDiscoveryResult, opts QueryOptions) (*ObrasResult, error) {
	resolvedURL := disc.URLResolvida
	if resolvedURL != "" {
		p := PortalQueryParams{
			BaseURL:          resolvedURL,
			CodAdministracao: disc.CodAdministracao,
			TenantID:         disc.TenantID,
			NomeFiltro:       opts.NomeFiltro,
			Inicio:           opts.Inicio,
			Fim:              opts.Fim,
			Ano:              opts.Ano,
			Mes:              opts.Mes,
			Limit:            opts.Limit,
		}
		switch disc.Plataforma {
		case "megasoft":
			if res, err := QueryMegasoftObras(ctx, h.HTTPClient, m, p); err == nil && res != nil && res.Disponivel {
				return res, nil
			}
		case "agili_blue":
			if res, err := QueryAgiliObras(ctx, h.HTTPClient, m, p); err == nil && res != nil && res.Disponivel {
				return res, nil
			}
		case "equiplano_cloud":
			if res, err := QueryEquiplanoObras(ctx, h.HTTPClient, m, p); err == nil && res != nil && res.Disponivel {
				return res, nil
			}
		case "el_portaltp":
			if res, err := QueryPortalTPObras(ctx, h.HTTPClient, m, p); err == nil && res != nil && res.Disponivel {
				return res, nil
			}
		case "sonner_grp":
			if res, err := QuerySonnerObras(ctx, h.HTTPClient, m, p); err == nil && res != nil && res.Disponivel {
				return res, nil
			}
		case "publicsoft_api":
			if res, err := QueryPublicsoftObras(ctx, h.HTTPClient, m, p); err == nil && res != nil && res.Disponivel {
				return res, nil
			}
		default:
			if IsHTMLPortalPlatform(disc.Plataforma) {
				if res, err := QueryHTMLPortalObras(ctx, h.HTTPClient, m, p, disc.Plataforma); err == nil && res != nil {
					return res, nil
				} else if res != nil && res.PortalURL != "" {
					resolvedURL = res.PortalURL
				}
			}
		}

		if !IsHTMLPortalPlatform(disc.Plataforma) {
			for _, targetURL := range h.htmlFallbackTargets(ctx, m, disc, "") {
				fp := p
				fp.BaseURL = targetURL
				if res, err := QueryHTMLPortalObras(ctx, h.HTTPClient, m, fp, "html_portal"); err == nil && res != nil && res.Disponivel {
					return res, nil
				}
			}
		}
	}
	if m.UF == "RJ" {
		if res, err := h.TCE.FetchTCERJObrasParalisadas(ctx, m, opts.Limit); err == nil {
			return res, nil
		}
	}
	return &ObrasResult{
		Municipio:  m,
		Fonte:      "Indisponível via API direta",
		Plataforma: disc.Plataforma,
		PortalURL:  resolvedURL,
		Obras:      []ObraItem{},
		Disponivel: false,
		Mensagem:   fmt.Sprintf("Consulta direta de obras via API está disponível para portais Megasoft, Ágili, Equiplano, PortalTP, Sonner, Publicsoft, HTML Portal e TCE-RJ; plataforma detectada: %s.", disc.Plataforma),
	}, nil
}

// QueryConvenios consulta convênios e transferências firmados pelo município.
func (h *Hub) QueryConvenios(ctx context.Context, query string, opts QueryOptions) (*ConveniosResult, error) {
	m, _, err := h.resolveMunicipio(query, opts.UF)
	if err != nil {
		return nil, err
	}
	disc := DiscoverPortal(ctx, m, DiscoverOptions{
		OverrideURL:  opts.OverrideURL,
		HTTPClient:   h.HTTPClient,
		TCEPIBaseURL: h.TCE.tcePIBaseURL,
		Timeout:      opts.Timeout,
	})
	return h.queryConveniosWithDiscovery(ctx, m, disc, opts)
}

func (h *Hub) queryConveniosWithDiscovery(ctx context.Context, m *Municipio, disc *PortalDiscoveryResult, opts QueryOptions) (*ConveniosResult, error) {
	resolvedURL := disc.URLResolvida
	if resolvedURL != "" {
		p := PortalQueryParams{
			BaseURL:          resolvedURL,
			CodAdministracao: disc.CodAdministracao,
			TenantID:         disc.TenantID,
			NomeFiltro:       opts.NomeFiltro,
			Inicio:           opts.Inicio,
			Fim:              opts.Fim,
			Ano:              opts.Ano,
			Mes:              opts.Mes,
			Limit:            opts.Limit,
		}
		switch disc.Plataforma {
		case "megasoft":
			if res, err := QueryMegasoftConvenios(ctx, h.HTTPClient, m, p); err == nil && res != nil && res.Disponivel {
				return res, nil
			}
		case "sonner_grp":
			if res, err := QuerySonnerConvenios(ctx, h.HTTPClient, m, p); err == nil && res != nil && res.Disponivel {
				return res, nil
			}
		case "publicsoft_api":
			if res, err := QueryPublicsoftConvenios(ctx, h.HTTPClient, m, p); err == nil && res != nil && res.Disponivel {
				return res, nil
			}
		default:
			if IsHTMLPortalPlatform(disc.Plataforma) {
				if res, err := QueryHTMLPortalConvenios(ctx, h.HTTPClient, m, p, disc.Plataforma); err == nil && res != nil {
					return res, nil
				} else if res != nil && res.PortalURL != "" {
					resolvedURL = res.PortalURL
				}
			}
		}

		if !IsHTMLPortalPlatform(disc.Plataforma) {
			for _, targetURL := range h.htmlFallbackTargets(ctx, m, disc, "") {
				fp := p
				fp.BaseURL = targetURL
				if res, err := QueryHTMLPortalConvenios(ctx, h.HTTPClient, m, fp, "html_portal"); err == nil && res != nil && res.Disponivel {
					return res, nil
				}
			}
		}
	}
	return &ConveniosResult{
		Municipio:  m,
		Fonte:      "Indisponível via API direta",
		Plataforma: disc.Plataforma,
		PortalURL:  resolvedURL,
		Convenios:  []ConvenioItem{},
		Disponivel: false,
		Mensagem:   fmt.Sprintf("Consulta direta de convênios via API está disponível para portais Megasoft, Sonner, Publicsoft e HTML Portal; plataforma detectada: %s.", disc.Plataforma),
	}, nil
}

// QueryLegislacao consulta legislação e atos normativos municipais.
func (h *Hub) QueryLegislacao(ctx context.Context, query string, opts QueryOptions) (*LegislacaoResult, error) {
	m, _, err := h.resolveMunicipio(query, opts.UF)
	if err != nil {
		return nil, err
	}
	disc := DiscoverPortal(ctx, m, DiscoverOptions{
		OverrideURL:  opts.OverrideURL,
		HTTPClient:   h.HTTPClient,
		TCEPIBaseURL: h.TCE.tcePIBaseURL,
		Timeout:      opts.Timeout,
	})
	return h.queryLegislacaoWithDiscovery(ctx, m, disc, opts)
}

func (h *Hub) queryLegislacaoWithDiscovery(ctx context.Context, m *Municipio, disc *PortalDiscoveryResult, opts QueryOptions) (*LegislacaoResult, error) {
	resolvedURL := disc.URLResolvida
	if resolvedURL != "" {
		p := PortalQueryParams{
			BaseURL:          resolvedURL,
			CodAdministracao: disc.CodAdministracao,
			TenantID:         disc.TenantID,
			NomeFiltro:       opts.NomeFiltro,
			Inicio:           opts.Inicio,
			Fim:              opts.Fim,
			Ano:              opts.Ano,
			Mes:              opts.Mes,
			Limit:            opts.Limit,
		}
		switch disc.Plataforma {
		case "megasoft":
			if res, err := QueryMegasoftLegislacao(ctx, h.HTTPClient, m, p); err == nil && res != nil && res.Disponivel {
				return res, nil
			}
		default:
			if IsHTMLPortalPlatform(disc.Plataforma) {
				if res, err := QueryHTMLPortalLegislacao(ctx, h.HTTPClient, m, p, disc.Plataforma); err == nil && res != nil {
					return res, nil
				} else if res != nil && res.PortalURL != "" {
					resolvedURL = res.PortalURL
				}
			}
		}

		if !IsHTMLPortalPlatform(disc.Plataforma) {
			for _, targetURL := range h.htmlFallbackTargets(ctx, m, disc, "") {
				fp := p
				fp.BaseURL = targetURL
				if res, err := QueryHTMLPortalLegislacao(ctx, h.HTTPClient, m, fp, "html_portal"); err == nil && res != nil && res.Disponivel {
					return res, nil
				}
			}
		}
	}
	return &LegislacaoResult{
		Municipio:  m,
		Fonte:      "Indisponível via API direta",
		Plataforma: disc.Plataforma,
		PortalURL:  resolvedURL,
		Atos:       []LegislacaoItem{},
		Disponivel: false,
		Mensagem:   fmt.Sprintf("Consulta direta de legislação municipal está disponível para portais Megasoft e HTML Portal; plataforma detectada: %s.", disc.Plataforma),
	}, nil
}

// QueryConcursos consulta lista de aprovados em concursos e processos seletivos.
func (h *Hub) QueryConcursos(ctx context.Context, query string, opts QueryOptions) (*ConcursosResult, error) {
	m, _, err := h.resolveMunicipio(query, opts.UF)
	if err != nil {
		return nil, err
	}
	disc := DiscoverPortal(ctx, m, DiscoverOptions{
		OverrideURL:  opts.OverrideURL,
		HTTPClient:   h.HTTPClient,
		TCEPIBaseURL: h.TCE.tcePIBaseURL,
		Timeout:      opts.Timeout,
	})
	return h.queryConcursosWithDiscovery(ctx, m, disc, opts)
}

func (h *Hub) queryConcursosWithDiscovery(ctx context.Context, m *Municipio, disc *PortalDiscoveryResult, opts QueryOptions) (*ConcursosResult, error) {
	resolvedURL := disc.URLResolvida
	if resolvedURL != "" {
		p := PortalQueryParams{
			BaseURL:          resolvedURL,
			CodAdministracao: disc.CodAdministracao,
			TenantID:         disc.TenantID,
			NomeFiltro:       opts.NomeFiltro,
			Inicio:           opts.Inicio,
			Fim:              opts.Fim,
			Ano:              opts.Ano,
			Mes:              opts.Mes,
			Limit:            opts.Limit,
		}
		switch disc.Plataforma {
		case "megasoft":
			if res, err := QueryMegasoftConcursos(ctx, h.HTTPClient, m, p); err == nil && res != nil && res.Disponivel {
				return res, nil
			}
		default:
			if IsHTMLPortalPlatform(disc.Plataforma) {
				if res, err := QueryHTMLPortalConcursos(ctx, h.HTTPClient, m, p, disc.Plataforma); err == nil && res != nil {
					return res, nil
				} else if res != nil && res.PortalURL != "" {
					resolvedURL = res.PortalURL
				}
			}
		}

		if !IsHTMLPortalPlatform(disc.Plataforma) {
			for _, targetURL := range h.htmlFallbackTargets(ctx, m, disc, "") {
				fp := p
				fp.BaseURL = targetURL
				if res, err := QueryHTMLPortalConcursos(ctx, h.HTTPClient, m, fp, "html_portal"); err == nil && res != nil && res.Disponivel {
					return res, nil
				}
			}
		}
	}
	return &ConcursosResult{
		Municipio:  m,
		Fonte:      "Indisponível via API direta",
		Plataforma: disc.Plataforma,
		PortalURL:  resolvedURL,
		Aprovados:  []ConcursoItem{},
		Disponivel: false,
		Mensagem:   fmt.Sprintf("Consulta direta de concursos está disponível para portais Megasoft e HTML Portal; plataforma detectada: %s.", disc.Plataforma),
	}, nil
}

// QueryServidorDetalhe consulta o contracheque detalhado de um servidor por matrícula (portais Megasoft).
func (h *Hub) QueryServidorDetalhe(ctx context.Context, query, matricula string, opts QueryOptions) (*ServidorDetalheResult, error) {
	m, _, err := h.resolveMunicipio(query, opts.UF)
	if err != nil {
		return nil, err
	}
	disc := DiscoverPortal(ctx, m, DiscoverOptions{
		OverrideURL:  opts.OverrideURL,
		HTTPClient:   h.HTTPClient,
		TCEPIBaseURL: h.TCE.tcePIBaseURL,
		Timeout:      opts.Timeout,
	})
	if disc.URLResolvida != "" && disc.Plataforma == "megasoft" {
		return QueryMegasoftServidorDetalhe(ctx, h.HTTPClient, m, disc.URLResolvida, matricula, opts.Ano, opts.Mes)
	}
	return &ServidorDetalheResult{
		Municipio:  m,
		Matricula:  matricula,
		PortalURL:  disc.URLResolvida,
		Fonte:      "Indisponível via API direta",
		Disponivel: false,
		Mensagem:   fmt.Sprintf("Detalhamento individual de contracheque via API está disponível para portais Megasoft; plataforma detectada: %s.", disc.Plataforma),
	}, nil
}

// QueryTransparenciaCompleta agrega em paralelo toda a transparência disponível para um município brasileiro.
func (h *Hub) QueryTransparenciaCompleta(ctx context.Context, query string, opts QueryOptions) (*TransparenciaCompletaReport, error) {
	start := time.Now()
	m, est, err := h.resolveMunicipio(query, opts.UF)
	if err != nil {
		return nil, err
	}
	if opts.Limit <= 0 {
		opts.Limit = 10
	}

	dos := BuildMunicipioDossier(m, est)
	disc := DiscoverPortal(ctx, m, DiscoverOptions{
		OverrideURL:  opts.OverrideURL,
		HTTPClient:   h.HTTPClient,
		TCEPIBaseURL: h.TCE.tcePIBaseURL,
		Timeout:      opts.Timeout,
	})
	if disc.URLResolvida != "" {
		dos.PortalTransparenciaPrefeitura = disc.URLResolvida
	}
	if disc.SitePrefeitura != "" {
		dos.PortalPrefeitura = disc.SitePrefeitura
	}
	if disc.PortalCamara != "" {
		dos.PortalCamara = disc.PortalCamara
	}

	report := &TransparenciaCompletaReport{
		Dossier:           dos,
		Discovery:         disc,
		FontesConsultadas: []string{"IBGE/SICONFI Cadastro Municipal"},
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	addFonte := func(f string) {
		mu.Lock()
		report.FontesConsultadas = append(report.FontesConsultadas, f)
		mu.Unlock()
	}
	addLimitacao := func(l string) {
		mu.Lock()
		report.Limitacoes = append(report.Limitacoes, l)
		mu.Unlock()
	}

	// 1. Servidores
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := h.queryServidoresWithDiscovery(ctx, m, disc, opts)
		if err == nil && res != nil {
			report.Servidores = res
			if res.Disponivel {
				addFonte(res.Fonte)
			} else if res.Mensagem != "" {
				addLimitacao("Servidores: " + res.Mensagem)
			}
		}
	}()

	// 2. Despesas
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := h.queryDespesasWithDiscovery(ctx, m, disc, opts)
		if err == nil && res != nil {
			report.Despesas = res
			if res.Disponivel {
				addFonte(res.Fonte)
			} else if res.Mensagem != "" {
				addLimitacao("Despesas detalhadas: " + res.Mensagem)
			}
		}
	}()

	// 3. Receitas
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := h.queryReceitasWithDiscovery(ctx, m, disc, opts)
		if err == nil && res != nil {
			report.Receitas = res
			if res.Disponivel {
				addFonte(res.Fonte)
			} else if res.Mensagem != "" {
				addLimitacao("Receitas detalhadas: " + res.Mensagem)
			}
		}
	}()

	// 4. Licitações
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := h.queryLicitacoesWithDiscovery(ctx, m, disc, opts)
		if err == nil && res != nil {
			report.Licitacoes = res
			if res.Disponivel && len(res.Licitacoes) > 0 {
				addFonte(res.Fonte)
			}
		}
	}()

	// 5. Contratos
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := h.queryContratosWithDiscovery(ctx, m, disc, opts)
		if err == nil && res != nil {
			report.Contratos = res
			if res.Disponivel && len(res.Contratos) > 0 {
				addFonte(res.Fonte)
			}
		}
	}()

	// 5b. Módulos adicionais do portal (Diárias, Obras, Convênios, Legislação, Concursos)
	if disc.URLResolvida != "" {
		wg.Add(5)
		go func() {
			defer wg.Done()
			if res, err := h.queryDiariasWithDiscovery(ctx, m, disc, opts); err == nil && res != nil {
				report.Diarias = res
				if res.Disponivel && len(res.Diarias) > 0 {
					addFonte(res.Fonte + " — Diárias")
				}
			}
		}()
		go func() {
			defer wg.Done()
			if res, err := h.queryObrasWithDiscovery(ctx, m, disc, opts); err == nil && res != nil {
				report.Obras = res
				if res.Disponivel && len(res.Obras) > 0 {
					addFonte(res.Fonte + " — Obras")
				}
			}
		}()
		go func() {
			defer wg.Done()
			if res, err := h.queryConveniosWithDiscovery(ctx, m, disc, opts); err == nil && res != nil {
				report.Convenios = res
				if res.Disponivel && len(res.Convenios) > 0 {
					addFonte(res.Fonte + " — Convênios")
				}
			}
		}()
		go func() {
			defer wg.Done()
			if res, err := h.queryLegislacaoWithDiscovery(ctx, m, disc, opts); err == nil && res != nil {
				report.Legislacao = res
				if res.Disponivel && len(res.Atos) > 0 {
					addFonte(res.Fonte + " — Legislação")
				}
			}
		}()
		go func() {
			defer wg.Done()
			if res, err := h.queryConcursosWithDiscovery(ctx, m, disc, opts); err == nil && res != nil {
				report.Concursos = res
				if res.Disponivel && len(res.Aprovados) > 0 {
					addFonte(res.Fonte + " — Concursos")
				}
			}
		}()
	}

	// 6. SICONFI DCA, RGF (gastos LRF com pessoal) e Entregas
	anoFiscal := opts.Ano
	if anoFiscal <= 0 {
		anoFiscal = time.Now().Year() - 1
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		dca, err := h.Siconfi.FetchDCA(ctx, m.CodigoIBGE, anoFiscal)
		if err == nil && len(dca) == 0 && opts.Ano <= 0 {
			dca, err = h.Siconfi.FetchDCA(ctx, m.CodigoIBGE, anoFiscal-1)
		}
		if err == nil && len(dca) > 0 {
			if len(dca) > opts.Limit {
				dca = dca[:opts.Limit]
			}
			report.SiconfiDCA = dca
			addFonte("SICONFI DCA (Tesouro Nacional)")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		rgf, err := h.Siconfi.FetchRGF(ctx, m.CodigoIBGE, anoFiscal, "", 0)
		if err == nil && len(rgf) == 0 && opts.Ano <= 0 {
			rgf, err = h.Siconfi.FetchRGF(ctx, m.CodigoIBGE, anoFiscal-1, "", 0)
		}
		if err == nil && len(rgf) > 0 {
			if len(rgf) > opts.Limit {
				rgf = rgf[:opts.Limit]
			}
			report.SiconfiRGF = rgf
			addFonte("SICONFI RGF / LRF Pessoal (Tesouro Nacional)")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ent, err := h.Siconfi.FetchEntregas(ctx, m.CodigoIBGE, anoFiscal)
		if err == nil && len(ent) > 0 {
			report.SiconfiEntregas = ent
			addFonte("SICONFI Entregas LRF")
		}
	}()

	wg.Wait()
	report.FontesConsultadas = dedupeStrings(report.FontesConsultadas)
	report.Limitacoes = dedupeStrings(report.Limitacoes)
	report.TempoTotalMs = time.Since(start).Milliseconds()
	return report, nil
}
