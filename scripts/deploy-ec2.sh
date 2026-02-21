#!/bin/bash
set -e

# Este script realiza o deploy da aplicação no EC2.
# Ele espera que as seguintes variáveis de ambiente estejam definidas:
# - APP_DIR
# - GCP_SERVICE_ACCOUNT_KEY (Base64)
# - SPREADSHEET_ID
# - AWS_REGION
# - ECR_REGISTRY
# - PLUGGY_CLIENT_ID
# - PLUGGY_CLIENT_SECRET

echo "--- Iniciando Deploy ---"

cd "$APP_DIR"

# 1. Atualiza o docker-compose para a versão enviada pelo SCP
if [ -f "docker-compose.new" ]; then
    echo "Movendo novo docker-compose.yml..."
    mv docker-compose.new docker-compose.yml
fi

# 2. Gera o key.json para acesso ao GCP
echo "Gerando key.json..."
echo "$GCP_SERVICE_ACCOUNT_KEY" | base64 -d > key.json
chmod 644 key.json

# 3. Cria o arquivo .env
echo "Criando arquivo .env..."

# BANKS_JSON pode vir formatado em múltiplas linhas (ex.: JSON pretty-printed)
# e quebrar o parse do docker compose. Compactamos para uma linha.
if command -v jq >/dev/null 2>&1; then
    BANKS_JSON_COMPACT=$(printf '%s' "$BANKS_JSON" | jq -c . 2>/dev/null || printf '%s' "$BANKS_JSON" | tr -d '\r\n')
else
    BANKS_JSON_COMPACT=$(printf '%s' "$BANKS_JSON" | tr -d '\r\n')
fi

cat > .env <<EOF
GOOGLE_APPLICATION_CREDENTIALS=$APP_DIR/key.json
SPREADSHEET_ID=$SPREADSHEET_ID
ECR_REGISTRY=$ECR_REGISTRY
ECR_REPOSITORY=$ECR_REPOSITORY
PLUGGY_CLIENT_ID=$PLUGGY_CLIENT_ID
PLUGGY_CLIENT_SECRET=$PLUGGY_CLIENT_SECRET
PORT=8080
SHEET_ES=Entradas e Saídas
SHEET_DIF=Diferença
SHEET_REJ=Rejeitados
ADMIN_USER=$ADMIN_USER
ADMIN_PASS=$ADMIN_PASS
JWT_SECRET=$JWT_SECRET
BANKS_JSON=$BANKS_JSON_COMPACT
COOKIE_DOMAIN=console.olivinha.site
COOKIE_SECURE=true
APP_ORIGIN=https://console.olivinha.site
EOF

# 4. Login no ECR
echo "Realizando login no ECR..."
aws ecr get-login-password --region "$AWS_REGION" | sudo docker login --username AWS --password-stdin "$ECR_REGISTRY" || {
    echo "Aviso: Falha no login do ECR. Continuando..."
}

# 5. Atualização dos containers
echo "Puxando novas imagens e reiniciando containers..."
sudo docker compose pull
sudo docker compose up -d

# 6. Limpeza
echo "Limpando imagens antigas..."
sudo docker image prune -f

echo "--- Deploy finalizado com sucesso! ---"
