package transparencia

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// e-Pública Portal da Transparência Connector
// Base padrão: https://transparencia.e-publica.net/epublica-portal
// ---------------------------------------------------------------------------

var (
	epublicaBaseMu     sync.RWMutex
	epublicaAPIBaseURL = "https://transparencia.e-publica.net/epublica-portal"
)

// SetEPublicaBaseURL configura a URL base do portal e-Pública (útil para testes).
func SetEPublicaBaseURL(u string) {
	epublicaBaseMu.Lock()
	defer epublicaBaseMu.Unlock()
	trimmed := strings.TrimRight(strings.TrimSpace(u), "/")
	if trimmed == "" {
		epublicaAPIBaseURL = "https://transparencia.e-publica.net/epublica-portal"
		return
	}
	epublicaAPIBaseURL = trimmed
}

func getEPublicaBaseURL() string {
	epublicaBaseMu.RLock()
	defer epublicaBaseMu.RUnlock()
	return strings.TrimRight(strings.TrimSpace(epublicaAPIBaseURL), "/")
}

func portalConnectorHTTPClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{Timeout: 8 * time.Second}
}

func epublicaCandidateAliases(m *Municipio) []string {
	if m == nil {
		return nil
	}
	compact := SlugifyCompact(m.Nome)
	ufLower := strings.ToLower(strings.TrimSpace(m.UF))
	slug := strings.TrimSpace(m.Slug)

	var raw []string
	if compact != "" && ufLower != "" {
		raw = append(raw, compact+"_"+ufLower)
	}
	if slug != "" {
		raw = append(raw, slug)
	}
	if compact != "" {
		raw = append(raw, compact)
	}
	if slug != "" {
		raw = append(raw, strings.ReplaceAll(slug, "-", "_"))
	}
	return dedupeStrings(raw)
}

func extractEPublicaEntidade(body []byte) (string, bool) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "", false
	}

	extractFromObj := func(obj map[string]any) (string, bool) {
		if obj == nil {
			return "", false
		}
		exRaw, ok := obj["exercicio"]
		if !ok || exRaw == nil {
			return "", false
		}
		exMap, ok := exRaw.(map[string]any)
		if !ok {
			return "", false
		}
		entRaw, ok := exMap["entidade"]
		if !ok || entRaw == nil {
			return "", false
		}
		switch v := entRaw.(type) {
		case string:
			s := strings.TrimSpace(v)
			if s != "" {
				return s, true
			}
		case map[string]any:
			s := mapGetStr(v, "nome", "descricao", "razaoSocial", "entidade")
			if s != "" {
				return s, true
			}
		}
		return "", false
	}

	if strings.HasPrefix(trimmed, "[") {
		var arr []map[string]any
		if err := json.Unmarshal([]byte(trimmed), &arr); err != nil {
			return "", false
		}
		for _, item := range arr {
			if ent, ok := extractFromObj(item); ok {
				return ent, true
			}
		}
		return "", false
	}

	var obj map[string]any
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil {
		return "", false
	}
	return extractFromObj(obj)
}

func probeEPublicaAlias(ctx context.Context, client *http.Client, baseURL, alias string) (string, string, bool) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	alias = strings.Trim(strings.TrimSpace(alias), "/")
	if base == "" || alias == "" {
		return "", "", false
	}
	target := fmt.Sprintf("%s/rest/%s/public/user/attr/exercicios", base, url.PathEscape(alias))
	body, status, err := doJSONGet(ctx, portalConnectorHTTPClient(client), target)
	if err != nil || status != http.StatusOK {
		return target, "", false
	}
	ent, ok := extractEPublicaEntidade(body)
	if !ok {
		return target, "", false
	}
	return target, ent, true
}

