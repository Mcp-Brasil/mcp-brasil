package transparencia

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Equiplano Cloud Connector
// Base API padrão: https://api.equiplano.cloud
// Tenant headers: x-entity-uuid, x-county-client-uuid, x-client-uuid
// ---------------------------------------------------------------------------

const (
	equiplanoDefaultAPIBaseURL = "https://api.equiplano.cloud"
	equiplanoPlataforma        = "equiplano_cloud"
	equiplanoFonte             = "Portal Municipal (Equiplano Cloud)"
)

var equiplanoAPIBaseURL = equiplanoDefaultAPIBaseURL

// SetEquiplanoAPIBaseURL configura a URL base da API Equiplano Cloud (útil para testes com httptest)
// e retorna o valor anterior para restauração. Passar string vazia restaura o padrão.
func SetEquiplanoAPIBaseURL(u string) string {
	prev := equiplanoAPIBaseURL
	trimmed := strings.TrimRight(strings.TrimSpace(u), "/")
	if trimmed == "" {
		equiplanoAPIBaseURL = equiplanoDefaultAPIBaseURL
	} else {
		equiplanoAPIBaseURL = trimmed
	}
	return prev
}

func resolveEquiplanoAPIBase(p PortalQueryParams) string {
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if base != "" {
		low := strings.ToLower(base)
		if !strings.Contains(low, "portal-") && !strings.Contains(low, ".equiplano.cloud") {
			return base
		}
	}
	return strings.TrimRight(strings.TrimSpace(equiplanoAPIBaseURL), "/")
}

func resolveEquiplanoPortalURL(p PortalQueryParams, apiBase string) string {
	if base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/"); base != "" {
		return base
	}
	return apiBase
}

func parseEquiplanoTenantID(tenantID string) (entityUUID, countyClientUUID, clientUUID string) {
	parts := strings.Split(strings.TrimSpace(tenantID), "|")
	if len(parts) > 0 {
		entityUUID = strings.TrimSpace(parts[0])
	}
	if len(parts) > 1 {
		countyClientUUID = strings.TrimSpace(parts[1])
	}
	if len(parts) > 2 {
		clientUUID = strings.TrimSpace(parts[2])
	}
	return entityUUID, countyClientUUID, clientUUID
}

func equiplanoTenantHeaders(tenantID string) map[string]string {
	entityUUID, countyClientUUID, clientUUID := parseEquiplanoTenantID(tenantID)
	headers := make(map[string]string, 3)
	if entityUUID != "" {
		headers["x-entity-uuid"] = entityUUID
	}
	if countyClientUUID != "" {
		headers["x-county-client-uuid"] = countyClientUUID
	}
	if clientUUID != "" {
		headers["x-client-uuid"] = clientUUID
	}
	return headers
}

func equiplanoHTTPClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{Timeout: 12 * time.Second}
}

func doEquiplanoGet(ctx context.Context, client *http.Client, targetURL string, headers map[string]string) ([]byte, int, error) {
	hc := equiplanoHTTPClient(client)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		if strings.TrimSpace(v) != "" {
			req.Header.Set(k, v)
		}
	}

	resp, err := hc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

type equiplanoDominioCheckResp struct {
	Entidade struct {
		UUID      string `json:"uuid"`
		Nome      string `json:"nome"`
		Descricao string `json:"descricao"`
	} `json:"entidade"`
	MunicipioClienteUUID string `json:"municipioClienteUuid"`
	ClienteUUID          string `json:"clienteUuid"`
	Dados                *struct {
		Entidade struct {
			UUID      string `json:"uuid"`
			Nome      string `json:"nome"`
			Descricao string `json:"descricao"`
		} `json:"entidade"`
		MunicipioClienteUUID string `json:"municipioClienteUuid"`
		ClienteUUID          string `json:"clienteUuid"`
	} `json:"dados,omitempty"`
}

