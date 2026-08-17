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

var tankKindNames = [...]string{
	TankEarthPond:     "viveiro_escavado",
	TankNetCage:       "tanque_rede",
	TankBiofloc:       "bioflocos",
	TankRecirculation: "recirculacao",
}

func TankKindNamed(name string) (TankKind, bool) {
	for kind, known := range tankKindNames {
		if known == name {
			return TankKind(kind), true
		}
	}

	return TankEarthPond, false
}

func TankKindNames() []string {
	return append([]string(nil), tankKindNames[:]...)
}

var _ [len(tankKindNames) - int(tankKindCount)]struct{}

func (k TankKind) String() string {
	if k >= tankKindCount {
		return invalidName
	}

	return tankKindNames[k]
}

const (
	maxTanks          = 64
	MaxBatchesPerTank = 4
	StateVersion      = 1
	RngVersion        = 1
)

type Accrual struct {
	Window           Tick
	HypoxiaDeaths    FishCount
	StarvationDeaths FishCount
	DiseaseDeaths    FishCount
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
	Cost            Coins
	CostCarry       int64
	Sick            int32
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
	ID           TankID
	Kind         TankKind
	Litres       Litres
	Batches      [MaxBatchesPerTank]Batch
	BatchCount   int32
	FeedStock    Micrograms
	ServedUntil  Tick
	Upgrades     uint32
	Oxygen       MicrogramsPerLiter
	Aerating     bool
	FeedCarry    int64
	FeedUnitCost Coins
	UpkeepCarry  int64
	CarrierUntil Tick
	Accrual      Accrual
}

func (t *Tank) Biomass() Micrograms {
	var total Micrograms
	for i := range t.BatchCount {
		total = Micrograms(addSat(int64(total), int64(t.Batches[i].Biomass())))
	}

	return total
}

func (k TankKind) Known() bool {
	return k < tankKindCount
}

func (t *Tank) Capacity(b *Balance) int64 {
	return b.Tanks[t.Kind].MaxDensityPerM3 * int64(t.Litres) / litresPerCubicMetre
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
	Tanks          [maxTanks]Tank
	TankCount      int32
	NextTankID     TankID
	NextBatchID    BatchID
	EventSeq       uint64
	Debt           Coins
	DebtCarry      int64
	LastCycle      Cycle
}

const (
	maxInt32            = int64(1)<<31 - 1
	litresPerCubicMetre = 1_000
)

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

func (t *Tank) compact() {
	kept := int32(0)
	for i := range t.BatchCount {
		if t.Batches[i].Empty() {
			continue
		}
		t.Batches[kept] = t.Batches[i]
		kept++
	}

	for i := kept; i < t.BatchCount; i++ {
		t.Batches[i] = Batch{}
	}
	t.BatchCount = kept
}

func (t *Tank) addBatch(id BatchID, fish FishCount, mass Micrograms, at Tick) bool {
	if t.BatchCount >= MaxBatchesPerTank {
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

func (s *State) StockTank(id TankID, fish FishCount, mass Micrograms, cost Coins) bool {
	t := s.tank(id)
	if t == nil {
		return false
	}

	if !t.addBatch(s.NextBatchID, fish, mass, s.Tick) {
		return false
	}
	t.Batches[t.BatchCount-1].Cost = cost
	s.NextBatchID++

	return true
}

func (s *State) LoadFeed(id TankID, mass Micrograms, unitCost Coins) bool {
	t := s.tank(id)
	if t == nil {
		return false
	}

	loadFeed(t, mass, Coins(mulDivFloor(int64(mass), int64(unitCost), int64(MicrogramsPerKilogram))))

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
