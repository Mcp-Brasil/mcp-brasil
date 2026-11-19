package transparencia

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const pncpDefaultBaseURL = "https://pncp.gov.br/api/consulta/v1"

// PNCPClient é o cliente nativo em Go para consultar contratações e contratos de qualquer município brasileiro pelo CNPJ no PNCP.
type PNCPClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewPNCPClient cria uma instância do cliente PNCP.
func NewPNCPClient() *PNCPClient {
	return &PNCPClient{
		httpClient: &http.Client{Timeout: 12 * time.Second},
		baseURL:    pncpDefaultBaseURL,
	}
}

// NewPNCPClientWithBaseURL cria um cliente PNCP apontando para uma URL customizada (útil para testes).
func NewPNCPClientWithBaseURL(baseURL string, httpClient *http.Client) *PNCPClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 12 * time.Second}
	}
	return &PNCPClient{
		httpClient: httpClient,
		baseURL:    strings.TrimRight(baseURL, "/"),
	}
}

type pncpContratacaoRaw struct {
	AnoCompra          int     `json:"anoCompra"`
	NumeroControlePNCP string  `json:"numeroControlePNCP"`
	ObjetoCompra       string  `json:"objetoCompra"`
	ModalidadeNome     string  `json:"modalidadeNome"`
	SituacaoCompraNome string  `json:"situacaoCompraNome"`
	ValorTotalEstimado float64 `json:"valorTotalEstimado"`
	ValorTotalHomolog  float64 `json:"valorTotalHomologado"`
	DataPublicacaoPNCP string  `json:"dataPublicacaoPncp"`
	DataAberturaProp   string  `json:"dataAberturaProposta"`
	LinkPNCP           string  `json:"linkPncp"`
}

type pncpContratoRaw struct {
	NumeroContratoEmpenho string  `json:"numeroContratoEmpenho"`
	AnoContrato           int     `json:"anoContrato"`
	ObjetoContrato        string  `json:"objetoContrato"`
	ValorInicial          float64 `json:"valorInicial"`
	ValorGlobal           float64 `json:"valorGlobal"`
	DataVigenciaInicio    string  `json:"dataVigenciaInicio"`
	DataVigenciaFim       string  `json:"dataVigenciaFim"`
	DataAssinatura        string  `json:"dataAssinatura"`
	NomeStatus            string  `json:"nomeStatus"`
	NiFornecedor          string  `json:"niFornecedor"`
	NomeRazaoFornecedor   string  `json:"nomeRazaoSocialFornecedor"`
	Fornecedor            *struct {
		CNPJ        string `json:"cnpj"`
		CPFCNPJ     string `json:"cpfCnpj"`
		RazaoSocial string `json:"razaoSocial"`
		NomeRazao   string `json:"nomeRazaoSocial"`
	} `json:"fornecedor"`
}

func formatDateToYYYYMMDD(dateISO string) string {
	clean := strings.TrimSpace(dateISO)
	clean = strings.ReplaceAll(clean, "-", "")
	if len(clean) == 8 {
		return clean
	}
	return ""
}

