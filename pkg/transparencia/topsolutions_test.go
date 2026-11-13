package transparencia_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gibavargas/mcp-brasil/pkg/transparencia"
)

func TestResolveTopSolutionsAPIBase(t *testing.T) {
	got := transparencia.ResolveTopSolutionsAPIBase("https://pmcurraisnovosrn.transparencia.topsolutionsrn.com.br/")
	want := "https://pmcurraisnovosrn.apitransparencia.topsolutionsrn.com.br"
	if got != want {
		t.Fatalf("ResolveTopSolutionsAPIBase = %q, want %q", got, want)
	}
}

func TestTopSolutionsConnectorAllModules(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/receitaprevistaarrecadada/receitaprevistaarrecadadaasync":
			_, _ = w.Write([]byte(`{"data":[{"codCategoria":"1000","txtDescricaoCategoria":"Receitas Correntes","vlrPrevisaoAtualizado":500000,"vlrArrecadacao":450000}]}`))
		case "/despesa/despesaporclassificacaoasync":
			if r.URL.Query().Get("dtIni") != "01/01/2025" {
				http.Error(w, "bad date", http.StatusInternalServerError)
				return
			}
			_, _ = w.Write([]byte(`{"data":[{"codOrgao":"02","codUnidade":"001","txtDescricaoUnidade":"GABINETE DO PREFEITO","vlrEmpenhado":120000.50,"vlrLiquidado":110000,"vlrPago":100000,"exercicio":2025},{"exercicio":1900,"vlrEmpenhado":0,"vlrLiquidado":0,"vlrPago":0}]}`))
		case "/Servidor/ServidorPorMesAnoAsync":
			_, _ = w.Write([]byte(`{"data":[{"nome":"MARIA TOP","cpf":"12345678901","vinculo":"Estatutário","cargo":"PROFESSORA","orgao":"EDUCACAO","numMatricula":"001","vlrRemuneracaoBruta":5200.5,"vlrDescontosObrigatorios":800.5,"vlrRemuAposDescObrig":4400.0,"dtMesAno":"2025-09-01T00:00:00"}]}`))
		case "/contrato/contratoasync":
			_, _ = w.Write([]byte(`{"data":[{"numExercicioFinanc":2025,"numContrato":"10/2025","txtCpfCnpjContratada":"12345678901","txtNomeRazaoContratada":"JOAO PRESTADOR","txtModalidade":"Dispensa","txtObjeto":"SERVICO TESTE","vlrContrato":15000}]}`))
		case "/licitacao/LicitacaoAsync":
			_, _ = w.Write([]byte(`{"data":[{"numExercicioFinanc":2025,"numLicitacao":"22","txtModalidade":"Pregão Eletrônico","txtObjeto":"AQUISICAO MATERIAIS","vlrEstimado":90000}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	m := &transparencia.Municipio{CodigoIBGE: "2403103", Nome: "Currais Novos", UF: "RN"}
	p := transparencia.PortalQueryParams{BaseURL: srv.URL, Ano: 2025, Mes: 9, Limit: 10}

	apiBase, ok := transparencia.ProbeTopSolutions(ctx, srv.Client(), srv.URL)
	if !ok || apiBase != srv.URL {
		t.Fatalf("ProbeTopSolutions falhou: ok=%v apiBase=%q", ok, apiBase)
	}

	serv, err := transparencia.QueryTopSolutionsServidores(ctx, srv.Client(), m, p)
	if err != nil || len(serv.Servidores) != 1 || serv.Servidores[0].CPFParcial != "***.456.789-**" || serv.Servidores[0].SalarioBruto != 5200.5 {
		t.Fatalf("QueryTopSolutionsServidores inesperado: err=%v res=%+v", err, serv)
	}

	desp, err := transparencia.QueryTopSolutionsDespesas(ctx, srv.Client(), m, p)
	if err != nil || len(desp.Despesas) != 1 || desp.Despesas[0].ValorEmpenhado != 120000.50 {
		t.Fatalf("QueryTopSolutionsDespesas inesperado: err=%v res=%+v", err, desp)
	}

	rec, err := transparencia.QueryTopSolutionsReceitas(ctx, srv.Client(), m, p)
	if err != nil || len(rec.Receitas) != 1 || rec.Receitas[0].ValorArrecadado != 450000 {
		t.Fatalf("QueryTopSolutionsReceitas inesperado: err=%v res=%+v", err, rec)
	}

	ct, err := transparencia.QueryTopSolutionsContratos(ctx, srv.Client(), m, p)
	if err != nil || len(ct.Contratos) != 1 || ct.Contratos[0].DocumentoFornecedor != "***.456.789-**" {
		t.Fatalf("QueryTopSolutionsContratos inesperado: err=%v res=%+v", err, ct)
	}

	lic, err := transparencia.QueryTopSolutionsLicitacoes(ctx, srv.Client(), m, p)
	if err != nil || len(lic.Licitacoes) != 1 || lic.Licitacoes[0].ValorEstimado != 90000 {
		t.Fatalf("QueryTopSolutionsLicitacoes inesperado: err=%v res=%+v", err, lic)
	}
}
