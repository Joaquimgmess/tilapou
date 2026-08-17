package sim

// WindowTicks e a janela, em ticks, em que os acumulados de um tanque viram um so evento.
const WindowTicks = Tick(60)

// EventKind e o tipo de um evento da simulacao.
type EventKind uint16

// Tipos de evento emitidos durante o avanco da simulacao.
const (
	EventUnknown EventKind = iota
	EventGrowth
	EventHypoxiaDeaths
	EventStarvationDeaths
	EventFeedExhausted
	EventHarvest
	EventStocked
	EventTankBought
	EventFeedBought
	EventActionRejected
	EventUpgradeBought
	EventPrestiged
	EventRestarted
	EventBorrowed
	EventRepaid
	EventDisease
	EventDiseaseDeaths
	EventTreated
	eventKindCount
)

var eventKindNames = [...]string{
	EventUnknown:          "unknown",
	EventGrowth:           "growth",
	EventHypoxiaDeaths:    "hypoxia_deaths",
	EventStarvationDeaths: "starvation_deaths",
	EventFeedExhausted:    "feed_exhausted",
	EventHarvest:          "harvest",
	EventStocked:          "stocked",
	EventTankBought:       "tank_bought",
	EventFeedBought:       "feed_bought",
	EventActionRejected:   "action_rejected",
	EventUpgradeBought:    "upgrade_bought",
	EventPrestiged:        "prestiged",
	EventRestarted:        "restarted",
	EventBorrowed:         "borrowed",
	EventRepaid:           "repaid",
	EventDisease:          "disease",
	EventDiseaseDeaths:    "disease_deaths",
	EventTreated:          "treated",
}

var _ [len(eventKindNames) - int(eventKindCount)]struct{}

func (k EventKind) String() string {
	if k >= eventKindCount {
		return invalidName
	}

	return eventKindNames[k]
}

// Event tem massa em microgramas e caixa em centavos; Seq ordena a emissao e os demais campos dependem do Kind.
type Event struct {
	Seq    uint64
	Kind   EventKind
	From   Tick
	To     Tick
	Tank   TankID
	Batch  BatchID
	Fish   FishCount
	Mass   Micrograms
	Cash   Coins
	Reason RejectReason
}

type eventSink struct {
	events  []Event
	nextSeq uint64
}

func (s *eventSink) emit(e Event) {
	e.Seq = s.nextSeq
	s.nextSeq++
	s.events = append(s.events, e)
}

func (a *Accrual) empty() bool {
	return a.HypoxiaDeaths == 0 && a.StarvationDeaths == 0 && a.DiseaseDeaths == 0 &&
		a.FeedEaten == 0 && a.MassGained == 0
}

func (t *Tank) flushAccrual(sink *eventSink, upTo Tick) {
	if t.Accrual.empty() {
		t.Accrual = Accrual{Window: upTo}
		return
	}

	from, to := t.Accrual.Window, t.Accrual.Window+WindowTicks

	if t.Accrual.MassGained != 0 || t.Accrual.FeedEaten != 0 {
		sink.emit(Event{
			Kind: EventGrowth,
			From: from,
			To:   to,
			Tank: t.ID,
			Mass: t.Accrual.MassGained,
			Fish: 0,
		})
	}
	if t.Accrual.HypoxiaDeaths > 0 {
		sink.emit(Event{
			Kind: EventHypoxiaDeaths,
			From: from,
			To:   to,
			Tank: t.ID,
			Fish: t.Accrual.HypoxiaDeaths,
		})
	}
	if t.Accrual.StarvationDeaths > 0 {
		sink.emit(Event{
			Kind: EventStarvationDeaths,
			From: from,
			To:   to,
			Tank: t.ID,
			Fish: t.Accrual.StarvationDeaths,
		})
	}
	if t.Accrual.DiseaseDeaths > 0 {
		sink.emit(Event{
			Kind: EventDiseaseDeaths,
			From: from,
			To:   to,
			Tank: t.ID,
			Fish: t.Accrual.DiseaseDeaths,
		})
	}

	t.Accrual = Accrual{Window: upTo}
}
