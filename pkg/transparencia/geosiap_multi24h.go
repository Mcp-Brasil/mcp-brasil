package transparencia

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// ===========================================================================
// 1. GeoSiap (plataforma = "geosiap_api")
// Base URL normalizada: https://<host>[:port]/portal-transparencia/api
// ===========================================================================

// NormalizeGeoSiapAPIBase normaliza qualquer URL de portal GeoSiap
// (ex: https://<host>[:port]/portal-transparencia/...) para <scheme>://<host>[:port]/portal-transparencia/api.
// Para URLs de teste locais (http / localhost) sem /portal-transparencia ou /api, preserva a URL base do mock.
func NormalizeGeoSiapAPIBase(rawURL string) string {
	raw := strings.TrimSpace(rawURL)
	if raw == "" {
		return ""
	}
	if idx := strings.Index(raw, "#"); idx >= 0 {
		raw = raw[:idx]
	}
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return ""
	}

	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	u.RawQuery = ""
	u.Fragment = ""

	p := strings.TrimRight(u.Path, "/")
	lowP := strings.ToLower(p)
	origin := u.Scheme + "://" + u.Host

	if idx := strings.Index(lowP, "/portal-transparencia"); idx >= 0 {
		prefix := p[:idx+len("/portal-transparencia")]
		return origin + prefix + "/api"
	}
	if strings.HasSuffix(lowP, "/api") {
		return origin + p
	}
	if isLocalTestURL(raw) {
		return strings.TrimRight(origin+p, "/")
	}
	return strings.TrimRight(origin+p, "/") + "/portal-transparencia/api"
}

func extractGeoSiapPortalRoots(rawURL string) []string {
	raw := strings.TrimSpace(rawURL)
	if idx := strings.Index(raw, "#"); idx >= 0 {
		raw = raw[:idx]
	}
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil
	}
	origin := u.Scheme + "://" + u.Host
	p := strings.TrimRight(u.Path, "/")
	lowP := strings.ToLower(p)

	var candidates []string
	if idx := strings.Index(lowP, "/portal-transparencia"); idx >= 0 {
		candidates = append(candidates, origin+p[:idx+len("/portal-transparencia")])
	}
	if p != "" && p != "/" {
		candidates = append(candidates, origin+p)
	}
	candidates = append(candidates, origin+"/portal-transparencia", origin)
	return dedupeNonEmptyStrings(candidates)
}

func fetchGeoSiapEnvClienteAPI(ctx context.Context, client *http.Client, rawURL string) string {
	hc := portalConnectorHTTPClient(client)
	for _, root := range extractGeoSiapPortalRoots(rawURL) {
		target := strings.TrimRight(root, "/") + "/assets/env.cliente.json"
		body, status, err := doJSONGet(ctx, hc, target)
		if err != nil || status != http.StatusOK {
			continue
		}
		var cfg map[string]any
		if err := json.Unmarshal(body, &cfg); err != nil || cfg == nil {
			continue
		}
		val := mapGetStr(cfg, "apiUrl", "api_url", "urlApi", "url_api", "baseUrl", "base_url", "api", "endpoint", "url")
		if val == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(val), "http://") || strings.HasPrefix(strings.ToLower(val), "https://") {
			return strings.TrimRight(val, "/")
		}
		if strings.HasPrefix(val, "/") {
			if u, pErr := url.Parse(root); pErr == nil && u.Scheme != "" && u.Host != "" {
				return strings.TrimRight(u.Scheme+"://"+u.Host+val, "/")
			}
		}
	}
	return ""
}

func isGeoSiapTruthy(v any) bool {
	if v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t == 1
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s == "1" || s == "true" || s == "s" || s == "sim"
	default:
		return false
	}
}

func isGeoSiapActive(r map[string]any) bool {
	v, exists := r["ativo"]
	if !exists || v == nil {
		return true
	}
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s == "" || s == "1" || s == "true" || s == "s" || s == "sim"
	default:
		return true
	}
}

