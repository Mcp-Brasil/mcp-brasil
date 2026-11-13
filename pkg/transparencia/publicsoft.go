package transparencia

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Publicsoft (plataforma = "publicsoft_api")
// Base padrão: https://portaldatransparencia.publicsoft.com.br
// ---------------------------------------------------------------------------

const defaultPublicsoftBaseURL = "https://portaldatransparencia.publicsoft.com.br"

var (
	publicsoftBaseMu  sync.RWMutex
	publicsoftBaseURL = defaultPublicsoftBaseURL

	publicsoftLegadoRegex = regexp.MustCompile(`(?i)/sistemas/ContabilidadePublica/([^/?#\s]+)`)
	publicsoftHashRegex   = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)
)

// PublicsoftUG representa uma unidade gestora retornada pelo catálogo da Publicsoft.
type PublicsoftUG struct {
	ID        int    `json:"id"`
	Nome      string `json:"nome"`
	HashID    string `json:"hash_id"`
	Municipio string `json:"municipio,omitempty"`
	UF        string `json:"uf,omitempty"`
}

// PublicsoftEstado representa um estado retornado por /api/v1/catalogo/estados.
type PublicsoftEstado struct {
	ID   int    `json:"id,omitempty"`
	UF   string `json:"uf"`
	Nome string `json:"nome"`
}

// PublicsoftMunicipio representa um município retornado por /api/v1/catalogo/municipios.
type PublicsoftMunicipio struct {
	ID         int    `json:"id,omitempty"`
	Nome       string `json:"nome"`
	UF         string `json:"uf,omitempty"`
	CodigoIBGE string `json:"codigo_ibge,omitempty"`
}

// SetPublicsoftBaseURL configura a URL base da API Publicsoft (útil para testes).
func SetPublicsoftBaseURL(u string) {
	publicsoftBaseMu.Lock()
	defer publicsoftBaseMu.Unlock()
	trimmed := strings.TrimRight(strings.TrimSpace(u), "/")
	if trimmed == "" {
		publicsoftBaseURL = defaultPublicsoftBaseURL
		return
	}
	publicsoftBaseURL = trimmed
}

func getPublicsoftBaseURL() string {
	publicsoftBaseMu.RLock()
	defer publicsoftBaseMu.RUnlock()
	return strings.TrimRight(strings.TrimSpace(publicsoftBaseURL), "/")
}

func resolvePublicsoftBaseURL(rawBaseURL string) string {
	configured := getPublicsoftBaseURL()
	raw := strings.TrimSpace(rawBaseURL)
	if raw == "" {
		return configured
	}
	if idx := strings.Index(raw, "#"); idx >= 0 {
		raw = raw[:idx]
	}
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return configured
	}

	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return configured
	}

	host := strings.ToLower(u.Hostname())
	if strings.Contains(host, "publicsoft.com.br") {
		if configured != defaultPublicsoftBaseURL {
			return configured
		}
		return u.Scheme + "://" + u.Host
	}

	p := strings.TrimRight(u.Path, "/")
	lowP := strings.ToLower(p)
	if idx := strings.Index(lowP, "/sistemas/"); idx >= 0 {
		p = p[:idx]
	} else if idx := strings.Index(lowP, "/api/"); idx >= 0 {
		p = p[:idx]
	}
	return strings.TrimRight(u.Scheme+"://"+u.Host+p, "/")
}

func isPublicsoftHashID(s string) bool {
	return publicsoftHashRegex.MatchString(strings.TrimSpace(s))
}

func extractPublicsoftLegadoSlug(slugOrURL string) string {
	trimmed := strings.TrimSpace(slugOrURL)
	if trimmed == "" {
		return ""
	}
	if m := publicsoftLegadoRegex.FindStringSubmatch(trimmed); len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	if !strings.Contains(trimmed, "://") && !strings.Contains(trimmed, "/") && !isPublicsoftHashID(trimmed) {
		return trimmed
	}
	return ""
}

