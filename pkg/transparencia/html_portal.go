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
	"sync"
	"time"
	"unicode/utf8"
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
	case "fiorilli_web", "el_transparencia", "betha", "sonner", "elmartecnologia", "sagres_tce_pb":
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

type htmlPageCacheEntry struct {
	done      chan struct{}
	finalURL  string
	body      string
	err       error
	expiresAt time.Time
}

var (
	htmlPageCacheMu sync.Mutex
	htmlPageCache   = make(map[string]*htmlPageCacheEntry)
)

var win1252Table = [32]rune{
	0x20AC, 0x0081, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021,
	0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0x008D, 0x017D, 0x008F,
	0x0090, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014,
	0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0x009D, 0x017E, 0x0178,
}

// decodeWebBody converte respostas legadas ISO-8859-1 / Windows-1252 para UTF-8 válido.
func decodeWebBody(raw []byte, contentType string) string {
	if len(raw) == 0 {
		return ""
	}
	// Remove UTF-8 BOM se presente
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		raw = raw[3:]
	}
	ctLow := strings.ToLower(contentType)
	explicitLatin := strings.Contains(ctLow, "iso-8859-1") ||
		strings.Contains(ctLow, "latin1") ||
		strings.Contains(ctLow, "latin-1") ||
		strings.Contains(ctLow, "windows-1252") ||
		strings.Contains(ctLow, "cp1252")

	if !explicitLatin && utf8.Valid(raw) {
		return string(raw)
	}
	var b strings.Builder
	b.Grow(len(raw) + len(raw)/8)
	for _, c := range raw {
		switch {
		case c < 0x80:
			b.WriteByte(c)
		case c >= 0x80 && c <= 0x9F:
			b.WriteRune(win1252Table[c-0x80])
		default:
			b.WriteRune(rune(c))
		}
	}
	return b.String()
}

