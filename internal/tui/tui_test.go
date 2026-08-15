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

	"github.com/Joaquimgmess/catalog/internal/client"
	"github.com/Joaquimgmess/catalog/internal/tui"
)

func fakeDaemon(t *testing.T, snapshot client.Snapshot) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var action client.Action
			_ = json.NewDecoder(r.Body).Decode(&action)
			snapshot.LastApplied = true
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
			FeedKg: 180, OxygenUgL: 1_900, DensityKg: 1, BatchID: 1,
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

	tm.Send(tea.KeyPressMsg{Code: 'q', Text: "q"})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestGameBoyScreenShowsTheFarm(t *testing.T) {
	t.Parallel()

	server := fakeDaemon(t, sampleSnapshot())
	tm := teatest.NewTestModel(t, tui.New(client.New(server.URL, time.Second)),
		teatest.WithInitialTermSize(120, 60))

	waitForAll(t, tm, "TILAPOU", "TANQUE 1", "306 g", "setas mover")
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

	waitForAll(t, tm, "biomassa", "612 kg", "306 g", "eventos")
}

func TestGameBoyWarnsAboutSuffocation(t *testing.T) {
	t.Parallel()

	server := fakeDaemon(t, sampleSnapshot())
	tm := teatest.NewTestModel(t, tui.New(client.New(server.URL, time.Second)),
		teatest.WithInitialTermSize(120, 60))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("TILAPOU"))
	}, teatest.WithDuration(5*time.Second), teatest.WithCheckInterval(20*time.Millisecond))

	tm.Send(tea.KeyPressMsg{Code: 'w', Text: "w"})
	tm.Send(tea.KeyPressMsg{Code: 'w', Text: "w"})

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
