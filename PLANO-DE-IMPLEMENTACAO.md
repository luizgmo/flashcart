# FlashCart — plano completo de implementação

Estado: AGUARDANDO APROVAÇÃO. Este documento autoriza somente planejamento até o usuário aprová-lo. Após aprovação, será a especificação principal da implementação.

Data: 2026-11-11. Idioma do produto e da documentação: português do Brasil. Código, identificadores e commits: inglês. Executor pretendido pelo usuário: modelo Luna, esforço high. Este arquivo não troca configurações de modelo.

## 1. Objetivo, prioridade e resultado esperado

Construir uma aplicação demonstrável para portfólio de candidatura a Desenvolvedor Júnior na Shopee. A aplicação deve evidenciar Go, SQL, consistência sob concorrência, React, testes, processamento assíncrono, preparação para GCP e integração controlada com IA.

Prioridade de decisões: correção das regras de negócio > reprodução local > clareza do código > experiência de uso > operação em cloud > extensões. Não otimizar para quantidade de ferramentas, serviços ou linhas de código.

Entregas obrigatórias:

1. Monorepositório com aplicação funcional e histórico local de implementação, quando Git estiver disponível.
2. API e processos em Go; interface React/TypeScript; PostgreSQL; ambiente Docker Compose.
3. Compra de um produto por pedido, reserva temporária, pagamento simulado, expiração e notificações internas assíncronas.
4. Isolamento de pedidos por sessão anônima e proteção dos endpoints internos.
5. Assistente de pedidos com adaptador Gemini real e fallback determinístico identificado.
6. Testes automatizados, CI, testes de concorrência, testes de navegador e relatório de carga com resultados medidos.
7. Infraestrutura declarativa GCP, procedimento de implantação e estimativa de custos com data e premissas.
8. README, arquitetura, decisões técnicas, evidências visuais e roteiro de demonstração.

Não prometer emprego, volume de produção, ausência absoluta de falhas ou resultados de desempenho antes de medi-los.

## 2. Limites do projeto

Incluído:

- Catálogo de oito produtos fictícios, uma moeda BRL, estoque global e preço fixo por produto.
- Um produto e de uma a cinco unidades por pedido; até três pedidos pendentes por sessão.
- Reserva por exatamente 300 segundos contados pelo banco na criação do pedido.
- Simulação explícita de pagamento aprovado; nenhuma cobrança, chave Pix ou cartão real.
- Estados PENDING_PAYMENT, PAID e EXPIRED; notificações internas de pagamento e expiração.
- Catálogo, meus pedidos, detalhe do pedido, checkout e painel de explicação do pedido.
- Processos separados para manutenção, publicação da outbox e consumo de eventos.

Excluído nesta execução:

- Cadastro, senha, OAuth, recuperação de conta, acesso entre dispositivos e dados pessoais.
- Carrinho com múltiplos produtos, cupons, frete, entrega, comissão, vendedores e marketplace completo.
- Cancelamento, reembolso, pagamento recusado, cobrança real e webhook de adquirente.
- CRUD administrativo público, reposição pública de estoque e reset público da demonstração.
- Kubernetes, Redis, Kafka, event sourcing, saga, CQRS, GraphQL, WebSockets e busca vetorial.
- E-mail, SMS e mensagem externa. Notificações existem somente dentro da aplicação.
- Site separado de marketing, customização de domínio e publicação do currículo.

Não ampliar esse escopo para compensar dificuldade de outra etapa. Registrar ideias adicionais em docs/future-work.md.

## 3. Autonomia, ambiente e regras para o executor

- Trabalhar exclusivamente no diretório flashcart deste workspace. Não alterar projetos irmãos nem o Git do diretório pai.
- Ler instruções AGENTS.md aplicáveis e /home/luizgmo/.codex/RTK.md. Prefixar comandos shell com rtk; usar rtk proxy quando necessário.
- Não implementar antes da aprovação deste plano. Após aprovação, executar os marcos em ordem sem pedir confirmação para decisões já fechadas aqui.
- Não criar subagentes, salvo autorização explícita posterior. A execução foi desenhada para um único modelo.
- Não iniciar meta persistente de ferramenta de goals sem pedido explícito do usuário.
- Não criar repositório remoto, publicar, comprar serviços ou ativar recursos pagos apenas pela aprovação do desenvolvimento. A aprovação deste plano cobre implementação local e preparação do deploy. Implantação real exige autorização para gasto e projeto/conta identificados.
- Não ler credenciais de projetos irmãos, imprimir segredos ou usar contas já conectadas por suposição.
- Se Docker, navegador, rede ou instalação estiver indisponível, procurar alternativa equivalente dentro das permissões. Registrar o impedimento; não marcar a validação como aprovada nem trocar integração por mock para alegar sucesso.
- Não parar todo o projeto por ausência de Gemini ou GCP. Implementar adaptadores, testar com servidores HTTP controlados e concluir tudo que for independente dessas credenciais.
- Consultar documentação oficial para APIs e versões atuais; congelar versões exatas em arquivos de lock e ferramentas. Não usar tags latest em imagens.
- Progresso em docs/execution.md: marco, tarefas, comandos executados, resultados, impedimentos e próximo passo. Atualizar ao finalizar cada marco e antes de interrupção/compactação.
- Se descobrir contradição técnica real, fazer a menor correção que preserve requisitos, registrar a razão em docs/decisions e atualizar este documento. Mudanças de escopo, custos ou regras de negócio exigem decisão do usuário.
- Nunca afirmar que algo foi publicado, filmado, testado ou integrado ao provedor real sem evidência.

## 4. Ferramentas e uso das skills

O MCP Skill Issue já foi consultado: list_domains, recommend_skills, search_skills e get_skill. O ranking retornou também resultados irrelevantes; não instalar tudo.

Referências selecionadas do catálogo:

- https://github.com/samber/cc-skills-golang — princípios de Go, testes e observabilidade.
- https://github.com/wshobson/agents/tree/main/plugins/database-design/skills/postgresql — schema, constraints e índices.
- https://github.com/wshobson/agents/tree/main/plugins/backend-development/skills/api-design-principles — contratos HTTP.

Essas referências foram identificadas pelo catálogo; seus conteúdos completos ainda devem ser lidos antes da aplicação. Verificar caminhos e conteúdo na execução. Instalar somente skills pertinentes e revisadas, dentro do projeto, se necessário e permitido. Ausência de uma skill não bloqueia o desenvolvimento quando a documentação oficial atende à tarefa.

Usar a skill frontend-design disponível no ambiente ao implementar a interface. Ler instruções locais exigidas por qualquer ferramenta de sites antes de utilizá-la; este projeto é uma aplicação em repositório com backend e banco, não deve ser substituído por site estático em hospedagem de plugin.

## 5. Stack e estrutura

- Go: última versão estável suportada disponível no início da implementação; fixar toolchain e imagem no marco M0. net/http, log/slog, pgx/v5, biblioteca UUID mantida e biblioteca oficial de Pub/Sub.
- SQL explícito em arquivos ou constantes curtas no pacote de persistência. Sem ORM e sem sqlc nesta versão.
- PostgreSQL 17, último patch disponível, fixado na imagem. Usar a mesma major em integração e Cloud SQL.
- Migrations: goose, arquivos SQL incrementais com up/down; execução por comando próprio. Não migrar silenciosamente no startup da API.
- Node.js LTS suportado, npm com package-lock.json, React estável, TypeScript strict, Vite, React Router, TanStack Query, Zod, CSS próprio e lucide-react para ícones.
- OpenAPI 3.1 em arquivo versionado. Tipos de API do frontend gerados com openapi-typescript; sem cliente HTTP enorme gerado.
- Testes: testing/httptest em Go, Vitest + Testing Library, Playwright e axe-core; k6 para carga.
- Terraform para GCP; sem provisionamento automático em CI.

