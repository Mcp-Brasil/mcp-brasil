package transparencia

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type portalAuditTarget struct {
	CodigoIBGE string `json:"codigo_ibge"`
	Nome       string `json:"nome"`
	UF         string `json:"uf"`
	Plataforma string `json:"plataforma"`
	PortalURL  string `json:"portal_url"`
	TenantID   string `json:"tenant_id,omitempty"`
}

type portalAuditModuleResult struct {
	Module     string `json:"module"`
	Status     string `json:"status"` // ok | waf_blocked | no_data | http_error
	Records    int    `json:"records"`
	Fonte      string `json:"fonte,omitempty"`
	Mensagem   string `json:"mensagem,omitempty"`
	DurationMs int64  `json:"duration_ms"`
}

type portalAuditEntry struct {
	Target          portalAuditTarget         `json:"target"`
	Status          string                    `json:"status"` // ok | waf_blocked | no_data | http_error
	TotalRecords    int                       `json:"total_records"`
	Modules         []portalAuditModuleResult `json:"modules"`
	TotalDurationMs int64                     `json:"total_duration_ms"`
}

type portalAuditCounts struct {
	Total      int `json:"total"`
	OK         int `json:"ok"`
	WAFBlocked int `json:"waf_blocked"`
	NoData     int `json:"no_data"`
	HTTPError  int `json:"http_error"`
}

type portalAuditReport struct {
	GeneratedAt  string                       `json:"generated_at"`
	TotalTargets int                          `json:"total_targets"`
	Modules      []string                     `json:"modules"`
	Summary      portalAuditCounts            `json:"summary"`
	ByPlatform   map[string]portalAuditCounts `json:"by_platform"`
	Results      []portalAuditEntry           `json:"results"`
}

type portalAuditConfig struct {
	Platforms         map[string]bool // empty = all
	SamplePerPlatform int             // 0 = all cataloged cities
	Modules           []string
	Concurrency       int
	Timeout           time.Duration
	ReportPath        string
}

func classifyAuditOutcome(disponivel bool, records int, msg string, err error) string {
	low := strings.ToLower(msg)
	if err != nil {
		errLow := strings.ToLower(err.Error())
		if strings.Contains(errLow, "anti-bot") || strings.Contains(errLow, "anti_bot") || strings.Contains(errLow, "captcha") || strings.Contains(errLow, "468") {
			return "waf_blocked"
		}
		return "http_error"
	}
	if disponivel && records > 0 {
		return "ok"
	}
	if strings.Contains(low, "anti-bot/waf") ||
		strings.Contains(low, "protegido por anti-bot") ||
		strings.Contains(low, "bloqueado por anti-bot") ||
		strings.Contains(low, "anti_bot_protegido") ||
		strings.Contains(low, "cloudflare") ||
		strings.Contains(low, "captcha") ||
		strings.Contains(low, "turnstile") ||
		strings.Contains(low, "safeline") {
		return "waf_blocked"
	}
	if strings.Contains(low, "retornou http 5") ||
		strings.Contains(low, "retornou http 4") ||
		strings.Contains(low, "http 5") ||
		strings.Contains(low, "timeout") ||
		strings.Contains(low, "connection refused") ||
		strings.Contains(low, "no such host") ||
		strings.Contains(low, "tls") {
		return "http_error"
	}
	return "no_data"
}

func selectAuditSamples(eng *Engine, cfg portalAuditConfig) []portalAuditTarget {
	byPlatform := make(map[string][]portalAuditTarget)

	for _, rec := range eng.ListPortaisCatalogMunicipios("") {
		if len(cfg.Platforms) > 0 && !cfg.Platforms[rec.Plataforma] {
			continue
		}
		byPlatform[rec.Plataforma] = append(byPlatform[rec.Plataforma], portalAuditTarget{
			CodigoIBGE: rec.CodigoIBGE,
			Nome:       rec.Nome,
			UF:         rec.UF,
			Plataforma: rec.Plataforma,
			PortalURL:  rec.PortalURL,
			TenantID:   rec.TenantID,
		})
	}
	for _, rec := range eng.ListPortaisHTMLMunicipios("") {
		if len(cfg.Platforms) > 0 && !cfg.Platforms[rec.Plataforma] {
			continue
		}
		byPlatform[rec.Plataforma] = append(byPlatform[rec.Plataforma], portalAuditTarget{
			CodigoIBGE: rec.CodigoIBGE,
			Nome:       rec.Nome,
			UF:         rec.UF,
			Plataforma: rec.Plataforma,
			PortalURL:  rec.PortalURL,
		})
	}

	platforms := make([]string, 0, len(byPlatform))
	for p := range byPlatform {
		platforms = append(platforms, p)
	}
	sort.Strings(platforms)

	var out []portalAuditTarget
	for _, p := range platforms {
		list := byPlatform[p]
		sort.Slice(list, func(i, j int) bool {
			return list[i].CodigoIBGE < list[j].CodigoIBGE
		})
		if cfg.SamplePerPlatform > 0 && len(list) > cfg.SamplePerPlatform {
			list = list[:cfg.SamplePerPlatform]
		}
		out = append(out, list...)
	}
	return out
}

