package transparencia_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Mcp-Brasil/mcp-brasil/pkg/transparencia"
)

func TestProbeDiretorioDigital(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	t.Run("swagger_yaml_with_cliente", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/dados-abertos/swagger.yaml" {
				w.Header().Set("Content-Type", "text/yaml")
				_, _ = w.Write([]byte(`
openapi: 3.0.0
servers:
  - url: https://transparencia.diretoriodigital.inf.br/api
paths:
  /listagem-servidores:
    get:
      parameters:
        - name: cliente
          in: query
          schema:
            type: string
            default: pm_exemplo_go
`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		slug, ok := transparencia.ProbeDiretorioDigital(ctx, client, srv.URL)
		if !ok || slug != "pm_exemplo_go" {
			t.Fatalf("esperava ('pm_exemplo_go', true), obteve (%q, %v)", slug, ok)
		}
	})

	t.Run("html_with_api_link", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte(`<html><a href="https://transparencia.diretoriodigital.inf.br/api/listagem-servidores?cliente=cidade_sistp">Servidores</a></html>`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		slug, ok := transparencia.ProbeDiretorioDigital(ctx, client, srv.URL)
		if !ok || slug != "cidade_sistp" {
			t.Fatalf("esperava ('cidade_sistp', true), obteve (%q, %v)", slug, ok)
		}
	})

	t.Run("non_diretorio", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if slug, ok := transparencia.ProbeDiretorioDigital(ctx, client, srv.URL); ok || slug != "" {
			t.Fatalf("esperava ('', false), obteve (%q, %v)", slug, ok)
		}
	})
}

func TestQueryDiretorioDigitalServidores(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "5205109", Nome: "Catalão", UF: "GO"}

	t.Run("auto_detect_month_and_mask_cpf", func(t *testing.T) {
		prev := time.Now().AddDate(0, -1, 0)
		var seenCliente string

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/listagem-servidores" {
				seenCliente = r.URL.Query().Get("cliente")
				mes := r.URL.Query().Get("mes")
				ano := r.URL.Query().Get("ano")
				w.Header().Set("Content-Type", "application/json")
				if ano == strconv.Itoa(prev.Year()) && mes == strconv.Itoa(int(prev.Month())) {
					_, _ = w.Write([]byte(`[
						{
							"nome": "ANA PAULA LIMA",
							"matricula": "3310",
							"cpf": "11122233344",
							"cargo": "Enfermeira",
							"lotacao": "Secretaria de Saúde",
							"vinculo": "Estatutário",
							"situacao": "Ativo",
							"bruto": "7.250,40",
							"desconto": "1.250,40",
							"liquido": "6.000,00"
						},
						{
							"nome": "PEDRO SANTOS",
							"matricula": "3311",
							"cpf": "55566677788",
							"cargo": "Vigia",
							"bruto": 2100.00
						}
					]`))
					return
				}
				_, _ = w.Write([]byte(`[]`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		res, err := transparencia.QueryDiretorioDigitalServidores(ctx, client, m, transparencia.PortalQueryParams{
			BaseURL:    srv.URL,
			NomeFiltro: "ANA",
			Limit:      10,
		}, "pm_catalao")
		if err != nil {
			t.Fatalf("QueryDiretorioDigitalServidores erro: %v", err)
		}
		if seenCliente != "pm_catalao" {
			t.Errorf("esperava cliente='pm_catalao', obteve %q", seenCliente)
		}
		if res.Plataforma != "diretoriodigital_sistp" || len(res.Servidores) != 1 {
			t.Fatalf("ServidoresResult inesperado: %+v", res)
		}
		s0 := res.Servidores[0]
		if s0.CPFParcial != "***.222.333-**" {
			t.Errorf("CPF não mascarado corretamente: %q", s0.CPFParcial)
		}
		if s0.SalarioBruto != 7250.40 || s0.Descontos != 1250.40 || s0.SalarioLiquido != 6000.00 {
			t.Errorf("valores salariais inesperados: %+v", s0)
		}
	})

	t.Run("override_api_base_for_external_portal", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/listagem-servidores" && r.URL.Query().Get("cliente") == "slug_ext" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"dados":[{"nome":"LUCAS ROCHA","cpf":"12345678901","cargo":"Técnico","bruto":3500}]}`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()
		transparencia.SetDiretorioDigitalBaseURL(srv.URL)
		defer transparencia.SetDiretorioDigitalBaseURL("")

		res, err := transparencia.QueryDiretorioDigitalServidores(ctx, client, m, transparencia.PortalQueryParams{
			BaseURL: "https://transparencia.municipio.go.gov.br",
			Ano:     2025,
			Mes:     9,
		}, "slug_ext")
		if err != nil {
			t.Fatalf("QueryDiretorioDigitalServidores external base erro: %v", err)
		}
		if len(res.Servidores) != 1 || res.Servidores[0].CPFParcial != "***.456.789-**" {
			t.Fatalf("resultado inesperado: %+v", res)
		}
	})
}

type rewriteTransport struct {
	handler    http.HandlerFunc
	lastScheme string
	lastHost   string
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.lastScheme = req.URL.Scheme
	rt.lastHost = req.URL.Host
	rec := httptest.NewRecorder()
	rt.handler(rec, req)
	return rec.Result(), nil
}

