# 🏦 Olivia Installments Conciliation

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Docker](https://img.shields.io/badge/Docker-Enabled-2496ED?style=flat&logo=docker)](https://www.docker.com/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

Sistema inteligente de conciliação financeira automatizada entre **Google Sheets** e **Pluggy API**. Gerencie transações, identifique parcelas e automatize fluxos de auditoria com segurança e alta performance.

---

A aplicação utiliza uma arquitetura de microservices orquestrada por Docker, protegida por um Proxy Reverso Nginx com suporte a HTTPS (Let's Encrypt) e autenticação por **cookie de sessão HttpOnly (JWT)** com proteção CSRF.

---

## 🚀 Como Executar

### 🐳 Via Docker (Recomendado)

Modo indicado para simular o ambiente de produção com Nginx, SSL e roteamento por domínio.

1.  **Configuração Inicial**:
    ```bash
    cp .env.example .env
    # Preencha as credenciais no arquivo .env
    ```

2.  **Deploy**:
    ```bash
    docker compose up -d
    ```

> [!IMPORTANT]
> Este modo assume:
> - Certificados já existentes em `certbot/conf` (usados pelo `nginx`).
> - Imagem `olivia-api` disponível no registry configurado em `ECR_REGISTRY`.
>
> O deploy via GitHub Actions não executa bootstrap/renovação de SSL. A gestão do certificado é manual na EC2 via `scripts/setup-ssl.sh`.

### 💻 Desenvolvimento Local (Sem Docker)

Para testar mudanças rapidamente sem subir toda a infraestrutura:

1.  **Crie o arquivo de ambiente**:
    ```bash
    cp .env.example .env
    ```

2.  **Ajuste variáveis no `.env` para ambiente local**:
    ```dotenv
    # já existentes
    GOOGLE_APPLICATION_CREDENTIALS=/caminho/absoluto/para/sua-chave.json
    SPREADSHEET_ID=seu_id_da_planilha
    ADMIN_USER=admin
    ADMIN_PASS=sua_senha
    JWT_SECRET=seu_jwt_secret

    # adicionar para dev local
    APP_ORIGIN=http://localhost:3001
    COOKIE_SECURE=false
    COOKIE_DOMAIN=
    ```

3.  **Backend (Go)**:
    ```bash
    go run backend/main.go
    # O backend subirá em http://localhost:8080
    ```

4.  **Frontend (Vanilla JS)**:
    Edite `frontend/app.js` para apontar para o backend local sem proxy:
    ```js
    const API_URL = 'http://localhost:8080/api';
    // opcional: se não estiver rodando olivia-api local, mantenha apenas a conciliação manual
    const EXECUTION_API_URL = 'http://localhost:3000/v1/executions';
    ```

5.  **Suba o frontend estático**:
    ```bash
    cd frontend
    python3 -m http.server 3001
    # Acesse http://localhost:3001
    ```

> [!NOTE]
> No modo sem Docker, as rotas relativas `/api` e `/executions` nao funcionam sem o proxy Nginx.

### 🛠️ Simulando Produção Localmente (Com Docker)

Para testar o roteamento do Nginx no seu computador:
1.  Edite seu arquivo de hosts (`/etc/hosts` no Linux ou `C:\Windows\System32\drivers\etc\hosts` no Windows).
2.  Adicione o mapeamento:
    ```text
    127.0.0.1 console.olivinha.site bff.olivinha.site api.olivinha.site n8n.olivinha.site waha.olivinha.site
    ```
3.  Suba os containers: `docker compose up -d`.

### 🌍 URLs de Acesso

| Serviço | URL |
| :--- | :--- |
| **Aplicação Principal** | [https://console.olivinha.site](https://console.olivinha.site) |
| **Automação n8n** | [https://n8n.olivinha.site](https://n8n.olivinha.site) |
| **WhatsApp API** | [https://waha.olivinha.site](https://waha.olivinha.site) |

---

## 🛠️ Configuração de CI/CD

O projeto utiliza **GitHub Actions** com **AWS Systems Manager (SSM)** para deploys automáticos e seguros, sem necessidade de chaves SSH expostas.

### 🔐 Secrets & Variables Necessárias

> [!IMPORTANT]
> Configure estas variáveis nas configurações do repositório GitHub para o pipeline `ecr-push.yml`.

| Tipo | Chaves |
| :--- | :--- |
| **Secrets** | `GCP_SERVICE_ACCOUNT_KEY`, `SPREADSHEET_ID`, `PLUGGY_CLIENT_ID`, `PLUGGY_CLIENT_SECRET`, `ADMIN_USER`, `ADMIN_PASS`, `JWT_SECRET` |
| **Variables** | `AWS_REGION`, `ECR_REGISTRY`, `ECR_REPOSITORY`, `AWS_ROLE_BUILD_ARN`, `AWS_ROLE_DEPLOY_ARN`, `APP_DIR`, `DEPLOY_TAG_KEY`, `DEPLOY_TAG_VALUE` |

### 🔒 SSL Manual na EC2

O pipeline executa apenas deploy da aplicação. O SSL deve ser executado manualmente na EC2:

1. `cd $APP_DIR`
2. `chmod +x scripts/setup-ssl.sh`
3. `sudo ./scripts/setup-ssl.sh`


> [!IMPORTANT]
> O certificado inicial é emitido para os subdomínios `console`, `n8n` e `waha` em `olivinha.site` (não inclui o domínio raiz `olivinha.site`).
> Garanta DNS válido para esses subdomínios e acesso público à porta `80/TCP` para o desafio HTTP-01.

### 👤 Roles IAM (OIDC) Esperadas

1. `AWS_ROLE_BUILD_ARN`: role usada no job de build/push para autenticar no ECR.
2. `AWS_ROLE_DEPLOY_ARN`: role usada no job de deploy para executar `ec2:DescribeInstances` e comandos via SSM.

---

## 🛡️ Segurança

*   **Proxy Reverso**: Todos os serviços rodam em rede interna Docker, acessíveis apenas via Nginx.
*   **Sessão HttpOnly + CSRF**: JWT em cookie `HttpOnly` validado na borda pelo Nginx (`auth_request`) e token CSRF para métodos mutáveis.
*   **SSL/TLS**: Criptografia de ponta a ponta via Let's Encrypt.
*   **Infrastructure Hardening**: As portas de gerência (SSH) são fechadas para a internet, utilizando o **AWS SSM Session Manager** para acesso administrativo.

---

## Funcionalidades

1. **Fila de Conciliações**: Exibe transações que aguardam pareamento manual.
2. **Algoritmo de Sugestão**: Cruza dados de valor e data para sugerir melhores candidatas.
3. **Fluxo de Aprovação**: Ao aceitar, o sistema escreve o ID de conciliação diretamente na planilha bancária.
4. **Gestão de Rejeitados**: Transações sem par podem ser movidas para uma aba de auditoria.
