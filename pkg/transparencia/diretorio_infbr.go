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
// 1. Diretório Digital / SISTP (plataforma = "diretoriodigital_sistp")
//    Endpoint: GET https://transparencia.diretoriodigital.inf.br/api/listagem-servidores?cliente=...&mes=...&ano=...
// ---------------------------------------------------------------------------

const (
	diretorioDigitalFonte      = "Portal Municipal (Diretório Digital / SISTP)"
	diretorioDigitalPlataforma = "diretoriodigital_sistp"
)

var (
	diretorioDigitalBaseMu  sync.RWMutex
	diretorioDigitalBaseURL = "https://transparencia.diretoriodigital.inf.br"

	clienteQueryRegex   = regexp.MustCompile(`(?i)cliente=([a-z0-9._-]+)`)
	clienteJSONRegex    = regexp.MustCompile(`(?i)["']cliente["']\s*[:=]\s*["']([a-z0-9._-]+)["']`)
	clienteYAMLDefRegex = regexp.MustCompile(`(?is)name:\s*["']?cliente["']?.*?(?:default|example)\s*:\s*["']?([a-z0-9._-]+)["']?`)
)

// SetDiretorioDigitalBaseURL configura a URL base padrão da API Diretório Digital (útil para testes).
func SetDiretorioDigitalBaseURL(u string) {
	diretorioDigitalBaseMu.Lock()
	defer diretorioDigitalBaseMu.Unlock()
	trimmed := strings.TrimRight(strings.TrimSpace(u), "/")
	if trimmed == "" {
		diretorioDigitalBaseURL = "https://transparencia.diretoriodigital.inf.br"
		return
	}
	diretorioDigitalBaseURL = trimmed
}

func getDiretorioDigitalBaseURL() string {
	diretorioDigitalBaseMu.RLock()
	defer diretorioDigitalBaseMu.RUnlock()
	return strings.TrimRight(strings.TrimSpace(diretorioDigitalBaseURL), "/")
}

func extractDiretorioClienteSlug(raw string) string {
	if raw == "" {
		return ""
	}
	if m := clienteQueryRegex.FindStringSubmatch(raw); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	if m := clienteJSONRegex.FindStringSubmatch(raw); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	if m := clienteYAMLDefRegex.FindStringSubmatch(raw); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func normalizeDiretorioPortalRoot(baseURL string) (root string, queryCliente string) {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return "", ""
	}
	u, err := url.Parse(trimmed)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return trimmed, extractDiretorioClienteSlug(trimmed)
	}
	queryCliente = strings.TrimSpace(u.Query().Get("cliente"))
	u.RawQuery = ""
	u.Fragment = ""
	lowPath := strings.ToLower(u.Path)
	for _, suffix := range []string{"/dados-abertos/swagger.yaml", "/dados-abertos/swagger.json", "/dados-abertos", "/swagger.yaml", "/api/listagem-servidores"} {
		if idx := strings.Index(lowPath, suffix); idx >= 0 {
			u.Path = u.Path[:idx]
			break
		}
	}
	u.Path = strings.TrimRight(u.Path, "/")
	return strings.TrimRight(u.String(), "/"), queryCliente
}

func resolveDiretorioAPIBase(baseURL string) string {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed != "" {
		if u, err := url.Parse(trimmed); err == nil && u.Scheme != "" && u.Host != "" {
			host := strings.ToLower(u.Hostname())
			if strings.Contains(host, "diretoriodigital.inf.br") || host == "127.0.0.1" || host == "localhost" {
				root, _ := normalizeDiretorioPortalRoot(trimmed)
				return root
			}
		}
	}
	return getDiretorioDigitalBaseURL()
}

