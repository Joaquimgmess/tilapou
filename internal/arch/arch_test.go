package arch_test

import (
	"os/exec"
	"slices"
	"strings"
	"testing"
)

const module = "github.com/Joaquimgmess/tilapou"

func deps(t *testing.T, pkg string) []string {
	t.Helper()

	out, err := exec.CommandContext(t.Context(), "go", "list", "-deps", pkg).Output()
	if err != nil {
		t.Fatalf("go list -deps %s: %v", pkg, err)
	}

	return strings.Fields(string(out))
}

func TestSimIsPureAllTheWayDown(t *testing.T) {
	t.Parallel()

	allowed := []string{"errors", "math/bits", "strconv", "unsafe", module + "/internal/sim"}

	for _, dep := range deps(t, module+"/internal/sim") {
		if isRuntimePlumbing(dep) || slices.Contains(allowed, dep) {
			continue
		}
		t.Errorf("internal/sim depende de %q, fora da lista permitida: o nucleo tem que ser puro", dep)
	}
}

func isRuntimePlumbing(pkg string) bool {
	return pkg == "runtime" ||
		strings.HasPrefix(pkg, "runtime/") ||
		strings.HasPrefix(pkg, "internal/")
}

func TestTuiNeverReachesTheSimulationOrTheDatabase(t *testing.T) {
	t.Parallel()

	forbidden := []string{
		module + "/internal/sim",
		module + "/internal/farm",
		module + "/internal/platform/postgres",
		"database/sql",
	}

	for _, pkg := range []string{module + "/internal/tui", module + "/internal/tui/..."} {
		for _, dep := range deps(t, pkg) {
			if slices.Contains(forbidden, dep) {
				t.Errorf("%s depende de %q: a TUI desenha, nao calcula nem persiste", pkg, dep)
			}
		}
	}
}
