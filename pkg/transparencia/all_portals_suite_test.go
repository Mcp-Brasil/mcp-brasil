package transparencia

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// failingRoundTripper garante que nenhum teste offline realize chamadas de rede reais.
type failingRoundTripper struct {
	calls atomic.Int64
}

func (f *failingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	f.calls.Add(1)
	return nil, fmt.Errorf("unexpected outbound network call in offline test: %s %s", req.Method, req.URL.String())
}

func validateCatalogHTTPURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return errors.New("empty URL")
	}
	if strings.ContainsAny(raw, " \t\r\n") {
		return fmt.Errorf("URL contains whitespace: %q", raw)
	}
	low := strings.ToLower(raw)
	if strings.Contains(low, "http://http://") ||
		strings.Contains(low, "https://http://") ||
		strings.Contains(low, "https://https://") ||
		strings.Contains(low, "http://https://") {
		return fmt.Errorf("URL contains duplicated scheme: %q", raw)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("url.Parse error for %q: %w", raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid scheme %q in %q", u.Scheme, raw)
	}
	if u.Host == "" {
		return fmt.Errorf("missing host in %q", raw)
	}
	return nil
}

// knownCatalogAPIPlatforms lista todas as plataformas estruturadas suportadas pelo catálogo e pelo Hub.
var knownCatalogAPIPlatforms = map[string]bool{
	"agili_blue":             true,
	"aspec_gt":               true,
	"assesi_portal":          true,
	"betha_cloud":            true,
	"centi_api":              true,
	"cittaweb_api":           true,
	"coplan_gp":              true,
	"diretoriodigital_sistp": true,
	"el_portaltp":            true,
	"elotech_oxy":            true,
	"epublica_portal":        true,
	"equiplano_cloud":        true,
	"facil_transparencia":    true,
	"fator_sistemas":         true,
	"fiorilli_scpi":          true,
	"geosiap_api":            true,
	"govbr_cidade360":        true,
	"infbr_netuse":           true,
	"ipm_atende":             true,
	"megasoft":               true,
	"memory_openknowledge":   true,
	"multi24h_xml":           true,
	"nucleogov":              true,
	"publicsoft_api":         true,
	"quasar_publico":         true,
	"smarapd_pai":            true,
	"sonner_grp":             true,
	"thema_apigw":            true,
	"tomeconta_appbr":        true,
	"topsolutions_rn":        true,
}

