package transparencia_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Mcp-Brasil/mcp-brasil/pkg/transparencia"
)

func TestExtractTomeContaID(t *testing.T) {
	if id := transparencia.ExtractTomeContaID("https://portaltransparencia.app.br/carregaPortalPM.aspx?p_i=43&p_t=0"); id != "43" {
		t.Errorf("ExtractTomeContaID URL PM = %q, esperava '43'", id)
	}
	if id := transparencia.ExtractTomeContaID("carregaGrupoEntidade.aspx?p_i=128&p_t=1"); id != "128" {
		t.Errorf("ExtractTomeContaID Location = %q, esperava '128'", id)
	}
	html := `<iframe src="https://portaltransparencia.app.br/carregaPortalPM.aspx?p_i=77&amp;p_t=0"></iframe>`
	if id := transparencia.ExtractTomeContaID(html); id != "77" {
		t.Errorf("ExtractTomeContaID HTML = %q, esperava '77'", id)
	}
	if id := transparencia.ExtractTomeContaID("https://prefeitura.exemplo.gov.br/transparencia"); id != "" {
		t.Errorf("esperava vazio quando p_i ausente, obteve %q", id)
	}
}

func TestProbeTomeConta(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/router" {
			http.Redirect(w, r, "/carregaGrupoEntidade.aspx?p_i=99&p_t=0", http.StatusFound)
			return
		}
		if r.URL.Path == "/api/apiDados.ashx" {
			w.Header().Set("Content-Type", "application/json")
			q := r.URL.Query()
			if q.Get("metodo") != "retornaDespesa" || q.Get("mes") != "1" {
				http.Error(w, "bad params", http.StatusBadRequest)
				return
			}
			switch q.Get("id") {
			case "43", "99":
				_, _ = w.Write([]byte(`{"status":"ok","total":1,"dados":[]}`))
			default:
				_, _ = w.Write([]byte(`{"status":"erro","mensagem":"entidade não encontrada"}`))
			}
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	// Probe direto com tenantID
	if !transparencia.ProbeTomeConta(ctx, client, srv.URL, "43") {
		t.Fatalf("esperava ProbeTomeConta=true para tenant 43")
	}

	// Probe extraindo p_i da URL com SetTomeContaBaseURL
	transparencia.SetTomeContaBaseURL(srv.URL)
	defer transparencia.SetTomeContaBaseURL("")
	if !transparencia.ProbeTomeConta(ctx, client, "https://portaltransparencia.app.br/carregaPortalPM.aspx?p_i=43&p_t=0", "") {
		t.Fatalf("esperava ProbeTomeConta=true extraindo p_i=43 da URL")
	}

	// Probe seguindo redirect Location carregaGrupoEntidade.aspx?p_i=99
	if !transparencia.ProbeTomeConta(ctx, client, srv.URL+"/router", "") {
		t.Fatalf("esperava ProbeTomeConta=true extraindo p_i=99 do redirect Location")
	}

	// Negativo
	if transparencia.ProbeTomeConta(ctx, client, srv.URL, "9999") {
		t.Fatalf("esperava ProbeTomeConta=false para status=erro")
	}
}

