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
- **`poll()`** — depois do start, faz `GET` no próprio subdomínio a cada 5s e
  redireciona quando a resposta **deixa de ser esta página** (ausência do
  marcador). Detecta por conteúdo, **não** por status code — ver o gotcha abaixo.
- **`FALLBACK_MARKER`** / `<meta name="olivia-fallback-page">` — a marca que
  identifica esta página. O `poll()` redireciona quando ela some da resposta (#40).

## Gotcha importante: o redirect depende do app **realmente subir** (`restart: always`)

O `poll()` redireciona quando a resposta **deixa de ser esta página** — ou seja,
quando o CloudFront para de servir o fallback (502/504) e passa a servir o app
real. Enquanto os containers `backend`/`frontend`/`olivia-api` estiverem fora, o
nginx devolve **502**, o CloudFront serve esta página, e o polling **espera pra
sempre**. Para o start sob demanda fechar o ciclo, esses containers precisam de
`restart: always` no `docker-compose.yml` (ver #34): 502 enquanto a app sobe é
esperado; o que não pode é a app **nunca** sair do 502.

> **Por que por conteúdo e não por status code (#40):** o custom error response
> do CloudFront serve esta página com um status **próprio** (`200`, ou `503` após
> #40) — não o 502/504 da origem. Então o status que o navegador vê **não**
> distingue "subindo" de "pronto" (ambos podem ser 200). Por isso o `poll()`
> detecta a troca pela ausência do marcador, e não por um limiar de status.

## Como publicar uma alteração

Editou o `index.html`? Rode (com credenciais AWS locais que tenham
`s3:PutObject` no bucket e `cloudfront:CreateInvalidation` nas 3 distribuições):

```bash
scripts/deploy-fallback.sh
```

Ele sobe o arquivo pro S3 e invalida `/index.html` nas três distribuições. Não há
automação no CI hoje — a página muda raramente e a role de deploy OIDC não tem
essas permissões (automatizar exigiria ampliá-la; fica como follow-up).
