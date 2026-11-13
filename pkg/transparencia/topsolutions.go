package transparencia

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// TopSolutions RN (plataforma = "topsolutions_rn")
// Padrão: https://<mun>.transparencia.topsolutionsrn.com.br
// API:    https://<mun>.apitransparencia.topsolutionsrn.com.br
// ---------------------------------------------------------------------------

// ResolveTopSolutionsAPIBase deriva a URL base da API REST a partir da URL do frontend Vue.
func ResolveTopSolutionsAPIBase(rawURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(rawURL), "/")
	if trimmed == "" {
		return ""
	}
	u, err := url.Parse(trimmed)
	if err != nil || u.Host == "" {
		return ""
	}
	host := u.Hostname()
	lowHost := strings.ToLower(host)
	// Em ambiente de teste local (http/localhost/127.0.0.1), usa diretamente o endpoint mock.
	if strings.EqualFold(u.Scheme, "http") || lowHost == "localhost" || lowHost == "127.0.0.1" || lowHost == "::1" {
		return u.Scheme + "://" + u.Host
	}
	parts := strings.Split(host, ".")
	if len(parts) >= 2 {
		p1 := strings.ToLower(parts[1])
		if p1 == "transparencia" {
			parts[1] = "apitransparencia"
		}
	}
	port := ""
	if p := u.Port(); p != "" && p != "443" && p != "80" {
		port = ":" + p
	}
	return "https://" + strings.Join(parts, ".") + port
}

