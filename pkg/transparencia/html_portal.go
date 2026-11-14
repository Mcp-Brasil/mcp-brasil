package transparencia

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// Conector Genérico para Portais Municipais HTML sem API Estruturada
// ---------------------------------------------------------------------------

var (
	errHTMLAntiBotBlocked = errors.New("portal HTML protegido por verificação anti-bot/WAF")
	errHTMLNoTableRows    = errors.New("portal HTML acessível, porém sem tabelas estáticas extraíveis para o módulo")

	htmlScriptStyleRegex = regexp.MustCompile(`(?is)<(script|style|noscript|svg)[^>]*>.*?</(script|style|noscript|svg)>`)
	htmlTableRegex       = regexp.MustCompile(`(?is)<table\b[^>]*>(.*?)</table>`)
	htmlTRRegex          = regexp.MustCompile(`(?is)<tr\b[^>]*>(.*?)</tr>`)
	htmlCellRegex        = regexp.MustCompile(`(?is)<(t[hd])\b([^>]*)>(.*?)</t[hd]>`)
	htmlAnchorFullRegex  = regexp.MustCompile(`(?is)<a\b([^>]*)>(.*?)</a>`)
	htmlIframeRegex      = regexp.MustCompile(`(?is)<iframe\b([^>]*)>`)
	htmlAttrHrefRegex    = regexp.MustCompile(`(?is)\bhref\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	htmlAttrSrcRegex     = regexp.MustCompile(`(?is)\bsrc\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	htmlAnyTagRegex      = regexp.MustCompile(`(?s)<[^>]+>`)
	htmlSpaceRegex       = regexp.MustCompile(`\s+`)
)

// IsHTMLPortalPlatform indica se a plataforma corresponde a um portal municipal HTML sem API estruturada.
func IsHTMLPortalPlatform(plat string) bool {
	p := strings.ToLower(strings.TrimSpace(plat))
	if p == "" {
		return false
	}
	if p == "html_portal" || strings.HasSuffix(p, "_html") {
		return true
	}
	switch p {
	case "fiorilli_web", "el_transparencia", "betha", "sonner":
		return true
	}
	return false
}

func htmlPlatformLabel(plat string) string {
	p := strings.TrimSpace(plat)
	if p == "" || p == "html_portal" {
		return "HTML"
	}
	return p
}

type htmlLink struct {
	URL   string
	Label string
}

type htmlTable struct {
	Headers []string
	Rows    [][]string
	Links   [][]string
}

func cleanHTMLText(s string) string {
	noTags := htmlAnyTagRegex.ReplaceAllString(s, " ")
	unescaped := html.UnescapeString(noTags)
	unescaped = strings.ReplaceAll(unescaped, "\u00a0", " ")
	return strings.TrimSpace(htmlSpaceRegex.ReplaceAllString(unescaped, " "))
}

func extractAttrValue(attrs string, re *regexp.Regexp) string {
	m := re.FindStringSubmatch(attrs)
	if len(m) == 0 {
		return ""
	}
	for i := 1; i < len(m); i++ {
		if v := strings.TrimSpace(m[i]); v != "" {
			return html.UnescapeString(v)
		}
	}
	return ""
}

func resolveRelativeHTTPURL(baseParsed *url.URL, rawRef string) string {
	ref := strings.TrimSpace(rawRef)
	if ref == "" || strings.HasPrefix(ref, "#") {
		return ""
	}
	low := strings.ToLower(ref)
	if strings.HasPrefix(low, "javascript:") || strings.HasPrefix(low, "mailto:") || strings.HasPrefix(low, "tel:") {
		return ""
	}
	parsed, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	resolved := baseParsed.ResolveReference(parsed).String()
	if !strings.HasPrefix(resolved, "http://") && !strings.HasPrefix(resolved, "https://") {
		return ""
	}
	return resolved
}

func extractHTMLAnchorLinks(baseURL, htmlBody string) []htmlLink {
	baseParsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return nil
	}
	cleaned := htmlScriptStyleRegex.ReplaceAllString(htmlBody, " ")
	seen := make(map[string]struct{})
	var links []htmlLink

	for _, m := range htmlAnchorFullRegex.FindAllStringSubmatch(cleaned, -1) {
		if len(m) < 3 {
			continue
		}
		href := extractAttrValue(m[1], htmlAttrHrefRegex)
		resolved := resolveRelativeHTTPURL(baseParsed, href)
		if resolved == "" {
			continue
		}
		if _, exists := seen[resolved]; exists {
			continue
		}
		seen[resolved] = struct{}{}
		links = append(links, htmlLink{
			URL:   resolved,
			Label: cleanHTMLText(m[2]),
		})
	}

	for _, m := range htmlIframeRegex.FindAllStringSubmatch(cleaned, -1) {
		if len(m) < 2 {
			continue
		}
		src := extractAttrValue(m[1], htmlAttrSrcRegex)
		resolved := resolveRelativeHTTPURL(baseParsed, src)
		if resolved == "" {
			continue
		}
		if _, exists := seen[resolved]; exists {
			continue
		}
		seen[resolved] = struct{}{}
		links = append(links, htmlLink{
			URL:   resolved,
			Label: "iframe transparência",
		})
	}

	return links
}

