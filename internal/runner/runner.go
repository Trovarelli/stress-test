package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/jtnet/stress-test/internal/report"
)

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Config struct {
	URL         string
	Requests    int
	Concurrency int
	Timeout     time.Duration
}

func (c *Config) Validate() error {
	if c.URL == "" {
		return errors.New("a URL do serviço (--url) é obrigatória")
	}

	parsedURL, err := url.ParseRequestURI(c.URL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return fmt.Errorf("URL inválida '%s': deve incluir esquema http:// ou https://", c.URL)
	}

	if c.Requests <= 0 {
		return errors.New("o número de requisições (--requests) deve ser maior que 0")
	}

	if c.Concurrency <= 0 {
		return errors.New("a concorrência (--concurrency) deve ser maior que 0")
	}

	if c.Concurrency > c.Requests {
		c.Concurrency = c.Requests
	}

	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}

	return nil
}

type Runner struct {
	client HTTPDoer
}

func NewRunner(client HTTPDoer) *Runner {
	return &Runner{
		client: client,
	}
}

type result struct {
	statusCode int
	err        error
}

func (r *Runner) Run(ctx context.Context, cfg Config) (*report.Report, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	client := r.client
	if client == nil {
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          cfg.Concurrency * 2,
			MaxIdleConnsPerHost:   cfg.Concurrency * 2,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
		client = &http.Client{
			Transport: transport,
			Timeout:   cfg.Timeout,
		}
	}

	rep := report.NewReport(cfg.URL, cfg.Requests, cfg.Concurrency)

	jobs := make(chan struct{}, cfg.Requests)
	for i := 0; i < cfg.Requests; i++ {
		jobs <- struct{}{}
	}
	close(jobs)

	results := make(chan result, cfg.Requests)

	var wg sync.WaitGroup
	startTime := time.Now()

	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				if ctx.Err() != nil {
					return
				}

				req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL, nil)
				if err != nil {
					results <- result{err: err}
					continue
				}

				req.Header.Set("User-Agent", "Go-StressTest-CLI/1.0")

				resp, err := client.Do(req)
				if err != nil {
					results <- result{err: err}
					continue
				}

				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()

				results <- result{statusCode: resp.StatusCode}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {
		rep.RecordResult(res.statusCode, res.err)
	}

	rep.TotalDuration = time.Since(startTime)

	return rep, nil
}
