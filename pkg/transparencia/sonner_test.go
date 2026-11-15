package transparencia_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Mcp-Brasil/mcp-brasil/pkg/transparencia"
)

func TestProbeSonner(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	t.Run("normalizes_portal_and_portalcidadao", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/GRP/pic/api/config/contexto" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"dadosOrgao": {"nome": "Prefeitura Municipal de Uberlândia", "municipio": "Uberlândia"},
					"exercicioAtual": 2025
				}`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		for _, candidate := range []string{
			srv.URL,
			srv.URL + "/",
			srv.URL + "/portal",
			srv.URL + "/portal/",
			srv.URL + "/portalcidadao",
			srv.URL + "/portalcidadao/",
		} {
			nome, ok := transparencia.ProbeSonner(ctx, client, candidate)
			if !ok {
				t.Fatalf("esperava ProbeSonner=true para %q", candidate)
			}
			if nome != "Prefeitura Municipal de Uberlândia" {
				t.Errorf("entidadeNome inesperado para %q: %q", candidate, nome)
			}
		}
	})

	t.Run("non_sonner", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}))
		defer srv.Close()

		if nome, ok := transparencia.ProbeSonner(ctx, client, srv.URL); ok || nome != "" {
			t.Fatalf("esperava ProbeSonner=false, obteve ok=%v nome=%q", ok, nome)
		}
	})
}

func TestSonnerSessionAndQueriesMock(t *testing.T) {
	const (
		csrfToken  = "sonner-xsrf-token-9988"
		sessionID  = "JSESSIONID-SONNER-1"
		keyLic     = "101"
		keyCont    = "102"
		keyObra    = "103"
		keyConv    = "104"
		keySalario = "105"
	)

	var (
		funcTreeCalls     int
		folhaEnrichCalls  int
		seenLicHeader     string
		seenContHeader    string
		seenObraHeader    string
		seenConvHeader    string
		seenSalarioHeader string
		seenFiltroToken   string
	)

	checkCSRF := func(w http.ResponseWriter, r *http.Request) bool {
		c, err := r.Cookie("PORTAL-XSRF-TOKEN")
		if err != nil || c.Value != csrfToken {
			http.Error(w, "missing PORTAL-XSRF-TOKEN cookie", http.StatusForbidden)
			return false
		}
		if r.Header.Get("X-PORTAL-XSRF-TOKEN") != csrfToken {
			http.Error(w, "missing X-PORTAL-XSRF-TOKEN header", http.StatusForbidden)
			return false
		}
		sc, err := r.Cookie("JSESSIONID")
		if err != nil || sc.Value != sessionID {
			http.Error(w, "missing JSESSIONID cookie", http.StatusForbidden)
			return false
		}
		return true
	}

	readAno := func(r *http.Request) int {
		b, _ := io.ReadAll(r.Body)
		var body struct {
			Ano int `json:"ano"`
		}
		_ = json.Unmarshal(b, &body)
		return body.Ano
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/GRP/pic/api/auth/csrf":
			http.SetCookie(w, &http.Cookie{Name: "PORTAL-XSRF-TOKEN", Value: csrfToken, Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "JSESSIONID", Value: sessionID, Path: "/"})
			_, _ = w.Write([]byte(`{"ok":true}`))

		case "/GRP/pic/api/funcionalidade":
			funcTreeCalls++
			node := r.URL.Query().Get("node")
			switch node {
			case "":
				_, _ = w.Write([]byte(`[
					{"id": "pasta-compras", "text": "Compras e Licitações", "leaf": false},
					{"id": "pasta-obras-conv", "text": "Obras e Convênios", "leaf": false},
					{"id": 105, "text": "Folha de Pagamento", "rota": "visualiza-salario", "leaf": true}
				]`))
			case "pasta-compras":
				_, _ = w.Write([]byte(`[
					{"id": 101, "text": "Licitações", "url": "/consulta-licitacao", "leaf": true},
					{"id": 102, "text": "Contratos", "url": "/consulta-contrato", "leaf": true}
				]`))
			case "pasta-obras-conv":
				_, _ = w.Write([]byte(`[
					{"key": "103", "text": "Obras", "caminho": "listagem-obra", "leaf": true},
					{"key": "104", "text": "Convênios", "caminho": "cbp-convenios", "leaf": true}
				]`))
			default:
				_, _ = w.Write([]byte(`[]`))
			}

		case "/GRP/pic/api/visualiza-salario/funcionario":
			seenSalarioHeader = r.Header.Get("X-Portal-Funcionalidade")
			seenFiltroToken = r.URL.Query().Get("token")
			// Gera 12 funcionários com campos sensíveis (CPF bruto, endereço, banco) que DEVEM ser descartados/mascarados
			var rows []map[string]any
			for i := 1; i <= 12; i++ {
				rows = append(rows, map[string]any{
					"id":        1000 + i,
					"matricula": fmt.Sprintf("MAT-%03d", i),
					"nome":      fmt.Sprintf("SERVIDOR TESTE %02d", i),
					"cpf":       "12345678901",
					"cargo":     "Analista Administrativo",
					"lotacao":   "Secretaria de Administração",
					"vinculo":   "Efetivo",
					"situacao":  "Ativo",
					"endereco":  "Rua Secreta, 999",
					"bairro":    "Bairro Privado",
					"cep":       "38400-000",
					"banco":     "001",
					"agencia":   "1234-5",
					"conta":     "98765-4",
					"pix":       "chave-secreta@example.com",
				})
			}
			_ = json.NewEncoder(w).Encode(rows)

		case "/GRP/pic/api/visualiza-salario/anos-folha":
			_, _ = w.Write([]byte(`[2024, 2025]`))

		case "/GRP/pic/api/visualiza-salario/matriculafolhas":
			folhaEnrichCalls++
			_, _ = w.Write([]byte(`[
				{
					"ano": 2025,
					"mes": 4,
					"salarioBase": 4500.00,
					"totalProventos": 5800.00,
					"totalDescontos": 1100.00,
					"totalLiquido": 4700.00,
					"cpf": "12345678901",
					"banco": "001",
					"conta": "98765-4",
					"endereco": "Rua Secreta, 999"
				}
			]`))

		case "/GRP/pic/api/dashboard/data/despesas":
			if !checkCSRF(w, r) {
				return
			}
			ano := readAno(r)
			if ano != 2025 {
				_, _ = w.Write([]byte(`{}`))
				return
			}
			// Simula retorno agregado do dashboard (sem lista), disparando drill para obter itens
			_, _ = w.Write([]byte(`{"empenhado": 5000000.00, "liquidado": 4200000.00, "pago": 3900000.00}`))

		case "/GRP/pic/api/dashboard/drill/despesas":
			if !checkCSRF(w, r) {
				return
			}
			_, _ = w.Write([]byte(`[
				{
					"numeroEmpenho": "777/2025",
					"ano": 2025,
					"data": "2025-03-15",
					"fornecedor": "TECH SOLUCOES LTDA",
					"cpfCnpj": "22.333.444/0001-55",
					"orgao": "Secretaria de Fazenda",
					"historico": "Licenciamento de software",
					"valorEmpenhado": 120000.00,
					"valorLiquidado": 60000.00,
					"valorPago": 60000.00
				},
				{
					"numeroEmpenho": "778/2025",
					"ano": 2025,
					"data": "2025-03-16",
					"fornecedor": "MARIA MEDICA",
					"cpfCnpj": "33344455566",
					"orgao": "Secretaria de Saúde",
					"historico": "Atendimento ambulatorial",
					"valorEmpenhado": 8000.00,
					"valorLiquidado": 8000.00,
					"valorPago": 8000.00
				}
			]`))

		case "/GRP/pic/api/dashboard/data/receitas":
			if !checkCSRF(w, r) {
				return
			}
			_, _ = w.Write([]byte(`[
				{
					"codigo": "1.1.1.2.00",
					"descricao": "ISSQN",
					"categoria": "Impostos",
					"valorPrevisto": 15000000.00,
					"valorArrecadado": 16250000.00
				}
			]`))

		case "/GRP/pic/api/consulta-licitacao":
			if !checkCSRF(w, r) {
				return
			}
			seenLicHeader = r.Header.Get("X-Portal-Funcionalidade")
			if readAno(r) != 2025 {
				_, _ = w.Write([]byte(`[]`))
				return
			}
			_, _ = w.Write([]byte(`{
				"registros": [
					{
						"numeroLicitacao": "045/2025",
						"ano": 2025,
						"modalidade": "Concorrência Eletrônica",
						"situacao": "Em Andamento",
						"dataAbertura": "2025-04-10",
						"objeto": "Reforma de escola municipal",
						"valorEstimado": 890000.00,
						"valorHomologado": 845000.00
					}
				],
				"total": 1
			}`))

		case "/GRP/pic/api/consulta-contrato":
			if !checkCSRF(w, r) {
				return
			}
			seenContHeader = r.Header.Get("X-Portal-Funcionalidade")
			_, _ = w.Write([]byte(`[
				{
					"numeroContrato": "210/2025",
					"ano": 2025,
					"fornecedor": "ENGENHARIA SUL LTDA",
					"cpfCnpj": "44.555.666/0001-77",
					"objeto": "Serviços de engenharia",
					"situacao": "Ativo",
					"dataAssinatura": "2025-02-01",
					"vigenciaInicio": "2025-02-01",
					"vigenciaFim": "2026-01-31",
					"valor": 540000.00
				},
				{
					"numeroContrato": "211/2025",
					"ano": 2025,
					"fornecedor": "PEDRO PERITO",
					"cpfCnpj": "99988877766",
					"objeto": "Perícia técnica",
					"valor": 15000.00
				}
			]`))

		case "/GRP/pic/api/listagem-obra/consultar":
			if !checkCSRF(w, r) {
				return
			}
			seenObraHeader = r.Header.Get("X-Portal-Funcionalidade")
			_, _ = w.Write([]byte(`[
				{
					"codigo": "OBR-19",
					"titulo": "UBS Central",
					"objeto": "Ampliação de Unidade Básica de Saúde",
					"orgao": "Secretaria de Saúde",
					"endereco": "Av. Brasil, 500",
					"bairro": "Centro",
					"dataInicial": "2025-01-05",
					"dataFinal": "2025-12-20",
					"situacao": "Em Execução",
					"valor": 1350000.00
				}
			]`))

		case "/GRP/pic/api/cbp-convenios/consultar":
			if !checkCSRF(w, r) {
				return
			}
			seenConvHeader = r.Header.Get("X-Portal-Funcionalidade")
			_, _ = w.Write([]byte(`[
				{
					"numero": "CV-303/2025",
					"concedente": "Ministério da Saúde",
					"convenente": "Município de Uberlândia",
					"categoria": "Federal",
					"tipo": "Repasse",
					"situacao": "Vigente",
					"data": "2025-02-20",
					"vigencia": "2026-02-20",
					"descricao": "Aquisição de equipamentos hospitalares",
					"valor": 750000.00,
					"valorLiberado": 375000.00
				}
			]`))

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	m := &transparencia.Municipio{CodigoIBGE: "3170206", Nome: "Uberlândia", UF: "MG"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	// 1. Servidores (sem NomeFiltro -> usa "A", enriquece no máx 10 dos 12 registros, mascara CPF e remove endereço/banco)
	servRes, err := transparencia.QuerySonnerServidores(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL + "/portalcidadao",
		Limit:   12,
	})
	if err != nil {
		t.Fatalf("QuerySonnerServidores erro: %v", err)
	}
	if seenFiltroToken != "A" {
		t.Errorf("esperava token='A' quando NomeFiltro vazio, obteve %q", seenFiltroToken)
	}
	if seenSalarioHeader != keySalario {
		t.Errorf("esperava X-Portal-Funcionalidade=%q em visualiza-salario, obteve %q", keySalario, seenSalarioHeader)
	}
	if funcTreeCalls == 0 {
		t.Errorf("esperava chamadas à árvore /funcionalidade")
	}
	if len(servRes.Servidores) != 12 {
		t.Fatalf("esperava 12 servidores, obteve %d", len(servRes.Servidores))
	}
	if folhaEnrichCalls != 10 {
		t.Errorf("esperava exatamente 10 chamadas de enriquecimento (N<=10), obteve %d", folhaEnrichCalls)
	}
	if servRes.Servidores[0].SalarioBruto != 5800.00 || servRes.Servidores[0].SalarioLiquido != 4700.00 || servRes.Servidores[0].Competencia != "04/2025" {
		t.Errorf("enriquecimento folha[0] inesperado: %+v", servRes.Servidores[0])
	}
	if servRes.Servidores[11].SalarioBruto != 0 {
		t.Errorf("registro 12 (índice 11) não deveria ser enriquecido além do teto 10: %+v", servRes.Servidores[11])
	}

	// Verifica conformidade LGPD no JSON serializado completo
	serialized, _ := json.Marshal(servRes)
	serStr := string(serialized)
	for _, forbidden := range []string{"12345678901", "Rua Secreta", "Bairro Privado", "38400-000", "98765-4", "chave-secreta"} {
		if strings.Contains(serStr, forbidden) {
			t.Fatalf("dado sensível vazou no JSON serializado (%q): %s", forbidden, serStr)
		}
	}
	if servRes.Servidores[0].CPFParcial != "***.456.789-**" {
		t.Errorf("CPF não mascarado: %q", servRes.Servidores[0].CPFParcial)
	}

	// 2. Despesas (testa fallback para /dashboard/drill/despesas + máscara de CPF)
	despRes, err := transparencia.QuerySonnerDespesas(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL + "/portal",
		Ano:     2025,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QuerySonnerDespesas erro: %v", err)
	}
	if len(despRes.Despesas) != 2 || despRes.Despesas[0].NumeroEmpenho != "777/2025" {
		t.Fatalf("DespesasResult inesperado: %+v", despRes)
	}
	if despRes.Despesas[1].DocumentoFornecedor != "***.444.555-**" {
		t.Errorf("CPF em despesa Sonner não mascarado: %q", despRes.Despesas[1].DocumentoFornecedor)
	}

	// 3. Receitas
	recRes, err := transparencia.QuerySonnerReceitas(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Ano:     2025,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QuerySonnerReceitas erro: %v", err)
	}
	if len(recRes.Receitas) != 1 || recRes.Receitas[0].Codigo != "1.1.1.2.00" || recRes.Receitas[0].ValorArrecadado != 16250000.00 {
		t.Errorf("ReceitasResult inesperado: %+v", recRes)
	}

	// 4. Licitações
	licRes, err := transparencia.QuerySonnerLicitacoes(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Ano:     2025,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QuerySonnerLicitacoes erro: %v", err)
	}
	if seenLicHeader != keyLic {
		t.Errorf("esperava X-Portal-Funcionalidade=%q em consulta-licitacao, obteve %q", keyLic, seenLicHeader)
	}
	if len(licRes.Licitacoes) != 1 || licRes.Licitacoes[0].Numero != "045/2025" || licRes.Licitacoes[0].ValorHomologado != 845000.00 {
		t.Errorf("LicitacoesResult inesperado: %+v", licRes)
	}

	// 5. Contratos
	contRes, err := transparencia.QuerySonnerContratos(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Ano:     2025,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QuerySonnerContratos erro: %v", err)
	}
	if seenContHeader != keyCont {
		t.Errorf("esperava X-Portal-Funcionalidade=%q em consulta-contrato, obteve %q", keyCont, seenContHeader)
	}
	if len(contRes.Contratos) != 2 || contRes.Contratos[0].Numero != "210/2025" || contRes.Contratos[1].DocumentoFornecedor != "***.888.777-**" {
		t.Errorf("ContratosResult inesperado: %+v", contRes)
	}

	// 6. Obras
	obraRes, err := transparencia.QuerySonnerObras(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Ano:     2025,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QuerySonnerObras erro: %v", err)
	}
	if seenObraHeader != keyObra {
		t.Errorf("esperava X-Portal-Funcionalidade=%q em listagem-obra, obteve %q", keyObra, seenObraHeader)
	}
	if len(obraRes.Obras) != 1 || obraRes.Obras[0].Codigo != "OBR-19" || obraRes.Obras[0].Valor != 1350000.00 {
		t.Errorf("ObrasResult inesperado: %+v", obraRes)
	}

	// 7. Convênios
	convRes, err := transparencia.QuerySonnerConvenios(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Ano:     2025,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QuerySonnerConvenios erro: %v", err)
	}
	if seenConvHeader != keyConv {
		t.Errorf("esperava X-Portal-Funcionalidade=%q em cbp-convenios, obteve %q", keyConv, seenConvHeader)
	}
	if len(convRes.Convenios) != 1 || convRes.Convenios[0].Numero != "CV-303/2025" || convRes.Convenios[0].ValorLiberado != 375000.00 {
		t.Errorf("ConveniosResult inesperado: %+v", convRes)
	}
}

func TestSonnerXSRFFallbackAndAggregateDashboard(t *testing.T) {
	const xsrfFallback = "xsrf-fallback-token-55"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/GRP/pic/api/auth/csrf":
			// Usa XSRF-TOKEN em vez de PORTAL-XSRF-TOKEN para testar fallback
			http.SetCookie(w, &http.Cookie{Name: "XSRF-TOKEN", Value: xsrfFallback, Path: "/"})
			_, _ = w.Write([]byte(`{}`))
		case "/GRP/pic/api/funcionalidade":
			_, _ = w.Write([]byte(`[]`))
		case "/GRP/pic/api/dashboard/data/despesas":
			if r.Header.Get("X-PORTAL-XSRF-TOKEN") != xsrfFallback {
				http.Error(w, "invalid csrf", http.StatusForbidden)
				return
			}
			_, _ = w.Write([]byte(`{"empenhado": 1000000.50, "liquidado": 850000.25, "pago": 700000.00}`))
		case "/GRP/pic/api/dashboard/drill/despesas":
			http.NotFound(w, r)
		case "/GRP/pic/api/dashboard/data/receitas":
			if r.Header.Get("X-PORTAL-XSRF-TOKEN") != xsrfFallback {
				http.Error(w, "invalid csrf", http.StatusForbidden)
				return
			}
			_, _ = w.Write([]byte(`{"previsto": 2000000.00, "arrecadado": 1950000.75}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	m := &transparencia.Municipio{CodigoIBGE: "3170206", Nome: "Uberlândia", UF: "MG"}
	ctx := context.Background()

	despRes, err := transparencia.QuerySonnerDespesas(ctx, nil, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Ano:     2025,
	})
	if err != nil {
		t.Fatalf("QuerySonnerDespesas agregado erro: %v", err)
	}
	if len(despRes.Despesas) != 1 || despRes.Despesas[0].ValorEmpenhado != 1000000.50 || despRes.Despesas[0].ValorPago != 700000.00 {
		t.Errorf("Despesas agregado inesperado: %+v", despRes)
	}

	recRes, err := transparencia.QuerySonnerReceitas(ctx, nil, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Ano:     2025,
	})
	if err != nil {
		t.Fatalf("QuerySonnerReceitas agregado erro: %v", err)
	}
	if len(recRes.Receitas) != 1 || recRes.Receitas[0].ValorPrevisto != 2000000.00 || recRes.Receitas[0].ValorArrecadado != 1950000.75 {
		t.Errorf("Receitas agregado inesperado: %+v", recRes)
	}
}
