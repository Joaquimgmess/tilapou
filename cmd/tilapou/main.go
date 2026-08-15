package main

import (
	"errors"
	"fmt"
	"os"
)

var errUsage = errors.New("uso: tilapou <serve|play|sim|status|health> [flags]")

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "tilapou:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errUsage
	}

	switch args[0] {
	case "sim":
		return runSim(args[1:])
	case "play":
		return runPlay(args[1:])
	case "serve":
		return runServe(args[1:])
	case "status":
		return runStatus(args[1:])
	case "health":
		return runHealth(args[1:])
	default:
		return fmt.Errorf("%w: comando desconhecido %q", errUsage, args[0])
	}
}
