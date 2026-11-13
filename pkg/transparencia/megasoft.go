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
	"sync"
	"time"
)

const megasoftClientHeader = "megasoft-portal-da-transparencia"

// MegasoftConfig representa o payload de /api/configuracao/carregamento-inicial.
type MegasoftConfig struct {
	Tipo              int    `json:"tipo"`
	CodigoDoOrgao     int    `json:"codigoDoOrgao"`
	TituloDoTopo      string `json:"tituloDoTopo"`
	Cidade            string `json:"cidade"`
	UF                string `json:"uf"`
	LinkDoSiteDoOrgao string `json:"linkDoSiteDoOrgao"`
	UrlDoMegaAdmWeb   string `json:"urlDoMegaAdmWeb"`
	TokenPortal       string `json:"tokenPortal"`
}

type cachedMegaToken struct {
	cfg       *MegasoftConfig
	expiresAt time.Time
}

var (
	megaTokenMu    sync.RWMutex
	megaTokenCache = make(map[string]cachedMegaToken)
)

// FetchMegasoftConfig consulta e armazena em cache temporário o tokenPortal de um portal Megasoft.
func FetchMegasoftConfig(ctx context.Context, client *http.Client, baseURL string, forceRefresh bool) (*MegasoftConfig, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("baseURL Megasoft vazia")
	}

	if !forceRefresh {
		megaTokenMu.RLock()
		cached, ok := megaTokenCache[base]
		megaTokenMu.RUnlock()
		if ok && cached.cfg != nil && cached.cfg.TokenPortal != "" && time.Now().Before(cached.expiresAt) {
			return cached.cfg, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/configuracao/carregamento-inicial", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("megasoft carregamento-inicial HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var cfg MegasoftConfig
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("erro ao decodificar carregamento-inicial Megasoft: %w", err)
	}
	if strings.TrimSpace(cfg.TokenPortal) == "" {
		return nil, fmt.Errorf("megasoft carregamento-inicial sem tokenPortal")
	}

	megaTokenMu.Lock()
	megaTokenCache[base] = cachedMegaToken{
		cfg:       &cfg,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	megaTokenMu.Unlock()

	return &cfg, nil
}

func doMegasoftRequest(ctx context.Context, client *http.Client, baseURL, method, path string, payload any) ([]byte, int, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")

	attempt := func(forceRefresh bool) ([]byte, int, error) {
		cfg, err := FetchMegasoftConfig(ctx, client, base, forceRefresh)
		if err != nil {
			return nil, 0, err
		}

		var bodyReader io.Reader
		if payload != nil {
			raw, err := json.Marshal(payload)
			if err != nil {
				return nil, 0, err
			}
			bodyReader = bytes.NewReader(raw)
		}

		req, err := http.NewRequestWithContext(ctx, method, base+path, bodyReader)
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("Authorization", "Bearer "+cfg.TokenPortal)
		req.Header.Set("cliente-integrado", megasoftClientHeader)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer resp.Body.Close()

		b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		if err != nil {
			return nil, resp.StatusCode, err
		}
		return b, resp.StatusCode, nil
	}

	b, status, err := attempt(false)
	if err == nil && status == http.StatusUnauthorized {
		return attempt(true)
	}
	return b, status, err
}

type megaPage[T any] struct {
	Total       int `json:"total"`
	Registros   []T `json:"registros"`
	PaginaAtual int `json:"paginaAtual"`
}

type megaServidorRaw struct {
	MesAno        string  `json:"mesAno"`
	Matricula     any     `json:"matricula"`
	CPF           string  `json:"cpf"`
	Nome          string  `json:"nome"`
	Cargo         string  `json:"cargo"`
	DataAdmissao  string  `json:"dataAdmissao"`
	Proventos     float64 `json:"proventos"`
	Descontos     float64 `json:"descontos"`
	TotalLiquido  float64 `json:"totalLiquido"`
	Departamento  string  `json:"departamento"`
	TipoDeVinculo string  `json:"tipoDeVinculo"`
	CargaHoraria  any     `json:"cargaHoraria"`
	Situacao      string  `json:"situacao"`
	Mes           string  `json:"mes"`
	Ano           int     `json:"ano"`
}

type megaMovRegistro struct {
	Nome      string  `json:"nome"`
	Proventos float64 `json:"proventos"`
	Descontos float64 `json:"descontos"`
}

type megaMovSomatorio struct {
	SomatorioProventos float64 `json:"somatorioProventos"`
	SomatorioDescontos float64 `json:"somatorioDescontos"`
	TotalLiquido       float64 `json:"totalLiquido"`
}

type megaServidorDetalheRaw struct {
	MesAno         string                  `json:"mesAno"`
	Matricula      any                     `json:"matricula"`
	CPF            string                  `json:"cpf"`
	Nome           string                  `json:"nome"`
	Orgao          string                  `json:"orgao"`
	Cargo          string                  `json:"cargo"`
	DataAdmissao   string                  `json:"dataAdmissao"`
	Departamento   string                  `json:"departamento"`
	TipoDeVinculo  string                  `json:"tipoDeVinculo"`
	CargaHoraria   any                     `json:"cargaHoraria"`
	Situacao       string                  `json:"situacao"`
	MovVencimentos megaPage[megaMovRegistro] `json:"movimentoVencimentos"`
	MovFerias      megaPage[megaMovRegistro] `json:"movimentoFerias"`
	Mov13          megaPage[megaMovRegistro] `json:"movimento13Salario"`
	SomVenc        megaMovSomatorio          `json:"somatorioMovimentoVencimentos"`
	SomFerias      megaMovSomatorio          `json:"somatorioMovimentoFerias"`
	Som13          megaMovSomatorio          `json:"somatorioMovimento13Salario"`
}

type megaDiariaRaw struct {
	CodigoDaViagem    any     `json:"codigoDaViagem"`
	Matricula         any     `json:"matriculaDoFuncionario"`
	Nome              string  `json:"nomeDoFuncionario"`
	Cargo             string  `json:"cargoDoFuncionario"`
	Destino           string  `json:"destino"`
	DataDaSaida       string  `json:"dataDaSaida"`
	DataDaChegada     string  `json:"dataDaChegada"`
	ValorDaDiaria     float64 `json:"valorDaDiaria"`
	NumeroDeDiarias   float64 `json:"numeroDeDiarias"`
	MotivoDaViagem    string  `json:"motivoDaViagem"`
	NumeroDaLiquidacao string `json:"numeroDaLiquidacao"`
}

type megaEmpenhoRaw struct {
	Codigo                 any     `json:"codigo"`
	NumeroDoTcm            any     `json:"numeroDoTcm"`
	Data                   string  `json:"data"`
	NomeDoFornecedor       string  `json:"nomeDoFornecedor"`
	CNPJENome              string  `json:"cnpjENomeDoFornecedor"`
	CodigoENomeDoOrgao     string  `json:"codigoENomeDoOrgao"`
	CodigoENomeDaUnidade   string  `json:"codigoENomeDaUnidade"`
	CodigoENomeDaFuncao    string  `json:"codigoENomeDaFuncao"`
	CodigoENomeDaNatureza  string  `json:"codigoENomeDaNaturezaDespesa"`
	ValorDoEmpenho         float64 `json:"valorDoEmpenho"`
	ValorTotalDoEmpenho    float64 `json:"valorTotalDoEmpenho"`
	ValorDaLiquidacao      float64 `json:"valorDaLiquidacao"`
	ValorTotalDaLiquidacao float64 `json:"valorTotalDaLiquidacao"`
	ValorDoPagamento       float64 `json:"valorDoPagamento"`
	ValorTotalDoPagamento  float64 `json:"valorTotalDoPagamento"`
	Historico              string  `json:"historico"`
}

type megaReceitaRaw struct {
	Chave struct {
		CodigoDaReceita any    `json:"codigoDaReceita"`
		Ano             int    `json:"ano"`
		Mes             string `json:"mes"`
	} `json:"chave"`
	FonteDaReceita   string  `json:"fonteDaReceita"`
	Receita          string  `json:"receita"`
	Natureza         string  `json:"naturezaDaReceita"`
	OrigemDoRecurso  string  `json:"origemDoRecurso"`
	ValorOrcado      float64 `json:"valorOrcado"`
	Arrecadacao      float64 `json:"arrecadacao"`
	ValorAcumulado   float64 `json:"valorAcumulado"`
}

type megaLicitacaoRaw struct {
	Codigo          any    `json:"codigo"`
	NumeroAno       string `json:"numeroAno"`
	DataJulgamento  string `json:"dataDeJulgamento"`
	DataPublicacao  string `json:"dataDePublicacao"`
	Modalidade      string `json:"modalidade"`
	Situacao        string `json:"situacao"`
	Historico       string `json:"historico"`
	Orgao           string `json:"orgao"`
}

type megaContratoRaw struct {
	Codigo         any     `json:"codigo"`
	Numero         string  `json:"numero"`
	Modalidade     string  `json:"modalidade"`
	Data           string  `json:"data"`
	Vigencia       string  `json:"vigencia"`
	NomeDoOrgao    string  `json:"nomeDoOrgao"`
	Documento      string  `json:"cpfOuCnpjDoFornecedor"`
	Fornecedor     string  `json:"nomeDoFornecedor"`
	Objeto         string  `json:"objeto"`
	Valor          float64 `json:"valor"`
	ExercicioAno   int     `json:"exercicioAno"`
	Situacao       string  `json:"situacao"`
}

type megaObraRaw struct {
	Codigo      any     `json:"codigoDaObra"`
	NomeOrgao   string  `json:"nomeOrgao"`
	Titulo      string  `json:"titulo"`
	DataInicial string  `json:"dataInicial"`
	DataFinal   string  `json:"dataFinal"`
	Valor       float64 `json:"valor"`
	Endereco    string  `json:"endereco"`
	Bairro      string  `json:"bairro"`
	Objeto      string  `json:"objeto"`
	Situacao    string  `json:"situacao"`
}

type megaConvenioRaw struct {
	Codigo        any     `json:"codigoDoConvenio"`
	Numero        string  `json:"numeroDoConvenio"`
	Data          string  `json:"data"`
	Vigencia      string  `json:"vigencia"`
	Descricao     string  `json:"descricao"`
	Valor         float64 `json:"valor"`
	ValorLiberado float64 `json:"valorLiberado"`
	Concedente    string  `json:"nomeDoConcedente"`
	Convenente    string  `json:"nomeDoConvenente"`
	Categoria     string  `json:"categoriaDoConvenio"`
	Situacao      string  `json:"situacaoDoConvenio"`
	Tipo          string  `json:"tipo"`
}

type megaLegislacaoRaw struct {
	Codigo         any    `json:"codigoDoDocumento"`
	TipoDocumento  string `json:"tipoDoDocumento"`
	Ano            int    `json:"ano"`
	Situacao       string `json:"situacao"`
	Numero         string `json:"numeroDoDocumento"`
	DataPublicacao string `json:"dataDaPublicacao"`
	Nome           string `json:"nomeDoDocumento"`
	Descricao      string `json:"descricao"`
}

type megaConcursoRaw struct {
	NumeroEdital   any    `json:"numeroDoEdital"`
	DataPublicacao string `json:"dataDePublicacao"`
	NomeDaPessoa   string `json:"nomeDaPessoa"`
	NomeDoCargo    string `json:"nomeDoCargo"`
	Inscricao      any    `json:"inscricao"`
	Classificacao  int    `json:"classificacao"`
	Situacao       string `json:"descricaoDaSituacao"`
}

func toMegasoftDate(isoOrBR string) string {
	s := strings.TrimSpace(isoOrBR)
	if len(s) == 10 && s[4] == '-' && s[7] == '-' {
		return s[8:10] + "/" + s[5:7] + "/" + s[0:4]
	}
	return s
}

func extractYearFromBRDate(d string) int {
	if len(d) >= 10 && d[2] == '/' && d[5] == '/' {
		y, _ := strconv.Atoi(d[6:10])
		return y
	}
	if len(d) >= 4 {
		y, _ := strconv.Atoi(d[:4])
		return y
	}
	return 0
}

func extractCNPJPrefix(s string) string {
	parts := strings.SplitN(strings.TrimSpace(s), "/", 2)
	if len(parts) > 0 && len(cleanDigits(parts[0])) >= 11 {
		// Note: CNPJ contains '/' in 00.000.000/0001-00, so take before second '/' if format is CNPJ/NOME
		if len(s) >= 18 && s[14] == '-' {
			return s[:18]
		}
	}
	return ""
}

// resolveYears retorna os anos de consulta (ou [anoAtual, anoAnterior] se ano == 0).
func resolveYears(ano int) []int {
	if ano > 0 {
		return []int{ano}
	}
	y := time.Now().Year()
	return []int{y, y - 1, y - 2}
}

// resolveMegasoftCompetencias gera pares (ano, mes) priorizando o mês informado ou fallback para meses fechados recentes.
func resolveMegasoftCompetencias(ano, mes int) [][2]int {
	now := time.Now()
	if ano > 0 && mes >= 1 && mes <= 12 {
		return [][2]int{{ano, mes}}
	}
	if ano > 0 {
		startMes := 12
		if ano == now.Year() {
			startMes = int(now.Month())
		}
		var out [][2]int
		for m := startMes; m >= 1 && len(out) < 4; m-- {
			out = append(out, [2]int{ano, m})
		}
		return out
	}
	var out [][2]int
	cur := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 4; i++ {
		t := cur.AddDate(0, -i, 0)
		out = append(out, [2]int{t.Year(), int(t.Month())})
	}
	return out
}

// QueryMegasoftServidores consulta a folha de pagamento de servidores públicos no portal Megasoft.
func QueryMegasoftServidores(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ServidoresResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	maxLimit := p.Limit
	if maxLimit <= 0 {
		maxLimit = 25
	}
	if maxLimit > 1000 {
		maxLimit = 1000
	}
	pageSize := 200
	if maxLimit < 200 {
		pageSize = maxLimit
	}
	competencias := resolveMegasoftCompetencias(p.Ano, p.Mes)

	var bestNonEmpty *ServidoresResult

	for _, comp := range competencias {
		q := url.Values{}
		q.Set("pagina", "1")
		q.Set("tamanhoDaPagina", strconv.Itoa(pageSize))
		q.Set("ano", strconv.Itoa(comp[0]))
		q.Set("mes", fmt.Sprintf("%02d", comp[1]))
		if strings.TrimSpace(p.NomeFiltro) != "" {
			q.Set("nomeDoFuncionario", strings.TrimSpace(p.NomeFiltro))
		}

		body, status, err := doMegasoftRequest(ctx, client, base, http.MethodGet, "/api/orgaos-e-servidores/servidor/paginado?"+q.Encode(), nil)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK || len(bytes.TrimSpace(body)) == 0 {
			continue
		}

		var page megaPage[megaServidorRaw]
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("erro ao decodificar /servidor/paginado Megasoft: %w", err)
		}
		if len(page.Registros) == 0 && page.Total == 0 {
			continue
		}

		hasPositiveSalary := false
		items := make([]ServidorItem, 0, len(page.Registros))
		addServidorItems := func(regs []megaServidorRaw) {
			for _, r := range regs {
				if r.Proventos > 0 || r.TotalLiquido > 0 {
					hasPositiveSalary = true
				}
				compStr := strings.TrimSpace(r.MesAno)
				if compStr == "" {
					compStr = fmt.Sprintf("%02d/%04d", comp[1], comp[0])
				}
				items = append(items, ServidorItem{
					Nome:           strings.TrimSpace(r.Nome),
					Matricula:      anyToString(r.Matricula),
					CPFParcial:     maskCPF(r.CPF, ""),
					Cargo:          strings.TrimSpace(r.Cargo),
					Lotacao:        strings.TrimSpace(r.Departamento),
					Vinculo:        strings.TrimSpace(r.TipoDeVinculo),
					Situacao:       strings.TrimSpace(r.Situacao),
					SalarioBruto:   r.Proventos,
					Descontos:      r.Descontos,
					SalarioLiquido: r.TotalLiquido,
					Competencia:    compStr,
					CargaHoraria:   anyToString(r.CargaHoraria),
					DataAdmissao:   strings.TrimSpace(r.DataAdmissao),
					Fonte:          "Portal Municipal (Megasoft)",
					PortalURL:      base,
				})
				if len(items) >= maxLimit {
					break
				}
			}
		}

		addServidorItems(page.Registros)

		if maxLimit > len(items) && page.Total > len(items) {
			for pageNum := 2; len(items) < maxLimit && (pageNum-1)*pageSize < page.Total; pageNum++ {
				q.Set("pagina", strconv.Itoa(pageNum))
				subBody, subStatus, subErr := doMegasoftRequest(ctx, client, base, http.MethodGet, "/api/orgaos-e-servidores/servidor/paginado?"+q.Encode(), nil)
				if subErr != nil || subStatus != http.StatusOK {
					break
				}
				var subPage megaPage[megaServidorRaw]
				if err := json.Unmarshal(subBody, &subPage); err != nil || len(subPage.Registros) == 0 {
					break
				}
				addServidorItems(subPage.Registros)
			}
		}

		res := &ServidoresResult{
			Municipio:      m,
			Fonte:          "Portal Municipal (Megasoft)",
			Plataforma:     "megasoft",
			PortalURL:      base,
			TotalRegistros: page.Total,
			Servidores:     items,
			Disponivel:     true,
		}

		// Se mês atual ainda está com folha zerada (em processamento), continua buscando mês anterior fechado.
		if hasPositiveSalary || (p.Ano > 0 && p.Mes > 0) {
			return res, nil
		}
		if bestNonEmpty == nil {
			bestNonEmpty = res
		}
	}

	if bestNonEmpty != nil {
		return bestNonEmpty, nil
	}

	return &ServidoresResult{
		Municipio:  m,
		Fonte:      "Portal Municipal (Megasoft)",
		Plataforma: "megasoft",
		PortalURL:  base,
		Servidores: []ServidorItem{},
		Disponivel: true,
		Mensagem:   "Nenhum servidor encontrado para os filtros informados.",
	}, nil
}

