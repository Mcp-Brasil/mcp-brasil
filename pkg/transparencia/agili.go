package transparencia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Ágili Software (transparencia.agilicloud.com.br) Connector
// ---------------------------------------------------------------------------

const (
	defaultAgiliAPIBaseURL = "https://transparencia.agilicloud.com.br/api"
	agiliPortalFrontendURL = "https://transparencia.agilicloud.com.br"

	agiliDTOServidores = "Agili.Blue.Portal.Shared.GestaoPessoas.Dto.Servidores.ServidoresGridDto"
	agiliDTODespesas   = "Agili.Blue.Portal.Shared.Contabilidade.Dto.Despesas.EmpenhosPorExercicio.EmpenhosPorExercicioGridDto"
	agiliDTOReceitas   = "Agili.Blue.Portal.Shared.Contabilidade.Dto.Receitas.ReceitasPorPeriodoGridDto"
	agiliDTOLicitacoes = "Agili.Blue.Portal.Shared.ComprasLicitacoes.Dto.Licitacoes.LicitacaoGridDto"
	agiliDTOContratos  = "Agili.Blue.Portal.Shared.ComprasLicitacoes.Dto.Contratos.ContratoGridDto"
	agiliDTOObras      = "Agili.Blue.Transparencia.Shared.ComprasLicitacoes.Dto.Obras.ObrasPublicasGridDto"

	agiliFallbackServidores = "/gestaopessoas/servidores/obterDadosServidores"
	agiliFallbackDespesas   = "/contabilidade/despesas/empenhosPorExercicio/obterDadosEmpenhosPorExercicio"
	agiliFallbackReceitas   = "/contabilidade/receitas/receitasPorPeriodo/obterDadosReceitasPorPeriodo"
	agiliFallbackLicitacoes = "/compraslicitacoes/licitacoes/licitacao/obterDadosLicitacao"
	agiliFallbackContratos  = "/compraslicitacoes/contratos/contrato/obterDadosContrato"
	agiliFallbackObras      = "/compraslicitacoes/obras/obrasPublicas/obterDadosObrasPublicas"
)

var (
	agiliURLMu      sync.RWMutex
	agiliAPIBaseURL = defaultAgiliAPIBaseURL
)

// SetAgiliAPIBaseURL configura a URL base da API Ágili Blue (útil para testes).
func SetAgiliAPIBaseURL(u string) {
	agiliURLMu.Lock()
	agiliAPIBaseURL = strings.TrimRight(strings.TrimSpace(u), "/")
	agiliURLMu.Unlock()
	ResetAgiliSeletorCache()
}

func getAgiliAPIBaseURL() string {
	agiliURLMu.RLock()
	defer agiliURLMu.RUnlock()
	return agiliAPIBaseURL
}

type agiliSeletorEntry struct {
	Shortname       string `json:"shortname"`
	Municipio       string `json:"municipio"`
	UF              string `json:"uf"`
	NomeRazaoSocial string `json:"nomeRazaoSocial"`
	TipoOrgao       string `json:"tipoOrgao"`
	CNPJ            string `json:"cnpj"`
}

type agiliSeletorCache struct {
	mu        sync.RWMutex
	baseURL   string
	entries   []agiliSeletorEntry
	expiresAt time.Time
}

var agiliSeletor = &agiliSeletorCache{}

// ResetAgiliSeletorCache limpa o cache em memória do diretório /seletor/get da Ágili.
func ResetAgiliSeletorCache() {
	agiliSeletor.mu.Lock()
	agiliSeletor.baseURL = ""
	agiliSeletor.entries = nil
	agiliSeletor.expiresAt = time.Time{}
	agiliSeletor.mu.Unlock()
}

