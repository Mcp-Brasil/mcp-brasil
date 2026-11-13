package transparencia

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Betha Sistemas (transparencia.betha.cloud) Connector
// ---------------------------------------------------------------------------

const (
	defaultBethaOAuthURL   = "https://plataforma-oauth.betha.cloud/auth/oauth2/authorize"
	defaultBethaAPIBaseURL = "https://api.transparencia.betha.cloud/transparencia"
	bethaClientID          = "91a97459-f1d8-4b29-b5fa-2e51d1692623"
	bethaRedirectURI       = "https://transparencia.betha.cloud/auth-callback.html"
	bethaPortalFrontendURL = "https://transparencia.betha.cloud/#/"
)

var (
	bethaURLMu      sync.RWMutex
	bethaOAuthURL   = defaultBethaOAuthURL
	bethaAPIBaseURL = defaultBethaAPIBaseURL
)

// SetBethaOAuthURL configura a URL do endpoint OAuth2 da Betha (útil para testes).
func SetBethaOAuthURL(u string) {
	bethaURLMu.Lock()
	bethaOAuthURL = strings.TrimRight(strings.TrimSpace(u), "/")
	bethaURLMu.Unlock()
	ResetBethaTokenCache()
}

// SetBethaAPIBaseURL configura a URL base da API Betha Transparência (útil para testes).
func SetBethaAPIBaseURL(u string) {
	bethaURLMu.Lock()
	bethaAPIBaseURL = strings.TrimRight(strings.TrimSpace(u), "/")
	bethaURLMu.Unlock()
}

func getBethaOAuthURL() string {
	bethaURLMu.RLock()
	defer bethaURLMu.RUnlock()
	return bethaOAuthURL
}

func getBethaAPIBaseURL() string {
	bethaURLMu.RLock()
	defer bethaURLMu.RUnlock()
	return bethaAPIBaseURL
}

type bethaTokenCache struct {
	mu        sync.Mutex
	oauthURL  string
	token     string
	expiresAt time.Time
}

var bethaToken = &bethaTokenCache{}

// ResetBethaTokenCache limpa o cache de token OAuth2 anônimo da Betha.
func ResetBethaTokenCache() {
	bethaToken.mu.Lock()
	bethaToken.oauthURL = ""
	bethaToken.token = ""
	bethaToken.expiresAt = time.Time{}
	bethaToken.mu.Unlock()
}

func bethaHTTPClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func getBethaToken(ctx context.Context, client *http.Client, forceRefresh bool) (string, error) {
	oauthURL := getBethaOAuthURL()

	bethaToken.mu.Lock()
	if !forceRefresh && bethaToken.token != "" && bethaToken.oauthURL == oauthURL && time.Now().Before(bethaToken.expiresAt) {
		tok := bethaToken.token
		bethaToken.mu.Unlock()
		return tok, nil
	}
	bethaToken.mu.Unlock()

	authReqURL := fmt.Sprintf(
		"%s?response_type=token&client_id=%s&redirect_uri=%s&scope=transparencia.public&bth_ignore_origin=true&access_mode=anonymous",
		oauthURL,
		url.QueryEscape(bethaClientID),
		url.QueryEscape(bethaRedirectURI),
	)

	baseClient := bethaHTTPClient(client)
	noRedirectClient := *baseClient
	noRedirectClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, authReqURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	req.Header.Set("Accept", "text/html,application/json,*/*")

	resp, err := noRedirectClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("erro na requisição OAuth2 Betha: %w", err)
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	loc := strings.TrimSpace(resp.Header.Get("Location"))
	var accessToken string
	var expiresIn int

	if loc != "" {
		frag := ""
		if idx := strings.Index(loc, "#"); idx >= 0 {
			frag = loc[idx+1:]
		} else if parsed, pErr := url.Parse(loc); pErr == nil {
			frag = parsed.Fragment
			if frag == "" {
				frag = parsed.RawQuery
			}
		}
		if vals, qErr := url.ParseQuery(frag); qErr == nil {
			accessToken = strings.TrimSpace(vals.Get("access_token"))
			if expStr := strings.TrimSpace(vals.Get("expires_in")); expStr != "" {
				expiresIn, _ = strconv.Atoi(expStr)
			}
		}
	}

	if accessToken == "" && resp.StatusCode == http.StatusOK && len(bodyBytes) > 0 {
		var tokResp struct {
			AccessToken string `json:"access_token"`
			ExpiresIn   int    `json:"expires_in"`
		}
		if json.Unmarshal(bodyBytes, &tokResp) == nil && strings.TrimSpace(tokResp.AccessToken) != "" {
			accessToken = strings.TrimSpace(tokResp.AccessToken)
			expiresIn = tokResp.ExpiresIn
		}
	}

	if accessToken == "" {
		return "", fmt.Errorf("token OAuth2 Betha não retornado (HTTP %d, Location=%q)", resp.StatusCode, loc)
	}

	if expiresIn <= 0 {
		expiresIn = 3600
	}
	ttl := time.Duration(expiresIn) * time.Second
	if ttl > 60*time.Second {
		ttl -= 30 * time.Second
	} else if ttl > 5*time.Second {
		ttl -= 2 * time.Second
	}

	bethaToken.mu.Lock()
	bethaToken.oauthURL = oauthURL
	bethaToken.token = accessToken
	bethaToken.expiresAt = time.Now().Add(ttl)
	bethaToken.mu.Unlock()

	return accessToken, nil
}