func equiplanoCandidateSlugs(m *Municipio) []string {
	if m == nil {
		return nil
	}
	slug := strings.TrimSpace(m.Slug)
	if slug == "" {
		slug = Slugify(m.Nome)
	}
	compact := SlugifyCompact(m.Nome)

	var raw []string
	if slug != "" {
		raw = append(raw, "prefeitura-"+slug)
	}
	if compact != "" {
		raw = append(raw, "prefeitura-"+compact)
	}
	if slug != "" {
		raw = append(raw, slug)
	}
	if compact != "" {
		raw = append(raw, compact)
	}
	return dedupeStrings(raw)
}

// DiscoverEquiplanoCloud descobre um município hospedado no Equiplano Cloud via /transparencia/dominio/check/{slug}.
func DiscoverEquiplanoCloud(ctx context.Context, client *http.Client, m *Municipio) (*PortalDiscoveryResult, bool) {
	if m == nil {
		return nil, false
	}
	start := time.Now()
	apiBase := strings.TrimRight(strings.TrimSpace(equiplanoAPIBaseURL), "/")
	slugs := equiplanoCandidateSlugs(m)
	tested := make([]string, 0, len(slugs))

	for _, slug := range slugs {
		u := fmt.Sprintf("%s/transparencia/dominio/check/%s", apiBase, url.PathEscape(slug))
		tested = append(tested, u)

		body, status, err := doEquiplanoGet(ctx, client, u, map[string]string{
			"x-dominio": slug,
		})
		if err != nil || status != http.StatusOK || len(strings.TrimSpace(string(body))) == 0 {
			continue
		}

		var resp equiplanoDominioCheckResp
		if err := json.Unmarshal(body, &resp); err != nil {
			continue
		}

		entUUID := strings.TrimSpace(resp.Entidade.UUID)
		entNome := strings.TrimSpace(resp.Entidade.Nome)
		if entNome == "" {
			entNome = strings.TrimSpace(resp.Entidade.Descricao)
		}
		munCliUUID := strings.TrimSpace(resp.MunicipioClienteUUID)
		cliUUID := strings.TrimSpace(resp.ClienteUUID)

		if entUUID == "" && resp.Dados != nil {
			entUUID = strings.TrimSpace(resp.Dados.Entidade.UUID)
			if entNome == "" {
				entNome = strings.TrimSpace(resp.Dados.Entidade.Nome)
				if entNome == "" {
					entNome = strings.TrimSpace(resp.Dados.Entidade.Descricao)
				}
			}
			if munCliUUID == "" {
				munCliUUID = strings.TrimSpace(resp.Dados.MunicipioClienteUUID)
			}
			if cliUUID == "" {
				cliUUID = strings.TrimSpace(resp.Dados.ClienteUUID)
			}
		}

		if entUUID == "" {
			continue
		}

		return &PortalDiscoveryResult{
			Municipio:           m,
			URLResolvida:        fmt.Sprintf("https://portal-%s.equiplano.cloud", slug),
			Plataforma:          equiplanoPlataforma,
			TenantID:            fmt.Sprintf("%s|%s|%s", entUUID, munCliUUID, cliUUID),
			EntidadeNome:        entNome,
			FonteDescoberta:     "equiplano_dominio_check",
			ServidoresSuportado: false,
			CandidatosTestados:  tested,
			TempoMs:             time.Since(start).Milliseconds(),
			Observacoes:         "Portal Equiplano Cloud identificado via /transparencia/dominio/check.",
		}, true
	}

	return nil, false
}

func resolveEquiplanoYear(p PortalQueryParams) int {
	if p.Ano > 0 {
		return p.Ano
	}
	if len(p.Inicio) >= 4 {
		if y, err := strconv.Atoi(p.Inicio[:4]); err == nil && y > 1900 {
			return y
		}
	}
	if len(p.Fim) >= 4 {
		if y, err := strconv.Atoi(p.Fim[:4]); err == nil && y > 1900 {
			return y
		}
	}
	return time.Now().Year()
}

func formatISODate(s string) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) >= 10 && trimmed[4] == '-' && trimmed[7] == '-' {
		return trimmed[:10]
	}
	if len(trimmed) == 10 && trimmed[2] == '/' && trimmed[5] == '/' {
		if t, err := time.Parse("02/01/2006", trimmed); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return trimmed
}

