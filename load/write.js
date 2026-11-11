import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const ORIGIN = __ENV.ORIGIN || 'http://localhost:5173';
const PRODUCT_ID = __ENV.PRODUCT_ID;
const ordersCreated = new Counter('flashcart_write_orders_created');
const paymentsPaid = new Counter('flashcart_write_payments_paid');

export const options = {
  scenarios: {
    writes: { executor: 'constant-vus', vus: 20, duration: '60s' },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    'http_req_duration{name:create_order}': ['p(95)<1000'],
    'http_req_duration{name:simulate_payment}': ['p(95)<1000'],
  },
};

function idempotencyKey() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (char) => {
    const random = Math.floor(Math.random() * 16);
    return (char === 'x' ? random : (random & 0x3) | 0x8).toString(16);
  });
}

export default function () {
  if (!PRODUCT_ID) throw new Error('Set PRODUCT_ID to a load-test product with sufficient stock.');

  const session = http.post(`${BASE_URL}/api/v1/session`, '{}', {
    headers: { Origin: ORIGIN, 'Content-Type': 'application/json' },
  });
  if (!check(session, { 'session created': (r) => r.status === 201 })) return;

  const order = http.post(`${BASE_URL}/api/v1/orders`, JSON.stringify({ product_id: PRODUCT_ID, quantity: 1 }), {
    headers: { Origin: ORIGIN, 'Content-Type': 'application/json', 'Idempotency-Key': idempotencyKey() },
    tags: { name: 'create_order' },
  });
  if (!check(order, { 'order created': (r) => r.status === 201 })) return;
  ordersCreated.add(1);

  const orderID = order.json('order.id');
  const payment = http.post(`${BASE_URL}/api/v1/orders/${orderID}/simulate-payment`, '{}', {
    headers: { Origin: ORIGIN, 'Content-Type': 'application/json' },
    tags: { name: 'simulate_payment' },
  });
  if (check(payment, { 'payment completed': (r) => r.status === 200 && r.json('order.status') === 'PAID' })) {
    paymentsPaid.add(1);
  }
  sleep(1);
}
