package transparencia

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// PortalQueryParams define os parâmetros de consulta para conectores de portais municipais.
type PortalQueryParams struct {
	BaseURL          string
	CodAdministracao int
	TenantID         string
	NomeFiltro       string
	Inicio           string
	Fim              string
	Ano              int
	Mes              int
	Limit            int
	OrdemSalarioDesc bool
	OrdemValorDesc   bool
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 25
	}
	if limit > 1000 {
		return 1000
	}
	return limit
}

func normalizePageSize(limit, maxPageSize int) int {
	l := normalizeLimit(limit)
	if maxPageSize > 0 && l > maxPageSize {
		return maxPageSize
	}
	return l
}

func resolveDateWindows(p PortalQueryParams) [][2]string {
	if p.Inicio != "" && p.Fim != "" {
		return [][2]string{{p.Inicio, p.Fim}}
	}
	if p.Mes >= 1 && p.Mes <= 12 {
		monthRange := func(year, month int) [2]string {
			first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
			last := first.AddDate(0, 1, -1)
			return [2]string{first.Format("2006-01-02"), last.Format("2006-01-02")}
		}
		if p.Ano > 0 {
			return [][2]string{monthRange(p.Ano, p.Mes)}
		}
		nowY := time.Now().Year()
		return [][2]string{
			monthRange(nowY, p.Mes),
			monthRange(nowY-1, p.Mes),
		}
	}
	if p.Ano > 0 {
		return [][2]string{{fmt.Sprintf("%04d-01-01", p.Ano), fmt.Sprintf("%04d-12-31", p.Ano)}}
	}
	nowY := time.Now().Year()
	return [][2]string{
		{fmt.Sprintf("%04d-01-01", nowY), fmt.Sprintf("%04d-12-31", nowY)},
		{fmt.Sprintf("%04d-01-01", nowY-1), fmt.Sprintf("%04d-12-31", nowY-1)},
		{fmt.Sprintf("%04d-01-01", nowY-2), fmt.Sprintf("%04d-12-31", nowY-2)},
	}
}

// maskCPF garante conformidade LGPD retornando apenas os dígitos centrais mascarados.
func maskCPF(formatted, raw string) string {
	candidate := strings.TrimSpace(formatted)
	if candidate == "" {
		candidate = strings.TrimSpace(raw)
	}
	if candidate == "" {
		return ""
	}
	if strings.Contains(candidate, "*") {
		return candidate
	}
	digits := cleanDigits(candidate)
	if len(digits) == 11 {
		return fmt.Sprintf("***.%s.%s-**", digits[3:6], digits[6:9])
	}
	return ""
}