func bethaAppContext(portalHash string) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(`{"portal":%q}`, portalHash)))
}

func doBethaRequest(ctx context.Context, client *http.Client, method, targetURL, portalHash string, payload any) ([]byte, int, error) {
	httpClient := bethaHTTPClient(client)

	attempt := func(forceRefresh bool) ([]byte, int, error) {
		token, err := getBethaToken(ctx, httpClient, forceRefresh)
		if err != nil {
			return nil, 0, err
		}

		var bodyReader io.Reader
		if payload != nil {
			raw, err := json.Marshal(payload)
			if err != nil {
				return nil, 0, err
			}
			bodyReader = bytes.NewReader(raw)
		}

		req, err := http.NewRequestWithContext(ctx, method, targetURL, bodyReader)
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
		req.Header.Set("Accept", "application/json")
		if portalHash != "" {
			req.Header.Set("App-Context", bethaAppContext(portalHash))
		}
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()

		b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		if err != nil {
			return nil, resp.StatusCode, err
		}
		return b, resp.StatusCode, nil
	}

	b, status, err := attempt(false)
	if err == nil && status == http.StatusUnauthorized {
		return attempt(true)
	}
	return b, status, err
}

type bethaPortalRaw struct {
	Hash       string `json:"hash"`
	ID         any    `json:"id"`
	PortalHash string `json:"portalHash"`
	Nome       string `json:"nome"`
	Descricao  string `json:"descricao"`
	CodigoIBGE any    `json:"codigoIbge"`
	CNPJ       string `json:"cnpj"`
}

func (p bethaPortalRaw) resolvedHash() string {
	for _, h := range []string{p.Hash, p.PortalHash, anyToString(p.ID)} {
		if t := strings.TrimSpace(h); t != "" {
			return t
		}
	}
	return ""
}

func (p bethaPortalRaw) resolvedNome() string {
	if t := strings.TrimSpace(p.Nome); t != "" {
		return t
	}
	return strings.TrimSpace(p.Descricao)
}

func decodeBethaPortais(body []byte) ([]bethaPortalRaw, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var list []bethaPortalRaw
		if err := json.Unmarshal([]byte(trimmed), &list); err != nil {
			return nil, err
		}
		return list, nil
	}
	var env struct {
		Content   []bethaPortalRaw `json:"content"`
		Registros []bethaPortalRaw `json:"registros"`
		Items     []bethaPortalRaw `json:"items"`
		Dados     []bethaPortalRaw `json:"dados"`
	}
	if err := json.Unmarshal([]byte(trimmed), &env); err != nil {
		return nil, err
	}
	for _, s := range [][]bethaPortalRaw{env.Content, env.Registros, env.Items, env.Dados} {
		if len(s) > 0 {
			return s, nil
		}
	}
	return nil, nil
}

// DiscoverBethaByIBGE consulta o diretório público da Betha Sistemas pelo código IBGE do município.
func DiscoverBethaByIBGE(ctx context.Context, client *http.Client, m *Municipio) (*PortalDiscoveryResult, bool) {
	if m == nil || strings.TrimSpace(m.CodigoIBGE) == "" {
		return nil, false
	}
	ibge := strings.TrimSpace(m.CodigoIBGE)
	apiBase := getBethaAPIBaseURL()
	targetURL := fmt.Sprintf("%s/auth/portais?limit=5&filter=codigoIbge%%20=%%20%s", apiBase, url.QueryEscape(ibge))

	body, status, err := doBethaRequest(ctx, client, http.MethodGet, targetURL, "", nil)
	if err != nil || status != http.StatusOK {
		return nil, false
	}

	portais, err := decodeBethaPortais(body)
	if err != nil || len(portais) == 0 {
		return nil, false
	}

	var best *bethaPortalRaw
	for i := range portais {
		if portais[i].resolvedHash() == "" {
			continue
		}
		if best == nil {
			best = &portais[i]
		}
		nomeNorm := RemoveAccents(strings.ToLower(portais[i].resolvedNome()))
		if strings.Contains(nomeNorm, "prefeitura") || strings.Contains(nomeNorm, "municipio") {
			best = &portais[i]
			break
		}
	}
	if best == nil {
		return nil, false
	}

	hash := best.resolvedHash()
	nome := best.resolvedNome()
	if nome == "" {
		nome = fmt.Sprintf("%s (%s)", m.Nome, m.UF)
	}

	return &PortalDiscoveryResult{
		Municipio:           m,
		URLResolvida:        bethaPortalFrontendURL + hash,
		Plataforma:          "betha_cloud",
		TenantID:            hash,
		EntidadeNome:        nome,
		FonteDescoberta:     "betha_directory",
		ServidoresSuportado: true,
	}, true
}

