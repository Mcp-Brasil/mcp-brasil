package transparencia

import (
	"context"
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
// IPM Sistemas / Atende.Net Connector
// Endpoints suportados:
//   - /api/WCPDadosAbertos/despesasOrcadas?dataFinal=31/12/2025
//   - /api/transparencia-pessoal-funcionarios?pagina=1
//   - /api/WCPDadosAbertos/despesas?dataInicial=DD/MM/YYYY&dataFinal=DD/MM/YYYY
//   - /api/WCPDadosAbertos/receitas?dataInicial=DD/MM/YYYY&dataFinal=DD/MM/YYYY
//   - /api/transparencia/dados-abertos/v1/wco/licitacao?pagina=0
//   - /api/transparencia/dados-abertos/v1/wco/contrato?pagina=0
// ---------------------------------------------------------------------------

const (
	ipmPlataforma = "ipm_atende"
	ipmFonte      = "Portal Municipal (IPM / Atende.Net)"
)

func ipmHTTPClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{Timeout: 12 * time.Second}
}

func ipmIsHTMLOrCaptcha(body []byte) bool {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return false
	}
	low := strings.ToLower(trimmed)
	if strings.HasPrefix(low, "<") || strings.Contains(low, "<html") || strings.Contains(low, "<!doctype") {
		return true
	}
	for _, marker := range antiBotMarkers {
		if strings.Contains(low, marker) {
			return true
		}
	}
	return false
}

func doIPMGet(ctx context.Context, client *http.Client, targetURL string) ([]byte, int, error) {
	hc := ipmHTTPClient(client)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	req.Header.Set("Accept", "application/json")

	resp, err := hc.Do(req)
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

func isIPMOKStatus(status int) bool {
	return status == http.StatusOK || status == http.StatusCreated
}

func formatIPMDate(s string) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return ""
	}
	// Já está no formato DD/MM/YYYY
	if len(trimmed) == 10 && trimmed[2] == '/' && trimmed[5] == '/' {
		return trimmed
	}
	// Converte YYYY-MM-DD (ou prefixo de timestamp ISO) para DD/MM/YYYY
	if len(trimmed) >= 10 && trimmed[4] == '-' && trimmed[7] == '-' {
		if t, err := time.Parse("2006-01-02", trimmed[:10]); err == nil {
			return t.Format("02/01/2006")
		}
	}
	return trimmed
}

func resolveIPMDateWindow(p PortalQueryParams) (string, string) {
	if p.Inicio != "" && p.Fim != "" {
		return formatIPMDate(p.Inicio), formatIPMDate(p.Fim)
	}
	ano := p.Ano
	if ano <= 0 {
		ano = time.Now().Year()
	}
	if p.Mes >= 1 && p.Mes <= 12 {
		start := time.Date(ano, time.Month(p.Mes), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, -1)
		return start.Format("02/01/2006"), end.Format("02/01/2006")
	}
	inicio := fmt.Sprintf("01/01/%04d", ano)
	fim := fmt.Sprintf("31/12/%04d", ano)
	if p.Inicio != "" {
		inicio = formatIPMDate(p.Inicio)
	}
	if p.Fim != "" {
		fim = formatIPMDate(p.Fim)
	}
	return inicio, fim
}

func decodeIPMRecordsMeta(body []byte, preferredKeys ...string) ([]map[string]any, int, bool, error) {
	if ipmIsHTMLOrCaptcha(body) {
		return nil, 0, false, fmt.Errorf("portal IPM retornou HTML ou desafio anti-bot/CAPTCHA")
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, 0, false, nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var rows []map[string]any
		if err := json.Unmarshal([]byte(trimmed), &rows); err != nil {
			return nil, 0, false, err
		}
		return rows, 0, false, nil
	}

	var obj map[string]any
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil {
		return nil, 0, false, err
	}

	total := int(mapGetFloat(obj, "totalRegistros", "TotalRegistros", "total", "Total", "totalElements", "TotalElements", "quantidade"))
	hasExplicitTotal := total > 0

	keys := append([]string{}, preferredKeys...)
	keys = append(keys, "retorno", "registros", "content", "dados", "items")
	for _, key := range keys {
		raw, ok := obj[key]
		if !ok || raw == nil {
			continue
		}
		arr, ok := raw.([]any)
		if !ok {
			continue
		}
		rows := make([]map[string]any, 0, len(arr))
		for _, el := range arr {
			if row, ok := el.(map[string]any); ok {
				rows = append(rows, row)
			}
		}
		return rows, total, hasExplicitTotal, nil
	}

	return nil, total, hasExplicitTotal, nil
}