// DiscoverEPublica tenta descobrir o tenant/alias de um município no portal centralizado e-Pública.
func DiscoverEPublica(ctx context.Context, client *http.Client, m *Municipio) (*PortalDiscoveryResult, bool) {
	if m == nil {
		return nil, false
	}
	start := time.Now()
	hc := portalConnectorHTTPClient(client)
	base := getEPublicaBaseURL()
	aliases := epublicaCandidateAliases(m)
	tested := make([]string, 0, len(aliases))

	for _, alias := range aliases {
		target, entidade, ok := probeEPublicaAlias(ctx, hc, base, alias)
		if target != "" {
			tested = append(tested, target)
		}
		if ok {
			return &PortalDiscoveryResult{
				Municipio:           m,
				URLResolvida:        base,
				Plataforma:          "epublica_portal",
				TenantID:            alias,
				EntidadeNome:        entidade,
				FonteDescoberta:     "epublica_api",
				ServidoresSuportado: true,
				CandidatosTestados:  tested,
				TempoMs:             time.Since(start).Milliseconds(),
			}, true
		}
	}

	return nil, false
}

// ProbeEPublicaURL verifica uma URL customizada ou de override do portal e-Pública (ex: .../epublica-portal/rest/{alias}/...).
func ProbeEPublicaURL(ctx context.Context, client *http.Client, baseURL string) (alias string, entidade string, ok bool) {
	portalBase, extractedAlias := splitEPublicaBaseAndAlias(baseURL, "")
	if portalBase == "" || extractedAlias == "" {
		return "", "", false
	}
	_, ent, found := probeEPublicaAlias(ctx, portalConnectorHTTPClient(client), portalBase, extractedAlias)
	if !found {
		return "", "", false
	}
	return extractedAlias, ent, true
}

func splitEPublicaBaseAndAlias(baseURL, tenantID string) (string, string) {
	raw := strings.TrimSpace(baseURL)
	alias := strings.Trim(strings.TrimSpace(tenantID), "/")

	if raw == "" {
		return getEPublicaBaseURL(), alias
	}

	lower := strings.ToLower(raw)
	if idx := strings.Index(lower, "/rest/"); idx >= 0 {
		portalBase := strings.TrimRight(raw[:idx], "/")
		rest := raw[idx+len("/rest/"):]
		rest = strings.SplitN(rest, "?", 2)[0]
		rest = strings.SplitN(rest, "#", 2)[0]
		parts := strings.Split(strings.Trim(rest, "/"), "/")
		if alias == "" && len(parts) > 0 && parts[0] != "" {
			alias = parts[0]
		}
		if portalBase == "" {
			portalBase = getEPublicaBaseURL()
		}
		return portalBase, alias
	}

	if idx := strings.Index(raw, "#/"); idx >= 0 {
		portalBase := strings.TrimRight(raw[:idx], "/")
		frag := strings.SplitN(raw[idx+2:], "?", 2)[0]
		parts := strings.Split(strings.Trim(frag, "/"), "/")
		if alias == "" && len(parts) > 0 && parts[0] != "" {
			alias = parts[0]
		}
		if portalBase == "" {
			portalBase = getEPublicaBaseURL()
		}
		return portalBase, alias
	}

	return strings.TrimRight(raw, "/"), alias
}

