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

func TestSlugifyHelpersAndStatsPopulation(t *testing.T) {
	if got := transparencia.SlugifyCompact("Serra da Saudade"); got != "serradasaudade" {
		t.Errorf("SlugifyCompact inesperado: %s", got)
	}
	if got := transparencia.SlugifyNoPrepositions("Serra da Saudade"); got != "serrasaudade" {
		t.Errorf("SlugifyNoPrepositions inesperado: %s", got)
	}

	engine, err := transparencia.NewEngine()
	if err != nil {
		t.Fatalf("NewEngine erro: %v", err)
	}
	stats := engine.GetStats()
	if stats.PopulacaoTotal <= 150000000 {
		t.Errorf("PopulacaoTotal inesperadamente baixa: %d", stats.PopulacaoTotal)
	}

	cands := transparencia.CandidatePortalURLs(&transparencia.Municipio{
		Nome: "Serra da Saudade",
		UF:   "MG",
		Slug: "serra-da-saudade",
	})
	hasCompact := false
	for _, c := range cands {
		if c == "https://transparencia.serradasaudade.mg.gov.br" {
			hasCompact = true
		}
	}
	if !hasCompact {
		t.Errorf("CandidatePortalURLs não gerou domínio compacto esperado: %v", cands)
	}
}

func TestDiscoverAndQueryQuasarMock(t *testing.T) {
	var seenTermoB64 string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/publico/versao":
			_, _ = w.Write([]byte(`"v26.10.30"`))
		case "/publico/buscarEntidades":
			_, _ = w.Write([]byte(`[
				{"codAdministracao": 1, "desAdministracao": "CAMARA MUNICIPAL", "numCnpj": "00000000000100"},
				{"codAdministracao": 2, "desAdministracao": "PREFEITURA MUNICIPAL", "numCnpj": "18500000000199"}
			]`))
		case "/publico/servidor":
			seenTermoB64 = r.URL.Query().Get("termoBase64")
			if r.URL.Query().Get("codAdministracao") != "2" {
				http.Error(w, "bad codAdministracao", http.StatusBadRequest)
				return
			}
			// Simula fallback de ano: retorna dados apenas para 2025
			if !strings.HasPrefix(r.URL.Query().Get("datAberturaInicio"), "2025") {
				_, _ = w.Write([]byte(`{"content": [], "totalElements": 0, "totalPages": 0}`))
				return
			}
			_, _ = w.Write([]byte(`{
				"content": [
					{
						"nomServidor": "MARIA TESTE SILVA",
						"numMatricula": 101,
						"cpfformatado": "",
						"numcpf": "12345678901",
						"nomCargo": "PROFESSORA",
						"nomLotacao": "EDUCACAO",
						"desTipoClassificacao": "EFETIVO",
						"vlrSalarioBase": 3000.00,
						"vlrSalarioBruto": 3500.50,
						"vlrDesconto": 400.25,
						"vlrSalarioLiquido": 3100.25,
						"mesReferenciaFormatado": "Nov/2025"
					}
				],
				"totalElements": 42,
				"totalPages": 1
			}`))
		case "/publico/despesaDetalhada":
			_, _ = w.Write([]byte(`{
				"content": [{
					"numEmpenho": "123/2025",
					"numAnoEmpenho": 2025,
					"datMovimento": "2025-11-10",
					"nomFornecedor": "EMPRESA EXEMPLO LTDA",
					"desDocumentoFornecedor": "12345678000199",
					"desEmpenhoHistorico": "FORNECIMENTO DE MEDICAMENTOS",
					"vlrEmpenhado": 10000.0,
					"vlrLiquidado": 5000.0,
					"vlrPago": 5000.0
				}],
				"totalElements": 10
			}`))
		case "/publico/receita":
			_, _ = w.Write([]byte(`{
				"content": [{
					"desClassificacaoReceita": "1.7.1.8",
					"desReceita": "COTA-PARTE DO FPM",
					"desFonteReceita": "RECURSOS ORDINARIOS",
					"datMovimentacao": "2025-11-01",
					"vlrReceita": 500000.0,
					"vlrArrecadado": 480000.0
				}],
				"totalElements": 5
			}`))
		case "/publico/licitacao":
			_, _ = w.Write([]byte(`{
				"content": [{
					"numeroLicitacaoEAno": "001/2025",
					"anoProcesso": 2025,
					"desModalidade": "PREGAO ELETRONICO",
					"indSituacaoFormatado": "HOMOLOGADA",
					"datAbertura": "2025-05-10",
					"txtObjeto": "AQUISICAO DE MATERIAL ESCOLAR",
					"vlrEstimado": 80000.0,
					"vlrHomologado": 75000.0
				}],
				"totalElements": 3
			}`))
		case "/publico/contrato":
			_, _ = w.Write([]byte(`{
				"content": [{
					"numeroContratoEAno": "010/2025",
					"numAnoContrato": 2025,
					"nomFornecedor": "PAPELARIA MODELO",
					"desObjeto": "MATERIAL DE EXPEDIENTE",
					"indSituacao": "VIGENTE",
					"datAssinatura": "2025-06-01",
					"vlrContratado": 75000.0
				}],
				"totalElements": 2
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	m := &transparencia.Municipio{
		CodigoIBGE: "3166600",
		Nome:       "Serra da Saudade",
		UF:         "MG",
		Slug:       "serra-da-saudade",
		CNPJ:       "18500000000199",
	}

	ctx := context.Background()
	disc := transparencia.DiscoverPortal(ctx, m, transparencia.DiscoverOptions{
		OverrideURL: srv.URL,
		HTTPClient:  srv.Client(),
	})
	if disc.Plataforma != "quasar_publico" {
		t.Fatalf("Esperava plataforma quasar_publico, obteve %s", disc.Plataforma)
	}
	if disc.CodAdministracao != 2 {
		t.Errorf("Esperava codAdministracao 2 (Prefeitura), obteve %d", disc.CodAdministracao)
	}
	if !disc.ServidoresSuportado {
		t.Errorf("ServidoresSuportado deveria ser true")
	}

	servRes, err := transparencia.QueryQuasarServidores(ctx, srv.Client(), m, transparencia.PortalQueryParams{
		BaseURL:          disc.URLResolvida,
		CodAdministracao: disc.CodAdministracao,
		NomeFiltro:       "MARIA",
		Limit:            10,
	})
	if err != nil {
		t.Fatalf("QueryQuasarServidores erro: %v", err)
	}
	expectedB64 := base64.StdEncoding.EncodeToString([]byte("MARIA"))
	if seenTermoB64 != expectedB64 {
		t.Errorf("termoBase64 incorreto: esperava %s, obteve %s", expectedB64, seenTermoB64)
	}
	if servRes.TotalRegistros != 42 || len(servRes.Servidores) != 1 {
		t.Fatalf("Servidores inesperados: %+v", servRes)
	}
	// Verifica LGPD: CPF bruto não pode vazar
	if servRes.Servidores[0].CPFParcial != "***.456.789-**" {
		t.Errorf("CPF não foi mascarado corretamente conforme LGPD: %q", servRes.Servidores[0].CPFParcial)
	}
	if strings.Contains(servRes.Servidores[0].CPFParcial, "12345678901") {
		t.Fatalf("CPF bruto vazou no resultado!")
	}

	desp, err := transparencia.QueryQuasarDespesas(ctx, srv.Client(), m, transparencia.PortalQueryParams{BaseURL: srv.URL, CodAdministracao: 2, Ano: 2025})
	if err != nil || len(desp.Despesas) != 1 || desp.Despesas[0].ValorPago != 5000.0 {
		t.Fatalf("QueryQuasarDespesas falhou: %+v, err=%v", desp, err)
	}

	rec, err := transparencia.QueryQuasarReceitas(ctx, srv.Client(), m, transparencia.PortalQueryParams{BaseURL: srv.URL, CodAdministracao: 2, Ano: 2025})
	if err != nil || len(rec.Receitas) != 1 || rec.Receitas[0].ValorArrecadado != 480000.0 {
		t.Fatalf("QueryQuasarReceitas falhou: %+v, err=%v", rec, err)
	}

	lic, err := transparencia.QueryQuasarLicitacoes(ctx, srv.Client(), m, transparencia.PortalQueryParams{BaseURL: srv.URL, CodAdministracao: 2, Ano: 2025})
	if err != nil || len(lic.Licitacoes) != 1 || lic.Licitacoes[0].ValorHomologado != 75000.0 {
		t.Fatalf("QueryQuasarLicitacoes falhou: %+v, err=%v", lic, err)
	}

	cont, err := transparencia.QueryQuasarContratos(ctx, srv.Client(), m, transparencia.PortalQueryParams{BaseURL: srv.URL, CodAdministracao: 2, Ano: 2025})
	if err != nil || len(cont.Contratos) != 1 || cont.Contratos[0].Valor != 75000.0 {
		t.Fatalf("QueryQuasarContratos falhou: %+v, err=%v", cont, err)
	}
}

func TestDiscoverFiorilliAndAntiBotAndQuery(t *testing.T) {
	fiorilliSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versaoJson/DadosAbertos/":
			_, _ = w.Write([]byte(`{"Sistema":"Fiorilli SCPI","Endpoints":["DespesasGerais","Pessoal"]}`))
		case "/VersaoJson/Pessoal/":
			_, _ = w.Write([]byte(`[
				{"Nome":"JOAO DA SILVA","CPF":"98765432100","Cargo":"MOTORISTA","Lotacao":"OBRAS","SalarioBruto":"2.500,50","Liquido":"2.100,25"}
			]`))
		case "/VersaoJson/DespesasGerais/":
			_, _ = w.Write([]byte(`{"Registros":[{"Empenho":"55","Fornecedor":"POSTO CENTRAL","Empenhado":"1.234,56","Pago":"1.234,56"}]}`))
		case "/VersaoJson/ReceitaOrcamentaria/":
			_, _ = w.Write([]byte(`[{"Codigo":"1.1","Descricao":"IPTU","Previsto":"10.000,00","Arrecadado":"8.500,00"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer fiorilliSrv.Close()

	m := &transparencia.Municipio{CodigoIBGE: "3500105", Nome: "Adamantina", UF: "SP", Slug: "adamantina"}
	ctx := context.Background()

	disc := transparencia.DiscoverPortal(ctx, m, transparencia.DiscoverOptions{OverrideURL: fiorilliSrv.URL})
	if disc.Plataforma != "fiorilli_scpi" {
		t.Fatalf("Esperava fiorilli_scpi, obteve %s", disc.Plataforma)
	}

	serv, err := transparencia.QueryFiorilliServidores(ctx, fiorilliSrv.Client(), m, transparencia.PortalQueryParams{BaseURL: fiorilliSrv.URL, Ano: 2025})
	if err != nil || len(serv.Servidores) != 1 {
		t.Fatalf("QueryFiorilliServidores falhou: %+v err=%v", serv, err)
	}
	if serv.Servidores[0].SalarioBruto != 2500.50 || serv.Servidores[0].CPFParcial != "***.654.321-**" {
		t.Errorf("Parse Fiorilli inesperado: %+v", serv.Servidores[0])
	}

	desp, err := transparencia.QueryFiorilliDespesas(ctx, fiorilliSrv.Client(), m, transparencia.PortalQueryParams{BaseURL: fiorilliSrv.URL, Ano: 2025})
	if err != nil || len(desp.Despesas) != 1 || desp.Despesas[0].ValorEmpenhado != 1234.56 {
		t.Fatalf("QueryFiorilliDespesas falhou: %+v err=%v", desp, err)
	}

	rec, err := transparencia.QueryFiorilliReceitas(ctx, fiorilliSrv.Client(), m, transparencia.PortalQueryParams{BaseURL: fiorilliSrv.URL, Ano: 2025})
	if err != nil || len(rec.Receitas) != 1 || rec.Receitas[0].ValorArrecadado != 8500.00 {
		t.Fatalf("QueryFiorilliReceitas falhou: %+v err=%v", rec, err)
	}

	// Teste detecção anti-bot (Cloudflare Turnstile / HTTP 403)
	wafSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`<html><body>Just a moment... <div class="cf-turnstile"></div></body></html>`))
			return
		}
		http.NotFound(w, r)
	}))
	defer wafSrv.Close()

	discWaf := transparencia.DiscoverPortal(ctx, m, transparencia.DiscoverOptions{OverrideURL: wafSrv.URL})
	if discWaf.Plataforma != "anti_bot_protegido" {
		t.Errorf("Esperava anti_bot_protegido, obteve %s", discWaf.Plataforma)
	}
}

func TestTCEConnectorsMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		// TCE-PI
		case r.URL.Path == "/pi/prefeituras":
			_, _ = w.Write([]byte(`[{"id":133,"nome":"TERESINA","codIBGE":"2211001","urlPrefeitura":"https://pmt.pi.gov.br","urlCamara":"https://camara.teresina.pi.leg.br"}]`))
		case r.URL.Path == "/pi/servidores/133/2025/lista":
			_, _ = w.Write([]byte(`[{"nome":"ANA PIAUI","cpf":"***.111.222-**","cargo":"ENFERMEIRA","lotacao":"FMS","vinculo":"EFETIVO"}]`))
		case r.URL.Path == "/pi/despesas/133/2025/porFuncao":
			_, _ = w.Write([]byte(`[{"funcao":"SAUDE","valorEmpenhado":100000.0,"valorPago":90000.0}]`))
		case r.URL.Path == "/pi/receitas/133/2025":
			_, _ = w.Write([]byte(`[{"codigo":"1.1","descricao":"ISS","valorPrevisto":50000.0,"valorArrecadado":55000.0}]`))

		// TCE-CE SIM
		case r.URL.Path == "/ce/municipios":
			_, _ = w.Write([]byte(`{"elements":[{"codigo_municipio":"001","nome_municipio":"ABAIARA","codigo_municipio_ibge":2300101}]}`))
		case r.URL.Path == "/ce/agentes_publicos_municipais":
			_, _ = w.Write([]byte(`{"elements":[{"nome_servidor":"CARLOS CEARA","cpf_servidor":"11122233344","numero_matricula":"99","nm_tipo_cargo":"PROFESSOR","codigo_vinculo":"EFETIVO","data_referencia_doc":202501}]}`))
		case r.URL.Path == "/ce/agentes_publicos_folha":
			_, _ = w.Write([]byte(`{"elements":[
				{"cpf_agente":"11122233344","valor_item":4000.00,"codigo_detutibilidade_item":false},
				{"cpf_agente":"11122233344","valor_item":500.00,"codigo_detutibilidade_item":true}
			]}`))

		// TCE-SP
		case r.URL.Path == "/sp/municipios":
			_, _ = w.Write([]byte(`[{"municipio":"adamantina","municipio_extenso":"Adamantina"}]`))
		case r.URL.Path == "/sp/despesas/adamantina/2025/1":
			_, _ = w.Write([]byte(`[{"orgao":"PREFEITURA","mes":"Janeiro","evento":"Valor Pago","nr_empenho":"77","nm_fornecedor":"FORNECEDOR SP","vl_despesa":"1.500,75"}]`))
		case r.URL.Path == "/sp/receitas/adamantina/2025/1":
			_, _ = w.Write([]byte(`[{"orgao":"PREFEITURA","mes":"Janeiro","ds_fonte_recurso":"TESOURO","ds_alinea":"IPTU","ds_subalinea":"PRINCIPAL","vl_arrecadacao":"9.876,54"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	tce := transparencia.NewTCEClient(srv.Client())
	tce.SetTCEPIBaseURL(srv.URL + "/pi")
	tce.SetTCECEBaseURL(srv.URL + "/ce")
	tce.SetTCESPBaseURL(srv.URL + "/sp")

	ctx := context.Background()

	// PI
	mPI := &transparencia.Municipio{CodigoIBGE: "2211001", Nome: "Teresina", UF: "PI", Slug: "teresina"}
	piServ, err := tce.FetchTCEPIServidores(ctx, mPI, 2025, "", 10)
	if err != nil || len(piServ.Servidores) != 1 || piServ.Servidores[0].Nome != "ANA PIAUI" {
		t.Fatalf("FetchTCEPIServidores falhou: %+v err=%v", piServ, err)
	}
	piDesp, err := tce.FetchTCEPIDespesas(ctx, mPI, 2025, 10)
	if err != nil || len(piDesp.Despesas) != 1 || piDesp.Despesas[0].ValorPago != 90000.0 {
		t.Fatalf("FetchTCEPIDespesas falhou: %+v err=%v", piDesp, err)
	}
	piRec, err := tce.FetchTCEPIReceitas(ctx, mPI, 2025, 10)
	if err != nil || len(piRec.Receitas) != 1 || piRec.Receitas[0].ValorArrecadado != 55000.0 {
		t.Fatalf("FetchTCEPIReceitas falhou: %+v err=%v", piRec, err)
	}

	// CE (verifica soma da folha bruto/descontos/líquido)
	mCE := &transparencia.Municipio{CodigoIBGE: "2300101", Nome: "Abaiara", UF: "CE", Slug: "abaiara"}
	ceServ, err := tce.FetchTCECEServidores(ctx, mCE, 2025, 1, "", 10)
	if err != nil || len(ceServ.Servidores) != 1 {
		t.Fatalf("FetchTCECEServidores falhou: %+v err=%v", ceServ, err)
	}
	if ceServ.Servidores[0].SalarioBruto != 4000.0 || ceServ.Servidores[0].Descontos != 500.0 || ceServ.Servidores[0].SalarioLiquido != 3500.0 {
		t.Errorf("Cálculo de folha TCE-CE incorreto: %+v", ceServ.Servidores[0])
	}

	// SP
	mSP := &transparencia.Municipio{CodigoIBGE: "3500105", Nome: "Adamantina", UF: "SP", Slug: "adamantina"}
	spDesp, err := tce.FetchTCESPDespesas(ctx, mSP, 2025, 1, 10)
	if err != nil || len(spDesp.Despesas) != 1 || spDesp.Despesas[0].ValorPago != 1500.75 {
		t.Fatalf("FetchTCESPDespesas falhou: %+v err=%v", spDesp, err)
	}
	spRec, err := tce.FetchTCESPReceitas(ctx, mSP, 2025, 1, 10)
	if err != nil || len(spRec.Receitas) != 1 || spRec.Receitas[0].ValorArrecadado != 9876.54 {
		t.Fatalf("FetchTCESPReceitas falhou: %+v err=%v", spRec, err)
	}
}

func TestPNCPAndSiconfiRGFAndHubMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/publico/versao":
			_, _ = w.Write([]byte(`"v26.10.30"`))
		case "/publico/buscarEntidades":
			_, _ = w.Write([]byte(`[{"codAdministracao":2,"desAdministracao":"PREFEITURA"}]`))
		case "/publico/servidor":
			_, _ = w.Write([]byte(`{"content":[{"nomServidor":"SERVIDOR HUB","numcpf":"11122233344","nomCargo":"ANALISTA","vlrSalarioBruto":5000,"vlrSalarioLiquido":4200}],"totalElements":1}`))
		case "/publico/despesaDetalhada":
			_, _ = w.Write([]byte(`{"content":[{"numEmpenho":"1/2025","nomFornecedor":"CREDOR HUB","vlrPago":999.0}],"totalElements":1}`))
		case "/publico/receita":
			_, _ = w.Write([]byte(`{"content":[{"desReceita":"IPTU HUB","vlrArrecadado":2000.0}],"totalElements":1}`))
		case "/publico/licitacao":
			_, _ = w.Write([]byte(`{"content":[],"totalElements":0}`))
		case "/publico/contrato":
			_, _ = w.Write([]byte(`{"content":[],"totalElements":0}`))
		case "/pncp/contratacoes/publicacao":
			_, _ = w.Write([]byte(`{"data":[{"numeroControlePNCP":"123-1-0001/2025","anoCompra":2025,"modalidadeNome":"Pregão","objetoCompra":"SERVICOS TI PNCP","valorTotalEstimado":12000.0}],"totalRegistros":1}`))
		case "/pncp/contratos":
			_, _ = w.Write([]byte(`{"data":[{"numeroContratoEmpenho":"CT-01/2025","anoContrato":2025,"nomeRazaoSocialFornecedor":"TECH LTDA","objetoContrato":"CONTRATO PNCP","valorGlobal":11000.0}],"totalRegistros":1}`))
		case "/siconfi/dca":
			_, _ = w.Write([]byte(`{"items":[{"exercicio":2025,"cod_ibge":3166600,"conta":"Ativo Total","valor":5000000.0}]}`))
		case "/siconfi/rgf":
			_, _ = w.Write([]byte(`{"items":[{"exercicio":2025,"cod_ibge":3166600,"conta":"Despesa Total com Pessoal","valor":1800000.0}]}`))
		case "/siconfi/extrato_entregas":
			_, _ = w.Write([]byte(`{"items":[{"exercicio":2025,"cod_ibge":3166600,"entregavel":"RGF","periodo":2,"periodicidade":"S","data_status":"2026-01-30","forma_envio":"M"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	engine, err := transparencia.NewEngine()
	if err != nil {
		t.Fatalf("NewEngine erro: %v", err)
	}

	hub := &transparencia.Hub{
		Engine:     engine,
		HTTPClient: srv.Client(),
		TCE:        transparencia.NewTCEClient(srv.Client()),
		Siconfi:    transparencia.NewSiconfiClientWithBaseURL(srv.URL+"/siconfi", srv.Client()),
		PNCP:       transparencia.NewPNCPClientWithBaseURL(srv.URL+"/pncp", srv.Client()),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Testa QueryTransparenciaCompleta integrando Quasar + Fallback PNCP + SICONFI DCA/RGF/Entregas
	report, err := hub.QueryTransparenciaCompleta(ctx, "Serra da Saudade", transparencia.QueryOptions{
		UF:          "MG",
		OverrideURL: srv.URL,
		Ano:         2025,
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("QueryTransparenciaCompleta erro: %v", err)
	}
	if report.Servidores == nil || len(report.Servidores.Servidores) != 1 || report.Servidores.Servidores[0].Nome != "SERVIDOR HUB" {
		t.Errorf("Servidores no relatório completo inesperados: %+v", report.Servidores)
	}
	if report.Despesas == nil || len(report.Despesas.Despesas) != 1 {
		t.Errorf("Despesas no relatório completo inesperadas: %+v", report.Despesas)
	}
	if report.Receitas == nil || len(report.Receitas.Receitas) != 1 {
		t.Errorf("Receitas no relatório completo inesperadas: %+v", report.Receitas)
	}
	// Como Quasar retornou [] para licitações e contratos, o Hub deve fazer fallback para PNCP!
	if report.Licitacoes == nil || len(report.Licitacoes.Licitacoes) == 0 || report.Licitacoes.Plataforma != "pncp" {
		t.Errorf("Fallback PNCP de licitações não acionado corretamente: %+v", report.Licitacoes)
	}
	if report.Contratos == nil || len(report.Contratos.Contratos) != 1 || report.Contratos.Plataforma != "pncp" {
		t.Errorf("Fallback PNCP de contratos não acionado corretamente: %+v", report.Contratos)
	}
	if len(report.SiconfiRGF) != 1 || report.SiconfiRGF[0].Valor != 1800000.0 {
		t.Errorf("SiconfiRGF inesperado: %+v", report.SiconfiRGF)
	}
	if len(report.SiconfiDCA) != 1 || len(report.SiconfiEntregas) != 1 {
		t.Errorf("SiconfiDCA ou Entregas inesperados")
	}

	// Garante que serialização JSON completa do report funciona sem erros
	b, err := json.Marshal(report)
	if err != nil || !strings.Contains(string(b), "***.222.333-**") {
		t.Fatalf("Serialização JSON ou mascaramento falhou: err=%v", err)
	}
}

func TestMegasoftCatalog(t *testing.T) {
	eng, err := transparencia.NewEngine()
	if err != nil {
		t.Fatalf("NewEngine erro: %v", err)
	}
	all := eng.ListMegasoftMunicipios("")
	if len(all) != 268 {
		t.Errorf("Esperava 268 municípios Megasoft, obteve %d", len(all))
	}
	goMunis := eng.ListMegasoftMunicipios("GO")
	if len(goMunis) != 156 {
		t.Errorf("Esperava 156 municípios de GO no Megasoft, obteve %d", len(goMunis))
	}
	msMunis := eng.ListMegasoftMunicipios("MS")
	if len(msMunis) != 2 {
		t.Errorf("Esperava 2 municípios de MS no Megasoft, obteve %d", len(msMunis))
	}
	tacuru, found := eng.GetMegasoftMunicipio("5007950")
	if !found || tacuru.Nome != "Tacuru" || tacuru.UF != "MS" {
		t.Fatalf("GetMegasoftMunicipio Tacuru falhou: %+v", tacuru)
	}
}

func TestMegasoftConnectorMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/configuracao/carregamento-inicial":
			_, _ = w.Write([]byte(`{
				"tokenPortal": "mock-token-xyz",
				"entidades": [
					{"id": 1, "nome": "Prefeitura Municipal", "tipoEntidade": "PREFEITURA"},
					{"id": 2, "nome": "Câmara Municipal", "tipoEntidade": "CAMARA"}
				],
				"anos": [2024, 2025, 2026]
			}`))
		case "/api/orgaos-e-servidores/servidor/paginado":
			_, _ = w.Write([]byte(`{
				"total": 1,
				"registros": [
					{
						"matricula": "M100",
						"nome": "MARIA DA SILVA",
						"cpf": "12345678901",
						"cargo": "PROFESSORA",
						"departamento": "EDUCACAO",
						"proventos": 5000.0,
						"totalLiquido": 4200.0,
						"descontos": 800.0,
						"situacao": "ATIVO"
					}
				]
			}`))
		case "/api/orgaos-e-servidores/servidor/detalhe":
			_, _ = w.Write([]byte(`{
				"matricula": "M100",
				"nome": "MARIA DA SILVA",
				"cpf": "12345678901",
				"cargo": "PROFESSORA",
				"orgao": "PREFEITURA",
				"departamento": "EDUCACAO",
				"mesAno": "10/2025",
				"somatorioMovimentoVencimentos": {
					"somatorioProventos": 5000.0,
					"somatorioDescontos": 800.0,
					"totalLiquido": 4200.0
				},
				"movimentoVencimentos": {
					"total": 4,
					"registros": [
						{"nome": "VENCIMENTO BASE", "proventos": 4500.0, "descontos": 0.0},
						{"nome": "QUINQUENIO", "proventos": 500.0, "descontos": 0.0},
						{"nome": "PREVIDENCIA", "proventos": 0.0, "descontos": 600.0},
						{"nome": "IRRF", "proventos": 0.0, "descontos": 200.0}
					]
				}
			}`))
		case "/api/orgaos-e-servidores/diaria/paginado":
			_, _ = w.Write([]byte(`{
				"total": 1,
				"registros": [
					{
						"nomeDoFuncionario": "JOAO SOUZA",
						"cargoDoFuncionario": "MOTORISTA",
						"valorDaDiaria": 350.0,
						"destino": "GOIANIA",
						"motivoDaViagem": "TRANSPORTE PACIENTES",
						"dataDaSaida": "10/05/2025"
					}
				]
			}`))
		case "/api/contratos-convenios-e-licitacoes/obra/paginado":
			_, _ = w.Write([]byte(`{
				"total": 1,
				"registros": [
					{
						"titulo": "REFORMA PRACA CENTRAL",
						"situacao": "EM ANDAMENTO",
						"valor": 150000.0,
						"dataInicial": "01/02/2025"
					}
				]
			}`))
		case "/api/contratos-convenios-e-licitacoes/convenio/paginado":
			_, _ = w.Write([]byte(`{
				"total": 1,
				"registros": [
					{
						"numeroDoConvenio": "001/2025",
						"descricao": "REFORMA ESCOLA",
						"valor": 80000.0,
						"nomeDoConcedente": "ESTADO",
						"data": "15/03/2025"
					}
				]
			}`))
		case "/api/legislacao-e-publicacoes/legislacao-municipal/paginado":
			_, _ = w.Write([]byte(`{
				"total": 1,
				"registros": [
					{
						"numeroDoDocumento": "1234",
						"ano": 2025,
						"tipoDoDocumento": "LEI ORDINARIA",
						"descricao": "DISPOE SOBRE O PLANO DIRETOR"
					}
				]
			}`))
		case "/api/orgaos-e-servidores/lista-de-aprovados-em-concursos-e-processos-seletivos/paginado":
			_, _ = w.Write([]byte(`{
				"total": 1,
				"registros": [
					{
						"nomeDaPessoa": "ANA PAULA",
						"nomeDoCargo": "ENFERMEIRA",
						"classificacao": 1,
						"descricaoDaSituacao": "APROVADO",
						"numeroDoEdital": "01/2024"
					}
				]
			}`))
		case "/api/contratos-convenios-e-licitacoes/procedimento-licitatorio/paginado":
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			_, _ = w.Write([]byte(`{
				"total": 1,
				"registros": [
					{
						"numeroAno": "005/2025",
						"historico": "AQUISICAO DE MEDICAMENTOS",
						"modalidade": "PREGAO ELETRONICO",
						"situacao": "HOMOLOGADO"
					}
				]
			}`))
		case "/api/contratos-convenios-e-licitacoes/contrato/paginado":
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			_, _ = w.Write([]byte(`{
				"total": 1,
				"registros": [
					{
						"numero": "010/2025",
						"nomeDoFornecedor": "FARMA DISTRIBUIDORA",
						"objeto": "FORNECIMENTO DE MEDICAMENTOS",
						"valor": 44000.0,
						"vigencia": "01/03/2025"
					}
				]
			}`))
		case "/api/receitas-e-despesas/empenho/paginado":
			_, _ = w.Write([]byte(`{
				"total": 1,
				"registros": [
					{
						"codigo": "2025/001",
						"nomeDoFornecedor": "FARMA DISTRIBUIDORA",
						"historico": "PAGAMENTO MEDICAMENTOS",
						"valorDoEmpenho": 44000.0
					}
				]
			}`))
		case "/api/receitas-e-despesas/receita/paginado":
			_, _ = w.Write([]byte(`{
				"total": 1,
				"registros": [
					{
						"receita": "IPTU",
						"arrecadacao": 75000.0
					}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	eng, err := transparencia.NewEngine()
	if err != nil {
		t.Fatalf("NewEngine erro: %v", err)
	}
	hub := &transparencia.Hub{
		Engine:     eng,
		HTTPClient: srv.Client(),
		TCE:        transparencia.NewTCEClient(srv.Client()),
		Siconfi:    transparencia.NewSiconfiClientWithBaseURL(srv.URL+"/siconfi", srv.Client()),
		PNCP:       transparencia.NewPNCPClientWithBaseURL(srv.URL+"/pncp", srv.Client()),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := transparencia.QueryOptions{
		UF:          "MS",
		OverrideURL: srv.URL,
		Ano:         2025,
		Limit:       10,
	}

	// 1. Servidores
	serv, err := hub.QueryServidores(ctx, "Tacuru", opts)
	if err != nil || len(serv.Servidores) != 1 {
		t.Fatalf("QueryServidores Megasoft falhou: err=%v, res=%+v", err, serv)
	}
	if serv.Servidores[0].CPFParcial != "***.456.789-**" {
		t.Errorf("CPF mascarado incorreto: %s", serv.Servidores[0].CPFParcial)
	}
	if serv.Plataforma != "megasoft" {
		t.Errorf("Plataforma esperada megasoft, obteve %s", serv.Plataforma)
	}

	// 2. Servidor Detalhe
	det, err := hub.QueryServidorDetalhe(ctx, "Tacuru", "M100", opts)
	if err != nil || det.SalarioBruto != 5000.0 || len(det.Vencimentos) != 2 || len(det.VerbasDesconto) != 2 {
		t.Fatalf("QueryServidorDetalhe Megasoft falhou: err=%v, res=%+v", err, det)
	}
	if det.CPFParcial != "***.456.789-**" {
		t.Errorf("CPFParcial em detalhe incorreto: %s", det.CPFParcial)
	}

	// 3. Diárias
	diar, err := hub.QueryDiarias(ctx, "Tacuru", opts)
	if err != nil || len(diar.Diarias) != 1 || diar.Diarias[0].Valor != 350.0 {
		t.Fatalf("QueryDiarias Megasoft falhou: err=%v, res=%+v", err, diar)
	}

	// 4. Obras
	obr, err := hub.QueryObras(ctx, "Tacuru", opts)
	if err != nil || len(obr.Obras) != 1 || obr.Obras[0].Valor != 150000.0 {
		t.Fatalf("QueryObras Megasoft falhou: err=%v, res=%+v", err, obr)
	}

	// 5. Convênios
	conv, err := hub.QueryConvenios(ctx, "Tacuru", opts)
	if err != nil || len(conv.Convenios) != 1 || conv.Convenios[0].Valor != 80000.0 {
		t.Fatalf("QueryConvenios Megasoft falhou: err=%v, res=%+v", err, conv)
	}

	// 6. Legislação
	leg, err := hub.QueryLegislacao(ctx, "Tacuru", opts)
	if err != nil || len(leg.Atos) != 1 || leg.Atos[0].Numero != "1234" {
		t.Fatalf("QueryLegislacao Megasoft falhou: err=%v, res=%+v", err, leg)
	}

	// 7. Concursos
	conc, err := hub.QueryConcursos(ctx, "Tacuru", opts)
	if err != nil || len(conc.Aprovados) != 1 || conc.Aprovados[0].Cargo != "ENFERMEIRA" {
		t.Fatalf("QueryConcursos Megasoft falhou: err=%v, res=%+v", err, conc)
	}

	// 8. Licitações
	lic, err := hub.QueryLicitacoes(ctx, "Tacuru", opts)
	if err != nil || len(lic.Licitacoes) != 1 || lic.Licitacoes[0].Objeto != "AQUISICAO DE MEDICAMENTOS" {
		t.Fatalf("QueryLicitacoes Megasoft falhou: err=%v, res=%+v", err, lic)
	}

	// 9. Contratos
	cnt, err := hub.QueryContratos(ctx, "Tacuru", opts)
	if err != nil || len(cnt.Contratos) != 1 || cnt.Contratos[0].Valor != 44000.0 {
		t.Fatalf("QueryContratos Megasoft falhou: err=%v, res=%+v", err, cnt)
	}

	// 10. Despesas & Receitas
	desp, err := hub.QueryDespesas(ctx, "Tacuru", opts)
	if err != nil || len(desp.Despesas) != 1 {
		t.Fatalf("QueryDespesas Megasoft falhou: err=%v", err)
	}
	rec, err := hub.QueryReceitas(ctx, "Tacuru", opts)
	if err != nil || len(rec.Receitas) != 1 {
		t.Fatalf("QueryReceitas Megasoft falhou: err=%v", err)
	}

	// 11. Transparência Completa
	rep, err := hub.QueryTransparenciaCompleta(ctx, "Tacuru", opts)
	if err != nil {
		t.Fatalf("QueryTransparenciaCompleta Megasoft falhou: %v", err)
	}
	if rep.Diarias == nil || len(rep.Diarias.Diarias) != 1 {
		t.Errorf("Diarias ausente no report completo: %+v", rep.Diarias)
	}
	if rep.Obras == nil || len(rep.Obras.Obras) != 1 {
		t.Errorf("Obras ausente no report completo: %+v", rep.Obras)
	}
	if rep.Convenios == nil || len(rep.Convenios.Convenios) != 1 {
		t.Errorf("Convenios ausente no report completo: %+v", rep.Convenios)
	}
	if rep.Legislacao == nil || len(rep.Legislacao.Atos) != 1 {
		t.Errorf("Legislacao ausente no report completo: %+v", rep.Legislacao)
	}
	if rep.Concursos == nil || len(rep.Concursos.Aprovados) != 1 {
		t.Errorf("Concursos ausente no report completo: %+v", rep.Concursos)
	}
}
