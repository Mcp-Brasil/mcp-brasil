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
// Memory / OpenKnowledge (plataforma = "memory_openknowledge")
// Base padrão: https://publico.memory.com.br/openknowledge
// Portal SPA:  https://ilai.memory.com.br/#/entidades/login/<TENANT>/<ENTIDADE>
// ---------------------------------------------------------------------------

const defaultMemoryAPIBaseURL = "https://publico.memory.com.br/openknowledge"

var (
	memoryBaseMu     sync.RWMutex
	memoryAPIBaseURL = defaultMemoryAPIBaseURL

	memoryTenantRegex = regexp.MustCompile(`(?i)ilai\.memory\.com\.br/#/entidades/login/([A-Z0-9]{6})/(\d+)`)
)

// SetMemoryAPIBaseURL configura a URL base da API Memory OpenKnowledge (útil para testes).
func SetMemoryAPIBaseURL(u string) {
	memoryBaseMu.Lock()
	defer memoryBaseMu.Unlock()
	trimmed := strings.TrimRight(strings.TrimSpace(u), "/")
	if trimmed == "" {
		memoryAPIBaseURL = defaultMemoryAPIBaseURL
		return
	}
	memoryAPIBaseURL = trimmed
}

func getMemoryAPIBaseURL() string {
	memoryBaseMu.RLock()
	defer memoryBaseMu.RUnlock()
	return strings.TrimRight(strings.TrimSpace(memoryAPIBaseURL), "/")
}

func isMemoryILAIURL(rawURL string) bool {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return strings.Contains(strings.ToLower(rawURL), "ilai.memory.com.br")
	}
	host := strings.ToLower(u.Hostname())
	return host == "ilai.memory.com.br" || strings.HasSuffix(host, ".ilai.memory.com.br")
}

func resolveMemoryBaseURL(p PortalQueryParams) string {
	raw := strings.TrimSpace(p.BaseURL)
	if raw != "" && !isMemoryILAIURL(raw) {
		if idx := strings.Index(raw, "#"); idx >= 0 {
			raw = raw[:idx]
		}
		raw = strings.TrimRight(strings.TrimSpace(raw), "/")
		if raw != "" {
			return raw
		}
	}
	return getMemoryAPIBaseURL()
}

// ExtractMemoryTenant extrai (tenantID, entidade) de URLs ou HTML contendo
// ilai.memory.com.br/#/entidades/login/([A-Z0-9]{6})/(\d+).
func ExtractMemoryTenant(urlOrHTML string) (tenantID string, entidade string) {
	m := memoryTenantRegex.FindStringSubmatch(urlOrHTML)
	if len(m) < 3 {
		return "", ""
	}
	return strings.ToUpper(strings.TrimSpace(m[1])), strings.TrimSpace(m[2])
}

func resolveMemoryTenantAndEntidade(p PortalQueryParams) (tenantID string, entidade string, err error) {
	rawTenant := strings.TrimSpace(p.TenantID)
	if rawTenant != "" {
		if t, e := ExtractMemoryTenant(rawTenant); t != "" {
			tenantID = t
			entidade = e
		} else if strings.Contains(rawTenant, "/") {
			parts := strings.SplitN(rawTenant, "/", 2)
			tenantID = strings.TrimSpace(parts[0])
			entidade = strings.TrimSpace(parts[1])
		} else if strings.Contains(rawTenant, ":") {
			parts := strings.SplitN(rawTenant, ":", 2)
			tenantID = strings.TrimSpace(parts[0])
			entidade = strings.TrimSpace(parts[1])
		} else {
			tenantID = rawTenant
		}
	}

	if tenantID == "" && strings.TrimSpace(p.BaseURL) != "" {
		if t, e := ExtractMemoryTenant(p.BaseURL); t != "" {
			tenantID = t
			if entidade == "" {
				entidade = e
			}
		}
	}

	if entidade == "" && p.CodAdministracao > 0 {
		entidade = strconv.Itoa(p.CodAdministracao)
	}
	if entidade == "" {
		entidade = "1"
	}

	if tenantID == "" {
		return "", "", fmt.Errorf("Memory OpenKnowledge: tenant-id não informado")
	}
	return tenantID, entidade, nil
}