func resolveEPublicaYears(p PortalQueryParams) []int {
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

func decodeEPublicaPage(body []byte) ([]map[string]any, int, error) {
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

	var rows []map[string]any
	for _, key := range []string{"content", "data", "rows", "registros", "Registros", "items", "dados", "lista", "resultado"} {
		if raw, ok := obj[key]; ok {
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

	total := int(mapGetFloat(obj, "totalElements", "total", "totalRegistros", "recordsTotal", "count"))
	if total == 0 {
		total = len(rows)
	}
	return rows, total, nil
}

func fetchEPublicaModule(ctx context.Context, client *http.Client, p PortalQueryParams, modulePath string) ([]map[string]any, int, int, string, error) {
	base, alias := splitEPublicaBaseAndAlias(p.BaseURL, p.TenantID)
	if alias == "" {
		return nil, 0, 0, base, fmt.Errorf("e-Pública: TenantID (alias da entidade) não informado")
	}
	limit := normalizeLimit(p.Limit)
	scanLimit := limit
	if strings.TrimSpace(p.NomeFiltro) != "" && scanLimit < 500 {
		scanLimit = 500
	}
	pageSize := normalizePageSize(limit, 200)
	hc := portalConnectorHTTPClient(client)
	years := resolveEPublicaYears(p)

	var lastStatus int
	for _, ano := range years {
		var allRows []map[string]any
		total := 0
		prevBody := ""
		failedYear := false

		for pageIdx := 1; pageIdx <= 10; pageIdx++ {
			u := fmt.Sprintf(
				"%s/rest/%s/%s?exercicio=%d&page=%d&size=%d",
				base, url.PathEscape(alias), strings.TrimLeft(modulePath, "/"), ano, pageIdx, pageSize,
			)
			body, status, err := doJSONGet(ctx, hc, u)
			if err != nil {
				if pageIdx == 1 {
					return nil, 0, ano, base, err
				}
				break
			}
			lastStatus = status
			if status != http.StatusOK {
				if pageIdx == 1 {
					if p.Ano > 0 {
						return nil, 0, ano, base, fmt.Errorf("e-Pública /%s retornou HTTP %d", modulePath, status)
					}
					failedYear = true
				}
				break
			}
			bodyStr := strings.TrimSpace(string(body))
			if pageIdx > 1 && (bodyStr == "" || bodyStr == prevBody) {
				break
			}
			prevBody = bodyStr

			rows, pageTotal, err := decodeEPublicaPage(body)
			if err != nil {
				if pageIdx == 1 {
					return nil, 0, ano, base, fmt.Errorf("erro ao decodificar e-Pública /%s: %w", modulePath, err)
				}
				break
			}
			if pageIdx == 1 {
				total = pageTotal
			}
			if len(rows) == 0 {
				break
			}
			allRows = append(allRows, rows...)
			if len(allRows) >= scanLimit || len(rows) < pageSize || (total > 0 && len(allRows) >= total) {
				break
			}
		}

		if failedYear || (len(allRows) == 0 && len(years) > 1) {
			continue
		}
		if len(allRows) > scanLimit {
			allRows = allRows[:scanLimit]
		}
		return allRows, total, ano, base, nil
	}

	if lastStatus >= 400 {
		return nil, 0, years[0], base, fmt.Errorf("e-Pública /%s retornou HTTP %d", modulePath, lastStatus)
	}
	return nil, 0, years[0], base, nil
}

func maskIfCPF(doc string) string {
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

// QueryEPublicaServidores consulta servidores públicos no portal e-Pública (/gestaoDePessoal/servidores/listAll).
func QueryEPublicaServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ServidoresResult, error) {
	rows, total, ano, base, err := fetchEPublicaModule(ctx, client, p, "gestaoDePessoal/servidores/listAll")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ServidorItem, 0, minInt(limit, len(rows)))

	for _, r := range rows {
		nome := mapGetStr(r, "nome", "nomeServidor", "servidor", "nomServidor", "funcionario")
		if filtro != "" && !strings.Contains(strings.ToLower(nome), filtro) {
			continue
		}
		comp := mapGetStr(r, "competencia", "mesAno", "referencia")
		if comp == "" {
			mes := int(mapGetFloat(r, "mes", "mesCompetencia"))
			if mes > 0 {
				comp = fmt.Sprintf("%02d/%04d", mes, ano)
			} else if ano > 0 {
				comp = strconv.Itoa(ano)
			}
		}

		items = append(items, ServidorItem{
			Nome:           nome,
			Matricula:      mapGetStr(r, "matricula", "registro", "codigo", "numMatricula"),
			CPFParcial:     maskCPF(mapGetStr(r, "cpfFormatado", "cpfParcial"), mapGetStr(r, "cpf", "numCpf", "documento")),
			Cargo:          mapGetStr(r, "cargo", "descricaoCargo", "nomeCargo", "funcaoCargo"),
			Funcao:         mapGetStr(r, "funcao", "descricaoFuncao"),
			Lotacao:        mapGetStr(r, "lotacao", "descricaoLotacao", "unidade", "setor", "departamento"),
			LocalTrabalho:  mapGetStr(r, "localTrabalho", "local"),
			Vinculo:        mapGetStr(r, "vinculo", "tipoVinculo", "regime", "regimeJuridico", "categoria"),
			Situacao:       mapGetStr(r, "situacao", "status", "situacaoFuncional"),
			SalarioBase:    mapGetFloat(r, "salarioBase", "vencimentoBase", "vencimento", "base"),
			SalarioBruto:   mapGetFloat(r, "salarioBruto", "totalProventos", "proventos", "remuneracaoBruta", "bruto", "remuneracao"),
			Descontos:      mapGetFloat(r, "descontos", "totalDescontos", "valorDescontos"),
			SalarioLiquido: mapGetFloat(r, "salarioLiquido", "liquido", "valorLiquido"),
			Competencia:    comp,
			CargaHoraria:   mapGetStr(r, "cargaHoraria", "horasSemanais"),
			DataAdmissao:   mapGetStr(r, "dataAdmissao", "admissao"),
			Orgao:          mapGetStr(r, "orgao", "entidade", "unidadeGestora"),
			Fonte:          "Portal Municipal (e-Pública)",
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
		Fonte:          "Portal Municipal (e-Pública)",
		Plataforma:     "epublica_portal",
		PortalURL:      base,
		TotalRegistros: total,
		Servidores:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryEPublicaDespesas consulta empenhos/despesas no portal e-Pública (/despesa/empenhos/listAll).
func QueryEPublicaDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	rows, total, ano, base, err := fetchEPublicaModule(ctx, client, p, "despesa/empenhos/listAll")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]DespesaItem, 0, minInt(limit, len(rows)))

	for _, r := range rows {
		fornecedor := mapGetStr(r, "fornecedor", "credor", "nomeFornecedor", "nomeCredor", "razaoSocial")
		historico := mapGetStr(r, "historico", "descricao", "objeto")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+historico), filtro) {
			continue
		}
		itemAno := int(mapGetFloat(r, "exercicio", "ano", "anoEmpenho"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, DespesaItem{
			NumeroEmpenho:       mapGetStr(r, "numeroEmpenho", "empenho", "numero", "numEmpenho", "codigo"),
			Ano:                 itemAno,
			Data:                mapGetStr(r, "data", "dataEmpenho", "dataEmissao"),
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(mapGetStr(r, "documentoFornecedor", "cnpjCpf", "cpfCnpj", "cnpj", "documento", "cpfCnpjCredor")),
			Orgao:               mapGetStr(r, "orgao", "nomeOrgao", "descricaoOrgao"),
			Unidade:             mapGetStr(r, "unidade", "unidadeOrcamentaria", "nomeUnidade"),
			Funcao:              mapGetStr(r, "funcao", "descricaoFuncao"),
			Elemento:            mapGetStr(r, "elemento", "elementoDespesa", "naturezaDespesa", "despesaOrcamentaria"),
			Historico:           historico,
			ValorEmpenhado:      mapGetFloat(r, "valorEmpenhado", "empenhado", "valor", "vlrEmpenhado"),
			ValorLiquidado:      mapGetFloat(r, "valorLiquidado", "liquidado", "vlrLiquidado"),
			ValorPago:           mapGetFloat(r, "valorPago", "pago", "vlrPago"),
			Fonte:               "Portal Municipal (e-Pública)",
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
		Fonte:          "Portal Municipal (e-Pública)",
		Plataforma:     "epublica_portal",
		PortalURL:      base,
		TotalRegistros: total,
		Despesas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryEPublicaReceitas consulta receitas arrecadadas no portal e-Pública (/receita/arrecadada/listAll).
func QueryEPublicaReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	rows, total, _, base, err := fetchEPublicaModule(ctx, client, p, "receita/arrecadada/listAll")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ReceitaItem, 0, minInt(limit, len(rows)))

	for _, r := range rows {
		desc := mapGetStr(r, "descricao", "especificacao", "nomeReceita", "receita", "nome")
		if filtro != "" && !strings.Contains(strings.ToLower(desc), filtro) {
			continue
		}
		items = append(items, ReceitaItem{
			Codigo:          mapGetStr(r, "codigo", "codigoReceita", "classificacao", "naturezaReceita", "conta"),
			Descricao:       desc,
			Categoria:       mapGetStr(r, "categoria", "fonteRecurso", "origem", "especie"),
			Data:            mapGetStr(r, "data", "dataArrecadacao", "competencia"),
			ValorPrevisto:   mapGetFloat(r, "valorPrevisto", "previsto", "orcado", "valorOrcado", "previsaoInicial"),
			ValorArrecadado: mapGetFloat(r, "valorArrecadado", "arrecadado", "realizado", "valorRealizado", "valor"),
			Fonte:           "Portal Municipal (e-Pública)",
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
		Fonte:          "Portal Municipal (e-Pública)",
		Plataforma:     "epublica_portal",
		PortalURL:      base,
		TotalRegistros: total,
		Receitas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryEPublicaLicitacoes consulta licitações no portal e-Pública (/compras/licitacao/listAll).
func QueryEPublicaLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LicitacoesResult, error) {
	rows, total, ano, base, err := fetchEPublicaModule(ctx, client, p, "compras/licitacao/listAll")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]LicitacaoItem, 0, minInt(limit, len(rows)))

	for _, r := range rows {
		objeto := mapGetStr(r, "objeto", "descricao", "resumo")
		if filtro != "" && !strings.Contains(strings.ToLower(objeto), filtro) {
			continue
		}
		itemAno := int(mapGetFloat(r, "ano", "exercicio", "anoProcesso"))
		if itemAno == 0 {
			itemAno = ano
		}
		link := mapGetStr(r, "link", "url")
		if link == "" {
			link = base
		}

		items = append(items, LicitacaoItem{
			Numero:          mapGetStr(r, "numero", "numeroLicitacao", "numeroProcesso", "edital", "processo"),
			Ano:             itemAno,
			Modalidade:      mapGetStr(r, "modalidade", "descricaoModalidade", "tipo"),
			Situacao:        mapGetStr(r, "situacao", "status", "situacaoLicitacao"),
			DataAbertura:    mapGetStr(r, "dataAbertura", "abertura", "dataPublicacao", "data"),
			Objeto:          objeto,
			ValorEstimado:   mapGetFloat(r, "valorEstimado", "valorMaximo", "valorPrevisto"),
			ValorHomologado: mapGetFloat(r, "valorHomologado", "valorTotal", "valorAdjudicado", "valorVencido"),
			Link:            link,
			Fonte:           "Portal Municipal (e-Pública)",
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
		Fonte:          "Portal Municipal (e-Pública)",
		Plataforma:     "epublica_portal",
		PortalURL:      base,
		TotalRegistros: total,
		Licitacoes:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryEPublicaContratos consulta contratos no portal e-Pública (/compras/contrato/listAll).
func QueryEPublicaContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ContratosResult, error) {
	rows, total, ano, base, err := fetchEPublicaModule(ctx, client, p, "compras/contrato/listAll")
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ContratoItem, 0, minInt(limit, len(rows)))

	for _, r := range rows {
		fornecedor := mapGetStr(r, "fornecedor", "contratado", "nomeFornecedor", "razaoSocial")
		objeto := mapGetStr(r, "objeto", "descricao")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+objeto), filtro) {
			continue
		}
		itemAno := int(mapGetFloat(r, "ano", "exercicio", "anoContrato"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, ContratoItem{
			Numero:              mapGetStr(r, "numero", "numeroContrato", "contrato"),
			Ano:                 itemAno,
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(mapGetStr(r, "documentoFornecedor", "cnpjCpf", "cpfCnpj", "cnpj", "documento")),
			Objeto:              objeto,
			Situacao:            mapGetStr(r, "situacao", "status"),
			DataAssinatura:      mapGetStr(r, "dataAssinatura", "assinatura", "data"),
			VigenciaInicio:      mapGetStr(r, "vigenciaInicio", "dataInicio", "inicioVigencia"),
			VigenciaFim:         mapGetStr(r, "vigenciaFim", "dataFim", "dataVencimento", "fimVigencia"),
			Valor:               mapGetFloat(r, "valor", "valorContrato", "valorGlobal", "valorTotal"),
			Fonte:               "Portal Municipal (e-Pública)",
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
		Fonte:          "Portal Municipal (e-Pública)",
		Plataforma:     "epublica_portal",
		PortalURL:      base,
		TotalRegistros: total,
		Contratos:      items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}
