package transparencia_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Mcp-Brasil/mcp-brasil/pkg/transparencia"
)

func TestProbeFatorSistemas(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	t.Run("url_com_id_e_html_fallback", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/portal.html":
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte(`<html><body><a href="/dados/diarias.php?id=pm_exemplo">Diárias</a></body></html>`))
			case "/dados/diarias.php":
				if r.URL.Query().Get("id") == "pm_exemplo" {
					_, _ = w.Write([]byte(`<html><body>Diárias</body></html>`))
					return
				}
				http.NotFound(w, r)
			case "/dados/carregaDiariasSub.php":
				if r.URL.Query().Get("id") == "pm_sub" {
					_, _ = w.Write([]byte(`[]`))
					return
				}
				http.NotFound(w, r)
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		// Extrai id diretamente da URL
		id1, ok1 := transparencia.ProbeFatorSistemas(ctx, client, srv.URL+"/dados/despesa.php?id=pm_exemplo")
		if !ok1 || id1 != "pm_exemplo" {
			t.Fatalf("ProbeFatorSistemas URL com id = (%q, %v), esperava ('pm_exemplo', true)", id1, ok1)
		}

		// Extrai id via HTML quando não está na URL
		id2, ok2 := transparencia.ProbeFatorSistemas(ctx, client, srv.URL+"/portal.html")
		if !ok2 || id2 != "pm_exemplo" {
			t.Fatalf("ProbeFatorSistemas via HTML = (%q, %v), esperava ('pm_exemplo', true)", id2, ok2)
		}

		// Fallback para carregaDiariasSub.php quando diarias.php retorna 404
		id3, ok3 := transparencia.ProbeFatorSistemas(ctx, client, srv.URL+"/dados/index.php?id=pm_sub")
		if !ok3 || id3 != "pm_sub" {
			t.Fatalf("ProbeFatorSistemas fallback carregaDiariasSub = (%q, %v), esperava ('pm_sub', true)", id3, ok3)
		}
	})

	t.Run("probe_negativo", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if id, ok := transparencia.ProbeFatorSistemas(ctx, client, srv.URL); ok || id != "" {
			t.Fatalf("esperava ('', false), obteve (%q, %v)", id, ok)
		}
	})
}