// ProbeDiretorioDigital verifica se a URL pertence ao ecossistema Diretório Digital / SISTP e extrai o slug cliente.
func ProbeDiretorioDigital(ctx context.Context, client *http.Client, baseURL string) (clienteSlug string, ok bool) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return "", false
	}
	hc := portalConnectorHTTPClient(client)
	root, slug := normalizeDiretorioPortalRoot(trimmed)
	lowBase := strings.ToLower(trimmed)
	isDiretorioHost := strings.Contains(lowBase, "diretoriodigital.inf.br")

	// 1. Verifica swagger.yaml / swagger.json / dados-abertos
	swaggerCandidates := dedupeStrings([]string{
		root + "/dados-abertos/swagger.yaml",
		root + "/swagger.yaml",
		root + "/dados-abertos/swagger.json",
		root + "/dados-abertos",
	})
	for _, target := range swaggerCandidates {
		if body, status, err := doPortalRawGet(ctx, hc, target, "text/yaml,application/yaml,application/json,text/html,*/*"); err == nil && status == http.StatusOK {
			text := string(body)
			low := strings.ToLower(text)
			if extracted := extractDiretorioClienteSlug(text); extracted != "" && slug == "" {
				slug = extracted
			}
			if strings.Contains(low, "diretoriodigital.inf.br") ||
				strings.Contains(low, "listagem-servidores") ||
				(strings.Contains(low, "sistp") && slug != "") {
				return slug, true
			}
		}
	}

	// 2. Verifica HTML da página informada / raiz
	for _, target := range dedupeStrings([]string{trimmed, root}) {
		if body, status, err := doPortalRawGet(ctx, hc, target, "text/html,application/xhtml+xml,*/*"); err == nil && status == http.StatusOK {
			text := string(body)
			low := strings.ToLower(text)
			if extracted := extractDiretorioClienteSlug(text); extracted != "" && slug == "" {
				slug = extracted
			}
			if strings.Contains(low, "diretoriodigital.inf.br") ||
				strings.Contains(low, "/api/listagem-servidores") ||
				(strings.Contains(low, "sistp") && (slug != "" || strings.Contains(low, "swagger"))) {
				return slug, true
			}
		}
	}

	// 3. Testa diretamente /api/listagem-servidores no host (ou se já é diretoriodigital.inf.br com cliente)
	if isDiretorioHost && slug != "" {
		now := time.Now()
		u := fmt.Sprintf("%s/api/listagem-servidores?cliente=%s&mes=%d&ano=%d", root, url.QueryEscape(slug), int(now.Month()), now.Year())
		if body, status, err := doJSONGet(ctx, hc, u); err == nil && (status == http.StatusOK || status == http.StatusBadRequest) {
			low := strings.ToLower(string(body))
			if status == http.StatusOK || strings.Contains(low, "cliente") || strings.Contains(low, "mes") || strings.Contains(low, "ano") {
				return slug, true
			}
		}
		return slug, true
	}

	now := time.Now()
	u := fmt.Sprintf("%s/api/listagem-servidores?mes=%d&ano=%d", root, int(now.Month()), now.Year())
	if slug != "" {
		u += "&cliente=" + url.QueryEscape(slug)
	}
	if body, status, err := doJSONGet(ctx, hc, u); err == nil && status == http.StatusOK {
		trimmedBody := strings.TrimSpace(string(body))
		if strings.HasPrefix(trimmedBody, "[") || strings.HasPrefix(trimmedBody, "{") {
			if extracted := extractDiretorioClienteSlug(trimmedBody); extracted != "" && slug == "" {
				slug = extracted
			}
			low := strings.ToLower(trimmedBody)
			if strings.Contains(low, "servidores") || strings.Contains(low, "nome") || strings.Contains(low, "matricula") || strings.Contains(low, "cargo") || strings.Contains(low, "bruto") {
				return slug, true
			}
		}
	}

	return "", false
}

func resolveDiretorioCompetencias(p PortalQueryParams) [][2]int {
	if p.Ano > 0 && p.Mes > 0 {
		return [][2]int{{p.Ano, p.Mes}}
	}
	now := time.Now()
	nowY := now.Year()
	nowM := int(now.Month())

	if p.Ano > 0 {
		var out [][2]int
		startM := 12
		if p.Ano == nowY {
			startM = nowM
		}
		for m := startM; m >= 1; m-- {
			out = append(out, [2]int{p.Ano, m})
		}
		return out
	}

	if p.Mes > 0 {
		return [][2]int{{nowY, p.Mes}, {nowY - 1, p.Mes}}
	}

	var out [][2]int
	y, m := nowY, nowM
	for i := 0; i < 6; i++ {
		out = append(out, [2]int{y, m})
		m--
		if m <= 0 {
			m = 12
			y--
		}
	}
	if nowY-1 != y || 12 != m {
		out = append(out, [2]int{nowY - 1, 12})
	}
	return out
}