func extractHTMLTables(baseURL, htmlBody string) []htmlTable {
	baseParsed, _ := url.Parse(strings.TrimSpace(baseURL))
	cleaned := htmlScriptStyleRegex.ReplaceAllString(htmlBody, " ")
	tableMatches := htmlTableRegex.FindAllStringSubmatch(cleaned, -1)
	if len(tableMatches) == 0 {
		return nil
	}

	var tables []htmlTable
	for _, tm := range tableMatches {
		if len(tm) < 2 {
			continue
		}
		trMatches := htmlTRRegex.FindAllStringSubmatch(tm[1], -1)
		if len(trMatches) < 2 {
			continue
		}

		var headers []string
		var rows [][]string
		var rowLinks [][]string
		headerFound := false

		for _, tr := range trMatches {
			if len(tr) < 2 {
				continue
			}
			cellMatches := htmlCellRegex.FindAllStringSubmatch(tr[1], -1)
			if len(cellMatches) == 0 {
				continue
			}

			cells := make([]string, 0, len(cellMatches))
			links := make([]string, 0, len(cellMatches))
			allTH := true
			for _, cm := range cellMatches {
				tag := strings.ToLower(cm[1])
				if tag != "th" {
					allTH = false
				}
				cells = append(cells, cleanHTMLText(cm[3]))
				cellLink := ""
				if baseParsed != nil {
					if am := htmlAnchorFullRegex.FindStringSubmatch(cm[3]); len(am) >= 2 {
						cellLink = resolveRelativeHTTPURL(baseParsed, extractAttrValue(am[1], htmlAttrHrefRegex))
					}
				}
				links = append(links, cellLink)
			}

			if !headerFound && allTH && len(cells) >= 2 {
				headers = cells
				headerFound = true
				continue
			}

			nonEmpty := false
			for _, c := range cells {
				if c != "" {
					nonEmpty = true
					break
				}
			}
			if !nonEmpty {
				continue
			}

			if !headerFound && len(cells) >= 2 {
				headers = cells
				headerFound = true
				continue
			}

			if len(cells) >= 2 {
				rows = append(rows, cells)
				rowLinks = append(rowLinks, links)
			}
		}

		if len(headers) >= 2 && len(rows) >= 1 {
			tables = append(tables, htmlTable{
				Headers: headers,
				Rows:    rows,
				Links:   rowLinks,
			})
		}
	}

	return tables
}

func fetchHTMLPage(ctx context.Context, client *http.Client, targetURL string) (finalURL string, body string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en;q=0.8")

	resp, err := portalConnectorHTTPClient(client).Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	finalURL = targetURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}

	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if readErr != nil {
		return finalURL, "", readErr
	}
	bodyStr := string(raw)
	bodyLow := strings.ToLower(bodyStr)

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusServiceUnavailable {
		for _, marker := range antiBotMarkers {
			if strings.Contains(bodyLow, marker) {
				return finalURL, "", fmt.Errorf("%w (HTTP %d)", errHTMLAntiBotBlocked, resp.StatusCode)
			}
		}
	}
	if resp.StatusCode >= 400 {
		return finalURL, "", fmt.Errorf("portal HTML retornou HTTP %d", resp.StatusCode)
	}
	return finalURL, bodyStr, nil
}

