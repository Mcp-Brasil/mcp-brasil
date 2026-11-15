package transparencia

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// E&L PortalTP (*.portaltp.com.br) Connector
// ASMX Web Services (/api/transparencia.asmx/*) returning XML <string> JSON
// ---------------------------------------------------------------------------

const (
	portalTPUserAgent  = "Mozilla/5.0 (compatible; mcp-brasil/1.0)"
	portalTPFonte      = "Portal Municipal (E&L PortalTP)"
	portalTPPlataforma = "portaltp"
)

func ensureHTTPClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func normalizePortalTPBase(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return ""
	}
	low := strings.ToLower(base)
	const suffix = "/api/transparencia.asmx"
	if strings.HasSuffix(low, suffix) {
		base = base[:len(base)-len(suffix)]
	}
	return strings.TrimRight(base, "/")
}

func doPortalTPGet(ctx context.Context, client *http.Client, targetURL string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", portalTPUserAgent)
	req.Header.Set("Accept", "application/xml, text/xml, application/json, */*")

	resp, err := ensureHTTPClient(client).Do(req)
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

// ProbePortalTP verifica se a URL base responde como um portal E&L PortalTP (/api/transparencia.asmx).
func ProbePortalTP(ctx context.Context, client *http.Client, baseURL string) bool {
	base := normalizePortalTPBase(baseURL)
	if base == "" {
		return false
	}

	// 1. Probe WSDL oficial do ASMX
	if body, status, err := doPortalTPGet(ctx, client, base+"/api/transparencia.asmx?WSDL"); err == nil && status == http.StatusOK {
		low := strings.ToLower(string(body))
		if strings.Contains(low, "json_empenhos") || strings.Contains(low, "tempuri.org") {
			return true
		}
	}

	// 2. Fallback: testa json_orcamento_despesas verificando envelope <string XML
	if body, status, err := doPortalTPGet(ctx, client, base+"/api/transparencia.asmx/json_orcamento_despesas?ano=2025"); err == nil && status == http.StatusOK {
		low := strings.ToLower(strings.TrimSpace(string(body)))
		if strings.Contains(low, "<string") {
			return true
		}
	}

	return false
}

// unwrapPortalTPXML extrai o conteúdo textual JSON de um envelope ASMX <string xmlns="http://tempuri.org/">...</string>.
func unwrapPortalTPXML(body []byte) ([]byte, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, nil
	}
	if strings.HasPrefix(trimmed, "<") {
		var s string
		if err := xml.Unmarshal([]byte(trimmed), &s); err != nil {
			return nil, fmt.Errorf("erro ao decodificar envelope XML PortalTP: %w", err)
		}
		return []byte(strings.TrimSpace(s)), nil
	}
	return []byte(trimmed), nil
}

func decodePortalTPRecordsMeta(body []byte) ([]map[string]any, int, bool, error) {
	raw, err := unwrapPortalTPXML(body)
	if err != nil {
		return nil, 0, false, err
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || strings.EqualFold(trimmed, "null") || trimmed == "[]" {
		return nil, 0, false, nil
	}

	if strings.HasPrefix(trimmed, "[") {
		var rows []map[string]any
		if err := json.Unmarshal([]byte(trimmed), &rows); err != nil {
			return nil, 0, false, fmt.Errorf("erro ao decodificar JSON array PortalTP: %w", err)
		}
		return rows, 0, false, nil
	}

	if strings.HasPrefix(trimmed, "{") {
		var obj map[string]any
		if err := json.Unmarshal([]byte(trimmed), &obj); err != nil {
			return nil, 0, false, fmt.Errorf("erro ao decodificar JSON objeto PortalTP: %w", err)
		}
		total := mapGetInt(obj, "total", "Total", "totalRegistros", "TotalRegistros", "totalElements", "TotalElements", "count", "Count")
		hasExplicitTotal := total > 0
		for _, key := range []string{"registros", "Registros", "dados", "data", "items", "content", "rows", "lista", "result"} {
			if rawArr, ok := obj[key]; ok {
				if arr, ok := rawArr.([]any); ok {
					rows := make([]map[string]any, 0, len(arr))
					for _, el := range arr {
						if item, ok := el.(map[string]any); ok {
							rows = append(rows, item)
						}
					}
					return rows, total, hasExplicitTotal, nil
				}
			}
		}
		return nil, total, hasExplicitTotal, nil
	}

	return nil, 0, false, fmt.Errorf("payload PortalTP inesperado: %.64s", trimmed)
}

func decodePortalTPRecords(body []byte) ([]map[string]any, int, error) {
	rows, total, hasTotal, err := decodePortalTPRecordsMeta(body)
	if err != nil {
		return nil, 0, err
	}
	if !hasTotal {
		total = len(rows)
	}
	return rows, total, nil
}

func resolvePortalTPYears(p PortalQueryParams) []int {
	if p.Ano > 0 {
		return []int{p.Ano}
	}
	if len(p.Inicio) >= 4 {
		if y, err := strconv.Atoi(p.Inicio[:4]); err == nil && y > 1900 {
			return []int{y}
		}
	}
	nowY := time.Now().Year()
	return []int{nowY, nowY - 1, nowY - 2}
}

func mapGetInt(m map[string]any, keys ...string) int {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			switch t := v.(type) {
			case float64:
				return int(t)
			case int:
				return t
			case string:
				if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
					return n
				}
			}
		}
	}
	return 0
}

