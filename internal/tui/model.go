// Package tui only draws the snapshot coming from the daemon, without computing or storing state.
package tui

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joaquimgmess/tilapou/internal/client"
)

const (
	refreshEvery   = time.Second
	callTimeout    = 5 * time.Second
	feedPurchaseKg = 100
	messageTicks   = 5
)

type snapshotMsg struct {
	snapshot client.Snapshot
	err      error
}

type tickMsg time.Time

// Mode is the active screen.
type Mode uint8

// The available screens.
const (
	ModeGameBoy Mode = iota
	ModeDashboard
)

// Model is the screen state and implements tea.Model.
type Model struct {
	mode       Mode
	farm       farmMap
	you        avatar
	frame      int
	message    string
	selected   int
	confirming bool
	restarting bool
	menu       *menu
	client     *client.Client
	snapshot   client.Snapshot
	err        error
	nextKey    uint64
	width      int
	height     int
	quitting   bool
	messageAt  int
	staleTicks int
}

// New creates the initial Model bound to the daemon client.
func New(c *client.Client) Model {
	var seed [8]byte
	_, _ = rand.Read(seed[:])

	farm := newFarmMap(1, minMapCols, minMapRows)

	return Model{
		client:  c,
		nextKey: binary.BigEndian.Uint64(seed[:]),
		farm:    farm,
		you:     newAvatar(farm),
	}
}

// Init starts the first snapshot fetch and the periodic tick.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.fetch(), tick())
}

// Update handles messages and returns a new Model, without changing the original.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.farm = m.resizedFarm(len(m.snapshot.Tanks))
		m.you = m.you.clampedTo(m.farm)

		return m, nil

	case tickMsg:
		m.frame++
		m.staleTicks++

		if m.message != "" && !m.confirming && m.frame-m.messageAt >= messageTicks {
			m.message = ""
		}

		return m, tea.Batch(m.fetch(), tick())

	case snapshotMsg:
		return m.onSnapshot(msg), nil

	case tea.KeyPressMsg:
		return m.onKey(msg)
	}

	return m, nil
}

func (m Model) onSnapshot(msg snapshotMsg) Model {
	m.err = msg.err
	if msg.err != nil {
		return m
	}

	m.snapshot, m.staleTicks = msg.snapshot, 0
	m.farm = m.resizedFarm(len(msg.snapshot.Tanks))
	m.selected = min(m.selected, max(len(msg.snapshot.Tanks)-1, 0))

	if failure := explain(msg.snapshot.LastOutcome, msg.snapshot.CashCents); failure != "" {
		return m.say(failure)
	}
	m.menu = m.refreshedMenu()

	return m
}

// resizedFarm rebuilds the map for the current terminal, so the grass grows with the screen.
func (m Model) resizedFarm(tanks int) farmMap {
	cols, rows := mapSizeFor(m.width, m.height)

	return newFarmMap(tanks, cols, rows)
}

func (m Model) say(text string) Model {
	m.message, m.messageAt = text, m.frame

	return m
}

type movement struct {
	dx, dy int
	facing facing
}

// Arrow keys, as bubbletea names them.
const (
	keyUp    = "up"
	keyDown  = "down"
	keyLeft  = "left"
	keyRight = "right"
)

func movementFor(key string) (movement, bool) {
	switch key {
	case keyUp:
		return movement{dx: 0, dy: -1, facing: facingUp}, true
	case keyDown:
		return movement{dx: 0, dy: 1, facing: facingDown}, true
	case keyLeft:
		return movement{dx: -1, dy: 0, facing: facingLeft}, true
	case keyRight:
		return movement{dx: 1, dy: 0, facing: facingRight}, true
	}

	return movement{}, false
}

