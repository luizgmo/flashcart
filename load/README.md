# Cenários k6

Rode somente contra banco local/teste descartável. `catalog.js` executa leitura com 20 VUs por 60 segundos e pausa de 1 segundo por iteração. `flash-sale.js` inicia 50 sessões concorrentes numa única rodada contra um produto com exatamente uma unidade; cada VU cria uma chave idempotente nova. `write.js` cria e paga pedidos por 20 VUs durante 60 segundos. `idempotency.js` disputa 20 requisições com a mesma sessão e chave.

Antes do cenário, prepare por SQL um produto dedicado com `stock_available=1`, sem pedidos ligados a ele. Exporte o UUID como `PRODUCT_ID` e use o origin Vite configurado. O script não repõe o estoque nem limpa pedidos; refaça o banco de teste antes de repetir. Quotas de sessão e novos pedidos precisam permanecer dentro dos limites de demonstração.

```sh
k6 run -e BASE_URL=http://localhost:8080 load/catalog.js
k6 run -e BASE_URL=http://localhost:8080 -e ORIGIN=http://localhost:5173 -e PRODUCT_ID=UUID_DO_PRODUTO load/flash-sale.js
k6 run -e BASE_URL=http://localhost:8080 -e ORIGIN=http://localhost:5173 -e PRODUCT_ID=UUID_PRODUTO_COM_ESTOQUE load/write.js
k6 run -e BASE_URL=http://localhost:8080 -e ORIGIN=http://localhost:5173 -e PRODUCT_ID=UUID_DO_PRODUTO_COM_UMA_UNIDADE load/idempotency.js
```

Para `write.js` e `idempotency.js`, configure explicitamente `RATE_LIMIT_MODE=load` somente na API local/teste. Nunca habilite esse modo em produção; o backend recusa essa configuração com `APP_ENV=production`.

Esperado: uma criação, 49 conflitos de domínio por falta de estoque, zero erros técnicos e estoque final zero. O cenário marca HTTP 409 como resposta esperada para a disputa; erros técnicos e demais status continuam contando como falha. Guarde a saída integral e anote commit, hardware, duração e configurações em `docs/load-report.md`; não trate os limites de referência do plano como resultado medido.

