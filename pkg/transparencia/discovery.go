package transparencia

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// DiscoverOptions configura a descoberta de portais de transparência municipais.
type DiscoverOptions struct {
	OverrideURL  string
	HTTPClient   *http.Client
	TCEPIBaseURL string
	Timeout      time.Duration
}

// QuasarEntidade representa uma entidade retornada por /publico/buscarEntidades.
type QuasarEntidade struct {
	CodAdministracao int    `json:"codAdministracao"`
	DesAdministracao string `json:"desAdministracao"`
	NumCnpj          string `json:"numCnpj"`
	DesSigla         string `json:"desSigla"`
}

var (
	hrefRegex          = regexp.MustCompile(`(?i)href\s*=\s*["']([^"']+)["']`)
	antiBotMarkers     = []string{"cf-turnstile", "g-recaptcha", "hcaptcha", "challenge-platform", "just a moment..."}
	quasarVersaoRegex  = regexp.MustCompile(`^v?\d+\.\d+`)
)

// CandidatePortalURLs gera as variações mais comuns de URL de portal de transparência municipal.
func CandidatePortalURLs(m *Municipio) []string {
	ufLower := strings.ToLower(strings.TrimSpace(m.UF))
	slug := m.Slug
	compact := SlugifyCompact(m.Nome)
	noPrep := SlugifyNoPrepositions(m.Nome)

	raw := []string{
		fmt.Sprintf("https://transparencia.%s.%s.gov.br", compact, ufLower),
		fmt.Sprintf("https://transparencia.%s.%s.gov.br", slug, ufLower),
		fmt.Sprintf("https://portaltransparencia.%s.%s.gov.br", compact, ufLower),
		fmt.Sprintf("https://portaldatransparencia.%s.%s.gov.br", compact, ufLower),
		fmt.Sprintf("https://%s.%s.gov.br/transparencia", compact, ufLower),
		fmt.Sprintf("https://www.%s.%s.gov.br/transparencia", compact, ufLower),
		fmt.Sprintf("https://transparencia.pm%s.%s.gov.br", compact, ufLower),
	}
	if noPrep != "" && noPrep != compact {
		raw = append(raw,
			fmt.Sprintf("https://transparencia.%s.%s.gov.br", noPrep, ufLower),
			fmt.Sprintf("https://portaltransparencia.%s.%s.gov.br", noPrep, ufLower),
		)
	}
	if slug != compact {
		raw = append(raw, fmt.Sprintf("https://portaltransparencia.%s.%s.gov.br", slug, ufLower))
	}
	raw = append(raw,
		fmt.Sprintf("https://%s.megasofttransparencia.com.br", compact),
		fmt.Sprintf("https://%s.oxy.elotech.com.br/portaltransparencia-api", compact),
		fmt.Sprintf("https://transparencia.%s.%s.gov.br/portaltransparencia-api", compact, ufLower),
		fmt.Sprintf("https://%s-%s.portaltp.com.br", compact, ufLower),
		fmt.Sprintf("https://%s.atende.net", compact),
		fmt.Sprintf("https://%s.sonner.com.br", compact),
		fmt.Sprintf("https://grp.%s.%s.gov.br", compact, ufLower),
		fmt.Sprintf("https://webapp1-%s.cidade360.cloud/dadosabertos", compact),
		fmt.Sprintf("https://www.gp.srv.br/transparencia_%s", compact),
	)
	if slug != compact {
		raw = append(raw,
			fmt.Sprintf("https://%s.megasofttransparencia.com.br", slug),
			fmt.Sprintf("https://%s.oxy.elotech.com.br/portaltransparencia-api", slug),
			fmt.Sprintf("https://%s-%s.portaltp.com.br", slug, ufLower),
			fmt.Sprintf("https://%s.atende.net", slug),
			fmt.Sprintf("https://webapp1-%s.cidade360.cloud/dadosabertos", slug),
		)
	}

	return dedupeStrings(raw)
}