func decodeIPMRecords(body []byte, preferredKeys ...string) ([]map[string]any, int, error) {
	rows, total, hasTotal, err := decodeIPMRecordsMeta(body, preferredKeys...)
	if err != nil {
		return nil, 0, err
	}
	if !hasTotal {
		total = len(rows)
	}
	return rows, total, nil
}

func ipmExtractYear(s string, fallback int) int {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) == 10 && trimmed[2] == '/' && trimmed[5] == '/' {
		if y, err := strconv.Atoi(trimmed[6:10]); err == nil && y > 1900 {
			return y
		}
	}
	if len(trimmed) >= 4 && (len(trimmed) == 4 || trimmed[4] == '-') {
		if y, err := strconv.Atoi(trimmed[:4]); err == nil && y > 1900 {
			return y
		}
	}
	return fallback
}

// ProbeIPM verifica se a URL base corresponde a um portal IPM Sistemas / Atende.Net com API JSON aberta.
func ProbeIPM(ctx context.Context, client *http.Client, baseURL string) bool {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return false
	}

	endpoints := []string{
		base + "/api/WCPDadosAbertos/despesasOrcadas?dataFinal=31/12/2025",
		base + "/api/transparencia-pessoal-funcionarios?pagina=1",
	}

	for _, u := range endpoints {
		body, status, err := doIPMGet(ctx, client, u)
		if err != nil || !isIPMOKStatus(status) {
			continue
		}
		if ipmIsHTMLOrCaptcha(body) {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal(body, &obj); err != nil {
			continue
		}
		if _, ok := obj["retorno"]; ok {
			return true
		}
		if _, ok := obj["registros"]; ok {
			return true
		}
	}

	return false
}