func doJSONGet(ctx context.Context, client *http.Client, targetURL string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
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

type quasarPage[T any] struct {
	Content       []T `json:"content"`
	TotalElements int `json:"totalElements"`
	TotalPages    int `json:"totalPages"`
}

type quasarServidorRaw struct {
	NomServidor          string  `json:"nomServidor"`
	NumMatricula         any     `json:"numMatricula"`
	CPFFormatado         string  `json:"cpfformatado"`
	NumCPF               string  `json:"numcpf"`
	NomCargo             string  `json:"nomCargo"`
	NomFuncaoExercida    string  `json:"nomFuncaoExercida"`
	NomLotacao           string  `json:"nomLotacao"`
	NomLocalTrabalho     string  `json:"nomLocalTrabalho"`
	DesTipoClassificacao string  `json:"desTipoClassificacao"`
	DesSituacao          string  `json:"desSituacao"`
	VlrSalarioBase       float64 `json:"vlrSalarioBase"`
	VlrSalarioBruto      float64 `json:"vlrSalarioBruto"`
	VlrDesconto          float64 `json:"vlrDesconto"`
	VlrSalarioLiquido    float64 `json:"vlrSalarioLiquido"`
	MesReferencia        any     `json:"mesReferencia"`
	MesReferenciaFmt     string  `json:"mesReferenciaFormatado"`
	DesCargaHoraria      string  `json:"desCargaHoraria"`
	DatAdmissao          string  `json:"datAdmissao"`
	NomAdministracao     string  `json:"nomAdministracao"`
}

type quasarDespesaRaw struct {
	NumEmpenho       string  `json:"numEmpenho"`
	NumAnoEmpenho    int     `json:"numAnoEmpenho"`
	DatMovimento     string  `json:"datMovimento"`
	DatUltimoMov     string  `json:"datUltimoMovimento"`
	NomFornecedor    string  `json:"nomFornecedor"`
	DesAgrupamento   string  `json:"desAgrupamento"`
	DesDocFornecedor string  `json:"desDocumentoFornecedor"`
	DesOrgao         string  `json:"desOrgao"`
	DesUnidade       string  `json:"desUnidade"`
	DesFuncao        string  `json:"desFuncao"`
	DesDespesaOrc    string  `json:"desDespesaOrcamentaria"`
	DesEmpenhoHist   string  `json:"desEmpenhoHistorico"`
	VlrEmpenhado     float64 `json:"vlrEmpenhado"`
	VlrLiquidado     float64 `json:"vlrLiquidado"`
	VlrPago          float64 `json:"vlrPago"`
}

type quasarReceitaRaw struct {
	DesClassificacao string  `json:"desClassificacaoReceita"`
	DesReceita       string  `json:"desReceita"`
	DesFonteReceita  string  `json:"desFonteReceita"`
	DatMovimentacao  string  `json:"datMovimentacao"`
	VlrReceita       float64 `json:"vlrReceita"`
	VlrArrecadado    float64 `json:"vlrArrecadado"`
}

type quasarLicitacaoRaw struct {
	NumeroLicitacaoEAno string  `json:"numeroLicitacaoEAno"`
	NumeroProcessoEAno  string  `json:"numeroProcessoEAno"`
	AnoProcesso         int     `json:"anoProcesso"`
	DesModalidade       string  `json:"desModalidade"`
	IndSituacaoFmt      string  `json:"indSituacaoFormatado"`
	DatAbertura         string  `json:"datAbertura"`
	TxtObjeto           string  `json:"txtObjeto"`
	VlrEstimado         float64 `json:"vlrEstimado"`
	VlrHomologado       float64 `json:"vlrHomologado"`
}

type quasarContratoRaw struct {
	NumeroContratoEAno string  `json:"numeroContratoEAno"`
	NumAnoContrato     int     `json:"numAnoContrato"`
	NomFornecedor      string  `json:"nomFornecedor"`
	DesDocFornecedor   string  `json:"desDocumentoFornecedor"`
	DesObjeto          string  `json:"desObjeto"`
	IndSituacao        string  `json:"indSituacao"`
	DatAssinatura      string  `json:"datAssinatura"`
	DatVencimento      string  `json:"datVencimento"`
	VlrContratado      float64 `json:"vlrContratado"`
}

func anyToString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", t)
	}
}

func encodeTermoBase64(q string) string {
	trimmed := strings.TrimSpace(q)
	if trimmed == "" {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(trimmed))
}

func resolveQuasarCodAdmin(ctx context.Context, client *http.Client, baseURL string, current int) int {
	if current > 0 {
		return current
	}
	ents := fetchQuasarEntidades(ctx, client, baseURL)
	if len(ents) == 0 {
		return 0
	}
	for _, e := range ents {
		up := strings.ToUpper(e.DesAdministracao)
		if strings.Contains(up, "PREF") || strings.Contains(up, "MUNIC") {
			return e.CodAdministracao
		}
	}
	return ents[0].CodAdministracao
}

