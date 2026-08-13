GOOSE_VERSION ?= v3.26.0
GOOSE = go run github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION) -dir ./migrations postgres "$(DATABASE_URL)"

.PHONY: dev-api dev-worker seed build test vet fmt lint migrate-up migrate-down-one migrate-status migrate-new docker-config

dev-api:
	go run ./cmd/api

dev-worker:
	go run ./cmd/worker

seed:
	go run ./cmd/seed

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

lint: vet

migrate-up:
	$(GOOSE) up

migrate-down-one:
	$(GOOSE) down

migrate-status:
	$(GOOSE) status

migrate-new:
	@test -n "$(name)" || (echo "usage: make migrate-new name=create_table" && exit 1)
	go run github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION) -dir ./migrations create "$(name)" sql

docker-config:
	docker compose -f ../docker-compose.yml config
