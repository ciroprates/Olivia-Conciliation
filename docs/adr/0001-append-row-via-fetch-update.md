# AppendRow usa batchUpdate com AppendCellsRequest + tableId

As abas ES e REJ usam Tabelas nativas do Google Sheets. Tanto `values.append` quanto `values.update` são métodos genéricos de células sem consciência de tabelas nativas — nenhum dos dois estende a tabela ao inserir uma nova linha.

A solução adotada é `batchUpdate` com `AppendCellsRequest` passando o `tableId` da tabela nativa. Quando `tableId` está presente, o campo `sheetId` é ignorado. O `tableId` é diferente do `sheetId` numérico e é obtido via `spreadsheets.get` percorrendo `sheet.Tables[].TableId`.

O `tableId` de ES e REJ é buscado e cacheado no startup dentro do `sheets.Client`. Se uma sheet não tiver exatamente uma tabela nativa, o startup falha. Valores são inseridos como `StringValue` (equivalente ao `ValueInputOption("RAW")` anterior).

## Alternativa considerada

`values.append` detecta o fim dos dados mas não estende a tabela nativa. `values.update` foi usado anteriormente sob a premissa incorreta de que auto-expandia tabelas nativas ao escrever na linha imediatamente abaixo — a documentação oficial não documenta esse comportamento e a premissa está errada.
