package transparencia

import (
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
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// TomeConta / PortalTransparencia.app.br (plataforma = "tomeconta_appbr")
// Base padrão: https://portaltransparencia.app.br
// ---------------------------------------------------------------------------

const defaultTomeContaBaseURL = "https://portaltransparencia.app.br"

var (
	tomeContaBaseMu  sync.RWMutex
	tomeContaBaseURL = defaultTomeContaBaseURL

	tomeContaIDRegex     = regexp.MustCompile(`(?i)(?:^|[?&\s"';])p_i=([A-Za-z0-9_-]+)`)
	tomeContaMSDateRegex = regexp.MustCompile(`(?i)/Date\((-?\d+)(?:[+-]\d{4})?\)/`)
)

// SetTomeContaBaseURL configura a URL base do portal TomeConta (útil para testes).
func SetTomeContaBaseURL(u string) {
	tomeContaBaseMu.Lock()
	defer tomeContaBaseMu.Unlock()
	trimmed := strings.TrimRight(strings.TrimSpace(u), "/")
	if trimmed == "" {
		tomeContaBaseURL = defaultTomeContaBaseURL
		return
	}
	tomeContaBaseURL = trimmed
}

func getTomeContaBaseURL() string {
	tomeContaBaseMu.RLock()
	defer tomeContaBaseMu.RUnlock()
	return strings.TrimRight(strings.TrimSpace(tomeContaBaseURL), "/")
}

func resolveTomeContaBaseURL(rawBaseURL string) string {
	configured := getTomeContaBaseURL()
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
	if strings.Contains(host, "portaltransparencia.app.br") {
		if configured != defaultTomeContaBaseURL {
			return configured
		}
		return defaultTomeContaBaseURL
	}

	return strings.TrimRight(u.Scheme+"://"+u.Host, "/")
}

// ExtractTomeContaID extrai o parâmetro p_i=<id> de URLs (incluindo Location de redirect) ou HTML
// do portal TomeConta (ex: carregaPortalPM.aspx?p_i=43&p_t=0 ou carregaGrupoEntidade.aspx?p_i=43).
func ExtractTomeContaID(rawURLOrHTML string) string {
	trimmed := strings.TrimSpace(rawURLOrHTML)
	if trimmed == "" {
		return ""
	}
	unescaped := html.UnescapeString(trimmed)

	if u, err := url.Parse(unescaped); err == nil {
		for k, vals := range u.Query() {
			if strings.EqualFold(k, "p_i") && len(vals) > 0 {
				if id := strings.TrimSpace(vals[0]); id != "" {
					return id
				}
			}
		}
	}

	if m := tomeContaIDRegex.FindStringSubmatch(unescaped); len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func parseTomeContaDate(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if m := tomeContaMSDateRegex.FindStringSubmatch(s); len(m) >= 2 {
		ms, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			return s
		}
		// Ignora DateTime.MinValue (.NET = /Date(-62135596800000)/)
		if ms <= -62135596800000 {
			return ""
		}
		return time.UnixMilli(ms).UTC().Format("2006-01-02")
	}
	return s
}

func resolveTomeContaIDViaRedirect(ctx context.Context, client *http.Client, rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return ""
	}
	if id := ExtractTomeContaID(trimmed); id != "" {
		return id
	}

	u, err := url.Parse(trimmed)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}

	baseClient := portalConnectorHTTPClient(client)
	noRedirectClient := &http.Client{
		Timeout:   baseClient.Timeout,
		Transport: baseClient.Transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, trimmed, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")

	resp, err := noRedirectClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if loc := resp.Header.Get("Location"); loc != "" {
		if id := ExtractTomeContaID(loc); id != "" {
			return id
		}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	if err == nil && len(body) > 0 {
		return ExtractTomeContaID(string(body))
	}
	return ""
}

func isTomeContaStatusOK(body []byte) bool {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
		return false
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil || obj == nil {
		return false
	}
	return strings.EqualFold(mapGetStr(obj, "status"), "ok")
}

// ProbeTomeConta valida o portal TomeConta consultando GET /api/apiDados.ashx?metodo=retornaDespesa&id=<id>&ano=<ano>&mes=1
// e verificando se retorna {"status":"ok",...}.
func ProbeTomeConta(ctx context.Context, client *http.Client, baseURL, tenantID string) bool {
	id := strings.TrimSpace(tenantID)
	if id == "" {
		id = ExtractTomeContaID(baseURL)
	}
	if id == "" {
		id = resolveTomeContaIDViaRedirect(ctx, client, baseURL)
	}
	if id == "" {
		return false
	}

	base := resolveTomeContaBaseURL(baseURL)
	if base == "" {
		return false
	}

	ano := time.Now().Year()
	target := fmt.Sprintf(
		"%s/api/apiDados.ashx?metodo=retornaDespesa&id=%s&ano=%d&mes=1",
		base, url.QueryEscape(id), ano,
	)
	body, status, err := doJSONGet(ctx, portalConnectorHTTPClient(client), target)
	if err != nil || status != http.StatusOK {
		return false
	}
	return isTomeContaStatusOK(body)
}

func resolveTomeContaTarget(ctx context.Context, client *http.Client, p PortalQueryParams) (baseURL, tenantID string, err error) {
	baseURL = resolveTomeContaBaseURL(p.BaseURL)
	tenantID = strings.TrimSpace(p.TenantID)
	if tenantID == "" {
		tenantID = ExtractTomeContaID(p.BaseURL)
	}
	if tenantID == "" && p.CodAdministracao > 0 {
		tenantID = strconv.Itoa(p.CodAdministracao)
	}
	if tenantID == "" && strings.TrimSpace(p.BaseURL) != "" {
		tenantID = resolveTomeContaIDViaRedirect(ctx, client, p.BaseURL)
	}
	if tenantID == "" {
		return "", "", fmt.Errorf("TomeConta: id do tenant (p_i) não informado")
	}
	return baseURL, tenantID, nil
}

func resolveTomeContaYear(p PortalQueryParams) int {
	if p.Ano > 0 {
		return p.Ano
	}
	if y := parseYearFromDateStr(p.Inicio); y > 0 {
		return y
	}
	return time.Now().Year()
}

func decodeTomeContaResponse(body []byte) (rows []map[string]any, total int, apiStatus string, apiMsg string, err error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, 0, "", "", nil
	}
	var obj map[string]any
	if uErr := json.Unmarshal([]byte(trimmed), &obj); uErr == nil && obj != nil {
		apiStatus = mapGetStr(obj, "status")
		apiMsg = mapGetStr(obj, "mensagem", "message", "erro", "error")
	}
	rows, total, err = decodeFlexiblePortalList(body, "dados", "Dados")
	return rows, total, apiStatus, apiMsg, err
}

