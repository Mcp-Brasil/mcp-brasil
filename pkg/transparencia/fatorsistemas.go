package transparencia

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Fator Sistemas (plataforma = "fator_sistemas")
// Fluxo stateful com cookies de sessão:
//   1. GET /dados/<modulo>.php?id=<id>
//   2. GET /dados/<carregaModulo>.php?id=<id>&...
//   3. GET /dados/imprimir.php?id=<id>&tipo=3&tipodoc=JSON
// ---------------------------------------------------------------------------

var (
	fatorIDInHTMLRegex = regexp.MustCompile(`(?i)(?:dados/[a-z0-9_]+\.php)?\?[^"'\s<>]*\bid=([a-z0-9_-]+)`)
	fatorPMRegex       = regexp.MustCompile(`(?i)\bid=(pm_[a-z0-9_-]+)`)
)

// ExtractFatorTenantID extrai o parâmetro ?id= (ex: pm_...) de URLs ou HTML do portal Fator Sistemas.
func ExtractFatorTenantID(urlOrHTML string) string {
	trimmed := strings.TrimSpace(urlOrHTML)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "http://") || strings.HasPrefix(strings.ToLower(trimmed), "https://") {
		if u, err := url.Parse(trimmed); err == nil {
			if id := strings.TrimSpace(u.Query().Get("id")); id != "" {
				return id
			}
		}
	}
	if m := fatorPMRegex.FindStringSubmatch(trimmed); len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	if m := fatorIDInHTMLRegex.FindStringSubmatch(trimmed); len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func normalizeFatorRootURL(baseURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return ""
	}
	u, err := url.Parse(trimmed)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return trimmed
	}
	p := strings.TrimRight(u.Path, "/")
	lowP := strings.ToLower(p)
	if idx := strings.Index(lowP, "/dados"); idx >= 0 {
		p = p[:idx]
	} else {
		for _, ext := range []string{".html", ".htm", ".php", ".asp", ".aspx", ".jsp"} {
			if strings.HasSuffix(lowP, ext) {
				if slash := strings.LastIndex(p, "/"); slash >= 0 {
					p = p[:slash]
				}
				break
			}
		}
	}
	return strings.TrimRight(u.Scheme+"://"+u.Host+p, "/")
}

func newFatorSessionClient(client *http.Client) *http.Client {
	jar, _ := cookiejar.New(nil)
	base := portalConnectorHTTPClient(client)
	return &http.Client{
		Transport:     base.Transport,
		CheckRedirect: base.CheckRedirect,
		Jar:           jar,
		Timeout:       base.Timeout,
	}
}

func resolveFatorRootAndID(ctx context.Context, client *http.Client, baseURL, tenantID string) (rootURL, id string) {
	rootURL = normalizeFatorRootURL(baseURL)
	id = strings.TrimSpace(tenantID)
	if id == "" {
		id = ExtractFatorTenantID(baseURL)
	}
	if id == "" && ctx != nil && strings.TrimSpace(baseURL) != "" {
		if body, status, err := doPortalRawGet(ctx, portalConnectorHTTPClient(client), baseURL, "text/html,application/xhtml+xml,*/*"); err == nil && status == http.StatusOK {
			id = ExtractFatorTenantID(string(body))
		}
	}
	return rootURL, id
}

// ProbeFatorSistemas extrai ?id=... da URL/HTML e verifica se /dados/diarias.php?id=...
// ou /dados/carregaDiariasSub.php?id=... retorna HTTP 200.
func ProbeFatorSistemas(ctx context.Context, client *http.Client, baseURL string) (tenantID string, ok bool) {
	rootURL, id := resolveFatorRootAndID(ctx, client, baseURL, "")
	if rootURL == "" || id == "" {
		return "", false
	}

	sess := newFatorSessionClient(client)
	qID := url.QueryEscape(id)

	for _, ep := range []string{
		fmt.Sprintf("%s/dados/diarias.php?id=%s", rootURL, qID),
		fmt.Sprintf("%s/dados/carregaDiariasSub.php?id=%s", rootURL, qID),
	} {
		body, status, err := doPortalRawGet(ctx, sess, ep, "text/html,application/json,*/*")
		if err == nil && status == http.StatusOK {
			low := strings.ToLower(string(body))
			if !strings.Contains(low, "404 not found") && !strings.Contains(low, "página não encontrada") {
				return id, true
			}
		}
	}
	return "", false
}