func TestQueryFatorSistemasStatefulCookieJar(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "2910800", Nome: "Feira de Santana", UF: "BA"}

	var mu sync.Mutex
	sessions := map[string]string{} // sessID -> module loaded

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id != "pm_feira" {
			http.NotFound(w, r)
			return
		}

		getSess := func() string {
			c, err := r.Cookie("PHPSESSID")
			if err != nil {
				return ""
			}
			return c.Value
		}

		switch r.URL.Path {
		case "/dados/despesa.php":
			sessID := "sess_despesa"
			http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: sessID, Path: "/"})
			mu.Lock()
			sessions[sessID] = "despesa_init"
			mu.Unlock()
			_, _ = w.Write([]byte(`<html>Despesa</html>`))

		case "/dados/receita.php":
			sessID := "sess_receita"
			http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: sessID, Path: "/"})
			mu.Lock()
			sessions[sessID] = "receita_init"
			mu.Unlock()
			_, _ = w.Write([]byte(`<html>Receita</html>`))

		case "/dados/contrato.php":
			sessID := "sess_contrato"
			http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: sessID, Path: "/"})
			mu.Lock()
			sessions[sessID] = "contrato_init"
			mu.Unlock()
			_, _ = w.Write([]byte(`<html>Contrato</html>`))

		case "/dados/licitacao.php":
			sessID := "sess_licitacao"
			http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: sessID, Path: "/"})
			mu.Lock()
			sessions[sessID] = "licitacao_init"
			mu.Unlock()
			_, _ = w.Write([]byte(`<html>Licitacao</html>`))

		case "/dados/folhapag.php":
			// Simula município sem folha de pagamento habilitada
			http.NotFound(w, r)

		case "/dados/carregaDespesa.php":
			sessID := getSess()
			mu.Lock()
			st := sessions[sessID]
			if st == "despesa_init" {
				sessions[sessID] = "despesa_loaded"
			}
			mu.Unlock()
			if st != "despesa_init" {
				http.Error(w, "sem sessao", http.StatusForbidden)
				return
			}
			_, _ = w.Write([]byte(`ok`))

		case "/dados/carregaReceita.php":
			sessID := getSess()
			mu.Lock()
			st := sessions[sessID]
			if st == "receita_init" {
				sessions[sessID] = "receita_loaded"
			}
			mu.Unlock()
			if st != "receita_init" {
				http.Error(w, "sem sessao", http.StatusForbidden)
				return
			}
			_, _ = w.Write([]byte(`ok`))

		case "/dados/carregaContrato.php":
			sessID := getSess()
			mu.Lock()
			st := sessions[sessID]
			if st == "contrato_init" {
				sessions[sessID] = "contrato_loaded"
			}
			mu.Unlock()
			if st != "contrato_init" {
				http.Error(w, "sem sessao", http.StatusForbidden)
				return
			}
			_, _ = w.Write([]byte(`ok`))

		case "/dados/carregaLicitacao.php":
			sessID := getSess()
			mu.Lock()
			st := sessions[sessID]
			if st == "licitacao_init" {
				sessions[sessID] = "licitacao_loaded"
			}
			mu.Unlock()
			if st != "licitacao_init" {
				http.Error(w, "sem sessao", http.StatusForbidden)
				return
			}
			_, _ = w.Write([]byte(`ok`))

		case "/dados/imprimir.php":
			if r.URL.Query().Get("tipo") != "3" || r.URL.Query().Get("tipodoc") != "JSON" {
				http.Error(w, "parametros invalidos", http.StatusBadRequest)
				return
			}
			sessID := getSess()
			mu.Lock()
			st := sessions[sessID]
			mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			switch st {
			case "despesa_loaded":
				_, _ = w.Write([]byte(`[
					{
						"numero": "778/2025",
						"ano": 2025,
						"data": "02/05/2025",
						"fornecedor": "ANTONIO SOUZA",
						"cpf_cnpj": "12345678901",
						"orgao": "Obras",
						"historico": "Reparo de calçada",
						"valor_empenhado": "3.450,00",
						"valor_pago": "3.450,00"
					}
				]`))
			case "receita_loaded":
				_, _ = w.Write([]byte(`[
					{
						"codigo": "1.1.1.2",
						"descricao": "IPTU",
						"valor_previsto": 1200000.00,
						"valor_arrecadado": 1150000.00
					}
				]`))
			case "contrato_loaded":
				_, _ = w.Write([]byte(`[
					{
						"numero": "90/2025",
						"ano": 2025,
						"fornecedor": "JOSE SANTOS",
						"cpf_cnpj": "99988877766",
						"objeto": "Serviço gráfico",
						"valor": 18000.00
					}
				]`))
			case "licitacao_loaded":
				_, _ = w.Write([]byte(`[
					{
						"numero": "PE-19/2025",
						"ano": 2025,
						"modalidade": "Pregão Eletrônico",
						"situacao": "Homologada",
						"objeto": "Compra de combustível",
						"valor_estimado": 400000.00,
						"valor_homologado": 390000.00
					}
				]`))
			default:
				http.Error(w, "sessao invalida", http.StatusForbidden)
			}

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	params := transparencia.PortalQueryParams{
		BaseURL:  srv.URL,
		TenantID: "pm_feira",
		Ano:      2025,
		Mes:      5,
	}

	// Despesas
	desp, err := transparencia.QueryFatorDespesas(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryFatorDespesas erro: %v", err)
	}
	if !desp.Disponivel || desp.Plataforma != "fator_sistemas" || len(desp.Despesas) != 1 {
		t.Fatalf("DespesasResult Fator inesperado: %+v", desp)
	}
	if desp.Despesas[0].DocumentoFornecedor != "***.456.789-**" || desp.Despesas[0].ValorEmpenhado != 3450.00 {
		t.Errorf("item despesa Fator inesperado: %+v", desp.Despesas[0])
	}

	// Receitas
	rec, err := transparencia.QueryFatorReceitas(ctx, client, m, params)
	if err != nil || !rec.Disponivel || len(rec.Receitas) != 1 {
		t.Fatalf("QueryFatorReceitas falhou: err=%v res=%+v", err, rec)
	}

	// Contratos
	ct, err := transparencia.QueryFatorContratos(ctx, client, m, params)
	if err != nil || !ct.Disponivel || len(ct.Contratos) != 1 {
		t.Fatalf("QueryFatorContratos falhou: err=%v res=%+v", err, ct)
	}
	if ct.Contratos[0].DocumentoFornecedor != "***.888.777-**" {
		t.Errorf("CPF não mascarado em contrato Fator: %q", ct.Contratos[0].DocumentoFornecedor)
	}

	// Licitações
	lic, err := transparencia.QueryFatorLicitacoes(ctx, client, m, params)
	if err != nil || !lic.Disponivel || len(lic.Licitacoes) != 1 {
		t.Fatalf("QueryFatorLicitacoes falhou: err=%v res=%+v", err, lic)
	}

	// Servidores (módulo não habilitado -> retorna slice vazio graciosamente sem erro)
	serv, err := transparencia.QueryFatorServidores(ctx, client, m, params)
	if err != nil {
		t.Fatalf("esperava retorno gracioso para folhapag desabilitado, obteve erro: %v", err)
	}
	if serv.Disponivel || serv.Servidores == nil || len(serv.Servidores) != 0 {
		t.Errorf("esperava Disponivel=false e slice vazio não-nil, obteve: %+v", serv)
	}
}

func TestQueryFatorServidoresEnabled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "2910800", Nome: "Feira de Santana", UF: "BA"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dados/folhapag.php":
			http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: "folha_ok", Path: "/"})
			_, _ = w.Write([]byte(`<html>Folha</html>`))
		case "/dados/carregaFolhapag.php":
			c, err := r.Cookie("PHPSESSID")
			if err != nil || c.Value != "folha_ok" {
				http.Error(w, "sem cookie", http.StatusForbidden)
				return
			}
			_, _ = w.Write([]byte(`ok`))
		case "/dados/imprimir.php":
			c, err := r.Cookie("PHPSESSID")
			if err != nil || c.Value != "folha_ok" {
				http.Error(w, "sem cookie", http.StatusForbidden)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[
				{
					"nome": "BEATRIZ LOPES",
					"matricula": "1022",
					"cpf": "44455566677",
					"cargo": "Médica",
					"lotacao": "SMS",
					"salario_bruto": 12000.00,
					"descontos": 2500.00,
					"salario_liquido": 9500.00
				}
			]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	res, err := transparencia.QueryFatorServidores(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL + "/dados/folhapag.php?id=pm_teste",
		Ano:     2025,
		Mes:     3,
	})
	if err != nil {
		t.Fatalf("QueryFatorServidores erro: %v", err)
	}
	if !res.Disponivel || len(res.Servidores) != 1 || res.Servidores[0].CPFParcial != "***.555.666-**" {
		t.Fatalf("ServidoresResult Fator inesperado: %+v", res)
	}
}