func fetchTomeContaEndpoint(
	ctx context.Context,
	client *http.Client,
	baseURL, tenantID, metodo string,
	ano, mes int,
	allowMonthFallback bool,
) ([]map[string]any, int, error) {
	hc := portalConnectorHTTPClient(client)

	call := func(m int) ([]map[string]any, int, int, string, string, error) {
		q := url.Values{}
		q.Set("metodo", metodo)
		q.Set("id", tenantID)
		if ano > 0 {
			q.Set("ano", strconv.Itoa(ano))
		}
		if m > 0 && m <= 12 {
			q.Set("mes", strconv.Itoa(m))
		}
		target := fmt.Sprintf("%s/api/apiDados.ashx?%s", baseURL, q.Encode())
		body, status, err := doJSONGet(ctx, hc, target)
		if err != nil {
			return nil, 0, status, "", "", err
		}
		if status != http.StatusOK {
			return nil, 0, status, "", "", fmt.Errorf("TomeConta %s retornou HTTP %d", metodo, status)
		}
		rows, total, apiStatus, apiMsg, decErr := decodeTomeContaResponse(body)
		if decErr != nil {
			return nil, 0, status, apiStatus, apiMsg, fmt.Errorf("erro ao decodificar TomeConta %s: %w", metodo, decErr)
		}
		return rows, total, status, apiStatus, apiMsg, nil
	}

	if mes > 0 && mes <= 12 {
		rows, total, _, apiStatus, apiMsg, err := call(mes)
		if err != nil {
			return nil, 0, err
		}
		if apiStatus != "" && !strings.EqualFold(apiStatus, "ok") {
			return nil, 0, fmt.Errorf("TomeConta %s retornou status=%q (%s)", metodo, apiStatus, apiMsg)
		}
		return rows, total, nil
	}

	// Primeiro tenta omitindo mes (caso o servidor aceite consulta anual direta)
	rows, total, _, apiStatus, apiMsg, err := call(0)
	if err == nil && (apiStatus == "" || strings.EqualFold(apiStatus, "ok")) && (len(rows) > 0 || !allowMonthFallback) {
		return rows, total, nil
	}
	if !allowMonthFallback {
		if err != nil {
			return nil, 0, err
		}
		return nil, 0, fmt.Errorf("TomeConta %s retornou status=%q (%s)", metodo, apiStatus, apiMsg)
	}

	// Fallback percorrendo meses 1..12 até encontrar dados válidos
	var lastErr error
	for m := 1; m <= 12; m++ {
		mRows, mTotal, _, mStatus, mMsg, mErr := call(m)
		if mErr != nil {
			lastErr = mErr
			continue
		}
		if mStatus != "" && !strings.EqualFold(mStatus, "ok") {
			lastErr = fmt.Errorf("TomeConta %s retornou status=%q (%s)", metodo, mStatus, mMsg)
			continue
		}
		if len(mRows) > 0 {
			return mRows, mTotal, nil
		}
		lastErr = nil
	}
	if lastErr != nil && err != nil {
		return nil, 0, lastErr
	}
	return []map[string]any{}, 0, nil
}

