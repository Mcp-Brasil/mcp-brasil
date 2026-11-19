package transparencia

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"
)

// all22HTMLPortalFamilies lista todas as 22 famílias HTML presentes em portais_html_municipais.csv.
var all22HTMLPortalFamilies = []string{
	"html_portal",
	"aspec_assesi_html",
	"ipm_html",
	"prefeitura_html",
	"fiorilli_html",
	"govbr_html",
	"cms_wordpress_html",
	"siplan_html",
	"equiplano_html",
	"el_gpi_html",
	"modernizacao_html",
	"betha_html",
	"ecrie_html",
	"aam_html",
	"centi_html",
	"agora_html",
	"nucleogov_html",
	"smarapd_html",
	"coplan_html",
	"sonner_html",
	"thema_html",
	"cms_wix_html",
}

func newFull10ModuleHTMLServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/", "":
			fmt.Fprint(w, `<!doctype html><html><body>
				<h1>Portal da Transparência Municipal</h1>
				<nav>
					<a href="/servidores">Servidores e Folha de Pagamento</a>
					<a href="/despesas">Despesas e Empenhos</a>
					<a href="/receitas">Receitas Arrecadadas</a>
					<a href="/licitacoes">Licitações e Editais</a>
					<a href="/contratos">Contratos Administrativos</a>
					<a href="/diarias">Diárias e Passagens</a>
					<a href="/obras">Obras Públicas</a>
					<a href="/convenios">Convênios e Repasses</a>
					<a href="/legislacao">Legislação e Atos Normativos</a>
					<a href="/concursos">Concursos e Processos Seletivos</a>
				</nav>
			</body></html>`)
		case "/servidores":
			fmt.Fprint(w, `<html><body><table>
				<tr><th>Nome</th><th>Matrícula</th><th>CPF</th><th>Cargo</th><th>Lotação</th><th>Salário Bruto</th><th>Descontos</th><th>Salário Líquido</th></tr>
				<tr><td>ANA SILVA</td><td>101</td><td>123.456.789-01</td><td>MÉDICA</td><td>SAÚDE</td><td>R$ 12.500,00</td><td>R$ 2.500,00</td><td>R$ 10.000,00</td></tr>
				<tr><td>BRUNO COSTA</td><td>102</td><td>234.567.890-12</td><td>PROFESSOR</td><td>EDUCAÇÃO</td><td>R$ 6.000,00</td><td>R$ 900,00</td><td>R$ 5.100,00</td></tr>
				<tr><td>CARLA DIAS</td><td>103</td><td>345.678.901-23</td><td>ENGENHEIRA</td><td>OBRAS</td><td>R$ 9.000,00</td><td>R$ 1.500,00</td><td>R$ 7.500,00</td></tr>
				<tr><td>DANIEL LIMA</td><td>104</td><td>456.789.012-34</td><td>ANALISTA</td><td>FINANÇAS</td><td>R$ 7.200,00</td><td>R$ 1.200,00</td><td>R$ 6.000,00</td></tr>
				<tr><td>ELENA ROCHA</td><td>105</td><td>567.890.123-45</td><td>ENFERMEIRA</td><td>SAÚDE</td><td>R$ 8.100,00</td><td>R$ 1.100,00</td><td>R$ 7.000,00</td></tr>
			</table></body></html>`)
		case "/despesas":
			fmt.Fprint(w, `<html><body><table>
				<tr><th>Empenho</th><th>Data</th><th>Fornecedor</th><th>CPF/CNPJ</th><th>Histórico</th><th>Valor Empenhado</th><th>Valor Pago</th></tr>
				<tr><td>001/2025</td><td>10/01/2025</td><td>CLINICA ALFA LTDA</td><td>12.345.678/0001-99</td><td>Servicos medicos</td><td>R$ 45.000,00</td><td>R$ 45.000,00</td></tr>
				<tr><td>002/2025</td><td>12/01/2025</td><td>JOAO AUTONOMO</td><td>123.456.789-01</td><td>Manutencao predial</td><td>R$ 2.500,00</td><td>R$ 2.500,00</td></tr>
				<tr><td>003/2025</td><td>15/01/2025</td><td>PAPELARIA BETA LTDA</td><td>98.765.432/0001-10</td><td>Material escolar</td><td>R$ 12.000,00</td><td>R$ 10.000,00</td></tr>
				<tr><td>004/2025</td><td>18/01/2025</td><td>CONSTRUTORA GAMA SA</td><td>11.222.333/0001-44</td><td>Asfalto</td><td>R$ 80.000,00</td><td>R$ 40.000,00</td></tr>
				<tr><td>005/2025</td><td>20/01/2025</td><td>AUTO POSTO DELTA</td><td>22.333.444/0001-55</td><td>Combustivel</td><td>R$ 15.000,00</td><td>R$ 15.000,00</td></tr>
			</table></body></html>`)
		case "/receitas":
			fmt.Fprint(w, `<html><body><table>
				<tr><th>Código</th><th>Descrição</th><th>Categoria</th><th>Valor Previsto</th><th>Valor Arrecadado</th></tr>
				<tr><td>1.7.1.8</td><td>FPM</td><td>Transferencias</td><td>R$ 500.000,00</td><td>R$ 480.000,00</td></tr>
				<tr><td>1.1.1.2</td><td>IPTU</td><td>Tributaria</td><td>R$ 200.000,00</td><td>R$ 195.000,00</td></tr>
				<tr><td>1.1.1.4</td><td>ISSQN</td><td>Tributaria</td><td>R$ 150.000,00</td><td>R$ 140.000,00</td></tr>
				<tr><td>1.7.2.8</td><td>ICMS</td><td>Transferencias</td><td>R$ 300.000,00</td><td>R$ 310.000,00</td></tr>
				<tr><td>1.7.1.5</td><td>FUNDEB</td><td>Transferencias</td><td>R$ 400.000,00</td><td>R$ 405.000,00</td></tr>
			</table></body></html>`)
		case "/licitacoes":
			fmt.Fprint(w, `<html><body><table>
				<tr><th>Número</th><th>Modalidade</th><th>Objeto</th><th>Situação</th><th>Data Abertura</th><th>Valor Estimado</th></tr>
				<tr><td>PE-01/2025</td><td>Pregão Eletrônico</td><td>Aquisição de medicamentos</td><td>Homologada</td><td>05/02/2025</td><td>R$ 120.000,00</td></tr>
				<tr><td>PE-02/2025</td><td>Pregão Eletrônico</td><td>Merenda escolar</td><td>Em andamento</td><td>08/02/2025</td><td>R$ 95.000,00</td></tr>
				<tr><td>CC-03/2025</td><td>Concorrência</td><td>Pavimentação rua central</td><td>Publicada</td><td>12/02/2025</td><td>R$ 350.000,00</td></tr>
				<tr><td>DL-04/2025</td><td>Dispensa</td><td>Manutenção ambulância</td><td>Concluída</td><td>14/02/2025</td><td>R$ 18.000,00</td></tr>
				<tr><td>PE-05/2025</td><td>Pregão Eletrônico</td><td>Equipamentos TI</td><td>Homologada</td><td>19/02/2025</td><td>R$ 65.000,00</td></tr>
			</table></body></html>`)
		case "/contratos":
			fmt.Fprint(w, `<html><body><table>
				<tr><th>Número</th><th>Fornecedor</th><th>CPF/CNPJ</th><th>Objeto</th><th>Assinatura</th><th>Valor</th></tr>
				<tr><td>CT-01/2025</td><td>EMPRESA A LTDA</td><td>12.345.678/0001-99</td><td>Limpeza urbana</td><td>10/01/2025</td><td>R$ 240.000,00</td></tr>
				<tr><td>CT-02/2025</td><td>PEDRO PRESTADOR</td><td>123.456.789-01</td><td>Consultoria tecnica</td><td>12/01/2025</td><td>R$ 18.000,00</td></tr>
				<tr><td>CT-03/2025</td><td>EMPRESA B SA</td><td>98.765.432/0001-10</td><td>Transporte escolar</td><td>15/01/2025</td><td>R$ 180.000,00</td></tr>
				<tr><td>CT-04/2025</td><td>EMPRESA C LTDA</td><td>11.222.333/0001-44</td><td>Iluminacao publica</td><td>18/01/2025</td><td>R$ 90.000,00</td></tr>
				<tr><td>CT-05/2025</td><td>EMPRESA D LTDA</td><td>22.333.444/0001-55</td><td>Software gestao</td><td>20/01/2025</td><td>R$ 60.000,00</td></tr>
			</table></body></html>`)
		case "/diarias":
			fmt.Fprint(w, `<html><body><table>
				<tr><th>Beneficiário</th><th>Cargo</th><th>Destino</th><th>Data</th><th>Motivo</th><th>Valor</th></tr>
				<tr><td>MARCOS MOTORISTA</td><td>Motorista</td><td>Capital</td><td>02/03/2025</td><td>Transporte paciente</td><td>R$ 350,00</td></tr>
				<tr><td>LUCIA TECNICA</td><td>Tecnica</td><td>Brasilia</td><td>05/03/2025</td><td>Capacitacao FNDE</td><td>R$ 900,00</td></tr>
				<tr><td>PAULO SECRETARIO</td><td>Secretario</td><td>Capital</td><td>08/03/2025</td><td>Reuniao TCE</td><td>R$ 450,00</td></tr>
				<tr><td>ROSA ENFERMEIRA</td><td>Enfermeira</td><td>Regional</td><td>10/03/2025</td><td>Seminario SUS</td><td>R$ 280,00</td></tr>
				<tr><td>TIAGO FISCAL</td><td>Fiscal</td><td>Distrito</td><td>12/03/2025</td><td>Vistoria obra</td><td>R$ 180,00</td></tr>
			</table></body></html>`)
		case "/obras":
			fmt.Fprint(w, `<html><body><table>
				<tr><th>Código</th><th>Título</th><th>Situação</th><th>Início</th><th>Empresa</th><th>Valor</th></tr>
				<tr><td>OB-01</td><td>Creche Municipal Norte</td><td>Em andamento</td><td>01/02/2025</td><td>Construtora A</td><td>R$ 1.200.000,00</td></tr>
				<tr><td>OB-02</td><td>Reforma UBS Centro</td><td>Concluída</td><td>10/01/2025</td><td>Construtora B</td><td>R$ 450.000,00</td></tr>
				<tr><td>OB-03</td><td>Pavimentação Av Brasil</td><td>Em andamento</td><td>15/02/2025</td><td>Construtora C</td><td>R$ 850.000,00</td></tr>
				<tr><td>OB-04</td><td>Praça da Juventude</td><td>Licitada</td><td>01/03/2025</td><td>Construtora D</td><td>R$ 320.000,00</td></tr>
				<tr><td>OB-05</td><td>Quadra Escolar Sul</td><td>Em andamento</td><td>20/01/2025</td><td>Construtora E</td><td>R$ 280.000,00</td></tr>
			</table></body></html>`)
		case "/convenios":
			fmt.Fprint(w, `<html><body><table>
				<tr><th>Número</th><th>Concedente</th><th>Descrição</th><th>Vigência</th><th>Valor</th><th>Valor Liberado</th></tr>
				<tr><td>CV-01/2025</td><td>FNDE</td><td>Ônibus escolar</td><td>31/12/2025</td><td>R$ 350.000,00</td><td>R$ 350.000,00</td></tr>
				<tr><td>CV-02/2025</td><td>Ministério Saúde</td><td>Equipamentos UBS</td><td>31/12/2025</td><td>R$ 200.000,00</td><td>R$ 100.000,00</td></tr>
				<tr><td>CV-03/2025</td><td>Estado</td><td>Pavimentação vicinal</td><td>30/06/2026</td><td>R$ 500.000,00</td><td>R$ 250.000,00</td></tr>
				<tr><td>CV-04/2025</td><td>MAPA</td><td>Patrulha mecanizada</td><td>31/12/2025</td><td>R$ 420.000,00</td><td>R$ 420.000,00</td></tr>
				<tr><td>CV-05/2025</td><td>Cultura</td><td>Biblioteca pública</td><td>31/12/2025</td><td>R$ 120.000,00</td><td>R$ 60.000,00</td></tr>
			</table></body></html>`)
		case "/legislacao":
			fmt.Fprint(w, `<html><body><table>
				<tr><th>Número</th><th>Tipo</th><th>Ano</th><th>Data</th><th>Descrição</th></tr>
				<tr><td>101</td><td>Lei Municipal</td><td>2025</td><td>10/01/2025</td><td>Lei Orçamentária Anual</td></tr>
				<tr><td>102</td><td>Decreto</td><td>2025</td><td>15/01/2025</td><td>Regulamenta pregão</td></tr>
				<tr><td>103</td><td>Portaria</td><td>2025</td><td>20/01/2025</td><td>Nomeia comissão licitação</td></tr>
				<tr><td>104</td><td>Lei Complementar</td><td>2025</td><td>01/02/2025</td><td>Plano diretor urbano</td></tr>
				<tr><td>105</td><td>Decreto</td><td>2025</td><td>10/02/2025</td><td>Calendário fiscal IPTU</td></tr>
			</table></body></html>`)
		case "/concursos":
			fmt.Fprint(w, `<html><body><table>
				<tr><th>Nome</th><th>Cargo</th><th>Classificação</th><th>Situação</th><th>Edital</th></tr>
				<tr><td>ALICE MENDES</td><td>Enfermeira</td><td>1</td><td>Convocado</td><td>01/2024</td></tr>
				<tr><td>BERNARDO SILVA</td><td>Professor</td><td>2</td><td>Aprovado</td><td>01/2024</td></tr>
				<tr><td>CAMILA NUNES</td><td>Médica</td><td>1</td><td>Nomeado</td><td>02/2024</td></tr>
				<tr><td>DIEGO ALVES</td><td>Motorista</td><td>3</td><td>Aprovado</td><td>02/2024</td></tr>
				<tr><td>EDUARDA LOPES</td><td>Contadora</td><td>1</td><td>Convocado</td><td>03/2024</td></tr>
			</table></body></html>`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestAll22HTMLPortalFamiliesAll10ModulesMatrix(t *testing.T) {
	eng, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}
	srv := newFull10ModuleHTMLServer()
	defer srv.Close()

	hub := NewHub(eng)
	hub.HTTPClient = srv.Client()

	m, err := eng.GetMunicipioByIBGE("3550308")
	if err != nil {
		t.Fatalf("GetMunicipioByIBGE failed: %v", err)
	}

	ctx := context.Background()
	opts := QueryOptions{UF: m.UF, Limit: 2, Timeout: 5 * time.Second}

	for _, fam := range all22HTMLPortalFamilies {
		fam := fam
		t.Run(fam, func(t *testing.T) {
			if !IsHTMLPortalPlatform(fam) {
				t.Fatalf("IsHTMLPortalPlatform(%q) == false", fam)
			}
			disc := &PortalDiscoveryResult{
				Municipio:       m,
				URLResolvida:    srv.URL,
				Plataforma:      fam,
				FonteDescoberta: "portal_html_catalog",
			}

			serv, err := hub.queryServidoresWithDiscovery(ctx, m, disc, opts)
			if err != nil || !serv.Disponivel || len(serv.Servidores) != 2 || serv.TotalRegistros != 5 {
				t.Fatalf("servidores(%s): err=%v disp=%v len=%d total=%d", fam, err, serv.Disponivel, len(serv.Servidores), serv.TotalRegistros)
			}
			if serv.Servidores[0].CPFParcial != "***.456.789-**" {
				t.Errorf("servidores(%s): CPF not masked properly: %q", fam, serv.Servidores[0].CPFParcial)
			}

			desp, err := hub.queryDespesasWithDiscovery(ctx, m, disc, opts)
			if err != nil || !desp.Disponivel || len(desp.Despesas) != 2 || desp.TotalRegistros != 5 {
				t.Fatalf("despesas(%s): err=%v disp=%v len=%d total=%d", fam, err, desp.Disponivel, len(desp.Despesas), desp.TotalRegistros)
			}
			if desp.Despesas[0].DocumentoFornecedor != "12.345.678/0001-99" {
				t.Errorf("despesas(%s): CNPJ altered: %q", fam, desp.Despesas[0].DocumentoFornecedor)
			}
			if desp.Despesas[1].DocumentoFornecedor != "***.456.789-**" {
				t.Errorf("despesas(%s): CPF in Despesas not masked: %q", fam, desp.Despesas[1].DocumentoFornecedor)
			}

			rec, err := hub.queryReceitasWithDiscovery(ctx, m, disc, opts)
			if err != nil || !rec.Disponivel || len(rec.Receitas) != 2 || rec.TotalRegistros != 5 {
				t.Fatalf("receitas(%s): err=%v disp=%v len=%d total=%d", fam, err, rec.Disponivel, len(rec.Receitas), rec.TotalRegistros)
			}

			lic, err := hub.queryLicitacoesWithDiscovery(ctx, m, disc, opts)
			if err != nil || !lic.Disponivel || len(lic.Licitacoes) != 2 || lic.TotalRegistros != 5 {
				t.Fatalf("licitacoes(%s): err=%v disp=%v len=%d total=%d", fam, err, lic.Disponivel, len(lic.Licitacoes), lic.TotalRegistros)
			}

			cont, err := hub.queryContratosWithDiscovery(ctx, m, disc, opts)
			if err != nil || !cont.Disponivel || len(cont.Contratos) != 2 || cont.TotalRegistros != 5 {
				t.Fatalf("contratos(%s): err=%v disp=%v len=%d total=%d", fam, err, cont.Disponivel, len(cont.Contratos), cont.TotalRegistros)
			}
			if cont.Contratos[0].DocumentoFornecedor != "12.345.678/0001-99" {
				t.Errorf("contratos(%s): CNPJ altered: %q", fam, cont.Contratos[0].DocumentoFornecedor)
			}
			if cont.Contratos[1].DocumentoFornecedor != "***.456.789-**" {
				t.Errorf("contratos(%s): CPF not masked: %q", fam, cont.Contratos[1].DocumentoFornecedor)
			}

			diar, err := hub.queryDiariasWithDiscovery(ctx, m, disc, opts)
			if err != nil || !diar.Disponivel || len(diar.Diarias) != 2 || diar.TotalRegistros != 5 {
				t.Fatalf("diarias(%s): err=%v disp=%v len=%d total=%d", fam, err, diar.Disponivel, len(diar.Diarias), diar.TotalRegistros)
			}

			obr, err := hub.queryObrasWithDiscovery(ctx, m, disc, opts)
			if err != nil || !obr.Disponivel || len(obr.Obras) != 2 || obr.TotalRegistros != 5 {
				t.Fatalf("obras(%s): err=%v disp=%v len=%d total=%d", fam, err, obr.Disponivel, len(obr.Obras), obr.TotalRegistros)
			}

			conv, err := hub.queryConveniosWithDiscovery(ctx, m, disc, opts)
			if err != nil || !conv.Disponivel || len(conv.Convenios) != 2 || conv.TotalRegistros != 5 {
				t.Fatalf("convenios(%s): err=%v disp=%v len=%d total=%d", fam, err, conv.Disponivel, len(conv.Convenios), conv.TotalRegistros)
			}

			leg, err := hub.queryLegislacaoWithDiscovery(ctx, m, disc, opts)
			if err != nil || !leg.Disponivel || len(leg.Atos) != 2 || leg.TotalRegistros != 5 {
				t.Fatalf("legislacao(%s): err=%v disp=%v len=%d total=%d", fam, err, leg.Disponivel, len(leg.Atos), leg.TotalRegistros)
			}

			conc, err := hub.queryConcursosWithDiscovery(ctx, m, disc, opts)
			if err != nil || !conc.Disponivel || len(conc.Aprovados) != 2 || conc.TotalRegistros != 5 {
				t.Fatalf("concursos(%s): err=%v disp=%v len=%d total=%d", fam, err, conc.Disponivel, len(conc.Aprovados), conc.TotalRegistros)
			}
		})
	}
}

