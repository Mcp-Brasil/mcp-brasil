package transparencia_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Mcp-Brasil/mcp-brasil/pkg/transparencia"
)

func utf8ToLatin1(s string) []byte {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		if r <= 0xFF {
			out = append(out, byte(r))
		} else {
			out = append(out, '?')
		}
	}
	return out
}

func TestTCEPEConnector(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/UnidadesJurisdicionadas!json":
			w.Write(utf8ToLatin1(`{"resposta":{"status":"OK","entidade":"UnidadesJurisdicionadas","conteudo":[{"Codigo":"101","Nome":"Prefeitura Municipal de Recife","Natureza":"Prefeitura","Municipio":"Recife","CodigoMunicipio":"2611606"}]}}`))
		case "/DespesasMunicipais!json":
			if r.URL.Query().Get("ANOREFERENCIA") == "2024" {
				w.Write(utf8ToLatin1(`{"resposta":{"status":"OK","entidade":"DespesasMunicipais","conteudo":[{"NUMEROEMPENHO":"001/2024","ANOREFERENCIA":2024,"MESREFERENCIA":3,"FORNECEDOR":"Construtora São João Ltda","CPF_CNPJ":"12345678000199","HISTORICO":"Manutenção de escolas","VALOREMPENHADO":50000.0,"VALORLIQUIDADO":40000.0,"VALORPAGO":35000.0,"FUNCAO":"Educação","ELEMENTODESPESA":"Obras e Instalações","NOMEUNIDADEGESTORA":"Secretaria de Educação"}]}}`))
				return
			}
			w.Write([]byte(`{"resposta":{"status":"OK","conteudo":[]}}`))
		case "/LicitacaoUG!json":
			w.Write(utf8ToLatin1(`{"resposta":{"status":"OK","entidade":"LicitacaoUG","conteudo":[{"NUMEROLICITACAO":"PE-010/2024","ANOLICITACAO":2024,"MODALIDADE":"Pregão Eletrônico","OBJETO":"Aquisição de merenda escolar","VALORESTIMADO":120000.50,"SITUACAO":"Homologada"}]}}`))
		case "/Contratos!json":
			w.Write(utf8ToLatin1(`{"resposta":{"status":"OK","entidade":"Contratos","conteudo":[{"NUMEROCONTRATO":"CT-055/2024","ANOREFERENCIA":2024,"OBJETO":"Fornecimento de alimentação","VALORCONTRATO":95000.0,"FORNECEDOR":"Alimentos PE Ltda","CPFCNPJ":"98765432000100"}]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	tce := transparencia.NewTCEClient(srv.Client())
	tce.SetTCEPEBaseURL(srv.URL)
	ctx := context.Background()
	m := &transparencia.Municipio{CodigoIBGE: "2611606", Nome: "Recife", UF: "PE"}

	unidades, err := tce.ResolvePEUnidades(ctx, "Recife")
	if err != nil {
		t.Fatalf("ResolvePEUnidades: %v", err)
	}
	if len(unidades) != 1 || unidades[0].Municipio != "Recife" {
		t.Fatalf("unidades inesperadas: %+v", unidades)
	}

	desp, err := tce.FetchTCEPEDespesas(ctx, m, 2024, 3, 10)
	if err != nil {
		t.Fatalf("FetchTCEPEDespesas: %v", err)
	}
	if len(desp.Despesas) != 1 || desp.Despesas[0].Funcao != "Educação" || desp.Despesas[0].Fornecedor != "Construtora São João Ltda" || desp.Despesas[0].ValorPago != 35000.0 {
		t.Fatalf("despesas TCE-PE inesperadas: %+v", desp.Despesas)
	}

	lic, err := tce.FetchTCEPELicitacoes(ctx, m, 2024, 10)
	if err != nil {
		t.Fatalf("FetchTCEPELicitacoes: %v", err)
	}
	if len(lic.Licitacoes) != 1 || lic.Licitacoes[0].Modalidade != "Pregão Eletrônico" {
		t.Fatalf("licitações TCE-PE inesperadas: %+v", lic.Licitacoes)
	}

	ct, err := tce.FetchTCEPEContratos(ctx, m, 2024, 10)
	if err != nil {
		t.Fatalf("FetchTCEPEContratos: %v", err)
	}
	if len(ct.Contratos) != 1 || ct.Contratos[0].Numero != "CT-055/2024" {
		t.Fatalf("contratos TCE-PE inesperados: %+v", ct.Contratos)
	}
}

func TestTCERJConnector(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/licitacoes":
			w.Write([]byte(`{"Count":1,"Licitacoes":[{"Ente":"Niterói","Unidade":"FMS","Ano":2024,"Modalidade":"Pregão Eletrônico","NumeroEdital":"012/2024","Objeto":"Compra de medicamentos","ValorEstimado":250000.0,"DataPublicacaoOficial":"2024-04-10"}]}`))
		case "/contratos_municipio":
			w.Write([]byte(`{"Count":1,"Contratos":[{"Ente":"Niterói","NumeroContrato":"100/2024","AnoContrato":2024,"Contratado":"Farma Rio Ltda","CNPJCPFContratado":"11222333000144","Objeto":"Fornecimento hospitalar","ValorContrato":180000.0,"DataAssinaturaContrato":"2024-05-01","DataVencimentoContrato":"2025-05-01"}]}`))
		case "/obras_paralisadas":
			w.Write([]byte(`{"Obras":[{"Ente":"Niterói","TipoEnte":"Município","Nome":"Creche Centro","ValorTotalContrato":900000.0,"MotivoParalisacao":"Rescisão contratual","DataInicioObra":"2022-01-15","DataParalisacao":"2023-08-20","StatusContrato":"Paralisada"},{"Ente":"Petrópolis","TipoEnte":"Município","Nome":"Outra Obra","ValorTotalContrato":100000.0}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	tce := transparencia.NewTCEClient(srv.Client())
	tce.SetTCERJBaseURL(srv.URL)
	ctx := context.Background()
	m := &transparencia.Municipio{CodigoIBGE: "3303302", Nome: "Niterói", UF: "RJ"}

	lic, err := tce.FetchTCERJLicitacoes(ctx, m, 2024, 10)
	if err != nil {
		t.Fatalf("FetchTCERJLicitacoes: %v", err)
	}
	if len(lic.Licitacoes) != 1 || lic.Licitacoes[0].Numero != "012/2024" {
		t.Fatalf("licitações TCE-RJ inesperadas: %+v", lic.Licitacoes)
	}

	ct, err := tce.FetchTCERJContratos(ctx, m, 2024, 10)
	if err != nil {
		t.Fatalf("FetchTCERJContratos: %v", err)
	}
	if len(ct.Contratos) != 1 || ct.Contratos[0].Fornecedor != "Farma Rio Ltda" {
		t.Fatalf("contratos TCE-RJ inesperados: %+v", ct.Contratos)
	}

	obras, err := tce.FetchTCERJObrasParalisadas(ctx, m, 10)
	if err != nil {
		t.Fatalf("FetchTCERJObrasParalisadas: %v", err)
	}
	if len(obras.Obras) != 1 || obras.Obras[0].Titulo != "Creche Centro" {
		t.Fatalf("obras TCE-RJ inesperadas: %+v", obras.Obras)
	}
}

func TestTCERNConnector(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/InformacoesBasicasApi/JurisdicionadosTCE/Json":
			w.Write([]byte(`[{"identificadorUnidade":42,"codigoOrgao":"100","nomeOrgao":"PREFEITURA MUNICIPAL DE NATAL","cnpj":"08.241.747/0001-43"}]`))
		case "/BalancoOrcamentarioApi/Despesa/Json/2024/6/42":
			w.Write([]byte(`[{"descricaoCategoriaEconomica":"Despesas Correntes","descricaoGrupoDespesa":"Pessoal e Encargos","descricaoElementoDespesa":"Vencimentos","valorEmpenhoAtePeriodo":1000000.0,"valorLiquidacaoAtePeriodo":950000.0,"valorPagoAtePeriodo":940000.0}]`))
		case "/BalancoOrcamentarioApi/Receita/Json/2024/6/42":
			w.Write([]byte(`[{"descricaoReceita":"IPTU","codNaturezaReceita":"11125001","valorPrevistoInicial":500000.0,"valorPrevistoAtualizado":520000.0,"valorRealizadoNoExecicio":480000.0}]`))
		case "/ProcedimentosLicitatoriosApi/LicitacaoPublica/Json/42/2024-01-01/2024-12-31":
			w.Write([]byte(`[{"numeroLicitacao":"001/2024","anoLicitacao":"2024","modalidade":"Pregão","descricaoObjeto":"Serviços de TI","valorTotalOrcado":300000.0,"situacaoProcedimentoLicitacao":"Homologada"}]`))
		case "/ContratosApi/Contratos/Json/42/false":
			w.Write([]byte(`[{"numeroContrato":"CT-01/2024","anoContrato":2024,"objetoContrato":"Locação de veículos","valorContrato":150000.0,"nomeContratado":"Auto Norte Ltda","cpfcnpjContratado":"00111222000133","dataInicioVigencia":"2024-01-01","dataTerminoVigencia":"2024-12-31"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	tce := transparencia.NewTCEClient(srv.Client())
	tce.SetTCERNBaseURL(srv.URL)
	ctx := context.Background()
	m := &transparencia.Municipio{CodigoIBGE: "2408102", Nome: "Natal", UF: "RN", CNPJ: "08241747000143"}

	desp, err := tce.FetchTCERNDespesas(ctx, m, 2024, 6, 10)
	if err != nil {
		t.Fatalf("FetchTCERNDespesas: %v", err)
	}
	if len(desp.Despesas) != 1 || desp.Despesas[0].ValorPago != 940000.0 {
		t.Fatalf("despesas TCE-RN inesperadas: %+v", desp.Despesas)
	}

	rec, err := tce.FetchTCERNReceitas(ctx, m, 2024, 6, 10)
	if err != nil {
		t.Fatalf("FetchTCERNReceitas: %v", err)
	}
	if len(rec.Receitas) != 1 || rec.Receitas[0].ValorArrecadado != 480000.0 {
		t.Fatalf("receitas TCE-RN inesperadas: %+v", rec.Receitas)
	}

	lic, err := tce.FetchTCERNLicitacoes(ctx, m, 2024, 10)
	if err != nil {
		t.Fatalf("FetchTCERNLicitacoes: %v", err)
	}
	if len(lic.Licitacoes) != 1 || lic.Licitacoes[0].Numero != "001/2024" {
		t.Fatalf("licitações TCE-RN inesperadas: %+v", lic.Licitacoes)
	}

	ct, err := tce.FetchTCERNContratos(ctx, m, 10)
	if err != nil {
		t.Fatalf("FetchTCERNContratos: %v", err)
	}
	if len(ct.Contratos) != 1 || ct.Contratos[0].Fornecedor != "Auto Norte Ltda" {
		t.Fatalf("contratos TCE-RN inesperados: %+v", ct.Contratos)
	}
}

func TestTCESCConnector(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/municipios.php":
			w.Write([]byte(`[{"codigo_municipio":4205407,"nome_municipio":"Florianópolis"}]`))
		case "/unidades-gestoras.php":
			w.Write([]byte(`[{"codigo_unidade":1,"nome_unidade":"Prefeitura de Florianópolis","sigla_unidade":"PMF","nome_municipio":"Florianópolis"},{"codigo_unidade":2,"nome_unidade":"Prefeitura de Joinville","sigla_unidade":"PMJ","nome_municipio":"Joinville"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	tce := transparencia.NewTCEClient(srv.Client())
	tce.SetTCESCBaseURL(srv.URL)
	ctx := context.Background()
	m := &transparencia.Municipio{CodigoIBGE: "4205407", Nome: "Florianópolis", UF: "SC"}

	scMun, err := tce.ResolveSCMunicipio(ctx, m)
	if err != nil || scMun.CodigoMunicipio != 4205407 {
		t.Fatalf("ResolveSCMunicipio: err=%v mun=%+v", err, scMun)
	}

	ugs, err := tce.FetchTCESCUnidadesGestoras(ctx, m)
	if err != nil || len(ugs) != 1 || ugs[0].SiglaUnidade != "PMF" {
		t.Fatalf("FetchTCESCUnidadesGestoras: err=%v ugs=%+v", err, ugs)
	}
}

func TestTCETOAndTCEPAConnectors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/to/pessoas"):
			if r.Header.Get("Accept") != "application/json" {
				http.Error(w, "missing accept", http.StatusBadRequest)
				return
			}
			w.Write([]byte(`[{"id":10,"nome":"JOAO SILVA","codigo":"123","processos":[{"numero_ano":"100/2024","assunto":"Contas","entidade_origem_municipio":"Palmas"}]}]`))
		case strings.HasPrefix(r.URL.Path, "/to/pautas"):
			w.Write([]byte(`{"pautas":[{"data":"2024-06-01","hora":"09:00","tipo":"Pleno","origem":"1a Câmara","url":"https://tce.to/pauta/1"}]}`))
		case r.URL.Path == "/to/processo/100/2024":
			w.Write([]byte(`{"numero_ano":"100/2024","assunto":"Prestação de Contas","entidade_origem_municipio":"Palmas"}`))
		case r.URL.Path == "/pa/v1/diario_oficial":
			w.Write([]byte(`{"data":[{"NumeroPublicacao":500,"DataPublicacao":"2024-05-10","TipoAto":"Contratos","Publicacao":"<p>Extrato de <b>contrato</b> firmado.</p>"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	tce := transparencia.NewTCEClient(srv.Client())
	tce.SetTCETOBaseURL(srv.URL + "/to")
	tce.SetTCEPABaseURL(srv.URL + "/pa")
	ctx := context.Background()

	pessoas, err := tce.FetchTCETOPessoas(ctx, "JOAO", 10)
	if err != nil || len(pessoas) != 1 || pessoas[0].Nome != "JOAO SILVA" {
		t.Fatalf("FetchTCETOPessoas: err=%v pessoas=%+v", err, pessoas)
	}
	pautas, err := tce.FetchTCETOPautas(ctx, 10)
	if err != nil || len(pautas) != 1 || pautas[0].Tipo != "Pleno" {
		t.Fatalf("FetchTCETOPautas: err=%v pautas=%+v", err, pautas)
	}
	proc, err := tce.FetchTCETOProcesso(ctx, 100, 2024)
	if err != nil || proc.EntidadeOrigemMunicipio != "Palmas" {
		t.Fatalf("FetchTCETOProcesso: err=%v proc=%+v", err, proc)
	}

	diarios, err := tce.FetchTCEPADiarioOficial(ctx, 2024, 5, "Contratos", 10)
	if err != nil || len(diarios) != 1 || diarios[0].Publicacao != "Extrato de contrato firmado." {
		t.Fatalf("FetchTCEPADiarioOficial: err=%v diarios=%+v", err, diarios)
	}
}

func TestTCERSConnector(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dados/auxiliar/municipios.json":
			w.Write([]byte(`{"municipios":[{"COD_MUNICIPIO":149,"NOME_MUNICIPIO":"Porto Alegre","UF":"RS","CD_MUNICIPIO_IBGE":4314902}]}`))
		case "/dados/municipal/gastos-lrf-mde-asps/2024.json":
			w.Write([]byte(`{"dados":[{"Ano":2024,"CD_Orgao":1,"NM_Orgao":"PM DE PORTO ALEGRE","VL_ReceitaCorrenteLiquida":8000000.0,"VL_DespesaPessoal":3500000.0,"VL_ReceitaMDE":2000000.0,"VL_DespesaMDE":600000.0,"VL_ReceitaASPS":2000000.0,"VL_DespesaASPS":450000.0}]}`))
		case "/api/3/action/package_search":
			w.Write([]byte(`{"result":{"count":1,"results":[{"name":"despesas-2024","title":"Despesas 2024","notes":"Dados abertos","num_resources":2,"groups":[{"title":"Despesa"}]}]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	tce := transparencia.NewTCEClient(srv.Client())
	tce.SetTCERSBaseURL(srv.URL)
	ctx := context.Background()
	m := &transparencia.Municipio{CodigoIBGE: "4314902", Nome: "Porto Alegre", UF: "RS"}

	rsMun, err := tce.ResolveRSMunicipio(ctx, m)
	if err != nil || rsMun.Codigo != 149 {
		t.Fatalf("ResolveRSMunicipio: err=%v mun=%+v", err, rsMun)
	}

	desp, err := tce.FetchTCERSDespesas(ctx, m, 2024, 10)
	if err != nil || len(desp.Despesas) != 3 {
		t.Fatalf("FetchTCERSDespesas: err=%v desp=%+v", err, desp)
	}

	rec, err := tce.FetchTCERSReceitas(ctx, m, 2024, 10)
	if err != nil || len(rec.Receitas) != 3 {
		t.Fatalf("FetchTCERSReceitas: err=%v rec=%+v", err, rec)
	}

	ds, err := tce.FetchTCERSDatasets(ctx, "despesas", 10)
	if err != nil || len(ds) != 1 || ds[0].Name != "despesas-2024" {
		t.Fatalf("FetchTCERSDatasets: err=%v ds=%+v", err, ds)
	}
}
