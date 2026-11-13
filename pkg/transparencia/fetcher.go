package transparencia

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// Fetcher executa verificações de integridade e consultas em lote aos portais de transparência
// com alto paralelismo via goroutines.
type Fetcher struct {
	client *http.Client
}

// NewFetcher cria um novo fetcher com timeout e keep-alive configurados.
func NewFetcher(timeout time.Duration) *Fetcher {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	tr := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     30 * time.Second,
	}
	return &Fetcher{
		client: &http.Client{
			Transport: tr,
			Timeout:   timeout,
		},
	}
}

// Probe verifica o status de resposta de uma URL de transparência.
func (f *Fetcher) Probe(ctx context.Context, alvo, url string) ProbeResult {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return ProbeResult{
			Alvo:    alvo,
			URL:     url,
			TempoMs: time.Since(start).Milliseconds(),
			Online:  false,
			Erro:    err.Error(),
		}
	}
	req.Header.Set("User-Agent", "mcp-brasil/1.0 (Go engine; Transparencia Nacional)")

	resp, err := f.client.Do(req)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		// Fallback para GET se HEAD falhar (alguns servidores bloqueiam HEAD)
		reqGet, errGet := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if errGet == nil {
			reqGet.Header.Set("User-Agent", "mcp-brasil/1.0 (Go engine; Transparencia Nacional)")
			respGet, errGet2 := f.client.Do(reqGet)
			if errGet2 == nil {
				defer respGet.Body.Close()
				return ProbeResult{
					Alvo:       alvo,
					URL:        url,
					StatusHTTP: respGet.StatusCode,
					TempoMs:    time.Since(start).Milliseconds(),
					Online:     respGet.StatusCode < 400 || respGet.StatusCode == 403, // 403 é comum em portais com WAF ativo
				}
			}
		}
		return ProbeResult{
			Alvo:    alvo,
			URL:     url,
			TempoMs: elapsed,
			Online:  false,
			Erro:    err.Error(),
		}
	}
	defer resp.Body.Close()

	return ProbeResult{
		Alvo:       alvo,
		URL:        url,
		StatusHTTP: resp.StatusCode,
		TempoMs:    elapsed,
		Online:     resp.StatusCode < 400 || resp.StatusCode == 403,
	}
}

// ProbeMany testa múltiplos portais de transparência em paralelo com worker pool.
func (f *Fetcher) ProbeMany(ctx context.Context, alvos map[string]string, maxWorkers int) []ProbeResult {
	if maxWorkers <= 0 {
		maxWorkers = 10
	}

	type task struct {
		alvo string
		url  string
	}

	jobs := make(chan task, len(alvos))
	results := make(chan ProbeResult, len(alvos))

	var wg sync.WaitGroup
	for w := 0; w < maxWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
					res := f.Probe(ctx, j.alvo, j.url)
					results <- res
				}
			}
		}()
	}

	for alvo, url := range alvos {
		jobs <- task{alvo: alvo, url: url}
	}
	close(jobs)

	wg.Wait()
	close(results)

	out := make([]ProbeResult, 0, len(alvos))
	for r := range results {
		out = append(out, r)
	}
	return out
}