// FetchContratacoesByCNPJ consulta publicações no PNCP filtradas pelo CNPJ da Prefeitura.
func (c *PNCPClient) FetchContratacoesByCNPJ(ctx context.Context, m *Municipio, inicioISO, fimISO string, ano, modalidade, limit int) (*LicitacoesResult, error) {
	cnpj := cleanDigits(m.CNPJ)
	if cnpj == "" {
		return &LicitacoesResult{
			Municipio:  m,
			Fonte:      "PNCP",
			Plataforma: "pncp",
			Licitacoes: []LicitacaoItem{},
			Disponivel: false,
			Mensagem:   "Município sem CNPJ cadastrado para consulta no PNCP.",
		}, nil
	}

	limit = normalizeLimit(limit)
	if limit < 10 {
		limit = 10 // tamanhoPagina mínimo exigido pela API do PNCP é 10
	}
	if limit > 50 {
		limit = 50
	}

	dataIni := formatDateToYYYYMMDD(inicioISO)
	dataFim := formatDateToYYYYMMDD(fimISO)
	if dataIni == "" || dataFim == "" {
		if ano > 0 {
			dataIni = fmt.Sprintf("%04d0101", ano)
			dataFim = fmt.Sprintf("%04d1231", ano)
		} else {
			now := time.Now()
			dataFim = now.Format("20060102")
			dataIni = now.AddDate(0, 0, -90).Format("20060102")
		}
	}

	modalidades := []int{modalidade}
	if modalidade <= 0 {
		modalidades = []int{6, 8, 9} // Pregão Eletrônico, Dispensa, Inexigibilidade
	}

	items := make([]LicitacaoItem, 0, limit)
	totalReg := 0

	for _, mod := range modalidades {
		u := fmt.Sprintf(
			"%s/contratacoes/publicacao?dataInicial=%s&dataFinal=%s&codigoModalidadeContratacao=%d&cnpj=%s&pagina=1&tamanhoPagina=%d",
			c.baseURL, dataIni, dataFim, mod, cnpj, limit,
		)
		body, status, err := doJSONGet(ctx, c.httpClient, u)
		if err != nil {
			continue
		}
		if status != http.StatusOK || len(strings.TrimSpace(string(body))) == 0 {
			continue
		}
		var page struct {
			Data           []pncpContratacaoRaw `json:"data"`
			TotalRegistros int                  `json:"totalRegistros"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			continue
		}
		totalReg += page.TotalRegistros
		for _, r := range page.Data {
			dt := r.DataAberturaProp
			if dt == "" {
				dt = r.DataPublicacaoPNCP
			}
			items = append(items, LicitacaoItem{
				Numero:          r.NumeroControlePNCP,
				Ano:             r.AnoCompra,
				Modalidade:      r.ModalidadeNome,
				Situacao:        r.SituacaoCompraNome,
				DataAbertura:    dt,
				Objeto:          r.ObjetoCompra,
				ValorEstimado:   r.ValorTotalEstimado,
				ValorHomologado: r.ValorTotalHomolog,
				Link:            r.LinkPNCP,
				Fonte:           "PNCP (Nacional)",
			})
			if len(items) >= limit {
				break
			}
		}
		if len(items) >= limit {
			break
		}
	}

	return &LicitacoesResult{
		Municipio:      m,
		Fonte:          "PNCP (Portal Nacional de Contratações Públicas)",
		Plataforma:     "pncp",
		PortalURL:      fmt.Sprintf("https://pncp.gov.br/app/editais?q=%s", cnpj),
		TotalRegistros: totalReg,
		Licitacoes:     items,
		Disponivel:     true,
	}, nil
}

// FetchContratosByCNPJ consulta contratos no PNCP filtrados pelo CNPJ da Prefeitura.
func (c *PNCPClient) FetchContratosByCNPJ(ctx context.Context, m *Municipio, inicioISO, fimISO string, ano, limit int) (*ContratosResult, error) {
	cnpj := cleanDigits(m.CNPJ)
	if cnpj == "" {
		return &ContratosResult{
			Municipio:  m,
			Fonte:      "PNCP",
			Plataforma: "pncp",
			Contratos:  []ContratoItem{},
			Disponivel: false,
			Mensagem:   "Município sem CNPJ cadastrado para consulta no PNCP.",
		}, nil
	}

	limit = normalizeLimit(limit)
	if limit < 10 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	dataIni := formatDateToYYYYMMDD(inicioISO)
	dataFim := formatDateToYYYYMMDD(fimISO)
	if dataIni == "" || dataFim == "" {
		if ano > 0 {
			dataIni = fmt.Sprintf("%04d0101", ano)
			dataFim = fmt.Sprintf("%04d1231", ano)
		} else {
			now := time.Now()
			dataFim = now.Format("20060102")
			dataIni = now.AddDate(0, 0, -90).Format("20060102")
		}
	}

	u := fmt.Sprintf(
		"%s/contratos?dataInicial=%s&dataFinal=%s&cnpjOrgao=%s&pagina=1&tamanhoPagina=%d",
		c.baseURL, dataIni, dataFim, cnpj, limit,
	)
	body, status, err := doJSONGet(ctx, c.httpClient, u)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNoContent || len(strings.TrimSpace(string(body))) == 0 {
		return &ContratosResult{
			Municipio:  m,
			Fonte:      "PNCP (Portal Nacional de Contratações Públicas)",
			Plataforma: "pncp",
			PortalURL:  fmt.Sprintf("https://pncp.gov.br/app/contratos?q=%s", cnpj),
			Contratos:  []ContratoItem{},
			Disponivel: true,
		}, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("PNCP /contratos retornou HTTP %d", status)
	}

	var page struct {
		Data           []pncpContratoRaw `json:"data"`
		TotalRegistros int               `json:"totalRegistros"`
	}
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, err
	}

	items := make([]ContratoItem, 0, len(page.Data))
	for _, r := range page.Data {
		forn := r.NomeRazaoFornecedor
		doc := r.NiFornecedor
		if r.Fornecedor != nil {
			if forn == "" {
				forn = r.Fornecedor.RazaoSocial
				if forn == "" {
					forn = r.Fornecedor.NomeRazao
				}
			}
			if doc == "" {
				doc = r.Fornecedor.CNPJ
				if doc == "" {
					doc = r.Fornecedor.CPFCNPJ
				}
			}
		}
		val := r.ValorGlobal
		if val == 0 {
			val = r.ValorInicial
		}
		items = append(items, ContratoItem{
			Numero:              r.NumeroContratoEmpenho,
			Ano:                 r.AnoContrato,
			Fornecedor:          forn,
			DocumentoFornecedor: maskIfCPF(doc),
			Objeto:              r.ObjetoContrato,
			Situacao:            r.NomeStatus,
			DataAssinatura:      r.DataAssinatura,
			VigenciaInicio:      r.DataVigenciaInicio,
			VigenciaFim:         r.DataVigenciaFim,
			Valor:               val,
			Fonte:               "PNCP (Nacional)",
		})
	}

	return &ContratosResult{
		Municipio:      m,
		Fonte:          "PNCP (Portal Nacional de Contratações Públicas)",
		Plataforma:     "pncp",
		PortalURL:      fmt.Sprintf("https://pncp.gov.br/app/contratos?q=%s", cnpj),
		TotalRegistros: page.TotalRegistros,
		Contratos:      items,
		Disponivel:     true,
	}, nil
}
