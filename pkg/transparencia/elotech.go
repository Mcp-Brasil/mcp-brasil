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
// Elotech Gestão Pública (Oxy / portaltransparencia-api) Connector
// OpenAPI 3.0: /v3/api-docs
// ---------------------------------------------------------------------------

type elotechPage[T any] struct {
	Content       []T `json:"content"`
	TotalElements int `json:"totalElements"`
	TotalPages    int `json:"totalPages"`
}

type elotechServidorFinanceiroDTO struct {
	Entidade          int64   `json:"entidade"`
	Matricula         int64   `json:"matricula"`
	AnoCompetencia    int64   `json:"anoCompetencia"`
	MesCompetencia    int64   `json:"mesCompetencia"`
	SalarioBase       float64 `json:"salarioBase"`
	Vencimentos       float64 `json:"vencimentos"`
	Descontos         float64 `json:"descontos"`
	OutrosProventos   float64 `json:"outrosProventos"`
	DescricaoTipoFolha string `json:"descricaoTipoFolha"`
}

type elotechServidorDTO struct {
	Entidade          int64                          `json:"entidade"`
	Exercicio         int64                          `json:"exercicio"`
	Matricula         int64                          `json:"matricula"`
	Nome              string                         `json:"nome"`
	CPF               string                         `json:"cpf"`
	CPFFormatado      string                         `json:"cpfFormatado"`
	Cargo             int64                          `json:"cargo"`
	HorasSemanais     float64                        `json:"horasSemanais"`
	DescricaoCargo    string                         `json:"descricaoCargo"`
	CodigoLotacao     string                         `json:"codigoLotacao"`
	DescricaoLotacao  string                         `json:"descricaoLotacao"`
	Vinculo           string                         `json:"vinculo"`
	DescricaoNatureza string                         `json:"descricaoNatureza"`
	Situacao          string                         `json:"situacao"`
	DataAdmissao      string                         `json:"dataAdmissao"`
	LocalTrabalho     string                         `json:"localTrabalho"`
	SalarioBase       float64                        `json:"salarioBase"`
	Remuneracao       float64                        `json:"remuneracao"`
	DadosFinanceiros  []elotechServidorFinanceiroDTO `json:"dadosFinanceiros"`
}

type elotechEmpenhoDTO struct {
	ID               int64   `json:"id"`
	Entidade         int64   `json:"entidade"`
	Empenho          int64   `json:"empenho"`
	Exercicio        int     `json:"exercicio"`
	EmpenhoExercicio string  `json:"empenhoExercicio"`
	Data             string  `json:"data"`
	Fornecedor       int64   `json:"fornecedor"`
	Nome             string  `json:"nome"`
	Historico        string  `json:"historico"`
	ValorEmpenhado   float64 `json:"valorEmpenhado"`
	ValorAnulado     float64 `json:"valorAnulado"`
	ValorLiquidado   float64 `json:"valorLiquidado"`
	ValorRetido      float64 `json:"valorRetido"`
	ValorPago        float64 `json:"valorPago"`
	ValorAPagar      float64 `json:"valorAPagar"`
	FonteRecurso     string  `json:"fonteRecurso"`
}

type elotechReceitaDTO struct {
	Entidade              string  `json:"entidade"`
	Receita               string  `json:"receita"`
	Descricao             string  `json:"descricao"`
	FonteRecurso          string  `json:"fonteRecurso"`
	DescricaoFonte        string  `json:"descricaoFonte"`
	Mes                   int     `json:"mes"`
	ValorOrcado           float64 `json:"valorOrcado"`
	ValorPrevisto         float64 `json:"valorPrevisto"`
	ValorAtualizado       float64 `json:"valorAtualizado"`
	ValorArrecadado       float64 `json:"valorArrecadado"`
	ValorOrcadoAtualizado float64 `json:"valorOrcadoAtualizado"`
	Data                  string  `json:"data"`
}

