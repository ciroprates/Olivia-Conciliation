# Proxy no container, restart policy e headroom de disco no deploy

Durante o cutover para CloudFront (#27), o stop/start diário da EC2 (#24) passou a expor, a cada boot, falhas latentes de configuração do host que derrubaram produção mais de uma vez. Este ADR registra as lições e as regras que passam a valer, para que a recorrência seja evitada por código/infra — não apenas por documentação.

## Contexto

Três incidentes distintos, todos originados no host (build e testes sempre passaram):

1. **Proxy órfão no host disputando as portas 80/443.** Um `caddy.service` (systemd, *enabled*, resíduo da era `olivinha.site`) subia no boot e tomava as portas 80/443 **antes** do Docker. O container `nginx` do compose não conseguia bindar (`address already in use`), o deploy do #33 falhou e o site ficou no ar pelo Caddy quebrado (TLS `internal error`).
2. **Containers sem `restart` policy.** `backend`/`frontend`/`olivia-api` não voltavam após o reboot noturno, e o nginx respondia 502 (`frontend could not be resolved`) toda manhã.
3. **Deploy frágil no `docker compose pull`.** Com o disco a 84% e o prune rodando só no fim do script, um deploy que falhava antes disso acumulava imagens; o pull seguinte de imagens grandes (n8n/waha `:latest`) morria por falta de espaço e, com `set -e`, derrubava o deploy inteiro.

## Decisões

1. **O reverse proxy é nginx-em-Docker; nenhum proxy roda no host.** As portas 80/443 pertencem ao container `nginx` do compose. O `scripts/deploy-ec2.sh` passa a: (a) parar/desabilitar serviços de host conhecidos que conflitem (`caddy`/`apache2`/`httpd`); (b) **remover definitivamente** o Caddy (desinstalar o pacote e apagar `/etc/caddy/` e unit files residuais, de forma idempotente e best-effort), para que um `enable` acidental ou reinstall não traga o proxy de host de volta; (c) falhar cedo, com mensagem clara, se 80/443 seguirem ocupadas por processo fora do Docker — em vez de deixar o `up -d` quebrar silenciosamente.

2. **Todo container de longa duração tem `restart` policy.** Com o stop/start diário (#24), `restart: always` deixa de ser opcional. Todos os serviços do `docker-compose.yml` (incluindo `backend`/`frontend`/`olivia-api`, corrigidos em #34) declaram `restart: always`.

3. **Deploy garante headroom de disco antes do pull.** O `docker image prune`/`builder prune` roda **antes** do `docker compose pull` (não só no fim), e as imagens grandes de terceiros (`n8n`, `waha`) são **pinadas por versão** em vez de `:latest`, evitando re-pull desnecessário. Ver #43.

## Consequências

- A configuração do host deixa de depender de intervenção manual via SSM: um deploy limpo reconcilia o estado (Caddy removido, portas livres, disco com folga).
- O deploy falha **cedo e com diagnóstico** quando o host está em estado inválido, em vez de deixar prod no ar por um proxy quebrado.

## Alternativa considerada

**IaC completo (Terraform/Ansible/user-data)** para tornar a instância reproduzível — foi justamente a ausência de IaC que permitiu o Caddy esquecido existir. Descartado por ora em favor das guardas idempotentes no `deploy-ec2.sh`, que resolvem a recorrência concreta com baixo custo. A terraformização da infra permanece rastreada no #32; este ADR registra a decisão consciente de não bloquear o hardening nela.
