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
// NúcleoGov Portal da Transparência Connector (plataforma = "nucleogov")
// Endpoints típicos:
//   POST <root>/api/orgaos          (acao=listar&limit=0,100)
//   POST <root>/api/servidores_cnt  (acao=listar&ano=Y&mes=M&limit=0,N)
//   POST <root>/api                 (multi_request=true&params={"r1":{...}})
//   POST <root>/api/centi           (acao=empenhos|receitas)
//   POST <root>/api/despesas        (acao=listar)
//   POST <root>/api/contratos       (acao=listar&limit=0,N)
//   POST <root>/api/licitacoes      (acao=listar&limit=0,N)
// ---------------------------------------------------------------------------

const (
	nucleoGovFonte      = "Portal Municipal (NúcleoGov)"
	nucleoGovPlataforma = "nucleogov"
)

func normalizeNucleoGovRoot(baseURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return ""
	}
	u, err := url.Parse(trimmed)
	if err == nil && u.Scheme != "" && u.Host != "" {
		u.RawQuery = ""
		u.Fragment = ""
		lowPath := strings.ToLower(u.Path)
		if idx := strings.Index(lowPath, "/cidadao"); idx >= 0 {
			u.Path = u.Path[:idx]
		}
		u.Path = strings.TrimRight(u.Path, "/")
		if strings.HasSuffix(strings.ToLower(u.Path), "/api") {
			u.Path = u.Path[:len(u.Path)-len("/api")]
		}
		return strings.TrimRight(u.String(), "/")
	}

	low := strings.ToLower(trimmed)
	if idx := strings.Index(low, "/cidadao"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	trimmed = strings.TrimRight(trimmed, "/")
	if strings.HasSuffix(strings.ToLower(trimmed), "/api") {
		trimmed = trimmed[:len(trimmed)-len("/api")]
	}
	return strings.TrimRight(trimmed, "/")
}

func doNucleoGovPostForm(ctx context.Context, client *http.Client, targetURL string, form url.Values) ([]byte, int, error) {
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

func decodeNucleoGovObjRecordsMeta(obj map[string]any, extraKeys ...string) ([]map[string]any, int, bool) {
	if rawR1, ok := obj["r1"]; ok && rawR1 != nil {
		switch t := rawR1.(type) {
		case []any:
			rows := make([]map[string]any, 0, len(t))
			for _, el := range t {
				if m, ok := el.(map[string]any); ok {
					rows = append(rows, m)
				}
			}
			return rows, 0, false
		case map[string]any:
			return decodeNucleoGovObjRecordsMeta(t, extraKeys...)
		}
	}

	keys := append([]string{}, extraKeys...)
	keys = append(keys,
		"dados", "registros", "Registros", "data", "lista", "resultado",
		"items", "content", "rows", "orgaos", "servidores", "empenhos",
		"despesas", "receitas", "contratos", "licitacoes",
	)
	var rows []map[string]any
	for _, k := range keys {
		if raw, ok := obj[k]; ok && raw != nil {
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

	total := int(mapGetFloat(obj, "total", "Total", "totalRegistros", "total_registros", "TotalRegistros", "totalElements", "TotalElements", "recordsTotal", "RecordsTotal", "count", "Count"))
	return rows, total, total > 0
}

func decodeNucleoGovObjRecords(obj map[string]any, extraKeys ...string) ([]map[string]any, int) {
	rows, total, hasTotal := decodeNucleoGovObjRecordsMeta(obj, extraKeys...)
	if !hasTotal {
		total = len(rows)
	}
	return rows, total
}

func decodeNucleoGovRecordsMeta(body []byte, extraKeys ...string) ([]map[string]any, int, bool, error) {
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
	rows, total, hasTotal := decodeNucleoGovObjRecordsMeta(obj, extraKeys...)
	return rows, total, hasTotal, nil
}

func decodeNucleoGovRecords(body []byte, extraKeys ...string) ([]map[string]any, int, error) {
	rows, total, hasTotal, err := decodeNucleoGovRecordsMeta(body, extraKeys...)
	if err != nil {
		return nil, 0, err
	}
	if !hasTotal {
		total = len(rows)
	}
	return rows, total, nil
}

func isNucleoGovOrgaosJSON(body []byte) (confirmed bool, validJSON bool) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return false, false
	}
	if strings.HasPrefix(trimmed, "[") {
		var arr []map[string]any
		if err := json.Unmarshal([]byte(trimmed), &arr); err != nil {
			return false, false
		}
		if len(arr) > 0 {
			return true, true
		}
		return false, true
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil {
		return false, false
	}
	for _, k := range []string{"dados", "registros", "orgaos", "r1", "sucesso", "status", "Id", "id", "Nome", "nome", "total", "totalRegistros"} {
		if _, ok := obj[k]; ok {
			return true, true
		}
	}
	return false, true
}

// ProbeNucleoGov verifica se a URL base corresponde a um portal NúcleoGov (/api/orgaos, /api multi_request ou static.nucleogov.com.br).
func ProbeNucleoGov(ctx context.Context, client *http.Client, baseURL string) bool {
	root := normalizeNucleoGovRoot(baseURL)
	if root == "" {
		return false
	}
	hc := portalConnectorHTTPClient(client)

	// 1. POST <root>/api/orgaos (acao=listar&limit=0,1)
	orgaoForm := url.Values{}
	orgaoForm.Set("acao", "listar")
	orgaoForm.Set("limit", "0,1")

	var orgaos200ValidJSON bool
	if body, status, err := doNucleoGovPostForm(ctx, hc, root+"/api/orgaos", orgaoForm); err == nil && status == http.StatusOK {
		confirmed, valid := isNucleoGovOrgaosJSON(body)
		if confirmed {
			return true
		}
		orgaos200ValidJSON = valid
	}

	// 2. POST <root>/api (multi_request=true&params=...)
	multiForm := url.Values{}
	multiForm.Set("multi_request", "true")
	multiForm.Set("params", `{"r1":{"acao":"orgaos/listar","limit":"0,1"}}`)
	if body, status, err := doNucleoGovPostForm(ctx, hc, root+"/api", multiForm); err == nil && status == http.StatusOK {
		var obj map[string]any
		if err := json.Unmarshal(body, &obj); err == nil {
			if _, hasR1 := obj["r1"]; hasR1 {
				return true
			}
		}
	}

	// 3. GET <baseURL> procurando static.nucleogov.com.br + resposta 200 JSON em /api/orgaos
	if orgaos200ValidJSON {
		candidates := dedupeStrings([]string{strings.TrimRight(strings.TrimSpace(baseURL), "/"), root})
		for _, target := range candidates {
			if body, status, err := doPortalRawGet(ctx, hc, target, "text/html,application/xhtml+xml,*/*"); err == nil && status == http.StatusOK {
				low := strings.ToLower(string(body))
				if strings.Contains(low, "static.nucleogov.com.br") || strings.Contains(low, "nucleogov") {
					return true
				}
			}
		}
	}

	return false
}

func isNucleoGovTruthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s == "1" || s == "true" || s == "s" || s == "sim"
	default:
		return false
	}
}

