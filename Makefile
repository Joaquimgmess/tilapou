.PHONY: build run test lint fmt tidy check up down migrate

BIN := bin/api

build:
	go build -o $(BIN) ./cmd/api

run:
	go run ./cmd/api

test:
	go test ./... -race -cover

lint:
	golangci-lint run ./...

fmt:
	golangci-lint fmt ./...

tidy:
	go mod tidy

check: fmt lint test build

up:
	docker compose up -d --build

down:
	docker compose down -v

migrate:
	docker compose exec -T postgres psql -U catalog -d catalog < migrations/0001_products.sql