func decodeTopSolutionsData(body []byte) ([]map[string]any, bool, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || (!strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[")) {
		return nil, false, fmt.Errorf("resposta TopSolutions não é JSON")
	}
	if strings.HasPrefix(trimmed, "[") {
		var rows []map[string]any
		if err := json.Unmarshal([]byte(trimmed), &rows); err != nil {
			return nil, false, err
		}
		return rows, true, nil
	}
	var env struct {
		Data     json.RawMessage `json:"data"`
		OK       *bool           `json:"ok"`
		Metadata map[string]any  `json:"metadata"`
	}
	if err := json.Unmarshal([]byte(trimmed), &env); err != nil {
		return nil, false, err
	}
	if env.OK != nil && !*env.OK {
		return nil, false, nil
	}
	dataTrim := strings.TrimSpace(string(env.Data))
	if dataTrim == "" || dataTrim == "null" || dataTrim == "{}" {
		return nil, true, nil
	}
	var rows []map[string]any
	if strings.HasPrefix(dataTrim, "[") {
		if err := json.Unmarshal([]byte(dataTrim), &rows); err != nil {
			return nil, false, err
		}
		return rows, true, nil
	}
	return nil, true, nil
}

// ProbeTopSolutions verifica se uma URL pertence à plataforma TopSolutions RN (API REST pública).
func ProbeTopSolutions(ctx context.Context, client *http.Client, rawURL string) (apiBase string, ok bool) {
	apiBase = ResolveTopSolutionsAPIBase(rawURL)
	if apiBase == "" {
		return "", false
	}
	hc := portalConnectorHTTPClient(client)
	ano := time.Now().Year() - 1
	probes := []string{
		fmt.Sprintf("%s/receitaprevistaarrecadada/receitaprevistaarrecadadaasync?classificacaoPor=origem&numExercicio=%d&mesIni=1&mesFim=12", apiBase, ano),
		fmt.Sprintf("%s/contrato/contratoasync", apiBase),
	}
	for _, target := range probes {
		body, status, err := doPortalRawGet(ctx, hc, target, "application/json")
		if err != nil || status != http.StatusOK {
			continue
		}
		rows, valid, err := decodeTopSolutionsData(body)
		if err == nil && valid && len(rows) > 0 {
			return apiBase, true
		}
	}
	return "", false
}

// QueryTopSolutionsServidores consulta servidores e remunerações via API TopSolutions RN.
func QueryTopSolutionsServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ServidoresResult, error) {
	apiBase := ResolveTopSolutionsAPIBase(p.BaseURL)
	if apiBase == "" {
		return nil, fmt.Errorf("TopSolutions RN: BaseURL não informada")
	}
	hc := portalConnectorHTTPClient(client)
	limit := normalizeLimit(p.Limit)
	ano := p.Ano
	if ano <= 0 {
		ano = time.Now().Year()
	}

	type compCand struct{ a, m int }
	var cands []compCand
	if p.Mes > 0 {
		cands = append(cands, compCand{ano, p.Mes})
	} else {
		now := time.Now()
		startMes := 12
		if ano == now.Year() {
			startMes = int(now.Month())
		}
		for mm := startMes; mm >= 1 && len(cands) < 6; mm-- {
			cands = append(cands, compCand{ano, mm})
		}
		if p.Ano <= 0 {
			for mm := 12; mm >= 9; mm-- {
				cands = append(cands, compCand{ano - 1, mm})
			}
		}
	}

	var rows []map[string]any
	var usedComp string
	for _, c := range cands {
		target := fmt.Sprintf("%s/Servidor/ServidorPorMesAnoAsync?numMes=%02d&numAno=%d", apiBase, c.m, c.a)
		body, status, err := doPortalRawGet(ctx, hc, target, "application/json")
		if err != nil || status != http.StatusOK {
			continue
		}
		r, valid, err := decodeTopSolutionsData(body)
		if err == nil && valid && len(r) > 0 {
			rows = r
			usedComp = fmt.Sprintf("%02d/%d", c.m, c.a)
			break
		}
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ServidorItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		nome := mapGetStr(r, "nome", "txtNome", "nomeServidor")
		if nome == "" {
			continue
		}
		cargo := mapGetStr(r, "cargo", "cargoFuncao", "txtCargo")
		orgao := mapGetStr(r, "orgao", "lotacao", "txtOrgao")
		if filtro != "" && !strings.Contains(strings.ToLower(nome+" "+cargo+" "+orgao), filtro) {
			continue
		}
		bruto := mapGetFloat(r, "vlrRemuneracaoBruta", "salarioBruto")
		desc := mapGetFloat(r, "vlrDescontosObrigatorios", "descontos") + mapGetFloat(r, "vlrDescontoOutros")
		liq := mapGetFloat(r, "vlrRemuAposDescObrig", "salarioLiquido")
		comp := usedComp
		if rawDt := mapGetStr(r, "dtMesAno"); len(rawDt) >= 7 {
			comp = rawDt[:7]
		}
		items = append(items, ServidorItem{
			Nome:           nome,
			Matricula:      mapGetStr(r, "numMatricula", "matricula"),
			CPFParcial:     maskIfCPF(mapGetStr(r, "cpf", "cpfServidor")),
			Cargo:          cargo,
			Funcao:         mapGetStr(r, "funcao"),
			Lotacao:        orgao,
			Orgao:          orgao,
			Vinculo:        mapGetStr(r, "vinculo"),
			CargaHoraria:   mapGetStr(r, "cargaHoraria"),
			DataAdmissao:   mapGetStr(r, "dataExercicio", "dataAdmissao"),
			SalarioBruto:   bruto,
			Descontos:      desc,
			SalarioLiquido: liq,
			Competencia:    comp,
			Fonte:          "Portal Municipal (TopSolutions RN)",
			PortalURL:      p.BaseURL,
		})
		if len(items) >= limit {
			break
		}
	}

	msg := ""
	if len(items) == 0 {
		msg = "Nenhum servidor encontrado na folha publicada para o filtro/período informado."
	}
	return &ServidoresResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (TopSolutions RN)",
		Plataforma:     "topsolutions_rn",
		PortalURL:      p.BaseURL,
		TotalRegistros: len(items),
		Servidores:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryTopSolutionsDespesas consulta despesas orçamentárias por órgão/unidade via TopSolutions RN.
func QueryTopSolutionsDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	apiBase := ResolveTopSolutionsAPIBase(p.BaseURL)
	if apiBase == "" {
		return nil, fmt.Errorf("TopSolutions RN: BaseURL não informada")
	}
	hc := portalConnectorHTTPClient(client)
	limit := normalizeLimit(p.Limit)
	ano := p.Ano
	if ano <= 0 {
		ano = time.Now().Year() - 1
	}

	dtIni := fmt.Sprintf("01/01/%d", ano)
	dtFim := fmt.Sprintf("31/12/%d", ano)
	if p.Inicio != "" {
		if t, err := time.Parse("2006-01-02", p.Inicio); err == nil {
			dtIni = t.Format("02/01/2006")
		}
	}
	if p.Fim != "" {
		if t, err := time.Parse("2006-01-02", p.Fim); err == nil {
			dtFim = t.Format("02/01/2006")
		}
	}

	target := fmt.Sprintf("%s/despesa/despesaporclassificacaoasync?strClassificarPor=orgao&dtIni=%s&dtFim=%s",
		apiBase, url.QueryEscape(dtIni), url.QueryEscape(dtFim))
	body, status, err := doPortalRawGet(ctx, hc, target, "application/json")
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("TopSolutions RN despesaporclassificacaoasync retornou HTTP %d", status)
	}
	rows, _, err := decodeTopSolutionsData(body)
	if err != nil {
		return nil, err
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]DespesaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		ex := int(mapGetFloat(r, "exercicio"))
		emp := mapGetFloat(r, "vlrEmpenhado")
		liq := mapGetFloat(r, "vlrLiquidado")
		pago := mapGetFloat(r, "vlrPago")
		if ex == 1900 && emp == 0 && liq == 0 && pago == 0 {
			continue
		}
		if ex == 0 {
			ex = ano
		}
		unidade := mapGetStr(r, "txtDescricaoUnidade", "txtDescricaoElementoDespesa", "txtDescricaoFuncao")
		if filtro != "" && !strings.Contains(strings.ToLower(unidade), filtro) {
			continue
		}
		items = append(items, DespesaItem{
			Ano:            ex,
			Orgao:          mapGetStr(r, "codOrgao"),
			Unidade:        unidade,
			Elemento:       mapGetStr(r, "codUnidade"),
			ValorEmpenhado: emp,
			ValorLiquidado: liq,
			ValorPago:      pago,
			Fonte:          "Portal Municipal (TopSolutions RN)",
		})
		if len(items) >= limit {
			break
		}
	}

	return &DespesasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (TopSolutions RN)",
		Plataforma:     "topsolutions_rn",
		PortalURL:      p.BaseURL,
		TotalRegistros: len(items),
		Despesas:       items,
		Disponivel:     true,
	}, nil
}