func TestAll5571CitiesCatalogIntegrity(t *testing.T) {
	eng, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	estados := eng.ListEstados()
	if len(estados) != 27 {
		t.Fatalf("expected 27 states, got %d", len(estados))
	}
	ufSet := make(map[string]*Estado, len(estados))
	for _, est := range estados {
		ufSet[est.UF] = est
		if err := validateCatalogHTTPURL(est.PortalTransparencia); err != nil {
			t.Errorf("state %s invalid PortalTransparencia: %v", est.UF, err)
		}
		if err := validateCatalogHTTPURL(est.TribunalContasURL); err != nil {
			t.Errorf("state %s invalid TribunalContasURL: %v", est.UF, err)
		}
	}

	ibgeRe := regexp.MustCompile(`^\d{7}$`)
	if len(eng.municipios) != 5571 {
		t.Fatalf("expected 5571 municipalities, got %d", len(eng.municipios))
	}

	seenMuniIBGE := make(map[string]bool, len(eng.municipios))
	for _, m := range eng.municipios {
		if !ibgeRe.MatchString(m.CodigoIBGE) {
			t.Errorf("municipality %q has invalid IBGE %q", m.Nome, m.CodigoIBGE)
		}
		if seenMuniIBGE[m.CodigoIBGE] {
			t.Errorf("duplicate municipality IBGE %s", m.CodigoIBGE)
		}
		seenMuniIBGE[m.CodigoIBGE] = true
		if m.Nome == "" || m.Slug == "" {
			t.Errorf("municipality %s missing Nome or Slug", m.CodigoIBGE)
		}
		if _, ok := ufSet[m.UF]; !ok {
			t.Errorf("municipality %s (%s) has unknown UF %q", m.CodigoIBGE, m.Nome, m.UF)
		}

		dos, err := eng.ConnectMunicipio(m.CodigoIBGE, m.UF)
		if err != nil {
			t.Errorf("ConnectMunicipio(%s, %s) failed: %v", m.CodigoIBGE, m.UF, err)
			continue
		}
		for label, u := range map[string]string{
			"PortalTransparenciaPrefeitura": dos.PortalTransparenciaPrefeitura,
			"PortalPrefeitura":              dos.PortalPrefeitura,
			"TribunalContasURL":             dos.TribunalContasURL,
			"SiconfiURL":                    dos.SiconfiURL,
			"PNCPURL":                       dos.PNCPURL,
		} {
			if err := validateCatalogHTTPURL(u); err != nil {
				t.Errorf("municipality %s (%s/%s) invalid %s: %v", m.CodigoIBGE, m.Nome, m.UF, label, err)
			}
		}
		if dos.TribunalContasCompetente == "" {
			t.Errorf("municipality %s missing TribunalContasCompetente", m.CodigoIBGE)
		}

		cands := CandidatePortalURLs(m)
		if len(cands) == 0 {
			t.Errorf("municipality %s produced 0 CandidatePortalURLs", m.CodigoIBGE)
		}
		for _, cu := range cands {
			if err := validateCatalogHTTPURL(cu); err != nil {
				t.Errorf("municipality %s invalid CandidatePortalURL: %v", m.CodigoIBGE, err)
			}
		}
		prefCands := CandidatePrefeituraURLs(m)
		if len(prefCands) == 0 {
			t.Errorf("municipality %s produced 0 CandidatePrefeituraURLs", m.CodigoIBGE)
		}
		for _, pu := range prefCands {
			if err := validateCatalogHTTPURL(pu); err != nil {
				t.Errorf("municipality %s invalid CandidatePrefeituraURL: %v", m.CodigoIBGE, err)
			}
		}
	}

	// Validar catálogo de APIs estruturadas (portais_municipais.csv)
	apiCatalog := eng.ListPortaisCatalogMunicipios("")
	if len(apiCatalog) < 2100 {
		t.Fatalf("expected >= 2100 API catalog portals, got %d", len(apiCatalog))
	}
	apiByIBGE := make(map[string]PortalCatalogMunicipio, len(apiCatalog))
	apiPlatformCounts := make(map[string]int)
	bethaTotal := 0
	bethaWithPadding := 0
	for _, rec := range apiCatalog {
		m, err := eng.GetMunicipioByIBGE(rec.CodigoIBGE)
		if err != nil {
			t.Errorf("API catalog IBGE %s (%s/%s) not in municipios.csv", rec.CodigoIBGE, rec.Nome, rec.UF)
			continue
		}
		if m.UF != rec.UF {
			t.Errorf("API catalog IBGE %s UF mismatch: catalog=%s municipios=%s", rec.CodigoIBGE, rec.UF, m.UF)
		}
		if _, dup := apiByIBGE[rec.CodigoIBGE]; dup {
			t.Errorf("duplicate API catalog IBGE %s", rec.CodigoIBGE)
		}
		apiByIBGE[rec.CodigoIBGE] = rec

		if err := validateCatalogHTTPURL(rec.PortalURL); err != nil {
			t.Errorf("API catalog %s (%s/%s) invalid PortalURL: %v", rec.CodigoIBGE, rec.Nome, rec.UF, err)
		}
		if !knownCatalogAPIPlatforms[rec.Plataforma] {
			t.Errorf("API catalog %s has unknown platform %q", rec.CodigoIBGE, rec.Plataforma)
		}
		if IsHTMLPortalPlatform(rec.Plataforma) {
			t.Errorf("API catalog %s platform %q mistakenly classified as HTML", rec.CodigoIBGE, rec.Plataforma)
		}
		apiPlatformCounts[rec.Plataforma]++

		if rec.Plataforma == "betha_cloud" {
			bethaTotal++
			match := bethaHashRegex.FindStringSubmatch(rec.PortalURL)
			if len(match) < 2 {
				t.Errorf("bethaHashRegex failed to extract hash from betha_cloud URL %q (%s)", rec.PortalURL, rec.CodigoIBGE)
			} else if strings.HasSuffix(rec.PortalURL, "=") && !strings.HasSuffix(match[1], "=") {
				t.Errorf("bethaHashRegex truncated Base64 '=' padding in %q (got %q)", rec.PortalURL, match[1])
			} else if strings.HasSuffix(match[1], "=") {
				bethaWithPadding++
			}
		}
	}
	if bethaTotal == 0 {
		t.Fatal("expected betha_cloud entries in API catalog")
	}
	if len(apiPlatformCounts) < 28 {
		t.Errorf("expected at least 28 API platforms in catalog, got %d: %v", len(apiPlatformCounts), apiPlatformCounts)
	}

	// Validar catálogo HTML (portais_html_municipais.csv)
	htmlCatalog := eng.ListPortaisHTMLMunicipios("")
	if len(htmlCatalog) < 2600 {
		t.Fatalf("expected >= 2600 HTML catalog portals, got %d", len(htmlCatalog))
	}
	htmlByIBGE := make(map[string]PortalHTMLMunicipio, len(htmlCatalog))
	htmlPlatformCounts := make(map[string]int)
	for _, rec := range htmlCatalog {
		m, err := eng.GetMunicipioByIBGE(rec.CodigoIBGE)
		if err != nil {
			t.Errorf("HTML catalog IBGE %s (%s/%s) not in municipios.csv", rec.CodigoIBGE, rec.Nome, rec.UF)
			continue
		}
		if m.UF != rec.UF {
			t.Errorf("HTML catalog IBGE %s UF mismatch: catalog=%s municipios=%s", rec.CodigoIBGE, rec.UF, m.UF)
		}
		if _, dup := htmlByIBGE[rec.CodigoIBGE]; dup {
			t.Errorf("duplicate HTML catalog IBGE %s", rec.CodigoIBGE)
		}
		htmlByIBGE[rec.CodigoIBGE] = rec

		if _, overlap := apiByIBGE[rec.CodigoIBGE]; overlap {
			t.Errorf("IBGE %s (%s/%s) appears in both API and HTML catalogs", rec.CodigoIBGE, rec.Nome, rec.UF)
		}
		if err := validateCatalogHTTPURL(rec.PortalURL); err != nil {
			t.Errorf("HTML catalog %s (%s/%s) invalid PortalURL: %v", rec.CodigoIBGE, rec.Nome, rec.UF, err)
		}
		if rec.SitePrefeitura != "" {
			if err := validateCatalogHTTPURL(rec.SitePrefeitura); err != nil {
				t.Errorf("HTML catalog %s (%s/%s) invalid SitePrefeitura: %v", rec.CodigoIBGE, rec.Nome, rec.UF, err)
			}
		}
		if !IsHTMLPortalPlatform(rec.Plataforma) {
			t.Errorf("HTML catalog %s platform %q not recognized by IsHTMLPortalPlatform", rec.CodigoIBGE, rec.Plataforma)
		}
		htmlPlatformCounts[rec.Plataforma]++
	}
	if len(htmlPlatformCounts) < 22 {
		t.Errorf("expected at least 22 HTML platform families in catalog, got %d: %v", len(htmlPlatformCounts), htmlPlatformCounts)
	}

	// Validar catálogo Megasoft (megasoft_municipios.csv)
	megaCatalog := eng.ListMegasoftMunicipios("")
	if len(megaCatalog) != 268 {
		t.Fatalf("expected 268 Megasoft cities, got %d", len(megaCatalog))
	}
	seenMega := make(map[string]bool, len(megaCatalog))
	for _, rec := range megaCatalog {
		m, err := eng.GetMunicipioByIBGE(rec.CodigoIBGE)
		if err != nil {
			t.Errorf("Megasoft IBGE %s not in municipios.csv", rec.CodigoIBGE)
			continue
		}
		if m.UF != rec.UF {
			t.Errorf("Megasoft IBGE %s UF mismatch: %s vs %s", rec.CodigoIBGE, rec.UF, m.UF)
		}
		if seenMega[rec.CodigoIBGE] {
			t.Errorf("duplicate Megasoft IBGE %s", rec.CodigoIBGE)
		}
		seenMega[rec.CodigoIBGE] = true
		if _, inAPI := apiByIBGE[rec.CodigoIBGE]; !inAPI {
			t.Errorf("Megasoft IBGE %s (%s/%s) missing from portais_municipais.csv", rec.CodigoIBGE, rec.Nome, rec.UF)
		}
		if err := validateCatalogHTTPURL(rec.PortalURL); err != nil {
			t.Errorf("Megasoft %s invalid PortalURL: %v", rec.CodigoIBGE, err)
		}
		if rec.PortalCamara != "" {
			if err := validateCatalogHTTPURL(rec.PortalCamara); err != nil {
				t.Errorf("Megasoft %s invalid PortalCamara: %v", rec.CodigoIBGE, err)
			}
		}
		if rec.SitePrefeitura != "" {
			if err := validateCatalogHTTPURL(rec.SitePrefeitura); err != nil {
				t.Errorf("Megasoft %s invalid SitePrefeitura: %v", rec.CodigoIBGE, err)
			}
		}
	}

	if len(apiCatalog)+len(htmlCatalog) != 5571 {
		t.Fatalf("expected 100%% catalog coverage (5571/5571 cities), got %d (%d API + %d HTML)",
			len(apiCatalog)+len(htmlCatalog), len(apiCatalog), len(htmlCatalog))
	}
	t.Logf("Validated 5571 cities (100%% coverage), 27 UFs, %d API portals (%d platforms, %d betha_cloud with %d padded hashes), %d HTML portals (%d families), %d Megasoft portals",
		len(apiCatalog), len(apiPlatformCounts), bethaTotal, bethaWithPadding, len(htmlCatalog), len(htmlPlatformCounts), len(megaCatalog))
}

