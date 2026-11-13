package transparencia_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gibavargas/mcp-brasil/pkg/transparencia"
)

func TestExtractFacilDatabaseID(t *testing.T) {
	if id := transparencia.ExtractFacilDatabaseID("http://www.transparenciafacil.com.br/0144301"); id != "0144301" {
		t.Errorf("ExtractFacilDatabaseID transparenciafacil = %q, esperava '0144301'", id)
	}
	if id := transparencia.ExtractFacilDatabaseID("https://relatorios.portalfacil.com.br/123456/#/receitas"); id != "123456" {
		t.Errorf("ExtractFacilDatabaseID portalfacil = %q, esperava '123456'", id)
	}
	html := `<iframe src="https://www.transparenciafacil.com.br/0054321"></iframe>`
	if id := transparencia.ExtractFacilDatabaseID(html); id != "0054321" {
		t.Errorf("ExtractFacilDatabaseID HTML = %q, esperava '0054321'", id)
	}
	if id := transparencia.ExtractFacilDatabaseID("https://prefeitura.exemplo.gov.br/2025"); id != "" {
		t.Errorf("esperava vazio para ano 4 dígitos, obteve %q", id)
	}
}

func TestFetchFacilEntidadesAndProbe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/geral/GetEntidades":
			if r.URL.Query().Get("search") != "Exemplo" {
				_, _ = w.Write([]byte(`[]`))
				return
			}
			_, _ = w.Write([]byte(`[
				{"databaseId":"0144301","nome":"PREFEITURA MUNICIPAL DE EXEMPLO","uf":"MG","cnpj":"18.000.000/0001-00","municipio":"Exemplo"}
			]`))
		case "/api/receita/ReceitasGetAno":
			if r.URL.Query().Get("databaseId") == "0144301" {
				_, _ = w.Write([]byte(`[2025, 2024, 2023]`))
				return
			}
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	transparencia.SetFacilBaseURL(srv.URL)
	defer transparencia.SetFacilBaseURL("")

	ents, err := transparencia.FetchFacilEntidades(ctx, client, "Exemplo")
	if err != nil {
		t.Fatalf("FetchFacilEntidades erro: %v", err)
	}
	if len(ents) != 1 || ents[0].DatabaseID != "0144301" || ents[0].UF != "MG" {
		t.Fatalf("entidades inesperadas: %+v", ents)
	}

	if !transparencia.ProbeFacil(ctx, client, srv.URL+"/0144301", "") {
		t.Fatalf("esperava ProbeFacil=true extraindo dbID da URL")
	}
	if transparencia.ProbeFacil(ctx, client, srv.URL, "999999") {
		t.Fatalf("esperava ProbeFacil=false quando ReceitasGetAno não retorna array")
	}
}

