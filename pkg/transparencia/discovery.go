package transparencia

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
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
	hrefRegex         = regexp.MustCompile(`(?i)(?:href|src)\s*=\s*["']([^"']+)["']`)
	antiBotMarkers    = []string{"cf-turnstile", "g-recaptcha", "hcaptcha", "challenge-platform", "just a moment..."}
	quasarVersaoRegex = regexp.MustCompile(`^v?\d+\.\d+`)
	bethaHashRegex    = regexp.MustCompile(`(?i)https?://transparencia\.betha\.cloud/#/([a-zA-Z0-9_-]{6,})`)
)

func supportsServidoresForPlatform(plataforma string) bool {
	switch strings.TrimSpace(plataforma) {
	case "megasoft", "quasar_publico", "fiorilli_scpi", "elotech_oxy", "betha_cloud",
		"agili_blue", "ipm_atende", "el_portaltp", "sonner_grp", "epublica_portal",
		"govbr_cidade360", "nucleogov", "diretoriodigital_sistp", "infbr_netuse",
		"assesi_portal", "centi_api", "fator_sistemas", "smarapd_pai",
		"memory_openknowledge", "facil_transparencia", "publicsoft_api",
		"cittaweb_api", "topsolutions_rn":
		return true
	default:
		return false
	}
}

