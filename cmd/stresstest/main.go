package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jtnet/stress-test/internal/runner"
)

func main() {
	urlFlag := flag.String("url", "", "URL do serviço a ser testado")
	requestsFlag := flag.Int("requests", 0, "Número total de requisições a serem realizadas")
	concurrencyFlag := flag.Int("concurrency", 1, "Número de chamadas simultâneas")
	timeoutFlag := flag.Duration("timeout", 30*time.Second, "Tempo limite para cada requisição HTTP (opcional)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Uso do Stress Test CLI:\n")
		fmt.Fprintf(os.Stderr, "  stresstest --url=<url> --requests=<num> --concurrency=<num>\n\n")
		fmt.Fprintf(os.Stderr, "Parâmetros disponíveis:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExemplo de uso local:\n")
		fmt.Fprintf(os.Stderr, "  go run cmd/stresstest/main.go --url=http://google.com --requests=1000 --concurrency=10\n\n")
		fmt.Fprintf(os.Stderr, "Exemplo de uso via Docker:\n")
		fmt.Fprintf(os.Stderr, "  docker run <sua-imagem-docker> --url=http://google.com --requests=1000 --concurrency=10\n\n")
	}

	flag.Parse()

	cfg := runner.Config{
		URL:         *urlFlag,
		Requests:    *requestsFlag,
		Concurrency: *concurrencyFlag,
		Timeout:     *timeoutFlag,
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "\n[ERRO DE CONFIGURAÇÃO] %v\n\n", err)
		flag.Usage()
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Println()
	fmt.Println("====================================================================")
	fmt.Println("               INICIANDO TESTE DE CARGA (STRESS TEST)               ")
	fmt.Println("====================================================================")
	fmt.Printf("  Alvo         : %s\n", cfg.URL)
	fmt.Printf("  Requisições  : %d\n", cfg.Requests)
	fmt.Printf("  Concorrência : %d\n", cfg.Concurrency)
	fmt.Printf("  Timeout/Req  : %v\n", cfg.Timeout)
	fmt.Println("--------------------------------------------------------------------")
	fmt.Println("Executando chamadas simultâneas, por favor aguarde...")

	r := runner.NewRunner(nil)
	rep, err := r.Run(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n[FALHA NA EXECUÇÃO] %v\n", err)
		os.Exit(1)
	}

	// ctx.Err() is checked after Run() returns — at this point the context
	// may already be Done if the user pressed Ctrl+C during execution.
	if ctx.Err() != nil {
		fmt.Println("\n[AVISO] Teste interrompido antes do término. Exibindo métricas parciais...")
	}

	rep.Render(os.Stdout)
}
