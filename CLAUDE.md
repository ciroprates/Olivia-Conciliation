# CLAUDE.md

Este arquivo orienta o Claude Code (claude.ai/code) ao trabalhar neste repositório.

> **Vocabulário de domínio:** veja [`CONTEXT.md`](CONTEXT.md) — leia antes de mexer na lógica de negócio (define Transação, HOM, DIF, ES, REJ, Parcelada, Candidata, Conciliação, etc.).
>
> **Decisões de arquitetura:** veja [`docs/adr/`](docs/adr/).

## Comandos

### Backend (Go)

```bash
# Rodar localmente (requer .env configurado)
go run backend/main.go

# Compilar o binário
go build -o olivia-backend ./backend

# Rodar os testes
go test ./backend/...

# Rodar os testes de um único pacote
go test ./backend/service/...

# Ajustar dependências
go mod tidy
```

### Frontend (Vanilla JS)

```bash
# Servir os arquivos estáticos localmente
cd frontend
python3 -m http.server 3001
```

### Docker

```bash
# Stack completa (recomendado)
cp .env.example .env
# preencha os valores obrigatórios
docker compose up -d

# Recompilar e reiniciar um único serviço
docker compose up -d --build backend
```

## Arquitetura

Sistema de conciliação financeira entre o Google Sheets e a API do Pluggy. O `olivia-api` importa transações do Pluggy para a aba HOM; uma fórmula nativa do Sheets gera a aba DIF com o que ainda não tem correspondência na ES. O backend Go lê a DIF e expõe dois fluxos: conciliação de transações **parceladas** (casar com candidatas na ES) e revisão manual de transações **não-parceladas** (mover direto para ES ou REJ).

**Fluxo dos dados:** Pluggy → `olivia-api` → HOM (import bruto, volátil) → fórmula do Sheets → DIF → `backend` (Go) → `sheets.Client` → Google Sheets API.

### Serviços (docker-compose)

| Serviço | Descrição |
|---|---|
| `backend` | API HTTP em Go, porta 8080 |
| `frontend` | Nginx servindo o Vanilla JS estático |
| `nginx` | Proxy reverso (portas 80/443), roteia `/api` e `/executions` |
| `olivia-api` | Serviço externo (imagem ECR) — integração com o Pluggy |
| `n8n` | Automação de workflows |
| `waha` | Ponte para a API do WhatsApp |

### Estrutura do backend (módulo Go `olivia-conciliation`)

```
backend/
  main.go              # Servidor HTTP, registro de rotas, validação de startup
  models/models.go     # Tipos de domínio e constantes de índice de coluna da planilha
  sheets/client.go     # Cliente da API do Google Sheets (FetchRows, WriteCell, AppendRow, ClearRow)
  handlers/auth.go     # Login, Logout, Verify, AuthMiddleware (JWT + CSRF)
  handlers/api.go      # Handlers REST para conciliação e operações da DIF
  service/conciliation.go  # Lógica de negócio: matching, aceitar, rejeitar, mover linhas
```

### Modelo da planilha

Quatro abas, configuráveis via env vars. O fluxo de import (Pluggy → HOM → fórmula → DIF) está descrito em **Arquitetura**, acima.

- **ES** (`SHEET_ES`, padrão `"Entradas e Saídas"`): Transações confirmadas. As pendentes têm `Recorrente=true` e sem `IdParcela`, ou `IdParcela` prefixado por `synthetic`.
- **DIF** (`SHEET_DIF`, padrão `"Diferença"`): Gerada por fórmula a partir da HOM — transações do Pluggy ainda sem correspondência na ES. As linhas parceladas (`Recorrente=true`) são candidatas a matching contra a ES.
- **REJ** (`SHEET_REJ`, padrão `"Rejeitados"`): Transações rejeitadas (trilha de auditoria).
- **HOM** (`SHEET_HOM`, padrão `"Homologação"`): Import bruto e volátil do Pluggy (apagado e reescrito a cada processamento). É a fonte que a fórmula da DIF lê, e onde se edita categoria e data das linhas não-parceladas antes de movê-las. **Não validada no startup** — um `SHEET_HOM` ausente só falha em runtime, quando esses endpoints são chamados.

Layout de colunas (índices conforme `backend/models/models.go`; o slice da linha é 0-based, com a coluna A no índice 0): B=Data (1), C=Descricao (2), D=Valor (3), E=Categoria (4), F=Dono (5), G=Banco (6), H=Conta (7), I=Recorrente (8), J=IdParcela (9).

### Fluxos de conciliação

Dois fluxos distintos, um por tipo de transação na DIF. **Nota de vocabulário:** o glossário (`CONTEXT.md`) chama essas linhas de "Transação Parcelada" / "Não-Parcelada"; o código e as rotas usam *recurring* / *non-recurring* (o flag é a coluna `Recorrente`).

