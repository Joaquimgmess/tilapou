package main

import (
	"errors"
	"fmt"
	"os"
)

var errUsage = errors.New("uso: tilapou <sim|serve|play|status> [flags]")

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
	default:
		return fmt.Errorf("%w: comando desconhecido %q", errUsage, args[0])
	}
}