func runPortalAuditSuite(ctx context.Context, hub *Hub, targets []portalAuditTarget, cfg portalAuditConfig) *portalAuditReport {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 8
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 12 * time.Second
	}
	if len(cfg.Modules) == 0 {
		cfg.Modules = []string{"servidores", "despesas", "receitas"}
	}

	results := make([]portalAuditEntry, len(targets))
	sem := make(chan struct{}, cfg.Concurrency)
	var wg sync.WaitGroup

	for idx, tgt := range targets {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, target portalAuditTarget) {
			defer wg.Done()
			defer func() { <-sem }()

			tStart := time.Now()
			m, err := hub.Engine.GetMunicipioByIBGE(target.CodigoIBGE)
			if err != nil {
				results[i] = portalAuditEntry{
					Target:          target,
					Status:          "http_error",
					TotalDurationMs: time.Since(tStart).Milliseconds(),
				}
				return
			}

			disc := &PortalDiscoveryResult{
				Municipio:       m,
				URLResolvida:    target.PortalURL,
				Plataforma:      target.Plataforma,
				TenantID:        target.TenantID,
				FonteDescoberta: "audit_target",
			}
			if mega, ok := LookupMegasoftByIBGE(m.CodigoIBGE); ok && target.Plataforma == "megasoft" {
				disc.CodAdministracao = mega.CodigoOrgao
			} else if target.TenantID != "" {
				if n, err := strconv.Atoi(strings.TrimSpace(target.TenantID)); err == nil && n > 0 {
					disc.CodAdministracao = n
				}
			}

			entry := portalAuditEntry{
				Target:  target,
				Modules: make([]portalAuditModuleResult, 0, len(cfg.Modules)),
			}

			opts := QueryOptions{
				UF:      m.UF,
				Limit:   5,
				Timeout: cfg.Timeout,
			}

			for _, mod := range cfg.Modules {
				mStart := time.Now()
				mCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
				var disp bool
				var recs int
				var fonte, msg string
				var qErr error

				switch mod {
				case "servidores":
					res, err := hub.queryServidoresWithDiscovery(mCtx, m, disc, opts)
					qErr = err
					if res != nil {
						disp, recs, fonte, msg = res.Disponivel, len(res.Servidores), res.Fonte, res.Mensagem
					}
				case "despesas":
					res, err := hub.queryDespesasWithDiscovery(mCtx, m, disc, opts)
					qErr = err
					if res != nil {
						disp, recs, fonte, msg = res.Disponivel, len(res.Despesas), res.Fonte, res.Mensagem
					}
				case "receitas":
					res, err := hub.queryReceitasWithDiscovery(mCtx, m, disc, opts)
					qErr = err
					if res != nil {
						disp, recs, fonte, msg = res.Disponivel, len(res.Receitas), res.Fonte, res.Mensagem
					}
				case "licitacoes":
					res, err := hub.queryLicitacoesWithDiscovery(mCtx, m, disc, opts)
					qErr = err
					if res != nil {
						disp, recs, fonte, msg = res.Disponivel, len(res.Licitacoes), res.Fonte, res.Mensagem
					}
				case "contratos":
					res, err := hub.queryContratosWithDiscovery(mCtx, m, disc, opts)
					qErr = err
					if res != nil {
						disp, recs, fonte, msg = res.Disponivel, len(res.Contratos), res.Fonte, res.Mensagem
					}
				case "diarias":
					res, err := hub.queryDiariasWithDiscovery(mCtx, m, disc, opts)
					qErr = err
					if res != nil {
						disp, recs, fonte, msg = res.Disponivel, len(res.Diarias), res.Fonte, res.Mensagem
					}
				case "obras":
					res, err := hub.queryObrasWithDiscovery(mCtx, m, disc, opts)
					qErr = err
					if res != nil {
						disp, recs, fonte, msg = res.Disponivel, len(res.Obras), res.Fonte, res.Mensagem
					}
				case "convenios":
					res, err := hub.queryConveniosWithDiscovery(mCtx, m, disc, opts)
					qErr = err
					if res != nil {
						disp, recs, fonte, msg = res.Disponivel, len(res.Convenios), res.Fonte, res.Mensagem
					}
				case "legislacao":
					res, err := hub.queryLegislacaoWithDiscovery(mCtx, m, disc, opts)
					qErr = err
					if res != nil {
						disp, recs, fonte, msg = res.Disponivel, len(res.Atos), res.Fonte, res.Mensagem
					}
				case "concursos":
					res, err := hub.queryConcursosWithDiscovery(mCtx, m, disc, opts)
					qErr = err
					if res != nil {
						disp, recs, fonte, msg = res.Disponivel, len(res.Aprovados), res.Fonte, res.Mensagem
					}
				}
				cancel()

				status := classifyAuditOutcome(disp, recs, msg, qErr)
				entry.Modules = append(entry.Modules, portalAuditModuleResult{
					Module:     mod,
					Status:     status,
					Records:    recs,
					Fonte:      fonte,
					Mensagem:   msg,
					DurationMs: time.Since(mStart).Milliseconds(),
				})
				entry.TotalRecords += recs
			}

			// Classifica status consolidado do portal: ok > waf_blocked > http_error > no_data
			entry.Status = "no_data"
			hasWAF, hasErr := false, false
			for _, mr := range entry.Modules {
				if mr.Status == "ok" {
					entry.Status = "ok"
					break
				}
				if mr.Status == "waf_blocked" {
					hasWAF = true
				}
				if mr.Status == "http_error" {
					hasErr = true
				}
			}
			if entry.Status != "ok" {
				if hasWAF {
					entry.Status = "waf_blocked"
				} else if hasErr {
					entry.Status = "http_error"
				}
			}
			entry.TotalDurationMs = time.Since(tStart).Milliseconds()
			results[i] = entry
		}(idx, tgt)
	}
	wg.Wait()

	report := &portalAuditReport{
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		TotalTargets: len(results),
		Modules:      cfg.Modules,
		ByPlatform:   make(map[string]portalAuditCounts),
		Results:      results,
	}

	for _, r := range results {
		report.Summary.Total++
		pc := report.ByPlatform[r.Target.Plataforma]
		pc.Total++
		switch r.Status {
		case "ok":
			report.Summary.OK++
			pc.OK++
		case "waf_blocked":
			report.Summary.WAFBlocked++
			pc.WAFBlocked++
		case "http_error":
			report.Summary.HTTPError++
			pc.HTTPError++
		default:
			report.Summary.NoData++
			pc.NoData++
		}
		report.ByPlatform[r.Target.Plataforma] = pc
	}

	if cfg.ReportPath != "" {
		_ = os.MkdirAll(filepath.Dir(cfg.ReportPath), 0o755)
		if data, err := json.MarshalIndent(report, "", "  "); err == nil {
			_ = os.WriteFile(cfg.ReportPath, data, 0o644)
		}
	}

	return report
}

