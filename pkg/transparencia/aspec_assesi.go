package transparencia

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// 1. Aspec ("Governo Transparente", plataforma = "aspec_gt")
// ---------------------------------------------------------------------------

const defaultAspecBaseURL = "https://www.governotransparente.com.br"

var (
	aspecURLPathRegex = regexp.MustCompile(`(?i)governotransparente\.com\.br([^\s"'<>]*)`)
	aspecDigitsRegex  = regexp.MustCompile(`^\d+$`)
)

// ExtractAspecEntityCode extrai o código numérico da entidade Aspec a partir de URLs
// (ex: governotransparente.com.br/11629488 ou governotransparente.com.br/transparencia/receitas/11509489)
// ou trechos HTML contendo links para governotransparente.com.br.
func ExtractAspecEntityCode(urlOrHTML string) string {
	extractFromPath := func(rawPath string) string {
		clean := strings.SplitN(rawPath, "?", 2)[0]
		clean = strings.SplitN(clean, "#", 2)[0]
		parts := strings.Split(clean, "/")
		var fallback4 string
		for _, seg := range parts {
			seg = strings.TrimSpace(seg)
			if seg == "" || !aspecDigitsRegex.MatchString(seg) {
				continue
			}
			if len(seg) >= 5 {
				return seg
			}
			if len(seg) == 4 && fallback4 == "" && !strings.HasPrefix(seg, "19") && !strings.HasPrefix(seg, "20") {
				fallback4 = seg
			}
		}
		return fallback4
	}

	matches := aspecURLPathRegex.FindAllStringSubmatch(urlOrHTML, -1)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		if code := extractFromPath(m[1]); code != "" {
			return code
		}
	}

	// Fallback para URLs de teste (ex: http://127.0.0.1:xxxx/11629488)
	trimmed := strings.TrimSpace(urlOrHTML)
	if strings.HasPrefix(strings.ToLower(trimmed), "http://") || strings.HasPrefix(strings.ToLower(trimmed), "https://") {
		if u, err := url.Parse(trimmed); err == nil {
			if code := extractFromPath(u.Path); code != "" {
				return code
			}
		}
	}
	return ""
}

func resolveAspecRootAndCode(baseURL, code string) (apiRoot, entityCode string) {
	entityCode = strings.TrimSpace(code)
	if entityCode == "" {
		entityCode = ExtractAspecEntityCode(baseURL)
	}

	apiRoot = defaultAspecBaseURL
	trimmedBase := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmedBase == "" {
		return apiRoot, entityCode
	}

	u, err := url.Parse(trimmedBase)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return apiRoot, entityCode
	}

	if strings.Contains(strings.ToLower(u.Host), "governotransparente.com.br") {
		return defaultAspecBaseURL, entityCode
	}

	// Servidor customizado / mock em testes
	p := strings.TrimRight(u.Path, "/")
	lowP := strings.ToLower(p)
	if idx := strings.Index(lowP, "/transparencia"); idx >= 0 {
		p = p[:idx]
	} else if entityCode != "" && strings.HasSuffix(p, "/"+entityCode) {
		p = strings.TrimSuffix(p, "/"+entityCode)
	}
	apiRoot = strings.TrimRight(u.Scheme+"://"+u.Host+p, "/")
	return apiRoot, entityCode
}

type aspecExercicio struct {
	ID   string
	Nome string
	Ano  int
}

var aspecYearRegex = regexp.MustCompile(`\b(19\d{2}|20\d{2})\b`)

func fetchAspecExercicios(ctx context.Context, client *http.Client, apiRoot, code string) ([]aspecExercicio, int, error) {
	u := fmt.Sprintf("%s/transparencia/api/v1/json/exercicios/%s", apiRoot, url.PathEscape(code))
	body, status, err := doJSONGet(ctx, portalConnectorHTTPClient(client), u)
	if err != nil {
		return nil, status, err
	}
	if status != http.StatusOK {
		return nil, status, fmt.Errorf("Aspec /exercicios/%s retornou HTTP %d", code, status)
	}

	trimmed := strings.TrimSpace(string(body))
	if !strings.HasPrefix(trimmed, "[") {
		return nil, status, fmt.Errorf("Aspec /exercicios/%s não retornou array JSON", code)
	}

	var rawRows []map[string]any
	if err := json.Unmarshal([]byte(trimmed), &rawRows); err != nil {
		return nil, status, fmt.Errorf("erro ao decodificar Aspec /exercicios/%s: %w", code, err)
	}

	exercicios := make([]aspecExercicio, 0, len(rawRows))
	for _, r := range rawRows {
		id := mapGetStr(r, "id", "ID", "codigo", "exer")
		nome := mapGetStr(r, "nome", "Nome", "exercicio", "ano", "descricao")
		if id == "" && nome == "" {
			continue
		}
		var ano int
		if m := aspecYearRegex.FindStringSubmatch(nome); len(m) >= 2 {
			ano, _ = strconv.Atoi(m[1])
		}
		exercicios = append(exercicios, aspecExercicio{
			ID:   id,
			Nome: nome,
			Ano:  ano,
		})
	}
	return exercicios, status, nil
}