func maskPortalTPDoc(doc string) string {
	clean := strings.TrimSpace(doc)
	if clean == "" {
		return ""
	}
	if strings.Contains(clean, "*") {
		return clean
	}
	digits := cleanDigits(clean)
	if len(digits) == 11 {
		return maskCPF(clean, digits)
	}
	return clean
}

// fetchPortalTPMethod consulta um método ASMX tentando paginação (pagina=1..N&tamanho=pageSize) e fallback sem paginação.
func fetchPortalTPMethod(ctx context.Context, client *http.Client, base, method string, baseParams url.Values, limit int) ([]map[string]any, int, int, error) {
	pageSize := normalizePageSize(limit, 200)
	var allRows []map[string]any
	total := 0
	hasTotal := false
	prevBody := ""

	for pagina := 1; pagina <= 10; pagina++ {
		pagParams := url.Values{}
		for k, vals := range baseParams {
			pagParams[k] = append([]string(nil), vals...)
		}
		pagParams.Set("pagina", strconv.Itoa(pagina))
		pagParams.Set("tamanho", strconv.Itoa(pageSize))

		pagURL := fmt.Sprintf("%s/api/transparencia.asmx/%s?%s", base, method, pagParams.Encode())
		body, status, err := doPortalTPGet(ctx, client, pagURL)
		if err != nil {
			if pagina == 1 {
				return nil, 0, 0, err
			}
			break
		}
		bodyStr := strings.TrimSpace(string(body))
		if status == http.StatusOK {
			if pagina > 1 && (bodyStr == "" || bodyStr == prevBody) {
				break
			}
			prevBody = bodyStr
			if rows, pageTotal, pageHasTotal, decErr := decodePortalTPRecordsMeta(body); decErr == nil {
				if pageHasTotal && pageTotal > total {
					total = pageTotal
					hasTotal = true
				}
				if len(rows) == 0 {
					break
				}
				allRows = append(allRows, rows...)
				if len(allRows) >= limit || (hasTotal && total > 0 && len(allRows) >= total) {
					break
				}
				continue
			}
		}

		if pagina > 1 {
			break
		}

		if ctx.Err() != nil {
			return nil, 0, status, ctx.Err()
		}

		// Fallback sem paginação para servidores ASMX que não aceitam pagina/tamanho
		unpagURL := fmt.Sprintf("%s/api/transparencia.asmx/%s?%s", base, method, baseParams.Encode())
		body2, status2, err2 := doPortalTPGet(ctx, client, unpagURL)
		if err2 != nil {
			return nil, 0, 0, err2
		}
		if status2 != http.StatusOK {
			return nil, 0, status2, nil
		}
		rows, total2, decErr := decodePortalTPRecords(body2)
		if decErr != nil {
			return nil, 0, status2, decErr
		}
		return rows, total2, status2, nil
	}

	if !hasTotal {
		total = len(allRows)
	}
	if len(allRows) > limit {
		allRows = allRows[:limit]
	}
	return allRows, total, http.StatusOK, nil
}

