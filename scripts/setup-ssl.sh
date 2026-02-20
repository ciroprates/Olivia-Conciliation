#!/bin/bash
set -euo pipefail

CERT_NAME="olivinha.site-0001"
DOMAINS=("console.olivinha.site" "bff.olivinha.site" "api.olivinha.site" "n8n.olivinha.site" "waha.olivinha.site")
EMAIL="vendas@p7v.com.br"

echo "[1/5] Subindo nginx (HTTP) para servir challenges..."
docker compose up -d nginx

echo "[2/5] Validando webroot do ACME dentro do nginx..."
PROBE_DIR="certbot/www/.well-known/acme-challenge"
PROBE_FILE="healthcheck-$(date +%s)"
mkdir -p "$PROBE_DIR"
echo "ok" > "$PROBE_DIR/$PROBE_FILE"

docker compose exec -T nginx sh -c "test -f /var/www/certbot/.well-known/acme-challenge/$PROBE_FILE"
curl -fsS "http://127.0.0.1/.well-known/acme-challenge/$PROBE_FILE" >/dev/null
rm -f "$PROBE_DIR/$PROBE_FILE"

echo "[3/5] Emitindo certificado Let's Encrypt (webroot)..."
ARGS=(certonly --webroot --webroot-path=/var/www/certbot --email "$EMAIL" --agree-tos --no-eff-email --cert-name "$CERT_NAME")
for d in "${DOMAINS[@]}"; do ARGS+=(-d "$d"); done

docker compose run --rm certbot "${ARGS[@]}"

echo "[4/5] (Opcional) Teste dry-run de renovação..."
docker compose run --rm certbot renew --dry-run || true

echo "[5/5] Recarregando nginx..."
docker compose exec -T nginx nginx -s reload || docker compose restart nginx

echo "OK: certificado emitido. Verifique HTTPS."
