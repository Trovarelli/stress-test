package runner_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jtnet/stress-test/internal/runner"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         runner.Config
		wantErr     bool
		expectedErr string
	}{
		{
			name: "Valid HTTP URL",
			cfg: runner.Config{
				URL:         "http://localhost:8080/test",
				Requests:    100,
				Concurrency: 10,
			},
			wantErr: false,
		},
		{
			name: "Valid HTTPS URL",
			cfg: runner.Config{
				URL:         "https://example.com/api",
				Requests:    50,
				Concurrency: 5,
			},
			wantErr: false,
		},
		{
			name: "Empty URL",
			cfg: runner.Config{
				URL:         "",
				Requests:    100,
				Concurrency: 10,
			},
			wantErr:     true,
			expectedErr: "a URL do serviço (--url) é obrigatória",
		},
		{
			name: "URL without scheme",
			cfg: runner.Config{
				URL:         "google.com",
				Requests:    100,
				Concurrency: 10,
			},
			wantErr:     true,
			expectedErr: "deve incluir esquema http:// ou https://",
		},
		{
			name: "Zero requests",
			cfg: runner.Config{
				URL:         "http://example.com",
				Requests:    0,
				Concurrency: 10,
			},
			wantErr:     true,
			expectedErr: "o número de requisições (--requests) deve ser maior que 0",
		},
		{
			name: "Negative requests",
			cfg: runner.Config{
				URL:         "http://example.com",
				Requests:    -5,
				Concurrency: 10,
			},
			wantErr:     true,
			expectedErr: "o número de requisições (--requests) deve ser maior que 0",
		},
		{
			name: "Zero concurrency",
			cfg: runner.Config{
				URL:         "http://example.com",
				Requests:    100,
				Concurrency: 0,
			},
			wantErr:     true,
			expectedErr: "a concorrência (--concurrency) deve ser maior que 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.expectedErr != "" {
				if !testing.Short() && !contains(err.Error(), tt.expectedErr) {
					t.Errorf("expected error containing '%s', got '%s'", tt.expectedErr, err.Error())
				}
			}
		})
	}
}

func TestConfig_Validate_ConcurrencyCappedToRequests(t *testing.T) {
	cfg := runner.Config{
		URL:         "http://example.com",
		Requests:    5,
		Concurrency: 20,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	if cfg.Concurrency != 5 {
		t.Errorf("expected concurrency to be capped at 5, got %d", cfg.Concurrency)
	}
}

func TestRunner_ExactRequestsAndStatus200(t *testing.T) {
	var requestCount int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	r := runner.NewRunner(server.Client())
	totalRequests := 100
	concurrency := 10

	rep, err := r.Run(context.Background(), runner.Config{
		URL:         server.URL,
		Requests:    totalRequests,
		Concurrency: concurrency,
	})

	if err != nil {
		t.Fatalf("unexpected error running stress test: %v", err)
	}

	if int(atomic.LoadInt64(&requestCount)) != totalRequests {
		t.Errorf("server received %d requests, expected %d", requestCount, totalRequests)
	}

	if rep.TotalRequests != totalRequests {
		t.Errorf("rep.TotalRequests = %d, expected %d", rep.TotalRequests, totalRequests)
	}

	if rep.Status200Count != totalRequests {
		t.Errorf("rep.Status200Count = %d, expected %d", rep.Status200Count, totalRequests)
	}

	if len(rep.OtherStatusCodes) != 0 {
		t.Errorf("rep.OtherStatusCodes should be empty, got %+v", rep.OtherStatusCodes)
	}

	if rep.NetworkErrors != 0 {
		t.Errorf("rep.NetworkErrors = %d, expected 0", rep.NetworkErrors)
	}
}

func TestRunner_MixedStatusCodes(t *testing.T) {
	var count int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idx := atomic.AddInt64(&count, 1)
		val := (idx - 1) % 10
		switch {
		case val < 6: // 60% 200 (val 0, 1, 2, 3, 4, 5)
			w.WriteHeader(http.StatusOK)
		case val < 8: // 20% 404 (val 6, 7)
			w.WriteHeader(http.StatusNotFound)
		default: // 20% 500 (val 8, 9)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	r := runner.NewRunner(server.Client())
	totalRequests := 100
	concurrency := 8

	rep, err := r.Run(context.Background(), runner.Config{
		URL:         server.URL,
		Requests:    totalRequests,
		Concurrency: concurrency,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rep.TotalRequests != totalRequests {
		t.Errorf("expected %d total requests, got %d", totalRequests, rep.TotalRequests)
	}

	expected200 := 60
	expected404 := 20
	expected500 := 20

	if rep.Status200Count != expected200 {
		t.Errorf("expected %d 200 OK, got %d", expected200, rep.Status200Count)
	}
	if rep.OtherStatusCodes[http.StatusNotFound] != expected404 {
		t.Errorf("expected %d 404 Not Found, got %d", expected404, rep.OtherStatusCodes[http.StatusNotFound])
	}
	if rep.OtherStatusCodes[http.StatusInternalServerError] != expected500 {
		t.Errorf("expected %d 500 Internal Error, got %d", expected500, rep.OtherStatusCodes[http.StatusInternalServerError])
	}
}

func TestRunner_NetworkErrors(t *testing.T) {
	// Create server and close it immediately to force network errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close()

	r := runner.NewRunner(nil)
	rep, err := r.Run(context.Background(), runner.Config{
		URL:         serverURL,
		Requests:    10,
		Concurrency: 2,
		Timeout:     500 * time.Millisecond,
	})

	if err != nil {
		t.Fatalf("unexpected runner error: %v", err)
	}

	if rep.TotalRequests != 10 {
		t.Errorf("expected 10 total requests, got %d", rep.TotalRequests)
	}

	if rep.NetworkErrors != 10 {
		t.Errorf("expected 10 network errors, got %d", rep.NetworkErrors)
	}
}

func TestRunner_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	r := runner.NewRunner(server.Client())
	rep, err := r.Run(ctx, runner.Config{
		URL:         server.URL,
		Requests:    50,
		Concurrency: 5,
	})

	if err != nil {
		t.Fatalf("unexpected error on context cancel: %v", err)
	}

	// Should not have completed all 50 requests
	if rep.TotalRequests >= 50 {
		t.Errorf("expected fewer than 50 completed requests due to cancellation, got %d", rep.TotalRequests)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || hasSubstr(s, substr))
}

func hasSubstr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
