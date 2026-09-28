# Desafio Técnico: Stress Test - Pós Go Expert

Projeto desenvolvido como parte dos desafios práticos da **Pós Go Expert** (Full Cycle).

O objetivo é implementar uma CLI em Go para realizar testes de carga em serviços web HTTP/HTTPS. A aplicação recebe a URL alvo, a quantidade total de requisições e a concorrência desejada via linha de comando, executa os disparos concorrentes e apresenta um relatório consolidado no console.

---

## Parâmetros da CLI

| Parâmetro | Tipo | Obrigatório | Descrição |
| :--- | :--- | :---: | :--- |
| `--url` | string | Sim | URL do serviço a ser testado (http:// ou https://) |
| `--requests` | int | Sim | Número total de requisições a serem realizadas |
| `--concurrency` | int | Sim | Número de chamadas simultâneas (goroutines concorrentes) |
| `--timeout` | duration | Não | Timeout por requisição (padrão: 30s) |

---

## Execução com Docker

A aplicação foi estruturada em build multi-stage gerando uma imagem estática baseada em `scratch` com certificados CA instalados.

### 1. Build da imagem

```bash
docker build -t stress-test .
```

### 2. Execução do teste

```bash
docker run --rm stress-test --url=http://google.com --requests=1000 --concurrency=10
```

Outro exemplo:

```bash
docker run --rm stress-test --url=https://httpbin.org/get --requests=500 --concurrency=20
```

---

## Execução Local (sem Docker)

### Requisitos

- Go 1.23+ instalado

### Com `go run`

```bash
go run ./cmd/stresstest --url=http://google.com --requests=1000 --concurrency=10
```

### Compilação do binário

```bash
go build -ldflags="-w -s" -o stresstest ./cmd/stresstest
./stresstest --url=http://google.com --requests=1000 --concurrency=10
```

*(No Windows: `.\stresstest.exe --url=http://google.com --requests=1000 --concurrency=10`)*

---

## Formato do Relatório

Ao término da execução, a CLI exibe no terminal:

- Tempo total gasto na execução
- Quantidade total de requisições executadas
- Quantidade de requisições com status HTTP 200
- Distribuição dos demais códigos de status HTTP (404, 500, etc.)
- Falhas de rede ou conexão (se houver)

Exemplo de saída:

```text
====================================================================
                   RELATÓRIO DO TESTE DE CARGA
====================================================================
  URL do Serviço                 : http://google.com
  Concorrência (Workers)         : 10
  Requisições Solicitadas        : 1000
  Requisições Executadas         : 1000
  Tempo Total de Execução        : 2.145s
  Vazão (Taxa Média)             : 466.20 reqs/s
--------------------------------------------------------------------
  Requisições com Status 200 (OK) : 1000
--------------------------------------------------------------------
  Distribuição de Outros Códigos de Status HTTP:
    Nenhum outro código HTTP retornado.
====================================================================
```

Exemplo quando ocorrem status diferentes de 200:

```text
====================================================================
                   RELATÓRIO DO TESTE DE CARGA
====================================================================
  URL do Serviço                 : http://api.exemplo.com/recurso
  Concorrência (Workers)         : 20
  Requisições Solicitadas        : 100
  Requisições Executadas         : 100
  Tempo Total de Execução        : 1.830s
  Vazão (Taxa Média)             : 54.64 reqs/s
--------------------------------------------------------------------
  Requisições com Status 200 (OK) : 70
--------------------------------------------------------------------
  Distribuição de Outros Códigos de Status HTTP:
    [HTTP 404 - Not Found]             : 20
    [HTTP 500 - Internal Server Error] : 10
====================================================================
```

---

## Testes Automatizados

Os testes cobrem validações de configuração, garantia da contagem exata de requisições, cenários de alta concorrência com servidores mock (`httptest`), distribuição de status codes e cancelamento por contexto.

Para rodar todos os testes:

```bash
go test -v ./...
```

Para verificar a cobertura:

```bash
go test -cover ./...
```

---

## Estrutura do Projeto

```text
stress-test/
├── cmd/
│   └── stresstest/
│       └── main.go          # CLI: parsing de flags, validação e tratamento de sinais
├── internal/
│   ├── runner/
│   │   ├── runner.go        # Worker pool, controle de concorrência e cliente HTTP
│   │   └── runner_test.go   # Testes unitários do runner
│   └── report/
│       ├── report.go        # Agregação de resultados e formatação do relatório
│       └── report_test.go   # Testes unitários do relatório
├── .dockerignore
├── .gitignore
├── Dockerfile
├── go.mod
└── README.md
```

### Detalhes de Implementação

- **Worker Pool:** O runner utiliza um canal de tarefas com buffer igual ao número total de requisições (`--requests`). Um pool de goroutines de tamanho fixo definido por `--concurrency` consome as tarefas em paralelo.
- **Conexões HTTP:** Configuração customizada de `http.Transport` com `MaxIdleConns` e `MaxIdleConnsPerHost` adequados ao nível de concorrência, além do consumo completo e fechamento de `resp.Body` para reutilização de conexões TCP (Keep-Alive).
- **Graceful Shutdown:** Interrupção controlada via `signal.NotifyContext` (`SIGINT` / `SIGTERM`), permitindo a geração de relatório parcial caso o teste seja cancelado antes do término.
- **Sem dependências externas:** O projeto utiliza apenas os pacotes da biblioteca padrão do Go.
