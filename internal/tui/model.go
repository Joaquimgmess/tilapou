package tui

import (
	"context"
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

type Model struct {
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
	return Model{client: c, nextKey: uint64(1)}
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
		return m, tea.Batch(m.fetch(), tick())

	case snapshotMsg:
		m.snapshot, m.err = msg.snapshot, msg.err
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

func (m Model) onKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true

		return m, tea.Quit

	case "r":
		return m, m.fetch()

	case "f":
		return m.act(client.Action{Kind: "buy_feed", Tank: m.firstTank(), Amount: feedPurchaseKg}, "comprando 100 kg de racao")

	case "a":
		aerating := int64(1)
		if m.firstTankIsAerating() {
			aerating = 0
		}

		return m.act(client.Action{Kind: "aerate", Tank: m.firstTank(), Amount: aerating}, "alternando aerador")

	case "h":
		return m.act(client.Action{Kind: "harvest", Tank: m.firstTank(), Batch: m.firstBatch()}, "despescando o lote")

	case "s":
		return m.act(client.Action{Kind: "stock", Tank: m.firstTank(), Amount: fingerlingsPerBuy}, "povoando com 500 alevinos")

	case "t":
		return m.act(client.Action{Kind: "buy_tank", TankKind: "viveiro_escavado"}, "comprando viveiro")

	case "p":
		return m.act(client.Action{Kind: "prestige"}, "tilapando: vendendo tudo e recomecando")

	case "1", "2", "3", "4", "5":
		return m.buyUpgrade(int(msg.String()[0] - '1'))
	}

	return m, nil
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
