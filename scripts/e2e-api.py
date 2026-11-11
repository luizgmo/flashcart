#!/usr/bin/env python3
"""Smoke-test the local FlashCart API using only Python's standard library."""

import http.cookiejar
import json
import os
import urllib.error
import urllib.request
import uuid


base_url = os.getenv("BASE_URL", "http://localhost:8080").rstrip("/")
origin = os.getenv("ORIGIN", "http://localhost:5173")
product_id = os.getenv(
    "PRODUCT_ID", "00000000-0000-4000-8000-000000000001"
)
cookies = http.cookiejar.CookieJar()
client = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cookies))


def request(method, path, payload=None, headers=None):
    request_headers = {"Origin": origin}
    if payload is not None:
        request_headers["Content-Type"] = "application/json"
    request_headers.update(headers or {})
    body = None if payload is None else json.dumps(payload).encode()
    req = urllib.request.Request(
        base_url + path, data=body, headers=request_headers, method=method
    )
    try:
        response = client.open(req, timeout=5)
    except urllib.error.HTTPError as error:
        response = error
    raw = response.read()
    decoded = json.loads(raw) if raw else None
    return response.status, response.headers, decoded


def expect(actual, expected, step):
    if actual != expected:
        raise SystemExit(f"{step}: esperado HTTP {expected}, recebido {actual}")


status, _, _ = request("POST", "/api/v1/session", {})
expect(status, 201, "criar sessão")

key = str(uuid.uuid4())
payload = {"product_id": product_id, "quantity": 1}
status, _, created = request(
    "POST", "/api/v1/orders", payload, {"Idempotency-Key": key}
)
expect(status, 201, "criar pedido")
order_id = created["order"]["id"]

status, headers, replay = request(
    "POST", "/api/v1/orders", payload, {"Idempotency-Key": key}
)
expect(status, 200, "repetir pedido idempotente")
if headers.get("Idempotency-Replayed") != "true":
    raise SystemExit("replay: faltou o header Idempotency-Replayed: true")
if replay["order"]["id"] != order_id:
    raise SystemExit("replay: retornou outro pedido")

status, _, paid = request(
    "POST", f"/api/v1/orders/{order_id}/simulate-payment", {}
)
expect(status, 200, "simular pagamento")
if paid["order"]["status"] != "PAID":
    raise SystemExit("pagamento: pedido não ficou PAID")

status, _, detail = request("GET", f"/api/v1/orders/{order_id}")
expect(status, 200, "consultar pedido")
if detail["order"]["status"] != "PAID":
    raise SystemExit("consulta: estado do pedido não é PAID")
if [event["type"] for event in detail["order"]["timeline"]] != [
    "ORDER_CREATED",
    "ORDER_PAID",
]:
    raise SystemExit("consulta: timeline do pedido não corresponde ao fluxo")

status, _, explanation = request(
    "POST",
    f"/api/v1/orders/{order_id}/assistant",
    {"question": "Meu pagamento foi confirmado?"},
)
expect(status, 200, "consultar assistente do pedido")
if explanation["snapshot"]["status"] != "PAID":
    raise SystemExit("assistente: não recebeu o estado atualizado do pedido")

print(f"OK: sessão, pedido idempotente, pagamento, timeline e assistente; order_id={order_id}")