Estrutura alvo:

```text
flashcart/
  PLANO-DE-IMPLEMENTACAO.md
  README.md
  Makefile
  compose.yaml
  .env.example
  .github/workflows/ci.yml
  backend/
    go.mod, go.sum
    cmd/flashcart/main.go
    internal/{config,httpapi,sessions,catalog,orders,events,assistant,storage,jobs}/
    migrations/
    tests/integration/
  frontend/
    src/{app,components,features,lib,styles}/
    public/products/
    package.json, package-lock.json
  api/openapi.yaml
  e2e/
  load/
  infra/terraform/
  scripts/
  docs/{decisions,evidence}/
```

Não criar camada genérica de repository para tudo. Interfaces pequenas apenas nas fronteiras que exigem substituição em teste ou em produção: provedor IA, publicador de eventos e configuração do transporte. SQL e transações ficam explícitos.

Um único binário Go com subcomandos: serve, migrate, seed, maintenance, publish-outbox, consume-pull e consume-push. Cada subcomando inicializa apenas as dependências necessárias. O servidor público nunca registra rotas internas do consumidor.

## 6. Arquitetura de execução

Local, com make up:

1. PostgreSQL com healthcheck e volume persistente.
2. Emulador oficial de Pub/Sub com imagem fixada.
3. Serviço de inicialização aplica migrations, seed idempotente e cria tópico/subscrição local.
4. Web/API serve o build React e /api/v1 no mesmo origin.
5. Maintenance executa expiração a cada cinco segundos localmente; publish-outbox roda a cada dois segundos; ambos encerram corretamente com SIGTERM.
6. Consumer local usa pull no emulador e chama o mesmo handler de domínio usado pelo push em cloud.

Durante desenvolvimento, Vite faz proxy de /api para a API; não abrir CORS genérico. O modo de desenvolvimento usa o origin do Vite configurado explicitamente.

GCP:

- Cloud Run web público: API e arquivos estáticos, sem loop de manutenção em background.
- Cloud Run consumer privado: somente POST /internal/pubsub e healthcheck.
- Cloud Run Job maintenance: expiração e limpeza em execução finita.
- Cloud Run Job publish-outbox: publicação em execução finita.
- Cloud Scheduler dispara os dois jobs a cada minuto em UTC.
- Pub/Sub push autenticado invoca consumer; DLQ para mensagens não processadas.
- Cloud SQL PostgreSQL compartilhado pelos componentes, com propriedade das tabelas documentada. Não apresentar como microsserviços com bancos independentes.
- Artifact Registry, Secret Manager e Cloud Logging.

Notificações são eventualmente consistentes. Em cloud podem levar mais de um minuto conforme fila, início do job e retentativas. A confirmação do pagamento e a reserva não dependem do broker.

## 7. Regras de domínio e tempo

IDs são UUID v4. Tempos são TIMESTAMPTZ no banco e RFC3339 UTC na API. Valores monetários são inteiros em centavos; quantidade é inteiro entre 1 e 5. Currency sempre BRL.

Estoque disponível diminui ao reservar. Ao pagar, continua diminuído; ao expirar, é devolvido. Sem reposição, a identidade obrigatória por produto é:

```text
stock_available + soma(quantidade de PENDING_PAYMENT e PAID) = initial_stock
```

Não existe estado intermediário visível de reserva. Pedido, redução de estoque e registro de idempotência são uma única transação.

Transições permitidas:

| Estado atual | Ação e condição | Resultado |
|---|---|---|
| inexistente | criar com estoque e limites válidos | PENDING_PAYMENT |
| PENDING_PAYMENT | pagar quando db_now < expires_at | PAID |
| PENDING_PAYMENT | pagar quando db_now >= expires_at | EXPIRED, devolver estoque, responder 409 |
| PENDING_PAYMENT | job com db_now >= expires_at | EXPIRED, devolver estoque |
| PAID | repetir pagamento | PAID, sem efeito adicional |
| EXPIRED | tentar pagamento | 409, sem efeito adicional |

PAID e EXPIRED são terminais. Não existe cancelamento. A comparação na fronteira do prazo é estrita: igualdade já está expirada.

Consultar clock_timestamp() no PostgreSQL depois de adquirir os locks relevantes. Não usar timestamp de início da transação para decidir pagamento após longa espera por lock. O horário do navegador serve apenas para apresentação.

GET não altera pedidos. Um pedido vencido pode permanecer armazenado como PENDING_PAYMENT até o job ou uma tentativa de pagamento processá-lo. A resposta contém can_pay=false e reservation_elapsed=true nesse intervalo; a UI mostra "Prazo encerrado; liberação em processamento". Não inventar estado EXPIRED já persistido. server_time acompanha respostas de detalhe para ajustar a contagem do navegador.

Preço e nome do produto são copiados para o pedido e permanecem imutáveis. O cliente nunca envia preço, status, session_id ou vencimento.

## 8. Sessão anônima e isolamento

POST /api/v1/session, body {}, cria sessão apenas quando não existe cookie válido. Com sessão válida, retorna a mesma sessão sem renovar o vencimento. Token aleatório criptográfico de 32 bytes, base64url; guardar somente SHA-256 no banco.

Cookie flashcart_session: HttpOnly, SameSite=Lax, Path=/, Secure em HTTPS, validade de 24 horas. HTTP sem Secure somente em ambiente local explícito. Não armazenar token no localStorage, URL ou logs.

Sem cookie ou com sessão vencida, endpoints protegidos retornam 401; a UI oferece iniciar nova sessão e avisa que pedidos antigos não serão recuperados. Não há mecanismo de recuperação e isso deve constar da demo.

Pedido de outra sessão e pedido inexistente retornam o mesmo 404. Verificar proprietário em consultas e mutações antes de fornecer qualquer dado ao provedor de IA. Incluir filtro session_id na consulta SQL; UUID não substitui autorização.

Toda mutação pública requer Content-Type application/json e Origin igual ao APP_ORIGIN configurado. Rejeitar Origin ausente, null ou diferente com 403; chamadas CLI/teste devem fornecê-lo. Não liberar wildcard. GET não modifica estado. Exceção: consumer interno autenticado usa contrato Pub/Sub e IAM, sem cookies.

Quantidade de pedidos, quotas e sessões não são identidade forte: o produto é uma demo anônima, e não uma plataforma autenticada de produção.

## 9. Schema obrigatório

Todas as FKs com ON DELETE RESTRICT salvo indicação explícita. IDs definidos pelo servidor. Não remover produtos/pedidos durante a operação normal.

### sessions

- id UUID PK; token_hash BYTEA UNIQUE NOT NULL, comprimento 32.
- created_at, expires_at TIMESTAMPTZ NOT NULL; expires_at > created_at.

### products

- id UUID PK estável no seed; slug TEXT UNIQUE; name TEXT; description TEXT; category TEXT; image_path TEXT.
- price_cents BIGINT CHECK > 0 e <= 100000000.
- initial_stock INTEGER CHECK >= 0; stock_available INTEGER CHECK BETWEEN 0 AND initial_stock.
- created_at TIMESTAMPTZ NOT NULL. Todos os campos descritos são NOT NULL.