// QueryQuasarServidores consulta servidores públicos em um portal Quasar (/publico/servidor).
func QueryQuasarServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ServidoresResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	limit := normalizeLimit(p.Limit)
	pageSize := normalizePageSize(limit, 200)
	codAdmin := resolveQuasarCodAdmin(ctx, client, base, p.CodAdministracao)
	termoB64 := encodeTermoBase64(p.NomeFiltro)

	for _, win := range resolveDateWindows(p) {
		var items []ServidorItem
		totalElements := 0
		prevBody := ""
		for pagina := 1; pagina <= 10; pagina++ {
			u := fmt.Sprintf(
				"%s/publico/servidor?pagina=%d&elementosPorPagina=%d&codCargo=0&indSituacao=null&datAberturaInicio=%s&datAberturaFim=%s&codAdministracao=%d&termoBase64=%s",
				base, pagina, pageSize, url.QueryEscape(win[0]), url.QueryEscape(win[1]), codAdmin, url.QueryEscape(termoB64),
			)
			body, status, err := doJSONGet(ctx, client, u)
			if err != nil {
				if pagina == 1 {
					return nil, err
				}
				break
			}
			bodyStr := strings.TrimSpace(string(body))
			if status != http.StatusOK || bodyStr == "" {
				break
			}
			if pagina > 1 && bodyStr == prevBody {
				break
			}
			prevBody = bodyStr

			var page quasarPage[quasarServidorRaw]
			if err := json.Unmarshal(body, &page); err != nil {
				if pagina == 1 {
					return nil, fmt.Errorf("erro ao decodificar /publico/servidor: %w", err)
				}
				break
			}
			if pagina == 1 {
				if len(page.Content) == 0 && page.TotalElements == 0 {
					break
				}
				totalElements = page.TotalElements
			}
			if len(page.Content) == 0 {
				break
			}
			for _, r := range page.Content {
				comp := strings.TrimSpace(r.MesReferenciaFmt)
				if comp == "" {
					comp = anyToString(r.MesReferencia)
				}
				items = append(items, ServidorItem{
					Nome:           strings.TrimSpace(r.NomServidor),
					Matricula:      anyToString(r.NumMatricula),
					CPFParcial:     maskCPF(r.CPFFormatado, r.NumCPF),
					Cargo:          strings.TrimSpace(r.NomCargo),
					Funcao:         strings.TrimSpace(r.NomFuncaoExercida),
					Lotacao:        strings.TrimSpace(r.NomLotacao),
					LocalTrabalho:  strings.TrimSpace(r.NomLocalTrabalho),
					Vinculo:        strings.TrimSpace(r.DesTipoClassificacao),
					Situacao:       strings.TrimSpace(r.DesSituacao),
					SalarioBase:    r.VlrSalarioBase,
					SalarioBruto:   r.VlrSalarioBruto,
					Descontos:      r.VlrDesconto,
					SalarioLiquido: r.VlrSalarioLiquido,
					Competencia:    comp,
					CargaHoraria:   strings.TrimSpace(r.DesCargaHoraria),
					DataAdmissao:   strings.TrimSpace(r.DatAdmissao),
					Orgao:          strings.TrimSpace(r.NomAdministracao),
					Fonte:          "Portal Municipal (Quasar /publico/servidor)",
					PortalURL:      base,
				})
				if len(items) >= limit {
					break
				}
			}
			if len(items) >= limit || len(page.Content) < pageSize || (page.TotalPages > 0 && pagina >= page.TotalPages) || (totalElements > 0 && len(items) >= totalElements) {
				break
			}
		}
		if len(items) > 0 || totalElements > 0 {
			return &ServidoresResult{
				Municipio:      m,
				Fonte:          "Portal Municipal (Quasar)",
				Plataforma:     "quasar_publico",
				PortalURL:      base,
				TotalRegistros: totalElements,
				Servidores:     items,
				Disponivel:     true,
			}, nil
		}
	}

	return &ServidoresResult{
		Municipio:  m,
		Fonte:      "Portal Municipal (Quasar)",
		Plataforma: "quasar_publico",
		PortalURL:  base,
		Servidores: []ServidorItem{},
		Disponivel: true,
		Mensagem:   "Nenhum servidor encontrado para os filtros e período consultados.",
	}, nil
}