func resolveEquiplanoDateRange(p PortalQueryParams, ano int) (string, string) {
	if p.Inicio != "" && p.Fim != "" {
		return formatISODate(p.Inicio), formatISODate(p.Fim)
	}
	if p.Mes >= 1 && p.Mes <= 12 {
		start := time.Date(ano, time.Month(p.Mes), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, -1)
		return start.Format("2006-01-02"), end.Format("2006-01-02")
	}
	inicio := fmt.Sprintf("%04d-01-01", ano)
	fim := fmt.Sprintf("%04d-12-31", ano)
	if p.Inicio != "" {
		inicio = formatISODate(p.Inicio)
	}
	if p.Fim != "" {
		fim = formatISODate(p.Fim)
	}
	return inicio, fim
}

func decodeBase64Flexible(encoded string) ([]byte, error) {
	clean := strings.TrimSpace(encoded)
	if clean == "" {
		return nil, nil
	}
	// Remove eventuais quebras de linha no conteúdo base64
	clean = strings.ReplaceAll(clean, "\n", "")
	clean = strings.ReplaceAll(clean, "\r", "")

	if b, err := base64.StdEncoding.DecodeString(clean); err == nil {
		return b, nil
	}
	if b, err := base64.RawStdEncoding.DecodeString(clean); err == nil {
		return b, nil
	}
	return base64.URLEncoding.DecodeString(clean)
}

func decodeEquiplanoDocumentoJSON(body []byte, collectionKeys ...string) ([]map[string]any, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, nil
	}

	var envelope map[string]any
	if err := json.Unmarshal([]byte(trimmed), &envelope); err != nil {
		return nil, err
	}

	var docB64 string
	if dadosRaw, ok := envelope["dados"]; ok && dadosRaw != nil {
		switch d := dadosRaw.(type) {
		case map[string]any:
			docB64 = mapGetStr(d, "documento")
		case []any:
			rows := make([]map[string]any, 0, len(d))
			for _, el := range d {
				if row, ok := el.(map[string]any); ok {
					rows = append(rows, row)
				}
			}
			return rows, nil
		}
	}
	if docB64 == "" {
		docB64 = mapGetStr(envelope, "documento")
	}

	var payload []byte
	if docB64 != "" {
		decoded, err := decodeBase64Flexible(docB64)
		if err != nil {
			return nil, fmt.Errorf("erro ao decodificar dados.documento base64: %w", err)
		}
		payload = decoded
	} else {
		payload = []byte(trimmed)
	}

	payloadTrimmed := strings.TrimSpace(string(payload))
	if payloadTrimmed == "" {
		return nil, nil
	}
	if strings.HasPrefix(payloadTrimmed, "[") {
		var rows []map[string]any
		if err := json.Unmarshal([]byte(payloadTrimmed), &rows); err != nil {
			return nil, err
		}
		return rows, nil
	}

	var obj map[string]any
	if err := json.Unmarshal([]byte(payloadTrimmed), &obj); err != nil {
		return nil, err
	}

	keys := append([]string{}, collectionKeys...)
	keys = append(keys, "collection2018", "contratoCollection", "collection", "registros", "dados", "content", "items")
	for _, k := range keys {
		if raw, ok := obj[k]; ok && raw != nil {
			if arr, ok := raw.([]any); ok {
				rows := make([]map[string]any, 0, len(arr))
				for _, el := range arr {
					if row, ok := el.(map[string]any); ok {
						rows = append(rows, row)
					}
				}
				return rows, nil
			}
		}
	}

	// Fallback: procura qualquer chave cujo nome comece com "collection"
	for k, raw := range obj {
		if strings.HasPrefix(strings.ToLower(k), "collection") {
			if arr, ok := raw.([]any); ok {
				rows := make([]map[string]any, 0, len(arr))
				for _, el := range arr {
					if row, ok := el.(map[string]any); ok {
						rows = append(rows, row)
					}
				}
				return rows, nil
			}
		}
	}

	return nil, nil
}