func resolveBethaContext(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (apiBase string, portalHash string, portalURL string, err error) {
	apiBase = getBethaAPIBaseURL()
	if apiBase == defaultBethaAPIBaseURL && strings.TrimSpace(p.BaseURL) != "" {
		b := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
		if !strings.Contains(b, "transparencia.betha.cloud") && !strings.Contains(b, "#/") {
			apiBase = b
		}
	}

	portalHash = strings.TrimSpace(p.TenantID)
	if portalHash == "" && strings.Contains(p.BaseURL, "#/") {
		parts := strings.SplitN(p.BaseURL, "#/", 2)
		portalHash = strings.Trim(strings.TrimSpace(parts[1]), "/")
	}
	if portalHash == "" && m != nil && strings.TrimSpace(m.CodigoIBGE) != "" {
		if disc, ok := DiscoverBethaByIBGE(ctx, client, m); ok && disc != nil {
			portalHash = disc.TenantID
			portalURL = disc.URLResolvida
		}
	}
	if portalHash == "" {
		return "", "", "", fmt.Errorf("TenantID (portal hash) Betha não informado e não descoberto")
	}

	if portalURL == "" {
		if strings.Contains(p.BaseURL, "#/") {
			portalURL = strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
		} else {
			portalURL = bethaPortalFrontendURL + portalHash
		}
	}
	return apiBase, portalHash, portalURL, nil
}

type bethaConsultaRef struct {
	ID   int64  `json:"id"`
	Tipo string `json:"tipo"`
}

type bethaMenuRaw struct {
	ID         int64             `json:"id"`
	Titulo     string            `json:"titulo"`
	Nome       string            `json:"nome"`
	Descricao  string            `json:"descricao"`
	ConsultaID int64             `json:"consultaId"`
	IDConsulta int64             `json:"idConsulta"`
	Tipo       string            `json:"tipo"`
	TipoCons   string            `json:"tipoConsulta"`
	Consulta   *bethaConsultaRef `json:"consulta"`
	Itens      []bethaMenuRaw    `json:"itens"`
	Filhos     []bethaMenuRaw    `json:"filhos"`
	Children   []bethaMenuRaw    `json:"children"`
	Submenus   []bethaMenuRaw    `json:"submenus"`
}

type bethaMenuEntry struct {
	Titulo     string
	NormTitulo string
	ConsultaID int64
	Tipo       string
}

func flattenBethaMenu(nodes []bethaMenuRaw, out *[]bethaMenuEntry) {
	for _, n := range nodes {
		title := strings.TrimSpace(n.Titulo)
		if title == "" {
			title = strings.TrimSpace(n.Nome)
		}
		if title == "" {
			title = strings.TrimSpace(n.Descricao)
		}

		cid := n.ConsultaID
		if cid == 0 {
			cid = n.IDConsulta
		}
		if cid == 0 && n.Consulta != nil {
			cid = n.Consulta.ID
		}

		tipo := strings.TrimSpace(n.Tipo)
		if tipo == "" {
			tipo = strings.TrimSpace(n.TipoCons)
		}
		if tipo == "" && n.Consulta != nil {
			tipo = strings.TrimSpace(n.Consulta.Tipo)
		}

		childrenCount := len(n.Itens) + len(n.Filhos) + len(n.Children) + len(n.Submenus)
		if cid == 0 && childrenCount == 0 && n.ID > 0 && tipo != "" {
			cid = n.ID
		}

		if cid > 0 && title != "" {
			*out = append(*out, bethaMenuEntry{
				Titulo:     title,
				NormTitulo: RemoveAccents(strings.ToLower(title)),
				ConsultaID: cid,
				Tipo:       strings.ToUpper(tipo),
			})
		}

		if len(n.Itens) > 0 {
			flattenBethaMenu(n.Itens, out)
		}
		if len(n.Filhos) > 0 {
			flattenBethaMenu(n.Filhos, out)
		}
		if len(n.Children) > 0 {
			flattenBethaMenu(n.Children, out)
		}
		if len(n.Submenus) > 0 {
			flattenBethaMenu(n.Submenus, out)
		}
	}
}

func fetchBethaMenu(ctx context.Context, client *http.Client, apiBase, portalHash string) ([]bethaMenuEntry, error) {
	u := apiBase + "/api/menu"
	body, status, err := doBethaRequest(ctx, client, http.MethodGet, u, portalHash, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Betha /api/menu retornou HTTP %d", status)
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, nil
	}

	var nodes []bethaMenuRaw
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &nodes); err != nil {
			return nil, fmt.Errorf("erro ao decodificar /api/menu Betha: %w", err)
		}
	} else {
		var env struct {
			Content []bethaMenuRaw `json:"content"`
			Itens   []bethaMenuRaw `json:"itens"`
			Menus   []bethaMenuRaw `json:"menus"`
			Dados   []bethaMenuRaw `json:"dados"`
		}
		if err := json.Unmarshal([]byte(trimmed), &env); err != nil {
			return nil, fmt.Errorf("erro ao decodificar /api/menu Betha: %w", err)
		}
		for _, s := range [][]bethaMenuRaw{env.Content, env.Itens, env.Menus, env.Dados} {
			if len(s) > 0 {
				nodes = s
				break
			}
		}
	}

	var out []bethaMenuEntry
	flattenBethaMenu(nodes, &out)
	return out, nil
}