func TestFacilAllQueriesMock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "3124005", Nome: "Exemplo", UF: "MG"}

	var (
		seenDespesaFornecedorNonNull bool
		seenDespesaFornecedorVal     string
		seenLicitacaoModalidade      string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Query().Get("databaseId") != "0144301" {
			http.Error(w, "missing databaseId", http.StatusBadRequest)
			return
		}

		raw, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(raw, &payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		params, ok := payload["parameters"].(map[string]any)
		if !ok || params["draw"] == nil || params["length"] == nil {
			http.Error(w, "missing DataTables parameters", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/servidor/ServidorGetNomeGrid":
			_, _ = w.Write([]byte(`{
				"draw": 1,
				"recordsTotal": 42,
				"recordsFiltered": 42,
				"data": [
					{
						"nome": "LUCAS MENDES",
						"matricula": "778",
						"cpf": "44455566677",
						"cargo": "Professor",
						"lotacao": "Educação",
						"salarioBase": 3800.00,
						"salarioBruto": 4300.00,
						"descontos": 650.00,
						"salarioLiquido": 3650.00
					}
				]
			}`))
		case "/api/despesa/DespesasGetEmpenhoGrid/":
			val, exists := payload["fornecedor"]
			if exists && val != nil {
				if s, isStr := val.(string); isStr {
					seenDespesaFornecedorNonNull = true
					seenDespesaFornecedorVal = s
				}
			}
			_, _ = w.Write([]byte(`{
				"draw": 1,
				"recordsTotal": 2,
				"data": [
					{
						"numeroEmpenho": "101/2025",
						"ano": 2025,
						"data": "2025-02-01",
						"fornecedor": "AUTO POSTO CENTRAL LTDA",
						"cpfCnpj": "33.444.555/0001-66",
						"historico": "Combustível frota",
						"valorEmpenhado": 18000.00,
						"valorLiquidado": 12000.00,
						"valorPago": 12000.00
					},
					{
						"numeroEmpenho": "102/2025",
						"ano": 2025,
						"data": "2025-02-02",
						"fornecedor": "MARIA CLARA",
						"cpfCnpj": "77788899900",
						"historico": "Diária",
						"valorEmpenhado": 450.00
					}
				]
			}`))
		case "/api/receita/ReceitasGetGrid":
			_, _ = w.Write([]byte(`{
				"draw": 1,
				"recordsTotal": 1,
				"data": [
					{
						"codigo": "1.1.1.3.03",
						"descricao": "ISSQN - Imposto Sobre Serviços",
						"categoria": "Receitas Correntes",
						"valorPrevisto": 2500000.00,
						"valorArrecadado": 910500.25
					}
				]
			}`))
		case "/api/licitacao/LicitacaoGetGrid":
			if mod, ok := payload["modalidade"].(string); ok {
				seenLicitacaoModalidade = mod
			}
			_, _ = w.Write([]byte(`{
				"draw": 1,
				"recordsTotal": 1,
				"data": [
					{
						"numero": "015/2025",
						"ano": 2025,
						"modalidade": "Pregão Eletrônico",
						"situacao": "Homologada",
						"dataAbertura": "2025-02-12",
						"objeto": "Aquisição de computadores",
						"valorEstimado": 150000.00,
						"valorHomologado": 142000.00
					}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Usa URL com sufixo /0144301 para validar extração de dbID e limpeza de path antes de /api/...
	p := transparencia.PortalQueryParams{
		BaseURL: srv.URL + "/0144301",
		Ano:     2025,
		Limit:   10,
	}

	// 1. Servidores
	servRes, err := transparencia.QueryFacilServidores(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryFacilServidores erro: %v", err)
	}
	if servRes.Plataforma != "facil_transparencia" || servRes.TotalRegistros != 42 || len(servRes.Servidores) != 1 {
		t.Fatalf("ServidoresResult inesperado: %+v", servRes)
	}
	if servRes.Servidores[0].CPFParcial != "***.555.666-**" || servRes.Servidores[0].SalarioLiquido != 3650.00 {
		t.Errorf("servidor[0] inesperado: %+v", servRes.Servidores[0])
	}

	// 2. Despesas (valida explicitamente "fornecedor": "" não-nulo)
	despRes, err := transparencia.QueryFacilDespesas(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryFacilDespesas erro: %v", err)
	}
	if !seenDespesaFornecedorNonNull || seenDespesaFornecedorVal != "" {
		t.Errorf("esperava campo 'fornecedor': '' não-nulo, got ok=%v val=%q", seenDespesaFornecedorNonNull, seenDespesaFornecedorVal)
	}
	if len(despRes.Despesas) != 2 {
		t.Fatalf("esperava 2 despesas, obteve %d", len(despRes.Despesas))
	}
	if despRes.Despesas[0].DocumentoFornecedor != "33.444.555/0001-66" {
		t.Errorf("CNPJ não deveria ser mascarado: %q", despRes.Despesas[0].DocumentoFornecedor)
	}
	if despRes.Despesas[1].DocumentoFornecedor != "***.888.999-**" {
		t.Errorf("CPF em despesa não mascarado: %q", despRes.Despesas[1].DocumentoFornecedor)
	}

	// 3. Receitas
	recRes, err := transparencia.QueryFacilReceitas(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryFacilReceitas erro: %v", err)
	}
	if len(recRes.Receitas) != 1 || recRes.Receitas[0].ValorArrecadado != 910500.25 {
		t.Errorf("ReceitasResult inesperado: %+v", recRes)
	}

	// 4. Licitações
	licRes, err := transparencia.QueryFacilLicitacoes(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryFacilLicitacoes erro: %v", err)
	}
	if seenLicitacaoModalidade != "" {
		t.Errorf("esperava modalidade='', got %q", seenLicitacaoModalidade)
	}
	if len(licRes.Licitacoes) != 1 || licRes.Licitacoes[0].ValorHomologado != 142000.00 {
		t.Errorf("LicitacoesResult inesperado: %+v", licRes)
	}
}
