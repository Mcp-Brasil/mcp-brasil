package transparencia

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// CittaWeb (plataforma = "cittaweb_api")
// Base URL normalizada: https://<subdominio>.cittaweb.com.br/transparencia/api
// ---------------------------------------------------------------------------

func isLocalTestURL(rawURL string) bool {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return strings.EqualFold(u.Scheme, "http") || host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// NormalizeCittaWebAPIBase normaliza uma URL do portal CittaWeb para a raiz da API
// (<scheme>://<host>[prefix]/transparencia/api). Para URLs de teste locais (http / localhost)
// sem /citta ou /transparencia no caminho, preserva a URL base do servidor mock.
func NormalizeCittaWebAPIBase(rawURL string) string {
	raw := strings.TrimSpace(rawURL)
	if raw == "" {
		return ""
	}
	if idx := strings.Index(raw, "#"); idx >= 0 {
		raw = raw[:idx]
	}
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return ""
	}

	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	u.RawQuery = ""
	u.Fragment = ""

	p := strings.TrimRight(u.Path, "/")
	lowP := strings.ToLower(p)
	origin := u.Scheme + "://" + u.Host

	switch {
	case strings.Contains(lowP, "/transparencia/api"):
		idx := strings.Index(lowP, "/transparencia/api")
		return origin + p[:idx] + "/transparencia/api"
	case strings.Contains(lowP, "/citta"):
		idx := strings.Index(lowP, "/citta")
		return origin + p[:idx] + "/transparencia/api"
	case strings.Contains(lowP, "/transparencia"):
		idx := strings.Index(lowP, "/transparencia")
		return origin + p[:idx] + "/transparencia/api"
	case isLocalTestURL(raw):
		return strings.TrimRight(origin+p, "/")
	default:
		return strings.TrimRight(origin+p, "/") + "/transparencia/api"
	}
}

func parseCittaWebUnidades(body []byte) (ugID int, entName string, ok bool) {
	trimmed := strings.TrimSpace(string(body))
	if !strings.HasPrefix(trimmed, "[") {
		return 0, "", false
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(trimmed), &rows); err != nil || len(rows) == 0 {
		return 0, "", false
	}

	var firstID int
	var firstName string
	var hasFirst bool
	for _, r := range rows {
		chave := int(mapGetFloat(r, "chave", "id", "codigo"))
		desc := mapGetStr(r, "descricao", "nome", "entidade")
		codigo := mapGetStr(r, "codigo")
		if chave == 0 && desc == "" && codigo == "" {
			continue
		}
		if !hasFirst {
			firstID = chave
			firstName = desc
			hasFirst = true
		}
		up := strings.ToUpper(RemoveAccents(desc))
		if strings.Contains(up, "PREFEITURA") || strings.Contains(up, "MUNICIPIO") {
			return chave, desc, true
		}
	}
	if !hasFirst {
		return 0, "", false
	}
	return firstID, firstName, true
}

// ProbeCittaWeb verifica se a URL corresponde a uma API CittaWeb via GET <apiBase>/public/todasunidades.
// Retorna a URL base normalizada, o chave da unidade gestora preferida (PREFEITURA/MUNICIPIO) e seu nome.
func ProbeCittaWeb(ctx context.Context, client *http.Client, rawURL string) (apiBase string, ugID int, entName string, ok bool) {
	base := NormalizeCittaWebAPIBase(rawURL)
	if base == "" {
		return "", 0, "", false
	}
	hc := portalConnectorHTTPClient(client)

	candidates := []string{base}
	if isLocalTestURL(rawURL) && !strings.HasSuffix(strings.ToLower(base), "/transparencia/api") {
		candidates = append(candidates, base+"/transparencia/api")
	}

	for _, cand := range candidates {
		body, status, err := doJSONGet(ctx, hc, cand+"/public/todasunidades")
		if err != nil || status != http.StatusOK {
			continue
		}
		if id, name, valid := parseCittaWebUnidades(body); valid {
			return cand, id, name, true
		}
	}
	return "", 0, "", false
}

