package sim

const (
	forecastCapDays   = 400
	forecastFeedFloor = 50 * MicrogramsPerKilogram
	forecastFeedTopUp = 500 * MicrogramsPerKilogram
)

// Forecast carries masses in micrograms and values in cents; Reached is false when the day cap came before the target.
type Forecast struct {
	Reached    bool
	Days       int64
	MeanMass   Micrograms
	Fish       FishCount
	Cost       Coins
	FeedEaten  Micrograms
	Value      Coins
	Margin     Coins
	PricePerKg Coins
}

// ForecastInput is everything the projection reads: the tank by value plus the farm
// scalars that change growth. What is not here cannot change the result.
type ForecastInput struct {
	Tank     Tank
	Batch    BatchID
	At       Tick
	Zone     ZoneOffset
	Prestige uint32
	Target   Micrograms
}

// ForecastAhead projects the batch in an isolated, well-tended tank; it returns the zero
// value if the batch does not exist or is empty.
func ForecastAhead(b *Balance, in ForecastInput) Forecast {
	start := State{
		Version:  StateVersion,
		Tick:     in.At,
		Zone:     in.Zone,
		Prestige: in.Prestige,
	}
	start.Tanks[0] = in.Tank
	start.TankCount = 1

	return forecastFrom(b, start, in.Tank.ID, in.Batch, in.Target)
}

// Forecast projects the batch in an isolated, well-tended tank; it returns the zero value if the batch does not exist or is empty.
func (s *State) Forecast(b *Balance, tank TankID, batch BatchID, target Micrograms) Forecast {
	return forecastFrom(b, s.isolate(tank), tank, batch, target)
}

func forecastFrom(b *Balance, start State, tank TankID, batch BatchID, target Micrograms) Forecast {
	found := start.batch(tank, batch)
	if found == nil || found.Empty() {
		return Forecast{}
	}

	spent, ate := found.Cost, found.FeedEaten
	from := start.Tick

	for day := range int64(forecastCapDays) {
		keepManaged(&start, b, tank)

		out, err := Advance(Input{State: start, Until: from + Tick(day+1)*TicksPerDay, Balance: b})
		if err != nil {
			break
		}
		start = out.State

		current := start.batch(tank, batch)
		if current == nil || current.Empty() {
			break
		}
		if current.MeanMass >= target {
			return closeForecast(&start, b, current, spent, ate, day+1, true)
		}
	}

	current := start.batch(tank, batch)
	if current == nil {
		return Forecast{}
	}

	return closeForecast(&start, b, current, spent, ate, forecastCapDays, false)
}

func (s *State) isolate(tank TankID) State {
	only := *s
	only.TankCount = 0

	if t := s.tank(tank); t != nil {
		only.Tanks[0] = *t
		only.TankCount = 1
	}

	return only
}

func keepManaged(s *State, b *Balance, tank TankID) {
	t := s.tank(tank)
	if t == nil {
		return
	}

	t.ServedUntil = s.Tick + TicksPerDay
	t.Aerating = wantsAeration(t, b)

	if t.FeedStock >= forecastFeedFloor {
		return
	}

	missing := forecastFeedTopUp - t.FeedStock
	kilos := int64(missing) / int64(MicrogramsPerKilogram)
	price := Coins(mulDivCeil(int64(MarketAt(b, s.Tick).FeedKg), kilos, 1))

	loadFeed(t, missing, price)
}

func closeForecast(s *State, b *Balance, batch *Batch, spent Coins, ate Micrograms, days int64, reached bool) Forecast {
	price := b.PriceFor(batch.MeanMass, s.Tick)
	value := Coins(mulDivFloor(int64(price), int64(batch.Biomass()), int64(MicrogramsPerKilogram)))

	return Forecast{
		Reached:    reached,
		Days:       days,
		MeanMass:   batch.MeanMass,
		Fish:       batch.Fish,
		Cost:       Coins(subSat(int64(batch.Cost), int64(spent))),
		FeedEaten:  Micrograms(subSat(int64(batch.FeedEaten), int64(ate))),
		Value:      value,
		Margin:     Coins(subSat(int64(value), int64(batch.Cost))),
		PricePerKg: price,
	}
}

func (s *State) batch(tank TankID, batch BatchID) *Batch {
	t := s.tank(tank)
	if t == nil {
		return nil
	}

	for i := range t.BatchCount {
		if t.Batches[i].ID == batch {
			return &t.Batches[i]
		}
	}

	return nil
}

