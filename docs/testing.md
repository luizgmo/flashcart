# Testes

## Verificações implementadas

- `cd backend && go test -race ./...`: regras da fronteira de expiração, paginação, validação exata de Origin, headers HTTP e adaptador Gemini com transporte HTTP simulado.
- `cd backend && go vet ./... && go build ./...`.
- `cd frontend && npm ci && npm run build`.
- `docker compose -f compose.yaml config --quiet` valida a sintaxe/expansão do Compose; não inicia serviços.
- `load/flash-sale.js` disputa uma última unidade com 50 VUs; resultado medido em `docs/load-report.md`.
- `load/catalog.js` executou 20 VUs/60s: 1,200 respostas 200, 0 falhas e p95 de 4.39 ms.
- `load/write.js` executou 20 VUs/60s: 920 pedidos pagos, 0 falhas, p95 de criação 509.31 ms e de pagamento 257.96 ms.
- `load/idempotency.js` executou 20 requests concorrentes com uma chave: exatamente uma criação e 19 replays, 0 falhas.
- `make e2e-api` valida o fluxo HTTP principal usando Python padrão; cria um pedido pago simulado no banco configurado.
- `cd frontend && npm run test:e2e` percorre vitrine, filtros, checkout, pagamento, assistente e auditoria axe-core; exige API/Vite ativos e Chromium.
- `.github/workflows/ci.yml` sobe PostgreSQL, aplica migração/seed e executa os testes Go, build, Chromium, fluxo de compra e axe-core em cada PR e push para `main`.

## Escopo e limites

Migration, seed idempotente, criação/replay de pedido, pagamento, expiração/restauração de estoque, processamento e redelivery Pub/Sub, contenção da última unidade, leitura e escrita sustentadas e idempotência concorrente foram exercitados contra PostgreSQL e o emulador local. Playwright e axe-core passaram no Chromium. A CI não provisiona GCP nem depende de Gemini.

Métricas de carga são específicas do ambiente local e não devem ser usadas como SLO de produção. O pagamento é simulado.

