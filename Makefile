.PHONY: doctor setup up down migrate seed serve-api serve-web serve-consumer maintenance maintenance-loop publish-outbox publish-outbox-loop build check verify e2e-api load-catalog load-contention load-write load-idempotency

doctor:
	@command -v go >/dev/null || { echo "Go não encontrado"; exit 1; }
	@command -v node >/dev/null || { echo "Node.js não encontrado"; exit 1; }
	@command -v docker >/dev/null || { echo "Docker não encontrado"; exit 1; }
	@docker compose version >/dev/null || { echo "Docker Compose plugin não encontrado"; exit 1; }

setup:
	@test -f .env || cp .env.example .env
	@cd frontend && npm ci

up:
	docker compose -f compose.yaml up -d db pubsub pubsub-init

down:
	docker compose -f compose.yaml down

migrate:
	cd backend && set -a && . ../.env && set +a && go run ./cmd/flashcart migrate

seed:
	cd backend && set -a && . ../.env && set +a && go run ./cmd/flashcart seed

serve-api:
	cd backend && set -a && . ../.env && set +a && go run ./cmd/flashcart serve

serve-web:
	cd frontend && npm run dev

serve-consumer:
	cd backend && set -a && . ../.env && set +a && go run ./cmd/flashcart consume-push

maintenance:
	cd backend && set -a && . ../.env && set +a && go run ./cmd/flashcart maintenance

maintenance-loop:
	cd backend && set -a && . ../.env && set +a && go run ./cmd/flashcart maintenance-loop

publish-outbox:
	cd backend && set -a && . ../.env && set +a && go run ./cmd/flashcart publish-outbox

publish-outbox-loop:
	cd backend && set -a && . ../.env && set +a && go run ./cmd/flashcart publish-outbox-loop

build:
	cd frontend && npm run build
	cd backend && go build ./...

check:
	cd backend && gofmt -w $$(find . -name '*.go') && go vet ./...
	cd frontend && npm run build

verify: check
	cd backend && go test -race ./...

e2e-api:
	python3 scripts/e2e-api.py

load-catalog:
	k6 run -e BASE_URL=$${BASE_URL:-http://localhost:8080} load/catalog.js

load-contention:
	k6 run -e BASE_URL=$${BASE_URL:-http://localhost:8080} -e ORIGIN=$${ORIGIN:-http://localhost:5173} -e PRODUCT_ID=$${PRODUCT_ID:?Set PRODUCT_ID to an isolated product with one unit} load/flash-sale.js

load-write:
	k6 run -e BASE_URL=$${BASE_URL:-http://localhost:8080} -e ORIGIN=$${ORIGIN:-http://localhost:5173} -e PRODUCT_ID=$${PRODUCT_ID:?Set PRODUCT_ID to an isolated product with sufficient stock} load/write.js

load-idempotency:
	k6 run -e BASE_URL=$${BASE_URL:-http://localhost:8080} -e ORIGIN=$${ORIGIN:-http://localhost:5173} -e PRODUCT_ID=$${PRODUCT_ID:?Set PRODUCT_ID to an isolated product with one unit} load/idempotency.js