func parseGeoSiapEntidades(body []byte) (idEntidade int, entName string, ok bool) {
	trimmed := strings.TrimSpace(string(body))
	if !strings.HasPrefix(trimmed, "[") {
		return 0, "", false
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(trimmed), &rows); err != nil || len(rows) == 0 {
		return 0, "", false
	}

	var firstID int
	var firstName string
	var hasFirst bool
	var activeID int
	var activeName string
	var hasActive bool

	for _, r := range rows {
		id := int(mapGetFloat(r, "id_entidade", "idEntidade", "id"))
		name := mapGetStr(r, "ds_entidade", "dsEntidade", "nome", "entidade")
		schema := mapGetStr(r, "ds_schema")
		if id == 0 && name == "" && schema == "" {
			continue
		}
		if !hasFirst {
			firstID = id
			firstName = name
			hasFirst = true
		}
		active := isGeoSiapActive(r)
		if active && !hasActive {
			activeID = id
			activeName = name
			hasActive = true
		}
		if isGeoSiapTruthy(r["principal"]) {
			return id, name, true
		}
	}
	if hasActive {
		return activeID, activeName, true
	}
	if hasFirst {
		return firstID, firstName, true
	}
	return 0, "", false
}

// ProbeGeoSiap verifica se a URL corresponde a uma API GeoSiap via GET <apiBase>/sis_entidade
// (com fallback para assets/env.cliente.json). Seleciona a entidade com principal=="1" ou a primeira entidade.
func ProbeGeoSiap(ctx context.Context, client *http.Client, rawURL string) (apiBase string, idEntidade int, entName string, ok bool) {
	base := NormalizeGeoSiapAPIBase(rawURL)
	if base == "" {
		return "", 0, "", false
	}
	hc := portalConnectorHTTPClient(client)

	candidates := []string{base}
	if isLocalTestURL(rawURL) && !strings.Contains(strings.ToLower(base), "/portal-transparencia/api") {
		candidates = append(candidates, base+"/portal-transparencia/api")
	}

	for _, cand := range candidates {
		body, status, err := doJSONGet(ctx, hc, cand+"/sis_entidade")
		if err != nil || status != http.StatusOK {
			continue
		}
		if id, name, valid := parseGeoSiapEntidades(body); valid {
			return cand, id, name, true
		}
	}

	if envAPI := fetchGeoSiapEnvClienteAPI(ctx, hc, rawURL); envAPI != "" {
		body, status, err := doJSONGet(ctx, hc, envAPI+"/sis_entidade")
		if err == nil && status == http.StatusOK {
			if id, name, valid := parseGeoSiapEntidades(body); valid {
				return envAPI, id, name, true
			}
		}
	}

	return "", 0, "", false
}

func doGeoSiapGet(ctx context.Context, client *http.Client, apiBase, endpointWithQuery string) ([]byte, int, string, error) {
	hc := portalConnectorHTTPClient(client)
	body, status, err := doJSONGet(ctx, hc, apiBase+endpointWithQuery)
	if err == nil && status == http.StatusNotFound && isLocalTestURL(apiBase) && !strings.Contains(strings.ToLower(apiBase), "/portal-transparencia/api") {
		altBase := apiBase + "/portal-transparencia/api"
		if altBody, altStatus, altErr := doJSONGet(ctx, hc, altBase+endpointWithQuery); altErr == nil && altStatus != http.StatusNotFound {
			return altBody, altStatus, altBase, nil
		}
	}
	return body, status, apiBase, err
}

func resolveGeoSiapTarget(ctx context.Context, client *http.Client, p PortalQueryParams) (apiBase string, eid int, ano int, err error) {
	apiBase = NormalizeGeoSiapAPIBase(p.BaseURL)
	if apiBase == "" {
		return "", 0, 0, fmt.Errorf("GeoSiap: BaseURL não informada")
	}

	eid = p.CodAdministracao
	if eid <= 0 && strings.TrimSpace(p.TenantID) != "" {
		if parsed, convErr := strconv.Atoi(strings.TrimSpace(p.TenantID)); convErr == nil && parsed > 0 {
			eid = parsed
		}
	}
	if eid <= 0 {
		if probedBase, probedID, _, ok := ProbeGeoSiap(ctx, client, p.BaseURL); ok && probedID > 0 {
			apiBase = probedBase
			eid = probedID
		} else {
			eid = 1
		}
	}

	ano = p.Ano
	if ano <= 0 {
		ano = parseYearFromDateStr(p.Inicio)
	}
	if ano <= 0 {
		ano = time.Now().Year()
	}
	return apiBase, eid, ano, nil
}

