package sim

// WindowTicks is the window, in ticks, over which a tank's accruals become a single event.
const WindowTicks = Tick(60)

// EventKind is the kind of a simulation event.
type EventKind uint16

// Event kinds emitted while the simulation advances.
const (
	EventUnknown EventKind = iota
	EventGrowth
	EventHypoxiaDeaths
	EventStarvationBegan
	EventStarvationEnded
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
	EventBankrupt
	EventKindCount
)

var eventKindNames = [...]string{
	EventUnknown:         "unknown",
	EventGrowth:          "growth",
	EventHypoxiaDeaths:   "hypoxia_deaths",
	EventStarvationBegan: "starvation_began",
	EventStarvationEnded: "starvation_ended",
	EventFeedExhausted:   "feed_exhausted",
	EventHarvest:         "harvest",
	EventStocked:         "stocked",
	EventTankBought:      "tank_bought",
	EventFeedBought:      "feed_bought",
	EventActionRejected:  "action_rejected",
	EventUpgradeBought:   "upgrade_bought",
	EventPrestiged:       "prestiged",
	EventRestarted:       "restarted",
	EventBorrowed:        "borrowed",
	EventRepaid:          "repaid",
	EventDisease:         "disease",
	EventDiseaseDeaths:   "disease_deaths",
	EventTreated:         "treated",
	EventBankrupt:        "bankrupt",
}

var _ [len(eventKindNames) - int(EventKindCount)]struct{}

func (k EventKind) String() string {
	if k >= EventKindCount {
		return invalidName
	}

	return eventKindNames[k]
}

// Event carries mass in micrograms and cash in cents; Seq orders the emission and the remaining fields depend on Kind.
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
	return a.HypoxiaDeaths == 0 && a.DiseaseDeaths == 0 &&
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