func matchBethaMenu(entries []bethaMenuEntry, primaryKeywords, secondaryKeywords, excludeKeywords []string) (bethaMenuEntry, bool) {
	bestScore := -1
	var best bethaMenuEntry

	for _, e := range entries {
		kwScore := 0
		for idx, kw := range primaryKeywords {
			if strings.Contains(e.NormTitulo, kw) {
				kwScore = 1000 - idx*10
				break
			}
		}
		if kwScore == 0 {
			for idx, kw := range secondaryKeywords {
				if strings.Contains(e.NormTitulo, kw) {
					kwScore = 500 - idx*10
					break
				}
			}
		}
		if kwScore == 0 {
			continue
		}

		for _, ex := range excludeKeywords {
			if strings.Contains(e.NormTitulo, ex) {
				kwScore -= 200
			}
		}

		// Prefer CONSULTA_PADRAO over AGRUPADOR / LISTA_EM_ARVORE
		typeBonus := 0
		switch e.Tipo {
		case "CONSULTA_PADRAO", "":
			typeBonus = 2000
		case "AGRUPADOR", "LISTA_EM_ARVORE":
			typeBonus = 100
		}

		totalScore := typeBonus + kwScore
		if totalScore > bestScore {
			bestScore = totalScore
			best = e
		}
	}

	return best, bestScore >= 0
}

type bethaAgrupadorRaw struct {
	ID          any    `json:"id"`
	AgrupadorID any    `json:"agrupadorId"`
	Codigo      any    `json:"codigo"`
	Titulo      string `json:"titulo"`
	Nome        string `json:"nome"`
	Descricao   string `json:"descricao"`
}

func (a bethaAgrupadorRaw) resolvedID() string {
	for _, v := range []any{a.ID, a.AgrupadorID, a.Codigo} {
		if s := strings.TrimSpace(anyToString(v)); s != "" && s != "0" {
			return s
		}
	}
	return ""
}

func (a bethaAgrupadorRaw) resolvedTitle() string {
	for _, s := range []string{a.Titulo, a.Nome, a.Descricao} {
		if t := strings.TrimSpace(s); t != "" {
			return t
		}
	}
	return ""
}

func decodeBethaAgrupadores(body []byte) ([]bethaAgrupadorRaw, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var list []bethaAgrupadorRaw
		if err := json.Unmarshal([]byte(trimmed), &list); err != nil {
			return nil, err
		}
		return list, nil
	}
	var env struct {
		Content []bethaAgrupadorRaw `json:"content"`
		Dados   []bethaAgrupadorRaw `json:"dados"`
		Items   []bethaAgrupadorRaw `json:"items"`
	}
	if err := json.Unmarshal([]byte(trimmed), &env); err != nil {
		return nil, err
	}
	for _, s := range [][]bethaAgrupadorRaw{env.Content, env.Dados, env.Items} {
		if len(s) > 0 {
			return s, nil
		}
	}
	return nil, nil
}

func decodeBethaRows(body []byte) ([]map[string]any, int, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, 0, nil
	}

	flattenRow := func(row map[string]any) map[string]any {
		flat := make(map[string]any, len(row))
		for k, v := range row {
			flat[k] = v
		}
		for _, nestedKey := range []string{"valores", "campos"} {
			if raw, ok := row[nestedKey]; ok {
				switch typed := raw.(type) {
				case map[string]any:
					for k, v := range typed {
						if _, exists := flat[k]; !exists {
							flat[k] = v
						}
					}
				case []any:
					for _, el := range typed {
						if m, ok := el.(map[string]any); ok {
							k := mapGetStr(m, "campo", "nome", "titulo", "chave", "id")
							if k != "" {
								if val, hasVal := m["valor"]; hasVal {
									if _, exists := flat[k]; !exists {
										flat[k] = val
									}
								}
							}
						}
					}
				}
			}
		}
		return flat
	}

	if strings.HasPrefix(trimmed, "[") {
		var rows []map[string]any
		if err := json.Unmarshal([]byte(trimmed), &rows); err != nil {
			return nil, 0, err
		}
		out := make([]map[string]any, len(rows))
		for i, r := range rows {
			out[i] = flattenRow(r)
		}
		return out, len(out), nil
	}

	var obj map[string]any
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil {
		return nil, 0, err
	}

	var rows []map[string]any
	for _, key := range []string{"content", "dados", "registros", "items", "linhas", "resultados", "data"} {
		if raw, ok := obj[key]; ok {
			if arr, ok := raw.([]any); ok {
				rows = make([]map[string]any, 0, len(arr))
				for _, el := range arr {
					if m, ok := el.(map[string]any); ok {
						rows = append(rows, flattenRow(m))
					}
				}
				break
			}
		}
	}

	total := int(mapGetFloat(obj, "total", "totalElements", "totalRegistros", "count", "totalCount"))
	if total == 0 && len(rows) > 0 {
		total = len(rows)
	}
	return rows, total, nil
}