func fetchHTMLPageUncached(ctx context.Context, client *http.Client, targetURL string) (finalURL string, body string, err error) {
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
	bodyStr := decodeWebBody(raw, resp.Header.Get("Content-Type"))
	bodyLow := strings.ToLower(bodyStr)

	if isAntiBotResponse(resp.StatusCode, bodyLow) {
		return finalURL, "", fmt.Errorf("%w (HTTP %d)", errHTMLAntiBotBlocked, resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return finalURL, "", fmt.Errorf("portal HTML retornou HTTP %d", resp.StatusCode)
	}
	return finalURL, bodyStr, nil
}

func fetchHTMLPage(ctx context.Context, client *http.Client, targetURL string) (finalURL string, body string, err error) {
	if strings.Contains(targetURL, "127.0.0.1") || strings.Contains(targetURL, "localhost") {
		return fetchHTMLPageUncached(ctx, client, targetURL)
	}

	now := time.Now()
	htmlPageCacheMu.Lock()
	if ent, ok := htmlPageCache[targetURL]; ok {
		if now.Before(ent.expiresAt) {
			done := ent.done
			htmlPageCacheMu.Unlock()
			select {
			case <-done:
				return ent.finalURL, ent.body, ent.err
			case <-ctx.Done():
				return "", "", ctx.Err()
			}
		}
		delete(htmlPageCache, targetURL)
	}
	ent := &htmlPageCacheEntry{
		done:      make(chan struct{}),
		expiresAt: now.Add(12 * time.Second),
	}
	htmlPageCache[targetURL] = ent
	htmlPageCacheMu.Unlock()

	fURL, bStr, fErr := fetchHTMLPageUncached(ctx, client, targetURL)
	ent.finalURL = fURL
	ent.body = bStr
	ent.err = fErr
	if fErr != nil {
		htmlPageCacheMu.Lock()
		delete(htmlPageCache, targetURL)
		htmlPageCacheMu.Unlock()
	}
	close(ent.done)
	return fURL, bStr, fErr
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
		return []string{"licita", "pregao", "edital", "dispensa", "inexigibilidade", "chamamento", "tomada de preco"}
	case "contratos":
		return []string{"contrat", "aditivo", "ata de registro", "termo"}
	case "diarias":
		return []string{"diaria", "passagem", "viagem", "adiantamento"}
	case "obras":
		return []string{"obra", "engenharia", "empreendimento", "execucao de obra", "reforma", "pavimentacao"}
	case "convenios":
		return []string{"convenio", "transferencia", "parceria", "acordo", "termo de fomento", "termo de colaboracao"}
	case "legislacao":
		return []string{"legislacao", "leis", "decreto", "portaria", "normativo", "ato oficial", "lei municipal"}
	case "concursos":
		return []string{"concurso", "processo seletivo", "edital de selecao", "aprovado", "convocacao"}
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

var allModuleStemKeywords = []string{
	"servidor", "folha", "remuneracao",
	"despesa", "empenho",
	"receita", "arrecadacao",
	"licita", "pregao",
	"contrat",
	"diaria", "passagem",
	"obra",
	"convenio",
	"legislacao", "decreto", "portaria",
	"concurso", "seletivo",
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
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(link.Label)), "iframe") {
		if score > 0 {
			score += 25
		} else {
			conflictsOtherModule := false
			for _, stem := range allModuleStemKeywords {
				if strings.Contains(haystack, stem) {
					conflictsOtherModule = true
					break
				}
			}
			if !conflictsOtherModule {
				score += 25
			}
		}
	}
	if score == 0 {
		if strings.Contains(haystack, "transparencia") ||
			strings.Contains(haystack, "portal-da-transparencia") ||
			strings.Contains(haystack, "acessoainformacao") ||
			strings.Contains(haystack, "acesso-a-informacao") {
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
	case "obras":
		return []string{"obra", "titulo", "objeto", "descricao", "local", "endereco", "bairro", "situacao", "percentual", "andamento", "valor", "contratada"}
	case "convenios":
		return []string{"convenio", "numero", "concedente", "convenente", "objeto", "descricao", "situacao", "vigencia", "valor"}
	case "legislacao":
		return []string{"tipo", "numero", "ano", "lei", "decreto", "portaria", "ementa", "descricao", "data", "publicacao"}
	case "concursos":
		return []string{"concurso", "edital", "cargo", "nome", "candidato", "inscricao", "classificacao", "situacao", "cpf"}
	default:
		return nil
	}
}

func fetchHTMLDomainTables(ctx context.Context, client *http.Client, baseURL, domain string) (usedURL string, topLinks []string, tables []htmlTable, err error) {
	return fetchHTMLDomainTablesWithParams(ctx, client, PortalQueryParams{BaseURL: baseURL}, domain)
}

func scorePageLinks(baseURL, body, domain string) []struct {
	link  htmlLink
	score int
} {
	baseParsed, _ := url.Parse(baseURL)
	baseHost := ""
	if baseParsed != nil {
		baseHost = baseParsed.Hostname()
	}
	links := extractHTMLAnchorLinks(baseURL, body)
	kws := htmlDomainKeywords(domain)
	var cands []struct {
		link  htmlLink
		score int
	}
	for _, lk := range links {
		if sc := scoreHTMLLink(lk, baseHost, kws); sc > 0 {
			cands = append(cands, struct {
				link  htmlLink
				score int
			}{link: lk, score: sc})
		}
	}
	sort.SliceStable(cands, func(i, j int) bool {
		return cands[i].score > cands[j].score
	})
	return cands
}

func fetchHTMLDomainTablesWithParams(ctx context.Context, client *http.Client, p PortalQueryParams, domain string) (usedURL string, topLinks []string, tables []htmlTable, err error) {
	sessClient := sessionHTTPClient(client)
	finalURL, body, err := fetchHTMLPage(ctx, sessClient, p.BaseURL)
	if err != nil {
		return p.BaseURL, nil, nil, err
	}
	usedURL = finalURL

	headerKWs := tableDomainHeaderKeywords(domain)
	rootUsedURL, rootMatched, rootFallback := extractPageAllTables(ctx, sessClient, finalURL, body, domain, headerKWs, p)

	cands := scorePageLinks(finalURL, body, domain)
	topLinkSeen := make(map[string]struct{})
	addTopLink := func(u string, priority bool) {
		if u == "" {
			return
		}
		if _, ok := topLinkSeen[u]; ok {
			return
		}
		if !priority && len(topLinks) >= 8 {
			return
		}
		topLinkSeen[u] = struct{}{}
		if priority {
			topLinks = append([]string{u}, topLinks...)
		} else {
			topLinks = append(topLinks, u)
		}
	}
	for i := 0; i < len(cands) && i < 4; i++ {
		addTopLink(cands[i].link.URL, strings.HasPrefix(strings.ToLower(cands[i].link.Label), "iframe") || publicsoftLegadoRegex.MatchString(cands[i].link.URL))
	}

	if len(rootMatched) > 0 {
		return rootUsedURL, topLinks, rootMatched, nil
	}

	// Visita até 2 subpáginas de nível 1 (ex.: Home -> /transparencia ou direto ao módulo)
	// e coleta candidatas de nível 2 caso o nível 1 seja um hub de links.
	visited := map[string]struct{}{finalURL: {}}
	type scoredCand struct {
		link  htmlLink
		score int
	}
	var level2Cands []scoredCand
	var fallbackTables []htmlTable
	fallbackUsedURL := usedURL

	fetchedL1 := 0
	for _, c := range cands {
		if fetchedL1 >= 2 {
			break
		}
		if _, ok := visited[c.link.URL]; ok {
			continue
		}
		visited[c.link.URL] = struct{}{}
		fetchedL1++

		subFinal, subBody, subErr := fetchHTMLPage(ctx, sessClient, c.link.URL)
		if subErr != nil {
			continue
		}
		visited[subFinal] = struct{}{}
		if fetchedL1 == 1 {
			usedURL = subFinal
		}

		subUsedURL, subMatched, subFallback := extractPageAllTables(ctx, sessClient, subFinal, subBody, domain, headerKWs, p)
		if len(subMatched) > 0 {
			return subUsedURL, topLinks, subMatched, nil
		}
		if len(subFallback) > 0 && len(fallbackTables) == 0 {
			fallbackTables = append(fallbackTables, subFallback...)
			fallbackUsedURL = subUsedURL
		}

		subLinks := scorePageLinks(subFinal, subBody, domain)
		for i, sl := range subLinks {
			isPrio := strings.HasPrefix(strings.ToLower(sl.link.Label), "iframe") || publicsoftLegadoRegex.MatchString(sl.link.URL)
			if i < 3 || isPrio {
				addTopLink(sl.link.URL, isPrio)
			}
			if publicsoftLegadoRegex.MatchString(sl.link.URL) {
				continue
			}
			// Apenas segue no salto 2 links que casam palavras-chave do domínio ou iframes (score > 20),
			// evitando loops em links genéricos "transparência" (+5/+15).
			if sl.score > 20 {
				if _, seen := visited[sl.link.URL]; !seen {
					level2Cands = append(level2Cands, scoredCand{link: sl.link, score: sl.score})
				}
			}
		}
	}

	// Salto 2: Home -> Hub Transparência -> Página específica do módulo / iframe
	sort.SliceStable(level2Cands, func(i, j int) bool {
		return level2Cands[i].score > level2Cands[j].score
	})
	fetchedL2 := 0
	for _, c2 := range level2Cands {
		if fetchedL2 >= 2 {
			break
		}
		if _, ok := visited[c2.link.URL]; ok {
			continue
		}
		visited[c2.link.URL] = struct{}{}
		fetchedL2++

		l2Final, l2Body, l2Err := fetchHTMLPage(ctx, sessClient, c2.link.URL)
		if l2Err != nil {
			continue
		}
		usedURL = l2Final
		l2UsedURL, l2Matched, l2Fallback := extractPageAllTables(ctx, sessClient, l2Final, l2Body, domain, headerKWs, p)
		if len(l2Matched) > 0 {
			return l2UsedURL, topLinks, l2Matched, nil
		}
		if len(l2Fallback) > 0 && len(fallbackTables) == 0 {
			fallbackTables = append(fallbackTables, l2Fallback...)
			fallbackUsedURL = l2UsedURL
		}
	}

	if len(fallbackTables) > 0 {
		return fallbackUsedURL, topLinks, fallbackTables, nil
	}
	if len(rootFallback) > 0 {
		return rootUsedURL, topLinks, rootFallback, nil
	}
	return usedURL, topLinks, nil, nil
}

func findEmbeddedPublicsoftURL(usedURL string, topLinks []string) string {
	if publicsoftLegadoRegex.MatchString(usedURL) {
		return usedURL
	}
	for _, lk := range topLinks {
		if publicsoftLegadoRegex.MatchString(lk) {
			return lk
		}
	}
	return ""
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
	usedURL, topLinks, tables, err := fetchHTMLDomainTablesWithParams(ctx, client, p, "servidores")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fonte := fmt.Sprintf("Portal Municipal HTML (%s)", htmlPlatformLabel(plataforma))
	var items []ServidorItem
	matchedTotal := 0

	for _, tb := range tables {
		nh := normalizeHTMLTableHeaders(tb.Headers)
		idxNome := findColIdx(nh, "nome", "servidor", "funcionario", "empregado")
		idxCargo := findColIdx(nh, "cargo", "funcao", "ocupacao")
		if idxNome < 0 && idxCargo < 0 {
			continue
		}
		idxMat := findColIdx(nh, "matricula", "registro")
		idxCPF := findColIdx(nh, "cpf", "documento")
		idxFuncao := findColIdx(nh, "funcao")
		idxLot := findColIdx(nh, "lotacao", "secretaria", "departamento", "setor", "unidade", "orgao")
		idxVinc := findColIdx(nh, "vinculo", "regime", "natureza", "tipo")
		idxSit := findColIdx(nh, "situacao", "status")
		idxBase := findColIdx(nh, "salario base", "vencimento base", "base")
		idxBruto := findColIdx(nh, "bruto", "proventos", "remuneracao", "total bruto", "salario", "vencimento")
		idxDesc := findColIdx(nh, "desconto")
		idxLiq := findColIdx(nh, "liquido", "valor liquido")
		idxComp := findColIdx(nh, "competencia", "mes/ano", "referencia", "periodo")
		idxCarga := findColIdx(nh, "carga horaria", "horas")
		idxAdm := findColIdx(nh, "admissao", "data admissao")
		idxServExplicit := findColIdx(nh, "servidor", "funcionario", "empregado")
		if idxCargo < 0 && idxBruto < 0 && idxLiq < 0 && idxBase < 0 && idxCPF < 0 && idxMat < 0 && idxServExplicit < 0 {
			continue
		}

		for _, row := range tb.Rows {
			nome := cellAt(row, idxNome)
			cargo := cellAt(row, idxCargo)
			if nome == "" && cargo == "" {
				continue
			}
			if filtro != "" && !strings.Contains(strings.ToLower(nome+" "+cargo), filtro) {
				continue
			}
			matchedTotal++
			if len(items) >= limit {
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
		}
	}

	if len(items) == 0 {
		if psURL := findEmbeddedPublicsoftURL(usedURL, topLinks); psURL != "" {
			psParams := p
			psParams.BaseURL = psURL
			psParams.TenantID = ""
			if psRes, psErr := QueryPublicsoftServidores(ctx, client, m, psParams); psErr == nil && psRes != nil && psRes.Disponivel {
				return psRes, nil
			}
		}
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
		TotalRegistros: matchedTotal,
		Servidores:     items,
		Disponivel:     true,
	}, nil
}

// QueryHTMLPortalDespesas consulta despesas/empenhos a partir de tabelas HTML em portais sem API estruturada.
func QueryHTMLPortalDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams, plataforma string) (*DespesasResult, error) {
	usedURL, topLinks, tables, err := fetchHTMLDomainTablesWithParams(ctx, client, p, "despesas")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fonte := fmt.Sprintf("Portal Municipal HTML (%s)", htmlPlatformLabel(plataforma))
	var items []DespesaItem
	matchedTotal := 0

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
			matchedTotal++
			if len(items) >= limit {
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
		}
	}

	if len(items) == 0 {
		if psURL := findEmbeddedPublicsoftURL(usedURL, topLinks); psURL != "" {
			psParams := p
			psParams.BaseURL = psURL
			psParams.TenantID = ""
			if psRes, psErr := QueryPublicsoftDespesas(ctx, client, m, psParams); psErr == nil && psRes != nil && psRes.Disponivel {
				return psRes, nil
			}
		}
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
		TotalRegistros: matchedTotal,
		Despesas:       items,
		Disponivel:     true,
	}, nil
}