func doPublicsoftGet(ctx context.Context, client *http.Client, targetURL, hashID string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Stage", "prod")
	if strings.TrimSpace(hashID) != "" {
		req.Header.Set("X-Hash-Id", strings.TrimSpace(hashID))
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

func decodePublicsoftPaginatedList(body []byte, extraKeys ...string) ([]map[string]any, int, error) {
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

	if dataObj, ok := obj["data"].(map[string]any); ok && dataObj != nil {
		keys := append([]string{"data", "items", "registros", "rows"}, extraKeys...)
		var rows []map[string]any
		for _, k := range keys {
			if raw, exists := dataObj[k]; exists && raw != nil {
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
		total := int(mapGetFloat(dataObj, "total", "totalElements", "recordsTotal", "count"))
		if total == 0 {
			total = int(mapGetFloat(obj, "total", "totalElements", "recordsTotal", "count"))
		}
		if total == 0 {
			total = len(rows)
		}
		return rows, total, nil
	}

	keys := append([]string{"data"}, extraKeys...)
	return decodeFlexiblePortalList(body, keys...)
}

func extractPublicsoftNestedStr(m map[string]any, nestedKey string, keys ...string) string {
	if sub, ok := m[nestedKey].(map[string]any); ok && sub != nil {
		return mapGetStr(sub, keys...)
	}
	return ""
}

func mapGetStringOnly(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			if s, isStr := v.(string); isStr {
				if trimmed := strings.TrimSpace(s); trimmed != "" {
					return trimmed
				}
			}
		}
	}
	return ""
}

// FetchPublicsoftUGs lista as unidades gestoras disponíveis em GET /api/v1/catalogo/ugs?com_instituicao=1.
func FetchPublicsoftUGs(ctx context.Context, client *http.Client) ([]PublicsoftUG, error) {
	base := getPublicsoftBaseURL()
	target := base + "/api/v1/catalogo/ugs?com_instituicao=1"
	body, status, err := doPublicsoftGet(ctx, client, target, "")
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Publicsoft /api/v1/catalogo/ugs retornou HTTP %d", status)
	}

	rows, _, err := decodePublicsoftPaginatedList(body, "ugs")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Publicsoft UGs: %w", err)
	}

	out := make([]PublicsoftUG, 0, len(rows))
	for _, r := range rows {
		hashID := mapGetStr(r, "hash_id", "hashId", "hash")
		nome := mapGetStringOnly(r, "nome", "razao_social", "descricao", "entidade")
		if nome == "" {
			nome = extractPublicsoftNestedStr(r, "instituicao", "nome", "razao_social", "descricao")
		}

		mun := mapGetStringOnly(r, "municipio", "municipio_nome", "cidade", "nome_municipio")
		if mun == "" {
			mun = extractPublicsoftNestedStr(r, "municipio", "nome", "municipio")
		}
		if mun == "" {
			if inst, ok := r["instituicao"].(map[string]any); ok && inst != nil {
				mun = mapGetStringOnly(inst, "municipio", "municipio_nome", "cidade")
				if mun == "" {
					mun = extractPublicsoftNestedStr(inst, "municipio", "nome")
				}
			}
		}

		uf := mapGetStringOnly(r, "uf", "estado_sigla", "sigla_uf", "estado")
		if uf == "" {
			uf = extractPublicsoftNestedStr(r, "estado", "sigla", "uf")
		}
		if uf == "" {
			if munMap, ok := r["municipio"].(map[string]any); ok && munMap != nil {
				uf = mapGetStringOnly(munMap, "uf", "estado_sigla")
				if uf == "" {
					uf = extractPublicsoftNestedStr(munMap, "estado", "sigla", "uf")
				}
			}
		}
		if uf == "" {
			if inst, ok := r["instituicao"].(map[string]any); ok && inst != nil {
				uf = mapGetStringOnly(inst, "uf", "estado_sigla", "sigla_uf")
				if uf == "" {
					if munMap, ok := inst["municipio"].(map[string]any); ok && munMap != nil {
						uf = mapGetStringOnly(munMap, "uf", "estado_sigla")
						if uf == "" {
							uf = extractPublicsoftNestedStr(munMap, "estado", "sigla", "uf")
						}
					}
				}
			}
		}

		id := int(mapGetFloat(r, "id", "ug_id", "codigo"))
		if hashID == "" && nome == "" && id == 0 {
			continue
		}
		out = append(out, PublicsoftUG{
			ID:        id,
			Nome:      nome,
			HashID:    hashID,
			Municipio: mun,
			UF:        strings.ToUpper(strings.TrimSpace(uf)),
		})
	}
	return out, nil
}

