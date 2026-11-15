package transparencia_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Mcp-Brasil/mcp-brasil/pkg/transparencia"
)

func TestAgiliConnectorMock(t *testing.T) {
	const shortname = "pm_cascavel_pr"

	var seletorCalls atomic.Int32
	var seenUCHeader string
	var seenServidoresBody map[string]any
	var seenDespesasBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/seletor/get" {
			seletorCalls.Add(1)
			_, _ = w.Write([]byte(`[
				{"shortname": "cm_cascavel_pr", "municipio": "Cascavel", "uf": "PR ", "nomeRazaoSocial": "Câmara Municipal de Cascavel"},
				{"shortname": "` + shortname + `", "municipio": "Cascavel", "uf": "PR ", "nomeRazaoSocial": "Prefeitura Municipal de Cascavel"},
				{"shortname": "pm_sorriso_mt", "municipio": "Sorriso", "uf": "MT", "nomeRazaoSocial": "Prefeitura Municipal de Sorriso"}
			]`))
			return
		}

		seenUCHeader = r.Header.Get("uc")
		if seenUCHeader != shortname {
			http.Error(w, "missing or invalid uc header", http.StatusBadRequest)
			return
		}

		switch r.URL.Path {
		case "/filtro/get/":
			model := r.URL.Query().Get("model")
			switch model {
			case "Agili.Blue.Portal.Shared.GestaoPessoas.Dto.Servidores.ServidoresGridDto":
				// Simula ausência no /filtro/get/ para validar fallback de endpoint
				http.NotFound(w, r)
			case "Agili.Blue.Portal.Shared.Contabilidade.Dto.Despesas.EmpenhosPorExercicio.EmpenhosPorExercicioGridDto":
				_, _ = w.Write([]byte(`{
					"urlStore": "/contabilidade/despesas/customEmpenhos",
					"fieldsSelected": ["NumeroEmpenho", "Fornecedor", "ValorEmpenhado"]
				}`))
			case "Agili.Blue.Portal.Shared.Contabilidade.Dto.Receitas.ReceitasPorPeriodoGridDto":
				_, _ = w.Write([]byte(`{
					"urlStore": "/contabilidade/receitas/customReceitas",
					"fieldsSelected": ["Codigo", "Descricao", "ValorArrecadado"]
				}`))
			case "Agili.Blue.Portal.Shared.ComprasLicitacoes.Dto.Licitacoes.LicitacaoGridDto":
				_, _ = w.Write([]byte(`{
					"urlStore": "",
					"fieldsSelected": ["Numero", "Objeto"]
				}`))
			case "Agili.Blue.Portal.Shared.ComprasLicitacoes.Dto.Contratos.ContratoGridDto":
				_, _ = w.Write([]byte(`{
					"urlStore": "/compraslicitacoes/contratos/customContratos",
					"fieldsSelected": ["Numero", "Fornecedor", "Valor"]
				}`))
			case "Agili.Blue.Transparencia.Shared.ComprasLicitacoes.Dto.Obras.ObrasPublicasGridDto":
				_, _ = w.Write([]byte(`{
					"urlStore": "/compraslicitacoes/obras/customObras",
					"fieldsSelected": ["Codigo", "Titulo", "Valor"]
				}`))
			default:
				http.NotFound(w, r)
			}

		case "/gestaopessoas/servidores/obterDadosServidores/":
			bodyBytes, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(bodyBytes, &seenServidoresBody)
			if r.URL.Query().Get("page") != "0" || r.URL.Query().Get("withCount") != "true" {
				http.Error(w, "bad query params", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{
				"total": 2,
				"dados": [
					{
						"Nome": "FERNANDA SOUZA",
						"Matricula": "8821",
						"CPF": "11122233344",
						"Cargo": "Médica",
						"Lotacao": "Saúde",
						"Vinculo": "Efetivo",
						"Situacao": "Ativo",
						"SalarioBase": 9000.00,
						"SalarioBruto": 12000.00,
						"Descontos": 2500.00,
						"SalarioLiquido": 9500.00,
						"Competencia": "04/2025"
					},
					{
						"nome": "LUCAS MENDES",
						"matricula": "8822",
						"cpfFormatado": "***.555.666-**",
						"cargo": "Engenheiro",
						"salarioBruto": 8000.00,
						"descontos": 1000.00
					}
				]
			}`))

		case "/contabilidade/despesas/customEmpenhos/":
			bodyBytes, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(bodyBytes, &seenDespesasBody)
			_, _ = w.Write([]byte(`{
				"totalCount": 2,
				"data": [
					{
						"NumeroEmpenho": "450/2025",
						"Ano": 2025,
						"Data": "2025-03-10",
						"Fornecedor": "PEDRO ALMEIDA",
						"CpfCnpj": "32165498700",
						"Orgao": "Secretaria de Obras",
						"Historico": "Serviços de manutenção",
						"ValorEmpenhado": 3200.00,
						"ValorLiquidado": 3200.00,
						"ValorPago": 3200.00
					},
					{
						"NumeroEmpenho": "451/2025",
						"Ano": 2025,
						"Fornecedor": "CONSTRUTORA BETA LTDA",
						"CpfCnpj": "45.678.901/0001-23",
						"ValorEmpenhado": 50000.00
					}
				]
			}`))

		case "/contabilidade/receitas/customReceitas/":
			_, _ = w.Write([]byte(`{
				"total": 1,
				"dados": [
					{
						"Codigo": "1.7.1.0",
						"Descricao": "FPM - Cota-Parte",
						"Categoria": "Transferências Correntes",
						"Data": "2025-03-31",
						"ValorPrevisto": 2500000.00,
						"ValorArrecadado": 2100000.00
					}
				]
			}`))

		case "/compraslicitacoes/licitacoes/licitacao/obterDadosLicitacao/":
			_, _ = w.Write([]byte(`{
				"total": 1,
				"dados": [
					{
						"Numero": "015/2025",
						"Ano": 2025,
						"Modalidade": "Concorrência",
						"Situacao": "Em Andamento",
						"DataAbertura": "2025-04-02",
						"Objeto": "Pavimentação asfáltica",
						"ValorEstimado": 1200000.00,
						"ValorHomologado": 1150000.00
					}
				]
			}`))

		case "/compraslicitacoes/contratos/customContratos/":
			_, _ = w.Write([]byte(`{
				"total": 1,
				"dados": [
					{
						"Numero": "102/2025",
						"Ano": 2025,
						"Fornecedor": "ROBERTO DIAS",
						"Documento": "555.666.777-88",
						"Objeto": "Locação de imóvel",
						"Situacao": "Ativo",
						"DataAssinatura": "2025-02-01",
						"VigenciaInicio": "2025-02-01",
						"VigenciaFim": "2026-01-31",
						"Valor": 36000.00
					}
				]
			}`))

		case "/compraslicitacoes/obras/customObras/":
			_, _ = w.Write([]byte(`{
				"total": 1,
				"dados": [
					{
						"Codigo": "OB-77",
						"Titulo": "Construção da UBS Central",
						"Objeto": "Construção de Unidade Básica de Saúde",
						"Orgao": "Secretaria de Saúde",
						"Endereco": "Av. Brasil, 1000",
						"Bairro": "Centro",
						"DataInicial": "2025-01-10",
						"DataFinal": "2025-12-20",
						"Situacao": "Em Execução",
						"Valor": 980000.00
					}
				]
			}`))

		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	transparencia.SetAgiliAPIBaseURL(srv.URL)
	t.Cleanup(func() {
		transparencia.SetAgiliAPIBaseURL("https://transparencia.agilicloud.com.br/api")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 3 * time.Second}
	m := &transparencia.Municipio{CodigoIBGE: "4104808", Nome: "Cascavel", UF: "PR"}

	// 1. Discovery (+ cache do /seletor/get)
	disc, ok := transparencia.DiscoverAgiliByMunicipio(ctx, client, m)
	if !ok || disc == nil {
		t.Fatalf("esperava DiscoverAgiliByMunicipio=true")
	}
	if disc.Plataforma != "agili_blue" ||
		disc.URLResolvida != "https://transparencia.agilicloud.com.br" ||
		disc.TenantID != shortname ||
		disc.EntidadeNome != "Prefeitura Municipal de Cascavel" ||
		disc.FonteDescoberta != "agili_directory" ||
		!disc.ServidoresSuportado {
		t.Fatalf("PortalDiscoveryResult inesperado: %+v", disc)
	}

	// Segunda busca deve usar cache em memória
	if _, ok := transparencia.DiscoverAgiliByMunicipio(ctx, client, &transparencia.Municipio{Nome: "Sorriso", UF: "MT"}); !ok {
		t.Fatalf("esperava encontrar Sorriso/MT no cache")
	}
	if seletorCalls.Load() != 1 {
		t.Errorf("esperava /seletor/get cacheado em 1 chamada, obteve %d", seletorCalls.Load())
	}

	if _, ok := transparencia.DiscoverAgiliByMunicipio(ctx, client, &transparencia.Municipio{Nome: "Cidade Inexistente", UF: "SP"}); ok {
		t.Fatalf("esperava false para município não cadastrado")
	}

	params := transparencia.PortalQueryParams{
		BaseURL:  disc.URLResolvida,
		TenantID: disc.TenantID,
		Ano:      2025,
		Limit:    15,
	}

	// 2. Servidores (usa fallback URLStore porque /filtro/get/ retornou 404)
	servRes, err := transparencia.QueryAgiliServidores(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryAgiliServidores erro: %v", err)
	}
	if seenUCHeader != shortname {
		t.Errorf("header uc inesperado: %q", seenUCHeader)
	}
	if anoVal, ok := seenServidoresBody["Ano"].(float64); !ok || int(anoVal) != 2025 {
		t.Errorf("esperava filtro Ano=2025 no POST body, obteve %+v", seenServidoresBody)
	}
	if len(servRes.Servidores) != 2 || servRes.TotalRegistros != 2 {
		t.Fatalf("ServidoresResult inesperado: %+v", servRes)
	}
	if servRes.Servidores[0].CPFParcial != "***.222.333-**" {
		t.Errorf("CPF não mascarado corretamente: %q", servRes.Servidores[0].CPFParcial)
	}
	if servRes.Servidores[1].SalarioLiquido != 7000.00 {
		t.Errorf("esperava SalarioLiquido=7000, obteve %v", servRes.Servidores[1].SalarioLiquido)
	}

	// 3. Despesas (usa urlStore + fieldsSelected do /filtro/get/ e mascara CPF)
	despRes, err := transparencia.QueryAgiliDespesas(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryAgiliDespesas erro: %v", err)
	}
	if _, hasFields := seenDespesasBody["fieldsSelected"]; !hasFields {
		t.Errorf("esperava fieldsSelected no POST body de despesas: %+v", seenDespesasBody)
	}
	if len(despRes.Despesas) != 2 {
		t.Fatalf("DespesasResult inesperado: %+v", despRes)
	}
	if despRes.Despesas[0].DocumentoFornecedor != "***.654.987-**" {
		t.Errorf("CPF do fornecedor não mascarado: %q", despRes.Despesas[0].DocumentoFornecedor)
	}
	if despRes.Despesas[1].DocumentoFornecedor != "45.678.901/0001-23" {
		t.Errorf("CNPJ não deveria ser mascarado: %q", despRes.Despesas[1].DocumentoFornecedor)
	}

	// 4. Receitas
	recRes, err := transparencia.QueryAgiliReceitas(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryAgiliReceitas erro: %v", err)
	}
	if len(recRes.Receitas) != 1 || recRes.Receitas[0].ValorArrecadado != 2100000.00 {
		t.Errorf("ReceitasResult inesperado: %+v", recRes)
	}

	// 5. Licitações (urlStore vazio no /filtro/get/, usa fallback /compraslicitacoes/licitacoes/licitacao/obterDadosLicitacao)
	licRes, err := transparencia.QueryAgiliLicitacoes(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryAgiliLicitacoes erro: %v", err)
	}
	if len(licRes.Licitacoes) != 1 || licRes.Licitacoes[0].Numero != "015/2025" {
		t.Errorf("LicitacoesResult inesperado: %+v", licRes)
	}

	// 6. Contratos
	contRes, err := transparencia.QueryAgiliContratos(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryAgiliContratos erro: %v", err)
	}
	if len(contRes.Contratos) != 1 || contRes.Contratos[0].DocumentoFornecedor != "***.666.777-**" {
		t.Errorf("ContratosResult inesperado: %+v", contRes)
	}

	// 7. Obras
	obrasRes, err := transparencia.QueryAgiliObras(ctx, client, m, params)
	if err != nil {
		t.Fatalf("QueryAgiliObras erro: %v", err)
	}
	if len(obrasRes.Obras) != 1 || obrasRes.Obras[0].Codigo != "OB-77" || obrasRes.Obras[0].Valor != 980000.00 {
		t.Errorf("ObrasResult inesperado: %+v", obrasRes)
	}
}