// QueryHTMLPortalReceitas consulta receitas a partir de tabelas HTML em portais sem API estruturada.
func QueryHTMLPortalReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams, plataforma string) (*ReceitasResult, error) {
	usedURL, topLinks, tables, err := fetchHTMLDomainTablesWithParams(ctx, client, p, "receitas")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fonte := fmt.Sprintf("Portal Municipal HTML (%s)", htmlPlatformLabel(plataforma))
	var items []ReceitaItem
	matchedTotal := 0

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
			matchedTotal++
			if len(items) >= limit {
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
		}
	}

	if len(items) == 0 {
		if psURL := findEmbeddedPublicsoftURL(usedURL, topLinks); psURL != "" {
			psParams := p
			psParams.BaseURL = psURL
			psParams.TenantID = ""
			if psRes, psErr := QueryPublicsoftReceitas(ctx, client, m, psParams); psErr == nil && psRes != nil && psRes.Disponivel {
				return psRes, nil
			}
		}
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
		TotalRegistros: matchedTotal,
		Receitas:       items,
		Disponivel:     true,
	}, nil
}

// QueryHTMLPortalLicitacoes consulta licitações a partir de tabelas HTML em portais sem API estruturada.
func QueryHTMLPortalLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams, plataforma string) (*LicitacoesResult, error) {
	usedURL, topLinks, tables, err := fetchHTMLDomainTablesWithParams(ctx, client, p, "licitacoes")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fonte := fmt.Sprintf("Portal Municipal HTML (%s)", htmlPlatformLabel(plataforma))
	var items []LicitacaoItem
	matchedTotal := 0

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
			matchedTotal++
			if len(items) >= limit {
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
		}
	}

	if len(items) == 0 {
		if psURL := findEmbeddedPublicsoftURL(usedURL, topLinks); psURL != "" {
			psParams := p
			psParams.BaseURL = psURL
			psParams.TenantID = ""
			if psRes, psErr := QueryPublicsoftLicitacoes(ctx, client, m, psParams); psErr == nil && psRes != nil && psRes.Disponivel {
				return psRes, nil
			}
		}
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
		TotalRegistros: matchedTotal,
		Licitacoes:     items,
		Disponivel:     true,
	}, nil
}

