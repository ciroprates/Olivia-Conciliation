# Olivia Installments Conciliation

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Docker](https://img.shields.io/badge/Docker-Enabled-2496ED?style=flat&logo=docker)](https://www.docker.com/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

Sistema de conciliação financeira entre Google Sheets e Pluggy API, com painel web para operação.

## Arquitetura

- `backend` (Go): autenticação, regras de conciliação e integração com Google Sheets.
- `frontend` (Vanilla JS): interface de fila, detalhes e ações de conciliação.
- `nginx`: proxy reverso para `frontend`, `backend`, `olivia-api`, `n8n` e `waha`.
- `olivia-api`: serviço externo consumido por imagem Docker no ECR.

## Quickstart (Docker)

```bash
cp .env.example .env
# ajuste os valores obrigatórios no .env
docker compose up -d
```

Pré-requisitos:
- Certificados válidos em `certbot/conf` para os domínios usados no Nginx.
- Imagens disponíveis no ECR definido por `ECR_REGISTRY` e `ECR_REPOSITORY`.
- Imagem `${ECR_REGISTRY}/olivia-api:latest` acessível no ambiente.

## Desenvolvimento local (sem Docker)

1. Crie o `.env`:
```bash
cp .env.example .env
```

2. Configure ao menos:
```dotenv
GOOGLE_APPLICATION_CREDENTIALS=/caminho/para/key.json
SHEET_SPREADSHEET_ID=seu_id_da_planilha
ADMIN_USER=admin
ADMIN_PASS=sua_senha
JWT_SECRET=seu_jwt_secret
SHEET_ES=Entradas e Saídas
SHEET_DIF=Diferença
SHEET_REJ=Rejeitados
APP_ORIGIN=http://localhost:3001
COOKIE_SECURE=false
COOKIE_DOMAIN=
```

3. Rode o backend:
```bash
go run backend/main.go
```

4. Rode o frontend estático:
```bash
cd frontend
python3 -m http.server 3001
```

5. Ajuste endpoints no `frontend/app.js` para ambiente sem Nginx:
```js
const API_URL = 'http://localhost:8080/api';
const EXECUTION_API_URL = 'http://localhost:3000/v1/executions';
```

Sem Nginx, os paths relativos (`/api` e `/executions`) não funcionam.

## Simulação de produção local

Adicione no `/etc/hosts`:

```text
127.0.0.1 console.olivinha.site bff.olivinha.site api.olivinha.site n8n.olivinha.site waha.olivinha.site
```

Depois execute:

```bash
docker compose up -d
```

## Variáveis de ambiente

Arquivo base: `.env.example`.

### Obrigatórias para o backend iniciar

| Variável | Obrigatória? | Exemplo | Descrição |
| :--- | :--- | :--- | :--- |
| `GOOGLE_APPLICATION_CREDENTIALS` | Sim | `key.json` | Caminho do arquivo de credencial GCP no host. |
| `SHEET_SPREADSHEET_ID` | Sim | `1AbCdEfGhIj...` | ID da planilha principal. |
| `PLUGGY_CLIENT_ID` | Sim (se usar `olivia-api`) | `plg_abc123` | Client ID da Pluggy. |
| `PLUGGY_CLIENT_SECRET` | Sim (se usar `olivia-api`) | `plg_secret_xyz` | Client Secret da Pluggy. |
| `ADMIN_USER` | Sim | `admin` | Usuário do login. |
| `ADMIN_PASS` | Sim | `senha_forte` | Senha do login. |
| `JWT_SECRET` | Sim | `segredo_super_secreto` | Segredo de assinatura JWT. |
| `BANKS_JSON` | Não | `[{"id":"...","name":"Nubank","owner":"Ciro"}]` | Lista JSON de bancos por owner usada na configuração de execução. |
| `APP_ORIGIN` | Recomendável | `http://localhost:3001` | Origem validada para CSRF. |
| `COOKIE_SECURE` | Recomendável | `false` | Flag `Secure` do cookie. |
| `COOKIE_DOMAIN` | Opcional | `` | Domínio do cookie. |
| `PORT` | Opcional | `8080` | Porta HTTP do backend (padrão `8080`). |
| `SHEET_ES` | Opcional | `Entradas e Saídas` | Nome da aba ES. |
| `SHEET_DIF` | Opcional | `Diferença` | Nome da aba DIF. |
| `SHEET_REJ` | Opcional | `Rejeitados` | Nome da aba de rejeitados. |
| `SHEET_HOMOLOG` | Opcional | `Homologação` | Aba usada para persistir edição de `data` dos itens não recorrentes. |
| `ECR_REGISTRY` | Obrigatória em Docker/CI | `683684736241.dkr.ecr.us-east-1.amazonaws.com` | Registry ECR das imagens. |
| `ECR_REPOSITORY` | Obrigatória em Docker/CI | `olivia-conciliation` | Repositório ECR das imagens. |

### Importantes para Docker/integrações

- `ECR_REGISTRY`
- `ECR_REPOSITORY`
- `PLUGGY_CLIENT_ID`
- `PLUGGY_CLIENT_SECRET`
- `APP_ORIGIN`
- `COOKIE_SECURE`
- `COOKIE_DOMAIN`
- `BANKS_JSON` (opcional, recomendado quando usado pelo processamento)

| Variável | Preenchimento obrigatório? | Exemplo | Descrição |
| :--- | :--- | :--- | :--- |
| `GOOGLE_APPLICATION_CREDENTIALS` | Não (gerada automaticamente) | `/home/ubuntu/olivia-installments-conciliation/key.json` | Caminho da chave GCP criada no deploy. |
| `SHEET_SPREADSHEET_ID` | Sim (`secrets.SHEET_SPREADSHEET_ID`) | `1AbCdEfGhIj...` | ID da planilha principal. |
| `ECR_REGISTRY` | Sim (`vars.ECR_REGISTRY`) | `683684736241.dkr.ecr.us-east-1.amazonaws.com` | Registry para `docker compose pull`. |
| `ECR_REPOSITORY` | Sim (`vars.ECR_REPOSITORY`) | `olivia-conciliation` | Repositório de imagens. |
| `PLUGGY_CLIENT_ID` | Sim (`secrets.PLUGGY_CLIENT_ID`) | `plg_abc123` | Credencial Pluggy. |
| `PLUGGY_CLIENT_SECRET` | Sim (`secrets.PLUGGY_CLIENT_SECRET`) | `plg_secret_xyz` | Credencial Pluggy. |
| `PORT` | Não (fixa no script) | `8080` | Porta do backend. |
| `SHEET_ES` | Não (fixa no script) | `Entradas e Saídas` | Aba ES no Google Sheets. |
| `SHEET_DIF` | Não (fixa no script) | `Diferença` | Aba DIF no Google Sheets. |
| `SHEET_REJ` | Não (fixa no script) | `Rejeitados` | Aba de rejeitados. |
| `SHEET_HOMOLOG` | Não (fixa no script) | `Homologação` | Aba usada para persistir edição de `data` dos itens não recorrentes. |
| `ADMIN_USER` | Sim (`secrets.ADMIN_USER`) | `admin` | Usuário de autenticação. |
| `ADMIN_PASS` | Sim (`secrets.ADMIN_PASS`) | `senha_forte` | Senha de autenticação. |
| `JWT_SECRET` | Sim (`secrets.JWT_SECRET`) | `segredo_super_secreto` | Chave de assinatura JWT. |
| `BANKS_JSON` | Não (`secrets.BANKS_JSON`) | `[{"id":"...","name":"Nubank","owner":"Ciro"}]` | Lista JSON de bancos por owner injetada no `.env` de produção. |
| `COOKIE_DOMAIN` | Não (fixa no script) | `console.olivinha.site` | Domínio de cookie em produção. |
| `COOKIE_SECURE` | Não (fixa no script) | `true` | Cookie com `Secure=true` em produção. |
| `APP_ORIGIN` | Não (fixa no script) | `https://console.olivinha.site` | Origem permitida para CSRF. |

## 🛠️ CI/CD (GitHub Actions)

Workflow: `.github/workflows/ecr-push.yml`

### Secrets obrigatórios

- `GCP_SERVICE_ACCOUNT_KEY`
- `SHEET_SPREADSHEET_ID`
- `PLUGGY_CLIENT_ID`
- `PLUGGY_CLIENT_SECRET`
- `ADMIN_USER`
- `ADMIN_PASS`
- `JWT_SECRET`
- `BANKS_JSON`

### Vars obrigatórias

- `AWS_REGION`
- `ECR_REGISTRY`
- `ECR_REPOSITORY`
- `AWS_ROLE_BUILD_ARN`
- `AWS_ROLE_DEPLOY_ARN`
- `APP_DIR`
- `DEPLOY_TAG_KEY`
- `DEPLOY_TAG_VALUE`

### Fluxo de deploy

1. Build e push de imagens para o ECR.
2. Envio de arquivos e variáveis para EC2 via AWS SSM.
3. Execução de `scripts/deploy-ec2.sh` na instância.
4. Geração de `key.json` e `.env`.
5. `docker compose pull && docker compose up -d`.

## SSL (EC2, manual)

```bash
cd $APP_DIR
chmod +x scripts/setup-ssl.sh
sudo ./scripts/setup-ssl.sh
```

## URLs de produção

| Serviço | URL |
| :--- | :--- |
| Console | https://console.olivinha.site |
| n8n | https://n8n.olivinha.site |
| WhatsApp API | https://waha.olivinha.site |

## Segurança

- JWT em cookie `HttpOnly`.
- Proteção CSRF via origem permitida (`APP_ORIGIN`).
- TLS com Let's Encrypt.
- Deploy remoto via AWS SSM Session Manager.

## Funcionalidades

1. Fila de conciliações pendentes.
2. Sugestão de candidatas por critérios de valor/data.
3. Aceite de conciliação com escrita direta na planilha.
4. Rejeição para aba de auditoria.