// FetchPublicsoftEstados lista os estados disponíveis em GET /api/v1/catalogo/estados.
func FetchPublicsoftEstados(ctx context.Context, client *http.Client) ([]PublicsoftEstado, error) {
	base := getPublicsoftBaseURL()
	target := base + "/api/v1/catalogo/estados"
	body, status, err := doPublicsoftGet(ctx, client, target, "")
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Publicsoft /api/v1/catalogo/estados retornou HTTP %d", status)
	}

	rows, _, err := decodePublicsoftPaginatedList(body, "estados")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Publicsoft estados: %w", err)
	}

	out := make([]PublicsoftEstado, 0, len(rows))
	for _, r := range rows {
		uf := strings.ToUpper(mapGetStr(r, "uf", "sigla", "UF", "Sigla", "estado"))
		if uf == "" {
			if rawID := strings.TrimSpace(mapGetStr(r, "id")); len(rawID) == 2 {
				uf = strings.ToUpper(rawID)
			}
		}
		nome := mapGetStr(r, "nome", "Nome", "descricao")
		id := int(mapGetFloat(r, "id", "codigo"))
		if uf == "" && nome == "" {
			continue
		}
		out = append(out, PublicsoftEstado{
			ID:   id,
			UF:   uf,
			Nome: nome,
		})
	}
	return out, nil
}

// FetchPublicsoftMunicipios lista os municípios disponíveis para uma UF em GET /api/v1/catalogo/municipios?estado=<UF>.
func FetchPublicsoftMunicipios(ctx context.Context, client *http.Client, uf string) ([]PublicsoftMunicipio, error) {
	ufNorm := strings.ToUpper(strings.TrimSpace(uf))
	if ufNorm == "" {
		return nil, fmt.Errorf("Publicsoft: UF não informada")
	}

	base := getPublicsoftBaseURL()
	target := fmt.Sprintf("%s/api/v1/catalogo/municipios?estado=%s", base, url.QueryEscape(ufNorm))
	body, status, err := doPublicsoftGet(ctx, client, target, "")
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Publicsoft /api/v1/catalogo/municipios retornou HTTP %d", status)
	}

	rows, _, err := decodePublicsoftPaginatedList(body, "municipios")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Publicsoft municipios: %w", err)
	}

	out := make([]PublicsoftMunicipio, 0, len(rows))
	for _, r := range rows {
		nome := mapGetStr(r, "nome", "Nome", "municipio", "descricao")
		if nome == "" {
			continue
		}
		rowUF := strings.ToUpper(mapGetStr(r, "uf", "UF", "estado_sigla", "sigla_uf"))
		if rowUF == "" {
			rowUF = ufNorm
		}
		out = append(out, PublicsoftMunicipio{
			ID:         int(mapGetFloat(r, "id", "codigo")),
			Nome:       nome,
			UF:         rowUF,
			CodigoIBGE: mapGetStr(r, "codigo_ibge", "codigoIbge", "ibge"),
		})
	}
	return out, nil
}

func resolvePublicsoftLegadoDetail(ctx context.Context, client *http.Client, baseURL, slugOrURL string) (hashID string, entNome string, err error) {
	slug := extractPublicsoftLegadoSlug(slugOrURL)
	if slug == "" {
		return "", "", fmt.Errorf("Publicsoft: slug legado não identificado em %q", slugOrURL)
	}

	base := resolvePublicsoftBaseURL(baseURL)
	target := fmt.Sprintf("%s/api/v1/portal/legado/%s/despesa-orcamentaria", base, url.PathEscape(slug))
	body, status, reqErr := doPublicsoftGet(ctx, client, target, "")
	if reqErr != nil {
		return "", "", reqErr
	}
	if status != http.StatusOK {
		return "", "", fmt.Errorf("Publicsoft /api/v1/portal/legado/%s/despesa-orcamentaria retornou HTTP %d", slug, status)
	}

	var obj map[string]any
	if decErr := json.Unmarshal(body, &obj); decErr != nil {
		return "", "", fmt.Errorf("erro ao decodificar legado Publicsoft: %w", decErr)
	}

	extractHashAndName := func(m map[string]any) (string, string) {
		if m == nil {
			return "", ""
		}
		h := mapGetStr(m, "hash_id", "hashId", "hash")
		n := mapGetStr(m, "entidade", "nome", "ug_nome", "nome_ug", "instituicao")
		if ugMap, ok := m["ug"].(map[string]any); ok && ugMap != nil {
			if h == "" {
				h = mapGetStr(ugMap, "hash_id", "hashId", "hash")
			}
			if n == "" {
				n = mapGetStr(ugMap, "nome", "razao_social", "descricao")
			}
		}
		return h, n
	}

	hashID, entNome = extractHashAndName(obj)
	if hashID == "" {
		if dataMap, ok := obj["data"].(map[string]any); ok {
			h, n := extractHashAndName(dataMap)
			hashID = h
			if entNome == "" {
				entNome = n
			}
		}
	}
	if hashID == "" {
		return "", "", fmt.Errorf("Publicsoft legado %q não retornou hash_id", slug)
	}
	return hashID, entNome, nil
}

