#!/bin/bash
set -euo pipefail

# Garante execução consistente mesmo quando chamado fora da raiz do projeto.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

# Este script automatiza a geração inicial dos certificados SSL usando Certbot
# IMPORTANTE: Os domínios já devem estar apontando para este IP na GoDaddy!

ROOT_DOMAIN="olivinha.site"
DOMAINS=("console.olivinha.site" "bff.olivinha.site" "api.olivinha.site" "n8n.olivinha.site" "waha.olivinha.site")
EMAIL="vendas@p7v.com.br" # Substitua pelo seu email se desejar
CERT_DIR="certbot/conf/live/$ROOT_DOMAIN"
CERT_FILE="$CERT_DIR/fullchain.pem"
KEY_FILE="$CERT_DIR/privkey.pem"

is_letsencrypt_cert() {
    local cert_path="$1"
    [ -s "$cert_path" ] && openssl x509 -in "$cert_path" -noout -issuer 2>/dev/null | grep -qi "Let's Encrypt"
}

# Remove renewal configs quebradas no host para não derrubar certbot renew depois.
# Isso resolve o caso: /etc/letsencrypt/renewal/olivinha.site.conf (parsefail).
cleanup_broken_renewal_configs() {
    local f
    shopt -s nullglob
    for f in /etc/letsencrypt/renewal/*.conf; do
        # Se não contém referências mínimas, remove.
        if ! grep -qE '^\s*cert\s*=' "$f" \
          || ! grep -qE '^\s*privkey\s*=' "$f" \
          || ! grep -qE '^\s*fullchain\s*=' "$f"; then
            echo "[SSL] Removendo renewal conf inválida (campos ausentes): $f"
            sudo rm -f "$f" || rm -f "$f"
            continue
        fi

        # Extrai paths e valida existência.
        local cert priv full
        cert="$(awk -F'= ' '/^\s*cert\s*=/{print $2}' "$f" | tail -n1)"
        priv="$(awk -F'= ' '/^\s*privkey\s*=/{print $2}' "$f" | tail -n1)"
        full="$(awk -F'= ' '/^\s*fullchain\s*=/{print $2}' "$f" | tail -n1)"

        if [ -n "${cert:-}" ] && [ -n "${priv:-}" ] && [ -n "${full:-}" ] \
          && [ -f "$cert" ] && [ -f "$priv" ] && [ -f "$full" ]; then
            continue
        fi

        echo "[SSL] Removendo renewal conf inválida (paths faltando): $f"
        sudo rm -f "$f" || rm -f "$f"
    done
    shopt -u nullglob
}

# Determinístico: escolhe lineage cujo certificado expira mais tarde (mais "novo").
find_latest_letsencrypt_lineage() {
    local candidate cert_path latest="" latest_ts=0 ts enddate
    shopt -s nullglob
    for candidate in "certbot/conf/live/$ROOT_DOMAIN" "certbot/conf/live/$ROOT_DOMAIN"-*; do
        cert_path="$candidate/fullchain.pem"
        if is_letsencrypt_cert "$cert_path"; then
            enddate="$(openssl x509 -in "$cert_path" -noout -enddate 2>/dev/null | cut -d= -f2 || true)"
            ts="$(date -d "$enddate" +%s 2>/dev/null || echo 0)"
            if [ "$ts" -ge "$latest_ts" ]; then
                latest_ts="$ts"
                latest="$candidate"
            fi
        fi
    done
    shopt -u nullglob
    [ -n "$latest" ] && printf '%s\n' "$latest"
}

link_primary_cert_to_lineage() {
    local lineage_dir="$1"
    local lineage_name
    lineage_name="$(basename "$lineage_dir")"
    mkdir -p "$CERT_DIR"
    ln -sfn "../$lineage_name/fullchain.pem" "$CERT_FILE"
    ln -sfn "../$lineage_name/privkey.pem" "$KEY_FILE"
}

echo "--- Iniciando processo de geração de certificados ---"

# 1. Criar diretórios necessários
mkdir -p certbot/conf certbot/www

# 1.1 Limpa lixo de renewal configs quebradas no host (idempotente)
cleanup_broken_renewal_configs || true

# Se já houver um certificado Let's Encrypt em qualquer lineage (olivinha.site ou olivinha.site-0001 etc.),
# normaliza o caminho primário para manter compatibilidade com nginx.conf.
EXISTING_LINEAGE="$(find_latest_letsencrypt_lineage || true)"
if [ -n "$EXISTING_LINEAGE" ]; then
    if [ "$EXISTING_LINEAGE" != "$CERT_DIR" ]; then
        echo "Encontrado certificado Let's Encrypt em $(basename "$EXISTING_LINEAGE"). Normalizando links em $CERT_DIR..."
        link_primary_cert_to_lineage "$EXISTING_LINEAGE"
    fi
    echo "Certificado Let's Encrypt já existe. Nada a fazer."
    exit 0
fi

# 2. Gera certificado temporário apenas se necessário para o nginx subir na 1a emissão.
if ! is_letsencrypt_cert "$CERT_FILE"; then
    echo "Gerando certificado temporário para o Nginx subir..."
    mkdir -p "$CERT_DIR"
    openssl req -x509 -nodes -newkey rsa:2048 -days 1 \
      -keyout "$KEY_FILE" \
      -out "$CERT_FILE" \
      -subj "/CN=localhost"
fi

# 3. Subir/recriar nginx para garantir bind mounts corretos do certbot.
docker compose up -d --force-recreate nginx

# 4. Aguardar nginx ficar pronto para responder os challenges ACME.
echo "Aguardando Nginx ficar pronto para validação ACME..."
PROBE_DIR="certbot/www/.well-known/acme-challenge"
PROBE_FILE="healthcheck-$(date +%s)"
PROBE_PATH="$PROBE_DIR/$PROBE_FILE"
PROBE_CONTENT="acme-ready"
mkdir -p "$PROBE_DIR"
echo "$PROBE_CONTENT" > "$PROBE_PATH"

# Sanity check: arquivo de probe deve estar visível dentro do container nginx.
if ! docker compose exec -T nginx sh -c "test -f /var/www/certbot/.well-known/acme-challenge/$PROBE_FILE"; then
    echo "Probe ACME não está visível dentro do nginx (/var/www/certbot)."
    echo "Verifique se o bind mount ./certbot/www -> /var/www/certbot está correto."
    rm -f "$PROBE_PATH"
    exit 1
fi

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

# Remove o fallback CN=localhost antes do certbot criar o lineage real "olivinha.site".
rm -f "$CERT_FILE" "$KEY_FILE"
rmdir "$CERT_DIR" 2>/dev/null || true

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

# Garante que o caminho primário continue estável mesmo quando o certbot usar sufixos (-0001, -0002...).
ISSUED_LINEAGE="$(find_latest_letsencrypt_lineage || true)"
if [ -n "$ISSUED_LINEAGE" ] && [ "$ISSUED_LINEAGE" != "$CERT_DIR" ]; then
    echo "Normalizando links do certificado primário para $(basename "$ISSUED_LINEAGE")..."
    link_primary_cert_to_lineage "$ISSUED_LINEAGE"
fi

# 6. Reiniciar o Nginx para ler os novos certificados
docker compose restart nginx

echo "--- Processo concluído! Verifique se seu site está acessível em HTTPS ---"