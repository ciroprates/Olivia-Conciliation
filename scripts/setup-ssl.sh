#!/bin/bash

# Este script automatiza a geração inicial dos certificados SSL usando Certbot
# IMPORTANTE: Os domínios já devem estar apontando para este IP na GoDaddy!

DOMAINS=("console.olivinha.site" "bff.olivinha.site" "api.olivinha.site" "n8n.olivinha.site" "waha.olivinha.site")
EMAIL="vendas@p7v.com.br" # Substitua pelo seu email se desejar

echo "--- Iniciando processo de geração de certificados ---"

# 1. Criar diretórios necessários
mkdir -p certbot/conf certbot/www

# 2. Subir apenas o Nginx (pode falhar se o nginx.conf já tiver SSL configurado)
# Se o Docker falhar porque não achou os arquivos .pem, precisamos comentar o SSL temporariamente.
# DICA: Vou gerar um certificado "fake" apenas para o Nginx conseguir subir a primeira vez.

if [ ! -f "certbot/conf/live/olivinha.site/fullchain.pem" ]; then
    echo "Gerando certificado temporário para o Nginx subir..."
    mkdir -p certbot/conf/live/olivinha.site
    openssl req -x509 -nodes -newkey rsa:2048 -days 1\
      -keyout "certbot/conf/live/olivinha.site/privkey.pem" \
      -out "certbot/conf/live/olivinha.site/fullchain.pem" \
      -subj "/CN=localhost"
fi

# 3. Subir containers
docker compose up -d nginx

# 4. Solicitar certificados reais
echo "Solicitando certificados para: ${DOMAINS[*]}"
docker compose run --rm certbot certonly --webroot --webroot-path=/var/www/certbot \
    --email $EMAIL --agree-tos --no-eff-email \
    -d olivinha.site \
    -d console.olivinha.site \
    -d bff.olivinha.site \
    -d api.olivinha.site \
    -d n8n.olivinha.site \
    -d waha.olivinha.site

# 5. Reiniciar o Nginx para ler os novos certificados
docker compose restart nginx

echo "--- Processo concluído! Verifique se seu site está acessível em HTTPS ---"