// QueryTopSolutionsReceitas consulta receitas previstas e arrecadadas via TopSolutions RN.
func QueryTopSolutionsReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	apiBase := ResolveTopSolutionsAPIBase(p.BaseURL)
	if apiBase == "" {
		return nil, fmt.Errorf("TopSolutions RN: BaseURL não informada")
	}
	hc := portalConnectorHTTPClient(client)
	limit := normalizeLimit(p.Limit)
	ano := p.Ano
	if ano <= 0 {
		ano = time.Now().Year() - 1
	}

	target := fmt.Sprintf("%s/receitaprevistaarrecadada/receitaprevistaarrecadadaasync?classificacaoPor=origem&numExercicio=%d&mesIni=1&mesFim=12", apiBase, ano)
	body, status, err := doPortalRawGet(ctx, hc, target, "application/json")
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("TopSolutions RN receita retornou HTTP %d", status)
	}
	rows, _, err := decodeTopSolutionsData(body)
	if err != nil {
		return nil, err
	}

	items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		cod := mapGetStr(r, "codOrigem", "codCategoria")
		desc := mapGetStr(r, "txtDescricaoOrigem", "txtDescricaoCategoria", "txtDescricaoNaturezaReceita")
		if desc == "" {
			continue
		}
		cat := mapGetStr(r, "txtDescricaoCategoria")
		if cat == "" {
			cat = mapGetStr(r, "txtDescricaoFonteRecurso", "txtDescricaoUnidade")
		}
		items = append(items, ReceitaItem{
			Codigo:          cod,
			Descricao:       desc,
			Categoria:       cat,
			ValorPrevisto:   mapGetFloat(r, "vlrPrevisaoAtualizado"),
			ValorArrecadado: mapGetFloat(r, "vlrArrecadacao"),
			Fonte:           "Portal Municipal (TopSolutions RN)",
		})
		if len(items) >= limit {
			break
		}
	}

	return &ReceitasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (TopSolutions RN)",
		Plataforma:     "topsolutions_rn",
		PortalURL:      p.BaseURL,
		TotalRegistros: len(items),
		Receitas:       items,
		Disponivel:     true,
	}, nil
}

