package transparencia

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Extração Avançada sem Browser para Portais HTML:
//   1. JSON embutido em <script> + endpoints XHR/DataTables/Bootstrap-Table
//   2. Links de exportação CSV / JSON na página
//   3. Submissão automática de <form> de filtro padrão com CookieJar
// ---------------------------------------------------------------------------

var (
	htmlScriptBlockRegex    = regexp.MustCompile(`(?is)<script\b([^>]*)>(.*?)</script>`)
	htmlAttrTypeRegex       = regexp.MustCompile(`(?is)\btype\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	htmlAttrIDRegex         = regexp.MustCompile(`(?is)\bid\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	htmlAttrNameRegex       = regexp.MustCompile(`(?is)\bname\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	htmlAttrValueRegex      = regexp.MustCompile(`(?is)\bvalue\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	htmlAttrActionRegex     = regexp.MustCompile(`(?is)\baction\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	htmlAttrMethodRegex     = regexp.MustCompile(`(?is)\bmethod\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	htmlAttrClassRegex      = regexp.MustCompile(`(?is)\bclass\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	htmlDataURLAttrRegex    = regexp.MustCompile(`(?is)\bdata-(?:url|ajax|source|src)\s*=\s*(?:"([^"]+)"|'([^']+)')`)
	htmlDataTablesAjaxRegex = regexp.MustCompile(`(?is)\bajax\s*:\s*(?:\{[^}]{0,300}?\burl\s*:\s*)?["']([^"']+)["']`)
	htmlFetchXHRRegex       = regexp.MustCompile(`(?is)(?:fetch|\$\.getJSON|\$\.get|\$\.post|axios\.get|axios\.post)\s*\(\s*["']([^"']+)["']`)
	htmlAjaxObjURLRegex     = regexp.MustCompile(`(?is)\$\.ajax\s*\(\s*\{[^}]{0,500}?\burl\s*:\s*["']([^"']+)["']`)
	htmlScriptVarJSONRegex  = regexp.MustCompile(`(?is)(?:window\.[A-Za-z0-9_]+|(?:var|let|const)\s+[A-Za-z0-9_]+)\s*=\s*([\[{])`)

	htmlFormRegex     = regexp.MustCompile(`(?is)<form\b([^>]*)>(.*?)</form>`)
	htmlInputRegex    = regexp.MustCompile(`(?is)<input\b([^>]*)/?>`)
	htmlSelectRegex   = regexp.MustCompile(`(?is)<select\b([^>]*)>(.*?)</select>`)
	htmlOptionRegex   = regexp.MustCompile(`(?is)<option\b([^>]*)>(.*?)</option>`)
	htmlButtonRegex   = regexp.MustCompile(`(?is)<button\b([^>]*)>(.*?)</button>`)
	htmlTHeadRegex    = regexp.MustCompile(`(?is)<thead\b[^>]*>(.*?)</thead>`)
	htmlCheckedRegex  = regexp.MustCompile(`(?is)\bchecked\b`)
	htmlSelectedRegex = regexp.MustCompile(`(?is)\bselected\b`)
	htmlYear4Regex    = regexp.MustCompile(`^(19|20)\d{2}$`)
)

// sessionHTTPClient cria uma cópia leve do client com CookieJar em memória quando ausente,
// preservando Transport, Timeout e CheckRedirect sem mutar o client original.
func sessionHTTPClient(base *http.Client) *http.Client {
	c := portalConnectorHTTPClient(base)
	if c.Jar != nil {
		return c
	}
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Transport:     c.Transport,
		CheckRedirect: c.CheckRedirect,
		Timeout:       c.Timeout,
		Jar:           jar,
	}
}

func fetchRawResource(ctx context.Context, client *http.Client, targetURL, referer string) (finalURL string, contentType string, body string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return "", "", "", err
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	req.Header.Set("Accept", "application/json, text/csv, text/html, application/xml;q=0.9, */*;q=0.8")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en;q=0.8")
	if referer != "" {
		req.Header.Set("Referer", referer)
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
	}

	resp, err := portalConnectorHTTPClient(client).Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	finalURL = targetURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}

	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if readErr != nil {
		return finalURL, "", "", readErr
	}
	bodyStr := string(raw)
	bodyLow := strings.ToLower(bodyStr)

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusServiceUnavailable {
		for _, marker := range antiBotMarkers {
			if strings.Contains(bodyLow, marker) {
				return finalURL, "", "", fmt.Errorf("%w (HTTP %d)", errHTMLAntiBotBlocked, resp.StatusCode)
			}
		}
	}
	if resp.StatusCode >= 400 {
		return finalURL, "", "", fmt.Errorf("recurso retornou HTTP %d", resp.StatusCode)
	}
	return finalURL, strings.ToLower(resp.Header.Get("Content-Type")), bodyStr, nil
}

func extractTableSkeletonHeaders(htmlBody string) [][]string {
	cleaned := htmlScriptStyleRegex.ReplaceAllString(htmlBody, " ")
	var headerSets [][]string
	for _, tm := range htmlTableRegex.FindAllStringSubmatch(cleaned, -1) {
		if len(tm) < 2 {
			continue
		}
		theadM := htmlTHeadRegex.FindStringSubmatch(tm[1])
		scope := tm[1]
		if len(theadM) >= 2 {
			scope = theadM[1]
		}
		trM := htmlTRRegex.FindStringSubmatch(scope)
		if len(trM) < 2 {
			continue
		}
		var cells []string
		for _, cm := range htmlCellRegex.FindAllStringSubmatch(trM[1], -1) {
			if len(cm) >= 4 {
				if txt := cleanHTMLText(cm[3]); txt != "" {
					cells = append(cells, txt)
				}
			}
		}
		if len(cells) >= 2 {
			headerSets = append(headerSets, cells)
		}
	}
	return headerSets
}

func jsonScalarToString(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(cleanHTMLText(val))
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', 2, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		return ""
	}
}