func doCittaWebGet(ctx context.Context, client *http.Client, apiBase, endpointWithQuery string) ([]byte, int, string, error) {
	hc := portalConnectorHTTPClient(client)
	body, status, err := doJSONGet(ctx, hc, apiBase+endpointWithQuery)
	if err == nil && status == http.StatusNotFound && isLocalTestURL(apiBase) && !strings.HasSuffix(strings.ToLower(apiBase), "/transparencia/api") {
		altBase := apiBase + "/transparencia/api"
		if altBody, altStatus, altErr := doJSONGet(ctx, hc, altBase+endpointWithQuery); altErr == nil && altStatus != http.StatusNotFound {
			return altBody, altStatus, altBase, nil
		}
	}
	return body, status, apiBase, err
}

func resolveCittaWebTarget(ctx context.Context, client *http.Client, p PortalQueryParams) (apiBase string, ugID int, ano int, err error) {
	apiBase = NormalizeCittaWebAPIBase(p.BaseURL)
	if apiBase == "" {
		return "", 0, 0, fmt.Errorf("CittaWeb: BaseURL não informada")
	}

	ugID = p.CodAdministracao
	if ugID <= 0 && strings.TrimSpace(p.TenantID) != "" {
		if parsed, convErr := strconv.Atoi(strings.TrimSpace(p.TenantID)); convErr == nil && parsed > 0 {
			ugID = parsed
		}
	}
	if ugID <= 0 {
		if probedBase, probedUG, _, ok := ProbeCittaWeb(ctx, client, p.BaseURL); ok && probedUG > 0 {
			apiBase = probedBase
			ugID = probedUG
		} else {
			ugID = 1
		}
	}

	ano = p.Ano
	if ano <= 0 {
		ano = parseYearFromDateStr(p.Inicio)
	}
	if ano <= 0 {
		ano = time.Now().Year()
	}
	return apiBase, ugID, ano, nil
}

// QueryCittaWebDespesas consulta despesas por elemento na API CittaWeb
// (GET /public/despesas/elementos?exercicio=<ano>&unidadeGestora=<ug>&mesInicial=1&mesFim=12).
func QueryCittaWebDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	apiBase, ugID, ano, err := resolveCittaWebTarget(ctx, client, p)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	ep := fmt.Sprintf("/public/despesas/elementos?exercicio=%d&unidadeGestora=%d&mesInicial=1&mesFim=12", ano, ugID)
	body, status, usedBase, err := doCittaWebGet(ctx, client, apiBase, ep)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("CittaWeb /public/despesas/elementos retornou HTTP %d", status)
	}

	rows, total, err := decodeFlexiblePortalList(body, "elementos", "despesas")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar CittaWeb /public/despesas/elementos: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]DespesaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		codigo := mapGetStr(r, "codigo", "codigoFormatado")
		desc := mapGetStr(r, "descricao", "nome", "elemento")
		if filtro != "" && !strings.Contains(strings.ToLower(codigo+" "+desc), filtro) {
			continue
		}
		elemento := desc
		if codigo != "" && desc != "" {
			elemento = codigo + " - " + desc
		} else if codigo != "" {
			elemento = codigo
		}

		items = append(items, DespesaItem{
			NumeroEmpenho:  codigo,
			Ano:            ano,
			Elemento:       elemento,
			Historico:      desc,
			ValorEmpenhado: mapGetFloat(r, "empenhado", "valorEmpenhado"),
			ValorLiquidado: mapGetFloat(r, "liquidado", "valorLiquidado"),
			ValorPago:      mapGetFloat(r, "pago", "valorPago"),
			Fonte:          "Portal Municipal (CittaWeb)",
		})
		if len(items) >= limit {
			break
		}
	}

	if total == 0 || filtro != "" {
		total = len(items)
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma despesa encontrada para o período informado."
	}

	return &DespesasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (CittaWeb)",
		Plataforma:     "cittaweb_api",
		PortalURL:      usedBase,
		TotalRegistros: total,
		Despesas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryCittaWebReceitas consulta o balancete de receitas na API CittaWeb