### orders

- id UUID PK; session_id FK sessions; product_id FK products.
- product_name TEXT; quantity INTEGER CHECK BETWEEN 1 AND 5; unit_price_cents BIGINT CHECK > 0.
- total_cents BIGINT, calculado no banco como unit_price_cents * quantity (coluna gerada armazenada).
- status TEXT CHECK IN ('PENDING_PAYMENT','PAID','EXPIRED').
- created_at, expires_at, updated_at NOT NULL; paid_at e expired_at anuláveis.
- CHECK expires_at > created_at; CHECK de status exige: pending sem paid_at/expired_at; paid com paid_at e sem expired_at; expired com expired_at e sem paid_at.
- Índice (session_id, created_at DESC, id DESC).
- Índice parcial (expires_at, id) WHERE status='PENDING_PAYMENT'.

### idempotency_keys

- session_id FK; key UUID; request_hash BYTEA tamanho 32; order_id UUID FK orders UNIQUE NOT NULL; created_at.
- PK (session_id, key). Registro dura tanto quanto o pedido; não aplicar TTL que permita duplicar compra antiga.
- Hash de representação canônica do payload validado, product_id normalizado + quantity; não do JSON bruto.

### order_events

- id UUID PK; order_id FK; type CHECK IN ('ORDER_CREATED','ORDER_PAID','ORDER_EXPIRED'); occurred_at.
- UNIQUE(order_id,type). Usado para timeline; inserido na transação da mudança correspondente.

### outbox_events

- id UUID PK; order_id FK; type CHECK IN ('ORDER_PAID','ORDER_EXPIRED'); schema_version INTEGER CHECK = 1.
- payload JSONB NOT NULL; created_at; published_at nullable; attempt_count INTEGER DEFAULT 0 CHECK >=0; available_at.
- lease_token UUID nullable, lease_until TIMESTAMPTZ nullable; last_error TEXT nullable (sem segredos; truncar em 500 caracteres).
- UNIQUE(order_id,type); índice parcial (available_at,created_at) WHERE published_at IS NULL.

### processed_events e notifications

- processed_events: event_id UUID PK, processed_at TIMESTAMPTZ.
- notifications: id UUID PK; event_id UUID UNIQUE FK processed_events; order_id FK orders; kind TEXT CHECK IN ('ORDER_PAID','ORDER_EXPIRED'); message TEXT; created_at.
- Mensagem é template fixo local; sem chamada de IA no consumer.
- Índice notifications(order_id,created_at).

### quota_buckets

- scope TEXT, subject_hash TEXT, action TEXT, bucket_start TIMESTAMPTZ, count INTEGER CHECK >=0.
- PK(scope,subject_hash,action,bucket_start). subject_hash nunca contém cookie ou IP bruto.
- Contadores com INSERT ... ON CONFLICT DO UPDATE condicionado a count < limite e RETURNING. Consumo de quota é confirmado separadamente da compra para tentativas inválidas também contarem.

Não incluir tabela de pagamentos reais: a operação de simulação e sua idempotência são representadas pela transição única do pedido. Não guardar payload de provedor financeiro fictício.

## 10. Algoritmos transacionais obrigatórios

Isolamento READ COMMITTED. Queries parametrizadas. Transações curtas; chamadas HTTP externas nunca dentro de transação de negócio.

### Criar pedido

1. Validar sessão, Origin, JSON, UUIDs e quantidade. Aplicar quota de tentativas.
2. BEGIN; adquirir lock da linha sessions da sessão válida (SELECT FOR UPDATE). Isso serializa criação dentro da mesma sessão e torna o limite de pendências e a chave idempotente corretos.
3. Procurar idempotency_keys por sessão/chave. Existindo, comparar hash: diferente => 409 IDEMPOTENCY_CONFLICT; igual => carregar pedido atual e retornar 200, Idempotency-Replayed: true, sem novo efeito. Replay não renova reserva e funciona mesmo após expiração/pagamento. Finalizar transação antes da resposta.
4. Contar PENDING_PAYMENT da sessão. Se houver três, responder 409 PENDING_ORDER_LIMIT; reservas vencidas ainda não processadas contam até serem liberadas.
5. Consumir quota global diária de pedidos novos na mesma transação, depois do lock da sessão e antes do produto; esgotada => 429. Depois executar UPDATE products SET stock_available=stock_available-quantity WHERE id=... AND stock_available >= quantity RETURNING name,price_cents. Zero linhas: distinguir produto inexistente (404) de estoque insuficiente (409). Rollback também desfaz o consumo da quota de pedidos novos.
6. Após lock/update do produto, consultar clock_timestamp(); inserir pedido com expires_at=db_now+300s, evento ORDER_CREATED e idempotency_keys. COMMIT.
7. Retornar 201, Location e Idempotency-Replayed: false.

Qualquer falha antes do commit reverte tudo. Somente criações bem-sucedidas consomem a chave; erros de validação/estoque não são cacheados e podem ser repetidos com a mesma chave. Essa política deve constar do OpenAPI e README.

### Simular pagamento

1. Validar acesso e quota. BEGIN; SELECT pedido da sessão FOR UPDATE.
2. Se PAID: COMMIT, retornar 200 com estado atual; nenhuma nova notificação/evento.
3. Se EXPIRED: encerrar e retornar 409 ORDER_EXPIRED.
4. Se pendente, consultar clock_timestamp(). Se vencido: mudar para EXPIRED, atualizar produto devolvendo quantity, gravar ORDER_EXPIRED em timeline e outbox, COMMIT e então retornar 409 ORDER_EXPIRED. Não dar rollback nessa resposta de domínio, pois a expiração foi efetivada.
5. Se dentro do prazo: mudar para PAID, definir paid_at, inserir timeline/outbox ORDER_PAID e COMMIT. Retornar 200.

Não é necessário Idempotency-Key no pagamento: a transição monotônica do pedido torna a operação idempotente. Não criar event_id de pagamento fornecido pelo navegador.

### Expirar reservas

- Buscar IDs candidatos de até 50 pendências vencidas, ORDER BY expires_at,id, sem manter transação ou lock do lote.
- Para cada pedido, usar transação individual, reler estado e clock_timestamp após lock e, se ainda elegível, expirar, devolver estoque e gravar timeline/outbox no mesmo commit.
- A transação individual usa SELECT ... FOR UPDATE SKIP LOCKED para o ID candidato; se estiver bloqueado ou não elegível, pular. Não manter locks de um lote enquanto se processam outros pedidos.
- Nunca tocar em PAID; repetir o job não muda estoque novamente.
- Lock de pedido precede lock de produto. Criação só trava a sessão e o produto; não bloqueia pedidos existentes. Manter essa ordem nos demais fluxos.
- Limitar rodada a 1000 pedidos ou 40 segundos. Retornar contadores e sair; próximo agendamento continua.
- Não usar sleeps reais nos testes de expiração: fixtures colocam expires_at no passado, preservando created_at anterior; testar igualdade com função de decisão pura e teste de integração antes/depois da fronteira.

Retentar deadlock/serialization failure no máximo duas vezes com pequeno jitter, somente após rollback completo. Esgotado o limite, 503 RETRYABLE_CONFLICT; nunca retentar commit de resultado desconhecido sem consultar chave/estado idempotente.

## 11. Contrato HTTP público