func extractPortalTPRemuneracao(r map[string]any) (salBase, salBruto, descontos, salLiquido float64) {
	salBase = mapGetFloat(r, "valor_salario_base", "salario_base", "vencimento_base", "valor_rem_base", "valor_remuneracao_base")
	salBruto = mapGetFloat(r, "valor_remuneracao_bruta", "valor_rem_bruta", "valor_remuneracao", "valor_rem_total", "salario_bruto", "valor_bruto", "remuneracao_bruta", "remuneracao", "total_proventos")
	descontos = mapGetFloat(r, "valor_descontos", "valor_rem_descontos", "valor_remuneracao_descontos", "descontos", "total_descontos")
	salLiquido = mapGetFloat(r, "valor_remuneracao_liquida", "valor_rem_liquida", "valor_liquido", "salario_liquido", "remuneracao_liquida")

	var remComponentsSum, extraDesc float64
	for k := range r {
		lk := strings.ToLower(k)
		if !strings.HasPrefix(lk, "valor_rem") {
			continue
		}
		val := mapGetFloat(r, k)
		switch {
		case strings.Contains(lk, "descont"):
			if descontos == 0 {
				extraDesc += val
			}
		case strings.Contains(lk, "liquid"):
			if salLiquido == 0 {
				salLiquido = val
			}
		case lk == "valor_remuneracao_bruta" || lk == "valor_rem_bruta" || lk == "valor_remuneracao" || lk == "valor_rem_total":
			// já tratado como total bruto
		default:
			remComponentsSum += val
		}
	}

	if salBruto == 0 && remComponentsSum > 0 {
		salBruto = remComponentsSum
	}
	if descontos == 0 && extraDesc > 0 {
		descontos = extraDesc
	}
	if salLiquido == 0 && salBruto > 0 && descontos > 0 {
		salLiquido = salBruto - descontos
	}
	return salBase, salBruto, descontos, salLiquido
}

// QueryPortalTPServidores consulta servidores públicos no endpoint E&L PortalTP (json_servidores).
func QueryPortalTPServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ServidoresResult, error) {
	base := normalizePortalTPBase(p.BaseURL)
	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fetchLimit := limit
	if filtro != "" && fetchLimit < 500 {
		fetchLimit = 500
	}

	var lastErr error
	var lastStatus int

	for _, ano := range resolvePortalTPYears(p) {
		mes := p.Mes
		if mes < 0 {
			mes = 0
		}
		params := url.Values{}
		params.Set("ano", strconv.Itoa(ano))
		params.Set("mes", strconv.Itoa(mes))

		rows, total, status, err := fetchPortalTPMethod(ctx, client, base, "json_servidores", params, fetchLimit)
		if err != nil {
			lastErr = err
			if p.Ano > 0 {
				return nil, err
			}
			continue
		}
		lastStatus = status
		if status != http.StatusOK || len(rows) == 0 {
			continue
		}

		items := make([]ServidorItem, 0, minInt(limit, len(rows)))
		matchedTotal := 0
		for _, r := range rows {
			nome := mapGetStr(r, "nome", "servidor", "nome_servidor")
			if filtro != "" && !strings.Contains(strings.ToLower(nome), filtro) {
				continue
			}
			matchedTotal++
			if len(items) >= limit {
				continue
			}
			salBase, salBruto, descontos, salLiquido := extractPortalTPRemuneracao(r)
			comp := mapGetStr(r, "competencia", "mes_ano")
			if comp == "" {
				if mes > 0 {
					comp = fmt.Sprintf("%02d/%04d", mes, ano)
				} else {
					comp = strconv.Itoa(ano)
				}
			}
			items = append(items, ServidorItem{
				Nome:           nome,
				Matricula:      mapGetStr(r, "matricula", "num_matricula", "codigo"),
				CPFParcial:     maskCPF("", mapGetStr(r, "documento", "cpf", "cpf_servidor", "num_cpf")),
				Cargo:          mapGetStr(r, "cargo", "descricao_cargo", "nome_cargo"),
				Funcao:         mapGetStr(r, "funcao", "funcao_exercida"),
				Lotacao:        mapGetStr(r, "lotacao", "unidade", "departamento", "secretaria"),
				Orgao:          mapGetStr(r, "orgao", "entidade", "nome_orgao"),
				Vinculo:        mapGetStr(r, "regime", "vinculo", "tipo_vinculo"),
				Situacao:       mapGetStr(r, "situacao", "status"),
				SalarioBase:    salBase,
				SalarioBruto:   salBruto,
				Descontos:      descontos,
				SalarioLiquido: salLiquido,
				Competencia:    comp,
				CargaHoraria:   mapGetStr(r, "horas_semanais", "carga_horaria"),
				DataAdmissao:   mapGetStr(r, "data_admissao", "admissao"),
				Fonte:          portalTPFonte,
				PortalURL:      base,
			})
		}

		if len(items) == 0 && filtro != "" {
			continue
		}
		if filtro != "" || total == 0 {
			total = matchedTotal
		}

		return &ServidoresResult{
			Municipio:      m,
			Fonte:          portalTPFonte,
			Plataforma:     portalTPPlataforma,
			PortalURL:      base,
			TotalRegistros: total,
			Servidores:     items,
			Disponivel:     true,
		}, nil
	}

	if lastErr != nil && lastStatus == 0 {
		return nil, lastErr
	}
	if p.Ano > 0 && lastStatus != 0 && lastStatus != http.StatusOK {
		return nil, fmt.Errorf("PortalTP json_servidores retornou HTTP %d", lastStatus)
	}

	return &ServidoresResult{
		Municipio:  m,
		Fonte:      portalTPFonte,
		Plataforma: portalTPPlataforma,
		PortalURL:  base,
		Servidores: []ServidorItem{},
		Disponivel: true,
		Mensagem:   "Nenhum servidor encontrado para os filtros e período consultados.",
	}, nil
}