// QueryDiretorioDigitalServidores consulta servidores via API Diretório Digital / SISTP (/api/listagem-servidores).
func QueryDiretorioDigitalServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams, clienteSlug string) (*ServidoresResult, error) {
	hc := portalConnectorHTTPClient(client)
	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))

	slug := strings.TrimSpace(clienteSlug)
	_, querySlug := normalizeDiretorioPortalRoot(p.BaseURL)
	if slug == "" {
		slug = querySlug
	}
	if slug == "" {
		slug = strings.TrimSpace(p.TenantID)
	}

	apiBase := resolveDiretorioAPIBase(p.BaseURL)
	portalURL := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if portalURL == "" {
		portalURL = apiBase
	}

	var lastErr error
	var lastStatus int

	for _, comp := range resolveDiretorioCompetencias(p) {
		q := url.Values{}
		if slug != "" {
			q.Set("cliente", slug)
		}
		q.Set("mes", strconv.Itoa(comp[1]))
		q.Set("ano", strconv.Itoa(comp[0]))

		target := fmt.Sprintf("%s/api/listagem-servidores?%s", apiBase, q.Encode())
		body, status, err := doJSONGet(ctx, hc, target)
		if err != nil {
			lastErr = err
			continue
		}
		lastStatus = status
		if status != http.StatusOK || len(strings.TrimSpace(string(body))) == 0 {
			continue
		}

		rows, total, err := decodeFlexiblePortalList(body, "servidores", "listagem", "folha", "result")
		if err != nil {
			lastErr = fmt.Errorf("erro ao decodificar Diretório Digital /api/listagem-servidores: %w", err)
			continue
		}
		if len(rows) == 0 {
			continue
		}

		items := make([]ServidorItem, 0, minInt(limit, len(rows)))
		matchedTotal := 0
		for _, r := range rows {
			nome := mapGetStr(r, "nome", "Nome", "servidor", "Servidor", "funcionario", "nom_servidor")
			if filtro != "" && !strings.Contains(strings.ToLower(nome), filtro) {
				continue
			}
			matchedTotal++
			if len(items) >= limit {
				continue
			}

			bruto := mapGetFloat(r, "bruto", "Bruto", "salario_bruto", "salarioBruto", "proventos", "total_proventos", "remuneracao", "vencimentos")
			descontos := mapGetFloat(r, "desconto", "descontos", "Desconto", "Descontos", "total_descontos", "valor_descontos")
			liquido := mapGetFloat(r, "liquido", "Liquido", "salario_liquido", "salarioLiquido", "valor_liquido")
			if liquido == 0 && bruto > 0 && descontos > 0 {
				liquido = bruto - descontos
			}

			compStr := mapGetStr(r, "competencia", "Competencia", "referencia")
			if compStr == "" {
				compStr = fmt.Sprintf("%02d/%04d", comp[1], comp[0])
			}

			items = append(items, ServidorItem{
				Nome:           nome,
				Matricula:      mapGetStr(r, "matricula", "Matricula", "MATRICULA", "registro", "codigo"),
				CPFParcial:     maskCPF(mapGetStr(r, "cpf_formatado", "cpfFormatado", "CPFFormatado"), mapGetStr(r, "cpf", "CPF", "Cpf", "num_cpf")),
				Cargo:          mapGetStr(r, "cargo", "Cargo", "CARGO", "descricao_cargo"),
				Funcao:         mapGetStr(r, "funcao", "Funcao"),
				Lotacao:        mapGetStr(r, "lotacao", "Lotacao", "LOTACAO", "departamento", "secretaria", "setor", "unidade"),
				Vinculo:        mapGetStr(r, "vinculo", "Vinculo", "regime", "tipo_vinculo", "tipo"),
				Situacao:       mapGetStr(r, "situacao", "Situacao", "status"),
				SalarioBase:    mapGetFloat(r, "salario_base", "salarioBase", "vencimento_base", "base"),
				SalarioBruto:   bruto,
				Descontos:      descontos,
				SalarioLiquido: liquido,
				Competencia:    compStr,
				CargaHoraria:   mapGetStr(r, "carga_horaria", "cargaHoraria", "horas"),
				DataAdmissao:   mapGetStr(r, "data_admissao", "dataAdmissao", "admissao"),
				Orgao:          mapGetStr(r, "orgao", "Orgao", "entidade"),
				Fonte:          diretorioDigitalFonte,
				PortalURL:      portalURL,
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
			Fonte:          diretorioDigitalFonte,
			Plataforma:     diretorioDigitalPlataforma,
			PortalURL:      portalURL,
			TotalRegistros: total,
			Servidores:     items,
			Disponivel:     true,
		}, nil
	}

	if lastErr != nil && lastStatus == 0 {
		return nil, lastErr
	}
	if lastStatus >= 400 && lastErr != nil {
		return nil, lastErr
	}

	return &ServidoresResult{
		Municipio:  m,
		Fonte:      diretorioDigitalFonte,
		Plataforma: diretorioDigitalPlataforma,
		PortalURL:  portalURL,
		Servidores: []ServidorItem{},
		Disponivel: true,
		Mensagem:   "Nenhum servidor encontrado para os filtros e período consultados.",
	}, nil
}

// ---------------------------------------------------------------------------
// 2. Portal da Transparência .inf.br — Net Use Sistemas (plataforma = "infbr_netuse")
//    Quirk: https://portaldatransparencia.inf.br falha com 503 proxy TLS;
//    reescrever https://portaldatransparencia.inf.br -> http://portaldatransparencia.inf.br
//    Endpoint: POST <httpPortalBase>/servidores/pe_remuneracao_servidores_ajax.php
// ---------------------------------------------------------------------------

const (
	infBrFonte      = "Portal Municipal (.inf.br / Net Use)"
	infBrPlataforma = "infbr_netuse"
	infBrAjaxPath   = "/servidores/pe_remuneracao_servidores_ajax.php"
)

func normalizeInfBrBase(baseURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return ""
	}

	u, err := url.Parse(trimmed)
	if err == nil && u.Scheme != "" && u.Host != "" {
		if strings.EqualFold(u.Scheme, "https") && strings.Contains(strings.ToLower(u.Hostname()), "portaldatransparencia.inf.br") {
			u.Scheme = "http"
		}
		u.RawQuery = ""
		u.Fragment = ""
		lowPath := strings.ToLower(u.Path)
		for _, suffix := range []string{"/servidores/pe_remuneracao_servidores_ajax.php", "/servidores"} {
			if idx := strings.Index(lowPath, suffix); idx >= 0 {
				u.Path = u.Path[:idx]
				break
			}
		}
		u.Path = strings.TrimRight(u.Path, "/")
		return strings.TrimRight(u.String(), "/")
	}

	low := strings.ToLower(trimmed)
	if strings.HasPrefix(low, "https://portaldatransparencia.inf.br") {
		trimmed = "http://" + trimmed[len("https://"):]
	} else if strings.HasPrefix(low, "https://www.portaldatransparencia.inf.br") {
		trimmed = "http://" + trimmed[len("https://"):]
	}
	return strings.TrimRight(trimmed, "/")
}

