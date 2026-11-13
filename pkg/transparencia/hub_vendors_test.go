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

	"github.com/gibavargas/mcp-brasil/pkg/transparencia"
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