func (m Model) onKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if move, ok := movementFor(key); ok {
		if m.mode == ModeGameBoy || m.menu != nil {
			return m.move(move.dx, move.dy, move.facing), nil
		}
		if move.dy != 0 {
			return m.selectDelta(move.dy), nil
		}

		return m, nil
	}

	if m.confirming {
		return m.onConfirm(key)
	}

	if m.menu != nil {
		return m.onMenuKey(key)
	}

	switch key {
	case "j":
		return m.selectDelta(1), nil
	case "k":
		return m.selectDelta(-1), nil
	}

	if len(key) == 1 {
		if index := strings.IndexByte("12345", key[0]); index >= 0 {
			return m.buyUpgrade(index)
		}
	}

	return m.onCommand(key)
}

func (m Model) onCommand(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "ctrl+c":
		m.quitting = true

		return m, tea.Quit

	case "r":
		return m, m.fetch()

	case "tab":
		m.mode = m.otherMode()

		return m, nil

	case "z", "enter":
		return m.onInteract()

	case jumpKey:
		return m.jumpToAdvice(), nil

	case "f":
		return m.act(client.Action{Kind: "feed", Tank: m.tankID()}, "servindo o trato")

	case "c":
		return m.act(client.Action{Kind: "buy_feed", Tank: m.tankID(), Amount: feedPurchaseKg},
			"comprando 100 kg de racao")

	case "a":
		return m.act(client.Action{Kind: "aerate", Tank: m.tankID(), Amount: m.aeratorToggle()},
			"alternando o aerador")

	case "h":
		return m.act(client.Action{Kind: "harvest", Tank: m.tankID(), Batch: m.batchID()}, "despescando o lote")

	case "g":
		return m.openShed()

	case "s":
		return m.stock()

	case "t":
		return m.act(client.Action{Kind: "buy_tank", TankKind: "viveiro_escavado"}, "comprando um viveiro")

	case "p":
		return m.askPrestige()

	case "b":
		return m.askRestart()
	}

	return m, nil
}

func (m Model) onConfirm(key string) (tea.Model, tea.Cmd) {
	restarting := m.restarting
	next := m.clearPrompt()

	if key != "y" && key != "s" {
		return next.say("cancelado"), nil
	}
	if restarting {
		return next.act(client.Action{Kind: "restart"}, "recomecando do zero")
	}

	return next.act(client.Action{Kind: "prestige"}, "tilapando: vendendo tudo e recomecando")
}

func (m Model) askPrestige() (tea.Model, tea.Cmd) {
	if m.snapshot.PrestigeNow <= m.snapshot.Prestige {
		return m.say("Ainda nao da para tilapar: fature mais antes"), nil
	}

	m = m.say("Tilapar zera tanques, caixa e automacoes. Confirma? [s/n]")
	m.confirming = true

	return m, nil
}

func (m Model) clearPrompt() Model {
	m.confirming, m.restarting = false, false

	return m
}

func (m Model) askRestart() (tea.Model, tea.Cmd) {
	if !m.snapshot.Broke {
		return m.say("So da para recomecar quando a fazenda quebra de vez"), nil
	}

	m = m.say("Recomecar zera a divida e devolve o lote inicial, sem ganhar prestigio. Confirma? [s/n]")
	m.confirming, m.restarting = true, true

	return m, nil
}

func (m Model) stock() (tea.Model, tea.Cmd) {
	tank, ok := m.tank()
	if !ok {
		return m, nil
	}

	amount := tank.StockAdvice
	if amount <= 0 {
		return m.say(stockBlocked(tank)), nil
	}

	return m.act(client.Action{Kind: "stock", Tank: tank.ID, Amount: amount},
		fmt.Sprintf("povoando com %d alevinos", amount))
}

func (m Model) openShed() (tea.Model, tea.Cmd) {
	tank, ok := m.tank()
	if !ok {
		return m, nil
	}
	m.menu, m.message = shedMenu(m.snapshot, tank), ""

	return m, nil
}