// ProbeAspec verifica se o código ou URL corresponde a um portal Aspec Governo Transparente válido.
func ProbeAspec(ctx context.Context, client *http.Client, baseURL, code string) (entityCode string, ok bool) {
	apiRoot, entityCode := resolveAspecRootAndCode(baseURL, code)
	if entityCode == "" {
		return "", false
	}
	exercicios, status, err := fetchAspecExercicios(ctx, client, apiRoot, entityCode)
	if err != nil || status != http.StatusOK || len(exercicios) == 0 {
		return "", false
	}
	return entityCode, true
}

func formatDateDDMMYYYY(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	// YYYY-MM-DD -> DD/MM/YYYY
	if len(s) >= 10 && s[4] == '-' && s[7] == '-' {
		if t, err := time.Parse("2006-01-02", s[:10]); err == nil {
			return t.Format("02/01/2006")
		}
	}
	// DD/MM/YYYY -> mantém
	if len(s) >= 10 && s[2] == '/' && s[5] == '/' {
		return s[:10]
	}
	return s
}

func parseYearFromDateStr(raw string) int {
	s := strings.TrimSpace(raw)
	if len(s) >= 10 && s[4] == '-' && s[7] == '-' {
		if y, err := strconv.Atoi(s[:4]); err == nil && y > 1900 {
			return y
		}
	}
	if len(s) >= 10 && s[2] == '/' && s[5] == '/' {
		if y, err := strconv.Atoi(s[6:10]); err == nil && y > 1900 {
			return y
		}
	}
	if len(s) >= 4 {
		if y, err := strconv.Atoi(s[:4]); err == nil && y > 1900 {
			return y
		}
	}
	return 0
}

func buildDDMMYYYYWindow(p PortalQueryParams, ano int) (inicio, fim string) {
	if ano <= 0 {
		ano = time.Now().Year()
	}
	inicio = formatDateDDMMYYYY(p.Inicio)
	fim = formatDateDDMMYYYY(p.Fim)

	if inicio == "" {
		if p.Mes > 0 && p.Mes <= 12 {
			inicio = fmt.Sprintf("01/%02d/%04d", p.Mes, ano)
		} else {
			inicio = fmt.Sprintf("01/01/%04d", ano)
		}
	}
	if fim == "" {
		if p.Mes > 0 && p.Mes <= 12 {
			lastDay := time.Date(ano, time.Month(p.Mes)+1, 0, 0, 0, 0, 0, time.UTC)
			fim = lastDay.Format("02/01/2006")
		} else {
			fim = fmt.Sprintf("31/12/%04d", ano)
		}
	}
	return inicio, fim
}

func resolveAspecExerAndWindow(ctx context.Context, client *http.Client, apiRoot, code string, p PortalQueryParams) (exerID string, ano int, inicio, fim string, found bool, err error) {
	exercicios, _, err := fetchAspecExercicios(ctx, client, apiRoot, code)
	if err != nil {
		return "", 0, "", "", false, err
	}
	if len(exercicios) == 0 {
		return "", 0, "", "", false, nil
	}

	targetYear := p.Ano
	if targetYear <= 0 {
		targetYear = parseYearFromDateStr(p.Inicio)
	}
	explicitYear := targetYear > 0
	if targetYear <= 0 {
		targetYear = time.Now().Year()
	}

	targetStr := strconv.Itoa(targetYear)
	for _, ex := range exercicios {
		if strings.TrimSpace(ex.Nome) == targetStr || ex.Ano == targetYear {
			exerID = ex.ID
			ano = targetYear
			break
		}
	}

	if exerID == "" && !explicitYear {
		// Escolhe o exercício mais recente disponível
		best := exercicios[0]
		for _, ex := range exercicios[1:] {
			if ex.Ano > best.Ano {
				best = ex
			}
		}
		exerID = best.ID
		if best.Ano > 0 {
			ano = best.Ano
		} else {
			ano = targetYear
		}
	}

	if exerID == "" {
		return "", targetYear, "", "", false, nil
	}

	inicio, fim = buildDDMMYYYYWindow(p, ano)
	return exerID, ano, inicio, fim, true, nil
}