func htmlDomainKeywords(domain string) []string {
	switch domain {
	case "servidores":
		return []string{"servidor", "folha", "remuneracao", "pessoal", "vencimento", "salario", "funcional", "cargos"}
	case "despesas":
		return []string{"despesa", "empenho", "pagamento", "gasto", "liquidacao", "ordem cronologica", "credor"}
	case "receitas":
		return []string{"receita", "arrecadacao", "tributo", "imposto", "repasses"}
	case "licitacoes":
		return []string{"licitac", "pregao", "edital", "dispensa", "inexigibilidade", "chamamento", "tomada de preco"}
	case "contratos":
		return []string{"contrat", "aditivo", "ata de registro", "termo"}
	case "diarias":
		return []string{"diaria", "passagem", "viagem", "adiantamento"}
	default:
		return []string{"transparencia", "dados abertos"}
	}
}

func isBinaryDocumentURL(rawURL string) bool {
	low := strings.ToLower(rawURL)
	if idx := strings.Index(low, "?"); idx >= 0 {
		low = low[:idx]
	}
	for _, ext := range []string{".pdf", ".zip", ".rar", ".xls", ".xlsx", ".doc", ".docx", ".jpg", ".jpeg", ".png", ".gif", ".svg", ".css", ".js", ".mp3", ".mp4"} {
		if strings.HasSuffix(low, ext) {
			return true
		}
	}
	return false
}

func scoreHTMLLink(link htmlLink, baseHost string, keywords []string) int {
	if isBinaryDocumentURL(link.URL) {
		return 0
	}
	u, err := url.Parse(link.URL)
	if err != nil {
		return 0
	}
	haystack := RemoveAccents(link.Label + " " + u.Path + " " + u.RawQuery)
	score := 0
	for i, kw := range keywords {
		if strings.Contains(haystack, kw) {
			score += (len(keywords) - i) * 10
		}
	}
	if score == 0 {
		if strings.Contains(haystack, "transparencia") || strings.Contains(haystack, "portal-da-transparencia") {
			score += 5
		}
	}
	if score == 0 {
		return 0
	}
	if strings.EqualFold(u.Hostname(), baseHost) {
		score += 15
	}
	return score
}

func tableHeaderScore(headers []string, keywords []string) int {
	joined := RemoveAccents(strings.Join(headers, " "))
	score := 0
	for _, kw := range keywords {
		if strings.Contains(joined, kw) {
			score++
		}
	}
	return score
}

func tableDomainHeaderKeywords(domain string) []string {
	switch domain {
	case "servidores":
		return []string{"nome", "servidor", "cargo", "funcao", "salario", "remuneracao", "bruto", "liquido", "lotacao", "matricula", "cpf"}
	case "despesas":
		return []string{"empenho", "fornecedor", "credor", "favorecido", "valor", "empenhado", "pago", "liquidado", "historico", "elemento", "orgao"}
	case "receitas":
		return []string{"receita", "codigo", "rubrica", "descricao", "previsto", "orcado", "arrecadado", "realizado", "categoria"}
	case "licitacoes":
		return []string{"licitacao", "edital", "processo", "modalidade", "objeto", "situacao", "abertura", "valor", "estimado", "homologado"}
	case "contratos":
		return []string{"contrato", "fornecedor", "contratado", "objeto", "vigencia", "assinatura", "valor", "situacao"}
	case "diarias":
		return []string{"beneficiario", "servidor", "nome", "destino", "motivo", "objetivo", "valor", "data", "periodo"}
	default:
		return nil
	}
}