// QueryPortalTPDespesas consulta empenhos/despesas no endpoint E&L PortalTP (json_empenhos).
func QueryPortalTPDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	base := normalizePortalTPBase(p.BaseURL)
	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fetchLimit := limit
	if filtro != "" && fetchLimit < 500 {
		fetchLimit = 500
	}

	var lastErr error
	var lastStatus int

	for _, ano := range resolvePortalTPYears(p) {
		mes := p.Mes
		if mes < 0 {
			mes = 0
		}
		params := url.Values{}
		params.Set("ano", strconv.Itoa(ano))
		params.Set("mes", strconv.Itoa(mes))

		rows, total, status, err := fetchPortalTPMethod(ctx, client, base, "json_empenhos", params, fetchLimit)
		if err != nil {
			lastErr = err
			if p.Ano > 0 {
				return nil, err
			}
			continue
		}
		lastStatus = status
		if status != http.StatusOK || len(rows) == 0 {
			continue
		}

		items := make([]DespesaItem, 0, minInt(limit, len(rows)))
		matchedTotal := 0
		for _, r := range rows {
			fornecedor := mapGetStr(r, "favorecido", "nome_favorecido", "fornecedor", "credor")
			historico := mapGetStr(r, "historico", "descricao", "objeto")
			if filtro != "" && !strings.Contains(strings.ToLower(fornecedor), filtro) && !strings.Contains(strings.ToLower(historico), filtro) {
				continue
			}
			matchedTotal++
			if len(items) >= limit {
				continue
			}
			itemAno := mapGetInt(r, "ano", "exercicio")
			if itemAno == 0 {
				itemAno = ano
			}
			doc := maskPortalTPDoc(mapGetStr(r, "documento_favorecido", "cpf_cnpj", "documento", "cnpj_cpf"))

			items = append(items, DespesaItem{
				NumeroEmpenho:       mapGetStr(r, "empenho", "numero_empenho", "numero", "num_empenho"),
				Ano:                 itemAno,
				Data:                mapGetStr(r, "data", "data_empenho", "dt_empenho"),
				Fornecedor:          fornecedor,
				DocumentoFornecedor: doc,
				Orgao:               mapGetStr(r, "orgao", "nome_orgao", "entidade"),
				Unidade:             mapGetStr(r, "unidade", "unidade_orcamentaria", "nome_unidade"),
				Funcao:              mapGetStr(r, "funcao", "nome_funcao"),
				Elemento:            mapGetStr(r, "elemento", "elemento_despesa", "natureza_despesa", "especificacao_elemento"),
				Historico:           historico,
				ValorEmpenhado:      mapGetFloat(r, "valor_empenhado", "empenhado", "valor", "vlr_empenhado"),
				ValorLiquidado:      mapGetFloat(r, "valor_liquidado", "liquidado", "vlr_liquidado"),
				ValorPago:           mapGetFloat(r, "valor_pago", "pago", "vlr_pago"),
				Fonte:               portalTPFonte,
			})
		}

		if len(items) == 0 && filtro != "" {
			continue
		}
		if filtro != "" || total == 0 {
			total = matchedTotal
		}

		return &DespesasResult{
			Municipio:      m,
			Fonte:          portalTPFonte,
			Plataforma:     portalTPPlataforma,
			PortalURL:      base,
			TotalRegistros: total,
			Despesas:       items,
			Disponivel:     true,
		}, nil
	}

	if lastErr != nil && lastStatus == 0 {
		return nil, lastErr
	}
	if p.Ano > 0 && lastStatus != 0 && lastStatus != http.StatusOK {
		return nil, fmt.Errorf("PortalTP json_empenhos retornou HTTP %d", lastStatus)
	}

	return &DespesasResult{
		Municipio:  m,
		Fonte:      portalTPFonte,
		Plataforma: portalTPPlataforma,
		PortalURL:  base,
		Despesas:   []DespesaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma despesa encontrada para o período informado.",
	}, nil
}

