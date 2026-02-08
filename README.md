# Olivia Installments Conciliation

Aplicação para conciliação de parcelas financeiras usando Google Sheets. Composta por um backend em Go e um frontend SPA.

## Requisitos

- Go 1.21+
- Google Cloud Service Account com a API do Sheets habilitada.
- Arquivo `credentials.json` na raiz do projeto.
- ID da Planilha Google.

## Estrutura da Planilha

A aplicação espera as seguintes abas na planilha:
1. **Entradas e Saídas (ES)**: Transações sem ID.
2. **Diferença (DIF)**: Transações de referência com ID.
3. **Rejeitados (REJ)**: Destino para transações rejeitadas.

## Configuração

1. **Credenciais**: Coloque o arquivo `credentials.json` (Service Account Key) na raiz do projeto (`/home/xavier10/workspace/olivia-installments-conciliation/`).
2. **Variáveis de Ambiente**:
   Você pode configurar via linha de comando ao executar.

## Execução

### Backend

Execute o servidor Go na porta 8080:

```bash
export GOOGLE_APPLICATION_CREDENTIALS="credentials.json"
export SPREADSHEET_ID="seu-id-da-planilha-aqui"
export SHEET_ES="Entradas e Saídas"
export SHEET_DIF="Diferença"
export SHEET_REJ="Rejeitados"
export PORT=8080

go run backend/main.go
```

### Frontend

Abra o arquivo `frontend/index.html` no seu navegador. 
Como a API habilita CORS, você pode abrir o arquivo diretamente ou usar um servidor simples:

```bash
cd frontend
python3 -m http.server 3000
```
Acesse `http://localhost:3000`.

## Funcionalidades

1. **Fila de Conciliações**: Lista transações da aba DIF que precisam de pareamento.
2. **Detalhamento**: Ao clicar em uma transação, vê detalhes e candidatas (ES) sugeridas.
3. **Aceitar**: Selecione as candidatas corretas e clique em Aceitar. O ID da DIF será escrito nas candidatas na aba ES.
4. **Rejeitar**: Move a transação DIF para a aba REJ e a remove da aba DIF.
