package farm_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Joaquimgmess/tilapou/internal/balance"
	"github.com/Joaquimgmess/tilapou/internal/farm"
	"github.com/Joaquimgmess/tilapou/internal/migrations"
	"github.com/Joaquimgmess/tilapou/internal/sim"
)

// storeOnPostgres opens a pool on a schema of its own, migrated and dropped at the end, so
// two runs never see each other and the QA database keeps no leftovers. It skips when
// TILAPOU_TEST_DATABASE_URL is empty: the CI has no Postgres.
func storeOnPostgres(t *testing.T) *farm.DB {
	t.Helper()

	url := os.Getenv("TILAPOU_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TILAPOU_TEST_DATABASE_URL vazio: sem Postgres, pulando a integracao")
	}

	schema := fmt.Sprintf("farm_test_%d", rand.Uint32())

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatalf("lendo TILAPOU_TEST_DATABASE_URL: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema

	admin, err := pgxpool.New(t.Context(), url)
	if err != nil {
		t.Fatalf("conectando: %v", err)
	}
	defer admin.Close()

	if _, err = admin.Exec(t.Context(), "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("criando o schema %s: %v", schema, err)
	}

	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("conectando no schema %s: %v", schema, err)
	}

	t.Cleanup(func() {
		pool.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		clean, cleanErr := pgxpool.New(ctx, url)
		if cleanErr != nil {
			t.Errorf("limpando o schema %s: %v", schema, cleanErr)

			return
		}
		defer clean.Close()

		if _, cleanErr = clean.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); cleanErr != nil {
			t.Errorf("dropando o schema %s: %v", schema, cleanErr)
		}
	})

	if err = migrations.Apply(t.Context(), pool); err != nil {
		t.Fatalf("migrando o schema %s: %v", schema, err)
	}

	return farm.NewDB(pool, 5*time.Second)
}

func insertedFarm(t *testing.T, db *farm.DB) farm.Farm {
	t.Helper()

	b, err := balance.Load()
	if err != nil {
		t.Fatalf("carregando o balance: %v", err)
	}

	created := farm.New(uuid.New(), uuid.New(), "t", time.Unix(0, 0).UTC(), 0, 1, &b)
	if err = db.Insert(t.Context(), created); err != nil {
		t.Fatalf("inserindo a fazenda: %v", err)
	}

	return created
}

func TestSaveComRevisaoVelhaRecusa(t *testing.T) {
	t.Parallel()

	db := storeOnPostgres(t)
	f := insertedFarm(t, db)

	f.State.Tick++
	if err := db.Save(t.Context(), f, nil, nil); err != nil {
		t.Fatalf("primeiro save: %v", err)
	}

	f.State.Tick++
	if err := db.Save(t.Context(), f, nil, nil); !errors.Is(err, farm.ErrStaleRevision) {
		t.Errorf("salvar com a revisao velha devolveu %v, queria ErrStaleRevision", err)
	}
}

func TestSaveDaMesmaChaveNaoAvancaARevisao(t *testing.T) {
	t.Parallel()

	db := storeOnPostgres(t)
	f := insertedFarm(t, db)

	outcome := &sim.Outcome{ID: 42, Applied: true, Reason: sim.RejectNone}

	f.State.Tick++
	if err := db.Save(t.Context(), f, nil, outcome); err != nil {
		t.Fatalf("primeiro save: %v", err)
	}

	f.Revision++
	f.State.Tick++

	err := db.Save(t.Context(), f, nil, outcome)
	if !errors.Is(err, farm.ErrAlreadyApplied) {
		t.Fatalf("repetir a chave devolveu %v, queria ErrAlreadyApplied", err)
	}

	stored, err := db.ByPlayer(t.Context(), f.PlayerID)
	if err != nil {
		t.Fatalf("relendo: %v", err)
	}
	if stored.Revision != f.Revision {
		t.Errorf("a revisao foi para %d mesmo com a acao repetida recusada, queria %d",
			stored.Revision, f.Revision)
	}
}

func TestEventosVoltamDoMaisNovoParaOMaisVelho(t *testing.T) {
	t.Parallel()

	db := storeOnPostgres(t)
	f := insertedFarm(t, db)

	events := []sim.Event{
		{Seq: 1, Kind: sim.EventTankBought, From: 1, To: 1},
		{Seq: 2, Kind: sim.EventStocked, From: 2, To: 2},
		{Seq: 3, Kind: sim.EventHarvest, From: 3, To: 3},
	}

	f.State.Tick++
	if err := db.Save(t.Context(), f, events, nil); err != nil {
		t.Fatalf("salvando eventos: %v", err)
	}

	stored, err := db.Events(t.Context(), f.ID, 2)
	if err != nil {
		t.Fatalf("lendo eventos: %v", err)
	}

	if len(stored) != 2 {
		t.Fatalf("vieram %d eventos, queria 2", len(stored))
	}
	if stored[0].Seq <= stored[1].Seq {
		t.Errorf("os eventos vieram do mais velho para o mais novo: %d antes de %d", stored[0].Seq, stored[1].Seq)
	}
}

func TestFazendaDeOutroJogadorNaoEhEncontrada(t *testing.T) {
	t.Parallel()

	db := storeOnPostgres(t)
	insertedFarm(t, db)

	if _, err := db.ByPlayer(t.Context(), uuid.New()); !errors.Is(err, farm.ErrNotFound) {
		t.Errorf("jogador sem fazenda devolveu %v, queria ErrNotFound", err)
	}
}
