package transparencia

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	siconfiBaseURL = "https://apidatalake.tesouro.gov.br/ords/siconfi/tt"
)

// SiconfiClient é o cliente HTTP nativo em Go para a API pública do Tesouro Nacional (SICONFI).
type SiconfiClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewSiconfiClient cria uma nova instância de cliente SICONFI.
func NewSiconfiClient() *SiconfiClient {
	return &SiconfiClient{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		baseURL: siconfiBaseURL,
	}
}

// NewSiconfiClientWithBaseURL cria um cliente SICONFI com URL base customizada (útil para testes).
func NewSiconfiClientWithBaseURL(baseURL string, httpClient *http.Client) *SiconfiClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &SiconfiClient{
		httpClient: httpClient,
		baseURL:    baseURL,
	}
}

type siconfiFiscalResponse struct {
	Items []FiscalItem `json:"items"`
}

type siconfiEntregasResponse struct {
	Items []EntregaItem `json:"items"`
}

// FetchDCA busca os dados da Declaração de Contas Anuais (Balanço Patrimonial e Orçamentário) no SICONFI.
func (c *SiconfiClient) FetchDCA(ctx context.Context, idEnte string, ano int) ([]FiscalItem, error) {
	if ano <= 0 {
		ano = time.Now().Year() - 1
	}

	url := fmt.Sprintf("%s/dca?an_exercicio=%d&id_ente=%s", c.baseURL, ano, idEnte)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição DCA: %w", err)
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha ao consultar DCA no SICONFI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("SICONFI retornou status %d: %s", resp.StatusCode, string(body))
	}

	var res siconfiFiscalResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("falha ao decodificar resposta JSON DCA: %w", err)
	}

	return res.Items, nil
}

// FetchRREO busca o Relatório Resumido da Execução Orçamentária bimestral no SICONFI.
func (c *SiconfiClient) FetchRREO(ctx context.Context, idEnte string, ano int, periodo int) ([]FiscalItem, error) {
	if ano <= 0 {
		ano = time.Now().Year()
	}
	if periodo <= 0 {
		periodo = 1
	}

	url := fmt.Sprintf("%s/rreo?an_exercicio=%d&nr_periodo=%d&co_tipo_demonstrativo=RREO&id_ente=%s",
		c.baseURL, ano, periodo, idEnte)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição RREO: %w", err)
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha ao consultar RREO no SICONFI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("SICONFI retornou status %d: %s", resp.StatusCode, string(body))
	}

	var res siconfiFiscalResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("falha ao decodificar resposta JSON RREO: %w", err)
	}

	return res.Items, nil
}

// FetchRGF busca o Relatório de Gestão Fiscal (Despesa com Pessoal LRF, Dívida e Limites) no SICONFI.
func (c *SiconfiClient) FetchRGF(ctx context.Context, idEnte string, ano int, periodicidade string, periodo int) ([]FiscalItem, error) {
	if ano <= 0 {
		ano = time.Now().Year() - 1
	}

	type attempt struct {
		per string
		nr  int
	}
	var attempts []attempt
	if periodicidade != "" {
		nr := periodo
		if nr <= 0 {
			nr = 1
		}
		attempts = append(attempts, attempt{per: periodicidade, nr: nr})
	} else {
		attempts = []attempt{
			{per: "S", nr: 2},
			{per: "Q", nr: 3},
			{per: "Q", nr: 1},
			{per: "S", nr: 1},
		}
	}

	var lastErr error
	for _, a := range attempts {
		url := fmt.Sprintf("%s/rgf?an_exercicio=%d&in_periodicidade=%s&nr_periodo=%d&co_tipo_demonstrativo=RGF&co_poder=E&id_ente=%s",
			c.baseURL, ano, a.per, a.nr, idEnte)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}
		var res siconfiFiscalResponse
		decErr := json.NewDecoder(resp.Body).Decode(&res)
		resp.Body.Close()
		if decErr == nil && len(res.Items) > 0 {
			return res.Items, nil
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("falha ao consultar RGF no SICONFI: %w", lastErr)
	}
	return []FiscalItem{}, nil
}

// FetchEntregas busca o status de entrega dos relatórios fiscais obrigatórios pela LRF.
func (c *SiconfiClient) FetchEntregas(ctx context.Context, idEnte string, ano int) ([]EntregaItem, error) {
	if ano <= 0 {
		ano = time.Now().Year()
	}

	url := fmt.Sprintf("%s/extrato_entregas?an_referencia=%d&id_ente=%s", c.baseURL, ano, idEnte)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição Entregas: %w", err)
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (transparencia-go)")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha ao consultar Extrato de Entregas no SICONFI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("SICONFI retornou status %d: %s", resp.StatusCode, string(body))
	}

	var res siconfiEntregasResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("falha ao decodificar resposta JSON Entregas: %w", err)
	}

	return res.Items, nil
}
