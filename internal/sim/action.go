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
	ActionBorrow
	ActionRepay
	ActionTreat
	actionKindCount
)

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
	RejectNothingSick
	rejectReasonCount
)

var rejectReasonNames = [rejectReasonCount]string{
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
	RejectNothingSick:       "nothing_sick",
}

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
