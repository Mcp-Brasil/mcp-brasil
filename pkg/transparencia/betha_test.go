package transparencia_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Mcp-Brasil/mcp-brasil/pkg/transparencia"
)

func TestBethaConnectorMock(t *testing.T) {
	const (
		portalHash = "portal-hash-4216602"
		token1     = "tok-anon-001"
		token2     = "tok-anon-002"
	)

	var oauthCalls atomic.Int32
	var forceExpiredOnce atomic.Bool
	var seenAppCtx string
	var seenAgrupadorCalled bool
	var seenBuscaAgrupadorCalled bool
	var seenBuscaTextualFilter string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/oauth/authorize":
			n := oauthCalls.Add(1)
			q := r.URL.Query()
			if q.Get("response_type") != "token" ||
				q.Get("client_id") != "91a97459-f1d8-4b29-b5fa-2e51d1692623" ||
				q.Get("scope") != "transparencia.public" ||
				q.Get("bth_ignore_origin") != "true" ||
				q.Get("access_mode") != "anonymous" ||
				!strings.Contains(q.Get("redirect_uri"), "transparencia.betha.cloud/auth-callback.html") {
				http.Error(w, "bad oauth params", http.StatusBadRequest)
				return
			}
			tok := token1
			if n > 1 {
				tok = token2
			}
			w.Header().Set("Location", "https://transparencia.betha.cloud/auth-callback.html#access_token="+tok+"&expires_in=3600&token_type=bearer")
			w.WriteHeader(http.StatusSeeOther)
			return

		case r.URL.Path == "/auth/portais":
			auth := r.Header.Get("Authorization")
			if auth != "Bearer "+token1 && auth != "Bearer "+token2 {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			filter := r.URL.Query().Get("filter")
			if strings.Contains(filter, "4216602") {
				_, _ = w.Write([]byte(`{
					"content": [
						{"hash": "camara-hash", "nome": "Câmara Municipal de São José", "codigoIbge": 4216602},
						{"hash": "` + portalHash + `", "nome": "Prefeitura Municipal de São José", "codigoIbge": 4216602}
					]
				}`))
				return
			}
			_, _ = w.Write([]byte(`{"content":[]}`))
			return
		}

		// Demais rotas exigem Bearer + App-Context
		if forceExpiredOnce.CompareAndSwap(true, false) {
			http.Error(w, "expired", http.StatusUnauthorized)
			return
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+token1 && auth != "Bearer "+token2 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		seenAppCtx = r.Header.Get("App-Context")
		if seenAppCtx == "" {
			http.Error(w, "missing App-Context", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/menu":
			_, _ = w.Write([]byte(`[
				{
					"id": 1,
					"titulo": "Recursos Humanos",
					"itens": [
						{"id": 10, "titulo": "Folha de Pagamento - Servidores", "consultaId": 101, "tipo": "CONSULTA_PADRAO"}
					]
				},
				{
					"id": 2,
					"titulo": "Execução Orçamentária",
					"filhos": [
						{"id": 20, "titulo": "Despesas e Empenhos", "consultaId": 102, "tipo": "CONSULTA_PADRAO"},
						{"id": 21, "titulo": "Receitas Arrecadadas", "consultaId": 103, "tipo": "CONSULTA_PADRAO"}
					]
				},
				{"id": 30, "titulo": "Licitações e Editais", "consultaId": 104, "tipo": "CONSULTA_PADRAO"},
				{"id": 40, "titulo": "Contratos Administrativos", "consultaId": 105, "tipo": "CONSULTA_PADRAO"},
				{"id": 50, "titulo": "Diárias e Passagens", "consultaId": 106, "tipo": "AGRUPADOR"}
			]`))

		case "/api/busca-textual/101":
			bodyBytes, _ := io.ReadAll(r.Body)
			seenBuscaTextualFilter = string(bodyBytes)
			_, _ = w.Write([]byte(`{
				"total": 2,
				"content": [
					{
						"nome": "MARIA CLARA OLIVEIRA",
						"matricula": "10452",
						"cpf": "12345678901",
						"cargo": "Enfermeira",
						"lotacao": "Secretaria Municipal de Saúde",
						"vinculo": "Estatutário",
						"situacao": "Ativo",
						"salarioBase": 5000.00,
						"salarioBruto": 6500.50,
						"descontos": 1200.25,
						"competencia": "03/2025",
						"cargaHoraria": "40h"
					},
					{
						"nome": "CARLOS EDUARDO",
						"matricula": "10453",
						"cpfFormatado": "***.987.654-**",
						"cargo": "Professor",
						"salarioBruto": "4.200,00",
						"descontos": "700,00",
						"salarioLiquido": "3.500,00"
					}
				]
			}`))

		case "/api/busca-textual/102":
			_, _ = w.Write([]byte(`{
				"totalElements": 2,
				"content": [
					{
						"numeroEmpenho": "2025/001",
						"ano": 2025,
						"data": "2025-02-10",
						"fornecedor": "JOAO DA SILVA",
						"cpfCnpj": "123.456.789-01",
						"orgao": "Prefeitura Municipal",
						"historico": "Serviço técnico",
						"valorEmpenhado": 1500.00,
						"valorLiquidado": 1500.00,
						"valorPago": 1500.00
					},
					{
						"numeroEmpenho": "2025/002",
						"ano": 2025,
						"data": "2025-02-11",
						"fornecedor": "TECH LTDA",
						"cpfCnpj": "12.345.678/0001-99",
						"valorEmpenhado": 8500.00,
						"valorPago": 8500.00
					}
				]
			}`))

		case "/api/busca-textual/103":
			_, _ = w.Write([]byte(`{
				"total": 1,
				"content": [
					{
						"codigo": "1.1.1.0.00",
						"descricao": "IPTU",
						"categoria": "Receitas Tributárias",
						"data": "2025-03-01",
						"valorPrevisto": 1000000.00,
						"valorArrecadado": 750000.50
					}
				]
			}`))

		case "/api/busca-textual/104":
			_, _ = w.Write([]byte(`{
				"total": 1,
				"content": [
					{
						"numero": "PE 12/2025",
						"ano": 2025,
						"modalidade": "Pregão Eletrônico",
						"situacao": "Homologada",
						"dataAbertura": "2025-01-20",
						"objeto": "Aquisição de equipamentos",
						"valorEstimado": 50000.00,
						"valorHomologado": 47500.00
					}
				]
			}`))

		case "/api/busca-textual/105":
			_, _ = w.Write([]byte(`{
				"total": 1,
				"content": [
					{
						"numero": "CT 08/2025",
						"ano": 2025,
						"fornecedor": "MARCOS CONSULTOR",
						"cpfCnpj": "98765432100",
						"objeto": "Consultoria contábil",
						"situacao": "Vigente",
						"dataAssinatura": "2025-01-15",
						"vigenciaInicio": "2025-01-15",
						"vigenciaFim": "2025-12-31",
						"valor": 24000.00
					}
				]
			}`))

		case "/api/consulta/106/agrupadores/nivel/1":
			seenAgrupadorCalled = true
			_, _ = w.Write([]byte(`[
				{"id": 9001, "titulo": "2024"},
				{"id": 9002, "titulo": "2025"}
			]`))

		case "/api/busca/106/agrupador/9002":
			seenBuscaAgrupadorCalled = true
			if r.URL.Query().Get("offset") != "0" || r.URL.Query().Get("nivel") != "1" {
				http.Error(w, "bad query params", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{
				"total": 1,
				"content": [
					{
						"beneficiario": "ANA PAULA",
						"matricula": "3344",
						"cargo": "Secretária",
						"destino": "Florianópolis/SC",
						"motivo": "Reunião técnica",
						"data": "2025-03-05",
						"dataSaida": "2025-03-05",
						"dataChegada": "2025-03-06",
						"numeroDiarias": 1.5,
						"valor": 600.00
					}
				]
			}`))

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	transparencia.SetBethaOAuthURL(srv.URL + "/oauth/authorize")
	transparencia.SetBethaAPIBaseURL(srv.URL)
	t.Cleanup(func() {
		transparencia.SetBethaOAuthURL("https://plataforma-oauth.betha.cloud/auth/oauth2/authorize")
		transparencia.SetBethaAPIBaseURL("https://api.transparencia.betha.cloud/transparencia")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "4216602", Nome: "São José", UF: "SC"}

	// 1. Discovery
	disc, ok := transparencia.DiscoverBethaByIBGE(ctx, client, m)
	if !ok || disc == nil {
		t.Fatalf("esperava DiscoverBethaByIBGE=true")
	}
	if disc.Plataforma != "betha_cloud" ||
		disc.URLResolvida != "https://transparencia.betha.cloud/#/"+portalHash ||
		disc.TenantID != portalHash ||
		disc.EntidadeNome != "Prefeitura Municipal de São José" ||
		disc.FonteDescoberta != "betha_directory" ||
		!disc.ServidoresSuportado {
		t.Fatalf("PortalDiscoveryResult inesperado: %+v", disc)
	}

	if _, ok := transparencia.DiscoverBethaByIBGE(ctx, client, &transparencia.Municipio{CodigoIBGE: "9999999"}); ok {
		t.Fatalf("esperava false para IBGE inexistente")
	}

	params := transparencia.PortalQueryParams{
		BaseURL:    disc.URLResolvida,
		TenantID:   disc.TenantID,
		NomeFiltro: "MARIA",
		Ano:        2025,
		Limit:      10,
	}

	// 2. Servidores (e checa cache do token + App-Context)
	servRes, err := transparencia.QueryBethaServidores(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryBethaServidores erro: %v", err)
	}
	if oauthCalls.Load() != 1 {
		t.Errorf("esperava token OAuth cacheado em 1 chamada, obteve %d", oauthCalls.Load())
	}
	decodedCtx, err := base64.StdEncoding.DecodeString(seenAppCtx)
	if err != nil {
		t.Fatalf("App-Context base64 inválido: %v", err)
	}
	var ctxObj map[string]string
	if err := json.Unmarshal(decodedCtx, &ctxObj); err != nil || ctxObj["portal"] != portalHash {
		t.Errorf("App-Context inesperado: %s", string(decodedCtx))
	}
	if !strings.Contains(seenBuscaTextualFilter, "MARIA") {
		t.Errorf("esperava filtro textual no body POST, obteve %s", seenBuscaTextualFilter)
	}
	if len(servRes.Servidores) != 2 || servRes.TotalRegistros != 2 {
		t.Fatalf("ServidoresResult inesperado: %+v", servRes)
	}
	if servRes.Servidores[0].CPFParcial != "***.456.789-**" {
		t.Errorf("CPF não mascarado corretamente: %q", servRes.Servidores[0].CPFParcial)
	}
	if servRes.Servidores[0].SalarioLiquido != 5300.25 {
		t.Errorf("esperava SalarioLiquido calculado=5300.25, obteve %v", servRes.Servidores[0].SalarioLiquido)
	}
	if servRes.Servidores[1].CPFParcial != "***.987.654-**" || servRes.Servidores[1].SalarioBruto != 4200.00 {
		t.Errorf("servidor[1] inesperado: %+v", servRes.Servidores[1])
	}

	// 3. Despesas (força 401 uma vez para validar renovação automática do token OAuth2)
	forceExpiredOnce.Store(true)
	despRes, err := transparencia.QueryBethaDespesas(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryBethaDespesas erro: %v", err)
	}
	if oauthCalls.Load() != 2 {
		t.Errorf("esperava renovação do token após 401 (2 chamadas OAuth), obteve %d", oauthCalls.Load())
	}
	if len(despRes.Despesas) != 2 {
		t.Fatalf("DespesasResult inesperado: %+v", despRes)
	}
	if despRes.Despesas[0].DocumentoFornecedor != "***.456.789-**" {
		t.Errorf("CPF de credor deveria estar mascarado: %q", despRes.Despesas[0].DocumentoFornecedor)
	}
	if despRes.Despesas[1].DocumentoFornecedor != "12.345.678/0001-99" {
		t.Errorf("CNPJ não deveria ser mascarado: %q", despRes.Despesas[1].DocumentoFornecedor)
	}

	// 4. Receitas
	recRes, err := transparencia.QueryBethaReceitas(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryBethaReceitas erro: %v", err)
	}
	if len(recRes.Receitas) != 1 || recRes.Receitas[0].ValorArrecadado != 750000.50 {
		t.Errorf("ReceitasResult inesperado: %+v", recRes)
	}

	// 5. Licitações
	licRes, err := transparencia.QueryBethaLicitacoes(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryBethaLicitacoes erro: %v", err)
	}
	if len(licRes.Licitacoes) != 1 || licRes.Licitacoes[0].Modalidade != "Pregão Eletrônico" {
		t.Errorf("LicitacoesResult inesperado: %+v", licRes)
	}

	// 6. Contratos (também mascara CPF de fornecedor)
	contRes, err := transparencia.QueryBethaContratos(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryBethaContratos erro: %v", err)
	}
	if len(contRes.Contratos) != 1 || contRes.Contratos[0].DocumentoFornecedor != "***.654.321-**" {
		t.Errorf("ContratosResult inesperado: %+v", contRes)
	}

	// 7. Diárias (via AGRUPADOR -> /api/consulta/106/agrupadores/nivel/1 + /api/busca/106/agrupador/9002)
	diarRes, err := transparencia.QueryBethaDiarias(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryBethaDiarias erro: %v", err)
	}
	if !seenAgrupadorCalled || !seenBuscaAgrupadorCalled {
		t.Errorf("esperava fluxo AGRUPADOR chamado: agrupador=%v busca=%v", seenAgrupadorCalled, seenBuscaAgrupadorCalled)
	}
	if len(diarRes.Diarias) != 1 || diarRes.Diarias[0].Destino != "Florianópolis/SC" || diarRes.Diarias[0].Valor != 600.00 {
		t.Errorf("DiariasResult inesperado: %+v", diarRes)
	}
}