// ResolvePublicsoftLegado extrai o slug legado de URLs /sistemas/ContabilidadePublica/<slug>/...
// e consulta GET /api/v1/portal/legado/<slug>/despesa-orcamentaria para obter o hash_id.
func ResolvePublicsoftLegado(ctx context.Context, client *http.Client, slugOrURL string) (string, error) {
	hashID, _, err := resolvePublicsoftLegadoDetail(ctx, client, slugOrURL, slugOrURL)
	return hashID, err
}

func validatePublicsoftHash(ctx context.Context, client *http.Client, baseURL, hashID string) (entNome string, ok bool) {
	base := resolvePublicsoftBaseURL(baseURL)
	ano := time.Now().Year()

	targetDesp := fmt.Sprintf("%s/api/v1/despesas/detalhe?exercicio=%d&per_page=1", base, ano)
	if body, status, err := doPublicsoftGet(ctx, client, targetDesp, hashID); err == nil && status == http.StatusOK {
		var obj map[string]any
		if json.Unmarshal(body, &obj) == nil && obj != nil {
			if !strings.EqualFold(mapGetStr(obj, "status"), "error") {
				return mapGetStr(obj, "entidade", "nome"), true
			}
		}
	}

	targetCfg := fmt.Sprintf("%s/api/v1/catalogo/ugs/%s/folha-config", base, url.PathEscape(hashID))
	if body, status, err := doPublicsoftGet(ctx, client, targetCfg, hashID); err == nil && status == http.StatusOK {
		var obj map[string]any
		if json.Unmarshal(body, &obj) == nil && obj != nil {
			if !strings.EqualFold(mapGetStr(obj, "status"), "error") {
				ent := mapGetStr(obj, "entidade", "nome")
				if ent == "" {
					ent = extractPublicsoftNestedStr(obj, "data", "entidade", "nome")
				}
				return ent, true
			}
		}
	}

	return "", false
}

func matchPublicsoftUGInCatalog(ugs []PublicsoftUG, uf, municipioNome string) (PublicsoftUG, bool) {
	targetSlug := Slugify(municipioNome)
	targetCompact := SlugifyCompact(municipioNome)
	if targetSlug == "" {
		return PublicsoftUG{}, false
	}
	ufNorm := strings.ToUpper(strings.TrimSpace(uf))

	var best PublicsoftUG
	var bestScore int

	for _, ug := range ugs {
		if ug.HashID == "" {
			continue
		}
		if ufNorm != "" && ug.UF != "" && !strings.EqualFold(ug.UF, ufNorm) {
			continue
		}

		munSlug := Slugify(ug.Municipio)
		munCompact := SlugifyCompact(ug.Municipio)
		nomeSlug := Slugify(ug.Nome)
		nomeCompact := SlugifyCompact(ug.Nome)

		matched := false
		if munSlug != "" {
			matched = munSlug == targetSlug || munCompact == targetCompact
		} else if nomeSlug != "" {
			matched = strings.Contains("-"+nomeSlug+"-", "-"+targetSlug+"-") || strings.Contains(nomeCompact, targetCompact)
		}
		if !matched {
			continue
		}

		score := 1
		upNome := strings.ToUpper(ug.Nome)
		if strings.Contains(upNome, "PREFEITURA") {
			score += 10
		} else if strings.Contains(upNome, "MUNICIPIO") && !strings.Contains(upNome, "CAMARA") {
			score += 5
		}
		if ufNorm != "" && strings.EqualFold(ug.UF, ufNorm) {
			score += 2
		}

		if score > bestScore {
			bestScore = score
			best = ug
		}
	}

	return best, bestScore > 0
}