// QueryQuasarDespesas consulta despesas detalhadas em um portal Quasar (/publico/despesaDetalhada).
func QueryQuasarDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	limit := normalizeLimit(p.Limit)
	pageSize := normalizePageSize(limit, 200)
	codAdmin := resolveQuasarCodAdmin(ctx, client, base, p.CodAdministracao)
	termoB64 := encodeTermoBase64(p.NomeFiltro)

	for _, win := range resolveDateWindows(p) {
		var items []DespesaItem
		totalElements := 0
		prevBody := ""
		for pagina := 1; pagina <= 10; pagina++ {
			u := fmt.Sprintf(
				"%s/publico/despesaDetalhada?pagina=%d&elementosPorPagina=%d&indAgrupamento=FOR&datInicio=%s&datFim=%s&desNatureza=&codAdministracao=%d&filtrarApenasCovid=false&numEmpenho=&termoBase64=%s",
				base, pagina, pageSize, url.QueryEscape(win[0]), url.QueryEscape(win[1]), codAdmin, url.QueryEscape(termoB64),
			)
			body, status, err := doJSONGet(ctx, client, u)
			if err != nil {
				if pagina == 1 {
					return nil, err
				}
				break
			}
			bodyStr := strings.TrimSpace(string(body))
			if status != http.StatusOK || bodyStr == "" {
				break
			}
			if pagina > 1 && bodyStr == prevBody {
				break
			}
			prevBody = bodyStr

			var page quasarPage[quasarDespesaRaw]
			if err := json.Unmarshal(body, &page); err != nil {
				if pagina == 1 {
					return nil, fmt.Errorf("erro ao decodificar /publico/despesaDetalhada: %w", err)
				}
				break
			}
			if pagina == 1 {
				if len(page.Content) == 0 && page.TotalElements == 0 {
					break
				}
				totalElements = page.TotalElements
			}
			if len(page.Content) == 0 {
				break
			}
			for _, r := range page.Content {
				fornecedor := strings.TrimSpace(r.NomFornecedor)
				if fornecedor == "" {
					fornecedor = strings.TrimSpace(r.DesAgrupamento)
				}
				dataStr := strings.TrimSpace(r.DatMovimento)
				if dataStr == "" {
					dataStr = strings.TrimSpace(r.DatUltimoMov)
				}
				items = append(items, DespesaItem{
					NumeroEmpenho:       strings.TrimSpace(r.NumEmpenho),
					Ano:                 r.NumAnoEmpenho,
					Data:                dataStr,
					Fornecedor:          fornecedor,
					DocumentoFornecedor: strings.TrimSpace(r.DesDocFornecedor),
					Orgao:               strings.TrimSpace(r.DesOrgao),
					Unidade:             strings.TrimSpace(r.DesUnidade),
					Funcao:              strings.TrimSpace(r.DesFuncao),
					Elemento:            strings.TrimSpace(r.DesDespesaOrc),
					Historico:           strings.TrimSpace(r.DesEmpenhoHist),
					ValorEmpenhado:      r.VlrEmpenhado,
					ValorLiquidado:      r.VlrLiquidado,
					ValorPago:           r.VlrPago,
					Fonte:               "Portal Municipal (Quasar)",
				})
				if len(items) >= limit {
					break
				}
			}
			if len(items) >= limit || len(page.Content) < pageSize || (page.TotalPages > 0 && pagina >= page.TotalPages) || (totalElements > 0 && len(items) >= totalElements) {
				break
			}
		}
		if len(items) > 0 || totalElements > 0 {
			return &DespesasResult{
				Municipio:      m,
				Fonte:          "Portal Municipal (Quasar)",
				Plataforma:     "quasar_publico",
				PortalURL:      base,
				TotalRegistros: totalElements,
				Despesas:       items,
				Disponivel:     true,
			}, nil
		}
	}

	return &DespesasResult{
		Municipio:  m,
		Fonte:      "Portal Municipal (Quasar)",
		Plataforma: "quasar_publico",
		PortalURL:  base,
		Despesas:   []DespesaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma despesa encontrada para o período informado.",
	}, nil
}