type elotechLicitacaoDTO struct {
	Entidade         int64   `json:"entidade"`
	NomeEntidade     string  `json:"nomeEntidade"`
	Exercicio        int     `json:"exercicio"`
	Licitacao        int64   `json:"licitacao"`
	DisplayLicitacao string  `json:"displayLicitacao"`
	Descricao        string  `json:"descricao"`
	NumeroLicitacao  int64   `json:"numeroLicitacao"`
	DataPublicacao   string  `json:"dataPublicacao"`
	DataAbertura     string  `json:"dataAbertura"`
	TipoCancelamento string  `json:"tipoCancelamento"`
	Objeto           string  `json:"objeto"`
	ValorMaximo      float64 `json:"valorMaximo"`
	TotalVencido     float64 `json:"totalVencido"`
}

func doElotechGet(ctx context.Context, client *http.Client, targetURL string, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
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

func resolveElotechYears(p PortalQueryParams) []int {
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

func resolveElotechEntidade(p PortalQueryParams) int {
	if p.CodAdministracao > 0 {
		return p.CodAdministracao
	}
	return 1
}

// ProbeElotech verifica se a URL base corresponde a uma instância da API Elotech Gestão Pública (portaltransparencia-api).
func ProbeElotech(ctx context.Context, client *http.Client, baseURL string) bool {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return false
	}

	// 1. Verifica /v3/api-docs (OpenAPI 3.0 oficial da Elotech)
	if body, status, err := doElotechGet(ctx, client, base+"/v3/api-docs", nil); err == nil && status == http.StatusOK {
		low := strings.ToLower(string(body))
		if strings.Contains(low, "elotech") ||
			strings.Contains(low, "portaltransparencia-api") ||
			(strings.Contains(low, "/api/servidores") && strings.Contains(low, "/empenhos/lista")) {
			return true
		}
	}

	// 2. Verifica /swagger-ui.html
	if body, status, err := doElotechGet(ctx, client, base+"/swagger-ui.html", nil); err == nil && status == http.StatusOK {
		low := strings.ToLower(string(body))
		if strings.Contains(low, "elotech") || strings.Contains(low, "portaltransparencia") {
			return true
		}
	}

	// 3. Fallback: testa /api/servidores com headers obrigatórios
	headers := map[string]string{
		"entidade":  "1",
		"exercicio": strconv.Itoa(time.Now().Year()),
	}
	if body, status, err := doElotechGet(ctx, client, base+"/api/servidores?page=0&size=1&entidade=1", headers); err == nil && status == http.StatusOK {
		low := strings.ToLower(string(body))
		if strings.Contains(low, `"content"`) && (strings.Contains(low, `"totalelements"`) || strings.Contains(low, `"matricula"`)) {
			return true
		}
	}

	return false
}

// QueryElotechServidores consulta servidores públicos no endpoint Elotech /api/servidores.
func QueryElotechServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ServidoresResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	limit := normalizeLimit(p.Limit)
	entidade := resolveElotechEntidade(p)

	for _, ano := range resolveElotechYears(p) {
		q := url.Values{}
		q.Set("page", "0")
		q.Set("size", strconv.Itoa(limit))
		q.Set("entidade", strconv.Itoa(entidade))
		q.Set("exercicio", strconv.Itoa(ano))
		q.Set("anoCompetencia", strconv.Itoa(ano))
		if p.Mes > 0 {
			q.Set("mesCompetencia", strconv.Itoa(p.Mes))
		}
		if strings.TrimSpace(p.NomeFiltro) != "" {
			q.Set("nomeServidor", strings.TrimSpace(p.NomeFiltro))
		}

		headers := map[string]string{
			"entidade":  strconv.Itoa(entidade),
			"exercicio": strconv.Itoa(ano),
		}
		u := fmt.Sprintf("%s/api/servidores?%s", base, q.Encode())
		body, status, err := doElotechGet(ctx, client, u, headers)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK || len(strings.TrimSpace(string(body))) == 0 {
			continue
		}

		var page elotechPage[elotechServidorDTO]
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("erro ao decodificar Elotech /api/servidores: %w", err)
		}
		if len(page.Content) == 0 && page.TotalElements == 0 {
			continue
		}

		items := make([]ServidorItem, 0, len(page.Content))
		for _, r := range page.Content {
			cargo := strings.TrimSpace(r.DescricaoCargo)
			if cargo == "" && r.Cargo > 0 {
				cargo = strconv.FormatInt(r.Cargo, 10)
			}
			lotacao := strings.TrimSpace(r.DescricaoLotacao)
			if lotacao == "" {
				lotacao = strings.TrimSpace(r.CodigoLotacao)
			}
			vinculo := strings.TrimSpace(r.Vinculo)
			if vinculo == "" {
				vinculo = strings.TrimSpace(r.DescricaoNatureza)
			}
			cargaHoraria := ""
			if r.HorasSemanais > 0 {
				cargaHoraria = strconv.FormatFloat(r.HorasSemanais, 'f', -1, 64) + "h"
			}

			salBase := r.SalarioBase
			salBruto := r.Remuneracao
			var descontos, salLiquido float64
			comp := strconv.Itoa(ano)
			if p.Mes > 0 {
				comp = fmt.Sprintf("%02d/%04d", p.Mes, ano)
			}

			if len(r.DadosFinanceiros) > 0 {
				df := r.DadosFinanceiros[len(r.DadosFinanceiros)-1]
				if df.SalarioBase > 0 {
					salBase = df.SalarioBase
				}
				if df.Vencimentos > 0 {
					salBruto = df.Vencimentos
				}
				if df.Descontos > 0 {
					descontos = df.Descontos
				}
				if salBruto > 0 {
					salLiquido = salBruto - descontos
				}
				if df.MesCompetencia > 0 && df.AnoCompetencia > 0 {
					comp = fmt.Sprintf("%02d/%04d", df.MesCompetencia, df.AnoCompetencia)
				}
			}

			items = append(items, ServidorItem{
				Nome:           strings.TrimSpace(r.Nome),
				Matricula:      strconv.FormatInt(r.Matricula, 10),
				CPFParcial:     maskCPF(r.CPFFormatado, r.CPF),
				Cargo:          cargo,
				Lotacao:        lotacao,
				LocalTrabalho:  strings.TrimSpace(r.LocalTrabalho),
				Vinculo:        vinculo,
				Situacao:       strings.TrimSpace(r.Situacao),
				SalarioBase:    salBase,
				SalarioBruto:   salBruto,
				Descontos:      descontos,
				SalarioLiquido: salLiquido,
				Competencia:    comp,
				CargaHoraria:   cargaHoraria,
				DataAdmissao:   strings.TrimSpace(r.DataAdmissao),
				Fonte:          "Portal Municipal (Elotech /api/servidores)",
				PortalURL:      base,
			})
		}

		return &ServidoresResult{
			Municipio:      m,
			Fonte:          "Portal Municipal (Elotech)",
			Plataforma:     "elotech",
			PortalURL:      base,
			TotalRegistros: page.TotalElements,
			Servidores:     items,
			Disponivel:     true,
		}, nil
	}

	return &ServidoresResult{
		Municipio:  m,
		Fonte:      "Portal Municipal (Elotech)",
		Plataforma: "elotech",
		PortalURL:  base,
		Servidores: []ServidorItem{},
		Disponivel: true,
		Mensagem:   "Nenhum servidor encontrado para os filtros e período consultados.",
	}, nil
}

