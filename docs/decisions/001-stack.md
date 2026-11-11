# Decisão 001 — stack e forma de deploy

Escolhemos Go + PostgreSQL para regras de pedido e consistência, React/TypeScript para a demo, Docker Compose para execução local e Cloud Run para preparar uma implantação enxuta. SQL fica explícito; não usamos ORM. A aplicação é um monólito modular; processo separado existe apenas para manutenção/publicação/consumo assíncronos.

## Consequências

Menos serviços simplificam desenvolvimento e deixam pedido + estoque numa transação local. Em troca, catálogo e pedido compartilham implantação e banco. Uma divisão futura precisa de evidência operacional, propriedade de dados e contrato de eventos.