// QueryTopSolutionsContratos consulta contratos via API TopSolutions RN.
func QueryTopSolutionsContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ContratosResult, error) {
	apiBase := ResolveTopSolutionsAPIBase(p.BaseURL)
	if apiBase == "" {
		return nil, fmt.Errorf("TopSolutions RN: BaseURL não informada")
	}
	hc := portalConnectorHTTPClient(client)
	limit := normalizeLimit(p.Limit)

	target := apiBase + "/contrato/contratoasync"
	body, status, err := doPortalRawGet(ctx, hc, target, "application/json")
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("TopSolutions RN contratoasync retornou HTTP %d", status)
	}
	rows, _, err := decodeTopSolutionsData(body)
	if err != nil {
		return nil, err
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ContratoItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		fornecedor := mapGetStr(r, "txtNomeRazaoContratada", "txtNomeRazaoContratante", "fornecedor")
		objeto := mapGetStr(r, "txtObjeto", "objeto")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+objeto), filtro) {
			continue
		}
		ex := int(mapGetFloat(r, "numExercicioFinanc", "ano"))
		if p.Ano > 0 && ex > 0 && ex != p.Ano {
			continue
		}
		items = append(items, ContratoItem{
			Numero:              mapGetStr(r, "numContrato", "numero"),
			Ano:                 ex,
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(mapGetStr(r, "txtCpfCnpjContratada", "txtCpfCnpjContratante")),
			Objeto:              objeto,
			Situacao:            mapGetStr(r, "txtEvento", "txtModalidade"),
			DataAssinatura:      mapGetStr(r, "dtAssinatura", "dtPublicacao"),
			VigenciaInicio:      mapGetStr(r, "dtInicioVigencia"),
			VigenciaFim:         mapGetStr(r, "dtFimVigencia"),
			Valor:               mapGetFloat(r, "vlrContrato", "vlrTotal", "vlrValor"),
			Fonte:               "Portal Municipal (TopSolutions RN)",
		})
		if len(items) >= limit {
			break
		}
	}

	return &ContratosResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (TopSolutions RN)",
		Plataforma:     "topsolutions_rn",
		PortalURL:      p.BaseURL,
		TotalRegistros: len(items),
		Contratos:      items,
		Disponivel:     true,
	}, nil
}

// QueryTopSolutionsLicitacoes consulta licitações via API TopSolutions RN.
func QueryTopSolutionsLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LicitacoesResult, error) {
	apiBase := ResolveTopSolutionsAPIBase(p.BaseURL)
	if apiBase == "" {
		return nil, fmt.Errorf("TopSolutions RN: BaseURL não informada")
	}
	hc := portalConnectorHTTPClient(client)
	limit := normalizeLimit(p.Limit)

	target := apiBase + "/licitacao/LicitacaoAsync"
	body, status, err := doPortalRawGet(ctx, hc, target, "application/json")
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("TopSolutions RN LicitacaoAsync retornou HTTP %d", status)
	}
	rows, _, err := decodeTopSolutionsData(body)
	if err != nil {
		return nil, err
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]LicitacaoItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		objeto := mapGetStr(r, "txtObjeto", "objeto")
		mod := mapGetStr(r, "txtModalidade", "txtModalidadeLicit")
		if filtro != "" && !strings.Contains(strings.ToLower(objeto+" "+mod), filtro) {
			continue
		}
		num := mapGetStr(r, "numLicitacao", "numProcesso", "numCertame")
		items = append(items, LicitacaoItem{
			Numero:          num,
			Ano:             int(mapGetFloat(r, "numExercicioFinanc", "ano")),
			Modalidade:      mod,
			Situacao:        mapGetStr(r, "txtSituacao", "situacao"),
			DataAbertura:    mapGetStr(r, "dtAbertura", "dataAbertura"),
			Objeto:          objeto,
			ValorEstimado:   mapGetFloat(r, "vlrEstimado", "valorEstimado"),
			ValorHomologado: mapGetFloat(r, "vlrHomologado", "valorHomologado"),
			Fonte:           "Portal Municipal (TopSolutions RN)",
		})
		if len(items) >= limit {
			break
		}
	}

	return &LicitacoesResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (TopSolutions RN)",
		Plataforma:     "topsolutions_rn",
		PortalURL:      p.BaseURL,
		TotalRegistros: len(items),
		Licitacoes:     items,
		Disponivel:     true,
	}, nil
}