func executeBethaConsulta(ctx context.Context, client *http.Client, apiBase, portalHash string, entry bethaMenuEntry, p PortalQueryParams) ([]map[string]any, int, error) {
	limit := normalizeLimit(p.Limit)
	payload := map[string]any{}
	if filtro := strings.TrimSpace(p.NomeFiltro); filtro != "" {
		payload["texto"] = filtro
	}

	if entry.Tipo == "AGRUPADOR" || entry.Tipo == "LISTA_EM_ARVORE" {
		agrupURL := fmt.Sprintf("%s/api/consulta/%d/agrupadores/nivel/1", apiBase, entry.ConsultaID)
		body, status, err := doBethaRequest(ctx, client, http.MethodGet, agrupURL, portalHash, nil)
		if err != nil {
			return nil, 0, err
		}
		if status != http.StatusOK {
			return nil, 0, fmt.Errorf("Betha agrupadores retornou HTTP %d", status)
		}
		agrupadores, err := decodeBethaAgrupadores(body)
		if err != nil {
			return nil, 0, fmt.Errorf("erro ao decodificar agrupadores Betha: %w", err)
		}
		if len(agrupadores) == 0 {
			return nil, 0, nil
		}

		chosenID := agrupadores[0].resolvedID()
		if p.Ano > 0 {
			yearStr := strconv.Itoa(p.Ano)
			for _, ag := range agrupadores {
				if strings.Contains(ag.resolvedTitle(), yearStr) || strings.Contains(ag.resolvedID(), yearStr) {
					if id := ag.resolvedID(); id != "" {
						chosenID = id
						break
					}
				}
			}
		}
		if chosenID == "" {
			return nil, 0, nil
		}

		buscaURL := fmt.Sprintf("%s/api/busca/%d/agrupador/%s?offset=0&nivel=1&limit=%d", apiBase, entry.ConsultaID, url.PathEscape(chosenID), limit)
		resBody, resStatus, err := doBethaRequest(ctx, client, http.MethodPost, buscaURL, portalHash, payload)
		if err != nil {
			return nil, 0, err
		}
		if resStatus != http.StatusOK {
			return nil, 0, fmt.Errorf("Betha /api/busca/%d/agrupador/%s retornou HTTP %d", entry.ConsultaID, chosenID, resStatus)
		}
		return decodeBethaRows(resBody)
	}

	buscaTextualURL := fmt.Sprintf("%s/api/busca-textual/%d?offset=0&limit=%d", apiBase, entry.ConsultaID, limit)
	resBody, resStatus, err := doBethaRequest(ctx, client, http.MethodPost, buscaTextualURL, portalHash, payload)
	if err != nil {
		return nil, 0, err
	}
	if resStatus != http.StatusOK {
		return nil, 0, fmt.Errorf("Betha /api/busca-textual/%d retornou HTTP %d", entry.ConsultaID, resStatus)
	}
	return decodeBethaRows(resBody)
}

func normFieldKey(s string) string {
	clean := RemoveAccents(strings.ToLower(strings.TrimSpace(s)))
	return nonAlphaNumRegex.ReplaceAllString(clean, "")
}

func bethaGetStr(row map[string]any, keys ...string) string {
	if v := mapGetStr(row, keys...); v != "" {
		return v
	}
	if len(row) == 0 {
		return ""
	}
	normIndex := make(map[string]any, len(row))
	for k, v := range row {
		nk := normFieldKey(k)
		if _, exists := normIndex[nk]; !exists {
			normIndex[nk] = v
		}
	}
	for _, k := range keys {
		if v, ok := normIndex[normFieldKey(k)]; ok && v != nil {
			if s := strings.TrimSpace(anyToString(v)); s != "" {
				return s
			}
		}
	}
	return ""
}

func bethaGetFloat(row map[string]any, keys ...string) float64 {
	if v := mapGetFloat(row, keys...); v != 0 {
		return v
	}
	if len(row) == 0 {
		return 0
	}
	normIndex := make(map[string]any, len(row))
	for k, v := range row {
		nk := normFieldKey(k)
		if _, exists := normIndex[nk]; !exists {
			normIndex[nk] = v
		}
	}
	for _, k := range keys {
		if v, ok := normIndex[normFieldKey(k)]; ok && v != nil {
			switch t := v.(type) {
			case float64:
				if t != 0 {
					return t
				}
			case string:
				if f, err := parseBRFloat(t); err == nil && f != 0 {
					return f
				}
			}
		}
	}
	return 0
}

func maskDocIfCPF(doc string) string {
	trimmed := strings.TrimSpace(doc)
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "*") {
		return trimmed
	}
	if len(cleanDigits(trimmed)) == 11 {
		return maskCPF("", trimmed)
	}
	return trimmed
}

func parseBethaInt(s string, fallback int) int {
	digits := cleanDigits(s)
	if digits == "" {
		return fallback
	}
	if v, err := strconv.Atoi(digits); err == nil && v > 0 {
		return v
	}
	return fallback
}

