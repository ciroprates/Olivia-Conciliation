# Olivia Installments Conciliation

Sistema de conciliação financeira entre Google Sheets e Pluggy API.

## Visão Geral

- `backend` (Go): autenticação, leitura/escrita em planilhas e regras de conciliação.
- `frontend` (Vanilla JS): interface operacional.
- `nginx`: proxy reverso, roteamento e camada de borda.
- `olivia-api`: serviço externo consumido via imagem no ECR.

## Execução

```bash
cp .env.example .env
# preencha os valores necessários
docker compose up -d
```

**Pré-requisitos:** certificados em `certbot/conf/` e imagem `olivia-api` acessível no ECR (`ECR_REGISTRY`/`ECR_REPOSITORY`).

## Variáveis de ambiente

Veja `.env.example` — variáveis obrigatórias estão marcadas com `# obrigatório`.

## URLs

| Serviço | URL |
| :--- | :--- |
| Console | https://console.olivinha.site |

## CI/CD

Deploy via `.github/workflows/ecr-push.yml` — variáveis e secrets necessários estão documentados no próprio workflow.