// QueryMegasoftServidorDetalhe consulta o detalhamento individual do contracheque no portal Megasoft.
func QueryMegasoftServidorDetalhe(ctx context.Context, client *http.Client, m *Municipio, baseURL, matricula string, ano, mes int) (*ServidorDetalheResult, error) {
	base := strings.TrimRight(baseURL, "/")
	if ano <= 0 || mes <= 0 {
		t := time.Now().AddDate(0, -1, 0)
		if ano <= 0 {
			ano = t.Year()
		}
		if mes <= 0 {
			mes = int(t.Month())
		}
	}

	q := url.Values{}
	q.Set("matricula", strings.TrimSpace(matricula))
	q.Set("mes", fmt.Sprintf("%02d", mes))
	q.Set("ano", strconv.Itoa(ano))

	body, status, err := doMegasoftRequest(ctx, client, base, http.MethodGet, "/api/orgaos-e-servidores/servidor/detalhe?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("megasoft /servidor/detalhe HTTP %d", status)
	}

	var raw megaServidorDetalheRaw
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	res := &ServidorDetalheResult{
		Municipio:    m,
		Nome:         strings.TrimSpace(raw.Nome),
		Matricula:    anyToString(raw.Matricula),
		CPFParcial:   maskCPF(raw.CPF, ""),
		Cargo:        strings.TrimSpace(raw.Cargo),
		Orgao:        strings.TrimSpace(raw.Orgao),
		Departamento: strings.TrimSpace(raw.Departamento),
		Vinculo:      strings.TrimSpace(raw.TipoDeVinculo),
		Situacao:     strings.TrimSpace(raw.Situacao),
		Competencia:  strings.TrimSpace(raw.MesAno),
		DataAdmissao: strings.TrimSpace(raw.DataAdmissao),
		CargaHoraria: anyToString(raw.CargaHoraria),
		PortalURL:    base,
		Fonte:        "Portal Municipal (Megasoft)",
		Disponivel:   true,
	}

	for _, r := range raw.MovVencimentos.Registros {
		if r.Proventos > 0 {
			res.Vencimentos = append(res.Vencimentos, VerbaItem{Descricao: r.Nome, Valor: r.Proventos})
		}
		if r.Descontos > 0 {
			res.VerbasDesconto = append(res.VerbasDesconto, VerbaItem{Descricao: r.Nome, Valor: r.Descontos})
		}
	}
	for _, r := range raw.MovFerias.Registros {
		res.Ferias = append(res.Ferias, VerbaItem{Descricao: r.Nome, Valor: r.Proventos})
	}
	for _, r := range raw.Mov13.Registros {
		res.DecimoTerceiro = append(res.DecimoTerceiro, VerbaItem{Descricao: r.Nome, Valor: r.Proventos})
	}

	res.SalarioBruto = raw.SomVenc.SomatorioProventos + raw.SomFerias.SomatorioProventos + raw.Som13.SomatorioProventos
	res.Descontos = raw.SomVenc.SomatorioDescontos + raw.SomFerias.SomatorioDescontos + raw.Som13.SomatorioDescontos
	res.SalarioLiquido = raw.SomVenc.TotalLiquido + raw.SomFerias.TotalLiquido + raw.Som13.TotalLiquido

	return res, nil
}

