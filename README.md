# Olivia Conciliation

Sistema pessoal de conciliação financeira entre transações parceladas importadas do Pluggy e uma planilha de controle no Google Sheets.

O problema: a cada importação do Pluggy chegam parcelas de cartão que precisam ser vinculadas a lançamentos já registrados na planilha — seja por entrada manual do usuário, seja por parcelas sintéticas geradas em importações anteriores. O Olivia cruza os dois lados, exibe os candidatos a conciliação e deixa o usuário aceitar ou rejeitar cada vínculo.

## Documentação

- [`CONTEXT.md`](CONTEXT.md) — glossário de domínio (Transação, HOM, DIF, ES, REJ, Conciliação…)
- [`docs/adr/`](docs/adr/) — decisões de arquitetura (ADRs)
- [`CLAUDE.md`](CLAUDE.md) — guia técnico do repositório (comandos, arquitetura, endpoints)

## Stack

| Camada | Tecnologia |
| :--- | :--- |
| Backend | Go |
| Frontend | Vanilla JS |
| Proxy | Nginx |
| Planilha | Google Sheets (API v4) |
| Banco de dados | — (a planilha é o banco) |
| Importação | Pluggy via `olivia-api` |

## Como funciona

O `olivia-api` consome o Pluggy e popula a aba **HOM** no Google Sheets. Uma fórmula nativa da planilha gera a aba **DIF** com as transações que ainda não têm correspondência na **ES** (Entradas e Saídas). O backend Go lê a DIF, compara com candidatas na ES e expõe endpoints REST para aceitar ou rejeitar cada conciliação. O frontend é uma interface operacional sobre esses endpoints.

## Pré-requisitos

- Conta no **Pluggy** com credenciais de API
- **Google Sheets** configurado com as abas ES, DIF, HOM e REJ, e uma service account com permissão de escrita
- Imagem `olivia-api` — pode ser construída a partir do projeto [ciroprates/olivia](https://github.com/ciroprates/olivia)

## Rodando

Em qualquer um dos caminhos, você precisa antes:

- Um `.env` (`cp .env.example .env`) com os valores obrigatórios preenchidos — em especial o `SHEET_SPREADSHEET_ID`.
- A chave da service account do Google como `key.json` na raiz (o `.env` aponta `GOOGLE_APPLICATION_CREDENTIALS=key.json`).

> ⚠️ O `SHEET_SPREADSHEET_ID` é o da planilha real. Rodando local, a conciliação (aceitar/rejeitar) **escreve na planilha de produção** (abas ES/REJ). Use uma planilha de teste se não quiser tocar nos dados reais.

### Stack completa (Docker)

```bash
docker compose up -d
```

O nginx sobe como proxy e serve tudo same-origin em `http://localhost` (`/api` → backend, `/executions` → `olivia-api`).

### Sem Docker (backend + frontend)

O frontend usa caminhos relativos (`/api`, `/executions`) e o backend não emite CORS, então o frontend precisa ser servido na **mesma origem** que a API. O helper `scripts/dev-proxy.go` faz esse papel (equivalente ao nginx):

```bash
# defina no .env: APP_ORIGIN=http://localhost:3001, COOKIE_SECURE=false, COOKIE_DOMAIN=
go run backend/main.go            # backend em :8080
go run scripts/dev-proxy.go       # frontend + proxy /api,/executions em :3001
```

Abra `http://localhost:3001`. O `/executions` aponta para `:3000` (o `olivia-api` local) por padrão — veja o cabeçalho de `scripts/dev-proxy.go` para outras opções.

## Variáveis de ambiente

Veja `.env.example`. As variáveis obrigatórias estão marcadas.

## CI/CD

Push para `main` dispara `.github/workflows/ecr-push.yml`, que constrói as imagens Docker, publica no ECR e faz deploy em EC2 via AWS SSM. Variáveis e secrets necessários estão documentados no próprio workflow.