func fetchNucleoGovPrincipalOrgao(ctx context.Context, client *http.Client, root string) string {
	form := url.Values{}
	form.Set("acao", "listar")
	form.Set("limit", "0,100")

	var rows []map[string]any
	if body, status, err := doNucleoGovPostForm(ctx, client, root+"/api/orgaos", form); err == nil && status == http.StatusOK {
		if parsed, _, decErr := decodeNucleoGovRecords(body, "orgaos"); decErr == nil {
			rows = parsed
		}
	}

	if len(rows) == 0 {
		multiForm := url.Values{}
		multiForm.Set("multi_request", "true")
		multiForm.Set("params", `{"r1":{"acao":"orgaos/listar","limit":"0,100"}}`)
		if body, status, err := doNucleoGovPostForm(ctx, client, root+"/api", multiForm); err == nil && status == http.StatusOK {
			if parsed, _, decErr := decodeNucleoGovRecords(body, "orgaos"); decErr == nil {
				rows = parsed
			}
		}
	}

	firstID := ""
	prefeituraID := ""
	for _, r := range rows {
		id := mapGetStr(r, "Id", "id", "cod_orgao", "CodOrgao", "codigo", "Codigo", "id_orgao", "IdOrgao")
		if id == "" {
			continue
		}
		if firstID == "" {
			firstID = id
		}
		for _, k := range []string{"Principal", "principal", "Padrao", "padrao"} {
			if raw, ok := r[k]; ok && isNucleoGovTruthy(raw) {
				return id
			}
		}
		nome := strings.ToUpper(mapGetStr(r, "Nome", "nome", "Descricao", "descricao", "Orgao", "orgao", "RazaoSocial", "razao_social"))
		if prefeituraID == "" && (strings.Contains(nome, "PREFEITURA") || strings.Contains(nome, "MUNIC")) {
			prefeituraID = id
		}
	}
	if prefeituraID != "" {
		return prefeituraID
	}
	if firstID != "" {
		return firstID
	}
	return "1"
}

type nucleoGovCompetencia struct {
	ano int
	mes int
}

func resolveNucleoGovCompetencias(p PortalQueryParams) []nucleoGovCompetencia {
	if p.Ano > 0 && p.Mes > 0 {
		return []nucleoGovCompetencia{{ano: p.Ano, mes: p.Mes}}
	}
	now := time.Now()
	nowY := now.Year()
	nowM := int(now.Month())

	if p.Ano > 0 {
		if p.Ano == nowY {
			var out []nucleoGovCompetencia
			for m := nowM; m >= 1 && len(out) < 4; m-- {
				out = append(out, nucleoGovCompetencia{ano: p.Ano, mes: m})
			}
			out = append(out, nucleoGovCompetencia{ano: p.Ano, mes: 0})
			return out
		}
		return []nucleoGovCompetencia{
			{ano: p.Ano, mes: 12},
			{ano: p.Ano, mes: 6},
			{ano: p.Ano, mes: 1},
			{ano: p.Ano, mes: 0},
		}
	}

	if p.Mes > 0 {
		return []nucleoGovCompetencia{
			{ano: nowY, mes: p.Mes},
			{ano: nowY - 1, mes: p.Mes},
		}
	}

	var out []nucleoGovCompetencia
	y, m := nowY, nowM
	for i := 0; i < 4; i++ {
		out = append(out, nucleoGovCompetencia{ano: y, mes: m})
		m--
		if m <= 0 {
			m = 12
			y--
		}
	}
	if nowY-1 != y || 12 != m {
		out = append(out, nucleoGovCompetencia{ano: nowY - 1, mes: 12})
	}
	out = append(out, nucleoGovCompetencia{ano: 0, mes: 0})
	return out
}