// QueryMegasoftDiarias consulta diárias e viagens de servidores/agentes públicos no portal Megasoft.
func QueryMegasoftDiarias(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DiariasResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	limit := normalizeLimit(p.Limit)
	years := resolveYears(p.Ano)

	for _, yr := range years {
		q := url.Values{}
		q.Set("pagina", "1")
		q.Set("tamanhoDaPagina", strconv.Itoa(limit))
		q.Set("ano", strconv.Itoa(yr))
		if strings.TrimSpace(p.NomeFiltro) != "" {
			q.Set("nomeDoFuncionario", strings.TrimSpace(p.NomeFiltro))
		}

		body, status, err := doMegasoftRequest(ctx, client, base, http.MethodGet, "/api/orgaos-e-servidores/diaria/paginado?"+q.Encode(), nil)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			continue
		}
		var page megaPage[megaDiariaRaw]
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, err
		}
		if len(page.Registros) == 0 && page.Total == 0 {
			continue
		}
		items := make([]DiariaItem, 0, len(page.Registros))
		for _, r := range page.Registros {
			items = append(items, DiariaItem{
				Beneficiario:  strings.TrimSpace(r.Nome),
				Matricula:     anyToString(r.Matricula),
				Cargo:         strings.TrimSpace(r.Cargo),
				Destino:       strings.TrimSpace(r.Destino),
				Motivo:        strings.TrimSpace(r.MotivoDaViagem),
				Data:          strings.TrimSpace(r.DataDaSaida),
				DataSaida:     strings.TrimSpace(r.DataDaSaida),
				DataChegada:   strings.TrimSpace(r.DataDaChegada),
				NumeroDiarias: r.NumeroDeDiarias,
				Valor:         r.ValorDaDiaria,
				Fonte:         "Portal Municipal (Megasoft)",
			})
		}
		return &DiariasResult{
			Municipio:      m,
			Fonte:          "Portal Municipal (Megasoft)",
			Plataforma:     "megasoft",
			PortalURL:      base,
			TotalRegistros: page.Total,
			Diarias:        items,
			Disponivel:     true,
		}, nil
	}

	return &DiariasResult{
		Municipio:  m,
		Fonte:      "Portal Municipal (Megasoft)",
		Plataforma: "megasoft",
		PortalURL:  base,
		Diarias:    []DiariaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma diária encontrada para o período consultado.",
	}, nil
}