// QueryGeoSiapReceitas consulta receitas próprias na API GeoSiap
// (GET <apiBase>/execucao/receitas/proprias/receitas_proprias?id_entidade=<eid>&ano=<ano>&layout=1).
func QueryGeoSiapReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	apiBase, eid, ano, err := resolveGeoSiapTarget(ctx, client, p)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	ep := fmt.Sprintf("/execucao/receitas/proprias/receitas_proprias?id_entidade=%d&ano=%d&layout=1", eid, ano)
	body, status, usedBase, err := doGeoSiapGet(ctx, client, apiBase, ep)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GeoSiap receitas_proprias retornou HTTP %d", status)
	}

	rows, total, err := decodeFlexiblePortalList(body, "receitas_proprias", "receitas")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar GeoSiap receitas_proprias: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		codigo := mapGetStr(r, "cd_plano_conta_rel", "cd_plano_conta", "codigo")
		desc := mapGetStr(r, "ds_plano_conta", "descricao", "nome")
		if filtro != "" && !strings.Contains(strings.ToLower(codigo+" "+desc), filtro) {
			continue
		}
		previsto := mapGetFloat(r, "vl_previsto", "previsto")
		if previsto == 0 {
			previsto = mapGetFloat(r, "vl_atualizado")
		}

		items = append(items, ReceitaItem{
			Codigo:          codigo,
			Descricao:       desc,
			Categoria:       mapGetStr(r, "nivel", "categoria"),
			ValorPrevisto:   previsto,
			ValorArrecadado: mapGetFloat(r, "vl_total", "vl_arrecadado", "arrecadado"),
			Fonte:           "Portal Municipal (GeoSiap)",
		})
		if len(items) >= limit {
			break
		}
	}

	if total == 0 || filtro != "" {
		total = len(items)
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma receita encontrada para o período informado."
	}

	return &ReceitasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (GeoSiap)",
		Plataforma:     "geosiap_api",
		PortalURL:      usedBase,
		TotalRegistros: total,
		Receitas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryGeoSiapDespesas consulta programas e ações de despesas na API GeoSiap
// (GET <apiBase>/execucao/despesas/programas_acoes?id_entidade=<eid>&cp_ano=<ano>).
func QueryGeoSiapDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	apiBase, eid, ano, err := resolveGeoSiapTarget(ctx, client, p)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	ep := fmt.Sprintf("/execucao/despesas/programas_acoes?id_entidade=%d&cp_ano=%d", eid, ano)
	body, status, usedBase, err := doGeoSiapGet(ctx, client, apiBase, ep)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GeoSiap programas_acoes retornou HTTP %d", status)
	}

	rows, total, err := decodeFlexiblePortalList(body, "programas_acoes", "despesas")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar GeoSiap programas_acoes: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]DespesaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		chave := mapGetStr(r, "chave", "codigo")
		desc := mapGetStr(r, "descricao", "nome")
		if filtro != "" && !strings.Contains(strings.ToLower(chave+" "+desc), filtro) {
			continue
		}

		items = append(items, DespesaItem{
			NumeroEmpenho:  chave,
			Ano:            ano,
			Funcao:         mapGetStr(r, "nivel", "funcao"),
			Elemento:       chave,
			Historico:      desc,
			ValorEmpenhado: mapGetFloat(r, "vl_empenhado", "empenhado"),
			ValorLiquidado: mapGetFloat(r, "vl_liquidado", "liquidado"),
			ValorPago:      mapGetFloat(r, "vl_pago", "pago"),
			Fonte:          "Portal Municipal (GeoSiap)",
		})
		if len(items) >= limit {
			break
		}
	}

	if total == 0 || filtro != "" {
		total = len(items)
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma despesa encontrada para o período informado."
	}

	return &DespesasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (GeoSiap)",
		Plataforma:     "geosiap_api",
		PortalURL:      usedBase,
		TotalRegistros: total,
		Despesas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryGeoSiapContratos consulta contratos na API GeoSiap
