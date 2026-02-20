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

is_letsencrypt_cert() {
    [ -s "$CERT_FILE" ] && openssl x509 -in "$CERT_FILE" -noout -issuer 2>/dev/null | grep -qi "Let's Encrypt"
}

echo "--- Iniciando processo de geração de certificados ---"

# 1. Criar diretórios necessários
mkdir -p certbot/conf certbot/www

# Se o certificado já for Let's Encrypt válido, não faz nada.
if is_letsencrypt_cert; then
    echo "Certificado Let's Encrypt já existe. Nada a fazer."
    exit 0
fi

# 2. Gera certificado temporário apenas se necessário para o nginx subir na 1a emissão.
if ! is_letsencrypt_cert; then
    echo "Gerando certificado temporário para o Nginx subir..."
    mkdir -p "$CERT_DIR"
    openssl req -x509 -nodes -newkey rsa:2048 -days 1\
      -keyout "$KEY_FILE" \
      -out "$CERT_FILE" \
      -subj "/CN=localhost"
fi

# 3. Subir containers
docker compose up -d nginx

# 4. Aguardar nginx ficar pronto para responder os challenges ACME.
echo "Aguardando Nginx ficar pronto para validação ACME..."
PROBE_DIR="certbot/www/.well-known/acme-challenge"
PROBE_FILE="healthcheck-$(date +%s)"
PROBE_PATH="$PROBE_DIR/$PROBE_FILE"
PROBE_CONTENT="acme-ready"
mkdir -p "$PROBE_DIR"
echo "$PROBE_CONTENT" > "$PROBE_PATH"

MAX_READY_ATTEMPTS=12
READY_ATTEMPT=1

while [ "$READY_ATTEMPT" -le "$MAX_READY_ATTEMPTS" ]; do
    READY_OK=true
    for domain in "${DOMAINS[@]}"; do
        RESPONSE="$(curl -fsS -H "Host: $domain" "http://127.0.0.1/.well-known/acme-challenge/$PROBE_FILE" 2>/dev/null || true)"
        if [ "$RESPONSE" != "$PROBE_CONTENT" ]; then
            READY_OK=false
            break
        fi
    done

    if [ "$READY_OK" = true ]; then
        echo "Nginx pronto para responder desafios ACME."
        break
    fi

    if [ "$READY_ATTEMPT" -eq "$MAX_READY_ATTEMPTS" ]; then
        echo "Nginx não ficou pronto para validação ACME a tempo."
        echo "Verifique logs: docker compose logs nginx"
        rm -f "$PROBE_PATH"
        exit 1
    fi

    sleep 5
    READY_ATTEMPT=$((READY_ATTEMPT + 1))
done
rm -f "$PROBE_PATH"

# 5. Solicitar certificados reais
echo "Solicitando certificados para: ${DOMAINS[*]}"
CERTBOT_ARGS=(certonly --cert-name "$ROOT_DOMAIN" --webroot --webroot-path=/var/www/certbot --email "$EMAIL" --agree-tos --no-eff-email)
for domain in "${DOMAINS[@]}"; do
    CERTBOT_ARGS+=(-d "$domain")
done

MAX_ATTEMPTS=3
ATTEMPT=1

while true; do
    if docker compose run --rm certbot "${CERTBOT_ARGS[@]}"; then
        echo "Certificado emitido com sucesso."
        break
    fi

    if [ "$ATTEMPT" -ge "$MAX_ATTEMPTS" ]; then
        echo "Falha ao emitir certificado após $MAX_ATTEMPTS tentativas."
        exit 1
    fi

    SLEEP_SECONDS=$((5 * ATTEMPT))
    echo "Tentativa $ATTEMPT falhou. Nova tentativa em ${SLEEP_SECONDS}s..."
    sleep "$SLEEP_SECONDS"
    ATTEMPT=$((ATTEMPT + 1))
done

# 6. Reiniciar o Nginx para ler os novos certificados
docker compose restart nginx

echo "--- Processo concluído! Verifique se seu site está acessível em HTTPS ---"