// QueryMegasoftDespesas consulta despesas e empenhos no portal Megasoft.
func QueryMegasoftDespesas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*DespesasResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	limit := normalizeLimit(p.Limit)

	for _, win := range resolveDateWindows(p) {
		q := url.Values{}
		q.Set("pagina", "1")
		q.Set("tamanhoDaPagina", strconv.Itoa(limit))
		q.Set("dataInicial", toMegasoftDate(win[0]))
		q.Set("dataFinal", toMegasoftDate(win[1]))
		if strings.TrimSpace(p.NomeFiltro) != "" {
			q.Set("nomeDoFornecedor", strings.TrimSpace(p.NomeFiltro))
		}

		body, status, err := doMegasoftRequest(ctx, client, base, http.MethodGet, "/api/receitas-e-despesas/empenho/paginado?"+q.Encode(), nil)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			continue
		}
		var page megaPage[megaEmpenhoRaw]
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, err
		}
		if len(page.Registros) == 0 && page.Total == 0 {
			continue
		}
		items := make([]DespesaItem, 0, len(page.Registros))
		for _, r := range page.Registros {
			vlrEmp := r.ValorTotalDoEmpenho
			if vlrEmp == 0 {
				vlrEmp = r.ValorDoEmpenho
			}
			vlrLiq := r.ValorTotalDaLiquidacao
			if vlrLiq == 0 {
				vlrLiq = r.ValorDaLiquidacao
			}
			vlrPag := r.ValorTotalDoPagamento
			if vlrPag == 0 {
				vlrPag = r.ValorDoPagamento
			}
			numEmp := anyToString(r.NumeroDoTcm)
			if numEmp == "" {
				numEmp = anyToString(r.Codigo)
			}
			items = append(items, DespesaItem{
				NumeroEmpenho:       numEmp,
				Ano:                 extractYearFromBRDate(r.Data),
				Data:                strings.TrimSpace(r.Data),
				Fornecedor:          strings.TrimSpace(r.NomeDoFornecedor),
				DocumentoFornecedor: extractCNPJPrefix(r.CNPJENome),
				Orgao:               strings.TrimSpace(r.CodigoENomeDoOrgao),
				Unidade:             strings.TrimSpace(r.CodigoENomeDaUnidade),
				Funcao:              strings.TrimSpace(r.CodigoENomeDaFuncao),
				Elemento:            strings.TrimSpace(r.CodigoENomeDaNatureza),
				Historico:           strings.TrimSpace(r.Historico),
				ValorEmpenhado:      vlrEmp,
				ValorLiquidado:      vlrLiq,
				ValorPago:           vlrPag,
				Fonte:               "Portal Municipal (Megasoft)",
			})
		}
		return &DespesasResult{
			Municipio:      m,
			Fonte:          "Portal Municipal (Megasoft)",
			Plataforma:     "megasoft",
			PortalURL:      base,
			TotalRegistros: page.Total,
			Despesas:       items,
			Disponivel:     true,
		}, nil
	}

	return &DespesasResult{
		Municipio:  m,
		Fonte:      "Portal Municipal (Megasoft)",
		Plataforma: "megasoft",
		PortalURL:  base,
		Despesas:   []DespesaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma despesa encontrada para os filtros informados.",
	}, nil
}