func TestTomeContaAllQueriesMock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "2601102", Nome: "Araripina", UF: "PE"}

	var requireMes atomic.Bool
	requireMes.Store(true)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/apiDados.ashx" {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		if q.Get("id") != "43" {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		switch q.Get("metodo") {
		case "retornaDespesa":
			mes := q.Get("mes")
			if requireMes.Load() && mes == "" {
				_, _ = w.Write([]byte(`{"status":"erro","mensagem":"mes obrigatorio"}`))
				return
			}
			if mes != "" && mes != "1" && mes != "2" {
				_, _ = w.Write([]byte(`{"status":"ok","total":0,"dados":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{
				"status": "ok",
				"total": 2,
				"dados": [
					{
						"id": 5001,
						"FORNECEDOR": "DISTRIBUIDORA NORDESTE LTDA",
						"CPF_CNPJ": "11.222.333/0001-44",
						"DATA": "/Date(1735786800000)/",
						"EMPENHO": 1,
						"PARCELA": 1,
						"ORGAO": "Fundo Municipal de Saúde",
						"HISTORICO": "Aquisição de insumos hospitalares",
						"VALOREMPENHO": "7117,90",
						"VALORPAGO": "85,42"
					},
					{
						"id": 5002,
						"FORNECEDOR": "JOSE CARLOS",
						"CPF_CNPJ": "12345678901",
						"DATA": "/Date(1735786800000-0300)/",
						"EMPENHO": 2,
						"VALOREMPENHO": "350,00",
						"VALORPAGO": "350,00"
					}
				]
			}`))
		case "retornaReceita":
			_, _ = w.Write([]byte(`{
				"status": "ok",
				"total": 1,
				"dados": [
					{
						"CODIGO": "1.7.1.8.01",
						"DESCRICAO": "Cota-Parte do FPM",
						"CATEGORIA": "Transferências Correntes",
						"DATA": "/Date(1735786800000)/",
						"VALOR_PREVISTO": "1200000,00",
						"VALOR_ARRECADADO": "450123,45"
					}
				]
			}`))
		case "Licitacoes":
			_, _ = w.Write([]byte(`{
				"status": "ok",
				"total": 1,
				"dados": [
					{
						"ID": 88,
						"data": "/Date(1735786800000)/",
						"TITULO": "Pregão Eletrônico 004/2025",
						"DESCRICAO": "Contratação de empresa para pavimentação",
						"MODALIDADE": "Pregão Eletrônico",
						"PROCESSO": "004/2025",
						"ANO": 2025,
						"VALOR_ESTIMADO": "480000,00",
						"VALOR_ADJUDICADO": "455250,50",
						"STATUS": "Adjudicada",
						"DATA_ABERTURA": "/Date(1735786800000)/",
						"ANEXO": "https://portaltransparencia.app.br/anexos/edital_004.pdf"
					}
				]
			}`))
		case "retornaContratos":
			_, _ = w.Write([]byte(`{
				"status": "ok",
				"total": 2,
				"dados": [
					{
						"NUM_CONTRATO": "CT-2025/015",
						"DATA": "/Date(1735786800000)/",
						"FORNECEDOR": "MARIA ANTONIA",
						"DESCRICAO": "Serviços de consultoria contábil",
						"CPF_CNPJ": "98765432100",
						"VALOR_CONTRATO": "48000,00",
						"VIGENCIAI": "/Date(1735786800000)/",
						"VIGENCIAF": "/Date(1767139200000)/"
					}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	p := transparencia.PortalQueryParams{
		BaseURL: srv.URL + "/carregaPortalPM.aspx?p_i=43&p_t=0",
		Ano:     2025,
		Limit:   10,
	}

	// 1a. Despesas com fallback mes 1..12 quando servidor exige mes
	desp, err := transparencia.QueryTomeContaDespesas(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryTomeContaDespesas (fallback mes) erro: %v", err)
	}
	if desp.Plataforma != "tomeconta_appbr" || desp.TotalRegistros != 2 || len(desp.Despesas) != 2 {
		t.Fatalf("DespesasResult inesperado: %+v", desp)
	}
	d0 := desp.Despesas[0]
	if d0.NumeroEmpenho != "1" || d0.Data != "2025-01-02" || d0.ValorEmpenhado != 7117.90 || d0.ValorPago != 85.42 || d0.DocumentoFornecedor != "11.222.333/0001-44" {
		t.Errorf("Despesa[0] inesperada: %+v", d0)
	}
	d1 := desp.Despesas[1]
	if d1.Data != "2025-01-02" || d1.DocumentoFornecedor != "***.456.789-**" {
		t.Errorf("Despesa[1] inesperada: %+v", d1)
	}

	// 1b. Despesas quando servidor aceita omitir mes
	requireMes.Store(false)
	despNoMes, err := transparencia.QueryTomeContaDespesas(ctx, client, m, p)
	if err != nil || len(despNoMes.Despesas) != 2 {
		t.Fatalf("QueryTomeContaDespesas (sem mes) erro=%v len=%d", err, len(despNoMes.Despesas))
	}

	// 2. Receitas
	rec, err := transparencia.QueryTomeContaReceitas(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryTomeContaReceitas erro: %v", err)
	}
	if len(rec.Receitas) != 1 || rec.Receitas[0].Codigo != "1.7.1.8.01" || rec.Receitas[0].ValorPrevisto != 1200000.00 || rec.Receitas[0].ValorArrecadado != 450123.45 || rec.Receitas[0].Data != "2025-01-02" {
		t.Fatalf("ReceitasResult inesperado: %+v", rec)
	}

	// 3. Licitações
	lic, err := transparencia.QueryTomeContaLicitacoes(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryTomeContaLicitacoes erro: %v", err)
	}
	if len(lic.Licitacoes) != 1 {
		t.Fatalf("LicitacoesResult inesperado: %+v", lic)
	}
	l0 := lic.Licitacoes[0]
	if l0.Numero != "004/2025" || l0.Modalidade != "Pregão Eletrônico" || l0.Situacao != "Adjudicada" || l0.DataAbertura != "2025-01-02" || l0.ValorEstimado != 480000.00 || l0.ValorHomologado != 455250.50 || l0.Link == "" {
		t.Errorf("Licitacao[0] inesperada: %+v", l0)
	}

	// 4. Contratos
	ct, err := transparencia.QueryTomeContaContratos(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryTomeContaContratos erro: %v", err)
	}
	if len(ct.Contratos) != 1 {
		t.Fatalf("ContratosResult inesperado: %+v", ct)
	}
	c0 := ct.Contratos[0]
	if c0.Numero != "CT-2025/015" || c0.DocumentoFornecedor != "***.654.321-**" || c0.Valor != 48000.00 || c0.DataAssinatura != "2025-01-02" || c0.VigenciaInicio != "2025-01-02" || c0.VigenciaFim != "2025-12-31" {
		t.Errorf("Contrato[0] inesperado: %+v", c0)
	}
}
