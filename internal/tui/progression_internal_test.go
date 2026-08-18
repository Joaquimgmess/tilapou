package tui

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joaquimgmess/tilapou/internal/client"
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

	return fmt.Sprintf("%-26s caixa %10s  %5d peixes de %3d g  racao %4d kg  auto[%s]\n"+
		"%-26s margem %10s  custo %s/kg  mercado %s/kg%s\n%-26s %s\n",
		label, coins(s.CashCents), t.Fish, t.MeanGrams, t.FeedKg, strings.Join(auto, ","),
		"", signedPlain(t.MarginCents), coins(t.CostPerKg), coins(t.PriceKgCents), note,
		"", goal)
}

func TestProgression(t *testing.T) {
	addr := os.Getenv("TILAPOU_DAEMON")
	if addr == "" {
		t.Skip("defina TILAPOU_DAEMON")
	}

	var model tea.Model = New(client.New(addr, 10*time.Second))
	d := &driver{t: t, model: model}
	d.run(model.Init())

	var out strings.Builder
	out.WriteString(d.line("inicio"))

	d.press("f")
	d.press("h")
	out.WriteString(d.line("vendi o lote herdado"))

	d.press("1")
	d.press("2")
	out.WriteString(d.line("comprei comedouro e aerador"))

	d.press("g")
	for range 2 {
		d.press("down")
	}
	d.press("z")
	d.press("x")
	out.WriteString(d.line("peguei o credito que a tela sugeriu"))

	d.press("s")
	out.WriteString(d.line("povoei ate o equilibrio"))

	for _, days := range []int{60, 60, 60} {
		qaJumpDays(t, days)
		out.WriteString(d.line(fmt.Sprintf("+%d dias", days)))
	}

	d.press("h")
	out.WriteString(d.line("despesquei"))

	d.press("g")
	for range 3 {
		d.press("down")
	}
	d.press("z")
	d.press("x")
	out.WriteString(d.line("quitei a divida"))

	d.press("s")
	out.WriteString(d.line("povoei o ciclo seguinte"))

	fmt.Fprint(os.Stdout, "\n"+out.String())
}