// QueryPortalTPReceitas consulta receitas no endpoint E&L PortalTP (json_execucao_receitas).
func QueryPortalTPReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	base := normalizePortalTPBase(p.BaseURL)
	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fetchLimit := limit
	if filtro != "" && fetchLimit < 500 {
		fetchLimit = 500
	}

	var lastErr error
	var lastStatus int

	for _, ano := range resolvePortalTPYears(p) {
		mes := p.Mes
		if mes < 0 {
			mes = 0
		}
		params := url.Values{}
		params.Set("ano", strconv.Itoa(ano))
		params.Set("mes", strconv.Itoa(mes))

		rows, total, status, err := fetchPortalTPMethod(ctx, client, base, "json_execucao_receitas", params, fetchLimit)
		if err != nil {
			lastErr = err
			if p.Ano > 0 {
				return nil, err
			}
			continue
		}
		lastStatus = status
		if status != http.StatusOK || len(rows) == 0 {
			continue
		}

		items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
		matchedTotal := 0
		for _, r := range rows {
			desc := mapGetStr(r, "descricao", "especificacao", "nome_receita", "receita")
			if filtro != "" && !strings.Contains(strings.ToLower(desc), filtro) {
				continue
			}
			matchedTotal++
			if len(items) >= limit {
				continue
			}
			items = append(items, ReceitaItem{
				Codigo:          mapGetStr(r, "codigo", "codigo_receita", "conta", "classificacao", "rubrica"),
				Descricao:       desc,
				Categoria:       mapGetStr(r, "categoria", "fonte_recurso", "origem", "especie"),
				Data:            mapGetStr(r, "data", "competencia"),
				ValorPrevisto:   mapGetFloat(r, "valor_previsto", "previsto", "orcado", "valor_orcado", "previsao_atualizada"),
				ValorArrecadado: mapGetFloat(r, "valor_arrecadado", "arrecadado", "realizado", "valor_realizado", "arrecadado_ate_mes"),
				Fonte:           portalTPFonte,
			})
		}

		if len(items) == 0 && filtro != "" {
			continue
		}
		if filtro != "" || total == 0 {
			total = matchedTotal
		}

		return &ReceitasResult{
			Municipio:      m,
			Fonte:          portalTPFonte,
			Plataforma:     portalTPPlataforma,
			PortalURL:      base,
			TotalRegistros: total,
			Receitas:       items,
			Disponivel:     true,
		}, nil
	}

	if lastErr != nil && lastStatus == 0 {
		return nil, lastErr
	}
	if p.Ano > 0 && lastStatus != 0 && lastStatus != http.StatusOK {
		return nil, fmt.Errorf("PortalTP json_execucao_receitas retornou HTTP %d", lastStatus)
	}

	return &ReceitasResult{
		Municipio:  m,
		Fonte:      portalTPFonte,
		Plataforma: portalTPPlataforma,
		PortalURL:  base,
		Receitas:   []ReceitaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma receita encontrada para o período informado.",
	}, nil
}