// QueryBethaServidores consulta servidores públicos no portal Betha Cloud.
func QueryBethaServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ServidoresResult, error) {
	apiBase, portalHash, portalURL, err := resolveBethaContext(ctx, client, m, p)
	if err != nil {
		return nil, err
	}
	entries, err := fetchBethaMenu(ctx, client, apiBase, portalHash)
	if err != nil {
		return nil, err
	}
	entry, ok := matchBethaMenu(entries, []string{"servidor", "folha", "remuneracao"}, []string{"pessoal", "vencimento", "salario"}, nil)
	if !ok {
		return &ServidoresResult{
			Municipio:  m,
			Fonte:      "Portal Municipal (Betha Cloud)",
			Plataforma: "betha_cloud",
			PortalURL:  portalURL,
			Servidores: []ServidorItem{},
			Disponivel: false,
			Mensagem:   "Consulta de servidores não localizada no menu do portal Betha.",
		}, nil
	}

	rows, total, err := executeBethaConsulta(ctx, client, apiBase, portalHash, entry, p)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	items := make([]ServidorItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		if len(items) >= limit {
			break
		}
		bruto := bethaGetFloat(r, "salarioBruto", "valorBruto", "bruto", "totalProventos", "proventos", "remuneracao", "vencimentos")
		descontos := bethaGetFloat(r, "descontos", "totalDescontos", "valorDescontos")
		liquido := bethaGetFloat(r, "salarioLiquido", "valorLiquido", "liquido")
		if liquido == 0 && bruto > 0 && descontos > 0 {
			liquido = bruto - descontos
		}
		items = append(items, ServidorItem{
			Nome:           bethaGetStr(r, "nome", "nomeServidor", "servidor", "funcionario"),
			Matricula:      bethaGetStr(r, "matricula", "registro", "codigo"),
			CPFParcial:     maskCPF(bethaGetStr(r, "cpfFormatado", "cpf_formatado"), bethaGetStr(r, "cpf", "documento", "cpfCnpj", "cpfServidor")),
			Cargo:          bethaGetStr(r, "cargo", "descricaoCargo", "funcaoCargo"),
			Funcao:         bethaGetStr(r, "funcao", "descricaoFuncao"),
			Lotacao:        bethaGetStr(r, "lotacao", "descricaoLotacao", "departamento", "setor"),
			LocalTrabalho:  bethaGetStr(r, "localTrabalho", "local"),
			Vinculo:        bethaGetStr(r, "vinculo", "tipoVinculo", "regime"),
			Situacao:       bethaGetStr(r, "situacao", "status"),
			SalarioBase:    bethaGetFloat(r, "salarioBase", "vencimentoBase", "base"),
			SalarioBruto:   bruto,
			Descontos:      descontos,
			SalarioLiquido: liquido,
			Competencia:    bethaGetStr(r, "competencia", "mesAno", "referencia"),
			CargaHoraria:   bethaGetStr(r, "cargaHoraria", "horasSemanais"),
			DataAdmissao:   bethaGetStr(r, "dataAdmissao", "admissao"),
			Orgao:          bethaGetStr(r, "orgao", "entidade", "unidade"),
			Fonte:          "Portal Municipal (Betha Cloud)",
			PortalURL:      portalURL,
		})
	}

	return &ServidoresResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Betha Cloud)",
		Plataforma:     "betha_cloud",
		PortalURL:      portalURL,
		TotalRegistros: total,
		Servidores:     items,
		Disponivel:     true,
	}, nil
}

// QueryBethaDespesas consulta despesas e empenhos no portal Betha Cloud.
func QueryBethaDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	apiBase, portalHash, portalURL, err := resolveBethaContext(ctx, client, m, p)
	if err != nil {
		return nil, err
	}
	entries, err := fetchBethaMenu(ctx, client, apiBase, portalHash)
	if err != nil {
		return nil, err
	}
	entry, ok := matchBethaMenu(entries, []string{"despesa", "empenho"}, []string{"gasto", "pagamento"}, []string{"diaria"})
	if !ok {
		return &DespesasResult{
			Municipio:  m,
			Fonte:      "Portal Municipal (Betha Cloud)",
			Plataforma: "betha_cloud",
			PortalURL:  portalURL,
			Despesas:   []DespesaItem{},
			Disponivel: false,
			Mensagem:   "Consulta de despesas não localizada no menu do portal Betha.",
		}, nil
	}

	rows, total, err := executeBethaConsulta(ctx, client, apiBase, portalHash, entry, p)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	items := make([]DespesaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		if len(items) >= limit {
			break
		}
		doc := bethaGetStr(r, "documentoFornecedor", "cpfCnpj", "cnpjCpf", "documento", "cpf", "cnpj", "credorDocumento")
		items = append(items, DespesaItem{
			NumeroEmpenho:       bethaGetStr(r, "numeroEmpenho", "empenho", "numero"),
			Ano:                 parseBethaInt(bethaGetStr(r, "ano", "exercicio"), p.Ano),
			Data:                bethaGetStr(r, "data", "dataEmpenho", "dataMovimento"),
			Fornecedor:          bethaGetStr(r, "fornecedor", "credor", "nomeCredor", "nomeFornecedor", "razaoSocial"),
			DocumentoFornecedor: maskDocIfCPF(doc),
			Orgao:               bethaGetStr(r, "orgao", "nomeOrgao"),
			Unidade:             bethaGetStr(r, "unidade", "unidadeOrcamentaria"),
			Funcao:              bethaGetStr(r, "funcao", "funcaoGoverno"),
			Elemento:            bethaGetStr(r, "elemento", "elementoDespesa", "natureza", "fonteRecurso"),
			Historico:           bethaGetStr(r, "historico", "descricao", "objeto"),
			ValorEmpenhado:      bethaGetFloat(r, "valorEmpenhado", "empenhado", "valor"),
			ValorLiquidado:      bethaGetFloat(r, "valorLiquidado", "liquidado"),
			ValorPago:           bethaGetFloat(r, "valorPago", "pago"),
			Fonte:               "Portal Municipal (Betha Cloud)",
		})
	}

	return &DespesasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Betha Cloud)",
		Plataforma:     "betha_cloud",
		PortalURL:      portalURL,
		TotalRegistros: total,
		Despesas:       items,
		Disponivel:     true,
	}, nil
}