func decodeEquiplanoRecords(body []byte, extraKeys ...string) ([]map[string]any, int, error) {
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

	total := int(mapGetFloat(obj, "totalElements", "totalRegistros", "total"))

	// Suporte a envelope {"dados": {"content": [...]}} ou {"dados": [...]}
	if dadosRaw, ok := obj["dados"]; ok && dadosRaw != nil {
		switch d := dadosRaw.(type) {
		case []any:
			rows := make([]map[string]any, 0, len(d))
			for _, el := range d {
				if row, ok := el.(map[string]any); ok {
					rows = append(rows, row)
				}
			}
			if total == 0 {
				total = len(rows)
			}
			return rows, total, nil
		case map[string]any:
			if subTotal := int(mapGetFloat(d, "totalElements", "totalRegistros", "total")); subTotal > 0 {
				total = subTotal
			}
			subKeys := append([]string{}, extraKeys...)
			subKeys = append(subKeys, "content", "registros", "items", "lista")
			for _, k := range subKeys {
				if arr, ok := d[k].([]any); ok {
					rows := make([]map[string]any, 0, len(arr))
					for _, el := range arr {
						if row, ok := el.(map[string]any); ok {
							rows = append(rows, row)
						}
					}
					if total == 0 {
						total = len(rows)
					}
					return rows, total, nil
				}
			}
		}
	}

	keys := append([]string{}, extraKeys...)
	keys = append(keys, "content", "registros", "items", "lista", "retorno")
	for _, k := range keys {
		if arr, ok := obj[k].([]any); ok {
			rows := make([]map[string]any, 0, len(arr))
			for _, el := range arr {
				if row, ok := el.(map[string]any); ok {
					rows = append(rows, row)
				}
			}
			if total == 0 {
				total = len(rows)
			}
			return rows, total, nil
		}
	}

	return nil, total, nil
}

