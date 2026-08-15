package tui_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest/v2"

	"github.com/Joaquimgmess/tilapou/internal/client"
	"github.com/Joaquimgmess/tilapou/internal/tui"
)

func fakeDaemon(t *testing.T, snapshot client.Snapshot) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var action client.Action
			_ = json.NewDecoder(r.Body).Decode(&action)
			snapshot.LastOutcome = &client.Outcome{Applied: true}
			snapshot.Tanks[0].Aerating = action.Kind == "aerate" && action.Amount != 0
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(snapshot)
	}))
	t.Cleanup(server.Close)

	return server
}

func sampleSnapshot() client.Snapshot {
	return client.Snapshot{
		FarmID:     "f0000000-0000-0000-0000-000000000001",
		Name:       "Tilapou",
		Tick:       4_321,
		Hour:       4,
		TempMilliC: 26_500,
		CashCents:  123_456,
		BiomassG:   612_000,
		Fish:       2_000,
		Tanks: []client.Tank{{
			ID: 1, Kind: "viveiro_escavado", Fish: 2_000, MeanGrams: 306,
			FeedKg: 180, OxygenUgL: 1_900, DensityMilli: 600, BatchID: 1,
			Capacity: 5_000, ServedFor: 120,
			Upgrades: []client.Upgrade{
				{Kind: "comedouro", CostCents: 250_000},
				{Kind: "aerador", CostCents: 600_000},
				{Kind: "peao", CostCents: 2_000_000},
				{Kind: "tecnico", CostCents: 7_500_000},
				{Kind: "contrato", CostCents: 25_000_000},
			},
		}},
		Events: []client.Event{
			{Seq: 2, Kind: "hypoxia_deaths", Tank: 1, Fish: 47},
			{Seq: 1, Kind: "feed_bought", Tank: 1, MassG: 100_000, CashTC: 32_000},
		},
	}
}

func waitForAll(t *testing.T, tm *teatest.TestModel, wants ...string) {
	t.Helper()

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		for _, want := range wants {
			if !bytes.Contains(b, []byte(want)) {
				return false
			}
		}

		return true
	}, teatest.WithDuration(5*time.Second), teatest.WithCheckInterval(20*time.Millisecond))

	tm.Send(tea.KeyPressMsg{Code: 'x', Text: "x"})
	tm.Send(tea.KeyPressMsg{Code: 'q', Text: "q"})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestGameBoyScreenShowsTheFarm(t *testing.T) {
	t.Parallel()

	server := fakeDaemon(t, sampleSnapshot())
	tm := teatest.NewTestModel(t, tui.New(client.New(server.URL, time.Second)),
		teatest.WithInitialTermSize(120, 60))

	waitForAll(t, tm, "TILAPOU", "URGENTE", "TANQUE 1", "306 g")
}

func TestInteractOpensTheTankMenu(t *testing.T) {
	t.Parallel()

	server := fakeDaemon(t, sampleSnapshot())
	tm := teatest.NewTestModel(t, tui.New(client.New(server.URL, time.Second)),
		teatest.WithInitialTermSize(120, 60))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("TILAPOU"))
	}, teatest.WithDuration(5*time.Second), teatest.WithCheckInterval(20*time.Millisecond))

	for range 3 {
		tm.Send(tea.KeyPressMsg{Code: tea.KeyUp})
	}
	tm.Send(tea.KeyPressMsg{Code: 'z', Text: "z"})

	waitForAll(t, tm, "Servir o trato", "Instalar comedouro", "faltam")
}

func TestDashboardShowsTheNumbers(t *testing.T) {
	t.Parallel()

	server := fakeDaemon(t, sampleSnapshot())
	tm := teatest.NewTestModel(t, tui.New(client.New(server.URL, time.Second)),
		teatest.WithInitialTermSize(120, 60))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("TILAPOU"))
	}, teatest.WithDuration(5*time.Second), teatest.WithCheckInterval(20*time.Millisecond))

	tm.Send(tea.KeyPressMsg{Code: 'm', Text: "m"})

	waitForAll(t, tm, "biomassa", "612 kg", "306 g", "automacao do tanque")
}

func TestGameBoyWarnsAboutSuffocation(t *testing.T) {
	t.Parallel()

	server := fakeDaemon(t, sampleSnapshot())
	tm := teatest.NewTestModel(t, tui.New(client.New(server.URL, time.Second)),
		teatest.WithInitialTermSize(120, 60))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("TILAPOU"))
	}, teatest.WithDuration(5*time.Second), teatest.WithCheckInterval(20*time.Millisecond))

	tm.Send(tea.KeyPressMsg{Code: tea.KeyUp})
	tm.Send(tea.KeyPressMsg{Code: tea.KeyUp})
	tm.Send(tea.KeyPressMsg{Code: tea.KeyUp})

	waitForAll(t, tm, "sufocando")
}

func TestTuiShowsDaemonFailure(t *testing.T) {
	t.Parallel()

	model := tui.New(client.New("http://127.0.0.1:1", 200*time.Millisecond))
	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(120, 60))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("daemon fora do ar"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyPressMsg{Code: 'q', Text: "q"})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