// (GET /public/receitas/balancete?page=0&size=<limit>&exercicio=<ano>&unidadeGestora=<ug>&mes=12&nivel=1).
func QueryCittaWebReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	apiBase, ugID, ano, err := resolveCittaWebTarget(ctx, client, p)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	mes := p.Mes
	if mes <= 0 || mes > 12 {
		mes = 12
	}

	ep := fmt.Sprintf(
		"/public/receitas/balancete?page=0&size=%d&exercicio=%d&unidadeGestora=%d&mes=%d&nivel=1",
		limit, ano, ugID, mes,
	)
	body, status, usedBase, err := doCittaWebGet(ctx, client, apiBase, ep)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("CittaWeb /public/receitas/balancete retornou HTTP %d", status)
	}

	rows, total, err := decodeFlexiblePortalList(body, "receitas")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar CittaWeb /public/receitas/balancete: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		codigo := mapGetStr(r, "codigoFormatado", "codigo")
		desc := mapGetStr(r, "descricao", "nome")
		if filtro != "" && !strings.Contains(strings.ToLower(codigo+" "+desc), filtro) {
			continue
		}
		previsto := mapGetFloat(r, "orcado", "valorPrevisto")
		if previsto == 0 {
			previsto = mapGetFloat(r, "atualizado")
		}

		items = append(items, ReceitaItem{
			Codigo:          codigo,
			Descricao:       desc,
			Categoria:       mapGetStr(r, "categoria", "nivel"),
			ValorPrevisto:   previsto,
			ValorArrecadado: mapGetFloat(r, "arrecadado", "valorArrecadado"),
			Fonte:           "Portal Municipal (CittaWeb)",
		})
		if len(items) >= limit {
			break
		}
	}

	if total == 0 || filtro != "" {
		total = len(items)
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma receita encontrada para o período informado."
	}

	return &ReceitasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (CittaWeb)",
		Plataforma:     "cittaweb_api",
		PortalURL:      usedBase,
		TotalRegistros: total,
		Receitas:       items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryCittaWebLicitacoes consulta licitações na API CittaWeb
// (GET /public/licitacoes/lista?page=0&size=<limit>).
func QueryCittaWebLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LicitacoesResult, error) {
	apiBase := NormalizeCittaWebAPIBase(p.BaseURL)
	if apiBase == "" {
		return nil, fmt.Errorf("CittaWeb: BaseURL não informada")
	}

	limit := normalizeLimit(p.Limit)
	ep := fmt.Sprintf("/public/licitacoes/lista?page=0&size=%d", limit)
	body, status, usedBase, err := doCittaWebGet(ctx, client, apiBase, ep)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("CittaWeb /public/licitacoes/lista retornou HTTP %d", status)
	}

	rows, total, err := decodeFlexiblePortalList(body, "licitacoes")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar CittaWeb /public/licitacoes/lista: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]LicitacaoItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		numero := mapGetStr(r, "licNumero", "numero", "edital")
		objeto := mapGetStr(r, "objeto", "descricao")
		modalidade := ""
		if modObj, ok := r["modalidade"].(map[string]any); ok && modObj != nil {
			modalidade = mapGetStr(modObj, "modNome", "modSigla", "nome", "descricao")
		} else {
			modalidade = mapGetStr(r, "modalidade", "modNome", "modSigla")
		}
		if filtro != "" && !strings.Contains(strings.ToLower(numero+" "+objeto+" "+modalidade), filtro) {
			continue
		}
		itemAno := int(mapGetFloat(r, "licAno", "ano", "exercicio"))
		if itemAno == 0 && p.Ano > 0 {
			itemAno = p.Ano
		}

		items = append(items, LicitacaoItem{
			Numero:        numero,
			Ano:           itemAno,
			Modalidade:    modalidade,
			Situacao:      mapGetStr(r, "situacao", "status"),
			DataAbertura:  mapGetStr(r, "data", "dataAbertura"),
			Objeto:        objeto,
			ValorEstimado: mapGetFloat(r, "valor", "valorEstimado"),
			Fonte:         "Portal Municipal (CittaWeb)",
		})
		if len(items) >= limit {
			break
		}
	}

	if total == 0 || filtro != "" {
		total = len(items)
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhuma licitação encontrada para os filtros consultados."
	}

	return &LicitacoesResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (CittaWeb)",
		Plataforma:     "cittaweb_api",
		PortalURL:      usedBase,
		TotalRegistros: total,
		Licitacoes:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryCittaWebContratos consulta contratos na API CittaWeb