// QueryMegasoftReceitas consulta receitas arrecadadas no portal Megasoft.
func QueryMegasoftReceitas(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ReceitasResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	limit := normalizeLimit(p.Limit)
	years := resolveYears(p.Ano)

	for _, yr := range years {
		q := url.Values{}
		q.Set("pagina", "1")
		q.Set("tamanhoDaPagina", strconv.Itoa(limit))
		q.Set("ano", strconv.Itoa(yr))
		if p.Mes > 0 {
			q.Set("mes", fmt.Sprintf("%02d", p.Mes))
		}

		body, status, err := doMegasoftRequest(ctx, client, base, http.MethodGet, "/api/receitas-e-despesas/receita/paginado?"+q.Encode(), nil)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			continue
		}
		var page megaPage[megaReceitaRaw]
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, err
		}
		if len(page.Registros) == 0 && page.Total == 0 {
			continue
		}
		items := make([]ReceitaItem, 0, len(page.Registros))
		for _, r := range page.Registros {
			arrec := r.ValorAcumulado
			if arrec == 0 {
				arrec = r.Arrecadacao
			}
			dataStr := ""
			if r.Chave.Ano > 0 {
				if r.Chave.Mes != "" {
					dataStr = r.Chave.Mes + "/" + strconv.Itoa(r.Chave.Ano)
				} else {
					dataStr = strconv.Itoa(r.Chave.Ano)
				}
			}
			items = append(items, ReceitaItem{
				Codigo:          strings.TrimSpace(r.Natureza),
				Descricao:       strings.TrimSpace(r.Receita),
				Categoria:       strings.TrimSpace(r.FonteDaReceita),
				Data:            dataStr,
				ValorPrevisto:   r.ValorOrcado,
				ValorArrecadado: arrec,
				Fonte:           "Portal Municipal (Megasoft)",
			})
		}
		return &ReceitasResult{
			Municipio:      m,
			Fonte:          "Portal Municipal (Megasoft)",
			Plataforma:     "megasoft",
			PortalURL:      base,
			TotalRegistros: page.Total,
			Receitas:       items,
			Disponivel:     true,
		}, nil
	}

	return &ReceitasResult{
		Municipio:  m,
		Fonte:      "Portal Municipal (Megasoft)",
		Plataforma: "megasoft",
		PortalURL:  base,
		Receitas:   []ReceitaItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma receita encontrada para o exercício consultado.",
	}, nil
}

