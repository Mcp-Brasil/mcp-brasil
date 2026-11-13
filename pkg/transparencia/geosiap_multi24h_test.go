package transparencia_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gibavargas/mcp-brasil/pkg/transparencia"
)

func TestNormalizeGeoSiapAPIBase(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{
			in:   "https://transparencia.exemplo.rs.gov.br:8443/portal-transparencia/#/inicio",
			want: "https://transparencia.exemplo.rs.gov.br:8443/portal-transparencia/api",
		},
		{
			in:   "https://transparencia.exemplo.rs.gov.br/portal-transparencia/despesas",
			want: "https://transparencia.exemplo.rs.gov.br/portal-transparencia/api",
		},
		{
			in:   "https://transparencia.exemplo.rs.gov.br/portal-transparencia/api",
			want: "https://transparencia.exemplo.rs.gov.br/portal-transparencia/api",
		},
		{
			in:   "https://transparencia.exemplo.rs.gov.br/municipio/portal-transparencia/home",
			want: "https://transparencia.exemplo.rs.gov.br/municipio/portal-transparencia/api",
		},
		{
			in:   "http://127.0.0.1:8080",
			want: "http://127.0.0.1:8080",
		},
	}

	for _, tc := range cases {
		if got := transparencia.NormalizeGeoSiapAPIBase(tc.in); got != tc.want {
			t.Errorf("NormalizeGeoSiapAPIBase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestProbeGeoSiap(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	t.Run("picks_principal_entity", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/portal-transparencia/api/sis_entidade" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[
					{"id_entidade": 2, "ds_entidade": "Câmara Municipal", "ativo": "1", "ds_schema": "camara", "principal": "0"},
					{"id_entidade": 4, "ds_entidade": "Prefeitura Municipal", "ativo": "1", "ds_schema": "pm", "principal": "1"}
				]`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		apiBase, eid, entName, ok := transparencia.ProbeGeoSiap(ctx, client, srv.URL+"/portal-transparencia/#/inicio")
		if !ok || eid != 4 || entName != "Prefeitura Municipal" || apiBase != srv.URL+"/portal-transparencia/api" {
			t.Fatalf("ProbeGeoSiap = (%q, %d, %q, %v), esperava eid=4", apiBase, eid, entName, ok)
		}
	})

	t.Run("env_cliente_json_fallback", func(t *testing.T) {
		var srvURL string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/portal-transparencia/assets/env.cliente.json":
				_, _ = fmt.Fprintf(w, `{"apiUrl":"%s/custom-geosiap-api"}`, srvURL)
			case "/custom-geosiap-api/sis_entidade":
				_, _ = w.Write([]byte(`[
					{"id_entidade": "9", "ds_entidade": "Fundo Municipal", "ativo": "1", "ds_schema": "fms", "principal": "0"}
				]`))
			default:
				http.NotFound(w, r)
			}
		}))
		srvURL = srv.URL
		defer srv.Close()

		apiBase, eid, entName, ok := transparencia.ProbeGeoSiap(ctx, client, srv.URL+"/portal-transparencia/home")
		if !ok || eid != 9 || entName != "Fundo Municipal" || apiBase != srv.URL+"/custom-geosiap-api" {
			t.Fatalf("ProbeGeoSiap env.cliente.json = (%q, %d, %q, %v), esperava eid=9", apiBase, eid, entName, ok)
		}
	})

	t.Run("negative", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if apiBase, eid, _, ok := transparencia.ProbeGeoSiap(ctx, client, srv.URL+"/portal-transparencia"); ok || apiBase != "" || eid != 0 {
			t.Fatalf("esperava false para 404, obteve (%q, %d, %v)", apiBase, eid, ok)
		}
	})
}

func TestQueryGeoSiapModules(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "4316808", Nome: "Santa Cruz do Sul", UF: "RS"}

	var seenRecQuery, seenDespQuery, seenContQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/portal-transparencia/api/sis_entidade":
			_, _ = w.Write([]byte(`[{"id_entidade": 3, "ds_entidade": "Prefeitura", "ativo": "1", "principal": "1"}]`))
		case "/portal-transparencia/api/execucao/receitas/proprias/receitas_proprias":
			seenRecQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`{
				"receitas_proprias": [
					{
						"nivel": "1",
						"cd_plano_conta_rel": "1.1.1.3.03",
						"ds_plano_conta": "ISSQN",
						"vl_previsto": "1250000.00",
						"vl_atualizado": "1300000.00",
						"vl_total": "1180500.50"
					}
				]
			}`))
		case "/portal-transparencia/api/execucao/despesas/programas_acoes":
			seenDespQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`[
				{
					"chave": "10.301.0010.2001",
					"descricao": "Manutenção da Atenção Básica",
					"nivel": "Ação",
					"vl_previsto": "500000.00",
					"vl_empenhado": "420000.50",
					"vl_liquidado": "390.100,25",
					"vl_pago": 380000.00
				}
			]`))
		case "/portal-transparencia/api/licitacoes/contratos/contratos":
			seenContQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`{
				"contratos": [
					{
						"contrato": "CT-10/2025",
						"objeto": "Serviços de engenharia",
						"dt_assinatura": "2025-02-01",
						"dt_inicio_contrato": "2025-02-01",
						"dt_termino_contrato": "2025-12-31",
						"situacao": "Ativo",
						"vl_inicial_f": "150.000,00",
						"vl_total_f": "175.500,75",
						"nome_razao": "CARLOS ALBERTO",
						"cnpj": "",
						"cpf": "11122233344"
					},
					{
						"contrato": "CT-11/2025",
						"objeto": "Fornecimento de combustível",
						"situacao": "Ativo",
						"vl_inicial_f": "200.000,00",
						"vl_total_f": "",
						"nome_razao": "AUTO POSTO SUL LTDA",
						"cnpj": "22.333.444/0001-55",
						"cpf": ""
					}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Usa URL SPA para validar normalização e descoberta automática de id_entidade=3
	p := transparencia.PortalQueryParams{
		BaseURL: srv.URL + "/portal-transparencia/#/despesas",
		Ano:     2025,
		Limit:   10,
	}

	rec, err := transparencia.QueryGeoSiapReceitas(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryGeoSiapReceitas erro: %v", err)
	}
	if rec.Plataforma != "geosiap_api" || !rec.Disponivel || len(rec.Receitas) != 1 {
		t.Fatalf("ReceitasResult GeoSiap inesperado: %+v", rec)
	}
	if rec.Receitas[0].ValorPrevisto != 1250000.00 || rec.Receitas[0].ValorArrecadado != 1180500.50 {
		t.Errorf("Receitas[0] valores inesperados: %+v", rec.Receitas[0])
	}
	if seenRecQuery != "id_entidade=3&ano=2025&layout=1" {
		t.Errorf("query receitas_proprias inesperada: %q", seenRecQuery)
	}

	desp, err := transparencia.QueryGeoSiapDespesas(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryGeoSiapDespesas erro: %v", err)
	}
	if desp.Plataforma != "geosiap_api" || len(desp.Despesas) != 1 {
		t.Fatalf("DespesasResult GeoSiap inesperado: %+v", desp)
	}
	if desp.Despesas[0].ValorEmpenhado != 420000.50 || desp.Despesas[0].ValorLiquidado != 390100.25 || desp.Despesas[0].ValorPago != 380000.00 {
		t.Errorf("Despesas[0] valores (string numbers) inesperados: %+v", desp.Despesas[0])
	}
	if seenDespQuery != "id_entidade=3&cp_ano=2025" {
		t.Errorf("query programas_acoes inesperada: %q", seenDespQuery)
	}

	ct, err := transparencia.QueryGeoSiapContratos(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryGeoSiapContratos erro: %v", err)
	}
	if ct.Plataforma != "geosiap_api" || len(ct.Contratos) != 2 {
		t.Fatalf("ContratosResult GeoSiap inesperado: %+v", ct)
	}
	if ct.Contratos[0].DocumentoFornecedor != "***.222.333-**" || ct.Contratos[0].Valor != 175500.75 {
		t.Errorf("Contratos[0] inesperado: %+v", ct.Contratos[0])
	}
	if ct.Contratos[1].DocumentoFornecedor != "22.333.444/0001-55" || ct.Contratos[1].Valor != 200000.00 {
		t.Errorf("Contratos[1] inesperado: %+v", ct.Contratos[1])
	}
	if seenContQuery != "id_entidade=3&ano=2025" {
		t.Errorf("query contratos inesperada: %q", seenContQuery)
	}
}

func TestNormalizeMulti24hWSBase(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{
			in:   "https://nuvem.multi24h.com.br/salvadordosul/sistemas/multi24/#/transparencia",
			want: "https://nuvem.multi24h.com.br/salvadordosul/sistemas/multi24/webservices",
		},
		{
			in:   "https://nuvem.multi24h.com.br/salvadordosul/sistemas/multi24/webservices",
			want: "https://nuvem.multi24h.com.br/salvadordosul/sistemas/multi24/webservices",
		},
		{
			in:   "https://saovendelino.multi24h.com.br/multi24/index.html",
			want: "https://saovendelino.multi24h.com.br/multi24/webservices",
		},
		{
			in:   "https://saovendelino.multi24h.com.br/multi24",
			want: "https://saovendelino.multi24h.com.br/multi24/webservices",
		},
		{
			in:   "http://127.0.0.1:8080",
			want: "http://127.0.0.1:8080",
		},
	}

	for _, tc := range cases {
		if got := transparencia.NormalizeMulti24hWSBase(tc.in); got != tc.want {
			t.Errorf("NormalizeMulti24hWSBase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestProbeMulti24h(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	t.Run("swagger_json", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/salvadordosul/sistemas/multi24/assets/files/contabil_planejamento/swagger-dados-abertos.json" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"openapi":"3.0.0","paths":{"/transparencia/dados-abertos/receita/index":{}}}`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		wsBase, ok := transparencia.ProbeMulti24h(ctx, client, srv.URL+"/salvadordosul/sistemas/multi24/#/inicio")
		want := srv.URL + "/salvadordosul/sistemas/multi24/webservices"
		if !ok || wsBase != want {
			t.Fatalf("ProbeMulti24h swagger = (%q, %v), want (%q, true)", wsBase, ok, want)
		}
	})

	t.Run("xml_post_fallback", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && r.URL.Path == "/multi24/webservices/transparencia/dados-abertos/receita/index" {
				w.Header().Set("Content-Type", "application/xml")
				_, _ = w.Write([]byte(`<receitas><receita><receita>1.1</receita><descricao>IPTU</descricao></receita></receitas>`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		wsBase, ok := transparencia.ProbeMulti24h(ctx, client, srv.URL+"/multi24/index.html")
		want := srv.URL + "/multi24/webservices"
		if !ok || wsBase != want {
			t.Fatalf("ProbeMulti24h XML fallback = (%q, %v), want (%q, true)", wsBase, ok, want)
		}
	})

	t.Run("negative", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if wsBase, ok := transparencia.ProbeMulti24h(ctx, client, srv.URL+"/multi24/"); ok || wsBase != "" {
			t.Fatalf("esperava false para 404, obteve (%q, %v)", wsBase, ok)
		}
	})
}

func TestQueryMulti24hReceitas(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "4316501", Nome: "Salvador do Sul", UF: "RS"}

	var (
		seenContentType string
		seenBody        map[string]any
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/salvadordosul/sistemas/multi24/webservices/transparencia/dados-abertos/receita/index" {
			http.NotFound(w, r)
			return
		}
		seenContentType = r.Header.Get("Content-Type")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &seenBody)

		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<receitas>
  <receita>
    <receita>1.1.1.2.50.0.1</receita>
    <categoria>Impostos</categoria>
    <descricao>IPTU - Imposto Predial</descricao>
    <entidade>Prefeitura Municipal</entidade>
    <orgao>Secretaria da Fazenda</orgao>
    <previsao_inicial>1000000.00</previsao_inicial>
    <previsao_adicional>50000.00</previsao_adicional>
    <previsao_anulada>10000.00</previsao_anulada>
    <arrecadacao>875250.40</arrecadacao>
  </receita>
  <receita>
    <receita>1.7.1.1.51.1.1</receita>
    <categoria>Transferências Correntes</categoria>
    <descricao>Cota-Parte do FPM</descricao>
    <entidade>Prefeitura Municipal</entidade>
    <orgao>Secretaria da Fazenda</orgao>
    <previsao_inicial>5000000.00</previsao_inicial>
    <previsao_adicional>0.00</previsao_adicional>
    <previsao_anulada>0.00</previsao_anulada>
    <arrecadacao>4900000.00</arrecadacao>
  </receita>
  <receita>
    <receita>1.3.2.1.01.0.1</receita>
    <categoria>Receita Patrimonial</categoria>
    <descricao>Rendimentos de Aplicação Financeira</descricao>
    <entidade>Prefeitura Municipal</entidade>
    <orgao>Tesouraria</orgao>
    <previsao_inicial>200000.00</previsao_inicial>
    <previsao_adicional>0.00</previsao_adicional>
    <previsao_anulada>0.00</previsao_anulada>
    <arrecadacao>215000.00</arrecadacao>
  </receita>
</receitas>`))
	}))
	defer srv.Close()

	res, err := transparencia.QueryMulti24hReceitas(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL + "/salvadordosul/sistemas/multi24/#/receitas",
		Ano:     2025,
		Limit:   2,
	})
	if err != nil {
		t.Fatalf("QueryMulti24hReceitas erro: %v", err)
	}
	if seenContentType != "application/json" {
		t.Errorf("Content-Type = %q, want 'application/json'", seenContentType)
	}
	if seenBody["exercicio"] != float64(2025) {
		t.Errorf("payload exercicio = %v, want 2025", seenBody["exercicio"])
	}
	for _, k := range []string{"codigo", "complemento_stn", "recurso_stn"} {
		v, exists := seenBody[k]
		if !exists || v != nil {
			t.Errorf("payload[%q] = (%v, exists=%v), want (nil, true)", k, v, exists)
		}
	}

	if res.Plataforma != "multi24h_xml" || !res.Disponivel || res.TotalRegistros != 3 || len(res.Receitas) != 2 {
		t.Fatalf("ReceitasResult Multi24h inesperado (Limit=2): %+v", res)
	}
	if res.Receitas[0].Codigo != "1.1.1.2.50.0.1" || res.Receitas[0].ValorPrevisto != 1040000.00 || res.Receitas[0].ValorArrecadado != 875250.40 {
		t.Errorf("Receitas[0] inesperado: %+v", res.Receitas[0])
	}
	if res.Receitas[1].Codigo != "1.7.1.1.51.1.1" || res.Receitas[1].ValorArrecadado != 4900000.00 {
		t.Errorf("Receitas[1] inesperado: %+v", res.Receitas[1])
	}

	// Testa NomeFiltro
	filtered, err := transparencia.QueryMulti24hReceitas(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL:    srv.URL + "/salvadordosul/sistemas/multi24/webservices",
		Ano:        2025,
		NomeFiltro: "FPM",
		Limit:      10,
	})
	if err != nil || len(filtered.Receitas) != 1 || filtered.Receitas[0].Codigo != "1.7.1.1.51.1.1" {
		t.Fatalf("filtro Multi24h inesperado: %+v err=%v", filtered, err)
	}
}