func TestHTMLPortalEncodingIframeAndSafetyRegressions(t *testing.T) {
	// 1. ISO-8859-1 / Windows-1252 body decoding & UTF-8 BOM stripping
	latin1HTML := []byte("<html><body><table>" +
		"<tr><th>Nome</th><th>Fun\xe7\xe3o</th><th>\xd3rg\xe3o</th><th>Sal\xe1rio Bruto</th><th>Sal\xe1rio L\xedquido</th></tr>" +
		"<tr><td>JO\xc3O PEREIRA</td><td>T\xe9cnico \x93S\xeanior\x94</td><td>Sa\xfade</td><td>R$ 4.500,00</td><td>R$ 3.800,00</td></tr>" +
		"</table></body></html>")

	latin1Srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=ISO-8859-1")
		_, _ = w.Write(latin1HTML)
	}))
	defer latin1Srv.Close()

	m := &Municipio{CodigoIBGE: "3550308", Nome: "São Paulo", UF: "SP"}
	res, err := QueryHTMLPortalServidores(context.Background(), latin1Srv.Client(), m, PortalQueryParams{BaseURL: latin1Srv.URL, Limit: 10}, "html_portal")
	if err != nil || !res.Disponivel || len(res.Servidores) != 1 {
		t.Fatalf("Latin-1 HTML extraction failed: err=%v res=%+v", err, res)
	}
	if res.Servidores[0].Nome != "JOÃO PEREIRA" {
		t.Errorf("expected decoded UTF-8 name 'JOÃO PEREIRA', got %q", res.Servidores[0].Nome)
	}
	if res.Servidores[0].SalarioBruto != 4500.0 || res.Servidores[0].SalarioLiquido != 3800.0 {
		t.Errorf("unexpected decoded salaries: bruto=%v liq=%v", res.Servidores[0].SalarioBruto, res.Servidores[0].SalarioLiquido)
	}

	// UTF-8 BOM stripping
	decodedBOM := decodeWebBody([]byte("\xef\xbb\xbf<html>Licitação</html>"), "text/html; charset=utf-8")
	if strings.HasPrefix(decodedBOM, "\ufeff") || !strings.Contains(decodedBOM, "Licitação") {
		t.Errorf("UTF-8 BOM not stripped cleanly: %q", decodedBOM)
	}

	// 2. Iframe scoring regression (link.Label starts with "iframe") & 2-hop iframe extraction
	iframeScore := scoreHTMLLink(htmlLink{URL: "http://127.0.0.1/embed/folha", Label: "iframe Folha Servidores"}, "127.0.0.1", htmlDomainKeywords("servidores"))
	plainScore := scoreHTMLLink(htmlLink{URL: "http://127.0.0.1/embed/folha", Label: "Folha Servidores"}, "127.0.0.1", htmlDomainKeywords("servidores"))
	if iframeScore <= plainScore {
		t.Errorf("expected iframe label bonus in scoreHTMLLink (iframe=%d, plain=%d)", iframeScore, plainScore)
	}

	iframeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/embed/folha":
			fmt.Fprint(w, `<html><body><table>
				<tr><th>Servidor</th><th>Cargo</th><th>Remuneração</th></tr>
				<tr><td>LUCAS IFRAME</td><td>AUDITOR</td><td>R$ 9.800,00</td></tr>
			</table></body></html>`)
		default:
			fmt.Fprint(w, `<html><body><h1>Transparência</h1><iframe title="Folha de Servidores" src="/embed/folha"></iframe></body></html>`)
		}
	}))
	defer iframeSrv.Close()

	resIframe, err := QueryHTMLPortalServidores(context.Background(), iframeSrv.Client(), m, PortalQueryParams{BaseURL: iframeSrv.URL, Limit: 5}, "html_portal")
	if err != nil || !resIframe.Disponivel || len(resIframe.Servidores) != 1 || resIframe.Servidores[0].Nome != "LUCAS IFRAME" {
		t.Fatalf("iframe extraction failed: err=%v res=%+v", err, resIframe)
	}

	// 3. Unquoted type=password and SafeLine form safety checks
	unsafeForms := []struct {
		attrs string
		body  string
	}{
		{`action="/consulta"`, `<input type=password name=senha><button>Consultar</button>`},
		{`action="/consulta"`, `<input type="password" name="p">`},
		{`action="/verify" class="safeline-challenge"`, `<input name="sl-challenge">`},
		{`action="/captcha"`, `<div class="h-captcha"></div>`},
	}
	for _, fm := range unsafeForms {
		if !isUnsafeOrLoginForm(fm.attrs, fm.body) {
			t.Errorf("isUnsafeOrLoginForm should reject: attrs=%s body=%s", fm.attrs, fm.body)
		}
	}

	// 4. Anti-bot / WAF detection across status codes (including HTTP 200 challenge pages)
	wafCases := []struct {
		name   string
		status int
		body   string
		want   bool
	}{
		{"cf-403", 403, "<html>just a moment... cf-turnstile</html>", true},
		{"hcaptcha-429", 429, "<html>please solve h-captcha</html>", true},
		{"safeline-468", 468, "<html>safeline sl-challenge</html>", true},
		{"cf-503", 503, "<html>checking your browser cloudflare</html>", true},
		{"cf-200-interstitial", 200, "<html>just a moment... enable javascript and cookies to continue</html>", true},
		{"normal-200-table", 200, "<html><body><table><tr><th>Nome</th></tr><tr><td>Ana</td></tr></table><!-- cloudflare footer --></body></html>", false},
	}
	for _, tc := range wafCases {
		got := isAntiBotResponse(tc.status, strings.ToLower(tc.body))
		if got != tc.want {
			t.Errorf("isAntiBotResponse(%s, status=%d) = %v, want %v", tc.name, tc.status, got, tc.want)
		}
	}

	// 5. htmlFallbackTargets must NOT re-fetch anti_bot_protegido resolved URL
	eng, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}
	hub := NewHub(eng)
	hub.HTTPClient = &http.Client{Transport: &failingRoundTripper{}, Timeout: 1 * time.Second}
	antiBotDisc := &PortalDiscoveryResult{
		Municipio:    &Municipio{CodigoIBGE: "9999999", Nome: "FakeCity", UF: "SP", Slug: "fakecity"},
		URLResolvida: "https://blocked.example.gov.br",
		Plataforma:   "anti_bot_protegido",
	}
	targets := hub.htmlFallbackTargets(context.Background(), antiBotDisc.Municipio, antiBotDisc, "")
	for _, tgt := range targets {
		if tgt == antiBotDisc.URLResolvida {
			t.Errorf("htmlFallbackTargets re-added anti_bot_protegido URL %q", tgt)
		}
	}
}