func fetchHTMLDomainTables(ctx context.Context, client *http.Client, baseURL, domain string) (usedURL string, topLinks []string, tables []htmlTable, err error) {
	finalURL, body, err := fetchHTMLPage(ctx, client, baseURL)
	if err != nil {
		return baseURL, nil, nil, err
	}
	usedURL = finalURL

	baseParsed, _ := url.Parse(finalURL)
	baseHost := ""
	if baseParsed != nil {
		baseHost = baseParsed.Hostname()
	}

	headerKWs := tableDomainHeaderKeywords(domain)
	rootTables := extractHTMLTables(finalURL, body)
	for _, tb := range rootTables {
		if tableHeaderScore(tb.Headers, headerKWs) >= 2 {
			tables = append(tables, tb)
		}
	}

	links := extractHTMLAnchorLinks(finalURL, body)
	kws := htmlDomainKeywords(domain)
	type scoredLink struct {
		link  htmlLink
		score int
	}
	var cands []scoredLink
	for _, lk := range links {
		if sc := scoreHTMLLink(lk, baseHost, kws); sc > 0 {
			cands = append(cands, scoredLink{link: lk, score: sc})
		}
	}
	sort.SliceStable(cands, func(i, j int) bool {
		return cands[i].score > cands[j].score
	})

	for i := 0; i < len(cands) && i < 5; i++ {
		topLinks = append(topLinks, cands[i].link.URL)
	}

	if len(tables) > 0 {
		return usedURL, topLinks, tables, nil
	}

	// Visita até 2 subpáginas mais relevantes quando a página inicial não contém a tabela do módulo
	visited := map[string]struct{}{finalURL: {}}
	fetched := 0
	for _, c := range cands {
		if fetched >= 2 {
			break
		}
		if _, ok := visited[c.link.URL]; ok {
			continue
		}
		visited[c.link.URL] = struct{}{}
		fetched++

		subFinal, subBody, subErr := fetchHTMLPage(ctx, client, c.link.URL)
		if subErr != nil {
			continue
		}
		subTables := extractHTMLTables(subFinal, subBody)
		matched := false
		for _, tb := range subTables {
			if tableHeaderScore(tb.Headers, headerKWs) >= 2 {
				tables = append(tables, tb)
				matched = true
			}
		}
		if matched {
			usedURL = subFinal
			break
		}
		if len(subTables) > 0 && len(tables) == 0 {
			tables = append(tables, subTables...)
			usedURL = subFinal
		}
	}

	if len(tables) == 0 {
		tables = rootTables
	}
	return usedURL, topLinks, tables, nil
}

func findColIdx(normHeaders []string, candidates ...string) int {
	for _, cand := range candidates {
		for idx, h := range normHeaders {
			if h == cand || strings.Contains(h, cand) {
				return idx
			}
		}
	}
	return -1
}

func cellAt(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func parseHTMLBRFloat(s string) float64 {
	cleaned := strings.TrimSpace(s)
	if cleaned == "" {
		return 0
	}
	cleaned = strings.TrimPrefix(cleaned, "R$")
	cleaned = strings.TrimPrefix(cleaned, "r$")
	cleaned = strings.TrimSpace(cleaned)
	v, err := parseBRFloat(cleaned)
	if err != nil {
		return 0
	}
	return v
}

func normalizeHTMLTableHeaders(headers []string) []string {
	out := make([]string, len(headers))
	for i, h := range headers {
		out[i] = RemoveAccents(h)
	}
	return out
}

func htmlNoRowsMessage(plat, usedURL string, topLinks []string) string {
	if len(topLinks) > 0 {
		return fmt.Sprintf(
			"Portal municipal HTML (%s) acessível em %s, mas sem tabela estática extraível neste formato (página interativa/JS). Links relacionados localizados: %s",
			htmlPlatformLabel(plat), usedURL, strings.Join(topLinks[:minInt(3, len(topLinks))], ", "),
		)
	}
	return fmt.Sprintf(
		"Portal municipal HTML (%s) acessível em %s, mas sem tabela HTML estática extraível para este módulo.",
		htmlPlatformLabel(plat), usedURL,
	)
}

// QueryHTMLPortalServidores consulta servidores públicos a partir de tabelas HTML em portais sem API estruturada.
func QueryHTMLPortalServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams, plataforma string) (*ServidoresResult, error) {
	usedURL, topLinks, tables, err := fetchHTMLDomainTables(ctx, client, p.BaseURL, "servidores")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fonte := fmt.Sprintf("Portal Municipal HTML (%s)", htmlPlatformLabel(plataforma))
	var items []ServidorItem

	for _, tb := range tables {
		nh := normalizeHTMLTableHeaders(tb.Headers)
		idxNome := findColIdx(nh, "nome", "servidor", "funcionario", "empregado")
		idxCargo := findColIdx(nh, "cargo", "funcao", "ocupacao")
		if idxNome < 0 && idxCargo < 0 {
			continue
		}
		idxMat := findColIdx(nh, "matricula", "registro", "codigo")
		idxCPF := findColIdx(nh, "cpf", "documento")
		idxFuncao := findColIdx(nh, "funcao")
		idxLot := findColIdx(nh, "lotacao", "secretaria", "departamento", "setor", "unidade", "orgao")
		idxVinc := findColIdx(nh, "vinculo", "regime", "natureza", "tipo")
		idxSit := findColIdx(nh, "situacao", "status")
		idxBase := findColIdx(nh, "salario base", "vencimento base", "base")
		idxBruto := findColIdx(nh, "bruto", "proventos", "remuneracao", "total bruto", "salario")
		idxDesc := findColIdx(nh, "desconto")
		idxLiq := findColIdx(nh, "liquido", "valor liquido")
		idxComp := findColIdx(nh, "competencia", "mes/ano", "referencia", "periodo")
		idxCarga := findColIdx(nh, "carga horaria", "horas")
		idxAdm := findColIdx(nh, "admissao", "data admissao")

		for _, row := range tb.Rows {
			nome := cellAt(row, idxNome)
			cargo := cellAt(row, idxCargo)
			if nome == "" && cargo == "" {
				continue
			}
			if filtro != "" && !strings.Contains(strings.ToLower(nome+" "+cargo), filtro) {
				continue
			}
			rawCPF := cellAt(row, idxCPF)
			items = append(items, ServidorItem{
				Nome:           nome,
				Matricula:      cellAt(row, idxMat),
				CPFParcial:     maskCPF("", rawCPF),
				Cargo:          cargo,
				Funcao:         cellAt(row, idxFuncao),
				Lotacao:        cellAt(row, idxLot),
				Vinculo:        cellAt(row, idxVinc),
				Situacao:       cellAt(row, idxSit),
				SalarioBase:    parseHTMLBRFloat(cellAt(row, idxBase)),
				SalarioBruto:   parseHTMLBRFloat(cellAt(row, idxBruto)),
				Descontos:      parseHTMLBRFloat(cellAt(row, idxDesc)),
				SalarioLiquido: parseHTMLBRFloat(cellAt(row, idxLiq)),
				Competencia:    cellAt(row, idxComp),
				CargaHoraria:   cellAt(row, idxCarga),
				DataAdmissao:   cellAt(row, idxAdm),
				Fonte:          fonte,
				PortalURL:      usedURL,
			})
			if len(items) >= limit {
				break
			}
		}
		if len(items) >= limit {
			break
		}
	}

	if len(items) == 0 {
		return &ServidoresResult{
			Municipio:  m,
			Fonte:      fonte,
			Plataforma: plataforma,
			PortalURL:  usedURL,
			Servidores: []ServidorItem{},
			Disponivel: false,
			Mensagem:   htmlNoRowsMessage(plataforma, usedURL, topLinks),
		}, errHTMLNoTableRows
	}

	return &ServidoresResult{
		Municipio:      m,
		Fonte:          fonte,
		Plataforma:     plataforma,
		PortalURL:      usedURL,
		TotalRegistros: len(items),
		Servidores:     items,
		Disponivel:     true,
	}, nil
}