// QueryBethaReceitas consulta receitas públicas no portal Betha Cloud.
func QueryBethaReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	apiBase, portalHash, portalURL, err := resolveBethaContext(ctx, client, m, p)
	if err != nil {
		return nil, err
	}
	entries, err := fetchBethaMenu(ctx, client, apiBase, portalHash)
	if err != nil {
		return nil, err
	}
	entry, ok := matchBethaMenu(entries, []string{"receita"}, []string{"arrecadacao"}, nil)
	if !ok {
		return &ReceitasResult{
			Municipio:  m,
			Fonte:      "Portal Municipal (Betha Cloud)",
			Plataforma: "betha_cloud",
			PortalURL:  portalURL,
			Receitas:   []ReceitaItem{},
			Disponivel: false,
			Mensagem:   "Consulta de receitas não localizada no menu do portal Betha.",
		}, nil
	}

	rows, total, err := executeBethaConsulta(ctx, client, apiBase, portalHash, entry, p)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		if len(items) >= limit {
			break
		}
		items = append(items, ReceitaItem{
			Codigo:          bethaGetStr(r, "codigo", "conta", "classificacao", "codigoReceita"),
			Descricao:       bethaGetStr(r, "descricao", "especificacao", "receita", "titulo"),
			Categoria:       bethaGetStr(r, "categoria", "fonteRecurso", "origem"),
			Data:            bethaGetStr(r, "data", "dataArrecadacao", "competencia"),
			ValorPrevisto:   bethaGetFloat(r, "valorPrevisto", "previsto", "orcado", "valorOrcado"),
			ValorArrecadado: bethaGetFloat(r, "valorArrecadado", "arrecadado", "realizado", "valorRealizado", "valor"),
			Fonte:           "Portal Municipal (Betha Cloud)",
		})
	}

	return &ReceitasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Betha Cloud)",
		Plataforma:     "betha_cloud",
		PortalURL:      portalURL,
		TotalRegistros: total,
		Receitas:       items,
		Disponivel:     true,
	}, nil
}

// QueryBethaLicitacoes consulta licitações no portal Betha Cloud.
func QueryBethaLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LicitacoesResult, error) {
	apiBase, portalHash, portalURL, err := resolveBethaContext(ctx, client, m, p)
	if err != nil {
		return nil, err
	}
	entries, err := fetchBethaMenu(ctx, client, apiBase, portalHash)
	if err != nil {
		return nil, err
	}
	entry, ok := matchBethaMenu(entries, []string{"licitacao", "licitacoes"}, []string{"pregao", "edital", "compra"}, nil)
	if !ok {
		return &LicitacoesResult{
			Municipio:  m,
			Fonte:      "Portal Municipal (Betha Cloud)",
			Plataforma: "betha_cloud",
			PortalURL:  portalURL,
			Licitacoes: []LicitacaoItem{},
			Disponivel: false,
			Mensagem:   "Consulta de licitações não localizada no menu do portal Betha.",
		}, nil
	}

	rows, total, err := executeBethaConsulta(ctx, client, apiBase, portalHash, entry, p)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	items := make([]LicitacaoItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		if len(items) >= limit {
			break
		}
		link := bethaGetStr(r, "link", "url")
		if link == "" {
			link = portalURL
		}
		items = append(items, LicitacaoItem{
			Numero:          bethaGetStr(r, "numero", "numeroLicitacao", "licitacao", "processo"),
			Ano:             parseBethaInt(bethaGetStr(r, "ano", "exercicio"), p.Ano),
			Modalidade:      bethaGetStr(r, "modalidade", "descricaoModalidade"),
			Situacao:        bethaGetStr(r, "situacao", "status"),
			DataAbertura:    bethaGetStr(r, "dataAbertura", "abertura", "dataPublicacao", "data"),
			Objeto:          bethaGetStr(r, "objeto", "descricao"),
			ValorEstimado:   bethaGetFloat(r, "valorEstimado", "estimado", "valorMaximo", "valorPrevisto"),
			ValorHomologado: bethaGetFloat(r, "valorHomologado", "homologado", "valorVencedor", "valorTotal", "valor"),
			Link:            link,
			Fonte:           "Portal Municipal (Betha Cloud)",
		})
	}

	return &LicitacoesResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Betha Cloud)",
		Plataforma:     "betha_cloud",
		PortalURL:      portalURL,
		TotalRegistros: total,
		Licitacoes:     items,
		Disponivel:     true,
	}, nil
}

