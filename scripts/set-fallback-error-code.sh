#!/usr/bin/env bash
set -euo pipefail

# Ajusta o ResponseCode do custom error response (502/504) das 3 distribuições
# CloudFront, mantendo ResponsePagePath (/index.html) e ErrorCachingMinTTL (5).
#
# Por que 503: a página de fallback (start sob demanda) faz polling no próprio
# subdomínio e precisa distinguir "app subindo" de "app pronto". Com o CloudFront
# rotulando o fallback como 200, o status sozinho não distinguia os dois estados
# (#40). A página nova detecta a troca por conteúdo (marcador), então já é imune
# ao status; ainda assim rotular o fallback como 503 (Service Unavailable) é
# semanticamente correto e funciona como segunda camada.
# Ver docs/adr/0002-cloudfront-s3-fallback-ec2-scheduler.md e infra/fallback/README.md.
#
# As distribuições NÃO estão em IaC — este script é a forma reproduzível de
# aplicar/reverter a mudança. Rode com credenciais AWS que tenham
# cloudfront:GetDistributionConfig e cloudfront:UpdateDistribution.
#
# Uso:
#   scripts/set-fallback-error-code.sh            # aplica 503 (padrão)
#   scripts/set-fallback-error-code.sh 200        # reverte para 200
#   DRY_RUN=1 scripts/set-fallback-error-code.sh  # só mostra o que mudaria

TARGET_CODE="${1:-503}"

# Uma distribuição por subdomínio (console, n8n, waha) — todas servem a mesma página.
DISTRIBUTIONS=(E32IJY9OV9FNS E3016IOEYUMT34 E2G767BCSY2JB0)

echo "Conta AWS: $(aws sts get-caller-identity --query Account --output text)"
echo "Alvo: custom error 502/504 -> ResponseCode $TARGET_CODE"
echo

for dist in "${DISTRIBUTIONS[@]}"; do
  tmp_cfg="$(mktemp)"
  new_cfg="$(mktemp)"
  aws cloudfront get-distribution-config --id "$dist" > "$tmp_cfg"

  etag="$(python3 -c "import json,sys; print(json.load(open(sys.argv[1]))['ETag'])" "$tmp_cfg")"

  # Extrai o DistributionConfig, ajusta o ResponseCode dos 502/504 e reporta o
  # que mudou. Passa caminhos via argv para não sofrer com quoting.
  changed="$(python3 - "$tmp_cfg" "$TARGET_CODE" "$new_cfg" <<'PY'
import json, sys
src, target, out = sys.argv[1], sys.argv[2], sys.argv[3]
cfg = json.load(open(src))["DistributionConfig"]
changed = []
for it in cfg.get("CustomErrorResponses", {}).get("Items", []):
    if it.get("ErrorCode") in (502, 504) and it.get("ResponseCode") != target:
        changed.append(f"{it['ErrorCode']}: {it.get('ResponseCode')} -> {target}")
        it["ResponseCode"] = target
json.dump(cfg, open(out, "w"))
print("\n".join(changed))
PY
)"

  if [ -z "$changed" ]; then
    echo "[$dist] já está em $TARGET_CODE — nada a fazer."
    rm -f "$tmp_cfg" "$new_cfg"
    continue
  fi

  echo "[$dist] mudanças:"
  echo "$changed" | sed 's/^/  /'

  if [ "${DRY_RUN:-0}" = "1" ]; then
    echo "[$dist] DRY_RUN=1 — não aplicado."
    rm -f "$tmp_cfg" "$new_cfg"
    continue
  fi

  status="$(aws cloudfront update-distribution \
    --id "$dist" \
    --if-match "$etag" \
    --distribution-config "file://$new_cfg" \
    --query "Distribution.Status" --output text)"
  echo "[$dist] aplicado (status: $status)"

  rm -f "$tmp_cfg" "$new_cfg"
done

echo
echo "Pronto. A propagação do CloudFront leva alguns minutos."
