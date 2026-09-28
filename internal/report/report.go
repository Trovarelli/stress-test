package report

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type Report struct {
	URL              string
	TargetRequests   int
	TotalRequests    int
	Concurrency      int
	TotalDuration    time.Duration
	Status200Count   int
	OtherStatusCodes map[int]int
	NetworkErrors    int
}

func NewReport(url string, targetRequests, concurrency int) *Report {
	return &Report{
		URL:              url,
		TargetRequests:   targetRequests,
		Concurrency:      concurrency,
		OtherStatusCodes: make(map[int]int),
	}
}

func (r *Report) RecordResult(statusCode int, err error) {
	r.TotalRequests++
	if err != nil {
		r.NetworkErrors++
		return
	}

	if statusCode == http.StatusOK {
		r.Status200Count++
	} else {
		r.OtherStatusCodes[statusCode]++
	}
}

func (r *Report) Render(w io.Writer) {
	border := strings.Repeat("=", 68)
	divider := strings.Repeat("-", 68)

	var rps float64
	if r.TotalDuration.Seconds() > 0 {
		rps = float64(r.TotalRequests) / r.TotalDuration.Seconds()
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, border)
	fmt.Fprintln(w, "                   RELATÓRIO DO TESTE DE CARGA")
	fmt.Fprintln(w, border)
	fmt.Fprintf(w, "  %-30s : %s\n", "URL do Serviço", r.URL)
	fmt.Fprintf(w, "  %-30s : %d\n", "Concorrência (Workers)", r.Concurrency)
	fmt.Fprintf(w, "  %-30s : %d\n", "Requisições Solicitadas", r.TargetRequests)
	fmt.Fprintf(w, "  %-30s : %d\n", "Requisições Executadas", r.TotalRequests)
	fmt.Fprintf(w, "  %-30s : %v\n", "Tempo Total de Execução", r.TotalDuration.Round(time.Millisecond))
	fmt.Fprintf(w, "  %-30s : %.2f reqs/s\n", "Vazão (Taxa Média)", rps)
	fmt.Fprintln(w, divider)
	fmt.Fprintf(w, "  %-30s : %d\n", "Requisições com Status 200 (OK)", r.Status200Count)
	fmt.Fprintln(w, divider)

	fmt.Fprintln(w, "  Distribuição de Outros Códigos de Status HTTP:")
	if len(r.OtherStatusCodes) == 0 {
		fmt.Fprintln(w, "    Nenhum outro código HTTP retornado.")
	} else {
		keys := make([]int, 0, len(r.OtherStatusCodes))
		for code := range r.OtherStatusCodes {
			keys = append(keys, code)
		}
		sort.Ints(keys)

		for _, code := range keys {
			count := r.OtherStatusCodes[code]
			statusText := http.StatusText(code)
			if statusText == "" {
				statusText = "Unknown Status"
			}
			fmt.Fprintf(w, "    [HTTP %d - %s] : %d\n", code, statusText, count)
		}
	}

	if r.NetworkErrors > 0 {
		fmt.Fprintln(w, divider)
		fmt.Fprintf(w, "  %-30s : %d\n", "Falhas de Rede/Conexão", r.NetworkErrors)
	}

	fmt.Fprintln(w, border)
	fmt.Fprintln(w)
}