// QueryHTMLPortalContratos consulta contratos a partir de tabelas HTML em portais sem API estruturada.
func QueryHTMLPortalContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams, plataforma string) (*ContratosResult, error) {
	usedURL, topLinks, tables, err := fetchHTMLDomainTablesWithParams(ctx, client, p, "contratos")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fonte := fmt.Sprintf("Portal Municipal HTML (%s)", htmlPlatformLabel(plataforma))
	var items []ContratoItem
	matchedTotal := 0

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
			matchedTotal++
			if len(items) >= limit {
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
		}
	}

	if len(items) == 0 {
		if psURL := findEmbeddedPublicsoftURL(usedURL, topLinks); psURL != "" {
			psParams := p
			psParams.BaseURL = psURL
			psParams.TenantID = ""
			if psRes, psErr := QueryPublicsoftContratos(ctx, client, m, psParams); psErr == nil && psRes != nil && psRes.Disponivel {
				return psRes, nil
			}
		}
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
		TotalRegistros: matchedTotal,
		Contratos:      items,
		Disponivel:     true,
	}, nil
}

// QueryHTMLPortalDiarias consulta diárias e passagens a partir de tabelas HTML em portais sem API estruturada.
func QueryHTMLPortalDiarias(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams, plataforma string) (*DiariasResult, error) {
	usedURL, topLinks, tables, err := fetchHTMLDomainTablesWithParams(ctx, client, p, "diarias")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fonte := fmt.Sprintf("Portal Municipal HTML (%s)", htmlPlatformLabel(plataforma))
	var items []DiariaItem
	matchedTotal := 0

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
			matchedTotal++
			if len(items) >= limit {
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
		}
	}

	if len(items) == 0 {
		if psURL := findEmbeddedPublicsoftURL(usedURL, topLinks); psURL != "" {
			psParams := p
			psParams.BaseURL = psURL
			psParams.TenantID = ""
			if psRes, psErr := QueryPublicsoftDiarias(ctx, client, m, psParams); psErr == nil && psRes != nil && psRes.Disponivel {
				return psRes, nil
			}
		}
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
		TotalRegistros: matchedTotal,
		Diarias:        items,
		Disponivel:     true,
	}, nil
}