func fetchFatorModuleJSON(
	ctx context.Context,
	client *http.Client,
	p PortalQueryParams,
	entryPage string,
	loaderPages []string,
	extraParams url.Values,
) (rows []map[string]any, total int, enabled bool, rootURL string, err error) {
	rootURL, id := resolveFatorRootAndID(ctx, client, p.BaseURL, p.TenantID)
	if rootURL == "" {
		return nil, 0, false, "", fmt.Errorf("Fator Sistemas: BaseURL não informada")
	}
	if id == "" {
		return nil, 0, false, rootURL, fmt.Errorf("Fator Sistemas: TenantID (?id=pm_...) não informado")
	}

	sess := newFatorSessionClient(client)
	qID := url.QueryEscape(id)

	// Passo 1: estabelece cookies de sessão PHP no módulo
	step1URL := fmt.Sprintf("%s/dados/%s?id=%s", rootURL, entryPage, qID)
	body1, status1, err := doPortalRawGet(ctx, sess, step1URL, "text/html,application/xhtml+xml,*/*")
	if err != nil {
		return nil, 0, false, rootURL, err
	}
	if status1 != http.StatusOK || isFatorDisabledResponse(body1) {
		return nil, 0, false, rootURL, nil
	}

	// Passo 2: aciona o script carrega*.php para popular a consulta na sessão
	loaderOK := false
	for _, loader := range loaderPages {
		params := url.Values{}
		params.Set("id", id)
		for k, vals := range extraParams {
			for _, v := range vals {
				params.Add(k, v)
			}
		}
		step2URL := fmt.Sprintf("%s/dados/%s?%s", rootURL, loader, params.Encode())
		body2, status2, err := doPortalRawGet(ctx, sess, step2URL, "text/html,application/json,*/*")
		if err != nil {
			return nil, 0, false, rootURL, err
		}
		if status2 == http.StatusOK && !isFatorDisabledResponse(body2) {
			loaderOK = true
			break
		}
	}
	if !loaderOK {
		return nil, 0, false, rootURL, nil
	}

	// Passo 3: exporta JSON da sessão via imprimir.php
	step3URL := fmt.Sprintf("%s/dados/imprimir.php?id=%s&tipo=3&tipodoc=JSON", rootURL, qID)
	body3, status3, err := doPortalRawGet(ctx, sess, step3URL, "application/json,text/plain,*/*")
	if err != nil {
		return nil, 0, false, rootURL, err
	}
	if status3 != http.StatusOK {
		return nil, 0, false, rootURL, nil
	}

	clean := strings.TrimSpace(strings.TrimPrefix(string(body3), "\xef\xbb\xbf"))
	if clean == "" || strings.HasPrefix(clean, "<") {
		return nil, 0, false, rootURL, nil
	}

	rows, total, err = decodeFlexiblePortalList([]byte(clean))
	if err != nil {
		return nil, 0, false, rootURL, nil
	}
	return rows, total, true, rootURL, nil
}

func isFatorDisabledResponse(body []byte) bool {
	low := strings.ToLower(string(body))
	return strings.Contains(low, "módulo não habilitado") ||
		strings.Contains(low, "modulo nao habilitado") ||
		strings.Contains(low, "recurso desabilitado") ||
		strings.Contains(low, "404 not found")
}

func resolveFatorYearAndDates(p PortalQueryParams) (ano, mes int, inicio, fim string) {
	now := time.Now()
	ano = p.Ano
	if ano <= 0 {
		ano = parseYearFromDateStr(p.Inicio)
	}
	if ano <= 0 {
		ano = now.Year()
	}
	mes = p.Mes
	if mes <= 0 || mes > 12 {
		if ano < now.Year() {
			mes = 12
		} else {
			mes = int(now.Month())
		}
	}
	inicio, fim = buildDDMMYYYYWindow(p, ano)
	return ano, mes, inicio, fim
}