// QueryTomeContaDespesas consulta despesas no portal TomeConta (metodo=retornaDespesa).
func QueryTomeContaDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	base, tenantID, err := resolveTomeContaTarget(ctx, client, p)
	if err != nil {
		return nil, err
	}

	ano := resolveTomeContaYear(p)
	limit := normalizeLimit(p.Limit)
	rows, total, err := fetchTomeContaEndpoint(ctx, client, base, tenantID, "retornaDespesa", ano, p.Mes, true)
	if err != nil {
		return nil, err
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]DespesaItem, 0, minInt(limit, len(rows)))
	matchedTotal := 0
	for _, r := range rows {
		fornecedor := mapGetStr(r, "FORNECEDOR", "fornecedor", "CREDOR", "credor", "NOME_FORNECEDOR")
		historico := mapGetStr(r, "HISTORICO", "historico", "DESCRICAO", "descricao", "OBJETO")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+historico), filtro) {
			continue
		}
		matchedTotal++
		if len(items) >= limit {
			continue
		}

		itemAno := int(mapGetFloat(r, "ANO", "ano", "EXERCICIO", "exercicio"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, DespesaItem{
			NumeroEmpenho:       mapGetStr(r, "EMPENHO", "empenho", "NUM_EMPENHO", "NUMERO_EMPENHO", "NUMERO", "id", "ID"),
			Ano:                 itemAno,
			Data:                parseTomeContaDate(mapGetStr(r, "DATA", "data", "DATA_EMPENHO", "DATA_PAGAMENTO")),
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(mapGetStr(r, "CPF_CNPJ", "cpf_cnpj", "CPFCNPJ", "cpfcnpj", "CNPJ_CPF", "CNPJ", "CPF", "DOCUMENTO", "documento")),
			Orgao:               mapGetStr(r, "ORGAO", "orgao", "UNIDADE_GESTORA", "ENTIDADE"),
			Unidade:             mapGetStr(r, "UNIDADE", "unidade", "UNIDADE_ORCAMENTARIA"),
			Funcao:              mapGetStr(r, "FUNCAO", "funcao"),
			Elemento:            mapGetStr(r, "ELEMENTO", "elemento", "NATUREZA", "DESPESA"),
			Historico:           historico,
			ValorEmpenhado:      mapGetFloat(r, "VALOREMPENHO", "VALOR_EMPENHO", "VALOR_EMPENHADO", "valor_empenhado", "valorEmpenhado", "EMPENHADO", "VALOR"),
			ValorLiquidado:      mapGetFloat(r, "VALORLIQUIDADO", "VALOR_LIQUIDADO", "valorLiquidado", "valor_liquidado", "LIQUIDADO"),
			ValorPago:           mapGetFloat(r, "VALORPAGO", "VALOR_PAGO", "valorPago", "valor_pago", "PAGO"),
			Fonte:               "Portal Municipal (TomeConta)",
		})
	}

	if total == 0 || filtro != "" {
		total = matchedTotal
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma despesa encontrada para o período informado."
	}

	return &DespesasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (TomeConta)",
		Plataforma:     "tomeconta_appbr",
		PortalURL:      base,
		TotalRegistros: total,
		Despesas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryTomeContaReceitas consulta receitas no portal TomeConta (metodo=retornaReceita).
func QueryTomeContaReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	base, tenantID, err := resolveTomeContaTarget(ctx, client, p)
	if err != nil {
		return nil, err
	}

	ano := resolveTomeContaYear(p)
	limit := normalizeLimit(p.Limit)
	rows, total, err := fetchTomeContaEndpoint(ctx, client, base, tenantID, "retornaReceita", ano, p.Mes, true)
	if err != nil {
		return nil, err
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
	matchedTotal := 0
	for _, r := range rows {
		desc := mapGetStr(r, "DESCRICAO", "descricao", "RECEITA", "receita", "ESPECIFICACAO", "NOME", "TITULO")
		if filtro != "" && !strings.Contains(strings.ToLower(desc), filtro) {
			continue
		}
		matchedTotal++
		if len(items) >= limit {
			continue
		}

		items = append(items, ReceitaItem{
			Codigo:          mapGetStr(r, "CODIGO", "codigo", "COD_RECEITA", "RUBRICA", "rubrica", "CLASSIFICACAO", "CONTA"),
			Descricao:       desc,
			Categoria:       mapGetStr(r, "CATEGORIA", "categoria", "FONTE", "fonte", "ORIGEM", "TIPO"),
			Data:            parseTomeContaDate(mapGetStr(r, "DATA", "data", "COMPETENCIA")),
			ValorPrevisto:   mapGetFloat(r, "VALOR_PREVISTO", "VALORPREVISTO", "PREVISTO", "previsto", "ORCADO", "PREVISAO", "VALOR_ORCADO"),
			ValorArrecadado: mapGetFloat(r, "VALOR_ARRECADADO", "VALORARRECADADO", "ARRECADADO", "arrecadado", "REALIZADO", "VALOR_REALIZADO", "VALOR"),
			Fonte:           "Portal Municipal (TomeConta)",
		})
	}

	if total == 0 || filtro != "" {
		total = matchedTotal
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma receita encontrada para o período informado."
	}

	return &ReceitasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (TomeConta)",
		Plataforma:     "tomeconta_appbr",
		PortalURL:      base,
		TotalRegistros: total,
		Receitas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryTomeContaLicitacoes consulta licitações no portal TomeConta (metodo=Licitacoes).
func QueryTomeContaLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LicitacoesResult, error) {
	base, tenantID, err := resolveTomeContaTarget(ctx, client, p)
	if err != nil {
		return nil, err
	}

	ano := resolveTomeContaYear(p)
	limit := normalizeLimit(p.Limit)
	rows, total, err := fetchTomeContaEndpoint(ctx, client, base, tenantID, "Licitacoes", ano, 0, false)
	if err != nil {
		return nil, err
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]LicitacaoItem, 0, minInt(limit, len(rows)))
	matchedTotal := 0
	for _, r := range rows {
		titulo := mapGetStr(r, "TITULO", "titulo")
		objeto := mapGetStr(r, "DESCRICAO", "descricao", "OBJETO", "objeto", "TITULO", "titulo")
		modalidade := mapGetStr(r, "MODALIDADE", "modalidade", "TIPO")
		if filtro != "" && !strings.Contains(strings.ToLower(objeto+" "+modalidade+" "+titulo), filtro) {
			continue
		}
		matchedTotal++
		if len(items) >= limit {
			continue
		}

		itemAno := int(mapGetFloat(r, "ANO", "ano", "EXERCICIO"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, LicitacaoItem{
			Numero:          mapGetStr(r, "PROCESSO", "NUMERO", "numero", "EDITAL", "TITULO", "ID", "id"),
			Ano:             itemAno,
			Modalidade:      modalidade,
			Situacao:        mapGetStr(r, "STATUS", "status", "SITUACAO", "situacao"),
			DataAbertura:    parseTomeContaDate(mapGetStr(r, "DATA_ABERTURA", "data_abertura", "dataAbertura", "data", "DATA")),
			Objeto:          objeto,
			ValorEstimado:   mapGetFloat(r, "VALOR_ESTIMADO", "valor_estimado", "valorEstimado", "ESTIMADO"),
			ValorHomologado: mapGetFloat(r, "VALOR_ADJUDICADO", "valor_adjudicado", "VALOR_HOMOLOGADO", "valor_homologado", "VALOR"),
			Link:            mapGetStr(r, "ANEXO", "anexo", "LINK", "link", "URL", "url"),
			Fonte:           "Portal Municipal (TomeConta)",
		})
	}

	if total == 0 || filtro != "" {
		total = matchedTotal
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma licitação encontrada para o período informado."
	}

	return &LicitacoesResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (TomeConta)",
		Plataforma:     "tomeconta_appbr",
		PortalURL:      base,
		TotalRegistros: total,
		Licitacoes:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryTomeContaContratos consulta contratos no portal TomeConta (metodo=retornaContratos).
func QueryTomeContaContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ContratosResult, error) {
	base, tenantID, err := resolveTomeContaTarget(ctx, client, p)
	if err != nil {
		return nil, err
	}

	ano := resolveTomeContaYear(p)
	limit := normalizeLimit(p.Limit)
	rows, total, err := fetchTomeContaEndpoint(ctx, client, base, tenantID, "retornaContratos", ano, 0, false)
	if err != nil {
		return nil, err
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ContratoItem, 0, minInt(limit, len(rows)))
	matchedTotal := 0
	for _, r := range rows {
		fornecedor := mapGetStr(r, "FORNECEDOR", "fornecedor", "CONTRATADO", "RAZAO_SOCIAL")
		objeto := mapGetStr(r, "DESCRICAO", "descricao", "OBJETO", "objeto")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+objeto), filtro) {
			continue
		}
		matchedTotal++
		if len(items) >= limit {
			continue
		}

		itemAno := int(mapGetFloat(r, "ANO", "ano", "EXERCICIO"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, ContratoItem{
			Numero:              mapGetStr(r, "NUM_CONTRATO", "num_contrato", "NUMERO", "numero", "CONTRATO", "ID", "id"),
			Ano:                 itemAno,
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(mapGetStr(r, "CPF_CNPJ", "cpf_cnpj", "CNPJ_CPF", "DOCUMENTO", "CPF", "CNPJ")),
			Objeto:              objeto,
			Situacao:            mapGetStr(r, "SITUACAO", "situacao", "STATUS", "status"),
			DataAssinatura:      parseTomeContaDate(mapGetStr(r, "DATA", "data", "DATA_ASSINATURA")),
			VigenciaInicio:      parseTomeContaDate(mapGetStr(r, "VIGENCIAI", "vigenciai", "VIGENCIA_INICIO", "DATA_INICIO")),
			VigenciaFim:         parseTomeContaDate(mapGetStr(r, "VIGENCIAF", "vigenciaf", "VIGENCIA_FIM", "DATA_FIM")),
			Valor:               mapGetFloat(r, "VALOR_CONTRATO", "valor_contrato", "VALOR", "valor"),
			Fonte:               "Portal Municipal (TomeConta)",
		})
	}

	if total == 0 || filtro != "" {
		total = matchedTotal
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhum contrato encontrado para o período informado."
	}

	return &ContratosResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (TomeConta)",
		Plataforma:     "tomeconta_appbr",
		PortalURL:      base,
		TotalRegistros: total,
		Contratos:      items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}