// ProbePublicsoft resolve e valida um tenant Publicsoft por hash_id direto, URL legada
// (/sistemas/ContabilidadePublica/<slug>/...) ou matching no catálogo de UGs.
func ProbePublicsoft(ctx context.Context, client *http.Client, baseURLOrTenant, uf, municipioNome string) (hashID string, entNome string, ok bool) {
	cand := strings.TrimSpace(baseURLOrTenant)

	if isPublicsoftHashID(cand) {
		if ent, valid := validatePublicsoftHash(ctx, client, "", cand); valid {
			return cand, ent, true
		}
		if ugs, err := FetchPublicsoftUGs(ctx, client); err == nil {
			for _, ug := range ugs {
				if strings.EqualFold(ug.HashID, cand) {
					return ug.HashID, ug.Nome, true
				}
			}
		}
		return "", "", false
	}

	if publicsoftLegadoRegex.MatchString(cand) {
		h, ent, err := resolvePublicsoftLegadoDetail(ctx, client, cand, cand)
		if err == nil && h != "" {
			return h, ent, true
		}
	}

	if strings.TrimSpace(municipioNome) != "" {
		if ugs, err := FetchPublicsoftUGs(ctx, client); err == nil {
			if ug, matched := matchPublicsoftUGInCatalog(ugs, uf, municipioNome); matched {
				return ug.HashID, ug.Nome, true
			}
		}
	}

	return "", "", false
}

func resolvePublicsoftYear(p PortalQueryParams) int {
	if p.Ano > 0 {
		return p.Ano
	}
	if y := parseYearFromDateStr(p.Inicio); y > 0 {
		return y
	}
	return time.Now().Year()
}