func doInfBrPostForm(ctx context.Context, client *http.Client, targetURL string, form url.Values) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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

func parseInfBrCompetencia(body []byte) (ano, mes int, ok bool) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return 0, 0, false
	}
	anoFloat := mapGetFloat(raw, "ano", "Ano", "exercicio", "Exercicio")
	mesFloat := mapGetFloat(raw, "mes", "Mes", "competencia_mes")
	ano = int(anoFloat)
	mes = int(mesFloat)
	if ano <= 0 || mes < 1 || mes > 12 {
		return 0, 0, false
	}
	return ano, mes, true
}

// ProbeInfBr verifica se a URL corresponde a um portal Net Use (.inf.br) testando action=ultima_competencia.
func ProbeInfBr(ctx context.Context, client *http.Client, baseURL string) (portalBase string, ok bool) {
	portalBase = normalizeInfBrBase(baseURL)
	if portalBase == "" {
		return "", false
	}
	hc := portalConnectorHTTPClient(client)

	form := url.Values{}
	form.Set("action", "ultima_competencia")

	body, status, err := doInfBrPostForm(ctx, hc, portalBase+infBrAjaxPath, form)
	if err != nil || status != http.StatusOK {
		return "", false
	}
	if _, _, valid := parseInfBrCompetencia(body); !valid {
		return "", false
	}
	return portalBase, true
}

func extractInfBrTotalPages(body []byte) int {
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return 0
	}
	return int(mapGetFloat(obj, "totalPages", "total_pages", "last_page", "paginas", "totalPaginas"))
}

