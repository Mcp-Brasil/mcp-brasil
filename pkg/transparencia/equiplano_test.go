package transparencia_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Mcp-Brasil/mcp-brasil/pkg/transparencia"
)

func TestDiscoverEquiplanoCloud(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	t.Run("discovers_prefeitura_compact_slug", func(t *testing.T) {
		var seenPaths []string
		var seenDominios []string

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seenPaths = append(seenPaths, r.URL.Path)
			seenDominios = append(seenDominios, r.Header.Get("x-dominio"))
			if r.URL.Path == "/transparencia/dominio/check/prefeitura-campomourao" &&
				r.Header.Get("x-dominio") == "prefeitura-campomourao" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"entidade": {
						"uuid": "ent-111",
						"nome": "Município de Campo Mourão"
					},
					"municipioClienteUuid": "mun-222",
					"clienteUuid": "cli-333"
				}`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		restore := transparencia.SetEquiplanoAPIBaseURL(srv.URL)
		defer transparencia.SetEquiplanoAPIBaseURL(restore)

		m := &transparencia.Municipio{
			CodigoIBGE: "4104303",
			Nome:       "Campo Mourão",
			UF:         "PR",
			Slug:       "campo-mourao",
		}

		res, ok := transparencia.DiscoverEquiplanoCloud(ctx, client, m)
		if !ok || res == nil {
			t.Fatalf("esperava DiscoverEquiplanoCloud=true, paths=%v dominios=%v", seenPaths, seenDominios)
		}
		if res.Plataforma != "equiplano_cloud" {
			t.Errorf("Plataforma esperada 'equiplano_cloud', obtida %q", res.Plataforma)
		}
		if res.URLResolvida != "https://portal-prefeitura-campomourao.equiplano.cloud" {
			t.Errorf("URLResolvida inesperada: %q", res.URLResolvida)
		}
		if res.TenantID != "ent-111|mun-222|cli-333" {
			t.Errorf("TenantID inesperado: %q", res.TenantID)
		}
		if res.EntidadeNome != "Município de Campo Mourão" {
			t.Errorf("EntidadeNome inesperado: %q", res.EntidadeNome)
		}
		if len(seenPaths) < 2 || seenPaths[0] != "/transparencia/dominio/check/prefeitura-campo-mourao" {
			t.Errorf("esperava testar prefeitura-campo-mourao antes de prefeitura-campomourao, obtido %v", seenPaths)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer srv.Close()

		restore := transparencia.SetEquiplanoAPIBaseURL(srv.URL)
		defer transparencia.SetEquiplanoAPIBaseURL(restore)

		m := &transparencia.Municipio{CodigoIBGE: "4100103", Nome: "Abatiá", UF: "PR", Slug: "abatia"}
		if res, ok := transparencia.DiscoverEquiplanoCloud(ctx, client, m); ok || res != nil {
			t.Fatalf("esperava DiscoverEquiplanoCloud=false para 404, obtido %+v", res)
		}
	})
}

func TestEquiplanoQueriesMock(t *testing.T) {
	receitasDocJSON := `{
		"collection2018": [
			{
				"codigoReceita": "1.1.1.3.03.00",
				"descricao": "IRRF - Imposto de Renda Retido na Fonte",
				"categoria": "Recursos Livres",
				"data": "2025-06-30",
				"valorPrevisto": 250000.00,
				"valorArrecadado": "312.450,75"
			},
			{
				"codigoReceita": "1.7.1.1.51.00",
				"descricao": "Cota-Parte do FPM",
				"categoria": "Transferências Constitucionais",
				"valorPrevisto": 5000000.00,
				"valorArrecadado": 4800000.00
			}
		]
	}`
	receitasDocB64 := base64.StdEncoding.EncodeToString([]byte(receitasDocJSON))

	contratosDocJSON := `{
		"contratoCollection": [
			{
				"numeroContrato": "045/2025",
				"exercicio": 2025,
				"nomeFornecedor": "ALFA SERVICOS LTDA",
				"cpfCnpj": "98.765.432/0001-10",
				"objeto": "Manutenção preventiva de frota municipal",
				"situacao": "Ativo",
				"dataAssinatura": "2025-02-01",
				"vigenciaInicio": "2025-02-01",
				"vigenciaFim": "2026-01-31",
				"valor": 180000.00
			},
			{
				"numeroContrato": "046/2025",
				"exercicio": 2025,
				"nomeFornecedor": "PEDRO ENGENHEIRO",
				"cpfCnpj": "123.456.789-01",
				"objeto": "Fiscalização técnica de obra",
				"situacao": "Ativo",
				"dataAssinatura": "2025-03-10",
				"valor": "12.500,00"
			}
		]
	}`
	contratosDocB64 := base64.StdEncoding.EncodeToString([]byte(contratosDocJSON))

	var seenHeaders [4][3]string
	var seenDespQuery, seenRecQuery, seenConQuery, seenObrasQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		captureHeaders := func(idx int) {
			seenHeaders[idx] = [3]string{
				r.Header.Get("x-entity-uuid"),
				r.Header.Get("x-county-client-uuid"),
				r.Header.Get("x-client-uuid"),
			}
		}

		switch r.URL.Path {
		case "/transparencia/ordem_cronologica_de_pagamentos":
			captureHeaders(0)
			seenDespQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`{
				"content": [
					{
						"numeroEmpenho": "1020/2025",
						"exercicio": 2025,
						"dataPagamento": "2025-03-15",
						"nomeFornecedor": "ANA PAULA MEI",
						"cpfCnpj": "12345678901",
						"orgao": "Secretaria de Administração",
						"unidade": "Departamento de Compras",
						"fonteRecurso": "Recursos Ordinários",
						"historico": "Aquisição de suprimentos",
						"valorEmpenhado": 2400.00,
						"valorLiquidado": 2400.00,
						"valorPago": 2400.00
					}
				],
				"totalElements": 1
			}`))

		case "/transparencia/arrecadacao_receitas_arrecadada/process":
			captureHeaders(1)
			seenRecQuery = r.URL.RawQuery
			resp, _ := json.Marshal(map[string]any{
				"dados": map[string]any{
					"documento": receitasDocB64,
				},
			})
			_, _ = w.Write(resp)

		case "/transparencia/cont_contratos_fornecedor/process":
			captureHeaders(2)
			seenConQuery = r.URL.RawQuery
			resp, _ := json.Marshal(map[string]any{
				"dados": map[string]any{
					"documento": contratosDocB64,
				},
			})
			_, _ = w.Write(resp)

		case "/transparencia/rel_obras_publicas/listarObrasPublicas":
			captureHeaders(3)
			seenObrasQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`{
				"dados": [
					{
						"codigo": "OB-09",
						"titulo": "Pavimentação Rua XV de Novembro",
						"objeto": "Recapeamento asfáltico CBUQ",
						"orgao": "Secretaria de Obras",
						"endereco": "Rua XV de Novembro, 100-900",
						"bairro": "Centro",
						"dataInicial": "2025-01-15",
						"dataFinal": "2025-09-30",
						"situacao": "Em andamento",
						"valor": "850.000,00"
					}
				]
			}`))

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	restore := transparencia.SetEquiplanoAPIBaseURL(srv.URL)
	defer transparencia.SetEquiplanoAPIBaseURL(restore)

	m := &transparencia.Municipio{CodigoIBGE: "4104303", Nome: "Campo Mourão", UF: "PR", Slug: "campo-mourao"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	params := transparencia.PortalQueryParams{
		BaseURL:  "https://portal-prefeitura-campomourao.equiplano.cloud",
		TenantID: "ent-111|mun-222|cli-333",
		Ano:      2025,
		Inicio:   "2025-01-01",
		Fim:      "2025-03-31",
		Limit:    15,
	}

	// 1. Despesas
	despRes, err := transparencia.QueryEquiplanoDespesas(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryEquiplanoDespesas erro: %v", err)
	}
	if !strings.Contains(seenDespQuery, "entidadeUuid=ent-111") ||
		!strings.Contains(seenDespQuery, "exercicio=2025") ||
		!strings.Contains(seenDespQuery, "dataInicial=2025-01-01") ||
		!strings.Contains(seenDespQuery, "dataFinal=2025-03-31") ||
		!strings.Contains(seenDespQuery, "nrPaginaInicial=1") ||
		!strings.Contains(seenDespQuery, "size=15") {
		t.Errorf("query despesas Equiplano inesperada: %q", seenDespQuery)
	}
	if !despRes.Disponivel || despRes.Plataforma != "equiplano_cloud" || len(despRes.Despesas) != 1 {
		t.Fatalf("DespesasResult inesperado: %+v", despRes)
	}
	if despRes.Despesas[0].DocumentoFornecedor != "***.456.789-**" || despRes.Despesas[0].ValorPago != 2400.00 {
		t.Errorf("despesa[0] Equiplano inesperada: %+v", despRes.Despesas[0])
	}

	// 2. Receitas (base64 + collection2018)
	recRes, err := transparencia.QueryEquiplanoReceitas(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryEquiplanoReceitas erro: %v", err)
	}
	if !strings.Contains(seenRecQuery, "entidadeUuid=ent-111") ||
		!strings.Contains(seenRecQuery, "entidadesSelecionadasUuid=ent-111") ||
		!strings.Contains(seenRecQuery, "mesInicial=1") ||
		!strings.Contains(seenRecQuery, "mesFinal=12") ||
		!strings.Contains(seenRecQuery, "type=JSON") ||
		!strings.Contains(seenRecQuery, "size=0") {
		t.Errorf("query receitas Equiplano inesperada: %q", seenRecQuery)
	}
	if !recRes.Disponivel || recRes.TotalRegistros != 2 || len(recRes.Receitas) != 2 {
		t.Fatalf("ReceitasResult inesperado: %+v", recRes)
	}
	if recRes.Receitas[0].Codigo != "1.1.1.3.03.00" || recRes.Receitas[0].ValorArrecadado != 312450.75 {
		t.Errorf("receita[0] Equiplano inesperada: %+v", recRes.Receitas[0])
	}

	// 3. Contratos (base64 + contratoCollection)
	conRes, err := transparencia.QueryEquiplanoContratos(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryEquiplanoContratos erro: %v", err)
	}
	if !strings.Contains(seenConQuery, "entidadeUuid=ent-111") ||
		!strings.Contains(seenConQuery, "exercicio=2025") ||
		!strings.Contains(seenConQuery, "type=JSON") ||
		!strings.Contains(seenConQuery, "size=0") {
		t.Errorf("query contratos Equiplano inesperada: %q", seenConQuery)
	}
	if !conRes.Disponivel || len(conRes.Contratos) != 2 {
		t.Fatalf("ContratosResult inesperado: %+v", conRes)
	}
	if conRes.Contratos[0].DocumentoFornecedor != "98.765.432/0001-10" || conRes.Contratos[0].Valor != 180000.00 {
		t.Errorf("contrato[0] Equiplano inesperado: %+v", conRes.Contratos[0])
	}
	if conRes.Contratos[1].DocumentoFornecedor != "***.456.789-**" || conRes.Contratos[1].Valor != 12500.00 {
		t.Errorf("contrato[1] Equiplano inesperado: %+v", conRes.Contratos[1])
	}

	// 4. Obras
	obrasRes, err := transparencia.QueryEquiplanoObras(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryEquiplanoObras erro: %v", err)
	}
	if !strings.Contains(seenObrasQuery, "entidadeUuid=ent-111") || !strings.Contains(seenObrasQuery, "exercicio=2025") {
		t.Errorf("query obras Equiplano inesperada: %q", seenObrasQuery)
	}
	if !obrasRes.Disponivel || len(obrasRes.Obras) != 1 {
		t.Fatalf("ObrasResult inesperado: %+v", obrasRes)
	}
	o0 := obrasRes.Obras[0]
	if o0.Codigo != "OB-09" || o0.Titulo != "Pavimentação Rua XV de Novembro" || o0.Bairro != "Centro" || o0.Valor != 850000.00 {
		t.Errorf("obra[0] Equiplano inesperada: %+v", o0)
	}

	// Verifica tenant headers em todas as 4 chamadas
	for i, h := range seenHeaders {
		if h[0] != "ent-111" || h[1] != "mun-222" || h[2] != "cli-333" {
			t.Errorf("endpoint[%d] recebeu headers inesperados: %v", i, h)
		}
	}
}

func TestEquiplanoReceitasTopLevelCollection(t *testing.T) {
	topLevelDoc := `[{"codigo":"1.2.1","descricao":"Contribuições","valorPrevisto":1000,"valorArrecadado":900}]`
	docB64 := base64.StdEncoding.EncodeToString([]byte(topLevelDoc))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp, _ := json.Marshal(map[string]any{
			"dados": map[string]any{
				"documento": docB64,
			},
		})
		_, _ = w.Write(resp)
	}))
	defer srv.Close()

	m := &transparencia.Municipio{CodigoIBGE: "4104303", Nome: "Campo Mourão", UF: "PR"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Testa passagem direta de BaseURL = srv.URL sem usar SetEquiplanoAPIBaseURL
	res, err := transparencia.QueryEquiplanoReceitas(ctx, &http.Client{Timeout: 3 * time.Second}, m, transparencia.PortalQueryParams{
		BaseURL:  srv.URL,
		TenantID: "e1|m1|c1",
		Ano:      2025,
	})
	if err != nil {
		t.Fatalf("QueryEquiplanoReceitas top-level array erro: %v", err)
	}
	if len(res.Receitas) != 1 || res.Receitas[0].Codigo != "1.2.1" || res.Receitas[0].ValorArrecadado != 900 {
		t.Errorf("receita top-level inesperada: %+v", res.Receitas)
	}
}
