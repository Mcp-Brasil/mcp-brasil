package transparencia_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mcp-Brasil/mcp-brasil/pkg/transparencia"
)

func TestTCEPRConnector(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("sgl_uf") != "PR" || q.Get("cod_municipio") != "4106902" {
			t.Errorf("query params inesperados: %s", r.URL.RawQuery)
		}
		switch r.URL.Path {
		case "/consulta/despesa":
			if q.Get("num_ano") == "2024" {
				w.Write([]byte(`{
					"dat_consulta": "2026-11-13T05:48:12",
					"ente": [{
						"sgl_uf": "PR",
						"cod_municipio": 410690,
						"num_ano": 2024,
						"num_periodo": 1,
						"despesa": [
							{
								"cod_fonte": 500,
								"des_elemento_despesa": "30000000",
								"vlr_despesa_empenhada": 148834482.15,
								"vlr_despesa_liquidada": 119081554.31,
								"vlr_despesa_paga": 116457250.88
							},
							{
								"cod_fonte": 500,
								"des_elemento_despesa": "31000000",
								"vlr_despesa_empenhada": 112022127.28,
								"vlr_despesa_liquidada": 111745440.92,
								"vlr_despesa_paga": 110569145.20
							}
						]
					}]
				}`))
				return
			}
			w.Write([]byte(`{"dat_consulta":"2026-11-13","ente":[]}`))
		case "/consulta/receita":
			if q.Get("num_ano") == "2024" {
				w.Write([]byte(`{
					"dat_consulta": "2026-11-13T05:48:18",
					"ente": [{
						"sgl_uf": "PR",
						"cod_municipio": 410690,
						"num_ano": 2024,
						"num_periodo": 1,
						"receita": [
							{
								"cod_conta": "10000000",
								"vlr_receita_realizada": 1984037061.37
							},
							{
								"cod_conta": "11000000",
								"vlr_receita_realizada": 687580989.15
							}
						]
					}]
				}`))
				return
			}
			w.Write([]byte(`{"dat_consulta":"2026-11-13","ente":[]}`))
		case "/consulta/funcao12":
			w.Write([]byte(`{
				"dat_consulta": "2026-11-13T05:48:31",
				"ente": [{
					"sgl_uf": "PR",
					"cod_municipio": 410690,
					"num_ano": 2024,
					"num_periodo": 1,
					"valores": [
						{
							"cod_subfuncao": 361,
							"vlr_despesa_empenhada": 180726234.56,
							"vlr_despesa_liquidada": 156297135.28,
							"vlr_despesa_paga": 154610499.15
						}
					]
				}]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	tce := transparencia.NewTCEClient(srv.Client())
	tce.SetTCEPRBaseURL(srv.URL)
	ctx := context.Background()
	m := &transparencia.Municipio{CodigoIBGE: "4106902", Nome: "Curitiba", UF: "PR"}

	desp, err := tce.FetchTCEPRDespesas(ctx, m, 2024, 1, 10)
	if err != nil {
		t.Fatalf("FetchTCEPRDespesas: %v", err)
	}
	if !desp.Disponivel || desp.Plataforma != "tce_pr_siope" || desp.TotalRegistros != 2 || len(desp.Despesas) != 2 {
		t.Fatalf("resultado despesas TCE-PR inesperado: %+v", desp)
	}
	if desp.Despesas[0].Elemento != "30000000" || desp.Despesas[0].Unidade != "Fonte 500" || desp.Despesas[0].ValorPago != 116457250.88 {
		t.Fatalf("item despesa TCE-PR inesperado: %+v", desp.Despesas[0])
	}

	rec, err := tce.FetchTCEPRReceitas(ctx, m, 2024, 1, 10)
	if err != nil {
		t.Fatalf("FetchTCEPRReceitas: %v", err)
	}
	if !rec.Disponivel || rec.TotalRegistros != 2 || len(rec.Receitas) != 2 {
		t.Fatalf("resultado receitas TCE-PR inesperado: %+v", rec)
	}
	if rec.Receitas[0].Codigo != "10000000" || rec.Receitas[0].ValorArrecadado != 1984037061.37 {
		t.Fatalf("item receita TCE-PR inesperado: %+v", rec.Receitas[0])
	}

	f12, err := tce.FetchTCEPRFuncao12Despesas(ctx, m, 2024, 1, 10)
	if err != nil {
		t.Fatalf("FetchTCEPRFuncao12Despesas: %v", err)
	}
	if len(f12.Despesas) != 1 || f12.Despesas[0].Funcao != "Educação (Subfunção 361)" {
		t.Fatalf("resultado funcao12 TCE-PR inesperado: %+v", f12.Despesas)
	}

	if _, err := tce.FetchTCEPRDespesas(ctx, &transparencia.Municipio{Nome: "Sem IBGE"}, 2024, 1, 10); err == nil {
		t.Fatal("esperava erro quando CodigoIBGE está vazio")
	}
}

func TestTCEMSConnector(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/Transparencia/GestaoPessoas/TRUMembros":
			if r.URL.Query().Get("Ano") != "2024" {
				w.Write([]byte(`[]`))
				return
			}
			w.Write([]byte(`[
				{
					"remuneracaoId": 1001,
					"matricula": 4521,
					"nome": "MARIA SILVA",
					"cpf": "***.123.456-**",
					"cargoDescr": "CONSELHEIRO",
					"lotacaoDescr": "GABINETE",
					"mesAno": "01/2024",
					"dataAdmissao": "2015-03-01T00:00:00",
					"tru1001": 35000.0,
					"tru1009": 2000.0,
					"tru1010": 0.0,
					"tru1014": 0.0,
					"tru2001": 5000.0,
					"tru2007": 2000.0,
					"tru2010": 0.0,
					"tru9002": 37000.0,
					"tru9003": 30000.0
				}
			]`))
		case "/api/Transparencia/Repasse/Duodecimos":
			if r.URL.Query().Get("UnidadeGestoraId") != "30101" {
				t.Errorf("UnidadeGestoraId esperado 30101, obteve %q", r.URL.Query().Get("UnidadeGestoraId"))
			}
			w.Write([]byte(`{
				"itens": [
					{
						"unidadeGestora": "30101 - TRIBUNAL DE CONTAS",
						"ano": 2024,
						"mes": 1,
						"previsto": 32741916.25,
						"recebido": 29443025.03
					},
					{
						"unidadeGestora": "30101 - TRIBUNAL DE CONTAS",
						"ano": 2024,
						"mes": 2,
						"previsto": 32741916.25,
						"recebido": 34785743.17
					}
				],
				"total": {
					"previsto": 65483832.50,
					"recebido": 64228768.20
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	tce := transparencia.NewTCEClient(srv.Client())
	tce.SetTCEMSBaseURL(srv.URL)
	ctx := context.Background()
	m := &transparencia.Municipio{CodigoIBGE: "5002704", Nome: "Campo Grande", UF: "MS"}

	serv, err := tce.FetchTCEMSServidores(ctx, m, 2024, 1, "MARIA", 10)
	if err != nil {
		t.Fatalf("FetchTCEMSServidores: %v", err)
	}
	if !serv.Disponivel || serv.Plataforma != "tce_ms_barramento" || len(serv.Servidores) != 1 {
		t.Fatalf("servidores TCE-MS inesperados: %+v", serv)
	}
	if serv.Servidores[0].Nome != "MARIA SILVA" || serv.Servidores[0].SalarioBruto != 37000.0 || serv.Servidores[0].Descontos != 7000.0 || serv.Servidores[0].SalarioLiquido != 30000.0 {
		t.Fatalf("item servidor TCE-MS inesperado: %+v", serv.Servidores[0])
	}

	duod, err := tce.FetchTCEMSDuodecimos(ctx, m, 0, 2024, 0, 10)
	if err != nil {
		t.Fatalf("FetchTCEMSDuodecimos: %v", err)
	}
	if !duod.Disponivel || duod.TotalRegistros != 2 || len(duod.Receitas) != 2 {
		t.Fatalf("duodecimos TCE-MS inesperados: %+v", duod)
	}
	if duod.Receitas[0].Descricao != "30101 - TRIBUNAL DE CONTAS" || duod.Receitas[0].Data != "01/2024" || duod.Receitas[0].ValorArrecadado != 29443025.03 {
		t.Fatalf("item duodecimo TCE-MS inesperado: %+v", duod.Receitas[0])
	}
}

func TestTCESEConnector(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/Despesas/Empenhos":
			w.Write([]byte(`[
				{
					"Exercicio": "2024",
					"UgCodigo": "021011",
					"Ug": "021011 - TCE",
					"Empenho": "1",
					"DataEmpenho": "2024-01-02T00:00:00",
					"FuncionalProgramatica": "01.032.0038.0148",
					"NaturezaDespesaElemento": "319011",
					"NaturezaDespesaSubelemento": "31901199",
					"FonteRecurso": "1500000000",
					"ModalidadeLicitacao": "DISPENSADO",
					"Credor": "021011 00001 - TCE",
					"ValorEmpenhado": 84940261.76,
					"ValorLiquidado": 84940261.76,
					"ValorPago": 84940261.76
				}
			]`))
		case "/api/Receitas/Arrecadadas":
			w.Write([]byte(`[
				{
					"Exercicio": 2024,
					"NaturezaReceita": "13110111 - Aluguéis e Arrendamentos - Principal",
					"FonteRecurso": "1899 - Outros Recursos Vinculados",
					"ValorPrevisto": 8000.00,
					"ValorArrecadado": 600.00
				}
			]`))
		case "/api/Contratos/ContratosVigentes":
			w.Write([]byte(`[
				{
					"IdContrato": 1452,
					"Fornecedor": "ENERGISA SERGIPE DISTRIBUIDORA DE ENERGIA S.A",
					"NrProcesso": "5",
					"AnoProcesso": "2016",
					"DataInicio": "2016-05-02T00:00:00",
					"DataFim": "2030-05-01T00:00:00",
					"SetorResponsavel": "COORDENADORIA DE SERVIÇOS GERAIS",
					"ValorGlobal": 843739.92,
					"Objeto": "SERVIÇO DE FORNECIMENTO DE ENERGIA ELÉTRICA",
					"TipoProcesso": "Inexigibilidade",
					"QtdAditivos": 5
				}
			]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	tce := transparencia.NewTCEClient(srv.Client())
	tce.SetTCESEBaseURL(srv.URL)
	ctx := context.Background()
	m := &transparencia.Municipio{CodigoIBGE: "2800308", Nome: "Aracaju", UF: "SE"}

	desp, err := tce.FetchTCESEDespesas(ctx, m, 2024, "", 10)
	if err != nil {
		t.Fatalf("FetchTCESEDespesas: %v", err)
	}
	if len(desp.Despesas) != 1 || desp.Despesas[0].NumeroEmpenho != "1" || desp.Despesas[0].Elemento != "31901199" {
		t.Fatalf("despesas TCE-SE inesperadas: %+v", desp.Despesas)
	}

	rec, err := tce.FetchTCESEReceitas(ctx, m, 2024, "", 10)
	if err != nil {
		t.Fatalf("FetchTCESEReceitas: %v", err)
	}
	if len(rec.Receitas) != 1 || rec.Receitas[0].ValorPrevisto != 8000.0 || rec.Receitas[0].ValorArrecadado != 600.0 {
		t.Fatalf("receitas TCE-SE inesperadas: %+v", rec.Receitas)
	}

	ct, err := tce.FetchTCESEContratos(ctx, m, 10)
	if err != nil {
		t.Fatalf("FetchTCESEContratos: %v", err)
	}
	if len(ct.Contratos) != 1 || ct.Contratos[0].Numero != "5/2016" || ct.Contratos[0].Valor != 843739.92 {
		t.Fatalf("contratos TCE-SE inesperados: %+v", ct.Contratos)
	}
}
