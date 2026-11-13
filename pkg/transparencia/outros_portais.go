package transparencia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Helpers HTTP compartilhados para conectores adicionais
// ---------------------------------------------------------------------------

func doPortalRawGet(ctx context.Context, client *http.Client, targetURL, accept string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := portalConnectorHTTPClient(client).Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func doPortalJSONPost(ctx context.Context, client *http.Client, targetURL string, payload any) ([]byte, int, error) {
	rawBody, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(rawBody))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := portalConnectorHTTPClient(client).Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func decodeFlexiblePortalList(body []byte, extraKeys ...string) ([]map[string]any, int, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, 0, nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var rows []map[string]any
		if err := json.Unmarshal([]byte(trimmed), &rows); err != nil {
			return nil, 0, err
		}
		return rows, len(rows), nil
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil {
		return nil, 0, err
	}

	keys := append([]string{}, extraKeys...)
	keys = append(keys,
		"Registros", "registros", "content", "dados", "items",
		"data", "lista", "resultado", "rows", "Servidores", "Despesas", "Receitas",
	)
	var rows []map[string]any
	for _, k := range keys {
		if raw, ok := obj[k]; ok && raw != nil {
			if arr, ok := raw.([]any); ok {
				rows = make([]map[string]any, 0, len(arr))
				for _, el := range arr {
					if m, ok := el.(map[string]any); ok {
						rows = append(rows, m)
					}
				}
				break
			}
		}
	}

	total := int(mapGetFloat(obj, "totalElements", "total", "totalRegistros", "TotalRegistros", "recordsTotal", "count"))
	if total == 0 {
		total = len(rows)
	}
	return rows, total, nil
}

// ---------------------------------------------------------------------------
// 1. GovBR / Pronim / Cidade360 (govbr_cidade360)
// ---------------------------------------------------------------------------

var (
	govbrSelectRegex = regexp.MustCompile(`(?is)<select\b[^>]*>(.*?)</select>`)
	govbrOptionRegex = regexp.MustCompile(`(?is)<option\b([^>]*)>(.*?)</option>`)
	govbrValueRegex  = regexp.MustCompile(`(?is)\bvalue\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	htmlTagRegex     = regexp.MustCompile(`<[^>]+>`)
)

func extractGovBRSelectEntity(htmlBody string) string {
	selects := govbrSelectRegex.FindAllStringSubmatch(htmlBody, -1)
	if len(selects) == 0 {
		return ""
	}

	var firstValid string
	for _, sel := range selects {
		options := govbrOptionRegex.FindAllStringSubmatch(sel[1], -1)
		for _, opt := range options {
			attrs := opt[1]
			text := html.UnescapeString(htmlTagRegex.ReplaceAllString(opt[2], ""))
			text = strings.TrimSpace(text)

			val := ""
			if m := govbrValueRegex.FindStringSubmatch(attrs); len(m) > 0 {
				for _, g := range m[1:] {
					if g != "" {
						val = html.UnescapeString(strings.TrimSpace(g))
						break
					}
				}
			}

			candidate := val
			if candidate == "" {
				candidate = text
			}
			lowCand := strings.ToLower(candidate)
			if candidate == "" || lowCand == "0" || lowCand == "-1" ||
				strings.HasPrefix(lowCand, "selecione") || strings.HasPrefix(lowCand, "escolha") {
				continue
			}

			if strings.Contains(strings.ToLower(attrs), "selected") {
				return candidate
			}
			if firstValid == "" {
				firstValid = candidate
			}
		}
	}
	return firstValid
}

func isGovBRSwagger(body []byte) bool {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return false
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(trimmed), &doc); err != nil {
		return false
	}
	if _, hasPaths := doc["paths"]; !hasPaths {
		if _, hasSwagger := doc["swagger"]; !hasSwagger {
			if _, hasOpenAPI := doc["openapi"]; !hasOpenAPI {
				return false
			}
		}
	}
	low := strings.ToLower(trimmed)
	return strings.Contains(low, "folhapagamento") ||
		strings.Contains(low, "buscardadosreceitas") ||
		strings.Contains(low, "servidor") ||
		strings.Contains(low, "govbr") ||
		strings.Contains(low, "cidade360") ||
		strings.Contains(low, "pronim") ||
		strings.Contains(low, "dadosabertos")
}

// ProbeGovBRCidade360 verifica se a URL corresponde a um portal GovBR / Pronim / Cidade360 (/dadosabertos/).
func ProbeGovBRCidade360(ctx context.Context, client *http.Client, baseURL string) (entidade string, ok bool) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "", false
	}
	hc := portalConnectorHTTPClient(client)

	dadosRoot := base + "/dadosabertos"
	if strings.HasSuffix(strings.ToLower(base), "/dadosabertos") {
		dadosRoot = base
	}

	var htmlMatched bool
	for _, candidateURL := range []string{dadosRoot + "/", dadosRoot} {
		if body, status, err := doPortalRawGet(ctx, hc, candidateURL, "text/html,application/xhtml+xml,*/*"); err == nil && status == http.StatusOK {
			htmlStr := string(body)
			entidade = extractGovBRSelectEntity(htmlStr)
			low := strings.ToLower(htmlStr)
			if entidade != "" ||
				strings.Contains(low, "govbr") ||
				strings.Contains(low, "cidade360") ||
				strings.Contains(low, "pronim") ||
				strings.Contains(low, "folhapagamento") ||
				strings.Contains(low, "nomebase") ||
				strings.Contains(low, "swagger/v1/swagger.json") {
				htmlMatched = true
			}
			break
		}
	}

	if body, status, err := doPortalRawGet(ctx, hc, dadosRoot+"/swagger/v1/swagger.json", "application/json"); err == nil && status == http.StatusOK {
		if isGovBRSwagger(body) {
			return entidade, true
		}
	}

	if htmlMatched {
		return entidade, true
	}
	return "", false
}

// QueryGovBRServidores consulta servidores públicos no portal GovBR / Cidade360 (/FolhaPagamento com fallback para /Servidor).
func QueryGovBRServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams, entidade string) (*ServidoresResult, error) {
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("GovBR/Cidade360: BaseURL não informada")
	}
	if strings.TrimSpace(entidade) == "" {
		entidade = strings.TrimSpace(p.TenantID)
	}
	ano := p.Ano
	if ano <= 0 {
		ano = time.Now().Year()
	}
	limit := normalizeLimit(p.Limit)
	hc := portalConnectorHTTPClient(client)

	dataInicial := fmt.Sprintf("01/01/%04d", ano)
	dataFinal := fmt.Sprintf("31/01/%04d", ano)
	if p.Inicio != "" && p.Fim != "" {
		dataInicial = p.Inicio
		dataFinal = p.Fim
	}

	queryTail := fmt.Sprintf(
		"dataInicial=%s&dataFinal=%s&numeroPagina=1&nomeBase=%s",
		url.QueryEscape(dataInicial), url.QueryEscape(dataFinal), url.QueryEscape(entidade),
	)

	var rows []map[string]any
	var total int
	var lastStatus int
	var lastErr error
	usedEndpoint := "/FolhaPagamento"

	for _, ep := range []string{"/FolhaPagamento", "/Servidor"} {
		target := fmt.Sprintf("%s%s?%s", base, ep, queryTail)
		body, status, err := doJSONGet(ctx, hc, target)
		if err != nil {
			lastErr = err
			continue
		}
		lastStatus = status
		if status != http.StatusOK {
			continue
		}
		parsed, tot, err := decodeFlexiblePortalList(body, "folhaPagamento", "servidores")
		if err != nil {
			lastErr = fmt.Errorf("erro ao decodificar GovBR %s: %w", ep, err)
			continue
		}
		if len(parsed) > 0 {
			rows = parsed
			total = tot
			usedEndpoint = ep
			lastErr = nil
			break
		}
	}

	if len(rows) == 0 {
		if lastErr != nil {
			return nil, lastErr
		}
		if lastStatus >= 400 {
			return nil, fmt.Errorf("GovBR %s retornou HTTP %d", usedEndpoint, lastStatus)
		}
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ServidorItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		nome := mapGetStr(r, "nome", "Nome", "nomeServidor", "NomeServidor", "servidor", "Servidor", "funcionario")
		if filtro != "" && !strings.Contains(strings.ToLower(nome), filtro) {
			continue
		}
		comp := mapGetStr(r, "competencia", "Competencia", "mesAno", "referencia")
		if comp == "" {
			comp = fmt.Sprintf("01/%04d", ano)
		}

		items = append(items, ServidorItem{
			Nome:           nome,
			Matricula:      mapGetStr(r, "matricula", "Matricula", "registro", "Registro", "codigo"),
			CPFParcial:     maskCPF(mapGetStr(r, "cpfFormatado", "CpfFormatado", "cpfParcial"), mapGetStr(r, "cpf", "CPF", "Cpf", "numCpf", "documento")),
			Cargo:          mapGetStr(r, "cargo", "Cargo", "descricaoCargo", "DescricaoCargo", "nomeCargo"),
			Funcao:         mapGetStr(r, "funcao", "Funcao", "descricaoFuncao"),
			Lotacao:        mapGetStr(r, "lotacao", "Lotacao", "unidade", "Unidade", "departamento", "Departamento", "setor"),
			LocalTrabalho:  mapGetStr(r, "localTrabalho", "LocalTrabalho", "local"),
			Vinculo:        mapGetStr(r, "vinculo", "Vinculo", "regime", "Regime", "tipoVinculo"),
			Situacao:       mapGetStr(r, "situacao", "Situacao", "status"),
			SalarioBase:    mapGetFloat(r, "salarioBase", "SalarioBase", "vencimentoBase", "base"),
			SalarioBruto:   mapGetFloat(r, "salarioBruto", "SalarioBruto", "totalProventos", "TotalProventos", "proventos", "Proventos", "valorBruto", "bruto", "remuneracao"),
			Descontos:      mapGetFloat(r, "descontos", "Descontos", "totalDescontos", "TotalDescontos", "valorDescontos"),
			SalarioLiquido: mapGetFloat(r, "salarioLiquido", "SalarioLiquido", "valorLiquido", "ValorLiquido", "liquido", "Liquido"),
			Competencia:    comp,
			CargaHoraria:   mapGetStr(r, "cargaHoraria", "CargaHoraria"),
			DataAdmissao:   mapGetStr(r, "dataAdmissao", "DataAdmissao", "admissao"),
			Orgao:          mapGetStr(r, "orgao", "Orgao", "unidadeGestora", "UnidadeGestora", "entidade"),
			Fonte:          fmt.Sprintf("Portal Municipal (GovBR / Cidade360 %s)", usedEndpoint),
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
		Fonte:          "Portal Municipal (GovBR / Cidade360)",
		Plataforma:     "govbr_cidade360",
		PortalURL:      base,
		TotalRegistros: total,
		Servidores:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryGovBRReceitas consulta receitas no portal GovBR / Cidade360 (/Receitas/buscarDadosReceitas/...).
func QueryGovBRReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams, entidade string) (*ReceitasResult, error) {
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("GovBR/Cidade360: BaseURL não informada")
	}
	if strings.TrimSpace(entidade) == "" {
		entidade = strings.TrimSpace(p.TenantID)
	}
	ano := p.Ano
	if ano <= 0 {
		ano = time.Now().Year()
	}
	limit := normalizeLimit(p.Limit)
	hc := portalConnectorHTTPClient(client)

	target := fmt.Sprintf(
		"%s/Receitas/buscarDadosReceitas/%d/1/%s?unidadeGestora=CONSOLIDADO",
		base, ano, url.PathEscape(entidade),
	)
	body, status, err := doJSONGet(ctx, hc, target)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GovBR /Receitas/buscarDadosReceitas retornou HTTP %d", status)
	}

	rows, total, err := decodeFlexiblePortalList(body, "receitas")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar GovBR receitas: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		desc := mapGetStr(r, "descricao", "Descricao", "especificacao", "Especificacao", "nome", "receita", "Receita")
		if filtro != "" && !strings.Contains(strings.ToLower(desc), filtro) {
			continue
		}
		items = append(items, ReceitaItem{
			Codigo:          mapGetStr(r, "codigo", "Codigo", "codigoReceita", "CodigoReceita", "classificacao", "Classificacao", "conta", "estrutural"),
			Descricao:       desc,
			Categoria:       mapGetStr(r, "categoria", "Categoria", "fonte", "Fonte", "origem", "unidadeGestora"),
			Data:            mapGetStr(r, "data", "Data", "competencia"),
			ValorPrevisto:   mapGetFloat(r, "valorPrevisto", "ValorPrevisto", "previsto", "Previsto", "orcado", "Orcado", "previsaoAtualizada"),
			ValorArrecadado: mapGetFloat(r, "valorArrecadado", "ValorArrecadado", "arrecadado", "Arrecadado", "realizado", "Realizado", "arrecadadoAteMes"),
			Fonte:           "Portal Municipal (GovBR / Cidade360)",
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
		Fonte:          "Portal Municipal (GovBR / Cidade360)",
		Plataforma:     "govbr_cidade360",
		PortalURL:      base,
		TotalRegistros: total,
		Receitas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// ---------------------------------------------------------------------------
// 2. Coplan Gestão Pública (coplan_gp)
// ---------------------------------------------------------------------------

// ProbeCoplan verifica um portal Coplan via POST /servlet/api_consulta?unidade_gestora.
func ProbeCoplan(ctx context.Context, client *http.Client, baseURL string) (ugID int, ugDesc string, ok bool) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return 0, "", false
	}
	target := base + "/servlet/api_consulta?unidade_gestora"
	body, status, err := doPortalJSONPost(ctx, portalConnectorHTTPClient(client), target, map[string]any{})
	if err != nil || status != http.StatusOK {
		return 0, "", false
	}

	rows, _, err := decodeFlexiblePortalList(body, "unidades", "unidade_gestora")
	if err != nil {
		return 0, "", false
	}
	if len(rows) == 0 {
		var single map[string]any
		if err := json.Unmarshal(body, &single); err == nil && single != nil {
			if _, hasID := single["unidade_gestora_id"]; hasID {
				rows = []map[string]any{single}
			}
		}
	}
	if len(rows) == 0 {
		return 0, "", false
	}

	var firstID int
	var firstDesc string
	for _, r := range rows {
		id := int(mapGetFloat(r, "unidade_gestora_id", "id", "codigo"))
		desc := mapGetStr(r, "unidade_gestora_descricao", "descricao", "nome")
		if id == 0 && desc == "" {
			continue
		}
		if firstID == 0 && firstDesc == "" {
			firstID = id
			firstDesc = desc
		}
		if strings.Contains(strings.ToUpper(desc), "PREFEITURA") {
			return id, desc, true
		}
	}
	if firstID == 0 && firstDesc == "" {
		return 0, "", false
	}
	return firstID, firstDesc, true
}

// QueryCoplanDespesas consulta despesas em um portal Coplan via POST /servlet/api_consulta?despesas_v1.
func QueryCoplanDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("Coplan: BaseURL não informada")
	}
	hc := portalConnectorHTTPClient(client)

	ugID := p.CodAdministracao
	if ugID <= 0 && p.TenantID != "" {
		if parsed, err := strconv.Atoi(strings.TrimSpace(p.TenantID)); err == nil && parsed > 0 {
			ugID = parsed
		}
	}
	if ugID <= 0 {
		if probedID, _, ok := ProbeCoplan(ctx, hc, base); ok && probedID > 0 {
			ugID = probedID
		} else {
			ugID = 1
		}
	}

	ano := p.Ano
	if ano <= 0 {
		ano = time.Now().Year()
	}
	mes := p.Mes
	if mes <= 0 {
		mes = 12
	}
	limit := normalizeLimit(p.Limit)

	payload := map[string]any{
		"unidade_gestora_id": ugID,
		"exercicio":          ano,
		"mes":                mes,
		"pagina":             1,
		"limite":             limit,
	}

	target := base + "/servlet/api_consulta?despesas_v1"
	body, status, err := doPortalJSONPost(ctx, hc, target, payload)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Coplan /servlet/api_consulta?despesas_v1 retornou HTTP %d", status)
	}

	rows, total, err := decodeFlexiblePortalList(body, "despesas")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Coplan despesas_v1: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]DespesaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		fornecedor := mapGetStr(r, "fornecedor", "credor", "nome_fornecedor", "nome_credor", "razao_social")
		historico := mapGetStr(r, "historico", "descricao", "historico_empenho")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+historico), filtro) {
			continue
		}
		itemAno := int(mapGetFloat(r, "exercicio", "ano", "ano_empenho"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, DespesaItem{
			NumeroEmpenho:       mapGetStr(r, "numero_empenho", "empenho", "num_empenho", "numero", "numeroEmpenho"),
			Ano:                 itemAno,
			Data:                mapGetStr(r, "data", "data_empenho", "dt_empenho", "dataEmpenho"),
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(mapGetStr(r, "documento_fornecedor", "cpf_cnpj", "cnpj_cpf", "cnpj", "documento")),
			Orgao:               mapGetStr(r, "orgao", "nome_orgao", "unidade_gestora_descricao", "unidade_gestora"),
			Unidade:             mapGetStr(r, "unidade", "unidade_orcamentaria", "nome_unidade"),
			Funcao:              mapGetStr(r, "funcao", "descricao_funcao"),
			Elemento:            mapGetStr(r, "elemento", "elemento_despesa", "natureza_despesa", "classificacao"),
			Historico:           historico,
			ValorEmpenhado:      mapGetFloat(r, "valor_empenhado", "empenhado", "vl_empenhado", "valor"),
			ValorLiquidado:      mapGetFloat(r, "valor_liquidado", "liquidado", "vl_liquidado"),
			ValorPago:           mapGetFloat(r, "valor_pago", "pago", "vl_pago"),
			Fonte:               "Portal Municipal (Coplan)",
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
		Fonte:          "Portal Municipal (Coplan)",
		Plataforma:     "coplan_gp",
		PortalURL:      base,
		TotalRegistros: total,
		Despesas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// ---------------------------------------------------------------------------
// 3. Thema API Gateway (thema_apigw)
// ---------------------------------------------------------------------------

const themaAPIGWPrefix = "/infra/apigw/transparencia"

func buildThemaURL(baseURL, servicePath string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	low := strings.ToLower(base)
	svc := "/" + strings.TrimLeft(servicePath, "/")
	if idx := strings.Index(low, themaAPIGWPrefix); idx >= 0 {
		prefix := base[:idx+len(themaAPIGWPrefix)]
		return prefix + svc
	}
	return base + themaAPIGWPrefix + svc
}

// ProbeThemaAPIGW verifica se a URL corresponde ao Thema API Gateway, tratando HTTP 403 SafeLine WAF de forma limpa.
func ProbeThemaAPIGW(ctx context.Context, client *http.Client, baseURL string) (adminCode int, adminName string, ok bool) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return 0, "", false
	}

	target := buildThemaURL(base, "/service/basico/administracao/listaAdministracoes/")
	body, status, err := doJSONGet(ctx, portalConnectorHTTPClient(client), target)
	if err != nil {
		return 0, "", false
	}
	if status == http.StatusForbidden || status != http.StatusOK {
		return 0, "", false
	}

	rows, _, err := decodeFlexiblePortalList(body, "administracoes", "listaAdministracoes")
	if err != nil || len(rows) == 0 {
		return 0, "", false
	}

	var firstCode int
	var firstName string
	for _, r := range rows {
		code := int(mapGetFloat(r, "codigo", "codAdministracao", "administracao", "id"))
		name := mapGetStr(r, "descricao", "desAdministracao", "nome", "nomeAdministracao", "razaoSocial")
		if code == 0 && name == "" {
			continue
		}
		if firstCode == 0 && firstName == "" {
			firstCode = code
			firstName = name
		}
		up := strings.ToUpper(name)
		if strings.Contains(up, "PREFEITURA") || strings.Contains(up, "MUNIC") {
			return code, name, true
		}
	}
	if firstCode == 0 && firstName == "" {
		return 0, "", false
	}
	return firstCode, firstName, true
}

// QueryThemaDespesas consulta detalhamento de despesas no Thema API Gateway.
func QueryThemaDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("Thema API Gateway: BaseURL não informada")
	}
	hc := portalConnectorHTTPClient(client)

	adminCode := p.CodAdministracao
	if adminCode <= 0 && p.TenantID != "" {
		if parsed, err := strconv.Atoi(strings.TrimSpace(p.TenantID)); err == nil && parsed > 0 {
			adminCode = parsed
		}
	}
	if adminCode <= 0 {
		if code, _, ok := ProbeThemaAPIGW(ctx, hc, base); ok && code > 0 {
			adminCode = code
		} else {
			adminCode = 1
		}
	}

	ano := p.Ano
	if ano <= 0 {
		ano = time.Now().Year()
	}
	limit := normalizeLimit(p.Limit)

	payload := map[string]any{
		"administracao": strconv.Itoa(adminCode),
		"exercicio":     ano,
	}

	target := buildThemaURL(base, "/service/contabilidade/dotacao/consultaDetalhamentoDespesas/")
	body, status, err := doPortalJSONPost(ctx, hc, target, payload)
	if err != nil {
		return nil, err
	}
	if status == http.StatusForbidden {
		return nil, fmt.Errorf("Thema API Gateway bloqueado por WAF (HTTP 403)")
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Thema consultaDetalhamentoDespesas retornou HTTP %d", status)
	}

	rows, total, err := decodeFlexiblePortalList(body, "despesas", "detalhamentoDespesas")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Thema consultaDetalhamentoDespesas: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]DespesaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		fornecedor := mapGetStr(r, "fornecedor", "credor", "nomeFornecedor", "razaoSocial")
		historico := mapGetStr(r, "historico", "descricao", "especificacao")
		elemento := mapGetStr(r, "elemento", "elementoDespesa", "descricaoElemento", "dotacao", "despesa")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+historico+" "+elemento), filtro) {
			continue
		}
		itemAno := int(mapGetFloat(r, "exercicio", "ano"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, DespesaItem{
			NumeroEmpenho:       mapGetStr(r, "numeroEmpenho", "empenho", "numEmpenho", "numero", "codigo"),
			Ano:                 itemAno,
			Data:                mapGetStr(r, "data", "dataEmpenho", "dataMovimento"),
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(mapGetStr(r, "documentoFornecedor", "cnpjCpf", "cpfCnpj", "cnpj", "documento")),
			Orgao:               mapGetStr(r, "orgao", "descricaoOrgao", "nomeOrgao", "administracao"),
			Unidade:             mapGetStr(r, "unidade", "descricaoUnidade", "unidadeOrcamentaria"),
			Funcao:              mapGetStr(r, "funcao", "descricaoFuncao"),
			Elemento:            elemento,
			Historico:           historico,
			ValorEmpenhado:      mapGetFloat(r, "valorEmpenhado", "empenhado", "vlrEmpenhado", "dotacaoAtualizada", "valor"),
			ValorLiquidado:      mapGetFloat(r, "valorLiquidado", "liquidado", "vlrLiquidado"),
			ValorPago:           mapGetFloat(r, "valorPago", "pago", "vlrPago"),
			Fonte:               "Portal Municipal (Thema API Gateway)",
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
		Fonte:          "Portal Municipal (Thema API Gateway)",
		Plataforma:     "thema_apigw",
		PortalURL:      base,
		TotalRegistros: total,
		Despesas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}
