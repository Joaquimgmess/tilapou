.PHONY: build run test lint fmt tidy check up down golden migrate-create vuln

BIN := bin/tilapou

build:
	go build -o $(BIN) ./cmd/tilapou

run:
	go run ./cmd/tilapou serve

test:
	go test ./... -race -cover

lint:
	golangci-lint config verify
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

golden:
	go test ./internal/sim/scenario/ -run TestScenarios -update

migrate-create:
	@test -n "$(name)" || (echo "usage: make migrate-create name=add_something" && exit 1)
	go run github.com/pressly/goose/v3/cmd/goose@latest -dir internal/migrations/sql create $(name) sql

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