func jsonObjectsToHTMLTable(objs []map[string]any) (htmlTable, bool) {
	if len(objs) == 0 {
		return htmlTable{}, false
	}
	if _, hasType := objs[0]["@type"]; hasType {
		return htmlTable{}, false
	}
	if _, hasCtx := objs[0]["@context"]; hasCtx {
		return htmlTable{}, false
	}
	keySet := make(map[string]struct{})
	for i := 0; i < len(objs) && i < 25; i++ {
		for k, v := range objs[i] {
			if strings.HasPrefix(k, "@") {
				continue
			}
			switch v.(type) {
			case map[string]any, []any:
				continue
			}
			if strings.TrimSpace(k) != "" {
				keySet[k] = struct{}{}
			}
		}
	}
	if len(keySet) < 2 {
		return htmlTable{}, false
	}
	headers := make([]string, 0, len(keySet))
	for k := range keySet {
		headers = append(headers, k)
	}
	sort.Strings(headers)

	rows := make([][]string, 0, len(objs))
	for _, obj := range objs {
		row := make([]string, len(headers))
		nonEmpty := false
		for i, h := range headers {
			cell := jsonScalarToString(obj[h])
			if cell != "" {
				nonEmpty = true
			}
			row[i] = cell
		}
		if nonEmpty {
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		return htmlTable{}, false
	}
	return htmlTable{Headers: headers, Rows: rows}, true
}

func jsonArrayOfArraysToHTMLTable(matrix [][]any, headersHint [][]string) (htmlTable, bool) {
	if len(matrix) == 0 {
		return htmlTable{}, false
	}
	// Detecta número de colunas na primeira linha válida
	numCols := 0
	for _, r := range matrix {
		if len(r) >= 2 {
			numCols = len(r)
			break
		}
	}
	if numCols < 2 {
		return htmlTable{}, false
	}

	var headers []string
	startIdx := 0
	for _, cand := range headersHint {
		if len(cand) == numCols {
			headers = cand
			break
		}
	}
	if len(headers) == 0 && len(matrix) >= 2 {
		// Usa a primeira linha como cabeçalho se todos os valores forem strings não numéricas
		firstAllText := true
		candHeaders := make([]string, numCols)
		for i := 0; i < numCols && i < len(matrix[0]); i++ {
			s := jsonScalarToString(matrix[0][i])
			if s == "" {
				firstAllText = false
				break
			}
			if _, err := parseBRFloat(s); err == nil {
				firstAllText = false
				break
			}
			candHeaders[i] = s
		}
		if firstAllText {
			headers = candHeaders
			startIdx = 1
		}
	}
	if len(headers) == 0 {
		return htmlTable{}, false
	}

	var rows [][]string
	for i := startIdx; i < len(matrix); i++ {
		rawRow := matrix[i]
		if len(rawRow) < 2 {
			continue
		}
		row := make([]string, len(headers))
		nonEmpty := false
		for c := 0; c < len(headers) && c < len(rawRow); c++ {
			cell := jsonScalarToString(rawRow[c])
			if cell != "" {
				nonEmpty = true
			}
			row[c] = cell
		}
		if nonEmpty {
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		return htmlTable{}, false
	}
	return htmlTable{Headers: headers, Rows: rows}, true
}

func collectTablesFromJSONValue(v any, headersHint [][]string, depth int, out *[]htmlTable) {
	if depth > 6 || v == nil {
		return
	}
	switch val := v.(type) {
	case []any:
		if len(val) == 0 {
			return
		}
		// Array de objetos JSON
		var objs []map[string]any
		for _, el := range val {
			if m, ok := el.(map[string]any); ok {
				objs = append(objs, m)
			}
		}
		if len(objs) > 0 {
			if tb, ok := jsonObjectsToHTMLTable(objs); ok {
				*out = append(*out, tb)
				return
			}
		}
		// Array de arrays (ex.: DataTables aaData / data)
		var matrix [][]any
		for _, el := range val {
			if arr, ok := el.([]any); ok {
				matrix = append(matrix, arr)
			}
		}
		if len(matrix) > 0 {
			if tb, ok := jsonArrayOfArraysToHTMLTable(matrix, headersHint); ok {
				*out = append(*out, tb)
				return
			}
		}
	case map[string]any:
		if _, hasCtx := val["@context"]; hasCtx {
			return
		}
		if _, hasType := val["@type"]; hasType {
			return
		}
		// Chaves comuns primeiro para priorizar os arrays principais de dados
		preferred := []string{"data", "aaData", "rows", "records", "items", "content", "resultado", "resultados", "lista", "dados", "servidores", "despesas", "receitas", "licitacoes", "contratos", "diarias", "obras", "convenios", "legislacao", "atos", "concursos", "aprovados"}
		seen := make(map[string]struct{})
		for _, k := range preferred {
			if child, exists := val[k]; exists {
				seen[k] = struct{}{}
				collectTablesFromJSONValue(child, headersHint, depth+1, out)
			}
		}
		keys := make([]string, 0, len(val))
		for k := range val {
			if _, ok := seen[k]; !ok {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		for _, k := range keys {
			collectTablesFromJSONValue(val[k], headersHint, depth+1, out)
		}
	}
}

func parseJSONToHTMLTables(raw string, headersHint [][]string) []htmlTable {
	trimmed := strings.TrimSpace(strings.TrimPrefix(raw, "\xef\xbb\xbf"))
	if trimmed == "" || (trimmed[0] != '{' && trimmed[0] != '[') {
		return nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return nil
	}
	var out []htmlTable
	collectTablesFromJSONValue(decoded, headersHint, 0, &out)
	return out
}

func parseCSVToHTMLTables(raw string) []htmlTable {
	trimmed := strings.TrimSpace(strings.TrimPrefix(raw, "\xef\xbb\xbf"))
	if trimmed == "" {
		return nil
	}
	low := strings.ToLower(trimmed)
	if strings.HasPrefix(low, "<!doctype") || strings.HasPrefix(low, "<html") || strings.HasPrefix(low, "{") || strings.HasPrefix(low, "[") {
		return nil
	}

	// Detecta delimitador predominante (; vs , vs \t) nas primeiras linhas
	lines := strings.Split(trimmed, "\n")
	if len(lines) < 2 {
		return nil
	}
	sample := strings.Join(lines[:minInt(5, len(lines))], "\n")
	delim := ','
	semiCount := strings.Count(sample, ";")
	commaCount := strings.Count(sample, ",")
	tabCount := strings.Count(sample, "\t")
	if semiCount > commaCount && semiCount >= tabCount {
		delim = ';'
	} else if tabCount > commaCount && tabCount > semiCount {
		delim = '\t'
	}

	r := csv.NewReader(bytes.NewBufferString(trimmed))
	r.Comma = delim
	r.LazyQuotes = true
	r.FieldsPerRecord = -1

	records, err := r.ReadAll()
	if err != nil || len(records) < 2 {
		return nil
	}

	// Pula linhas iniciais de título/metadados com menos de 2 colunas
	headerIdx := -1
	for i := 0; i < len(records) && i < 5; i++ {
		nonEmpty := 0
		for _, c := range records[i] {
			if strings.TrimSpace(c) != "" {
				nonEmpty++
			}
		}
		if nonEmpty >= 2 {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 || headerIdx+1 >= len(records) {
		return nil
	}

	headers := make([]string, len(records[headerIdx]))
	for i, h := range records[headerIdx] {
		headers[i] = strings.TrimSpace(h)
	}
	var rows [][]string
	for i := headerIdx + 1; i < len(records); i++ {
		if len(records[i]) < 2 {
			continue
		}
		row := make([]string, len(headers))
		nonEmpty := false
		for c := 0; c < len(headers) && c < len(records[i]); c++ {
			v := strings.TrimSpace(records[i][c])
			if v != "" {
				nonEmpty = true
			}
			row[c] = v
		}
		if nonEmpty {
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		return nil
	}
	return []htmlTable{{Headers: headers, Rows: rows}}
}

func parseStructuredResponseToTables(baseURL, contentType, body string, headersHint [][]string) []htmlTable {
	trimmed := strings.TrimSpace(strings.TrimPrefix(body, "\xef\xbb\xbf"))
	if trimmed == "" {
		return nil
	}
	if strings.Contains(contentType, "json") || trimmed[0] == '{' || trimmed[0] == '[' {
		if tbs := parseJSONToHTMLTables(trimmed, headersHint); len(tbs) > 0 {
			return tbs
		}
	}
	if strings.Contains(contentType, "html") || strings.Contains(strings.ToLower(trimmed[:minInt(256, len(trimmed))]), "<table") {
		if tbs := extractHTMLTables(baseURL, body); len(tbs) > 0 {
			return tbs
		}
	}
	if strings.Contains(contentType, "csv") || strings.Contains(contentType, "text/plain") || !strings.HasPrefix(trimmed, "<") {
		if tbs := parseCSVToHTMLTables(trimmed); len(tbs) > 0 {
			return tbs
		}
	}
	return extractHTMLTables(baseURL, body)
}

func isStaticAssetOrThirdPartyNoise(resolvedURL string) bool {
	u, err := url.Parse(resolvedURL)
	if err != nil {
		return true
	}
	host := strings.ToLower(u.Hostname())
	for _, noisy := range []string{"google", "gstatic", "googletagmanager", "facebook", "fbcdn", "instagram", "youtube", "twitter", "x.com", "tiktok", "whatsapp", "cloudflare", "jsdelivr", "unpkg", "cdnjs", "fontawesome"} {
		if strings.Contains(host, noisy) {
			return true
		}
	}
	lowPath := strings.ToLower(u.Path)
	for _, ext := range []string{".js", ".css", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot", ".map"} {
		if strings.HasSuffix(lowPath, ext) {
			return true
		}
	}
	return false
}

func isSameOrSubdomainHost(candidateHost, baseHost string) bool {
	c := strings.ToLower(strings.TrimSpace(candidateHost))
	b := strings.ToLower(strings.TrimSpace(baseHost))
	if c == "" || b == "" {
		return false
	}
	return c == b || strings.HasSuffix(c, "."+b) || strings.HasSuffix(b, "."+c)
}

func extractEmbeddedScriptTables(body string, headersHint [][]string) []htmlTable {
	var tables []htmlTable
	for _, sm := range htmlScriptBlockRegex.FindAllStringSubmatch(body, -1) {
		if len(sm) < 3 {
			continue
		}
		attrs := strings.ToLower(sm[1])
		if strings.Contains(attrs, "ld+json") {
			continue
		}
		scriptBody := strings.TrimSpace(sm[2])
		if scriptBody == "" {
			continue
		}
		if strings.Contains(attrs, "application/json") || strings.Contains(attrs, "__next_data__") || scriptBody[0] == '{' || scriptBody[0] == '[' {
			if tbs := parseJSONToHTMLTables(scriptBody, headersHint); len(tbs) > 0 {
				tables = append(tables, tbs...)
			}
			continue
		}
		// Tenta extrair atribuições JS do tipo window.__STATE__ = {...} ou var dados = [...]
		for _, loc := range htmlScriptVarJSONRegex.FindAllStringSubmatchIndex(scriptBody, 8) {
			if len(loc) < 4 {
				continue
			}
			start := loc[2]
			sub := scriptBody[start:]
			dec := json.NewDecoder(strings.NewReader(sub))
			var v any
			if err := dec.Decode(&v); err == nil {
				collectTablesFromJSONValue(v, headersHint, 0, &tables)
			}
		}
	}
	return tables
}

func extractXHRCandidateURLs(baseURL, body, domain string) []string {
	baseParsed, err := url.Parse(baseURL)
	if err != nil {
		return nil
	}
	baseHost := baseParsed.Hostname()
	kws := htmlDomainKeywords(domain)

	type scoredURL struct {
		url   string
		score int
	}
	seen := make(map[string]struct{})
	var cands []scoredURL

	addCand := func(rawRef string, bonus int) {
		resolved := resolveRelativeHTTPURL(baseParsed, rawRef)
		if resolved == "" || resolved == baseURL || isStaticAssetOrThirdPartyNoise(resolved) {
			return
		}
		u, err := url.Parse(resolved)
		if err != nil || !isSameOrSubdomainHost(u.Hostname(), baseHost) {
			return
		}
		if _, ok := seen[resolved]; ok {
			return
		}
		seen[resolved] = struct{}{}

		hay := RemoveAccents(u.Path + " " + u.RawQuery)
		sc := bonus
		for i, kw := range kws {
			if strings.Contains(hay, kw) {
				sc += (len(kws) - i) * 10
			}
		}
		for _, hint := range []string{"api", "dados", "json", "csv", "datatable", "lista", "consulta", "tabela", "export", "ajax"} {
			if strings.Contains(hay, hint) {
				sc += 8
			}
		}
		if sc > 0 {
			cands = append(cands, scoredURL{url: resolved, score: sc})
		}
	}

	for _, m := range htmlDataURLAttrRegex.FindAllStringSubmatch(body, -1) {
		raw := m[1]
		if raw == "" {
			raw = m[2]
		}
		addCand(raw, 15)
	}
	for _, sm := range htmlScriptBlockRegex.FindAllStringSubmatch(body, -1) {
		if len(sm) < 3 {
			continue
		}
		s := sm[2]
		for _, m := range htmlDataTablesAjaxRegex.FindAllStringSubmatch(s, -1) {
			addCand(m[1], 20)
		}
		for _, m := range htmlFetchXHRRegex.FindAllStringSubmatch(s, -1) {
			addCand(m[1], 12)
		}
		for _, m := range htmlAjaxObjURLRegex.FindAllStringSubmatch(s, -1) {
			addCand(m[1], 15)
		}
	}

	sort.SliceStable(cands, func(i, j int) bool {
		return cands[i].score > cands[j].score
	})
	out := make([]string, 0, len(cands))
	for _, c := range cands {
		out = append(out, c.url)
	}
	return out
}

func isExportDataLink(lk htmlLink) bool {
	u, err := url.Parse(lk.URL)
	if err != nil {
		return false
	}
	lowPath := strings.ToLower(u.Path)
	for _, ext := range []string{".pdf", ".zip", ".rar", ".xls", ".xlsx", ".doc", ".docx", ".jpg", ".jpeg", ".png", ".gif", ".svg", ".js", ".css"} {
		if strings.HasSuffix(lowPath, ext) {
			return false
		}
	}
	if strings.HasSuffix(lowPath, ".csv") || strings.HasSuffix(lowPath, ".json") {
		return true
	}
	hay := RemoveAccents(lk.Label + " " + u.Path + " " + u.RawQuery)
	if strings.Contains(hay, "csv") || strings.Contains(hay, "json") || strings.Contains(hay, "exportar dados") || strings.Contains(hay, "dados abertos") {
		return true
	}
	return strings.Contains(hay, "export") && (strings.Contains(hay, "formato=") || strings.Contains(hay, "tipo=") || strings.Contains(hay, "download"))
}

func extractExportLinkTables(ctx context.Context, client *http.Client, baseURL, body, domain string, headersHint [][]string) (usedURL string, tables []htmlTable) {
	baseParsed, err := url.Parse(baseURL)
	if err != nil {
		return "", nil
	}
	baseHost := baseParsed.Hostname()
	links := extractHTMLAnchorLinks(baseURL, body)
	kws := htmlDomainKeywords(domain)

	type scoredExport struct {
		url   string
		score int
	}
	var cands []scoredExport
	for _, lk := range links {
		if !isExportDataLink(lk) || isStaticAssetOrThirdPartyNoise(lk.URL) {
			continue
		}
		u, err := url.Parse(lk.URL)
		if err != nil || !isSameOrSubdomainHost(u.Hostname(), baseHost) {
			continue
		}
		hay := RemoveAccents(lk.Label + " " + u.Path + " " + u.RawQuery)
		sc := 10
		if strings.HasSuffix(strings.ToLower(u.Path), ".csv") || strings.Contains(hay, "csv") {
			sc += 12
		}
		if strings.HasSuffix(strings.ToLower(u.Path), ".json") || strings.Contains(hay, "json") {
			sc += 14
		}
		for i, kw := range kws {
			if strings.Contains(hay, kw) {
				sc += (len(kws) - i) * 10
			}
		}
		cands = append(cands, scoredExport{url: lk.URL, score: sc})
	}
	sort.SliceStable(cands, func(i, j int) bool {
		return cands[i].score > cands[j].score
	})

	for i := 0; i < len(cands) && i < 2; i++ {
		finalU, ctype, rawBody, fetchErr := fetchRawResource(ctx, client, cands[i].url, baseURL)
		if fetchErr != nil {
			continue
		}
		if tbs := parseStructuredResponseToTables(finalU, ctype, rawBody, headersHint); len(tbs) > 0 {
			return finalU, tbs
		}
	}
	return "", nil
}

func pickSelectOptionValue(selectBody, fieldName string, targetAno int) string {
	type opt struct {
		val      string
		label    string
		selected bool
	}
	var opts []opt
	for _, om := range htmlOptionRegex.FindAllStringSubmatch(selectBody, -1) {
		if len(om) < 3 {
			continue
		}
		attrs := om[1]
		label := cleanHTMLText(om[2])
		val := extractAttrValue(attrs, htmlAttrValueRegex)
		if val == "" {
			val = label
		}
		opts = append(opts, opt{
			val:      val,
			label:    label,
			selected: htmlSelectedRegex.MatchString(attrs),
		})
	}
	if len(opts) == 0 {
		return ""
	}
	normName := RemoveAccents(fieldName)
	if strings.Contains(normName, "ano") || strings.Contains(normName, "exerc") {
		if targetAno > 0 {
			want := strconv.Itoa(targetAno)
			for _, o := range opts {
				if o.val == want || o.label == want {
					return o.val
				}
			}
		}
		// Prefere opção selecionada se for ano válido
		for _, o := range opts {
			if o.selected && htmlYear4Regex.MatchString(strings.TrimSpace(o.val)) {
				return o.val
			}
		}
		// Caso contrário pega o maior ano válido
		bestYear := ""
		bestVal := ""
		for _, o := range opts {
			v := strings.TrimSpace(o.val)
			if htmlYear4Regex.MatchString(v) && v > bestYear {
				bestYear = v
				bestVal = o.val
			}
		}
		if bestVal != "" {
			return bestVal
		}
	}
	for _, o := range opts {
		if o.selected && strings.TrimSpace(o.val) != "" {
			return o.val
		}
	}
	for _, o := range opts {
		v := strings.TrimSpace(o.val)
		if v != "" && !strings.EqualFold(v, "0") && !strings.Contains(RemoveAccents(o.label), "selecione") && !strings.Contains(RemoveAccents(o.label), "todos") {
			return o.val
		}
	}
	return opts[0].val
}

func isUnsafeOrLoginForm(formAttrs, formBody string) bool {
	lowBody := strings.ToLower(formBody)
	lowAttrs := strings.ToLower(formAttrs)
	if strings.Contains(lowBody, `type="password"`) || strings.Contains(lowBody, `type='password'`) {
		return true
	}
	for _, marker := range []string{"g-recaptcha", "h-captcha", "hcaptcha", "cf-turnstile", "recaptcha", "captcha"} {
		if strings.Contains(lowBody, marker) {
			return true
		}
	}
	combined := RemoveAccents(lowAttrs + " " + extractAttrValue(formAttrs, htmlAttrActionRegex))
	for _, blocked := range []string{"login", "logar", "autentic", "signin", "entrar", "newsletter", "ouvidoria", "fale-conosco", "contato"} {
		if strings.Contains(combined, blocked) {
			return true
		}
	}
	return false
}

func submitBestDomainForm(ctx context.Context, client *http.Client, baseURL, body, domain string, params PortalQueryParams) (usedURL string, tables []htmlTable, err error) {
	baseParsed, err := url.Parse(baseURL)
	if err != nil {
		return "", nil, nil
	}
	baseHost := baseParsed.Hostname()
	kws := htmlDomainKeywords(domain)

	type candidateForm struct {
		action string
		method string
		body   string
		score  int
	}
	var cands []candidateForm
	for _, fm := range htmlFormRegex.FindAllStringSubmatch(body, -1) {
		if len(fm) < 3 {
			continue
		}
		attrs, fBody := fm[1], fm[2]
		if isUnsafeOrLoginForm(attrs, fBody) {
			continue
		}
		actionAttr := extractAttrValue(attrs, htmlAttrActionRegex)
		actionURL := baseURL
		if actionAttr != "" {
			resolved := resolveRelativeHTTPURL(baseParsed, actionAttr)
			if resolved == "" {
				continue
			}
			u, uErr := url.Parse(resolved)
			if uErr != nil || !isSameOrSubdomainHost(u.Hostname(), baseHost) {
				continue
			}
			actionURL = resolved
		}
		method := strings.ToUpper(strings.TrimSpace(extractAttrValue(attrs, htmlAttrMethodRegex)))
		if method != http.MethodPost {
			method = http.MethodGet
		}

		hay := RemoveAccents(attrs + " " + cleanHTMLText(fBody))
		sc := 0
		for i, kw := range kws {
			if strings.Contains(hay, kw) {
				sc += (len(kws) - i) * 10
			}
		}
		for _, hint := range []string{"pesquisar", "consultar", "buscar", "filtrar", "exercicio", "ano", "competencia", "mes", "viewstate", "_token"} {
			if strings.Contains(hay, hint) {
				sc += 5
			}
		}
		if sc > 0 {
			cands = append(cands, candidateForm{
				action: actionURL,
				method: method,
				body:   fBody,
				score:  sc,
			})
		}
	}
	if len(cands) == 0 {
		return "", nil, nil
	}
	sort.SliceStable(cands, func(i, j int) bool {
		return cands[i].score > cands[j].score
	})

	best := cands[0]
	formVals := url.Values{}
	submitAdded := false

	for _, im := range htmlInputRegex.FindAllStringSubmatch(best.body, -1) {
		if len(im) < 2 {
			continue
		}
		iAttrs := im[1]
		name := extractAttrValue(iAttrs, htmlAttrNameRegex)
		if name == "" {
			continue
		}
		itype := strings.ToLower(extractAttrValue(iAttrs, htmlAttrTypeRegex))
		val := extractAttrValue(iAttrs, htmlAttrValueRegex)
		switch itype {
		case "reset", "button", "file", "image", "password":
			continue
		case "checkbox", "radio":
			if htmlCheckedRegex.MatchString(iAttrs) {
				if val == "" {
					val = "on"
				}
				formVals.Set(name, val)
			}
		case "submit":
			if !submitAdded {
				if val == "" {
					val = "Consultar"
				}
				formVals.Set(name, val)
				submitAdded = true
			}
		default:
			normName := RemoveAccents(name)
			if val == "" {
				if strings.Contains(normName, "ano") || strings.Contains(normName, "exerc") {
					ano := params.Ano
					if ano <= 0 {
						ano = time.Now().Year()
					}
					val = strconv.Itoa(ano)
				} else if params.NomeFiltro != "" && (strings.Contains(normName, "nome") || strings.Contains(normName, "busca") || strings.Contains(normName, "pesq") || strings.Contains(normName, "filtro")) {
					val = params.NomeFiltro
				}
			}
			formVals.Set(name, val)
		}
	}

	for _, sm := range htmlSelectRegex.FindAllStringSubmatch(best.body, -1) {
		if len(sm) < 3 {
			continue
		}
		name := extractAttrValue(sm[1], htmlAttrNameRegex)
		if name == "" {
			continue
		}
		if chosen := pickSelectOptionValue(sm[2], name, params.Ano); chosen != "" {
			formVals.Set(name, chosen)
		}
	}

	if !submitAdded {
		for _, bm := range htmlButtonRegex.FindAllStringSubmatch(best.body, -1) {
			if len(bm) < 3 {
				continue
			}
			name := extractAttrValue(bm[1], htmlAttrNameRegex)
			if name == "" {
				continue
			}
			btype := strings.ToLower(extractAttrValue(bm[1], htmlAttrTypeRegex))
			if btype == "" || btype == "submit" {
				val := extractAttrValue(bm[1], htmlAttrValueRegex)
				if val == "" {
					val = cleanHTMLText(bm[2])
				}
				formVals.Set(name, val)
				break
			}
		}
	}

	var req *http.Request
	if best.method == http.MethodPost {
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, best.action, strings.NewReader(formVals.Encode()))
		if err != nil {
			return "", nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		u, uErr := url.Parse(best.action)
		if uErr != nil {
			return "", nil, uErr
		}
		q := u.Query()
		for k, vs := range formVals {
			if len(vs) > 0 {
				q.Set(k, vs[0])
			}
		}
		u.RawQuery = q.Encode()
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return "", nil, err
		}
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	req.Header.Set("Accept", "text/html,application/json,text/csv,*/*;q=0.8")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en;q=0.8")
	req.Header.Set("Referer", baseURL)

	resp, err := portalConnectorHTTPClient(client).Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	finalURL := best.action
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if readErr != nil {
		return finalURL, nil, readErr
	}
	respBody := string(raw)
	bodyLow := strings.ToLower(respBody)
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusServiceUnavailable {
		for _, marker := range antiBotMarkers {
			if strings.Contains(bodyLow, marker) {
				return finalURL, nil, fmt.Errorf("%w (HTTP %d)", errHTMLAntiBotBlocked, resp.StatusCode)
			}
		}
	}
	if resp.StatusCode >= 400 {
		return finalURL, nil, fmt.Errorf("submissão de formulário retornou HTTP %d", resp.StatusCode)
	}

	ctype := strings.ToLower(resp.Header.Get("Content-Type"))
	headersHint := extractTableSkeletonHeaders(respBody)
	tbs := parseStructuredResponseToTables(finalURL, ctype, respBody, headersHint)
	if len(tbs) == 0 {
		tbs = extractEmbeddedScriptTables(respBody, headersHint)
	}
	return finalURL, tbs, nil
}

// extractPageAllTables tenta extrair tabelas de uma página HTML combinando:
//   - tabelas HTML estáticas (<table>)
//   - JSON embutido em <script> (__NEXT_DATA__, __INITIAL_STATE__, var dados = [...])
//   - endpoints XHR / DataTables ajax / Bootstrap-Table data-url
//   - links diretos de exportação CSV / JSON
//   - submissão automática do <form> padrão da página quando necessário
func extractPageAllTables(ctx context.Context, client *http.Client, pageURL, body, domain string, headerKWs []string, params PortalQueryParams) (usedURL string, matchedTables []htmlTable, fallbackTables []htmlTable) {
	usedURL = pageURL
	headersHint := extractTableSkeletonHeaders(body)

	filterMatching := func(tbs []htmlTable) (matched, all []htmlTable) {
		for _, tb := range tbs {
			all = append(all, tb)
			if tableHeaderScore(tb.Headers, headerKWs) >= 2 {
				matched = append(matched, tb)
			}
		}
		return matched, all
	}

	// 1. Tabelas estáticas HTML + JSON embutido no próprio HTML
	staticTables := extractHTMLTables(pageURL, body)
	mStatic, aStatic := filterMatching(staticTables)
	if len(mStatic) > 0 {
		return pageURL, mStatic, aStatic
	}
	fallbackTables = append(fallbackTables, aStatic...)

	embeddedTables := extractEmbeddedScriptTables(body, headersHint)
	mEmb, aEmb := filterMatching(embeddedTables)
	if len(mEmb) > 0 {
		return pageURL, mEmb, aEmb
	}
	fallbackTables = append(fallbackTables, aEmb...)

	// 2. Endpoints XHR / DataTables encontrados em <script> e atributos data-url
	xhrURLs := extractXHRCandidateURLs(pageURL, body, domain)
	for i := 0; i < len(xhrURLs) && i < 2; i++ {
		finalU, ctype, rawBody, err := fetchRawResource(ctx, client, xhrURLs[i], pageURL)
		if err != nil {
			continue
		}
		xhrTables := parseStructuredResponseToTables(finalU, ctype, rawBody, headersHint)
		mXHR, aXHR := filterMatching(xhrTables)
		if len(mXHR) > 0 {
			return finalU, mXHR, aXHR
		}
		fallbackTables = append(fallbackTables, aXHR...)
	}

	// 3. Links de exportação direta (CSV / JSON)
	if expURL, expTables := extractExportLinkTables(ctx, client, pageURL, body, domain, headersHint); len(expTables) > 0 {
		mExp, aExp := filterMatching(expTables)
		if len(mExp) > 0 {
			return expURL, mExp, aExp
		}
		fallbackTables = append(fallbackTables, aExp...)
	}

	// 4. Submissão automática de <form> padrão da página
	if formURL, formTables, err := submitBestDomainForm(ctx, client, pageURL, body, domain, params); err == nil && len(formTables) > 0 {
		mForm, aForm := filterMatching(formTables)
		if len(mForm) > 0 {
			return formURL, mForm, aForm
		}
		if len(fallbackTables) == 0 {
			usedURL = formURL
		}
		fallbackTables = append(fallbackTables, aForm...)
	}

	return usedURL, nil, fallbackTables
}
