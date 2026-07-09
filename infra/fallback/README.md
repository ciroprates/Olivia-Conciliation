# Página de indisponibilidade (fallback)

`index.html` é a página estática que o CloudFront serve quando a EC2 está
**parada** e a origem responde 502/504. É um HTML único, compartilhado pelas três
distribuições (`console`, `n8n`, `waha`) — o JS lê `window.location.hostname` para
saber em qual subdomínio está.

A decisão de arquitetura (por que CloudFront + S3 + start sob demanda) está em
[`docs/adr/0002`](../../docs/adr/0002-cloudfront-s3-fallback-ec2-scheduler.md).
Este README é só o operacional do artefato.

## Onde roda

- **Bucket:** `s3://olivia-fallback-page/index.html` (privado, acesso via OAC).
- **Servida por:** as 3 distribuições CloudFront, como *custom error response* de
  502/504 (`console` → `E32IJY9OV9FNS`, `n8n` → `E3016IOEYUMT34`,
  `waha` → `E2G767BCSY2JB0`).

## Knobs dentro do `index.html`

- **`LAMBDA_URL`** — endpoint do API Gateway (`kebjn6uy6l…execute-api…`). O botão
  "Iniciar" faz `POST {action:'start', token}`; a Lambda `olivia-ec2-scheduler`
  valida o token e chama `ec2:StartInstances`. O token **não** fica na página.
- **`poll()`** — depois do start, faz `HEAD` no próprio subdomínio a cada 5s e
  redireciona quando a resposta for **`status < 500`**.

## Gotcha importante: o limiar `< 500` depende do `restart: always`

O redirect só dispara com `status < 500`. Um **502** (nginx de pé, mas containers
`backend`/`frontend`/`olivia-api` fora) é `>= 500` → o polling **fica em loop pra
sempre** e a página nunca sai do "Iniciando…". Ou seja: para o start sob demanda
funcionar de ponta a ponta, esses containers precisam de `restart: always` no
`docker-compose.yml` (ver #34). Não "conserte" o limiar sem entender isso — 502
enquanto a app sobe é esperado; o que não pode é a app **nunca** sair do 502.

## Como publicar uma alteração

Editou o `index.html`? Rode (com credenciais AWS locais que tenham
`s3:PutObject` no bucket e `cloudfront:CreateInvalidation` nas 3 distribuições):

```bash
scripts/deploy-fallback.sh
```

Ele sobe o arquivo pro S3 e invalida `/index.html` nas três distribuições. Não há
automação no CI hoje — a página muda raramente e a role de deploy OIDC não tem
essas permissões (automatizar exigiria ampliá-la; fica como follow-up).