Prefixo /api/v1. JSON em snake_case. Strings limitadas, desconhecidos rejeitados nas mutações. Body público máximo 16 KiB; pergunta da IA até 500 caracteres Unicode. Sem HTML retornado pela IA.

Erro padrão:

```json
{"error":{"code":"INSUFFICIENT_STOCK","message":"Estoque insuficiente.","request_id":"uuid","fields":{}}}
```

fields é opcional e contém erros de validação, nunca SQL, stack trace ou segredo. Status: 400 payload inválido; 401 sessão; 403 Origin; 404 recurso não acessível; 409 conflito de domínio; 413 body; 415 media type; 429 quota; 500 inesperado; 503 indisponibilidade transitória. 429 inclui Retry-After em segundos.

| Método e rota | Acesso | Entrada / saída |
|---|---|---|
| GET /healthz | público | 200 {status:"ok"}; sem consulta ao banco |
| GET /readyz | público | ping banco com timeout 1s; 200 ou 503, sem detalhes internos |
| GET /api/v1/config | público | currency, reservation_ttl_seconds, assistant_mode e demo=true; nenhum segredo |
| POST /api/v1/session | Origin válido | {}; 201 nova/200 existente; {expires_at,server_time}, Set-Cookie somente quando nova |
| GET /api/v1/products | público | page default 1, page_size default 12 max 24, category opcional enum do seed; {items,page,page_size,total} |
| GET /api/v1/products/{id} | público | produto ou 404 |
| POST /api/v1/orders | sessão + Origin | header Idempotency-Key UUID obrigatório; {product_id,quantity}; 201 novo/200 replay |
| GET /api/v1/orders | sessão | page default 1, page_size default 10 max 20; {items,page,page_size,total}, ordenação created_at DESC,id DESC |
| GET /api/v1/orders/{id} | dono | pedido completo com timeline, notifications e server_time |
| POST /api/v1/orders/{id}/simulate-payment | dono + Origin | {}; 200 PAID ou 409 ORDER_EXPIRED |
| POST /api/v1/orders/{id}/assistant | dono + Origin | {question}; 200 com resposta e modo; sem streaming |

Produto: id, slug, name, description, category, image_path, price_cents, currency, stock_available. Não retornar initial_stock na API pública.

Pedido: id, product_id, product_name, quantity, unit_price_cents, total_cents, currency, status, created_at, expires_at, paid_at, expired_at, can_pay, reservation_elapsed. Campos paid_at/expired_at sempre existem e são null quando não aplicáveis. No detalhe o objeto order acrescenta timeline [{type,occurred_at}] e notifications [{id,kind,message,created_at}]. Não expor session_id, hash de token ou chave de idempotência. Ordenar timeline e notificações por tempo e id. Detalhe e mutações de pedido retornam {order,server_time}; mutações incluem o mesmo objeto completo do detalhe. Lista retorna {items,page,page_size,total,server_time} com pedidos resumidos sem timeline/notifications. Detalhe de produto retorna {product}; lista de produtos mantém o envelope da tabela.

Resposta do assistente: {answer,mode,fallback_reason,snapshot}. mode enum gemini|template|fallback; fallback_reason null ou enum provider_unavailable|invalid_response|global_quota. snapshot={order_id,status,total_cents,currency,expires_at,can_pay,observed_at}. template é modo configurado sem provedor; fallback indica que o modo gemini estava ativo e não foi possível usá-lo. Nunca devolver mensagem bruta de erro do provedor. Sessão/config/produtos mantêm envelopes próprios da tabela; não envelopar toda a API num campo data.

Códigos de erro mínimos: INVALID_REQUEST, SESSION_REQUIRED, ORIGIN_FORBIDDEN, RESOURCE_NOT_FOUND, IDEMPOTENCY_KEY_REQUIRED, IDEMPOTENCY_CONFLICT, INSUFFICIENT_STOCK, PENDING_ORDER_LIMIT, ORDER_EXPIRED, RATE_LIMITED, PAYLOAD_TOO_LARGE, UNSUPPORTED_MEDIA_TYPE, RETRYABLE_CONFLICT e INTERNAL_ERROR. OpenAPI deve enumerar código e status de cada operação.

Paginação inválida retorna 400; página além do final retorna items=[]. Identificador malformado em recurso retorna 400; UUID válido sem acesso retorna 404.

Cache-Control: no-store em sessão/pedidos/assistente/config. Catálogo sem cache HTTP na primeira versão para evitar confusão de estoque. Assets com hash têm cache imutável; index.html sem cache prolongado.

## 12. Quotas e proteção da demo

Quotas aplicadas atomicamente no PostgreSQL para funcionar com múltiplas instâncias. Janelas fixas UTC, sem dependência de Redis.

- Criação de sessão: global 100 por hora e 500 por dia.
- Tentativas de criação de pedido: 10 por minuto por sessão e 20 por dia por sessão.
- Novos pedidos, além das tentativas: global 500 por dia; consumir apenas para criação nova dentro da transação, não em replay.
- Tentativas de pagamento: 20 por minuto por sessão.
- Assistente: 5 por minuto e 20 por dia por sessão; chamadas reais ao modelo: máximo global de 100 por dia.
- Endpoints de leitura: limitador por instância de 120 req/min por sessão, ou por IP quando não há sessão. Esse limite é apenas proteção básica; não alegar limite global distribuído de leitura.

Valores configuráveis com esses defaults; modo de carga deve habilitar quotas altas somente no ambiente local/teste, nunca automaticamente em produção. Manter ordem fixa de aquisição dos contadores globais para evitar deadlocks. Excesso global da IA retorna fallback sem consumir provedor; excesso por sessão retorna 429.

Evitar confiar cegamente em X-Forwarded-For. Local usar RemoteAddr; cloud documentar a cadeia de proxy da plataforma e hash do endereço com salt secreto se houver uso para logs/limitador. Não persistir IP bruto.

Sem endpoint de reset público. Seed usa INSERT ON CONFLICT DO NOTHING; não repõe estoque ao reiniciar. Reset somente make reset-local, confirma ambiente local e opera apenas sobre o volume Compose deste projeto, com confirmação explícita no comando. Nenhum script de reset pode apontar para Cloud SQL.

## 13. Outbox, broker e notificações

Envelope de evento publicado, JSON UTF-8:

```json
{"event_id":"uuid","schema_version":1,"type":"ORDER_PAID","order_id":"uuid","occurred_at":"RFC3339 UTC"}
```

Não incluir sessão, token, pergunta ou dado pessoal. Atributos Pub/Sub event_id e type podem duplicar os campos para diagnóstico. Deduplicação usa event_id do domínio; não o messageId do broker.

Publicador:

1. Claim de até 50 eventos não publicados e elegíveis, usando FOR UPDATE SKIP LOCKED e lease de 60s com lease_token aleatório; incrementar attempt_count e confirmar claim.
2. Publicar fora da transação com timeout de 10s por mensagem. Paralelismo máximo 5.
3. Em sucesso, marcar published_at apenas se lease_token ainda for o mesmo. Em falha, limpar lease e definir available_at com backoff min(2^attempt,300) segundos mais jitter; limitar tamanho do erro.
4. Lease vencido pode ser reclamado. Falha após envio antes de marcar produz duplicata esperada.
5. Após 10 tentativas, continuar com backoff máximo e log de erro visível; não descartar evento. Runbook descreve inspeção e nova tentativa.
6. Rodada finita de até 500 eventos ou 40s. Não iniciar novo envio perto do deadline.

