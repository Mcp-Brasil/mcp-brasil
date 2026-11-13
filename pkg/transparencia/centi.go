package transparencia

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Centi Soluções (plataforma = "centi_api")
// Base padrão: https://api.centi.com.br
// ---------------------------------------------------------------------------

const defaultCentiAPIBaseURL = "https://api.centi.com.br"

var (
	centiURLMu      sync.RWMutex
	centiAPIBaseURL = defaultCentiAPIBaseURL
)

// SetCentiAPIBaseURL configura a URL base da API Centi (útil para testes).
func SetCentiAPIBaseURL(u string) {
	centiURLMu.Lock()
	defer centiURLMu.Unlock()
	trimmed := strings.TrimRight(strings.TrimSpace(u), "/")
	if trimmed == "" {
		centiAPIBaseURL = defaultCentiAPIBaseURL
		return
	}
	centiAPIBaseURL = trimmed
}

func getCentiAPIBaseURL() string {
	centiURLMu.RLock()
	defer centiURLMu.RUnlock()
	return strings.TrimRight(strings.TrimSpace(centiAPIBaseURL), "/")
}

func isMockOrCentiAPIURL(rawURL string) bool {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if strings.EqualFold(u.Scheme, "http") {
		return true
	}
	return host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.Contains(host, "api.centi.com.br")
}

func resolveCentiBaseURL(p PortalQueryParams) string {
	base := getCentiAPIBaseURL()
	if base == defaultCentiAPIBaseURL && strings.TrimSpace(p.BaseURL) != "" && isMockOrCentiAPIURL(p.BaseURL) {
		return strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	}
	return base
}

func dedupeNonEmptyStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		s := strings.TrimSpace(item)
		if s == "" {
			continue
		}
		if _, exists := seen[s]; exists {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func waitCentiRetry(ctx context.Context) error {
	timer := time.NewTimer(50 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func doCentiGetWithRetry(ctx context.Context, client *http.Client, targetURL string) ([]byte, int, error) {
	hc := portalConnectorHTTPClient(client)
	body, status, err := doJSONGet(ctx, hc, targetURL)
	if err == nil && status == http.StatusBadGateway {
		if wErr := waitCentiRetry(ctx); wErr != nil {
			return body, status, wErr
		}
		return doJSONGet(ctx, hc, targetURL)
	}
	return body, status, err
}

func doCentiPostWithRetry(ctx context.Context, client *http.Client, targetURL string, payload any) ([]byte, int, error) {
	hc := portalConnectorHTTPClient(client)
	body, status, err := doPortalJSONPost(ctx, hc, targetURL, payload)
	if err == nil && status == http.StatusBadGateway {
		if wErr := waitCentiRetry(ctx); wErr != nil {
			return body, status, wErr
		}
		return doPortalJSONPost(ctx, hc, targetURL, payload)
	}
	return body, status, err
}

// ProbeCenti testa GET /portal/getlistaorgaos/{UF}/{tenant} com slugs candidatos
// (SlugifyCompact, slug original, SlugifyNoPrepositions) e 1 retry em HTTP 502 transitório.
func ProbeCenti(ctx context.Context, client *http.Client, uf, tenantSlug string) (resolvedTenant string, ok bool) {
	ufNorm := strings.ToUpper(strings.TrimSpace(uf))
	rawTenant := strings.TrimSpace(tenantSlug)
	if ufNorm == "" || rawTenant == "" {
		return "", false
	}

	rawCandidates := []string{
		SlugifyCompact(rawTenant),
	}
	if !strings.ContainsAny(rawTenant, " \t\r\n") {
		rawCandidates = append(rawCandidates, strings.ToLower(rawTenant))
	}
	rawCandidates = append(rawCandidates,
		Slugify(rawTenant),
		SlugifyNoPrepositions(rawTenant),
	)
	candidates := dedupeNonEmptyStrings(rawCandidates)

	base := getCentiAPIBaseURL()
	for _, cand := range candidates {
		target := fmt.Sprintf("%s/portal/getlistaorgaos/%s/%s", base, url.PathEscape(ufNorm), url.PathEscape(cand))
		body, status, err := doCentiGetWithRetry(ctx, client, target)
		if err != nil || status != http.StatusOK {
			continue
		}
		trimmed := strings.TrimSpace(string(body))
		if !strings.HasPrefix(trimmed, "[") {
			continue
		}
		var orgaos []map[string]any
		if err := json.Unmarshal([]byte(trimmed), &orgaos); err != nil || len(orgaos) == 0 {
			continue
		}
		for _, org := range orgaos {
			if mapGetStr(org, "Id", "id") != "" || mapGetStr(org, "Nome", "nome") != "" {
				return cand, true
			}
		}
	}
	return "", false
}

func resolveCentiTarget(m *Municipio, p PortalQueryParams) (baseURL, uf, tenant string, err error) {
	baseURL = resolveCentiBaseURL(p)
	if m != nil {
		uf = strings.ToUpper(strings.TrimSpace(m.UF))
	}
	if uf == "" {
		return "", "", "", fmt.Errorf("Centi: UF do município não informada")
	}

	tenant = strings.TrimSpace(p.TenantID)
	if tenant == "" && m != nil {
		tenant = SlugifyCompact(m.Nome)
		if tenant == "" {
			tenant = strings.TrimSpace(m.Slug)
		}
	}
	if tenant == "" {
		return "", "", "", fmt.Errorf("Centi: TenantID não informado")
	}
	return baseURL, uf, tenant, nil
}

func resolveCentiYearAndMonth(p PortalQueryParams) (ano, mes int) {
	now := time.Now()
	ano = p.Ano
	if ano <= 0 {
		ano = parseYearFromDateStr(p.Inicio)
	}
	if ano <= 0 {
		ano = now.Year()
	}
	mes = p.Mes
	if mes <= 0 || mes > 12 {
		if ano < now.Year() {
			mes = 12
		} else {
			mes = int(now.Month())
		}
	}
	return ano, mes
}

func resolveCentiOrgao(p PortalQueryParams) int {
	if p.CodAdministracao > 0 {
		return p.CodAdministracao
	}
	return 1
}

// QueryCentiServidores consulta servidores/funcionários na API Centi (POST /portal/getfuncionarios/{UF}/{Tenant}).
func QueryCentiServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ServidoresResult, error) {
	base, uf, tenant, err := resolveCentiTarget(m, p)
	if err != nil {
		return nil, err
	}

	ano, mes := resolveCentiYearAndMonth(p)
	limit := normalizeLimit(p.Limit)
	orgaoID := resolveCentiOrgao(p)

	payload := map[string]any{
		"ano":     ano,
		"mes":     mes,
		"idOrgao": orgaoID,
		"page": map[string]any{
			"size":   limit,
			"number": 1,
		},
	}

	target := fmt.Sprintf("%s/portal/getfuncionarios/%s/%s", base, url.PathEscape(uf), url.PathEscape(tenant))
	body, status, err := doCentiPostWithRetry(ctx, client, target, payload)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Centi /portal/getfuncionarios retornou HTTP %d", status)
	}

	rows, total, err := decodeFlexiblePortalList(body, "funcionarios", "Funcionarios", "lista")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Centi /portal/getfuncionarios: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ServidorItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		nome := mapGetStr(r, "Nome", "nome", "NomeServidor", "nomeServidor", "Funcionario")
		if filtro != "" && !strings.Contains(strings.ToLower(nome), filtro) {
			continue
		}
		if rowTot := int(mapGetFloat(r, "TotalRegistros", "totalRegistros", "total")); rowTot > total {
			total = rowTot
		}
		comp := mapGetStr(r, "Competencia", "competencia", "MesAno", "mesAno")
		if comp == "" {
			comp = fmt.Sprintf("%02d/%04d", mes, ano)
		}

		items = append(items, ServidorItem{
			Nome:           nome,
			Matricula:      mapGetStr(r, "Matricula", "matricula", "Registro", "Codigo"),
			CPFParcial:     maskCPF(mapGetStr(r, "CpfFormatado", "cpfFormatado", "CpfParcial"), mapGetStr(r, "Cpf", "CPF", "cpf", "Documento")),
			Cargo:          mapGetStr(r, "Cargo", "cargo", "DescricaoCargo", "NomeCargo"),
			Funcao:         mapGetStr(r, "Funcao", "funcao", "DescricaoFuncao"),
			Lotacao:        mapGetStr(r, "Lotacao", "lotacao", "Departamento", "Unidade", "Setor"),
			LocalTrabalho:  mapGetStr(r, "LocalTrabalho", "localTrabalho", "Local"),
			Vinculo:        mapGetStr(r, "Vinculo", "vinculo", "Regime", "TipoVinculo"),
			Situacao:       mapGetStr(r, "Situacao", "situacao", "Status"),
			SalarioBase:    mapGetFloat(r, "SalarioBase", "salarioBase", "VencimentoBase", "Base"),
			SalarioBruto:   mapGetFloat(r, "SalarioBruto", "salarioBruto", "TotalProventos", "Proventos", "Bruto", "Remuneracao"),
			Descontos:      mapGetFloat(r, "Descontos", "descontos", "TotalDescontos", "totalDescontos"),
			SalarioLiquido: mapGetFloat(r, "SalarioLiquido", "salarioLiquido", "Liquido", "liquido", "ValorLiquido"),
			Competencia:    comp,
			CargaHoraria:   mapGetStr(r, "CargaHoraria", "cargaHoraria"),
			DataAdmissao:   mapGetStr(r, "DataAdmissao", "dataAdmissao", "Admissao"),
			Orgao:          mapGetStr(r, "Orgao", "orgao", "NomeOrgao"),
			Fonte:          "Portal Municipal (Centi API)",
			PortalURL:      base,
		})
		if len(items) >= limit {
			break
		}
	}

	if total == 0 {
		total = len(items)
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhum servidor encontrado para os filtros e período consultados."
	}

	return &ServidoresResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Centi API)",
		Plataforma:     "centi_api",
		PortalURL:      base,
		TotalRegistros: total,
		Servidores:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryCentiDespesas consulta empenhos/despesas na API Centi (POST /portal/getempenhos/{UF}/{Tenant}).
func QueryCentiDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	base, uf, tenant, err := resolveCentiTarget(m, p)
	if err != nil {
		return nil, err
	}

	ano, _ := resolveCentiYearAndMonth(p)
	inicio, fim := buildDDMMYYYYWindow(p, ano)
	limit := normalizeLimit(p.Limit)

	payload := map[string]any{
		"DataInicio": inicio,
		"DataFim":    fim,
		"page": map[string]any{
			"size":   limit,
			"number": 1,
		},
	}

	target := fmt.Sprintf("%s/portal/getempenhos/%s/%s", base, url.PathEscape(uf), url.PathEscape(tenant))
	body, status, err := doCentiPostWithRetry(ctx, client, target, payload)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Centi /portal/getempenhos retornou HTTP %d", status)
	}

	rows, total, err := decodeFlexiblePortalList(body, "empenhos", "Empenhos", "lista")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Centi /portal/getempenhos: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]DespesaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		fornecedor := mapGetStr(r, "Credor", "credor", "Fornecedor", "fornecedor", "NomeCredor", "RazaoSocial")
		historico := mapGetStr(r, "Historico", "historico", "Descricao", "descricao")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+historico), filtro) {
			continue
		}
		if rowTot := int(mapGetFloat(r, "TotalRegistros", "totalRegistros", "total")); rowTot > total {
			total = rowTot
		}
		itemAno := int(mapGetFloat(r, "Ano", "ano", "Exercicio", "exercicio"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, DespesaItem{
			NumeroEmpenho:       mapGetStr(r, "NumeroEmpenho", "numeroEmpenho", "Empenho", "empenho", "Numero", "numero"),
			Ano:                 itemAno,
			Data:                mapGetStr(r, "Data", "data", "DataEmpenho", "dataEmpenho"),
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(mapGetStr(r, "CpfCnpj", "cpfCnpj", "CnpjCpf", "cnpjCpf", "Documento", "documento", "Cpf", "Cnpj")),
			Orgao:               mapGetStr(r, "Orgao", "orgao", "NomeOrgao"),
			Unidade:             mapGetStr(r, "Unidade", "unidade", "UnidadeOrcamentaria"),
			Funcao:              mapGetStr(r, "Funcao", "funcao"),
			Elemento:            mapGetStr(r, "Elemento", "elemento", "ElementoDespesa", "NaturezaDespesa"),
			Historico:           historico,
			ValorEmpenhado:      mapGetFloat(r, "ValorEmpenhado", "valorEmpenhado", "Empenhado", "empenhado", "Valor", "valor"),
			ValorLiquidado:      mapGetFloat(r, "ValorLiquidado", "valorLiquidado", "Liquidado", "liquidado"),
			ValorPago:           mapGetFloat(r, "ValorPago", "valorPago", "Pago", "pago"),
			Fonte:               "Portal Municipal (Centi API)",
		})
		if len(items) >= limit {
			break
		}
	}

	if total == 0 {
		total = len(items)
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma despesa encontrada para o período informado."
	}

	return &DespesasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Centi API)",
		Plataforma:     "centi_api",
		PortalURL:      base,
		TotalRegistros: total,
		Despesas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryCentiReceitas consulta receitas na API Centi (POST /portal/GetReceita/{UF}/{Tenant}).
func QueryCentiReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	base, uf, tenant, err := resolveCentiTarget(m, p)
	if err != nil {
		return nil, err
	}

	ano, mes := resolveCentiYearAndMonth(p)
	limit := normalizeLimit(p.Limit)
	orgaoID := resolveCentiOrgao(p)

	payload := map[string]any{
		"Ano":   ano,
		"Mes":   mes,
		"Orgao": orgaoID,
		"page": map[string]any{
			"size":   limit,
			"number": 1,
		},
	}

	target := fmt.Sprintf("%s/portal/GetReceita/%s/%s", base, url.PathEscape(uf), url.PathEscape(tenant))
	body, status, err := doCentiPostWithRetry(ctx, client, target, payload)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Centi /portal/GetReceita retornou HTTP %d", status)
	}

	rows, total, err := decodeFlexiblePortalList(body, "receitas", "Receitas", "lista")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Centi /portal/GetReceita: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		desc := mapGetStr(r, "Descricao", "descricao", "Especificacao", "Receita", "Nome", "nome")
		if filtro != "" && !strings.Contains(strings.ToLower(desc), filtro) {
			continue
		}
		if rowTot := int(mapGetFloat(r, "TotalRegistros", "totalRegistros", "total")); rowTot > total {
			total = rowTot
		}
		items = append(items, ReceitaItem{
			Codigo:          mapGetStr(r, "Codigo", "codigo", "CodigoReceita", "Classificacao", "Conta", "Rubrica"),
			Descricao:       desc,
			Categoria:       mapGetStr(r, "Categoria", "categoria", "Fonte", "FonteRecurso", "Origem"),
			Data:            mapGetStr(r, "Data", "data", "Competencia"),
			ValorPrevisto:   mapGetFloat(r, "ValorPrevisto", "valorPrevisto", "Previsto", "previsto", "Orcado", "PrevisaoInicial"),
			ValorArrecadado: mapGetFloat(r, "ValorArrecadado", "valorArrecadado", "Arrecadado", "arrecadado", "Realizado", "ArrecadadoAteMes"),
			Fonte:           "Portal Municipal (Centi API)",
		})
		if len(items) >= limit {
			break
		}
	}

	if total == 0 {
		total = len(items)
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma receita encontrada para o período informado."
	}

	return &ReceitasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Centi API)",
		Plataforma:     "centi_api",
		PortalURL:      base,
		TotalRegistros: total,
		Receitas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryCentiContratos consulta contratos na API Centi (POST /portal/GetContratoPortal/{UF}/{Tenant}).
func QueryCentiContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ContratosResult, error) {
	base, uf, tenant, err := resolveCentiTarget(m, p)
	if err != nil {
		return nil, err
	}

	ano, _ := resolveCentiYearAndMonth(p)
	limit := normalizeLimit(p.Limit)

	payload := map[string]any{
		"ano": ano,
		"page": map[string]any{
			"size":   limit,
			"number": 1,
		},
	}

	target := fmt.Sprintf("%s/portal/GetContratoPortal/%s/%s", base, url.PathEscape(uf), url.PathEscape(tenant))
	body, status, err := doCentiPostWithRetry(ctx, client, target, payload)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Centi /portal/GetContratoPortal retornou HTTP %d", status)
	}

	rows, total, err := decodeFlexiblePortalList(body, "contratos", "Contratos", "lista")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Centi /portal/GetContratoPortal: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ContratoItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		fornecedor := mapGetStr(r, "Fornecedor", "fornecedor", "Contratado", "contratado", "RazaoSocial", "Credor")
		objeto := mapGetStr(r, "Objeto", "objeto", "Descricao", "descricao")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+objeto), filtro) {
			continue
		}
		if rowTot := int(mapGetFloat(r, "TotalRegistros", "totalRegistros", "total")); rowTot > total {
			total = rowTot
		}
		itemAno := int(mapGetFloat(r, "Ano", "ano", "Exercicio", "exercicio"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, ContratoItem{
			Numero:              mapGetStr(r, "Numero", "numero", "NumeroContrato", "numeroContrato"),
			Ano:                 itemAno,
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(mapGetStr(r, "CpfCnpj", "cpfCnpj", "CnpjCpf", "Documento", "documento", "Cpf", "Cnpj")),
			Objeto:              objeto,
			Situacao:            mapGetStr(r, "Situacao", "situacao", "Status"),
			DataAssinatura:      mapGetStr(r, "DataAssinatura", "dataAssinatura", "Assinatura", "Data"),
			VigenciaInicio:      mapGetStr(r, "VigenciaInicio", "vigenciaInicio", "DataInicio", "InicioVigencia"),
			VigenciaFim:         mapGetStr(r, "VigenciaFim", "vigenciaFim", "DataFim", "FimVigencia"),
			Valor:               mapGetFloat(r, "Valor", "valor", "ValorContrato", "ValorGlobal"),
			Fonte:               "Portal Municipal (Centi API)",
		})
		if len(items) >= limit {
			break
		}
	}

	if total == 0 {
		total = len(items)
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhum contrato encontrado para o período informado."
	}

	return &ContratosResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Centi API)",
		Plataforma:     "centi_api",
		PortalURL:      base,
		TotalRegistros: total,
		Contratos:      items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryCentiLicitacoes consulta licitações na API Centi (POST /portal/ListaLicitacao/{UF}/{Tenant}).
func QueryCentiLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LicitacoesResult, error) {
	base, uf, tenant, err := resolveCentiTarget(m, p)
	if err != nil {
		return nil, err
	}

	ano, _ := resolveCentiYearAndMonth(p)
	limit := normalizeLimit(p.Limit)

	payload := map[string]any{
		"Ano":                 ano,
		"ModalidadeLicitacao": []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		"page": map[string]any{
			"size":   limit,
			"number": 1,
		},
	}

	target := fmt.Sprintf("%s/portal/ListaLicitacao/%s/%s", base, url.PathEscape(uf), url.PathEscape(tenant))
	body, status, err := doCentiPostWithRetry(ctx, client, target, payload)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Centi /portal/ListaLicitacao retornou HTTP %d", status)
	}

	rows, total, err := decodeFlexiblePortalList(body, "licitacoes", "Licitacoes", "lista")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Centi /portal/ListaLicitacao: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]LicitacaoItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		objeto := mapGetStr(r, "Objeto", "objeto", "Descricao", "descricao")
		modalidade := mapGetStr(r, "Modalidade", "modalidade", "DescricaoModalidade")
		if filtro != "" && !strings.Contains(strings.ToLower(objeto+" "+modalidade), filtro) {
			continue
		}
		if rowTot := int(mapGetFloat(r, "TotalRegistros", "totalRegistros", "total")); rowTot > total {
			total = rowTot
		}
		itemAno := int(mapGetFloat(r, "Ano", "ano", "Exercicio", "exercicio"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, LicitacaoItem{
			Numero:          mapGetStr(r, "Numero", "numero", "NumeroLicitacao", "numeroLicitacao", "Edital", "Processo"),
			Ano:             itemAno,
			Modalidade:      modalidade,
			Situacao:        mapGetStr(r, "Situacao", "situacao", "Status"),
			DataAbertura:    mapGetStr(r, "DataAbertura", "dataAbertura", "Data", "DataPublicacao"),
			Objeto:          objeto,
			ValorEstimado:   mapGetFloat(r, "ValorEstimado", "valorEstimado", "Estimado", "ValorMaximo"),
			ValorHomologado: mapGetFloat(r, "ValorHomologado", "valorHomologado", "Homologado", "Valor"),
			Link:            mapGetStr(r, "Link", "link", "Url"),
			Fonte:           "Portal Municipal (Centi API)",
		})
		if len(items) >= limit {
			break
		}
	}

	if total == 0 {
		total = len(items)
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma licitação encontrada para o período informado."
	}

	return &LicitacoesResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Centi API)",
		Plataforma:     "centi_api",
		PortalURL:      base,
		TotalRegistros: total,
		Licitacoes:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}
