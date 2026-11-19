package transparencia_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Mcp-Brasil/mcp-brasil/pkg/transparencia"
)

func TestProbeCenti(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/portal/getlistaorgaos/MG/serrasaudade":
			// Simula 502 transitório na primeira tentativa para este tenant
			if calls.Add(1) == 1 {
				http.Error(w, "Bad Gateway", http.StatusBadGateway)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"Id":1,"Nome":"PREFEITURA MUNICIPAL DE SERRA DA SAUDADE"}]`))
		case "/portal/getlistaorgaos/GO/goiania":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"Id":1,"Nome":"PREFEITURA MUNICIPAL DE GOIANIA"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	transparencia.SetCentiAPIBaseURL(srv.URL)
	defer transparencia.SetCentiAPIBaseURL("")

	// Testa fallback sem preposição ("Serra da Saudade" -> "serrasaudade") + retry 502
	resolved, ok := transparencia.ProbeCenti(ctx, client, "mg", "Serra da Saudade")
	if !ok || resolved != "serrasaudade" {
		t.Fatalf("ProbeCenti = (%q, %v), esperava ('serrasaudade', true)", resolved, ok)
	}
	if calls.Load() < 2 {
		t.Errorf("esperava retry após HTTP 502, total calls=%d", calls.Load())
	}

	// Testa SlugifyCompact ("Goiânia" -> "goiania")
	resolved2, ok2 := transparencia.ProbeCenti(ctx, client, "GO", "Goiânia")
	if !ok2 || resolved2 != "goiania" {
		t.Fatalf("ProbeCenti Goiânia = (%q, %v), esperava ('goiania', true)", resolved2, ok2)
	}

	// Negativo
	if rNeg, okNeg := transparencia.ProbeCenti(ctx, client, "GO", "naoexiste"); okNeg || rNeg != "" {
		t.Fatalf("ProbeCenti negativo = (%q, %v), esperava ('', false)", rNeg, okNeg)
	}
}

func TestQueryCentiAllModules(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{
		CodigoIBGE: "5208707",
		Nome:       "Goiânia",
		UF:         "GO",
		Slug:       "goiania",
	}

	var (
		seenFuncBody      map[string]any
		seenEmpenhoBody   map[string]any
		seenReceitaBody   map[string]any
		seenContratoBody  map[string]any
		seenLicitacaoBody map[string]any
		empenhoCalls      atomic.Int32
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/portal/getfuncionarios/GO/goiania":
			_ = json.Unmarshal(raw, &seenFuncBody)
			_, _ = w.Write([]byte(`[
				{
					"Nome": "FERNANDA LIMA",
					"Matricula": "9012",
					"Cpf": "32165498700",
					"Cargo": "Analista",
					"Lotacao": "Fazenda",
					"SalarioBase": 5000.00,
					"SalarioBruto": 6200.00,
					"Descontos": 1100.00,
					"SalarioLiquido": 5100.00
				}
			]`))
		case "/portal/getempenhos/GO/goiania":
			if empenhoCalls.Add(1) == 1 {
				http.Error(w, "Bad Gateway", http.StatusBadGateway)
				return
			}
			_ = json.Unmarshal(raw, &seenEmpenhoBody)
			_, _ = w.Write([]byte(`[
				{
					"NumeroEmpenho": "2025/001",
					"Ano": 2025,
					"Data": "15/01/2025",
					"Credor": "MARCOS PEREIRA",
					"CpfCnpj": "45678912300",
					"Orgao": "Saúde",
					"Historico": "Prestação de serviços",
					"ValorEmpenhado": 15000.00,
					"ValorLiquidado": 15000.00,
					"ValorPago": 12000.00
				}
			]`))
		case "/portal/GetReceita/GO/goiania":
			_ = json.Unmarshal(raw, &seenReceitaBody)
			_, _ = w.Write([]byte(`[
				{
					"Codigo": "1.1.1.3",
					"Descricao": "ISSQN",
					"Categoria": "Tributária",
					"ValorPrevisto": 900000.00,
					"ValorArrecadado": 920000.00
				}
			]`))
		case "/portal/GetContratoPortal/GO/goiania":
			_ = json.Unmarshal(raw, &seenContratoBody)
			_, _ = w.Write([]byte(`[
				{
					"Numero": "CT-88/2025",
					"Ano": 2025,
					"Fornecedor": "ROBERTO DIAS",
					"CpfCnpj": "78945612300",
					"Objeto": "Manutenção de frota",
					"Valor": 60000.00
				}
			]`))
		case "/portal/ListaLicitacao/GO/goiania":
			_ = json.Unmarshal(raw, &seenLicitacaoBody)
			_, _ = w.Write([]byte(`[
				{
					"Numero": "PE-44/2025",
					"Ano": 2025,
					"Modalidade": "Pregão Eletrônico",
					"Situacao": "Aberta",
					"Objeto": "Aquisição de merenda",
					"ValorEstimado": 300000.00,
					"ValorHomologado": 280000.00
				}
			]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Usa override via p.BaseURL apontando para mock server e TenantID vazio (SlugifyCompact)
	params := transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Ano:     2025,
		Mes:     4,
		Limit:   12,
	}

	serv, err := transparencia.QueryCentiServidores(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryCentiServidores erro: %v", err)
	}
	if serv.Plataforma != "centi_api" || len(serv.Servidores) != 1 || serv.Servidores[0].CPFParcial != "***.654.987-**" {
		t.Fatalf("ServidoresResult Centi inesperado: %+v", serv)
	}
	if seenFuncBody["ano"] != float64(2025) || seenFuncBody["mes"] != float64(4) || seenFuncBody["idOrgao"] != float64(1) {
		t.Errorf("payload getfuncionarios inesperado: %+v", seenFuncBody)
	}

	desp, err := transparencia.QueryCentiDespesas(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryCentiDespesas erro: %v", err)
	}
	if len(desp.Despesas) != 1 || desp.Despesas[0].DocumentoFornecedor != "***.789.123-**" {
		t.Fatalf("DespesasResult Centi inesperado: %+v", desp)
	}
	if seenEmpenhoBody["DataInicio"] != "01/04/2025" || seenEmpenhoBody["DataFim"] != "30/04/2025" {
		t.Errorf("datas getempenhos inesperadas: %+v", seenEmpenhoBody)
	}
	if empenhoCalls.Load() != 2 {
		t.Errorf("esperava 2 chamadas em getempenhos (retry 502), obteve %d", empenhoCalls.Load())
	}

	rec, err := transparencia.QueryCentiReceitas(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryCentiReceitas erro: %v", err)
	}
	if len(rec.Receitas) != 1 || rec.Receitas[0].ValorArrecadado != 920000.00 {
		t.Fatalf("ReceitasResult Centi inesperado: %+v", rec)
	}
	if seenReceitaBody["Ano"] != float64(2025) || seenReceitaBody["Mes"] != float64(4) || seenReceitaBody["Orgao"] != float64(1) {
		t.Errorf("payload GetReceita inesperado: %+v", seenReceitaBody)
	}

	ct, err := transparencia.QueryCentiContratos(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryCentiContratos erro: %v", err)
	}
	if len(ct.Contratos) != 1 || ct.Contratos[0].DocumentoFornecedor != "***.456.123-**" {
		t.Fatalf("ContratosResult Centi inesperado: %+v", ct)
	}
	if seenContratoBody["ano"] != float64(2025) {
		t.Errorf("payload GetContratoPortal inesperado: %+v", seenContratoBody)
	}

	lic, err := transparencia.QueryCentiLicitacoes(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryCentiLicitacoes erro: %v", err)
	}
	if len(lic.Licitacoes) != 1 || lic.Licitacoes[0].ValorHomologado != 280000.00 {
		t.Fatalf("LicitacoesResult Centi inesperado: %+v", lic)
	}
	mods, okMods := seenLicitacaoBody["ModalidadeLicitacao"].([]any)
	if !okMods || len(mods) != 10 {
		t.Errorf("ModalidadeLicitacao inesperado: %+v", seenLicitacaoBody["ModalidadeLicitacao"])
	}
}
