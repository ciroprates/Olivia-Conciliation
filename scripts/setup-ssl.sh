#!/bin/bash
set -euo pipefail

# Este script automatiza a geração inicial dos certificados SSL usando Certbot
# IMPORTANTE: Os domínios já devem estar apontando para este IP na GoDaddy!

ROOT_DOMAIN="olivinha.site"
DOMAINS=("console.olivinha.site" "bff.olivinha.site" "api.olivinha.site" "n8n.olivinha.site" "waha.olivinha.site")
EMAIL="vendas@p7v.com.br" # Substitua pelo seu email se desejar
CERT_DIR="certbot/conf/live/$ROOT_DOMAIN"
CERT_FILE="$CERT_DIR/fullchain.pem"
KEY_FILE="$CERT_DIR/privkey.pem"

echo "--- Iniciando processo de geração de certificados ---"

# 1. Criar diretórios necessários
mkdir -p certbot/conf certbot/www

# Se o certificado já for Let's Encrypt válido, não faz nada.
if [ -s "$CERT_FILE" ] && openssl x509 -in "$CERT_FILE" -noout -issuer 2>/dev/null | grep -qi "Let's Encrypt"; then
    echo "Certificado Let's Encrypt já existe. Nada a fazer."
    exit 0
fi

# 2. Gera certificado temporário apenas se necessário para o nginx subir na 1a emissão.
if [ ! -s "$CERT_FILE" ] || ! openssl x509 -in "$CERT_FILE" -noout -issuer 2>/dev/null | grep -qi "Let's Encrypt"; then
    echo "Gerando certificado temporário para o Nginx subir..."
    mkdir -p "$CERT_DIR"
    openssl req -x509 -nodes -newkey rsa:2048 -days 1\
      -keyout "$KEY_FILE" \
      -out "$CERT_FILE" \
      -subj "/CN=localhost"
fi

# 3. Subir containers
docker compose up -d nginx

# 4. Solicitar certificados reais
echo "Solicitando certificados para: ${DOMAINS[*]}"
CERTBOT_ARGS=(certonly --webroot --webroot-path=/var/www/certbot --email "$EMAIL" --agree-tos --no-eff-email -d "$ROOT_DOMAIN")
for domain in "${DOMAINS[@]}"; do
    CERTBOT_ARGS+=(-d "$domain")
done
docker compose run --rm certbot "${CERTBOT_ARGS[@]}"

# 5. Reiniciar o Nginx para ler os novos certificados
docker compose restart nginx

echo "--- Processo concluído! Verifique se seu site está acessível em HTTPS ---"
