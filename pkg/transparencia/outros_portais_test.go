package transparencia_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gibavargas/mcp-brasil/pkg/transparencia"
)

func TestProbeGovBRCidade360(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	t.Run("html_select_e_swagger", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/dadosabertos/":
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte(`
					<html><body>
						<h1>Portal Cidade360 - Dados Abertos</h1>
						<select id="nomeBase" name="nomeBase">
							<option value="">Selecione...</option>
							<option value="PM_EXEMPLO" selected="selected">Prefeitura de Exemplo</option>
							<option value="CM_EXEMPLO">Câmara de Exemplo</option>
						</select>
					</body></html>
				`))
			case "/dadosabertos/swagger/v1/swagger.json":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"openapi":"3.0.1","paths":{"/FolhaPagamento":{}}}`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		ent, ok := transparencia.ProbeGovBRCidade360(ctx, client, srv.URL)
		if !ok || ent != "PM_EXEMPLO" {
			t.Fatalf("esperava (PM_EXEMPLO, true), obteve (%q, %v)", ent, ok)
		}

		// Também quando baseURL já termina em /dadosabertos
		ent2, ok2 := transparencia.ProbeGovBRCidade360(ctx, client, srv.URL+"/dadosabertos")
		if !ok2 || ent2 != "PM_EXEMPLO" {
			t.Fatalf("esperava (PM_EXEMPLO, true) com sufixo /dadosabertos, obteve (%q, %v)", ent2, ok2)
		}
	})

	t.Run("swagger_only", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/dadosabertos/swagger/v1/swagger.json" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"swagger":"2.0","paths":{"/Receitas/buscarDadosReceitas":{}}}`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		_, ok := transparencia.ProbeGovBRCidade360(ctx, client, srv.URL)
		if !ok {
			t.Fatalf("esperava ok=true via swagger.json")
		}
	})

	t.Run("non_govbr", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if ent, ok := transparencia.ProbeGovBRCidade360(ctx, client, srv.URL); ok || ent != "" {
			t.Fatalf("esperava ('', false), obteve (%q, %v)", ent, ok)
		}
	})
}