// QueryHTMLPortalObras consulta obras públicas a partir de tabelas HTML em portais sem API estruturada.
func QueryHTMLPortalObras(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams, plataforma string) (*ObrasResult, error) {
	usedURL, topLinks, tables, err := fetchHTMLDomainTablesWithParams(ctx, client, p, "obras")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fonte := fmt.Sprintf("Portal Municipal HTML (%s)", htmlPlatformLabel(plataforma))
	var items []ObraItem
	matchedTotal := 0

	for _, tb := range tables {
		nh := normalizeHTMLTableHeaders(tb.Headers)
		idxTit := findColIdx(nh, "titulo", "obra", "empreendimento", "nome")
		idxObj := findColIdx(nh, "objeto", "descricao", "especificacao")
		idxCod := findColIdx(nh, "codigo", "numero", "processo", "contrato")
		if idxTit < 0 && idxObj < 0 && idxCod < 0 {
			continue
		}
		idxOrg := findColIdx(nh, "orgao", "secretaria", "unidade")
		idxEnd := findColIdx(nh, "endereco", "local", "localizacao", "logradouro")
		idxBairro := findColIdx(nh, "bairro", "distrito", "regiao")
		idxIni := findColIdx(nh, "inicio", "data inicial", "ordem de servico")
		idxFim := findColIdx(nh, "final", "fim", "conclusao", "termino", "prazo")
		idxSit := findColIdx(nh, "situacao", "status", "andamento", "estagio")
		idxVal := findColIdx(nh, "valor", "orcado", "contratado", "total", "investimento")

		for _, row := range tb.Rows {
			tit := cellAt(row, idxTit)
			obj := cellAt(row, idxObj)
			cod := cellAt(row, idxCod)
			if tit == "" {
				tit = obj
			}
			if obj == "" {
				obj = tit
			}
			if tit == "" && obj == "" && cod == "" {
				continue
			}
			if filtro != "" && !strings.Contains(strings.ToLower(tit+" "+obj+" "+cod), filtro) {
				continue
			}
			matchedTotal++
			if len(items) >= limit {
				continue
			}
			items = append(items, ObraItem{
				Codigo:      cod,
				Titulo:      tit,
				Objeto:      obj,
				Orgao:       cellAt(row, idxOrg),
				Endereco:    cellAt(row, idxEnd),
				Bairro:      cellAt(row, idxBairro),
				DataInicial: cellAt(row, idxIni),
				DataFinal:   cellAt(row, idxFim),
				Situacao:    cellAt(row, idxSit),
				Valor:       parseHTMLBRFloat(cellAt(row, idxVal)),
				Fonte:       fonte,
			})
		}
	}

	if len(items) == 0 {
		if psURL := findEmbeddedPublicsoftURL(usedURL, topLinks); psURL != "" {
			psParams := p
			psParams.BaseURL = psURL
			psParams.TenantID = ""
			if psRes, psErr := QueryPublicsoftObras(ctx, client, m, psParams); psErr == nil && psRes != nil && psRes.Disponivel {
				return psRes, nil
			}
		}
		return &ObrasResult{
			Municipio:  m,
			Fonte:      fonte,
			Plataforma: plataforma,
			PortalURL:  usedURL,
			Obras:      []ObraItem{},
			Disponivel: false,
			Mensagem:   htmlNoRowsMessage(plataforma, usedURL, topLinks),
		}, errHTMLNoTableRows
	}

	return &ObrasResult{
		Municipio:      m,
		Fonte:          fonte,
		Plataforma:     plataforma,
		PortalURL:      usedURL,
		TotalRegistros: matchedTotal,
		Obras:          items,
		Disponivel:     true,
	}, nil
}