func TestHubConcurrentTransparenciaCompletaRaceRegression(t *testing.T) {
	eng, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	// Servidor que retorna página HTML sem tabelas (força os branches que antes mutavam disc.URLResolvida / disc.Observacoes!)
	emptySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<html><body><h1>Portal Municipal</h1><p>Consulte pessoalmente.</p></body></html>`)
	}))
	defer emptySrv.Close()

	// Servidor SICONFI & PNCP mock local para evitar chamadas externas durante QueryTransparenciaCompleta
	fedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"items":[],"data":[]}`)
	}))
	defer fedSrv.Close()

	hub := NewHub(eng)
	hub.HTTPClient = emptySrv.Client()
	hub.Siconfi = NewSiconfiClientWithBaseURL(fedSrv.URL, fedSrv.Client())
	hub.PNCP = NewPNCPClientWithBaseURL(fedSrv.URL, fedSrv.Client())
	hub.TCE.client = fedSrv.Client()
	hub.TCE.tceSPBaseURL = fedSrv.URL
	hub.TCE.tcePIBaseURL = fedSrv.URL
	hub.TCE.tceCEBaseURL = fedSrv.URL

	// Executa QueryTransparenciaCompleta apontando para o servidor HTML vazio com OverrideURL.
	// Sob `go test -race`, qualquer mutação concorrente de `*PortalDiscoveryResult` falharia imediatamente.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	report, err := hub.QueryTransparenciaCompleta(ctx, "3550308", QueryOptions{
		UF:          "SP",
		OverrideURL: emptySrv.URL,
		Ano:         2025,
		Limit:       5,
		Timeout:     3 * time.Second,
	})
	if err != nil {
		t.Fatalf("QueryTransparenciaCompleta failed: %v", err)
	}
	if report == nil || report.Dossier == nil || report.Discovery == nil {
		t.Fatalf("incomplete report: %+v", report)
	}
}

