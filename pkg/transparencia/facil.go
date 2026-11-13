package transparencia

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Fácil Transparência (plataforma = "facil_transparencia")
// Base padrão: http://www.transparenciafacil.com.br
// Hosts alternativos: relatorios.portalfacil.com.br
// ---------------------------------------------------------------------------

const defaultFacilBaseURL = "http://www.transparenciafacil.com.br"

var (
	facilBaseMu  sync.RWMutex
	facilBaseURL = defaultFacilBaseURL

	facilDomainDBRegex = regexp.MustCompile(`(?i)(?:transparenciafacil|portalfacil)\.com\.br/(\d{5,8})\b`)
	facilQueryDBRegex  = regexp.MustCompile(`(?i)\bdatabaseId=(\d{5,8})\b`)
	facilPathSegRegex  = regexp.MustCompile(`^\d{5,8}$`)
	facilHTMLPathRegex = regexp.MustCompile(`/(\d{5,8})(?:[/"'?#\s<]|$)`)
)

// FacilEntidade representa uma entidade retornada por POST /api/geral/GetEntidades.
type FacilEntidade struct {
	DatabaseID string `json:"database_id"`
	Nome       string `json:"nome"`
	UF         string `json:"uf,omitempty"`
	CNPJ       string `json:"cnpj,omitempty"`
	Municipio  string `json:"municipio,omitempty"`
}

// SetFacilBaseURL configura a URL base do portal Fácil Transparência (útil para testes).
func SetFacilBaseURL(u string) {
	facilBaseMu.Lock()
	defer facilBaseMu.Unlock()
	trimmed := strings.TrimRight(strings.TrimSpace(u), "/")
	if trimmed == "" {
		facilBaseURL = defaultFacilBaseURL
		return
	}
	facilBaseURL = trimmed
}

func getFacilBaseURL() string {
	facilBaseMu.RLock()
	defer facilBaseMu.RUnlock()
	return strings.TrimRight(strings.TrimSpace(facilBaseURL), "/")
}

// ExtractFacilDatabaseID extrai o databaseId numérico de 5 a 8 dígitos de URLs ou HTML
// (ex: transparenciafacil.com.br/0144301 ou portalfacil.com.br/0144301).
func ExtractFacilDatabaseID(urlOrHTML string) string {
	trimmed := strings.TrimSpace(urlOrHTML)
	if trimmed == "" {
		return ""
	}
	if m := facilDomainDBRegex.FindStringSubmatch(trimmed); len(m) >= 2 {
		return m[1]
	}
	if m := facilQueryDBRegex.FindStringSubmatch(trimmed); len(m) >= 2 {
		return m[1]
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "http://") || strings.HasPrefix(strings.ToLower(trimmed), "https://") {
		if u, err := url.Parse(trimmed); err == nil {
			for _, seg := range strings.Split(strings.Trim(u.Path, "/"), "/") {
				seg = strings.TrimSpace(seg)
				if facilPathSegRegex.MatchString(seg) {
					return seg
				}
			}
		}
	}
	if m := facilHTMLPathRegex.FindStringSubmatch(trimmed); len(m) >= 2 {
		return m[1]
	}
	return ""
}

func resolveFacilBaseURL(rawBaseURL string) string {
	configured := getFacilBaseURL()
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
	if configured != defaultFacilBaseURL && strings.Contains(host, "transparenciafacil.com.br") {
		return configured
	}

	// Remove o segmento /<databaseId> ou /api/... caso esteja presente na URL fornecida
	p := strings.TrimRight(u.Path, "/")
	lowP := strings.ToLower(p)
	if idx := strings.Index(lowP, "/api/"); idx >= 0 {
		p = p[:idx]
	} else if dbID := ExtractFacilDatabaseID(raw); dbID != "" {
		suffix := "/" + dbID
		if strings.HasSuffix(p, suffix) {
			p = strings.TrimSuffix(p, suffix)
		} else if idx := strings.Index(p, suffix+"/"); idx >= 0 {
			p = p[:idx]
		}
	}

	return strings.TrimRight(u.Scheme+"://"+u.Host+p, "/")
}

