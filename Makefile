.PHONY: build run test lint fmt tidy check up down migrate vuln

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

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