// QueryInfBrServidores consulta servidores públicos no Portal da Transparência .inf.br (Net Use Sistemas).
func QueryInfBrServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ServidoresResult, error) {
	portalBase := normalizeInfBrBase(p.BaseURL)
	if portalBase == "" {
		return nil, fmt.Errorf("Portal .inf.br: BaseURL não informada")
	}
	hc := portalConnectorHTTPClient(client)
	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))

	ano, mes := p.Ano, p.Mes
	if ano <= 0 || mes <= 0 {
		compForm := url.Values{}
		compForm.Set("action", "ultima_competencia")
		body, status, err := doInfBrPostForm(ctx, hc, portalBase+infBrAjaxPath, compForm)
		if err != nil {
			return nil, fmt.Errorf("Portal .inf.br ultima_competencia falhou: %w", err)
		}
		if status != http.StatusOK {
			return nil, fmt.Errorf("Portal .inf.br ultima_competencia retornou HTTP %d", status)
		}
		cAno, cMes, ok := parseInfBrCompetencia(body)
		if !ok {
			return nil, fmt.Errorf("Portal .inf.br retornou competência inválida")
		}
		if ano <= 0 {
			ano = cAno
		}
		if mes <= 0 {
			mes = cMes
		}
	}

	items := make([]ServidorItem, 0, limit)
	totalRegistros := 0
	const maxPages = 20

	for page := 1; page <= maxPages; page++ {
		form := url.Values{}
		form.Set("action", "listar")
		form.Set("ano", strconv.Itoa(ano))
		form.Set("mes", fmt.Sprintf("%02d", mes))
		form.Set("page", strconv.Itoa(page))

		body, status, err := doInfBrPostForm(ctx, hc, portalBase+infBrAjaxPath, form)
		if err != nil {
			if page == 1 {
				return nil, err
			}
			break
		}
		if status != http.StatusOK {
			if page == 1 {
				return nil, fmt.Errorf("Portal .inf.br action=listar retornou HTTP %d", status)
			}
			break
		}

		rows, total, err := decodeFlexiblePortalList(body, "servidores", "folha", "result")
		if err != nil {
			if page == 1 {
				return nil, fmt.Errorf("erro ao decodificar Portal .inf.br action=listar: %w", err)
			}
			break
		}
		if total > totalRegistros {
			totalRegistros = total
		}
		if len(rows) == 0 {
			break
		}

		for _, r := range rows {
			nome := mapGetStr(r, "nome", "Nome", "servidor", "Servidor", "funcionario")
			if filtro != "" && !strings.Contains(strings.ToLower(nome), filtro) {
				continue
			}

			bruto := mapGetFloat(r, "bruto", "Bruto", "salario_bruto", "salarioBruto", "proventos", "total_proventos", "vencimentos")
			desconto := mapGetFloat(r, "desconto", "descontos", "Desconto", "Descontos", "total_descontos")
			liquido := mapGetFloat(r, "liquido", "Liquido", "salario_liquido", "salarioLiquido", "valor_liquido")
			if liquido == 0 && bruto > 0 && desconto > 0 {
				liquido = bruto - desconto
			}

			items = append(items, ServidorItem{
				Nome:           nome,
				Matricula:      mapGetStr(r, "Matricula", "matricula", "MATRICULA", "registro", "codigo"),
				CPFParcial:     maskCPF(mapGetStr(r, "cpf_formatado", "cpfFormatado", "CPFFormatado"), mapGetStr(r, "cpf", "CPF", "Cpf")),
				Cargo:          mapGetStr(r, "cargo", "Cargo", "CARGO", "descricao_cargo"),
				Lotacao:        mapGetStr(r, "lotacao", "Lotacao", "LOTACAO", "departamento", "secretaria", "unidade"),
				Vinculo:        mapGetStr(r, "vinculo", "Vinculo", "VINCULO", "regime", "tipo_vinculo"),
				Situacao:       mapGetStr(r, "situacao", "Situacao", "SITUACAO", "status"),
				SalarioBase:    mapGetFloat(r, "salario_base", "base", "vencimento_base"),
				SalarioBruto:   bruto,
				Descontos:      desconto,
				SalarioLiquido: liquido,
				Competencia:    fmt.Sprintf("%02d/%04d", mes, ano),
				Orgao:          mapGetStr(r, "orgao", "Orgao", "entidade"),
				Fonte:          infBrFonte,
				PortalURL:      portalBase,
			})
			if len(items) >= limit {
				break
			}
		}

		if len(items) >= limit {
			break
		}
		totalPages := extractInfBrTotalPages(body)
		if totalPages > 0 && page >= totalPages {
			break
		}
	}

	if totalRegistros == 0 {
		totalRegistros = len(items)
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhum servidor encontrado para os filtros e período consultados."
	}

	return &ServidoresResult{
		Municipio:      m,
		Fonte:          infBrFonte,
		Plataforma:     infBrPlataforma,
		PortalURL:      portalBase,
		TotalRegistros: totalRegistros,
		Servidores:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}