func formatAuditReportMarkdown(r *portalAuditReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Municipal Transparency Portal Audit Report\n\n")
	fmt.Fprintf(&b, "- Generated: `%s`\n", r.GeneratedAt)
	fmt.Fprintf(&b, "- Targets tested: **%d** | Modules: `%s`\n", r.TotalTargets, strings.Join(r.Modules, ", "))
	fmt.Fprintf(&b, "- Summary: OK=%d | WAF Blocked=%d | No Data=%d | HTTP Error=%d\n\n",
		r.Summary.OK, r.Summary.WAFBlocked, r.Summary.NoData, r.Summary.HTTPError)
	fmt.Fprintf(&b, "| Platform | Total | OK | WAF | No Data | HTTP Error |\n")
	fmt.Fprintf(&b, "|---|---:|---:|---:|---:|---:|\n")

	platforms := make([]string, 0, len(r.ByPlatform))
	for p := range r.ByPlatform {
		platforms = append(platforms, p)
	}
	sort.Strings(platforms)
	for _, p := range platforms {
		c := r.ByPlatform[p]
		fmt.Fprintf(&b, "| `%s` | %d | %d | %d | %d | %d |\n", p, c.Total, c.OK, c.WAFBlocked, c.NoData, c.HTTPError)
	}
	return b.String()
}

