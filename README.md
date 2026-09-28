# Stress Test CLI

Ferramenta de linha de comando (CLI) desenvolvida em **Go** para realizar testes de carga e estresse em serviços web.

A ferramenta permite especificar a URL alvo, o volume total de requisições e a quantidade de chamadas simultâneas (concorrência). Ao término do processamento, um relatório detalhado e estruturado é impresso no console.

---

## 📋 Requisitos e Parâmetros

A aplicação aceita os seguintes parâmetros de linha de comando:

| Parâmetro | Tipo | Obrigatório | Descrição |
| :--- | :--- | :---: | :--- |
| `--url` | `string` | **Sim** | URL completa do serviço a ser testado (deve iniciar com `http://` ou `https://`). |
| `--requests` | `int` | **Sim** | Quantidade total de requisições a serem realizadas (`> 0`). |
| `--concurrency` | `int` | **Sim** | Número de chamadas simultâneas / workers concorrentes (`> 0`). |
| `--timeout` | `duration` | Não | Tempo limite máximo por requisição HTTP (padrão: `30s`, ex: `10s`, `1m`). |

---

## 🚀 Execução via Docker (Obrigatório)

A aplicação foi empacotada em uma imagem Docker minimalista multi-stage (baseada em `scratch`), pesando apenas **~10MB** e contendo os certificados raiz de autoridade certificadora (CA) para chamadas HTTPS seguras.

### 1. Construir a Imagem Docker

No diretório raiz do projeto, execute:

```bash
docker build -t stress-test .
```

### 2. Executar o Teste de Carga

Execute o container passando os parâmetros desejados. O formato segue o padrão estipulado:

```bash
docker run --rm stress-test --url=http://google.com --requests=1000 --concurrency=10
```

> **Nota:** A flag `--rm` garante que o container seja removido após o término da execução.

### Exemplos Adicionais:

- **Teste com HTTPS:**
  ```bash
  docker run --rm stress-test --url=https://httpbin.org/get --requests=500 --concurrency=20
  ```

- **Exibir ajuda / parâmetros disponíveis:**
  ```bash
  docker run --rm stress-test --help
  ```

---

## 💻 Execução Local (sem Docker)

Caso prefira executar diretamente em seu ambiente Go local:

### Pré-requisitos
- [Go](https://golang.org/) 1.23 ou superior instalado.

### 1. Executar com `go run`

```bash
go run ./cmd/stresstest --url=http://google.com --requests=1000 --concurrency=10
```

### 2. Compilar o Binário

```bash
# Compilar binário local
go build -ldflags="-w -s" -o stresstest ./cmd/stresstest

# Executar o binário compilado
./stresstest --url=http://google.com --requests=1000 --concurrency=10
```

*(No Windows PowerShell: `.\stresstest.exe --url=http://google.com --requests=1000 --concurrency=10`)*

---

## 📊 Relatório de Execução

Ao final do teste, o sistema consolida as métricas e exibe no console um relatório no seguinte formato:

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

### Exemplo com Códigos de Erro / Diversos:

Caso o serviço retorne outros status HTTP (ex: 404, 500) ou haja erros de rede:

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
--------------------------------------------------------------------
  Falhas de Rede/Conexão         : 0
====================================================================
```

---

## 🧪 Testes Automatizados

O projeto conta com cobertura de testes unitários e de integração, utilizando o pacote nativo `net/http/httptest` para simular servidores HTTP, cenários de concorrência, respostas com códigos HTTP variados e cancelamentos graciosos via `context.Context`.

Para executar a suíte completa de testes:

```bash
go test -v ./...
```

Verificar a cobertura de testes:

```bash
go test -cover ./...
```

---

## 🏗️ Arquitetura e Decisões de Projeto

O projeto foi estruturado seguindo as melhores práticas e convenções da comunidade Go:

```text
stress-test/
├── cmd/
│   └── stresstest/
│       └── main.go           # Ponto de entrada da CLI, parsing de flags e tratamento de sinais
├── internal/
│   ├── runner/
│   │   ├── runner.go         # Orquestração do Worker Pool e disparo concorrente das requisições
│   │   └── runner_test.go    # Testes unitários do runner com servidores mock httptest
│   └── report/
│       ├── report.go         # Agregação das métricas e formatação do relatório para o console
│       └── report_test.go    # Testes unitários do relatório
├── .dockerignore             # Arquivos ignorados pelo build Docker
├── .gitignore                # Arquivos ignorados pelo Git
├── Dockerfile                # Configuração multi-stage para construção da imagem
├── go.mod                    # Definição do módulo Go
└── README.md                 # Documentação detalhada
```

### Destaques Técnicos:
1. **Worker Pool Pattern**: As requisições são enfileiradas através de canais (`chan struct{}`) e consumidas por um grupo de goroutines de tamanho fixo determinado pelo parâmetro `--concurrency`.
2. **Garantia de Quantidade Exata**: O canal de tarefas contém exatamente o número especificado em `--requests`, garantindo que nenhuma requisição a mais ou a menos seja disparada.
3. **Otimização do `http.Transport`**: O `http.Client` padrão do Go possui limite de conexões ociosas por host (`MaxIdleConnsPerHost = 2`), o que geraria gargalos severos de socket em testes de alta concorrência. O runner configura um transporte otimizado com `MaxIdleConns` e `MaxIdleConnsPerHost` dimensionados para a concorrência solicitada, além de garantir o esgotamento (`io.Copy(io.Discard, resp.Body)`) e fechamento do body da resposta para reutilização eficiente das conexões TCP (HTTP Keep-Alive).
4. **Encerramento Gracioso (Graceful Shutdown)**: O sistema escuta sinais do sistema operacional (`SIGINT`, `SIGTERM`). Se o usuário pressionar `Ctrl+C`, os workers interrompem novas requisições de forma limpa e um relatório parcial é gerado imediatamente.
5. **Zero Dependências Externas**: Implementado integralmente com a biblioteca padrão do Go (`net/http`, `sync`, `flag`, `time`, `context`), proporcionando binários ultrarrápidos, leves e livres de vulnerabilidades de dependências terceiras.