func fetchAgiliSeletor(ctx context.Context, client *http.Client, apiBase string) ([]agiliSeletorEntry, error) {
	agiliSeletor.mu.RLock()
	if agiliSeletor.baseURL == apiBase && len(agiliSeletor.entries) > 0 && time.Now().Before(agiliSeletor.expiresAt) {
		entries := agiliSeletor.entries
		agiliSeletor.mu.RUnlock()
		return entries, nil
	}
	agiliSeletor.mu.RUnlock()

	httpClient := bethaHTTPClient(client)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiBase+"/seletor/get", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ágili /seletor/get retornou HTTP %d", resp.StatusCode)
	}

	// O payload completo do seletor nacional pode ser grande; limita leitura a 32 MiB.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, nil
	}

	var entries []agiliSeletorEntry
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &entries); err != nil {
			return nil, fmt.Errorf("erro ao decodificar /seletor/get Ágili: %w", err)
		}
	} else {
		var env struct {
			Dados   []agiliSeletorEntry `json:"dados"`
			Data    []agiliSeletorEntry `json:"data"`
			Items   []agiliSeletorEntry `json:"items"`
			Content []agiliSeletorEntry `json:"content"`
		}
		if err := json.Unmarshal([]byte(trimmed), &env); err != nil {
			return nil, fmt.Errorf("erro ao decodificar /seletor/get Ágili: %w", err)
		}
		for _, s := range [][]agiliSeletorEntry{env.Dados, env.Data, env.Items, env.Content} {
			if len(s) > 0 {
				entries = s
				break
			}
		}
	}

	agiliSeletor.mu.Lock()
	agiliSeletor.baseURL = apiBase
	agiliSeletor.entries = entries
	agiliSeletor.expiresAt = time.Now().Add(15 * time.Minute)
	agiliSeletor.mu.Unlock()

	return entries, nil
}

// DiscoverAgiliByMunicipio localiza o tenant Ágili Blue correspondente ao município informado.
func DiscoverAgiliByMunicipio(ctx context.Context, client *http.Client, m *Municipio) (*PortalDiscoveryResult, bool) {
	if m == nil || strings.TrimSpace(m.Nome) == "" {
		return nil, false
	}

	apiBase := getAgiliAPIBaseURL()
	entries, err := fetchAgiliSeletor(ctx, client, apiBase)
	if err != nil || len(entries) == 0 {
		return nil, false
	}

	targetNome := RemoveAccents(strings.ToLower(strings.TrimSpace(m.Nome)))
	targetUF := strings.ToUpper(strings.TrimSpace(m.UF))

	var bestEntry *agiliSeletorEntry
	bestScore := -1

	for i := range entries {
		e := &entries[i]
		shortname := strings.TrimSpace(e.Shortname)
		if shortname == "" {
			continue
		}

		entryUF := strings.ToUpper(strings.TrimSpace(e.UF))
		if targetUF != "" && entryUF != "" && entryUF != targetUF {
			continue
		}

		entryMun := RemoveAccents(strings.ToLower(strings.TrimSpace(e.Municipio)))
		entryRazao := RemoveAccents(strings.ToLower(strings.TrimSpace(e.NomeRazaoSocial)))

		score := -1
		if entryMun != "" && entryMun == targetNome {
			score = 100
		} else if entryRazao != "" && strings.Contains(entryRazao, targetNome) {
			score = 50
		}
		if score < 0 {
			continue
		}

		if strings.Contains(entryRazao, "prefeitura") || strings.Contains(entryRazao, "municipio") {
			score += 30
		} else if strings.Contains(entryRazao, "camara") || strings.Contains(entryRazao, "previdencia") || strings.Contains(entryRazao, "fundo") {
			score -= 10
		}
		if entryUF == targetUF && targetUF != "" {
			score += 10
		}

		if score > bestScore {
			bestScore = score
			bestEntry = e
		}
	}

	if bestEntry == nil {
		return nil, false
	}

	entidade := strings.TrimSpace(bestEntry.NomeRazaoSocial)
	if entidade == "" {
		entidade = strings.TrimSpace(bestEntry.Municipio)
	}

	return &PortalDiscoveryResult{
		Municipio:           m,
		URLResolvida:        agiliPortalFrontendURL,
		Plataforma:          "agili_blue",
		TenantID:            strings.TrimSpace(bestEntry.Shortname),
		EntidadeNome:        entidade,
		FonteDescoberta:     "agili_directory",
		ServidoresSuportado: true,
	}, true
}