// TestLiveAllCityPortalsAudit executa uma auditoria real contra portais de transparência municipais.
// É opt-in (pulado em `go test` normal) e ativado via `TRANSPARENCIA_LIVE_TEST=1`.
//
// Variáveis de ambiente suportadas:
//   - TRANSPARENCIA_LIVE_TEST=1                         (obrigatório para rodar)
//   - TRANSPARENCIA_LIVE_PLATFORMS=all|megasoft,...     (padrão: all — todas as 50 plataformas catalogadas)
//   - TRANSPARENCIA_LIVE_SAMPLE_PER_PLATFORM=1          (padrão: 1 por plataforma; 0 = TODOS os 4.789 portais)
//   - TRANSPARENCIA_LIVE_MODULES=servidores,despesas,receitas
//   - TRANSPARENCIA_LIVE_CONCURRENCY=8
//   - TRANSPARENCIA_LIVE_TIMEOUT_SEC=12
//   - TRANSPARENCIA_LIVE_REPORT_PATH=/tmp/portal_audit.json
func TestLiveAllCityPortalsAudit(t *testing.T) {
	if os.Getenv("TRANSPARENCIA_LIVE_TEST") != "1" {
		t.Skip("live municipal portal audit skipped (set TRANSPARENCIA_LIVE_TEST=1 to run)")
	}

	eng, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}
	hub := NewHub(eng)

	cfg := portalAuditConfig{
		SamplePerPlatform: 1,
		Modules:           []string{"servidores", "despesas", "receitas"},
		Concurrency:       8,
		Timeout:           12 * time.Second,
		ReportPath:        strings.TrimSpace(os.Getenv("TRANSPARENCIA_LIVE_REPORT_PATH")),
	}
	if rawPlats := strings.TrimSpace(os.Getenv("TRANSPARENCIA_LIVE_PLATFORMS")); rawPlats != "" && !strings.EqualFold(rawPlats, "all") {
		cfg.Platforms = make(map[string]bool)
		for _, p := range strings.Split(rawPlats, ",") {
			if p = strings.TrimSpace(p); p != "" {
				cfg.Platforms[p] = true
			}
		}
	}
	if rawSample := strings.TrimSpace(os.Getenv("TRANSPARENCIA_LIVE_SAMPLE_PER_PLATFORM")); rawSample != "" {
		if n, err := strconv.Atoi(rawSample); err == nil && n >= 0 {
			cfg.SamplePerPlatform = n
		}
	}
	if rawMods := strings.TrimSpace(os.Getenv("TRANSPARENCIA_LIVE_MODULES")); rawMods != "" {
		var mods []string
		for _, m := range strings.Split(rawMods, ",") {
			if m = strings.TrimSpace(m); m != "" {
				mods = append(mods, m)
			}
		}
		if len(mods) > 0 {
			cfg.Modules = mods
		}
	}
	if rawConc := strings.TrimSpace(os.Getenv("TRANSPARENCIA_LIVE_CONCURRENCY")); rawConc != "" {
		if n, err := strconv.Atoi(rawConc); err == nil && n > 0 {
			cfg.Concurrency = n
		}
	}
	if rawTimeout := strings.TrimSpace(os.Getenv("TRANSPARENCIA_LIVE_TIMEOUT_SEC")); rawTimeout != "" {
		if n, err := strconv.Atoi(rawTimeout); err == nil && n > 0 {
			cfg.Timeout = time.Duration(n) * time.Second
		}
	}

	targets := selectAuditSamples(eng, cfg)
	if len(targets) == 0 {
		t.Fatalf("no catalog targets matched filters: %+v", cfg)
	}

	report := runPortalAuditSuite(context.Background(), hub, targets, cfg)
	t.Log("\n" + formatAuditReportMarkdown(report))

	if report.Summary.OK == 0 {
		t.Errorf("live portal audit returned 0 OK portals out of %d tested", report.TotalTargets)
	}
}
