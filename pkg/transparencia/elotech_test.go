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

func TestProbeElotech(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	t.Run("openapi_v3", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v3/api-docs" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"openapi":"3.0.1",
					"info":{"title":"Web Services","contact":{"name":"Elotech Gestão Pública"}},
					"servers":[{"url":"https://exemplo.oxy.elotech.com.br/portaltransparencia-api"}],
					"paths":{"/api/servidores":{}}
				}`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if !transparencia.ProbeElotech(ctx, client, srv.URL) {
			t.Fatalf("esperava ProbeElotech=true via /v3/api-docs")
		}
	})

	t.Run("swagger_ui_fallback", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/swagger-ui.html" {
				_, _ = w.Write([]byte(`<html><title>Elotech PortalTransparencia-API</title><div id="swagger-ui"></div></html>`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if !transparencia.ProbeElotech(ctx, client, srv.URL) {
			t.Fatalf("esperava ProbeElotech=true via /swagger-ui.html")
		}
	})

	t.Run("servidores_fallback", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/servidores" && r.Header.Get("entidade") != "" && r.Header.Get("exercicio") != "" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"content":[{"matricula":1}],"totalElements":1}`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if !transparencia.ProbeElotech(ctx, client, srv.URL) {
			t.Fatalf("esperava ProbeElotech=true via /api/servidores")
		}
	})

	t.Run("non_elotech", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if transparencia.ProbeElotech(ctx, client, srv.URL) {
			t.Fatalf("esperava ProbeElotech=false para servidor não-Elotech")
		}
	})

	t.Run("spa_catchall_html_rejected", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`<html><title>PortalTransparencia Generico</title></html>`))
		}))
		defer srv.Close()

		if transparencia.ProbeElotech(ctx, client, srv.URL) {
			t.Fatalf("esperava ProbeElotech=false para SPA catch-all HTML")
		}
	})
}