func TestQueryGovBRServidoresAndReceitas(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "3509502", Nome: "Campinas", UF: "SP"}

	t.Run("folha_pagamento_e_receitas", func(t *testing.T) {
		var seenFolhaQuery, seenReceitaPath string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch {
			case r.URL.Path == "/FolhaPagamento":
				seenFolhaQuery = r.URL.RawQuery
				_, _ = w.Write([]byte(`[
					{
						"nome": "ANA CLARA LIMA",
						"matricula": "8821",
						"cpf": "12345678901",
						"cargo": "Enfermeira",
						"lotacao": "Secretaria de Saúde",
						"salarioBase": 4500.00,
						"totalProventos": 5200.00,
						"totalDescontos": 900.00,
						"salarioLiquido": 4300.00
					}
				]`))
			case strings.HasPrefix(r.URL.Path, "/Receitas/buscarDadosReceitas/"):
				seenReceitaPath = r.URL.RequestURI()
				_, _ = w.Write([]byte(`[
					{
						"codigo": "1.7.1.8.01.2.1",
						"descricao": "Cota-Parte do FPM",
						"categoria": "Transferências Correntes",
						"valorPrevisto": 2500000.00,
						"valorArrecadado": 2450000.50
					}
				]`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		servRes, err := transparencia.QueryGovBRServidores(ctx, client, m, transparencia.PortalQueryParams{
			BaseURL: srv.URL,
			Ano:     2025,
			Limit:   10,
		}, "PM_CAMPINAS")
		if err != nil {
			t.Fatalf("QueryGovBRServidores erro: %v", err)
		}
		if servRes.Plataforma != "govbr_cidade360" || len(servRes.Servidores) != 1 {
			t.Fatalf("ServidoresResult inesperado: %+v", servRes)
		}
		if servRes.Servidores[0].CPFParcial != "***.456.789-**" {
			t.Errorf("CPF não mascarado: %q", servRes.Servidores[0].CPFParcial)
		}
		if !strings.Contains(seenFolhaQuery, "dataInicial=01%2F01%2F2025") || !strings.Contains(seenFolhaQuery, "nomeBase=PM_CAMPINAS") {
			t.Errorf("query FolhaPagamento inesperada: %q", seenFolhaQuery)
		}

		recRes, err := transparencia.QueryGovBRReceitas(ctx, client, m, transparencia.PortalQueryParams{
			BaseURL: srv.URL,
			Ano:     2025,
			Limit:   10,
		}, "PM CAMPINAS")
		if err != nil {
			t.Fatalf("QueryGovBRReceitas erro: %v", err)
		}
		if recRes.Plataforma != "govbr_cidade360" || len(recRes.Receitas) != 1 || recRes.Receitas[0].ValorArrecadado != 2450000.50 {
			t.Fatalf("ReceitasResult inesperado: %+v", recRes)
		}
		if !strings.Contains(seenReceitaPath, "/Receitas/buscarDadosReceitas/2025/1/PM%20CAMPINAS?unidadeGestora=CONSOLIDADO") {
			t.Errorf("path Receitas inesperado: %q", seenReceitaPath)
		}
	})

	t.Run("fallback_para_servidor", func(t *testing.T) {
		var hitServidor bool
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/FolhaPagamento":
				_, _ = w.Write([]byte(`[]`))
			case "/Servidor":
				hitServidor = true
				_, _ = w.Write([]byte(`[{"nome":"BRUNO ALVES","cpf":"99988877766","cargo":"Motorista","matricula":"331"}]`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		servRes, err := transparencia.QueryGovBRServidores(ctx, client, m, transparencia.PortalQueryParams{
			BaseURL: srv.URL,
			Ano:     2025,
		}, "PM_EXEMPLO")
		if err != nil {
			t.Fatalf("QueryGovBRServidores fallback erro: %v", err)
		}
		if !hitServidor || len(servRes.Servidores) != 1 || servRes.Servidores[0].CPFParcial != "***.888.777-**" {
			t.Fatalf("fallback /Servidor falhou: hit=%v res=%+v", hitServidor, servRes)
		}
	})
}

func TestProbeAndQueryCoplan(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "5103403", Nome: "Cuiabá", UF: "MT"}

	var seenDespesasBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.RequestURI() {
		case "/servlet/api_consulta?unidade_gestora":
			_, _ = w.Write([]byte(`[
				{"unidade_gestora_id": 2, "unidade_gestora_descricao": "CÂMARA MUNICIPAL"},
				{"unidade_gestora_id": 5, "unidade_gestora_descricao": "PREFEITURA MUNICIPAL DE EXEMPLO"}
			]`))
		case "/servlet/api_consulta?despesas_v1":
			_ = json.Unmarshal(raw, &seenDespesasBody)
			_, _ = w.Write([]byte(`{
				"dados": [
					{
						"numero_empenho": "901/2025",
						"exercicio": 2025,
						"data_empenho": "2025-06-15",
						"fornecedor": "CONSTRUTORA BRASIL LTDA",
						"cpf_cnpj": "98.765.432/0001-10",
						"orgao": "Secretaria de Obras",
						"historico": "Pavimentação",
						"valor_empenhado": 85000.00,
						"valor_liquidado": 40000.00,
						"valor_pago": 40000.00
					}
				],
				"total": 12
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	ugID, ugDesc, ok := transparencia.ProbeCoplan(ctx, client, srv.URL)
	if !ok || ugID != 5 || !strings.Contains(ugDesc, "PREFEITURA") {
		t.Fatalf("ProbeCoplan inesperado: ugID=%d ugDesc=%q ok=%v", ugID, ugDesc, ok)
	}

	res, err := transparencia.QueryCoplanDespesas(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Ano:     2025,
		Mes:     6,
		Limit:   15,
	})
	if err != nil {
		t.Fatalf("QueryCoplanDespesas erro: %v", err)
	}
	if res.Plataforma != "coplan_gp" || res.TotalRegistros != 12 || len(res.Despesas) != 1 {
		t.Fatalf("DespesasResult Coplan inesperado: %+v", res)
	}
	if seenDespesasBody["unidade_gestora_id"] != float64(5) ||
		seenDespesasBody["exercicio"] != float64(2025) ||
		seenDespesasBody["mes"] != float64(6) ||
		seenDespesasBody["pagina"] != float64(1) ||
		seenDespesasBody["limite"] != float64(15) {
		t.Errorf("payload Coplan inesperado: %+v", seenDespesasBody)
	}

	// Probe negativo em servidor 404
	srvErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srvErr.Close()
	if id, desc, ok := transparencia.ProbeCoplan(ctx, client, srvErr.URL); ok || id != 0 || desc != "" {
		t.Errorf("esperava (0, '', false), obteve (%d, %q, %v)", id, desc, ok)
	}
}

func TestProbeAndQueryThemaAPIGW(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "4316907", Nome: "Santa Maria", UF: "RS"}

	t.Run("sucesso_com_e_sem_prefixo", func(t *testing.T) {
		var seenPostBody map[string]any
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/infra/apigw/transparencia/service/basico/administracao/listaAdministracoes/":
				_, _ = w.Write([]byte(`[
					{"codigo": 2, "descricao": "INSTITUTO DE PREVIDÊNCIA"},
					{"codigo": 1, "descricao": "PREFEITURA MUNICIPAL DE SANTA MARIA"}
				]`))
			case "/infra/apigw/transparencia/service/contabilidade/dotacao/consultaDetalhamentoDespesas/":
				if r.Method != http.MethodPost {
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
					return
				}
				raw, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(raw, &seenPostBody)
				_, _ = w.Write([]byte(`{
					"dados": [
						{
							"numeroEmpenho": "1234/2025",
							"exercicio": 2025,
							"dataEmpenho": "2025-03-01",
							"fornecedor": "FORNECEDOR SUL LTDA",
							"cnpjCpf": "11.222.333/0001-44",
							"orgao": "Secretaria de Educação",
							"elemento": "Material de Consumo",
							"valorEmpenhado": 12000.00,
							"valorLiquidado": 12000.00,
							"valorPago": 9500.00
						}
					],
					"totalElements": 30
				}`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		// Sem /infra/apigw/transparencia na BaseURL
		code, name, ok := transparencia.ProbeThemaAPIGW(ctx, client, srv.URL)
		if !ok || code != 1 || !strings.Contains(name, "PREFEITURA") {
			t.Fatalf("ProbeThemaAPIGW sem prefixo falhou: code=%d name=%q ok=%v", code, name, ok)
		}

		// Já incluindo /infra/apigw/transparencia na BaseURL
		code2, name2, ok2 := transparencia.ProbeThemaAPIGW(ctx, client, srv.URL+"/infra/apigw/transparencia")
		if !ok2 || code2 != 1 || name2 != name {
			t.Fatalf("ProbeThemaAPIGW com prefixo falhou: code=%d name=%q ok=%v", code2, name2, ok2)
		}

		res, err := transparencia.QueryThemaDespesas(ctx, client, m, transparencia.PortalQueryParams{
			BaseURL: srv.URL + "/infra/apigw/transparencia",
			Ano:     2025,
			Limit:   10,
		})
		if err != nil {
			t.Fatalf("QueryThemaDespesas erro: %v", err)
		}
		if res.Plataforma != "thema_apigw" || res.TotalRegistros != 30 || len(res.Despesas) != 1 {
			t.Fatalf("DespesasResult Thema inesperado: %+v", res)
		}
		if seenPostBody["administracao"] != "1" || seenPostBody["exercicio"] != float64(2025) {
			t.Errorf("payload Thema inesperado: %+v", seenPostBody)
		}
	})

	t.Run("safeline_waf_403", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`<html><title>SafeLine WAF</title><body>403 Forbidden</body></html>`))
		}))
		defer srv.Close()

		code, name, ok := transparencia.ProbeThemaAPIGW(ctx, client, srv.URL)
		if ok || code != 0 || name != "" {
			t.Fatalf("esperava (0, '', false) em HTTP 403 SafeLine WAF, obteve (%d, %q, %v)", code, name, ok)
		}

		if _, err := transparencia.QueryThemaDespesas(ctx, client, m, transparencia.PortalQueryParams{
			BaseURL:          srv.URL,
			CodAdministracao: 1,
			Ano:              2025,
		}); err == nil {
			t.Errorf("esperava erro limpo em HTTP 403 SafeLine WAF")
		}
	})
}
