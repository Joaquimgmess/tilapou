package sim

const invalidName = "invalid"

// ActionID matches an action to the Outcome with the same ID.
type ActionID uint64

// ActionKind is the kind of action requested by the player.
type ActionKind uint8

// Accepted actions; ActionUnknown is the zero value and is always rejected.
const (
	ActionUnknown ActionKind = iota
	ActionBuyTank
	ActionStock
	ActionBuyFeed
	ActionFeed
	ActionAerate
	ActionHarvest
	ActionBuyUpgrade
	ActionPrestige
	ActionRestart
	ActionBorrow
	ActionRepay
	ActionTreat
	actionKindCount
)

var actionKindNames = [...]string{
	ActionUnknown:    "unknown",
	ActionBuyTank:    "buy_tank",
	ActionStock:      "stock",
	ActionBuyFeed:    "buy_feed",
	ActionFeed:       "feed",
	ActionAerate:     "aerate",
	ActionHarvest:    "harvest",
	ActionBuyUpgrade: "buy_upgrade",
	ActionPrestige:   "prestige",
	ActionRestart:    "restart",
	ActionBorrow:     "borrow",
	ActionRepay:      "repay",
	ActionTreat:      "treat",
}

var _ [len(actionKindNames) - int(actionKindCount)]struct{}

// ActionKindNamed returns ActionUnknown and false for an unknown name or "unknown".
func ActionKindNamed(name string) (ActionKind, bool) {
	for kind, known := range actionKindNames {
		if known == name && ActionKind(kind) != ActionUnknown {
			return ActionKind(kind), true
		}
	}

	return ActionUnknown, false
}

// ActionKindNames returns a copy of the valid names, without ActionUnknown.
func ActionKindNames() []string {
	return append([]string(nil), actionKindNames[ActionUnknown+1:]...)
}

// String returns "invalid" outside the enum.
func (k ActionKind) String() string {
	if k >= actionKindCount {
		return invalidName
	}

	return actionKindNames[k]
}

// RejectReason explains the rejection; RejectNone means accepted.
type RejectReason uint8

// Reasons an action is rejected.
const (
	RejectNone RejectReason = iota
	RejectUnknownKind
	RejectNoSuchTank
	RejectNoSuchBatch
	RejectNotEnoughCash
	RejectNotEnoughFeed
	RejectTankFull
	RejectFarmFull
	RejectBadAmount
	RejectTooDense
	RejectAlreadyOwned
	RejectNotEnoughLifetime
	RejectCreditLimit
	RejectNoDebt
	RejectNotBroke
	RejectNothingSick
	RejectStaleView
	RejectPrestigeFirst
	RejectReasonCount
)

var rejectReasonNames = [...]string{
	RejectNone:              "none",
	RejectUnknownKind:       "unknown_kind",
	RejectNoSuchTank:        "no_such_tank",
	RejectNoSuchBatch:       "no_such_batch",
	RejectNotEnoughCash:     "not_enough_cash",
	RejectNotEnoughFeed:     "not_enough_feed",
	RejectTankFull:          "tank_full",
	RejectFarmFull:          "farm_full",
	RejectBadAmount:         "bad_amount",
	RejectTooDense:          "too_dense",
	RejectAlreadyOwned:      "already_owned",
	RejectNotEnoughLifetime: "not_enough_lifetime",
	RejectCreditLimit:       "credit_limit",
	RejectNoDebt:            "no_debt",
	RejectNotBroke:          "not_broke",
	RejectNothingSick:       "nothing_sick",
	RejectStaleView:         "stale_view",
	RejectPrestigeFirst:     "prestige_first",
}

// RejectReasonNamed returns RejectNone and false for an unknown name.
func RejectReasonNamed(name string) (RejectReason, bool) {
	for reason, known := range rejectReasonNames {
		if known == name {
			return RejectReason(reason), true
		}
	}

	return RejectNone, false
}

var _ [len(rejectReasonNames) - int(RejectReasonCount)]struct{}

// String returns "invalid" outside the enum.
func (r RejectReason) String() string {
	if r >= RejectReasonCount {
		return invalidName
	}

	return rejectReasonNames[r]
}

// Action is a request scheduled for tick At; the remaining fields are only valid
// according to Kind, and Amount comes in fish, micrograms or cents.
type Action struct {
	ID   ActionID
	Kind ActionKind
	At   Tick
	// SeenAt e o tick que o jogador tinha na tela quando decidiu. Zero quer dizer que o
	// cliente nao informou, e ai a idade nao e cobrada.
	SeenAt   Tick
	Tank     TankID
	Batch    BatchID
	TankKind TankKind
	Auto     AutoKind
	Amount   int64
}

// Outcome answers an Action by ID; if Applied is false, Reason gives the cause and
// Needed carries in cents the cash that was missing.
type Outcome struct {
	ID      ActionID
	At      Tick
	Applied bool
	Reason  RejectReason
	Needed  Coins
}