// QueryBethaContratos consulta contratos administrativos no portal Betha Cloud.
func QueryBethaContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ContratosResult, error) {
	apiBase, portalHash, portalURL, err := resolveBethaContext(ctx, client, m, p)
	if err != nil {
		return nil, err
	}
	entries, err := fetchBethaMenu(ctx, client, apiBase, portalHash)
	if err != nil {
		return nil, err
	}
	entry, ok := matchBethaMenu(entries, []string{"contrato"}, []string{"aditivo"}, nil)
	if !ok {
		return &ContratosResult{
			Municipio:  m,
			Fonte:      "Portal Municipal (Betha Cloud)",
			Plataforma: "betha_cloud",
			PortalURL:  portalURL,
			Contratos:  []ContratoItem{},
			Disponivel: false,
			Mensagem:   "Consulta de contratos não localizada no menu do portal Betha.",
		}, nil
	}

	rows, total, err := executeBethaConsulta(ctx, client, apiBase, portalHash, entry, p)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	items := make([]ContratoItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		if len(items) >= limit {
			break
		}
		doc := bethaGetStr(r, "documentoFornecedor", "cpfCnpj", "cnpjCpf", "documento", "cpf", "cnpj")
		items = append(items, ContratoItem{
			Numero:              bethaGetStr(r, "numero", "numeroContrato", "contrato"),
			Ano:                 parseBethaInt(bethaGetStr(r, "ano", "exercicio"), p.Ano),
			Fornecedor:          bethaGetStr(r, "fornecedor", "contratado", "credor", "nomeFornecedor", "razaoSocial"),
			DocumentoFornecedor: maskDocIfCPF(doc),
			Objeto:              bethaGetStr(r, "objeto", "descricao"),
			Situacao:            bethaGetStr(r, "situacao", "status"),
			DataAssinatura:      bethaGetStr(r, "dataAssinatura", "assinatura", "data"),
			VigenciaInicio:      bethaGetStr(r, "vigenciaInicio", "dataInicio", "inicioVigencia"),
			VigenciaFim:         bethaGetStr(r, "vigenciaFim", "dataFim", "fimVigencia", "vencimento"),
			Valor:               bethaGetFloat(r, "valor", "valorContrato", "valorGlobal", "valorTotal"),
			Fonte:               "Portal Municipal (Betha Cloud)",
		})
	}

	return &ContratosResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Betha Cloud)",
		Plataforma:     "betha_cloud",
		PortalURL:      portalURL,
		TotalRegistros: total,
		Contratos:      items,
		Disponivel:     true,
	}, nil
}

// QueryBethaDiarias consulta diárias e passagens no portal Betha Cloud.
func QueryBethaDiarias(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DiariasResult, error) {
	apiBase, portalHash, portalURL, err := resolveBethaContext(ctx, client, m, p)
	if err != nil {
		return nil, err
	}
	entries, err := fetchBethaMenu(ctx, client, apiBase, portalHash)
	if err != nil {
		return nil, err
	}
	entry, ok := matchBethaMenu(entries, []string{"diaria"}, []string{"passagem", "viagem"}, nil)
	if !ok {
		return &DiariasResult{
			Municipio:  m,
			Fonte:      "Portal Municipal (Betha Cloud)",
			Plataforma: "betha_cloud",
			PortalURL:  portalURL,
			Diarias:    []DiariaItem{},
			Disponivel: false,
			Mensagem:   "Consulta de diárias não localizada no menu do portal Betha.",
		}, nil
	}

	rows, total, err := executeBethaConsulta(ctx, client, apiBase, portalHash, entry, p)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	items := make([]DiariaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		if len(items) >= limit {
			break
		}
		items = append(items, DiariaItem{
			Beneficiario:  bethaGetStr(r, "beneficiario", "servidor", "nome", "favorecido"),
			Matricula:     bethaGetStr(r, "matricula", "registro"),
			Cargo:         bethaGetStr(r, "cargo", "funcao"),
			Destino:       bethaGetStr(r, "destino", "localDestino", "cidadeDestino"),
			Motivo:        bethaGetStr(r, "motivo", "objetivo", "finalidade", "historico", "descricao"),
			Data:          bethaGetStr(r, "data", "dataPagamento", "dataEmissao"),
			DataSaida:     bethaGetStr(r, "dataSaida", "saida", "inicio"),
			DataChegada:   bethaGetStr(r, "dataChegada", "chegada", "retorno", "fim"),
			NumeroDiarias: bethaGetFloat(r, "numeroDiarias", "quantidade", "qtdDiarias", "dias"),
			Valor:         bethaGetFloat(r, "valor", "valorTotal", "valorPago"),
			Fonte:         "Portal Municipal (Betha Cloud)",
		})
	}

	return &DiariasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Betha Cloud)",
		Plataforma:     "betha_cloud",
		PortalURL:      portalURL,
		TotalRegistros: total,
		Diarias:        items,
		Disponivel:     true,
	}, nil
}
