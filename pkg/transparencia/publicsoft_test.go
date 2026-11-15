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

const testPublicsoftHash = "0123456789abcdef0123456789abcdef01234567"
const testPublicsoftHashCamara = "fedcba9876543210fedcba9876543210fedcba98"

func TestPublicsoftCatalogHelpers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Stage") != "prod" {
			http.Error(w, "missing X-Stage: prod", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/catalogo/ugs":
			if r.URL.Query().Get("com_instituicao") != "1" {
				http.Error(w, "missing com_instituicao=1", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{
				"status": "success",
				"data": [
					{
						"id": 10,
						"nome": "Câmara Municipal de Cabedelo",
						"hash_id": "` + testPublicsoftHashCamara + `",
						"municipio": "Cabedelo",
						"uf": "PB"
					},
					{
						"id": 11,
						"nome": "Prefeitura Municipal de Cabedelo",
						"hash_id": "` + testPublicsoftHash + `",
						"instituicao": {
							"nome": "Prefeitura Municipal de Cabedelo",
							"municipio": {"nome": "Cabedelo", "estado": {"sigla": "PB"}}
						}
					}
				]
			}`))
		case "/api/v1/catalogo/estados":
			_, _ = w.Write([]byte(`{
				"status": "success",
				"data": [
					{"id": 15, "uf": "PB", "nome": "Paraíba"},
					{"id": 16, "sigla": "PE", "nome": "Pernambuco"}
				]
			}`))
		case "/api/v1/catalogo/municipios":
			if r.URL.Query().Get("estado") != "PB" {
				http.Error(w, "unexpected estado", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{
				"status": "success",
				"data": [
					{"id": 101, "nome": "Cabedelo", "uf": "PB", "codigo_ibge": "2503209"},
					{"id": 102, "nome": "João Pessoa", "codigo_ibge": "2507507"}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	transparencia.SetPublicsoftBaseURL(srv.URL)
	defer transparencia.SetPublicsoftBaseURL("")

	ugs, err := transparencia.FetchPublicsoftUGs(ctx, client)
	if err != nil {
		t.Fatalf("FetchPublicsoftUGs erro: %v", err)
	}
	if len(ugs) != 2 || ugs[1].HashID != testPublicsoftHash || ugs[1].Municipio != "Cabedelo" || ugs[1].UF != "PB" {
		t.Fatalf("UGs inesperadas: %+v", ugs)
	}

	estados, err := transparencia.FetchPublicsoftEstados(ctx, client)
	if err != nil {
		t.Fatalf("FetchPublicsoftEstados erro: %v", err)
	}
	if len(estados) != 2 || estados[0].UF != "PB" || estados[1].UF != "PE" {
		t.Fatalf("estados inesperados: %+v", estados)
	}

	muns, err := transparencia.FetchPublicsoftMunicipios(ctx, client, "pb")
	if err != nil {
		t.Fatalf("FetchPublicsoftMunicipios erro: %v", err)
	}
	if len(muns) != 2 || muns[0].CodigoIBGE != "2503209" || muns[1].UF != "PB" {
		t.Fatalf("municipios inesperados: %+v", muns)
	}
}

func TestResolvePublicsoftLegadoAndProbe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/portal/legado/pmcabedelo/despesa-orcamentaria":
			_, _ = w.Write([]byte(`{
				"status": "success",
				"data": {
					"hash_id": "` + testPublicsoftHash + `",
					"ug": {"id": 11, "nome": "Prefeitura Municipal de Cabedelo", "hash_id": "` + testPublicsoftHash + `"}
				}
			}`))
		case "/api/v1/despesas/detalhe":
			if r.Header.Get("X-Stage") == "prod" && r.Header.Get("X-Hash-Id") == testPublicsoftHash {
				_, _ = w.Write([]byte(`{"status":"success","data":{"data":[],"total":0}}`))
				return
			}
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		case "/api/v1/catalogo/ugs":
			_, _ = w.Write([]byte(`{
				"status": "success",
				"data": [
					{"id": 10, "nome": "Câmara Municipal de Cabedelo", "hash_id": "` + testPublicsoftHashCamara + `", "municipio": "Cabedelo", "uf": "PB"},
					{"id": 11, "nome": "Prefeitura Municipal de Cabedelo", "hash_id": "` + testPublicsoftHash + `", "municipio": "Cabedelo", "uf": "PB"}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	transparencia.SetPublicsoftBaseURL(srv.URL)
	defer transparencia.SetPublicsoftBaseURL("")

	// Legado via URL completa
	legURL := "https://portaldatransparencia.publicsoft.com.br/sistemas/ContabilidadePublica/pmcabedelo/views/despesa.php"
	hash, err := transparencia.ResolvePublicsoftLegado(ctx, client, legURL)
	if err != nil || hash != testPublicsoftHash {
		t.Fatalf("ResolvePublicsoftLegado URL = (%q, %v), esperava %q", hash, err, testPublicsoftHash)
	}

	// Legado via slug direto
	hash2, err := transparencia.ResolvePublicsoftLegado(ctx, client, "pmcabedelo")
	if err != nil || hash2 != testPublicsoftHash {
		t.Fatalf("ResolvePublicsoftLegado slug = (%q, %v), esperava %q", hash2, err, testPublicsoftHash)
	}

	// Probe direto por hash_id
	h1, _, ok1 := transparencia.ProbePublicsoft(ctx, client, testPublicsoftHash, "", "")
	if !ok1 || h1 != testPublicsoftHash {
		t.Fatalf("ProbePublicsoft hash direto = (%q, %v)", h1, ok1)
	}

	// Probe via legado URL
	h2, ent2, ok2 := transparencia.ProbePublicsoft(ctx, client, legURL, "", "")
	if !ok2 || h2 != testPublicsoftHash || ent2 != "Prefeitura Municipal de Cabedelo" {
		t.Fatalf("ProbePublicsoft legado = (%q, %q, %v)", h2, ent2, ok2)
	}

	// Probe via catálogo (deve priorizar Prefeitura sobre Câmara)
	h3, ent3, ok3 := transparencia.ProbePublicsoft(ctx, client, "", "PB", "Cabedelo")
	if !ok3 || h3 != testPublicsoftHash || ent3 != "Prefeitura Municipal de Cabedelo" {
		t.Fatalf("ProbePublicsoft catálogo = (%q, %q, %v)", h3, ent3, ok3)
	}

	// Probe negativo
	if hNeg, _, okNeg := transparencia.ProbePublicsoft(ctx, client, "", "PB", "Inexistente"); okNeg || hNeg != "" {
		t.Fatalf("esperava ok=false para município inexistente, obteve (%q, %v)", hNeg, okNeg)
	}
}

func TestPublicsoftAllQueriesMock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "2503209", Nome: "Cabedelo", UF: "PB"}

	var folhaModo atomic.Value
	folhaModo.Store("nao_integrado")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Stage") != "prod" || r.Header.Get("X-Hash-Id") != testPublicsoftHash {
			http.Error(w, "missing tenant headers", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/despesas/detalhe":
			if r.URL.Query().Get("exercicio") != "2025" || r.URL.Query().Get("per_page") != "10" {
				http.Error(w, "unexpected query params", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{
				"status": "success",
				"data": {
					"current_page": 1,
					"total": 45,
					"data": [
						{
							"id": 901,
							"numero": "00123/2025",
							"data": "2025-02-10",
							"descricao": "Aquisição de medicamentos",
							"credor": "FARMACIA CENTRAL LTDA",
							"cpfCnpj": "12.345.678/0001-99",
							"orgao": "Secretaria de Saúde",
							"funcao": "10 - Saúde",
							"elemento": "3.3.90.30 - Material de Consumo",
							"historico": "Compra de antibióticos",
							"valorEmpenhado": 25000.50,
							"valorLiquidado": 20000.00,
							"valorPago": 15000.00
						},
						{
							"id": 902,
							"data": "2025-02-11",
							"descricao": "Diária de viagem",
							"credor": "JOAO DA SILVA",
							"cpfCnpj": "12345678901",
							"valorEmpenhado": "500,00",
							"valorLiquidado": "500,00",
							"valorPago": "500,00"
						}
					]
				}
			}`))
		case "/api/v1/receitas/detalhe":
			_, _ = w.Write([]byte(`{
				"status": "success",
				"data": {
					"total": 12,
					"data": [
						{
							"rubrica": "1.1.1.3.03.1.1",
							"cod_receita_stn": "11130311",
							"descricao": "ISSQN Principal",
							"categoria": "Receitas Correntes",
							"fonte": "Recursos Próprios",
							"valor_previsto": 1500000.00,
							"arrecadado": 820450.75,
							"data": "2025-02-28"
						}
					]
				}
			}`))
		case "/api/v1/contratos":
			_, _ = w.Write([]byte(`{
				"status": "success",
				"data": {
					"total": 8,
					"data": [
						{
							"numero": "CT-042/2025",
							"objetivo": "Locação de veículos leves",
							"fornecedor": "CARLOS ALBERTO",
							"cpfCnpj": "98765432100",
							"situacao": "Vigente",
							"dataInicial": "2025-01-05",
							"dataFinal": "2025-12-31",
							"dataPublicacao": "2025-01-04",
							"valor": 96000.00
						}
					]
				}
			}`))
		case "/api/v1/licitacoes":
			_, _ = w.Write([]byte(`{
				"status": "success",
				"data": {
					"total": 5,
					"data": [
						{
							"numero": "PE-009/2025",
							"modalidade": "Pregão Eletrônico",
							"situacao": "Homologada",
							"dataAbertura": "2025-01-20",
							"descObjeto": "Aquisição de equipamentos de informática",
							"valorEstimado": 320000.00,
							"valorHomologado": 298500.00
						}
					]
				}
			}`))
		case "/api/v1/catalogo/ugs/" + testPublicsoftHash + "/folha-config":
			modo := folhaModo.Load().(string)
			if modo == "nao_integrado" {
				_, _ = w.Write([]byte(`{
					"status": "success",
					"data": {"modo": "nao_integrado", "url_externa": "https://folha.cabedelo.pb.gov.br"}
				}`))
				return
			}
			_, _ = w.Write([]byte(`{"status":"success","data":{"modo":"integrado"}}`))
		case "/api/v1/servidores":
			_, _ = w.Write([]byte(`{
				"status": "success",
				"data": {
					"total": 2,
					"data": [
						{
							"nome": "MARIA EDUARDA",
							"matricula": "1045",
							"cpf": "11122233344",
							"cargo": "Enfermeira",
							"lotacao": "Hospital Municipal",
							"salario_base": 4200.00,
							"salario_bruto": 5100.00,
							"descontos": 900.00,
							"salario_liquido": 4200.00,
							"competencia": "02/2025"
						}
					]
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	p := transparencia.PortalQueryParams{
		BaseURL:  srv.URL,
		TenantID: testPublicsoftHash,
		Ano:      2025,
		Limit:    10,
	}

	// 1. Despesas
	desp, err := transparencia.QueryPublicsoftDespesas(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryPublicsoftDespesas erro: %v", err)
	}
	if desp.Plataforma != "publicsoft_api" || desp.TotalRegistros != 45 || len(desp.Despesas) != 2 {
		t.Fatalf("DespesasResult inesperado: %+v", desp)
	}
	if desp.Despesas[0].NumeroEmpenho != "00123/2025" || desp.Despesas[0].DocumentoFornecedor != "12.345.678/0001-99" || desp.Despesas[0].ValorEmpenhado != 25000.50 {
		t.Errorf("Despesas[0] inesperado: %+v", desp.Despesas[0])
	}
	if desp.Despesas[1].NumeroEmpenho != "902" || desp.Despesas[1].DocumentoFornecedor != "***.456.789-**" || desp.Despesas[1].Historico != "Diária de viagem" {
		t.Errorf("Despesas[1] inesperado: %+v", desp.Despesas[1])
	}

	// 2. Receitas
	rec, err := transparencia.QueryPublicsoftReceitas(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryPublicsoftReceitas erro: %v", err)
	}
	if rec.TotalRegistros != 12 || len(rec.Receitas) != 1 || rec.Receitas[0].Codigo != "1.1.1.3.03.1.1" || rec.Receitas[0].ValorArrecadado != 820450.75 {
		t.Fatalf("ReceitasResult inesperado: %+v", rec)
	}

	// 3. Contratos
	ct, err := transparencia.QueryPublicsoftContratos(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryPublicsoftContratos erro: %v", err)
	}
	if ct.TotalRegistros != 8 || len(ct.Contratos) != 1 {
		t.Fatalf("ContratosResult inesperado: %+v", ct)
	}
	if ct.Contratos[0].DocumentoFornecedor != "***.654.321-**" || ct.Contratos[0].Objeto != "Locação de veículos leves" || ct.Contratos[0].VigenciaInicio != "2025-01-05" || ct.Contratos[0].VigenciaFim != "2025-12-31" || ct.Contratos[0].DataAssinatura != "2025-01-04" {
		t.Errorf("Contrato[0] inesperado: %+v", ct.Contratos[0])
	}

	// 4. Licitações
	lic, err := transparencia.QueryPublicsoftLicitacoes(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryPublicsoftLicitacoes erro: %v", err)
	}
	if lic.TotalRegistros != 5 || len(lic.Licitacoes) != 1 || lic.Licitacoes[0].Objeto != "Aquisição de equipamentos de informática" || lic.Licitacoes[0].ValorEstimado != 320000.00 {
		t.Fatalf("LicitacoesResult inesperado: %+v", lic)
	}

	// 5a. Servidores (folha-config nao_integrado)
	servNaoInt, err := transparencia.QueryPublicsoftServidores(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryPublicsoftServidores (nao_integrado) erro: %v", err)
	}
	if servNaoInt.Disponivel || servNaoInt.Mensagem == "" {
		t.Fatalf("esperava Disponivel=false com mensagem folha-config, obteve: %+v", servNaoInt)
	}

	// 5b. Servidores (folha-config integrado)
	folhaModo.Store("integrado")
	servInt, err := transparencia.QueryPublicsoftServidores(ctx, client, m, p)
	if err != nil {
		t.Fatalf("QueryPublicsoftServidores (integrado) erro: %v", err)
	}
	if !servInt.Disponivel || len(servInt.Servidores) != 1 || servInt.Servidores[0].CPFParcial != "***.222.333-**" || servInt.Servidores[0].SalarioLiquido != 4200.00 {
		t.Fatalf("ServidoresResult (integrado) inesperado: %+v", servInt)
	}
}