// QueryHTMLPortalConvenios consulta convênios e transferências a partir de tabelas HTML em portais sem API estruturada.
func QueryHTMLPortalConvenios(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams, plataforma string) (*ConveniosResult, error) {
	usedURL, topLinks, tables, err := fetchHTMLDomainTablesWithParams(ctx, client, p, "convenios")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fonte := fmt.Sprintf("Portal Municipal HTML (%s)", htmlPlatformLabel(plataforma))
	var items []ConvenioItem
	matchedTotal := 0

	for _, tb := range tables {
		nh := normalizeHTMLTableHeaders(tb.Headers)
		idxNum := findColIdx(nh, "convenio", "numero", "termo", "codigo")
		idxConc := findColIdx(nh, "concedente", "orgao concedente", "origem")
		idxConv := findColIdx(nh, "convenente", "beneficiario", "entidade", "parceiro")
		idxDesc := findColIdx(nh, "objeto", "descricao", "finalidade")
		if idxNum < 0 && idxConc < 0 && idxConv < 0 && idxDesc < 0 {
			continue
		}
		idxCat := findColIdx(nh, "categoria", "modalidade")
		idxTipo := findColIdx(nh, "tipo", "instrumento")
		idxSit := findColIdx(nh, "situacao", "status")
		idxData := findColIdx(nh, "data", "assinatura", "celebracao")
		idxVig := findColIdx(nh, "vigencia", "vencimento", "fim")
		idxVal := findColIdx(nh, "valor", "global", "total", "pactuado")
		idxLib := findColIdx(nh, "liberado", "repassado", "pago")

		for _, row := range tb.Rows {
			num := cellAt(row, idxNum)
			conc := cellAt(row, idxConc)
			conv := cellAt(row, idxConv)
			desc := cellAt(row, idxDesc)
			if num == "" && conc == "" && conv == "" && desc == "" {
				continue
			}
			if filtro != "" && !strings.Contains(strings.ToLower(num+" "+conc+" "+conv+" "+desc), filtro) {
				continue
			}
			matchedTotal++
			if len(items) >= limit {
				continue
			}
			items = append(items, ConvenioItem{
				Numero:        num,
				Concedente:    conc,
				Convenente:    conv,
				Categoria:     cellAt(row, idxCat),
				Tipo:          cellAt(row, idxTipo),
				Situacao:      cellAt(row, idxSit),
				Data:          cellAt(row, idxData),
				Vigencia:      cellAt(row, idxVig),
				Descricao:     desc,
				Valor:         parseHTMLBRFloat(cellAt(row, idxVal)),
				ValorLiberado: parseHTMLBRFloat(cellAt(row, idxLib)),
				Fonte:         fonte,
			})
		}
	}

	if len(items) == 0 {
		if psURL := findEmbeddedPublicsoftURL(usedURL, topLinks); psURL != "" {
			psParams := p
			psParams.BaseURL = psURL
			psParams.TenantID = ""
			if psRes, psErr := QueryPublicsoftConvenios(ctx, client, m, psParams); psErr == nil && psRes != nil && psRes.Disponivel {
				return psRes, nil
			}
		}
		return &ConveniosResult{
			Municipio:  m,
			Fonte:      fonte,
			Plataforma: plataforma,
			PortalURL:  usedURL,
			Convenios:  []ConvenioItem{},
			Disponivel: false,
			Mensagem:   htmlNoRowsMessage(plataforma, usedURL, topLinks),
		}, errHTMLNoTableRows
	}

	return &ConveniosResult{
		Municipio:      m,
		Fonte:          fonte,
		Plataforma:     plataforma,
		PortalURL:      usedURL,
		TotalRegistros: matchedTotal,
		Convenios:      items,
		Disponivel:     true,
	}, nil
}

