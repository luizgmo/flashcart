# Registro de execução

Atualizado em 2026-11-11. O caminho local da aplicação, a infraestrutura Terraform em modo de validação, os cenários k6 e os fluxos de navegador foram executados. A CI agora inclui Chromium, fluxo end-to-end e axe-core. Nenhum recurso GCP foi criado.

## Ambiente e evidências

- Go 1.27.4: `go test -race ./...`, `go vet ./...` e `go build ./...` passaram durante a implementação. Nesta revisão final, o sandbox não permitiu completar novo download de dependências ausentes do cache; os pacotes independentes `config` e `orders` permanecem executáveis sem esses downloads.
- Frontend: `npm run build` passou; Playwright executou os dois testes no Chromium: catálogo/filtros/imagens/auditoria axe-core e reserva/pagamento/assistente. Resultado: 2 passaram. O assistente pode esperar até 8 segundos pelo Gemini antes do fallback; o teste aceita esse prazo.
- Compose: `docker compose config -q` passou. PostgreSQL e Pub/Sub emulator ficaram ativos para os testes locais; PostgreSQL usa a porta 5433 para não interferir em outro projeto.
- Migration Goose 001 aplicou com sucesso. Seed executado duas vezes resultou em 8 produtos e estoque total 252 nas duas execuções.
- Fluxo HTTP integrado passou incluindo sessão, pedido, replay idempotente, pagamento simulado, timeline e snapshot `PAID` do assistente.
- Eventos integrados: publisher enviou a outbox ao Pub/Sub emulator; consumer gravou notificações de pagamento/expiração. O job expirou reserva e restaurou estoque. Reentrega duplicada não criou notificações nem eventos processados adicionais.
- k6 executou catálogo, disputa da última unidade, escrita/pagamento e replay idempotente concorrente; todos os cenários passaram. Os números e limitações estão em `docs/load-report.md`.
- Terraform 1.11.4: `fmt -check -recursive`, `init -backend=false` e `validate` passaram. Nenhum plano contra projeto GCP foi gerado e nenhum recurso foi aplicado.
- Git independente em `flashcart/`; alterações ainda não foram commitadas.

## Marcos

| Marco | Estado | Evidência |
| --- | --- | --- |
| M0 Preparação | Concluído | Estrutura, decisões, configuração local e dependências documentadas. |
| M1 Persistência, sessão e catálogo | Concluído | Migração, seed idempotente e endpoints exercitados com PostgreSQL. |
| M2 Compra e consistência | Concluído | Idempotência, pagamento, expiração/restauração e disputa da última unidade exercitados. |
| M3 Eventos e execução local | Concluído | Outbox, Pub/Sub, notificações e reentrega duplicada exercitados localmente. |
| M4 Interface | Concluído | Vitrine React, checkout, pedidos, assistente, build e navegador validados. |
| M5 Assistente | Concluído | Resposta determinística exercitada; Gemini opcional tem quota, validação e fallback. |
| M6 CI e endurecimento | Concluído | Workflow inclui testes Go, build, banco, Chromium, E2E e axe-core; E2E local passou. |
| M7 Carga e revisão técnica | Concluído | Quatro cenários k6 medidos em banco descartável. |
| M8 Infraestrutura e runbooks | Concluído localmente | Terraform formatado e validado; runbooks de execução/documentação prontos. GCP não foi aplicado. |
| M9 Portfólio e auditoria final | Concluído | README, arquitetura, segurança, execução, carga e capturas de tela versionadas. |

## Limites de escopo

- Gemini continua opcional: sem chave, o assistente usa resposta determinística. A suíte E2E não depende de chamada paga.
- O modelo de pagamento é simulado e não movimenta dinheiro.
- Terraform foi validado localmente, sem plano ou `apply` em projeto GCP. A implantação exige projeto, região, domínio/segredos e análise de custo escolhidos pelo usuário.
- As métricas k6 refletem somente este ambiente local e não são promessa de capacidade em produção.
