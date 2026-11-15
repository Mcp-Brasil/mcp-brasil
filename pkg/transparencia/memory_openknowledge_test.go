package transparencia_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/Mcp-Brasil/mcp-brasil/pkg/transparencia"
)

func TestExtractMemoryTenant(t *testing.T) {
	tid, ent := transparencia.ExtractMemoryTenant("https://ilai.memory.com.br/#/entidades/login/AB12CD/101")
	if tid != "AB12CD" || ent != "101" {
		t.Fatalf("ExtractMemoryTenant URL = (%q, %q), esperava ('AB12CD', '101')", tid, ent)
	}

	html := `<html><body><a href="https://ilai.memory.com.br/#/entidades/login/xy98z1/42">Portal</a></body></html>`
	tid2, ent2 := transparencia.ExtractMemoryTenant(html)
	if tid2 != "XY98Z1" || ent2 != "42" {
		t.Fatalf("ExtractMemoryTenant HTML = (%q, %q), esperava ('XY98Z1', '42')", tid2, ent2)
	}

	tid3, ent3 := transparencia.ExtractMemoryTenant("https://outro.portal.gov.br/")
	if tid3 != "" || ent3 != "" {
		t.Fatalf("esperava vazio para URL não-Memory, obteve (%q, %q)", tid3, ent3)
	}
}