// (GET <apiBase>/licitacoes/contratos/contratos?id_entidade=<eid>&ano=<ano>).
func QueryGeoSiapContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ContratosResult, error) {
	apiBase, eid, ano, err := resolveGeoSiapTarget(ctx, client, p)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	ep := fmt.Sprintf("/licitacoes/contratos/contratos?id_entidade=%d&ano=%d", eid, ano)
	body, status, usedBase, err := doGeoSiapGet(ctx, client, apiBase, ep)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GeoSiap contratos retornou HTTP %d", status)
	}

	rows, total, err := decodeFlexiblePortalList(body, "contratos")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar GeoSiap contratos: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ContratoItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		numero := mapGetStr(r, "contrato", "numero", "numero_contrato")
		fornecedor := mapGetStr(r, "nome_razao", "fornecedor", "razao_social", "nome")
		objeto := mapGetStr(r, "objeto", "descricao")
		if filtro != "" && !strings.Contains(strings.ToLower(numero+" "+fornecedor+" "+objeto), filtro) {
			continue
		}
		itemAno := int(mapGetFloat(r, "ano", "exercicio"))
		if itemAno == 0 {
			itemAno = ano
		}
		doc := mapGetStr(r, "cpf", "cnpj", "documento", "cpf_cnpj")

		items = append(items, ContratoItem{
			Numero:              numero,
			Ano:                 itemAno,
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(doc),
			Objeto:              objeto,
			Situacao:            mapGetStr(r, "situacao", "status"),
			DataAssinatura:      mapGetStr(r, "dt_assinatura", "data_assinatura"),
			VigenciaInicio:      mapGetStr(r, "dt_inicio_contrato", "vigencia_inicio"),
			VigenciaFim:         mapGetStr(r, "dt_termino_contrato", "vigencia_fim"),
			Valor:               mapGetFloat(r, "vl_total_f", "vl_inicial_f", "vl_total", "vl_inicial", "valor"),
			Fonte:               "Portal Municipal (GeoSiap)",
		})
		if len(items) >= limit {
			break
		}
	}

	if total == 0 || filtro != "" {
		total = len(items)
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhum contrato encontrado para os filtros consultados."
	}

	return &ContratosResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (GeoSiap)",
		Plataforma:     "geosiap_api",
		PortalURL:      usedBase,
		TotalRegistros: total,
		Contratos:      items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// ===========================================================================
// 2. Multi24h (plataforma = "multi24h_xml")
// Webservices base normalizada: https://<host>/<prefix>/multi24/webservices
// ===========================================================================

// NormalizeMulti24hWSBase normaliza URLs contendo /multi24/ para a base de webservices
// (ex: https://nuvem.multi24h.com.br/salvadordosul/sistemas/multi24/webservices).
// Para URLs de teste locais (http / localhost) sem /multi24 ou /webservices, preserva a URL base do mock.
func NormalizeMulti24hWSBase(rawURL string) string {
	raw := strings.TrimSpace(rawURL)
	if raw == "" {
		return ""
	}
	if idx := strings.Index(raw, "#"); idx >= 0 {
		raw = raw[:idx]
	}
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return ""
	}

	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	u.RawQuery = ""
	u.Fragment = ""

	p := strings.TrimRight(u.Path, "/")
	lowP := strings.ToLower(p)
	origin := u.Scheme + "://" + u.Host

	if idx := strings.Index(lowP, "/multi24/"); idx >= 0 {
		prefix := p[:idx+len("/multi24")]
		return origin + prefix + "/webservices"
	}
	if strings.HasSuffix(lowP, "/multi24") {
		return origin + p + "/webservices"
	}
	if strings.HasSuffix(lowP, "/webservices") {
		return origin + p
	}
	if isLocalTestURL(raw) {
		return strings.TrimRight(origin+p, "/")
	}
	return strings.TrimRight(origin+p, "/") + "/multi24/webservices"
}