func TestHubDespesasReceitasUnavailableFallbackChain(t *testing.T) {
	eng, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	// Servidor 1: API estruturada retorna 200 mas sem dados (Disponivel: false)
	emptyAPISrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[]`)
	}))
	defer emptyAPISrv.Close()

	// Servidor 2: TCE-SP mock retorna despesas e receitas válidas
	tceSPSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/municipios"):
			fmt.Fprint(w, `[{"municipio":"sao-paulo","municipio_extenso":"São Paulo"}]`)
		case strings.Contains(r.URL.Path, "/despesas/"):
			fmt.Fprint(w, `[{"nr_empenho":"EMP-99","nm_fornecedor":"FORNECEDOR TCE","vl_despesa":"1500,50","dt_emissao_despesa":"10/01/2025","cpf_cnpj_fornecedor":"12345678901"}]`)
		case strings.Contains(r.URL.Path, "/receitas/"):
			fmt.Fprint(w, `[{"ds_alinea":"IPTU","ds_fonte_recurso":"TESOURO","vl_arrecadacao":"9000,00","mes_ref_extenso":"Janeiro"}]`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer tceSPSrv.Close()

	hub := NewHub(eng)
	hub.HTTPClient = emptyAPISrv.Client()
	hub.TCE.client = tceSPSrv.Client()
	hub.TCE.tceSPBaseURL = tceSPSrv.URL

	m, err := eng.GetMunicipioByIBGE("3550308")
	if err != nil {
		t.Fatalf("GetMunicipioByIBGE failed: %v", err)
	}

	disc := &PortalDiscoveryResult{
		Municipio:    m,
		URLResolvida: emptyAPISrv.URL,
		Plataforma:   "quasar_publico",
	}

	desp, err := hub.queryDespesasWithDiscovery(context.Background(), m, disc, QueryOptions{UF: "SP", Ano: 2025, Mes: 1, Limit: 5})
	if err != nil || desp == nil || !desp.Disponivel || len(desp.Despesas) == 0 {
		t.Fatalf("expected despesas fallback to TCE-SP when primary returned unavailable: err=%v res=%+v", err, desp)
	}
	if !strings.Contains(strings.ToUpper(desp.Fonte), "TCE-SP") {
		t.Errorf("expected TCE-SP fallback source, got %q", desp.Fonte)
	}

	rec, err := hub.queryReceitasWithDiscovery(context.Background(), m, disc, QueryOptions{UF: "SP", Ano: 2025, Mes: 1, Limit: 5})
	if err != nil || rec == nil || !rec.Disponivel || len(rec.Receitas) == 0 {
		t.Fatalf("expected receitas fallback to TCE-SP when primary returned unavailable: err=%v res=%+v", err, rec)
	}
	if !strings.Contains(strings.ToUpper(rec.Fonte), "TCE-SP") {
		t.Errorf("expected TCE-SP fallback source, got %q", rec.Fonte)
	}
}

func TestAll30APIPlatformsHubDispatchAndResilience(t *testing.T) {
	eng, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	// Servidor local que responde 500 para testar que todas as 30 plataformas degradam
	// graciosamente sem panic, sem race e dentro do timeout.
	errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"simulated upstream failure"}`)
	}))
	defer errSrv.Close()

	hub := NewHub(eng)
	hub.HTTPClient = errSrv.Client()
	hub.PNCP = NewPNCPClientWithBaseURL(errSrv.URL, errSrv.Client())
	hub.TCE.client = errSrv.Client()
	hub.TCE.tcePIBaseURL = errSrv.URL
	hub.TCE.tceCEBaseURL = errSrv.URL
	hub.TCE.tceSPBaseURL = errSrv.URL

	// Usar município de UF sem TCE JSON fallback automático extra (ex: AC) para manter o teste 100% local
	m, err := eng.GetMunicipioByIBGE("1200401") // Rio Branco/AC
	if err != nil {
		t.Fatalf("GetMunicipioByIBGE failed: %v", err)
	}

	platforms := make([]string, 0, len(knownCatalogAPIPlatforms))
	for p := range knownCatalogAPIPlatforms {
		platforms = append(platforms, p)
	}
	sort.Strings(platforms)

	if len(platforms) != 30 {
		t.Fatalf("expected 30 API platforms in matrix, got %d", len(platforms))
	}

	opts := QueryOptions{UF: m.UF, Ano: 2025, Mes: 1, Limit: 3, Timeout: 2 * time.Second}
	for _, plat := range platforms {
		plat := plat
		t.Run(plat, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			disc := &PortalDiscoveryResult{
				Municipio:        m,
				URLResolvida:     errSrv.URL,
				Plataforma:       plat,
				CodAdministracao: 1,
				TenantID:         "1",
			}
			if _, err := hub.queryServidoresWithDiscovery(ctx, m, disc, opts); err != nil {
				t.Errorf("platform %s servidores returned error instead of graceful result: %v", plat, err)
			}
			if _, err := hub.queryDespesasWithDiscovery(ctx, m, disc, opts); err != nil {
				t.Errorf("platform %s despesas returned error instead of graceful result: %v", plat, err)
			}
			if _, err := hub.queryReceitasWithDiscovery(ctx, m, disc, opts); err != nil {
				t.Errorf("platform %s receitas returned error instead of graceful result: %v", plat, err)
			}
			if _, err := hub.queryLicitacoesWithDiscovery(ctx, m, disc, opts); err != nil {
				t.Errorf("platform %s licitacoes returned error instead of graceful result: %v", plat, err)
			}
			if _, err := hub.queryContratosWithDiscovery(ctx, m, disc, opts); err != nil {
				t.Errorf("platform %s contratos returned error instead of graceful result: %v", plat, err)
			}
		})
	}
}