// (GET /public/contratos/lista?unidadeGestora=<ug>&page=0&size=<limit>).
func QueryCittaWebContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ContratosResult, error) {
	apiBase, ugID, ano, err := resolveCittaWebTarget(ctx, client, p)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	ep := fmt.Sprintf("/public/contratos/lista?unidadeGestora=%d&page=0&size=%d", ugID, limit)
	body, status, usedBase, err := doCittaWebGet(ctx, client, apiBase, ep)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("CittaWeb /public/contratos/lista retornou HTTP %d", status)
	}

	rows, total, err := decodeFlexiblePortalList(body, "contratos")
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar CittaWeb /public/contratos/lista: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ContratoItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		fornecedor := mapGetStr(r, "fornecedor", "contratado", "credor", "nomeFornecedor", "razaoSocial", "nome")
		doc := mapGetStr(r, "cpfCnpj", "cnpjCpf", "documento", "cpf", "cnpj")
		if fornecedorObj, ok := r["fornecedor"].(map[string]any); ok && fornecedorObj != nil {
			if nome := mapGetStr(fornecedorObj, "nome", "razaoSocial", "nomeRazao"); nome != "" {
				fornecedor = nome
			}
			if doc == "" {
				doc = mapGetStr(fornecedorObj, "cpfCnpj", "cnpjCpf", "documento", "cpf", "cnpj")
			}
		} else if pessoaObj, ok := r["pessoa"].(map[string]any); ok && pessoaObj != nil {
			if fornecedor == "" {
				fornecedor = mapGetStr(pessoaObj, "nome", "razaoSocial")
			}
			if doc == "" {
				doc = mapGetStr(pessoaObj, "cpfCnpj", "cnpjCpf", "documento", "cpf", "cnpj")
			}
		}

		numero := mapGetStr(r, "numero", "conNumero", "contrato", "numeroContrato", "codigo")
		objeto := mapGetStr(r, "objeto", "descricao", "historico")
		if filtro != "" && !strings.Contains(strings.ToLower(numero+" "+fornecedor+" "+objeto), filtro) {
			continue
		}
		itemAno := int(mapGetFloat(r, "ano", "conAno", "exercicio"))
		if itemAno == 0 {
			itemAno = ano
		}

		items = append(items, ContratoItem{
			Numero:              numero,
			Ano:                 itemAno,
			Fornecedor:          fornecedor,
			DocumentoFornecedor: maskIfCPF(doc),
			Objeto:              objeto,
			Situacao:            mapGetStr(r, "situacao", "status"),
			DataAssinatura:      mapGetStr(r, "dataAssinatura", "data", "assinatura"),
			VigenciaInicio:      mapGetStr(r, "vigenciaInicio", "dataInicio", "inicioVigencia"),
			VigenciaFim:         mapGetStr(r, "vigenciaFim", "dataFim", "fimVigencia"),
			Valor:               mapGetFloat(r, "valor", "valorContrato", "valorGlobal", "valorTotal", "valorAtualizado"),
			Fonte:               "Portal Municipal (CittaWeb)",
		})
		if len(items) >= limit {
			break
		}
	}

	if total == 0 || filtro != "" {
		total = len(items)
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhum contrato encontrado para os filtros consultados."
	}

	return &ContratosResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (CittaWeb)",
		Plataforma:     "cittaweb_api",
		PortalURL:      usedBase,
		TotalRegistros: total,
		Contratos:      items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}

