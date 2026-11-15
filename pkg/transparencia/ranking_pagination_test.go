package transparencia_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Mcp-Brasil/mcp-brasil/pkg/transparencia"
)

func TestHubRankingServidoresMultiPageAndDedup(t *testing.T) {
	eng, err := transparencia.NewEngine()
	if err != nil {
		t.Fatalf("NewEngine erro: %v", err)
	}

	var requestedPages []int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/publico/versao":
			_, _ = w.Write([]byte(`"v26.10.30"`))
		case "/publico/buscarEntidades":
			_, _ = w.Write([]byte(`[{"codAdministracao":2,"desAdministracao":"PREFEITURA MUNICIPAL"}]`))
		case "/publico/servidor":
			pagina, _ := strconv.Atoi(r.URL.Query().Get("pagina"))
			elem, _ := strconv.Atoi(r.URL.Query().Get("elementosPorPagina"))
			if elem > 200 {
				t.Errorf("elementosPorPagina deveria respeitar cap 200, obteve %d", elem)
			}
			requestedPages = append(requestedPages, pagina)

			// Gera 220 registros em 2 páginas (elementosPorPagina=200):
			// - Na página 1: salários baixos + duplicata mensal (01/2025, 20000) do TOP EARNER
			// - Na página 2: TOP EARNER em 02/2025 com 25000 + dois homônimos distintos "JOAO DA SILVA"
			var rows []map[string]any
			if pagina == 1 {
				rows = append(rows, map[string]any{
					"nomServidor":            "TOP EARNER",
					"numMatricula":           "MAT-01",
					"numcpf":                 "11122233344",
					"nomCargo":               "PREFEITO",
					"vlrSalarioBruto":        20000.0,
					"vlrSalarioLiquido":      15000.0,
					"mesReferenciaFormatado": "01/2025",
				})
				for i := 2; i <= 200; i++ {
					rows = append(rows, map[string]any{
						"nomServidor":            fmt.Sprintf("SERVIDOR %03d", i),
						"numMatricula":           fmt.Sprintf("MAT-%03d", i),
						"numcpf":                 "99988877766",
						"nomCargo":               "AUXILIAR",
						"vlrSalarioBruto":        float64(2000 + i),
						"vlrSalarioLiquido":      float64(1700 + i),
						"mesReferenciaFormatado": "02/2025",
					})
				}
			} else if pagina == 2 {
				rows = append(rows,
					map[string]any{
						"nomServidor":            "TOP EARNER",
						"numMatricula":           "MAT-01",
						"numcpf":                 "11122233344",
						"nomCargo":               "PREFEITO",
						"vlrSalarioBruto":        25000.0,
						"vlrSalarioLiquido":      18500.0,
						"mesReferenciaFormatado": "02/2025",
					},
					map[string]any{
						"nomServidor":            "JOAO DA SILVA",
						"numMatricula":           "MAT-02",
						"numcpf":                 "22233344455",
						"nomCargo":               "MEDICO",
						"vlrSalarioBruto":        22000.0,
						"vlrSalarioLiquido":      16500.0,
						"mesReferenciaFormatado": "02/2025",
					},
					map[string]any{
						"nomServidor":            "JOAO DA SILVA",
						"numMatricula":           "MAT-03",
						"numcpf":                 "33344455566",
						"nomCargo":               "PROCURADOR",
						"vlrSalarioBruto":        21000.0,
						"vlrSalarioLiquido":      16000.0,
						"mesReferenciaFormatado": "02/2025",
					},
				)
				for i := 4; i <= 20; i++ {
					rows = append(rows, map[string]any{
						"nomServidor":            fmt.Sprintf("EXTRA %02d", i),
						"numMatricula":           fmt.Sprintf("EXT-%02d", i),
						"numcpf":                 "44455566677",
						"nomCargo":               "ANALISTA",
						"vlrSalarioBruto":        3500.0,
						"vlrSalarioLiquido":      2900.0,
						"mesReferenciaFormatado": "02/2025",
					})
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"content":       rows,
				"totalElements": 220,
				"totalPages":    2,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	hub := &transparencia.Hub{
		Engine:     eng,
		HTTPClient: srv.Client(),
		TCE:        transparencia.NewTCEClient(srv.Client()),
		Siconfi:    transparencia.NewSiconfiClient(),
		PNCP:       transparencia.NewPNCPClient(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("ordem_salario_desc_multi_page_dedup_and_slice", func(t *testing.T) {
		requestedPages = nil
		res, err := hub.QueryServidores(ctx, "Serra da Saudade", transparencia.QueryOptions{
			UF:               "MG",
			OverrideURL:      srv.URL,
			Ano:              2025,
			Limit:            3,
			OrdemSalarioDesc: true,
		})
		if err != nil {
			t.Fatalf("QueryServidores erro: %v", err)
		}
		if len(requestedPages) < 2 || requestedPages[0] != 1 || requestedPages[1] != 2 {
			t.Fatalf("esperava paginação completa (páginas 1 e 2) antes de ordenar, obteve %v", requestedPages)
		}
		if res.TotalRegistros != 220 {
			t.Errorf("TotalRegistros deveria preservar 220, obteve %d", res.TotalRegistros)
		}
		if len(res.Servidores) != 3 {
			t.Fatalf("esperava 3 servidores fatiados para Limit=3, obteve %d", len(res.Servidores))
		}
		if res.Servidores[0].Nome != "TOP EARNER" || res.Servidores[0].SalarioBruto != 25000.0 {
			t.Errorf("primeiro colocado deveria manter maior salário (25000) após dedup, obteve %+v", res.Servidores[0])
		}
		if res.Servidores[1].Nome != "JOAO DA SILVA" || res.Servidores[1].Matricula != "MAT-02" || res.Servidores[1].SalarioBruto != 22000.0 {
			t.Errorf("segundo colocado inesperado: %+v", res.Servidores[1])
		}
		if res.Servidores[2].Nome != "JOAO DA SILVA" || res.Servidores[2].Matricula != "MAT-03" || res.Servidores[2].SalarioBruto != 21000.0 {
			t.Errorf("terceiro colocado (homônimo com matrícula distinta) não foi preservado: %+v", res.Servidores[2])
		}
	})

	t.Run("mes_especifico_nao_deduplica", func(t *testing.T) {
		res, err := hub.QueryServidores(ctx, "Serra da Saudade", transparencia.QueryOptions{
			UF:               "MG",
			OverrideURL:      srv.URL,
			Ano:              2025,
			Mes:              1,
			Limit:            10,
			OrdemSalarioDesc: true,
		})
		if err != nil {
			t.Fatalf("QueryServidores erro: %v", err)
		}
		topCount := 0
		for _, s := range res.Servidores {
			if s.Nome == "TOP EARNER" && s.Matricula == "MAT-01" {
				topCount++
			}
		}
		if topCount != 2 {
			t.Errorf("com Mes>0 não deve deduplicar registros repetidos; esperava 2 ocorrências, obteve %d", topCount)
		}
	})
}

func TestHubRankingDespesasValorDesc(t *testing.T) {
	eng, err := transparencia.NewEngine()
	if err != nil {
		t.Fatalf("NewEngine erro: %v", err)
	}

	var pages []int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/publico/versao":
			_, _ = w.Write([]byte(`"v26.10.30"`))
		case "/publico/buscarEntidades":
			_, _ = w.Write([]byte(`[{"codAdministracao":2,"desAdministracao":"PREFEITURA"}]`))
		case "/publico/despesaDetalhada":
			pagina, _ := strconv.Atoi(r.URL.Query().Get("pagina"))
			pages = append(pages, pagina)
			var rows []map[string]any
			if pagina == 1 {
				for i := 1; i <= 200; i++ {
					rows = append(rows, map[string]any{
						"numEmpenho":    fmt.Sprintf("EMP-%03d", i),
						"numAnoEmpenho": 2025,
						"nomFornecedor": fmt.Sprintf("FORNECEDOR %03d", i),
						"vlrEmpenhado":  float64(1000 + i),
						"vlrLiquidado":  float64(900 + i),
						"vlrPago":       float64(800 + i),
					})
				}
			} else if pagina == 2 {
				rows = append(rows,
					map[string]any{
						"numEmpenho":    "EMP-EMPENHO-ONLY",
						"numAnoEmpenho": 2025,
						"nomFornecedor": "SOMENTE EMPENHADO",
						"vlrEmpenhado":  80000.0,
						"vlrLiquidado":  0.0,
						"vlrPago":       0.0,
					},
					map[string]any{
						"numEmpenho":    "EMP-LIQ-ONLY",
						"numAnoEmpenho": 2025,
						"nomFornecedor": "SOMENTE LIQUIDADO",
						"vlrEmpenhado":  95000.0,
						"vlrLiquidado":  90000.0,
						"vlrPago":       0.0,
					},
					map[string]any{
						"numEmpenho":    "EMP-PAGO-TOP",
						"numAnoEmpenho": 2025,
						"nomFornecedor": "MAIOR PAGO PAGINA 2",
						"vlrEmpenhado":  120000.0,
						"vlrLiquidado":  110000.0,
						"vlrPago":       105000.0,
					},
				)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"content":       rows,
				"totalElements": 203,
				"totalPages":    2,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	hub := &transparencia.Hub{
		Engine:     eng,
		HTTPClient: srv.Client(),
		TCE:        transparencia.NewTCEClient(srv.Client()),
		Siconfi:    transparencia.NewSiconfiClient(),
		PNCP:       transparencia.NewPNCPClient(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := hub.QueryDespesas(ctx, "Serra da Saudade", transparencia.QueryOptions{
		UF:             "MG",
		OverrideURL:    srv.URL,
		Ano:            2025,
		Limit:          3,
		OrdemValorDesc: true,
	})
	if err != nil {
		t.Fatalf("QueryDespesas erro: %v", err)
	}
	if len(pages) < 2 || pages[0] != 1 || pages[1] != 2 {
		t.Fatalf("esperava busca em múltiplas páginas para despesas, obteve %v", pages)
	}
	if res.TotalRegistros != 203 {
		t.Errorf("TotalRegistros esperado 203, obteve %d", res.TotalRegistros)
	}
	if len(res.Despesas) != 3 {
		t.Fatalf("esperava 3 despesas após slice, obteve %d", len(res.Despesas))
	}
	if res.Despesas[0].Fornecedor != "MAIOR PAGO PAGINA 2" ||
		res.Despesas[1].Fornecedor != "SOMENTE LIQUIDADO" ||
		res.Despesas[2].Fornecedor != "SOMENTE EMPENHADO" {
		t.Errorf("ordem ValorPago -> ValorLiquidado -> ValorEmpenhado incorreta: %+v", res.Despesas)
	}
}

func TestQuasarPaginationAndMonthWindow(t *testing.T) {
	var (
		gotInicio string
		gotFim    string
		gotPages  []int
		gotSizes  []int
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/publico/buscarEntidades":
			_, _ = w.Write([]byte(`[{"codAdministracao":2,"desAdministracao":"PREFEITURA"}]`))
		case "/publico/servidor":
			q := r.URL.Query()
			gotInicio = q.Get("datAberturaInicio")
			gotFim = q.Get("datAberturaFim")
			pg, _ := strconv.Atoi(q.Get("pagina"))
			sz, _ := strconv.Atoi(q.Get("elementosPorPagina"))
			gotPages = append(gotPages, pg)
			gotSizes = append(gotSizes, sz)

			count := 200
			if pg == 2 {
				count = 60
			}
			rows := make([]map[string]any, 0, count)
			for i := 0; i < count; i++ {
				rows = append(rows, map[string]any{
					"nomServidor":     fmt.Sprintf("SRV-%d-%d", pg, i),
					"numMatricula":    fmt.Sprintf("M-%d-%d", pg, i),
					"vlrSalarioBruto": 3000.0,
				})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"content":       rows,
				"totalElements": 260,
				"totalPages":    2,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	m := &transparencia.Municipio{CodigoIBGE: "3166600", Nome: "Serra da Saudade", UF: "MG"}
	res, err := transparencia.QueryQuasarServidores(context.Background(), srv.Client(), m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Ano:     2024,
		Mes:     2,
		Limit:   250,
	})
	if err != nil {
		t.Fatalf("QueryQuasarServidores erro: %v", err)
	}
	if gotInicio != "2024-02-01" || gotFim != "2024-02-29" {
		t.Errorf("janela de fevereiro bissexto 2024 incorreta: %s .. %s", gotInicio, gotFim)
	}
	if len(gotPages) != 2 || gotPages[0] != 1 || gotPages[1] != 2 {
		t.Errorf("páginas esperadas [1 2], obteve %v", gotPages)
	}
	if gotSizes[0] != 200 || gotSizes[1] != 200 {
		t.Errorf("elementosPorPagina esperado 200, obteve %v", gotSizes)
	}
	if len(res.Servidores) != 250 || res.TotalRegistros != 260 {
		t.Errorf("esperava 250 servidores e total 260, obteve %d e %d", len(res.Servidores), res.TotalRegistros)
	}
}

func TestConnectorPaginationMonthAndFilterRegressions(t *testing.T) {
	m := &transparencia.Municipio{CodigoIBGE: "3166600", Nome: "Serra da Saudade", UF: "MG"}

	t.Run("megasoft_filter_and_pagination", func(t *testing.T) {
		var pages []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/api/configuracao/carregamento-inicial":
				_, _ = w.Write([]byte(`{"tokenPortal":"tok","entidades":[{"id":1,"nome":"Prefeitura","tipoEntidade":"PREFEITURA"}],"anos":[2025]}`))
			case "/api/orgaos-e-servidores/servidor/paginado":
				pg := r.URL.Query().Get("pagina")
				pages = append(pages, pg)
				if r.URL.Query().Get("nomeDoFuncionario") != "ALVO" {
					t.Errorf("esperava nomeDoFuncionario=ALVO, obteve %q", r.URL.Query().Get("nomeDoFuncionario"))
				}
				var regs []map[string]any
				if pg == "1" {
					for i := 0; i < 200; i++ {
						regs = append(regs, map[string]any{"nome": fmt.Sprintf("ALVO %03d", i), "matricula": fmt.Sprintf("M%d", i), "proventos": 4000.0})
					}
				} else {
					for i := 0; i < 20; i++ {
						regs = append(regs, map[string]any{"nome": fmt.Sprintf("ALVO P2 %02d", i), "matricula": fmt.Sprintf("M2%d", i), "proventos": 5000.0})
					}
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"total": 220, "registros": regs})
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		res, err := transparencia.QueryMegasoftServidores(context.Background(), srv.Client(), m, transparencia.PortalQueryParams{
			BaseURL:    srv.URL,
			Ano:        2025,
			Mes:        3,
			NomeFiltro: "ALVO",
			Limit:      210,
		})
		if err != nil {
			t.Fatalf("QueryMegasoftServidores erro: %v", err)
		}
		if len(pages) != 2 || pages[0] != "1" || pages[1] != "2" {
			t.Errorf("esperava páginas [1 2], obteve %v", pages)
		}
		if len(res.Servidores) != 210 || res.TotalRegistros != 220 {
			t.Errorf("esperava 210 itens e total 220, obteve %d e %d", len(res.Servidores), res.TotalRegistros)
		}
	})

	t.Run("elotech_pagination_cap_and_mes", func(t *testing.T) {
		var gotPages []string
		var gotMes string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/api/servidores" {
				q := r.URL.Query()
				if sz, _ := strconv.Atoi(q.Get("size")); sz > 200 {
					t.Errorf("size Elotech excedeu 200: %d", sz)
				}
				gotMes = q.Get("mesCompetencia")
				pg := q.Get("page")
				gotPages = append(gotPages, pg)
				n := 200
				if pg == "1" {
					n = 60
				}
				var content []map[string]any
				pgNum, _ := strconv.Atoi(pg)
				for i := 0; i < n; i++ {
					content = append(content, map[string]any{"nome": fmt.Sprintf("ELO-%s-%d", pg, i), "matricula": pgNum*1000 + i + 1, "salarioBruto": 3000.0})
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"content": content, "totalElements": 260})
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		res, err := transparencia.QueryElotechServidores(context.Background(), srv.Client(), m, transparencia.PortalQueryParams{
			BaseURL: srv.URL,
			Ano:     2025,
			Mes:     5,
			Limit:   250,
		})
		if err != nil {
			t.Fatalf("QueryElotechServidores erro: %v", err)
		}
		if gotMes != "5" {
			t.Errorf("mesCompetencia esperado '5', obteve %q", gotMes)
		}
		if len(gotPages) != 2 || gotPages[0] != "0" || gotPages[1] != "1" {
			t.Errorf("páginas esperadas [0 1], obteve %v", gotPages)
		}
		if len(res.Servidores) != 250 {
			t.Errorf("esperava 250 servidores Elotech, obteve %d", len(res.Servidores))
		}
	})

	t.Run("betha_offset_pagination_and_filter", func(t *testing.T) {
		transparencia.ResetBethaTokenCache()
		var offsets []string
		var seenTexto string
		var srvURL string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/oauth":
				w.Header().Set("Location", "https://transparencia.betha.cloud/auth-callback.html#access_token=tokb&expires_in=3600")
				w.WriteHeader(http.StatusSeeOther)
			case "/api/api/menu":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[{"id":1,"titulo":"RH","itens":[{"id":10,"titulo":"Servidores","consultaId":10,"tipo":"CONSULTA_PADRAO"}]}]`))
			case "/api/api/busca-textual/10":
				w.Header().Set("Content-Type", "application/json")
				bodyBytes, _ := io.ReadAll(r.Body)
				var pl map[string]any
				_ = json.Unmarshal(bodyBytes, &pl)
				if s, ok := pl["texto"].(string); ok {
					seenTexto = s
				}
				off := r.URL.Query().Get("offset")
				offsets = append(offsets, off)
				n := 200
				if off != "0" {
					n = 55
				}
				var content []map[string]any
				for i := 0; i < n; i++ {
					content = append(content, map[string]any{"nome": fmt.Sprintf("SILVA %s-%d", off, i), "matricula": fmt.Sprintf("B%s-%d", off, i), "salarioBruto": 3500.0})
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"content": content, "totalElements": 255})
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()
		srvURL = srv.URL

		transparencia.SetBethaOAuthURL(srvURL + "/oauth")
		transparencia.SetBethaAPIBaseURL(srvURL + "/api")
		defer func() {
			transparencia.SetBethaOAuthURL("https://plataforma-oauth.betha.cloud/auth/oauth2/authorize")
			transparencia.SetBethaAPIBaseURL("https://api.transparencia.betha.cloud/transparencia")
			transparencia.ResetBethaTokenCache()
		}()

		res, err := transparencia.QueryBethaServidores(context.Background(), srv.Client(), m, transparencia.PortalQueryParams{
			TenantID:   "HASH_BETHA",
			NomeFiltro: "SILVA",
			Limit:      250,
		})
		if err != nil {
			t.Fatalf("QueryBethaServidores erro: %v", err)
		}
		if seenTexto != "SILVA" {
			t.Errorf("esperava texto='SILVA' no POST, obteve %q", seenTexto)
		}
		if len(offsets) != 2 || offsets[0] != "0" || offsets[1] != "200" {
			t.Errorf("offsets esperados [0 200], obteve %v", offsets)
		}
		if len(res.Servidores) != 250 {
			t.Errorf("esperava 250 servidores Betha, obteve %d", len(res.Servidores))
		}
	})

	t.Run("ipm_pagination_and_matched_total", func(t *testing.T) {
		var pages []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/api/transparencia-pessoal-funcionarios" {
				pg := r.URL.Query().Get("pagina")
				pages = append(pages, pg)
				if pg == "1" {
					_, _ = w.Write([]byte(`{"total":4,"registros":[{"nome":"OUTRO 1","cargo":"AUX"},{"nome":"OUTRO 2","cargo":"AUX"}]}`))
					return
				}
				_, _ = w.Write([]byte(`{"total":4,"registros":[{"nome":"ALVO IPM 1","cargo":"MEDICO"},{"nome":"ALVO IPM 2","cargo":"ENFERMEIRO"}]}`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		res, err := transparencia.QueryIPMServidores(context.Background(), srv.Client(), m, transparencia.PortalQueryParams{
			BaseURL:    srv.URL,
			NomeFiltro: "ALVO",
			Limit:      1,
		})
		if err != nil {
			t.Fatalf("QueryIPMServidores erro: %v", err)
		}
		if len(pages) != 2 || pages[1] != "2" {
			t.Errorf("esperava leitura da página 2 no IPM, obteve %v", pages)
		}
		if len(res.Servidores) != 1 || res.TotalRegistros != 2 {
			t.Errorf("esperava 1 item e TotalRegistros=2 (matchedTotal), obteve %d e %d", len(res.Servidores), res.TotalRegistros)
		}
	})

	t.Run("portaltp_filter_scan_limit_500", func(t *testing.T) {
		var pages []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/api/transparencia.asmx/json_servidores" {
				pg := r.URL.Query().Get("pagina")
				if pg == "" {
					pg = "1"
				}
				pages = append(pages, pg)
				if pg == "1" {
					var rows []map[string]any
					for i := 0; i < 10; i++ {
						rows = append(rows, map[string]any{"nome": fmt.Sprintf("OUTRO %d", i), "salarioBruto": 2000})
					}
					_ = json.NewEncoder(w).Encode(rows)
					return
				}
				if pg == "2" {
					_ = json.NewEncoder(w).Encode([]map[string]any{
						{"nome": "ALVO PORTALTP 1", "salarioBruto": 6000},
						{"nome": "ALVO PORTALTP 2", "salarioBruto": 7000},
					})
					return
				}
				_, _ = w.Write([]byte(`[]`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		res, err := transparencia.QueryPortalTPServidores(context.Background(), srv.Client(), m, transparencia.PortalQueryParams{
			BaseURL:    srv.URL,
			Ano:        2025,
			NomeFiltro: "ALVO",
			Limit:      1,
		})
		if err != nil {
			t.Fatalf("QueryPortalTPServidores erro: %v", err)
		}
		if len(pages) < 2 {
			t.Errorf("esperava avanço para página 2 quando NomeFiltro está ativo com Limit=1, páginas=%v", pages)
		}
		if len(res.Servidores) != 1 || res.TotalRegistros != 2 || res.Servidores[0].Nome != "ALVO PORTALTP 1" {
			t.Errorf("resultado PortalTP inesperado: items=%+v total=%d", res.Servidores, res.TotalRegistros)
		}
	})

	t.Run("centi_month_window_and_filter_total", func(t *testing.T) {
		var gotInicio, gotFim string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if strings.HasPrefix(r.URL.Path, "/portal/getempenhos/") {
				b, _ := io.ReadAll(r.Body)
				var pl map[string]any
				_ = json.Unmarshal(b, &pl)
				gotInicio, _ = pl["DataInicio"].(string)
				gotFim, _ = pl["DataFim"].(string)
				pageObj, _ := pl["page"].(map[string]any)
				pgNum := int(pageObj["number"].(float64))
				if pgNum == 1 {
					_ = json.NewEncoder(w).Encode(map[string]any{
						"total": 4,
						"empenhos": []map[string]any{
							{"Credor": "OUTRO 1", "Historico": "X", "ValorEmpenhado": 100.0},
							{"Credor": "ALVO CENTI 1", "Historico": "Y", "ValorEmpenhado": 200.0},
							{"Credor": "ALVO CENTI 2", "Historico": "Z", "ValorEmpenhado": 300.0},
						},
					})
					return
				}
				_, _ = w.Write([]byte(`{"empenhos":[]}`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		res, err := transparencia.QueryCentiDespesas(context.Background(), srv.Client(), m, transparencia.PortalQueryParams{
			BaseURL:    srv.URL,
			TenantID:   "serradasaudade",
			Ano:        2024,
			Mes:        2,
			NomeFiltro: "ALVO",
			Limit:      1,
		})
		if err != nil {
			t.Fatalf("QueryCentiDespesas erro: %v", err)
		}
		if gotInicio != "01/02/2024" || gotFim != "29/02/2024" {
			t.Errorf("janela DD/MM/YYYY Centi incorreta: %s .. %s", gotInicio, gotFim)
		}
		if len(res.Despesas) != 1 || res.TotalRegistros != 2 {
			t.Errorf("esperava 1 item e TotalRegistros=2, obteve %d e %d", len(res.Despesas), res.TotalRegistros)
		}
	})

	t.Run("govbr_and_coplan_mes_pagination", func(t *testing.T) {
		var govInicio, govFim string
		var govPages []string
		var coplanMes float64
		var coplanPages []float64
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/FolhaPagamento":
				q := r.URL.Query()
				govInicio = q.Get("dataInicial")
				govFim = q.Get("dataFinal")
				pg := q.Get("numeroPagina")
				govPages = append(govPages, pg)
				if pg == "1" {
					var rows []map[string]any
					for i := 0; i < 10; i++ {
						rows = append(rows, map[string]any{"nome": fmt.Sprintf("OUTRO %d", i)})
					}
					_ = json.NewEncoder(w).Encode(rows)
					return
				}
				_ = json.NewEncoder(w).Encode([]map[string]any{
					{"nome": "ALVO GOVBR 1", "salarioBruto": 4000.0},
					{"nome": "ALVO GOVBR 2", "salarioBruto": 4500.0},
				})
			case "/servlet/api_consulta":
				b, _ := io.ReadAll(r.Body)
				var pl map[string]any
				_ = json.Unmarshal(b, &pl)
				coplanMes, _ = pl["mes"].(float64)
				pg, _ := pl["pagina"].(float64)
				coplanPages = append(coplanPages, pg)
				if pg == 1 {
					var rows []map[string]any
					for i := 0; i < 200; i++ {
						rows = append(rows, map[string]any{"fornecedor": fmt.Sprintf("COPLAN %d", i), "valorEmpenhado": 100.0})
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"despesas": rows, "total": 220})
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"despesas": []map[string]any{{"fornecedor": "COPLAN P2", "valorEmpenhado": 200.0}},
					"total":    220,
				})
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		govRes, err := transparencia.QueryGovBRServidores(context.Background(), srv.Client(), m, transparencia.PortalQueryParams{
			BaseURL:    srv.URL,
			Ano:        2024,
			Mes:        2,
			NomeFiltro: "ALVO",
			Limit:      1,
		}, "ENT")
		if err != nil {
			t.Fatalf("QueryGovBRServidores erro: %v", err)
		}
		if govInicio != "01/02/2024" || govFim != "29/02/2024" || len(govPages) < 2 {
			t.Errorf("GovBR mês/paginação inesperado: ini=%s fim=%s pages=%v", govInicio, govFim, govPages)
		}
		if len(govRes.Servidores) != 1 || govRes.TotalRegistros != 2 {
			t.Errorf("GovBR resultado inesperado: len=%d total=%d", len(govRes.Servidores), govRes.TotalRegistros)
		}

		copRes, err := transparencia.QueryCoplanDespesas(context.Background(), srv.Client(), m, transparencia.PortalQueryParams{
			BaseURL:  srv.URL,
			TenantID: "1",
			Ano:      2025,
			Mes:      7,
			Limit:    201,
		})
		if err != nil {
			t.Fatalf("QueryCoplanDespesas erro: %v", err)
		}
		if coplanMes != 7 || len(coplanPages) != 2 {
			t.Errorf("Coplan mes/páginas inesperado: mes=%v pages=%v", coplanMes, coplanPages)
		}
		if len(copRes.Despesas) != 201 {
			t.Errorf("Coplan esperava 201 itens, obteve %d", len(copRes.Despesas))
		}
	})

	t.Run("sonner_ordem_salario_desc_multi_token_and_concurrent_enrich", func(t *testing.T) {
		var enrichCalls atomic.Int32
		var tokensMu sync.Mutex
		seenTokens := map[string]bool{}

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/GRP/pic/api/config/contexto":
				_, _ = w.Write([]byte(`{"dadosOrgao":{"nome":"Prefeitura Teste"},"exercicioAtual":2025}`))
			case "/GRP/pic/api/auth/csrf":
				http.SetCookie(w, &http.Cookie{Name: "PORTAL-XSRF-TOKEN", Value: "xsrf", Path: "/"})
				http.SetCookie(w, &http.Cookie{Name: "JSESSIONID", Value: "jsess", Path: "/"})
				_, _ = w.Write([]byte(`{"ok":true}`))
			case "/GRP/pic/api/funcionalidade":
				_, _ = w.Write([]byte(`[{"id":105,"text":"Folha","rota":"visualiza-salario","leaf":true}]`))
			case "/GRP/pic/api/visualiza-salario/funcionario":
				tok := r.URL.Query().Get("token")
				tokensMu.Lock()
				seenTokens[tok] = true
				tokensMu.Unlock()

				var rows []map[string]any
				if tok == " " {
					for i := 1; i <= 10; i++ {
						rows = append(rows, map[string]any{
							"id":        i,
							"matricula": fmt.Sprintf("MAT-%02d", i),
							"nome":      fmt.Sprintf("SERVIDOR %02d", i),
						})
					}
				} else if tok == "A" {
					// Inclui duplicata do id 1 e novos 11..14
					rows = append(rows, map[string]any{"id": 1, "matricula": "MAT-01", "nome": "SERVIDOR 01"})
					for i := 11; i <= 14; i++ {
						rows = append(rows, map[string]any{
							"id":        i,
							"matricula": fmt.Sprintf("MAT-%02d", i),
							"nome":      fmt.Sprintf("SERVIDOR %02d", i),
						})
					}
				} else {
					rows = []map[string]any{}
				}
				_ = json.NewEncoder(w).Encode(rows)
			case "/GRP/pic/api/visualiza-salario/anos-folha":
				_, _ = w.Write([]byte(`[2025]`))
			case "/GRP/pic/api/visualiza-salario/matriculafolhas":
				enrichCalls.Add(1)
				idStr := r.URL.Query().Get("funcionarioId")
				id, _ := strconv.Atoi(idStr)
				sal := float64(3000 + id*1000)
				if id == 13 {
					sal = 42000.0 // > 10º registro, garante que enriqueceu além do antigo cap de 10!
				}
				_ = json.NewEncoder(w).Encode([]map[string]any{
					{"ano": 2025, "mes": 4, "totalProventos": sal, "totalLiquido": sal * 0.8},
				})
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		eng, err := transparencia.NewEngine()
		if err != nil {
			t.Fatalf("NewEngine erro: %v", err)
		}
		hub := &transparencia.Hub{
			Engine:     eng,
			HTTPClient: srv.Client(),
			TCE:        transparencia.NewTCEClient(srv.Client()),
			Siconfi:    transparencia.NewSiconfiClient(),
			PNCP:       transparencia.NewPNCPClient(),
		}

		res, err := hub.QueryServidores(context.Background(), "Serra da Saudade", transparencia.QueryOptions{
			UF:               "MG",
			OverrideURL:      srv.URL,
			Ano:              2025,
			Limit:            3,
			OrdemSalarioDesc: true,
		})
		if err != nil {
			t.Fatalf("Hub QueryServidores Sonner erro: %v", err)
		}
		if enrichCalls.Load() != 14 {
			t.Errorf("esperava 14 chamadas de enriquecimento concorrente (sem duplicar id=1), obteve %d", enrichCalls.Load())
		}
		if !seenTokens[" "] || !seenTokens["A"] {
			t.Errorf("esperava múltiplos tokens (' ', 'A') consultados no modo OrdemSalarioDesc, obteve %v", seenTokens)
		}
		if len(res.Servidores) != 3 || res.Servidores[0].Matricula != "MAT-13" || res.Servidores[0].SalarioBruto != 42000.0 {
			t.Errorf("top servidor Sonner inesperado após ordenação/enriquecimento: %+v", res.Servidores)
		}
	})

	t.Run("publicsoft_page2_and_per_page_cap", func(t *testing.T) {
		var seenURLs []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/api/v1/despesas/detalhe" {
				seenURLs = append(seenURLs, r.URL.RawQuery)
				pg := r.URL.Query().Get("page")
				n := 200
				if pg == "2" {
					n = 60
				}
				var rows []map[string]any
				for i := 0; i < n; i++ {
					rows = append(rows, map[string]any{"numero": fmt.Sprintf("P%s-%d", pg, i), "fornecedor": "FORN", "valorEmpenhado": 100.0})
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"data": rows, "total": 260}})
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		res, err := transparencia.QueryPublicsoftDespesas(context.Background(), srv.Client(), m, transparencia.PortalQueryParams{
			BaseURL:  srv.URL,
			TenantID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Ano:      2025,
			Limit:    250,
		})
		if err != nil {
			t.Fatalf("QueryPublicsoftDespesas erro: %v", err)
		}
		if len(seenURLs) != 2 || strings.Contains(seenURLs[0], "&page=") || !strings.Contains(seenURLs[1], "page=2") {
			t.Errorf("URLs de paginação Publicsoft inesperadas: %v", seenURLs)
		}
		if !strings.Contains(seenURLs[0], "per_page=200") {
			t.Errorf("esperava per_page=200, obteve %s", seenURLs[0])
		}
		if len(res.Despesas) != 250 {
			t.Errorf("esperava 250 despesas Publicsoft, obteve %d", len(res.Despesas))
		}
	})

	t.Run("cittaweb_mes_and_contratos_pagination", func(t *testing.T) {
		var seenDespQuery string
		var seenContPages []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/public/despesas/elementos":
				seenDespQuery = r.URL.RawQuery
				_, _ = w.Write([]byte(`[{"codigo":"339030","descricao":"Material","empenhado":500.0}]`))
			case "/public/contratos/lista":
				pg := r.URL.Query().Get("page")
				seenContPages = append(seenContPages, pg)
				n := 200
				if pg == "1" {
					n = 60
				}
				var rows []map[string]any
				for i := 0; i < n; i++ {
					rows = append(rows, map[string]any{"numero": fmt.Sprintf("CT-%s-%d", pg, i), "fornecedor": "F", "valor": 1000.0})
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"content": rows, "totalElements": 260})
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		_, err := transparencia.QueryCittaWebDespesas(context.Background(), srv.Client(), m, transparencia.PortalQueryParams{
			BaseURL:          srv.URL,
			CodAdministracao: 1,
			Ano:              2025,
			Mes:              5,
		})
		if err != nil {
			t.Fatalf("QueryCittaWebDespesas erro: %v", err)
		}
		if !strings.Contains(seenDespQuery, "mesInicial=5") || !strings.Contains(seenDespQuery, "mesFim=5") {
			t.Errorf("CittaWeb despesas deveria enviar mesInicial=5&mesFim=5, obteve %s", seenDespQuery)
		}

		cRes, err := transparencia.QueryCittaWebContratos(context.Background(), srv.Client(), m, transparencia.PortalQueryParams{
			BaseURL:          srv.URL,
			CodAdministracao: 1,
			Ano:              2025,
			Limit:            250,
		})
		if err != nil {
			t.Fatalf("QueryCittaWebContratos erro: %v", err)
		}
		if len(seenContPages) != 2 || seenContPages[0] != "0" || seenContPages[1] != "1" {
			t.Errorf("páginas CittaWeb esperadas [0 1], obteve %v", seenContPages)
		}
		if len(cRes.Contratos) != 250 {
			t.Errorf("esperava 250 contratos CittaWeb, obteve %d", len(cRes.Contratos))
		}
	})

	t.Run("aspec_pagination_cap_and_matched_total", func(t *testing.T) {
		var seenPages []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/transparencia/api/v1/json/exercicios/11629488":
				_, _ = w.Write([]byte(`[{"id":"ex25","nome":"2025"}]`))
			case "/transparencia/api/v1/json/empenhos/11629488":
				pg := r.URL.Query().Get("page")
				ps, _ := strconv.Atoi(r.URL.Query().Get("pagesize"))
				if ps > 200 {
					t.Errorf("Aspec pagesize excedeu 200: %d", ps)
				}
				seenPages = append(seenPages, pg)
				if pg == "1" {
					var rows []map[string]any
					for i := 0; i < 200; i++ {
						rows = append(rows, map[string]any{"numero": fmt.Sprintf("E%d", i), "credor": "OUTRO"})
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"empenhos": rows, "total": 210})
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"empenhos": []map[string]any{
						{"numero": "A1", "credor": "ALVO ASPEC 1", "valorEmpenhado": 500.0},
						{"numero": "A2", "credor": "ALVO ASPEC 2", "valorEmpenhado": 600.0},
					},
					"total": 210,
				})
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		res, err := transparencia.QueryAspecDespesas(context.Background(), srv.Client(), m, transparencia.PortalQueryParams{
			BaseURL:    srv.URL + "/11629488",
			Ano:        2025,
			Mes:        3,
			NomeFiltro: "ALVO",
			Limit:      1,
		})
		if err != nil {
			t.Fatalf("QueryAspecDespesas erro: %v", err)
		}
		if len(seenPages) != 2 || seenPages[0] != "1" || seenPages[1] != "2" {
			t.Errorf("Aspec páginas esperadas [1 2], obteve %v", seenPages)
		}
		if len(res.Despesas) != 1 || res.TotalRegistros != 2 {
			t.Errorf("esperava 1 despesa e TotalRegistros=2, obteve %d e %d", len(res.Despesas), res.TotalRegistros)
		}
	})

	t.Run("nucleogov_leap_month_window_and_offset_pagination", func(t *testing.T) {
		var seenIni, seenFim string
		var seenLimits []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/api/centi" {
				_ = r.ParseForm()
				seenIni = r.Form.Get("data_inicio")
				seenFim = r.Form.Get("data_fim")
				lim := r.Form.Get("limit")
				seenLimits = append(seenLimits, lim)
				n := 200
				if strings.HasPrefix(lim, "200,") {
					n = 60
				}
				var rows []map[string]any
				for i := 0; i < n; i++ {
					rows = append(rows, map[string]any{"NumeroEmpenho": fmt.Sprintf("%s-%d", lim, i), "Fornecedor": "F", "ValorEmpenhado": 100.0})
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"empenhos": rows, "total": 260})
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		res, err := transparencia.QueryNucleoGovDespesas(context.Background(), srv.Client(), m, transparencia.PortalQueryParams{
			BaseURL: srv.URL,
			Ano:     2024,
			Mes:     2,
			Limit:   250,
		})
		if err != nil {
			t.Fatalf("QueryNucleoGovDespesas erro: %v", err)
		}
		if seenIni != "01/02/2024" || seenFim != "29/02/2024" {
			t.Errorf("janela NucleoGov fev/2024 bissexto incorreta: %s .. %s", seenIni, seenFim)
		}
		if len(seenLimits) != 2 || seenLimits[0] != "0,200" || seenLimits[1] != "200,200" {
			t.Errorf("limits NucleoGov esperados [0,200 200,200], obteve %v", seenLimits)
		}
		if len(res.Despesas) != 250 {
			t.Errorf("esperava 250 despesas NucleoGov, obteve %d", len(res.Despesas))
		}
	})

	t.Run("facil_smarapd_memory_pagination", func(t *testing.T) {
		// Fácil
		var facilStarts []int
		facilSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/api/servidor/ServidorGetNomeGrid" {
				b, _ := io.ReadAll(r.Body)
				var pl map[string]any
				_ = json.Unmarshal(b, &pl)
				params := pl["parameters"].(map[string]any)
				start := int(params["start"].(float64))
				facilStarts = append(facilStarts, start)
				n := 200
				if start > 0 {
					n = 60
				}
				var rows []map[string]any
				for i := 0; i < n; i++ {
					rows = append(rows, map[string]any{"nome": fmt.Sprintf("FAC-%d-%d", start, i)})
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"data": rows, "recordsTotal": 260})
				return
			}
			http.NotFound(w, r)
		}))
		defer facilSrv.Close()

		fRes, err := transparencia.QueryFacilServidores(context.Background(), facilSrv.Client(), m, transparencia.PortalQueryParams{
			BaseURL:  facilSrv.URL,
			TenantID: "DB1",
			Ano:      2025,
			Limit:    250,
		})
		if err != nil || len(facilStarts) != 2 || facilStarts[0] != 0 || facilStarts[1] != 200 || len(fRes.Servidores) != 250 {
			t.Fatalf("Fácil paginação falhou: err=%v starts=%v len=%d", err, facilStarts, len(fRes.Servidores))
		}

		// SmarAPD
		var smarPages []int
		smarSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/paiportalserver/DadosAbertos":
				_, _ = w.Write([]byte(`{"Modulos":[{"ChaveModulo":"Servidores","NomeVisao":"Servidores"}]}`))
			case "/paiportalserver/modulovisao/filter":
				b, _ := io.ReadAll(r.Body)
				var pl map[string]any
				_ = json.Unmarshal(b, &pl)
				pg := int(pl["Pagina"].(float64))
				smarPages = append(smarPages, pg)
				n := 200
				if pg == 2 {
					n = 60
				}
				var rows []map[string]any
				for i := 0; i < n; i++ {
					rows = append(rows, map[string]any{"Nome": fmt.Sprintf("SMAR-%d-%d", pg, i)})
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"Dados": rows, "Total": 260})
			default:
				http.NotFound(w, r)
			}
		}))
		defer smarSrv.Close()

		sRes, err := transparencia.QuerySmarapdServidores(context.Background(), smarSrv.Client(), m, transparencia.PortalQueryParams{
			BaseURL: smarSrv.URL,
			Limit:   250,
		})
		if err != nil || len(smarPages) != 2 || smarPages[0] != 1 || smarPages[1] != 2 || len(sRes.Servidores) != 250 {
			t.Fatalf("SmarAPD paginação falhou: err=%v pages=%v len=%d", err, smarPages, len(sRes.Servidores))
		}

		// Memory
		var memPages []string
		memSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/lai/pessoal/servidor/" {
				pg := r.URL.Query().Get("page")
				memPages = append(memPages, pg)
				n := 200
				if pg == "2" {
					n = 60
				}
				var rows []map[string]any
				for i := 0; i < n; i++ {
					rows = append(rows, map[string]any{"nome": fmt.Sprintf("MEM-%s-%d", pg, i)})
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"results": rows, "count": 260})
				return
			}
			http.NotFound(w, r)
		}))
		defer memSrv.Close()

		mRes, err := transparencia.QueryMemoryServidores(context.Background(), memSrv.Client(), m, transparencia.PortalQueryParams{
			BaseURL:  memSrv.URL,
			TenantID: "ABCDEF:1",
			Ano:      2025,
			Limit:    250,
		})
		if err != nil || len(memPages) != 2 || memPages[0] != "1" || memPages[1] != "2" || len(mRes.Servidores) != 250 {
			t.Fatalf("Memory paginação falhou: err=%v pages=%v len=%d", err, memPages, len(mRes.Servidores))
		}
	})

	t.Run("tce_ce_folha_pagination_and_matched_total", func(t *testing.T) {
		var folhaStarts []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/municipios":
				_, _ = w.Write([]byte(`{"elements":[{"codigo_municipio":"001","nome_municipio":"Fortaleza","codigo_municipio_ibge":"2304400"}]}`))
			case "/agentes_publicos_municipais":
				_, _ = w.Write([]byte(`{"elements":[
					{"nome_servidor":"OUTRO CE","cpf_servidor":"111","numero_matricula":"1","nm_tipo_cargo":"AUX"},
					{"nome_servidor":"ALVO CE 1","cpf_servidor":"222","numero_matricula":"2","nm_tipo_cargo":"PREFEITO"},
					{"nome_servidor":"ALVO CE 2","cpf_servidor":"333","numero_matricula":"3","nm_tipo_cargo":"MEDICO"}
				]}`))
			case "/agentes_publicos_folha":
				si := r.URL.Query().Get("$start_index")
				folhaStarts = append(folhaStarts, si)
				if si == "0" {
					elems := make([]map[string]any, 0, 500)
					for i := 0; i < 500; i++ {
						elems = append(elems, map[string]any{"cpf_agente": "111", "valor_item": 10.0, "codigo_detutibilidade_item": false})
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"elements": elems})
					return
				}
				// Página 2 ($start_index=500) traz a remuneração do ALVO CE 1
				_ = json.NewEncoder(w).Encode(map[string]any{
					"elements": []map[string]any{
						{"cpf_agente": "222", "valor_item": 28000.0, "codigo_detutibilidade_item": false},
						{"cpf_agente": "222", "valor_item": 3000.0, "codigo_detutibilidade_item": true},
					},
				})
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		tce := transparencia.NewTCEClient(srv.Client())
		tce.SetTCECEBaseURL(srv.URL)
		ceMuni := &transparencia.Municipio{CodigoIBGE: "2304400", Nome: "Fortaleza", UF: "CE"}
		res, err := tce.FetchTCECEServidores(context.Background(), ceMuni, 2025, 1, "ALVO", 1)
		if err != nil {
			t.Fatalf("FetchTCECEServidores erro: %v", err)
		}
		if len(folhaStarts) != 2 || folhaStarts[0] != "0" || folhaStarts[1] != "500" {
			t.Errorf("esperava paginação da folha TCE-CE em $start_index=[0 500], obteve %v", folhaStarts)
		}
		if len(res.Servidores) != 1 || res.TotalRegistros != 2 {
			t.Errorf("esperava 1 servidor e TotalRegistros=2, obteve %d e %d", len(res.Servidores), res.TotalRegistros)
		}
		if res.Servidores[0].SalarioBruto != 28000.0 || res.Servidores[0].SalarioLiquido != 25000.0 {
			t.Errorf("enriquecimento da folha pág 2 TCE-CE incorreto: %+v", res.Servidores[0])
		}
	})
}