func TestAll4789CatalogedPortalsOfflineDiscovery(t *testing.T) {
	eng, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	rt := &failingRoundTripper{}
	offlineClient := &http.Client{
		Transport: rt,
		Timeout:   2 * time.Second,
	}

	ctx := context.Background()
	start := time.Now()

	// 1. Todas as cidades do catálogo de APIs estruturadas
	apiCatalog := eng.ListPortaisCatalogMunicipios("")
	seenAPIPlatforms := make(map[string]int)
	for _, rec := range apiCatalog {
		m, err := eng.GetMunicipioByIBGE(rec.CodigoIBGE)
		if err != nil {
			t.Fatalf("missing municipality %s: %v", rec.CodigoIBGE, err)
		}
		disc := DiscoverPortal(ctx, m, DiscoverOptions{
			HTTPClient: offlineClient,
			Timeout:    2 * time.Second,
		})
		if disc.Plataforma != rec.Plataforma {
			t.Errorf("IBGE %s (%s/%s): expected platform %q, got %q (fonte=%s)",
				rec.CodigoIBGE, rec.Nome, rec.UF, rec.Plataforma, disc.Plataforma, disc.FonteDescoberta)
		}
		if disc.URLResolvida == "" {
			t.Errorf("IBGE %s (%s/%s): empty URLResolvida", rec.CodigoIBGE, rec.Nome, rec.UF)
		}
		if disc.FonteDescoberta != "portal_catalog" && disc.FonteDescoberta != "megasoft_catalog" {
			t.Errorf("IBGE %s (%s/%s): unexpected FonteDescoberta %q", rec.CodigoIBGE, rec.Nome, rec.UF, disc.FonteDescoberta)
		}
		seenAPIPlatforms[disc.Plataforma]++
	}

	// 2. Todas as cidades do catálogo HTML (regressão: NÃO pode disparar probes SaaS antes do catálogo!)
	htmlCatalog := eng.ListPortaisHTMLMunicipios("")
	seenHTMLPlatforms := make(map[string]int)
	for _, rec := range htmlCatalog {
		m, err := eng.GetMunicipioByIBGE(rec.CodigoIBGE)
		if err != nil {
			t.Fatalf("missing municipality %s: %v", rec.CodigoIBGE, err)
		}
		disc := DiscoverPortal(ctx, m, DiscoverOptions{
			HTTPClient: offlineClient,
			Timeout:    2 * time.Second,
		})
		if disc.Plataforma != rec.Plataforma {
			t.Errorf("HTML IBGE %s (%s/%s): expected platform %q, got %q (fonte=%s)",
				rec.CodigoIBGE, rec.Nome, rec.UF, rec.Plataforma, disc.Plataforma, disc.FonteDescoberta)
		}
		if !IsHTMLPortalPlatform(disc.Plataforma) {
			t.Errorf("HTML IBGE %s: IsHTMLPortalPlatform(%q) == false", rec.CodigoIBGE, disc.Plataforma)
		}
		if disc.URLResolvida != rec.PortalURL {
			t.Errorf("HTML IBGE %s: expected URLResolvida=%q, got %q", rec.CodigoIBGE, rec.PortalURL, disc.URLResolvida)
		}
		if disc.FonteDescoberta != "portal_html_catalog" {
			t.Errorf("HTML IBGE %s: expected FonteDescoberta=portal_html_catalog, got %q", rec.CodigoIBGE, disc.FonteDescoberta)
		}
		seenHTMLPlatforms[disc.Plataforma]++
	}

	elapsed := time.Since(start)
	if calls := rt.calls.Load(); calls != 0 {
		t.Fatalf("expected 0 outbound HTTP calls for all %d cataloged cities, got %d", len(apiCatalog)+len(htmlCatalog), calls)
	}
	if len(apiCatalog)+len(htmlCatalog) != 5571 {
		t.Fatalf("expected all 5571 cities resolved offline, got %d", len(apiCatalog)+len(htmlCatalog))
	}
	if len(seenAPIPlatforms) < 28 {
		t.Errorf("expected >= 28 API platforms resolved offline, got %d", len(seenAPIPlatforms))
	}
	if len(seenHTMLPlatforms) < 22 {
		t.Errorf("expected >= 22 HTML platforms resolved offline, got %d", len(seenHTMLPlatforms))
	}
	t.Logf("Resolved ALL %d cataloged city portals (%d API + %d HTML) offline in %v with 0 HTTP calls",
		len(apiCatalog)+len(htmlCatalog), len(apiCatalog), len(htmlCatalog), elapsed)
}

