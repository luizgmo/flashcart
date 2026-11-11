# Relatório de carga

## Contenção de última unidade — 2026-11-11

- Objetivo: confirmar que 50 pedidos concorrentes não vendem mais que uma unidade.
- Banco: PostgreSQL 17.6 isolado (`flashcart_load`), migration 001 aplicada; produto fixture iniciado com `initial_stock=1` e `stock_available=1`.
- Aplicação: Go 1.27.4, binário estático dentro da rede Docker local.
- Carga: Grafana k6 v2.3.0, 50 VUs, 50 iterações compartilhadas numa única rodada. Cada VU cria sessão e envia uma chave idempotente única.
- Máquina host: Linux x86_64, 8 CPUs lógicas, 7.7 GiB de memória reportada. Limites Docker dos containers não foram configurados.
- Código: árvore local sem commit no momento da execução; cenário em `load/flash-sale.js`.
- Comando equivalente: `k6 run -e BASE_URL=http://localhost:8080 -e ORIGIN=http://localhost:5173 -e PRODUCT_ID=10000000-0000-4000-8000-000000000001 load/flash-sale.js`.

Resultado medido:

- Pedidos criados: **1**.
- Conflitos por estoque insuficiente: **49**.
- Checagens: **50/50 passaram**.
- Falhas HTTP inesperadas: **0/100** (`http_req_failed = 0%`).
- Latência HTTP agregada: média **434.97 ms**, p95 **742.35 ms**, máximo **795.6 ms**.
- Duração da rodada: **0.9 s**; iterações **54.99/s**, requests **109.99/s**.
- Banco após a rodada: uma ordem e estoque disponível **0**.

Na primeira tentativa, o cenário descobriu espera circular do pool: a criação de pedido mantinha uma conexão transacional enquanto buscava outra conexão para a quota. Com `DB_MAX_CONNS=5`, as transações podiam ocupar o pool inteiro e esperar indefinidamente. A quota de tentativas agora é consumida antes de abrir a transação do pedido. Após a correção, o cenário repetido terminou em menos de um segundo com estoque consistente.

Esse resultado caracteriza somente a contenção local de última unidade. Não representa teste de throughput sustentado, produção, cloud, nem a meta de leitura/escrita sob 20 VUs.

## Leitura de catálogo — 2026-11-11

- Banco descartável: `flashcart_load`; aplicação e PostgreSQL 17.6 na rede Docker local.
- Carga: Grafana k6 v2.3.0, 20 VUs por 60 segundos, pausa de 1 segundo por iteração.
- Máquina host e código: mesmos dados do cenário acima; limites Docker não configurados.
- Requisições: **1,200**, todas HTTP 200; **0** falhas.
- Vazão: **19.94 req/s**.
- Latência: média **2.14 ms**, p95 **4.39 ms**, máximo **13.36 ms**.

Este é um resultado de catálogo em ambiente local e não deve ser extrapolado para GCP ou produção.

## Criação e pagamento — 2026-11-11

- Banco: `flashcart_load`, mesmo host/versões/fixture do perfil de contenção; produto de escrita iniciou com 2,000 unidades.
- Carga: Grafana k6 v2.3.0, 20 VUs por 60 segundos; cada iteração cria sessão, reserva uma unidade, paga e espera um segundo.
- Ambiente explicitamente configurado com `RATE_LIMIT_MODE=load`, permitido somente em local/teste.
- Requisições: **2,760**, todas esperadas; **0** falhas HTTP.
- Iterações completas: **920**; pedidos criados e pagos: **920**.
- Vazão: **45.19 requests/s**, **15.06 pedidos pagos/s**.
- Latência de criação: p95 **509.31 ms**; pagamento: p95 **257.96 ms**; agregada: p95 **383.58 ms**; máximo agregado **2 s**.
- Estado do banco: 920 pedidos PAID; estoque do fixture **1,080** de 2,000.

## Replay idempotente concorrente — 2026-11-11

- Banco: `flashcart_load`; produto isolado com uma unidade.
- Carga: Grafana k6 v2.3.0, 20 VUs e 20 iterações, todas usando a mesma sessão, chave e payload; modo `RATE_LIMIT_MODE=load`.
- Resultado: **1** criação `201`, **19** replays `200`, todos com `Idempotency-Replayed: true`; **0** falhas HTTP.
- Latência HTTP: média **389.65 ms**, p95 **492.61 ms**, máximo **500.55 ms**; 21 requests incluindo a criação da sessão.
- Estado do banco: exatamente **1** pedido ligado à chave e estoque final **0**.