func resolveNucleoGovYears(p PortalQueryParams) []int {
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

func formatNucleoGovDatePTBR(raw string) string {
	s := strings.TrimSpace(raw)
	if len(s) == 10 && s[4] == '-' && s[7] == '-' {
		return s[8:10] + "/" + s[5:7] + "/" + s[0:4]
	}
	return s
}

func resolveNucleoGovDateWindows(p PortalQueryParams) [][3]string {
	if p.Inicio != "" && p.Fim != "" {
		ano := p.Ano
		if ano <= 0 && len(p.Inicio) >= 4 {
			if y, err := strconv.Atoi(p.Inicio[:4]); err == nil {
				ano = y
			}
		}
		if ano <= 0 {
			ano = time.Now().Year()
		}
		return [][3]string{{formatNucleoGovDatePTBR(p.Inicio), formatNucleoGovDatePTBR(p.Fim), strconv.Itoa(ano)}}
	}
	years := resolveNucleoGovYears(p)
	out := make([][3]string, 0, len(years))
	for _, y := range years {
		if p.Mes >= 1 && p.Mes <= 12 {
			first := time.Date(y, time.Month(p.Mes), 1, 0, 0, 0, 0, time.UTC)
			last := first.AddDate(0, 1, -1)
			out = append(out, [3]string{
				first.Format("02/01/2006"),
				last.Format("02/01/2006"),
				strconv.Itoa(y),
			})
		} else {
			out = append(out, [3]string{
				fmt.Sprintf("01/01/%04d", y),
				fmt.Sprintf("31/12/%04d", y),
				strconv.Itoa(y),
			})
		}
	}
	return out
}

func fetchNucleoGovMultiRequest(ctx context.Context, client *http.Client, root, acao string, extra map[string]any) ([]map[string]any, int, int, error) {
	r1 := map[string]any{
		"acao": acao,
	}
	for k, v := range extra {
		r1[k] = v
	}
	payloadBytes, err := json.Marshal(map[string]any{"r1": r1})
	if err != nil {
		return nil, 0, 0, err
	}
	form := url.Values{}
	form.Set("multi_request", "true")
	form.Set("params", string(payloadBytes))

	body, status, err := doNucleoGovPostForm(ctx, client, root+"/api", form)
	if err != nil {
		return nil, 0, status, err
	}
	if status != http.StatusOK || len(strings.TrimSpace(string(body))) == 0 {
		return nil, 0, status, nil
	}
	rows, total, hasTotal, err := decodeNucleoGovRecordsMeta(body)
	if !hasTotal {
		total = 0
	}
	return rows, total, status, err
}

func nucleoGovRowsSignature(rows []map[string]any) string {
	b, err := json.Marshal(rows)
	if err != nil {
		return ""
	}
	return string(b)
}

func fetchNucleoGovPaginated(
	scanLimit int,
	fetchPage func(offset, pageSize int) (rows []map[string]any, total int, status int, err error),
) (allRows []map[string]any, total int, lastStatus int, err error) {
	pageSize := normalizePageSize(scanLimit, 200)
	prevSig := ""
	for page := 0; page < 10; page++ {
		offset := page * pageSize
		pageRows, pageTotal, status, pageErr := fetchPage(offset, pageSize)
		if status > 0 {
			lastStatus = status
		}
		if page == 0 {
			if pageErr != nil {
				return nil, 0, lastStatus, pageErr
			}
			if len(pageRows) == 0 {
				return nil, 0, lastStatus, nil
			}
		} else {
			if pageErr != nil || len(pageRows) == 0 {
				break
			}
		}

		sig := nucleoGovRowsSignature(pageRows)
		if page > 0 && sig != "" && sig == prevSig {
			break
		}
		prevSig = sig

		if pageTotal > total {
			total = pageTotal
		}
		allRows = append(allRows, pageRows...)
		if len(allRows) >= scanLimit || len(pageRows) < pageSize || (total > 0 && len(allRows) >= total) {
			break
		}
	}
	return allRows, total, lastStatus, nil
}

// QueryNucleoGovServidores consulta servidores públicos no portal NúcleoGov (/api/servidores_cnt ou /api multi_request).
func QueryNucleoGovServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ServidoresResult, error) {
	root := normalizeNucleoGovRoot(p.BaseURL)
	if root == "" {
		return nil, fmt.Errorf("NúcleoGov: BaseURL não informada")
	}
	hc := portalConnectorHTTPClient(client)
	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	scanLimit := limit
	if filtro != "" && scanLimit < 500 {
		scanLimit = 500
	}

	var lastErr error
	var lastStatus int

	for _, comp := range resolveNucleoGovCompetencias(p) {
		useMulti := false
		rows, total, status, pageErr := fetchNucleoGovPaginated(scanLimit, func(offset, pageSize int) ([]map[string]any, int, int, error) {
			limitParam := fmt.Sprintf("%d,%d", offset, pageSize)
			if !useMulti {
				form := url.Values{}
				form.Set("acao", "listar")
				form.Set("limit", limitParam)
				if comp.ano > 0 {
					form.Set("ano", strconv.Itoa(comp.ano))
				}
				if comp.mes > 0 {
					form.Set("mes", strconv.Itoa(comp.mes))
				}
				if strings.TrimSpace(p.NomeFiltro) != "" {
					form.Set("nome", strings.TrimSpace(p.NomeFiltro))
				}

				body, st, reqErr := doNucleoGovPostForm(ctx, hc, root+"/api/servidores_cnt", form)
				if reqErr != nil {
					if offset == 0 {
						lastErr = reqErr
					}
				} else if st == http.StatusOK && len(strings.TrimSpace(string(body))) > 0 {
					parsed, tot, _, decErr := decodeNucleoGovRecordsMeta(body, "servidores")
					if decErr == nil && len(parsed) > 0 {
						return parsed, tot, st, nil
					}
					if decErr != nil && offset == 0 {
						lastErr = fmt.Errorf("erro ao decodificar NúcleoGov /api/servidores_cnt: %w", decErr)
					}
				} else if offset == 0 {
					lastStatus = st
				}

				if offset > 0 {
					return nil, 0, st, nil
				}
				useMulti = true
			}

			extra := map[string]any{
				"limit": limitParam,
			}
			if comp.ano > 0 {
				extra["ano"] = comp.ano
			}
			if comp.mes > 0 {
				extra["mes"] = comp.mes
			}
			if strings.TrimSpace(p.NomeFiltro) != "" {
				extra["nome"] = strings.TrimSpace(p.NomeFiltro)
			}
			mRows, mTotal, mStatus, mErr := fetchNucleoGovMultiRequest(ctx, hc, root, "servidores_cnt/listar", extra)
			if mErr != nil && lastErr == nil {
				lastErr = mErr
			}
			return mRows, mTotal, mStatus, mErr
		})
		if status > 0 {
			lastStatus = status
		}
		if pageErr != nil && lastErr == nil {
			lastErr = pageErr
		}
		if len(rows) == 0 {
			continue
		}

		items := make([]ServidorItem, 0, minInt(limit, len(rows)))
		matchedTotal := 0
		for _, r := range rows {
			nome := mapGetStr(r, "Nome", "nome", "Servidor", "servidor", "Funcionario", "funcionario", "nom_servidor", "NomServidor")
			if filtro != "" && !strings.Contains(strings.ToLower(nome), filtro) {
				continue
			}
			matchedTotal++
			if len(items) >= limit {
				continue
			}

			bruto := mapGetFloat(r, "SalarioBruto", "salario_bruto", "Bruto", "bruto", "TotalProventos", "total_proventos", "Proventos", "proventos", "Remuneracao", "remuneracao", "ValorBruto", "valor_bruto")
			descontos := mapGetFloat(r, "Descontos", "descontos", "TotalDescontos", "total_descontos", "ValorDescontos", "valor_descontos", "Desconto", "desconto")
			liquido := mapGetFloat(r, "SalarioLiquido", "salario_liquido", "Liquido", "liquido", "ValorLiquido", "valor_liquido")
			if liquido == 0 && bruto > 0 && descontos > 0 {
				liquido = bruto - descontos
			}

			compStr := mapGetStr(r, "Competencia", "competencia", "Referencia", "referencia")
			if compStr == "" && comp.ano > 0 && comp.mes > 0 {
				compStr = fmt.Sprintf("%02d/%04d", comp.mes, comp.ano)
			} else if compStr == "" && comp.ano > 0 {
				compStr = strconv.Itoa(comp.ano)
			}

			items = append(items, ServidorItem{
				Nome:           nome,
				Matricula:      mapGetStr(r, "Matricula", "matricula", "Registro", "registro", "Codigo", "codigo", "Id", "id"),
				CPFParcial:     maskCPF(mapGetStr(r, "CPF_Formatado", "cpf_formatado", "CPFFormatado", "cpfFormatado"), mapGetStr(r, "CPF", "cpf", "NumCPF", "num_cpf", "Documento", "documento")),
				Cargo:          mapGetStr(r, "Cargo", "cargo", "DescricaoCargo", "descricao_cargo", "NomeCargo", "nome_cargo"),
				Funcao:         mapGetStr(r, "Funcao", "funcao", "DescricaoFuncao", "descricao_funcao"),
				Lotacao:        mapGetStr(r, "Lotacao", "lotacao", "Departamento", "departamento", "Secretaria", "secretaria", "Setor", "setor"),
				LocalTrabalho:  mapGetStr(r, "LocalTrabalho", "local_trabalho", "Unidade", "unidade"),
				Vinculo:        mapGetStr(r, "Vinculo", "vinculo", "Regime", "regime", "TipoVinculo", "tipo_vinculo", "Tipo", "tipo"),
				Situacao:       mapGetStr(r, "Situacao", "situacao", "Status", "status"),
				SalarioBase:    mapGetFloat(r, "SalarioBase", "salario_base", "VencimentoBase", "vencimento_base", "Base", "base"),
				SalarioBruto:   bruto,
				Descontos:      descontos,
				SalarioLiquido: liquido,
				Competencia:    compStr,
				CargaHoraria:   mapGetStr(r, "CargaHoraria", "carga_horaria", "Horas", "horas"),
				DataAdmissao:   mapGetStr(r, "DataAdmissao", "data_admissao", "Admissao", "admissao"),
				Orgao:          mapGetStr(r, "Orgao", "orgao", "NomeOrgao", "nome_orgao", "Entidade", "entidade"),
				Fonte:          nucleoGovFonte,
				PortalURL:      root,
			})
		}

		if len(items) == 0 && filtro != "" {
			continue
		}
		if filtro != "" {
			total = matchedTotal
		} else if total == 0 {
			total = matchedTotal
		}
		return &ServidoresResult{
			Municipio:      m,
			Fonte:          nucleoGovFonte,
			Plataforma:     nucleoGovPlataforma,
			PortalURL:      root,
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
		Fonte:      nucleoGovFonte,
		Plataforma: nucleoGovPlataforma,
		PortalURL:  root,
		Servidores: []ServidorItem{},
		Disponivel: true,
		Mensagem:   "Nenhum servidor encontrado para os filtros e período consultados.",
	}, nil
}

// QueryNucleoGovDespesas consulta empenhos/despesas no portal NúcleoGov (/api/centi acao=empenhos ou fallback /api/despesas).
func QueryNucleoGovDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	root := normalizeNucleoGovRoot(p.BaseURL)
	if root == "" {
		return nil, fmt.Errorf("NúcleoGov: BaseURL não informada")
	}
	hc := portalConnectorHTTPClient(client)
	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	scanLimit := limit
	if filtro != "" && scanLimit < 500 {
		scanLimit = 500
	}

	var lastErr error
	var lastStatus int

	for _, win := range resolveNucleoGovDateWindows(p) {
		anoInt, _ := strconv.Atoi(win[2])
		useFallback := false
		rows, total, status, pageErr := fetchNucleoGovPaginated(scanLimit, func(offset, pageSize int) ([]map[string]any, int, int, error) {
			limitParam := fmt.Sprintf("%d,%d", offset, pageSize)
			if !useFallback {
				centiForm := url.Values{}
				centiForm.Set("acao", "empenhos")
				centiForm.Set("data_inicio", win[0])
				centiForm.Set("data_fim", win[1])
				centiForm.Set("limit", limitParam)

				body, st, reqErr := doNucleoGovPostForm(ctx, hc, root+"/api/centi", centiForm)
				if reqErr != nil {
					if offset == 0 {
						lastErr = reqErr
					}
				} else if st == http.StatusOK && len(strings.TrimSpace(string(body))) > 0 {
					parsed, tot, _, decErr := decodeNucleoGovRecordsMeta(body, "empenhos", "despesas")
					if decErr == nil && len(parsed) > 0 {
						return parsed, tot, st, nil
					}
					if decErr != nil && offset == 0 {
						lastErr = fmt.Errorf("erro ao decodificar NúcleoGov /api/centi empenhos: %w", decErr)
					}
				} else if offset == 0 {
					lastStatus = st
				}

				if offset > 0 {
					return nil, 0, st, nil
				}
				useFallback = true
			}

			despForm := url.Values{}
			despForm.Set("acao", "listar")
			despForm.Set("ano", win[2])
			despForm.Set("data_inicio", win[0])
			despForm.Set("data_fim", win[1])
			despForm.Set("limit", limitParam)

			body, st, reqErr := doNucleoGovPostForm(ctx, hc, root+"/api/despesas", despForm)
			if reqErr != nil {
				return nil, 0, st, reqErr
			}
			if st == http.StatusOK && len(strings.TrimSpace(string(body))) > 0 {
				parsed, tot, _, decErr := decodeNucleoGovRecordsMeta(body, "despesas", "empenhos")
				if decErr != nil {
					return nil, 0, st, fmt.Errorf("erro ao decodificar NúcleoGov /api/despesas: %w", decErr)
				}
				return parsed, tot, st, nil
			}
			return nil, 0, st, nil
		})
		if status > 0 {
			lastStatus = status
		}
		if pageErr != nil && lastErr == nil {
			lastErr = pageErr
		}
		if len(rows) == 0 {
			continue
		}

		items := make([]DespesaItem, 0, minInt(limit, len(rows)))
		matchedTotal := 0
		for _, r := range rows {
			fornecedor := mapGetStr(r, "Fornecedor", "fornecedor", "Credor", "credor", "NomeFornecedor", "nome_fornecedor", "RazaoSocial", "razao_social", "Nome", "nome")
			historico := mapGetStr(r, "Historico", "historico", "Descricao", "descricao", "Objeto", "objeto")
			elemento := mapGetStr(r, "Elemento", "elemento", "ElementoDespesa", "elemento_despesa", "Natureza", "natureza")
			if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+historico+" "+elemento), filtro) {
				continue
			}
			matchedTotal++
			if len(items) >= limit {
				continue
			}

			itemAno := int(mapGetFloat(r, "Ano", "ano", "Exercicio", "exercicio"))
			if itemAno == 0 {
				itemAno = anoInt
			}
			doc := maskIfCPF(mapGetStr(r, "DocumentoFornecedor", "documento_fornecedor", "CPFCNPJ", "cpf_cnpj", "CNPJCPF", "cnpj_cpf", "CNPJ", "cnpj", "CPF", "cpf", "Documento", "documento"))

			items = append(items, DespesaItem{
				NumeroEmpenho:       mapGetStr(r, "NumeroEmpenho", "numero_empenho", "Numero", "numero", "Empenho", "empenho", "NumEmpenho", "num_empenho"),
				Ano:                 itemAno,
				Data:                mapGetStr(r, "Data", "data", "DataEmpenho", "data_empenho", "DataMovimento", "data_movimento"),
				Fornecedor:          fornecedor,
				DocumentoFornecedor: doc,
				Orgao:               mapGetStr(r, "Orgao", "orgao", "NomeOrgao", "nome_orgao"),
				Unidade:             mapGetStr(r, "Unidade", "unidade", "UnidadeGestora", "unidade_gestora"),
				Funcao:              mapGetStr(r, "Funcao", "funcao"),
				Elemento:            elemento,
				Historico:           historico,
				ValorEmpenhado:      mapGetFloat(r, "ValorEmpenhado", "valor_empenhado", "Empenhado", "empenhado", "Valor", "valor"),
				ValorLiquidado:      mapGetFloat(r, "ValorLiquidado", "valor_liquidado", "Liquidado", "liquidado"),
				ValorPago:           mapGetFloat(r, "ValorPago", "valor_pago", "Pago", "pago"),
				Fonte:               nucleoGovFonte,
			})
		}

		if len(items) == 0 && filtro != "" {
			continue
		}
		if filtro != "" {
			total = matchedTotal
		} else if total == 0 {
			total = matchedTotal
		}
		return &DespesasResult{
			Municipio:      m,
			Fonte:          nucleoGovFonte,
			Plataforma:     nucleoGovPlataforma,
			PortalURL:      root,
			TotalRegistros: total,
			Despesas:       items,
			Disponivel:     true,
		}, nil
	}

	if lastErr != nil && lastStatus == 0 {
		return nil, lastErr
	}

	return &DespesasResult{
		Municipio:  m,
		Fonte:      nucleoGovFonte,
		Plataforma: nucleoGovPlataforma,
		PortalURL:  root,
		Despesas:   []DespesaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma despesa encontrada para o período informado.",
	}, nil
}

