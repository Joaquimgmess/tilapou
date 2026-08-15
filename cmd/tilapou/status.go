package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Joaquimgmess/catalog/internal/client"
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
	if err != nil {
		fmt.Fprintln(os.Stdout, "tilapou offline")

		return nil
	}

	var line strings.Builder
	fmt.Fprintf(&line, "%d,%02d TC | %d kg | %d peixes",
		snapshot.CashCents/centsPerCoin, snapshot.CashCents%centsPerCoin, snapshot.BiomassG/gramsPerKg, snapshot.Fish)

	for _, t := range snapshot.Tanks {
		switch {
		case t.OxygenUgL < lowOxygenUgL && !t.Aerating:
			fmt.Fprintf(&line, " | ALERTA O2 no tanque %d", t.ID)
		case t.FeedKg == 0:
			fmt.Fprintf(&line, " | sem racao no tanque %d", t.ID)
		case t.Ready:
			fmt.Fprintf(&line, " | tanque %d pronto", t.ID)
		}
	}

	fmt.Fprintln(os.Stdout, line.String())

	return nil
}
