// Package sim e o nucleo deterministico do jogo: Advance leva um estado ao proximo
// tick. Nao faz I/O nem importa time; todo instante e um Tick.
package sim

// TankID e unico no State e nunca reaproveitado.
type TankID uint32

// BatchID e unico no State e nunca reaproveitado.
type BatchID uint32

// TankKind escolhe o TankSpec do tanque.
type TankKind uint8

// Tipos de tanque, do mais simples ao mais intensivo.
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

// TankKindNamed devolve TankEarthPond e false para nome desconhecido, entao o bool
// precisa ser checado.
func TankKindNamed(name string) (TankKind, bool) {
	for kind, known := range tankKindNames {
		if known == name {
			return TankKind(kind), true
		}
	}

	return TankEarthPond, false
}

// TankKindNames devolve uma copia dos nomes, em ordem de enum.
func TankKindNames() []string {
	return append([]string(nil), tankKindNames[:]...)
}

var _ [len(tankKindNames) - int(tankKindCount)]struct{}

// String devolve "invalid" fora do enum.
func (k TankKind) String() string {
	if k >= tankKindCount {
		return invalidName
	}

	return tankKindNames[k]
}

// Teto de lotes por tanque e as versoes de formato gravadas no save.
const (
	maxTanks          = 64
	MaxBatchesPerTank = 4
	StateVersion      = 1
	RngVersion        = 1
)

// Accrual junta o que houve no tanque na janela iniciada em Window para virar um
// unico evento; mortes em peixes, massas em microgramas.
type Accrual struct {
	Window           Tick
	HypoxiaDeaths    FishCount
	StarvationDeaths FishCount
	DiseaseDeaths    FishCount
	FeedEaten        Micrograms
	MassGained       Micrograms
}

// Batch e um lote estocado junto, com massa em microgramas e custo em centavos; os
// campos Carry guardam o resto das divisoes inteiras entre ticks.
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

// Biomass satura em vez de estourar.
func (b *Batch) Biomass() Micrograms {
	return Micrograms(mulDivFloor(int64(b.MeanMass), int64(b.Fish), 1))
}

// Empty diz que o lote nao tem mais peixes vivos.
func (b *Batch) Empty() bool {
	return b.Fish <= 0
}

// Tank tem racao em microgramas e oxigenio em microgramas por litro; so os primeiros
// BatchCount elementos de Batches valem.
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

// Biomass soma os lotes do tanque, saturando.
func (t *Tank) Biomass() Micrograms {
	var total Micrograms
	for i := range t.BatchCount {
		total = Micrograms(addSat(int64(total), int64(t.Batches[i].Biomass())))
	}

	return total
}

// Known diz se o valor esta dentro do enum.
func (k TankKind) Known() bool {
	return k < tankKindCount
}

// Capacity devolve quantos peixes cabem no tanque, pela densidade maxima do seu tipo.
func (t *Tank) Capacity(b *Balance) int64 {
	return b.Tanks[t.Kind].MaxDensityPerM3 * int64(t.Litres) / litresPerCubicMetre
}

// Fish satura no maximo de FishCount.
func (t *Tank) Fish() FishCount {
	var total int64
	for i := range t.BatchCount {
		total = addSat(total, int64(t.Batches[i].Fish))
	}

	return FishCount(min(total, maxInt32))
}

// State e todo o jogo num tick, copiavel por valor e sem ponteiros. So os primeiros
// TankCount elementos de Tanks valem.
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

// NewState cria uma fazenda vazia comecando no tick at.
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

// Biomass soma todos os tanques, saturando.
func (s *State) Biomass() Micrograms {
	var total Micrograms
	for i := range s.TankCount {
		total = Micrograms(addSat(int64(total), int64(s.Tanks[i].Biomass())))
	}

	return total
}

// Fish satura no maximo de FishCount.
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

// AddTank devolve false, sem mudar nada, se a fazenda ja estiver no teto de tanques.
func (s *State) AddTank(kind TankKind, litres Litres) (TankID, bool) {
	return s.addTank(kind, litres)
}

// StockTank recebe massa em microgramas e custo em centavos, sem checar densidade nem
// caixa; devolve false se o tanque nao existir ou ja estiver cheio de lotes.
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

// LoadFeed recebe unitCost em centavos por quilo e devolve false se o tanque nao existir.
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
