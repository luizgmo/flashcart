import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const ORIGIN = __ENV.ORIGIN || 'http://localhost:5173';
const PRODUCT_ID = __ENV.PRODUCT_ID;
const created = new Counter('flashcart_idempotency_created');
const replays = new Counter('flashcart_idempotency_replays');
const expectedOrderResponse = http.expectedStatuses(200, 201);

export const options = {
  scenarios: {
    sharedKey: { executor: 'shared-iterations', vus: 20, iterations: 20 },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    flashcart_idempotency_created: ['count==1'],
    flashcart_idempotency_replays: ['count==19'],
  },
};

function idempotencyKey() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (char) => {
    const random = Math.floor(Math.random() * 16);
    return (char === 'x' ? random : (random & 0x3) | 0x8).toString(16);
  });
}

export function setup() {
  if (!PRODUCT_ID) throw new Error('Set PRODUCT_ID to an isolated product with one available unit.');
  const response = http.post(`${BASE_URL}/api/v1/session`, '{}', {
    headers: { Origin: ORIGIN, 'Content-Type': 'application/json' },
  });
  if (response.status !== 201 || !response.cookies.flashcart_session?.[0]?.value) {
    throw new Error(`Could not create the shared test session: HTTP ${response.status}`);
  }
  return {
    cookie: response.cookies.flashcart_session[0].value,
    key: idempotencyKey(),
  };
}

export default function (data) {
  const response = http.post(`${BASE_URL}/api/v1/orders`, JSON.stringify({ product_id: PRODUCT_ID, quantity: 1 }), {
    headers: {
      Origin: ORIGIN,
      'Content-Type': 'application/json',
      Cookie: `flashcart_session=${data.cookie}`,
      'Idempotency-Key': data.key,
    },
    responseCallback: expectedOrderResponse,
  });
  const accepted = check(response, { 'new request or idempotent replay': (r) => r.status === 201 || r.status === 200 });
  if (!accepted) return;
  if (response.status === 201) created.add(1);
  else {
    const replayHeader = Object.entries(response.headers).some(([key, value]) =>
      key.toLowerCase() === 'idempotency-replayed' && value === 'true'
    );
    check(response, { 'replay header present': () => replayHeader });
    if (replayHeader) replays.add(1);
  }
}
