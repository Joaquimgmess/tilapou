package sim

const (
	// loanFeedFloorKg is the restock the loan has to cover, in kilos: it is what the shop
	// sells in one go.
	loanFeedFloorKg   = 100
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
	// A projecao precifica racao a mercado, e nao pela media contabil do silo: com a media,
	// o custo por dia projetado mudava com a ultima compra de racao e a decisao de segurar
	// ou vender virava resposta do nivel do silo.
	t.FeedUnitCost = MarketAt(b, s.Tick).FeedKg

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
func (s *State) StockAdvice(b *Balance, tank TankID, plan CyclePlan) (fish FishCount, perFish Coins) {
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

	// O que sobra depois de guardar o custo fixo do ciclo: povoar com o caixa inteiro deixa
	// o lote sem racao no meio do caminho, e ai ele morre de fome em vez de crescer.
	spendable := int64(s.Cash) - int64(s.fixedCost(b, t, plan))
	if spendable <= 0 {
		return 0, perFish
	}

	return FishCount(min(room, spendable/int64(perFish))), perFish
}

// fixedCost is the upkeep and the interest that run while the batch grows, whether or not
// the player does anything.
//
// O plano vem de fora: monta-lo custa duas simulacoes de ciclo, e so quem orquestra sabe
// quando vale pagar por isso e por quanto tempo um plano do dia anterior ainda serve.
func (s *State) fixedCost(b *Balance, t *Tank, plan CyclePlan) Coins {
	return fixedCostOn(b, t, plan, s.Debt)
}

// fixedCostOn recebe a divida em vez de le-la do estado, para perguntar quanto o ciclo
// custaria com a divida que um emprestimo deixaria.
func fixedCostOn(b *Balance, t *Tank, plan CyclePlan, debt Coins) Coins {
	if plan.Days <= 0 {
		return 0
	}

	daily := int64(b.Tanks[t.Kind].UpkeepPerDay) +
		mulDivCeil(int64(debt), int64(b.Credit.DailyRatePPM), int64(UnitPPM))

	return Coins(mulDivCeil(daily, plan.Days, 1))
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
	LoanNoCycle
	loanBlockCount
)

var loanBlockNames = [...]string{
	LoanOpen:     "open",
	LoanNoCredit: "no_credit",
	LoanNoRoom:   "no_room",
	LoanNoNeed:   "no_need",
	LoanNoCycle:  "no_cycle",
}

var _ [len(loanBlockNames) - int(loanBlockCount)]struct{}

func (l LoanBlock) String() string {
	if l >= loanBlockCount {
		return invalidName
	}

	return loanBlockNames[l]
}

// LoanOffer is what taking credit now would buy: the amount in cents, the fish it makes
// stockable and, when Cents is zero, what blocks it.
type LoanOffer struct {
	Cents Coins
	Fish  FishCount
	Block LoanBlock
}

// LoanAdvice suggests how much to borrow up to the break-even stocking, with the fish that
// money actually stocks — nao o emprestimo dividido pelo custo do peixe, que conta como peixe
// a parte dele que vai pagar o custo fixo do ciclo.
func (s *State) LoanAdvice(b *Balance, tank TankID, plan CyclePlan) LoanOffer {
	room := Coins(subSat(int64(b.Credit.MaxPrincipal), int64(s.Debt)))
	if room <= 0 {
		return LoanOffer{Block: LoanNoCredit}
	}

	t := s.tank(tank)
	if t == nil {
		return LoanOffer{Cents: room, Block: LoanOpen}
	}
	if t.BatchCount >= MaxBatchesPerTank || t.Capacity(b)-int64(t.Fish()) <= 0 {
		return LoanOffer{Block: LoanNoRoom}
	}

	fish, perFish := s.StockAdvice(b, tank, plan)
	if perFish <= 0 {
		return LoanOffer{Cents: room, Block: LoanOpen}
	}

	// O alvo nunca passa do que o tanque comporta: financiar o break-even num tanque menor
	// que ele e prometer peixe que o povoar recusa depois, com a divida ja tomada.
	goal := min(int64(plan.BreakEven), t.Capacity(b))
	if int64(t.Fish())+int64(fish) >= goal {
		goal = t.Capacity(b)
	}

	short := goal - int64(t.Fish()) - int64(fish)
	if short <= 0 {
		return LoanOffer{Block: LoanNoNeed}
	}

	// Nunca menos que um saco de racao: um emprestimo dimensionado pelos poucos peixes que
	// faltam nao paga o proximo gasto obrigatorio, e o jogador fica com a divida sem poder
	// alimentar o que tem.
	wanted := int64(lendable(b, t, plan, s.Debt, s.Cash,
		Coins(max(s.loanFor(b, t, plan, short, perFish), int64(s.feedSack(b))))))
	if wanted <= 0 {
		// Nem o piso do ciclo cabe no limite: aceitar so sobe o juro sem povoar nada, e e
		// esse o estado em que a fazenda precisa do resgate, nao de mais divida.
		return LoanOffer{Block: LoanNoCycle}
	}

	// Os peixes saem sempre da inversa do dimensionamento, e nunca de short: o arredondamento
	// para cima que dimensiona o emprestimo sobra um peixe, e prometer um a mais do que o
	// povoar entrega e o defeito que esta conta existe para nao ter.
	// O que a oferta promete e o ACRESCIMO: fishFor devolve o total que o caixa mais o
	// emprestimo povoam, e o jogador ja podia povoar fish deles sem divida nenhuma.
	buys := min(s.fishFor(b, t, plan, Coins(wanted), perFish)-int64(fish), short)
	if buys <= 0 {
		// Cabe no limite e mesmo assim nao povoa um peixe: aceitar so sobe o juro, e o juro
		// e o que ja esta engolindo o ciclo. E a mesma jogada pior que LoanNoCycle recusa.
		return LoanOffer{Block: LoanNoCycle}
	}

	return LoanOffer{Cents: Coins(wanted), Fish: FishCount(min(buys, maxInt32)), Block: LoanOpen}
}

// fishFor e a mesma conta que StockAdvice fara depois do emprestimo, com a divida que ele
// deixa: e assim que o numero prometido e o numero entregue nao podem divergir.
func (s *State) fishFor(b *Balance, t *Tank, plan CyclePlan, loan, perFish Coins) int64 {
	spendable := int64(s.Cash) + int64(loan) - int64(fixedCostOn(b, t, plan, s.Debt+loan))
	// O espaco livre limita junto com o dinheiro, como em StockAdvice: e a mesma conta que o
	// povoar fara depois, e e assim que o numero prometido e o entregue nao divergem.
	room := t.Capacity(b) - int64(t.Fish())

	return max(min(spendable/int64(perFish), room), 0)
}

// loanFor e quanto o jogador precisa pegar para de fato povoar short peixes. Nao basta o
// alevino: povoar so acontece com o custo fixo do ciclo guardado, e o proprio emprestimo
// aumenta esse custo, porque o juro da divida entra nele. Dai o emprestimo aparecer nos dois
// lados da conta e a divisao no fim.
func (s *State) loanFor(b *Balance, t *Tank, plan CyclePlan, short int64, perFish Coins) int64 {
	need := mulDivCeil(int64(perFish), short, 1) + int64(s.fixedCost(b, t, plan))

	// A fatia de cada centavo emprestado que volta como juro ao longo do ciclo.
	bite := mulDivCeil(int64(b.Credit.DailyRatePPM), plan.Days, 1)
	if bite >= int64(UnitPPM) {
		return need
	}

	return mulDivCeil(need, int64(UnitPPM), int64(UnitPPM)-bite)
}

// feedSack is the smallest restock that keeps a batch fed, in cents.
func (s *State) feedSack(b *Balance) Coins {
	return feedSackAt(b, s.Tick)
}

func feedSackAt(b *Balance, at Tick) Coins {
	return Coins(mulDivCeil(int64(MarketAt(b, at).FeedKg), loanFeedFloorKg, 1))
}