// QueryQuasarReceitas consulta receitas em um portal Quasar (/publico/receita).
func QueryQuasarReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	limit := normalizeLimit(p.Limit)
	codAdmin := resolveQuasarCodAdmin(ctx, client, base, p.CodAdministracao)

	for _, win := range resolveDateWindows(p) {
		u := fmt.Sprintf(
			"%s/publico/receita?datInicio=%s&datFim=%s&codAdministracao=%d&filtrarApenasCovid=false",
			base, url.QueryEscape(win[0]), url.QueryEscape(win[1]), codAdmin,
		)
		body, status, err := doJSONGet(ctx, client, u)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK || len(strings.TrimSpace(string(body))) == 0 {
			continue
		}
		var page quasarPage[quasarReceitaRaw]
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("erro ao decodificar /publico/receita: %w", err)
		}
		if len(page.Content) == 0 && page.TotalElements == 0 {
			continue
		}
		items := make([]ReceitaItem, 0, minInt(limit, len(page.Content)))
		for _, r := range page.Content {
			if len(items) >= limit {
				break
			}
			items = append(items, ReceitaItem{
				Codigo:          strings.TrimSpace(r.DesClassificacao),
				Descricao:       strings.TrimSpace(r.DesReceita),
				Categoria:       strings.TrimSpace(r.DesFonteReceita),
				Data:            strings.TrimSpace(r.DatMovimentacao),
				ValorPrevisto:   r.VlrReceita,
				ValorArrecadado: r.VlrArrecadado,
				Fonte:           "Portal Municipal (Quasar)",
			})
		}
		total := page.TotalElements
		if total == 0 {
			total = len(page.Content)
		}
		return &ReceitasResult{
			Municipio:      m,
			Fonte:          "Portal Municipal (Quasar)",
			Plataforma:     "quasar_publico",
			PortalURL:      base,
			TotalRegistros: total,
			Receitas:       items,
			Disponivel:     true,
		}, nil
	}

	return &ReceitasResult{
		Municipio:  m,
		Fonte:      "Portal Municipal (Quasar)",
		Plataforma: "quasar_publico",
		PortalURL:  base,
		Receitas:   []ReceitaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma receita encontrada para o período informado.",
	}, nil
}

// QueryQuasarLicitacoes consulta licitações em um portal Quasar (/publico/licitacao).
func QueryQuasarLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LicitacoesResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	limit := normalizeLimit(p.Limit)
	pageSize := normalizePageSize(limit, 200)
	codAdmin := resolveQuasarCodAdmin(ctx, client, base, p.CodAdministracao)
	termoB64 := encodeTermoBase64(p.NomeFiltro)

	for _, win := range resolveDateWindows(p) {
		var items []LicitacaoItem
		totalElements := 0
		prevBody := ""
		for pagina := 1; pagina <= 10; pagina++ {
			u := fmt.Sprintf(
				"%s/publico/licitacao?pagina=%d&elementosPorPagina=%d&datAberturaInicio=%s&datAberturaFim=%s&codAdministracao=%d&desModalidade=&filtrarApenasCovid=false&termoBase64=%s",
				base, pagina, pageSize, url.QueryEscape(win[0]), url.QueryEscape(win[1]), codAdmin, url.QueryEscape(termoB64),
			)
			body, status, err := doJSONGet(ctx, client, u)
			if err != nil {
				if pagina == 1 {
					return nil, err
				}
				break
			}
			bodyStr := strings.TrimSpace(string(body))
			if status != http.StatusOK || bodyStr == "" {
				break
			}
			if pagina > 1 && bodyStr == prevBody {
				break
			}
			prevBody = bodyStr

			var page quasarPage[quasarLicitacaoRaw]
			if err := json.Unmarshal(body, &page); err != nil {
				if pagina == 1 {
					return nil, fmt.Errorf("erro ao decodificar /publico/licitacao: %w", err)
				}
				break
			}
			if pagina == 1 {
				if len(page.Content) == 0 && page.TotalElements == 0 {
					break
				}
				totalElements = page.TotalElements
			}
			if len(page.Content) == 0 {
				break
			}
			for _, r := range page.Content {
				num := strings.TrimSpace(r.NumeroLicitacaoEAno)
				if num == "" {
					num = strings.TrimSpace(r.NumeroProcessoEAno)
				}
				items = append(items, LicitacaoItem{
					Numero:          num,
					Ano:             r.AnoProcesso,
					Modalidade:      strings.TrimSpace(r.DesModalidade),
					Situacao:        strings.TrimSpace(r.IndSituacaoFmt),
					DataAbertura:    strings.TrimSpace(r.DatAbertura),
					Objeto:          strings.TrimSpace(r.TxtObjeto),
					ValorEstimado:   r.VlrEstimado,
					ValorHomologado: r.VlrHomologado,
					Link:            base,
					Fonte:           "Portal Municipal (Quasar)",
				})
				if len(items) >= limit {
					break
				}
			}
			if len(items) >= limit || len(page.Content) < pageSize || (page.TotalPages > 0 && pagina >= page.TotalPages) || (totalElements > 0 && len(items) >= totalElements) {
				break
			}
		}
		if len(items) > 0 || totalElements > 0 {
			return &LicitacoesResult{
				Municipio:      m,
				Fonte:          "Portal Municipal (Quasar)",
				Plataforma:     "quasar_publico",
				PortalURL:      base,
				TotalRegistros: totalElements,
				Licitacoes:     items,
				Disponivel:     true,
			}, nil
		}
	}

	return &LicitacoesResult{
		Municipio:  m,
		Fonte:      "Portal Municipal (Quasar)",
		Plataforma: "quasar_publico",
		PortalURL:  base,
		Licitacoes: []LicitacaoItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma licitação encontrada para o período informado.",
	}, nil
}

