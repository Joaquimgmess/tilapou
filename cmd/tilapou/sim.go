package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/Joaquimgmess/catalog/internal/balance"
	"github.com/Joaquimgmess/catalog/internal/sim/scenario"
)

var errNoScenario = errors.New("cenario nao encontrado")

func runSim(args []string) error {
	fs := flag.NewFlagSet("sim", flag.ContinueOnError)
	list := fs.Bool("list", false, "lista os cenarios disponiveis")
	name := fs.String("run", "", "roda um cenario pelo nome")
	days := fs.Int64("days", 0, "sobrescreve a duracao do cenario em dias")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("lendo flags de sim: %w", err)
	}

	if *list {
		for _, s := range scenario.All() {
			fmt.Fprintf(os.Stdout, "%-28s %4d dias  semente %d\n", s.Name, s.Days, s.Seed)
		}

		return nil
	}

	if *name == "" {
		fs.Usage()

		return nil
	}

	target, ok := scenario.ByName(*name)
	if !ok {
		return fmt.Errorf("%w: %q", errNoScenario, *name)
	}
	if *days > 0 {
		target.Days = *days
	}

	b, err := balance.Load()
	if err != nil {
		return err
	}

	result, err := scenario.Run(target, &b)
	if err != nil {
		return err
	}

	fmt.Fprint(os.Stdout, result.Render())

	return nil
}
