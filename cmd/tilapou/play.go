package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joaquimgmess/tilapou/internal/client"
	"github.com/Joaquimgmess/tilapou/internal/tui"
)

const playTimeout = 5 * time.Second

func runPlay(args []string) error {
	fs := flag.NewFlagSet("play", flag.ContinueOnError)
	addr := fs.String("daemon", daemonAddr(), "endereco do daemon")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("lendo flags de play: %w", err)
	}

	program := tea.NewProgram(tui.New(client.New(*addr, playTimeout)))
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("rodando a interface: %w", err)
	}

	return nil
}

func daemonAddr() string {
	if v := os.Getenv("TILAPOU_DAEMON"); v != "" {
		return v
	}

	return "http://localhost:8080"
}
