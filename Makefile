.PHONY: build run test lint fmt tidy check up down migrate migrate-status migrate-create vuln

BIN := bin/api

build:
	go build -o $(BIN) ./cmd/api
	go build -o bin/migrate ./cmd/migrate

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

check: fmt lint test build vuln

up:
	docker compose up -d --build

down:
	docker compose down -v

migrate:
	go run ./cmd/migrate

migrate-status:
	go run ./cmd/migrate -status

migrate-create:
	@test -n "$(name)" || (echo "usage: make migrate-create name=add_something" && exit 1)
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir internal/migrations/sql create $(name) sql

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
