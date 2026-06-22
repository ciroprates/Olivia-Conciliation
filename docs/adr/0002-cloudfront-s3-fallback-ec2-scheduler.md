# CloudFront + S3 como fallback de indisponibilidade com start sob demanda

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
