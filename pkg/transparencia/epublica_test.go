package transparencia_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gibavargas/mcp-brasil/pkg/transparencia"
)

func TestDiscoverEPublica(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	t.Run("match_compact_uf", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/rest/saojose_sc/public/user/attr/exercicios" {
				_, _ = w.Write([]byte(`{"exercicio":{"ano":2025,"entidade":"PREFEITURA MUNICIPAL DE SÃO JOSÉ"}}`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		transparencia.SetEPublicaBaseURL(srv.URL)
		t.Cleanup(func() { transparencia.SetEPublicaBaseURL("") })

		m := &transparencia.Municipio{CodigoIBGE: "4216602", Nome: "São José", UF: "SC", Slug: "sao-jose"}
		res, ok := transparencia.DiscoverEPublica(ctx, client, m)
		if !ok || res == nil {
			t.Fatalf("esperava descoberta positiva para São José/SC")
		}
		if res.Plataforma != "epublica_portal" {
			t.Errorf("Plataforma = %q, esperado epublica_portal", res.Plataforma)
		}
		if res.URLResolvida != srv.URL {
			t.Errorf("URLResolvida = %q, esperado %q", res.URLResolvida, srv.URL)
		}
		if res.TenantID != "saojose_sc" {
			t.Errorf("TenantID = %q, esperado saojose_sc", res.TenantID)
		}
		if res.EntidadeNome != "PREFEITURA MUNICIPAL DE SÃO JOSÉ" {
			t.Errorf("EntidadeNome = %q", res.EntidadeNome)
		}
		if !res.ServidoresSuportado {
			t.Errorf("esperava ServidoresSuportado=true")
		}
	})

	t.Run("fallback_slug_underscore_array_response", func(t *testing.T) {
		var seenPaths []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seenPaths = append(seenPaths, r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/rest/porto_alegre/public/user/attr/exercicios" {
				_, _ = w.Write([]byte(`[{"exercicio":{"ano":2025,"entidade":{"nome":"MUNICIPIO DE PORTO ALEGRE"}}}]`))
				return
			}
			if r.URL.Path == "/rest/portoalegre_rs/public/user/attr/exercicios" {
				_, _ = w.Write([]byte(`{"exercicio":{}}`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		transparencia.SetEPublicaBaseURL(srv.URL)
		t.Cleanup(func() { transparencia.SetEPublicaBaseURL("") })

		m := &transparencia.Municipio{CodigoIBGE: "4314902", Nome: "Porto Alegre", UF: "RS", Slug: "porto-alegre"}
		res, ok := transparencia.DiscoverEPublica(ctx, client, m)
		if !ok || res == nil {
			t.Fatalf("esperava descoberta positiva para porto_alegre; paths=%v", seenPaths)
		}
		if res.TenantID != "porto_alegre" {
			t.Errorf("TenantID = %q, esperado porto_alegre", res.TenantID)
		}
		if res.EntidadeNome != "MUNICIPIO DE PORTO ALEGRE" {
			t.Errorf("EntidadeNome = %q", res.EntidadeNome)
		}
		if len(seenPaths) != 4 {
			t.Errorf("esperava testar 4 candidatos em ordem, testou %d: %v", len(seenPaths), seenPaths)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer srv.Close()

		transparencia.SetEPublicaBaseURL(srv.URL)
		t.Cleanup(func() { transparencia.SetEPublicaBaseURL("") })

		m := &transparencia.Municipio{CodigoIBGE: "1100015", Nome: "Alta Floresta", UF: "RO", Slug: "alta-floresta"}
		res, ok := transparencia.DiscoverEPublica(ctx, client, m)
		if ok || res != nil {
			t.Fatalf("esperava ok=false, obteve %+v", res)
		}
	})
}

func TestProbeEPublicaURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/epublica-portal/rest/chapeco_sc/public/user/attr/exercicios" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"exercicio":{"entidade":"PREFEITURA DE CHAPECÓ"}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	alias, ent, ok := transparencia.ProbeEPublicaURL(ctx, client, srv.URL+"/epublica-portal/rest/chapeco_sc/despesa/empenhos")
	if !ok {
		t.Fatalf("esperava ProbeEPublicaURL=true")
	}
	if alias != "chapeco_sc" || ent != "PREFEITURA DE CHAPECÓ" {
		t.Errorf("resultado inesperado: alias=%q ent=%q", alias, ent)
	}

	if _, _, ok := transparencia.ProbeEPublicaURL(ctx, client, srv.URL+"/sem-rest"); ok {
		t.Errorf("esperava ProbeEPublicaURL=false para URL sem /rest/")
	}
}

func TestEPublicaQueriesMock(t *testing.T) {
	var seenQueries []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenQueries = append(seenQueries, r.URL.Path+"?"+r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Query().Get("exercicio") != "2025" || r.URL.Query().Get("page") != "1" {
			http.Error(w, "parametros invalidos", http.StatusBadRequest)
			return
		}

		switch r.URL.Path {
		case "/rest/saojose_sc/gestaoDePessoal/servidores/listAll":
			_, _ = w.Write([]byte(`{
				"content": [
					{
						"nome": "CARLOS PEREIRA",
						"matricula": "10293",
						"cpf": "12345678901",
						"cargo": "Analista de Sistemas",
						"lotacao": "Tecnologia da Informação",
						"vinculo": "Estatutário",
						"situacao": "Ativo",
						"salarioBase": 5000.00,
						"salarioBruto": 6500.50,
						"descontos": 1200.25,
						"salarioLiquido": 5300.25,
						"competencia": "03/2025"
					},
					{
						"nome": "MARIA SOUZA",
						"matricula": "10294",
						"cpfFormatado": "***.987.654-**",
						"cargo": "Professora",
						"salarioBruto": 4200.00
					}
				],
				"totalElements": 200
			}`))

		case "/rest/saojose_sc/despesa/empenhos/listAll":
			_, _ = w.Write([]byte(`{
				"content": [
					{
						"numeroEmpenho": "450/2025",
						"exercicio": 2025,
						"dataEmpenho": "2025-02-10",
						"fornecedor": "EMPRESA TECH LTDA",
						"cnpjCpf": "12.345.678/0001-99",
						"orgao": "Secretaria de Administração",
						"elementoDespesa": "339039",
						"historico": "Serviços de TI",
						"valorEmpenhado": 15000.00,
						"valorLiquidado": 10000.00,
						"valorPago": 10000.00
					},
					{
						"numeroEmpenho": "451/2025",
						"exercicio": 2025,
						"fornecedor": "JOAO CREDOR",
						"cpfCnpj": "11122233344",
						"historico": "Diária",
						"valorEmpenhado": 500.00
					}
				],
				"totalElements": 50
			}`))

		case "/rest/saojose_sc/receita/arrecadada/listAll":
			_, _ = w.Write([]byte(`[
				{
					"codigo": "1.1.1.3.03.1.1",
					"descricao": "IRRF - Rendimento do Trabalho",
					"categoria": "Receitas Correntes",
					"data": "2025-02-28",
					"valorPrevisto": 1200000.00,
					"valorArrecadado": 980500.40
				}
			]`))

		case "/rest/saojose_sc/compras/licitacao/listAll":
			_, _ = w.Write([]byte(`{
				"data": [
					{
						"numeroLicitacao": "PE 12/2025",
						"ano": 2025,
						"modalidade": "Pregão Eletrônico",
						"situacao": "Homologada",
						"dataAbertura": "2025-01-20",
						"objeto": "Aquisição de computadores",
						"valorEstimado": 120000.00,
						"valorHomologado": 112500.00
					}
				],
				"total": 12
			}`))

		case "/rest/saojose_sc/compras/contrato/listAll":
			_, _ = w.Write([]byte(`{
				"content": [
					{
						"numeroContrato": "CT 08/2025",
						"ano": 2025,
						"fornecedor": "EMPRESA TECH LTDA",
						"cnpjCpf": "12345678000199",
						"objeto": "Manutenção de rede",
						"situacao": "Vigente",
						"dataAssinatura": "2025-02-01",
						"vigenciaInicio": "2025-02-01",
						"vigenciaFim": "2026-01-31",
						"valorContrato": 95000.00
					}
				],
				"totalElements": 8
			}`))

		case "/rest/erro_500/despesa/empenhos/listAll":
			http.Error(w, "erro interno", http.StatusInternalServerError)

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	m := &transparencia.Municipio{CodigoIBGE: "4216602", Nome: "São José", UF: "SC"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	params := transparencia.PortalQueryParams{
		BaseURL:  srv.URL,
		TenantID: "saojose_sc",
		Ano:      2025,
		Limit:    10,
	}

	// 1. Servidores
	servRes, err := transparencia.QueryEPublicaServidores(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryEPublicaServidores erro: %v", err)
	}
	if servRes.Plataforma != "epublica_portal" || servRes.TotalRegistros != 200 || len(servRes.Servidores) != 2 {
		t.Fatalf("ServidoresResult inesperado: %+v", servRes)
	}
	if servRes.Servidores[0].CPFParcial != "***.456.789-**" {
		t.Errorf("CPF não mascarado corretamente: %q", servRes.Servidores[0].CPFParcial)
	}
	if servRes.Servidores[1].CPFParcial != "***.987.654-**" {
		t.Errorf("CPF já mascarado alterado: %q", servRes.Servidores[1].CPFParcial)
	}
	if servRes.Servidores[0].SalarioLiquido != 5300.25 {
		t.Errorf("SalarioLiquido inesperado: %v", servRes.Servidores[0].SalarioLiquido)
	}

	// 2. Despesas (testando também extração automática do alias a partir de BaseURL com /rest/{alias})
	despParams := transparencia.PortalQueryParams{
		BaseURL: srv.URL + "/rest/saojose_sc/despesa/empenhos",
		Ano:     2025,
		Limit:   10,
	}
	despRes, err := transparencia.QueryEPublicaDespesas(ctx, client, m, despParams)
	if err != nil {
		t.Fatalf("QueryEPublicaDespesas erro: %v", err)
	}
	if despRes.Plataforma != "epublica_portal" || despRes.TotalRegistros != 50 || len(despRes.Despesas) != 2 {
		t.Fatalf("DespesasResult inesperado: %+v", despRes)
	}
	if despRes.Despesas[0].DocumentoFornecedor != "12.345.678/0001-99" {
		t.Errorf("CNPJ alterado indevidamente: %q", despRes.Despesas[0].DocumentoFornecedor)
	}
	if despRes.Despesas[1].DocumentoFornecedor != "***.222.333-**" {
		t.Errorf("CPF de credor não mascarado: %q", despRes.Despesas[1].DocumentoFornecedor)
	}

	// 3. Receitas
	recRes, err := transparencia.QueryEPublicaReceitas(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryEPublicaReceitas erro: %v", err)
	}
	if len(recRes.Receitas) != 1 || recRes.Receitas[0].ValorArrecadado != 980500.40 {
		t.Fatalf("ReceitasResult inesperado: %+v", recRes)
	}

	// 4. Licitações
	licRes, err := transparencia.QueryEPublicaLicitacoes(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryEPublicaLicitacoes erro: %v", err)
	}
	if licRes.TotalRegistros != 12 || len(licRes.Licitacoes) != 1 || licRes.Licitacoes[0].ValorHomologado != 112500.00 {
		t.Fatalf("LicitacoesResult inesperado: %+v", licRes)
	}

	// 5. Contratos
	contRes, err := transparencia.QueryEPublicaContratos(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryEPublicaContratos erro: %v", err)
	}
	if contRes.TotalRegistros != 8 || len(contRes.Contratos) != 1 || contRes.Contratos[0].Valor != 95000.00 {
		t.Fatalf("ContratosResult inesperado: %+v", contRes)
	}

	// 6. Valida parâmetros de query enviados
	foundSize10 := false
	for _, q := range seenQueries {
		if strings.Contains(q, "size=10") {
			foundSize10 = true
		}
	}
	if !foundSize10 {
		t.Errorf("esperava size=10 nas URLs consultadas: %v", seenQueries)
	}

	// 7. Erro quando TenantID ausente e BaseURL sem /rest/
	if _, err := transparencia.QueryEPublicaServidores(ctx, client, m, transparencia.PortalQueryParams{BaseURL: srv.URL, Ano: 2025}); err == nil {
		t.Errorf("esperava erro quando TenantID não informado")
	}

	// 8. Erro quando API retorna HTTP 500
	if _, err := transparencia.QueryEPublicaDespesas(ctx, client, m, transparencia.PortalQueryParams{BaseURL: srv.URL, TenantID: "erro_500", Ano: 2025}); err == nil {
		t.Errorf("esperava erro para HTTP 500")
	}
}
