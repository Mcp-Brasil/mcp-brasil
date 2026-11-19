package transparencia_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/Mcp-Brasil/mcp-brasil/pkg/transparencia"
)

func TestProbeSmarapd(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	t.Run("cliente_com_fragmento_e_headers_obrigatorios", func(t *testing.T) {
		var seenOrigin, seenReferer string
		var srv *httptest.Server
		srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seenOrigin = r.Header.Get("Origin")
			seenReferer = r.Header.Get("Referer")
			if seenOrigin == "" || seenReferer == "" {
				http.Error(w, "Missing Origin/Referer", http.StatusBadRequest)
				return
			}
			if r.URL.Path == "/paiportalserver/cliente" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"Nome":"PREFEITURA MUNICIPAL DE MARILIA","CNPJ":"44.477.909/0001-00"}`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		// Passa URL com #/inicio para validar remoção do fragmento
		if !transparencia.ProbeSmarapd(ctx, client, srv.URL+"/#/inicio") {
			t.Fatalf("esperava ProbeSmarapd=true com /paiportalserver/cliente e fragmento")
		}
		u, _ := url.Parse(srv.URL)
		wantOrigin := u.Scheme + "://" + u.Host
		if seenOrigin != wantOrigin || seenReferer != wantOrigin+"/" {
			t.Errorf("headers Origin/Referer inesperados: Origin=%q (want %q), Referer=%q (want %q)",
				seenOrigin, wantOrigin, seenReferer, wantOrigin+"/")
		}
	})

	t.Run("dados_abertos_fallback", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Origin") == "" || r.Header.Get("Referer") == "" {
				http.Error(w, "Missing Origin/Referer", http.StatusBadRequest)
				return
			}
			if r.URL.Path == "/portal/paiportalserver/DadosAbertos" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"Modulos":[{"ChaveModulo":"Folha","Visoes":[{"NomeVisao":"Servidores"}]}]}`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if !transparencia.ProbeSmarapd(ctx, client, srv.URL+"/portal/#/home") {
			t.Fatalf("esperava ProbeSmarapd=true via /paiportalserver/DadosAbertos")
		}
	})

	t.Run("negativo_html_ou_404", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<html><body>Not SmarAPD</body></html>`))
		}))
		defer srv.Close()

		if transparencia.ProbeSmarapd(ctx, client, srv.URL) {
			t.Fatalf("esperava ProbeSmarapd=false para resposta HTML")
		}
	})
}

func TestSmarapdAllQueriesMock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "3529005", Nome: "Marília", UF: "SP"}

	var filterRequests []map[string]any
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simula comportamento real de Marília/Cubatão/Bertioga exigindo Origin e Referer
		u, _ := url.Parse(srv.URL)
		wantOrigin := u.Scheme + "://" + u.Host
		if r.Header.Get("Origin") != wantOrigin || r.Header.Get("Referer") != wantOrigin+"/" {
			http.Error(w, "Missing or invalid Origin/Referer", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/paiportalserver/DadosAbertos":
			_, _ = w.Write([]byte(`{
				"Modulos": [
					{
						"ChaveModulo": "MOD_RH_CUSTOM",
						"Titulo": "Recursos Humanos",
						"Visoes": [
							{"NomeVisao": "VIS_FOLHA_SERVIDORES", "Titulo": "Folha de Pagamento e Servidores"}
						]
					},
					{
						"ChaveModulo": "MOD_EXEC_ORCAMENTARIA",
						"Titulo": "Execução Orçamentária",
						"Visoes": [
							{"NomeVisao": "VIS_EMPENHOS_DESPESAS", "Titulo": "Despesas e Empenhos"},
							{"NomeVisao": "VIS_RECEITAS_ARREC", "Titulo": "Receitas Arrecadadas"}
						]
					}
				]
			}`))
		case "/paiportalserver/MenuPortal":
			_, _ = w.Write([]byte(`[
				{
					"Titulo": "Compras e Licitações",
					"SubMenus": [
						{"ChaveModulo": "MOD_CONTRATOS_2025", "NomeVisao": "VIS_CONTRATOS_ATIVOS", "Titulo": "Contratos Administrativos"},
						{"ChaveModulo": "MOD_LICITACOES_PORTAL", "NomeVisao": "VIS_EDITAIS_PREGAO", "Titulo": "Licitações e Editais"}
					]
				}
			]`))
		case "/paiportalserver/modulovisao/filter":
			raw, _ := io.ReadAll(r.Body)
			var req map[string]any
			_ = json.Unmarshal(raw, &req)
			filterRequests = append(filterRequests, req)

			chave, _ := req["ChaveModulo"].(string)
			visao, _ := req["NomeVisao"].(string)
			switch chave + "/" + visao {
			case "MOD_RH_CUSTOM/VIS_FOLHA_SERVIDORES":
				_, _ = w.Write([]byte(`{
					"Registros": [
						{
							"Nome": "CARLOS ALBERTO",
							"Matricula": "1045",
							"CPF": "12345678901",
							"Cargo": "Engenheiro Civil",
							"Lotacao": "Secretaria de Obras",
							"Vinculo": "Estatutário",
							"Situacao": "Ativo",
							"SalarioBase": "7.500,00",
							"TotalProventos": 9200.50,
							"TotalDescontos": 1700.25,
							"SalarioLiquido": 7500.25,
							"Competencia": "03/2025"
						}
					],
					"TotalRegistros": 320
				}`))
			case "MOD_EXEC_ORCAMENTARIA/VIS_EMPENHOS_DESPESAS":
				_, _ = w.Write([]byte(`{
					"Registros": [
						{
							"NumeroEmpenho": "889/2025",
							"Ano": 2025,
							"Data": "2025-03-10",
							"Fornecedor": "CONSTRUTORA PAULISTA LTDA",
							"CPFCNPJ": "11.222.333/0001-44",
							"Orgao": "Obras",
							"Historico": "Pavimentação asfáltica",
							"ValorEmpenhado": 120000.00,
							"ValorLiquidado": 60000.00,
							"ValorPago": 60000.00
						},
						{
							"NumeroEmpenho": "890/2025",
							"Ano": 2025,
							"Data": "2025-03-11",
							"Fornecedor": "JOSE PEREIRA",
							"CPFCNPJ": "98765432100",
							"Orgao": "Gabinete",
							"Historico": "Adiantamento",
							"ValorEmpenhado": 800.00
						}
					],
					"TotalRegistros": 2
				}`))
			case "MOD_EXEC_ORCAMENTARIA/VIS_RECEITAS_ARREC":
				_, _ = w.Write([]byte(`{
					"Registros": [
						{
							"Codigo": "1.1.1.2.01",
							"Descricao": "IPTU - Imposto Predial",
							"Categoria": "Tributária",
							"Data": "2025-03-15",
							"ValorPrevisto": 15000000.00,
							"ValorArrecadado": "8.450.320,10"
						}
					],
					"TotalRegistros": 1
				}`))
			case "MOD_CONTRATOS_2025/VIS_CONTRATOS_ATIVOS":
				_, _ = w.Write([]byte(`{
					"Registros": [
						{
							"NumeroContrato": "045/2025",
							"Ano": 2025,
							"Fornecedor": "MEDICA SUL LTDA",
							"CNPJ": "55.666.777/0001-88",
							"Objeto": "Fornecimento de medicamentos",
							"Situacao": "Vigente",
							"DataAssinatura": "2025-01-20",
							"VigenciaInicio": "2025-01-20",
							"VigenciaFim": "2026-01-19",
							"Valor": 450000.00
						}
					],
					"TotalRegistros": 1
				}`))
			case "MOD_LICITACOES_PORTAL/VIS_EDITAIS_PREGAO":
				_, _ = w.Write([]byte(`{
					"Registros": [
						{
							"NumeroLicitacao": "PE 12/2025",
							"Ano": 2025,
							"Modalidade": "Pregão Eletrônico",
							"Situacao": "Homologada",
							"DataAbertura": "2025-02-05",
							"Objeto": "Aquisição de ambulâncias",
							"ValorEstimado": 600000.00,
							"ValorHomologado": 580000.00
						}
					],
					"TotalRegistros": 1
				}`))
			default:
				http.NotFound(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	baseWithFrag := srv.URL + "/#/inicio"

	// 1. Servidores
	servRes, err := transparencia.QuerySmarapdServidores(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: baseWithFrag,
		Limit:   15,
	})
	if err != nil {
		t.Fatalf("QuerySmarapdServidores erro: %v", err)
	}
	if servRes.Plataforma != "smarapd_pai" || servRes.TotalRegistros != 320 || len(servRes.Servidores) != 1 {
		t.Fatalf("ServidoresResult inesperado: %+v", servRes)
	}
	s0 := servRes.Servidores[0]
	if s0.CPFParcial != "***.456.789-**" || s0.SalarioBase != 7500.00 || s0.SalarioBruto != 9200.50 || s0.SalarioLiquido != 7500.25 {
		t.Errorf("servidor[0] inesperado: %+v", s0)
	}

	// 2. Despesas
	despRes, err := transparencia.QuerySmarapdDespesas(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: baseWithFrag,
		Ano:     2025,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QuerySmarapdDespesas erro: %v", err)
	}
	if len(despRes.Despesas) != 2 {
		t.Fatalf("DespesasResult len=%d, esperava 2", len(despRes.Despesas))
	}
	if despRes.Despesas[0].DocumentoFornecedor != "11.222.333/0001-44" {
		t.Errorf("CNPJ não deveria ser mascarado: %q", despRes.Despesas[0].DocumentoFornecedor)
	}
	if despRes.Despesas[1].DocumentoFornecedor != "***.654.321-**" {
		t.Errorf("CPF em despesa deveria estar mascarado: %q", despRes.Despesas[1].DocumentoFornecedor)
	}

	// 3. Receitas
	recRes, err := transparencia.QuerySmarapdReceitas(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: baseWithFrag,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QuerySmarapdReceitas erro: %v", err)
	}
	if len(recRes.Receitas) != 1 || recRes.Receitas[0].ValorArrecadado != 8450320.10 {
		t.Errorf("ReceitasResult inesperado: %+v", recRes)
	}

	// 4. Contratos (descoberto via MenuPortal)
	contRes, err := transparencia.QuerySmarapdContratos(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: baseWithFrag,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QuerySmarapdContratos erro: %v", err)
	}
	if len(contRes.Contratos) != 1 || contRes.Contratos[0].Numero != "045/2025" || contRes.Contratos[0].Valor != 450000.00 {
		t.Errorf("ContratosResult inesperado: %+v", contRes)
	}

	// 5. Licitações (descoberto via MenuPortal)
	licRes, err := transparencia.QuerySmarapdLicitacoes(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: baseWithFrag,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QuerySmarapdLicitacoes erro: %v", err)
	}
	if len(licRes.Licitacoes) != 1 || licRes.Licitacoes[0].Numero != "PE 12/2025" || licRes.Licitacoes[0].ValorHomologado != 580000.00 {
		t.Errorf("LicitacoesResult inesperado: %+v", licRes)
	}

	if len(filterRequests) != 5 {
		t.Errorf("esperava 5 chamadas a /paiportalserver/modulovisao/filter, obteve %d", len(filterRequests))
	}
}
