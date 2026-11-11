# Decisão 003 — outbox e mensagens

O evento da compra é gravado na outbox na mesma transação da mudança do pedido. Publisher reivindica lote com `FOR UPDATE SKIP LOCKED` e lease curto, publica fora da transação e confirma o lease depois do envio. Se o processo cair entre publish e confirmação, haverá redelivery; consumer grava `processed_events` e a notificação na mesma transação, fazendo deduplicação por `event_id`.

## Consequência

A entrega é pelo menos uma vez. O efeito interno da notificação é único no banco. Não prometemos entrega exatamente uma vez entre serviços.