// QueryHTMLPortalDespesas consulta despesas/empenhos a partir de tabelas HTML em portais sem API estruturada.
func QueryHTMLPortalDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams, plataforma string) (*DespesasResult, error) {
	usedURL, topLinks, tables, err := fetchHTMLDomainTables(ctx, client, p.BaseURL, "despesas")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fonte := fmt.Sprintf("Portal Municipal HTML (%s)", htmlPlatformLabel(plataforma))
	var items []DespesaItem

	for _, tb := range tables {
		nh := normalizeHTMLTableHeaders(tb.Headers)
		idxForn := findColIdx(nh, "fornecedor", "credor", "favorecido", "razao social", "nome")
		idxEmp := findColIdx(nh, "empenho", "numero", "processo")
		idxVal := findColIdx(nh, "empenhado", "valor empenhado", "valor", "total")
		if idxForn < 0 && idxEmp < 0 && idxVal < 0 {
			continue
		}
		idxData := findColIdx(nh, "data")
		idxDoc := findColIdx(nh, "cpf/cnpj", "cnpj/cpf", "cpf", "cnpj", "documento")
		idxOrg := findColIdx(nh, "orgao", "entidade")
		idxUni := findColIdx(nh, "unidade")
		idxFunc := findColIdx(nh, "funcao")
		idxElem := findColIdx(nh, "elemento", "natureza", "classificacao")
		idxHist := findColIdx(nh, "historico", "descricao", "objeto", "especificacao")
		idxLiq := findColIdx(nh, "liquidado", "valor liquidado")
		idxPago := findColIdx(nh, "pago", "valor pago")

		for _, row := range tb.Rows {
			forn := cellAt(row, idxForn)
			hist := cellAt(row, idxHist)
			numEmp := cellAt(row, idxEmp)
			if forn == "" && hist == "" && numEmp == "" {
				continue
			}
			if filtro != "" && !strings.Contains(strings.ToLower(forn+" "+hist+" "+numEmp), filtro) {
				continue
			}
			items = append(items, DespesaItem{
				NumeroEmpenho:       numEmp,
				Ano:                 p.Ano,
				Data:                cellAt(row, idxData),
				Fornecedor:          forn,
				DocumentoFornecedor: maskIfCPF(cellAt(row, idxDoc)),
				Orgao:               cellAt(row, idxOrg),
				Unidade:             cellAt(row, idxUni),
				Funcao:              cellAt(row, idxFunc),
				Elemento:            cellAt(row, idxElem),
				Historico:           hist,
				ValorEmpenhado:      parseHTMLBRFloat(cellAt(row, idxVal)),
				ValorLiquidado:      parseHTMLBRFloat(cellAt(row, idxLiq)),
				ValorPago:           parseHTMLBRFloat(cellAt(row, idxPago)),
				Fonte:               fonte,
			})
			if len(items) >= limit {
				break
			}
		}
		if len(items) >= limit {
			break
		}
	}

	if len(items) == 0 {
		return &DespesasResult{
			Municipio:  m,
			Fonte:      fonte,
			Plataforma: plataforma,
			PortalURL:  usedURL,
			Despesas:   []DespesaItem{},
			Disponivel: false,
			Mensagem:   htmlNoRowsMessage(plataforma, usedURL, topLinks),
		}, errHTMLNoTableRows
	}

	return &DespesasResult{
		Municipio:      m,
		Fonte:          fonte,
		Plataforma:     plataforma,
		PortalURL:      usedURL,
		TotalRegistros: len(items),
		Despesas:       items,
		Disponivel:     true,
	}, nil
}