func TestProbeMemory(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	curYear := strconv.Itoa(time.Now().Year())
	var seenTenant, seenEnt, seenEx string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/lai/pessoal/estruturaorganizacional/" {
			seenTenant = r.Header.Get("tenant-id")
			seenEnt = r.Header.Get("entidade")
			seenEx = r.Header.Get("exercicio")
			if seenTenant == "ABC123" && seenEnt == "7" && seenEx == curYear {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[{"codigo":1,"descricao":"GABINETE DO PREFEITO"}]`))
				return
			}
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	transparencia.SetMemoryAPIBaseURL(srv.URL)
	defer transparencia.SetMemoryAPIBaseURL("")

	if !transparencia.ProbeMemory(ctx, client, "ABC123", "7") {
		t.Fatalf("esperava ProbeMemory=true")
	}
	if seenTenant != "ABC123" || seenEnt != "7" || seenEx != curYear {
		t.Errorf("headers inesperados: tenant-id=%q entidade=%q exercicio=%q", seenTenant, seenEnt, seenEx)
	}
	if transparencia.ProbeMemory(ctx, client, "INVALID", "7") {
		t.Fatalf("esperava ProbeMemory=false para tenant inválido")
	}
}

func TestMemoryAllQueriesMock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "3106200", Nome: "Exemplo", UF: "MG"}

	var seenReceitaHeaderPresent bool
	var seenReceitaHeaderVal string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("tenant-id") != "MEM001" || r.Header.Get("entidade") != "12" || r.Header.Get("exercicio") != "2025" {
			http.Error(w, "missing required tenant headers", http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("page") != "1" || r.URL.Query().Get("size") == "" {
			http.Error(w, "missing pagination query params", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/lai/pessoal/servidor/":
			_, _ = w.Write([]byte(`{
				"content": [
					{
						"nome": "ANA BEATRIZ",
						"matricula": "3344",
						"cpf": "11122233344",
						"cargo": "Médica",
						"lotacao": "Secretaria de Saúde",
						"vinculo": "Efetivo",
						"situacao": "Ativo",
						"salarioBase": 8500.00,
						"salarioBruto": 10200.00,
						"descontos": 2100.00,
						"salarioLiquido": 8100.00,
						"mes": "02"
					}
				],
				"totalElements": 85
			}`))
		case "/lai/contabilidade/despesa/empenho":
			_, _ = w.Write([]byte(`{
				"content": [
					{
						"numeroEmpenho": "2025/091",
						"exercicio": 2025,
						"dataEmpenho": "2025-02-10",
						"fornecedor": "ENERGIA MG SA",
						"cpfCnpj": "12.345.678/0001-99",
						"orgao": "Administração",
						"historico": "Fornecimento de energia elétrica",
						"valorEmpenhado": 15400.00,
						"valorLiquidado": 15400.00,
						"valorPago": 15400.00
					},
					{
						"numeroEmpenho": "2025/092",
						"exercicio": 2025,
						"dataEmpenho": "2025-02-11",
						"fornecedor": "PEDRO HENRIQUE",
						"cpfCnpj": "55566677788",
						"historico": "Serviço autônomo",
						"valorEmpenhado": 1200.00
					}
				],
				"totalElements": 2
			}`))
		case "/lai/contabilidade/receita/":
			vals, exists := r.Header[http.CanonicalHeaderKey("codigo_receita")]
			seenReceitaHeaderPresent = exists
			if len(vals) > 0 {
				seenReceitaHeaderVal = vals[0]
			}
			_, _ = w.Write([]byte(`[
				{
					"codigo": "1.7.1.8.01",
					"descricao": "FPM - Fundo de Participação dos Municípios",
					"categoria": "Transferências Correntes",
					"valorPrevisto": 22000000.00,
					"valorArrecadado": 5400000.50
				}
			]`))
		case "/lai/compras/contrato/":
			_, _ = w.Write([]byte(`{
				"items": [
					{
						"numeroContrato": "019/2025",
						"exercicio": 2025,
						"fornecedor": "TECNOLOGIA MINAS LTDA",
						"cnpj": "22.333.444/0001-55",
						"objeto": "Locação de sistema de gestão",
						"situacao": "Vigente",
						"dataAssinatura": "2025-01-05",
						"valor": 96000.00
					}
				],
				"total": 1
			}`))
		case "/lai/compras/licitacao/":
			_, _ = w.Write([]byte(`{
				"items": [
					{
						"numeroLicitacao": "007/2025",
						"exercicio": 2025,
						"modalidade": "Pregão Eletrônico",
						"situacao": "Concluída",
						"dataAbertura": "2025-01-25",
						"objeto": "Compra de merenda escolar",
						"valorEstimado": 350000.00,
						"valorHomologado": 332000.00
					}
				],
				"total": 1
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Testa override via SetMemoryAPIBaseURL quando BaseURL aponta para ilai.memory.com.br
	transparencia.SetMemoryAPIBaseURL(srv.URL)
	defer transparencia.SetMemoryAPIBaseURL("")

	ilaiURL := "https://ilai.memory.com.br/#/entidades/login/MEM001/12"

	servRes, err := transparencia.QueryMemoryServidores(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: ilaiURL,
		Ano:     2025,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QueryMemoryServidores erro: %v", err)
	}
	if servRes.Plataforma != "memory_openknowledge" || servRes.TotalRegistros != 85 || len(servRes.Servidores) != 1 {
		t.Fatalf("ServidoresResult inesperado: %+v", servRes)
	}
	if servRes.Servidores[0].CPFParcial != "***.222.333-**" || servRes.Servidores[0].Competencia != "02/2025" {
		t.Errorf("servidor[0] inesperado: %+v", servRes.Servidores[0])
	}

	// Testa override direto via p.BaseURL (sem depender de SetMemoryAPIBaseURL)
	transparencia.SetMemoryAPIBaseURL("")
	paramsDirect := transparencia.PortalQueryParams{
		BaseURL:          srv.URL,
		TenantID:         "MEM001",
		CodAdministracao: 12,
		Ano:              2025,
		Limit:            10,
	}

	despRes, err := transparencia.QueryMemoryDespesas(ctx, client, m, paramsDirect)
	if err != nil {
		t.Fatalf("QueryMemoryDespesas erro: %v", err)
	}
	if len(despRes.Despesas) != 2 {
		t.Fatalf("esperava 2 despesas, obteve %d", len(despRes.Despesas))
	}
	if despRes.Despesas[0].DocumentoFornecedor != "12.345.678/0001-99" {
		t.Errorf("CNPJ alterado indevidamente: %q", despRes.Despesas[0].DocumentoFornecedor)
	}
	if despRes.Despesas[1].DocumentoFornecedor != "***.666.777-**" {
		t.Errorf("CPF não mascarado na despesa: %q", despRes.Despesas[1].DocumentoFornecedor)
	}

	recRes, err := transparencia.QueryMemoryReceitas(ctx, client, m, paramsDirect)
	if err != nil {
		t.Fatalf("QueryMemoryReceitas erro: %v", err)
	}
	if !seenReceitaHeaderPresent || seenReceitaHeaderVal != "" {
		t.Errorf("esperava header codigo_receita='' presente, got present=%v val=%q", seenReceitaHeaderPresent, seenReceitaHeaderVal)
	}
	if len(recRes.Receitas) != 1 || recRes.Receitas[0].ValorArrecadado != 5400000.50 {
		t.Errorf("ReceitasResult inesperado: %+v", recRes)
	}

	contRes, err := transparencia.QueryMemoryContratos(ctx, client, m, paramsDirect)
	if err != nil {
		t.Fatalf("QueryMemoryContratos erro: %v", err)
	}
	if len(contRes.Contratos) != 1 || contRes.Contratos[0].Numero != "019/2025" || contRes.Contratos[0].Valor != 96000.00 {
		t.Errorf("ContratosResult inesperado: %+v", contRes)
	}

	licRes, err := transparencia.QueryMemoryLicitacoes(ctx, client, m, paramsDirect)
	if err != nil {
		t.Fatalf("QueryMemoryLicitacoes erro: %v", err)
	}
	if len(licRes.Licitacoes) != 1 || licRes.Licitacoes[0].Numero != "007/2025" || licRes.Licitacoes[0].ValorHomologado != 332000.00 {
		t.Errorf("LicitacoesResult inesperado: %+v", licRes)
	}
}
