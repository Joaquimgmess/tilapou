// Package api carries the JSON contract of the tilapou daemon: one definition of each type,
// shared by the handler that writes it and the client that reads it.
//
// Os dois lados eram structs espelhadas casadas so pela tag JSON, e campo acrescentado de um
// lado compilava, passava nos testes e nunca chegava a tela. Com um tipo so, o compilador
// cobra os dois.
//
// Unidades vivem no nome do campo e na tag: Cents para dinheiro, Grams para peso, PPM para
// razoes, Ticks para tempo.
package api

// Batch is one batch inside a tank: money in cents, weight in grams, ratios in PPM.
// A tank holds up to MaxBatchesPerTank of them, and every one is a decision of its own.
type Batch struct {
	ID             uint32   `json:"batch_id"`
	Fish           int32    `json:"fish"`
	MeanGrams      int64    `json:"mean_grams"`
	Ready          bool     `json:"ready_to_harvest"`
	Sick           bool     `json:"sick"`
	PriceKgCents   int64    `json:"price_kg_cents"`
	ValueCents     int64    `json:"value_cents"`
	CostCents      int64    `json:"cost_cents"`
	MarginCents    int64    `json:"margin_cents"`
	CostPerKg      int64    `json:"cost_per_kg_cents"`
	ClassPPM       int64    `json:"class_ppm"`
	NextClassGrams int64    `json:"next_class_grams"`
	NextClassGain  int64    `json:"next_class_gain_ppm"`
	Decision       Decision `json:"decision"`
}

// Tank is the tank in the API: money in cents, weight in grams, oxygen in
// micrograms per litre, density in thousandths of kg/m3 and ratios in PPM. What belongs to
// a batch lives in Batches, so a tank with four of them does not hide three.
type Tank struct {
	ID           uint32 `json:"id"`
	Kind         string `json:"kind"`
	Fish         int32  `json:"fish"`
	FeedKg       int64  `json:"feed_kg"`
	OxygenUgL    int32  `json:"oxygen_ugl"`
	Aerating     bool   `json:"aerating"`
	DensityMilli int64  `json:"density_milli_kg_m3"`

	Batches     []Batch    `json:"batches"`
	Capacity    int64      `json:"capacity_fish"`
	StockAdvice int64      `json:"stock_advice_fish"`
	BatchCount  int32      `json:"batch_count"`
	MaxBatches  int32      `json:"max_batches"`
	BreakEven   int64      `json:"break_even_fish"`
	LoanAdvice  int64      `json:"loan_advice_cents"`
	LoanFish    int64      `json:"loan_advice_fish"`
	LoanOwed    int64      `json:"loan_owed_cents"`
	CycleDays   int64      `json:"cycle_days"`
	CycleMargin int64      `json:"cycle_margin_cents"`
	LoanBlock   LoanBlock  `json:"loan_block"`
	StockBlock  StockBlock `json:"stock_block"`
	StockShort  int64      `json:"stock_short_cents"`
	ServedFor   int64      `json:"served_for_ticks"`
	Upgrades    []Upgrade  `json:"upgrades"`
}

// LoanBlock e StockBlock atravessam o contrato tipados, e nao como string nua: e o tipo que
// faz o switch da tela ser cobrado por exaustividade quando um motivo novo nascer no sim.
type LoanBlock string

// Motivos pelos quais o credito esta aberto ou bloqueado.
const (
	LoanOpen     LoanBlock = "open"
	LoanNoCredit LoanBlock = "no_credit"
	LoanNoRoom   LoanBlock = "no_room"
	LoanNoNeed   LoanBlock = "no_need"
	LoanNoCycle  LoanBlock = "no_cycle"
)

// StockBlock e o motivo pelo qual povoar esta ou nao na mesa.
type StockBlock string

// Motivos pelos quais povoar esta aberto ou bloqueado.
const (
	StockOpen    StockBlock = "open"
	StockNoTank  StockBlock = "no_tank"
	StockNoRoom  StockBlock = "no_room"
	StockNoBatch StockBlock = "no_batch"
	StockNoCash  StockBlock = "no_cash"
	StockNoCycle StockBlock = "no_cycle"
)

// Event is the event in the API, with mass in grams and cash in cents.
type Event struct {
	Seq       uint64 `json:"seq"`
	Kind      string `json:"kind"`
	From      int64  `json:"from_tick"`
	To        int64  `json:"to_tick"`
	Tank      uint32 `json:"tank_id"`
	Fish      int32  `json:"fish"`
	MassGrams int64  `json:"mass_grams"`
	CashCents int64  `json:"cash_cents"`
	Reason    string `json:"reason"`
}