// CandidatePrefeituraURLs gera variações do site institucional da prefeitura para raspagem de links.
func CandidatePrefeituraURLs(m *Municipio) []string {
	ufLower := strings.ToLower(strings.TrimSpace(m.UF))
	slug := m.Slug
	compact := SlugifyCompact(m.Nome)
	raw := []string{
		fmt.Sprintf("https://www.%s.%s.gov.br", compact, ufLower),
		fmt.Sprintf("https://%s.%s.gov.br", compact, ufLower),
		fmt.Sprintf("https://www.%s.%s.gov.br", slug, ufLower),
		fmt.Sprintf("https://pm%s.%s.gov.br", compact, ufLower),
	}
	return dedupeStrings(raw)
}

func dedupeStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, it := range items {
		norm := strings.TrimRight(strings.TrimSpace(it), "/")
		if norm == "" {
			continue
		}
		if _, ok := seen[norm]; ok {
			continue
		}
		seen[norm] = struct{}{}
		out = append(out, norm)
	}
	return out
}

// DiscoverPortal descobre a URL real e a plataforma de transparência de um município.
func DiscoverPortal(ctx context.Context, m *Municipio, opts DiscoverOptions) *PortalDiscoveryResult {
	start := time.Now()
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        50,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     15 * time.Second,
			},
		}
	}

	res := &PortalDiscoveryResult{
		Municipio:  m,
		Plataforma: "nao_identificado",
	}

	// 1. Se URL explícita foi informada pelo usuário, testa ela diretamente primeiro.
	if override := strings.TrimRight(strings.TrimSpace(opts.OverrideURL), "/"); override != "" {
		res.CandidatosTestados = append(res.CandidatosTestados, override)
		probeCtx, cancel := context.WithTimeout(ctx, timeout*2)
		det, ok := inspectPortalURL(probeCtx, client, override, "override")
		cancel()
		if ok {
			det.Municipio = m
			det.CandidatosTestados = res.CandidatosTestados
			det.TempoMs = time.Since(start).Milliseconds()
			return det
		}
		res.URLResolvida = override
		res.FonteDescoberta = "override"
		res.Observacoes = "URL fornecida diretamente não respondeu com plataforma estruturada conhecida."
		res.TempoMs = time.Since(start).Milliseconds()
		return res
	}

	// 1b. Catálogo nacional verificado Megasoft (descoberta instantânea para municípios Megasoft).
	if mega, ok := LookupMegasoftByIBGE(m.CodigoIBGE); ok && mega.PortalURL != "" {
		return &PortalDiscoveryResult{
			Municipio:           m,
			URLResolvida:        mega.PortalURL,
			SitePrefeitura:      mega.SitePrefeitura,
			PortalCamara:        mega.PortalCamara,
			Plataforma:          "megasoft",
			CodAdministracao:    mega.CodigoOrgao,
			EntidadeNome:        fmt.Sprintf("%s (%s)", m.Nome, m.UF),
			FonteDescoberta:     "megasoft_catalog",
			ServidoresSuportado: true,
			CandidatosTestados:  []string{mega.PortalURL},
			TempoMs:             time.Since(start).Milliseconds(),
			Observacoes:         "Portal Megasoft (MegaAdmWeb) verificado no catálogo nacional integrado.",
		}
	}

	// Orçamento máximo de descoberta automática para não bloquear fallbacks (TCE/SICONFI/PNCP)
	discCtx, discCancel := context.WithTimeout(ctx, timeout*2)
	defer discCancel()

	// 1c. Descoberta paralela em diretórios SaaS multi-tenant (Betha, Ágili, e-Pública, Equiplano Cloud)
	saasCtx, saasCancel := context.WithTimeout(discCtx, timeout)
	saasHit := probeSaaSDirectoriesConcurrently(saasCtx, client, m)
	saasCancel()
	if saasHit != nil {
		saasHit.Municipio = m
		saasHit.TempoMs = time.Since(start).Milliseconds()
		return saasHit
	}

	// 2. Candidatos de URL + enriquecimento com TCE-PI para municípios do Piauí.
	candidates := CandidatePortalURLs(m)
	prefSites := CandidatePrefeituraURLs(m)

	if m.UF == "PI" {
		tceClient := NewTCEClient(client)
		if opts.TCEPIBaseURL != "" {
			tceClient.SetTCEPIBaseURL(opts.TCEPIBaseURL)
		}
		piCtx, piCancel := context.WithTimeout(discCtx, timeout)
		pref, err := tceClient.ResolvePIPrefeitura(piCtx, m.CodigoIBGE)
		piCancel()
		if err == nil && pref != nil {
			if pref.URLPrefeitura != "" {
				u := strings.TrimRight(strings.TrimSpace(pref.URLPrefeitura), "/")
				res.SitePrefeitura = u
				prefSites = dedupeStrings(append([]string{u}, prefSites...))
				candidates = dedupeStrings(append([]string{u + "/transparencia", u}, candidates...))
			}
			if pref.URLCamara != "" {
				res.PortalCamara = strings.TrimRight(strings.TrimSpace(pref.URLCamara), "/")
			}
		}
	}

	res.CandidatosTestados = append(res.CandidatosTestados, candidates...)

	// 3. Probe paralelo nos candidatos principais de portal de transparência.
	probeCtx, probeCancel := context.WithTimeout(discCtx, timeout)
	best := probeCandidatesConcurrently(probeCtx, client, candidates, "dns_probe")
	probeCancel()
	if best != nil {
		best.Municipio = m
		if res.SitePrefeitura != "" {
			best.SitePrefeitura = res.SitePrefeitura
		}
		if res.PortalCamara != "" {
			best.PortalCamara = res.PortalCamara
		}
		best.CandidatosTestados = res.CandidatosTestados
		best.TempoMs = time.Since(start).Milliseconds()
		return best
	}

	// 4. Fallback: raspa a página inicial da prefeitura buscando link explícito para "transparência" / "LAI".
	scrapeCtx, scrapeCancel := context.WithTimeout(discCtx, timeout)
	discoveredLinks := scrapeTransparencyLinks(scrapeCtx, client, prefSites[:minInt(2, len(prefSites))])
	scrapeCancel()
	if len(discoveredLinks) > 0 {
		res.CandidatosTestados = append(res.CandidatosTestados, discoveredLinks...)
		probe2Ctx, probe2Cancel := context.WithTimeout(discCtx, timeout)
		best := probeCandidatesConcurrently(probe2Ctx, client, discoveredLinks, "homepage_scrape")
		probe2Cancel()
		if best != nil {
			best.Municipio = m
			best.CandidatosTestados = res.CandidatosTestados
			best.TempoMs = time.Since(start).Milliseconds()
			return best
		}
		res.URLResolvida = discoveredLinks[0]
		res.Plataforma = "html_portal"
		res.FonteDescoberta = "homepage_scrape"
		res.Observacoes = "Link de transparência localizado no site da prefeitura, porém sem API JSON aberta identificada."
	} else {
		res.Observacoes = "Nenhum subdomínio padrão respondeu diretamente; utilize o TCE estadual ou SICONFI/PNCP ou informe --url manualmente."
	}

	res.TempoMs = time.Since(start).Milliseconds()
	return res
}