// QueryCittaWebServidores consulta servidores/folha na API CittaWeb
// (GET /public/pessoal/remuneracao?unidadeGestora=<ug> com fallback para /public/pessoal/folha/funcoes?exercicio=<ano>&unidadeGestora=<ug>&mes=<mes>).
func QueryCittaWebServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ServidoresResult, error) {
	apiBase, ugID, ano, err := resolveCittaWebTarget(ctx, client, p)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(p.Limit)
	mes := p.Mes
	if mes <= 0 || mes > 12 {
		if ano < time.Now().Year() {
			mes = 12
		} else {
			mes = int(time.Now().Month())
		}
	}

	endpoints := []string{
		fmt.Sprintf("/public/pessoal/remuneracao?unidadeGestora=%d", ugID),
		fmt.Sprintf("/public/pessoal/folha/funcoes?exercicio=%d&unidadeGestora=%d&mes=%d", ano, ugID, mes),
	}

	var rows []map[string]any
	var total int
	var lastStatus int
	var lastErr error
	usedBase := apiBase

	for _, ep := range endpoints {
		body, status, candBase, reqErr := doCittaWebGet(ctx, client, apiBase, ep)
		usedBase = candBase
		if reqErr != nil {
			lastErr = reqErr
			continue
		}
		lastStatus = status
		if status != http.StatusOK {
			continue
		}
		parsed, tot, decErr := decodeFlexiblePortalList(body, "servidores", "remuneracao", "funcoes")
		if decErr != nil {
			lastErr = fmt.Errorf("erro ao decodificar CittaWeb %s: %w", ep, decErr)
			continue
		}
		lastErr = nil
		if len(parsed) > 0 {
			rows = parsed
			total = tot
			break
		}
	}

	if len(rows) == 0 {
		if lastErr != nil {
			return nil, lastErr
		}
		if lastStatus >= 400 {
			return nil, fmt.Errorf("CittaWeb pessoal retornou HTTP %d", lastStatus)
		}
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ServidorItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		nome := mapGetStr(r, "nome", "servidor", "nomeServidor", "funcionario", "descricao", "funcao", "cargo")
		cargo := mapGetStr(r, "cargo", "nomeCargo", "descricaoCargo", "funcao", "descricao")
		if filtro != "" && !strings.Contains(strings.ToLower(nome+" "+cargo), filtro) {
			continue
		}
		comp := mapGetStr(r, "competencia", "mesAno", "referencia")
		if comp == "" {
			comp = fmt.Sprintf("%02d/%04d", mes, ano)
		}

		items = append(items, ServidorItem{
			Nome:           nome,
			Matricula:      mapGetStr(r, "matricula", "registro", "codigo"),
			CPFParcial:     maskCPF(mapGetStr(r, "cpfFormatado", "cpfParcial"), mapGetStr(r, "cpf", "CPF", "documento")),
			Cargo:          cargo,
			Funcao:         mapGetStr(r, "funcao", "descricaoFuncao"),
			Lotacao:        mapGetStr(r, "lotacao", "unidade", "departamento", "setor", "orgao"),
			Vinculo:        mapGetStr(r, "vinculo", "regime", "tipoVinculo"),
			Situacao:       mapGetStr(r, "situacao", "status"),
			SalarioBase:    mapGetFloat(r, "salarioBase", "vencimentoBase", "base"),
			SalarioBruto:   mapGetFloat(r, "salarioBruto", "proventos", "totalProventos", "bruto", "remuneracao", "valorBruto"),
			Descontos:      mapGetFloat(r, "descontos", "totalDescontos", "valorDescontos"),
			SalarioLiquido: mapGetFloat(r, "salarioLiquido", "liquido", "valorLiquido"),
			Competencia:    comp,
			Orgao:          mapGetStr(r, "orgao", "unidadeGestora", "entidade"),
			Fonte:          "Portal Municipal (CittaWeb)",
			PortalURL:      usedBase,
		})
		if len(items) >= limit {
			break
		}
	}

	if total == 0 || filtro != "" {
		total = len(items)
	}
	msg := ""
	if len(items) == 0 {
		msg = "Nenhum servidor encontrado para os filtros consultados."
	}

	return &ServidoresResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (CittaWeb)",
		Plataforma:     "cittaweb_api",
		PortalURL:      usedBase,
		TotalRegistros: total,
		Servidores:     items,
		Disponivel:     true,
		Mensagem:       msg,
	}, nil
}
