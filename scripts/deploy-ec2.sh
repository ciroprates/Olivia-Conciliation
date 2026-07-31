#!/bin/bash
set -e

# Este script realiza o deploy da aplicação no EC2.
# Ele espera que as seguintes variáveis de ambiente estejam definidas:
# - APP_DIR
# - GCP_SERVICE_ACCOUNT_KEY (Base64)
# - SHEET_SPREADSHEET_ID
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

# BANKS_JSON: lista JSON de bancos/owners.
BANKS_JSON_CLEAN=$(printf '%s' "$BANKS_JSON" | tr -d '\r\n')

cat > .env <<EOF
GOOGLE_APPLICATION_CREDENTIALS=$APP_DIR/key.json
SHEET_SPREADSHEET_ID=$SHEET_SPREADSHEET_ID
ECR_REGISTRY=$ECR_REGISTRY
ECR_REPOSITORY=$ECR_REPOSITORY
PLUGGY_CLIENT_ID=$PLUGGY_CLIENT_ID
PLUGGY_CLIENT_SECRET=$PLUGGY_CLIENT_SECRET
PORT=8080
SHEET_ES=Entradas e Saídas
SHEET_DIF=Diferença
SHEET_REJ=Rejeitados
SHEET_HOM=Homologação
ADMIN_USER=$ADMIN_USER
ADMIN_PASS=$ADMIN_PASS
JWT_SECRET=$JWT_SECRET
BANKS_JSON=$BANKS_JSON_CLEAN
COOKIE_DOMAIN=console.olivinha.online
COOKIE_SECURE=true
APP_ORIGIN=https://console.olivinha.online
EOF

# 4. Login no ECR
echo "Realizando login no ECR..."
aws ecr get-login-password --region "$AWS_REGION" | sudo docker login --username AWS --password-stdin "$ECR_REGISTRY" || {
    echo "Aviso: Falha no login do ECR. Continuando..."
}

# 5. Bootstrap SSL se certificado não existir
CERT_PATH="$APP_DIR/certbot/conf/live/olivinha.online/fullchain.pem"
if [ ! -f "$CERT_PATH" ]; then
    echo "Certificado olivinha.online não encontrado. Criando placeholder e emitindo certificado real..."
    mkdir -p "$(dirname "$CERT_PATH")"
    openssl req -x509 -nodes -newkey rsa:2048 -days 1 \
        -keyout "$(dirname "$CERT_PATH")/privkey.pem" \
        -out "$CERT_PATH" \
        -subj '/CN=localhost' 2>/dev/null
    sudo docker compose up -d nginx
    bash "$APP_DIR/scripts/setup-ssl.sh" --force
fi

# 6. Atualização dos containers
# Limpa imagens/camadas não usadas ANTES do pull, para evitar disco cheio:
# o prune antigo só rodava no fim do script; quando um deploy falhava antes
# disso, as imagens antigas acumulavam e enchiam o disco, fazendo o
# `docker compose pull` seguinte morrer no meio da extração ("no space left on
# device") — e, com `set -e`, isso derrubava o deploy inteiro.
# Containers em execução e volumes nomeados NÃO são afetados.
echo "Limpando imagens/camadas não usadas antes do pull..."
sudo docker image prune -af || true
sudo docker builder prune -f || true

echo "Puxando novas imagens e reiniciando containers..."
sudo docker compose pull

# 6.2 Guarda de portas: garante que 80/443 estão livres ANTES do `up -d`.
# Um Caddy órfão instalado no host (systemd) tomava 80/443 a cada boot,
# impedindo o nginx do compose de bindar e derrubando prod. Aqui paramos e
# desabilitamos qualquer serviço de host conhecido que conflite; se, ainda
# assim, alguma das portas seguir ocupada por processo fora do Docker,
# falhamos cedo com mensagem clara em vez de deixar o `up -d` quebrar.
echo "Verificando se as portas 80/443 estão livres para o nginx..."
for svc in caddy apache2 httpd; do
    if systemctl list-unit-files 2>/dev/null | grep -q "^${svc}\.service"; then
        echo "Serviço de host '${svc}' encontrado — parando e desabilitando..."
        sudo systemctl stop "$svc" || true
        sudo systemctl disable "$svc" || true
    fi
done

# Remoção DEFINITIVA do Caddy órfão (resíduo da era `olivinha.site`).
# Parar/desabilitar não basta: enquanto o pacote e o unit file existirem, um
# `enable` acidental ou um reinstall traz o proxy de host de volta às portas
# 80/443 (ver #35). Este bloco é idempotente e best-effort — desinstala o
# pacote e apaga config/unit residuais, para que um deploy limpo reconcilie o
# host de vez. Nenhum passo pode derrubar o deploy (`|| true`).
if command -v caddy >/dev/null 2>&1 || systemctl list-unit-files 2>/dev/null | grep -q '^caddy\.service' || [ -d /etc/caddy ]; then
    echo "Removendo Caddy órfão do host de forma definitiva..."
    if command -v dnf >/dev/null 2>&1; then
        sudo dnf remove -y caddy || true
    elif command -v yum >/dev/null 2>&1; then
        sudo yum remove -y caddy || true
    fi
    # Resíduos que sobrevivem à remoção do pacote (ou instalações via binário).
    sudo rm -f /etc/systemd/system/caddy.service /usr/lib/systemd/system/caddy.service || true
    sudo rm -rf /etc/caddy || true
    sudo systemctl daemon-reload || true
fi

port_owner() {
    # Lista PIDs (fora do Docker) escutando na porta $1, ou vazio se livre.
    sudo ss -ltnpH "sport = :$1" 2>/dev/null | grep -v 'docker\|containerd' || true
}

for port in 80 443; do
    if [ -n "$(port_owner "$port")" ]; then
        echo "ERRO: a porta $port já está ocupada por um processo fora do Docker:" >&2
        port_owner "$port" >&2
        echo "Libere a porta (ex.: pare/desinstale o serviço de host) e rode o deploy novamente." >&2
        exit 1
    fi
done

sudo docker compose up -d

# 6.1 Recarrega o Nginx para atualizar resolução de upstreams após recriação de containers
echo "Recarregando Nginx..."
sudo docker compose exec -T nginx nginx -s reload || sudo docker compose restart nginx

# 7. Configura renovação automática de certificado SSL (idempotente)
echo "Configurando cron de renovação SSL..."
CERTBOT_CRON="0 0,12 * * * cd $APP_DIR && sudo docker compose run --rm certbot renew >> /var/log/certbot-renew.log 2>&1 && sudo docker compose exec -T nginx nginx -s reload >> /var/log/certbot-renew.log 2>&1"
(sudo crontab -l 2>/dev/null | grep -v "certbot renew"; echo "$CERTBOT_CRON") | sudo crontab -

# 8. Renova certificado SSL imediatamente
echo "Renovando certificado SSL..."
echo "--- $(date) ---" >> /var/log/certbot-renew.log
sudo docker compose run --rm certbot renew >> /var/log/certbot-renew.log 2>&1 || true
sudo docker compose exec -T nginx nginx -s reload >> /var/log/certbot-renew.log 2>&1 || sudo docker compose restart nginx

echo "Limpando imagens antigas..."
sudo docker image prune -f

echo "--- Deploy finalizado com sucesso! ---"
