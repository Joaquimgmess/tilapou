// Package sim is the deterministic core of the game: Advance takes a state to the next
// tick. It does no I/O and does not import time; every instant is a Tick.
package sim

// TankID is unique within the State and never reused.
type TankID uint32

// BatchID is unique within the State and never reused.
type BatchID uint32

// TankKind selects the tank's TankSpec.
type TankKind uint8

// Tank kinds, from the simplest to the most intensive.
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

// TankKindNamed returns TankEarthPond and false for an unknown name, so the bool
// must be checked.
func TankKindNamed(name string) (TankKind, bool) {
	for kind, known := range tankKindNames {
		if known == name {
			return TankKind(kind), true
		}
	}

	return TankEarthPond, false
}

// TankKindNames returns a copy of the names, in enum order.
func TankKindNames() []string {
	return append([]string(nil), tankKindNames[:]...)
}

var _ [len(tankKindNames) - int(tankKindCount)]struct{}

// Known reports whether the value is inside the enum.
func (k TankKind) Known() bool {
	return k < tankKindCount
}

// String returns "invalid" outside the enum.
func (k TankKind) String() string {
	if k >= tankKindCount {
		return invalidName
	}

	return tankKindNames[k]
}

// Cap of batches per tank and the format versions written to the save.
const (
	maxTanks          = 64
	MaxBatchesPerTank = 4
	StateVersion      = 1
	RngVersion        = 1
)

// Accrual gathers what happened in the tank in the window starting at Window to become a
// single event; deaths in fish, masses in micrograms.
type Accrual struct {
	Window        Tick
	HypoxiaDeaths FishCount
	DiseaseDeaths FishCount
	FeedEaten     Micrograms
	MassGained    Micrograms
}

// Batch is a batch stocked together, with mass in micrograms and cost in cents; the
// Carry fields hold the remainder of the integer divisions between ticks.
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
	// StarvationEpisodeDeaths conta os mortos da seca em curso e StarvationEpisodeFrom guarda
	// o tick em que ela abriu. Sao o payload do fechamento, e nao gatilho: quem abre e fecha
	// o episodio sao as transicoes. Sem o tick de abertura o fechamento afirmava a seca
	// inteira num tick so.
	StarvationEpisodeDeaths FishCount
	StarvationEpisodeFrom   Tick
}

// Biomass saturates instead of overflowing.
func (b *Batch) Biomass() Micrograms {
	return Micrograms(mulDivFloor(int64(b.MeanMass), int64(b.Fish), 1))
}

// Empty reports whether the batch has no live fish left.
func (b *Batch) Empty() bool {
	return b.Fish <= 0
}

// Tank carries feed in micrograms and oxygen in micrograms per litre; only the first
// BatchCount elements of Batches are valid.
type Tank struct {
	ID          TankID
	Kind        TankKind
	Litres      Litres
	Batches     [MaxBatchesPerTank]Batch
	BatchCount  int32
	FeedStock   Micrograms
	ServedUntil Tick
	Upgrades    uint32
	Oxygen      MicrogramsPerLiter
	Aerating    bool
	// AeratorManual guarda a escolha do jogador sobre o aerador. Zero e "sem escolha": e o
	// automatico que manda. A escolha vale ate a histerese fechar — quando o automatico quiser
	// o mesmo valor, ela deixa de existir. Sem isso, quem comprava o aerador perdia a tecla:
	// o automatico reescrevia o estado todo tick e o [a] durava um segundo.
	AeratorManual AeratorChoice
	FeedCarry     int64
	FeedUnitCost  Coins
	UpkeepCarry   int64
	CarrierUntil  Tick
	Accrual       Accrual
}

// Biomass sums the tank's batches, saturating.
func (t *Tank) Biomass() Micrograms {
	var total Micrograms
	for i := range t.BatchCount {
		total = Micrograms(addSat(int64(total), int64(t.Batches[i].Biomass())))
	}

	return total
}

// Capacity returns how many fish fit in the tank, by the maximum density of its kind.
func (t *Tank) Capacity(b *Balance) int64 {
	return b.Tanks[t.Kind].MaxDensityPerM3 * int64(t.Litres) / LitresPerCubicMetre
}

// Fish saturates at the FishCount maximum.
func (t *Tank) Fish() FishCount {
	var total int64
	for i := range t.BatchCount {
		total = addSat(total, int64(t.Batches[i].Fish))
	}

	return FishCount(min(total, maxInt32))
}

// State is the whole game at a tick, copyable by value and free of pointers. Only the
// first TankCount elements of Tanks are valid.
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
	// StuckTicks conta quanto tempo a fazenda esta sem nenhuma acao possivel: sem peixe que
	// valha despesca e sem caixa para um saco de racao. Zera assim que uma das duas volta.
	StuckTicks Tick
}

const (
	maxInt32 = int64(1)<<31 - 1
	// LitresPerCubicMetre converts a tank volume in litres to cubic metres.
	LitresPerCubicMetre = 1_000
)

// NewState creates an empty farm starting at tick at.
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

// Biomass sums every tank, saturating.
func (s *State) Biomass() Micrograms {
	var total Micrograms
	for i := range s.TankCount {
		total = Micrograms(addSat(int64(total), int64(s.Tanks[i].Biomass())))
	}

	return total
}

// Fish saturates at the FishCount maximum.
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
		MassRoot:  MassRootOf(mass),
		StockedAt: at,
	}
	t.BatchCount++

	return true
}

// StockTank takes mass in micrograms and cost in cents, checking neither density nor
// cash; it returns false if the tank does not exist or is already full of batches.
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

// LoadFeed takes unitCost in cents per kilo and returns false if the tank does not exist.
func (s *State) LoadFeed(id TankID, mass Micrograms, unitCost Coins) bool {
	t := s.tank(id)
	if t == nil {
		return false
	}

	loadFeed(t, mass, Coins(mulDivFloor(int64(mass), int64(unitCost), int64(MicrogramsPerKilogram))))

	return true
}

// AddTank returns false, changing nothing, if the farm is already at the tank cap. The new
// tank starts with the oxygen of the water around it, so nothing reads it as suffocating
// until the next tick catches up.
func (s *State) AddTank(b *Balance, kind TankKind, litres Litres) (TankID, bool) {
	if s.TankCount >= maxTanks {
		return 0, false
	}

	id := s.NextTankID
	s.Tanks[s.TankCount] = Tank{ID: id, Kind: kind, Litres: litres}
	s.Tanks[s.TankCount].Oxygen = oxygenAt(b, &s.Tanks[s.TankCount], s.Tick, s.Zone)
	s.TankCount++
	s.NextTankID++

	return id, true
}

func (s *State) tank(id TankID) *Tank {
	for i := range s.TankCount {
		if s.Tanks[i].ID == id {
			return &s.Tanks[i]
		}
	}

	return nil
}