// QueryNucleoGovReceitas consulta receitas no portal NúcleoGov (obtém órgão principal via /api/orgaos e consulta /api/centi acao=receitas).
func QueryNucleoGovReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	root := normalizeNucleoGovRoot(p.BaseURL)
	if root == "" {
		return nil, fmt.Errorf("NúcleoGov: BaseURL não informada")
	}
	hc := portalConnectorHTTPClient(client)
	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	scanLimit := limit
	if filtro != "" && scanLimit < 500 {
		scanLimit = 500
	}

	orgaoID := ""
	if p.CodAdministracao > 0 {
		orgaoID = strconv.Itoa(p.CodAdministracao)
	} else {
		orgaoID = fetchNucleoGovPrincipalOrgao(ctx, hc, root)
	}

	now := time.Now()
	nowY := now.Year()
	nowM := int(now.Month())

	var comps []nucleoGovCompetencia
	if p.Ano > 0 && p.Mes > 0 {
		comps = []nucleoGovCompetencia{{ano: p.Ano, mes: p.Mes}}
	} else {
		for _, ano := range resolveNucleoGovYears(p) {
			if p.Mes > 0 {
				comps = append(comps, nucleoGovCompetencia{ano: ano, mes: p.Mes})
			} else if ano == nowY {
				comps = append(comps, nucleoGovCompetencia{ano: ano, mes: nowM})
				if nowM != 12 {
					comps = append(comps, nucleoGovCompetencia{ano: ano, mes: 12})
				}
			} else {
				comps = append(comps, nucleoGovCompetencia{ano: ano, mes: 12})
			}
		}
	}

	var lastErr error
	var lastStatus int

	for _, comp := range comps {
		useFallback := false
		rows, total, status, pageErr := fetchNucleoGovPaginated(scanLimit, func(offset, pageSize int) ([]map[string]any, int, int, error) {
			limitParam := fmt.Sprintf("%d,%d", offset, pageSize)
			if !useFallback {
				centiForm := url.Values{}
				centiForm.Set("acao", "receitas")
				centiForm.Set("ano", strconv.Itoa(comp.ano))
				centiForm.Set("mes", strconv.Itoa(comp.mes))
				centiForm.Set("orgao", orgaoID)
				centiForm.Set("limit", limitParam)

				body, st, reqErr := doNucleoGovPostForm(ctx, hc, root+"/api/centi", centiForm)
				if reqErr != nil {
					if offset == 0 {
						lastErr = reqErr
					}
				} else if st == http.StatusOK && len(strings.TrimSpace(string(body))) > 0 {
					parsed, tot, _, decErr := decodeNucleoGovRecordsMeta(body, "receitas")
					if decErr == nil && len(parsed) > 0 {
						return parsed, tot, st, nil
					}
					if decErr != nil && offset == 0 {
						lastErr = fmt.Errorf("erro ao decodificar NúcleoGov /api/centi receitas: %w", decErr)
					}
				} else if offset == 0 {
					lastStatus = st
				}

				if offset > 0 {
					return nil, 0, st, nil
				}
				useFallback = true
			}

			recForm := url.Values{}
			recForm.Set("acao", "listar")
			recForm.Set("ano", strconv.Itoa(comp.ano))
			recForm.Set("mes", strconv.Itoa(comp.mes))
			recForm.Set("orgao", orgaoID)
			recForm.Set("limit", limitParam)

			body, st, reqErr := doNucleoGovPostForm(ctx, hc, root+"/api/receitas", recForm)
			if reqErr != nil {
				return nil, 0, st, reqErr
			}
			if st == http.StatusOK && len(strings.TrimSpace(string(body))) > 0 {
				parsed, tot, _, decErr := decodeNucleoGovRecordsMeta(body, "receitas")
				if decErr != nil {
					return nil, 0, st, fmt.Errorf("erro ao decodificar NúcleoGov /api/receitas: %w", decErr)
				}
				return parsed, tot, st, nil
			}
			return nil, 0, st, nil
		})
		if status > 0 {
			lastStatus = status
		}
		if pageErr != nil && lastErr == nil {
			lastErr = pageErr
		}
		if len(rows) == 0 {
			continue
		}

		items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
		matchedTotal := 0
		for _, r := range rows {
			desc := mapGetStr(r, "Descricao", "descricao", "Receita", "receita", "Especificacao", "especificacao", "Nome", "nome")
			cod := mapGetStr(r, "Codigo", "codigo", "Conta", "conta", "Classificacao", "classificacao", "Natureza", "natureza")
			if filtro != "" && !strings.Contains(strings.ToLower(desc+" "+cod), filtro) {
				continue
			}
			matchedTotal++
			if len(items) >= limit {
				continue
			}

			items = append(items, ReceitaItem{
				Codigo:          cod,
				Descricao:       desc,
				Categoria:       mapGetStr(r, "Categoria", "categoria", "FonteRecurso", "fonte_recurso", "Origem", "origem"),
				Data:            mapGetStr(r, "Data", "data", "Competencia", "competencia"),
				ValorPrevisto:   mapGetFloat(r, "ValorPrevisto", "valor_previsto", "Previsto", "previsto", "Orcado", "orcado", "ValorOrcado", "valor_orcado"),
				ValorArrecadado: mapGetFloat(r, "ValorArrecadado", "valor_arrecadado", "Arrecadado", "arrecadado", "Realizado", "realizado", "ValorRealizado", "valor_realizado"),
				Fonte:           nucleoGovFonte,
			})
		}

		if len(items) == 0 && filtro != "" {
			continue
		}
		if filtro != "" {
			total = matchedTotal
		} else if total == 0 {
			total = matchedTotal
		}
		return &ReceitasResult{
			Municipio:      m,
			Fonte:          nucleoGovFonte,
			Plataforma:     nucleoGovPlataforma,
			PortalURL:      root,
			TotalRegistros: total,
			Receitas:       items,
			Disponivel:     true,
		}, nil
	}

	if lastErr != nil && lastStatus == 0 {
		return nil, lastErr
	}

	return &ReceitasResult{
		Municipio:  m,
		Fonte:      nucleoGovFonte,
		Plataforma: nucleoGovPlataforma,
		PortalURL:  root,
		Receitas:   []ReceitaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma receita encontrada para o período informado.",
	}, nil
}