func doMemoryGet(ctx context.Context, client *http.Client, targetURL, tenantID, entidade string, exercicio int, extraHeaders map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("tenant-id", tenantID)
	req.Header.Set("entidade", entidade)
	req.Header.Set("exercicio", strconv.Itoa(exercicio))
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
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

func isValidMemoryJSON(body []byte) bool {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || (!strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[")) {
		return false
	}
	var v any
	if err := json.Unmarshal([]byte(trimmed), &v); err != nil || v == nil {
		return false
	}
	return true
}

// ProbeMemory verifica se a API Memory OpenKnowledge responde para o tenant e entidade informados
// via GET <apiBase>/lai/pessoal/estruturaorganizacional/.
func ProbeMemory(ctx context.Context, client *http.Client, tenantID, entidade string) bool {
	tid := strings.TrimSpace(tenantID)
	if tid == "" {
		return false
	}
	ent := strings.TrimSpace(entidade)
	if ent == "" {
		ent = "1"
	}

	apiBase := getMemoryAPIBaseURL()
	target := apiBase + "/lai/pessoal/estruturaorganizacional/"
	body, status, err := doMemoryGet(ctx, client, target, tid, ent, time.Now().Year(), nil)
	if err != nil || status != http.StatusOK {
		return false
	}
	return isValidMemoryJSON(body)
}

func resolveMemoryYear(p PortalQueryParams) int {
	if p.Ano > 0 {
		return p.Ano
	}
	if y := parseYearFromDateStr(p.Inicio); y > 0 {
		return y
	}
	return time.Now().Year()
}

func fetchMemoryList(
	ctx context.Context,
	client *http.Client,
	p PortalQueryParams,
	endpointPath string,
	extraHeaders map[string]string,
	extraKeys ...string,
) (rows []map[string]any, total int, baseURL string, ano int, err error) {
	tenantID, entidade, err := resolveMemoryTenantAndEntidade(p)
	if err != nil {
		return nil, 0, "", 0, err
	}
	baseURL = resolveMemoryBaseURL(p)
	ano = resolveMemoryYear(p)
	limit := normalizeLimit(p.Limit)

	q := url.Values{}
	q.Set("page", "1")
	q.Set("size", strconv.Itoa(limit))
	target := fmt.Sprintf("%s%s?%s", baseURL, endpointPath, q.Encode())

	body, status, reqErr := doMemoryGet(ctx, client, target, tenantID, entidade, ano, extraHeaders)
	if reqErr != nil {
		return nil, 0, baseURL, ano, reqErr
	}
	if status != http.StatusOK {
		return nil, 0, baseURL, ano, fmt.Errorf("Memory OpenKnowledge %s retornou HTTP %d", endpointPath, status)
	}

	keys := append([]string{"results", "servidores", "despesas", "empenhos", "receitas", "contratos", "licitacoes"}, extraKeys...)
	rows, total, decErr := decodeFlexiblePortalList(body, keys...)
	if decErr != nil {
		return nil, 0, baseURL, ano, fmt.Errorf("erro ao decodificar Memory OpenKnowledge %s: %w", endpointPath, decErr)
	}
	return rows, total, baseURL, ano, nil
}

// QueryMemoryServidores consulta servidores públicos no endpoint Memory /lai/pessoal/servidor/.
func QueryMemoryServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ServidoresResult, error) {
	rows, total, baseURL, ano, err := fetchMemoryList(ctx, client, p, "/lai/pessoal/servidor/", nil)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ServidorItem, 0, minInt(limit, len(rows)))

	for _, r := range rows {
		nome := mapGetStr(r, "nome", "Nome", "nomeServidor", "servidor", "funcionario")
		if filtro != "" && !strings.Contains(strings.ToLower(nome), filtro) {
			continue
		}
		comp := mapGetStr(r, "competencia", "Competencia", "mesAno", "referencia")
		if comp == "" {
			mes := mapGetStr(r, "mes", "Mes", "mesCompetencia")
			if mes != "" {
				comp = fmt.Sprintf("%s/%04d", mes, ano)
			} else {
				comp = strconv.Itoa(ano)
			}
		}

		items = append(items, ServidorItem{
			Nome:           nome,
			Matricula:      mapGetStr(r, "matricula", "Matricula", "registro", "codigo"),
			CPFParcial:     maskCPF(mapGetStr(r, "cpfFormatado", "cpfParcial", "cpf_parcial"), mapGetStr(r, "cpf", "CPF", "numCpf", "documento")),
			Cargo:          mapGetStr(r, "cargo", "Cargo", "descricaoCargo", "nomeCargo"),
			Funcao:         mapGetStr(r, "funcao", "Funcao", "descricaoFuncao"),
			Lotacao:        mapGetStr(r, "lotacao", "Lotacao", "descricaoLotacao", "departamento", "setor", "unidade"),
			LocalTrabalho:  mapGetStr(r, "localTrabalho", "local_trabalho", "local"),
			Vinculo:        mapGetStr(r, "vinculo", "Vinculo", "regime", "tipoVinculo"),
			Situacao:       mapGetStr(r, "situacao", "Situacao", "status"),
			SalarioBase:    mapGetFloat(r, "salarioBase", "salario_base", "vencimentoBase", "base"),
			SalarioBruto:   mapGetFloat(r, "salarioBruto", "salario_bruto", "totalProventos", "proventos", "bruto", "remuneracao"),
			Descontos:      mapGetFloat(r, "descontos", "Descontos", "totalDescontos", "valorDescontos"),
			SalarioLiquido: mapGetFloat(r, "salarioLiquido", "salario_liquido", "valorLiquido", "liquido"),
			Competencia:    comp,
			CargaHoraria:   mapGetStr(r, "cargaHoraria", "carga_horaria", "horasSemanais"),
			DataAdmissao:   mapGetStr(r, "dataAdmissao", "data_admissao", "admissao"),
			Orgao:          mapGetStr(r, "orgao", "Orgao", "nomeOrgao", "entidade", "estruturaOrganizacional"),
			Fonte:          "Portal Municipal (Memory OpenKnowledge)",
			PortalURL:      baseURL,
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
		Fonte:          "Portal Municipal (Memory OpenKnowledge)",
		Plataforma:     "memory_openknowledge",
		PortalURL:      baseURL,
		TotalRegistros: total,
		Servidores:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryMemoryDespesas consulta empenhos/despesas no endpoint Memory /lai/contabilidade/despesa/empenho.
func QueryMemoryDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	rows, total, baseURL, ano, err := fetchMemoryList(ctx, client, p, "/lai/contabilidade/despesa/empenho", nil)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]DespesaItem, 0, minInt(limit, len(rows)))

	for _, r := range rows {
		fornecedor := mapGetStr(r, "fornecedor", "Fornecedor", "credor", "nomeCredor", "nomeFornecedor", "razaoSocial")
		historico := mapGetStr(r, "historico", "Historico", "descricao", "especificacao")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+historico), filtro) {
			continue
		}
		itemAno := int(mapGetFloat(r, "exercicio", "ano", "Ano"))
		if itemAno == 0 {
			itemAno = ano
		}
		doc := mapGetStr(r, "documentoFornecedor", "documento", "cpfCnpj", "cnpjCpf", "cnpj", "cpf")

		items = append(items, DespesaItem{
			NumeroEmpenho:       mapGetStr(r, "numeroEmpenho", "numero_empenho", "empenho", "numero"),
			Ano:                 itemAno,
			Data:                mapGetStr(r, "data", "dataEmpenho", "data_empenho", "dataMovimento"),
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(doc),
			Orgao:               mapGetStr(r, "orgao", "Orgao", "nomeOrgao", "unidadeGestora"),
			Unidade:             mapGetStr(r, "unidade", "Unidade", "unidadeOrcamentaria", "nomeUnidade"),
			Funcao:              mapGetStr(r, "funcao", "Funcao", "descricaoFuncao"),
			Elemento:            mapGetStr(r, "elemento", "elementoDespesa", "naturezaDespesa", "classificacao"),
			Historico:           historico,
			ValorEmpenhado:      mapGetFloat(r, "valorEmpenhado", "valor_empenhado", "empenhado", "valor"),
			ValorLiquidado:      mapGetFloat(r, "valorLiquidado", "valor_liquidado", "liquidado"),
			ValorPago:           mapGetFloat(r, "valorPago", "valor_pago", "pago"),
			Fonte:               "Portal Municipal (Memory OpenKnowledge)",
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
		msg = "Nenhuma despesa encontrada para os filtros e período consultados."
	}

	return &DespesasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Memory OpenKnowledge)",
		Plataforma:     "memory_openknowledge",
		PortalURL:      baseURL,
		TotalRegistros: total,
		Despesas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryMemoryReceitas consulta receitas no endpoint Memory /lai/contabilidade/receita/ com header codigo_receita: "".
func QueryMemoryReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	headers := map[string]string{
		"codigo_receita": "",
	}
	rows, total, baseURL, _, err := fetchMemoryList(ctx, client, p, "/lai/contabilidade/receita/", headers)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ReceitaItem, 0, minInt(limit, len(rows)))

	for _, r := range rows {
		desc := mapGetStr(r, "descricao", "Descricao", "especificacao", "nome", "receita")
		if filtro != "" && !strings.Contains(strings.ToLower(desc), filtro) {
			continue
		}
		items = append(items, ReceitaItem{
			Codigo:          mapGetStr(r, "codigo", "codigo_receita", "codigoReceita", "classificacao", "conta"),
			Descricao:       desc,
			Categoria:       mapGetStr(r, "categoria", "Categoria", "fonteRecurso", "fonte", "origem"),
			Data:            mapGetStr(r, "data", "Data", "competencia"),
			ValorPrevisto:   mapGetFloat(r, "valorPrevisto", "valor_previsto", "previsto", "orcado", "valorOrcado"),
			ValorArrecadado: mapGetFloat(r, "valorArrecadado", "valor_arrecadado", "arrecadado", "realizado", "valorRealizado"),
			Fonte:           "Portal Municipal (Memory OpenKnowledge)",
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
		msg = "Nenhuma receita encontrada para o período consultado."
	}

	return &ReceitasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Memory OpenKnowledge)",
		Plataforma:     "memory_openknowledge",
		PortalURL:      baseURL,
		TotalRegistros: total,
		Receitas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryMemoryContratos consulta contratos no endpoint Memory /lai/compras/contrato/.
func QueryMemoryContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ContratosResult, error) {
	rows, total, baseURL, ano, err := fetchMemoryList(ctx, client, p, "/lai/compras/contrato/", nil)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ContratoItem, 0, minInt(limit, len(rows)))

	for _, r := range rows {
		fornecedor := mapGetStr(r, "fornecedor", "Fornecedor", "contratado", "nomeFornecedor", "razaoSocial")
		objeto := mapGetStr(r, "objeto", "Objeto", "descricao", "Descricao")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+objeto), filtro) {
			continue
		}
		itemAno := int(mapGetFloat(r, "exercicio", "ano", "Ano"))
		if itemAno == 0 {
			itemAno = ano
		}
		doc := mapGetStr(r, "documentoFornecedor", "documento", "cpfCnpj", "cnpjCpf", "cnpj", "cpf")

		items = append(items, ContratoItem{
			Numero:              mapGetStr(r, "numero", "numeroContrato", "numero_contrato", "codigo"),
			Ano:                 itemAno,
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(doc),
			Objeto:              objeto,
			Situacao:            mapGetStr(r, "situacao", "Situacao", "status"),
			DataAssinatura:      mapGetStr(r, "dataAssinatura", "data_assinatura", "data"),
			VigenciaInicio:      mapGetStr(r, "vigenciaInicio", "vigencia_inicio", "dataInicio"),
			VigenciaFim:         mapGetStr(r, "vigenciaFim", "vigencia_fim", "dataFim", "dataTermino"),
			Valor:               mapGetFloat(r, "valor", "Valor", "valorContratado", "valorTotal", "valorGlobal"),
			Fonte:               "Portal Municipal (Memory OpenKnowledge)",
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
		msg = "Nenhum contrato encontrado para os filtros e período consultados."
	}

	return &ContratosResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Memory OpenKnowledge)",
		Plataforma:     "memory_openknowledge",
		PortalURL:      baseURL,
		TotalRegistros: total,
		Contratos:      items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryMemoryLicitacoes consulta licitações no endpoint Memory /lai/compras/licitacao/.
func QueryMemoryLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LicitacoesResult, error) {
	rows, total, baseURL, ano, err := fetchMemoryList(ctx, client, p, "/lai/compras/licitacao/", nil)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]LicitacaoItem, 0, minInt(limit, len(rows)))

	for _, r := range rows {
		objeto := mapGetStr(r, "objeto", "Objeto", "descricao", "Descricao")
		modalidade := mapGetStr(r, "modalidade", "Modalidade", "tipo", "descricaoModalidade")
		if filtro != "" && !strings.Contains(strings.ToLower(objeto+" "+modalidade), filtro) {
			continue
		}
		itemAno := int(mapGetFloat(r, "exercicio", "ano", "Ano"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, LicitacaoItem{
			Numero:          mapGetStr(r, "numero", "numeroLicitacao", "numero_licitacao", "edital", "codigo"),
			Ano:             itemAno,
			Modalidade:      modalidade,
			Situacao:        mapGetStr(r, "situacao", "Situacao", "status"),
			DataAbertura:    mapGetStr(r, "dataAbertura", "data_abertura", "dataPublicacao", "data"),
			Objeto:          objeto,
			ValorEstimado:   mapGetFloat(r, "valorEstimado", "valor_estimado", "valorMaximo", "valorPrevisto", "valor"),
			ValorHomologado: mapGetFloat(r, "valorHomologado", "valor_homologado", "valorVencedor", "valorTotal"),
			Link:            mapGetStr(r, "link", "url", "Link"),
			Fonte:           "Portal Municipal (Memory OpenKnowledge)",
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
		msg = "Nenhuma licitação encontrada para os filtros e período consultados."
	}

	return &LicitacoesResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Memory OpenKnowledge)",
		Plataforma:     "memory_openknowledge",
		PortalURL:      baseURL,
		TotalRegistros: total,
		Licitacoes:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}