// QueryQuasarContratos consulta contratos em um portal Quasar (/publico/contrato).
func QueryQuasarContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ContratosResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	limit := normalizeLimit(p.Limit)
	pageSize := normalizePageSize(limit, 200)
	codAdmin := resolveQuasarCodAdmin(ctx, client, base, p.CodAdministracao)
	termoB64 := encodeTermoBase64(p.NomeFiltro)

	for _, win := range resolveDateWindows(p) {
		var items []ContratoItem
		totalElements := 0
		prevBody := ""
		for pagina := 1; pagina <= 10; pagina++ {
			u := fmt.Sprintf(
				"%s/publico/contrato?pagina=%d&elementosPorPagina=%d&datAssinaturaInicio=%s&datAssinaturaFim=%s&codAdministracao=%d&filtrarApenasCovid=false&filtrarApenasPublicidade=false&termoBase64=%s",
				base, pagina, pageSize, url.QueryEscape(win[0]), url.QueryEscape(win[1]), codAdmin, url.QueryEscape(termoB64),
			)
			body, status, err := doJSONGet(ctx, client, u)
			if err != nil {
				if pagina == 1 {
					return nil, err
				}
				break
			}
			bodyStr := strings.TrimSpace(string(body))
			if status != http.StatusOK || bodyStr == "" {
				break
			}
			if pagina > 1 && bodyStr == prevBody {
				break
			}
			prevBody = bodyStr

			var page quasarPage[quasarContratoRaw]
			if err := json.Unmarshal(body, &page); err != nil {
				if pagina == 1 {
					return nil, fmt.Errorf("erro ao decodificar /publico/contrato: %w", err)
				}
				break
			}
			if pagina == 1 {
				if len(page.Content) == 0 && page.TotalElements == 0 {
					break
				}
				totalElements = page.TotalElements
			}
			if len(page.Content) == 0 {
				break
			}
			for _, r := range page.Content {
				items = append(items, ContratoItem{
					Numero:              strings.TrimSpace(r.NumeroContratoEAno),
					Ano:                 r.NumAnoContrato,
					Fornecedor:          strings.TrimSpace(r.NomFornecedor),
					DocumentoFornecedor: strings.TrimSpace(r.DesDocFornecedor),
					Objeto:              strings.TrimSpace(r.DesObjeto),
					Situacao:            strings.TrimSpace(r.IndSituacao),
					DataAssinatura:      strings.TrimSpace(r.DatAssinatura),
					VigenciaFim:         strings.TrimSpace(r.DatVencimento),
					Valor:               r.VlrContratado,
					Fonte:               "Portal Municipal (Quasar)",
				})
				if len(items) >= limit {
					break
				}
			}
			if len(items) >= limit || len(page.Content) < pageSize || (page.TotalPages > 0 && pagina >= page.TotalPages) || (totalElements > 0 && len(items) >= totalElements) {
				break
			}
		}
		if len(items) > 0 || totalElements > 0 {
			return &ContratosResult{
				Municipio:      m,
				Fonte:          "Portal Municipal (Quasar)",
				Plataforma:     "quasar_publico",
				PortalURL:      base,
				TotalRegistros: totalElements,
				Contratos:      items,
				Disponivel:     true,
			}, nil
		}
	}

	return &ContratosResult{
		Municipio:  m,
		Fonte:      "Portal Municipal (Quasar)",
		Plataforma: "quasar_publico",
		PortalURL:  base,
		Contratos:  []ContratoItem{},
		Disponivel: true,
		Mensagem:   "Nenhum contrato encontrado para o período informado.",
	}, nil
}