// QueryPortalTPLicitacoes consulta licitações no endpoint E&L PortalTP (json_licitacoes).
func QueryPortalTPLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LicitacoesResult, error) {
	base := normalizePortalTPBase(p.BaseURL)
	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fetchLimit := limit
	if filtro != "" && fetchLimit < 500 {
		fetchLimit = 500
	}

	var lastErr error
	var lastStatus int

	for _, ano := range resolvePortalTPYears(p) {
		params := url.Values{}
		params.Set("ano", strconv.Itoa(ano))
		params.Set("mes", "0")
		params.Set("extra", "true")
		params.Set("situacao", "")

		rows, total, status, err := fetchPortalTPMethod(ctx, client, base, "json_licitacoes", params, fetchLimit)
		if err != nil {
			lastErr = err
			if p.Ano > 0 {
				return nil, err
			}
			continue
		}
		lastStatus = status
		if status != http.StatusOK || len(rows) == 0 {
			continue
		}

		items := make([]LicitacaoItem, 0, minInt(limit, len(rows)))
		matchedTotal := 0
		for _, r := range rows {
			objeto := mapGetStr(r, "objeto", "descricao")
			if filtro != "" && !strings.Contains(strings.ToLower(objeto), filtro) {
				continue
			}
			matchedTotal++
			if len(items) >= limit {
				continue
			}
			itemAno := mapGetInt(r, "ano", "exercicio")
			if itemAno == 0 {
				itemAno = ano
			}
			link := mapGetStr(r, "link", "url")
			if link == "" {
				link = base
			}
			items = append(items, LicitacaoItem{
				Numero:          mapGetStr(r, "numero", "numero_licitacao", "numero_processo", "edital"),
				Ano:             itemAno,
				Modalidade:      mapGetStr(r, "modalidade", "descricao_modalidade", "tipo"),
				Situacao:        mapGetStr(r, "situacao", "status"),
				DataAbertura:    mapGetStr(r, "data_abertura", "data_publicacao", "data", "dt_abertura"),
				Objeto:          objeto,
				ValorEstimado:   mapGetFloat(r, "valor_estimado", "valor_maximo", "valor_previsto", "valor"),
				ValorHomologado: mapGetFloat(r, "valor_homologado", "valor_vencedor", "valor_adjudicado", "valor_total"),
				Link:            link,
				Fonte:           portalTPFonte,
			})
		}

		if len(items) == 0 && filtro != "" {
			continue
		}
		if filtro != "" || total == 0 {
			total = matchedTotal
		}

		return &LicitacoesResult{
			Municipio:      m,
			Fonte:          portalTPFonte,
			Plataforma:     portalTPPlataforma,
			PortalURL:      base,
			TotalRegistros: total,
			Licitacoes:     items,
			Disponivel:     true,
		}, nil
	}

	if lastErr != nil && lastStatus == 0 {
		return nil, lastErr
	}
	if p.Ano > 0 && lastStatus != 0 && lastStatus != http.StatusOK {
		return nil, fmt.Errorf("PortalTP json_licitacoes retornou HTTP %d", lastStatus)
	}

	return &LicitacoesResult{
		Municipio:  m,
		Fonte:      portalTPFonte,
		Plataforma: portalTPPlataforma,
		PortalURL:  base,
		Licitacoes: []LicitacaoItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma licitação encontrada para o período informado.",
	}, nil
}