// FetchFacilEntidades lista entidades disponíveis via POST <base>/api/geral/GetEntidades?search=<q>.
func FetchFacilEntidades(ctx context.Context, client *http.Client, search string) ([]FacilEntidade, error) {
	base := getFacilBaseURL()
	target := fmt.Sprintf("%s/api/geral/GetEntidades?search=%s", base, url.QueryEscape(strings.TrimSpace(search)))
	body, status, err := doPortalJSONPost(ctx, client, target, map[string]any{})
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Fácil Transparência /api/geral/GetEntidades retornou HTTP %d", status)
	}

	rows, _, err := decodeFlexiblePortalList(body, "entidades", "Entidades", "data", "Data")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Fácil GetEntidades: %w", err)
	}

	out := make([]FacilEntidade, 0, len(rows))
	for _, r := range rows {
		dbID := mapGetStr(r, "databaseId", "DatabaseId", "database_id", "DatabaseID", "id", "Id", "codigo", "Codigo")
		nome := mapGetStr(r, "nome", "Nome", "entidade", "Entidade", "descricao", "Descricao", "razaoSocial")
		if dbID == "" && nome == "" {
			continue
		}
		out = append(out, FacilEntidade{
			DatabaseID: dbID,
			Nome:       nome,
			UF:         mapGetStr(r, "uf", "UF", "estado", "Estado"),
			CNPJ:       mapGetStr(r, "cnpj", "CNPJ"),
			Municipio:  mapGetStr(r, "municipio", "Municipio", "cidade", "Cidade"),
		})
	}
	return out, nil
}

// ProbeFacil verifica um portal Fácil Transparência via POST <base>/api/receita/ReceitasGetAno?databaseId=<id>
// esperando um array JSON válido.
func ProbeFacil(ctx context.Context, client *http.Client, baseURL, dbID string) bool {
	id := strings.TrimSpace(dbID)
	if id == "" {
		id = ExtractFacilDatabaseID(baseURL)
	}
	if id == "" {
		return false
	}

	base := resolveFacilBaseURL(baseURL)
	if base == "" {
		return false
	}

	target := fmt.Sprintf("%s/api/receita/ReceitasGetAno?databaseId=%s", base, url.QueryEscape(id))
	body, status, err := doPortalJSONPost(ctx, client, target, map[string]any{})
	if err != nil || status != http.StatusOK {
		return false
	}

	trimmed := strings.TrimSpace(string(body))
	if !strings.HasPrefix(trimmed, "[") {
		return false
	}
	var anos []any
	if err := json.Unmarshal([]byte(trimmed), &anos); err != nil {
		return false
	}
	return true
}

func resolveFacilTarget(p PortalQueryParams) (baseURL, dbID string, err error) {
	baseURL = resolveFacilBaseURL(p.BaseURL)
	dbID = strings.TrimSpace(p.TenantID)
	if dbID == "" {
		dbID = ExtractFacilDatabaseID(p.BaseURL)
	}
	if dbID == "" && p.CodAdministracao > 0 {
		dbID = strconv.Itoa(p.CodAdministracao)
	}
	if dbID == "" {
		return "", "", fmt.Errorf("Fácil Transparência: databaseId não informado")
	}
	return baseURL, dbID, nil
}

func resolveFacilYear(p PortalQueryParams) int {
	if p.Ano > 0 {
		return p.Ano
	}
	if y := parseYearFromDateStr(p.Inicio); y > 0 {
		return y
	}
	return time.Now().Year()
}

func fetchFacilGrid(
	ctx context.Context,
	client *http.Client,
	p PortalQueryParams,
	endpointPath string,
	extraFields map[string]any,
) (rows []map[string]any, total int, baseURL string, ano int, err error) {
	baseURL, dbID, err := resolveFacilTarget(p)
	if err != nil {
		return nil, 0, "", 0, err
	}
	ano = resolveFacilYear(p)
	limit := normalizeLimit(p.Limit)

	payload := map[string]any{
		"parameters": map[string]any{
			"start":  0,
			"length": limit,
			"draw":   1,
		},
		"ano":      strconv.Itoa(ano),
		"dtInicio": strings.TrimSpace(p.Inicio),
		"dtFim":    strings.TrimSpace(p.Fim),
	}
	for k, v := range extraFields {
		payload[k] = v
	}

	target := fmt.Sprintf("%s%s?databaseId=%s", baseURL, endpointPath, url.QueryEscape(dbID))
	body, status, reqErr := doPortalJSONPost(ctx, client, target, payload)
	if reqErr != nil {
		return nil, 0, baseURL, ano, reqErr
	}
	if status != http.StatusOK {
		return nil, 0, baseURL, ano, fmt.Errorf("Fácil Transparência %s retornou HTTP %d", endpointPath, status)
	}

	rows, total, decErr := decodeFlexiblePortalList(body, "data", "Data")
	if decErr != nil {
		return nil, 0, baseURL, ano, fmt.Errorf("erro ao decodificar Fácil %s: %w", endpointPath, decErr)
	}
	if total == 0 {
		var obj map[string]any
		if json.Unmarshal(body, &obj) == nil && obj != nil {
			total = int(mapGetFloat(obj, "recordsFiltered", "recordsTotal", "total"))
		}
	}
	return rows, total, baseURL, ano, nil
}

