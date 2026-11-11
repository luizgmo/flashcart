# FlashCart

FlashCart é um projeto de portfólio que demonstra uma compra de produto único com reserva temporária de estoque, pagamento simulado e processamento assíncrono. Nenhum pagamento real acontece.

## Estado

Projeto local implementado: vitrine React, API Go, sessão anônima, reserva de estoque, pedidos idempotentes, pagamento simulado, timeline, expiração, outbox, Pub/Sub e assistente opcional. A infraestrutura GCP está preparada em Terraform, mas não foi aplicada. O estado preciso de validação está em [docs/execution.md](docs/execution.md).

## Stack planejada

Go, PostgreSQL, React/TypeScript, Docker Compose e GCP Cloud Run. O app não captura pagamentos reais. Veja [docs/deploy-gcp.md](docs/deploy-gcp.md) antes de provisionar recursos pagos.

## Executar localmente

Requisitos: Go 1.27.4, Node 24/npm e Docker Engine com Docker Compose. Copie `.env.example` para `.env`; os valores padrão são somente locais.

```sh
make setup
make up
make migrate
make seed
```

Em três terminais na raiz do projeto, inicie:

```sh
make serve-api
```

```sh
make serve-web
```

```sh
make serve-consumer
```

Em um quarto terminal, mantenha expiração e publicação de eventos ativas:

```sh
make maintenance-loop
```

```sh
make publish-outbox-loop
```

Abra `http://localhost:5173`. A API fica em `http://localhost:8080`; o consumidor local usa `http://localhost:8081`. Para parar os containers sem apagar os dados, use `make down`.

O assistente usa respostas determinísticas por padrão. Para ativar Gemini, configure `ASSISTANT_MODE=gemini` e `GEMINI_API_KEY` no `.env`; isso faz chamadas externas que podem gerar cobrança.

Consulte `docs/execution.md` para gates concluídos e pendências. O deploy GCP ainda não foi executado.

Com PostgreSQL migrado, produtos semeados e API ativa, rode `make e2e-api` para validar sessão, pedido idempotente, pagamento simulado, timeline e resposta do assistente. O roteiro cria um pedido de demonstração. A suite Playwright/axe está em `frontend/e2e`; execute `npm run test:e2e` em `frontend` com a API/Vite ativos e Chromium instalado. A CI roda esses fluxos no Chromium.

## Capturas

![Vitrine FlashCart em desktop](docs/screenshots/catalog-desktop.png)

[Vitrine em celular](docs/screenshots/catalog-mobile.png) · [Pedido pago](docs/screenshots/order-paid.png)