func resolveAgiliContext(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (apiBase string, shortname string, portalURL string, err error) {
	apiBase = getAgiliAPIBaseURL()
	if apiBase == defaultAgiliAPIBaseURL && strings.TrimSpace(p.BaseURL) != "" {
		b := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
		if !strings.Contains(b, "transparencia.agilicloud.com.br") {
			apiBase = b
		}
	}

	shortname = strings.TrimSpace(p.TenantID)
	if shortname == "" && m != nil {
		if disc, ok := DiscoverAgiliByMunicipio(ctx, client, m); ok && disc != nil {
			shortname = disc.TenantID
		}
	}
	if shortname == "" {
		return "", "", "", fmt.Errorf("TenantID (shortname uc) Ágili não informado e não descoberto")
	}

	portalURL = agiliPortalFrontendURL
	if strings.TrimSpace(p.BaseURL) != "" {
		portalURL = strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	}
	return apiBase, shortname, portalURL, nil
}

func doAgiliRequest(ctx context.Context, client *http.Client, method, targetURL, shortname string, payload any) ([]byte, int, error) {
	httpClient := bethaHTTPClient(client)

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
	req.Header.Set("uc", shortname)
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	req.Header.Set("Accept", "application/json")
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

type agiliFiltroMeta struct {
	URLStore       string
	FieldsSelected any
}

func fetchAgiliFiltroMeta(ctx context.Context, client *http.Client, apiBase, shortname, dtoModel string) agiliFiltroMeta {
	u := fmt.Sprintf("%s/filtro/get/?model=%s", apiBase, url.QueryEscape(dtoModel))
	body, status, err := doAgiliRequest(ctx, client, http.MethodGet, u, shortname, nil)
	if err != nil || status != http.StatusOK || len(strings.TrimSpace(string(body))) == 0 {
		return agiliFiltroMeta{}
	}

	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return agiliFiltroMeta{}
	}

	meta := agiliFiltroMeta{
		URLStore: mapGetStr(obj, "urlStore", "UrlStore", "url"),
	}
	for _, k := range []string{"fieldsSelected", "FieldsSelected", "fields"} {
		if v, ok := obj[k]; ok && v != nil {
			meta.FieldsSelected = v
			break
		}
	}
	return meta
}

func decodeAgiliRows(body []byte) ([]map[string]any, int, error) {
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
	for _, key := range []string{"dados", "data", "content", "items", "rows", "registros", "Dados", "Data", "Items"} {
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

	total := int(mapGetFloat(obj, "total", "totalCount", "totalElements", "count", "Total", "TotalCount"))
	if total == 0 && len(rows) > 0 {
		total = len(rows)
	}
	return rows, total, nil
}

func executeAgiliGridQuery(ctx context.Context, client *http.Client, apiBase, shortname, dtoModel, fallbackURLStore string, p PortalQueryParams) ([]map[string]any, int, error) {
	limit := normalizeLimit(p.Limit)
	pageSize := normalizePageSize(limit, 200)
	meta := fetchAgiliFiltroMeta(ctx, client, apiBase, shortname, dtoModel)

	urlStore := strings.TrimSpace(meta.URLStore)
	if urlStore == "" {
		urlStore = fallbackURLStore
	}
	urlStore = "/" + strings.Trim(urlStore, "/")

	payload := map[string]any{}
	if meta.FieldsSelected != nil {
		payload["fieldsSelected"] = meta.FieldsSelected
	}
	if p.Ano > 0 {
		payload["Ano"] = p.Ano
	}
	if filtro := strings.TrimSpace(p.NomeFiltro); filtro != "" {
		payload["filtro"] = filtro
	}

	var allRows []map[string]any
	total := 0
	prevBody := ""
	for pageIdx := 0; pageIdx < 10; pageIdx++ {
		targetURL := fmt.Sprintf("%s%s/?model=%s&page=%d&size=%d&withCount=true", apiBase, urlStore, url.QueryEscape(dtoModel), pageIdx, pageSize)
		body, status, err := doAgiliRequest(ctx, client, http.MethodPost, targetURL, shortname, payload)
		if err != nil {
			if pageIdx == 0 {
				return nil, 0, err
			}
			break
		}
		if status != http.StatusOK {
			if pageIdx == 0 {
				return nil, 0, fmt.Errorf("Ágili %s retornou HTTP %d", urlStore, status)
			}
			break
		}
		bodyStr := strings.TrimSpace(string(body))
		if pageIdx > 0 && bodyStr == prevBody {
			break
		}
		prevBody = bodyStr

		rows, pageTotal, err := decodeAgiliRows(body)
		if err != nil {
			if pageIdx == 0 {
				return nil, 0, err
			}
			break
		}
		if pageIdx == 0 {
			total = pageTotal
		}
		if len(rows) == 0 {
			break
		}
		allRows = append(allRows, rows...)
		if len(allRows) >= limit || len(rows) < pageSize || (total > 0 && len(allRows) >= total) {
			break
		}
	}
	if len(allRows) > limit {
		allRows = allRows[:limit]
	}
	return allRows, total, nil
}

func agiliGetStr(row map[string]any, keys ...string) string {
	return bethaGetStr(row, keys...)
}

func agiliGetFloat(row map[string]any, keys ...string) float64 {
	return bethaGetFloat(row, keys...)
}

// QueryAgiliServidores consulta servidores públicos em um portal Ágili Blue.
func QueryAgiliServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ServidoresResult, error) {
	apiBase, shortname, portalURL, err := resolveAgiliContext(ctx, client, m, p)
	if err != nil {
		return nil, err
	}

	rows, total, err := executeAgiliGridQuery(ctx, client, apiBase, shortname, agiliDTOServidores, agiliFallbackServidores, p)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	items := make([]ServidorItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		if len(items) >= limit {
			break
		}
		bruto := agiliGetFloat(r, "salarioBruto", "valorBruto", "totalProventos", "proventos", "remuneracao", "vencimentos")
		descontos := agiliGetFloat(r, "descontos", "totalDescontos", "valorDescontos")
		liquido := agiliGetFloat(r, "salarioLiquido", "valorLiquido", "liquido")
		if liquido == 0 && bruto > 0 && descontos > 0 {
			liquido = bruto - descontos
		}
		items = append(items, ServidorItem{
			Nome:           agiliGetStr(r, "nome", "nomeServidor", "servidor", "funcionario"),
			Matricula:      agiliGetStr(r, "matricula", "registro", "codigo"),
			CPFParcial:     maskCPF(agiliGetStr(r, "cpfFormatado", "cpf_formatado"), agiliGetStr(r, "cpf", "documento", "cpfServidor")),
			Cargo:          agiliGetStr(r, "cargo", "descricaoCargo", "cargoEfetivo"),
			Funcao:         agiliGetStr(r, "funcao", "descricaoFuncao"),
			Lotacao:        agiliGetStr(r, "lotacao", "descricaoLotacao", "departamento", "setor"),
			LocalTrabalho:  agiliGetStr(r, "localTrabalho", "local"),
			Vinculo:        agiliGetStr(r, "vinculo", "tipoVinculo", "regime"),
			Situacao:       agiliGetStr(r, "situacao", "status"),
			SalarioBase:    agiliGetFloat(r, "salarioBase", "vencimentoBase", "base"),
			SalarioBruto:   bruto,
			Descontos:      descontos,
			SalarioLiquido: liquido,
			Competencia:    agiliGetStr(r, "competencia", "mesAno", "referencia"),
			CargaHoraria:   agiliGetStr(r, "cargaHoraria", "horasSemanais"),
			DataAdmissao:   agiliGetStr(r, "dataAdmissao", "admissao"),
			Orgao:          agiliGetStr(r, "orgao", "entidade", "unidade"),
			Fonte:          "Portal Municipal (Ágili Blue)",
			PortalURL:      portalURL,
		})
	}

	return &ServidoresResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Ágili Blue)",
		Plataforma:     "agili_blue",
		PortalURL:      portalURL,
		TotalRegistros: total,
		Servidores:     items,
		Disponivel:     true,
	}, nil
}