// Upgrade carries the automation cost in cents.
type Upgrade struct {
	Kind      string `json:"kind"`
	Owned     bool   `json:"owned"`
	CostCents int64  `json:"cost_cents"`
}

// Outcome carries the reason for the refusal and the cash that was missing, in cents.
type Outcome struct {
	Applied    bool   `json:"applied"`
	Reason     string `json:"reason"`
	NeededCash int64  `json:"needed_cents"`
}

// Prices carries the current tick's prices in cents and the feed-to-fish exchange in PPM.
type Prices struct {
	FeedKgCents     int64 `json:"feed_kg_cents"`
	FingerlingCents int64 `json:"fingerling_cents"`
	FishKgCents     int64 `json:"fish_kg_cents"`
	RatioPPM        int64 `json:"equivalence_ppm"`
	ViablePPM       int64 `json:"equivalence_viable_ppm"`
}

// Cycle closes the last cycle: mass in grams, values in cents and feed
// conversion in PPM.
type Cycle struct {
	Fish         int32 `json:"fish"`
	MassGrams    int64 `json:"mass_grams"`
	RevenueCents int64 `json:"revenue_cents"`
	CostCents    int64 `json:"cost_cents"`
	MarginCents  int64 `json:"margin_cents"`
	CostPerKg    int64 `json:"cost_per_kg_cents"`
	PricePerKg   int64 `json:"price_per_kg_cents"`
	FCRPPM       int64 `json:"fcr_ppm"`
}

// Decision compares selling now with holding to the next weight class:
// cents, daily gain in milligrams, daily feed in grams.
type Decision struct {
	SellNowCents  int64 `json:"sell_now_cents"`
	SellNowMargin int64 `json:"sell_now_margin_cents"`
	HoldToGrams   int64 `json:"hold_to_grams"`
	HoldDays      int64 `json:"hold_days"`
	HoldCents     int64 `json:"hold_cents"`
	HoldMargin    int64 `json:"hold_margin_cents"`
	HoldCostCents int64 `json:"hold_cost_cents"`
	// HoldReached diz que a projecao alcanca o alvo dentro do teto de dias, e nao que o
	// peixe ja chegou la: com 30 g e alvo de 400 g ela e verdadeira e correta.
	HoldReached    bool  `json:"hold_reached"`
	BreakEvenPerKg int64 `json:"break_even_per_kg_cents"`
	GainPerDayMg   int64 `json:"gain_per_day_mg"`
	FeedPerDayG    int64 `json:"feed_per_day_grams"`
	CostPerDay     int64 `json:"cost_per_day_cents"`
	DaysOfFeed     int64 `json:"days_of_feed"`
	CycleDays      int64 `json:"cycle_days"`
}

// Series carries prices in cents per kilo, one point every StepTicks.
type Series struct {
	FishKgCents []int64 `json:"fish_kg_cents"`
	FeedKgCents []int64 `json:"feed_kg_cents"`
	StepTicks   int64   `json:"step_ticks"`
}

// Snapshot is the farm in the API: cents, grams and thousandths of a degree, with
// RunwayDays at -1 when there is no daily cost.
type Snapshot struct {
	FarmID        string   `json:"farm_id"`
	Name          string   `json:"name"`
	Tick          int64    `json:"tick"`
	Hour          int32    `json:"hour"`
	TempMilliC    int32    `json:"temp_milli_c"`
	CashCents     int64    `json:"cash_cents"`
	LifetimeCents int64    `json:"lifetime_cents"`
	BiomassGrams  int64    `json:"biomass_grams"`
	Fish          int32    `json:"fish"`
	Prestige      uint32   `json:"prestige"`
	Tanks         []Tank   `json:"tanks"`
	PrestigeNow   uint32   `json:"prestige_available"`
	NextTankCents int64    `json:"next_tank_cents"`
	Prices        Prices   `json:"prices"`
	Debt          int64    `json:"debt_cents"`
	LastCycle     Cycle    `json:"last_cycle"`
	Series        Series   `json:"series"`
	InterestDay   int64    `json:"interest_per_day_cents"`
	RunwayDays    int64    `json:"runway_days"`
	Broke         bool     `json:"broke"`
	Events        []Event  `json:"events"`
	LastOutcome   *Outcome `json:"last_outcome,omitempty"`
}
