package transparencia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

// ---------------------------------------------------------------------------
// SmarAPD PAI (plataforma = "smarapd_pai")
// Endpoints principais:
//   GET  /paiportalserver/cliente
//   GET  /paiportalserver/DadosAbertos
//   GET  /paiportalserver/MenuPortal
//   POST /paiportalserver/modulovisao/filter
// ---------------------------------------------------------------------------

type smarapdVisaoRef struct {
	ChaveModulo string
	NomeVisao   string
	Label       string
}

func normalizeSmarapdRootURL(baseURL string) (rootURL, origin, referer string) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return "", "", ""
	}
	if idx := strings.Index(trimmed, "#"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	if idx := strings.Index(trimmed, "?"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	trimmed = strings.TrimRight(strings.TrimSpace(trimmed), "/")
	if trimmed == "" {
		return "", "", ""
	}

	u, err := url.Parse(trimmed)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return trimmed, "", ""
	}

	origin = u.Scheme + "://" + u.Host
	referer = origin + "/"

	p := strings.TrimRight(u.Path, "/")
	lowP := strings.ToLower(p)
	if idx := strings.Index(lowP, "/paiportalserver"); idx >= 0 {
		p = p[:idx]
	} else {
		for _, ext := range []string{".html", ".htm", ".aspx", ".asp"} {
			if strings.HasSuffix(lowP, ext) {
				if slash := strings.LastIndex(p, "/"); slash >= 0 {
					p = p[:slash]
				}
				break
			}
		}
	}
	rootURL = strings.TrimRight(origin+p, "/")
	return rootURL, origin, referer
}

