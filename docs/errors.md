# erros

toda resposta de erro é `application/problem+json` (RFC 7807), com `instance` carregando o
request id. o campo `type` aponta para esta página, com o status no fragmento.

## status http

### 400

corpo malformado ou parâmetro fora do tipo declarado no openapi.

### 404

rota inexistente, ou a fazenda não existe.

### 405

método errado para a rota.

### 409

a fazenda mudou no meio da escrita. repita a chamada.

### 415

content-type diferente de `application/json` num corpo com conteúdo.

### 422

o corpo passou no formato mas a ação não faz sentido. `detail` diz o quê:

| detail | o que fazer |
| --- | --- |
| `buy_upgrade precisa do campo auto` | mande `auto` com uma automação válida |
| `buy_tank precisa do campo tank_kind` | mande `tank_kind` |
| `essa acao precisa do campo tank_id` | mande `tank_id` |
| `acao desconhecida` | use um `kind` do enum |

### 500

erro interno. o corpo nunca traz o texto do erro original; o `instance` é o que liga a resposta
à linha de log do daemon.

## ação recusada

recusa **não** é erro http. a chamada volta 200 e o snapshot traz `last_outcome.applied = false`
com o motivo em `last_outcome.reason`. quando faltou dinheiro, `needed_cents` diz quanto a ação
custava.

| reason | significado |
| --- | --- |
| `no_such_tank` | o tanque não existe |
| `no_such_batch` | não há lote com esse id no tanque |
| `not_enough_cash` | caixa insuficiente, veja `needed_cents` |
| `not_enough_feed` | o tanque está sem ração no estoque |
| `tank_full` | o tanque já tem o máximo de lotes |
| `farm_full` | a fazenda já tem o máximo de tanques |
| `bad_amount` | quantidade zero ou negativa |
| `too_dense` | povoar passaria da densidade máxima do tanque |
| `already_owned` | esse tanque já tem essa automação |
| `not_enough_lifetime` | faturamento vitalício ainda não dá prestígio |
| `credit_limit` | o empréstimo passa do limite de crédito |
| `no_debt` | não há dívida para pagar |
| `nothing_sick` | não há doença nesse tanque para tratar |
| `not_broke` | recomeçar só vale quando a fazenda quebra de vez |
| `unknown_kind` | ação ou tipo desconhecido |
