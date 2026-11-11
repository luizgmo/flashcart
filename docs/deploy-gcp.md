# Deploy na GCP

Esta configuração prepara Cloud Run, Cloud SQL, Artifact Registry, Pub/Sub, Secret Manager e Cloud Scheduler na região `southamerica-east1`. O apply cria recursos faturáveis. Nenhum recurso foi aplicado nesta execução.

## Preparar antes do primeiro apply

1. Instale Terraform 1.11+ e `gcloud`; confirme que tem acesso ao projeto escolhido e billing habilitado.
2. Estime o gasto no [calculador oficial do Google Cloud](https://cloud.google.com/products/calculator). Cloud SQL mantém custo mesmo quando Cloud Run escala a zero; `db-f1-micro` é configuração de demonstração, sem SLA de produção.
3. Faça login, configure projeto e region. Não salve credenciais localmente no repositório.
4. O container inicial precisa existir antes do primeiro Cloud Run service. Crie o repositório Artifact Registry pelo procedimento da organização, faça build/push da imagem e registre o digest. Essa etapa pode usar Cloud Build, que também pode ter cobrança.
5. Copie `infra/terraform/terraform.tfvars.example` para `terraform.tfvars` e substitua projeto e digest imutável da imagem. `app_origin` é provisório no primeiro apply. Como o repositório foi criado no bootstrap, importe-o no state do Terraform antes do primeiro apply:

   ```sh
   terraform import google_artifact_registry_repository.containers \
     "projects/PROJECT_ID/locations/REGION/repositories/flashcart"
   ```

6. O Terraform cria a instância e o banco, mas não salva senhas no state. Crie um usuário PostgreSQL com uma senha forte por procedimento administrativo autorizado. Grave em Secret Manager, fora do Terraform, um `DATABASE_URL` no formato Cloud SQL socket Unix:

   ```text
   postgres://DB_USER:URL_ENCODED_PASSWORD@/flashcart?host=/cloudsql/PROJECT:REGION:INSTANCE&sslmode=disable
   ```

   `sslmode=disable` vale somente para a conexão por socket dentro do contêiner; o conector gerenciado protege a conexão com Cloud SQL. Não use TCP público nem crie authorized network aberta. Migrations usam a mesma conta por simplicidade nesta demo; reduzir privilégios com uma conta de migração é uma melhoria de operação.

7. Rode `terraform init`, `terraform fmt -check` e `terraform validate`. Crie/revise o plano antes de apply. State deve permanecer em armazenamento privado; o state local ignorado pelo Git serve apenas para uma demo individual.
8. Faça o primeiro apply com imagem já publicada no Artifact Registry e origem provisória. Esse apply cria Cloud SQL, que tem custo contínuo.
9. Crie o usuário e o secret version de `DATABASE_URL`; aplique novamente para ligar os serviços que leem o secret. Rode migration/seed com o binário versionado do mesmo digest (`migrate`, depois `seed`). O Terraform não faz reset nem roda esses comandos destrutivos.
10. Atualize `app_origin` com a URL retornada por `terraform output -raw web_url`, aplique novamente e verifique sessão, pedido, pagamento, outbox, notificação e expiração. Pub/Sub chama o consumer pela URL autenticada; o serviço consumer não permite chamadas anônimas.
11. Para Gemini, primeiro crie uma versão do secret indicada por `gemini_secret`, depois mude `enable_gemini=true`. A chamada usa o modelo Flash estável configurado, pode gerar cobrança e tem limite global de 100 chamadas/dia.

## Atualizações e encerramento

- Publique uma imagem nova com tag de commit e use o digest `sha256` em `image`; a variável rejeita tags mutáveis.
- Antes de atualizar Go, Node, PostgreSQL ou provider Terraform, revise as fontes oficiais e gere novamente o lockfile do provider.
- Pause Scheduler jobs antes de investigar comportamento de jobs. Reversão de aplicação é feita apontando `image` para digest anterior.
- Inspecione a DLQ `flashcart-events-dead-letter`; depois de corrigir a causa, republique somente após verificar `event_id` e a proteção de deduplicação.
- Faça backup Cloud SQL antes de migrations incompatíveis. A rotina atual usa alterações aditivas; rollback da migration inicial remove todas as tabelas e não é um rollback operacional de produção.
- Para excluir demonstração: pause schedulers, preserve/importe o state, confirme backups e custo, defina `deletion_protection=false`, revise `terraform plan`, aplique essa alteração e só então avalie destroy. O destroy não é automático.

## IAM e limites atuais

As service accounts da aplicação separam web, publisher, consumer, maintenance, scheduler e identidade push. O publisher só publica no tópico; Pub/Sub invoca o consumer por OIDC; os workloads de aplicação acessam Cloud SQL pelo connector e buscam a URL via Secret Manager.

O secret `DATABASE_URL` e a criação do usuário são bootstrap manual, pois senha em Terraform entraria no state. Um único banco atende a API e jobs; esta demo não implementa isolamento físico por microsserviço. Os jobs de maintenance e publisher executam uma vez por minuto, então expiração e notificações podem demorar até mais de um minuto.

