# Olivia Installments Conciliation

Aplicação para conciliação de parcelas financeiras usando Google Sheets. Composta por um backend em Go e um frontend SPA.

---

## 📌 Sumário
- [Requisitos](#requisitos)
- [Estrutura da Planilha](#estrutura-da-planilha)
- [Configuração Local](#configuração-local)
- [Como Executar](#como-executar)
  - [Docker (Recomendado)](#docker-recomendado)
  - [Sem Docker](#sem-docker)
- [Pipeline de CI/CD](#pipeline-de-cicd)
  - [Configuração GitHub Actions](#configuração-github-actions)
- [Funcionalidades](#funcionalidades)

---

## ## Requisitos

- **Go 1.21+**
- **Google Cloud Service Account** com a API do Sheets habilitada.
- Arquivo de Service Account (JSON) na raiz do projeto.
- ID de uma Planilha Google válida.

## ## Estrutura da Planilha

A aplicação espera as seguintes abas na planilha:
1. **Entradas e Saídas (ES)**: Transações bancárias sem identificação.
2. **Diferença (DIF)**: Transações de referência com IDs únicos.
3. **Rejeitados (REJ)**: Destino para transações marcadas como inválidas ou rejeitadas.

---

## ## Configuração Local

1. **Variáveis de Ambiente**:
   Copie o arquivo de exemplo e preencha com seus dados reais:
   ```bash
   cp .env.example .env
   ```

2. **Credenciais Google**:
   Coloque o arquivo JSON da sua Service Account na raiz do projeto conforme configurado na chave `GOOGLE_APPLICATION_CREDENTIALS` do seu `.env`.

---

## ## Como Executar

### ### Docker (Recomendado)

A forma mais rápida de subir o ambiente completo (incluindo serviços auxiliares):

1. **Autentique no ECR** (opcional, se estiver usando imagens remotas):
   ```bash
   aws ecr get-login-password --region us-east-1 | sudo docker login --username AWS --password-stdin 683684736241.dkr.ecr.us-east-1.amazonaws.com
   ```

2. **Suba os containers**:
   ```bash
   docker compose up -d
   ```

*   **Frontend**: [http://localhost:3001](http://localhost:3001)
*   **Backend**: [http://localhost:8080](http://localhost:8080)

> [!NOTE]
> O projeto inclui **n8n** e **waha** para automações. Para subir apenas o core: `docker compose up -d backend frontend`.

### ### Sem Docker

#### #### Backend
```bash
# O backend carrega o .env automaticamente
go run backend/main.go
```

#### #### Frontend
```bash
cd frontend
python3 -m http.server 3000
# Acesse http://localhost:3000
```

---

## ## Pipeline de CI/CD

O projeto utiliza **GitHub Actions** para automação total do build e deploy no EC2.

### ### Configuração GitHub Actions

Para o funcionamento do pipeline [ecr-push.yml](.github/workflows/ecr-push.yml), configure as seguintes chaves no GitHub:

#### #### 🔐 Secrets
| Chave | Descrição |
| :--- | :--- |
| `EC2_SSH_KEY` | Chave privada SSH para acesso ao servidor. |
| `GCP_SERVICE_ACCOUNT_KEY` | JSON da Service Account do Google em **Base64**. |
| `SPREADSHEET_ID` | ID da planilha que será conciliada. |
| `PLUGGY_CLIENT_ID` | Client ID para integração Pluggy. |
| `PLUGGY_CLIENT_SECRET` | Client Secret para integração Pluggy. |

#### #### ⚙️ Variables
| Nome | Exemplo / Valor |
| :--- | :--- |
| `AWS_REGION` | `us-east-1` |
| `ECR_REGISTRY` | `123456789.dkr.ecr.us-east-1.amazonaws.com` |
| `ECR_REPOSITORY` | `olivia-conciliation` |
| `EC2_HOST` | IP Elástico do servidor EC2. |
| `EC2_USER` | `ubuntu` |
| `APP_DIR` | `/var/app` |

> [!TIP]
> O uso de **Variables** permite trocar de servidor ou região AWS sem precisar alterar uma linha de código, mantendo o processo dinâmico e seguro.

---

## ## Funcionalidades

1. **Fila de Conciliações**: Exibe transações que aguardam pareamento manual.
2. **Algoritmo de Sugestão**: Cruza dados de valor e data para sugerir melhores candidatas.
3. **Fluxo de Aprovação**: Ao aceitar, o sistema escreve o ID de conciliação diretamente na planilha bancária.
4. **Gestão de Rejeitados**: Transações sem par podem ser movidas para uma aba de auditoria.
