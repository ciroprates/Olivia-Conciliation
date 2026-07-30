# Edições de categoria/data na HOM endereçadas por IdParcela, não por índice de linha

## Contexto

Os endpoints `PATCH /api/dif/non-recurring/{rowIndex}/category` e `/date` editam a **HOM**. Eles recebiam o índice da linha da **DIF** e o usavam diretamente como índice da linha da HOM.

Mas a DIF é gerada por uma fórmula `FILTER` sobre a HOM que remove as linhas cujo `IdParcela` já está na ES ou na REJ e **compacta** o resultado. Os índices de DIF e HOM só coincidem se nenhuma linha da HOM tiver sido filtrada antes da posição em questão — o que raramente é verdade. Resultado: a edição caía silenciosamente em outra transação (bug #21). Além do descasamento estático, um índice de linha também não sobrevive a um novo Processamento de Transações, que apaga e reescreve a HOM.

Fato de domínio que habilita a solução: o `IdParcela` é único e sempre preenchido em toda transação vinda do Pluggy (ver `CONTEXT.md`). É uma identidade estável, independente da posição da linha.

## Decisão

Ancorar as edições de categoria/data no `IdParcela`, enviado pelo frontend no corpo do PATCH — o valor que o usuário **de fato viu** ao listar. O backend localiza na HOM a linha com esse `IdParcela` e escreve nela. Como o `IdParcela` é único, a busca retorna no máximo uma linha.

O índice sai da URL: `PATCH /api/dif/non-recurring/category` e `/date`, com corpo `{ idParcela, categoria }` / `{ idParcela, data }`. O dispatcher já roteia por sufixo, então o roteamento não muda. Se o `IdParcela` não estiver na HOM, o serviço devolve um erro sentinela que o handler mapeia para HTTP `404`.

Implementação rastreada no #44; bug original no #21.

## Alternativas consideradas

- **Backend deduzir o `IdParcela` de `DIF[rowIndex]` no momento do save** (mantendo o índice na URL). Corrige o descasamento estático do #21, mas continua exposto à corrida: se um Processamento de Transações reescrever a DIF entre listar e salvar, o backend lê a linha errada e edita a transação errada de forma "consistente" — pior de detectar. Descartada por não carregar a identidade que o usuário viu.
- **Manter o índice na URL como enfeite**, ignorado pelo backend. A URL passaria a mentir sobre o que endereça. Descartada.
- **Chave composta para linhas sem `IdParcela`** (aventada no #21 como "a definir"). Desnecessária — o `IdParcela` é sempre presente e único.

## Consequências

- **Assimetria consciente.** category/date endereçam por `IdParcela` (agem na HOM, aba-fórmula onde a posição da DIF não vale); `move`/`reject` seguem por `rowIndex` (agem na própria DIF, onde a posição é o handle correto). Não é inconsistência a "corrigir" — quem uniformizar reintroduz o #21.
- **O `404` é um canto de um canto.** Só ocorre se um Processamento de Transações reescrever a HOM entre listar e salvar **e** o Pluggy não trouxer mais aquela transação. Tratado com aviso ao usuário e **sem** recarregar a lista, para preservar rascunhos não-salvos das outras linhas.
- O frontend passa a enviar o `idParcela` no corpo; nenhuma busca nova, pois o valor já vem no `NonRecurringDifSummary`.