// QueryMegasoftLicitacoes consulta procedimentos licitatórios (POST) no portal Megasoft.
func QueryMegasoftLicitacoes(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LicitacoesResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	limit := normalizeLimit(p.Limit)

	for _, tipoMod := range []int{1, 2} {
		payload := map[string]any{
			"pagina":                     1,
			"tamanhoDaPagina":            limit,
			"tipoDeConsultaDeModalidade": tipoMod,
		}
		if p.Ano > 0 {
			payload["ano"] = p.Ano
		}
		if strings.TrimSpace(p.NomeFiltro) != "" {
			payload["produto"] = strings.TrimSpace(p.NomeFiltro)
		}

		body, status, err := doMegasoftRequest(ctx, client, base, http.MethodPost, "/api/contratos-convenios-e-licitacoes/procedimento-licitatorio/paginado", payload)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			continue
		}
		var page megaPage[megaLicitacaoRaw]
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, err
		}
		if len(page.Registros) == 0 && page.Total == 0 {
			continue
		}
		items := make([]LicitacaoItem, 0, len(page.Registros))
		for _, r := range page.Registros {
			abertura := strings.TrimSpace(r.DataJulgamento)
			if abertura == "" {
				abertura = strings.TrimSpace(r.DataPublicacao)
			}
			items = append(items, LicitacaoItem{
				Numero:       strings.TrimSpace(r.NumeroAno),
				Ano:          extractYearFromBRDate(abertura),
				Modalidade:   strings.TrimSpace(r.Modalidade),
				Situacao:     strings.TrimSpace(r.Situacao),
				DataAbertura: abertura,
				Objeto:       strings.TrimSpace(r.Historico),
				Fonte:        "Portal Municipal (Megasoft)",
			})
		}
		return &LicitacoesResult{
			Municipio:      m,
			Fonte:          "Portal Municipal (Megasoft)",
			Plataforma:     "megasoft",
			PortalURL:      base,
			TotalRegistros: page.Total,
			Licitacoes:     items,
			Disponivel:     true,
		}, nil
	}

	return &LicitacoesResult{
		Municipio:  m,
		Fonte:      "Portal Municipal (Megasoft)",
		Plataforma: "megasoft",
		PortalURL:  base,
		Licitacoes: []LicitacaoItem{},
		Disponivel: true,
		Mensagem:   "Nenhuma licitação encontrada no portal Megasoft.",
	}, nil
}

