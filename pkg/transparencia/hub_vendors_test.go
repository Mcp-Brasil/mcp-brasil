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

func TestHubRoutingAllNewVendors(t *testing.T) {
	eng, err := transparencia.NewEngine()
	if err != nil {
		t.Fatalf("NewEngine erro: %v", err)
	}

	// 1. Betha Cloud (SaaS directory discovery via IBGE + Hub routing)
	t.Run("betha_cloud", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		transparencia.ResetBethaTokenCache()
		var srvURL string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/oauth":
				w.Header().Set("Location", "https://transparencia.betha.cloud/auth-callback.html#access_token=tok123&expires_in=3600")
				w.WriteHeader(http.StatusSeeOther)
			case "/api/auth/portais":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"content":[{"id":1,"nome":"Prefeitura de Abdon Batista","uf":"SC","codigoIbge":4200051,"hash":"HASH123"}]}`))
			case "/api/api/menu":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[{"id":1,"titulo":"Recursos Humanos","itens":[{"id":10,"titulo":"Folha de Pagamento - Servidores","consultaId":10,"tipo":"CONSULTA_PADRAO"}]}]`))
			case "/api/api/busca-textual/10":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"content":[{"nome":"MARIA BETHA","cpf":"12345678901","cargo":"PROFESSORA","salarioBruto":4500.0}],"totalElements":1}`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()
		srvURL = srv.URL

		transparencia.SetBethaOAuthURL(srvURL + "/oauth")
		transparencia.SetBethaAPIBaseURL(srvURL + "/api")
		transparencia.SetAgiliAPIBaseURL(srvURL + "/none")
		transparencia.SetEPublicaBaseURL(srvURL + "/none")
		prevEq := transparencia.SetEquiplanoAPIBaseURL(srvURL + "/none")
		defer func() {
			transparencia.SetBethaOAuthURL("https://plataforma-oauth.betha.cloud/auth/oauth2/authorize")
			transparencia.SetBethaAPIBaseURL("https://api.transparencia.betha.cloud/transparencia")
			transparencia.SetAgiliAPIBaseURL("https://transparencia.agilicloud.com.br/api")
			transparencia.SetEPublicaBaseURL("")
			transparencia.SetEquiplanoAPIBaseURL(prevEq)
			transparencia.ResetBethaTokenCache()
		}()

		hub := &transparencia.Hub{
			Engine:     eng,
			HTTPClient: srv.Client(),
			TCE:        transparencia.NewTCEClient(srv.Client()),
			Siconfi:    transparencia.NewSiconfiClient(),
			PNCP:       transparencia.NewPNCPClient(),
		}

		res, err := hub.QueryServidores(ctx, "Abdon Batista", transparencia.QueryOptions{UF: "SC", Limit: 5})
		if err != nil {
			t.Fatalf("QueryServidores Betha erro: %v", err)
		}
		if res.Plataforma != "betha_cloud" || len(res.Servidores) != 1 || res.Servidores[0].CPFParcial != "***.456.789-**" {
			t.Fatalf("Resultado Betha inesperado via Hub: %+v", res)
		}
	})

	// 2. IPM / Atende.Net via OverrideURL + Hub routing
	t.Run("ipm_atende", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/api/WCPDadosAbertos/despesasOrcadas":
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"status":"ok","retorno":[]}`))
			case "/api/transparencia-pessoal-funcionarios":
				_, _ = w.Write([]byte(`{"registros":[{"nome":"JOAO IPM","cpf":"98765432100","cargo":"TECNICO","salarioBase":"3100,00"}]}`))
			case "/api/WCPDadosAbertos/despesas":
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"status":"ok","retorno":[{"nomeCredor":"FORNECEDOR IPM","cpfCnpjCredor":"12345678000199","valorEmpenhado":"1200.50","valorPago":"1200.50"}]}`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		hub := &transparencia.Hub{
			Engine:     eng,
			HTTPClient: srv.Client(),
			TCE:        transparencia.NewTCEClient(srv.Client()),
			Siconfi:    transparencia.NewSiconfiClient(),
			PNCP:       transparencia.NewPNCPClient(),
		}
		serv, err := hub.QueryServidores(ctx, "Brusque", transparencia.QueryOptions{UF: "SC", OverrideURL: srv.URL, Limit: 5})
		if err != nil || serv.Plataforma != "ipm_atende" || len(serv.Servidores) != 1 || serv.Servidores[0].CPFParcial != "***.654.321-**" {
			t.Fatalf("QueryServidores IPM inesperado: err=%v res=%+v", err, serv)
		}
		desp, err := hub.QueryDespesas(ctx, "Brusque", transparencia.QueryOptions{UF: "SC", OverrideURL: srv.URL, Ano: 2025, Limit: 5})
		if err != nil || desp.Plataforma != "ipm_atende" || len(desp.Despesas) != 1 || desp.Despesas[0].ValorPago != 1200.50 {
			t.Fatalf("QueryDespesas IPM inesperado: err=%v res=%+v", err, desp)
		}
	})

	// 3. E&L PortalTP via OverrideURL + Hub routing
	t.Run("el_portaltp", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.RawQuery, "WSDL") || r.URL.RawQuery == "WSDL" {
				_, _ = w.Write([]byte(`<definitions name="transparencia"><operation name="json_empenhos"/></definitions>`))
				return
			}
			if r.URL.Path == "/api/transparencia.asmx/json_empenhos" {
				w.Header().Set("Content-Type", "text/xml; charset=utf-8")
				_, _ = w.Write([]byte(`<string xmlns="http://tempuri.org/">[{"empenho":"10/2025","nome_favorecido":"EMPRESA TP","documento_favorecido":"11122233344","valor":950.25}]</string>`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		hub := &transparencia.Hub{
			Engine:     eng,
			HTTPClient: srv.Client(),
			TCE:        transparencia.NewTCEClient(srv.Client()),
			Siconfi:    transparencia.NewSiconfiClient(),
			PNCP:       transparencia.NewPNCPClient(),
		}
		desp, err := hub.QueryDespesas(ctx, "Alegre", transparencia.QueryOptions{UF: "ES", OverrideURL: srv.URL, Ano: 2025, Mes: 1, Limit: 5})
		if err != nil || desp.Plataforma != "portaltp" || len(desp.Despesas) != 1 || desp.Despesas[0].DocumentoFornecedor != "***.222.333-**" {
			t.Fatalf("QueryDespesas PortalTP inesperado: err=%v res=%+v", err, desp)
		}
	})

	// 4. Sonner GRP via OverrideURL + Hub routing
	t.Run("sonner_grp", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/GRP/pic/api/config/contexto":
				_, _ = w.Write([]byte(`{"exercicioAtual":2025,"municipio":{"nome":"Suzano","uf":"SP","codigoIbge":"3552502"},"dadosOrgao":{"nome":"Prefeitura de Suzano"}}`))
			case "/GRP/pic/api/auth/csrf":
				http.SetCookie(w, &http.Cookie{Name: "PORTAL-XSRF-TOKEN", Value: "xsrf-ok", Path: "/"})
				w.WriteHeader(http.StatusNoContent)
			case "/GRP/pic/api/funcionalidade":
				_, _ = w.Write([]byte(`[{"id":"1","numericKey":77,"url":"/consulta-contrato"}]`))
			case "/GRP/pic/api/consulta-contrato":
				if r.Header.Get("X-PORTAL-XSRF-TOKEN") != "xsrf-ok" {
					http.Error(w, "missing xsrf", http.StatusForbidden)
					return
				}
				_, _ = w.Write([]byte(`[{"numeroContrato":"CT-99","ano":2025,"fornecedor":"OBRA SONNER LTDA","objeto":"REFORMA","valorInicial":88000.0}]`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		hub := &transparencia.Hub{
			Engine:     eng,
			HTTPClient: srv.Client(),
			TCE:        transparencia.NewTCEClient(srv.Client()),
			Siconfi:    transparencia.NewSiconfiClient(),
			PNCP:       transparencia.NewPNCPClient(),
		}
		ct, err := hub.QueryContratos(ctx, "Suzano", transparencia.QueryOptions{UF: "SP", OverrideURL: srv.URL + "/portal", Ano: 2025, Limit: 5})
		if err != nil || ct.Plataforma != "sonner" || len(ct.Contratos) != 1 || ct.Contratos[0].Valor != 88000.0 {
			t.Fatalf("QueryContratos Sonner inesperado: err=%v res=%+v", err, ct)
		}
	})

	// 5. Equiplano Cloud via SaaS discovery + Hub routing
	t.Run("equiplano_cloud", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/transparencia/dominio/check/prefeitura-capanema":
				_, _ = w.Write([]byte(`{"entidade":{"uuid":"e-uuid"},"municipioClienteUuid":"mc-uuid","clienteUuid":"c-uuid"}`))
			case "/transparencia/cont_contratos_fornecedor/process":
				payload := base64.StdEncoding.EncodeToString([]byte(`{"contratoCollection":[{"numero":"12/2025","exercicio":2025,"nmForncedor":"CONTRATADA EQ","valorContrato":15000.0}]}`))
				_ = json.NewEncoder(w).Encode(map[string]any{"sucesso": true, "dados": map[string]any{"documento": payload}})
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		transparencia.SetBethaOAuthURL(srv.URL + "/none")
		transparencia.SetBethaAPIBaseURL(srv.URL + "/none")
		transparencia.SetAgiliAPIBaseURL(srv.URL + "/none")
		transparencia.SetEPublicaBaseURL(srv.URL + "/none")
		prev := transparencia.SetEquiplanoAPIBaseURL(srv.URL)
		defer func() {
			transparencia.SetBethaOAuthURL("https://plataforma-oauth.betha.cloud/auth/oauth2/authorize")
			transparencia.SetBethaAPIBaseURL("https://api.transparencia.betha.cloud/transparencia")
			transparencia.SetAgiliAPIBaseURL("https://transparencia.agilicloud.com.br/api")
			transparencia.SetEPublicaBaseURL("")
			transparencia.SetEquiplanoAPIBaseURL(prev)
			transparencia.ResetBethaTokenCache()
		}()

		hub := &transparencia.Hub{
			Engine:     eng,
			HTTPClient: srv.Client(),
			TCE:        transparencia.NewTCEClient(srv.Client()),
			Siconfi:    transparencia.NewSiconfiClient(),
			PNCP:       transparencia.NewPNCPClient(),
		}
		ct, err := hub.QueryContratos(ctx, "Capanema", transparencia.QueryOptions{UF: "PR", Ano: 2025, Limit: 5})
		if err != nil || ct.Plataforma != "equiplano_cloud" || len(ct.Contratos) != 1 || ct.Contratos[0].Valor != 15000.0 {
			t.Fatalf("QueryContratos Equiplano inesperado: err=%v res=%+v", err, ct)
		}
	})
}

func newTestHub(eng *transparencia.Engine, client *http.Client) *transparencia.Hub {
	return &transparencia.Hub{
		Engine:     eng,
		HTTPClient: client,
		TCE:        transparencia.NewTCEClient(client),
		Siconfi:    transparencia.NewSiconfiClient(),
		PNCP:       transparencia.NewPNCPClient(),
	}
}

func TestHubRoutingExtendedVendors(t *testing.T) {
	eng, err := transparencia.NewEngine()
	if err != nil {
		t.Fatalf("NewEngine erro: %v", err)
	}

	t.Run("nucleogov", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = r.ParseForm()
			switch r.URL.Path {
			case "/api/orgaos":
				_, _ = w.Write([]byte(`{"dados":[{"Id":1,"Nome":"Prefeitura","Principal":true}],"total":1}`))
			case "/api/servidores_cnt":
				_, _ = w.Write([]byte(`{"dados":[{"Nome":"MARIA NUCLEO","cpf":"12345678901","Cargo":"Auditora","SalarioBruto":7500.0}],"total":1}`))
			case "/api/centi":
				if r.Form.Get("acao") == "empenhos" {
					_, _ = w.Write([]byte(`{"dados":[{"NumeroEmpenho":"10/2025","Ano":2025,"Fornecedor":"EMPRESA NUCLEO","ValorEmpenhado":2200.0}],"total":1}`))
					return
				}
				http.NotFound(w, r)
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		hub := newTestHub(eng, srv.Client())
		serv, err := hub.QueryServidores(ctx, "Anápolis", transparencia.QueryOptions{UF: "GO", OverrideURL: srv.URL, Ano: 2025, Mes: 3, Limit: 5})
		if err != nil || serv.Plataforma != "nucleogov" || len(serv.Servidores) != 1 || serv.Servidores[0].CPFParcial != "***.456.789-**" {
			t.Fatalf("Hub nucleogov servidores inesperado: err=%v res=%+v", err, serv)
		}
		desp, err := hub.QueryDespesas(ctx, "Anápolis", transparencia.QueryOptions{UF: "GO", OverrideURL: srv.URL, Ano: 2025, Limit: 5})
		if err != nil || desp.Plataforma != "nucleogov" || len(desp.Despesas) != 1 || desp.Despesas[0].ValorEmpenhado != 2200.0 {
			t.Fatalf("Hub nucleogov despesas inesperado: err=%v res=%+v", err, desp)
		}
	})

	t.Run("diretoriodigital_sistp", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/dados-abertos/swagger.yaml":
				w.Header().Set("Content-Type", "text/yaml")
				_, _ = w.Write([]byte("openapi: 3.0.0\npaths:\n  /listagem-servidores:\n    get:\n      parameters:\n        - name: cliente\n          in: query\n          schema:\n            default: pm_hub\n"))
			case "/api/listagem-servidores":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[{"nome":"ANA SISTP","cpf":"11122233344","cargo":"Enfermeira","bruto":6000.0}]`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		transparencia.SetDiretorioDigitalBaseURL(srv.URL)
		defer transparencia.SetDiretorioDigitalBaseURL("")

		hub := newTestHub(eng, srv.Client())
		serv, err := hub.QueryServidores(ctx, "Catalão", transparencia.QueryOptions{UF: "GO", OverrideURL: srv.URL, Ano: 2025, Mes: 3, Limit: 5})
		if err != nil || serv.Plataforma != "diretoriodigital_sistp" || len(serv.Servidores) != 1 || serv.Servidores[0].CPFParcial != "***.222.333-**" {
			t.Fatalf("Hub diretoriodigital_sistp inesperado: err=%v res=%+v", err, serv)
		}
	})

	t.Run("infbr_netuse", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/servidores/pe_remuneracao_servidores_ajax.php" && r.Method == http.MethodPost {
				_ = r.ParseForm()
				w.Header().Set("Content-Type", "application/json")
				switch r.Form.Get("action") {
				case "ultima_competencia":
					_, _ = w.Write([]byte(`{"ano":2025,"mes":8}`))
				case "listar":
					_, _ = w.Write([]byte(`{"dados":[{"nome":"PEDRO INFBR","cpf":"98765432100","cargo":"Fiscal","bruto":"4.100,00"}],"total":1,"total_pages":1}`))
				default:
					http.NotFound(w, r)
				}
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		hub := newTestHub(eng, srv.Client())
		serv, err := hub.QueryServidores(ctx, "Arcoverde", transparencia.QueryOptions{UF: "PE", OverrideURL: srv.URL, Limit: 5})
		if err != nil || serv.Plataforma != "infbr_netuse" || len(serv.Servidores) != 1 || serv.Servidores[0].CPFParcial != "***.654.321-**" {
			t.Fatalf("Hub infbr_netuse inesperado: err=%v res=%+v", err, serv)
		}
	})

	t.Run("aspec_gt", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/transparencia/api/v1/json/exercicios/11629488":
				_, _ = w.Write([]byte(`[{"id":42,"nome":"2025"}]`))
			case "/transparencia/api/v1/json/empenhos/11629488":
				_, _ = w.Write([]byte(`[{"numero":"1/2025","exercicio":2025,"credor":"JOAO ASPEC","cpfCnpj":"12345678901","valorEmpenhado":3100.0}]`))
			case "/transparencia/api/v1/json/contratos/11629488":
				_, _ = w.Write([]byte(`[{"numero":"CT-1","ano":2025,"contratado":"CONSTRUTORA ASPEC","valor":55000.0}]`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		hub := newTestHub(eng, srv.Client())
		desp, err := hub.QueryDespesas(ctx, "Acopiara", transparencia.QueryOptions{UF: "CE", OverrideURL: srv.URL + "/11629488", Ano: 2025, Limit: 5})
		if err != nil || desp.Plataforma != "aspec_gt" || len(desp.Despesas) != 1 || desp.Despesas[0].DocumentoFornecedor != "***.456.789-**" {
			t.Fatalf("Hub aspec_gt despesas inesperado: err=%v res=%+v", err, desp)
		}
		ct, err := hub.QueryContratos(ctx, "Acopiara", transparencia.QueryOptions{UF: "CE", OverrideURL: srv.URL + "/11629488", Ano: 2025, Limit: 5})
		if err != nil || ct.Plataforma != "aspec_gt" || len(ct.Contratos) != 1 || ct.Contratos[0].Valor != 55000.0 {
			t.Fatalf("Hub aspec_gt contratos inesperado: err=%v res=%+v", err, ct)
		}
	})

	t.Run("assesi_portal", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/dadosabertosexportar.php" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Query().Get("d") {
			case "pessoal":
				_, _ = w.Write([]byte(`[{"nome":"CARLOS ASSESI","cpf":"11122233344","cargo":"Tecnico","salario_bruto":4200.0}]`))
			case "receitas":
				_, _ = w.Write([]byte(`[{"codigo":"1.1","descricao":"IPTU ASSESI","valor_previsto":10000,"valor_arrecadado":9000}]`))
			default:
				_, _ = w.Write([]byte(`[]`))
			}
		}))
		defer srv.Close()

		hub := newTestHub(eng, srv.Client())
		serv, err := hub.QueryServidores(ctx, "Calçoene", transparencia.QueryOptions{UF: "AP", OverrideURL: srv.URL, Ano: 2025, Limit: 5})
		if err != nil || serv.Plataforma != "assesi_portal" || len(serv.Servidores) != 1 || serv.Servidores[0].CPFParcial != "***.222.333-**" {
			t.Fatalf("Hub assesi_portal servidores inesperado: err=%v res=%+v", err, serv)
		}
		rec, err := hub.QueryReceitas(ctx, "Calçoene", transparencia.QueryOptions{UF: "AP", OverrideURL: srv.URL, Ano: 2025, Limit: 5})
		if err != nil || rec.Plataforma != "assesi_portal" || len(rec.Receitas) != 1 || rec.Receitas[0].ValorArrecadado != 9000 {
			t.Fatalf("Hub assesi_portal receitas inesperado: err=%v res=%+v", err, rec)
		}
	})

	t.Run("centi_api", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/portal/getlistaorgaos/GO/goiania":
				_, _ = w.Write([]byte(`[{"Id":1,"Nome":"PREFEITURA DE GOIANIA"}]`))
			case "/portal/getfuncionarios/GO/goiania":
				_, _ = w.Write([]byte(`[{"Nome":"FERNANDA CENTI","Cpf":"32165498700","Cargo":"Analista","SalarioBruto":6200.0}]`))
			case "/portal/GetContratoPortal/GO/goiania":
				_, _ = w.Write([]byte(`[{"Numero":"CT-CENTI","Ano":2025,"Contratado":"FORNECEDOR CENTI","Valor":44000.0}]`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		transparencia.SetCentiAPIBaseURL(srv.URL)
		defer transparencia.SetCentiAPIBaseURL("")

		hub := newTestHub(eng, srv.Client())
		serv, err := hub.QueryServidores(ctx, "Goiânia", transparencia.QueryOptions{
			UF:          "GO",
			OverrideURL: "https://acessoainformacao.goiania.go.gov.br",
			Ano:         2025,
			Limit:       5,
		})
		if err != nil || serv.Plataforma != "centi_api" || len(serv.Servidores) != 1 || serv.Servidores[0].CPFParcial != "***.654.987-**" {
			t.Fatalf("Hub centi_api servidores inesperado: err=%v res=%+v", err, serv)
		}
		ct, err := hub.QueryContratos(ctx, "Goiânia", transparencia.QueryOptions{
			UF:          "GO",
			OverrideURL: "https://acessoainformacao.goiania.go.gov.br",
			Ano:         2025,
			Limit:       5,
		})
		if err != nil || ct.Plataforma != "centi_api" || len(ct.Contratos) != 1 || ct.Contratos[0].Valor != 44000.0 {
			t.Fatalf("Hub centi_api contratos inesperado: err=%v res=%+v", err, ct)
		}
	})

	t.Run("fator_sistemas", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/dados/diarias.php":
				_, _ = w.Write([]byte(`<html>Diárias</html>`))
			case "/dados/despesa.php":
				http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: "desp", Path: "/"})
				_, _ = w.Write([]byte(`<html>Despesa</html>`))
			case "/dados/carregaDespesa.php":
				_, _ = w.Write([]byte(`ok`))
			case "/dados/folhapag.php":
				http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: "folha", Path: "/"})
				_, _ = w.Write([]byte(`<html>Folha</html>`))
			case "/dados/carregaFolhapag.php":
				_, _ = w.Write([]byte(`ok`))
			case "/dados/imprimir.php":
				w.Header().Set("Content-Type", "application/json")
				if c, err := r.Cookie("PHPSESSID"); err == nil && c.Value == "folha" {
					_, _ = w.Write([]byte(`[{"nome":"BEATRIZ FATOR","cpf":"44455566677","cargo":"Médica","salario_bruto":12000.0}]`))
					return
				}
				_, _ = w.Write([]byte(`[{"numero":"778/2025","ano":2025,"fornecedor":"ANTONIO FATOR","cpf_cnpj":"12345678901","valor_empenhado":3450.0}]`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		hub := newTestHub(eng, srv.Client())
		desp, err := hub.QueryDespesas(ctx, "Almadina", transparencia.QueryOptions{
			UF:          "BA",
			OverrideURL: srv.URL + "/dados/despesa.php?id=pm_almadina",
			Ano:         2025,
			Limit:       5,
		})
		if err != nil || desp.Plataforma != "fator_sistemas" || len(desp.Despesas) != 1 || desp.Despesas[0].DocumentoFornecedor != "***.456.789-**" {
			t.Fatalf("Hub fator_sistemas despesas inesperado: err=%v res=%+v", err, desp)
		}
		serv, err := hub.QueryServidores(ctx, "Almadina", transparencia.QueryOptions{
			UF:          "BA",
			OverrideURL: srv.URL + "/dados/despesa.php?id=pm_almadina",
			Ano:         2025,
			Mes:         3,
			Limit:       5,
		})
		if err != nil || serv.Plataforma != "fator_sistemas" || len(serv.Servidores) != 1 || serv.Servidores[0].CPFParcial != "***.555.666-**" {
			t.Fatalf("Hub fator_sistemas servidores inesperado: err=%v res=%+v", err, serv)
		}
	})

	t.Run("smarapd_pai", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/paiportalserver/cliente":
				_, _ = w.Write([]byte(`{"Nome":"PREFEITURA DE MARILIA"}`))
			case "/paiportalserver/DadosAbertos":
				_, _ = w.Write([]byte(`{"Modulos":[{"ChaveModulo":"RH","Visoes":[{"NomeVisao":"SERVIDORES","Titulo":"Folha Servidores"}]},{"ChaveModulo":"ORC","Visoes":[{"NomeVisao":"RECEITAS","Titulo":"Receitas Arrecadadas"}]}]}`))
			case "/paiportalserver/MenuPortal":
				_, _ = w.Write([]byte(`[]`))
			case "/paiportalserver/modulovisao/filter":
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				chave, _ := body["ChaveModulo"].(string)
				switch chave {
				case "RH":
					_, _ = w.Write([]byte(`{"Registros":[{"Nome":"PAULO SMARAPD","CPF":"12345678901","Cargo":"Engenheiro","SalarioBruto":8800.0}],"TotalRegistros":1}`))
				case "ORC":
					_, _ = w.Write([]byte(`{"Registros":[{"Codigo":"1.1.1","Descricao":"IPTU SMARAPD","ValorPrevisto":500000,"ValorArrecadado":450000}],"TotalRegistros":1}`))
				default:
					_, _ = w.Write([]byte(`{"Registros":[],"TotalRegistros":0}`))
				}
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		hub := newTestHub(eng, srv.Client())
		serv, err := hub.QueryServidores(ctx, "Marília", transparencia.QueryOptions{UF: "SP", OverrideURL: srv.URL, Ano: 2025, Limit: 5})
		if err != nil || serv.Plataforma != "smarapd_pai" || len(serv.Servidores) != 1 || serv.Servidores[0].CPFParcial != "***.456.789-**" {
			t.Fatalf("Hub smarapd_pai servidores inesperado: err=%v res=%+v", err, serv)
		}
		rec, err := hub.QueryReceitas(ctx, "Marília", transparencia.QueryOptions{UF: "SP", OverrideURL: srv.URL, Ano: 2025, Limit: 5})
		if err != nil || rec.Plataforma != "smarapd_pai" || len(rec.Receitas) != 1 || rec.Receitas[0].ValorArrecadado != 450000 {
			t.Fatalf("Hub smarapd_pai receitas inesperado: err=%v res=%+v", err, rec)
		}
	})

	t.Run("memory_openknowledge", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/lai/pessoal/estruturaorganizacional/":
				_, _ = w.Write([]byte(`[{"codigo":1,"descricao":"GABINETE"}]`))
			case "/lai/pessoal/servidor/":
				_, _ = w.Write([]byte(`{"content":[{"nome":"ANA MEMORY","cpf":"11122233344","cargo":"Médica","salarioBruto":10200.0}],"totalElements":1}`))
			case "/lai/compras/contrato/":
				_, _ = w.Write([]byte(`{"items":[{"numeroContrato":"19/2025","exercicio":2025,"fornecedor":"TECH MEMORY","valor":96000.0}],"total":1}`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		transparencia.SetMemoryAPIBaseURL(srv.URL)
		defer transparencia.SetMemoryAPIBaseURL("")

		override := "https://ilai.memory.com.br/#/entidades/login/MEM001/12"
		hub := newTestHub(eng, srv.Client())
		serv, err := hub.QueryServidores(ctx, "Araçaí", transparencia.QueryOptions{UF: "MG", OverrideURL: override, Ano: 2025, Limit: 5})
		if err != nil || serv.Plataforma != "memory_openknowledge" || len(serv.Servidores) != 1 || serv.Servidores[0].CPFParcial != "***.222.333-**" {
			t.Fatalf("Hub memory_openknowledge servidores inesperado: err=%v res=%+v", err, serv)
		}
		ct, err := hub.QueryContratos(ctx, "Araçaí", transparencia.QueryOptions{UF: "MG", OverrideURL: override, Ano: 2025, Limit: 5})
		if err != nil || ct.Plataforma != "memory_openknowledge" || len(ct.Contratos) != 1 || ct.Contratos[0].Valor != 96000.0 {
			t.Fatalf("Hub memory_openknowledge contratos inesperado: err=%v res=%+v", err, ct)
		}
	})

	t.Run("facil_transparencia", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/api/receita/ReceitasGetAno":
				_, _ = w.Write([]byte(`[2025, 2024]`))
			case "/api/servidor/ServidorGetNomeGrid":
				_, _ = w.Write([]byte(`{"recordsTotal":1,"data":[{"nome":"LUCAS FACIL","cpf":"44455566677","cargo":"Professor","salarioBruto":4300.0}]}`))
			case "/api/despesa/DespesasGetEmpenhoGrid/":
				_, _ = w.Write([]byte(`{"recordsTotal":1,"data":[{"numeroEmpenho":"101/2025","ano":2025,"fornecedor":"POSTO FACIL","cpfCnpj":"77788899900","valorEmpenhado":450.0}]}`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		transparencia.SetFacilBaseURL(srv.URL)
		defer transparencia.SetFacilBaseURL("")

		hub := newTestHub(eng, srv.Client())
		serv, err := hub.QueryServidores(ctx, "Alto Jequitibá", transparencia.QueryOptions{UF: "MG", OverrideURL: srv.URL + "/0144301", Ano: 2025, Limit: 5})
		if err != nil || serv.Plataforma != "facil_transparencia" || len(serv.Servidores) != 1 || serv.Servidores[0].CPFParcial != "***.555.666-**" {
			t.Fatalf("Hub facil_transparencia servidores inesperado: err=%v res=%+v", err, serv)
		}
		desp, err := hub.QueryDespesas(ctx, "Alto Jequitibá", transparencia.QueryOptions{UF: "MG", OverrideURL: srv.URL + "/0144301", Ano: 2025, Limit: 5})
		if err != nil || desp.Plataforma != "facil_transparencia" || len(desp.Despesas) != 1 || desp.Despesas[0].DocumentoFornecedor != "***.888.999-**" {
			t.Fatalf("Hub facil_transparencia despesas inesperado: err=%v res=%+v", err, desp)
		}
	})

	t.Run("publicsoft_api", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		const hashID = "0123456789abcdef0123456789abcdef01234567"
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/api/v1/portal/legado/pmcabedelo/despesa-orcamentaria":
				_, _ = w.Write([]byte(`{"status":"success","data":{"hash_id":"` + hashID + `","entidade":{"nome":"Prefeitura de Cabedelo"}}}`))
			case "/api/v1/catalogo/ugs/" + hashID + "/folha-config":
				_, _ = w.Write([]byte(`{"status":"success","data":{"modo":"integrado"}}`))
			case "/api/v1/servidores":
				_, _ = w.Write([]byte(`{"status":"success","data":{"data":[{"nome":"JULIA PUBLICSOFT","cpf":"12345678901","cargo":"Enfermeira","salario_bruto":5800.0}],"total":1}}`))
			case "/api/v1/contratos":
				_, _ = w.Write([]byte(`{"status":"success","data":{"data":[{"numero":"55/2025","exercicio":2025,"contratado":"TECH PB","valor":32000.0}],"total":1}}`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		transparencia.SetPublicsoftBaseURL(srv.URL)
		defer transparencia.SetPublicsoftBaseURL("")

		hub := newTestHub(eng, srv.Client())
		override := srv.URL + "/sistemas/ContabilidadePublica/pmcabedelo"
		serv, err := hub.QueryServidores(ctx, "Cabedelo", transparencia.QueryOptions{UF: "PB", OverrideURL: override, Ano: 2025, Limit: 5})
		if err != nil || serv.Plataforma != "publicsoft_api" || len(serv.Servidores) != 1 || serv.Servidores[0].CPFParcial != "***.456.789-**" {
			t.Fatalf("Hub publicsoft_api servidores inesperado: err=%v res=%+v", err, serv)
		}
		ct, err := hub.QueryContratos(ctx, "Cabedelo", transparencia.QueryOptions{UF: "PB", OverrideURL: override, Ano: 2025, Limit: 5})
		if err != nil || ct.Plataforma != "publicsoft_api" || len(ct.Contratos) != 1 || ct.Contratos[0].Valor != 32000.0 {
			t.Fatalf("Hub publicsoft_api contratos inesperado: err=%v res=%+v", err, ct)
		}
	})

	t.Run("tomeconta_appbr", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/router" {
				http.Redirect(w, r, "/carregaPortalPM.aspx?p_i=43&p_t=0", http.StatusFound)
				return
			}
			if r.URL.Path == "/carregaPortalPM.aspx" {
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte(`<html><body>Portal TomeConta</body></html>`))
				return
			}
			if r.URL.Path == "/api/apiDados.ashx" {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Query().Get("metodo") {
				case "retornaDespesa":
					_, _ = w.Write([]byte(`{"status":"ok","total":1,"dados":[{"FORNECEDOR":"FORNECEDOR TC","CPFCNPJ":"12345678901","EMPENHO":"90/2025","VALOR_EMPENHADO":1750.0}]}`))
				case "retornaReceita":
					_, _ = w.Write([]byte(`{"status":"ok","total":1,"dados":[{"CODIGO":"1.1.1","DESCRICAO":"IPTU TC","VALOR_PREVISTO":100000,"VALOR_ARCADADO":95000,"VALOR_ARRECADADO":95000}]}`))
				default:
					http.NotFound(w, r)
				}
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		transparencia.SetTomeContaBaseURL(srv.URL)
		defer transparencia.SetTomeContaBaseURL("")

		hub := newTestHub(eng, srv.Client())
		desp, err := hub.QueryDespesas(ctx, "Afogados da Ingazeira", transparencia.QueryOptions{UF: "PE", OverrideURL: srv.URL + "/router", Ano: 2025, Mes: 1, Limit: 5})
		if err != nil || desp.Plataforma != "tomeconta_appbr" || len(desp.Despesas) != 1 || desp.Despesas[0].DocumentoFornecedor != "***.456.789-**" {
			t.Fatalf("Hub tomeconta_appbr despesas inesperado: err=%v res=%+v", err, desp)
		}
		rec, err := hub.QueryReceitas(ctx, "Afogados da Ingazeira", transparencia.QueryOptions{UF: "PE", OverrideURL: srv.URL + "/router", Ano: 2025, Mes: 1, Limit: 5})
		if err != nil || rec.Plataforma != "tomeconta_appbr" || len(rec.Receitas) != 1 || rec.Receitas[0].ValorArrecadado != 95000 {
			t.Fatalf("Hub tomeconta_appbr receitas inesperado: err=%v res=%+v", err, rec)
		}
	})

	t.Run("cittaweb_api", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/transparencia/api/public/todasunidades":
				_, _ = w.Write([]byte(`[{"chave":339,"codigo":"001","descricao":"PREFEITURA MUNICIPAL DE ALMIRANTE"}]`))
			case "/transparencia/api/public/pessoal/remuneracao":
				_, _ = w.Write([]byte(`[{"nome":"ROSA CITTA","cpf":"11122233344","cargo":"Diretora","salarioBruto":7100.0}]`))
			case "/transparencia/api/public/contratos/lista":
				_, _ = w.Write([]byte(`[{"numero":"77/2025","ano":2025,"fornecedor":"OBRAS SUL","valor":125000.0}]`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		hub := newTestHub(eng, srv.Client())
		serv, err := hub.QueryServidores(ctx, "Almirante Tamandaré do Sul", transparencia.QueryOptions{UF: "RS", OverrideURL: srv.URL + "/citta/#/transparencia", Ano: 2025, Limit: 5})
		if err != nil || serv.Plataforma != "cittaweb_api" || len(serv.Servidores) != 1 || serv.Servidores[0].CPFParcial != "***.222.333-**" {
			t.Fatalf("Hub cittaweb_api servidores inesperado: err=%v res=%+v", err, serv)
		}
		ct, err := hub.QueryContratos(ctx, "Almirante Tamandaré do Sul", transparencia.QueryOptions{UF: "RS", OverrideURL: srv.URL + "/citta/#/transparencia", Ano: 2025, Limit: 5})
		if err != nil || ct.Plataforma != "cittaweb_api" || len(ct.Contratos) != 1 || ct.Contratos[0].Valor != 125000.0 {
			t.Fatalf("Hub cittaweb_api contratos inesperado: err=%v res=%+v", err, ct)
		}
	})

	t.Run("geosiap_api", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/portal-transparencia/api/sis_entidade":
				_, _ = w.Write([]byte(`[{"id_entidade":1,"ds_entidade":"PREFEITURA MUNICIPAL","ativo":"1","principal":"1"}]`))
			case "/portal-transparencia/api/execucao/despesas/programas_acoes":
				_, _ = w.Write([]byte(`[{"chave":"10.301.0010.2001","descricao":"SAUDE GEOSIAP","nivel":"Ação","vl_empenhado":6400.0,"vl_liquidado":6000.0,"vl_pago":5800.0}]`))
			case "/portal-transparencia/api/execucao/receitas/proprias/receitas_proprias":
				_, _ = w.Write([]byte(`[{"cd_plano_conta_rel":"1.1.1","ds_plano_conta":"IPTU GEOSIAP","vl_previsto":200000,"vl_total":180000}]`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		hub := newTestHub(eng, srv.Client())
		desp, err := hub.QueryDespesas(ctx, "Barra do Piraí", transparencia.QueryOptions{UF: "RJ", OverrideURL: srv.URL + "/portal-transparencia", Ano: 2025, Limit: 5})
		if err != nil || desp.Plataforma != "geosiap_api" || len(desp.Despesas) != 1 || desp.Despesas[0].ValorEmpenhado != 6400.0 {
			t.Fatalf("Hub geosiap_api despesas inesperado: err=%v res=%+v", err, desp)
		}
		rec, err := hub.QueryReceitas(ctx, "Barra do Piraí", transparencia.QueryOptions{UF: "RJ", OverrideURL: srv.URL + "/portal-transparencia", Ano: 2025, Limit: 5})
		if err != nil || rec.Plataforma != "geosiap_api" || len(rec.Receitas) != 1 || rec.Receitas[0].ValorArrecadado != 180000 {
			t.Fatalf("Hub geosiap_api receitas inesperado: err=%v res=%+v", err, rec)
		}
	})

	t.Run("multi24h_xml", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/multi24/webservices/transparencia/dados-abertos/receita/index" && r.Method == http.MethodPost {
				w.Header().Set("Content-Type", "application/xml")
				_, _ = w.Write([]byte(`<receitas><receita><receita>1.1.1</receita><descricao>IPTU MULTI24</descricao><previsao_inicial>50000</previsao_inicial><arrecadacao>48000</arrecadacao></receita></receitas>`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		hub := newTestHub(eng, srv.Client())
		rec, err := hub.QueryReceitas(ctx, "Salvador do Sul", transparencia.QueryOptions{UF: "RS", OverrideURL: srv.URL + "/multi24", Ano: 2025, Limit: 5})
		if err != nil || rec.Plataforma != "multi24h_xml" || len(rec.Receitas) != 1 || rec.Receitas[0].ValorArrecadado != 48000 {
			t.Fatalf("Hub multi24h_xml receitas inesperado: err=%v res=%+v", err, rec)
		}
	})

	t.Run("topsolutions_rn", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/receitaprevistaarrecadada/receitaprevistaarrecadadaasync":
				_, _ = w.Write([]byte(`{"data":[{"codCategoria":"1000","txtDescricaoCategoria":"Receitas RN","vlrPrevisaoAtualizado":500000,"vlrArrecadacao":450000}]}`))
			case "/Servidor/ServidorPorMesAnoAsync":
				_, _ = w.Write([]byte(`{"data":[{"nome":"MARIA TOP","cpf":"12345678901","cargo":"PROFESSORA","vlrRemuneracaoBruta":5200.5}]}`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		hub := newTestHub(eng, srv.Client())
		serv, err := hub.QueryServidores(ctx, "Angicos", transparencia.QueryOptions{UF: "RN", OverrideURL: srv.URL, Ano: 2025, Mes: 9, Limit: 5})
		if err != nil || serv.Plataforma != "topsolutions_rn" || len(serv.Servidores) != 1 || serv.Servidores[0].CPFParcial != "***.456.789-**" {
			t.Fatalf("Hub topsolutions_rn servidores inesperado: err=%v res=%+v", err, serv)
		}
		rec, err := hub.QueryReceitas(ctx, "Angicos", transparencia.QueryOptions{UF: "RN", OverrideURL: srv.URL, Ano: 2025, Limit: 5})
		if err != nil || rec.Plataforma != "topsolutions_rn" || len(rec.Receitas) != 1 || rec.Receitas[0].ValorArrecadado != 450000 {
			t.Fatalf("Hub topsolutions_rn receitas inesperado: err=%v res=%+v", err, rec)
		}
	})
}
