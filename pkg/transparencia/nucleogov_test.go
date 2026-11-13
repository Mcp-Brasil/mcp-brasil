package transparencia_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gibavargas/mcp-brasil/pkg/transparencia"
)

func TestProbeNucleoGov(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	t.Run("orgaos_and_cidadao_strip", func(t *testing.T) {
		var hitPath string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && r.URL.Path == "/api/orgaos" {
				if err := r.ParseForm(); err == nil && r.Form.Get("acao") == "listar" && r.Form.Get("limit") == "0,1" {
					hitPath = r.URL.Path
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"dados":[{"Id":10,"Nome":"Prefeitura Municipal","Principal":true}],"total":1}`))
					return
				}
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if !transparencia.ProbeNucleoGov(ctx, client, srv.URL+"/cidadao/transparencia") {
			t.Fatalf("esperava ProbeNucleoGov=true para URL com /cidadao/transparencia")
		}
		if hitPath != "/api/orgaos" {
			t.Errorf("esperava hit em /api/orgaos, obteve %q", hitPath)
		}
	})

	t.Run("multi_request_api", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && r.URL.Path == "/api" {
				_ = r.ParseForm()
				if r.Form.Get("multi_request") == "true" && strings.Contains(r.Form.Get("params"), "orgaos/listar") {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"r1":[{"Id":1,"Nome":"Câmara"}]}`))
					return
				}
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if !transparencia.ProbeNucleoGov(ctx, client, srv.URL) {
			t.Fatalf("esperava ProbeNucleoGov=true via /api multi_request")
		}
	})

	t.Run("html_static_nucleogov_plus_orgaos_200", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPost && r.URL.Path == "/api/orgaos":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[]`))
			case r.Method == http.MethodGet && (r.URL.Path == "/cidadao/transparencia" || r.URL.Path == "/"):
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte(`<html><script src="https://static.nucleogov.com.br/app.js"></script></html>`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		if !transparencia.ProbeNucleoGov(ctx, client, srv.URL+"/cidadao/transparencia") {
			t.Fatalf("esperava ProbeNucleoGov=true via static.nucleogov.com.br + /api/orgaos 200")
		}
	})

	t.Run("non_nucleogov", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if transparencia.ProbeNucleoGov(ctx, client, srv.URL) {
			t.Fatalf("esperava ProbeNucleoGov=false")
		}
	})
}

func TestNucleoGovQueriesMock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "5201108", Nome: "Anápolis", UF: "GO"}

	t.Run("servidores_despesas_receitas_contratos_licitacoes", func(t *testing.T) {
		var seenOrgaoReceita, seenEmpenhoInicio, seenEmpenhoFim string

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = r.ParseForm()
			switch r.URL.Path {
			case "/api/orgaos":
				_, _ = w.Write([]byte(`[
					{"Id": 2, "Nome": "Fundo Municipal de Saúde", "Principal": false},
					{"Id": 7, "Nome": "Prefeitura Municipal de Anápolis", "Principal": true}
				]`))

			case "/api/servidores_cnt":
				_, _ = w.Write([]byte(`{
					"dados": [
						{
							"Nome": "MARIA CLARA SOUZA",
							"Matricula": "1045",
							"CPF": "12345678901",
							"Cargo": "Professora",
							"Lotacao": "Secretaria de Educação",
							"Vinculo": "Efetivo",
							"Situacao": "Ativo",
							"SalarioBase": "4.200,00",
							"SalarioBruto": "6.543,21",
							"Descontos": "1.043,21",
							"SalarioLiquido": "5.500,00",
							"Orgao": "Prefeitura Municipal"
						},
						{
							"Nome": "JOSE CARLOS",
							"Matricula": "2099",
							"CPF": "99988877766",
							"Cargo": "Motorista",
							"SalarioBruto": 3000.00
						}
					],
					"total": 42
				}`))

			case "/api/centi":
				acao := r.Form.Get("acao")
				switch acao {
				case "empenhos":
					seenEmpenhoInicio = r.Form.Get("data_inicio")
					seenEmpenhoFim = r.Form.Get("data_fim")
					_, _ = w.Write([]byte(`{
						"dados": [
							{
								"NumeroEmpenho": "102/2025",
								"Ano": 2025,
								"Data": "2025-03-10",
								"Fornecedor": "TECH SOLUCOES LTDA",
								"CPFCNPJ": "11.222.333/0001-44",
								"Orgao": "Secretaria de Administração",
								"Historico": "Manutenção de sistemas",
								"ValorEmpenhado": "15.000,50",
								"ValorLiquidado": "10.000,00",
								"ValorPago": "10.000,00"
							},
							{
								"NumeroEmpenho": "103/2025",
								"Ano": 2025,
								"Data": "2025-03-11",
								"Fornecedor": "JOAO PRESTADOR",
								"CPFCNPJ": "123.456.789-01",
								"Historico": "Serviço eventual",
								"ValorEmpenhado": 800.00
							}
						],
						"total": 2
					}`))
				case "receitas":
					seenOrgaoReceita = r.Form.Get("orgao")
					_, _ = w.Write([]byte(`{
						"dados": [
							{
								"Codigo": "1.1.1.2.01",
								"Descricao": "IPTU - Imposto Predial",
								"Categoria": "Recursos Próprios",
								"ValorPrevisto": "1.200.000,00",
								"ValorArrecadado": "980.500,25"
							}
						],
						"total": 1
					}`))
				default:
					http.NotFound(w, r)
				}

			case "/api/contratos":
				_, _ = w.Write([]byte(`{
					"dados": [
						{
							"Numero": "045/2025",
							"Ano": 2025,
							"Fornecedor": "CONSTRUTORA CENTRAL",
							"CNPJ": "98.765.432/0001-10",
							"Objeto": "Reforma de escola municipal",
							"Situacao": "Vigente",
							"DataAssinatura": "2025-01-15",
							"VigenciaInicio": "2025-01-15",
							"VigenciaFim": "2025-12-31",
							"Valor": "250.000,00"
						}
					],
					"total": 1
				}`))

			case "/api/licitacoes":
				_, _ = w.Write([]byte(`{
					"dados": [
						{
							"Numero": "PE 012/2025",
							"Ano": 2025,
							"Modalidade": "Pregão Eletrônico",
							"Situacao": "Homologada",
							"DataAbertura": "2025-02-01",
							"Objeto": "Aquisição de merenda escolar",
							"ValorEstimado": "180.000,00",
							"ValorHomologado": "165.400,00"
						}
					],
					"total": 1
				}`))

			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		portalURL := srv.URL + "/cidadao/transparencia"

		// 1. Servidores (com filtro de nome)
		servRes, err := transparencia.QueryNucleoGovServidores(ctx, client, m, transparencia.PortalQueryParams{
			BaseURL:    portalURL,
			Ano:        2025,
			Mes:        10,
			NomeFiltro: "MARIA",
			Limit:      10,
		})
		if err != nil {
			t.Fatalf("QueryNucleoGovServidores erro: %v", err)
		}
		if servRes.Plataforma != "nucleogov" || servRes.PortalURL != srv.URL {
			t.Fatalf("PortalURL/Plataforma inesperados: %+v", servRes)
		}
		if len(servRes.Servidores) != 1 {
			t.Fatalf("esperava 1 servidor filtrado, obteve %d", len(servRes.Servidores))
		}
		s0 := servRes.Servidores[0]
		if s0.CPFParcial != "***.456.789-**" {
			t.Errorf("CPF não mascarado corretamente: %q", s0.CPFParcial)
		}
		if s0.SalarioBruto != 6543.21 || s0.SalarioLiquido != 5500.00 || s0.Competencia != "10/2025" {
			t.Errorf("dados salariais inesperados: %+v", s0)
		}

		// 2. Despesas (Centi)
		despRes, err := transparencia.QueryNucleoGovDespesas(ctx, client, m, transparencia.PortalQueryParams{
			BaseURL: portalURL,
			Ano:     2025,
			Limit:   10,
		})
		if err != nil {
			t.Fatalf("QueryNucleoGovDespesas erro: %v", err)
		}
		if seenEmpenhoInicio != "01/01/2025" || seenEmpenhoFim != "31/12/2025" {
			t.Errorf("datas Centi inesperadas: inicio=%q fim=%q", seenEmpenhoInicio, seenEmpenhoFim)
		}
		if len(despRes.Despesas) != 2 {
			t.Fatalf("esperava 2 despesas, obteve %d", len(despRes.Despesas))
		}
		if despRes.Despesas[0].DocumentoFornecedor != "11.222.333/0001-44" || despRes.Despesas[0].ValorEmpenhado != 15000.50 {
			t.Errorf("despesa[0] inesperada: %+v", despRes.Despesas[0])
		}
		if despRes.Despesas[1].DocumentoFornecedor != "***.456.789-**" {
			t.Errorf("despesa[1] CPF deveria estar mascarado: %q", despRes.Despesas[1].DocumentoFornecedor)
		}

		// 3. Receitas (busca orgao principal = 7 e consulta /api/centi acao=receitas)
		recRes, err := transparencia.QueryNucleoGovReceitas(ctx, client, m, transparencia.PortalQueryParams{
			BaseURL: portalURL,
			Ano:     2025,
			Mes:     10,
			Limit:   10,
		})
		if err != nil {
			t.Fatalf("QueryNucleoGovReceitas erro: %v", err)
		}
		if seenOrgaoReceita != "7" {
			t.Errorf("esperava orgao principal '7', obteve %q", seenOrgaoReceita)
		}
		if len(recRes.Receitas) != 1 || recRes.Receitas[0].ValorArrecadado != 980500.25 {
			t.Errorf("receitas inesperadas: %+v", recRes.Receitas)
		}

		// 4. Contratos
		contRes, err := transparencia.QueryNucleoGovContratos(ctx, client, m, transparencia.PortalQueryParams{
			BaseURL: portalURL,
			Limit:   10,
		})
		if err != nil {
			t.Fatalf("QueryNucleoGovContratos erro: %v", err)
		}
		if len(contRes.Contratos) != 1 || contRes.Contratos[0].Numero != "045/2025" || contRes.Contratos[0].Valor != 250000.00 {
			t.Errorf("contratos inesperados: %+v", contRes.Contratos)
		}

		// 5. Licitações
		licRes, err := transparencia.QueryNucleoGovLicitacoes(ctx, client, m, transparencia.PortalQueryParams{
			BaseURL: portalURL,
			Limit:   10,
		})
		if err != nil {
			t.Fatalf("QueryNucleoGovLicitacoes erro: %v", err)
		}
		if len(licRes.Licitacoes) != 1 || licRes.Licitacoes[0].Numero != "PE 012/2025" || licRes.Licitacoes[0].ValorHomologado != 165400.00 {
			t.Errorf("licitacoes inesperadas: %+v", licRes.Licitacoes)
		}
	})

	t.Run("multi_request_servidores_and_despesas_fallback", func(t *testing.T) {
		now := time.Now()
		prev := now.AddDate(0, -1, 0)

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = r.ParseForm()
			switch r.URL.Path {
			case "/api":
				var params map[string]map[string]any
				_ = json.Unmarshal([]byte(r.Form.Get("params")), &params)
				r1 := params["r1"]
				if r1 != nil && r1["acao"] == "servidores_cnt/listar" {
					anoVal, _ := r1["ano"].(float64)
					mesVal, _ := r1["mes"].(float64)
					if int(anoVal) == prev.Year() && int(mesVal) == int(prev.Month()) {
						_, _ = w.Write([]byte(`{
							"r1": {
								"dados": [
									{"Nome":"CARLOS MENDES","Matricula":"55","CPF":"32165498700","Cargo":"Fiscal","SalarioBruto":4500,"Descontos":500}
								],
								"total": 1
							}
						}`))
						return
					}
				}
				_, _ = w.Write([]byte(`{"r1":[]}`))

			case "/api/despesas":
				if r.Form.Get("acao") == "listar" && r.Form.Get("ano") == "2025" {
					_, _ = w.Write([]byte(`[{"NumeroEmpenho":"900/2025","Ano":2025,"Fornecedor":"PAPELARIA X","ValorEmpenhado":1200.00}]`))
					return
				}
				_, _ = w.Write([]byte(`[]`))

			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		// Servidores via multi_request + fallback automático de mês
		servRes, err := transparencia.QueryNucleoGovServidores(ctx, client, m, transparencia.PortalQueryParams{
			BaseURL: srv.URL,
			Limit:   10,
		})
		if err != nil {
			t.Fatalf("QueryNucleoGovServidores multi_request erro: %v", err)
		}
		if len(servRes.Servidores) != 1 || servRes.Servidores[0].CPFParcial != "***.654.987-**" || servRes.Servidores[0].SalarioLiquido != 4000 {
			t.Fatalf("servidores multi_request inesperado: %+v", servRes)
		}

		// Despesas via fallback /api/despesas quando /api/centi retorna 404
		despRes, err := transparencia.QueryNucleoGovDespesas(ctx, client, m, transparencia.PortalQueryParams{
			BaseURL: srv.URL,
			Ano:     2025,
			Limit:   10,
		})
		if err != nil {
			t.Fatalf("QueryNucleoGovDespesas fallback erro: %v", err)
		}
		if len(despRes.Despesas) != 1 || despRes.Despesas[0].NumeroEmpenho != "900/2025" {
			t.Fatalf("despesas fallback inesperado: %+v", despRes)
		}
	})
}