// QueryAgiliDespesas consulta despesas e empenhos em um portal Ágili Blue.
func QueryAgiliDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	apiBase, shortname, portalURL, err := resolveAgiliContext(ctx, client, m, p)
	if err != nil {
		return nil, err
	}

	rows, total, err := executeAgiliGridQuery(ctx, client, apiBase, shortname, agiliDTODespesas, agiliFallbackDespesas, p)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	items := make([]DespesaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		if len(items) >= limit {
			break
		}
		doc := agiliGetStr(r, "documentoFornecedor", "cpfCnpj", "cnpjCpf", "documento", "cpf", "cnpj", "cpfCnpjCredor")
		items = append(items, DespesaItem{
			NumeroEmpenho:       agiliGetStr(r, "numeroEmpenho", "empenho", "numero"),
			Ano:                 parseBethaInt(agiliGetStr(r, "ano", "exercicio"), p.Ano),
			Data:                agiliGetStr(r, "data", "dataEmpenho", "dataMovimento"),
			Fornecedor:          agiliGetStr(r, "fornecedor", "credor", "nomeCredor", "nomeFornecedor", "razaoSocial"),
			DocumentoFornecedor: maskDocIfCPF(doc),
			Orgao:               agiliGetStr(r, "orgao", "nomeOrgao"),
			Unidade:             agiliGetStr(r, "unidade", "unidadeOrcamentaria"),
			Funcao:              agiliGetStr(r, "funcao", "funcaoGoverno"),
			Elemento:            agiliGetStr(r, "elemento", "elementoDespesa", "natureza", "fonteRecurso"),
			Historico:           agiliGetStr(r, "historico", "descricao", "objeto"),
			ValorEmpenhado:      agiliGetFloat(r, "valorEmpenhado", "empenhado", "valor"),
			ValorLiquidado:      agiliGetFloat(r, "valorLiquidado", "liquidado"),
			ValorPago:           agiliGetFloat(r, "valorPago", "pago"),
			Fonte:               "Portal Municipal (Ágili Blue)",
		})
	}

	return &DespesasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Ágili Blue)",
		Plataforma:     "agili_blue",
		PortalURL:      portalURL,
		TotalRegistros: total,
		Despesas:       items,
		Disponivel:     true,
	}, nil
}

