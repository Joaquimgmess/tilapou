-include .env
export

TILAPOU_DAEMON ?= http://localhost:$(or $(API_PORT),8080)

.PHONY: build run play status test lint fmt tidy check up down golden migrate-create vuln dead

BIN := bin/tilapou

build:
	go build -o $(BIN) ./cmd/tilapou

run:
	go run ./cmd/tilapou serve

play:
	TILAPOU_DAEMON=$(TILAPOU_DAEMON) go run ./cmd/tilapou play

status:
	TILAPOU_DAEMON=$(TILAPOU_DAEMON) go run ./cmd/tilapou status

test:
	go test ./... -race
	go test ./... -cover

lint:
	golangci-lint config verify
	golangci-lint run ./...

fmt:
	golangci-lint fmt ./...

tidy:
	go mod tidy

check: fmt lint test build vuln dead

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

dead:
	@out=$$(go run golang.org/x/tools/cmd/deadcode@latest -test ./...); \
	if [ -n "$$out" ]; then echo "$$out"; echo "codigo morto encontrado"; exit 1; fi