// QueryFatorDespesas consulta despesas em portal Fator Sistemas.
func QueryFatorDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	ano, _, inicio, fim := resolveFatorYearAndDates(p)
	params := url.Values{}
	params.Set("exercicio", strconv.Itoa(ano))
	params.Set("dtInicial", inicio)
	params.Set("dtFinal", fim)

	rows, total, enabled, rootURL, err := fetchFatorModuleJSON(
		ctx, client, p, "despesa.php", []string{"carregaDespesa.php"}, params,
	)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return &DespesasResult{
			Municipio:  m,
			Fonte:      "Portal Municipal (Fator Sistemas)",
			Plataforma: "fator_sistemas",
			PortalURL:  rootURL,
			Despesas:   []DespesaItem{},
			Disponivel: false,
			Mensagem:   "Módulo de despesas não habilitado ou indisponível neste portal Fator Sistemas.",
		}, nil
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]DespesaItem, 0, minInt(limit, len(rows)))
	matchedTotal := 0
	for _, r := range rows {
		fornecedor := mapGetStr(r, "fornecedor", "Fornecedor", "credor", "Credor", "nome_fornecedor", "razao_social")
		historico := mapGetStr(r, "historico", "Historico", "descricao", "Descricao")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+historico), filtro) {
			continue
		}
		matchedTotal++
		if len(items) >= limit {
			continue
		}
		itemAno := int(mapGetFloat(r, "ano", "Ano", "exercicio", "Exercicio"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, DespesaItem{
			NumeroEmpenho:       mapGetStr(r, "numero", "Numero", "empenho", "Empenho", "numero_empenho", "num_empenho"),
			Ano:                 itemAno,
			Data:                mapGetStr(r, "data", "Data", "data_empenho", "dt_empenho"),
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(mapGetStr(r, "cpf_cnpj", "cnpj_cpf", "cpfCnpj", "documento", "cnpj", "cpf", "Documento")),
			Orgao:               mapGetStr(r, "orgao", "Orgao", "unidade_gestora"),
			Unidade:             mapGetStr(r, "unidade", "Unidade"),
			Funcao:              mapGetStr(r, "funcao", "Funcao"),
			Elemento:            mapGetStr(r, "elemento", "Elemento", "natureza"),
			Historico:           historico,
			ValorEmpenhado:      mapGetFloat(r, "valor_empenhado", "valorEmpenhado", "empenhado", "Empenhado", "valor", "Valor"),
			ValorLiquidado:      mapGetFloat(r, "valor_liquidado", "valorLiquidado", "liquidado", "Liquidado"),
			ValorPago:           mapGetFloat(r, "valor_pago", "valorPago", "pago", "Pago"),
			Fonte:               "Portal Municipal (Fator Sistemas)",
		})
	}

	if filtro != "" || total == 0 {
		total = matchedTotal
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma despesa encontrada para o período informado."
	}

	return &DespesasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Fator Sistemas)",
		Plataforma:     "fator_sistemas",
		PortalURL:      rootURL,
		TotalRegistros: total,
		Despesas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryFatorReceitas consulta receitas em portal Fator Sistemas.
func QueryFatorReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	ano, _, inicio, fim := resolveFatorYearAndDates(p)
	params := url.Values{}
	params.Set("exercicio", strconv.Itoa(ano))
	params.Set("dtInicial", inicio)
	params.Set("dtFinal", fim)

	rows, total, enabled, rootURL, err := fetchFatorModuleJSON(
		ctx, client, p, "receita.php", []string{"carregaReceita.php"}, params,
	)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return &ReceitasResult{
			Municipio:  m,
			Fonte:      "Portal Municipal (Fator Sistemas)",
			Plataforma: "fator_sistemas",
			PortalURL:  rootURL,
			Receitas:   []ReceitaItem{},
			Disponivel: false,
			Mensagem:   "Módulo de receitas não habilitado ou indisponível neste portal Fator Sistemas.",
		}, nil
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
	matchedTotal := 0
	for _, r := range rows {
		desc := mapGetStr(r, "descricao", "Descricao", "especificacao", "receita", "Receita", "nome")
		if filtro != "" && !strings.Contains(strings.ToLower(desc), filtro) {
			continue
		}
		matchedTotal++
		if len(items) >= limit {
			continue
		}
		items = append(items, ReceitaItem{
			Codigo:          mapGetStr(r, "codigo", "Codigo", "conta", "classificacao", "rubrica"),
			Descricao:       desc,
			Categoria:       mapGetStr(r, "categoria", "Categoria", "fonte", "origem"),
			Data:            mapGetStr(r, "data", "Data", "competencia"),
			ValorPrevisto:   mapGetFloat(r, "valor_previsto", "valorPrevisto", "previsto", "Previsto", "orcado", "Orcado"),
			ValorArrecadado: mapGetFloat(r, "valor_arrecadado", "valorArrecadado", "arrecadado", "Arrecadado", "realizado", "Realizado"),
			Fonte:           "Portal Municipal (Fator Sistemas)",
		})
	}

	if filtro != "" || total == 0 {
		total = matchedTotal
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma receita encontrada para o período informado."
	}

	return &ReceitasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Fator Sistemas)",
		Plataforma:     "fator_sistemas",
		PortalURL:      rootURL,
		TotalRegistros: total,
		Receitas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryFatorContratos consulta contratos em portal Fator Sistemas.
func QueryFatorContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ContratosResult, error) {
	ano, _, _, _ := resolveFatorYearAndDates(p)
	params := url.Values{}
	params.Set("exercicio", strconv.Itoa(ano))

	rows, total, enabled, rootURL, err := fetchFatorModuleJSON(
		ctx, client, p, "contrato.php", []string{"carregaContrato.php"}, params,
	)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return &ContratosResult{
			Municipio:  m,
			Fonte:      "Portal Municipal (Fator Sistemas)",
			Plataforma: "fator_sistemas",
			PortalURL:  rootURL,
			Contratos:  []ContratoItem{},
			Disponivel: false,
			Mensagem:   "Módulo de contratos não habilitado ou indisponível neste portal Fator Sistemas.",
		}, nil
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ContratoItem, 0, minInt(limit, len(rows)))
	matchedTotal := 0
	for _, r := range rows {
		fornecedor := mapGetStr(r, "fornecedor", "Fornecedor", "contratado", "Contratado", "razao_social", "credor")
		objeto := mapGetStr(r, "objeto", "Objeto", "descricao", "Descricao")
		if filtro != "" && !strings.Contains(strings.ToLower(fornecedor+" "+objeto), filtro) {
			continue
		}
		matchedTotal++
		if len(items) >= limit {
			continue
		}
		itemAno := int(mapGetFloat(r, "ano", "Ano", "exercicio", "Exercicio"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, ContratoItem{
			Numero:              mapGetStr(r, "numero", "Numero", "numero_contrato", "numeroContrato", "contrato"),
			Ano:                 itemAno,
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(mapGetStr(r, "cpf_cnpj", "cnpj_cpf", "cpfCnpj", "documento", "cnpj", "cpf", "Documento")),
			Objeto:              objeto,
			Situacao:            mapGetStr(r, "situacao", "Situacao", "status"),
			DataAssinatura:      mapGetStr(r, "data_assinatura", "dataAssinatura", "assinatura", "data"),
			VigenciaInicio:      mapGetStr(r, "vigencia_inicio", "vigenciaInicio", "data_inicio", "inicio"),
			VigenciaFim:         mapGetStr(r, "vigencia_fim", "vigenciaFim", "data_fim", "fim"),
			Valor:               mapGetFloat(r, "valor", "Valor", "valor_contrato", "valorContrato", "valor_global"),
			Fonte:               "Portal Municipal (Fator Sistemas)",
		})
	}

	if filtro != "" || total == 0 {
		total = matchedTotal
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhum contrato encontrado para o período informado."
	}

	return &ContratosResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Fator Sistemas)",
		Plataforma:     "fator_sistemas",
		PortalURL:      rootURL,
		TotalRegistros: total,
		Contratos:      items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryFatorLicitacoes consulta licitações em portal Fator Sistemas.
func QueryFatorLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LicitacoesResult, error) {
	ano, _, _, _ := resolveFatorYearAndDates(p)
	params := url.Values{}
	params.Set("exercicio", strconv.Itoa(ano))

	rows, total, enabled, rootURL, err := fetchFatorModuleJSON(
		ctx, client, p, "licitacao.php", []string{"carregaLicitacao.php"}, params,
	)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return &LicitacoesResult{
			Municipio:  m,
			Fonte:      "Portal Municipal (Fator Sistemas)",
			Plataforma: "fator_sistemas",
			PortalURL:  rootURL,
			Licitacoes: []LicitacaoItem{},
			Disponivel: false,
			Mensagem:   "Módulo de licitações não habilitado ou indisponível neste portal Fator Sistemas.",
		}, nil
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]LicitacaoItem, 0, minInt(limit, len(rows)))
	matchedTotal := 0
	for _, r := range rows {
		objeto := mapGetStr(r, "objeto", "Objeto", "descricao", "Descricao")
		modalidade := mapGetStr(r, "modalidade", "Modalidade")
		if filtro != "" && !strings.Contains(strings.ToLower(objeto+" "+modalidade), filtro) {
			continue
		}
		matchedTotal++
		if len(items) >= limit {
			continue
		}
		itemAno := int(mapGetFloat(r, "ano", "Ano", "exercicio", "Exercicio"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, LicitacaoItem{
			Numero:          mapGetStr(r, "numero", "Numero", "numero_licitacao", "numeroLicitacao", "edital", "processo"),
			Ano:             itemAno,
			Modalidade:      modalidade,
			Situacao:        mapGetStr(r, "situacao", "Situacao", "status"),
			DataAbertura:    mapGetStr(r, "data_abertura", "dataAbertura", "data", "abertura", "data_publicacao"),
			Objeto:          objeto,
			ValorEstimado:   mapGetFloat(r, "valor_estimado", "valorEstimado", "estimado"),
			ValorHomologado: mapGetFloat(r, "valor_homologado", "valorHomologado", "homologado", "valor"),
			Link:            mapGetStr(r, "link", "url", "Link"),
			Fonte:           "Portal Municipal (Fator Sistemas)",
		})
	}

	if filtro != "" || total == 0 {
		total = matchedTotal
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma licitação encontrada para o período informado."
	}

	return &LicitacoesResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Fator Sistemas)",
		Plataforma:     "fator_sistemas",
		PortalURL:      rootURL,
		TotalRegistros: total,
		Licitacoes:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryFatorServidores consulta folha de pagamento / servidores em portal Fator Sistemas.
func QueryFatorServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ServidoresResult, error) {
	ano, mes, _, _ := resolveFatorYearAndDates(p)
	params := url.Values{}
	params.Set("exercicio", strconv.Itoa(ano))
	params.Set("mes", strconv.Itoa(mes))

	rows, total, enabled, rootURL, err := fetchFatorModuleJSON(
		ctx, client, p, "folhapag.php", []string{"carregaFolhapag.php", "carregaFolha.php"}, params,
	)
	if err != nil {
		return nil, err
	}
	if !enabled {
		return &ServidoresResult{
			Municipio:  m,
			Fonte:      "Portal Municipal (Fator Sistemas)",
			Plataforma: "fator_sistemas",
			PortalURL:  rootURL,
			Servidores: []ServidorItem{},
			Disponivel: false,
			Mensagem:   "Módulo de folha de pagamento não habilitado ou indisponível neste portal Fator Sistemas.",
		}, nil
	}

	limit := normalizeLimit(p.Limit)
	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ServidorItem, 0, minInt(limit, len(rows)))
	matchedTotal := 0
	for _, r := range rows {
		nome := mapGetStr(r, "nome", "Nome", "servidor", "Servidor", "funcionario", "Funcionario")
		if filtro != "" && !strings.Contains(strings.ToLower(nome), filtro) {
			continue
		}
		matchedTotal++
		if len(items) >= limit {
			continue
		}
		comp := mapGetStr(r, "competencia", "Competencia", "mes_ano", "mesAno", "referencia")
		if comp == "" {
			comp = fmt.Sprintf("%02d/%04d", mes, ano)
		}

		items = append(items, ServidorItem{
			Nome:           nome,
			Matricula:      mapGetStr(r, "matricula", "Matricula", "registro", "codigo"),
			CPFParcial:     maskCPF(mapGetStr(r, "cpf_formatado", "cpfFormatado", "cpf_parcial"), mapGetStr(r, "cpf", "CPF", "Cpf", "num_cpf")),
			Cargo:          mapGetStr(r, "cargo", "Cargo", "descricao_cargo", "descricaoCargo"),
			Funcao:         mapGetStr(r, "funcao", "Funcao"),
			Lotacao:        mapGetStr(r, "lotacao", "Lotacao", "unidade", "Unidade", "departamento", "setor"),
			LocalTrabalho:  mapGetStr(r, "local_trabalho", "localTrabalho", "local"),
			Vinculo:        mapGetStr(r, "vinculo", "Vinculo", "regime", "Regime"),
			Situacao:       mapGetStr(r, "situacao", "Situacao", "status"),
			SalarioBase:    mapGetFloat(r, "salario_base", "salarioBase", "SalarioBase", "vencimento_base", "base"),
			SalarioBruto:   mapGetFloat(r, "salario_bruto", "salarioBruto", "SalarioBruto", "proventos", "Proventos", "bruto", "remuneracao"),
			Descontos:      mapGetFloat(r, "descontos", "Descontos", "total_descontos", "totalDescontos"),
			SalarioLiquido: mapGetFloat(r, "salario_liquido", "salarioLiquido", "SalarioLiquido", "liquido", "Liquido"),
			Competencia:    comp,
			CargaHoraria:   mapGetStr(r, "carga_horaria", "cargaHoraria"),
			DataAdmissao:   mapGetStr(r, "data_admissao", "dataAdmissao", "admissao"),
			Orgao:          mapGetStr(r, "orgao", "Orgao", "entidade"),
			Fonte:          "Portal Municipal (Fator Sistemas)",
			PortalURL:      rootURL,
		})
	}

	if filtro != "" || total == 0 {
		total = matchedTotal
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhum servidor encontrado para os filtros e período consultados."
	}

	return &ServidoresResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Fator Sistemas)",
		Plataforma:     "fator_sistemas",
		PortalURL:      rootURL,
		TotalRegistros: total,
		Servidores:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}