// QueryHTMLPortalLegislacao consulta legislação e atos normativos a partir de tabelas HTML em portais sem API estruturada.
func QueryHTMLPortalLegislacao(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams, plataforma string) (*LegislacaoResult, error) {
	usedURL, topLinks, tables, err := fetchHTMLDomainTablesWithParams(ctx, client, p, "legislacao")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fonte := fmt.Sprintf("Portal Municipal HTML (%s)", htmlPlatformLabel(plataforma))
	var items []LegislacaoItem
	matchedTotal := 0

	for _, tb := range tables {
		nh := normalizeHTMLTableHeaders(tb.Headers)
		idxTipo := findColIdx(nh, "tipo", "especie")
		idxNum := findColIdx(nh, "numero", "ato", "lei", "decreto", "portaria")
		idxEmenta := findColIdx(nh, "ementa", "assunto", "resumo")
		if idxTipo < 0 && idxNum < 0 && idxEmenta < 0 {
			continue
		}
		idxDesc := findColIdx(nh, "ementa", "descricao", "assunto", "objeto", "resumo")
		idxNome := findColIdx(nh, "nome", "titulo")
		idxCod := findColIdx(nh, "codigo", "id")
		idxData := findColIdx(nh, "publicacao", "data")
		idxSit := findColIdx(nh, "situacao", "status", "vigencia")

		for _, row := range tb.Rows {
			tipo := cellAt(row, idxTipo)
			num := cellAt(row, idxNum)
			desc := cellAt(row, idxDesc)
			nome := cellAt(row, idxNome)
			if nome == "" && tipo != "" && num != "" {
				nome = tipo + " " + num
			}
			if tipo == "" && num == "" && desc == "" && nome == "" {
				continue
			}
			if filtro != "" && !strings.Contains(strings.ToLower(tipo+" "+num+" "+desc+" "+nome), filtro) {
				continue
			}
			matchedTotal++
			if len(items) >= limit {
				continue
			}
			items = append(items, LegislacaoItem{
				Codigo:         cellAt(row, idxCod),
				TipoDocumento:  tipo,
				Numero:         num,
				Ano:            p.Ano,
				DataPublicacao: cellAt(row, idxData),
				Situacao:       cellAt(row, idxSit),
				Nome:           nome,
				Descricao:      desc,
				Fonte:          fonte,
			})
		}
	}

	if len(items) == 0 {
		return &LegislacaoResult{
			Municipio:  m,
			Fonte:      fonte,
			Plataforma: plataforma,
			PortalURL:  usedURL,
			Atos:       []LegislacaoItem{},
			Disponivel: false,
			Mensagem:   htmlNoRowsMessage(plataforma, usedURL, topLinks),
		}, errHTMLNoTableRows
	}

	return &LegislacaoResult{
		Municipio:      m,
		Fonte:          fonte,
		Plataforma:     plataforma,
		PortalURL:      usedURL,
		TotalRegistros: matchedTotal,
		Atos:           items,
		Disponivel:     true,
	}, nil
}