func probeCandidatesConcurrently(ctx context.Context, client *http.Client, candidates []string, source string) *PortalDiscoveryResult {
	if len(candidates) == 0 {
		return nil
	}

	results := make([]*PortalDiscoveryResult, len(candidates))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)

	for i, cand := range candidates {
		wg.Add(1)
		go func(idx int, base string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}
			if det, ok := inspectPortalURL(ctx, client, base, source); ok {
				results[idx] = det
			}
		}(i, cand)
	}
	wg.Wait()

	// Prioriza plataformas estruturadas com suporte a servidores na ordem dos candidatos.
	for _, r := range results {
		if r != nil && r.ServidoresSuportado {
			return r
		}
	}
	for _, r := range results {
		if r != nil && r.Plataforma != "nao_identificado" {
			return r
		}
	}
	for _, r := range results {
		if r != nil {
			return r
		}
	}
	return nil
}

func inspectPortalURL(ctx context.Context, client *http.Client, baseURL string, source string) (*PortalDiscoveryResult, bool) {
	base := strings.TrimRight(baseURL, "/")
	det := &PortalDiscoveryResult{
		URLResolvida:    base,
		Plataforma:      "nao_identificado",
		FonteDescoberta: source,
	}

	// 1) Checa assinatura Quasar /publico/versao
	if ver, ok := probeQuasarVersao(ctx, client, base); ok {
		det.Plataforma = "quasar_publico"
		det.Versao = ver
		det.ServidoresSuportado = true
		if ents := fetchQuasarEntidades(ctx, client, base); len(ents) > 0 {
			det.CodAdministracao = ents[0].CodAdministracao
			det.EntidadeNome = ents[0].DesAdministracao
			// Prefere prefeitura caso haja múltiplas entidades (prefeitura + câmara + previdência)
			for _, e := range ents {
				up := strings.ToUpper(e.DesAdministracao)
				if strings.Contains(up, "PREF") || strings.Contains(up, "MUNICIPIO") || strings.Contains(up, "MUNICÍPIO") {
					det.CodAdministracao = e.CodAdministracao
					det.EntidadeNome = e.DesAdministracao
					break
				}
			}
		}
		return det, true
	}

	// 2) Checa assinatura Fiorilli SCPI (/versaoJson/DadosAbertos/ ou /VersaoJson/DadosAbertos/)
	for _, path := range []string{"/versaoJson/DadosAbertos/", "/VersaoJson/DadosAbertos/", "/Transparencia/VersaoJson/DadosAbertos/"} {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
		resp, err := client.Do(req)
		if err == nil {
			bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			bodyLow := strings.ToLower(string(bodyBytes))
			if resp.StatusCode == http.StatusOK && (strings.Contains(bodyLow, "despesasgerais") || strings.Contains(bodyLow, "fiorilli") || strings.Contains(bodyLow, "versaojson")) {
				det.Plataforma = "fiorilli_scpi"
				det.ServidoresSuportado = true
				return det, true
			}
		}
	}

	// 2b) Checa assinatura Megasoft (/api/configuracao/carregamento-inicial)
	if cfg, err := FetchMegasoftConfig(ctx, client, base, false); err == nil && cfg != nil && cfg.TokenPortal != "" {
		det.Plataforma = "megasoft"
		det.ServidoresSuportado = true
		det.CodAdministracao = cfg.CodigoDoOrgao
		det.EntidadeNome = strings.TrimSpace(strings.TrimSpace(cfg.TituloDoTopo) + " " + strings.TrimSpace(cfg.Cidade))
		det.SitePrefeitura = strings.TrimSpace(cfg.LinkDoSiteDoOrgao)
		det.Observacoes = "Portal Megasoft (MegaAdmWeb) com API REST pública completa."
		return det, true
	}

	// 2c) Checa assinatura Elotech Gestão Pública (portaltransparencia-api OpenAPI 3.0)
	elotechCandidates := []string{base}
	if !strings.HasSuffix(strings.ToLower(base), "/portaltransparencia-api") {
		elotechCandidates = append(elotechCandidates, base+"/portaltransparencia-api")
	}
	for _, eloBase := range elotechCandidates {
		if ProbeElotech(ctx, client, eloBase) {
			det.URLResolvida = eloBase
			det.Plataforma = "elotech_oxy"
			det.ServidoresSuportado = true
			det.Observacoes = "Portal Elotech Gestão Pública (oxy.elotech.com.br) com API OpenAPI 3.0 pública."
			return det, true
		}
	}

	// 2d) Checa URL direta Betha Cloud (https://transparencia.betha.cloud/#/<hash>)
	if strings.Contains(strings.ToLower(base), "transparencia.betha.cloud") {
		if idx := strings.Index(base, "#/"); idx != -1 {
			rest := strings.TrimPrefix(base[idx+2:], "/")
			if slash := strings.IndexByte(rest, '/'); slash != -1 {
				rest = rest[:slash]
			}
			if rest != "" {
				det.Plataforma = "betha_cloud"
				det.TenantID = rest
				det.ServidoresSuportado = true
				det.Observacoes = "Portal Betha Cloud com API JSON pública anônima."
				return det, true
			}
		}
	}

	// 2e) Checa assinatura IPM / Atende.Net (/api/WCPDadosAbertos e /api/transparencia-pessoal-funcionarios)
	if ProbeIPM(ctx, client, base) {
		det.Plataforma = "ipm_atende"
		det.ServidoresSuportado = true
		det.Observacoes = "Portal IPM / Atende.Net com API REST pública de Dados Abertos."
		return det, true
	}

	// 2f) Checa assinatura E&L PortalTP (/api/transparencia.asmx)
	if ProbePortalTP(ctx, client, base) {
		det.Plataforma = "el_portaltp"
		det.ServidoresSuportado = true
		det.Observacoes = "Portal E&L PortalTP com WebService ASMX JSON público."
		return det, true
	}

	// 2g) Checa assinatura Sonner GRP (/GRP/pic/api/config/contexto)
	if ent, ok := ProbeSonner(ctx, client, base); ok {
		det.Plataforma = "sonner_grp"
		det.EntidadeNome = ent
		det.ServidoresSuportado = true
		det.Observacoes = "Portal Sonner GRP com API REST pública (/GRP/pic/api)."
		return det, true
	}

	// 2h) Checa assinatura e-Pública (/rest/{alias}/public/user/attr/exercicios)
	if alias, ent, ok := ProbeEPublicaURL(ctx, client, base); ok {
		det.Plataforma = "epublica_portal"
		det.TenantID = alias
		det.EntidadeNome = ent
		det.ServidoresSuportado = true
		det.Observacoes = "Portal e-Pública com API REST multi-tenant pública."
		return det, true
	}

	// 2i) Checa assinatura GovBR / Pronim / Cidade360 (/dadosabertos/swagger/v1/swagger.json)
	if ent, ok := ProbeGovBRCidade360(ctx, client, base); ok {
		det.Plataforma = "govbr_cidade360"
		det.TenantID = ent
		det.EntidadeNome = ent
		det.ServidoresSuportado = true
		det.Observacoes = "Portal GovBR / Pronim Cidade360 com API OpenAPI pública."
		return det, true
	}

	// 2j) Checa assinatura Coplan (/servlet/api_consulta?unidade_gestora)
	if ugID, ugDesc, ok := ProbeCoplan(ctx, client, base); ok {
		det.Plataforma = "coplan_gp"
		det.CodAdministracao = ugID
		det.EntidadeNome = ugDesc
		det.Observacoes = "Portal Coplan com API JSON pública (api_consulta)."
		return det, true
	}

	// 2k) Checa assinatura Thema API Gateway (/infra/apigw/transparencia/service/...)
	if adminCode, adminName, ok := ProbeThemaAPIGW(ctx, client, base); ok {
		det.Plataforma = "thema_apigw"
		det.CodAdministracao = adminCode
		det.EntidadeNome = adminName
		det.Observacoes = "Portal Thema API Gateway com endpoints JSON públicos."
		return det, true
	}

	// 3) Fallback HTML / portal genérico (detecção de vendor e anti-bot)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base, nil)
	if err != nil {
		return nil, false
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	resp, err := client.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()

	// Atualiza URL final após redirects
	if resp.Request != nil && resp.Request.URL != nil {
		finalBase := strings.TrimRight(resp.Request.URL.Scheme+"://"+resp.Request.URL.Host+resp.Request.URL.Path, "/")
		if finalBase != "" {
			det.URLResolvida = finalBase
		}
	}

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	bodyLow := strings.ToLower(string(bodyBytes))

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusServiceUnavailable {
		for _, m := range antiBotMarkers {
			if strings.Contains(bodyLow, m) {
				det.Plataforma = "anti_bot_protegido"
				det.Observacoes = fmt.Sprintf("Portal protegido por verificação anti-bot/WAF (HTTP %d).", resp.StatusCode)
				return det, true
			}
		}
	}

	if resp.StatusCode < 400 {
		switch {
		case strings.Contains(bodyLow, "atende.net") || strings.Contains(bodyLow, "ipm"):
			det.Plataforma = "ipm_atende"
			det.Observacoes = "Portal IPM/Atende.Net detectado (sem endpoints abertos de Dados Abertos neste endereço)."
		case strings.Contains(bodyLow, "betha"):
			det.Plataforma = "betha"
			det.Observacoes = "Portal Betha Sistemas detectado."
		case strings.Contains(bodyLow, "el.com.br"):
			det.Plataforma = "el_transparencia"
		case strings.Contains(bodyLow, "sonner"):
			det.Plataforma = "sonner"
		case strings.Contains(bodyLow, "fiorilli") || strings.Contains(bodyLow, "scpi"):
			det.Plataforma = "fiorilli_web"
		case strings.Contains(bodyLow, "<html") || strings.Contains(bodyLow, "<!doctype"):
			det.Plataforma = "html_portal"
		}
		if det.Plataforma != "nao_identificado" {
			return det, true
		}
	}

	return nil, false
}

func probeSaaSDirectoriesConcurrently(ctx context.Context, client *http.Client, m *Municipio) *PortalDiscoveryResult {
	type saasSlot struct {
		idx int
		res *PortalDiscoveryResult
	}

	probeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	probes := []func(context.Context, *http.Client, *Municipio) (*PortalDiscoveryResult, bool){
		DiscoverBethaByIBGE,
		DiscoverAgiliByMunicipio,
		DiscoverEPublica,
		DiscoverEquiplanoCloud,
	}

	ch := make(chan saasSlot, len(probes))
	for i, fn := range probes {
		go func(idx int, probeFn func(context.Context, *http.Client, *Municipio) (*PortalDiscoveryResult, bool)) {
			res, ok := probeFn(probeCtx, client, m)
			if !ok {
				res = nil
			}
			ch <- saasSlot{idx: idx, res: res}
		}(i, fn)
	}

	slots := make([]*PortalDiscoveryResult, len(probes))
	var graceTimer *time.Timer
	var graceCh <-chan time.Time
	defer func() {
		if graceTimer != nil {
			graceTimer.Stop()
		}
	}()

	for completed := 0; completed < len(probes); {
		select {
		case item := <-ch:
			completed++
			if item.res != nil {
				slots[item.idx] = item.res
				if item.idx == 0 {
					cancel()
					return item.res
				}
				if graceTimer == nil {
					graceTimer = time.NewTimer(150 * time.Millisecond)
					graceCh = graceTimer.C
				}
			}
		case <-graceCh:
			cancel()
			for _, r := range slots {
				if r != nil {
					return r
				}
			}
			return nil
		case <-probeCtx.Done():
			for _, r := range slots {
				if r != nil {
					return r
				}
			}
			return nil
		}
	}

	for _, res := range slots {
		if res != nil {
			return res
		}
	}
	return nil
}

func probeQuasarVersao(ctx context.Context, client *http.Client, baseURL string) (string, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/publico/versao", nil)
	if err != nil {
		return "", false
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	resp, err := client.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 256))
	if err != nil {
		return "", false
	}
	txt := strings.TrimSpace(strings.Trim(string(b), `"`))
	if txt == "" || strings.Contains(strings.ToLower(txt), "<html") {
		return "", false
	}
	if quasarVersaoRegex.MatchString(txt) || len(txt) <= 32 {
		return txt, true
	}
	return "", false
}