// QueryAspecDespesas consulta empenhos/despesas na API Aspec Governo Transparente.
func QueryAspecDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	apiRoot, code := resolveAspecRootAndCode(p.BaseURL, p.TenantID)
	if code == "" && p.CodAdministracao > 0 {
		code = strconv.Itoa(p.CodAdministracao)
	}
	if code == "" {
		return nil, fmt.Errorf("Aspec: código da entidade não informado")
	}

	hc := portalConnectorHTTPClient(client)
	exerID, ano, inicio, fim, found, err := resolveAspecExerAndWindow(ctx, hc, apiRoot, code, p)
	if err != nil {
		return nil, err
	}
	portalURL := fmt.Sprintf("%s/%s", apiRoot, code)
	if !found {
		return &DespesasResult{
			Municipio:  m,
			Fonte:      "Portal Municipal (Aspec Governo Transparente)",
			Plataforma: "aspec_gt",
			PortalURL:  portalURL,
			Despesas:   []DespesaItem{},
			Disponivel: true,
			Mensagem:   fmt.Sprintf("Exercício %d não disponível no portal Aspec.", ano),
		}, nil
	}

	limit := normalizeLimit(p.Limit)
	q := url.Values{}
	q.Set("page", "1")
	q.Set("pagesize", strconv.Itoa(limit))
	q.Set("exer", exerID)
	q.Set("inicio", inicio)
	q.Set("fim", fim)

	target := fmt.Sprintf("%s/transparencia/api/v1/json/empenhos/%s?%s", apiRoot, url.PathEscape(code), q.Encode())
	body, status, err := doJSONGet(ctx, hc, target)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Aspec /empenhos/%s retornou HTTP %d", code, status)
	}

	rows, total, err := decodeFlexiblePortalList(body, "empenhos", "Empenhos")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Aspec /empenhos/%s: %w", code, err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]DespesaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		fornecedor := mapGetStr(r, "credor", "fornecedor", "nomeCredor", "nomeFornecedor", "razaoSocial", "Credor", "Fornecedor")
		historico := mapGetStr(r, "historico", "descricao", "Historico", "Descricao")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+historico), filtro) {
			continue
		}
		itemAno := int(mapGetFloat(r, "exercicio", "ano", "Exercicio", "Ano"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, DespesaItem{
			NumeroEmpenho:       mapGetStr(r, "numero", "numeroEmpenho", "numEmpenho", "empenho", "Numero", "Empenho"),
			Ano:                 itemAno,
			Data:                mapGetStr(r, "data", "dataEmpenho", "dtEmpenho", "Data"),
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(mapGetStr(r, "cpfCnpj", "cnpjCpf", "documento", "documentoFornecedor", "cpf_cnpj", "cnpj", "CpfCnpj")),
			Orgao:               mapGetStr(r, "orgao", "nomeOrgao", "unidadeGestora", "Orgao"),
			Unidade:             mapGetStr(r, "unidade", "nomeUnidade", "unidadeOrcamentaria", "Unidade"),
			Funcao:              mapGetStr(r, "funcao", "nomeFuncao", "descricaoFuncao", "Funcao"),
			Elemento:            mapGetStr(r, "elemento", "elementoDespesa", "naturezaDespesa", "Elemento"),
			Historico:           historico,
			ValorEmpenhado:      mapGetFloat(r, "valorEmpenhado", "empenhado", "valor", "vlEmpenhado", "ValorEmpenhado", "Valor"),
			ValorLiquidado:      mapGetFloat(r, "valorLiquidado", "liquidado", "vlLiquidado", "ValorLiquidado"),
			ValorPago:           mapGetFloat(r, "valorPago", "pago", "vlPago", "ValorPago"),
			Fonte:               "Portal Municipal (Aspec Governo Transparente)",
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
		Fonte:          "Portal Municipal (Aspec Governo Transparente)",
		Plataforma:     "aspec_gt",
		PortalURL:      portalURL,
		TotalRegistros: total,
		Despesas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryAspecReceitas consulta receitas orçamentárias na API Aspec Governo Transparente.
func QueryAspecReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	apiRoot, code := resolveAspecRootAndCode(p.BaseURL, p.TenantID)
	if code == "" && p.CodAdministracao > 0 {
		code = strconv.Itoa(p.CodAdministracao)
	}
	if code == "" {
		return nil, fmt.Errorf("Aspec: código da entidade não informado")
	}

	hc := portalConnectorHTTPClient(client)
	exerID, ano, inicio, fim, found, err := resolveAspecExerAndWindow(ctx, hc, apiRoot, code, p)
	if err != nil {
		return nil, err
	}
	portalURL := fmt.Sprintf("%s/%s", apiRoot, code)
	if !found {
		return &ReceitasResult{
			Municipio:  m,
			Fonte:      "Portal Municipal (Aspec Governo Transparente)",
			Plataforma: "aspec_gt",
			PortalURL:  portalURL,
			Receitas:   []ReceitaItem{},
			Disponivel: true,
			Mensagem:   fmt.Sprintf("Exercício %d não disponível no portal Aspec.", ano),
		}, nil
	}

	limit := normalizeLimit(p.Limit)
	q := url.Values{}
	q.Set("page", "1")
	q.Set("pagesize", strconv.Itoa(limit))
	q.Set("exer", exerID)
	q.Set("inicio", inicio)
	q.Set("fim", fim)

	target := fmt.Sprintf("%s/transparencia/api/v1/json/receitasorc/%s?%s", apiRoot, url.PathEscape(code), q.Encode())
	body, status, err := doJSONGet(ctx, hc, target)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Aspec /receitasorc/%s retornou HTTP %d", code, status)
	}

	rows, total, err := decodeFlexiblePortalList(body, "receitas", "Receitas")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Aspec /receitasorc/%s: %w", code, err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		desc := mapGetStr(r, "descricao", "nome", "especificacao", "receita", "Descricao", "Nome")
		if filtro != "" && !strings.Contains(strings.ToLower(desc), filtro) {
			continue
		}
		items = append(items, ReceitaItem{
			Codigo:          mapGetStr(r, "codigo", "codigoReceita", "estrutural", "classificacao", "conta", "Codigo"),
			Descricao:       desc,
			Categoria:       mapGetStr(r, "categoria", "fonte", "fonteRecurso", "origem", "Categoria"),
			Data:            mapGetStr(r, "data", "dataArrecadacao", "competencia", "Data"),
			ValorPrevisto:   mapGetFloat(r, "valorPrevisto", "previsto", "orcado", "previsaoAtualizada", "vlPrevisto", "ValorPrevisto", "Previsto"),
			ValorArrecadado: mapGetFloat(r, "valorArrecadado", "arrecadado", "realizado", "vlArrecadado", "ValorArrecadado", "Arrecadado"),
			Fonte:           "Portal Municipal (Aspec Governo Transparente)",
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
		Fonte:          "Portal Municipal (Aspec Governo Transparente)",
		Plataforma:     "aspec_gt",
		PortalURL:      portalURL,
		TotalRegistros: total,
		Receitas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryAspecContratos consulta contratos na API Aspec Governo Transparente.
func QueryAspecContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ContratosResult, error) {
	apiRoot, code := resolveAspecRootAndCode(p.BaseURL, p.TenantID)
	if code == "" && p.CodAdministracao > 0 {
		code = strconv.Itoa(p.CodAdministracao)
	}
	if code == "" {
		return nil, fmt.Errorf("Aspec: código da entidade não informado")
	}

	hc := portalConnectorHTTPClient(client)
	exerID, ano, inicio, fim, found, err := resolveAspecExerAndWindow(ctx, hc, apiRoot, code, p)
	if err != nil {
		return nil, err
	}
	portalURL := fmt.Sprintf("%s/%s", apiRoot, code)
	if !found {
		return &ContratosResult{
			Municipio:  m,
			Fonte:      "Portal Municipal (Aspec Governo Transparente)",
			Plataforma: "aspec_gt",
			PortalURL:  portalURL,
			Contratos:  []ContratoItem{},
			Disponivel: true,
			Mensagem:   fmt.Sprintf("Exercício %d não disponível no portal Aspec.", ano),
		}, nil
	}

	limit := normalizeLimit(p.Limit)
	q := url.Values{}
	q.Set("page", "1")
	q.Set("pagesize", strconv.Itoa(limit))
	q.Set("exer", exerID)
	q.Set("inicio", inicio)
	q.Set("fim", fim)

	target := fmt.Sprintf("%s/transparencia/api/v1/json/contratos/%s?%s", apiRoot, url.PathEscape(code), q.Encode())
	body, status, err := doJSONGet(ctx, hc, target)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Aspec /contratos/%s retornou HTTP %d", code, status)
	}

	rows, total, err := decodeFlexiblePortalList(body, "contratos", "Contratos")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Aspec /contratos/%s: %w", code, err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ContratoItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		fornecedor := mapGetStr(r, "fornecedor", "contratado", "nomeContratado", "credor", "razaoSocial", "Fornecedor", "Contratado")
		objeto := mapGetStr(r, "objeto", "descricao", "Objeto", "Descricao")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+objeto), filtro) {
			continue
		}
		itemAno := int(mapGetFloat(r, "ano", "exercicio", "Ano", "Exercicio"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, ContratoItem{
			Numero:              mapGetStr(r, "numero", "numeroContrato", "numContrato", "Numero"),
			Ano:                 itemAno,
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(mapGetStr(r, "cpfCnpj", "cnpjCpf", "documento", "documentoFornecedor", "cpf_cnpj", "cnpj", "CpfCnpj")),
			Objeto:              objeto,
			Situacao:            mapGetStr(r, "situacao", "status", "Situacao"),
			DataAssinatura:      mapGetStr(r, "dataAssinatura", "assinatura", "data", "DataAssinatura"),
			VigenciaInicio:      mapGetStr(r, "vigenciaInicio", "inicioVigencia", "dataInicio", "VigenciaInicio"),
			VigenciaFim:         mapGetStr(r, "vigenciaFim", "fimVigencia", "dataFim", "terminoVigencia", "VigenciaFim"),
			Valor:               mapGetFloat(r, "valor", "valorContrato", "valorGlobal", "valorInicial", "Valor"),
			Fonte:               "Portal Municipal (Aspec Governo Transparente)",
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
		Fonte:          "Portal Municipal (Aspec Governo Transparente)",
		Plataforma:     "aspec_gt",
		PortalURL:      portalURL,
		TotalRegistros: total,
		Contratos:      items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryAspecLicitacoes consulta licitações na API Aspec Governo Transparente.
func QueryAspecLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LicitacoesResult, error) {
	apiRoot, code := resolveAspecRootAndCode(p.BaseURL, p.TenantID)
	if code == "" && p.CodAdministracao > 0 {
		code = strconv.Itoa(p.CodAdministracao)
	}
	if code == "" {
		return nil, fmt.Errorf("Aspec: código da entidade não informado")
	}

	ano := p.Ano
	if ano <= 0 {
		ano = parseYearFromDateStr(p.Inicio)
	}
	if ano <= 0 {
		ano = time.Now().Year()
	}
	inicio, fim := buildDDMMYYYYWindow(p, ano)
	limit := normalizeLimit(p.Limit)
	hc := portalConnectorHTTPClient(client)
	portalURL := fmt.Sprintf("%s/%s", apiRoot, code)

	q := url.Values{}
	q.Set("page", "1")
	q.Set("pagesize", strconv.Itoa(limit))
	q.Set("inicio", inicio)
	q.Set("fim", fim)

	target := fmt.Sprintf("%s/transparencia/api/v1/json/licitacoes/%s?%s", apiRoot, url.PathEscape(code), q.Encode())
	body, status, err := doJSONGet(ctx, hc, target)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Aspec /licitacoes/%s retornou HTTP %d", code, status)
	}

	rows, total, err := decodeFlexiblePortalList(body, "licitacoes", "Licitacoes")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Aspec /licitacoes/%s: %w", code, err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]LicitacaoItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		objeto := mapGetStr(r, "objeto", "descricao", "Objeto", "Descricao")
		modalidade := mapGetStr(r, "modalidade", "descricaoModalidade", "Modalidade")
		if filtro != "" && !strings.Contains(strings.ToLower(objeto+" "+modalidade), filtro) {
			continue
		}
		itemAno := int(mapGetFloat(r, "ano", "exercicio", "Ano", "Exercicio"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, LicitacaoItem{
			Numero:          mapGetStr(r, "numero", "numeroLicitacao", "numLicitacao", "numeroProcesso", "edital", "Numero"),
			Ano:             itemAno,
			Modalidade:      modalidade,
			Situacao:        mapGetStr(r, "situacao", "status", "Situacao"),
			DataAbertura:    mapGetStr(r, "dataAbertura", "data", "abertura", "dataPublicacao", "DataAbertura"),
			Objeto:          objeto,
			ValorEstimado:   mapGetFloat(r, "valorEstimado", "estimado", "valorMaximo", "ValorEstimado"),
			ValorHomologado: mapGetFloat(r, "valorHomologado", "homologado", "valorAdjudicado", "valor", "ValorHomologado"),
			Link:            mapGetStr(r, "link", "url", "Link"),
			Fonte:           "Portal Municipal (Aspec Governo Transparente)",
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
		Fonte:          "Portal Municipal (Aspec Governo Transparente)",
		Plataforma:     "aspec_gt",
		PortalURL:      portalURL,
		TotalRegistros: total,
		Licitacoes:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// ---------------------------------------------------------------------------
// 2. Assesi (plataforma = "assesi_portal")
// ---------------------------------------------------------------------------

func normalizeAssesiBaseURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return ""
	}
	low := strings.ToLower(base)
	for _, suffix := range []string{"/dadosabertos.php", "/dadosabertosexportar.php"} {
		if strings.HasSuffix(low, suffix) {
			base = strings.TrimRight(base[:len(base)-len(suffix)], "/")
			break
		}
	}
	return base
}

func isAssesiHTMLOrEmpty(body []byte) bool {
	trimmed := strings.TrimSpace(strings.TrimPrefix(string(body), "\xef\xbb\xbf"))
	if trimmed == "" {
		return true
	}
	return strings.HasPrefix(trimmed, "<")
}

// ProbeAssesi verifica se a URL base corresponde a um portal Assesi (/dadosabertosexportar.php ou /dadosabertos.php).
func ProbeAssesi(ctx context.Context, client *http.Client, baseURL string) bool {
	base := normalizeAssesiBaseURL(baseURL)
	if base == "" {
		return false
	}
	hc := portalConnectorHTTPClient(client)

	for _, dataset := range []string{"empenhos", "receitas"} {
		target := fmt.Sprintf("%s/dadosabertosexportar.php?d=%s&a=2025&f=json", base, dataset)
		body, status, err := doPortalRawGet(ctx, hc, target, "application/json,text/html,*/*")
		if err != nil || status != http.StatusOK {
			continue
		}
		if isAssesiHTMLOrEmpty(body) {
			continue
		}
		if _, _, err := decodeFlexiblePortalList(body, dataset); err == nil {
			return true
		}
	}

	if body, status, err := doPortalRawGet(ctx, hc, base+"/dadosabertos.php", "text/html,application/xhtml+xml,*/*"); err == nil && status == http.StatusOK {
		if strings.Contains(strings.ToLower(string(body)), "dadosabertosexportar.php") {
			return true
		}
	}

	return false
}

func resolveAssesiYears(p PortalQueryParams) []int {
	if p.Ano > 0 {
		return []int{p.Ano}
	}
	if y := parseYearFromDateStr(p.Inicio); y > 0 {
		return []int{y}
	}
	nowY := time.Now().Year()
	return []int{nowY, nowY - 1, nowY - 2}
}

func fetchAssesiDataset(ctx context.Context, client *http.Client, p PortalQueryParams, dataset string) ([]map[string]any, int, int, string, error) {
	base := normalizeAssesiBaseURL(p.BaseURL)
	if base == "" {
		return nil, 0, 0, "", fmt.Errorf("Assesi: BaseURL não informada")
	}
	hc := portalConnectorHTTPClient(client)
	years := resolveAssesiYears(p)

	var lastStatus int
	for _, ano := range years {
		q := url.Values{}
		q.Set("d", dataset)
		q.Set("a", strconv.Itoa(ano))
		q.Set("f", "json")
		if p.Mes > 0 && p.Mes <= 12 {
			q.Set("m", strconv.Itoa(p.Mes))
		}
		target := fmt.Sprintf("%s/dadosabertosexportar.php?%s", base, q.Encode())
		body, status, err := doPortalRawGet(ctx, hc, target, "application/json,text/html,*/*")
		if err != nil {
			return nil, 0, ano, base, err
		}
		lastStatus = status
		if status != http.StatusOK {
			if p.Ano > 0 {
				return nil, 0, ano, base, fmt.Errorf("Assesi dadosabertosexportar.php?d=%s retornou HTTP %d", dataset, status)
			}
			continue
		}
		if isAssesiHTMLOrEmpty(body) {
			// Assesi retorna <script>alert(...)</script> quando não há registros no ano.
			continue
		}
		cleanBody := []byte(strings.TrimSpace(strings.TrimPrefix(string(body), "\xef\xbb\xbf")))
		rows, total, err := decodeFlexiblePortalList(cleanBody, dataset, "dados", "registros")
		if err != nil {
			return nil, 0, ano, base, fmt.Errorf("erro ao decodificar Assesi %s: %w", dataset, err)
		}
		if len(rows) > 0 {
			return rows, total, ano, base, nil
		}
	}

	if lastStatus >= 400 && len(years) == 1 {
		return nil, 0, years[0], base, fmt.Errorf("Assesi dadosabertosexportar.php?d=%s retornou HTTP %d", dataset, lastStatus)
	}
	return nil, 0, years[0], base, nil
}

// QueryAssesiServidores consulta folha/pessoal no portal Assesi (d=pessoal).
func QueryAssesiServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ServidoresResult, error) {
	rows, total, ano, base, err := fetchAssesiDataset(ctx, client, p, "pessoal")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ServidorItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		nome := mapGetStr(r, "nome", "Nome", "servidor", "Servidor", "funcionario", "Funcionario", "nome_servidor")
		if filtro != "" && !strings.Contains(strings.ToLower(nome), filtro) {
			continue
		}
		comp := mapGetStr(r, "competencia", "Competencia", "mes_ano", "mesAno", "referencia")
		if comp == "" {
			if p.Mes > 0 {
				comp = fmt.Sprintf("%02d/%04d", p.Mes, ano)
			} else {
				comp = strconv.Itoa(ano)
			}
		}

		items = append(items, ServidorItem{
			Nome:           nome,
			Matricula:      mapGetStr(r, "matricula", "Matricula", "registro", "codigo"),
			CPFParcial:     maskCPF(mapGetStr(r, "cpf_formatado", "cpfFormatado", "cpf_parcial"), mapGetStr(r, "cpf", "CPF", "Cpf", "num_cpf")),
			Cargo:          mapGetStr(r, "cargo", "Cargo", "descricao_cargo", "descricaoCargo"),
			Funcao:         mapGetStr(r, "funcao", "Funcao"),
			Lotacao:        mapGetStr(r, "lotacao", "Lotacao", "unidade", "Unidade", "departamento", "setor"),
			LocalTrabalho:  mapGetStr(r, "local_trabalho", "localTrabalho", "local"),
			Vinculo:        mapGetStr(r, "vinculo", "Vinculo", "regime", "Regime", "tipo_vinculo"),
			Situacao:       mapGetStr(r, "situacao", "Situacao", "status"),
			SalarioBase:    mapGetFloat(r, "salario_base", "salarioBase", "SalarioBase", "vencimento_base", "base"),
			SalarioBruto:   mapGetFloat(r, "salario_bruto", "salarioBruto", "SalarioBruto", "proventos", "Proventos", "total_proventos", "bruto", "remuneracao"),
			Descontos:      mapGetFloat(r, "descontos", "Descontos", "total_descontos", "totalDescontos"),
			SalarioLiquido: mapGetFloat(r, "salario_liquido", "salarioLiquido", "SalarioLiquido", "liquido", "Liquido"),
			Competencia:    comp,
			CargaHoraria:   mapGetStr(r, "carga_horaria", "cargaHoraria", "CargaHoraria"),
			DataAdmissao:   mapGetStr(r, "data_admissao", "dataAdmissao", "DataAdmissao", "admissao"),
			Orgao:          mapGetStr(r, "orgao", "Orgao", "entidade"),
			Fonte:          "Portal Municipal (Assesi)",
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
		Fonte:          "Portal Municipal (Assesi)",
		Plataforma:     "assesi_portal",
		PortalURL:      base,
		TotalRegistros: total,
		Servidores:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryAssesiDespesas consulta empenhos/despesas no portal Assesi (d=empenhos).
func QueryAssesiDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	rows, total, ano, base, err := fetchAssesiDataset(ctx, client, p, "empenhos")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]DespesaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		fornecedor := mapGetStr(r, "fornecedor", "Fornecedor", "credor", "Credor", "nome_credor", "razao_social")
		historico := mapGetStr(r, "historico", "Historico", "descricao", "Descricao")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+historico), filtro) {
			continue
		}
		itemAno := int(mapGetFloat(r, "ano", "Ano", "exercicio", "Exercicio"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, DespesaItem{
			NumeroEmpenho:       mapGetStr(r, "numero_empenho", "numeroEmpenho", "empenho", "Empenho", "numero", "Numero"),
			Ano:                 itemAno,
			Data:                mapGetStr(r, "data", "Data", "data_empenho", "dataEmpenho"),
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(mapGetStr(r, "cpf_cnpj", "cnpj_cpf", "cpfCnpj", "documento", "cnpj", "cpf", "Documento")),
			Orgao:               mapGetStr(r, "orgao", "Orgao", "unidade_gestora"),
			Unidade:             mapGetStr(r, "unidade", "Unidade", "unidade_orcamentaria"),
			Funcao:              mapGetStr(r, "funcao", "Funcao"),
			Elemento:            mapGetStr(r, "elemento", "Elemento", "elemento_despesa", "natureza"),
			Historico:           historico,
			ValorEmpenhado:      mapGetFloat(r, "valor_empenhado", "valorEmpenhado", "empenhado", "Empenhado", "valor", "Valor"),
			ValorLiquidado:      mapGetFloat(r, "valor_liquidado", "valorLiquidado", "liquidado", "Liquidado"),
			ValorPago:           mapGetFloat(r, "valor_pago", "valorPago", "pago", "Pago"),
			Fonte:               "Portal Municipal (Assesi)",
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
		Fonte:          "Portal Municipal (Assesi)",
		Plataforma:     "assesi_portal",
		PortalURL:      base,
		TotalRegistros: total,
		Despesas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryAssesiReceitas consulta receitas no portal Assesi (d=receitas).
func QueryAssesiReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	rows, total, _, base, err := fetchAssesiDataset(ctx, client, p, "receitas")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		desc := mapGetStr(r, "descricao", "Descricao", "especificacao", "receita", "Receita", "nome")
		if filtro != "" && !strings.Contains(strings.ToLower(desc), filtro) {
			continue
		}
		items = append(items, ReceitaItem{
			Codigo:          mapGetStr(r, "codigo", "Codigo", "codigo_receita", "classificacao", "conta"),
			Descricao:       desc,
			Categoria:       mapGetStr(r, "categoria", "Categoria", "fonte", "origem"),
			Data:            mapGetStr(r, "data", "Data", "competencia"),
			ValorPrevisto:   mapGetFloat(r, "valor_previsto", "valorPrevisto", "previsto", "Previsto", "orcado", "Orcado"),
			ValorArrecadado: mapGetFloat(r, "valor_arrecadado", "valorArrecadado", "arrecadado", "Arrecadado", "realizado", "Realizado"),
			Fonte:           "Portal Municipal (Assesi)",
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
		Fonte:          "Portal Municipal (Assesi)",
		Plataforma:     "assesi_portal",
		PortalURL:      base,
		TotalRegistros: total,
		Receitas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryAssesiContratos consulta contratos no portal Assesi (d=contratos).
func QueryAssesiContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ContratosResult, error) {
	rows, total, ano, base, err := fetchAssesiDataset(ctx, client, p, "contratos")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ContratoItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		fornecedor := mapGetStr(r, "fornecedor", "Fornecedor", "contratado", "Contratado", "razao_social", "credor")
		objeto := mapGetStr(r, "objeto", "Objeto", "descricao", "Descricao")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+objeto), filtro) {
			continue
		}
		itemAno := int(mapGetFloat(r, "ano", "Ano", "exercicio", "Exercicio"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, ContratoItem{
			Numero:              mapGetStr(r, "numero", "Numero", "numero_contrato", "numeroContrato", "contrato"),
			Ano:                 itemAno,
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(mapGetStr(r, "cpf_cnpj", "cnpj_cpf", "cpfCnpj", "documento", "cnpj", "cpf", "Documento")),
			Objeto:              objeto,
			Situacao:            mapGetStr(r, "situacao", "Situacao", "status"),
			DataAssinatura:      mapGetStr(r, "data_assinatura", "dataAssinatura", "assinatura", "data"),
			VigenciaInicio:      mapGetStr(r, "vigencia_inicio", "vigenciaInicio", "data_inicio", "inicio"),
			VigenciaFim:         mapGetStr(r, "vigencia_fim", "vigenciaFim", "data_fim", "fim"),
			Valor:               mapGetFloat(r, "valor", "Valor", "valor_contrato", "valorContrato", "valor_global"),
			Fonte:               "Portal Municipal (Assesi)",
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
		Fonte:          "Portal Municipal (Assesi)",
		Plataforma:     "assesi_portal",
		PortalURL:      base,
		TotalRegistros: total,
		Contratos:      items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryAssesiLicitacoes consulta licitações no portal Assesi (d=licitacoes).
func QueryAssesiLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LicitacoesResult, error) {
	rows, total, ano, base, err := fetchAssesiDataset(ctx, client, p, "licitacoes")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]LicitacaoItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		objeto := mapGetStr(r, "objeto", "Objeto", "descricao", "Descricao")
		modalidade := mapGetStr(r, "modalidade", "Modalidade")
		if filtro != "" && !strings.Contains(strings.ToLower(objeto+" "+modalidade), filtro) {
			continue
		}
		itemAno := int(mapGetFloat(r, "ano", "Ano", "exercicio", "Exercicio"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, LicitacaoItem{
			Numero:          mapGetStr(r, "numero", "Numero", "numero_licitacao", "numeroLicitacao", "edital", "processo"),
			Ano:             itemAno,
			Modalidade:      modalidade,
			Situacao:        mapGetStr(r, "situacao", "Situacao", "status"),
			DataAbertura:    mapGetStr(r, "data_abertura", "dataAbertura", "data", "abertura", "data_publicacao"),
			Objeto:          objeto,
			ValorEstimado:   mapGetFloat(r, "valor_estimado", "valorEstimado", "estimado"),
			ValorHomologado: mapGetFloat(r, "valor_homologado", "valorHomologado", "homologado", "valor"),
			Link:            mapGetStr(r, "link", "url", "Link"),
			Fonte:           "Portal Municipal (Assesi)",
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
		Fonte:          "Portal Municipal (Assesi)",
		Plataforma:     "assesi_portal",
		PortalURL:      base,
		TotalRegistros: total,
		Licitacoes:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}