// QueryElotechDespesas consulta empenhos/despesas no endpoint Elotech /empenhos/lista.
func QueryElotechDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	limit := normalizeLimit(p.Limit)
	entidade := resolveElotechEntidade(p)

	for _, ano := range resolveElotechYears(p) {
		rsql := fmt.Sprintf("id.exercicio=='%d' and id.entidade=='%d'", ano, entidade)
		if p.Inicio != "" {
			rsql += fmt.Sprintf(" and data>='%s'", p.Inicio)
		}
		if p.Fim != "" {
			rsql += fmt.Sprintf(" and data<='%s'", p.Fim)
		}
		if filtro := strings.TrimSpace(p.NomeFiltro); filtro != "" {
			safe := strings.ReplaceAll(filtro, "'", "")
			rsql += fmt.Sprintf(" and fornecedor=='*%s*'", safe)
		}

		q := url.Values{}
		q.Set("page", "0")
		q.Set("size", strconv.Itoa(limit))
		q.Set("entidade", strconv.Itoa(entidade))
		q.Set("exercicio", strconv.Itoa(ano))
		q.Set("search", rsql)
		if p.Inicio != "" {
			q.Set("dataInicial", p.Inicio)
		}
		if p.Fim != "" {
			q.Set("dataFinal", p.Fim)
		}

		u := fmt.Sprintf("%s/empenhos/lista?%s", base, q.Encode())
		body, status, err := doElotechGet(ctx, client, u, nil)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK || len(strings.TrimSpace(string(body))) == 0 {
			continue
		}

		var page elotechPage[elotechEmpenhoDTO]
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("erro ao decodificar Elotech /empenhos/lista: %w", err)
		}
		if len(page.Content) == 0 && page.TotalElements == 0 {
			continue
		}

		items := make([]DespesaItem, 0, len(page.Content))
		for _, r := range page.Content {
			numEmpenho := strings.TrimSpace(r.EmpenhoExercicio)
			if numEmpenho == "" && r.Empenho > 0 {
				numEmpenho = strconv.FormatInt(r.Empenho, 10)
			}

			fornecedor := strings.TrimSpace(r.Nome)
			docFornecedor := ""
			if parts := strings.SplitN(fornecedor, " - ", 2); len(parts) == 2 {
				digits := cleanDigits(parts[0])
				if len(digits) == 11 {
					docFornecedor = maskCPF(parts[0], digits)
					fornecedor = strings.TrimSpace(parts[1])
				} else if len(digits) == 14 {
					docFornecedor = strings.TrimSpace(parts[0])
					fornecedor = strings.TrimSpace(parts[1])
				}
			}

			items = append(items, DespesaItem{
				NumeroEmpenho:       numEmpenho,
				Ano:                 r.Exercicio,
				Data:                strings.TrimSpace(r.Data),
				Fornecedor:          fornecedor,
				DocumentoFornecedor: docFornecedor,
				Elemento:            strings.TrimSpace(r.FonteRecurso),
				Historico:           strings.TrimSpace(r.Historico),
				ValorEmpenhado:      r.ValorEmpenhado,
				ValorLiquidado:      r.ValorLiquidado,
				ValorPago:           r.ValorPago,
				Fonte:               "Portal Municipal (Elotech)",
			})
		}

		return &DespesasResult{
			Municipio:      m,
			Fonte:          "Portal Municipal (Elotech)",
			Plataforma:     "elotech",
			PortalURL:      base,
			TotalRegistros: page.TotalElements,
			Despesas:       items,
			Disponivel:     true,
		}, nil
	}

	return &DespesasResult{
		Municipio:  m,
		Fonte:      "Portal Municipal (Elotech)",
		Plataforma: "elotech",
		PortalURL:  base,
		Despesas:   []DespesaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma despesa encontrada para o período informado.",
	}, nil
}