// ---------------------------------------------------------------------------
// Fiorilli SCPI JSON Adapter (/VersaoJson/*)
// ---------------------------------------------------------------------------

func decodeFlexibleRecords(body []byte) ([]map[string]any, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var rows []map[string]any
		if err := json.Unmarshal([]byte(trimmed), &rows); err != nil {
			return nil, err
		}
		return rows, nil
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil {
		return nil, err
	}
	for _, key := range []string{"Registros", "registros", "content", "dados", "items", "Servidores", "Despesas", "Receitas", "Licitacoes", "Contratos"} {
		if raw, ok := obj[key]; ok {
			if arr, ok := raw.([]any); ok {
				rows := make([]map[string]any, 0, len(arr))
				for _, el := range arr {
					if m, ok := el.(map[string]any); ok {
						rows = append(rows, m)
					}
				}
				return rows, nil
			}
		}
	}
	return nil, nil
}

func mapGetStr(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			s := strings.TrimSpace(anyToString(v))
			if s != "" {
				return s
			}
		}
	}
	return ""
}

func mapGetFloat(m map[string]any, keys ...string) float64 {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
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

// QueryFiorilliServidores consulta servidores públicos via endpoint Fiorilli /VersaoJson/Pessoal/.
func QueryFiorilliServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ServidoresResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	limit := normalizeLimit(p.Limit)
	ano := p.Ano
	if ano <= 0 {
		ano = time.Now().Year() - 1
	}
	mes := p.Mes
	if mes <= 0 {
		mes = 12
	}
	empresa := p.CodAdministracao
	if empresa <= 0 {
		empresa = 1
	}

	u := fmt.Sprintf(
		"%s/VersaoJson/Pessoal/?ConectarExercicio=%d&Listagem=Servidores&Empresa=%d&Ano=%d&MesFinalPeriodo=%d",
		base, ano, empresa, ano, mes,
	)
	body, status, err := doJSONGet(ctx, client, u)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Fiorilli /VersaoJson/Pessoal retornou HTTP %d", status)
	}
	rows, err := decodeFlexibleRecords(body)
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar Fiorilli Pessoal: %w", err)
	}

	filtro := strings.ToLower(strings.TrimSpace(p.NomeFiltro))
	items := make([]ServidorItem, 0, minInt(limit, len(rows)))
	totalMatched := 0
	for _, r := range rows {
		nome := mapGetStr(r, "Nome", "NOME", "Servidor", "nomServidor")
		if filtro != "" && !strings.Contains(strings.ToLower(nome), filtro) {
			continue
		}
		totalMatched++
		if len(items) < limit {
			items = append(items, ServidorItem{
				Nome:           nome,
				Matricula:      mapGetStr(r, "Matricula", "MATRICULA", "Registro"),
				CPFParcial:     maskCPF("", mapGetStr(r, "CPF", "Cpf")),
				Cargo:          mapGetStr(r, "Cargo", "CARGO", "DescricaoCargo"),
				Lotacao:        mapGetStr(r, "Lotacao", "LOTACAO", "Departamento", "Orgao"),
				Vinculo:        mapGetStr(r, "Vinculo", "Regime", "TipoVinculo"),
				Situacao:       mapGetStr(r, "Situacao", "SITUACAO"),
				SalarioBase:    mapGetFloat(r, "SalarioBase", "VencimentoBase", "Base"),
				SalarioBruto:   mapGetFloat(r, "SalarioBruto", "Bruto", "TotalProventos", "Proventos"),
				Descontos:      mapGetFloat(r, "Descontos", "TotalDescontos"),
				SalarioLiquido: mapGetFloat(r, "SalarioLiquido", "Liquido"),
				Competencia:    fmt.Sprintf("%02d/%04d", mes, ano),
				Fonte:          "Portal Municipal (Fiorilli SCPI)",
				PortalURL:      base,
			})
		}
	}

	return &ServidoresResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Fiorilli SCPI)",
		Plataforma:     "fiorilli_scpi",
		PortalURL:      base,
		TotalRegistros: totalMatched,
		Servidores:     items,
		Disponivel:     true,
	}, nil
}