func fetchQuasarEntidades(ctx context.Context, client *http.Client, baseURL string) []QuasarEntidade {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/publico/buscarEntidades", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var ents []QuasarEntidade
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&ents); err != nil {
		return nil
	}
	return ents
}

func scrapeTransparencyLinks(ctx context.Context, client *http.Client, siteURLs []string) []string {
	var found []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, site := range siteURLs {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
			if err != nil {
				return
			}
			req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode >= 400 {
				return
			}
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 128*1024))
			matches := hrefRegex.FindAllStringSubmatch(string(b), -1)
			baseParsed, err := url.Parse(u)
			if err != nil {
				return
			}
			for _, m := range matches {
				if len(m) < 2 {
					continue
				}
				href := strings.TrimSpace(m[1])
				low := strings.ToLower(href)
				if !strings.Contains(low, "transparencia") && !strings.Contains(low, "portal-da-transparencia") {
					continue
				}
				parsed, err := url.Parse(href)
				if err != nil {
					continue
				}
				resolved := baseParsed.ResolveReference(parsed).String()
				if strings.HasPrefix(resolved, "http://") || strings.HasPrefix(resolved, "https://") {
					mu.Lock()
					found = append(found, resolved)
					mu.Unlock()
				}
			}
		}(site)
	}
	wg.Wait()
	return dedupeStrings(found)
}
