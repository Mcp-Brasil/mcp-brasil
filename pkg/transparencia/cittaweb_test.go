package transparencia_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Mcp-Brasil/mcp-brasil/pkg/transparencia"
)

func TestNormalizeCittaWebAPIBase(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{
			in:   "https://almirante.cittaweb.com.br/citta/#/transparencia",
			want: "https://almirante.cittaweb.com.br/transparencia/api",
		},
		{
			in:   "https://camargo.cittaweb.com.br:443/citta",
			want: "https://camargo.cittaweb.com.br:443/transparencia/api",
		},
		{
			in:   "https://almirante.cittaweb.com.br/transparencia/api",
			want: "https://almirante.cittaweb.com.br/transparencia/api",
		},
		{
			in:   "https://almirante.cittaweb.com.br/transparencia/api/public/todasunidades",
			want: "https://almirante.cittaweb.com.br/transparencia/api",
		},
		{
			in:   "https://almirante.cittaweb.com.br",
			want: "https://almirante.cittaweb.com.br/transparencia/api",
		},
		{
			in:   "http://127.0.0.1:8080",
			want: "http://127.0.0.1:8080",
		},
		{
			in:   "http://localhost:9090/citta/#/transparencia",
			want: "http://localhost:9090/transparencia/api",
		},
		{
			in:   "",
			want: "",
		},
	}

	for _, tc := range cases {
		if got := transparencia.NormalizeCittaWebAPIBase(tc.in); got != tc.want {
			t.Errorf("NormalizeCittaWebAPIBase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestProbeCittaWeb(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	t.Run("prefers_prefeitura_or_municipio", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/transparencia/api/public/todasunidades" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[
					{"chave": 2, "codigo": "002", "descricao": "Câmara Municipal de Camargo"},
					{"chave": 5, "codigo": "001", "descricao": "Prefeitura Municipal de Camargo"},
					{"chave": 9, "codigo": "003", "descricao": "Fundo de Previdência"}
				]`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		apiBase, ugID, entName, ok := transparencia.ProbeCittaWeb(ctx, client, srv.URL+"/citta/#/transparencia")
		if !ok || ugID != 5 || entName != "Prefeitura Municipal de Camargo" || apiBase != srv.URL+"/transparencia/api" {
			t.Fatalf("ProbeCittaWeb = (%q, %d, %q, %v), esperava ugID=5", apiBase, ugID, entName, ok)
		}
	})

	t.Run("prefers_municipio_and_falls_back_to_first", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/public/todasunidades":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[
					{"chave": 10, "codigo": "010", "descricao": "Autarquia"},
					{"chave": 12, "codigo": "012", "descricao": "Município de Almirante"}
				]`))
			case "/fallback/transparencia/api/public/todasunidades":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[{"chave": 3, "codigo": "003", "descricao": "Fundo Municipal de Saúde"}]`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		_, ugID, entName, ok := transparencia.ProbeCittaWeb(ctx, client, srv.URL)
		if !ok || ugID != 12 || entName != "Município de Almirante" {
			t.Fatalf("ProbeCittaWeb município = (%d, %q, %v), esperava ugID=12", ugID, entName, ok)
		}

		apiBase2, ugID2, entName2, ok2 := transparencia.ProbeCittaWeb(ctx, client, srv.URL+"/fallback/citta")
		if !ok2 || ugID2 != 3 || entName2 != "Fundo Municipal de Saúde" || apiBase2 != srv.URL+"/fallback/transparencia/api" {
			t.Fatalf("ProbeCittaWeb fallback = (%q, %d, %q, %v), esperava ugID=3", apiBase2, ugID2, entName2, ok2)
		}
	})

	t.Run("negative", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if apiBase, ugID, _, ok := transparencia.ProbeCittaWeb(ctx, client, srv.URL); ok || apiBase != "" || ugID != 0 {
			t.Fatalf("esperava false para 404, obteve (%q, %d, %v)", apiBase, ugID, ok)
		}
	})
}

func TestQueryCittaWebAllModules(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "4303558", Nome: "Camargo", UF: "RS"}

	var (
		seenDespQuery string
		seenRecQuery  string
		seenLicQuery  string
		seenContQuery string
		seenRemQuery  string
		seenFunQuery  string
		remun404      bool
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/transparencia/api/public/todasunidades":
			_, _ = w.Write([]byte(`[
				{"chave": 7, "codigo": "001", "descricao": "PREFEITURA MUNICIPAL DE CAMARGO"}
			]`))
		case "/transparencia/api/public/despesas/elementos":
			seenDespQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`[
				{
					"codigo": "3.3.90.30",
					"descricao": "MATERIAL DE CONSUMO",
					"orcado": 150000.00,
					"atualizado": 175000.00,
					"empenhado": 120000.50,
					"liquidado": 110000.25,
					"pago": 100000.00
				},
				{
					"codigo": "3.3.90.39",
					"descricao": "OUTROS SERVICOS DE TERCEIROS",
					"orcado": 80000.00,
					"atualizado": 80000.00,
					"empenhado": 40000.00,
					"liquidado": 30000.00,
					"pago": 25000.00
				}
			]`))
		case "/transparencia/api/public/receitas/balancete":
			seenRecQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`{
				"totalElements": 15,
				"data": [
					{
						"codigo": "11125001",
						"codigoFormatado": "1.1.1.2.50.0.1",
						"descricao": "IPTU - Principal",
						"orcado": 500000.00,
						"atualizado": 520000.00,
						"arrecadado": 485000.75
					}
				]
			}`))
		case "/transparencia/api/public/licitacoes/lista":
			seenLicQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`{
				"totalElements": 8,
				"data": [
					{
						"licNumero": "014/2025",
						"licAno": 2025,
						"modalidade": {"modSigla": "PE", "modNome": "Pregão Eletrônico"},
						"objeto": "Aquisição de medicamentos",
						"valor": 250000.00,
						"data": "2025-03-10",
						"status": "A",
						"situacao": "Homologada"
					}
				]
			}`))
		case "/transparencia/api/public/contratos/lista":
			seenContQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`{
				"data": [
					{
						"numero": "042/2025",
						"ano": 2025,
						"fornecedor": "JOAO DA SILVA",
						"cpfCnpj": "12345678901",
						"objeto": "Prestação de serviços médicos",
						"situacao": "Vigente",
						"dataAssinatura": "2025-01-15",
						"vigenciaInicio": "2025-01-15",
						"vigenciaFim": "2025-12-31",
						"valor": 96000.00
					},
					{
						"numero": "043/2025",
						"ano": 2025,
						"fornecedor": "TECH SOLUCOES LTDA",
						"cpfCnpj": "11.222.333/0001-44",
						"objeto": "Licença de software",
						"situacao": "Vigente",
						"valor": 48000.00
					}
				]
			}`))
		case "/transparencia/api/public/pessoal/remuneracao":
			seenRemQuery = r.URL.RawQuery
			if remun404 {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(`{
				"data": [
					{
						"nome": "CARLA SOUZA",
						"matricula": "331",
						"cpf": "98765432100",
						"cargo": "Médica",
						"lotacao": "Saúde",
						"salarioBase": 8500.00,
						"salarioBruto": 10200.00,
						"descontos": 2100.00,
						"salarioLiquido": 8100.00,
						"competencia": "04/2025"
					}
				]
			}`))
		case "/transparencia/api/public/pessoal/folha/funcoes":
			seenFunQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`[
				{
					"codigo": "10",
					"funcao": "Professor",
					"descricao": "Professor Ensino Fundamental",
					"salarioBruto": 4500.00,
					"salarioLiquido": 3900.00
				}
			]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// BaseURL no formato portal SPA -> auto-normaliza para /transparencia/api e resolve ugID=7 via ProbeCittaWeb
	p := transparencia.PortalQueryParams{
		BaseURL: srv.URL + "/citta/#/transparencia",
		Ano:     2025,
		Limit:   10,
	}

	desp, err := transparencia.QueryCittaWebDespesas(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryCittaWebDespesas erro: %v", err)
	}
	if desp.Plataforma != "cittaweb_api" || !desp.Disponivel || len(desp.Despesas) != 2 {
		t.Fatalf("DespesasResult inesperado: %+v", desp)
	}
	if desp.Despesas[0].ValorEmpenhado != 120000.50 || desp.Despesas[0].ValorPago != 100000.00 {
		t.Errorf("Despesas[0] valores inesperados: %+v", desp.Despesas[0])
	}
	if seenDespQuery != "exercicio=2025&unidadeGestora=7&mesInicial=1&mesFim=12" {
		t.Errorf("query despesas inesperada: %q", seenDespQuery)
	}

	// Testa Limit e NomeFiltro em despesas
	despFiltered, err := transparencia.QueryCittaWebDespesas(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL:          srv.URL + "/transparencia/api",
		CodAdministracao: 7,
		Ano:              2025,
		NomeFiltro:       "CONSUMO",
		Limit:            1,
	})
	if err != nil || len(despFiltered.Despesas) != 1 || despFiltered.Despesas[0].NumeroEmpenho != "3.3.90.30" {
		t.Fatalf("filtro despesas inesperado: %+v err=%v", despFiltered, err)
	}

	rec, err := transparencia.QueryCittaWebReceitas(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryCittaWebReceitas erro: %v", err)
	}
	if rec.TotalRegistros != 15 || len(rec.Receitas) != 1 || rec.Receitas[0].Codigo != "1.1.1.2.50.0.1" || rec.Receitas[0].ValorArrecadado != 485000.75 {
		t.Fatalf("ReceitasResult inesperado: %+v", rec)
	}
	if seenRecQuery != "page=0&size=10&exercicio=2025&unidadeGestora=7&mes=12&nivel=1" {
		t.Errorf("query receitas inesperada: %q", seenRecQuery)
	}

	lic, err := transparencia.QueryCittaWebLicitacoes(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryCittaWebLicitacoes erro: %v", err)
	}
	if lic.TotalRegistros != 8 || len(lic.Licitacoes) != 1 || lic.Licitacoes[0].Modalidade != "Pregão Eletrônico" || lic.Licitacoes[0].Situacao != "Homologada" {
		t.Fatalf("LicitacoesResult inesperado: %+v", lic)
	}
	if seenLicQuery != "page=0&size=10" {
		t.Errorf("query licitacoes inesperada: %q", seenLicQuery)
	}

	ct, err := transparencia.QueryCittaWebContratos(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryCittaWebContratos erro: %v", err)
	}
	if len(ct.Contratos) != 2 || ct.Contratos[0].DocumentoFornecedor != "***.456.789-**" || ct.Contratos[1].DocumentoFornecedor != "11.222.333/0001-44" {
		t.Fatalf("ContratosResult inesperado: %+v", ct)
	}
	if seenContQuery != "unidadeGestora=7&page=0&size=10" {
		t.Errorf("query contratos inesperada: %q", seenContQuery)
	}

	serv, err := transparencia.QueryCittaWebServidores(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryCittaWebServidores (remuneracao) erro: %v", err)
	}
	if len(serv.Servidores) != 1 || serv.Servidores[0].CPFParcial != "***.654.321-**" || serv.Servidores[0].SalarioLiquido != 8100.00 {
		t.Fatalf("ServidoresResult (remuneracao) inesperado: %+v", serv)
	}
	if seenRemQuery != "unidadeGestora=7" {
		t.Errorf("query remuneracao inesperada: %q", seenRemQuery)
	}

	// Testa fallback para /public/pessoal/folha/funcoes quando /remuneracao falha
	remun404 = true
	pFun := p
	pFun.Mes = 5
	servFun, err := transparencia.QueryCittaWebServidores(ctx, client, m, pFun)
	if err != nil {
		t.Fatalf("QueryCittaWebServidores (folha/funcoes) erro: %v", err)
	}
	if len(servFun.Servidores) != 1 || servFun.Servidores[0].Cargo != "Professor" || servFun.Servidores[0].Competencia != "05/2025" {
		t.Fatalf("ServidoresResult (folha/funcoes) inesperado: %+v", servFun)
	}
	if seenFunQuery != "exercicio=2025&unidadeGestora=7&mes=5" {
		t.Errorf("query folha/funcoes inesperada: %q", seenFunQuery)
	}
}