// QueryMegasoftContratos consulta contratos administrativos (POST) no portal Megasoft.
func QueryMegasoftContratos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ContratosResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	limit := normalizeLimit(p.Limit)

	payload := map[string]any{
		"pagina":          1,
		"tamanhoDaPagina": limit,
	}
	if p.Ano > 0 {
		payload["ano"] = p.Ano
	}
	if strings.TrimSpace(p.NomeFiltro) != "" {
		payload["nomeDoFornecedor"] = strings.TrimSpace(p.NomeFiltro)
	}

	body, status, err := doMegasoftRequest(ctx, client, base, http.MethodPost, "/api/contratos-convenios-e-licitacoes/contrato/paginado", payload)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("megasoft /contrato/paginado HTTP %d", status)
	}

	var page megaPage[megaContratoRaw]
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, err
	}

	items := make([]ContratoItem, 0, len(page.Registros))
	for _, r := range page.Registros {
		items = append(items, ContratoItem{
			Numero:              strings.TrimSpace(r.Numero),
			Ano:                 r.ExercicioAno,
			Fornecedor:          strings.TrimSpace(r.Fornecedor),
			DocumentoFornecedor: strings.TrimSpace(r.Documento),
			Objeto:              strings.TrimSpace(r.Objeto),
			Situacao:            strings.TrimSpace(r.Situacao),
			DataAssinatura:      strings.TrimSpace(r.Data),
			VigenciaFim:         strings.TrimSpace(r.Vigencia),
			Valor:               r.Valor,
			Fonte:               "Portal Municipal (Megasoft)",
		})
	}

	return &ContratosResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Megasoft)",
		Plataforma:     "megasoft",
		PortalURL:      base,
		TotalRegistros: page.Total,
		Contratos:      items,
		Disponivel:     true,
	}, nil
}

// QueryMegasoftObras consulta acompanhamento de obras públicas no portal Megasoft.
func QueryMegasoftObras(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ObrasResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	limit := normalizeLimit(p.Limit)

	q := url.Values{}
	q.Set("pagina", "1")
	q.Set("tamanhoDaPagina", strconv.Itoa(limit))
	if strings.TrimSpace(p.NomeFiltro) != "" {
		q.Set("objeto", strings.TrimSpace(p.NomeFiltro))
	}

	body, status, err := doMegasoftRequest(ctx, client, base, http.MethodGet, "/api/contratos-convenios-e-licitacoes/obra/paginado?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("megasoft /obra/paginado HTTP %d", status)
	}

	var page megaPage[megaObraRaw]
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, err
	}

	items := make([]ObraItem, 0, len(page.Registros))
	for _, r := range page.Registros {
		titulo := strings.TrimSpace(r.Titulo)
		if titulo == "" {
			titulo = strings.TrimSpace(r.Objeto)
		}
		items = append(items, ObraItem{
			Codigo:      anyToString(r.Codigo),
			Titulo:      titulo,
			Objeto:      strings.TrimSpace(r.Objeto),
			Orgao:       strings.TrimSpace(r.NomeOrgao),
			Endereco:    strings.TrimSpace(r.Endereco),
			Bairro:      strings.TrimSpace(r.Bairro),
			DataInicial: strings.TrimSpace(r.DataInicial),
			DataFinal:   strings.TrimSpace(r.DataFinal),
			Situacao:    strings.TrimSpace(r.Situacao),
			Valor:       r.Valor,
			Fonte:       "Portal Municipal (Megasoft)",
		})
	}

	return &ObrasResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Megasoft)",
		Plataforma:     "megasoft",
		PortalURL:      base,
		TotalRegistros: page.Total,
		Obras:          items,
		Disponivel:     true,
	}, nil
}