func (m Model) onInteract() (tea.Model, tea.Cmd) {
	if m.mode == ModeDashboard {
		tank, ok := m.tank()
		if !ok {
			return m, nil
		}
		m.menu, m.message = tankMenu(m.snapshot, tank), ""

		return m, nil
	}

	index, kind := m.target()

	tank, ok := m.tank()
	if !ok {
		return m, nil
	}

	switch kind {
	case targetShed:
		m.menu = shedMenu(m.snapshot, tank)
	case targetTank:
		m.selected = index
		m.menu = tankMenu(m.snapshot, m.snapshot.Tanks[index])
	case targetNone:
		return m, nil
	}

	m.message = ""

	return m, nil
}

func (m Model) onMenuKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j":
		m.menu.move(1)

		return m, nil

	case "k":
		m.menu.move(-1)

		return m, nil

	case "x", "esc", "q":
		m.menu = nil

		return m, nil

	case "z", "enter":
		item, ok := m.menu.current()

		if !ok {
			return m, nil
		}
		if item.panel {
			m.menu, m.mode = nil, ModeDashboard

			return m, nil
		}
		if !item.enabled {
			return m.say("Nao da agora: " + item.hint), nil
		}

		return m.act(item.action, item.status)
	}

	return m, nil
}

func (m Model) buyUpgrade(index int) (tea.Model, tea.Cmd) {
	tank, ok := m.tank()
	if !ok || index >= len(tank.Upgrades) {
		return m, nil
	}

	upgrade := tank.Upgrades[index]
	if upgrade.Owned {
		return m.say("o tanque " + strconv.FormatUint(uint64(tank.ID), 10) + " ja tem " + upgrade.Kind), nil
	}
	if m.snapshot.CashCents < upgrade.CostCents {
		return m.say("Sem grana: " + upgrade.Kind + " custa " + coins(upgrade.CostCents) +
			" e faltam " + coins(upgrade.CostCents-m.snapshot.CashCents)), nil
	}

	return m.act(client.Action{Kind: "buy_upgrade", Tank: tank.ID, Auto: upgrade.Kind},
		"comprando "+upgrade.Kind+" para o tanque "+strconv.FormatUint(uint64(tank.ID), 10))
}

func (m Model) act(action client.Action, status string) (tea.Model, tea.Cmd) {
	action.Key = m.nextKey
	m.nextKey++
	m = m.say(status)

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

func (m Model) refreshedMenu() *menu {
	if m.menu == nil {
		return nil
	}

	tank, ok := m.tank()
	if !ok {
		return nil
	}

	rebuilt := tankMenu(m.snapshot, tank)
	if m.menu.title == shedTitle {
		rebuilt = shedMenu(m.snapshot, tank)
	}
	rebuilt.cursor = min(m.menu.cursor, len(rebuilt.items)-1)

	return rebuilt
}

func (m Model) otherMode() Mode {
	if m.mode == ModeGameBoy {
		return ModeDashboard
	}

	return ModeGameBoy
}

func (m Model) selectDelta(delta int) Model {
	if len(m.snapshot.Tanks) == 0 {
		return m
	}

	m.selected = (m.selected + delta + len(m.snapshot.Tanks)) % len(m.snapshot.Tanks)

	return m
}

func (m Model) jumpToAdvice() Model {
	target := adviceTank(m.snapshot)
	for i := range m.snapshot.Tanks {
		if m.snapshot.Tanks[i].ID == target {
			m.selected, m.message = i, ""

			return m
		}
	}

	return m
}

func (m Model) tank() (client.Tank, bool) {
	if m.selected < 0 || m.selected >= len(m.snapshot.Tanks) {
		return client.Tank{}, false
	}

	return m.snapshot.Tanks[m.selected], true
}

func (m Model) tankID() uint32 {
	tank, ok := m.tank()
	if !ok {
		return 0
	}

	return tank.ID
}

func (m Model) batchID() uint32 {
	tank, ok := m.tank()
	if !ok {
		return 0
	}

	return tank.BatchID
}

func (m Model) aeratorToggle() int64 {
	tank, ok := m.tank()
	if ok && tank.Aerating {
		return 0
	}

	return 1
}