// QueryFiorilliDespesas consulta despesas via Fiorilli /VersaoJson/DespesasGerais/.
func QueryFiorilliDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	limit := normalizeLimit(p.Limit)
	ano := p.Ano
	if ano <= 0 {
		ano = time.Now().Year() - 1
	}
	u := fmt.Sprintf("%s/VersaoJson/DespesasGerais/?ConectarExercicio=%d&Ano=%d", base, ano, ano)
	body, status, err := doJSONGet(ctx, client, u)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Fiorilli /VersaoJson/DespesasGerais retornou HTTP %d", status)
	}
	rows, err := decodeFlexibleRecords(body)
	if err != nil {
		return nil, err
	}
	items := make([]DespesaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		items = append(items, DespesaItem{
			NumeroEmpenho:       mapGetStr(r, "Empenho", "NumEmpenho", "NumeroEmpenho"),
			Ano:                 ano,
			Data:                mapGetStr(r, "Data", "DataEmpenho"),
			Fornecedor:          mapGetStr(r, "Fornecedor", "Credor", "NomeFornecedor"),
			DocumentoFornecedor: mapGetStr(r, "CNPJ", "CPF_CNPJ", "Documento"),
			Orgao:               mapGetStr(r, "Orgao", "Unidade"),
			Historico:           mapGetStr(r, "Historico", "Descricao"),
			ValorEmpenhado:      mapGetFloat(r, "Empenhado", "ValorEmpenhado", "Valor"),
			ValorLiquidado:      mapGetFloat(r, "Liquidado", "ValorLiquidado"),
			ValorPago:           mapGetFloat(r, "Pago", "ValorPago"),
			Fonte:               "Portal Municipal (Fiorilli SCPI)",
		})
		if len(items) >= limit {
			break
		}
	}
	return &DespesasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Fiorilli SCPI)",
		Plataforma:     "fiorilli_scpi",
		PortalURL:      base,
		TotalRegistros: len(rows),
		Despesas:       items,
		Disponivel:     true,
	}, nil
}

// QueryFiorilliReceitas consulta receitas via Fiorilli /VersaoJson/ReceitaOrcamentaria/.
func QueryFiorilliReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	limit := normalizeLimit(p.Limit)
	ano := p.Ano
	if ano <= 0 {
		ano = time.Now().Year() - 1
	}
	u := fmt.Sprintf("%s/VersaoJson/ReceitaOrcamentaria/?ConectarExercicio=%d&Ano=%d", base, ano, ano)
	body, status, err := doJSONGet(ctx, client, u)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Fiorilli /VersaoJson/ReceitaOrcamentaria retornou HTTP %d", status)
	}
	rows, err := decodeFlexibleRecords(body)
	if err != nil {
		return nil, err
	}
	items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
	for _, r := range rows {
		items = append(items, ReceitaItem{
			Codigo:          mapGetStr(r, "Codigo", "Conta", "Classificacao"),
			Descricao:       mapGetStr(r, "Descricao", "Especificacao", "Receita"),
			ValorPrevisto:   mapGetFloat(r, "Previsto", "ValorPrevisto", "Orcado"),
			ValorArrecadado: mapGetFloat(r, "Arrecadado", "ValorArrecadado", "Realizado"),
			Fonte:           "Portal Municipal (Fiorilli SCPI)",
		})
		if len(items) >= limit {
			break
		}
	}
	return &ReceitasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Fiorilli SCPI)",
		Plataforma:     "fiorilli_scpi",
		PortalURL:      base,
		TotalRegistros: len(rows),
		Receitas:       items,
		Disponivel:     true,
	}, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func parseBRFloat(s string) (float64, error) {
	clean := strings.TrimSpace(s)
	clean = strings.TrimPrefix(clean, "R$")
	clean = strings.TrimSpace(clean)
	if clean == "" {
		return 0, fmt.Errorf("vazio")
	}
	// Se tem vírgula como separador decimal (padrão pt-BR: "1.234,56" ou "100,00")
	if strings.Contains(clean, ",") {
		clean = strings.ReplaceAll(clean, ".", "")
		clean = strings.ReplaceAll(clean, ",", ".")
	}
	return strconv.ParseFloat(clean, 64)
}
