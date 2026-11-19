package transparencia_test

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Mcp-Brasil/mcp-brasil/pkg/transparencia"
)

func wrapPortalTPXML(jsonPayload string) []byte {
	type xmlString struct {
		XMLName xml.Name `xml:"http://tempuri.org/ string"`
		Value   string   `xml:",chardata"`
	}
	b, _ := xml.Marshal(xmlString{Value: jsonPayload})
	return b
}

func TestProbePortalTP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	t.Run("wsdl_with_browser_ua", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("User-Agent") != "Mozilla/5.0 (compatible; mcp-brasil/1.0)" {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			if r.URL.Path == "/api/transparencia.asmx" && strings.EqualFold(r.URL.RawQuery, "WSDL") {
				w.Header().Set("Content-Type", "text/xml")
				_, _ = w.Write([]byte(`<wsdl:definitions targetNamespace="http://tempuri.org/"><wsdl:operation name="json_empenhos"/></wsdl:definitions>`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if !transparencia.ProbePortalTP(ctx, client, srv.URL) {
			t.Fatalf("esperava ProbePortalTP=true via WSDL")
		}
		if !transparencia.ProbePortalTP(ctx, client, srv.URL+"/api/transparencia.asmx") {
			t.Fatalf("esperava ProbePortalTP=true mesmo com sufixo /api/transparencia.asmx")
		}
	})

	t.Run("orcamento_despesas_fallback", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("User-Agent") != "Mozilla/5.0 (compatible; mcp-brasil/1.0)" {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			if r.URL.Path == "/api/transparencia.asmx/json_orcamento_despesas" && r.URL.Query().Get("ano") == "2025" {
				w.Header().Set("Content-Type", "text/xml")
				_, _ = w.Write(wrapPortalTPXML(`[{"dotacao":"100.00"}]`))
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if !transparencia.ProbePortalTP(ctx, client, srv.URL) {
			t.Fatalf("esperava ProbePortalTP=true via json_orcamento_despesas")
		}
	})

	t.Run("non_portaltp", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer srv.Close()

		if transparencia.ProbePortalTP(ctx, client, srv.URL) {
			t.Fatalf("esperava ProbePortalTP=false para servidor não-PortalTP")
		}
	})
}

func TestPortalTPQueriesMock(t *testing.T) {
	var seenLicitacoesExtra, seenLicitacoesMes string
	var contratosPaginatedAttempts, contratosUnpaginatedAttempts int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "Mozilla/5.0 (compatible; mcp-brasil/1.0)" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")

		switch r.URL.Path {
		case "/api/transparencia.asmx/json_servidores":
			_, _ = w.Write(wrapPortalTPXML(`[
				{
					"nome": "CARLOS ALMEIDA",
					"matricula": "1020",
					"documento": "12345678901",
					"cargo": "Engenheiro Civil",
					"regime": "Estatutário",
					"situacao": "Ativo",
					"data_admissao": "2018-03-10",
					"horas_semanais": 40,
					"valor_rem_base": 5000.00,
					"valor_rem_vantagens": 1200.50,
					"valor_rem_descontos": 950.25,
					"valor_rem_liquida": 5250.25
				},
				{
					"nome": "MARIA SOUZA",
					"matricula": "2040",
					"documento": "987.654.321-00",
					"cargo": "Professora",
					"regime": "CLT",
					"situacao": "Ativo",
					"data_admissao": "2020-01-15",
					"horas_semanais": "30h",
					"valor_remuneracao": 4100.00
				}
			]`))

		case "/api/transparencia.asmx/json_empenhos":
			_, _ = w.Write(wrapPortalTPXML(`[
				{
					"empenho": "0045/2025",
					"ano": 2025,
					"data": "2025-02-10",
					"favorecido": "CONSTRUTORA MODELO LTDA",
					"documento_favorecido": "12.345.678/0001-99",
					"orgao": "Secretaria de Obras",
					"unidade": "Infraestrutura",
					"funcao": "Urbanismo",
					"elemento": "Obras e Instalações",
					"historico": "Pavimentação asfáltica",
					"valor_empenhado": 150000.00,
					"valor_liquidado": 100000.00,
					"valor_pago": 80000.00
				},
				{
					"empenho": "0046/2025",
					"ano": 2025,
					"data": "2025-02-12",
					"favorecido": "JOSE DA SILVA",
					"documento_favorecido": "11122233344",
					"orgao": "Secretaria de Saúde",
					"historico": "Serviço médico eventual",
					"valor_empenhado": 3500.00,
					"valor_liquidado": 3500.00,
					"valor_pago": 3500.00
				}
			]`))

		case "/api/transparencia.asmx/json_execucao_receitas":
			_, _ = w.Write(wrapPortalTPXML(`[
				{
					"codigo": "1.1.1.0.00",
					"descricao": "IPTU",
					"categoria": "Receitas Tributárias",
					"valor_previsto": 2500000.00,
					"valor_arrecadado": 2750000.50
				},
				{
					"codigo": "1.7.1.0.00",
					"descricao": "FPM",
					"categoria": "Transferências Correntes",
					"valor_previsto": 12000000.00,
					"valor_arrecadado": 11500000.00
				}
			]`))

		case "/api/transparencia.asmx/json_licitacoes":
			seenLicitacoesExtra = r.URL.Query().Get("extra")
			seenLicitacoesMes = r.URL.Query().Get("mes")
			_, _ = w.Write(wrapPortalTPXML(`[
				{
					"numero": "012/2025",
					"ano": 2025,
					"modalidade": "Pregão Eletrônico",
					"situacao": "Homologada",
					"data_abertura": "2025-03-01",
					"objeto": "Aquisição de medicamentos básicos",
					"valor_estimado": 400000.00,
					"valor_homologado": 365000.00
				}
			]`))

		case "/api/transparencia.asmx/json_contratos":
			// Simula servidor ASMX que rejeita parâmetros pagina/tamanho e exige fallback sem paginação
			if r.URL.Query().Get("pagina") != "" {
				contratosPaginatedAttempts++
				http.Error(w, "System.InvalidOperationException: pagina unexpected", http.StatusInternalServerError)
				return
			}
			contratosUnpaginatedAttempts++
			_, _ = w.Write(wrapPortalTPXML(`[
				{
					"numero": "089/2025",
					"ano": 2025,
					"fornecedor": "ALFA SERVICOS LTDA",
					"documento_fornecedor": "99.888.777/0001-66",
					"objeto": "Manutenção predial",
					"situacao": "Vigente",
					"data_assinatura": "2025-01-20",
					"vigencia_inicio": "2025-01-20",
					"vigencia_fim": "2025-12-31",
					"valor": 92000.00
				},
				{
					"numero": "090/2025",
					"ano": 2025,
					"fornecedor": "CONSULTOR INDIVIDUAL",
					"documento_fornecedor": "55566677788",
					"objeto": "Consultoria técnica",
					"valor": 12000.00
				}
			]`))

		case "/api/transparencia.asmx/json_diarias":
			_, _ = w.Write(wrapPortalTPXML(`[
				{
					"beneficiario": "ANA PAULA",
					"matricula": "301",
					"cargo": "Secretária de Educação",
					"destino": "Vitória - ES",
					"motivo": "Reunião estadual UNDIME",
					"data_saida": "2025-04-02",
					"data_chegada": "2025-04-04",
					"numero_diarias": 2.5,
					"valor": 1250.00
				}
			]`))

		case "/api/transparencia.asmx/json_obras":
			_, _ = w.Write(wrapPortalTPXML(`[
				{
					"codigo": "OB-007",
					"titulo": "Construção de Creche Municipal",
					"objeto": "Construção de creche padrão FNDE",
					"orgao": "Secretaria de Educação",
					"endereco": "Rua das Flores, 100",
					"bairro": "Centro",
					"data_inicial": "2025-01-10",
					"data_final": "2025-11-30",
					"situacao": "Em andamento",
					"valor": 2150000.00
				}
			]`))

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	m := &transparencia.Municipio{CodigoIBGE: "3205002", Nome: "Serra", UF: "ES"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}

	// 1. Servidores
	servRes, err := transparencia.QueryPortalTPServidores(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Ano:     2025,
		Mes:     3,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QueryPortalTPServidores erro: %v", err)
	}
	if servRes.Plataforma != "portaltp" || len(servRes.Servidores) != 2 {
		t.Fatalf("ServidoresResult inesperado: %+v", servRes)
	}
	s0 := servRes.Servidores[0]
	if s0.CPFParcial != "***.456.789-**" {
		t.Errorf("CPF não mascarado corretamente: %q", s0.CPFParcial)
	}
	if s0.SalarioBase != 5000.00 || s0.SalarioBruto != 6200.50 || s0.Descontos != 950.25 || s0.SalarioLiquido != 5250.25 {
		t.Errorf("remuneração valor_rem* inesperada: %+v", s0)
	}
	if s0.Vinculo != "Estatutário" || s0.CargaHoraria != "40" || s0.DataAdmissao != "2018-03-10" || s0.Competencia != "03/2025" {
		t.Errorf("metadados servidor[0] inesperados: %+v", s0)
	}
	s1 := servRes.Servidores[1]
	if s1.CPFParcial != "***.654.321-**" || s1.SalarioBruto != 4100.00 {
		t.Errorf("servidor[1] inesperado: %+v", s1)
	}

	// 2. Despesas
	despRes, err := transparencia.QueryPortalTPDespesas(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Ano:     2025,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QueryPortalTPDespesas erro: %v", err)
	}
	if len(despRes.Despesas) != 2 {
		t.Fatalf("esperava 2 despesas, obteve %d", len(despRes.Despesas))
	}
	if despRes.Despesas[0].DocumentoFornecedor != "12.345.678/0001-99" {
		t.Errorf("CNPJ deveria ser preservado: %q", despRes.Despesas[0].DocumentoFornecedor)
	}
	if despRes.Despesas[1].DocumentoFornecedor != "***.222.333-**" {
		t.Errorf("CPF em despesa deveria ser mascarado: %q", despRes.Despesas[1].DocumentoFornecedor)
	}

	// 3. Receitas (testa cap de Limit=1)
	recRes, err := transparencia.QueryPortalTPReceitas(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Ano:     2025,
		Limit:   1,
	})
	if err != nil {
		t.Fatalf("QueryPortalTPReceitas erro: %v", err)
	}
	if len(recRes.Receitas) != 1 || recRes.TotalRegistros != 2 || recRes.Receitas[0].Codigo != "1.1.1.0.00" || recRes.Receitas[0].ValorArrecadado != 2750000.50 {
		t.Errorf("ReceitasResult inesperado: %+v", recRes)
	}

	// 4. Licitações
	licRes, err := transparencia.QueryPortalTPLicitacoes(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Ano:     2025,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QueryPortalTPLicitacoes erro: %v", err)
	}
	if seenLicitacoesExtra != "true" || seenLicitacoesMes != "0" {
		t.Errorf("query params de licitações inesperados: extra=%q mes=%q", seenLicitacoesExtra, seenLicitacoesMes)
	}
	if len(licRes.Licitacoes) != 1 || licRes.Licitacoes[0].Numero != "012/2025" || licRes.Licitacoes[0].ValorHomologado != 365000.00 {
		t.Errorf("LicitacoesResult inesperado: %+v", licRes)
	}

	// 5. Contratos (testa fallback sem paginação e cap Limit=1)
	contRes, err := transparencia.QueryPortalTPContratos(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Ano:     2025,
		Limit:   1,
	})
	if err != nil {
		t.Fatalf("QueryPortalTPContratos erro: %v", err)
	}
	if contratosPaginatedAttempts == 0 || contratosUnpaginatedAttempts == 0 {
		t.Errorf("esperava tentativa paginada seguida de fallback não-paginado (pag=%d, unpag=%d)", contratosPaginatedAttempts, contratosUnpaginatedAttempts)
	}
	if len(contRes.Contratos) != 1 || contRes.Contratos[0].Numero != "089/2025" || contRes.Contratos[0].Valor != 92000.00 {
		t.Errorf("ContratosResult inesperado: %+v", contRes)
	}

	// 6. Diárias
	diaRes, err := transparencia.QueryPortalTPDiarias(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Ano:     2025,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QueryPortalTPDiarias erro: %v", err)
	}
	if len(diaRes.Diarias) != 1 || diaRes.Diarias[0].Beneficiario != "ANA PAULA" || diaRes.Diarias[0].NumeroDiarias != 2.5 || diaRes.Diarias[0].Valor != 1250.00 {
		t.Errorf("DiariasResult inesperado: %+v", diaRes)
	}

	// 7. Obras
	obraRes, err := transparencia.QueryPortalTPObras(ctx, client, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Ano:     2025,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("QueryPortalTPObras erro: %v", err)
	}
	if len(obraRes.Obras) != 1 || obraRes.Obras[0].Codigo != "OB-007" || obraRes.Obras[0].Valor != 2150000.00 {
		t.Errorf("ObrasResult inesperado: %+v", obraRes)
	}
}

func TestPortalTPRejectsWithoutBrowserUA(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "Mozilla/5.0 (compatible; mcp-brasil/1.0)" {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}
		_, _ = fmt.Fprint(w, string(wrapPortalTPXML(`[]`)))
	}))
	defer srv.Close()

	ctx := context.Background()
	m := &transparencia.Municipio{CodigoIBGE: "3205002", Nome: "Serra", UF: "ES"}
	res, err := transparencia.QueryPortalTPServidores(ctx, nil, m, transparencia.PortalQueryParams{
		BaseURL: srv.URL,
		Ano:     2025,
	})
	if err != nil {
		t.Fatalf("não deveria retornar 403 com UA browser: %v", err)
	}
	if !res.Disponivel || len(res.Servidores) != 0 {
		t.Fatalf("resultado inesperado para lista vazia: %+v", res)
	}
}