// QueryElotechReceitas consulta receitas orçamentárias no endpoint Elotech /api/receitas.
func QueryElotechReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	limit := normalizeLimit(p.Limit)
	entidade := resolveElotechEntidade(p)

	for _, ano := range resolveElotechYears(p) {
		q := url.Values{}
		q.Set("entidade", strconv.Itoa(entidade))
		q.Set("exercicio", strconv.Itoa(ano))
		if p.Inicio != "" {
			q.Set("dataInicial", p.Inicio)
		}
		if p.Fim != "" {
			q.Set("dataFinal", p.Fim)
		}
		if strings.TrimSpace(p.NomeFiltro) != "" {
			q.Set("descricaoReceita", strings.TrimSpace(p.NomeFiltro))
		}

		u := fmt.Sprintf("%s/api/receitas?%s", base, q.Encode())
		body, status, err := doElotechGet(ctx, client, u, nil)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK || len(strings.TrimSpace(string(body))) == 0 {
			continue
		}

		var rows []elotechReceitaDTO
		if err := json.Unmarshal(body, &rows); err != nil {
			return nil, fmt.Errorf("erro ao decodificar Elotech /api/receitas: %w", err)
		}
		if len(rows) == 0 {
			continue
		}

		items := make([]ReceitaItem, 0, minInt(limit, len(rows)))
		for _, r := range rows {
			if len(items) >= limit {
				break
			}
			categoria := strings.TrimSpace(r.DescricaoFonte)
			if categoria == "" {
				categoria = strings.TrimSpace(r.FonteRecurso)
			}
			previsto := r.ValorOrcado
			if previsto == 0 {
				previsto = r.ValorPrevisto
			}
			if previsto == 0 {
				previsto = r.ValorOrcadoAtualizado
			}

			items = append(items, ReceitaItem{
				Codigo:          strings.TrimSpace(r.Receita),
				Descricao:       strings.TrimSpace(r.Descricao),
				Categoria:       categoria,
				Data:            strings.TrimSpace(r.Data),
				ValorPrevisto:   previsto,
				ValorArrecadado: r.ValorArrecadado,
				Fonte:           "Portal Municipal (Elotech)",
			})
		}

		return &ReceitasResult{
			Municipio:      m,
			Fonte:          "Portal Municipal (Elotech)",
			Plataforma:     "elotech",
			PortalURL:      base,
			TotalRegistros: len(rows),
			Receitas:       items,
			Disponivel:     true,
		}, nil
	}

	return &ReceitasResult{
		Municipio:  m,
		Fonte:      "Portal Municipal (Elotech)",
		Plataforma: "elotech",
		PortalURL:  base,
		Receitas:   []ReceitaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma receita encontrada para o período informado.",
	}, nil
}

