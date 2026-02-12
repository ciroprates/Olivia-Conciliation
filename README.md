# Olivia Installments Conciliation

Aplicação para conciliação de parcelas financeiras usando Google Sheets. Composta por um backend em Go e um frontend SPA.

## Requisitos

- Go 1.21+
- Google Cloud Service Account com a API do Sheets habilitada.
- Arquivo de Service Account (JSON) na raiz do projeto.
- ID da Planilha Google.

## Estrutura da Planilha

A aplicação espera as seguintes abas na planilha:
1. **Entradas e Saídas (ES)**: Transações sem ID.
2. **Diferença (DIF)**: Transações de referência com ID.
3. **Rejeitados (REJ)**: Destino para transações rejeitadas.

## Configuração

1. **Credenciais**: Coloque o arquivo de Service Account (JSON) na raiz do projeto.
2. **Variáveis de Ambiente (preferencialmente via `.env`)**:
   Crie um arquivo `.env` na raiz com as variáveis abaixo:

```
GOOGLE_APPLICATION_CREDENTIALS="olivia-service-account-key.json"
SPREADSHEET_ID="seu-id-da-planilha-aqui"
SHEET_ES="Entradas e Saídas"
SHEET_DIF="Diferença"
SHEET_REJ="Rejeitados"
PORT=8080
```

Se preferir, você também pode exportar as variáveis manualmente no terminal.

## Execução

### Docker (recomendado)

1. **Crie o `.env` na raiz** com as variáveis do bloco acima.
2. **Garanta o arquivo da Service Account** no caminho do `GOOGLE_APPLICATION_CREDENTIALS` no seu `.env`.
   O `docker-compose.yml` usa esse caminho do host para montar o arquivo no container.
3. **Autentique no ECR** (caso precise baixar imagens atualizadas):

```bash
aws ecr get-login-password --region us-east-1 | sudo docker login --username AWS --password-stdin 683684736241.dkr.ecr.us-east-1.amazonaws.com
```

4. **Suba os containers**:

```bash
docker compose up -d
```

5. **Acesse**:
   - Frontend: `http://localhost:3001`
   - Backend: `http://localhost:8080`

Para acompanhar logs:

```bash
docker compose logs -f backend
```

### n8n e waha (opcional)

O `docker-compose.yml` também inclui os serviços `n8n` e `waha` para automação e integração com WhatsApp.  
Se você não precisa deles, pode comentar esses serviços no arquivo ou subir apenas backend e frontend:

```bash
docker compose up -d backend frontend
```

### Execução local (sem Docker)

#### Backend

Execute o servidor Go na porta 8080 (o backend carrega `.env` automaticamente):

```bash
go run backend/main.go
```

#### Frontend

Abra o arquivo `frontend/index.html` no seu navegador.
Como a API habilita CORS, você pode abrir o arquivo diretamente ou usar um servidor simples:

```bash
cd frontend
python3 -m http.server 3000
```
Acesse `http://localhost:3000`.

## Pipeline (GitHub Actions)

O workflow `.github/workflows/ecr-push.yml` automatiza a construção e implantação da aplicação na AWS.

### Fluxo de Trabalho

1.  **Gatilho**: Dispara automaticamente a cada `push` na branch `main`.
2.  **Job `build-and-push`**:
    - Constrói as imagens Docker do **backend** e **frontend**.
    - Faz o login no Amazon ECR.
    - Envia (push) as imagens para o ECR com as tags `:backend-latest` e `:frontend-latest`.
3.  **Job `deploy-ec2`**:
    - **Cópia de Arquivos**: Envia o `docker-compose.yml` atualizado para o servidor (diretório `/var/app`) via SCP.
    - **Deploy Remoto (SSH)**:
        1. Gera o arquivo de credenciais do Google (`key.json`) decodificando o secret `GCP_SERVICE_ACCOUNT_KEY`.
        2. Cria o arquivo `.env` dinamicamente com as variáveis de ambiente necessárias (incluindo `SPREADSHEET_ID` e configurações de abas).
        3. Autentica o Docker no Amazon ECR.
        4. Atualiza os containers (`docker compose pull` e `docker compose up -d`).
        5. Remove imagens antigas (`docker image prune`).

### Configuração do GitHub Actions

Para que o pipeline funcione, configure os seguintes **Secrets** no repositório (`Settings > Secrets and variables > Actions`):

- `AWS_ACCESS_KEY_ID`: ID da chave de acesso AWS.
- `AWS_SECRET_ACCESS_KEY`: Chave secreta de acesso AWS.
- `EC2_SSH_KEY`: Chave privada SSH para acesso à instância EC2.
- `GCP_SERVICE_ACCOUNT_KEY`: Conteúdo do JSON da Service Account do Google **codificado em Base64**.
- `SPREADSHEET_ID`: ID da planilha Google.

As variáveis de ambiente gerais (Região, Registry, IP da EC2, etc.) são definidas diretamente no bloco `env` do arquivo `.github/workflows/ecr-push.yml`.

## Funcionalidades

1. **Fila de Conciliações**: Lista transações da aba DIF que precisam de pareamento.
2. **Detalhamento**: Ao clicar em uma transação, vê detalhes e candidatas (ES) sugeridas.
3. **Aceitar**: Selecione as candidatas corretas e clique em Aceitar. O ID da DIF será escrito nas candidatas na aba ES.
4. **Rejeitar**: Move a transação DIF para a aba REJ e a remove da aba DIF.