func TestLGPDCPFMaskingAllConnectorsRegression(t *testing.T) {
	// 1. Helpers unitários: CPF mascarado, CNPJ preservado
	cpfCases := []string{"12345678901", "123.456.789-01"}
	for _, raw := range cpfCases {
		if got := maskIfCPF(raw); got != "***.456.789-**" {
			t.Errorf("maskIfCPF(%q) = %q, want ***.456.789-**", raw, got)
		}
		if got := maskCPF("", raw); got != "***.456.789-**" {
			t.Errorf("maskCPF('', %q) = %q, want ***.456.789-**", raw, got)
		}
	}
	cnpjCases := []string{"12345678000199", "12.345.678/0001-99"}
	for _, raw := range cnpjCases {
		if got := maskIfCPF(raw); got != raw {
			t.Errorf("maskIfCPF altered CNPJ %q -> %q", raw, got)
		}
	}

	// 2. PNCP Contratos: CPF pessoa física mascarado, CNPJ empresa preservado
	pncpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"totalRegistros": 2,
			"data": [
				{"numeroContratoEmpenho":"01/2025","nomeRazaoSocialFornecedor":"JOAO PESSOA FISICA","niFornecedor":"12345678901","objetoContrato":"Consultoria","valorGlobal":5000},
				{"numeroContratoEmpenho":"02/2025","nomeRazaoSocialFornecedor":"EMPRESA PJ LTDA","niFornecedor":"12345678000199","objetoContrato":"Obra","valorGlobal":50000}
			]
		}`)
	}))
	defer pncpSrv.Close()

	pncp := NewPNCPClientWithBaseURL(pncpSrv.URL, pncpSrv.Client())
	m := &Municipio{CodigoIBGE: "3550308", Nome: "São Paulo", UF: "SP", CNPJ: "46395000000139"}
	contRes, err := pncp.FetchContratosByCNPJ(context.Background(), m, "2025-01-01", "2025-12-31", 2025, 10)
	if err != nil || len(contRes.Contratos) != 2 {
		t.Fatalf("PNCP FetchContratos failed: err=%v res=%+v", err, contRes)
	}
	if contRes.Contratos[0].DocumentoFornecedor != "***.456.789-**" {
		t.Errorf("PNCP leaked raw CPF: %q", contRes.Contratos[0].DocumentoFornecedor)
	}
	if contRes.Contratos[1].DocumentoFornecedor != "12345678000199" {
		t.Errorf("PNCP altered CNPJ: %q", contRes.Contratos[1].DocumentoFornecedor)
	}

	// 3. TCE-PI Servidores: CPF mascarado
	tcePISrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "/prefeituras"):
			fmt.Fprint(w, `[{"id":77,"nome":"Teresina","codIBGE":"2211001"}]`)
		case strings.Contains(r.URL.Path, "/servidores"):
			fmt.Fprint(w, `[{"nome":"MARIA PI","cpf":"12345678901","cargo":"PROFESSORA","remuneracao":4500.0}]`)
		default:
			fmt.Fprint(w, `[]`)
		}
	}))
	defer tcePISrv.Close()

	tce := NewTCEClient(tcePISrv.Client())
	tce.tcePIBaseURL = tcePISrv.URL
	mPI := &Municipio{CodigoIBGE: "2211001", Nome: "Teresina", UF: "PI"}
	piRes, err := tce.FetchTCEPIServidores(context.Background(), mPI, 2025, "", 10)
	if err != nil || len(piRes.Servidores) != 1 {
		t.Fatalf("TCE-PI FetchTCEPIServidores failed: err=%v res=%+v", err, piRes)
	}
	if piRes.Servidores[0].CPFParcial != "***.456.789-**" {
		t.Errorf("TCE-PI leaked raw CPF: %q", piRes.Servidores[0].CPFParcial)
	}
}
