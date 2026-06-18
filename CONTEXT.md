# Olivia Conciliation — Glossário de Domínio

## Transação
Uma linha em qualquer aba da planilha (ES, DIF, HOM, REJ). Toda transação tem Dono, Banco, Conta, Valor, Data, Descrição e Categoria.

## Processamento de Transações
Ação disparada pelo usuário que chama o Olivia API para importar transações do Pluggy. Apaga e repopula a HOM a cada execução. Em seguida, a fórmula da DIF é recalculada automaticamente pelo Google Sheets.

## HOM — Homologação
Aba que contém as transações brutas importadas do Pluggy via Olivia API. É volátil: seu conteúdo é apagado e reescrito a cada Processamento de Transações. É a fonte de verdade para edições de categoria e data de transações não-parceladas antes de serem movidas para a ES.

## DIF — Diferença
Aba gerada por fórmula do Google Sheets que exibe as transações da HOM que ainda não têm correspondência na ES. É regenerada automaticamente sempre que a HOM muda. Não é editada diretamente — edições de categoria e data são feitas na HOM.

## ES — Entradas e Saídas
Aba das transações aprovadas pelo usuário. É o registro definitivo de transações confirmadas.

## REJ — Rejeitados
Aba de auditoria. Recebe transações da DIF que foram rejeitadas pelo usuário.

## Transação Parcelada
Transação com `Recorrente=true`. Representa uma parcela de compra parcelada. Na DIF: parcela importada do Pluggy. Na ES: parcela registrada pelo usuário aguardando vinculação.

## Transação Não-Parcelada
Transação com `Recorrente=false`. Não passa pelo fluxo de conciliação — é movida diretamente para ES ou REJ.

## Parcela Sintética
Transação Parcelada gerada automaticamente na importação como placeholder para uma parcela futura ainda não cobrada pelo Pluggy. Tem `IdParcela` prefixado por `synthetic`. Aguarda conciliação com a parcela real quando ela for cobrada.

## Transação Pendente
Transação Parcelada na ES que ainda não foi vinculada a uma parcela real do Pluggy — sem `IdParcela` ou com `IdParcela` de Parcela Sintética. É candidata a conciliação.

## Conciliação
Processo de casar uma Transação Parcelada da DIF com exatamente uma Transação Pendente da ES, vinculando-a pelo `IdParcela`.

## Candidata
Transação Pendente da ES que satisfaz os critérios de correspondência com uma Transação Parcelada da DIF: mesmo Dono, Banco e Conta; diferença de Valor inferior a R$ 5,00. A tolerância existe porque o Pluggy às vezes retorna valores ligeiramente diferentes dos registrados (taxas, IOF, arredondamentos).

## IdParcela
Identificador da parcela atribuído pelo Pluggy. Escrito na ES ao aceitar uma conciliação, vinculando a Transação Pendente à parcela importada.

## Aceitar (Conciliação)
Ação que vincula uma Transação Parcelada da DIF a exatamente uma Candidata escolhida pelo usuário, escrevendo o `IdParcela` da DIF na linha correspondente da ES.

## Rejeitar (Conciliação)
Ação que move uma Transação Parcelada da DIF para a REJ e limpa a linha na DIF.

## Dono
Pessoa física responsável pela transação (ex: nome do titular do cartão ou conta).