// QueryFacilServidores consulta servidores via POST /api/servidor/ServidorGetNomeGrid?databaseId=<id>.
func QueryFacilServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ServidoresResult, error) {
	rows, total, baseURL, ano, err := fetchFacilGrid(ctx, client, p, "/api/servidor/ServidorGetNomeGrid", nil)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ServidorItem, 0, minInt(limit, len(rows)))

	for _, r := range rows {
		nome := mapGetStr(r, "nome", "Nome", "servidor", "Servidor", "nomeServidor", "funcionario")
		if filtro != "" && !strings.Contains(strings.ToLower(nome), filtro) {
			continue
		}
		comp := mapGetStr(r, "competencia", "Competencia", "mesAno", "referencia")
		if comp == "" {
			mes := mapGetStr(r, "mes", "Mes")
			if mes != "" {
				comp = fmt.Sprintf("%s/%04d", mes, ano)
			} else {
				comp = strconv.Itoa(ano)
			}
		}

		items = append(items, ServidorItem{
			Nome:           nome,
			Matricula:      mapGetStr(r, "matricula", "Matricula", "registro", "codigo"),
			CPFParcial:     maskCPF(mapGetStr(r, "cpfFormatado", "CPFFormatado", "cpfParcial"), mapGetStr(r, "cpf", "CPF", "Cpf", "numCpf", "documento")),
			Cargo:          mapGetStr(r, "cargo", "Cargo", "descricaoCargo", "nomeCargo"),
			Funcao:         mapGetStr(r, "funcao", "Funcao", "descricaoFuncao"),
			Lotacao:        mapGetStr(r, "lotacao", "Lotacao", "departamento", "setor", "unidade"),
			LocalTrabalho:  mapGetStr(r, "localTrabalho", "LocalTrabalho", "local"),
			Vinculo:        mapGetStr(r, "vinculo", "Vinculo", "regime", "tipoVinculo"),
			Situacao:       mapGetStr(r, "situacao", "Situacao", "status"),
			SalarioBase:    mapGetFloat(r, "salarioBase", "SalarioBase", "vencimentoBase", "base"),
			SalarioBruto:   mapGetFloat(r, "salarioBruto", "SalarioBruto", "totalProventos", "proventos", "bruto", "remuneracao"),
			Descontos:      mapGetFloat(r, "descontos", "Descontos", "totalDescontos", "valorDescontos"),
			SalarioLiquido: mapGetFloat(r, "salarioLiquido", "SalarioLiquido", "valorLiquido", "liquido"),
			Competencia:    comp,
			CargaHoraria:   mapGetStr(r, "cargaHoraria", "CargaHoraria"),
			DataAdmissao:   mapGetStr(r, "dataAdmissao", "DataAdmissao", "admissao"),
			Orgao:          mapGetStr(r, "orgao", "Orgao", "entidade", "unidadeGestora"),
			Fonte:          "Portal Municipal (Fácil Transparência)",
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
		msg = "Nenhum servidor encontrado para os filtros consultados."
	}

	return &ServidoresResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Fácil Transparência)",
		Plataforma:     "facil_transparencia",
		PortalURL:      baseURL,
		TotalRegistros: total,
		Servidores:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryFacilDespesas consulta despesas/empenhos via POST /api/despesa/DespesasGetEmpenhoGrid/?databaseId=<id>
// garantindo o campo não-nulo "fornecedor": "".
func QueryFacilDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	extra := map[string]any{
		"fornecedor": strings.TrimSpace(p.NomeFiltro),
	}
	rows, total, baseURL, ano, err := fetchFacilGrid(ctx, client, p, "/api/despesa/DespesasGetEmpenhoGrid/", extra)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]DespesaItem, 0, minInt(limit, len(rows)))

	for _, r := range rows {
		fornecedor := mapGetStr(r, "fornecedor", "Fornecedor", "credor", "Credor", "nomeFornecedor", "razaoSocial")
		historico := mapGetStr(r, "historico", "Historico", "descricao", "Descricao", "especificacao")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+historico), filtro) {
			continue
		}
		itemAno := int(mapGetFloat(r, "ano", "Ano", "exercicio", "Exercicio"))
		if itemAno == 0 {
			itemAno = ano
		}
		doc := mapGetStr(r, "documentoFornecedor", "cpfCnpj", "CpfCnpj", "cnpjCpf", "CnpjCpf", "cnpj", "cpf", "documento")

		items = append(items, DespesaItem{
			NumeroEmpenho:       mapGetStr(r, "numeroEmpenho", "NumeroEmpenho", "empenho", "Empenho", "numero", "Numero"),
			Ano:                 itemAno,
			Data:                mapGetStr(r, "data", "Data", "dataEmpenho", "DataEmpenho"),
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(doc),
			Orgao:               mapGetStr(r, "orgao", "Orgao", "unidadeGestora"),
			Unidade:             mapGetStr(r, "unidade", "Unidade", "unidadeOrcamentaria"),
			Funcao:              mapGetStr(r, "funcao", "Funcao", "descricaoFuncao"),
			Elemento:            mapGetStr(r, "elemento", "Elemento", "elementoDespesa", "naturezaDespesa"),
			Historico:           historico,
			ValorEmpenhado:      mapGetFloat(r, "valorEmpenhado", "ValorEmpenhado", "empenhado", "Empenhado", "valor", "Valor"),
			ValorLiquidado:      mapGetFloat(r, "valorLiquidado", "ValorLiquidado", "liquidado", "Liquidado"),
			ValorPago:           mapGetFloat(r, "valorPago", "ValorPago", "pago", "Pago"),
			Fonte:               "Portal Municipal (Fácil Transparência)",
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
		msg = "Nenhuma despesa encontrada para os filtros consultados."
	}

	return &DespesasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Fácil Transparência)",
		Plataforma:     "facil_transparencia",
		PortalURL:      baseURL,
		TotalRegistros: total,
		Despesas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryFacilReceitas consulta receitas via POST /api/receita/ReceitasGetGrid?databaseId=<id>.
func QueryFacilReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	rows, total, baseURL, _, err := fetchFacilGrid(ctx, client, p, "/api/receita/ReceitasGetGrid", nil)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ReceitaItem, 0, minInt(limit, len(rows)))

	for _, r := range rows {
		desc := mapGetStr(r, "descricao", "Descricao", "especificacao", "receita", "Receita", "nome")
		if filtro != "" && !strings.Contains(strings.ToLower(desc), filtro) {
			continue
		}
		items = append(items, ReceitaItem{
			Codigo:          mapGetStr(r, "codigo", "Codigo", "codigoReceita", "classificacao", "estrutural", "conta"),
			Descricao:       desc,
			Categoria:       mapGetStr(r, "categoria", "Categoria", "fonte", "Fonte", "origem"),
			Data:            mapGetStr(r, "data", "Data", "competencia"),
			ValorPrevisto:   mapGetFloat(r, "valorPrevisto", "ValorPrevisto", "previsto", "Previsto", "orcado", "Orcado"),
			ValorArrecadado: mapGetFloat(r, "valorArrecadado", "ValorArrecadado", "arrecadado", "Arrecadado", "realizado", "Realizado"),
			Fonte:           "Portal Municipal (Fácil Transparência)",
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
		Fonte:          "Portal Municipal (Fácil Transparência)",
		Plataforma:     "facil_transparencia",
		PortalURL:      baseURL,
		TotalRegistros: total,
		Receitas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryFacilLicitacoes consulta licitações via POST /api/licitacao/LicitacaoGetGrid?databaseId=<id>.
func QueryFacilLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LicitacoesResult, error) {
	extra := map[string]any{
		"modalidade": "",
	}
	rows, total, baseURL, ano, err := fetchFacilGrid(ctx, client, p, "/api/licitacao/LicitacaoGetGrid", extra)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]LicitacaoItem, 0, minInt(limit, len(rows)))

	for _, r := range rows {
		objeto := mapGetStr(r, "objeto", "Objeto", "descricao", "Descricao")
		modalidade := mapGetStr(r, "modalidade", "Modalidade", "tipo")
		if filtro != "" && !strings.Contains(strings.ToLower(objeto+" "+modalidade), filtro) {
			continue
		}
		itemAno := int(mapGetFloat(r, "ano", "Ano", "exercicio", "Exercicio"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, LicitacaoItem{
			Numero:          mapGetStr(r, "numero", "Numero", "numeroLicitacao", "NumeroLicitacao", "edital", "processo"),
			Ano:             itemAno,
			Modalidade:      modalidade,
			Situacao:        mapGetStr(r, "situacao", "Situacao", "status"),
			DataAbertura:    mapGetStr(r, "dataAbertura", "DataAbertura", "dataPublicacao", "data"),
			Objeto:          objeto,
			ValorEstimado:   mapGetFloat(r, "valorEstimado", "ValorEstimado", "valorMaximo", "valorPrevisto", "valor"),
			ValorHomologado: mapGetFloat(r, "valorHomologado", "ValorHomologado", "valorVencedor", "valorTotal"),
			Link:            mapGetStr(r, "link", "url", "Link"),
			Fonte:           "Portal Municipal (Fácil Transparência)",
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
		msg = "Nenhuma licitação encontrada para os filtros consultados."
	}

	return &LicitacoesResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Fácil Transparência)",
		Plataforma:     "facil_transparencia",
		PortalURL:      baseURL,
		TotalRegistros: total,
		Licitacoes:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}