// QueryHTMLPortalReceitas consulta receitas a partir de tabelas HTML em portais sem API estruturada.
func QueryHTMLPortalReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams, plataforma string) (*ReceitasResult, error) {
	usedURL, topLinks, tables, err := fetchHTMLDomainTables(ctx, client, p.BaseURL, "receitas")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fonte := fmt.Sprintf("Portal Municipal HTML (%s)", htmlPlatformLabel(plataforma))
	var items []ReceitaItem

	for _, tb := range tables {
		nh := normalizeHTMLTableHeaders(tb.Headers)
		idxDesc := findColIdx(nh, "descricao", "especificacao", "receita", "nome", "titulo")
		idxPrev := findColIdx(nh, "previsto", "orcado", "previsao")
		idxArr := findColIdx(nh, "arrecadado", "realizado", "valor")
		if idxDesc < 0 && idxPrev < 0 && idxArr < 0 {
			continue
		}
		idxCod := findColIdx(nh, "codigo", "rubrica", "classificacao", "conta")
		idxCat := findColIdx(nh, "categoria", "fonte", "origem", "tipo")
		idxData := findColIdx(nh, "data", "competencia", "periodo")

		for _, row := range tb.Rows {
			desc := cellAt(row, idxDesc)
			cod := cellAt(row, idxCod)
			if desc == "" && cod == "" {
				continue
			}
			if filtro != "" && !strings.Contains(strings.ToLower(desc+" "+cod), filtro) {
				continue
			}
			items = append(items, ReceitaItem{
				Codigo:          cod,
				Descricao:       desc,
				Categoria:       cellAt(row, idxCat),
				Data:            cellAt(row, idxData),
				ValorPrevisto:   parseHTMLBRFloat(cellAt(row, idxPrev)),
				ValorArrecadado: parseHTMLBRFloat(cellAt(row, idxArr)),
				Fonte:           fonte,
			})
			if len(items) >= limit {
				break
			}
		}
		if len(items) >= limit {
			break
		}
	}

	if len(items) == 0 {
		return &ReceitasResult{
			Municipio:  m,
			Fonte:      fonte,
			Plataforma: plataforma,
			PortalURL:  usedURL,
			Receitas:   []ReceitaItem{},
			Disponivel: false,
			Mensagem:   htmlNoRowsMessage(plataforma, usedURL, topLinks),
		}, errHTMLNoTableRows
	}

	return &ReceitasResult{
		Municipio:      m,
		Fonte:          fonte,
		Plataforma:     plataforma,
		PortalURL:      usedURL,
		TotalRegistros: len(items),
		Receitas:       items,
		Disponivel:     true,
	}, nil
}