// QueryElotechLicitacoes consulta licitações no endpoint Elotech /api/licitacoes.
func QueryElotechLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LicitacoesResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	limit := normalizeLimit(p.Limit)
	entidade := resolveElotechEntidade(p)

	for _, ano := range resolveElotechYears(p) {
		q := url.Values{}
		q.Set("page", "0")
		q.Set("size", strconv.Itoa(limit))
		q.Set("entidade", strconv.Itoa(entidade))
		q.Set("exercicio", strconv.Itoa(ano))
		if strings.TrimSpace(p.NomeFiltro) != "" {
			q.Set("objeto", strings.TrimSpace(p.NomeFiltro))
		}
		if p.Inicio != "" {
			q.Set("dataAberturaMaior", p.Inicio)
		}

		u := fmt.Sprintf("%s/api/licitacoes?%s", base, q.Encode())
		body, status, err := doElotechGet(ctx, client, u, nil)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK || len(strings.TrimSpace(string(body))) == 0 {
			continue
		}

		var page elotechPage[elotechLicitacaoDTO]
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("erro ao decodificar Elotech /api/licitacoes: %w", err)
		}
		if len(page.Content) == 0 && page.TotalElements == 0 {
			continue
		}

		items := make([]LicitacaoItem, 0, len(page.Content))
		for _, r := range page.Content {
			num := strings.TrimSpace(r.DisplayLicitacao)
			if num == "" && r.NumeroLicitacao > 0 {
				num = strconv.FormatInt(r.NumeroLicitacao, 10)
			}
			if num == "" && r.Licitacao > 0 {
				num = strconv.FormatInt(r.Licitacao, 10)
			}
			dataAbertura := strings.TrimSpace(r.DataAbertura)
			if dataAbertura == "" {
				dataAbertura = strings.TrimSpace(r.DataPublicacao)
			}

			items = append(items, LicitacaoItem{
				Numero:          num,
				Ano:             r.Exercicio,
				Modalidade:      strings.TrimSpace(r.Descricao),
				Situacao:        strings.TrimSpace(r.TipoCancelamento),
				DataAbertura:    dataAbertura,
				Objeto:          strings.TrimSpace(r.Objeto),
				ValorEstimado:   r.ValorMaximo,
				ValorHomologado: r.TotalVencido,
				Link:            base,
				Fonte:           "Portal Municipal (Elotech)",
			})
		}

		return &LicitacoesResult{
			Municipio:      m,
			Fonte:          "Portal Municipal (Elotech)",
			Plataforma:     "elotech",
			PortalURL:      base,
			TotalRegistros: page.TotalElements,
			Licitacoes:     items,
			Disponivel:     true,
		}, nil
	}

	return &LicitacoesResult{
		Municipio:  m,
		Fonte:      "Portal Municipal (Elotech)",
		Plataforma: "elotech",
		PortalURL:  base,
		Licitacoes: []LicitacaoItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma licitação encontrada para o período informado.",
	}, nil
}