**Linhas parceladas da DIF** (`Recorrente=true`; *recurring* no código) — fluxo de conciliação:
- `GET /api/conciliations` — lista linhas parceladas da DIF com contagem de candidatas
- `GET /api/conciliations/{rowIndex}` — detalhes com as candidatas da ES
- `POST /api/conciliations/{rowIndex}/accept` — escreve o `IdParcela` da DIF na coluna `ColumnIdParcela` da linha casada na ES
- `POST /api/conciliations/{rowIndex}/reject` — anexa a linha da DIF na REJ e limpa a linha na DIF

**Linhas não-parceladas da DIF** (`Recorrente=false`; *non-recurring* no código) — fluxo de revisão manual:
- `GET /api/dif/non-recurring` — lista as linhas não-parceladas da DIF
- `POST /api/dif/non-recurring/{rowIndex}/move-to-es` — move a linha da DIF para a ES
- `POST /api/dif/non-recurring/{rowIndex}/move-to-rej` — move a linha da DIF para a REJ
- `POST /api/dif/non-recurring/move-all-to-es` — move em lote todas as linhas não-parceladas da DIF para a ES
- `PATCH /api/dif/non-recurring/category` — atualiza a categoria na **SHEET_HOM** (não na SHEET_DIF); corpo `{ idParcela, categoria }`
- `PATCH /api/dif/non-recurring/date` — atualiza a data na **SHEET_HOM** (não na SHEET_DIF); corpo `{ idParcela, data }`

Os PATCH escrevem na HOM (não na DIF) porque a DIF é gerada por fórmula: editar a HOM faz o Sheets recalcular a DIF.

Diferente dos endpoints de `move`/`reject`, os dois PATCH **endereçam por `IdParcela`** (identidade estável da transação), não pelo `{rowIndex}` — o índice da DIF não bate com o da HOM, que é gerada por `FILTER`. O backend localiza na HOM a linha com o `IdParcela` recebido; se ela não existir mais, devolve `404`. Ver [`docs/adr/0004`](docs/adr/0004-edicao-hom-enderecada-por-idparcela.md) para a decisão e a assimetria consciente com `move`/`reject`.

**Importante:** o `{rowIndex}` em todas as rotas é o **índice 0-based da linha no array da aba** (a linha 0 é o cabeçalho, os dados começam em 1). Não é um ID sequencial nem opaco.

### Lógica de matching da conciliação

O matching entre a DIF (referência) e a ES (candidatas) exige:
1. Mesmo `Dono`, `Banco` e `Conta`
2. `abs(DIF.Valor - ES.Valor) < 5.00`

**Aceitar**: escreve o `IdParcela` da DIF na coluna `ColumnIdParcela` da linha casada na ES.
**Rejeitar**: anexa a linha da DIF na REJ e limpa a linha da DIF (conteúdo limpo, linha não deletada, para preservar os índices de linha).

### Credenciais do Google Sheets

O cliente do Sheets lê um arquivo de chave de service account. Verifica primeiro a env var `GOOGLE_APPLICATION_CREDENTIALS`; se ausente, usa `credentials.json` no diretório de trabalho. Em produção, o script de deploy gera esse arquivo como `key.json` e ajusta a env var de acordo.

### Autenticação

- Usuário admin único (`ADMIN_USER`/`ADMIN_PASS` do env)
- JWT em cookie `HttpOnly` `olivia_session` (expira em 24h)
- Padrão CSRF double-submit: cookie `olivia_csrf` + header `X-CSRF-Token`, obrigatório em POST/PUT/PATCH/DELETE
- Validação de Origin/Referer contra `APP_ORIGIN`

### Env vars obrigatórias (validadas no startup)

`SHEET_SPREADSHEET_ID`, `ADMIN_USER`, `ADMIN_PASS`, `JWT_SECRET`, `SHEET_ES`, `SHEET_DIF`, `SHEET_REJ`. Obs: `SHEET_HOM` e `APP_ORIGIN` **não** estão na checagem de startup.

### Dev local vs. produção

Para dev local, defina no `.env`:
```
APP_ORIGIN=http://localhost:3001
COOKIE_SECURE=false
COOKIE_DOMAIN=
```

Para o dev local do frontend, fixe no `frontend/app.js`:
```js
const API_URL = 'http://localhost:8080/api';
const EXECUTION_API_URL = 'http://localhost:3000/v1/executions';
```

### CI/CD

Push para `main` dispara `.github/workflows/ecr-push.yml`:
1. Compila e publica as imagens Docker `backend-latest` e `frontend-latest` no AWS ECR
2. Faz deploy na EC2 via AWS SSM (sem SSH), rodando `scripts/deploy-ec2.sh`
3. O script de deploy gera `key.json` e `.env` na EC2 e roda `docker compose pull && docker compose up -d`
