APP_NAME := sentry-lite-api
GO_IMAGE := golang:1.23-alpine

.PHONY: dev deps down logs test fmt api workers deploy

dev:
	docker compose up -d postgres clickhouse redis nats minio

deps:
	docker run --rm -v "$(PWD):/app" -w /app $(GO_IMAGE) go mod download

down:
	docker compose down

logs:
	docker compose logs -f

fmt:
	docker run --rm -v "$(PWD):/app" -w /app $(GO_IMAGE) gofmt -w cmd internal pkg

test:
	docker run --rm -v "$(PWD):/app" -w /app $(GO_IMAGE) go test ./...

api:
	docker compose up --build api

workers:
	docker compose up --build worker-normalize worker-grouping worker-event-writer worker-alert worker-session worker-outcome

deploy:
	powershell -ExecutionPolicy Bypass -File scripts/deploy.ps1
