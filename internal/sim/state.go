package sim

type TankID uint32

type BatchID uint32

type TankKind uint8

const (
	TankEarthPond TankKind = iota
	TankNetCage
	TankBiofloc
	TankRecirculation
	tankKindCount
)

const (
	maxTanks          = 64
	maxBatchesPerTank = 4
	StateVersion      = 1
	RngVersion        = 1
)

type Accrual struct {
	Window           Tick
	HypoxiaDeaths    FishCount
	StarvationDeaths FishCount
	FeedEaten        Micrograms
	MassGained       Micrograms
}

type Batch struct {
	ID              BatchID
	Fish            FishCount
	MeanMass        Micrograms
	MassRoot        int64
	GrowthCarry     int64
	StockedAt       Tick
	FeedEaten       Micrograms
	MassGained      Micrograms
	HypoxiaTicks    int32
	StarvationTicks int32
}

func (b *Batch) Biomass() Micrograms {
	return Micrograms(mulDivFloor(int64(b.MeanMass), int64(b.Fish), 1))
}

func (b *Batch) Empty() bool {
	return b.Fish <= 0
}

type Tank struct {
	ID         TankID
	Kind       TankKind
	Litres     Litres
	Batches    [maxBatchesPerTank]Batch
	BatchCount int32
	FeedStock  Micrograms
	Oxygen     MicrogramsPerLiter
	Aerating   bool
	FeedCarry  int64
	Accrual    Accrual
}

func (t *Tank) Biomass() Micrograms {
	var total Micrograms
	for i := range t.BatchCount {
		total = Micrograms(addSat(int64(total), int64(t.Batches[i].Biomass())))
	}

	return total
}

func (t *Tank) Fish() FishCount {
	var total int64
	for i := range t.BatchCount {
		total = addSat(total, int64(t.Batches[i].Fish))
	}

	return FishCount(min(total, maxInt32))
}

type State struct {
	Version        uint16
	BalanceVersion uint16
	RngVersion     uint16
	Seed           Seed
	Zone           ZoneOffset
	Tick           Tick
	Cash           Coins
	LifetimeEarned Coins
	Prestige       uint32
	Upgrades       uint32
	Tanks          [maxTanks]Tank
	TankCount      int32
	NextTankID     TankID
	NextBatchID    BatchID
	EventSeq       uint64
	Saturated      bool
}

const maxInt32 = int64(1)<<31 - 1

func NewState(seed Seed, zone ZoneOffset, at Tick) State {
	return State{
		Version:        StateVersion,
		BalanceVersion: 1,
		RngVersion:     RngVersion,
		Seed:           seed,
		Zone:           zone,
		Tick:           at,
		NextTankID:     1,
		NextBatchID:    1,
	}
}

func (s *State) Biomass() Micrograms {
	var total Micrograms
	for i := range s.TankCount {
		total = Micrograms(addSat(int64(total), int64(s.Tanks[i].Biomass())))
	}

	return total
}

func (s *State) Fish() FishCount {
	var total int64
	for i := range s.TankCount {
		total = addSat(total, int64(s.Tanks[i].Fish()))
	}

	return FishCount(min(total, maxInt32))
}

func (t *Tank) addBatch(id BatchID, fish FishCount, mass Micrograms, at Tick) bool {
	if t.BatchCount >= maxBatchesPerTank {
		return false
	}

	t.Batches[t.BatchCount] = Batch{
		ID:        id,
		Fish:      fish,
		MeanMass:  mass,
		MassRoot:  massRootOf(mass),
		StockedAt: at,
	}
	t.BatchCount++

	return true
}

func (s *State) AddTank(kind TankKind, litres Litres) (TankID, bool) {
	return s.addTank(kind, litres)
}

func (s *State) StockTank(id TankID, fish FishCount, mass Micrograms) bool {
	t := s.tank(id)
	if t == nil {
		return false
	}

	if !t.addBatch(s.NextBatchID, fish, mass, s.Tick) {
		return false
	}
	s.NextBatchID++

	return true
}

func (s *State) LoadFeed(id TankID, mass Micrograms) bool {
	t := s.tank(id)
	if t == nil {
		return false
	}

	t.FeedStock = Micrograms(addSat(int64(t.FeedStock), int64(mass)))

	return true
}

func (s *State) tank(id TankID) *Tank {
	for i := range s.TankCount {
		if s.Tanks[i].ID == id {
			return &s.Tanks[i]
		}
	}

	return nil
}

func (s *State) addTank(kind TankKind, litres Litres) (TankID, bool) {
	if s.TankCount >= maxTanks {
		return 0, false
	}

	id := s.NextTankID
	s.Tanks[s.TankCount] = Tank{ID: id, Kind: kind, Litres: litres}
	s.TankCount++
	s.NextTankID++

	return id, true
}