// QueryHTMLPortalConcursos consulta aprovados e editais de concursos/processos seletivos a partir de tabelas HTML.
func QueryHTMLPortalConcursos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams, plataforma string) (*ConcursosResult, error) {
	usedURL, topLinks, tables, err := fetchHTMLDomainTablesWithParams(ctx, client, p, "concursos")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	fonte := fmt.Sprintf("Portal Municipal HTML (%s)", htmlPlatformLabel(plataforma))
	var items []ConcursoItem
	matchedTotal := 0

	for _, tb := range tables {
		nh := normalizeHTMLTableHeaders(tb.Headers)
		idxNome := findColIdx(nh, "nome", "candidato", "aprovado")
		idxEdital := findColIdx(nh, "edital", "concurso", "processo seletivo")
		idxCargo := findColIdx(nh, "cargo", "funcao", "vaga")
		idxInsc := findColIdx(nh, "inscricao", "matricula")
		idxClass := findColIdx(nh, "classificacao", "posicao", "colocacao")
		idxCandExplicit := findColIdx(nh, "candidato", "aprovado")
		if idxEdital < 0 && idxCargo < 0 && idxInsc < 0 && idxClass < 0 && idxCandExplicit < 0 {
			continue
		}
		idxData := findColIdx(nh, "publicacao", "data")
		idxSit := findColIdx(nh, "situacao", "status", "resultado")

		for _, row := range tb.Rows {
			nome := cellAt(row, idxNome)
			edital := cellAt(row, idxEdital)
			cargo := cellAt(row, idxCargo)
			if nome == "" && edital == "" && cargo == "" {
				continue
			}
			if filtro != "" && !strings.Contains(strings.ToLower(nome+" "+edital+" "+cargo), filtro) {
				continue
			}
			matchedTotal++
			if len(items) >= limit {
				continue
			}
			classif := int(parseHTMLBRFloat(cellAt(row, idxClass)))
			items = append(items, ConcursoItem{
				Edital:         edital,
				DataPublicacao: cellAt(row, idxData),
				Nome:           nome,
				Cargo:          cargo,
				Inscricao:      cellAt(row, idxInsc),
				Classificacao:  classif,
				Situacao:       cellAt(row, idxSit),
				Fonte:          fonte,
			})
		}
	}

	if len(items) == 0 {
		return &ConcursosResult{
			Municipio:  m,
			Fonte:      fonte,
			Plataforma: plataforma,
			PortalURL:  usedURL,
			Aprovados:  []ConcursoItem{},
			Disponivel: false,
			Mensagem:   htmlNoRowsMessage(plataforma, usedURL, topLinks),
		}, errHTMLNoTableRows
	}

	return &ConcursosResult{
		Municipio:      m,
		Fonte:          fonte,
		Plataforma:     plataforma,
		PortalURL:      usedURL,
		TotalRegistros: matchedTotal,
		Aprovados:      items,
		Disponivel:     true,
	}, nil
}