// QueryNucleoGovContratos consulta contratos administrativos no portal NúcleoGov (/api/contratos).
func QueryNucleoGovContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ContratosResult, error) {
	root := normalizeNucleoGovRoot(p.BaseURL)
	if root == "" {
		return nil, fmt.Errorf("NúcleoGov: BaseURL não informada")
	}
	hc := portalConnectorHTTPClient(client)
	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	scanLimit := limit
	if filtro != "" && scanLimit < 500 {
		scanLimit = 500
	}

	useMulti := false
	rows, total, _, err := fetchNucleoGovPaginated(scanLimit, func(offset, pageSize int) ([]map[string]any, int, int, error) {
		limitParam := fmt.Sprintf("%d,%d", offset, pageSize)
		if !useMulti {
			form := url.Values{}
			form.Set("acao", "listar")
			form.Set("limit", limitParam)
			if p.Ano > 0 {
				form.Set("ano", strconv.Itoa(p.Ano))
			}

			body, status, reqErr := doNucleoGovPostForm(ctx, hc, root+"/api/contratos", form)
			if reqErr != nil {
				return nil, 0, status, reqErr
			}
			if status == http.StatusOK && len(strings.TrimSpace(string(body))) > 0 {
				parsed, tot, _, decErr := decodeNucleoGovRecordsMeta(body, "contratos")
				if decErr != nil {
					return nil, 0, status, fmt.Errorf("erro ao decodificar NúcleoGov /api/contratos: %w", decErr)
				}
				if len(parsed) > 0 {
					return parsed, tot, status, nil
				}
			}

			if offset > 0 {
				return nil, 0, status, nil
			}
			useMulti = true

			extra := map[string]any{"limit": limitParam}
			if p.Ano > 0 {
				extra["ano"] = p.Ano
			}
			mRows, mTotal, mStatus, mErr := fetchNucleoGovMultiRequest(ctx, hc, root, "contratos/listar", extra)
			if mErr == nil && len(mRows) > 0 {
				return mRows, mTotal, mStatus, nil
			}
			if status != http.StatusOK && status != 0 && mStatus != http.StatusOK {
				return nil, 0, status, fmt.Errorf("NúcleoGov /api/contratos retornou HTTP %d", status)
			}
			return mRows, mTotal, mStatus, mErr
		}

		extra := map[string]any{"limit": limitParam}
		if p.Ano > 0 {
			extra["ano"] = p.Ano
		}
		return fetchNucleoGovMultiRequest(ctx, hc, root, "contratos/listar", extra)
	})
	if err != nil {
		return nil, err
	}

	items := make([]ContratoItem, 0, minInt(limit, len(rows)))
	matchedTotal := 0
	for _, r := range rows {
		fornecedor := mapGetStr(r, "Fornecedor", "fornecedor", "Contratado", "contratado", "RazaoSocial", "razao_social", "NomeFornecedor", "nome_fornecedor")
		objeto := mapGetStr(r, "Objeto", "objeto", "Descricao", "descricao")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+objeto), filtro) {
			continue
		}
		matchedTotal++
		if len(items) >= limit {
			continue
		}

		items = append(items, ContratoItem{
			Numero:              mapGetStr(r, "Numero", "numero", "NumeroContrato", "numero_contrato", "Contrato", "contrato"),
			Ano:                 int(mapGetFloat(r, "Ano", "ano", "Exercicio", "exercicio")),
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(mapGetStr(r, "DocumentoFornecedor", "documento_fornecedor", "CPFCNPJ", "cpf_cnpj", "CNPJ", "cnpj", "CPF", "cpf", "Documento", "documento")),
			Objeto:              objeto,
			Situacao:            mapGetStr(r, "Situacao", "situacao", "Status", "status"),
			DataAssinatura:      mapGetStr(r, "DataAssinatura", "data_assinatura", "Data", "data", "Assinatura", "assinatura"),
			VigenciaInicio:      mapGetStr(r, "VigenciaInicio", "vigencia_inicio", "DataInicio", "data_inicio", "Inicio", "inicio"),
			VigenciaFim:         mapGetStr(r, "VigenciaFim", "vigencia_fim", "DataFim", "data_fim", "Fim", "fim", "Vencimento", "vencimento"),
			Valor:               mapGetFloat(r, "Valor", "valor", "ValorContrato", "valor_contrato", "ValorGlobal", "valor_global", "ValorTotal", "valor_total"),
			Fonte:               nucleoGovFonte,
		})
	}

	if filtro != "" {
		total = matchedTotal
	} else if total == 0 {
		total = matchedTotal
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhum contrato encontrado para os critérios informados."
	}

	return &ContratosResult{
		Municipio:      m,
		Fonte:          nucleoGovFonte,
		Plataforma:     nucleoGovPlataforma,
		PortalURL:      root,
		TotalRegistros: total,
		Contratos:      items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryNucleoGovLicitacoes consulta processos licitatórios no portal NúcleoGov (/api/licitacoes).
func QueryNucleoGovLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LicitacoesResult, error) {
	root := normalizeNucleoGovRoot(p.BaseURL)
	if root == "" {
		return nil, fmt.Errorf("NúcleoGov: BaseURL não informada")
	}
	hc := portalConnectorHTTPClient(client)
	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	scanLimit := limit
	if filtro != "" && scanLimit < 500 {
		scanLimit = 500
	}

	useMulti := false
	rows, total, _, err := fetchNucleoGovPaginated(scanLimit, func(offset, pageSize int) ([]map[string]any, int, int, error) {
		limitParam := fmt.Sprintf("%d,%d", offset, pageSize)
		if !useMulti {
			form := url.Values{}
			form.Set("acao", "listar")
			form.Set("limit", limitParam)
			if p.Ano > 0 {
				form.Set("ano", strconv.Itoa(p.Ano))
			}

			body, status, reqErr := doNucleoGovPostForm(ctx, hc, root+"/api/licitacoes", form)
			if reqErr != nil {
				return nil, 0, status, reqErr
			}
			if status == http.StatusOK && len(strings.TrimSpace(string(body))) > 0 {
				parsed, tot, _, decErr := decodeNucleoGovRecordsMeta(body, "licitacoes")
				if decErr != nil {
					return nil, 0, status, fmt.Errorf("erro ao decodificar NúcleoGov /api/licitacoes: %w", decErr)
				}
				if len(parsed) > 0 {
					return parsed, tot, status, nil
				}
			}

			if offset > 0 {
				return nil, 0, status, nil
			}
			useMulti = true

			extra := map[string]any{"limit": limitParam}
			if p.Ano > 0 {
				extra["ano"] = p.Ano
			}
			mRows, mTotal, mStatus, mErr := fetchNucleoGovMultiRequest(ctx, hc, root, "licitacoes/listar", extra)
			if mErr == nil && len(mRows) > 0 {
				return mRows, mTotal, mStatus, nil
			}
			if status != http.StatusOK && status != 0 && mStatus != http.StatusOK {
				return nil, 0, status, fmt.Errorf("NúcleoGov /api/licitacoes retornou HTTP %d", status)
			}
			return mRows, mTotal, mStatus, mErr
		}

		extra := map[string]any{"limit": limitParam}
		if p.Ano > 0 {
			extra["ano"] = p.Ano
		}
		return fetchNucleoGovMultiRequest(ctx, hc, root, "licitacoes/listar", extra)
	})
	if err != nil {
		return nil, err
	}

	items := make([]LicitacaoItem, 0, minInt(limit, len(rows)))
	matchedTotal := 0
	for _, r := range rows {
		objeto := mapGetStr(r, "Objeto", "objeto", "Descricao", "descricao")
		modalidade := mapGetStr(r, "Modalidade", "modalidade", "Tipo", "tipo")
		if filtro != "" && !strings.Contains(strings.ToLower(objeto+" "+modalidade), filtro) {
			continue
		}
		matchedTotal++
		if len(items) >= limit {
			continue
		}

		link := mapGetStr(r, "Link", "link", "URL", "url")
		if link == "" {
			link = root
		}

		items = append(items, LicitacaoItem{
			Numero:          mapGetStr(r, "Numero", "numero", "NumeroLicitacao", "numero_licitacao", "Processo", "processo", "NumeroProcesso", "numero_processo"),
			Ano:             int(mapGetFloat(r, "Ano", "ano", "Exercicio", "exercicio")),
			Modalidade:      modalidade,
			Situacao:        mapGetStr(r, "Situacao", "situacao", "Status", "status"),
			DataAbertura:    mapGetStr(r, "DataAbertura", "data_abertura", "Data", "data", "DataPublicacao", "data_publicacao"),
			Objeto:          objeto,
			ValorEstimado:   mapGetFloat(r, "ValorEstimado", "valor_estimado", "ValorMaximo", "valor_maximo", "Valor", "valor"),
			ValorHomologado: mapGetFloat(r, "ValorHomologado", "valor_homologado", "ValorVencedor", "valor_vencedor", "ValorAdjudicado", "valor_adjudicado"),
			Link:            link,
			Fonte:           nucleoGovFonte,
		})
	}

	if filtro != "" {
		total = matchedTotal
	} else if total == 0 {
		total = matchedTotal
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma licitação encontrada para os critérios informados."
	}

	return &LicitacoesResult{
		Municipio:      m,
		Fonte:          nucleoGovFonte,
		Plataforma:     nucleoGovPlataforma,
		PortalURL:      root,
		TotalRegistros: total,
		Licitacoes:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}
