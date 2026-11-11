import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

const created = new Counter('flashcart_orders_created');
const conflicts = new Counter('flashcart_stock_conflicts');
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const ORIGIN = __ENV.ORIGIN || 'http://localhost:5173';
const PRODUCT_ID = __ENV.PRODUCT_ID;
const expectedOrderResponse = http.expectedStatuses(201, 409);

function idempotencyKey() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (char) => {
    const random = Math.floor(Math.random() * 16);
    return (char === 'x' ? random : (random & 0x3) | 0x8).toString(16);
  });
}

export const options = {
  scenarios: {
    lastUnit: { executor: 'shared-iterations', vus: 50, iterations: 50, exec: 'raceForLastUnit' },
  },
  thresholds: {
    http_req_failed: ['rate<0.02'],
    flashcart_orders_created: ['count==1'],
    flashcart_stock_conflicts: ['count==49'],
  },
};

export function raceForLastUnit() {
  if (!PRODUCT_ID) throw new Error('Set PRODUCT_ID to an isolated product with exactly one available unit.');
  const session = http.post(`${BASE_URL}/api/v1/session`, '{}', { headers: { Origin: ORIGIN, 'Content-Type': 'application/json' } });
  if (session.status !== 201 && session.status !== 200) {
    check(session, { 'session created': (r) => r.status === 201 || r.status === 200 });
    return;
  }
  const response = http.post(`${BASE_URL}/api/v1/orders`, JSON.stringify({ product_id: PRODUCT_ID, quantity: 1 }), {
    headers: { Origin: ORIGIN, 'Content-Type': 'application/json', 'Idempotency-Key': idempotencyKey() },
    responseCallback: expectedOrderResponse,
  });
  if (response.status === 201) created.add(1);
  if (response.status === 409) conflicts.add(1);
  check(response, { 'order created or expected business conflict': (r) => r.status === 201 || r.status === 409 });
}