// QueryAgiliReceitas consulta receitas públicas em um portal Ágili Blue.
func QueryAgiliReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	apiBase, shortname, portalURL, err := resolveAgiliContext(ctx, client, m, p)
	if err != nil {
		return nil, err
	}

	rows, total, err := executeAgiliGridQuery(ctx, client, apiBase, shortname, agiliDTOReceitas, agiliFallbackReceitas, p)
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
			Codigo:          agiliGetStr(r, "codigo", "conta", "classificacao", "codigoReceita"),
			Descricao:       agiliGetStr(r, "descricao", "especificacao", "receita", "titulo"),
			Categoria:       agiliGetStr(r, "categoria", "fonteRecurso", "origem"),
			Data:            agiliGetStr(r, "data", "dataArrecadacao", "competencia"),
			ValorPrevisto:   agiliGetFloat(r, "valorPrevisto", "previsto", "orcado", "valorOrcado"),
			ValorArrecadado: agiliGetFloat(r, "valorArrecadado", "arrecadado", "realizado", "valorRealizado", "valor"),
			Fonte:           "Portal Municipal (Ágili Blue)",
		})
	}

	return &ReceitasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Ágili Blue)",
		Plataforma:     "agili_blue",
		PortalURL:      portalURL,
		TotalRegistros: total,
		Receitas:       items,
		Disponivel:     true,
	}, nil
}

// QueryAgiliLicitacoes consulta licitações em um portal Ágili Blue.
func QueryAgiliLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LicitacoesResult, error) {
	apiBase, shortname, portalURL, err := resolveAgiliContext(ctx, client, m, p)
	if err != nil {
		return nil, err
	}

	rows, total, err := executeAgiliGridQuery(ctx, client, apiBase, shortname, agiliDTOLicitacoes, agiliFallbackLicitacoes, p)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	items := make([]LicitacaoItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		if len(items) >= limit {
			break
		}
		link := agiliGetStr(r, "link", "url")
		if link == "" {
			link = portalURL
		}
		items = append(items, LicitacaoItem{
			Numero:          agiliGetStr(r, "numero", "numeroLicitacao", "licitacao", "processo"),
			Ano:             parseBethaInt(agiliGetStr(r, "ano", "exercicio"), p.Ano),
			Modalidade:      agiliGetStr(r, "modalidade", "descricaoModalidade"),
			Situacao:        agiliGetStr(r, "situacao", "status"),
			DataAbertura:    agiliGetStr(r, "dataAbertura", "abertura", "dataPublicacao", "data"),
			Objeto:          agiliGetStr(r, "objeto", "descricao"),
			ValorEstimado:   agiliGetFloat(r, "valorEstimado", "estimado", "valorMaximo", "valorPrevisto"),
			ValorHomologado: agiliGetFloat(r, "valorHomologado", "homologado", "valorVencedor", "valorTotal", "valor"),
			Link:            link,
			Fonte:           "Portal Municipal (Ágili Blue)",
		})
	}

	return &LicitacoesResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Ágili Blue)",
		Plataforma:     "agili_blue",
		PortalURL:      portalURL,
		TotalRegistros: total,
		Licitacoes:     items,
		Disponivel:     true,
	}, nil
}

