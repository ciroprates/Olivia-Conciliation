# AppendRow usa fetch + update em vez de values.append

As abas ES e REJ usam Tabelas nativas do Google Sheets. O método `values.append` da Sheets API detecta o fim dos dados mas não estende a tabela nativa — a linha nova cai fora dos limites da tabela. A solução adotada é buscar as linhas atuais com `FetchRows`, calcular a próxima posição (`len(rows)+1`), e escrever com `values.update` diretamente nessa linha. Tabelas nativas do Google Sheets auto-expandem quando se escreve na linha imediatamente abaixo delas via API.

## Alternativa considerada

`batchUpdate` com `AppendCells` resolveria o mesmo problema em um único call à API, mas exige o sheet ID numérico (não o nome), o que adiciona complexidade desnecessária para o ganho obtido.
