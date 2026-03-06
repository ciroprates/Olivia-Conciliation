# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Backend (Go)

```bash
# Run locally (requires .env configured)
go run backend/main.go

# Build binary
go build -o olivia-backend ./backend

# Tidy dependencies
go mod tidy
```

### Frontend (Vanilla JS)

```bash
# Serve static files locally
cd frontend
python3 -m http.server 3001
```

### Docker

```bash
# Full stack (recommended)
cp .env.example .env
# fill in required values
docker compose up -d

# Rebuild and restart a single service
docker compose up -d --build backend
```

## Architecture

This is a financial reconciliation system between Google Sheets and the Pluggy API. It bridges installment transactions (from Pluggy, stored in the DIF sheet) with recorded transactions (in the ES sheet).

### Services (docker-compose)

| Service | Description |
|---|---|
| `backend` | Go HTTP API, port 8080 |
| `frontend` | Nginx serving static Vanilla JS |
| `nginx` | Reverse proxy (ports 80/443), routes `/api` and `/executions` |
| `olivia-api` | External service (ECR image) — Pluggy integration |
| `n8n` | Workflow automation |
| `waha` | WhatsApp API bridge |

### Backend structure (`olivia-conciliation` Go module)

```
backend/
  main.go              # HTTP server, route registration, startup validation
  models/models.go     # Domain types and spreadsheet column index constants
  sheets/client.go     # Google Sheets API client (FetchRows, WriteCell, AppendRow, ClearRow)
  handlers/auth.go     # Login, Logout, Verify, AuthMiddleware (JWT + CSRF)
  handlers/api.go      # REST handlers for conciliation and DIF operations
  service/conciliation.go  # Business logic: matching, accepting, rejecting, moving rows
```

**Request flow:** `nginx` -> `backend` (Go) -> `sheets.Client` (Google Sheets API)

### Spreadsheet model

Three sheets, configurable via env vars:
- **ES** (`SHEET_ES`, default `"Entradas e Saídas"`): Confirmed transactions. Pending ones have `Recorrente=true` and no `IdParcela`, or `IdParcela` prefixed with `synthetic`.
- **DIF** (`SHEET_DIF`, default `"Diferença"`): Installment transactions from Pluggy needing reconciliation. Recurring rows are candidates for matching against ES.
- **REJ** (`SHEET_REJ`, default `"Rejeitados"`): Rejected transactions (audit trail).

Column layout (0-based index in row slice): B=Data, C=Descricao, D=Valor, E=Categoria, F=Dono, G=Banco, H=Conta, I=Recorrente, J=IdParcela.

### Conciliation matching logic

Matching between DIF (reference) and ES (candidates) requires:
1. Same `Dono`, `Banco`, and `Conta`
2. `abs(DIF.Valor - ES.Valor) < 5.00`

**Accept**: writes DIF's `IdParcela` into the matched ES row's `ColumnIdParcela` column.
**Reject**: appends DIF row to REJ, then clears the DIF row (row content cleared, not deleted, to preserve row indices).

### Authentication

- Single admin user (`ADMIN_USER`/`ADMIN_PASS` from env)
- JWT stored in `HttpOnly` cookie `olivia_session` (24h expiry)
- CSRF double-submit pattern: cookie `olivia_csrf` + header `X-CSRF-Token`, required for POST/PUT/PATCH/DELETE
- Origin/Referer validation against `APP_ORIGIN`

### Local dev vs production differences

For local dev, set in `.env`:
```
APP_ORIGIN=http://localhost:3001
COOKIE_SECURE=false
COOKIE_DOMAIN=
```

For frontend local dev, hardcode in `frontend/app.js`:
```js
const API_URL = 'http://localhost:8080/api';
const EXECUTION_API_URL = 'http://localhost:3000/v1/executions';
```

### CI/CD

Push to `main` triggers `.github/workflows/ecr-push.yml`:
1. Builds and pushes `backend-latest` and `frontend-latest` Docker images to AWS ECR
2. Deploys to EC2 via AWS SSM (no SSH), running `scripts/deploy-ec2.sh`
3. The deploy script generates `key.json` and `.env` on the EC2, then runs `docker compose pull && docker compose up -d`
