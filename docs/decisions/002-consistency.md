# Decisão 002 — consistência do pedido

A sessão é bloqueada ao criar pedido; a chave de idempotência é consultada antes de aplicar o efeito; o estoque é decrementado com `UPDATE ... WHERE stock_available >= quantity RETURNING`. Pedido, preço congelado, chave e timeline gravam na mesma transação. Pagamento e expiração bloqueiam o pedido, consultam o relógio do banco depois do lock e aplicam somente uma transição terminal.

## Consequência

PostgreSQL coordena concorrência e é fonte de verdade. A garantia deve ser medida com testes em conexões distintas; testes unitários das funções de tempo não provam isolamento do banco.