// QueryIPMServidores consulta servidores públicos no endpoint IPM /api/transparencia-pessoal-funcionarios.
func QueryIPMServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ServidoresResult, error) {
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	scanCap := limit
	if filtro != "" && scanCap < 500 {
		scanCap = 500
	}

	var items []ServidorItem
	total := 0
	hasExplicitTotal := false
	scanned := 0
	matchedTotal := 0
	prevBody := ""

	for pagina := 1; pagina <= 10; pagina++ {
		u := fmt.Sprintf("%s/api/transparencia-pessoal-funcionarios?pagina=%d", base, pagina)
		body, status, err := doIPMGet(ctx, client, u)
		if err != nil {
			if pagina == 1 {
				return nil, err
			}
			break
		}
		if !isIPMOKStatus(status) {
			if pagina == 1 {
				return nil, fmt.Errorf("IPM /api/transparencia-pessoal-funcionarios retornou HTTP %d", status)
			}
			break
		}
		bodyStr := strings.TrimSpace(string(body))
		if pagina > 1 && (bodyStr == "" || bodyStr == prevBody) {
			break
		}
		prevBody = bodyStr

		rows, pageTotal, pageHasTotal, err := decodeIPMRecordsMeta(body, "registros", "retorno")
		if err != nil {
			if pagina == 1 {
				return nil, fmt.Errorf("erro ao decodificar IPM servidores: %w", err)
			}
			break
		}
		if pageHasTotal && pageTotal > total {
			total = pageTotal
			hasExplicitTotal = true
		}
		if len(rows) == 0 {
			break
		}
		scanned += len(rows)

		for _, r := range rows {
			nome := mapGetStr(r, "nome", "servidor", "funcionario", "nomServidor")
			if filtro != "" && !strings.Contains(strings.ToLower(nome), filtro) {
				continue
			}
			matchedTotal++
			if len(items) >= limit {
				continue
			}

			local := mapGetStr(r, "localTrabalho", "centroCusto", "local")
			lotacao := mapGetStr(r, "centroCusto", "lotacao", "departamento", "setor")
			if lotacao == "" {
				lotacao = local
			}

			items = append(items, ServidorItem{
				Nome:           nome,
				Matricula:      mapGetStr(r, "matricula", "codigo", "numMatricula"),
				CPFParcial:     maskCPF(mapGetStr(r, "cpfFormatado", "cpf_parcial"), mapGetStr(r, "cpf", "numCpf")),
				Cargo:          mapGetStr(r, "cargo", "descricaoCargo", "cargoDescricao"),
				Funcao:         mapGetStr(r, "funcao", "funcaoDescricao", "descricaoFuncao"),
				Lotacao:        lotacao,
				LocalTrabalho:  local,
				Vinculo:        mapGetStr(r, "regime", "vinculo", "tipoVinculo", "categoria"),
				Situacao:       mapGetStr(r, "situacao", "status"),
				SalarioBase:    mapGetFloat(r, "salarioBase", "vencimentoBase", "remuneracaoBase"),
				SalarioBruto:   mapGetFloat(r, "salarioBruto", "totalProventos", "remuneracaoBruta"),
				Descontos:      mapGetFloat(r, "descontos", "totalDescontos"),
				SalarioLiquido: mapGetFloat(r, "salarioLiquido", "liquido"),
				Competencia:    mapGetStr(r, "competencia", "mesAno"),
				CargaHoraria:   mapGetStr(r, "horasSemanais", "cargaHoraria"),
				DataAdmissao:   mapGetStr(r, "admissao", "dataAdmissao", "dtAdmissao"),
				Orgao:          mapGetStr(r, "entidade", "orgao", "entidadeDescricao"),
				Fonte:          ipmFonte,
				PortalURL:      base,
			})
		}

		if (filtro == "" && len(items) >= limit) || scanned >= scanCap || (hasExplicitTotal && total > 0 && scanned >= total) {
			break
		}
	}

	if filtro != "" {
		total = matchedTotal
	} else if total == 0 {
		total = scanned
	}

	msg := ""
	if len(items) == 0 {
		msg = "Nenhum servidor encontrado para os filtros consultados."
	}

	return &ServidoresResult{
		Municipio:      m,
		Fonte:          ipmFonte,
		Plataforma:     ipmPlataforma,
		PortalURL:      base,
		TotalRegistros: total,
		Servidores:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryIPMDespesas consulta despesas públicas no endpoint IPM /api/WCPDadosAbertos/despesas.
func QueryIPMDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	limit := normalizeLimit(p.Limit)
	dataInicial, dataFinal := resolveIPMDateWindow(p)

	q := url.Values{}
	q.Set("dataInicial", dataInicial)
	q.Set("dataFinal", dataFinal)
	u := fmt.Sprintf("%s/api/WCPDadosAbertos/despesas?%s", base, q.Encode())

	body, status, err := doIPMGet(ctx, client, u)
	if err != nil {
		return nil, err
	}
	if !isIPMOKStatus(status) {
		return nil, fmt.Errorf("IPM /api/WCPDadosAbertos/despesas retornou HTTP %d", status)
	}

	rows, total, err := decodeIPMRecords(body, "retorno", "registros")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar IPM despesas: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]DespesaItem, 0, minInt(limit, len(rows)))
	matchedTotal := 0
	for _, r := range rows {
		fornecedor := mapGetStr(r, "nomeCredor", "credor", "fornecedor", "nomeFornecedor")
		historico := mapGetStr(r, "historico", "especificacao", "descricao")
		if filtro != "" &&
			!strings.Contains(strings.ToLower(fornecedor), filtro) &&
			!strings.Contains(strings.ToLower(historico), filtro) {
			continue
		}
		matchedTotal++
		if len(items) >= limit {
			continue
		}

		doc := mapGetStr(r, "cpfCnpjCredor", "cpfCnpj", "cnpjCpf", "documento")
		if len(cleanDigits(doc)) == 11 {
			doc = maskCPF("", doc)
		}

		dataStr := mapGetStr(r, "data", "dataEmpenho", "dtEmpenho")
		anoItem := int(mapGetFloat(r, "ano", "exercicio"))
		if anoItem == 0 {
			anoItem = ipmExtractYear(dataStr, p.Ano)
		}

		items = append(items, DespesaItem{
			NumeroEmpenho:       mapGetStr(r, "numeroEmpenho", "empenho", "nroEmpenho"),
			Ano:                 anoItem,
			Data:                dataStr,
			Fornecedor:          fornecedor,
			DocumentoFornecedor: doc,
			Orgao:               mapGetStr(r, "orgaoDescricao", "orgao", "entidadeDescricao"),
			Unidade:             mapGetStr(r, "unidadeDescricao", "unidade"),
			Funcao:              mapGetStr(r, "funcaoDescricao", "funcao"),
			Elemento:            mapGetStr(r, "elementoDescricao", "elemento", "fonteRecursoDescricao", "fonteRecurso"),
			Historico:           historico,
			ValorEmpenhado:      mapGetFloat(r, "valorEmpenhado", "empenhado"),
			ValorLiquidado:      mapGetFloat(r, "valorLiquidado", "liquidado"),
			ValorPago:           mapGetFloat(r, "valorPago", "pago"),
			Fonte:               ipmFonte,
		})
	}

	if filtro != "" {
		total = matchedTotal
	} else if total == 0 {
		total = len(rows)
	}

	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma despesa encontrada para o período informado."
	}

	return &DespesasResult{
		Municipio:      m,
		Fonte:          ipmFonte,
		Plataforma:     ipmPlataforma,
		PortalURL:      base,
		TotalRegistros: total,
		Despesas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryIPMReceitas consulta receitas públicas no endpoint IPM /api/WCPDadosAbertos/receitas.
func QueryIPMReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	limit := normalizeLimit(p.Limit)
	dataInicial, dataFinal := resolveIPMDateWindow(p)

	q := url.Values{}
	q.Set("dataInicial", dataInicial)
	q.Set("dataFinal", dataFinal)
	u := fmt.Sprintf("%s/api/WCPDadosAbertos/receitas?%s", base, q.Encode())

	body, status, err := doIPMGet(ctx, client, u)
	if err != nil {
		return nil, err
	}
	if !isIPMOKStatus(status) {
		return nil, fmt.Errorf("IPM /api/WCPDadosAbertos/receitas retornou HTTP %d", status)
	}

	rows, total, err := decodeIPMRecords(body, "retorno", "registros")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar IPM receitas: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
	matchedTotal := 0
	for _, r := range rows {
		codigo := mapGetStr(r, "codigoReceita", "codigo", "receitaCodigo", "estrutural", "rubrica")
		descricao := mapGetStr(r, "receitaDescricao", "descricao", "especificacao", "nomeReceita")
		if filtro != "" &&
			!strings.Contains(strings.ToLower(descricao), filtro) &&
			!strings.Contains(strings.ToLower(codigo), filtro) {
			continue
		}
		matchedTotal++
		if len(items) >= limit {
			continue
		}

		items = append(items, ReceitaItem{
			Codigo:          codigo,
			Descricao:       descricao,
			Categoria:       mapGetStr(r, "fonteRecursoDescricao", "categoria", "origemDescricao", "tipoReceita"),
			Data:            mapGetStr(r, "data", "dataArrecadacao", "competencia"),
			ValorPrevisto:   mapGetFloat(r, "valorPrevisto", "valorOrcado", "previsto", "orcado"),
			ValorArrecadado: mapGetFloat(r, "valorArrecadado", "valorRealizado", "arrecadado", "realizado", "valorReceita"),
			Fonte:           ipmFonte,
		})
	}

	if filtro != "" {
		total = matchedTotal
	} else if total == 0 {
		total = len(rows)
	}

	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma receita encontrada para o período informado."
	}

	return &ReceitasResult{
		Municipio:      m,
		Fonte:          ipmFonte,
		Plataforma:     ipmPlataforma,
		PortalURL:      base,
		TotalRegistros: total,
		Receitas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryIPMLicitacoes consulta licitações no endpoint IPM /api/transparencia/dados-abertos/v1/wco/licitacao.
func QueryIPMLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LicitacoesResult, error) {
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	limit := normalizeLimit(p.Limit)
	u := fmt.Sprintf("%s/api/transparencia/dados-abertos/v1/wco/licitacao?pagina=0", base)

	body, status, err := doIPMGet(ctx, client, u)
	if err != nil {
		return &LicitacoesResult{
			Municipio:  m,
			Fonte:      ipmFonte,
			Plataforma: ipmPlataforma,
			PortalURL:  base,
			Licitacoes: []LicitacaoItem{},
			Disponivel: false,
			Mensagem:   fmt.Sprintf("Falha ao consultar licitações IPM: %v", err),
		}, err
	}
	if !isIPMOKStatus(status) {
		msg := fmt.Sprintf("Endpoint WCO de licitações indisponível neste portal IPM (HTTP %d).", status)
		return &LicitacoesResult{
			Municipio:  m,
			Fonte:      ipmFonte,
			Plataforma: ipmPlataforma,
			PortalURL:  base,
			Licitacoes: []LicitacaoItem{},
			Disponivel: false,
			Mensagem:   msg,
		}, fmt.Errorf("IPM wco/licitacao retornou HTTP %d", status)
	}

	rows, total, err := decodeIPMRecords(body, "registros", "retorno", "content", "licitacoes")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar IPM licitações: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]LicitacaoItem, 0, minInt(limit, len(rows)))
	matchedTotal := 0
	for _, r := range rows {
		numero := mapGetStr(r, "numeroLicitacao", "numero", "nroLicitacao", "numeroProcesso", "processo")
		modalidade := mapGetStr(r, "modalidadeDescricao", "modalidade", "tipoModalidade")
		objeto := mapGetStr(r, "objeto", "descricao", "resumo")
		if filtro != "" &&
			!strings.Contains(strings.ToLower(objeto), filtro) &&
			!strings.Contains(strings.ToLower(modalidade), filtro) &&
			!strings.Contains(strings.ToLower(numero), filtro) {
			continue
		}
		matchedTotal++
		if len(items) >= limit {
			continue
		}

		dataAbertura := mapGetStr(r, "dataAbertura", "dtAbertura", "dataPublicacao", "data")
		anoItem := int(mapGetFloat(r, "ano", "exercicio", "anoLicitacao", "anoProcesso"))
		if anoItem == 0 {
			anoItem = ipmExtractYear(dataAbertura, p.Ano)
		}
		link := mapGetStr(r, "link", "url")
		if link == "" {
			link = base
		}

		items = append(items, LicitacaoItem{
			Numero:          numero,
			Ano:             anoItem,
			Modalidade:      modalidade,
			Situacao:        mapGetStr(r, "situacaoDescricao", "situacao", "status"),
			DataAbertura:    dataAbertura,
			Objeto:          objeto,
			ValorEstimado:   mapGetFloat(r, "valorEstimado", "valorMaximo", "valorPrevisto", "valor"),
			ValorHomologado: mapGetFloat(r, "valorHomologado", "valorAdjudicado", "valorTotal", "valorFinal"),
			Link:            link,
			Fonte:           ipmFonte,
		})
	}

	if filtro != "" {
		total = matchedTotal
	} else if total == 0 {
		total = len(rows)
	}

	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma licitação encontrada."
	}

	return &LicitacoesResult{
		Municipio:      m,
		Fonte:          ipmFonte,
		Plataforma:     ipmPlataforma,
		PortalURL:      base,
		TotalRegistros: total,
		Licitacoes:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryIPMContratos consulta contratos no endpoint IPM /api/transparencia/dados-abertos/v1/wco/contrato.
func QueryIPMContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ContratosResult, error) {
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	limit := normalizeLimit(p.Limit)
	u := fmt.Sprintf("%s/api/transparencia/dados-abertos/v1/wco/contrato?pagina=0", base)

	body, status, err := doIPMGet(ctx, client, u)
	if err != nil {
		return &ContratosResult{
			Municipio:  m,
			Fonte:      ipmFonte,
			Plataforma: ipmPlataforma,
			PortalURL:  base,
			Contratos:  []ContratoItem{},
			Disponivel: false,
			Mensagem:   fmt.Sprintf("Falha ao consultar contratos IPM: %v", err),
		}, err
	}
	if !isIPMOKStatus(status) {
		msg := fmt.Sprintf("Endpoint WCO de contratos indisponível neste portal IPM (HTTP %d).", status)
		return &ContratosResult{
			Municipio:  m,
			Fonte:      ipmFonte,
			Plataforma: ipmPlataforma,
			PortalURL:  base,
			Contratos:  []ContratoItem{},
			Disponivel: false,
			Mensagem:   msg,
		}, fmt.Errorf("IPM wco/contrato retornou HTTP %d", status)
	}

	rows, total, err := decodeIPMRecords(body, "registros", "retorno", "content", "contratos")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar IPM contratos: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ContratoItem, 0, minInt(limit, len(rows)))
	matchedTotal := 0
	for _, r := range rows {
		numero := mapGetStr(r, "numeroContrato", "numero", "nroContrato")
		fornecedor := mapGetStr(r, "nomeContratado", "nomeFornecedor", "fornecedor", "credor", "razaoSocial", "contratado")
		objeto := mapGetStr(r, "objeto", "descricao")
		if filtro != "" &&
			!strings.Contains(strings.ToLower(fornecedor), filtro) &&
			!strings.Contains(strings.ToLower(objeto), filtro) &&
			!strings.Contains(strings.ToLower(numero), filtro) {
			continue
		}
		matchedTotal++
		if len(items) >= limit {
			continue
		}

		doc := mapGetStr(r, "cpfCnpjContratado", "cpfCnpjFornecedor", "cpfCnpj", "cnpjCpf", "documento")
		if len(cleanDigits(doc)) == 11 {
			doc = maskCPF("", doc)
		}

		dataAssinatura := mapGetStr(r, "dataAssinatura", "dtAssinatura", "data")
		anoItem := int(mapGetFloat(r, "ano", "exercicio", "anoContrato"))
		if anoItem == 0 {
			anoItem = ipmExtractYear(dataAssinatura, p.Ano)
		}

		items = append(items, ContratoItem{
			Numero:              numero,
			Ano:                 anoItem,
			Fornecedor:          fornecedor,
			DocumentoFornecedor: doc,
			Objeto:              objeto,
			Situacao:            mapGetStr(r, "situacaoDescricao", "situacao", "status"),
			DataAssinatura:      dataAssinatura,
			VigenciaInicio:      mapGetStr(r, "dataInicioVigencia", "vigenciaInicio", "dataInicio"),
			VigenciaFim:         mapGetStr(r, "dataFimVigencia", "vigenciaFim", "dataVencimento", "dataTermino"),
			Valor:               mapGetFloat(r, "valor", "valorContrato", "valorTotal", "valorInicial"),
			Fonte:               ipmFonte,
		})
	}

	if filtro != "" {
		total = matchedTotal
	} else if total == 0 {
		total = len(rows)
	}

	msg := ""
	if len(items) == 0 {
		msg = "Nenhum contrato encontrado."
	}

	return &ContratosResult{
		Municipio:      m,
		Fonte:          ipmFonte,
		Plataforma:     ipmPlataforma,
		PortalURL:      base,
		TotalRegistros: total,
		Contratos:      items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}