func doSmarapdRequest(ctx context.Context, client *http.Client, method, origin, referer, targetURL string, payload any) ([]byte, int, error) {
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
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if referer != "" {
		req.Header.Set("Referer", referer)
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

func isValidSmarapdJSONProbe(body []byte, requireModulosOrMeta bool) bool {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || (!strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[")) {
		return false
	}
	var parsed any
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil || parsed == nil {
		return false
	}
	if !requireModulosOrMeta {
		return true
	}
	low := strings.ToLower(trimmed)
	for _, kw := range []string{
		`"modulos"`, `"chavemodulo"`, `"nomevisao"`,
		`"nome"`, `"razaosocial"`, `"cnpj"`, `"cliente"`,
		`"municipio"`, `"entidade"`, `"codigo"`, `"titulo"`,
	} {
		if strings.Contains(low, kw) {
			return true
		}
	}
	return false
}

// ProbeSmarapd verifica se a URL base responde como um portal SmarAPD PAI
// testando GET /paiportalserver/cliente e GET /paiportalserver/DadosAbertos com Origin e Referer.
func ProbeSmarapd(ctx context.Context, client *http.Client, baseURL string) bool {
	rootURL, origin, referer := normalizeSmarapdRootURL(baseURL)
	if rootURL == "" {
		return false
	}

	if body, status, err := doSmarapdRequest(ctx, client, http.MethodGet, origin, referer, rootURL+"/paiportalserver/cliente", nil); err == nil && status == http.StatusOK {
		if isValidSmarapdJSONProbe(body, true) {
			return true
		}
	}
	if body, status, err := doSmarapdRequest(ctx, client, http.MethodGet, origin, referer, rootURL+"/paiportalserver/DadosAbertos", nil); err == nil && status == http.StatusOK {
		if isValidSmarapdJSONProbe(body, true) {
			return true
		}
	}
	return false
}

func normalizeSmarapdKey(s string) string {
	clean := RemoveAccents(s)
	var b strings.Builder
	b.Grow(len(clean))
	for _, r := range clean {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func smarapdGetStr(m map[string]any, keys ...string) string {
	if v := mapGetStr(m, keys...); v != "" {
		return v
	}
	if len(m) == 0 {
		return ""
	}
	normIndex := make(map[string]any, len(m))
	for k, v := range m {
		if v == nil {
			continue
		}
		nk := normalizeSmarapdKey(k)
		if _, exists := normIndex[nk]; !exists {
			normIndex[nk] = v
		}
	}
	for _, k := range keys {
		if v, ok := normIndex[normalizeSmarapdKey(k)]; ok && v != nil {
			if s := strings.TrimSpace(anyToString(v)); s != "" {
				return s
			}
		}
	}
	return ""
}

func smarapdGetFloat(m map[string]any, keys ...string) float64 {
	if v := mapGetFloat(m, keys...); v != 0 {
		return v
	}
	if len(m) == 0 {
		return 0
	}
	normIndex := make(map[string]any, len(m))
	for k, v := range m {
		if v == nil {
			continue
		}
		nk := normalizeSmarapdKey(k)
		if _, exists := normIndex[nk]; !exists {
			normIndex[nk] = v
		}
	}
	for _, k := range keys {
		if v, ok := normIndex[normalizeSmarapdKey(k)]; ok && v != nil {
			switch t := v.(type) {
			case float64:
				return t
			case string:
				if f, err := parseBRFloat(t); err == nil {
					return f
				}
			}
		}
	}
	return 0
}

func collectSmarapdVisoes(node any, parentModulo, parentLabel string, inVisoes bool, out *[]smarapdVisaoRef, seen map[string]struct{}) {
	switch v := node.(type) {
	case []any:
		for _, el := range v {
			collectSmarapdVisoes(el, parentModulo, parentLabel, inVisoes, out, seen)
		}
	case map[string]any:
		mod := smarapdGetStr(v, "ChaveModulo", "chaveModulo", "chave_modulo", "ModuloChave", "CodigoModulo")
		if mod == "" && !inVisoes {
			mod = smarapdGetStr(v, "Chave", "Modulo")
		}
		if mod == "" {
			mod = parentModulo
		}

		visao := smarapdGetStr(v, "NomeVisao", "nomeVisao", "nome_visao", "ChaveVisao", "Visao")
		if visao == "" && inVisoes {
			visao = smarapdGetStr(v, "Nome", "Chave")
		}

		label := smarapdGetStr(v, "Titulo", "Descricao", "Nome", "Label", "Texto")
		combinedLabel := strings.TrimSpace(strings.TrimSpace(parentLabel) + " " + label)

		if mod != "" && visao != "" {
			key := mod + "|" + visao
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				*out = append(*out, smarapdVisaoRef{
					ChaveModulo: mod,
					NomeVisao:   visao,
					Label:       combinedLabel,
				})
			}
		}

		for k, child := range v {
			if child == nil {
				continue
			}
			nk := normalizeSmarapdKey(k)
			childInVisoes := inVisoes || nk == "visoes" || nk == "views" || nk == "modulovisoes"
			switch child.(type) {
			case []any, map[string]any:
				nextLabel := parentLabel
				if label != "" {
					nextLabel = label
				}
				collectSmarapdVisoes(child, mod, nextLabel, childInVisoes, out, seen)
			}
		}
	}
}

func discoverSmarapdVisoes(ctx context.Context, client *http.Client, rootURL, origin, referer string) []smarapdVisaoRef {
	var visoes []smarapdVisaoRef
	seen := make(map[string]struct{})

	for _, ep := range []string{"/paiportalserver/DadosAbertos", "/paiportalserver/MenuPortal"} {
		body, status, err := doSmarapdRequest(ctx, client, http.MethodGet, origin, referer, rootURL+ep, nil)
		if err != nil || status != http.StatusOK {
			continue
		}
		trimmed := strings.TrimSpace(string(body))
		if trimmed == "" || (!strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[")) {
			continue
		}
		var doc any
		if err := json.Unmarshal([]byte(trimmed), &doc); err != nil || doc == nil {
			continue
		}
		collectSmarapdVisoes(doc, "", "", false, &visoes, seen)
	}
	return visoes
}

func selectSmarapdCandidates(discovered []smarapdVisaoRef, keywords []string, fallbacks []smarapdVisaoRef) []smarapdVisaoRef {
	type scored struct {
		ref   smarapdVisaoRef
		score int
	}
	var matches []scored
	for _, ref := range discovered {
		haystack := RemoveAccents(ref.ChaveModulo + " " + ref.NomeVisao + " " + ref.Label)
		score := 0
		for i, kw := range keywords {
			if strings.Contains(haystack, kw) {
				// Palavras-chave iniciais têm peso maior
				score += (len(keywords) - i) * 10
			}
		}
		if score > 0 {
			matches = append(matches, scored{ref: ref, score: score})
		}
	}

	// Ordenação simples por score decrescente
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].score > matches[i].score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	out := make([]smarapdVisaoRef, 0, len(matches)+len(fallbacks))
	seen := make(map[string]struct{})
	for _, m := range matches {
		key := m.ref.ChaveModulo + "|" + m.ref.NomeVisao
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			out = append(out, m.ref)
		}
	}
	for _, fb := range fallbacks {
		key := fb.ChaveModulo + "|" + fb.NomeVisao
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			out = append(out, fb)
		}
	}
	return out
}

func decodeSmarapdFilterResponseMeta(body []byte) ([]map[string]any, int, bool, error) {
	rows, total, hasTotal, err := decodeFlexiblePortalListMeta(body, "Dados", "Lista", "Itens", "Items", "Resultado", "Rows")
	if err != nil {
		return nil, 0, false, err
	}
	if len(rows) > 0 {
		return rows, total, hasTotal, nil
	}

	trimmed := strings.TrimSpace(string(body))
	if !strings.HasPrefix(trimmed, "{") {
		return rows, total, hasTotal, nil
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil {
		return nil, 0, false, err
	}
	for _, wrapperKey := range []string{"Dados", "dados", "Data", "data", "Resultado", "resultado"} {
		if sub, ok := obj[wrapperKey].(map[string]any); ok && sub != nil {
			subBytes, mErr := json.Marshal(sub)
			if mErr == nil {
				if subRows, subTot, subHasTotal, dErr := decodeFlexiblePortalListMeta(subBytes, "Dados", "Lista", "Itens", "Items"); dErr == nil && len(subRows) > 0 {
					if !subHasTotal {
						subTot = int(smarapdGetFloat(obj, "TotalRegistros", "totalRegistros", "Total", "total", "TotalItems", "totalItems", "Count", "count"))
						subHasTotal = subTot > 0
					}
					return subRows, subTot, subHasTotal, nil
				}
			}
		}
	}
	return rows, total, hasTotal, nil
}

func decodeSmarapdFilterResponse(body []byte) ([]map[string]any, int, error) {
	rows, total, hasTotal, err := decodeSmarapdFilterResponseMeta(body)
	if err != nil {
		return nil, 0, err
	}
	if !hasTotal {
		total = len(rows)
	}
	return rows, total, nil
}

func fetchSmarapdModule(
	ctx context.Context,
	client *http.Client,
	p PortalQueryParams,
	keywords []string,
	fallbacks []smarapdVisaoRef,
) (rows []map[string]any, total int, rootURL string, err error) {
	rootURL, origin, referer := normalizeSmarapdRootURL(p.BaseURL)
	if rootURL == "" {
		return nil, 0, "", fmt.Errorf("SmarAPD PAI: BaseURL não informada")
	}

	limit := normalizeLimit(p.Limit)
	scanLimit := limit
	if strings.TrimSpace(p.NomeFiltro) != "" && scanLimit < 500 {
		scanLimit = 500
	}
	pageSize := normalizePageSize(scanLimit, 200)
	discovered := discoverSmarapdVisoes(ctx, client, rootURL, origin, referer)
	candidates := selectSmarapdCandidates(discovered, keywords, fallbacks)

	var lastErr error
	var lastStatus int
	hadSuccess := false

	maxAttempts := minInt(len(candidates), 4)
	for i := 0; i < maxAttempts; i++ {
		cand := candidates[i]
		var allRows []map[string]any
		candTotal := 0
		candHasTotal := false
		prevBody := ""
		candOK := false

		for page := 1; page <= 10; page++ {
			payload := map[string]any{
				"ChaveModulo":         cand.ChaveModulo,
				"NomeVisao":           cand.NomeVisao,
				"Pagina":              page,
				"QuantidadeRegistros": pageSize,
				"Filtros":             []any{},
				"Ordenacao":           []any{},
			}
			body, status, reqErr := doSmarapdRequest(ctx, client, http.MethodPost, origin, referer, rootURL+"/paiportalserver/modulovisao/filter", payload)
			if reqErr != nil {
				if page == 1 {
					lastErr = reqErr
				}
				break
			}
			lastStatus = status
			if status != http.StatusOK {
				if page == 1 {
					lastErr = fmt.Errorf("SmarAPD /paiportalserver/modulovisao/filter (%s/%s) retornou HTTP %d", cand.ChaveModulo, cand.NomeVisao, status)
				}
				break
			}
			bodyStr := strings.TrimSpace(string(body))
			if page > 1 && (bodyStr == "" || bodyStr == prevBody) {
				break
			}
			prevBody = bodyStr

			parsed, tot, totHas, decErr := decodeSmarapdFilterResponseMeta(body)
			if decErr != nil {
				if page == 1 {
					lastErr = fmt.Errorf("erro ao decodificar SmarAPD (%s/%s): %w", cand.ChaveModulo, cand.NomeVisao, decErr)
				}
				break
			}
			candOK = true
			hadSuccess = true
			if totHas && tot > candTotal {
				candTotal = tot
				candHasTotal = true
			}
			if len(parsed) == 0 {
				break
			}
			allRows = append(allRows, parsed...)
			if len(allRows) >= scanLimit || len(parsed) < pageSize || (candHasTotal && candTotal > 0 && len(allRows) >= candTotal) {
				break
			}
		}

		if !candHasTotal && len(allRows) > 0 {
			candTotal = len(allRows)
		}
		if candOK && len(allRows) > 0 {
			return allRows, candTotal, rootURL, nil
		}
		if total == 0 {
			total = candTotal
		}
	}

	if hadSuccess {
		return nil, total, rootURL, nil
	}
	if lastErr != nil {
		return nil, 0, rootURL, lastErr
	}
	if lastStatus >= 400 {
		return nil, 0, rootURL, fmt.Errorf("SmarAPD retornou HTTP %d", lastStatus)
	}
	return nil, 0, rootURL, nil
}

// QuerySmarapdServidores consulta servidores/folha de pagamento no portal SmarAPD PAI.
func QuerySmarapdServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ServidoresResult, error) {
	rows, total, rootURL, err := fetchSmarapdModule(
		ctx, client, p,
		[]string{"servidor", "folha", "remuneracao", "pessoal", "funcionario"},
		[]smarapdVisaoRef{
			{ChaveModulo: "Servidores", NomeVisao: "Servidores"},
			{ChaveModulo: "Pessoal", NomeVisao: "Servidores"},
			{ChaveModulo: "FolhaPagamento", NomeVisao: "FolhaPagamento"},
		},
	)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ServidorItem, 0, minInt(limit, len(rows)))
	matchedTotal := 0

	for _, r := range rows {
		nome := smarapdGetStr(r, "Nome", "NomeServidor", "Servidor", "Funcionario", "NomeFuncionario")
		if filtro != "" && !strings.Contains(strings.ToLower(nome), filtro) {
			continue
		}
		matchedTotal++
		if len(items) >= limit {
			continue
		}
		cpfFmt := smarapdGetStr(r, "CPFFormatado", "CPFParcial", "CpfMascarado")
		cpfRaw := smarapdGetStr(r, "CPF", "NumCPF", "NumeroCPF", "Documento")
		comp := smarapdGetStr(r, "Competencia", "MesAno", "Referencia")
		if comp == "" {
			mes := smarapdGetStr(r, "Mes", "MesCompetencia")
			ano := smarapdGetStr(r, "Ano", "Exercicio", "AnoCompetencia")
			if mes != "" && ano != "" {
				comp = mes + "/" + ano
			} else if ano != "" {
				comp = ano
			}
		}

		items = append(items, ServidorItem{
			Nome:           nome,
			Matricula:      smarapdGetStr(r, "Matricula", "Registro", "Codigo", "NumeroMatricula"),
			CPFParcial:     maskCPF(cpfFmt, cpfRaw),
			Cargo:          smarapdGetStr(r, "Cargo", "DescricaoCargo", "NomeCargo"),
			Funcao:         smarapdGetStr(r, "Funcao", "DescricaoFuncao", "NomeFuncao"),
			Lotacao:        smarapdGetStr(r, "Lotacao", "DescricaoLotacao", "Departamento", "Setor", "Unidade"),
			LocalTrabalho:  smarapdGetStr(r, "LocalTrabalho", "Local"),
			Vinculo:        smarapdGetStr(r, "Vinculo", "Regime", "TipoVinculo", "Natureza"),
			Situacao:       smarapdGetStr(r, "Situacao", "Status"),
			SalarioBase:    smarapdGetFloat(r, "SalarioBase", "VencimentoBase", "Base"),
			SalarioBruto:   smarapdGetFloat(r, "SalarioBruto", "TotalProventos", "Proventos", "ValorBruto", "Bruto", "Remuneracao"),
			Descontos:      smarapdGetFloat(r, "Descontos", "TotalDescontos", "ValorDescontos"),
			SalarioLiquido: smarapdGetFloat(r, "SalarioLiquido", "ValorLiquido", "Liquido"),
			Competencia:    comp,
			CargaHoraria:   smarapdGetStr(r, "CargaHoraria", "HorasSemanais"),
			DataAdmissao:   smarapdGetStr(r, "DataAdmissao", "Admissao"),
			Orgao:          smarapdGetStr(r, "Orgao", "NomeOrgao", "Entidade", "UnidadeGestora"),
			Fonte:          "Portal Municipal (SmarAPD PAI)",
			PortalURL:      rootURL,
		})
	}

	if filtro != "" || total == 0 {
		total = matchedTotal
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhum servidor encontrado para os filtros consultados."
	}

	return &ServidoresResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (SmarAPD PAI)",
		Plataforma:     "smarapd_pai",
		PortalURL:      rootURL,
		TotalRegistros: total,
		Servidores:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QuerySmarapdDespesas consulta empenhos/despesas no portal SmarAPD PAI.
func QuerySmarapdDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	rows, total, rootURL, err := fetchSmarapdModule(
		ctx, client, p,
		[]string{"despesa", "empenho", "pagamento", "gasto"},
		[]smarapdVisaoRef{
			{ChaveModulo: "Despesas", NomeVisao: "Empenhos"},
			{ChaveModulo: "Despesa", NomeVisao: "Despesas"},
			{ChaveModulo: "Despesas", NomeVisao: "Despesas"},
		},
	)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]DespesaItem, 0, minInt(limit, len(rows)))
	matchedTotal := 0

	for _, r := range rows {
		fornecedor := smarapdGetStr(r, "Fornecedor", "Credor", "NomeFornecedor", "NomeCredor", "RazaoSocial", "Favorecido")
		historico := smarapdGetStr(r, "Historico", "Descricao", "HistoricoEmpenho", "Especificacao")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+historico), filtro) {
			continue
		}
		matchedTotal++
		if len(items) >= limit {
			continue
		}
		itemAno := int(smarapdGetFloat(r, "Ano", "Exercicio", "AnoEmpenho"))
		if itemAno == 0 {
			itemAno = p.Ano
		}
		doc := smarapdGetStr(r, "DocumentoFornecedor", "CPFCNPJ", "CNPJCPF", "CNPJ", "CPF", "Documento")

		items = append(items, DespesaItem{
			NumeroEmpenho:       smarapdGetStr(r, "NumeroEmpenho", "Empenho", "NumEmpenho", "Numero"),
			Ano:                 itemAno,
			Data:                smarapdGetStr(r, "Data", "DataEmpenho", "DataMovimento"),
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(doc),
			Orgao:               smarapdGetStr(r, "Orgao", "NomeOrgao", "UnidadeGestora", "Entidade"),
			Unidade:             smarapdGetStr(r, "Unidade", "UnidadeOrcamentaria", "NomeUnidade"),
			Funcao:              smarapdGetStr(r, "Funcao", "DescricaoFuncao"),
			Elemento:            smarapdGetStr(r, "Elemento", "ElementoDespesa", "NaturezaDespesa", "Classificacao"),
			Historico:           historico,
			ValorEmpenhado:      smarapdGetFloat(r, "ValorEmpenhado", "Empenhado", "VlrEmpenhado", "Valor"),
			ValorLiquidado:      smarapdGetFloat(r, "ValorLiquidado", "Liquidado", "VlrLiquidado"),
			ValorPago:           smarapdGetFloat(r, "ValorPago", "Pago", "VlrPago"),
			Fonte:               "Portal Municipal (SmarAPD PAI)",
		})
	}

	if filtro != "" || total == 0 {
		total = matchedTotal
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma despesa encontrada para os filtros consultados."
	}

	return &DespesasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (SmarAPD PAI)",
		Plataforma:     "smarapd_pai",
		PortalURL:      rootURL,
		TotalRegistros: total,
		Despesas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QuerySmarapdReceitas consulta receitas no portal SmarAPD PAI.
func QuerySmarapdReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	rows, total, rootURL, err := fetchSmarapdModule(
		ctx, client, p,
		[]string{"receita", "arrecadacao", "tributo"},
		[]smarapdVisaoRef{
			{ChaveModulo: "Receitas", NomeVisao: "Receitas"},
			{ChaveModulo: "Receita", NomeVisao: "Receitas"},
			{ChaveModulo: "Receitas", NomeVisao: "Receita"},
		},
	)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
	matchedTotal := 0

	for _, r := range rows {
		desc := smarapdGetStr(r, "Descricao", "Especificacao", "Receita", "Nome", "NomeReceita")
		if filtro != "" && !strings.Contains(strings.ToLower(desc), filtro) {
			continue
		}
		matchedTotal++
		if len(items) >= limit {
			continue
		}
		items = append(items, ReceitaItem{
			Codigo:          smarapdGetStr(r, "Codigo", "CodigoReceita", "Classificacao", "Estrutural", "Conta"),
			Descricao:       desc,
			Categoria:       smarapdGetStr(r, "Categoria", "FonteRecurso", "Fonte", "Origem", "Tipo"),
			Data:            smarapdGetStr(r, "Data", "Competencia", "DataArrecadacao"),
			ValorPrevisto:   smarapdGetFloat(r, "ValorPrevisto", "Previsto", "Orcado", "ValorOrcado", "PrevisaoAtualizada"),
			ValorArrecadado: smarapdGetFloat(r, "ValorArrecadado", "Arrecadado", "Realizado", "ValorRealizado"),
			Fonte:           "Portal Municipal (SmarAPD PAI)",
		})
	}

	if filtro != "" || total == 0 {
		total = matchedTotal
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma receita encontrada para os filtros consultados."
	}

	return &ReceitasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (SmarAPD PAI)",
		Plataforma:     "smarapd_pai",
		PortalURL:      rootURL,
		TotalRegistros: total,
		Receitas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QuerySmarapdContratos consulta contratos administrativos no portal SmarAPD PAI.
func QuerySmarapdContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ContratosResult, error) {
	rows, total, rootURL, err := fetchSmarapdModule(
		ctx, client, p,
		[]string{"contrato", "aditivo", "ata"},
		[]smarapdVisaoRef{
			{ChaveModulo: "Contratos", NomeVisao: "Contratos"},
			{ChaveModulo: "Contrato", NomeVisao: "Contratos"},
			{ChaveModulo: "ComprasContratos", NomeVisao: "Contratos"},
		},
	)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ContratoItem, 0, minInt(limit, len(rows)))
	matchedTotal := 0

	for _, r := range rows {
		fornecedor := smarapdGetStr(r, "Fornecedor", "Contratado", "NomeFornecedor", "NomeContratado", "RazaoSocial")
		objeto := smarapdGetStr(r, "Objeto", "Descricao", "ObjetoContrato")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+objeto), filtro) {
			continue
		}
		matchedTotal++
		if len(items) >= limit {
			continue
		}
		itemAno := int(smarapdGetFloat(r, "Ano", "Exercicio", "AnoContrato"))
		if itemAno == 0 {
			itemAno = p.Ano
		}
		doc := smarapdGetStr(r, "DocumentoFornecedor", "CPFCNPJ", "CNPJCPF", "CNPJ", "CPF", "Documento")

		items = append(items, ContratoItem{
			Numero:              smarapdGetStr(r, "Numero", "NumeroContrato", "Contrato", "Codigo"),
			Ano:                 itemAno,
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(doc),
			Objeto:              objeto,
			Situacao:            smarapdGetStr(r, "Situacao", "Status"),
			DataAssinatura:      smarapdGetStr(r, "DataAssinatura", "Assinatura", "Data"),
			VigenciaInicio:      smarapdGetStr(r, "VigenciaInicio", "DataInicio", "InicioVigencia"),
			VigenciaFim:         smarapdGetStr(r, "VigenciaFim", "DataFim", "FimVigencia", "DataTermino"),
			Valor:               smarapdGetFloat(r, "Valor", "ValorTotal", "ValorContrato", "ValorGlobal", "ValorContratado"),
			Fonte:               "Portal Municipal (SmarAPD PAI)",
		})
	}

	if filtro != "" || total == 0 {
		total = matchedTotal
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhum contrato encontrado para os filtros consultados."
	}

	return &ContratosResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (SmarAPD PAI)",
		Plataforma:     "smarapd_pai",
		PortalURL:      rootURL,
		TotalRegistros: total,
		Contratos:      items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QuerySmarapdLicitacoes consulta licitações no portal SmarAPD PAI.
func QuerySmarapdLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LicitacoesResult, error) {
	rows, total, rootURL, err := fetchSmarapdModule(
		ctx, client, p,
		[]string{"licitacao", "licitacoes", "pregao", "edital", "compra"},
		[]smarapdVisaoRef{
			{ChaveModulo: "Licitacoes", NomeVisao: "Licitacoes"},
			{ChaveModulo: "Licitacao", NomeVisao: "Licitacoes"},
			{ChaveModulo: "Compras", NomeVisao: "Licitacoes"},
		},
	)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]LicitacaoItem, 0, minInt(limit, len(rows)))
	matchedTotal := 0

	for _, r := range rows {
		objeto := smarapdGetStr(r, "Objeto", "Descricao", "ObjetoLicitacao")
		modalidade := smarapdGetStr(r, "Modalidade", "DescricaoModalidade", "Tipo")
		if filtro != "" && !strings.Contains(strings.ToLower(objeto+" "+modalidade), filtro) {
			continue
		}
		matchedTotal++
		if len(items) >= limit {
			continue
		}
		itemAno := int(smarapdGetFloat(r, "Ano", "Exercicio", "AnoLicitacao"))
		if itemAno == 0 {
			itemAno = p.Ano
		}
		num := smarapdGetStr(r, "Numero", "NumeroLicitacao", "NumeroProcesso", "Edital", "Codigo")
		if num == "" {
			if n := int(smarapdGetFloat(r, "Id", "Sequencial")); n > 0 {
				num = strconv.Itoa(n)
			}
		}

		items = append(items, LicitacaoItem{
			Numero:          num,
			Ano:             itemAno,
			Modalidade:      modalidade,
			Situacao:        smarapdGetStr(r, "Situacao", "Status"),
			DataAbertura:    smarapdGetStr(r, "DataAbertura", "DataPublicacao", "Data"),
			Objeto:          objeto,
			ValorEstimado:   smarapdGetFloat(r, "ValorEstimado", "ValorMaximo", "ValorPrevisto", "Valor"),
			ValorHomologado: smarapdGetFloat(r, "ValorHomologado", "ValorVencedor", "ValorAdjudicado", "ValorTotal"),
			Link:            smarapdGetStr(r, "Link", "URL", "Url"),
			Fonte:           "Portal Municipal (SmarAPD PAI)",
		})
	}

	if filtro != "" || total == 0 {
		total = matchedTotal
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma licitação encontrada para os filtros consultados."
	}

	return &LicitacoesResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (SmarAPD PAI)",
		Plataforma:     "smarapd_pai",
		PortalURL:      rootURL,
		TotalRegistros: total,
		Licitacoes:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}