// QueryPortalTPContratos consulta contratos no endpoint E&L PortalTP (json_contratos).
func QueryPortalTPContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ContratosResult, error) {
	base := normalizePortalTPBase(p.BaseURL)
	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fetchLimit := limit
	if filtro != "" && fetchLimit < 500 {
		fetchLimit = 500
	}

	var lastErr error
	var lastStatus int

	for _, ano := range resolvePortalTPYears(p) {
		params := url.Values{}
		params.Set("ano", strconv.Itoa(ano))

		rows, total, status, err := fetchPortalTPMethod(ctx, client, base, "json_contratos", params, fetchLimit)
		if err != nil {
			lastErr = err
			if p.Ano > 0 {
				return nil, err
			}
			continue
		}
		lastStatus = status
		if status != http.StatusOK || len(rows) == 0 {
			continue
		}

		items := make([]ContratoItem, 0, minInt(limit, len(rows)))
		matchedTotal := 0
		for _, r := range rows {
			fornecedor := mapGetStr(r, "fornecedor", "contratado", "nome_contratado", "favorecido", "razao_social")
			objeto := mapGetStr(r, "objeto", "descricao")
			if filtro != "" && !strings.Contains(strings.ToLower(fornecedor), filtro) && !strings.Contains(strings.ToLower(objeto), filtro) {
				continue
			}
			matchedTotal++
			if len(items) >= limit {
				continue
			}
			itemAno := mapGetInt(r, "ano", "exercicio")
			if itemAno == 0 {
				itemAno = ano
			}
			doc := maskPortalTPDoc(mapGetStr(r, "documento_fornecedor", "documento_contratado", "documento", "cpf_cnpj", "cnpj_cpf"))

			items = append(items, ContratoItem{
				Numero:              mapGetStr(r, "numero", "numero_contrato", "contrato"),
				Ano:                 itemAno,
				Fornecedor:          fornecedor,
				DocumentoFornecedor: doc,
				Objeto:              objeto,
				Situacao:            mapGetStr(r, "situacao", "status"),
				DataAssinatura:      mapGetStr(r, "data_assinatura", "data", "dt_assinatura"),
				VigenciaInicio:      mapGetStr(r, "vigencia_inicio", "data_inicio", "inicio_vigencia"),
				VigenciaFim:         mapGetStr(r, "vigencia_fim", "data_fim", "fim_vigencia", "data_termino"),
				Valor:               mapGetFloat(r, "valor", "valor_contrato", "valor_global", "valor_inicial"),
				Fonte:               portalTPFonte,
			})
		}

		if len(items) == 0 && filtro != "" {
			continue
		}
		if filtro != "" || total == 0 {
			total = matchedTotal
		}

		return &ContratosResult{
			Municipio:      m,
			Fonte:          portalTPFonte,
			Plataforma:     portalTPPlataforma,
			PortalURL:      base,
			TotalRegistros: total,
			Contratos:      items,
			Disponivel:     true,
		}, nil
	}

	if lastErr != nil && lastStatus == 0 {
		return nil, lastErr
	}
	if p.Ano > 0 && lastStatus != 0 && lastStatus != http.StatusOK {
		return nil, fmt.Errorf("PortalTP json_contratos retornou HTTP %d", lastStatus)
	}

	return &ContratosResult{
		Municipio:  m,
		Fonte:      portalTPFonte,
		Plataforma: portalTPPlataforma,
		PortalURL:  base,
		Contratos:  []ContratoItem{},
		Disponivel: true,
		Mensagem:   "Nenhum contrato encontrado para o período informado.",
	}, nil
}

// QueryPortalTPDiarias consulta diárias e passagens no endpoint E&L PortalTP (json_diarias).
func QueryPortalTPDiarias(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DiariasResult, error) {
	base := normalizePortalTPBase(p.BaseURL)
	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fetchLimit := limit
	if filtro != "" && fetchLimit < 500 {
		fetchLimit = 500
	}

	var lastErr error
	var lastStatus int

	for _, ano := range resolvePortalTPYears(p) {
		mes := p.Mes
		if mes < 0 {
			mes = 0
		}
		params := url.Values{}
		params.Set("ano", strconv.Itoa(ano))
		params.Set("mes", strconv.Itoa(mes))

		rows, total, status, err := fetchPortalTPMethod(ctx, client, base, "json_diarias", params, fetchLimit)
		if err != nil {
			lastErr = err
			if p.Ano > 0 {
				return nil, err
			}
			continue
		}
		lastStatus = status
		if status != http.StatusOK || len(rows) == 0 {
			continue
		}

		items := make([]DiariaItem, 0, minInt(limit, len(rows)))
		matchedTotal := 0
		for _, r := range rows {
			beneficiario := mapGetStr(r, "beneficiario", "favorecido", "nome", "servidor", "nome_servidor")
			destino := mapGetStr(r, "destino", "local_destino", "cidade_destino")
			if filtro != "" && !strings.Contains(strings.ToLower(beneficiario), filtro) && !strings.Contains(strings.ToLower(destino), filtro) {
				continue
			}
			matchedTotal++
			if len(items) >= limit {
				continue
			}
			dataSaida := mapGetStr(r, "data_saida", "dt_saida")
			data := mapGetStr(r, "data", "data_pagamento", "dt_pagamento")
			if data == "" {
				data = dataSaida
			}
			items = append(items, DiariaItem{
				Beneficiario:  beneficiario,
				Matricula:     mapGetStr(r, "matricula", "num_matricula"),
				Cargo:         mapGetStr(r, "cargo", "descricao_cargo"),
				Destino:       destino,
				Motivo:        mapGetStr(r, "motivo", "finalidade", "objetivo", "historico", "descricao"),
				Data:          data,
				DataSaida:     dataSaida,
				DataChegada:   mapGetStr(r, "data_chegada", "data_retorno", "dt_retorno"),
				NumeroDiarias: mapGetFloat(r, "numero_diarias", "quantidade", "qtd_diarias"),
				Valor:         mapGetFloat(r, "valor", "valor_diaria", "valor_total", "valor_pago"),
				Fonte:         portalTPFonte,
			})
		}

		if len(items) == 0 && filtro != "" {
			continue
		}
		if filtro != "" || total == 0 {
			total = matchedTotal
		}

		return &DiariasResult{
			Municipio:      m,
			Fonte:          portalTPFonte,
			Plataforma:     portalTPPlataforma,
			PortalURL:      base,
			TotalRegistros: total,
			Diarias:        items,
			Disponivel:     true,
		}, nil
	}

	if lastErr != nil && lastStatus == 0 {
		return nil, lastErr
	}
	if p.Ano > 0 && lastStatus != 0 && lastStatus != http.StatusOK {
		return nil, fmt.Errorf("PortalTP json_diarias retornou HTTP %d", lastStatus)
	}

	return &DiariasResult{
		Municipio:  m,
		Fonte:      portalTPFonte,
		Plataforma: portalTPPlataforma,
		PortalURL:  base,
		Diarias:    []DiariaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma diária encontrada para o período consultado.",
	}, nil
}

