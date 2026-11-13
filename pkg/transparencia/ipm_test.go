package transparencia_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gibavargas/mcp-brasil/pkg/transparencia"
)

func TestProbeIPM(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	t.Run("despesas_orcadas_200_retorno", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/WCPDadosAbertos/despesasOrcadas" && r.URL.Query().Get("dataFinal") == "31/12/2025" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"retorno":[{"orgao":"01"}]}`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if !transparencia.ProbeIPM(ctx, client, srv.URL) {
			t.Fatalf("esperava ProbeIPM=true para despesasOrcadas HTTP 200 com retorno")
		}
	})

	t.Run("despesas_orcadas_201_retorno", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/WCPDadosAbertos/despesasOrcadas" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"retorno":[]}`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if !transparencia.ProbeIPM(ctx, client, srv.URL) {
			t.Fatalf("esperava ProbeIPM=true para despesasOrcadas HTTP 201 com retorno")
		}
	})

	t.Run("funcionarios_fallback_registros", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/transparencia-pessoal-funcionarios" && r.URL.Query().Get("pagina") == "1" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"registros":[{"nome":"JOAO"}]}`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if !transparencia.ProbeIPM(ctx, client, srv.URL) {
			t.Fatalf("esperava ProbeIPM=true via fallback /api/transparencia-pessoal-funcionarios")
		}
	})

	t.Run("html_captcha_rejected", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<!DOCTYPE html><html><body><div class="cf-turnstile">Just a moment... "retorno"</div></body></html>`))
		}))
		defer srv.Close()

		if transparencia.ProbeIPM(ctx, client, srv.URL) {
			t.Fatalf("esperava ProbeIPM=false quando portal retorna HTML/CAPTCHA")
		}
	})

	t.Run("not_found", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if transparencia.ProbeIPM(ctx, client, srv.URL) {
			t.Fatalf("esperava ProbeIPM=false para HTTP 404")
		}
	})
}

func TestIPMQueriesMock(t *testing.T) {
	var seenDespInicio, seenDespFim string
	var seenRecInicio, seenRecFim string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/transparencia-pessoal-funcionarios":
			if r.URL.Query().Get("pagina") != "1" {
				http.Error(w, "pagina invalida", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{
				"registros": [
					{
						"nome": "MARIA DA SILVA",
						"matricula": 1042,
						"cpf": "12345678901",
						"cargo": "Enfermeira",
						"funcao": "Coordenação UBS",
						"localTrabalho": "UBS Central",
						"centroCusto": "Fundo Municipal de Saúde",
						"regime": "Estatutário",
						"situacao": "Ativo",
						"admissao": "10/03/2018",
						"horasSemanais": "40",
						"salarioBase": "5.432,10",
						"entidade": "Prefeitura Municipal de Blumenau"
					},
					{
						"nome": "CARLOS SOUZA",
						"matricula": "2055",
						"cpf": "98765432100",
						"cargo": "Motorista",
						"localTrabalho": "Secretaria de Obras",
						"regime": "CLT",
						"situacao": "Ativo",
						"admissao": "01/02/2020",
						"horasSemanais": 44,
						"salarioBase": 3200.00,
						"entidade": "Prefeitura Municipal de Blumenau"
					}
				],
				"total": 2
			}`))

		case "/api/WCPDadosAbertos/despesas":
			seenDespInicio = r.URL.Query().Get("dataInicial")
			seenDespFim = r.URL.Query().Get("dataFinal")
			// IPM frequentemente retorna HTTP 201 Created neste endpoint
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"retorno": [
					{
						"numeroEmpenho": "450/2025",
						"dataEmpenho": "15/01/2025",
						"orgaoDescricao": "Secretaria Municipal de Educação",
						"unidadeDescricao": "Ensino Fundamental",
						"cpfCnpjCredor": "123.456.789-01",
						"nomeCredor": "JOSE PEREIRA MEI",
						"valorEmpenhado": "1.500,50",
						"valorLiquidado": 1500.50,
						"valorPago": 1200.00,
						"fonteRecursoDescricao": "Recursos Próprios"
					},
					{
						"numeroEmpenho": "451/2025",
						"dataEmpenho": "20/01/2025",
						"orgaoDescricao": "Secretaria Municipal de Saúde",
						"unidadeDescricao": "Atenção Básica",
						"cpfCnpjCredor": "12.345.678/0001-99",
						"nomeCredor": "DISTRIBUIDORA MEDICA LTDA",
						"valorEmpenhado": 25000.00,
						"valorLiquidado": 25000.00,
						"valorPago": 25000.00,
						"fonteRecursoDescricao": "SUS Federal"
					}
				]
			}`))

		case "/api/WCPDadosAbertos/receitas":
			seenRecInicio = r.URL.Query().Get("dataInicial")
			seenRecFim = r.URL.Query().Get("dataFinal")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"retorno": [
					{
						"codigoReceita": "1.1.1.2.50.00",
						"receitaDescricao": "IPTU - Imposto Predial e Territorial Urbano",
						"fonteRecursoDescricao": "Receita Própria",
						"data": "31/01/2025",
						"valorPrevisto": "1000000,00",
						"valorArrecadado": 450123.45
					}
				]
			}`))

		case "/api/transparencia/dados-abertos/v1/wco/licitacao":
			if r.URL.Query().Get("pagina") != "0" {
				http.Error(w, "pagina deve ser 0", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{
				"registros": [
					{
						"numeroLicitacao": "PE 12/2025",
						"ano": 2025,
						"modalidadeDescricao": "Pregão Eletrônico",
						"situacaoDescricao": "Homologada",
						"dataAbertura": "10/02/2025",
						"objeto": "Aquisição de material escolar",
						"valorEstimado": 150000.00,
						"valorHomologado": 139500.00
					}
				]
			}`))

		case "/api/transparencia/dados-abertos/v1/wco/contrato":
			if r.URL.Query().Get("pagina") != "0" {
				http.Error(w, "pagina deve ser 0", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{
				"registros": [
					{
						"numeroContrato": "CT-088/2025",
						"ano": 2025,
						"nomeContratado": "JOAO CONSULTORIA",
						"cpfCnpjContratado": "11122233344",
						"objeto": "Consultoria técnica em gestão pública",
						"situacaoDescricao": "Vigente",
						"dataAssinatura": "01/03/2025",
						"dataInicioVigencia": "01/03/2025",
						"dataFimVigencia": "01/03/2026",
						"valor": "48.000,00"
					}
				]
			}`))

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	m := &transparencia.Municipio{CodigoIBGE: "4202404", Nome: "Blumenau", UF: "SC", Slug: "blumenau"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	// 1. Servidores (com filtro case-insensitive)
	servRes, err := transparencia.QueryIPMServidores(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL:    srv.URL,
		NomeFiltro: "maria",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("QueryIPMServidores erro: %v", err)
	}
	if !servRes.Disponivel || servRes.Plataforma != "ipm_atende" || len(servRes.Servidores) != 1 {
		t.Fatalf("ServidoresResult inesperado: %+v", servRes)
	}
	s0 := servRes.Servidores[0]
	if s0.Nome != "MARIA DA SILVA" || s0.Matricula != "1042" || s0.CPFParcial != "***.456.789-**" {
		t.Errorf("identificação de servidor inesperada: %+v", s0)
	}
	if s0.Cargo != "Enfermeira" || s0.Funcao != "Coordenação UBS" || s0.LocalTrabalho != "UBS Central" ||
		s0.Lotacao != "Fundo Municipal de Saúde" || s0.Vinculo != "Estatutário" || s0.Situacao != "Ativo" ||
		s0.DataAdmissao != "10/03/2018" || s0.CargaHoraria != "40" || s0.SalarioBase != 5432.10 ||
		s0.Orgao != "Prefeitura Municipal de Blumenau" {
		t.Errorf("campos de servidor IPM inesperados: %+v", s0)
	}

	// 2. Despesas (HTTP 201 + datas ISO convertidas para DD/MM/YYYY + CPF mascarado)
	despRes, err := transparencia.QueryIPMDespesas(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Inicio:  "2025-01-01",
		Fim:     "2025-01-31",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QueryIPMDespesas erro: %v", err)
	}
	if seenDespInicio != "01/01/2025" || seenDespFim != "31/01/2025" {
		t.Errorf("datas IPM inesperadas: inicio=%q fim=%q", seenDespInicio, seenDespFim)
	}
	if !despRes.Disponivel || len(despRes.Despesas) != 2 {
		t.Fatalf("DespesasResult inesperado: %+v", despRes)
	}
	d0 := despRes.Despesas[0]
	if d0.DocumentoFornecedor != "***.456.789-**" || d0.Fornecedor != "JOSE PEREIRA MEI" ||
		d0.Orgao != "Secretaria Municipal de Educação" || d0.Unidade != "Ensino Fundamental" ||
		d0.Elemento != "Recursos Próprios" || d0.ValorEmpenhado != 1500.50 || d0.ValorLiquidado != 1500.50 || d0.ValorPago != 1200.00 {
		t.Errorf("despesa[0] IPM inesperada: %+v", d0)
	}
	d1 := despRes.Despesas[1]
	if d1.DocumentoFornecedor != "12.345.678/0001-99" {
		t.Errorf("CNPJ não deveria ser mascarado: %q", d1.DocumentoFornecedor)
	}

	// 3. Receitas (Ano -> 01/01/YYYY..31/12/YYYY + HTTP 201)
	recRes, err := transparencia.QueryIPMReceitas(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Ano:     2025,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QueryIPMReceitas erro: %v", err)
	}
	if seenRecInicio != "01/01/2025" || seenRecFim != "31/12/2025" {
		t.Errorf("datas receitas IPM inesperadas: inicio=%q fim=%q", seenRecInicio, seenRecFim)
	}
	if !recRes.Disponivel || len(recRes.Receitas) != 1 {
		t.Fatalf("ReceitasResult inesperado: %+v", recRes)
	}
	if recRes.Receitas[0].Codigo != "1.1.1.2.50.00" || recRes.Receitas[0].ValorPrevisto != 1000000.00 || recRes.Receitas[0].ValorArrecadado != 450123.45 {
		t.Errorf("receita[0] IPM inesperada: %+v", recRes.Receitas[0])
	}

	// 4. Licitações WCO
	licRes, err := transparencia.QueryIPMLicitacoes(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QueryIPMLicitacoes erro: %v", err)
	}
	if !licRes.Disponivel || len(licRes.Licitacoes) != 1 {
		t.Fatalf("LicitacoesResult inesperado: %+v", licRes)
	}
	if licRes.Licitacoes[0].Numero != "PE 12/2025" || licRes.Licitacoes[0].ValorHomologado != 139500.00 {
		t.Errorf("licitacao[0] IPM inesperada: %+v", licRes.Licitacoes[0])
	}

	// 5. Contratos WCO
	conRes, err := transparencia.QueryIPMContratos(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QueryIPMContratos erro: %v", err)
	}
	if !conRes.Disponivel || len(conRes.Contratos) != 1 {
		t.Fatalf("ContratosResult inesperado: %+v", conRes)
	}
	c0 := conRes.Contratos[0]
	if c0.Numero != "CT-088/2025" || c0.DocumentoFornecedor != "***.222.333-**" || c0.Valor != 48000.00 {
		t.Errorf("contrato[0] IPM inesperado: %+v", c0)
	}
}

func TestIPMWCOUnavailableStatuses(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "4202404", Nome: "Blumenau", UF: "SC"}

	for _, statusCode := range []int{http.StatusNotFound, http.StatusServiceUnavailable} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "indisponivel", statusCode)
		}))

		licRes, err := transparencia.QueryIPMLicitacoes(ctx, client, m, transparencia.PortalQueryParams{BaseURL: srv.URL})
		if err == nil {
			t.Errorf("esperava erro para HTTP %d em QueryIPMLicitacoes", statusCode)
		}
		if licRes == nil || licRes.Disponivel {
			t.Errorf("esperava LicitacoesResult com Disponivel=false para HTTP %d, obtido %+v", statusCode, licRes)
		}

		conRes, err := transparencia.QueryIPMContratos(ctx, client, m, transparencia.PortalQueryParams{BaseURL: srv.URL})
		if err == nil {
			t.Errorf("esperava erro para HTTP %d em QueryIPMContratos", statusCode)
		}
		if conRes == nil || conRes.Disponivel {
			t.Errorf("esperava ContratosResult com Disponivel=false para HTTP %d, obtido %+v", statusCode, conRes)
		}

		srv.Close()
	}
}
