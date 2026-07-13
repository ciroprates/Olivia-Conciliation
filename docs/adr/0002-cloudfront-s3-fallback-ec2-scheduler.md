# CloudFront + S3 como fallback de indisponibilidade com start sob demanda

> **Atualização 2026-06-23:** o início automático às 06:00 BRT foi removido — o início passou a ser **exclusivamente sob demanda**. Ver a seção "Atualização (2026-06-23)" no fim deste documento.

## Contexto

A EC2 é parada automaticamente à meia-noite BRT e iniciada às 06:00 BRT para reduzir custos. Fora dessa janela, quem tentar acessar qualquer subdomínio da aplicação (`console`, `n8n`, `waha`) receberia um timeout ou connection refused.

## Decisão

**Implementado** (issues #24, #25, #26, #27).

Três distribuições CloudFront — uma por subdomínio (`console.olivinha.online`, `n8n.olivinha.online`, `waha.olivinha.online`) — cada uma com a EC2 como origem HTTP:80 (apontando para o DNS público do Elastic IP, que é estável no stop/start). Quando a EC2 está parada e retorna 502/504, o CloudFront serve uma página estática hospedada no S3 com uma caixinha de senha e botão "Iniciar". O botão chama uma API Gateway HTTP API (proxy para a Lambda) que valida um token e aciona `ec2:StartInstances` — a Lambda Function URL foi descartada por causa do Lambda Block Public Access. Após confirmar o envio, a página faz polling ativo no próprio subdomínio (`window.location.hostname`) a cada 5 segundos e redireciona assim que a aplicação responder.

A página S3 é um único HTML compartilhado entre os três CloudFronts: o JS lê `window.location.hostname` para determinar o alvo do polling e do redirect final. O bucket S3 é privado; o acesso é feito via Origin Access Control (OAC). Em cada distribuição, o behavior `/index.html` aponta para o S3 e os custom error responses 502/504 servem essa página.

HTTPS é terminado no CloudFront com um certificado wildcard `*.olivinha.online` emitido no ACM (região `us-east-1`). Com isso, o Let's Encrypt e o certbot podem ser removidos da EC2.

Comunicação CloudFront → EC2 é HTTP puro (porta 80). Para isso o nginx passou a escutar
também na `:80` servindo conteúdo (antes só redirecionava para HTTPS).

## Estado da implementação

Recursos provisionados (conta `683684736241`, região `us-east-1`):

| Recurso | Identificador |
|---|---|
| ACM wildcard `*.olivinha.online` | `…certificate/6b41f7ab-d82b-40d9-b2db-71a9f321918f` |
| Origin Access Control (S3) | `EPJEEQLSAKSXY` |
| Bucket S3 da página | `olivia-fallback-page` |
| EC2 origin (Elastic IP) | `ec2-100-27-130-161.compute-1.amazonaws.com` |
| CloudFront `console.olivinha.online` | `E32IJY9OV9FNS` → `d13sbsx4fe7pcj.cloudfront.net` |
| CloudFront `n8n.olivinha.online` | `E3016IOEYUMT34` → `d2iznjdnpgpn39.cloudfront.net` |
| CloudFront `waha.olivinha.online` | `E2G767BCSY2JB0` → `djh6cxmumjp5e.cloudfront.net` |

> **Código-fonte da página:** versionado em `infra/fallback/index.html`; publicação via `scripts/deploy-fallback.sh`. Operacional em [`infra/fallback/README.md`](../../infra/fallback/README.md).

## Alternativas consideradas

- **Servidor sempre ligado para servir a página**: derrota o propósito de reduzir custos.
- **API Gateway + Lambda para servir a página**: mais complexo sem ganho real — S3 estático é suficiente.
- **Usuário inicia via AWS CLI ou Console**: requer acesso AWS configurado, muito atrito para uso cotidiano.
- **Certificados individuais por subdomínio**: wildcard `*.olivinha.online` cobre todos os subdomínios atuais e futuros com um único artefato.
- **Redirect fixo de 60s após o start**: substituído por polling ativo — o redirect fixo pode disparar antes dos containers Docker estarem prontos.

## Consequências

- DNS de `console.olivinha.online`, `n8n.olivinha.online` e `waha.olivinha.online` precisarão ser migrados de apontamento direto para EC2 para CNAMEs dos respectivos CloudFronts (configurado na Hostinger).
- O token de start (`START_TOKEN`) é uma variável de ambiente no Lambda — se vazar, o pior caso é alguém ligar a EC2 repetidamente. Risco aceito dado o blast radius pequeno.
- Startup da EC2 leva ~60s até os containers Docker estarem prontos; o polling elimina o risco de redirecionar antes da app estar pronta.
- Serviços `n8n` e `waha` ficam indisponíveis entre meia-noite e 06:00 BRT. Mensagens WhatsApp recebidas nesse período são perdidas silenciosamente — perda aceitável enquanto o chatbot (#22) não estiver em produção.

## Atualização (2026-06-23) — início automático removido

O início automático às 06:00 BRT foi **descartado**. A regra EventBridge `olivia-ec2-start` (`cron(0 9 * * ? *)`, alvo `olivia-ec2-scheduler` com input `{"action":"start"}`) foi **desabilitada** — não deletada, para rollback trivial via `aws events enable-rule --name olivia-ec2-start`. Planeja-se deletá-la após ~1 semana de validação.

A partir de agora o início é **exclusivamente sob demanda**: a EC2 só liga quando alguém abre um subdomínio, digita a senha e aciona o botão "Iniciar" na página de fallback (API Gateway → `olivia-ec2-scheduler`, ramo do token). O desligamento às 00:00 BRT (regra `olivia-ec2-stop`, `cron(0 3 * * ? *)`, input `{"action":"stop"}`) permanece **inalterado**.

**Motivo:** há um único usuário, o boot de ~60s no primeiro acesso do dia é aceitável, e não existe nenhum workflow no n8n que dependa do warm-up matinal. O job diário de atualização de Items do Pluggy (`daily-update-items-schedule` → Lambda `update-items-function`, 03:00 BRT) é **cloud-to-cloud** (chama a API do Pluggy diretamente, não toca a EC2), então é indiferente ao estado da máquina.

**Consequência revista:** a janela de indisponibilidade de `n8n`/`waha` deixa de ser "meia-noite–06:00 BRT" e passa a ser "da meia-noite até a primeira visita do dia" — potencialmente o dia inteiro, se ninguém acessar.

**⚠️ Gotcha operacional — cache da página de fallback (2026-07-12):** a página de fallback (custom error response 502/504) pode ser servida do cache do navegador/CloudFront mesmo depois da EC2 já ter subido, dando a falsa impressão de que o botão "Iniciar"/token não funcionou. Um hard refresh (Ctrl+Shift+R) fura o cache e cai na aplicação. Já enganou ao menos duas vezes (o incidente de 2026-07-09 abaixo e a issue #46, fechada como inválida). Se recorrer, reduzir o TTL do custom error response 502/504 no CloudFront. Falso positivo de cache **não** é regressão no fluxo token→Lambda→`ec2:StartInstances`.

**⚠️ Pré-requisito não satisfeito — start sob demanda ainda inativo (2026-07-09):** o início sob demanda descrito acima depende do **cutover de DNS** dos subdomínios para o CloudFront, que **ainda não foi feito**. Enquanto o DNS apontar direto para o EIP, a página de fallback (servida pelo CloudFront apenas nos erros 502/504) não é alcançável — logo, com o auto-start desabilitado, a **única** forma de ligar a EC2 é manual (`aws ec2 start-instances`). Um incidente em 2026-07-09 (EC2 parada ~2 semanas, console inacessível) expôs exatamente isso. O cutover destrava simultaneamente o start sob demanda e o HTTPS na borda (cert ACM). **Até o cutover, o modelo "sob demanda" está efetivamente inativo e o start permanece manual.**