// NextClass returns the entry mass of the next class and the gain in PPM over the current one; ok is false if there is no better class.
func (b *Balance) NextClass(mass Micrograms) (entry Micrograms, gain PPM, ok bool) {
	current := b.ClassPPM(mass)

	for i := range b.Market.ClassCount {
		class := b.Market.Classes[i]
		if class.UpToMass <= mass || i+1 >= b.Market.ClassCount {
			continue
		}

		next := b.Market.Classes[i+1]
		if next.PPM <= current {
			return 0, 0, false
		}

		return class.UpToMass + 1,
			PPM(mulDivFloor(int64(next.PPM), int64(UnitPPM), int64(current))) - UnitPPM, true
	}

	return 0, 0, false
}

// Series samples prices per kilo in cents, every step ticks up to the current tick; nil if points or step are not positive.
func (s *State) Series(b *Balance, points int, step Tick) (fish, feed []Coins) {
	if points <= 0 || step <= 0 {
		return nil, nil
	}

	points = min(points, int(s.Tick/step)+1)
	fish = make([]Coins, 0, points)
	feed = make([]Coins, 0, points)

	for i := points - 1; i >= 0; i-- {
		at := s.Tick - Tick(i)*step
		market := MarketAt(b, at)
		fish = append(fish, market.FishKg)
		feed = append(feed, market.FeedKg)
	}

	return fish, feed
}

// StockAdvice suggests fingerlings that fit the tank and the cash, with the cost per fish in cents up to grow-out.
func (s *State) StockAdvice(b *Balance, tank TankID) (fish FishCount, perFish Coins) {
	t := s.tank(tank)
	if t == nil {
		return 0, 0
	}

	perFish = b.Economy.FingerlingPrice + feedToRaise(b, s.Tick)
	if perFish <= 0 {
		return 0, 0
	}

	room := t.Capacity(b) - int64(t.Fish())
	if room <= 0 || t.BatchCount >= MaxBatchesPerTank {
		return 0, perFish
	}

	return FishCount(min(room, int64(s.Cash)/int64(perFish))), perFish
}

func feedToRaise(b *Balance, at Tick) Coins {
	gain := int64(topClass(b)) - int64(b.Growth.FingerlingMass)
	if gain <= 0 {
		return 0
	}

	feed := mulDivCeil(gain, int64(b.Ration.TargetFCRPPM), int64(UnitPPM))

	return Coins(mulDivCeil(feed, int64(MarketAt(b, at).FeedKg), int64(MicrogramsPerKilogram)))
}

// LoanBlock reports whether taking credit now is worth it and, if not, what prevents it.
type LoanBlock uint8

// Reasons why taking credit is allowed or blocked.
const (
	LoanOpen LoanBlock = iota
	LoanNoCredit
	LoanNoRoom
	LoanNoNeed
	loanBlockCount
)

var loanBlockNames = [...]string{
	LoanOpen:     "open",
	LoanNoCredit: "no_credit",
	LoanNoRoom:   "no_room",
	LoanNoNeed:   "no_need",
}

var _ [len(loanBlockNames) - int(loanBlockCount)]struct{}

func (l LoanBlock) String() string {
	if l >= loanBlockCount {
		return invalidName
	}

	return loanBlockNames[l]
}

// LoanAdvice suggests how much to borrow in cents up to the break-even stocking, or 0 with the reason for the block.
func (s *State) LoanAdvice(b *Balance, tank TankID, breakEven FishCount) (Coins, LoanBlock) {
	room := Coins(subSat(int64(b.Credit.MaxPrincipal), int64(s.Debt)))
	if room <= 0 {
		return 0, LoanNoCredit
	}

	t := s.tank(tank)
	if t == nil {
		return room, LoanOpen
	}
	if t.BatchCount >= MaxBatchesPerTank || t.Capacity(b)-int64(t.Fish()) <= 0 {
		return 0, LoanNoRoom
	}

	fish, perFish := s.StockAdvice(b, tank)
	if perFish <= 0 {
		return room, LoanOpen
	}

	goal := int64(breakEven)
	if int64(t.Fish())+int64(fish) >= goal {
		goal = t.Capacity(b)
	}

	short := goal - int64(t.Fish()) - int64(fish)
	if short <= 0 {
		return 0, LoanNoNeed
	}

	return min(Coins(mulDivCeil(int64(perFish), short, 1)), room), LoanOpen
}
