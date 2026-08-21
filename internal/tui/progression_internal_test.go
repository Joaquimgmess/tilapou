package tui

import (
	"fmt"
	"strings"
)

func (d *driver) line(label string) string {
	d.t.Helper()

	d.press("r")

	s := d.model.(Model).snapshot
	if len(s.Tanks) == 0 {
		return fmt.Sprintf("%-26s caixa %10s  sem tanque\n", label, coins(s.CashCents))
	}

	t := s.Tanks[0]
	auto := make([]string, 0, len(t.Upgrades))
	for _, u := range t.Upgrades {
		if u.Owned {
			auto = append(auto, u.Kind)
		}
	}

	goal, _ := objective(s, d.model.(Model).tankID())
	note := d.model.(Model).message
	if note != "" {
		note = "  << " + note
	}

	front, _ := frontBatch(t)

	return fmt.Sprintf("%-26s caixa %10s  %5d peixes de %3d g  racao %4d kg  auto[%s]\n"+
		"%-26s margem %10s  custo %s/kg  mercado %s/kg%s\n%-26s %s\n",
		label, coins(s.CashCents), t.Fish, front.MeanGrams, t.FeedKg, strings.Join(auto, ","),
		"", signedPlain(front.MarginCents), coins(front.CostPerKg), coins(front.PriceKgCents), note,
		"", goal)
}