// QueryEquiplanoDespesas consulta despesas na ordem cronológica de pagamentos do Equiplano Cloud.
func QueryEquiplanoDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	apiBase := resolveEquiplanoAPIBase(p)
	portalURL := resolveEquiplanoPortalURL(p, apiBase)
	limit := normalizeLimit(p.Limit)
	ano := resolveEquiplanoYear(p)
	dataInicial, dataFinal := resolveEquiplanoDateRange(p, ano)
	entityUUID, _, _ := parseEquiplanoTenantID(p.TenantID)

	q := url.Values{}
	if entityUUID != "" {
		q.Set("entidadeUuid", entityUUID)
	}
	q.Set("exercicio", strconv.Itoa(ano))
	q.Set("dataInicial", dataInicial)
	q.Set("dataFinal", dataFinal)
	q.Set("nrPaginaInicial", "1")
	q.Set("size", strconv.Itoa(limit))

	u := fmt.Sprintf("%s/transparencia/ordem_cronologica_de_pagamentos?%s", apiBase, q.Encode())
	body, status, err := doEquiplanoGet(ctx, client, u, equiplanoTenantHeaders(p.TenantID))
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Equiplano ordem_cronologica_de_pagamentos retornou HTTP %d", status)
	}

	rows, total, err := decodeEquiplanoRecords(body, "pagamentos")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Equiplano despesas: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]DespesaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		fornecedor := mapGetStr(r, "nomeFornecedor", "fornecedor", "credor", "nomeCredor", "nmFornecedor", "razaoSocial")
		historico := mapGetStr(r, "historico", "objeto", "descricao", "observacao", "justificativa")
		numEmpenho := mapGetStr(r, "numeroEmpenho", "empenho", "nrEmpenho", "ordem", "numeroOrdem")
		if filtro != "" &&
			!strings.Contains(strings.ToLower(fornecedor), filtro) &&
			!strings.Contains(strings.ToLower(historico), filtro) &&
			!strings.Contains(strings.ToLower(numEmpenho), filtro) {
			continue
		}

		doc := mapGetStr(r, "cpfCnpj", "cpfCnpjCredor", "cpfCnpjFornecedor", "documento", "cnpjCpf", "nrCpfCnpj")
		if len(cleanDigits(doc)) == 11 {
			doc = maskCPF("", doc)
		}

		anoItem := int(mapGetFloat(r, "exercicio", "ano", "anoEmpenho"))
		if anoItem == 0 {
			anoItem = ano
		}

		items = append(items, DespesaItem{
			NumeroEmpenho:       numEmpenho,
			Ano:                 anoItem,
			Data:                mapGetStr(r, "dataPagamento", "dataEmpenho", "dataLiquidacao", "dataExigibilidade", "data", "dtPagamento"),
			Fornecedor:          fornecedor,
			DocumentoFornecedor: doc,
			Orgao:               mapGetStr(r, "orgao", "orgaoDescricao", "entidade", "nmOrgao"),
			Unidade:             mapGetStr(r, "unidade", "unidadeDescricao", "nmUnidade"),
			Funcao:              mapGetStr(r, "funcao", "funcaoDescricao"),
			Elemento:            mapGetStr(r, "fonteRecurso", "fonteRecursoDescricao", "elemento", "naturezaDespesa"),
			Historico:           historico,
			ValorEmpenhado:      mapGetFloat(r, "valorEmpenhado", "vlEmpenhado", "valorEmpenho", "valorOriginal"),
			ValorLiquidado:      mapGetFloat(r, "valorLiquidado", "vlLiquidado", "valorDocumento"),
			ValorPago:           mapGetFloat(r, "valorPago", "vlPago", "valorPagamento", "valor"),
			Fonte:               equiplanoFonte,
		})
		if len(items) >= limit {
			break
		}
	}

	if filtro != "" {
		total = len(items)
	} else if total == 0 {
		total = len(rows)
	}

	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma despesa encontrada para o período informado."
	}

	return &DespesasResult{
		Municipio:      m,
		Fonte:          equiplanoFonte,
		Plataforma:     equiplanoPlataforma,
		PortalURL:      portalURL,
		TotalRegistros: total,
		Despesas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryEquiplanoReceitas consulta receitas arrecadadas no Equiplano Cloud (/transparencia/arrecadacao_receitas_arrecadada/process).
func QueryEquiplanoReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	apiBase := resolveEquiplanoAPIBase(p)
	portalURL := resolveEquiplanoPortalURL(p, apiBase)
	limit := normalizeLimit(p.Limit)
	ano := resolveEquiplanoYear(p)
	entityUUID, _, _ := parseEquiplanoTenantID(p.TenantID)

	mesInicial, mesFinal := 1, 12
	if p.Mes >= 1 && p.Mes <= 12 {
		mesInicial = p.Mes
		mesFinal = p.Mes
	}

	q := url.Values{}
	if entityUUID != "" {
		q.Set("entidadeUuid", entityUUID)
		q.Set("entidadesSelecionadasUuid", entityUUID)
	}
	q.Set("exercicio", strconv.Itoa(ano))
	q.Set("mesInicial", strconv.Itoa(mesInicial))
	q.Set("mesFinal", strconv.Itoa(mesFinal))
	q.Set("type", "JSON")
	q.Set("size", "0")

	u := fmt.Sprintf("%s/transparencia/arrecadacao_receitas_arrecadada/process?%s", apiBase, q.Encode())
	body, status, err := doEquiplanoGet(ctx, client, u, equiplanoTenantHeaders(p.TenantID))
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Equiplano arrecadacao_receitas_arrecadada retornou HTTP %d", status)
	}

	rows, err := decodeEquiplanoDocumentoJSON(body, "collection2018")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Equiplano receitas: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
	totalMatching := 0
	for _, r := range rows {
		codigo := mapGetStr(r, "codigoReceita", "codigo", "cdReceita", "estrutural", "rubrica", "receita")
		descricao := mapGetStr(r, "descricao", "dsReceita", "nomeReceita", "especificacao", "descricaoReceita")
		if filtro != "" &&
			!strings.Contains(strings.ToLower(descricao), filtro) &&
			!strings.Contains(strings.ToLower(codigo), filtro) {
			continue
		}
		totalMatching++
		if len(items) < limit {
			items = append(items, ReceitaItem{
				Codigo:          codigo,
				Descricao:       descricao,
				Categoria:       mapGetStr(r, "categoria", "fonteRecurso", "dsFonteRecurso", "origem", "tipo"),
				Data:            mapGetStr(r, "data", "dtArrecadacao", "competencia", "mesAno"),
				ValorPrevisto:   mapGetFloat(r, "valorPrevisto", "vlPrevisto", "orcado", "valorOrcado", "previsaoInicial", "previsaoAtualizada"),
				ValorArrecadado: mapGetFloat(r, "valorArrecadado", "vlArrecadado", "arrecadado", "valorRealizado", "realizado", "arrecadadoAteMes", "arrecadadoMes"),
				Fonte:           equiplanoFonte,
			})
		}
	}

	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma receita encontrada para o período informado."
	}

	return &ReceitasResult{
		Municipio:      m,
		Fonte:          equiplanoFonte,
		Plataforma:     equiplanoPlataforma,
		PortalURL:      portalURL,
		TotalRegistros: totalMatching,
		Receitas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryEquiplanoContratos consulta contratos com fornecedores no Equiplano Cloud (/transparencia/cont_contratos_fornecedor/process).
func QueryEquiplanoContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ContratosResult, error) {
	apiBase := resolveEquiplanoAPIBase(p)
	portalURL := resolveEquiplanoPortalURL(p, apiBase)
	limit := normalizeLimit(p.Limit)
	ano := resolveEquiplanoYear(p)
	entityUUID, _, _ := parseEquiplanoTenantID(p.TenantID)

	q := url.Values{}
	if entityUUID != "" {
		q.Set("entidadeUuid", entityUUID)
	}
	q.Set("exercicio", strconv.Itoa(ano))
	q.Set("type", "JSON")
	q.Set("size", "0")

	u := fmt.Sprintf("%s/transparencia/cont_contratos_fornecedor/process?%s", apiBase, q.Encode())
	body, status, err := doEquiplanoGet(ctx, client, u, equiplanoTenantHeaders(p.TenantID))
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Equiplano cont_contratos_fornecedor retornou HTTP %d", status)
	}

	rows, err := decodeEquiplanoDocumentoJSON(body, "contratoCollection")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Equiplano contratos: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ContratoItem, 0, minInt(limit, len(rows)))
	totalMatching := 0
	for _, r := range rows {
		numero := mapGetStr(r, "numeroContrato", "nrContrato", "numero", "contrato")
		fornecedor := mapGetStr(r, "nomeFornecedor", "fornecedor", "nmFornecedor", "contratado", "razaoSocial")
		objeto := mapGetStr(r, "objeto", "dsObjeto", "descricao")
		if filtro != "" &&
			!strings.Contains(strings.ToLower(fornecedor), filtro) &&
			!strings.Contains(strings.ToLower(objeto), filtro) &&
			!strings.Contains(strings.ToLower(numero), filtro) {
			continue
		}
		totalMatching++
		if len(items) < limit {
			doc := mapGetStr(r, "cpfCnpj", "cpfCnpjFornecedor", "cnpjCpf", "nrCpfCnpj", "documento")
			if len(cleanDigits(doc)) == 11 {
				doc = maskCPF("", doc)
			}
			anoItem := int(mapGetFloat(r, "exercicio", "ano", "anoContrato"))
			if anoItem == 0 {
				anoItem = ano
			}
			items = append(items, ContratoItem{
				Numero:              numero,
				Ano:                 anoItem,
				Fornecedor:          fornecedor,
				DocumentoFornecedor: doc,
				Objeto:              objeto,
				Situacao:            mapGetStr(r, "situacao", "dsSituacao", "status"),
				DataAssinatura:      mapGetStr(r, "dataAssinatura", "dtAssinatura", "dataContrato", "data"),
				VigenciaInicio:      mapGetStr(r, "vigenciaInicio", "dataInicioVigencia", "dtInicioVigencia", "dataInicio"),
				VigenciaFim:         mapGetStr(r, "vigenciaFim", "dataFimVigencia", "dtFimVigencia", "dataTermino", "dataVencimento"),
				Valor:               mapGetFloat(r, "valor", "vlContrato", "valorContrato", "valorTotal", "vlTotal"),
				Fonte:               equiplanoFonte,
			})
		}
	}

	msg := ""
	if len(items) == 0 {
		msg = "Nenhum contrato encontrado para o exercício informado."
	}

	return &ContratosResult{
		Municipio:      m,
		Fonte:          equiplanoFonte,
		Plataforma:     equiplanoPlataforma,
		PortalURL:      portalURL,
		TotalRegistros: totalMatching,
		Contratos:      items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryEquiplanoObras consulta obras públicas no Equiplano Cloud (/transparencia/rel_obras_publicas/listarObrasPublicas).
func QueryEquiplanoObras(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ObrasResult, error) {
	apiBase := resolveEquiplanoAPIBase(p)
	portalURL := resolveEquiplanoPortalURL(p, apiBase)
	limit := normalizeLimit(p.Limit)
	ano := resolveEquiplanoYear(p)
	entityUUID, _, _ := parseEquiplanoTenantID(p.TenantID)

	q := url.Values{}
	if entityUUID != "" {
		q.Set("entidadeUuid", entityUUID)
	}
	q.Set("exercicio", strconv.Itoa(ano))

	u := fmt.Sprintf("%s/transparencia/rel_obras_publicas/listarObrasPublicas?%s", apiBase, q.Encode())
	body, status, err := doEquiplanoGet(ctx, client, u, equiplanoTenantHeaders(p.TenantID))
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Equiplano listarObrasPublicas retornou HTTP %d", status)
	}

	rows, total, err := decodeEquiplanoRecords(body, "obras", "obrasPublicas")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Equiplano obras: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ObraItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		titulo := mapGetStr(r, "titulo", "nome", "nmObra", "descricao", "objeto")
		objeto := mapGetStr(r, "objeto", "dsObjeto", "descricao", "detalhamento")
		orgao := mapGetStr(r, "orgao", "entidade", "nmOrgao", "secretaria")
		bairro := mapGetStr(r, "bairro", "nmBairro")
		if filtro != "" &&
			!strings.Contains(strings.ToLower(titulo), filtro) &&
			!strings.Contains(strings.ToLower(objeto), filtro) &&
			!strings.Contains(strings.ToLower(bairro), filtro) &&
			!strings.Contains(strings.ToLower(orgao), filtro) {
			continue
		}

		items = append(items, ObraItem{
			Codigo:      mapGetStr(r, "codigo", "cdObra", "id", "numero"),
			Titulo:      titulo,
			Objeto:      objeto,
			Orgao:       orgao,
			Endereco:    mapGetStr(r, "endereco", "localizacao", "dsEndereco", "logradouro"),
			Bairro:      bairro,
			DataInicial: mapGetStr(r, "dataInicial", "dtInicio", "dataInicio", "dtInicial"),
			DataFinal:   mapGetStr(r, "dataFinal", "dtFim", "dataTermino", "dataPrevistaTermino", "dtFinal"),
			Situacao:    mapGetStr(r, "situacao", "dsSituacao", "status"),
			Valor:       mapGetFloat(r, "valor", "vlObra", "valorObra", "valorTotal", "valorContratado", "valorPrevisto"),
			Fonte:       equiplanoFonte,
		})
		if len(items) >= limit {
			break
		}
	}

	if filtro != "" {
		total = len(items)
	} else if total == 0 {
		total = len(rows)
	}

	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma obra encontrada para o exercício informado."
	}

	return &ObrasResult{
		Municipio:      m,
		Fonte:          equiplanoFonte,
		Plataforma:     equiplanoPlataforma,
		PortalURL:      portalURL,
		TotalRegistros: total,
		Obras:          items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}