func doMulti24hXMLPost(ctx context.Context, client *http.Client, targetURL string, payload any) ([]byte, int, error) {
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
	req.Header.Set("Accept", "application/xml, text/xml, */*")

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

func isMulti24hSwagger(body []byte) bool {
	trimmed := strings.TrimSpace(string(body))
	if !strings.HasPrefix(trimmed, "{") {
		return false
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(trimmed), &doc); err != nil || doc == nil {
		return false
	}
	_, hasPaths := doc["paths"]
	_, hasSwagger := doc["swagger"]
	_, hasOpenAPI := doc["openapi"]
	if !hasPaths && !hasSwagger && !hasOpenAPI {
		return false
	}
	low := strings.ToLower(trimmed)
	return strings.Contains(low, "dados-abertos") ||
		strings.Contains(low, "receita") ||
		strings.Contains(low, "multi24")
}

type multi24hReceitasXML struct {
	XMLName  xml.Name             `xml:"receitas"`
	Receitas []multi24hReceitaXML `xml:"receita"`
}

type multi24hReceitaXML struct {
	Receita           string `xml:"receita"`
	Categoria         string `xml:"categoria"`
	Descricao         string `xml:"descricao"`
	Entidade          string `xml:"entidade"`
	Orgao             string `xml:"orgao"`
	PrevisaoInicial   string `xml:"previsao_inicial"`
	PrevisaoAdicional string `xml:"previsao_adicional"`
	PrevisaoAnulada   string `xml:"previsao_anulada"`
	Arrecadacao       string `xml:"arrecadacao"`
}

func iso88591CharsetReader(charset string, input io.Reader) (io.Reader, error) {
	cs := strings.ToLower(strings.TrimSpace(charset))
	if cs == "" || cs == "utf-8" || cs == "utf8" {
		return input, nil
	}
	if cs == "iso-8859-1" || cs == "latin1" || cs == "latin-1" || cs == "windows-1252" || cs == "cp1252" {
		raw, err := io.ReadAll(input)
		if err != nil {
			return nil, err
		}
		if utf8.Valid(raw) {
			return bytes.NewReader(raw), nil
		}
		var b strings.Builder
		b.Grow(len(raw) * 2)
		for _, c := range raw {
			b.WriteRune(rune(c))
		}
		return strings.NewReader(b.String()), nil
	}
	return nil, fmt.Errorf("charset XML não suportado: %s", charset)
}

func decodeMulti24hReceitasXML(body []byte) (*multi24hReceitasXML, error) {
	cleaned := bytes.TrimSpace(bytes.TrimPrefix(body, []byte("\xef\xbb\xbf")))
	if len(cleaned) == 0 {
		return nil, fmt.Errorf("resposta XML vazia")
	}
	dec := xml.NewDecoder(bytes.NewReader(cleaned))
	dec.CharsetReader = iso88591CharsetReader
	var out multi24hReceitasXML
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ProbeMulti24h verifica se a URL corresponde a um portal Multi24h via
// <prefix>/multi24/assets/files/contabil_planejamento/swagger-dados-abertos.json
// ou POST <wsBase>/transparencia/dados-abertos/receita/index retornando XML <receitas>.
func ProbeMulti24h(ctx context.Context, client *http.Client, rawURL string) (wsBase string, ok bool) {
	base := NormalizeMulti24hWSBase(rawURL)
	if base == "" {
		return "", false
	}
	hc := portalConnectorHTTPClient(client)

	const swaggerSuffix = "/assets/files/contabil_planejamento/swagger-dados-abertos.json"
	var swaggerURLs []string
	if strings.HasSuffix(strings.ToLower(base), "/webservices") {
		root := base[:len(base)-len("/webservices")]
		swaggerURLs = append(swaggerURLs, root+swaggerSuffix)
	} else {
		swaggerURLs = append(swaggerURLs, base+swaggerSuffix, base+"/multi24"+swaggerSuffix)
	}

	for _, swURL := range swaggerURLs {
		body, status, err := doJSONGet(ctx, hc, swURL)
		if err == nil && status == http.StatusOK && isMulti24hSwagger(body) {
			return base, true
		}
	}

	wsCandidates := []string{base}
	if isLocalTestURL(rawURL) && !strings.HasSuffix(strings.ToLower(base), "/webservices") {
		wsCandidates = append(wsCandidates, base+"/multi24/webservices")
	}

	payload := map[string]any{
		"exercicio": time.Now().Year(),
	}
	for _, cand := range wsCandidates {
		target := cand + "/transparencia/dados-abertos/receita/index"
		body, status, err := doMulti24hXMLPost(ctx, hc, target, payload)
		if err != nil || status != http.StatusOK {
			continue
		}
		if _, decErr := decodeMulti24hReceitasXML(body); decErr == nil {
			return cand, true
		}
	}

	return "", false
}

// QueryMulti24hReceitas consulta receitas no webservice XML da Multi24h
// (POST <wsBase>/transparencia/dados-abertos/receita/index).
func QueryMulti24hReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	wsBase := NormalizeMulti24hWSBase(p.BaseURL)
	if wsBase == "" {
		return nil, fmt.Errorf("Multi24h: BaseURL não informada")
	}

	ano := p.Ano
	if ano <= 0 {
		ano = parseYearFromDateStr(p.Inicio)
	}
	if ano <= 0 {
		ano = time.Now().Year()
	}
	limit := normalizeLimit(p.Limit)

	payload := map[string]any{
		"exercicio":       ano,
		"codigo":          nil,
		"complemento_stn": nil,
		"recurso_stn":     nil,
	}

	const endpointPath = "/transparencia/dados-abertos/receita/index"
	usedBase := wsBase
	body, status, err := doMulti24hXMLPost(ctx, client, wsBase+endpointPath, payload)
	if err == nil && status == http.StatusNotFound && isLocalTestURL(wsBase) && !strings.HasSuffix(strings.ToLower(wsBase), "/webservices") {
		altBase := wsBase + "/multi24/webservices"
		if altBody, altStatus, altErr := doMulti24hXMLPost(ctx, client, altBase+endpointPath, payload); altErr == nil && altStatus != http.StatusNotFound {
			body, status, usedBase = altBody, altStatus, altBase
		}
	}
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Multi24h %s retornou HTTP %d", endpointPath, status)
	}

	parsed, err := decodeMulti24hReceitasXML(body)
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar XML Multi24h receitas: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ReceitaItem, 0, minInt(limit, len(parsed.Receitas)))
	matched := 0
	for _, r := range parsed.Receitas {
		codigo := strings.TrimSpace(r.Receita)
		desc := strings.TrimSpace(r.Descricao)
		categoria := strings.TrimSpace(r.Categoria)
		entidade := strings.TrimSpace(r.Entidade)
		orgao := strings.TrimSpace(r.Orgao)
		if categoria == "" {
			categoria = orgao
		}
		if filtro != "" && !strings.Contains(strings.ToLower(codigo+" "+desc+" "+categoria+" "+orgao+" "+entidade), filtro) {
			continue
		}
		matched++
		if len(items) >= limit {
			continue
		}

		prevInicial, _ := parseBRFloat(r.PrevisaoInicial)
		prevAdic, _ := parseBRFloat(r.PrevisaoAdicional)
		prevAnul, _ := parseBRFloat(r.PrevisaoAnulada)
		valorPrevisto := prevInicial + prevAdic - math.Abs(prevAnul)
		valorArrecadado, _ := parseBRFloat(r.Arrecadacao)

		items = append(items, ReceitaItem{
			Codigo:          codigo,
			Descricao:       desc,
			Categoria:       categoria,
			ValorPrevisto:   valorPrevisto,
			ValorArrecadado: valorArrecadado,
			Fonte:           "Portal Municipal (Multi24h XML)",
		})
	}

	total := len(parsed.Receitas)
	if filtro != "" {
		total = matched
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma receita encontrada para o período informado."
	}

	return &ReceitasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Multi24h XML)",
		Plataforma:     "multi24h_xml",
		PortalURL:      usedBase,
		TotalRegistros: total,
		Receitas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}
