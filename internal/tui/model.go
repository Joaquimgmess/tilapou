package tui

import (
	"context"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joaquimgmess/catalog/internal/client"
)

const (
	refreshEvery      = time.Second
	callTimeout       = 5 * time.Second
	feedPurchaseKg    = 100
	fingerlingsPerBuy = 500
)

type snapshotMsg struct {
	snapshot client.Snapshot
	err      error
}

type tickMsg time.Time

type Mode uint8

const (
	ModeGameBoy Mode = iota
	ModeDashboard
)

type Model struct {
	mode     Mode
	farm     farmMap
	you      avatar
	frame    int
	message  string
	client   *client.Client
	snapshot client.Snapshot
	err      error
	status   string
	nextKey  uint64
	width    int
	height   int
	quitting bool
	view     string
}

func New(c *client.Client) Model {
	return Model{client: c, nextKey: uint64(1), farm: newFarmMap(1), you: newAvatar()}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.fetch(), tick())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.view = ""

		return m, nil

	case tickMsg:
		m.frame++
		m.view = ""

		return m, tea.Batch(m.fetch(), tick())

	case snapshotMsg:
		m.snapshot, m.err = msg.snapshot, msg.err
		m.farm = newFarmMap(len(msg.snapshot.Tanks))
		if msg.err == nil && msg.snapshot.LastReason != "" && msg.snapshot.LastReason != "none" {
			m.status = "recusado: " + msg.snapshot.LastReason
		}
		m.view = ""

		return m, nil

	case tea.KeyPressMsg:
		return m.onKey(msg)
	}

	return m, nil
}

var movements = map[string]struct {
	dx, dy int
	facing byte
}{
	"up":    {0, -1, 'u'},
	"w":     {0, -1, 'u'},
	"down":  {0, 1, 'd'},
	"s":     {0, 1, 'd'},
	"left":  {-1, 0, 'l'},
	"right": {1, 0, 'r'},
}

func (m Model) onKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if move, ok := movements[key]; ok && m.mode == ModeGameBoy {
		return m.move(move.dx, move.dy, move.facing), nil
	}

	if index := strings.IndexByte("12345", key[0]); len(key) == 1 && index >= 0 {
		return m.buyUpgrade(index)
	}

	switch key {
	case "q", "ctrl+c":
		m.quitting = true

		return m, tea.Quit

	case "r":
		return m, m.fetch()

	case "m":
		m.mode, m.view = m.otherMode(), ""

		return m, nil

	case "z", "enter":
		return m.onInteract()

	case "f":
		return m.act(client.Action{Kind: "buy_feed", Tank: m.firstTank(), Amount: feedPurchaseKg}, "comprando racao")

	case "a":
		return m.act(client.Action{Kind: "aerate", Tank: m.firstTank(), Amount: m.aeratorToggle()}, "alternando aerador")

	case "h":
		return m.act(client.Action{Kind: "harvest", Tank: m.firstTank(), Batch: m.firstBatch()}, "despescando o lote")

	case "S":
		return m.act(client.Action{Kind: "stock", Tank: m.firstTank(), Amount: fingerlingsPerBuy}, "povoando com alevinos")

	case "t":
		return m.act(client.Action{Kind: "buy_tank", TankKind: "viveiro_escavado"}, "comprando viveiro")

	case "p":
		return m.act(client.Action{Kind: "prestige"}, "tilapando: vendendo tudo e recomecando")
	}

	return m, nil
}

func (m Model) otherMode() Mode {
	if m.mode == ModeGameBoy {
		return ModeDashboard
	}

	return ModeGameBoy
}

func (m Model) aeratorToggle() int64 {
	if m.firstTankIsAerating() {
		return 0
	}

	return 1
}

func (m Model) onInteract() (tea.Model, tea.Cmd) {
	updated, target := m.interact()
	switch {
	case target == "feed":
		return updated.act(client.Action{Kind: "buy_feed", Tank: updated.firstTank(), Amount: feedPurchaseKg}, "comprando 100 kg de racao")
	case strings.HasPrefix(target, "tank:"):
		updated.message = ""
		updated.view = ""

		return updated, nil
	}

	return updated, nil
}

func (m Model) buyUpgrade(index int) (tea.Model, tea.Cmd) {
	if index < 0 || index >= len(m.snapshot.Upgrades) {
		return m, nil
	}

	upgrade := m.snapshot.Upgrades[index]
	if upgrade.Owned {
		m.status = upgrade.Kind + " ja esta na fazenda"
		m.view = ""

		return m, nil
	}

	return m.act(client.Action{Kind: "buy_upgrade", Auto: upgrade.Kind}, "comprando "+upgrade.Kind)
}

func (m Model) act(action client.Action, status string) (tea.Model, tea.Cmd) {
	action.Key = m.nextKey
	m.nextKey++
	m.status = status
	m.message = status
	m.view = ""

	c := m.client

	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()

		snapshot, err := c.Act(ctx, action)

		return snapshotMsg{snapshot: snapshot, err: err}
	}
}

func (m Model) fetch() tea.Cmd {
	c := m.client

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()

		snapshot, err := c.Snapshot(ctx)

		return snapshotMsg{snapshot: snapshot, err: err}
	}
}

func tick() tea.Cmd {
	return tea.Tick(refreshEvery, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) firstTank() uint32 {
	if len(m.snapshot.Tanks) == 0 {
		return 0
	}

	return m.snapshot.Tanks[0].ID
}

func (m Model) firstBatch() uint32 {
	if len(m.snapshot.Tanks) == 0 {
		return 0
	}

	return m.snapshot.Tanks[0].BatchID
}

func (m Model) firstTankIsAerating() bool {
	if len(m.snapshot.Tanks) == 0 {
		return false
	}

	return m.snapshot.Tanks[0].Aerating
}
