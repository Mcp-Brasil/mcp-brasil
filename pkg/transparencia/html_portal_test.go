package transparencia_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Mcp-Brasil/mcp-brasil/pkg/transparencia"
)

func TestPortaisHTMLCatalogAndDiscovery(t *testing.T) {
	eng, err := transparencia.NewEngine()
	if err != nil {
		t.Fatalf("NewEngine erro: %v", err)
	}

	all := eng.ListPortaisHTMLMunicipios("")
	if len(all) < 2400 {
		t.Fatalf("esperava >= 2400 portais HTML no catálogo, obteve %d", len(all))
	}

	// Santa Luzia (MG) — portal WordPress HTML sem API estruturada
	cat, ok := eng.GetPortalHTMLMunicipio("3157807")
	if !ok || !transparencia.IsHTMLPortalPlatform(cat.Plataforma) || cat.PortalURL == "" {
		t.Fatalf("esperava Santa Luzia (3157807) no catálogo HTML, obteve ok=%v cat=%+v", ok, cat)
	}

	munis := eng.SearchMunicipios("3157807", "MG", 1)
	if len(munis) == 0 {
		t.Fatalf("município 3157807 não encontrado")
	}
	m := munis[0]
	disc := transparencia.DiscoverPortal(context.Background(), m, transparencia.DiscoverOptions{Timeout: 2 * time.Second})
	if disc == nil || disc.FonteDescoberta != "portal_html_catalog" || !transparencia.IsHTMLPortalPlatform(disc.Plataforma) {
		t.Fatalf("DiscoverPortal inesperado para portal HTML catalogado: %+v", disc)
	}
}