Consumer:

- Local: pull no emulador; cloud: push wrapped padrão, decodificar message.data base64.
- Tamanho máximo do envelope HTTP 64 KiB; schema_version=1; type enum e UUIDs válidos.
- BEGIN; INSERT processed_events ON CONFLICT DO NOTHING RETURNING. Sem inserção => duplicata, COMMIT e ACK.
- Validar evento contra outbox persistida (id, tipo, pedido, versão e payload). Ausência/incompatibilidade: rollback e falha; sem notificação inventada.
- Inserir notificação com template fixo e COMMIT. Somente depois fazer ACK/HTTP 204.
- Falha transitória de banco retorna 503/nack. Payload inválido retorna 400/nack, permitindo redelivery e encaminhamento posterior à DLQ. Não considerar 4xx como ACK.
- DLQ com máximo de tentativas configurado em 5, reconhecendo que o broker trata esse limite como aproximado; retention de 7 dias. Dar permissões necessárias à service agent e documentar replay manual após corrigir causa.

Notificações são internas, sem efeitos externos. Ingestão e notificação exatamente uma vez no banco por event_id; transporte é pelo menos uma vez. Não declarar exactly-once end-to-end.

## 14. Assistente de IA

Função: responder perguntas sobre o pedido da sessão, em português, usando snapshot autorizado de status, produto, valores, prazo e timeline. Não recomendar produtos, decidir pagamentos, chamar SQL arbitrário ou executar ferramentas.

Adaptador escolhido: Gemini via REST ou SDK Go oficial se estiver estável e adequado. Modelo definido por GEMINI_MODEL; selecionar um modelo Flash estável disponível no momento da implementação e registrar versão/data. Nunca fixar nome de modelo presumido sem verificar documentação. Chave somente backend/Secret Manager.

ASSISTANT_MODE=template por padrão local e sem credenciais; gemini somente configurado explicitamente. Implementar de verdade os dois modos:

- template: resposta determinística conforme estado e assunto suportado; rotulada "Explicação automática — sem IA conectada".
- gemini: chamada real com timeout total de 8s, sem retentativa automática, no máximo 384 tokens de saída; falha, quota global, JSON inválido ou resposta incompatível => template com motivo categórico seguro.

Antes da chamada, buscar pedido com filtro de sessão e capturar snapshot único. Não manter transação enquanto chama provedor. Não persistir conversa, prompt ou resposta livre. Logs só de latência, modo, erro categorizado e uso de tokens quando disponível.

Prompt separa instruções de dados; pergunta do usuário é conteúdo não confiável. Esclarecer que descrições de produto também são dados. Structured output exige {answer:string,order_id:string,status:enum}. Conferir id/status contra snapshot, máximo de 1200 caracteres, ausência de URLs e ausência de caracteres de controle. Escapar saída pelo React; nunca usar dangerouslySetInnerHTML.

O snapshot é exibido em card determinístico ao lado da explicação, com horário de consulta e aviso de que o texto gerado pode errar. Não alegar garantia contra alucinação por usar JSON Schema. Se o pedido mudar durante a chamada, a resposta continua identificada como snapshot daquele horário.

Fallback cobre: pedido pendente no prazo, pendente vencido aguardando liberação, pago, expirado, prazo, valor e pergunta fora do escopo. Pergunta fora do escopo recebe orientação para consultar somente esse pedido. Sem devolver detalhes de pedidos mencionados no texto do usuário.

Testes unitários com httptest verificam request real ao adaptador, timeout, 429/500, JSON malformado, status/id incompatíveis e prompt injection. Smoke com provedor real é opcional e só ocorre com credenciais e autorização de uso; sem isso o relatório diz "adaptador validado com servidor controlado; chamada real pendente".

## 15. Interface e direção visual

Produto em pt-BR, moeda com Intl.NumberFormat('pt-BR'), horários locais com Intl.DateTimeFormat; datas da API continuam UTC. Não imitar marca, nome, logo ou identidade da Shopee.

