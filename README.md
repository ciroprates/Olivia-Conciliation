# 🏦 Olivia Installments Conciliation

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Docker](https://img.shields.io/badge/Docker-Enabled-2496ED?style=flat&logo=docker)](https://www.docker.com/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

Sistema de conciliação financeira entre Google Sheets e Pluggy API.

## Visão Geral

- `backend` (Go): autenticação, leitura/escrita em planilhas e regras de conciliação.
- `frontend` (Vanilla JS): interface operacional.
- `nginx`: proxy reverso, roteamento e camada de borda.
- `olivia-api`: serviço externo consumido via imagem no ECR.

## 🚀 Execução

### 1. Docker (recomendado)

```bash
cp .env.example .env
# preencha os valores necessários
docker compose up -d
```

Pré-requisitos deste modo:
- Certificados já presentes em `certbot/conf`.
- Imagem `olivia-api` disponível no registry definido por `ECR_REGISTRY`/`ECR_REPOSITORY`.

### 2. Desenvolvimento local (sem Docker)

1. Crie o `.env`:
```bash
cp .env.example .env
```

2. Ajuste ao menos estas variáveis para local:
```dotenv
GOOGLE_APPLICATION_CREDENTIALS=/caminho/para/key.json
SHEET_SPREADSHEET_ID=seu_id
ADMIN_USER=admin
ADMIN_PASS=sua_senha
JWT_SECRET=seu_jwt_secret
APP_ORIGIN=http://localhost:3001
COOKIE_SECURE=false
COOKIE_DOMAIN=
```

3. Suba o backend:
```bash
go run backend/main.go
```

4. No `frontend/app.js`, use URLs diretas no modo local:
```js
const API_URL = 'http://localhost:8080/api';
const EXECUTION_API_URL = 'http://localhost:3000/v1/executions';
```

5. Suba frontend estático:
```bash
cd frontend
python3 -m http.server 3001
```

Observação: sem Nginx, rotas relativas (`/api`, `/executions`) não funcionam.

### 3. Simular produção local com Nginx

Adicione no hosts:

```text
127.0.0.1 console.olivinha.site bff.olivinha.site api.olivinha.site n8n.olivinha.site waha.olivinha.site
```

Depois execute `docker compose up -d`.

## 🌍 URLs

| Serviço | URL |
| :--- | :--- |
| Console | https://console.olivinha.site |
| n8n | https://n8n.olivinha.site |
| WhatsApp API | https://waha.olivinha.site |

## 🧩 Variáveis de Ambiente

### `.env` local (base: `.env.example`)

| Variável | Obrigatória? | Exemplo | Descrição |
| :--- | :--- | :--- | :--- |
| `GOOGLE_APPLICATION_CREDENTIALS` | Sim | `key.json` | Caminho do arquivo de credencial GCP no host. |
| `SHEET_SPREADSHEET_ID` | Sim | `1AbCdEfGhIj...` | ID da planilha principal. |
| `PLUGGY_CLIENT_ID` | Sim (se usar `olivia-api`) | `plg_abc123` | Client ID da Pluggy. |
| `PLUGGY_CLIENT_SECRET` | Sim (se usar `olivia-api`) | `plg_secret_xyz` | Client Secret da Pluggy. |
| `ADMIN_USER` | Sim | `admin` | Usuário do login. |
| `ADMIN_PASS` | Sim | `senha_forte` | Senha do login. |
| `JWT_SECRET` | Sim | `segredo_super_secreto` | Segredo de assinatura JWT. |
| `BANKS` | Não | `item-id-1,item-id-2` | Lista CSV de item IDs da Pluggy usada quando `payload.banks` não é enviado. |
| `APP_ORIGIN` | Recomendável | `http://localhost:3001` | Origem validada para CSRF. |
| `COOKIE_SECURE` | Recomendável | `false` | Flag `Secure` do cookie. |
| `COOKIE_DOMAIN` | Opcional | `` | Domínio do cookie. |
| `PORT` | Opcional | `8080` | Porta HTTP do backend (padrão `8080`). |
| `SHEET_ES` | Opcional | `Entradas e Saídas` | Nome da aba ES. |
| `SHEET_DIF` | Opcional | `Diferença` | Nome da aba DIF. |
| `SHEET_REJ` | Opcional | `Rejeitados` | Nome da aba de rejeitados. |
| `ECR_REGISTRY` | Obrigatória em Docker/CI | `683684736241.dkr.ecr.us-east-1.amazonaws.com` | Registry ECR das imagens. |
| `ECR_REPOSITORY` | Obrigatória em Docker/CI | `olivia-conciliation` | Repositório ECR das imagens. |

### Variáveis que o pipeline copia para o `.env` (produção)

Fonte: `scripts/deploy-ec2.sh` (`cat > .env <<EOF`).

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
| `ADMIN_USER` | Sim (`secrets.ADMIN_USER`) | `admin` | Usuário de autenticação. |
| `ADMIN_PASS` | Sim (`secrets.ADMIN_PASS`) | `senha_forte` | Senha de autenticação. |
| `JWT_SECRET` | Sim (`secrets.JWT_SECRET`) | `segredo_super_secreto` | Chave de assinatura JWT. |
| `BANKS` | Não | `item-id-1,item-id-2` | Lista CSV de item IDs da Pluggy usada quando `payload.banks` não é enviado. |
| `COOKIE_DOMAIN` | Não (fixa no script) | `console.olivinha.site` | Domínio de cookie em produção. |
| `COOKIE_SECURE` | Não (fixa no script) | `true` | Cookie com `Secure=true` em produção. |
| `APP_ORIGIN` | Não (fixa no script) | `https://console.olivinha.site` | Origem permitida para CSRF. |

## 🛠️ CI/CD (GitHub Actions)

Workflow: `.github/workflows/ecr-push.yml`

### Variáveis obrigatórias no GitHub

| Tipo | Chaves |
| :--- | :--- |
| `secrets` | `GCP_SERVICE_ACCOUNT_KEY`, `SHEET_SPREADSHEET_ID`, `PLUGGY_CLIENT_ID`, `PLUGGY_CLIENT_SECRET`, `ADMIN_USER`, `ADMIN_PASS`, `JWT_SECRET`, `BANKS` |
| `vars` | `AWS_REGION`, `ECR_REGISTRY`, `ECR_REPOSITORY`, `AWS_ROLE_BUILD_ARN`, `AWS_ROLE_DEPLOY_ARN`, `APP_DIR`, `DEPLOY_TAG_KEY`, `DEPLOY_TAG_VALUE` |

### Fluxo de deploy

1. Workflow faz build/push das imagens no ECR.
2. Workflow envia arquivos e variáveis via AWS SSM.
3. `scripts/deploy-ec2.sh` na EC2 gera `key.json` e `.env`.
4. `docker compose pull && docker compose up -d` aplica o deploy.

### SSL (manual na EC2)

```bash
cd $APP_DIR
chmod +x scripts/setup-ssl.sh
sudo ./scripts/setup-ssl.sh
```

O certificado inicial cobre `console`, `n8n` e `waha` em `olivinha.site`.

## 🔒 Segurança

- Serviços internos expostos apenas via Nginx.
- JWT em cookie `HttpOnly` + proteção CSRF.
- TLS com Let's Encrypt.
- Acesso operacional via AWS SSM Session Manager.

## Funcionalidades

1. Fila de conciliações pendentes.
2. Sugestão automática de correspondência por valor/data.
3. Aprovação com escrita direta na planilha.
4. Gestão de rejeitados para auditoria.