// QueryMegasoftConvenios consulta convênios e parcerias no portal Megasoft.
func QueryMegasoftConvenios(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ConveniosResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	limit := normalizeLimit(p.Limit)

	q := url.Values{}
	q.Set("pagina", "1")
	q.Set("tamanhoDaPagina", strconv.Itoa(limit))
	if strings.TrimSpace(p.NomeFiltro) != "" {
		q.Set("objeto", strings.TrimSpace(p.NomeFiltro))
	}

	body, status, err := doMegasoftRequest(ctx, client, base, http.MethodGet, "/api/contratos-convenios-e-licitacoes/convenio/paginado?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("megasoft /convenio/paginado HTTP %d", status)
	}

	var page megaPage[megaConvenioRaw]
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, err
	}

	items := make([]ConvenioItem, 0, len(page.Registros))
	for _, r := range page.Registros {
		num := strings.TrimSpace(r.Numero)
		if num == "" {
			num = anyToString(r.Codigo)
		}
		items = append(items, ConvenioItem{
			Numero:        num,
			Concedente:    strings.TrimSpace(r.Concedente),
			Convenente:    strings.TrimSpace(r.Convenente),
			Categoria:     strings.TrimSpace(r.Categoria),
			Tipo:          strings.TrimSpace(r.Tipo),
			Situacao:      strings.TrimSpace(r.Situacao),
			Data:          strings.TrimSpace(r.Data),
			Vigencia:      strings.TrimSpace(r.Vigencia),
			Descricao:     strings.TrimSpace(r.Descricao),
			Valor:         r.Valor,
			ValorLiberado: r.ValorLiberado,
			Fonte:         "Portal Municipal (Megasoft)",
		})
	}

	return &ConveniosResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Megasoft)",
		Plataforma:     "megasoft",
		PortalURL:      base,
		TotalRegistros: page.Total,
		Convenios:      items,
		Disponivel:     true,
	}, nil
}

// QueryMegasoftLegislacao consulta leis, decretos e atos normativos municipais no portal Megasoft.
func QueryMegasoftLegislacao(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*LegislacaoResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	limit := normalizeLimit(p.Limit)

	q := url.Values{}
	q.Set("pagina", "1")
	q.Set("tamanhoDaPagina", strconv.Itoa(limit))
	if p.Ano > 0 {
		q.Set("ano", strconv.Itoa(p.Ano))
	}
	if strings.TrimSpace(p.NomeFiltro) != "" {
		q.Set("descricao", strings.TrimSpace(p.NomeFiltro))
	}

	body, status, err := doMegasoftRequest(ctx, client, base, http.MethodGet, "/api/legislacao-e-publicacoes/legislacao-municipal/paginado?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("megasoft /legislacao-municipal/paginado HTTP %d", status)
	}

	var page megaPage[megaLegislacaoRaw]
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, err
	}

	items := make([]LegislacaoItem, 0, len(page.Registros))
	for _, r := range page.Registros {
		items = append(items, LegislacaoItem{
			Codigo:         anyToString(r.Codigo),
			TipoDocumento:  strings.TrimSpace(r.TipoDocumento),
			Numero:         strings.TrimSpace(r.Numero),
			Ano:            r.Ano,
			DataPublicacao: strings.TrimSpace(r.DataPublicacao),
			Situacao:       strings.TrimSpace(r.Situacao),
			Nome:           strings.TrimSpace(r.Nome),
			Descricao:      strings.TrimSpace(r.Descricao),
			Fonte:          "Portal Municipal (Megasoft)",
		})
	}

	return &LegislacaoResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Megasoft)",
		Plataforma:     "megasoft",
		PortalURL:      base,
		TotalRegistros: page.Total,
		Atos:           items,
		Disponivel:     true,
	}, nil
}

// QueryMegasoftConcursos consulta lista de aprovados em concursos e processos seletivos no portal Megasoft.
func QueryMegasoftConcursos(ctx context.Context, client *http.Client, m *Municipio, p PortalQueryParams) (*ConcursosResult, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	limit := normalizeLimit(p.Limit)

	q := url.Values{}
	q.Set("pagina", "1")
	q.Set("tamanhoDaPagina", strconv.Itoa(limit))
	if strings.TrimSpace(p.NomeFiltro) != "" {
		q.Set("nomeDaPessoa", strings.TrimSpace(p.NomeFiltro))
	}

	body, status, err := doMegasoftRequest(ctx, client, base, http.MethodGet, "/api/orgaos-e-servidores/lista-de-aprovados-em-concursos-e-processos-seletivos/paginado?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("megasoft /lista-de-aprovados HTTP %d", status)
	}

	var page megaPage[megaConcursoRaw]
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, err
	}

	items := make([]ConcursoItem, 0, len(page.Registros))
	for _, r := range page.Registros {
		items = append(items, ConcursoItem{
			Edital:         anyToString(r.NumeroEdital),
			DataPublicacao: strings.TrimSpace(r.DataPublicacao),
			Nome:           strings.TrimSpace(r.NomeDaPessoa),
			Cargo:          strings.TrimSpace(r.NomeDoCargo),
			Inscricao:      anyToString(r.Inscricao),
			Classificacao:  r.Classificacao,
			Situacao:       strings.TrimSpace(r.Situacao),
			Fonte:          "Portal Municipal (Megasoft)",
		})
	}

	return &ConcursosResult{
		Municipio:      m,
		Fonte:          "Portal Municipal (Megasoft)",
		Plataforma:     "megasoft",
		PortalURL:      base,
		TotalRegistros: page.Total,
		Aprovados:      items,
		Disponivel:     true,
	}, nil
}