// QueryHTMLPortalLicitacoes consulta licitações a partir de tabelas HTML em portais sem API estruturada.
func QueryHTMLPortalLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams, plataforma string) (*LicitacoesResult, error) {
	usedURL, topLinks, tables, err := fetchHTMLDomainTables(ctx, client, p.BaseURL, "licitacoes")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fonte := fmt.Sprintf("Portal Municipal HTML (%s)", htmlPlatformLabel(plataforma))
	var items []LicitacaoItem

	for _, tb := range tables {
		nh := normalizeHTMLTableHeaders(tb.Headers)
		idxObj := findColIdx(nh, "objeto", "descricao", "finalidade")
		idxNum := findColIdx(nh, "numero", "edital", "processo", "licitacao")
		idxMod := findColIdx(nh, "modalidade", "tipo")
		if idxObj < 0 && idxNum < 0 && idxMod < 0 {
			continue
		}
		idxSit := findColIdx(nh, "situacao", "status", "fase")
		idxData := findColIdx(nh, "abertura", "data", "publicacao")
		idxEst := findColIdx(nh, "estimado", "previsto", "valor")
		idxHom := findColIdx(nh, "homologado", "adjudicado", "vencedor")

		for rIdx, row := range tb.Rows {
			obj := cellAt(row, idxObj)
			num := cellAt(row, idxNum)
			mod := cellAt(row, idxMod)
			if obj == "" && num == "" {
				continue
			}
			if filtro != "" && !strings.Contains(strings.ToLower(obj+" "+num+" "+mod), filtro) {
				continue
			}
			link := ""
			if rIdx < len(tb.Links) {
				for _, l := range tb.Links[rIdx] {
					if l != "" {
						link = l
						break
					}
				}
			}
			items = append(items, LicitacaoItem{
				Numero:          num,
				Ano:             p.Ano,
				Modalidade:      mod,
				Situacao:        cellAt(row, idxSit),
				DataAbertura:    cellAt(row, idxData),
				Objeto:          obj,
				ValorEstimado:   parseHTMLBRFloat(cellAt(row, idxEst)),
				ValorHomologado: parseHTMLBRFloat(cellAt(row, idxHom)),
				Link:            link,
				Fonte:           fonte,
			})
			if len(items) >= limit {
				break
			}
		}
		if len(items) >= limit {
			break
		}
	}

	if len(items) == 0 {
		return &LicitacoesResult{
			Municipio:  m,
			Fonte:      fonte,
			Plataforma: plataforma,
			PortalURL:  usedURL,
			Licitacoes: []LicitacaoItem{},
			Disponivel: false,
			Mensagem:   htmlNoRowsMessage(plataforma, usedURL, topLinks),
		}, errHTMLNoTableRows
	}

	return &LicitacoesResult{
		Municipio:      m,
		Fonte:          fonte,
		Plataforma:     plataforma,
		PortalURL:      usedURL,
		TotalRegistros: len(items),
		Licitacoes:     items,
		Disponivel:     true,
	}, nil
}

