# Segurança da demonstração

- Sessão aleatória de 256 bits; somente SHA-256 do token fica no banco. Cookie `HttpOnly`, `SameSite=Lax`, 24 horas e `Secure` em HTTPS.
- Pedidos sempre consultam pelo `session_id`; ID não substitui autorização. Sem identidade de conta, perder o cookie significa perder acesso ao histórico.
- Mutação exige `Origin` exatamente igual ao `APP_ORIGIN` e JSON. Cookie não vai para `localStorage`; chave idempotente não inclui informação secreta.
- Idempotência por sessão/chave e hash do payload canônico. Pedido, redução de estoque, idempotência e timeline confirmam numa transação.
- SQL parametrizado, valor monetário em centavos, limite de body, quotas no banco, timeout HTTP/SQL e headers de segurança.
- Logs usam rota normalizada, request ID, método/status/duração. Não registram token, corpo, pergunta, chave Gemini nem DSN.
- Gemini só recebe snapshot do pedido depois da autorização por sessão; a pergunta é conteúdo não confiável. O resultado precisa coincidir com `order_id` e estado do snapshot, é escapado como texto e tem fallback. Structured output não garante veracidade.
- Endpoint consumidor é separado do serviço público; em cloud, Cloud Run IAM verifica OIDC do Pub/Sub. No Compose, o listener local é para desenvolvimento.
- Demo não aceita cartão, Pix, endereço, nome ou e-mail. Não usar dados reais.

