-include .env
export

TILAPOU_DAEMON ?= http://localhost:$(or $(API_PORT),8080)

.PHONY: build run play status test test-db test-live lint fmt tidy check up down golden migrate-create vuln dead

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

# test-db roda tambem a integracao do farm/database.go, que pula sem esta variavel.
# Aponta para o Postgres do compose (5433) e nunca para o banco do jogador.
test-db: export TILAPOU_TEST_DATABASE_URL = postgres://$(or $(POSTGRES_USER),tilapou):$(or $(POSTGRES_PASSWORD),tilapou)@localhost:5433/tilapou_qa?sslmode=disable
test-db:
	go test ./internal/farm/ -race -count=1

# test-live roda os tres testes que falam com um daemon de verdade. Eles moram atras da tag
# live: sem ela um go test ./... limpo os pulava com SKIP verde. Fora do check de proposito —
# regra que depende de daemon de pe nao converge.
test-live: export TILAPOU_DAEMON ?= http://localhost:8098
test-live: export QA_DATABASE ?= tilapou_qa
test-live:
	go test -tags live ./internal/tui/ -count=1 -run "TestLiveSession|TestProgression|TestQASession"

lint:
	golangci-lint config verify
	golangci-lint run ./...

fmt:
	golangci-lint fmt ./...

tidy:
	go mod tidy

check: fmt lint test test-db build vuln dead

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
	@out=$$(go run golang.org/x/tools/cmd/deadcode@latest -tags live -test ./...); \
	if [ -n "$$out" ]; then echo "$$out"; echo "codigo morto encontrado"; exit 1; fi