func resolvePublicsoftTarget(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (baseURL, hashID string, err error) {
	baseURL = resolvePublicsoftBaseURL(p.BaseURL)

	rawTenant := strings.TrimSpace(p.TenantID)
	if rawTenant != "" {
		if publicsoftLegadoRegex.MatchString(rawTenant) {
			h, _, legErr := resolvePublicsoftLegadoDetail(ctx, client, baseURL, rawTenant)
			if legErr != nil {
				return "", "", legErr
			}
			return baseURL, h, nil
		}
		return baseURL, rawTenant, nil
	}

	if publicsoftLegadoRegex.MatchString(p.BaseURL) {
		h, _, legErr := resolvePublicsoftLegadoDetail(ctx, client, p.BaseURL, p.BaseURL)
		if legErr != nil {
			return "", "", legErr
		}
		return baseURL, h, nil
	}

	if m != nil && strings.TrimSpace(m.Nome) != "" {
		if h, _, ok := ProbePublicsoft(ctx, client, p.BaseURL, m.UF, m.Nome); ok && h != "" {
			return baseURL, h, nil
		}
	}

	return "", "", fmt.Errorf("Publicsoft: hash_id (X-Hash-Id) não informado")
}

// QueryPublicsoftDespesas consulta despesas detalhadas na API Publicsoft (/api/v1/despesas/detalhe).
func QueryPublicsoftDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	base, hashID, err := resolvePublicsoftTarget(ctx, client, m, p)
	if err != nil {
		return nil, err
	}

	ano := resolvePublicsoftYear(p)
	limit := normalizeLimit(p.Limit)
	target := fmt.Sprintf("%s/api/v1/despesas/detalhe?exercicio=%d&per_page=%d", base, ano, limit)

	body, status, err := doPublicsoftGet(ctx, client, target, hashID)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Publicsoft /api/v1/despesas/detalhe retornou HTTP %d", status)
	}

	rows, total, err := decodePublicsoftPaginatedList(body, "despesas")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Publicsoft despesas: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]DespesaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		fornecedor := mapGetStr(r, "credor", "fornecedor", "nome_credor", "nome_fornecedor", "razao_social")
		descricao := mapGetStr(r, "descricao", "especificacao")
		historico := mapGetStr(r, "historico", "historico_empenho")
		if historico == "" {
			historico = descricao
		}
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+historico+" "+descricao), filtro) {
			continue
		}

		itemAno := int(mapGetFloat(r, "exercicio", "ano", "ano_empenho"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, DespesaItem{
			NumeroEmpenho:       mapGetStr(r, "numero", "numero_empenho", "numeroEmpenho", "empenho", "num_empenho", "id"),
			Ano:                 itemAno,
			Data:                mapGetStr(r, "data", "data_empenho", "dataEmpenho", "dt_empenho"),
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(mapGetStr(r, "cpfCnpj", "cpf_cnpj", "cnpj_cpf", "cnpjCpf", "documento", "cpf", "cnpj")),
			Orgao:               mapGetStr(r, "orgao", "nome_orgao", "descricao_orgao", "unidade_gestora"),
			Unidade:             mapGetStr(r, "unidade", "unidade_orcamentaria", "nome_unidade"),
			Funcao:              mapGetStr(r, "funcao", "descricao_funcao", "nome_funcao"),
			Elemento:            mapGetStr(r, "elemento", "elemento_despesa", "descricao_elemento", "natureza_despesa"),
			Historico:           historico,
			ValorEmpenhado:      mapGetFloat(r, "valorEmpenhado", "valor_empenhado", "empenhado", "vl_empenhado", "valor"),
			ValorLiquidado:      mapGetFloat(r, "valorLiquidado", "valor_liquidado", "liquidado", "vl_liquidado"),
			ValorPago:           mapGetFloat(r, "valorPago", "valor_pago", "pago", "vl_pago"),
			Fonte:               "Portal Municipal (Publicsoft)",
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
		Fonte:          "Portal Municipal (Publicsoft)",
		Plataforma:     "publicsoft_api",
		PortalURL:      base,
		TotalRegistros: total,
		Despesas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryPublicsoftReceitas consulta receitas detalhadas na API Publicsoft (/api/v1/receitas/detalhe).
func QueryPublicsoftReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	base, hashID, err := resolvePublicsoftTarget(ctx, client, m, p)
	if err != nil {
		return nil, err
	}

	ano := resolvePublicsoftYear(p)
	limit := normalizeLimit(p.Limit)
	target := fmt.Sprintf("%s/api/v1/receitas/detalhe?exercicio=%d&per_page=%d", base, ano, limit)

	body, status, err := doPublicsoftGet(ctx, client, target, hashID)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Publicsoft /api/v1/receitas/detalhe retornou HTTP %d", status)
	}

	rows, total, err := decodePublicsoftPaginatedList(body, "receitas")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Publicsoft receitas: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		desc := mapGetStr(r, "descricao", "especificacao", "nome", "receita")
		if filtro != "" && !strings.Contains(strings.ToLower(desc), filtro) {
			continue
		}

		items = append(items, ReceitaItem{
			Codigo:          mapGetStr(r, "rubrica", "cod_receita_stn", "codigo", "codigo_receita", "cod_receita", "estrutural"),
			Descricao:       desc,
			Categoria:       mapGetStr(r, "categoria", "fonte", "fonte_recurso", "origem"),
			Data:            mapGetStr(r, "data", "data_arrecadacao", "competencia"),
			ValorPrevisto:   mapGetFloat(r, "valor_previsto", "previsto", "valorPrevisto", "orcado", "previsao"),
			ValorArrecadado: mapGetFloat(r, "arrecadado", "valor_arrecadado", "valorArrecadado", "realizado"),
			Fonte:           "Portal Municipal (Publicsoft)",
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
		Fonte:          "Portal Municipal (Publicsoft)",
		Plataforma:     "publicsoft_api",
		PortalURL:      base,
		TotalRegistros: total,
		Receitas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryPublicsoftContratos consulta contratos na API Publicsoft (/api/v1/contratos).
func QueryPublicsoftContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ContratosResult, error) {
	base, hashID, err := resolvePublicsoftTarget(ctx, client, m, p)
	if err != nil {
		return nil, err
	}

	ano := resolvePublicsoftYear(p)
	limit := normalizeLimit(p.Limit)
	target := fmt.Sprintf("%s/api/v1/contratos?exercicio=%d&per_page=%d", base, ano, limit)

	body, status, err := doPublicsoftGet(ctx, client, target, hashID)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Publicsoft /api/v1/contratos retornou HTTP %d", status)
	}

	rows, total, err := decodePublicsoftPaginatedList(body, "contratos")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Publicsoft contratos: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ContratoItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		fornecedor := mapGetStr(r, "fornecedor", "contratado", "credor", "razao_social", "nome_fornecedor")
		objeto := mapGetStr(r, "objetivo", "objeto", "descricao", "desc_objeto")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+objeto), filtro) {
			continue
		}

		itemAno := int(mapGetFloat(r, "exercicio", "ano", "ano_contrato"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, ContratoItem{
			Numero:              mapGetStr(r, "numero", "numero_contrato", "numeroContrato", "id"),
			Ano:                 itemAno,
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(mapGetStr(r, "cpfCnpj", "cpf_cnpj", "cnpj_cpf", "cnpjCpf", "documento", "cpf", "cnpj")),
			Objeto:              objeto,
			Situacao:            mapGetStr(r, "situacao", "status"),
			DataAssinatura:      mapGetStr(r, "dataPublicacao", "data_publicacao", "dataAssinatura", "data_assinatura", "data"),
			VigenciaInicio:      mapGetStr(r, "dataInicial", "data_inicial", "vigencia_inicio", "vigenciaInicio", "data_inicio"),
			VigenciaFim:         mapGetStr(r, "dataFinal", "data_final", "vigencia_fim", "vigenciaFim", "data_fim"),
			Valor:               mapGetFloat(r, "valor", "valor_contrato", "valorContrato", "valor_global"),
			Fonte:               "Portal Municipal (Publicsoft)",
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
		Fonte:          "Portal Municipal (Publicsoft)",
		Plataforma:     "publicsoft_api",
		PortalURL:      base,
		TotalRegistros: total,
		Contratos:      items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryPublicsoftLicitacoes consulta licitações na API Publicsoft (/api/v1/licitacoes).
func QueryPublicsoftLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LicitacoesResult, error) {
	base, hashID, err := resolvePublicsoftTarget(ctx, client, m, p)
	if err != nil {
		return nil, err
	}

	ano := resolvePublicsoftYear(p)
	limit := normalizeLimit(p.Limit)
	target := fmt.Sprintf("%s/api/v1/licitacoes?exercicio=%d&per_page=%d", base, ano, limit)

	body, status, err := doPublicsoftGet(ctx, client, target, hashID)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Publicsoft /api/v1/licitacoes retornou HTTP %d", status)
	}

	rows, total, err := decodePublicsoftPaginatedList(body, "licitacoes")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Publicsoft licitacoes: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]LicitacaoItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		objeto := mapGetStr(r, "descObjeto", "desc_objeto", "objeto", "descricao", "objetivo")
		modalidade := mapGetStr(r, "modalidade", "descricao_modalidade", "tipo")
		if filtro != "" && !strings.Contains(strings.ToLower(objeto+" "+modalidade), filtro) {
			continue
		}

		itemAno := int(mapGetFloat(r, "exercicio", "ano", "ano_licitacao"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, LicitacaoItem{
			Numero:          mapGetStr(r, "numero", "numero_licitacao", "numeroLicitacao", "edital", "processo", "id"),
			Ano:             itemAno,
			Modalidade:      modalidade,
			Situacao:        mapGetStr(r, "situacao", "status"),
			DataAbertura:    mapGetStr(r, "dataAbertura", "data_abertura", "dataPublicacao", "data_publicacao", "data"),
			Objeto:          objeto,
			ValorEstimado:   mapGetFloat(r, "valorEstimado", "valor_estimado", "estimado", "valor"),
			ValorHomologado: mapGetFloat(r, "valorHomologado", "valor_homologado", "homologado", "valor_adjudicado"),
			Link:            mapGetStr(r, "link", "url", "anexo"),
			Fonte:           "Portal Municipal (Publicsoft)",
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
		Fonte:          "Portal Municipal (Publicsoft)",
		Plataforma:     "publicsoft_api",
		PortalURL:      base,
		TotalRegistros: total,
		Licitacoes:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryPublicsoftServidores verifica a configuração de folha (/api/v1/catalogo/ugs/<hash>/folha-config)
// e consulta /api/v1/servidores quando integrado, ou retorna Disponivel: false quando modo == "nao_integrado".
func QueryPublicsoftServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ServidoresResult, error) {
	base, hashID, err := resolvePublicsoftTarget(ctx, client, m, p)
	if err != nil {
		return nil, err
	}

	cfgURL := fmt.Sprintf("%s/api/v1/catalogo/ugs/%s/folha-config", base, url.PathEscape(hashID))
	if cfgBody, cfgStatus, cfgErr := doPublicsoftGet(ctx, client, cfgURL, hashID); cfgErr == nil && cfgStatus == http.StatusOK {
		var cfgObj map[string]any
		if json.Unmarshal(cfgBody, &cfgObj) == nil && cfgObj != nil {
			modo := mapGetStr(cfgObj, "modo")
			extURL := mapGetStr(cfgObj, "url", "url_externa", "link")
			cfgMsg := mapGetStr(cfgObj, "mensagem", "message")
			if dataMap, ok := cfgObj["data"].(map[string]any); ok && dataMap != nil {
				if modo == "" {
					modo = mapGetStr(dataMap, "modo")
				}
				if extURL == "" {
					extURL = mapGetStr(dataMap, "url", "url_externa", "link")
				}
				if cfgMsg == "" {
					cfgMsg = mapGetStr(dataMap, "mensagem", "message")
				}
			}

			if strings.EqualFold(modo, "nao_integrado") || strings.Contains(strings.ToLower(modo), "nao_integrado") {
				msg := "Folha de pagamento não integrada à API Publicsoft (folha-config modo=nao_integrado)."
				if cfgMsg != "" {
					msg = cfgMsg
				}
				if extURL != "" {
					msg += " Portal externo: " + extURL
				}
				return &ServidoresResult{
					Municipio:  m,
					Fonte:      "Portal Municipal (Publicsoft)",
					Plataforma: "publicsoft_api",
					PortalURL:  base,
					Servidores: []ServidorItem{},
					Disponivel: false,
					Mensagem:   msg,
				}, nil
			}
		}
	}

	ano := resolvePublicsoftYear(p)
	limit := normalizeLimit(p.Limit)
	target := fmt.Sprintf("%s/api/v1/servidores?exercicio=%d&per_page=%d", base, ano, limit)
	if p.Mes > 0 && p.Mes <= 12 {
		target += fmt.Sprintf("&mes=%d", p.Mes)
	}

	body, status, err := doPublicsoftGet(ctx, client, target, hashID)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound || status == http.StatusNotImplemented || status == http.StatusForbidden {
		return &ServidoresResult{
			Municipio:  m,
			Fonte:      "Portal Municipal (Publicsoft)",
			Plataforma: "publicsoft_api",
			PortalURL:  base,
			Servidores: []ServidorItem{},
			Disponivel: false,
			Mensagem:   fmt.Sprintf("Endpoint de servidores indisponível nesta unidade gestora Publicsoft (HTTP %d).", status),
		}, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Publicsoft /api/v1/servidores retornou HTTP %d", status)
	}

	rows, total, err := decodePublicsoftPaginatedList(body, "servidores")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Publicsoft servidores: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ServidorItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		nome := mapGetStr(r, "nome", "servidor", "nome_servidor", "funcionario")
		if filtro != "" && !strings.Contains(strings.ToLower(nome), filtro) {
			continue
		}
		comp := mapGetStr(r, "competencia", "mes_ano", "referencia")
		if comp == "" {
			if mes := mapGetStr(r, "mes"); mes != "" {
				comp = fmt.Sprintf("%s/%04d", mes, ano)
			} else if p.Mes > 0 {
				comp = fmt.Sprintf("%02d/%04d", p.Mes, ano)
			} else {
				comp = strconv.Itoa(ano)
			}
		}

		items = append(items, ServidorItem{
			Nome:           nome,
			Matricula:      mapGetStr(r, "matricula", "registro", "codigo"),
			CPFParcial:     maskCPF(mapGetStr(r, "cpf_formatado", "cpfFormatado", "cpf_parcial"), mapGetStr(r, "cpf", "CPF", "documento")),
			Cargo:          mapGetStr(r, "cargo", "descricao_cargo", "nome_cargo"),
			Funcao:         mapGetStr(r, "funcao", "descricao_funcao"),
			Lotacao:        mapGetStr(r, "lotacao", "departamento", "setor", "unidade"),
			LocalTrabalho:  mapGetStr(r, "local_trabalho", "local"),
			Vinculo:        mapGetStr(r, "vinculo", "regime", "tipo_vinculo"),
			Situacao:       mapGetStr(r, "situacao", "status"),
			SalarioBase:    mapGetFloat(r, "salario_base", "salarioBase", "vencimento_base", "base"),
			SalarioBruto:   mapGetFloat(r, "salario_bruto", "salarioBruto", "total_proventos", "proventos", "bruto", "remuneracao"),
			Descontos:      mapGetFloat(r, "descontos", "total_descontos", "valor_descontos"),
			SalarioLiquido: mapGetFloat(r, "salario_liquido", "salarioLiquido", "valor_liquido", "liquido"),
			Competencia:    comp,
			CargaHoraria:   mapGetStr(r, "carga_horaria", "cargaHoraria"),
			DataAdmissao:   mapGetStr(r, "data_admissao", "dataAdmissao", "admissao"),
			Orgao:          mapGetStr(r, "orgao", "unidade_gestora", "entidade"),
			Fonte:          "Portal Municipal (Publicsoft)",
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
		Fonte:          "Portal Municipal (Publicsoft)",
		Plataforma:     "publicsoft_api",
		PortalURL:      base,
		TotalRegistros: total,
		Servidores:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}