func hasActiveSaaSTestOverride() bool {
	return getBethaAPIBaseURL() != defaultBethaAPIBaseURL ||
		getAgiliAPIBaseURL() != defaultAgiliAPIBaseURL ||
		getEPublicaBaseURL() != "https://transparencia.e-publica.net/epublica-portal" ||
		equiplanoAPIBaseURL != equiplanoDefaultAPIBaseURL ||
		getPublicsoftBaseURL() != defaultPublicsoftBaseURL ||
		getCentiAPIBaseURL() != defaultCentiAPIBaseURL ||
		getMemoryAPIBaseURL() != defaultMemoryAPIBaseURL ||
		getFacilBaseURL() != defaultFacilBaseURL
}

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
		fmt.Sprintf("https://%s.eloweb.net/portaltransparencia-api", compact),
		fmt.Sprintf("https://transparencia.%s.%s.gov.br/portaltransparencia-api", compact, ufLower),
		fmt.Sprintf("https://%s-%s.portaltp.com.br", compact, ufLower),
		fmt.Sprintf("https://%s.atende.net", compact),
		fmt.Sprintf("https://%s.sonner.com.br", compact),
		fmt.Sprintf("https://grp.%s.%s.gov.br", compact, ufLower),
		fmt.Sprintf("https://webapp1-%s.cidade360.cloud/dadosabertos", compact),
		fmt.Sprintf("https://www.gp.srv.br/transparencia_%s", compact),
		fmt.Sprintf("https://%s.cittaweb.com.br", compact),
	)
	if ufLower == "rn" {
		raw = append(raw,
			fmt.Sprintf("https://pm%srn.apitransparencia.topsolutionsrn.com.br", compact),
			fmt.Sprintf("https://%srn.apitransparencia.topsolutionsrn.com.br", compact),
		)
	}
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

	// 1b-bis. Catálogo nacional verificado de portais municipais (API aberta estritamente validada).
	if !hasActiveSaaSTestOverride() {
		if cat, ok := LookupPortalCatalogByIBGE(m.CodigoIBGE); ok && cat.PortalURL != "" && cat.Plataforma != "" {
			entNome := strings.TrimSpace(cat.EntidadeNome)
			if entNome == "" {
				entNome = fmt.Sprintf("%s (%s)", m.Nome, m.UF)
			}
			var codAdm int
			if cat.Plataforma == "cittaweb_api" || cat.Plataforma == "geosiap_api" || cat.Plataforma == "coplan_gp" || cat.Plataforma == "thema_apigw" {
				if n, err := strconv.Atoi(strings.TrimSpace(cat.TenantID)); err == nil && n > 0 {
					codAdm = n
				}
			}
			return &PortalDiscoveryResult{
				Municipio:           m,
				URLResolvida:        cat.PortalURL,
				Plataforma:          cat.Plataforma,
				CodAdministracao:    codAdm,
				TenantID:            cat.TenantID,
				EntidadeNome:        entNome,
				FonteDescoberta:     "portal_catalog",
				ServidoresSuportado: supportsServidoresForPlatform(cat.Plataforma),
				CandidatosTestados:  []string{cat.PortalURL},
				TempoMs:             time.Since(start).Milliseconds(),
				Observacoes:         fmt.Sprintf("Portal %s verificado no catálogo nacional integrado.", cat.Plataforma),
			}
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

	// 1d. Catálogo nacional de portais municipais HTML (sem API JSON/XML estruturada).
	if !hasActiveSaaSTestOverride() {
		if htmlCat, ok := LookupPortalHTMLByIBGE(m.CodigoIBGE); ok && htmlCat.PortalURL != "" {
			obs := strings.TrimSpace(htmlCat.Observacoes)
			if obs == "" {
				obs = "Portal municipal HTML sem API JSON/XML estruturada; consulta via extração HTML."
			}
			return &PortalDiscoveryResult{
				Municipio:           m,
				URLResolvida:        htmlCat.PortalURL,
				SitePrefeitura:      htmlCat.SitePrefeitura,
				Plataforma:          htmlCat.Plataforma,
				EntidadeNome:        fmt.Sprintf("%s (%s)", m.Nome, m.UF),
				FonteDescoberta:     "portal_html_catalog",
				ServidoresSuportado: true,
				CandidatosTestados:  []string{htmlCat.PortalURL},
				TempoMs:             time.Since(start).Milliseconds(),
				Observacoes:         obs,
			}
		}
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

	// 0) URLs SaaS dedicadas com padrão inequívoco (Betha Cloud, Memory ILAI, Fácil Transparência, TomeConta)
	if mHash := bethaHashRegex.FindStringSubmatch(base); len(mHash) >= 2 {
		det.URLResolvida = "https://transparencia.betha.cloud/#/" + mHash[1]
		det.Plataforma = "betha_cloud"
		det.TenantID = mHash[1]
		det.ServidoresSuportado = true
		det.Observacoes = "Portal Betha Cloud com API JSON pública anônima."
		return det, true
	}
	if t, ent := ExtractMemoryTenant(base); t != "" && ProbeMemory(ctx, client, t, ent) {
		det.Plataforma = "memory_openknowledge"
		if ent != "" {
			det.TenantID = t + "/" + ent
		} else {
			det.TenantID = t
		}
		det.ServidoresSuportado = true
		det.Observacoes = "Portal Memory OpenKnowledge com API REST pública."
		return det, true
	}
	if dbID := ExtractFacilDatabaseID(base); dbID != "" && ProbeFacil(ctx, client, base, dbID) {
		det.Plataforma = "facil_transparencia"
		det.TenantID = dbID
		det.ServidoresSuportado = true
		det.Observacoes = "Portal Fácil Transparência com API JSON pública."
		return det, true
	}
	if tcID := ExtractTomeContaID(base); tcID != "" && ProbeTomeConta(ctx, client, base, tcID) {
		det.Plataforma = "tomeconta_appbr"
		det.TenantID = tcID
		det.ServidoresSuportado = false
		det.Observacoes = "Portal TomeConta (portaltransparencia.app.br) com API JSON pública."
		return det, true
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
	if pu, pErr := url.Parse(base); pErr == nil && pu.Scheme != "" && pu.Host != "" {
		rootAPI := pu.Scheme + "://" + pu.Host + "/portaltransparencia-api"
		elotechCandidates = dedupeStrings(append(elotechCandidates, rootAPI))
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
	govCands := []string{base}
	lowBase := strings.ToLower(base)
	if strings.Contains(lowBase, "cidade360") || strings.Contains(lowBase, "/pronimtb") {
		if pu, pErr := url.Parse(base); pErr == nil && pu.Scheme != "" && pu.Host != "" {
			govCands = dedupeStrings(append(govCands, pu.Scheme+"://"+pu.Host+"/dadosabertos"))
		}
	}
	for _, gc := range govCands {
		if ent, ok := ProbeGovBRCidade360(ctx, client, gc); ok {
			det.URLResolvida = gc
			det.Plataforma = "govbr_cidade360"
			det.TenantID = ent
			det.EntidadeNome = ent
			det.ServidoresSuportado = true
			det.Observacoes = "Portal GovBR / Pronim Cidade360 com API OpenAPI pública."
			return det, true
		}
	}

	// 2j) Checa assinatura Coplan (/servlet/api_consulta?unidade_gestora)
	coplanCands := []string{base}
	if strings.Contains(lowBase, "gp.srv.br/") {
		if pu, pErr := url.Parse(base); pErr == nil && pu.Scheme != "" && pu.Host != "" {
			segs := strings.Split(strings.Trim(pu.Path, "/"), "/")
			if len(segs) > 0 && segs[0] != "" {
				coplanCands = dedupeStrings(append(coplanCands, pu.Scheme+"://"+pu.Host+"/"+segs[0]))
			}
		}
	}
	for _, cc := range coplanCands {
		if ugID, ugDesc, ok := ProbeCoplan(ctx, client, cc); ok {
			det.URLResolvida = cc
			det.Plataforma = "coplan_gp"
			det.CodAdministracao = ugID
			det.EntidadeNome = ugDesc
			det.Observacoes = "Portal Coplan com API JSON pública (api_consulta)."
			return det, true
		}
	}

	// 2k) Checa assinatura Thema API Gateway (/infra/apigw/transparencia/service/...)
	if adminCode, adminName, ok := ProbeThemaAPIGW(ctx, client, base); ok {
		det.Plataforma = "thema_apigw"
		det.CodAdministracao = adminCode
		det.EntidadeNome = adminName
		det.Observacoes = "Portal Thema API Gateway com endpoints JSON públicos."
		return det, true
	}

	isOverride := source == "override"
	// 2l) Conectores especializados por domínio ou OverrideURL
	if isOverride || strings.Contains(lowBase, "publicsoft") || publicsoftLegadoRegex.MatchString(base) {
		if hid, entName, ok := ProbePublicsoft(ctx, client, base, "", ""); ok {
			det.Plataforma = "publicsoft_api"
			det.TenantID = hid
			det.EntidadeNome = entName
			det.ServidoresSuportado = true
			det.Observacoes = "Portal Publicsoft com API REST v1 pública."
			return det, true
		}
	}
	if isOverride || strings.Contains(lowBase, "topsolutions") {
		if apiBase, ok := ProbeTopSolutions(ctx, client, base); ok {
			det.URLResolvida = apiBase
			det.Plataforma = "topsolutions_rn"
			det.ServidoresSuportado = true
			det.Observacoes = "Portal TopSolutionsRN com API REST pública."
			return det, true
		}
	}
	if isOverride || strings.Contains(lowBase, "cittaweb") {
		if apiBase, ugID, entName, ok := ProbeCittaWeb(ctx, client, base); ok {
			det.URLResolvida = apiBase
			det.Plataforma = "cittaweb_api"
			det.CodAdministracao = ugID
			det.EntidadeNome = entName
			det.ServidoresSuportado = true
			det.Observacoes = "Portal CittaWeb com API REST pública."
			return det, true
		}
	}
	if isOverride || strings.Contains(lowBase, "geosiap") {
		if apiBase, eid, entName, ok := ProbeGeoSiap(ctx, client, base); ok {
			det.URLResolvida = apiBase
			det.Plataforma = "geosiap_api"
			det.CodAdministracao = eid
			det.EntidadeNome = entName
			det.ServidoresSuportado = false
			det.Observacoes = "Portal GeoSiap com API REST pública."
			return det, true
		}
	}
	if isOverride || strings.Contains(lowBase, "multi24") {
		if wsBase, ok := ProbeMulti24h(ctx, client, base); ok {
			det.URLResolvida = wsBase
			det.Plataforma = "multi24h_xml"
			det.ServidoresSuportado = false
			det.Observacoes = "Portal Multi24h com WebService XML público."
			return det, true
		}
	}

	// 2m) Demais conectores municipais com endpoints estruturados padronizados
	if ProbeNucleoGov(ctx, client, base) {
		det.Plataforma = "nucleogov"
		det.ServidoresSuportado = true
		det.Observacoes = "Portal NúcleoGov com API REST pública."
		return det, true
	}
	if clienteSlug, ok := ProbeDiretorioDigital(ctx, client, base); ok {
		det.Plataforma = "diretoriodigital_sistp"
		det.TenantID = clienteSlug
		det.ServidoresSuportado = true
		det.Observacoes = "Portal Diretório Digital (SisTP) com API pública."
		return det, true
	}
	if portalBase, ok := ProbeInfBr(ctx, client, base); ok {
		det.URLResolvida = portalBase
		det.Plataforma = "infbr_netuse"
		det.ServidoresSuportado = true
		det.Observacoes = "Portal InfBr / Netuse com endpoint JSON público."
		return det, true
	}
	if ProbeAssesi(ctx, client, base) {
		det.Plataforma = "assesi_portal"
		det.ServidoresSuportado = true
		det.Observacoes = "Portal Assesi Transparência com API pública."
		return det, true
	}
	if entityCode, ok := ProbeAspec(ctx, client, base, ExtractAspecEntityCode(base)); ok {
		det.Plataforma = "aspec_gt"
		det.TenantID = entityCode
		det.ServidoresSuportado = false
		det.Observacoes = "Portal Aspec Governo Transparente com API JSON pública."
		return det, true
	}
	if ProbeSmarapd(ctx, client, base) {
		det.Plataforma = "smarapd_pai"
		det.ServidoresSuportado = true
		det.Observacoes = "Portal SMARAPD (PAI) com API REST pública."
		return det, true
	}
	if tid, ok := ProbeFatorSistemas(ctx, client, base); ok {
		det.Plataforma = "fator_sistemas"
		det.TenantID = tid
		det.ServidoresSuportado = true
		det.Observacoes = "Portal Fator Sistemas com exportação JSON pública."
		return det, true
	}
	if isOverride || strings.Contains(lowBase, "acessoainformacao.") {
		if pu, pErr := url.Parse(base); pErr == nil {
			parts := strings.Split(strings.ToLower(pu.Hostname()), ".")
			// acessoainformacao.<slug>.<uf>.gov.br
			if len(parts) >= 4 && parts[0] == "acessoainformacao" && len(parts[2]) == 2 {
				if resolved, ok := ProbeCenti(ctx, client, strings.ToUpper(parts[2]), parts[1]); ok {
					det.Plataforma = "centi_api"
					det.TenantID = resolved
					det.ServidoresSuportado = true
					det.Observacoes = "Portal Centi Soluções com API REST pública."
					return det, true
				}
			}
		}
	}

	// 3) Fallback HTML / portal genérico (detecção de vendor, links embutidos e anti-bot)
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
		if tcID := ExtractTomeContaID(resp.Request.URL.String()); tcID != "" && ProbeTomeConta(ctx, client, "https://portaltransparencia.app.br", tcID) {
			det.URLResolvida = "https://portaltransparencia.app.br"
			det.Plataforma = "tomeconta_appbr"
			det.TenantID = tcID
			det.ServidoresSuportado = false
			det.Observacoes = "Portal TomeConta localizado via redirecionamento."
			return det, true
		}
	}

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 65536))
	bodyStr := string(bodyBytes)
	bodyLow := strings.ToLower(bodyStr)

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
		if mHash := bethaHashRegex.FindStringSubmatch(bodyStr); len(mHash) >= 2 {
			det.URLResolvida = "https://transparencia.betha.cloud/#/" + mHash[1]
			det.Plataforma = "betha_cloud"
			det.TenantID = mHash[1]
			det.ServidoresSuportado = true
			det.Observacoes = "Portal Betha Cloud incorporado via link/iframe."
			return det, true
		}
		if t, ent := ExtractMemoryTenant(bodyStr); t != "" && ProbeMemory(ctx, client, t, ent) {
			det.Plataforma = "memory_openknowledge"
			if ent != "" {
				det.TenantID = t + "/" + ent
			} else {
				det.TenantID = t
			}
			det.ServidoresSuportado = true
			det.Observacoes = "Portal Memory OpenKnowledge incorporado via link/iframe."
			return det, true
		}
		if dbID := ExtractFacilDatabaseID(bodyStr); dbID != "" && ProbeFacil(ctx, client, "http://www.transparenciafacil.com.br", dbID) {
			det.URLResolvida = "http://www.transparenciafacil.com.br/" + dbID
			det.Plataforma = "facil_transparencia"
			det.TenantID = dbID
			det.ServidoresSuportado = true
			det.Observacoes = "Portal Fácil Transparência incorporado via link/iframe."
			return det, true
		}
		if tcID := ExtractTomeContaID(bodyStr); tcID != "" && ProbeTomeConta(ctx, client, "https://portaltransparencia.app.br", tcID) {
			det.URLResolvida = "https://portaltransparencia.app.br"
			det.Plataforma = "tomeconta_appbr"
			det.TenantID = tcID
			det.ServidoresSuportado = false
			det.Observacoes = "Portal TomeConta incorporado via link/iframe."
			return det, true
		}

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
			bodyStr := string(b)
			matches := hrefRegex.FindAllStringSubmatch(bodyStr, -1)
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
				if !strings.Contains(low, "transparencia") &&
					!strings.Contains(low, "portal-da-transparencia") &&
					!strings.Contains(low, "dadosabertos") &&
					!strings.Contains(low, "dados-abertos") &&
					!strings.Contains(low, "betha.cloud") &&
					!strings.Contains(low, "ilai.memory.com.br") &&
					!strings.Contains(low, "portalfacil.com.br") &&
					!strings.Contains(low, "portaltransparencia.app.br") &&
					!strings.Contains(low, "cittaweb.com.br") &&
					!strings.Contains(low, "geosiap.net.br") &&
					!strings.Contains(low, "publicsoft.com.br") &&
					!strings.Contains(low, "topsolutionsrn.com.br") &&
					!strings.Contains(low, "gp.srv.br") &&
					!strings.Contains(low, "cidade360.cloud") &&
					!strings.Contains(low, "eloweb.net") &&
					!strings.Contains(low, "acessoainformacao.") {
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
			for _, bm := range bethaHashRegex.FindAllStringSubmatch(bodyStr, -1) {
				if len(bm) >= 2 {
					mu.Lock()
					found = append(found, "https://transparencia.betha.cloud/#/"+bm[1])
					mu.Unlock()
				}
			}
			if t, ent := ExtractMemoryTenant(bodyStr); t != "" {
				mu.Lock()
				if ent != "" {
					found = append(found, "https://ilai.memory.com.br/#/entidades/login/"+t+"/"+ent)
				} else {
					found = append(found, "https://ilai.memory.com.br/#/entidades/login/"+t+"/1")
				}
				mu.Unlock()
			}
		}(site)
	}
	wg.Wait()
	return dedupeStrings(found)
}