func TestHTMLPortalAllQueriesMock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "3157807", Nome: "Santa Luzia", UF: "MG"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/":
			_, _ = w.Write([]byte(`<!doctype html><html><body>
				<h1>Portal da Transparência</h1>
				<nav>
					<a href="/transparencia/servidores">Servidores e Folha</a>
					<a href="/transparencia/despesas">Despesas e Empenhos</a>
					<a href="/transparencia/receitas">Receitas Arrecadadas</a>
					<a href="/transparencia/licitacoes">Licitações e Editais</a>
					<a href="/transparencia/contratos">Contratos</a>
					<a href="/transparencia/diarias">Diárias e Passagens</a>
				</nav>
			</body></html>`))
		case "/transparencia/servidores":
			_, _ = w.Write([]byte(`<html><body><table>
				<thead><tr><th>Nome</th><th>Matrícula</th><th>CPF</th><th>Cargo</th><th>Lotação</th><th>Salário Bruto</th><th>Salário Líquido</th><th>Competência</th></tr></thead>
				<tbody>
					<tr><td>MARIA APARECIDA</td><td>101</td><td>123.456.789-01</td><td>Professora</td><td>Educação</td><td>R$ 5.432,10</td><td>R$ 4.300,00</td><td>03/2025</td></tr>
				</tbody>
			</table></body></html>`))
		case "/transparencia/despesas":
			_, _ = w.Write([]byte(`<html><body><table>
				<tr><th>Empenho</th><th>Data</th><th>Fornecedor</th><th>CPF/CNPJ</th><th>Histórico</th><th>Valor Empenhado</th><th>Valor Pago</th></tr>
				<tr><td>2025/44</td><td>10/03/2025</td><td>JOAO PRESTADOR</td><td>98765432100</td><td>Serviços técnicos</td><td>R$ 2.500,00</td><td>R$ 2.500,00</td></tr>
				<tr><td>2025/45</td><td>11/03/2025</td><td>ENERGIA MG SA</td><td>11.222.333/0001-44</td><td>Energia elétrica</td><td>R$ 12.000,00</td><td>R$ 12.000,00</td></tr>
			</table></body></html>`))
		case "/transparencia/receitas":
			_, _ = w.Write([]byte(`<html><body><table>
				<tr><th>Código</th><th>Descrição</th><th>Categoria</th><th>Valor Previsto</th><th>Valor Arrecadado</th></tr>
				<tr><td>1.1.1.2</td><td>IPTU</td><td>Tributária</td><td>R$ 1.000.000,00</td><td>R$ 650.000,50</td></tr>
			</table></body></html>`))
		case "/transparencia/licitacoes":
			_, _ = w.Write([]byte(`<html><body><table>
				<tr><th>Número</th><th>Modalidade</th><th>Objeto</th><th>Situação</th><th>Data Abertura</th><th>Valor Estimado</th></tr>
				<tr><td><a href="/arquivos/edital_01_2025.pdf">01/2025</a></td><td>Pregão Eletrônico</td><td>Aquisição de merenda</td><td>Homologada</td><td>05/02/2025</td><td>R$ 180.000,00</td></tr>
			</table></body></html>`))
		case "/transparencia/contratos":
			_, _ = w.Write([]byte(`<html><body><table>
				<tr><th>Contrato</th><th>Fornecedor</th><th>CPF/CNPJ</th><th>Objeto</th><th>Valor</th></tr>
				<tr><td>CT-12/2025</td><td>CONSTRUTORA SUL</td><td>22.333.444/0001-55</td><td>Reforma escolar</td><td>R$ 250.000,00</td></tr>
			</table></body></html>`))
		case "/transparencia/diarias":
			_, _ = w.Write([]byte(`<html><body><table>
				<tr><th>Beneficiário</th><th>Cargo</th><th>Destino</th><th>Motivo</th><th>Período</th><th>Valor</th></tr>
				<tr><td>CARLOS SOUZA</td><td>Secretário</td><td>Belo Horizonte</td><td>Reunião técnica</td><td>12/03/2025</td><td>R$ 450,00</td></tr>
			</table></body></html>`))
		case "/waf":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`<html><title>Attention Required! | Cloudflare</title></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	p := transparencia.PortalQueryParams{BaseURL: srv.URL, Ano: 2025, Limit: 10}

	serv, err := transparencia.QueryHTMLPortalServidores(ctx, client, m, p, "cms_wordpress_html")
	if err != nil || !serv.Disponivel || len(serv.Servidores) != 1 {
		t.Fatalf("QueryHTMLPortalServidores inesperado: err=%v res=%+v", err, serv)
	}
	if serv.Servidores[0].CPFParcial != "***.456.789-**" || serv.Servidores[0].SalarioBruto != 5432.10 {
		t.Errorf("servidor HTML inesperado: %+v", serv.Servidores[0])
	}

	desp, err := transparencia.QueryHTMLPortalDespesas(ctx, client, m, p, "html_portal")
	if err != nil || !desp.Disponivel || len(desp.Despesas) != 2 {
		t.Fatalf("QueryHTMLPortalDespesas inesperado: err=%v res=%+v", err, desp)
	}
	if desp.Despesas[0].DocumentoFornecedor != "***.654.321-**" || desp.Despesas[1].DocumentoFornecedor != "11.222.333/0001-44" {
		t.Errorf("LGPD doc despesa HTML inesperado: %+v", desp.Despesas)
	}

	rec, err := transparencia.QueryHTMLPortalReceitas(ctx, client, m, p, "html_portal")
	if err != nil || !rec.Disponivel || len(rec.Receitas) != 1 || rec.Receitas[0].ValorArrecadado != 650000.50 {
		t.Fatalf("QueryHTMLPortalReceitas inesperado: err=%v res=%+v", err, rec)
	}

	lic, err := transparencia.QueryHTMLPortalLicitacoes(ctx, client, m, p, "html_portal")
	if err != nil || !lic.Disponivel || len(lic.Licitacoes) != 1 || !strings.HasSuffix(lic.Licitacoes[0].Link, "/arquivos/edital_01_2025.pdf") {
		t.Fatalf("QueryHTMLPortalLicitacoes inesperado: err=%v res=%+v", err, lic)
	}

	ct, err := transparencia.QueryHTMLPortalContratos(ctx, client, m, p, "html_portal")
	if err != nil || !ct.Disponivel || len(ct.Contratos) != 1 || ct.Contratos[0].Valor != 250000.00 {
		t.Fatalf("QueryHTMLPortalContratos inesperado: err=%v res=%+v", err, ct)
	}

	di, err := transparencia.QueryHTMLPortalDiarias(ctx, client, m, p, "html_portal")
	if err != nil || !di.Disponivel || len(di.Diarias) != 1 || di.Diarias[0].Valor != 450.00 {
		t.Fatalf("QueryHTMLPortalDiarias inesperado: err=%v res=%+v", err, di)
	}

	// Anti-bot nunca deve ser contornado
	if _, err := transparencia.QueryHTMLPortalServidores(ctx, client, m, transparencia.PortalQueryParams{BaseURL: srv.URL + "/waf"}, "html_portal"); err == nil {
		t.Fatalf("esperava erro em página protegida por anti-bot/WAF")
	}
}

func TestHubRoutingHTMLPortal(t *testing.T) {
	eng, err := transparencia.NewEngine()
	if err != nil {
		t.Fatalf("NewEngine erro: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/transparencia":
			_, _ = w.Write([]byte(`<html><body>
				<a href="/transparencia/servidores">Servidores</a>
				<a href="/transparencia/despesas">Despesas</a>
				<a href="/transparencia/receitas">Receitas</a>
				<a href="/transparencia/licitacoes">Licitações</a>
				<a href="/transparencia/contratos">Contratos</a>
				<a href="/transparencia/diarias">Diárias</a>
			</body></html>`))
		case "/transparencia/servidores":
			_, _ = w.Write([]byte(`<html><body><table>
				<tr><th>Nome</th><th>CPF</th><th>Cargo</th><th>Salário Bruto</th></tr>
				<tr><td>ANA HUB HTML</td><td>11122233344</td><td>Enfermeira</td><td>R$ 4.800,00</td></tr>
			</table></body></html>`))
		case "/transparencia/despesas":
			_, _ = w.Write([]byte(`<html><body><table>
				<tr><th>Empenho</th><th>Fornecedor</th><th>Valor Empenhado</th></tr>
				<tr><td>10/2025</td><td>POSTO HUB</td><td>R$ 950,00</td></tr>
			</table></body></html>`))
		case "/transparencia/receitas":
			_, _ = w.Write([]byte(`<html><body><table>
				<tr><th>Código</th><th>Descrição</th><th>Valor Arrecadado</th></tr>
				<tr><td>1.7.1</td><td>FPM</td><td>R$ 300.000,00</td></tr>
			</table></body></html>`))
		case "/transparencia/licitacoes":
			_, _ = w.Write([]byte(`<html><body><table>
				<tr><th>Número</th><th>Modalidade</th><th>Objeto</th><th>Valor Estimado</th></tr>
				<tr><td>03/2025</td><td>Dispensa</td><td>Material de limpeza</td><td>R$ 15.000,00</td></tr>
			</table></body></html>`))
		case "/transparencia/contratos":
			_, _ = w.Write([]byte(`<html><body><table>
				<tr><th>Contrato</th><th>Fornecedor</th><th>Valor</th></tr>
				<tr><td>CT-99/2025</td><td>TECH HUB</td><td>R$ 60.000,00</td></tr>
			</table></body></html>`))
		case "/transparencia/diarias":
			_, _ = w.Write([]byte(`<html><body><table>
				<tr><th>Beneficiário</th><th>Destino</th><th>Valor</th></tr>
				<tr><td>PEDRO HUB</td><td>Vitória</td><td>R$ 320,00</td></tr>
			</table></body></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	hub := newTestHub(eng, srv.Client())
	opts := transparencia.QueryOptions{UF: "MG", OverrideURL: srv.URL + "/transparencia", Ano: 2025, Limit: 5}

	serv, err := hub.QueryServidores(ctx, "Santa Luzia", opts)
	if err != nil || !serv.Disponivel || serv.Plataforma != "html_portal" || len(serv.Servidores) != 1 || serv.Servidores[0].CPFParcial != "***.222.333-**" {
		t.Fatalf("Hub html_portal servidores inesperado: err=%v res=%+v", err, serv)
	}

	desp, err := hub.QueryDespesas(ctx, "Santa Luzia", opts)
	if err != nil || !desp.Disponivel || desp.Plataforma != "html_portal" || len(desp.Despesas) != 1 || desp.Despesas[0].ValorEmpenhado != 950.00 {
		t.Fatalf("Hub html_portal despesas inesperado: err=%v res=%+v", err, desp)
	}

	rec, err := hub.QueryReceitas(ctx, "Santa Luzia", opts)
	if err != nil || !rec.Disponivel || rec.Plataforma != "html_portal" || len(rec.Receitas) != 1 || rec.Receitas[0].ValorArrecadado != 300000.00 {
		t.Fatalf("Hub html_portal receitas inesperado: err=%v res=%+v", err, rec)
	}

	lic, err := hub.QueryLicitacoes(ctx, "Santa Luzia", opts)
	if err != nil || !lic.Disponivel || lic.Plataforma != "html_portal" || len(lic.Licitacoes) != 1 || lic.Licitacoes[0].ValorEstimado != 15000.00 {
		t.Fatalf("Hub html_portal licitacoes inesperado: err=%v res=%+v", err, lic)
	}

	ct, err := hub.QueryContratos(ctx, "Santa Luzia", opts)
	if err != nil || !ct.Disponivel || ct.Plataforma != "html_portal" || len(ct.Contratos) != 1 || ct.Contratos[0].Valor != 60000.00 {
		t.Fatalf("Hub html_portal contratos inesperado: err=%v res=%+v", err, ct)
	}

	di, err := hub.QueryDiarias(ctx, "Santa Luzia", opts)
	if err != nil || !di.Disponivel || di.Plataforma != "html_portal" || len(di.Diarias) != 1 || di.Diarias[0].Valor != 320.00 {
		t.Fatalf("Hub html_portal diarias inesperado: err=%v res=%+v", err, di)
	}
}

