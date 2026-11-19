package transparencia_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Mcp-Brasil/mcp-brasil/pkg/transparencia"
)

func TestExtractAspecEntityCode(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "root_url",
			input: "https://www.governotransparente.com.br/11629488",
			want:  "11629488",
		},
		{
			name:  "receitas_path",
			input: "governotransparente.com.br/transparencia/receitas/11509489",
			want:  "11509489",
		},
		{
			name:  "html_anchor",
			input: `<html><body><a href="https://www.governotransparente.com.br/transparencia/empenhos/11629488?ano=2025">Transparência</a></body></html>`,
			want:  "11629488",
		},
		{
			name:  "non_aspec_html",
			input: `<html><body><a href="https://prefeitura.exemplo.gov.br/2025">Site</a></body></html>`,
			want:  "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := transparencia.ExtractAspecEntityCode(tc.input)
			if got != tc.want {
				t.Errorf("ExtractAspecEntityCode(%q) = %q, esperava %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestProbeAndQueryAspec(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "2304400", Nome: "Fortaleza", UF: "CE"}

	var seenEmpenhoQuery, seenReceitaQuery, seenContratoQuery, seenLicitacaoQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/transparencia/api/v1/json/exercicios/11629488":
			_, _ = w.Write([]byte(`[
				{"id": 41, "nome": "2024"},
				{"id": 42, "nome": "2025"}
			]`))
		case "/transparencia/api/v1/json/empenhos/11629488":
			seenEmpenhoQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`[
				{
					"numero": "1020/2025",
					"exercicio": 2025,
					"data": "15/02/2025",
					"credor": "JOAO DA SILVA",
					"cpfCnpj": "12345678901",
					"orgao": "Secretaria de Saúde",
					"historico": "Serviço médico",
					"valorEmpenhado": "12.500,50",
					"valorLiquidado": 10000.00,
					"valorPago": 9500.00
				},
				{
					"numero": "1021/2025",
					"exercicio": 2025,
					"data": "20/02/2025",
					"credor": "EMPRESA ALFA LTDA",
					"cpfCnpj": "12.345.678/0001-90",
					"orgao": "Secretaria de Educação",
					"historico": "Material escolar",
					"valorEmpenhado": 5000.00
				}
			]`))
		case "/transparencia/api/v1/json/receitasorc/11629488":
			seenReceitaQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`{
				"registros": [
					{
						"codigo": "1.1.1.2.50.0.1",
						"descricao": "IPTU",
						"categoria": "Receitas Correntes",
						"valorPrevisto": "1000000,00",
						"valorArrecadado": "850000,25"
					}
				],
				"total": 40
			}`))
		case "/transparencia/api/v1/json/contratos/11629488":
			seenContratoQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`[
				{
					"numero": "CT-055/2025",
					"ano": 2025,
					"contratado": "MARIA OLIVEIRA",
					"cpfCnpj": "987.654.321-00",
					"objeto": "Consultoria técnica",
					"situacao": "Vigente",
					"dataAssinatura": "05/01/2025",
					"vigenciaInicio": "05/01/2025",
					"vigenciaFim": "31/12/2025",
					"valor": 48000.00
				}
			]`))
		case "/transparencia/api/v1/json/licitacoes/11629488":
			seenLicitacaoQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`[
				{
					"numero": "PE-010/2025",
					"ano": 2025,
					"modalidade": "Pregão Eletrônico",
					"situacao": "Homologada",
					"dataAbertura": "12/03/2025",
					"objeto": "Aquisição de medicamentos",
					"valorEstimado": 250000.00,
					"valorHomologado": 230000.00
				}
			]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Probe positivo
	code, ok := transparencia.ProbeAspec(ctx, client, srv.URL, "11629488")
	if !ok || code != "11629488" {
		t.Fatalf("ProbeAspec = (%q, %v), esperava (11629488, true)", code, ok)
	}

	// Probe negativo
	if c, okNeg := transparencia.ProbeAspec(ctx, client, srv.URL, "99999999"); okNeg || c != "" {
		t.Fatalf("ProbeAspec negativo = (%q, %v), esperava ('', false)", c, okNeg)
	}

	// Despesas (converte YYYY-MM-DD -> dd/MM/yyyy e resolve exer=42)
	desp, err := transparencia.QueryAspecDespesas(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL:  srv.URL + "/11629488",
		TenantID: "11629488",
		Ano:      2025,
		Inicio:   "2025-01-01",
		Fim:      "2025-03-31",
		Limit:    15,
	})
	if err != nil {
		t.Fatalf("QueryAspecDespesas erro: %v", err)
	}
	if desp.Plataforma != "aspec_gt" || len(desp.Despesas) != 2 {
		t.Fatalf("DespesasResult inesperado: %+v", desp)
	}
	if desp.Despesas[0].DocumentoFornecedor != "***.456.789-**" {
		t.Errorf("CPF não mascarado em despesa: %q", desp.Despesas[0].DocumentoFornecedor)
	}
	if desp.Despesas[0].ValorEmpenhado != 12500.50 {
		t.Errorf("ValorEmpenhado inesperado: %v", desp.Despesas[0].ValorEmpenhado)
	}
	if desp.Despesas[1].DocumentoFornecedor != "12.345.678/0001-90" {
		t.Errorf("CNPJ deveria ser preservado: %q", desp.Despesas[1].DocumentoFornecedor)
	}
	if !strings.Contains(seenEmpenhoQuery, "exer=42") ||
		!strings.Contains(seenEmpenhoQuery, "pagesize=15") ||
		!strings.Contains(seenEmpenhoQuery, "inicio=01%2F01%2F2025") ||
		!strings.Contains(seenEmpenhoQuery, "fim=31%2F03%2F2025") {
		t.Errorf("query empenhos inesperada: %q", seenEmpenhoQuery)
	}

	// Receitas
	rec, err := transparencia.QueryAspecReceitas(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL:  srv.URL,
		TenantID: "11629488",
		Ano:      2025,
		Limit:    20,
	})
	if err != nil {
		t.Fatalf("QueryAspecReceitas erro: %v", err)
	}
	if rec.TotalRegistros != 40 || len(rec.Receitas) != 1 || rec.Receitas[0].ValorArrecadado != 850000.25 {
		t.Fatalf("ReceitasResult inesperado: %+v", rec)
	}
	if !strings.Contains(seenReceitaQuery, "exer=42") || !strings.Contains(seenReceitaQuery, "inicio=01%2F01%2F2025") {
		t.Errorf("query receitas inesperado: %q", seenReceitaQuery)
	}

	// Contratos
	ct, err := transparencia.QueryAspecContratos(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL:  srv.URL,
		TenantID: "11629488",
		Ano:      2025,
	})
	if err != nil {
		t.Fatalf("QueryAspecContratos erro: %v", err)
	}
	if len(ct.Contratos) != 1 || ct.Contratos[0].DocumentoFornecedor != "***.654.321-**" {
		t.Fatalf("ContratosResult inesperado: %+v", ct)
	}
	if !strings.Contains(seenContratoQuery, "exer=42") {
		t.Errorf("query contratos inesperado: %q", seenContratoQuery)
	}

	// Licitações
	lic, err := transparencia.QueryAspecLicitacoes(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL:  srv.URL,
		TenantID: "11629488",
		Ano:      2025,
	})
	if err != nil {
		t.Fatalf("QueryAspecLicitacoes erro: %v", err)
	}
	if len(lic.Licitacoes) != 1 || lic.Licitacoes[0].ValorHomologado != 230000.00 {
		t.Fatalf("LicitacoesResult inesperado: %+v", lic)
	}
	if !strings.Contains(seenLicitacaoQuery, "inicio=01%2F01%2F2025") || !strings.Contains(seenLicitacaoQuery, "fim=31%2F12%2F2025") {
		t.Errorf("query licitacoes inesperado: %q", seenLicitacaoQuery)
	}

	// Ano inexistente retorna lista vazia sem erro
	despEmpty, err := transparencia.QueryAspecDespesas(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL:  srv.URL,
		TenantID: "11629488",
		Ano:      2018,
	})
	if err != nil {
		t.Fatalf("esperava resultado vazio sem erro para ano 2018: %v", err)
	}
	if len(despEmpty.Despesas) != 0 {
		t.Errorf("esperava 0 despesas para ano 2018, obteve %d", len(despEmpty.Despesas))
	}
}

func TestProbeAndQueryAssesi(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "2307650", Nome: "Maracanaú", UF: "CE"}

	t.Run("probe_receitas_e_dadosabertos_com_script_alert_guard", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/dadosabertosexportar.php" {
				d := r.URL.Query().Get("d")
				if d == "empenhos" {
					w.Header().Set("Content-Type", "text/html")
					_, _ = w.Write([]byte(`<script>alert('Nenhum registro encontrado!');history.back();</script>`))
					return
				}
				if d == "receitas" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`[{"codigo":"1.1","descricao":"IPTU","valor_previsto":1000,"valor_arrecadado":900}]`))
					return
				}
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if !transparencia.ProbeAssesi(ctx, client, srv.URL) {
			t.Fatalf("ProbeAssesi deveria retornar true quando receitas retorna JSON após <script>alert em empenhos")
		}

		srvHTML := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/dadosabertosexportar.php":
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte(`<script>alert('Sem dados');</script>`))
			case "/dadosabertos.php":
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte(`<html><body><a href="dadosabertosexportar.php?d=empenhos&a=2025&f=json">Exportar</a></body></html>`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srvHTML.Close()

		if !transparencia.ProbeAssesi(ctx, client, srvHTML.URL+"/dadosabertos.php") {
			t.Fatalf("ProbeAssesi deveria detectar via dadosabertos.php contendo dadosabertosexportar.php")
		}

		srvNeg := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<html><body>Portal genérico</body></html>`))
		}))
		defer srvNeg.Close()

		if transparencia.ProbeAssesi(ctx, client, srvNeg.URL) {
			t.Fatalf("ProbeAssesi não deveria dar positivo em HTML genérico")
		}
	})

	t.Run("queries_completas_e_guard_script_alert", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/dadosabertosexportar.php" {
				http.NotFound(w, r)
				return
			}
			d := r.URL.Query().Get("d")
			a := r.URL.Query().Get("a")
			if a == "2020" {
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte(`<script>alert('Nenhum registro encontrado para o exercício 2020');</script>`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			switch d {
			case "pessoal":
				_, _ = w.Write([]byte(`[
					{
						"nome": "CARLOS MENDES",
						"matricula": "4412",
						"cpf": "11122233344",
						"cargo": "Agente Administrativo",
						"lotacao": "Administração",
						"salario_base": "3200,00",
						"salario_bruto": "3800,00",
						"descontos": "600,00",
						"salario_liquido": "3200,00"
					}
				]`))
			case "empenhos":
				_, _ = w.Write([]byte(`[
					{
						"numero_empenho": "501/2025",
						"ano": 2025,
						"data": "10/04/2025",
						"fornecedor": "PEDRO ALMEIDA",
						"cpf_cnpj": "55566677788",
						"historico": "Manutenção predial",
						"valor_empenhado": 7500.00,
						"valor_pago": 7500.00
					}
				]`))
			case "receitas":
				_, _ = w.Write([]byte(`[
					{
						"codigo": "1.7.1.8.01",
						"descricao": "FPM",
						"categoria": "Transferências",
						"valor_previsto": 5000000,
						"valor_arrecadado": 4800000
					}
				]`))
			case "contratos":
				_, _ = w.Write([]byte(`[
					{
						"numero_contrato": "12/2025",
						"ano": 2025,
						"fornecedor": "LUCIA COSTA",
						"cpf_cnpj": "22233344455",
						"objeto": "Locação de impressoras",
						"valor_contrato": 24000.00
					}
				]`))
			case "licitacoes":
				_, _ = w.Write([]byte(`[
					{
						"numero_licitacao": "003/2025",
						"ano": 2025,
						"modalidade": "Pregão Eletrônico",
						"situacao": "Concluída",
						"objeto": "Aquisição de computadores",
						"valor_estimado": 90000.00,
						"valor_homologado": 82000.00
					}
				]`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		// Ano vazio com <script>alert(...) não deve dar erro de parse
		emptyDesp, err := transparencia.QueryAssesiDespesas(ctx, client, m, transparencia.PortalQueryParams{
			BaseURL: srv.URL,
			Ano:     2020,
		})
		if err != nil {
			t.Fatalf("esperava tratamento gracioso de <script>alert(...), obteve erro: %v", err)
		}
		if len(emptyDesp.Despesas) != 0 || !emptyDesp.Disponivel {
			t.Errorf("resultado vazio inesperado: %+v", emptyDesp)
		}

		// Servidores
		serv, err := transparencia.QueryAssesiServidores(ctx, client, m, transparencia.PortalQueryParams{BaseURL: srv.URL, Ano: 2025})
		if err != nil || len(serv.Servidores) != 1 {
			t.Fatalf("QueryAssesiServidores falhou: err=%v res=%+v", err, serv)
		}
		if serv.Servidores[0].CPFParcial != "***.222.333-**" {
			t.Errorf("CPF não mascarado em Assesi servidores: %q", serv.Servidores[0].CPFParcial)
		}

		// Despesas
		desp, err := transparencia.QueryAssesiDespesas(ctx, client, m, transparencia.PortalQueryParams{BaseURL: srv.URL, Ano: 2025})
		if err != nil || len(desp.Despesas) != 1 {
			t.Fatalf("QueryAssesiDespesas falhou: err=%v res=%+v", err, desp)
		}
		if desp.Despesas[0].DocumentoFornecedor != "***.666.777-**" {
			t.Errorf("CPF não mascarado em Assesi despesas: %q", desp.Despesas[0].DocumentoFornecedor)
		}

		// Receitas
		rec, err := transparencia.QueryAssesiReceitas(ctx, client, m, transparencia.PortalQueryParams{BaseURL: srv.URL, Ano: 2025})
		if err != nil || len(rec.Receitas) != 1 || rec.Receitas[0].ValorArrecadado != 4800000 {
			t.Fatalf("QueryAssesiReceitas falhou: err=%v res=%+v", err, rec)
		}

		// Contratos
		ct, err := transparencia.QueryAssesiContratos(ctx, client, m, transparencia.PortalQueryParams{BaseURL: srv.URL, Ano: 2025})
		if err != nil || len(ct.Contratos) != 1 {
			t.Fatalf("QueryAssesiContratos falhou: err=%v res=%+v", err, ct)
		}
		if ct.Contratos[0].DocumentoFornecedor != "***.333.444-**" {
			t.Errorf("CPF não mascarado em Assesi contratos: %q", ct.Contratos[0].DocumentoFornecedor)
		}

		// Licitações
		lic, err := transparencia.QueryAssesiLicitacoes(ctx, client, m, transparencia.PortalQueryParams{BaseURL: srv.URL, Ano: 2025})
		if err != nil || len(lic.Licitacoes) != 1 || lic.Licitacoes[0].ValorHomologado != 82000 {
			t.Fatalf("QueryAssesiLicitacoes falhou: err=%v res=%+v", err, lic)
		}
	})
}