// QueryAgiliContratos consulta contratos administrativos em um portal Ágili Blue.
func QueryAgiliContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ContratosResult, error) {
	apiBase, shortname, portalURL, err := resolveAgiliContext(ctx, client, m, p)
	if err != nil {
		return nil, err
	}

	rows, total, err := executeAgiliGridQuery(ctx, client, apiBase, shortname, agiliDTOContratos, agiliFallbackContratos, p)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	items := make([]ContratoItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		if len(items) >= limit {
			break
		}
		doc := agiliGetStr(r, "documentoFornecedor", "cpfCnpj", "cnpjCpf", "documento", "cpf", "cnpj")
		items = append(items, ContratoItem{
			Numero:              agiliGetStr(r, "numero", "numeroContrato", "contrato"),
			Ano:                 parseBethaInt(agiliGetStr(r, "ano", "exercicio"), p.Ano),
			Fornecedor:          agiliGetStr(r, "fornecedor", "contratado", "credor", "nomeFornecedor", "razaoSocial"),
			DocumentoFornecedor: maskDocIfCPF(doc),
			Objeto:              agiliGetStr(r, "objeto", "descricao"),
			Situacao:            agiliGetStr(r, "situacao", "status"),
			DataAssinatura:      agiliGetStr(r, "dataAssinatura", "assinatura", "data"),
			VigenciaInicio:      agiliGetStr(r, "vigenciaInicio", "dataInicio", "inicioVigencia"),
			VigenciaFim:         agiliGetStr(r, "vigenciaFim", "dataFim", "fimVigencia", "vencimento"),
			Valor:               agiliGetFloat(r, "valor", "valorContrato", "valorGlobal", "valorTotal"),
			Fonte:               "Portal Municipal (Ágili Blue)",
		})
	}

	return &ContratosResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Ágili Blue)",
		Plataforma:     "agili_blue",
		PortalURL:      portalURL,
		TotalRegistros: total,
		Contratos:      items,
		Disponivel:     true,
	}, nil
}

// QueryAgiliObras consulta obras públicas em um portal Ágili Blue.
func QueryAgiliObras(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ObrasResult, error) {
	apiBase, shortname, portalURL, err := resolveAgiliContext(ctx, client, m, p)
	if err != nil {
		return nil, err
	}

	rows, total, err := executeAgiliGridQuery(ctx, client, apiBase, shortname, agiliDTOObras, agiliFallbackObras, p)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	items := make([]ObraItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		if len(items) >= limit {
			break
		}
		titulo := agiliGetStr(r, "titulo", "nome", "descricao", "obra")
		objeto := agiliGetStr(r, "objeto", "descricaoDetalhada", "descricao")
		if titulo == "" {
			titulo = objeto
		}
		items = append(items, ObraItem{
			Codigo:      agiliGetStr(r, "codigo", "numero", "id"),
			Titulo:      titulo,
			Objeto:      objeto,
			Orgao:       agiliGetStr(r, "orgao", "unidade", "secretaria"),
			Endereco:    agiliGetStr(r, "endereco", "localizacao", "logradouro"),
			Bairro:      agiliGetStr(r, "bairro"),
			DataInicial: agiliGetStr(r, "dataInicial", "dataInicio", "inicio"),
			DataFinal:   agiliGetStr(r, "dataFinal", "dataFim", "previsaoTermino", "conclusao"),
			Situacao:    agiliGetStr(r, "situacao", "status"),
			Valor:       agiliGetFloat(r, "valor", "valorObra", "valorContratado", "valorTotal"),
			Fonte:       "Portal Municipal (Ágili Blue)",
		})
	}

	return &ObrasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Ágili Blue)",
		Plataforma:     "agili_blue",
		PortalURL:      portalURL,
		TotalRegistros: total,
		Obras:          items,
		Disponivel:     true,
	}, nil
}