func TestHTMLPortalScriptJSONAndXHRMock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "3157807", Nome: "Santa Luzia", UF: "MG"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/next-servidores":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!doctype html><html><body>
				<div id="__next"></div>
				<script id="__NEXT_DATA__" type="application/json">{"props":{"pageProps":{"servidores":[{"nome":"ANA NEXTJS","cpf":"123.456.789-01","cargo":"Engenheira","bruto":"9.800,50","liquido":"7.500,00"}]}}}</script>
			</body></html>`))
		case "/datatables-despesas":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!doctype html><html><body>
				<table id="tblDesp"><thead><tr><th>Empenho</th><th>Fornecedor</th><th>CPF/CNPJ</th><th>Valor Empenhado</th></tr></thead><tbody></tbody></table>
				<script>$('#tblDesp').DataTable({ ajax: { url: '/api/despesas-dt.json' } });</script>
			</body></html>`))
		case "/api/despesas-dt.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"draw":1,"recordsTotal":1,"aaData":[["2025/001","CONSTRUTORA BETA","11.222.333/0001-44","R$ 42.500,00"]]}`))
		case "/fetch-receitas":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!doctype html><html><body>
				<div id="app"></div>
				<script>fetch('/dados/receitas.json').then(r => r.json());</script>
			</body></html>`))
		case "/dados/receitas.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"dados":[{"codigo":"1.1.1","descricao":"IPTU","arrecadado":"85000.25","previsto":"90000.00"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	serv, err := transparencia.QueryHTMLPortalServidores(ctx, client, m, transparencia.PortalQueryParams{BaseURL: srv.URL + "/next-servidores", Limit: 5}, "cms_wordpress_html")
	if err != nil || !serv.Disponivel || len(serv.Servidores) != 1 || serv.Servidores[0].Nome != "ANA NEXTJS" || serv.Servidores[0].CPFParcial != "***.456.789-**" || serv.Servidores[0].SalarioBruto != 9800.50 {
		t.Fatalf("Next/script JSON servidores inesperado: err=%v res=%+v", err, serv)
	}

	desp, err := transparencia.QueryHTMLPortalDespesas(ctx, client, m, transparencia.PortalQueryParams{BaseURL: srv.URL + "/datatables-despesas", Ano: 2025, Limit: 5}, "html_portal")
	if err != nil || !desp.Disponivel || len(desp.Despesas) != 1 || desp.Despesas[0].Fornecedor != "CONSTRUTORA BETA" || desp.Despesas[0].DocumentoFornecedor != "11.222.333/0001-44" || desp.Despesas[0].ValorEmpenhado != 42500.00 {
		t.Fatalf("DataTables ajax despesas inesperado: err=%v res=%+v", err, desp)
	}

	rec, err := transparencia.QueryHTMLPortalReceitas(ctx, client, m, transparencia.PortalQueryParams{BaseURL: srv.URL + "/fetch-receitas", Limit: 5}, "html_portal")
	if err != nil || !rec.Disponivel || len(rec.Receitas) != 1 || rec.Receitas[0].Descricao != "IPTU" || rec.Receitas[0].ValorArrecadado != 85000.25 {
		t.Fatalf("fetch XHR receitas inesperado: err=%v res=%+v", err, rec)
	}
}

func TestHTMLPortalTwoHopAndCSVExportMock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "3157807", Nome: "Santa Luzia", UF: "MG"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<html><body>
				<h1>Prefeitura Municipal</h1>
				<a href="/transparencia">Portal da Transparência</a>
			</body></html>`))
		case "/transparencia":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<html><body>
				<h2>Menu Transparência</h2>
				<a href="/transparencia/licitacoes-hub">Licitações e Editais</a>
			</body></html>`))
		case "/transparencia/licitacoes-hub":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<html><body>
				<h2>Licitações</h2>
				<p>Clique para baixar:</p>
				<a href="/exports/licitacoes.csv">Exportar CSV de Licitações</a>
			</body></html>`))
		case "/exports/licitacoes.csv":
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			_, _ = w.Write([]byte("\xef\xbb\xbfRelatorio Municipal\nNumero;Modalidade;Objeto;Situacao;Valor Estimado\n012/2025;Pregão Eletrônico;Compra de computadores;Homologada;R$ 125.000,00\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	lic, err := transparencia.QueryHTMLPortalLicitacoes(ctx, client, m, transparencia.PortalQueryParams{BaseURL: srv.URL, Ano: 2025, Limit: 5}, "prefeitura_html")
	if err != nil || !lic.Disponivel || len(lic.Licitacoes) != 1 {
		t.Fatalf("2-hop + CSV export licitacoes inesperado: err=%v res=%+v", err, lic)
	}
	if lic.Licitacoes[0].Numero != "012/2025" || lic.Licitacoes[0].ValorEstimado != 125000.00 {
		t.Fatalf("dados CSV inesperados: %+v", lic.Licitacoes[0])
	}
}

func TestHTMLPortalFormSubmissionAndSafetyMock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "3157807", Nome: "Santa Luzia", UF: "MG"}

	var loginSubmitted bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/contratos-form":
			if r.Method == http.MethodGet {
				http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: "sess-abc-123", Path: "/"})
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write([]byte(`<html><body>
					<form method="POST" action="/contratos-form">
						<input type="hidden" name="_token" value="csrf-xyz" />
						<select name="ano">
							<option value="2024">2024</option>
							<option value="2025" selected>2025</option>
						</select>
						<input type="submit" value="Pesquisar Contratos" />
					</form>
				</body></html>`))
				return
			}
			c, err := r.Cookie("PHPSESSID")
			if err != nil || c.Value != "sess-abc-123" {
				http.Error(w, "missing session cookie", http.StatusBadRequest)
				return
			}
			if err := r.ParseForm(); err != nil || r.FormValue("_token") != "csrf-xyz" || r.FormValue("ano") != "2025" {
				http.Error(w, "invalid form payload", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<html><body><table>
				<tr><th>Contrato</th><th>Fornecedor</th><th>CPF/CNPJ</th><th>Valor</th></tr>
				<tr><td>CT-55/2025</td><td>SERVICOS FORM LTDA</td><td>22.333.444/0001-55</td><td>R$ 75.400,00</td></tr>
			</table></body></html>`))
		case "/login-portal":
			if r.Method == http.MethodPost {
				loginSubmitted = true
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<html><body>
				<form method="POST" action="/login-portal">
					<input type="text" name="usuario" />
					<input type="password" name="senha" />
					<input type="submit" value="Entrar" />
				</form>
			</body></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	ct, err := transparencia.QueryHTMLPortalContratos(ctx, client, m, transparencia.PortalQueryParams{BaseURL: srv.URL + "/contratos-form", Ano: 2025, Limit: 5}, "html_portal")
	if err != nil || !ct.Disponivel || len(ct.Contratos) != 1 {
		t.Fatalf("auto form submission contratos inesperado: err=%v res=%+v", err, ct)
	}
	if ct.Contratos[0].Numero != "CT-55/2025" || ct.Contratos[0].Valor != 75400.00 {
		t.Fatalf("contrato pós-POST inesperado: %+v", ct.Contratos[0])
	}

	if _, err := transparencia.QueryHTMLPortalContratos(ctx, client, m, transparencia.PortalQueryParams{BaseURL: srv.URL + "/login-portal", Ano: 2025, Limit: 5}, "html_portal"); err == nil {
		t.Fatalf("esperava erro sem tabelas para formulário de login")
	}
	if loginSubmitted {
		t.Fatalf("formulário com campo password NÃO deveria ter sido submetido")
	}
}

func TestHTMLPortalObrasConveniosLegislacaoConcursosAndIframeMock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "2512101", Nome: "Pombal", UF: "PB"}

	var srvURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/portal":
			// Inclui JSON-LD Yoast SEO BreadcrumbList que deve ser ignorado pelo extrator
			_, _ = w.Write([]byte(`<!doctype html><html><head>
				<script type="application/ld+json" class="yoast-schema-graph">{"@context":"https://schema.org","@graph":[{"@type":"BreadcrumbList","itemListElement":[{"@type":"ListItem","position":1,"name":"Início"},{"@type":"ListItem","position":2,"name":"Portal da Transparência"}]}]}</script>
			</head><body>
				<a href="/portal/folha-de-pessoal">Folha de Pessoal</a>
				<a href="/portal/obras-publicas">Obras Públicas</a>
				<a href="/portal/convenios">Convênios e Transferências</a>
				<a href="/portal/legislacao">Legislação Municipal e Leis</a>
				<a href="/portal/concursos">Concursos e Processos Seletivos</a>
			</body></html>`))
		case "/portal/folha-de-pessoal":
			_, _ = w.Write([]byte(`<html><head>
				<script type="application/ld+json">{"@context":"https://schema.org","@type":"BreadcrumbList","itemListElement":[{"@type":"ListItem","position":1,"name":"Folha"}]}</script>
			</head><body>
				<iframe src="` + srvURL + `/iframe/folha"></iframe>
			</body></html>`))
		case "/iframe/folha":
			_, _ = w.Write([]byte(`<html><body><table>
				<tr><th>Nome</th><th>CPF</th><th>Cargo</th><th>Lotação</th><th>Salário Bruto</th><th>Salário Líquido</th><th>Competência</th></tr>
				<tr><td>JOSE SERVIDOR IFRAME</td><td>123.456.789-00</td><td>Médico</td><td>Saúde</td><td>R$ 18.500,00</td><td>R$ 14.200,00</td><td>03/2025</td></tr>
			</table></body></html>`))
		case "/portal/obras-publicas":
			_, _ = w.Write([]byte(`<html><body><table>
				<tr><th>Descrição da Obra</th><th>Local</th><th>Contratada</th><th>Situação</th><th>Percentual</th><th>Valor</th></tr>
				<tr><td>Pavimentação de vias</td><td>Centro</td><td>ENGENHARIA PB LTDA</td><td>Em andamento</td><td>65,5%</td><td>R$ 890.000,00</td></tr>
			</table></body></html>`))
		case "/portal/convenios":
			_, _ = w.Write([]byte(`<html><body><table>
				<tr><th>Número</th><th>Concedente</th><th>Objeto</th><th>Situação</th><th>Vigência</th><th>Valor</th></tr>
				<tr><td>CV-10/2025</td><td>Ministério da Saúde</td><td>Reforma de UBS</td><td>Vigente</td><td>31/12/2025</td><td>R$ 450.000,00</td></tr>
			</table></body></html>`))
		case "/portal/legislacao":
			_, _ = w.Write([]byte(`<html><body><table>
				<tr><th>Tipo</th><th>Número</th><th>Ano</th><th>Data</th><th>Ementa</th></tr>
				<tr><td><a href="/leis/lei_2100.pdf">Lei Ordinária</a></td><td>2.100</td><td>2025</td><td>15/02/2025</td><td>Dispõe sobre o plano de cargos municipal</td></tr>
			</table></body></html>`))
		case "/portal/concursos":
			_, _ = w.Write([]byte(`<html><body><table>
				<tr><th>Concurso</th><th>Cargo</th><th>Candidato</th><th>CPF</th><th>Classificação</th><th>Situação</th></tr>
				<tr><td>Edital 01/2025</td><td>Enfermeiro</td><td>MARIA CONCURSADA</td><td>987.654.321-99</td><td>1º Lugar</td><td>Convocado</td></tr>
			</table></body></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	srvURL = srv.URL

	p := transparencia.PortalQueryParams{BaseURL: srv.URL + "/portal", Ano: 2025, Limit: 10}

	// 1. 2-hop subpage -> iframe ignorando Yoast BreadcrumbList JSON-LD
	serv, err := transparencia.QueryHTMLPortalServidores(ctx, client, m, p, "cms_wordpress_html")
	if err != nil || !serv.Disponivel || len(serv.Servidores) != 1 || serv.Servidores[0].Nome != "JOSE SERVIDOR IFRAME" || serv.Servidores[0].CPFParcial != "***.456.789-**" {
		t.Fatalf("Servidores 2-hop iframe inesperado: err=%v res=%+v", err, serv)
	}

	// 2. Obras, Convênios, Legislação e Concursos
	obras, err := transparencia.QueryHTMLPortalObras(ctx, client, m, p, "html_portal")
	if err != nil || !obras.Disponivel || len(obras.Obras) != 1 || obras.Obras[0].Valor != 890000.00 || obras.Obras[0].Situacao != "Em andamento" {
		t.Fatalf("QueryHTMLPortalObras inesperado: err=%v res=%+v", err, obras)
	}

	conv, err := transparencia.QueryHTMLPortalConvenios(ctx, client, m, p, "html_portal")
	if err != nil || !conv.Disponivel || len(conv.Convenios) != 1 || conv.Convenios[0].Valor != 450000.00 {
		t.Fatalf("QueryHTMLPortalConvenios inesperado: err=%v res=%+v", err, conv)
	}

	leg, err := transparencia.QueryHTMLPortalLegislacao(ctx, client, m, p, "html_portal")
	if err != nil || !leg.Disponivel || len(leg.Atos) != 1 || leg.Atos[0].Numero != "2.100" {
		t.Fatalf("QueryHTMLPortalLegislacao inesperado: err=%v res=%+v", err, leg)
	}

	conc, err := transparencia.QueryHTMLPortalConcursos(ctx, client, m, p, "html_portal")
	if err != nil || !conc.Disponivel || len(conc.Aprovados) != 1 || conc.Aprovados[0].Nome != "MARIA CONCURSADA" {
		t.Fatalf("QueryHTMLPortalConcursos inesperado: err=%v res=%+v", err, conc)
	}

	// 3. Roteamento via Hub para os novos módulos
	eng, err := transparencia.NewEngine()
	if err != nil {
		t.Fatalf("NewEngine erro: %v", err)
	}
	hub := newTestHub(eng, srv.Client())
	opts := transparencia.QueryOptions{UF: "PB", OverrideURL: srv.URL + "/portal", Ano: 2025, Limit: 5}

	if res, err := hub.QueryObras(ctx, "Pombal", opts); err != nil || !res.Disponivel || len(res.Obras) != 1 {
		t.Fatalf("Hub.QueryObras HTML inesperado: err=%v res=%+v", err, res)
	}
	if res, err := hub.QueryConvenios(ctx, "Pombal", opts); err != nil || !res.Disponivel || len(res.Convenios) != 1 {
		t.Fatalf("Hub.QueryConvenios HTML inesperado: err=%v res=%+v", err, res)
	}
	if res, err := hub.QueryLegislacao(ctx, "Pombal", opts); err != nil || !res.Disponivel || len(res.Atos) != 1 {
		t.Fatalf("Hub.QueryLegislacao HTML inesperado: err=%v res=%+v", err, res)
	}
	if res, err := hub.QueryConcursos(ctx, "Pombal", opts); err != nil || !res.Disponivel || len(res.Aprovados) != 1 {
		t.Fatalf("Hub.QueryConcursos HTML inesperado: err=%v res=%+v", err, res)
	}
}