// QueryHTMLPortalContratos consulta contratos a partir de tabelas HTML em portais sem API estruturada.
func QueryHTMLPortalContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams, plataforma string) (*ContratosResult, error) {
	usedURL, topLinks, tables, err := fetchHTMLDomainTables(ctx, client, p.BaseURL, "contratos")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fonte := fmt.Sprintf("Portal Municipal HTML (%s)", htmlPlatformLabel(plataforma))
	var items []ContratoItem

	for _, tb := range tables {
		nh := normalizeHTMLTableHeaders(tb.Headers)
		idxForn := findColIdx(nh, "fornecedor", "contratado", "credor", "razao social", "empresa")
		idxNum := findColIdx(nh, "contrato", "numero")
		idxObj := findColIdx(nh, "objeto", "descricao")
		if idxForn < 0 && idxNum < 0 && idxObj < 0 {
			continue
		}
		idxDoc := findColIdx(nh, "cpf/cnpj", "cnpj/cpf", "cpf", "cnpj", "documento")
		idxSit := findColIdx(nh, "situacao", "status")
		idxAss := findColIdx(nh, "assinatura", "data")
		idxVigIni := findColIdx(nh, "inicio", "vigencia inicio")
		idxVigFim := findColIdx(nh, "fim", "termino", "vencimento", "vigencia")
		idxVal := findColIdx(nh, "valor", "global", "total")

		for _, row := range tb.Rows {
			forn := cellAt(row, idxForn)
			num := cellAt(row, idxNum)
			obj := cellAt(row, idxObj)
			if forn == "" && num == "" && obj == "" {
				continue
			}
			if filtro != "" && !strings.Contains(strings.ToLower(forn+" "+num+" "+obj), filtro) {
				continue
			}
			items = append(items, ContratoItem{
				Numero:              num,
				Ano:                 p.Ano,
				Fornecedor:          forn,
				DocumentoFornecedor: maskIfCPF(cellAt(row, idxDoc)),
				Objeto:              obj,
				Situacao:            cellAt(row, idxSit),
				DataAssinatura:      cellAt(row, idxAss),
				VigenciaInicio:      cellAt(row, idxVigIni),
				VigenciaFim:         cellAt(row, idxVigFim),
				Valor:               parseHTMLBRFloat(cellAt(row, idxVal)),
				Fonte:               fonte,
			})
			if len(items) >= limit {
				break
			}
		}
		if len(items) >= limit {
			break
		}
	}

	if len(items) == 0 {
		return &ContratosResult{
			Municipio:  m,
			Fonte:      fonte,
			Plataforma: plataforma,
			PortalURL:  usedURL,
			Contratos:  []ContratoItem{},
			Disponivel: false,
			Mensagem:   htmlNoRowsMessage(plataforma, usedURL, topLinks),
		}, errHTMLNoTableRows
	}

	return &ContratosResult{
		Municipio:      m,
		Fonte:          fonte,
		Plataforma:     plataforma,
		PortalURL:      usedURL,
		TotalRegistros: len(items),
		Contratos:      items,
		Disponivel:     true,
	}, nil
}

// QueryHTMLPortalDiarias consulta diárias e passagens a partir de tabelas HTML em portais sem API estruturada.
func QueryHTMLPortalDiarias(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams, plataforma string) (*DiariasResult, error) {
	usedURL, topLinks, tables, err := fetchHTMLDomainTables(ctx, client, p.BaseURL, "diarias")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fonte := fmt.Sprintf("Portal Municipal HTML (%s)", htmlPlatformLabel(plataforma))
	var items []DiariaItem

	for _, tb := range tables {
		nh := normalizeHTMLTableHeaders(tb.Headers)
		idxBen := findColIdx(nh, "beneficiario", "servidor", "nome", "favorecido")
		idxDest := findColIdx(nh, "destino", "local", "cidade")
		idxVal := findColIdx(nh, "valor", "total", "pago")
		if idxBen < 0 && idxDest < 0 && idxVal < 0 {
			continue
		}
		idxCargo := findColIdx(nh, "cargo", "funcao")
		idxMot := findColIdx(nh, "motivo", "objetivo", "finalidade", "historico", "descricao")
		idxPer := findColIdx(nh, "periodo", "data", "saida")

		for _, row := range tb.Rows {
			ben := cellAt(row, idxBen)
			dest := cellAt(row, idxDest)
			mot := cellAt(row, idxMot)
			if ben == "" && dest == "" && mot == "" {
				continue
			}
			if filtro != "" && !strings.Contains(strings.ToLower(ben+" "+dest+" "+mot), filtro) {
				continue
			}
			items = append(items, DiariaItem{
				Beneficiario: ben,
				Cargo:        cellAt(row, idxCargo),
				Destino:      dest,
				Motivo:       mot,
				Data:         cellAt(row, idxPer),
				Valor:        parseHTMLBRFloat(cellAt(row, idxVal)),
				Fonte:        fonte,
			})
			if len(items) >= limit {
				break
			}
		}
		if len(items) >= limit {
			break
		}
	}

	if len(items) == 0 {
		return &DiariasResult{
			Municipio:  m,
			Fonte:      fonte,
			Plataforma: plataforma,
			PortalURL:  usedURL,
			Diarias:    []DiariaItem{},
			Disponivel: false,
			Mensagem:   htmlNoRowsMessage(plataforma, usedURL, topLinks),
		}, errHTMLNoTableRows
	}

	return &DiariasResult{
		Municipio:      m,
		Fonte:          fonte,
		Plataforma:     plataforma,
		PortalURL:      usedURL,
		TotalRegistros: len(items),
		Diarias:        items,
		Disponivel:     true,
	}, nil
}
