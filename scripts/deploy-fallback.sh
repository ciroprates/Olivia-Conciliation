#!/usr/bin/env bash
set -euo pipefail

# Publica a página de indisponibilidade (fallback) no S3 e invalida os CloudFronts.
# Rodado sob demanda com credenciais AWS locais — a página muda raramente e não
# passa pelo CI (a role de deploy OIDC não tem permissão no bucket/CloudFront).
# Contexto: infra/fallback/README.md e docs/adr/0002-cloudfront-s3-fallback-ec2-scheduler.md

REGION="us-east-1"
BUCKET="olivia-fallback-page"

# Uma distribuição por subdomínio (console, n8n, waha) — todas servem a mesma página.
DISTRIBUTIONS=(E32IJY9OV9FNS E3016IOEYUMT34 E2G767BCSY2JB0)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC="$SCRIPT_DIR/../infra/fallback/index.html"

if [ ! -f "$SRC" ]; then
  echo "Erro: não encontrei $SRC" >&2
  exit 1
fi

echo "Conta AWS: $(aws sts get-caller-identity --query Account --output text)"

echo "Enviando index.html -> s3://$BUCKET/index.html"
aws s3 cp "$SRC" "s3://$BUCKET/index.html" \
  --region "$REGION" \
  --content-type "text/html; charset=utf-8" \
  --cache-control "no-cache"

for dist in "${DISTRIBUTIONS[@]}"; do
  echo "Invalidando /index.html em $dist"
  aws cloudfront create-invalidation \
    --distribution-id "$dist" \
    --paths "/index.html" \
    --query "Invalidation.Id" --output text
done

echo "Pronto — página publicada e CloudFronts invalidados."