// QueryPortalTPObras consulta obras públicas no endpoint E&L PortalTP (json_obras).
func QueryPortalTPObras(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ObrasResult, error) {
	base := normalizePortalTPBase(p.BaseURL)
	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fetchLimit := limit
	if filtro != "" && fetchLimit < 500 {
		fetchLimit = 500
	}

	var lastErr error
	var lastStatus int

	for _, ano := range resolvePortalTPYears(p) {
		mes := p.Mes
		if mes < 0 {
			mes = 0
		}
		params := url.Values{}
		params.Set("ano", strconv.Itoa(ano))
		params.Set("mes", strconv.Itoa(mes))

		rows, total, status, err := fetchPortalTPMethod(ctx, client, base, "json_obras", params, fetchLimit)
		if err != nil {
			lastErr = err
			if p.Ano > 0 {
				return nil, err
			}
			continue
		}
		lastStatus = status
		if status != http.StatusOK || len(rows) == 0 {
			continue
		}

		items := make([]ObraItem, 0, minInt(limit, len(rows)))
		matchedTotal := 0
		for _, r := range rows {
			titulo := mapGetStr(r, "titulo", "nome", "descricao", "objeto")
			objeto := mapGetStr(r, "objeto", "descricao", "titulo")
			if filtro != "" && !strings.Contains(strings.ToLower(titulo), filtro) && !strings.Contains(strings.ToLower(objeto), filtro) {
				continue
			}
			matchedTotal++
			if len(items) >= limit {
				continue
			}
			items = append(items, ObraItem{
				Codigo:      mapGetStr(r, "codigo", "codigo_obra", "id", "numero"),
				Titulo:      titulo,
				Objeto:      objeto,
				Orgao:       mapGetStr(r, "orgao", "secretaria", "unidade"),
				Endereco:    mapGetStr(r, "endereco", "localizacao", "local"),
				Bairro:      mapGetStr(r, "bairro"),
				DataInicial: mapGetStr(r, "data_inicial", "data_inicio", "dt_inicio"),
				DataFinal:   mapGetStr(r, "data_final", "data_fim", "data_termino", "previsao_termino"),
				Situacao:    mapGetStr(r, "situacao", "status"),
				Valor:       mapGetFloat(r, "valor", "valor_obra", "valor_contrato", "valor_previsto"),
				Fonte:       portalTPFonte,
			})
		}

		if len(items) == 0 && filtro != "" {
			continue
		}
		if filtro != "" || total == 0 {
			total = matchedTotal
		}

		return &ObrasResult{
			Municipio:      m,
			Fonte:          portalTPFonte,
			Plataforma:     portalTPPlataforma,
			PortalURL:      base,
			TotalRegistros: total,
			Obras:          items,
			Disponivel:     true,
		}, nil
	}

	if lastErr != nil && lastStatus == 0 {
		return nil, lastErr
	}
	if p.Ano > 0 && lastStatus != 0 && lastStatus != http.StatusOK {
		return nil, fmt.Errorf("PortalTP json_obras retornou HTTP %d", lastStatus)
	}

	return &ObrasResult{
		Municipio:  m,
		Fonte:      portalTPFonte,
		Plataforma: portalTPPlataforma,
		PortalURL:  base,
		Obras:      []ObraItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma obra encontrada para o período consultado.",
	}, nil
}