func TestElotechQueriesMock(t *testing.T) {
	var seenServidorHeaderEnt, seenServidorHeaderEx, seenSearchRSQL string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/servidores":
			seenServidorHeaderEnt = r.Header.Get("entidade")
			seenServidorHeaderEx = r.Header.Get("exercicio")
			if r.URL.Query().Get("exercicio") != "2025" {
				_, _ = w.Write([]byte(`{"content":[],"totalElements":0,"totalPages":0}`))
				return
			}
			_, _ = w.Write([]byte(`{
				"content": [
					{
						"matricula": 56141,
						"nome": "ADILSON TOGNATO",
						"cpf": "12345678901",
						"horasSemanais": 40,
						"descricaoCargo": "Motorista",
						"descricaoLotacao": "Obras, Serv Urb e Transp - EF",
						"vinculo": "FUNCIONARIO",
						"situacao": "ATIVO",
						"dataAdmissao": "2012-02-07",
						"remuneracao": 6765.02,
						"dadosFinanceiros": []
					},
					{
						"matricula": 99001,
						"nome": "MARIA FINANCEIRO",
						"cpfFormatado": "***.222.333-**",
						"descricaoCargo": "Professora",
						"descricaoLotacao": "Educacao",
						"situacao": "ATIVO",
						"dadosFinanceiros": [
							{
								"anoCompetencia": 2025,
								"mesCompetencia": 12,
								"salarioBase": 4000.00,
								"vencimentos": 5200.50,
								"descontos": 800.25
							}
						]
					}
				],
				"totalElements": 145,
				"totalPages": 8
			}`))

		case "/empenhos/lista":
			seenSearchRSQL = r.URL.Query().Get("search")
			if !strings.Contains(seenSearchRSQL, "id.exercicio=='2025'") {
				_, _ = w.Write([]byte(`{"content":[],"totalElements":0,"totalPages":0}`))
				return
			}
			_, _ = w.Write([]byte(`{
				"content": [
					{
						"id": -79963,
						"entidade": 1,
						"empenho": 1161,
						"exercicio": 2025,
						"empenhoExercicio": "1161/2025",
						"data": "2025-02-14",
						"fornecedor": 263196,
						"nome": "04.321.321/0001-49 - FUNDO ESTADUAL DO MEIO AMBIENTE-FEMA",
						"historico": "Empenho Orcamentário",
						"valorEmpenhado": 3753.65,
						"valorLiquidado": 3753.65,
						"valorPago": 3753.65,
						"fonteRecurso": "0 - Recursos Ordinários (Livres)"
					},
					{
						"id": -79964,
						"entidade": 1,
						"empenho": 1162,
						"exercicio": 2025,
						"empenhoExercicio": "1162/2025",
						"data": "2025-02-15",
						"fornecedor": 263197,
						"nome": "123.456.789-01 - JOAO DA SILVA",
						"historico": "Diária",
						"valorEmpenhado": 500.00,
						"valorLiquidado": 500.00,
						"valorPago": 500.00,
						"fonteRecurso": "0 - Recursos Ordinários (Livres)"
					}
				],
				"totalElements": 9904,
				"totalPages": 4952
			}`))

		case "/api/receitas":
			_, _ = w.Write([]byte(`[
				{
					"entidade": "",
					"receita": "1",
					"descricao": "Receitas Correntes",
					"fonteRecurso": "",
					"descricaoFonte": "Recursos Livres",
					"valorOrcado": 31724668.21,
					"valorPrevisto": 0,
					"valorArrecadado": 44860915.31,
					"data": "2025-12-31"
				},
				{
					"entidade": "",
					"receita": "11",
					"descricao": "Impostos, Taxas e Contribuições de Melhoria",
					"valorOrcado": 731677.02,
					"valorArrecadado": 3124917.99
				}
			]`))

		case "/api/licitacoes":
			_, _ = w.Write([]byte(`{
				"content": [
					{
						"entidade": 1,
						"nomeEntidade": "MUNICIPIO DE QUINTA DO SOL",
						"exercicio": 2025,
						"licitacao": 128,
						"displayLicitacao": "128/2025",
						"descricao": "Dispensa",
						"numeroLicitacao": 128,
						"dataPublicacao": "2025-12-23",
						"dataAbertura": "2025-12-26",
						"tipoCancelamento": "Homologada",
						"objeto": "AQUISIÇÃO DE FOGOS DE ARTIFÍCIOS",
						"valorMaximo": 28900.00,
						"totalVencido": 28900.00
					}
				],
				"totalElements": 128,
				"totalPages": 7
			}`))

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	m := &transparencia.Municipio{CodigoIBGE: "4121109", Nome: "Quinta do Sol", UF: "PR"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	// 1. Servidores
	servRes, err := transparencia.QueryElotechServidores(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL:    srv.URL,
		Ano:        2025,
		Mes:        12,
		NomeFiltro: "ADILSON",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("QueryElotechServidores erro: %v", err)
	}
	if seenServidorHeaderEnt != "1" || seenServidorHeaderEx != "2025" {
		t.Errorf("headers inesperados: entidade=%q exercicio=%q", seenServidorHeaderEnt, seenServidorHeaderEx)
	}
	if servRes.Plataforma != "elotech" || servRes.TotalRegistros != 145 || len(servRes.Servidores) != 2 {
		t.Fatalf("ServidoresResult inesperado: %+v", servRes)
	}
	s0 := servRes.Servidores[0]
	if s0.CPFParcial != "***.456.789-**" {
		t.Errorf("CPF não mascarado corretamente: %q", s0.CPFParcial)
	}
	if s0.SalarioBruto != 6765.02 || s0.CargaHoraria != "40h" {
		t.Errorf("campos de servidor inesperados: %+v", s0)
	}
	s1 := servRes.Servidores[1]
	if s1.SalarioBase != 4000.00 || s1.SalarioBruto != 5200.50 || s1.Descontos != 800.25 || s1.SalarioLiquido != 4400.25 || s1.Competencia != "12/2025" {
		t.Errorf("dadosFinanceiros inesperados: %+v", s1)
	}

	// 2. Despesas
	despRes, err := transparencia.QueryElotechDespesas(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL:    srv.URL,
		Ano:        2025,
		NomeFiltro: "FEMA",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("QueryElotechDespesas erro: %v", err)
	}
	if !strings.Contains(seenSearchRSQL, "id.exercicio=='2025'") || !strings.Contains(seenSearchRSQL, "fornecedor=='*FEMA*'") {
		t.Errorf("RSQL search inesperado: %q", seenSearchRSQL)
	}
	if despRes.TotalRegistros != 9904 || len(despRes.Despesas) != 2 {
		t.Fatalf("DespesasResult inesperado: %+v", despRes)
	}
	if despRes.Despesas[0].DocumentoFornecedor != "04.321.321/0001-49" || despRes.Despesas[0].Fornecedor != "FUNDO ESTADUAL DO MEIO AMBIENTE-FEMA" {
		t.Errorf("despesa[0] inesperada: %+v", despRes.Despesas[0])
	}
	if despRes.Despesas[1].DocumentoFornecedor != "***.456.789-**" || despRes.Despesas[1].Fornecedor != "JOAO DA SILVA" {
		t.Errorf("despesa[1] CPF deveria estar mascarado: %+v", despRes.Despesas[1])
	}

	// 3. Receitas (testa limite=1 sobre array de 2 itens)
	recRes, err := transparencia.QueryElotechReceitas(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Ano:     2025,
		Limit:   1,
	})
	if err != nil {
		t.Fatalf("QueryElotechReceitas erro: %v", err)
	}
	if recRes.TotalRegistros != 2 || len(recRes.Receitas) != 1 {
		t.Fatalf("ReceitasResult inesperado: total=%d len=%d", recRes.TotalRegistros, len(recRes.Receitas))
	}
	if recRes.Receitas[0].Codigo != "1" || recRes.Receitas[0].ValorArrecadado != 44860915.31 {
		t.Errorf("receita[0] inesperada: %+v", recRes.Receitas[0])
	}

	// 4. Licitações
	licRes, err := transparencia.QueryElotechLicitacoes(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Ano:     2025,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QueryElotechLicitacoes erro: %v", err)
	}
	if licRes.TotalRegistros != 128 || len(licRes.Licitacoes) != 1 {
		t.Fatalf("LicitacoesResult inesperado: %+v", licRes)
	}
	l0 := licRes.Licitacoes[0]
	if l0.Numero != "128/2025" || l0.Modalidade != "Dispensa" || l0.Situacao != "Homologada" || l0.ValorEstimado != 28900.00 || l0.ValorHomologado != 28900.00 {
		t.Errorf("licitacao[0] inesperada: %+v", l0)
	}
}
