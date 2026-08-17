package sim

const invalidName = "invalid"

type ActionID uint64

type ActionKind uint8

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

func ActionKindNamed(name string) (ActionKind, bool) {
	for kind, known := range actionKindNames {
		if known == name && ActionKind(kind) != ActionUnknown {
			return ActionKind(kind), true
		}
	}

	return ActionUnknown, false
}

func ActionKindNames() []string {
	return append([]string(nil), actionKindNames[ActionUnknown+1:]...)
}

func (k ActionKind) String() string {
	if k >= actionKindCount {
		return invalidName
	}

	return actionKindNames[k]
}

type RejectReason uint8

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
	rejectReasonCount
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
}

func RejectReasonNamed(name string) (RejectReason, bool) {
	for reason, known := range rejectReasonNames {
		if known == name {
			return RejectReason(reason), true
		}
	}

	return RejectNone, false
}

var _ [len(rejectReasonNames) - int(rejectReasonCount)]struct{}

func (r RejectReason) String() string {
	if r >= rejectReasonCount {
		return invalidName
	}

	return rejectReasonNames[r]
}

type Action struct {
	ID       ActionID
	Kind     ActionKind
	At       Tick
	Tank     TankID
	Batch    BatchID
	TankKind TankKind
	Auto     AutoKind
	Amount   int64
}

type Outcome struct {
	ID      ActionID
	At      Tick
	Applied bool
	Reason  RejectReason
	Needed  Coins
}
