package transparencia

import (
	"context"
	"fmt"
	"net/http"
	"sort"
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

func (h *Hub) queryServidoresWithDiscovery(ctx context.Context, m *Municipio, disc *PortalDiscoveryResult, opts QueryOptions) (*ServidoresResult, error) {
	p := PortalQueryParams{
		BaseURL:          disc.URLResolvida,
		CodAdministracao: disc.CodAdministracao,
		TenantID:         disc.TenantID,
		NomeFiltro:       opts.NomeFiltro,
		Inicio:           opts.Inicio,
		Fim:              opts.Fim,
		Ano:              opts.Ano,
		Mes:              opts.Mes,
		Limit:            opts.Limit,
	}

	// 1. Conector direto do portal municipal
	if disc.URLResolvida != "" {
		switch disc.Plataforma {
		case "megasoft":
			res, err := QueryMegasoftServidores(ctx, h.HTTPClient, m, p)
			if err == nil && res != nil {
				h.applyServidoresPostProcess(res, opts)
				return res, nil
			}
		case "quasar_publico":
			res, err := QueryQuasarServidores(ctx, h.HTTPClient, m, p)
			if err == nil && res != nil {
				h.applyServidoresPostProcess(res, opts)
				return res, nil
			}
		case "fiorilli_scpi":
			res, err := QueryFiorilliServidores(ctx, h.HTTPClient, m, p)
			if err == nil && res != nil {
				h.applyServidoresPostProcess(res, opts)
				return res, nil
			}
		case "elotech_oxy":
			res, err := QueryElotechServidores(ctx, h.HTTPClient, m, p)
			if err == nil && res != nil {
				h.applyServidoresPostProcess(res, opts)
				return res, nil
			}
		case "betha_cloud":
			res, err := QueryBethaServidores(ctx, h.HTTPClient, m, p)
			if err == nil && res != nil {
				h.applyServidoresPostProcess(res, opts)
				return res, nil
			}
		case "agili_blue":
			res, err := QueryAgiliServidores(ctx, h.HTTPClient, m, p)
			if err == nil && res != nil {
				h.applyServidoresPostProcess(res, opts)
				return res, nil
			}
		case "ipm_atende":
			res, err := QueryIPMServidores(ctx, h.HTTPClient, m, p)
			if err == nil && res != nil {
				h.applyServidoresPostProcess(res, opts)
				return res, nil
			}
		case "el_portaltp":
			res, err := QueryPortalTPServidores(ctx, h.HTTPClient, m, p)
			if err == nil && res != nil {
				h.applyServidoresPostProcess(res, opts)
				return res, nil
			}
		case "sonner_grp":
			res, err := QuerySonnerServidores(ctx, h.HTTPClient, m, p)
			if err == nil && res != nil {
				h.applyServidoresPostProcess(res, opts)
				return res, nil
			}
		case "epublica_portal":
			res, err := QueryEPublicaServidores(ctx, h.HTTPClient, m, p)
			if err == nil && res != nil {
				h.applyServidoresPostProcess(res, opts)
				return res, nil
			}
		case "govbr_cidade360":
			res, err := QueryGovBRServidores(ctx, h.HTTPClient, m, p, disc.TenantID)
			if err == nil && res != nil {
				h.applyServidoresPostProcess(res, opts)
				return res, nil
			}
		}
	}

	// 2. Fallback / Rota estadual centralizada nos TCEs com API aberta (PI e CE)
	switch m.UF {
	case "PI":
		if res, err := h.TCE.FetchTCEPIServidores(ctx, m, opts.Ano, opts.NomeFiltro, opts.Limit); err == nil {
			h.applyServidoresPostProcess(res, opts)
			return res, nil
		}
	case "CE":
		if res, err := h.TCE.FetchTCECEServidores(ctx, m, opts.Ano, opts.Mes, opts.NomeFiltro, opts.Limit); err == nil {
			h.applyServidoresPostProcess(res, opts)
			return res, nil
		}
	}

	// 3. Caso portal não tenha API JSON aberta de folha, informa claramente o que foi descoberto
	dos := BuildMunicipioDossier(m, nil)
	portalLink := disc.URLResolvida
	if portalLink == "" {
		portalLink = dos.PortalTransparenciaPrefeitura
	}
	msg := fmt.Sprintf(
		"Portal identificado (%s) não expõe API JSON padronizada de servidores sem autenticação/WAF. "+
			"Acesse diretamente %s ou consulte os gastos totais com pessoal no SICONFI (RGF).",
		disc.Plataforma, portalLink,
	)
	if disc.Observacoes != "" {
		msg = disc.Observacoes + " " + msg
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
	p := PortalQueryParams{
		BaseURL:          disc.URLResolvida,
		CodAdministracao: disc.CodAdministracao,
		TenantID:         disc.TenantID,
		NomeFiltro:       opts.NomeFiltro,
		Inicio:           opts.Inicio,
		Fim:              opts.Fim,
		Ano:              opts.Ano,
		Mes:              opts.Mes,
		Limit:            opts.Limit,
	}

	if disc.URLResolvida != "" {
		switch disc.Plataforma {
		case "megasoft":
			if res, err := QueryMegasoftDespesas(ctx, h.HTTPClient, m, p); err == nil {
				h.applyDespesasPostProcess(res, opts)
				return res, nil
			}
		case "quasar_publico":
			if res, err := QueryQuasarDespesas(ctx, h.HTTPClient, m, p); err == nil {
				h.applyDespesasPostProcess(res, opts)
				return res, nil
			}
		case "fiorilli_scpi":
			if res, err := QueryFiorilliDespesas(ctx, h.HTTPClient, m, p); err == nil {
				h.applyDespesasPostProcess(res, opts)
				return res, nil
			}
		case "elotech_oxy":
			if res, err := QueryElotechDespesas(ctx, h.HTTPClient, m, p); err == nil {
				h.applyDespesasPostProcess(res, opts)
				return res, nil
			}
		case "betha_cloud":
			if res, err := QueryBethaDespesas(ctx, h.HTTPClient, m, p); err == nil {
				h.applyDespesasPostProcess(res, opts)
				return res, nil
			}
		case "agili_blue":
			if res, err := QueryAgiliDespesas(ctx, h.HTTPClient, m, p); err == nil {
				h.applyDespesasPostProcess(res, opts)
				return res, nil
			}
		case "ipm_atende":
			if res, err := QueryIPMDespesas(ctx, h.HTTPClient, m, p); err == nil {
				h.applyDespesasPostProcess(res, opts)
				return res, nil
			}
		case "equiplano_cloud":
			if res, err := QueryEquiplanoDespesas(ctx, h.HTTPClient, m, p); err == nil {
				h.applyDespesasPostProcess(res, opts)
				return res, nil
			}
		case "el_portaltp":
			if res, err := QueryPortalTPDespesas(ctx, h.HTTPClient, m, p); err == nil {
				h.applyDespesasPostProcess(res, opts)
				return res, nil
			}
		case "sonner_grp":
			if res, err := QuerySonnerDespesas(ctx, h.HTTPClient, m, p); err == nil {
				h.applyDespesasPostProcess(res, opts)
				return res, nil
			}
		case "epublica_portal":
			if res, err := QueryEPublicaDespesas(ctx, h.HTTPClient, m, p); err == nil {
				h.applyDespesasPostProcess(res, opts)
				return res, nil
			}
		case "coplan_gp":
			if res, err := QueryCoplanDespesas(ctx, h.HTTPClient, m, p); err == nil {
				h.applyDespesasPostProcess(res, opts)
				return res, nil
			}
		case "thema_apigw":
			if res, err := QueryThemaDespesas(ctx, h.HTTPClient, m, p); err == nil {
				h.applyDespesasPostProcess(res, opts)
				return res, nil
			}
		}
	}

	switch m.UF {
	case "SP":
		if res, err := h.TCE.FetchTCESPDespesas(ctx, m, opts.Ano, opts.Mes, opts.Limit); err == nil {
			h.applyDespesasPostProcess(res, opts)
			return res, nil
		}
	case "PI":
		if res, err := h.TCE.FetchTCEPIDespesas(ctx, m, opts.Ano, opts.Limit); err == nil {
			h.applyDespesasPostProcess(res, opts)
			return res, nil
		}
	case "PE":
		if res, err := h.TCE.FetchTCEPEDespesas(ctx, m, opts.Ano, opts.Mes, opts.Limit); err == nil {
			h.applyDespesasPostProcess(res, opts)
			return res, nil
		}
	case "RN":
		if res, err := h.TCE.FetchTCERNDespesas(ctx, m, opts.Ano, 0, opts.Limit); err == nil {
			h.applyDespesasPostProcess(res, opts)
			return res, nil
		}
	case "RS":
		if res, err := h.TCE.FetchTCERSDespesas(ctx, m, opts.Ano, opts.Limit); err == nil {
			h.applyDespesasPostProcess(res, opts)
			return res, nil
		}
	case "PR":
		if res, err := h.TCE.FetchTCEPRDespesas(ctx, m, opts.Ano, 0, opts.Limit); err == nil {
			h.applyDespesasPostProcess(res, opts)
			return res, nil
		}
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

func (h *Hub) applyServidoresPostProcess(res *ServidoresResult, opts QueryOptions) {
	if res == nil || len(res.Servidores) <= 1 {
		return
	}
	if opts.OrdemSalarioDesc {
		sort.SliceStable(res.Servidores, func(i, j int) bool {
			return res.Servidores[i].SalarioBruto > res.Servidores[j].SalarioBruto
		})
	}
}

func (h *Hub) applyDespesasPostProcess(res *DespesasResult, opts QueryOptions) {
	if res == nil || len(res.Despesas) <= 1 {
		return
	}
	if opts.OrdemValorDesc {
		sort.SliceStable(res.Despesas, func(i, j int) bool {
			vi := res.Despesas[i].ValorPago
			if vi == 0 {
				vi = res.Despesas[i].ValorEmpenhado
			}
			vj := res.Despesas[j].ValorPago
			if vj == 0 {
				vj = res.Despesas[j].ValorEmpenhado
			}
			return vi > vj
		})
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

	if disc.URLResolvida != "" {
		switch disc.Plataforma {
		case "megasoft":
			if res, err := QueryMegasoftReceitas(ctx, h.HTTPClient, m, p); err == nil {
				return res, nil
			}
		case "quasar_publico":
			if res, err := QueryQuasarReceitas(ctx, h.HTTPClient, m, p); err == nil {
				return res, nil
			}
		case "fiorilli_scpi":
			if res, err := QueryFiorilliReceitas(ctx, h.HTTPClient, m, p); err == nil {
				return res, nil
			}
		case "elotech_oxy":
			if res, err := QueryElotechReceitas(ctx, h.HTTPClient, m, p); err == nil {
				return res, nil
			}
		case "betha_cloud":
			if res, err := QueryBethaReceitas(ctx, h.HTTPClient, m, p); err == nil {
				return res, nil
			}
		case "agili_blue":
			if res, err := QueryAgiliReceitas(ctx, h.HTTPClient, m, p); err == nil {
				return res, nil
			}
		case "ipm_atende":
			if res, err := QueryIPMReceitas(ctx, h.HTTPClient, m, p); err == nil {
				return res, nil
			}
		case "equiplano_cloud":
			if res, err := QueryEquiplanoReceitas(ctx, h.HTTPClient, m, p); err == nil {
				return res, nil
			}
		case "el_portaltp":
			if res, err := QueryPortalTPReceitas(ctx, h.HTTPClient, m, p); err == nil {
				return res, nil
			}
		case "sonner_grp":
			if res, err := QuerySonnerReceitas(ctx, h.HTTPClient, m, p); err == nil {
				return res, nil
			}
		case "epublica_portal":
			if res, err := QueryEPublicaReceitas(ctx, h.HTTPClient, m, p); err == nil {
				return res, nil
			}
		case "govbr_cidade360":
			if res, err := QueryGovBRReceitas(ctx, h.HTTPClient, m, p, disc.TenantID); err == nil {
				return res, nil
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
	return h.PNCP.FetchContratacoesByCNPJ(ctx, m, opts.Inicio, opts.Fim, opts.Ano, 0, opts.Limit)
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

	return h.PNCP.FetchContratosByCNPJ(ctx, m, opts.Inicio, opts.Fim, opts.Ano, opts.Limit)
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
	if disc.URLResolvida != "" {
		p := PortalQueryParams{
			BaseURL:    disc.URLResolvida,
			TenantID:   disc.TenantID,
			NomeFiltro: opts.NomeFiltro,
			Ano:        opts.Ano,
			Mes:        opts.Mes,
			Limit:      opts.Limit,
		}
		switch disc.Plataforma {
		case "megasoft":
			if res, err := QueryMegasoftDiarias(ctx, h.HTTPClient, m, p); err == nil {
				return res, nil
			}
		case "betha_cloud":
			if res, err := QueryBethaDiarias(ctx, h.HTTPClient, m, p); err == nil {
				return res, nil
			}
		case "el_portaltp":
			if res, err := QueryPortalTPDiarias(ctx, h.HTTPClient, m, p); err == nil {
				return res, nil
			}
		}
	}
	return &DiariasResult{
		Municipio:  m,
		Fonte:      "Indisponível via API direta",
		Plataforma: disc.Plataforma,
		PortalURL:  disc.URLResolvida,
		Diarias:    []DiariaItem{},
		Disponivel: false,
		Mensagem:   fmt.Sprintf("Consulta direta de diárias via API está disponível para portais Megasoft, Betha Cloud e E&L PortalTP; plataforma detectada: %s.", disc.Plataforma),
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
	if disc.URLResolvida != "" {
		p := PortalQueryParams{
			BaseURL:    disc.URLResolvida,
			TenantID:   disc.TenantID,
			NomeFiltro: opts.NomeFiltro,
			Ano:        opts.Ano,
			Mes:        opts.Mes,
			Limit:      opts.Limit,
		}
		switch disc.Plataforma {
		case "megasoft":
			if res, err := QueryMegasoftObras(ctx, h.HTTPClient, m, p); err == nil {
				return res, nil
			}
		case "agili_blue":
			if res, err := QueryAgiliObras(ctx, h.HTTPClient, m, p); err == nil {
				return res, nil
			}
		case "equiplano_cloud":
			if res, err := QueryEquiplanoObras(ctx, h.HTTPClient, m, p); err == nil {
				return res, nil
			}
		case "el_portaltp":
			if res, err := QueryPortalTPObras(ctx, h.HTTPClient, m, p); err == nil {
				return res, nil
			}
		case "sonner_grp":
			if res, err := QuerySonnerObras(ctx, h.HTTPClient, m, p); err == nil {
				return res, nil
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
		PortalURL:  disc.URLResolvida,
		Obras:      []ObraItem{},
		Disponivel: false,
		Mensagem:   fmt.Sprintf("Consulta direta de obras via API está disponível para portais Megasoft, Ágili, Equiplano, PortalTP, Sonner e TCE-RJ; plataforma detectada: %s.", disc.Plataforma),
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
	if disc.URLResolvida != "" {
		p := PortalQueryParams{
			BaseURL:    disc.URLResolvida,
			TenantID:   disc.TenantID,
			NomeFiltro: opts.NomeFiltro,
			Ano:        opts.Ano,
			Limit:      opts.Limit,
		}
		switch disc.Plataforma {
		case "megasoft":
			if res, err := QueryMegasoftConvenios(ctx, h.HTTPClient, m, p); err == nil {
				return res, nil
			}
		case "sonner_grp":
			if res, err := QuerySonnerConvenios(ctx, h.HTTPClient, m, p); err == nil {
				return res, nil
			}
		}
	}
	return &ConveniosResult{
		Municipio:  m,
		Fonte:      "Indisponível via API direta",
		Plataforma: disc.Plataforma,
		PortalURL:  disc.URLResolvida,
		Convenios:  []ConvenioItem{},
		Disponivel: false,
		Mensagem:   fmt.Sprintf("Consulta direta de convênios via API está disponível para portais Megasoft e Sonner; plataforma detectada: %s.", disc.Plataforma),
	}, nil
}

// QueryLegislacao consulta legislação e atos normativos municipais (portais Megasoft).
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
	if disc.URLResolvida != "" && disc.Plataforma == "megasoft" {
		if res, err := QueryMegasoftLegislacao(ctx, h.HTTPClient, m, PortalQueryParams{
			BaseURL:    disc.URLResolvida,
			NomeFiltro: opts.NomeFiltro,
			Ano:        opts.Ano,
			Limit:      opts.Limit,
		}); err == nil {
			return res, nil
		}
	}
	return &LegislacaoResult{
		Municipio:  m,
		Fonte:      "Indisponível via API direta",
		Plataforma: disc.Plataforma,
		PortalURL:  disc.URLResolvida,
		Atos:       []LegislacaoItem{},
		Disponivel: false,
		Mensagem:   fmt.Sprintf("Consulta direta de legislação municipal via API está disponível para portais Megasoft; plataforma detectada: %s.", disc.Plataforma),
	}, nil
}

// QueryConcursos consulta lista de aprovados em concursos e processos seletivos (portais Megasoft).
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
	if disc.URLResolvida != "" && disc.Plataforma == "megasoft" {
		if res, err := QueryMegasoftConcursos(ctx, h.HTTPClient, m, PortalQueryParams{
			BaseURL:    disc.URLResolvida,
			NomeFiltro: opts.NomeFiltro,
			Limit:      opts.Limit,
		}); err == nil {
			return res, nil
		}
	}
	return &ConcursosResult{
		Municipio:  m,
		Fonte:      "Indisponível via API direta",
		Plataforma: disc.Plataforma,
		PortalURL:  disc.URLResolvida,
		Aprovados:  []ConcursoItem{},
		Disponivel: false,
		Mensagem:   fmt.Sprintf("Consulta direta de concursos via API está disponível para portais Megasoft; plataforma detectada: %s.", disc.Plataforma),
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

	// 5b. Módulos adicionais Megasoft (Diárias, Obras, Convênios, Legislação, Concursos)
	if disc.Plataforma == "megasoft" && disc.URLResolvida != "" {
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