func TestLiveAuditRunnerWithMockPortals(t *testing.T) {
	eng, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/servidores":
			fmt.Fprint(w, `<html><body><table>
				<tr><th>Nome</th><th>Cargo</th><th>Salário Bruto</th><th>Salário Líquido</th></tr>
				<tr><td>MARIA AUDIT</td><td>PROFESSORA</td><td>R$ 5.200,00</td><td>R$ 4.300,00</td></tr>
			</table></body></html>`)
		default:
			fmt.Fprint(w, `<html><body><a href="/servidores">Servidores e Folha</a></body></html>`)
		}
	}))
	defer okSrv.Close()

	wafSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `<html><body>Just a moment... cf-turnstile Checking your browser</body></html>`)
	}))
	defer wafSrv.Close()

	emptySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<html><body><h1>Portal Institucional</h1><p>Em manutenção.</p></body></html>`)
	}))
	defer emptySrv.Close()

	errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal error")
	}))
	defer errSrv.Close()

	targets := []portalAuditTarget{
		{CodigoIBGE: "3550308", Nome: "São Paulo", UF: "SP", Plataforma: "html_portal", PortalURL: okSrv.URL},
		{CodigoIBGE: "3304557", Nome: "Rio de Janeiro", UF: "RJ", Plataforma: "govbr_html", PortalURL: wafSrv.URL},
		{CodigoIBGE: "3106200", Nome: "Belo Horizonte", UF: "MG", Plataforma: "prefeitura_html", PortalURL: emptySrv.URL},
		{CodigoIBGE: "4106902", Nome: "Curitiba", UF: "PR", Plataforma: "betha_html", PortalURL: errSrv.URL},
	}

	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "audit_report.json")

	hub := NewHub(eng)
	hub.HTTPClient = okSrv.Client()
	cfg := portalAuditConfig{
		Modules:     []string{"servidores"},
		Concurrency: 4,
		Timeout:     4 * time.Second,
		ReportPath:  reportPath,
	}

	report := runPortalAuditSuite(context.Background(), hub, targets, cfg)
	if report.TotalTargets != 4 {
		t.Fatalf("expected 4 targets, got %d", report.TotalTargets)
	}
	if report.Summary.OK != 1 {
		t.Errorf("expected 1 OK target, got %d (results=%+v)", report.Summary.OK, report.Results)
	}
	if report.Summary.WAFBlocked != 1 {
		t.Errorf("expected 1 WAFBlocked target, got %d (results=%+v)", report.Summary.WAFBlocked, report.Results)
	}
	if report.Summary.NoData+report.Summary.HTTPError != 2 {
		t.Errorf("expected 2 no_data/http_error targets, got no_data=%d http_error=%d", report.Summary.NoData, report.Summary.HTTPError)
	}

	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("expected JSON report file at %s: %v", reportPath, err)
	}
	var decoded portalAuditReport
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("invalid JSON report: %v", err)
	}
	if decoded.TotalTargets != 4 {
		t.Errorf("decoded report TotalTargets=%d, want 4", decoded.TotalTargets)
	}

	md := formatAuditReportMarkdown(report)
	if !strings.Contains(md, "html_portal") || !strings.Contains(md, "WAF") {
		t.Errorf("unexpected markdown summary:\n%s", md)
	}
}