Direção: loja editorial de tecnologia, fundo marfim (#F5F2EB), texto carvão (#1C2420), verde escuro (#174D3B), destaque amarelo (#EBCB59). A combinação final deve passar contraste WCAG AA; amarelo não deve ser texto claro em branco. Títulos Bricolage Grotesque, corpo Source Sans 3, fontes auto-hospedadas com licença incluída e fallback. Sem gradiente roxo, hero genérico de SaaS ou dependência de fotos remotas.

Oito produtos fictícios de acessórios: teclado, mouse, fone, hub USB, webcam, suporte, luminária e caixa de som. Criar ilustrações vetoriais próprias e simples em SVG local, com proporção 4:3. Esses assets não exigem gerador de imagens. Seeds têm UUIDs estáveis, nomes, categoria, preço, descrição e estoque inicial entre 20 e 100. Dados de teste e carga são fixtures separadas.

Rotas:

- /: catálogo, título "Poucas unidades. Compra sem complicação.", aviso visível "Demonstração: nenhum pagamento real", filtros Tudo/Áudio/Periféricos/Acessórios e grade de produtos.
- /products/:id: detalhe, quantidade 1–5, preço total e botão "Reservar por 5 minutos". Este é o checkout, sem coleta de endereço.
- /orders: lista da sessão, status, valores e links. Sem sessão, explicar e oferecer iniciá-la.
- /orders/:id: status em destaque, countdown, resumo, timeline, notificações e assistente. Botão "Simular pagamento aprovado" apenas quando can_pay=true.
- Página 404 e erro global com opção de tentar novamente.

Header com marca FlashCart, Catálogo e Meus pedidos. Rodapé com link para "Sobre a demonstração", modal explicando limites e arquitetura em linguagem simples; não ocupar o fluxo de compra com detalhes de implementação.

Layouts obrigatórios em 360, 768 e 1440px, sem overflow horizontal. Desktop: grid de três/quatro cards conforme largura; mobile: uma coluna, ações confortáveis com alvo >=44px. Detalhe do pedido desktop usa resumo e painel lateral; mobile empilha conteúdo.

Estados por tela: skeleton, vazio, erro recuperável, estoque esgotado, sessão vencida, pedido não acessível, submissão em andamento, timeout de resultado desconhecido, assistente template/IA/fallback. Não criar avaliações, compradores ao vivo, métricas ou depoimentos fictícios.

No primeiro clique em reservar, a UI chama POST /session se ainda não possui sessão validada; só então envia a criação. Não criar nova sessão em todo carregamento de catálogo. Criar Idempotency-Key no primeiro clique de confirmar reserva e mantê-la, junto com payload, em sessionStorage até resultado definitivo. Duplo clique e retry de rede reutilizam chave. Em timeout, oferecer "Verificar reserva" repetindo o mesmo POST com a mesma chave; não gerar uma chave nova automaticamente. Alterar quantidade/produto após erro definitivo inicia nova tentativa com nova chave. Depois de sucesso, limpar tentativa salva. Não expor token da sessão nesse storage. Se a sessão vencer com resultado de compra desconhecido, não reenviar a compra em uma sessão nova: avisar que o pedido antigo não pode ser recuperado e exigir nova ação consciente do visitante.

Pedido pendente: polling a cada 3s com aba visível; terminal: uma consulta a cada 5s por no máximo 90s até aparecer notificação esperada, depois parar. Pausar em aba oculta, retomar no foco; backoff para erros, parar em 401/404. Não consultar assistente automaticamente.

Countdown usa expires_at e diferença server_time/relógio local; ao zerar, desabilitar botão e refazer consulta. Servidor permanece autoridade. Mostrar skeleton sem movimento excessivo e respeitar prefers-reduced-motion.

Acessibilidade: HTML semântico, labels, navegação por teclado, foco visível, dialog com foco contido/restaurado, aria-live educado só para mudanças de estado. Não anunciar countdown a cada segundo. Contraste AA, mensagens com texto além de cor, SVG decorativo aria-hidden ou texto alternativo quando informativo.

## 16. Configuração, operação e segurança técnica

.env.example sem segredos reais, com descrição de cada variável:

- APP_ENV=local|test|production; APP_ORIGIN; PORT=8080; DATABASE_URL; DB_MAX_CONNS=5.
- DATABASE_URL é usado somente local/teste. Em produção, configurar DB_HOST=/cloudsql/INSTANCE_CONNECTION_NAME, DB_NAME, DB_USER e DB_PASSWORD (Secret Manager); compor pgx.Config em memória, sem logar DSN. Jobs migrate usam DB_USER/DB_PASSWORD próprios. Definir INSTANCE_CONNECTION_NAME nos recursos que montam a conexão Cloud SQL.
- RESERVATION_TTL_SECONDS=300, permitido alterar somente em testes; produção exige 300.
- ASSISTANT_MODE=template|gemini; GEMINI_API_KEY; GEMINI_MODEL.
- GOOGLE_CLOUD_PROJECT; PUBSUB_TOPIC; PUBSUB_SUBSCRIPTION; PUBSUB_EMULATOR_HOST (somente local/test).
- JOB_MAX_SECONDS=40; LOG_LEVEL=info; RATE_LIMIT_MODE=normal|load (load proibido em production).
- COOKIE_SECURE derivado do ambiente/origin; não aceitar false em produção.

Config inválida impede startup com mensagem segura. Em cloud, usar socket Unix montado pela integração oficial Cloud Run/Cloud SQL, com conexão remota cifrada pelo proxy gerenciado. Desabilitar TLS somente no trecho de socket local ao contêiner é permitido; nunca configurar conexão TCP pública ao banco com sslmode=disable. Não expor authorized networks 0.0.0.0/0. TCP direto sem conector fica fora desta versão.

HTTP: ReadHeaderTimeout 5s, ReadTimeout 10s, WriteTimeout 20s, IdleTimeout 60s, shutdown gracioso até 15s. Contexto de DB com prazo por operação e cancelamento ao desconectar cliente. Headers de segurança: nosniff, frame-ancestors 'none', Referrer-Policy, CSP compatível com assets locais e sem scripts inline, HSTS somente HTTPS produção.

Logs JSON via slog com timestamp, severity, request_id gerado/validado pelo servidor, rota normalizada, método, status e latência. Não registrar body, cookie, chave Gemini, DSN, pergunta ou SQL com valores sensíveis. Evitar usar UUID do pedido como label de métrica.

Maintenance limpa quota_buckets com mais de 48h; sessões expiradas sem pedidos podem ser removidas; pedidos e sessões referenciadas não são apagados automaticamente nesta versão. Documentar crescimento e reset local. Jobs emitirão contadores expired/published/failed e idade da outbox pendente mais antiga.

## 17. GCP e implantação reproduzível

Terraform com versões de providers fixadas, variables, outputs e terraform.tfvars.example sem segredos. Região default southamerica-east1; região alterável em uma variável, aplicada a todos os recursos compatíveis. Verificar suporte e custos antes de apply.

Recursos:

- Artifact Registry Docker.
- Cloud SQL PostgreSQL 17, edição Enterprise, tier db-f1-micro para demonstração, disponibilidade zonal, disco inicial SSD 10 GiB, backups diários 7 dias, deletion_protection=true. Verificar disponibilidade regional e de versão no M0/M8. Caso esse tier não seja suportado, preparar variável de tier e informar alternativa/custo no relatório; não ativar automaticamente uma instância mais cara. Não declarar camada gratuita nem recomendar shared-core para produção.
- Web Cloud Run 1 vCPU/512 MiB, min=0,max=2,concurrency=40.
- Consumer Cloud Run 1 vCPU/256 MiB, min=0,max=1,concurrency=10, autenticação obrigatória.
- Jobs maintenance e publish-outbox com 1 task, parallelism=1, timeout=60s, max_retries=1, máximo 5 conexões cada.
- Jobs migrate e seed com a mesma imagem, 1 task, parallelism=1, timeout=300s, max_retries=0, execução manual e sem Scheduler. migrate usa segredo da credencial de migrations; seed usa credencial da aplicação e não repõe produtos existentes.
- Pub/Sub tópico, push subscription, DLQ e subscrição de inspeção da DLQ; retenção 7 dias e ack deadline 30s; retry backoff 10–60s.
- Scheduler */1 * * * *, UTC, separado por job; OAuth para Jobs API. Consumer recebe OIDC da identidade push, com audience no URL correto.
- Secret Manager para DB password e chave Gemini quando modo real ativo. Não gravar secret payloads no Terraform/state; criar versões por script seguro a partir de variável/arquivo ignorado, sem echo de segredo.

Contas distintas: web, maintenance, publisher, consumer, scheduler-invoker e pubsub-push. Conceder cloudsql.client às que acessam banco; publisher tem pubsub.publisher apenas no tópico; push tem run.invoker apenas no consumer; scheduler tem run.invoker nos jobs; acesso de segredos somente ao serviço que necessita. Service agent Pub/Sub recebe permissões requeridas para assinatura OIDC e DLQ. CI local de validação não precisa de credencial GCP.

Para a primeira versão, uma credencial SQL de aplicação compartilhada entre componentes é aceitável; registrar essa limitação. O usuário de aplicação não é superuser; migrations usam credencial separada. Não afirmar isolamento de dados por serviço.

Imagem multi-stage: build frontend, build Go, runtime não root com certificados CA. Usar a mesma imagem com subcomandos em web, consumer e jobs para simplificar; assets são lidos somente em serve. Lockfiles e .dockerignore impedem inclusão de .env/artefatos/segredos. Migrations ficam embutidas no binário via embed para o subcomando migrate.

Procedimento em docs/deploy-gcp.md:

1. Verificar conta/projeto autorizado, faturamento e orçamento. Preparar estimativa datada no calculador oficial, incluindo custo contínuo do Cloud SQL mesmo sem tráfego.
2. Configurar APIs, Artifact Registry e service accounts/bootstrap. State local ignorado é aceitável para demo individual; explicar migração futura para bucket privado. Nunca versionar state.
3. Build/push imagem com tag do commit e registrar digest.
4. Provisionar infraestrutura inicialmente com schedulers pausados e serviços sem tráfego público até migrations concluírem.
5. Criar segredos sem imprimi-los; rodar migration job e seed job de execução manual única, sem reset.
6. Configurar APP_ORIGIN com URL efetiva, ativar revisão web, IAM push, subscrição e schedulers. Resolver dependência circular URL/IAM por duas fases documentadas, sem inventar URL.
7. Executar smoke: sessão, reserva, pagamento, notificação, expiração e tentativa de acesso cruzado. Sem teste de carga em cloud por padrão.
8. Validar modo Gemini somente se autorizado/configurado; caso contrário manter template identificado.
9. Documentar pausa, rollback de imagem, inspeção da DLQ, backup e destruição. Não executar destroy automaticamente; deletion protection exige ação deliberada.

Preparar terraform fmt -check e validate local; terraform plan precisa de ambiente e credenciais e só conta como validado se realmente executado. Scripts de deploy têm preflight e param obrigatórios PROJECT_ID/REGION; não inferem projeto de configurações globais silenciosamente.

Não executar apply, push para registro remoto ou publicação só para completar checklist sem autorização externa. Ausência disso muda o status para "pronto para deploy, não publicado", sem impedir a entrega local.

## 18. Plano de testes e evidências

### Unidade Go

- Validação de payload, hash canônico, transições, comparação exata do vencimento e templates.
- Serialização segura de erro, Origin/cookie, cálculo de can_pay e DTOs.
- Adapter Gemini com servidor HTTP controlado, incluindo latência, timeout e resposta inválida.
- Handler de eventos com payloads válidos/inválidos e versões não suportadas.

### Integração PostgreSQL real

Banco de teste isolado e migrations reais. Nunca rodar integração destrutiva contra DATABASE_URL de produção. Exigir APP_ENV=test e nome/database isolado. Testes concorrentes usam conexões separadas e barreiras de início; não executar tudo numa única transação que mascare contenção.

Casos obrigatórios:

1. Última unidade, 30 sessões comprando simultaneamente: uma criação, 29 conflitos, estoque zero.
2. Mesmo payload/chave/sessão, 20 requisições simultâneas: um pedido/evento/reserva; demais replays.
3. Chave igual com payload diferente: 409 e nenhum novo efeito.
4. Mesma chave em sessões diferentes: independentes.
5. Falha forçada depois de reduzir estoque e antes de inserir pedido: rollback restaura estoque; chave pode ser reutilizada.
6. Quatro criações simultâneas na mesma sessão: no máximo três pendentes.
7. Pagamentos concorrentes: uma transição PAID e um evento/outbox/notificação.
8. Expiração repetida: estoque devolvido uma única vez.
9. Pagamento contra expiração concorrente: exatamente um estado terminal, estoque coerente com o vencedor e apenas um tipo de evento terminal.
10. Payment após prazo e antes do job: 409 com expiração efetivamente persistida.
11. GET de vencido não muda banco e indica can_pay=false.
12. Sessão alheia não lista, lê, paga ou consulta IA de outro pedido.
13. Consumer repetido e concorrente: um processed_event e uma notification.
14. Falha antes de commit do consumer: nada persiste; retry funciona.
15. Publicação antes de crash de confirmação: redelivery não duplica notificação.
16. Lease vencido pode ser retomado; token antigo não marca publicação de claim novo.
17. Broker indisponível não desfaz pagamento; evento permanece e é publicado após recuperação.
18. Quota concorrente não excede limite; replay não consome quota global de novos pedidos.
19. Schema impede estoque negativo e estados/timestamps inconsistentes.
20. Seed repetido não altera estoque consumido nem duplica catálogo.

Testes podem usar hooks internos de falha apenas em dependências de teste; nunca expor endpoint HTTP de falha na aplicação pública.

### Frontend e navegador

- Vitest: formulário, quantidade, labels de estado e retry que mantém chave.
- Playwright com Compose real: compra/pagamento/notificação; compra/expiração; estoque esgotado; acesso entre dois browser contexts; refresh após erro de rede sem duplicar pedido; assistant template e erro controlado; sessão vencida. Fixtures E2E alteram datas/estoque diretamente no banco de teste por helper do runner, nunca por endpoint público. Não aguardar cinco minutos para testar expiração.
- Capturas em 360/768/1440px; navegação teclado; axe-core sem violações serious/critical nas telas principais. Inspeção manual visual complementar obrigatória.
- CI não chama Gemini nem GCP reais. Emulador não prova IAM; registrar isso.

### Carga

Scripts k6 em load/ e relatório docs/load-report.md. Somente local, fixtures separadas, quotas em modo load declarado. Cada cenário começa em banco limpo isolado e documenta estoque inicial.

- Smoke: 1 VU/30s para validar script.
- Leitura: 20 VUs/60s catálogo.
- Contenção: 50 sessões, uma unidade, uma rodada sincronizada; uma criação exata.
- Escrita: 20 VUs/60s, produtos com estoque suficiente, sessões com quota adequada e limites de pendência resolvidos por pagamento; medir criação e pagamento separadamente.
- Idempotência: 20 envios concorrentes com chave compartilhada e verificação final de pedido único.

Relatar máquina/CPU/RAM, versões, commit, limites Docker, aquecimento, duração, requests/s, p50/p95/p99, erros técnicos e conflitos esperados separados. Alvo de referência, não promessa: p95 <=500ms de catálogo e <=1000ms de criação no cenário local de escrita. Se não atingir, investigar e registrar gargalo; não falsificar resultado nem reduzir carga silenciosamente. Critérios rígidos: zero overselling/duplicação e zero 5xx inesperado no cenário funcional de contenção. Não declarar alta escala com base nesses números.

## 19. CI e comandos padronizados

Makefile deve oferecer:

- make doctor: checa Go/Node/Docker/Compose e portas sem expor env.
- make setup: dependências locais e orientação de configuração; não sobrescreve .env existente.
- make up / make down: subir build completo / parar sem apagar volumes.
- make dev: instruções ou processo para API + Vite + dependências locais.
- make migrate / make seed: ações idempotentes apropriadas.
- make check: formatting, go vet, lint/typecheck frontend, validação OpenAPI e tipos gerados sem diff.
- make test: unidade Go/frontend; make test-integration: banco/emulador isolados; make test-e2e: Playwright.
- make build: frontend e binário; make load-smoke / make load: carga isolada.
- make infra-check: Terraform format e validate quando provider disponível.
- make verify: sequência check + test + test-integration + build + test-e2e; retornar erro no primeiro gate obrigatório que falhar.

Workflows em PR e push: validações acima em jobs com dependências explícitas; usar npm ci, Go modules verificáveis, cache e Postgres/emulador com healthchecks. go test -race nos testes Go e integração que suportam o detector. Não rodar carga pesada em toda PR. Actions fixadas em SHA após verificar versões oficiais; dependabot para manutenção. Sem deploy automático na primeira versão.

Evitar meta de 100% de cobertura. Cobrir os cenários críticos descritos, sem testes que só espelham getters ou mocks da implementação.

## 20. Marcos de implementação e gates

O executor deve transformar os itens abaixo em checklist real em docs/execution.md. Não marcar marco como concluído só porque arquivos foram criados.

### M0 — preparar repositório e registrar decisões

- Inspecionar instruções e ferramentas; verificar se flashcart já tem conteúdo além deste plano.
- Inicializar Git isolado se não houver repositório próprio, sem tocar no pai; não fazer commit de arquivos alheios.
- Fixar versões, estrutura, .gitignore, .env.example e decisões 001-stack/002-consistency/003-demo-security.
- Definir recursos GCP concretos suportados e registrar dependências externas, sem provisionar.
- Gate: doctor documentado, árvore criada e dependências coerentes. Se runtime ausente, registrar e prosseguir com trabalho independente.

### M1 — persistência, sessões e catálogo

- Migrations completas, fixtures, seed idempotente, health/readiness, sessão, Origin e catálogo.
- OpenAPI inicial com schemas compartilhados e erros.
- Gate: migrations sobem em DB vazio; seed repetido não muda dados; isolamento de sessão testado.

### M2 — núcleo de compra e consistência

- Criação idempotente, limite de pendências, consultas, pagamento, expiração e timeline.
- Implementar testes de integração 1–12, 18–20 e revisar locks.
- Gate: todos esses cenários passam com PostgreSQL real; sem dependência de broker/IA para comprar.

### M3 — eventos e execução local completa

- Outbox/lease, emulador, publicador, consumer pull/push, notificações e maintenance.
- Compose com inicialização ordenada e volumes; scripts de espera limitados e falhas explícitas.
- Gate: testes 13–17 passam; pagamento continua correto com broker temporariamente indisponível; make up entrega fluxo assíncrono local.

### M4 — interface completa

- Design tokens, SVGs, fontes/licenças, rotas, componentes, formulários, polling e retry seguro.
- Gerar tipos a partir do OpenAPI e tratar todos os erros documentados.
- Gate: compra completa pelo navegador, screenshots nos três tamanhos, teclado e ausência de overflow/erros no console.

### M5 — assistente

- Template, adaptador Gemini, quotas, autorização, snapshot, fallback e estados de UI.
- Gate: testes de request real contra httptest, timeout/injection/contrato e isolamento; modo sem credencial claramente identificado.

### M6 — CI, endurecimento e E2E

- Finalizar check/test/build, headers, timeouts, shutdown, redaction e acessibilidade.
- CI executável sem serviços pagos, browser tests com todos os componentes locais.
- Gate: make verify aprovado ou impedimentos externos registrados por comando, sem alegação de aprovação fictícia.

### M7 — carga e revisão técnica

- Rodar cenários, checar invariantes via SQL, investigar erros e produzir relatório.
- Rever schema, transações, OpenAPI e claims do README contra código.
- Gate: resultados reproduzíveis e nenhuma violação de consistência; metas de latência tratadas conforme seção 18.

### M8 — infraestrutura e runbooks

- Terraform, Docker runtime, scripts preflight/deploy, estimativa de custos e runbooks.
- Validar format/config local, sem aplicar recursos pagos por suposição.
- Gate local: infraestrutura preparada e limites de validação explícitos. Gate externo: somente com acesso autorizado, apply e smoke real; registrar URL/digest/data.

### M9 — entrega de portfólio e auditoria final

- README final, documentação, evidências e roteiro; remover TODOs dos fluxos obrigatórios e código morto.
- Executar verificação final após últimas mudanças funcionais; documentação simples não exige rerodar toda a carga.
- Preencher matriz de conclusão e relatório final por evidência.
- Gate: checkout limpo reproduz aplicação, documentação corresponde à implementação, segredos ausentes e pendências externas listadas.

Commits sugeridos por marco, podendo dividir em mudanças revisáveis. Não fazer squash, push ou publicação remota sem necessidade/autorização. Não incluir números de teste ou desempenho inventados em mensagem de commit.

## 21. Documentação e apresentação obrigatórias

- README.md: problema, screenshots reais, stack, quickstart, comandos, arquitetura resumida, garantias e limitações, link de demo somente se existir.
- docs/architecture.md: Mermaid de componentes, sequência de compra e sequência de eventos; fronteira transacional e propriedade das tabelas.
- docs/decisions/: stack; consistência/locks; sessão anônima; outbox e entrega; Cloud Run Jobs/push; IA/fallback.
- docs/api.md: link ao OpenAPI, exemplos curl com Origin e cookie jar, replay de idempotência e erro de estoque.
- docs/testing.md, docs/load-report.md, docs/deploy-gcp.md, docs/runbook.md, docs/security.md, docs/future-work.md.
- docs/skills-and-sources.md: skills realmente lidas/aplicadas, links e data; não confundir recomendação do catálogo com skill utilizada.
- docs/evidence/: screenshots, resumos de comandos e resultados; não versionar gravações pesadas, tokens ou dumps com sessões.
- docs/demo-script.md: demonstração de 2–3 minutos (catálogo, reserva, pagamento, timeline, idempotência, arquitetura e IA com modo verdadeiro).
- docs/portfolio.md: texto curto para currículo e apresentação técnica, descrevendo projeto de portfólio e medições reais. Sem alegar experiência profissional na Shopee ou produção em escala.
- Se captura de vídeo for viável localmente, gerar uma gravação de navegador do fluxo e informar que não tem narração. Roteiro e screenshots são obrigatórios; vídeo narrado não é gate.

## 22. Matriz final e tratamento de bloqueios externos

Ao encerrar, preencher para cada item: IMPLEMENTADO, VALIDADO, PENDENTE EXTERNO ou FALHOU, com evidência e próximo comando quando aplicável.

| Item | Evidência exigida |
|---|---|
| Build e execução local | comandos, versões e URL local |
| Regras transacionais | relatório de testes PostgreSQL |
| Eventos e duplicatas | testes de falha/redelivery e notification única |
| Interface e acessibilidade | testes, screenshots e inspeção |
| Assistente template | teste completo de UI/API |
| Adaptador Gemini | httptest obrigatório; chamada real listada separadamente |
| Carga | relatório com contexto e resultados |
| CI | workflow validado localmente; execução remota somente se houver push autorizado |
| Infraestrutura | fmt/validate; plan/apply listados separadamente |
| Publicação GCP | URL e smoke real ou PENDENTE EXTERNO |
| Documentação | arquivos completos e correspondentes ao código |

Definição de conclusão local: aplicação completa implementada, gates locais disponíveis aprovados, documentação e infraestrutura preparadas, nenhum defeito conhecido nas regras obrigatórias. Teste que não pôde rodar é pendência, não aprovação.

Definição de conclusão externa: serviço GCP publicado e smoke real aprovado; Gemini real validado se for ativado. Não confundir conclusão local com externa.

Se faltar credencial/projeto/budget, concluir todos os marcos independentes, entregar o estado local e informar exatamente os dados necessários para executar o restante. Se faltar ferramenta essencial que impeça validação, relatar quais gates continuam pendentes. Não continuar indefinidamente tentando a mesma operação negada, nem terminar com somente esqueletos ou mocks quando houver trabalho independente possível.

## 23. Fontes técnicas para revisão na implementação

Consultadas no planejamento em 2026-11-11; revisitar apenas quando a implementação depender de detalhes atualizados:

- PostgreSQL locks: https://www.postgresql.org/docs/current/explicit-locking.html
- Cloud Run com Pub/Sub: https://docs.cloud.google.com/run/docs/tutorials/pubsub
- Pub/Sub push e semântica ACK: https://docs.cloud.google.com/pubsub/docs/push
- Jobs agendados: https://docs.cloud.google.com/run/docs/execute/jobs-on-schedule
- Cloud Run billing/CPU: https://docs.cloud.google.com/run/docs/configuring/billing-settings
- Gemini structured output: https://ai.google.dev/gemini-api/docs/structured-output

Não tratar JSON Schema como garantia de fatos verdadeiros, emulador como validação de IAM ou Terraform validate como prova de deploy.

## 24. Instrução de início após aprovação

"Implemente o FlashCart seguindo PLANO-DE-IMPLEMENTACAO.md, do marco M0 ao M9. Execute o trabalho autorizado até concluir os gates locais possíveis. Mantenha docs/execution.md atualizado, verifique os resultados com ferramentas e não pare para confirmar escolhas já especificadas. Não amplie o escopo, não publique nem gere custos sem autorização. Registre separadamente qualquer dependência externa e entregue evidências reais."

Este é o comando de continuidade sugerido; o usuário pode aprovar o plano em linguagem natural sem copiá-lo.