func TestProbeAndQueryInfBr(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("probe_and_https_to_http_rewrite", func(t *testing.T) {
		rt := &rewriteTransport{
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost && r.URL.Path == "/prefeitura/servidores/pe_remuneracao_servidores_ajax.php" {
					_ = r.ParseForm()
					if r.Form.Get("action") == "ultima_competencia" {
						w.Header().Set("Content-Type", "application/json")
						_, _ = w.Write([]byte(`{"ano":2025,"mes":11}`))
						return
					}
				}
				http.NotFound(w, r)
			},
		}
		client := &http.Client{Transport: rt, Timeout: 3 * time.Second}

		portalBase, ok := transparencia.ProbeInfBr(ctx, client, "https://portaldatransparencia.inf.br/prefeitura")
		if !ok {
			t.Fatalf("esperava ProbeInfBr=true")
		}
		if portalBase != "http://portaldatransparencia.inf.br/prefeitura" {
			t.Errorf("esperava portalBase reescrito para http://, obteve %q", portalBase)
		}
		if rt.lastScheme != "http" || rt.lastHost != "portaldatransparencia.inf.br" {
			t.Errorf("esperava chamada HTTP para portaldatransparencia.inf.br, obteve %s://%s", rt.lastScheme, rt.lastHost)
		}
	})

	t.Run("probe_non_infbr", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer srv.Close()
		client := &http.Client{Timeout: 3 * time.Second}

		if base, ok := transparencia.ProbeInfBr(ctx, client, srv.URL); ok || base != "" {
			t.Fatalf("esperava ('', false), obteve (%q, %v)", base, ok)
		}
	})

	t.Run("query_servidores_pagination_and_fields", func(t *testing.T) {
		var seenActions []string
		var seenPages []string

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/servidores/pe_remuneracao_servidores_ajax.php" || r.Method != http.MethodPost {
				http.NotFound(w, r)
				return
			}
			_ = r.ParseForm()
			action := r.Form.Get("action")
			seenActions = append(seenActions, action)
			w.Header().Set("Content-Type", "application/json")

			switch action {
			case "ultima_competencia":
				_, _ = w.Write([]byte(`{"ano":"2025","mes":"08"}`))
			case "listar":
				if r.Form.Get("ano") != "2025" || r.Form.Get("mes") != "08" {
					t.Errorf("esperava ano=2025 mes=08, obteve ano=%q mes=%q", r.Form.Get("ano"), r.Form.Get("mes"))
				}
				page := r.Form.Get("page")
				seenPages = append(seenPages, page)
				switch page {
				case "1":
					_, _ = w.Write([]byte(`{
						"dados": [
							{
								"nome": "ROBERTO ALVES",
								"Matricula": "001122",
								"cpf": "12345678901",
								"cargo": "Contador",
								"lotacao": "Finanças",
								"situacao": "Ativo",
								"vinculo": "Efetivo",
								"bruto": "8.400,00",
								"desconto": "1.900,00",
								"liquido": "6.500,00"
							},
							{
								"nome": "FERNANDA DIAS",
								"Matricula": "001123",
								"cpf": "98765432100",
								"cargo": "Médica",
								"lotacao": "Saúde",
								"situacao": "Ativo",
								"vinculo": "Contratado",
								"bruto": "12.000,00",
								"desconto": "2.500,00",
								"liquido": "9.500,00"
							}
						],
						"total": 3,
						"total_pages": 2
					}`))
				case "2":
					_, _ = w.Write([]byte(`{
						"dados": [
							{
								"nome": "PAULO HENRIQUE",
								"Matricula": "001124",
								"cpf": "11144477799",
								"cargo": "Auxiliar",
								"lotacao": "Obras",
								"situacao": "Ativo",
								"vinculo": "Comissionado",
								"bruto": "2.500,00",
								"desconto": "300,00",
								"liquido": "2.200,00"
							}
						],
						"total": 3,
						"total_pages": 2
					}`))
				default:
					_, _ = w.Write([]byte(`{"dados":[]}`))
				}
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		client := &http.Client{Timeout: 3 * time.Second}
		m := &transparencia.Municipio{CodigoIBGE: "2601201", Nome: "Arcoverde", UF: "PE"}

		res, err := transparencia.QueryInfBrServidores(ctx, client, m, transparencia.PortalQueryParams{
			BaseURL: srv.URL,
			Limit:   3,
		})
		if err != nil {
			t.Fatalf("QueryInfBrServidores erro: %v", err)
		}
		if res.Plataforma != "infbr_netuse" || res.TotalRegistros != 3 || len(res.Servidores) != 3 {
			t.Fatalf("ServidoresResult inesperado: %+v", res)
		}
		if strings.Join(seenActions, ",") != "ultima_competencia,listar,listar" {
			t.Errorf("sequência de actions inesperada: %v", seenActions)
		}
		if strings.Join(seenPages, ",") != "1,2" {
			t.Errorf("paginação inesperada: %v", seenPages)
		}
		s0 := res.Servidores[0]
		if s0.CPFParcial != "***.456.789-**" ||
			s0.Matricula != "001122" ||
			s0.Cargo != "Contador" ||
			s0.Lotacao != "Finanças" ||
			s0.Situacao != "Ativo" ||
			s0.Vinculo != "Efetivo" ||
			s0.SalarioBruto != 8400.00 ||
			s0.Descontos != 1900.00 ||
			s0.SalarioLiquido != 6500.00 ||
			s0.Competencia != "08/2025" {
			t.Errorf("mapeamento de servidor[0] inesperado: %+v", s0)
		}
	})
}
