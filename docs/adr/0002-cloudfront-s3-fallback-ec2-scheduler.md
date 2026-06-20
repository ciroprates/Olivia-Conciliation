# CloudFront + S3 como fallback de indisponibilidade com start sob demanda

## Contexto

A EC2 é parada automaticamente à meia-noite BRT e iniciada às 06:00 BRT para reduzir custos. Fora dessa janela, o usuário que tentar acessar `console.olivinha.site` receberia um timeout ou connection refused.

## Decisão

Colocar um CloudFront na frente da EC2 como única origem. Quando a EC2 está parada e retorna 502/504, o CloudFront serve uma página estática hospedada no S3 com uma caixinha de senha e botão "Iniciar". O botão chama uma Lambda Function URL que valida um token e aciona `ec2:StartInstances`. Após confirmar o envio, a página redireciona automaticamente para `https://console.olivinha.site` após 60 segundos.

Comunicação CloudFront → EC2 é HTTP (porta 80). HTTPS é terminado no CloudFront com certificado ACM. O Let's Encrypt na EC2 pode ser removido.

## Alternativas consideradas

- **Servidor sempre ligado para servir a página**: derrota o propósito de reduzir custos.
- **API Gateway + Lambda para servir a página**: mais complexo sem ganho real — S3 estático é suficiente.
- **Usuário inicia via AWS CLI ou Console**: requer acesso AWS configurado, muito atrito para uso cotidiano.

## Consequências

- DNS de `console.olivinha.site` precisa ser migrado de apontamento direto para EC2 para um CNAME para o CloudFront (configurado no GoDaddy).
- O token de start (`START_TOKEN`) é uma variável de ambiente no Lambda — se vazar, o pior caso é alguém ligar a EC2 repetidamente.
- Startup da EC2 leva ~60s até os containers Docker estarem prontos; o redirect fixo de 60s pode não ser suficiente em casos de inicialização lenta.
