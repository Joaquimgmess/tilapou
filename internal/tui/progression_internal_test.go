package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joaquimgmess/tilapou/internal/client"
)

func jumpDays(t *testing.T, days int) {
	t.Helper()

	seconds := days * 24 * 60
	query := "UPDATE farms SET epoch = epoch - interval '" + strconv.Itoa(seconds) + " seconds'"

	cmd := exec.CommandContext(t.Context(), "docker", "compose", "exec", "-T", "postgres",
		"psql", "-U", "tilapou", "-d", "tilapou", "-c", query)
	cmd.Dir = "../.."

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("pulando %d dias: %v: %s", days, err, out)
	}
}

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

	goal, _ := objective(s)

	return fmt.Sprintf("%-26s caixa %10s  %5d peixes de %3d g  racao %4d kg  auto[%s]\n%-26s %s\n",
		label, coins(s.CashCents), t.Fish, t.MeanGrams, t.FeedKg, strings.Join(auto, ","), "", goal)
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
	out.WriteString(d.line("servi e despesquei"))

	d.press("1")
	d.press("2")
	out.WriteString(d.line("comprei comedouro+aerador"))

	d.press("s")
	out.WriteString(d.line("povoei com alevinos"))

	for _, days := range []int{15, 30, 60, 60} {
		jumpDays(t, days)
		out.WriteString(d.line(fmt.Sprintf("+%d dias", days)))
	}

	for range 3 {
		d.press("up")
	}
	d.press("z")
	for range 4 {
		d.press("down")
	}
	d.press("z")
	d.press("x")
	out.WriteString(d.line("raleei 30% do lote"))

	for _, days := range []int{60, 60} {
		jumpDays(t, days)
		out.WriteString(d.line(fmt.Sprintf("+%d dias", days)))
	}

	d.press("h")
	out.WriteString(d.line("despesquei o resto"))

	d.press("3")
	out.WriteString(d.line("comprei o peao"))

	d.press("p")
	d.press("s")
	out.WriteString(d.line("tilapei"))

	fmt.Fprint(os.Stdout, "\n"+out.String())
}
