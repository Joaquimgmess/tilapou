package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Joaquimgmess/tilapou/internal/api"
	"github.com/Joaquimgmess/tilapou/internal/client"
)

const (
	statusTimeout = 3 * time.Second
	centsPerCoin  = 100
	gramsPerKg    = 1000
	lowOxygenUgL  = 3000
)

func runStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	addr := fs.String("daemon", daemonAddr(), "endereco do daemon")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("lendo flags de status: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), statusTimeout)
	defer cancel()

	snapshot, err := client.New(*addr, statusTimeout).Snapshot(ctx)
	if errors.Is(err, client.ErrDaemonUnreachable) {
		fmt.Fprintln(os.Stdout, "tilapou offline")

		return nil
	}
	if err != nil {
		return fmt.Errorf("lendo o estado da fazenda: %w", err)
	}

	var line strings.Builder
	fmt.Fprintf(&line, "%d,%02d TC | %d kg | %d peixes",
		snapshot.CashCents/centsPerCoin, snapshot.CashCents%centsPerCoin, snapshot.BiomassGrams/gramsPerKg, snapshot.Fish)

	for i := range snapshot.Tanks {
		t := &snapshot.Tanks[i]
		switch {
		case t.OxygenUgL < lowOxygenUgL && !t.Aerating:
			fmt.Fprintf(&line, " | ALERTA O2 no tanque %d", t.ID)
		case t.FeedKg == 0:
			fmt.Fprintf(&line, " | sem racao no tanque %d", t.ID)
		case readyBatch(t):
			fmt.Fprintf(&line, " | tanque %d pronto", t.ID)
		}
	}

	fmt.Fprintln(os.Stdout, line.String())

	return nil
}

func readyBatch(t *api.Tank) bool {
	for i := range t.Batches {
		if t.Batches[i].Ready {
			return true
		}
	}

	return false
}
