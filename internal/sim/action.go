package sim

const invalidName = "invalid"

// ActionID casa uma acao com seu Outcome de mesmo ID.
type ActionID uint64

// ActionKind e o tipo de acao pedida pelo jogador.
type ActionKind uint8

// Acoes aceitas; ActionUnknown e o zero e sempre rejeitado.
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

// ActionKindNamed devolve ActionUnknown e false para nome desconhecido ou "unknown".
func ActionKindNamed(name string) (ActionKind, bool) {
	for kind, known := range actionKindNames {
		if known == name && ActionKind(kind) != ActionUnknown {
			return ActionKind(kind), true
		}
	}

	return ActionUnknown, false
}

// ActionKindNames devolve uma copia dos nomes validos, sem ActionUnknown.
func ActionKindNames() []string {
	return append([]string(nil), actionKindNames[ActionUnknown+1:]...)
}

// String devolve "invalid" fora do enum.
func (k ActionKind) String() string {
	if k >= actionKindCount {
		return invalidName
	}

	return actionKindNames[k]
}

// RejectReason explica a rejeicao; RejectNone significa aceita.
type RejectReason uint8

// Motivos de rejeicao de uma acao.
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

// RejectReasonNamed devolve RejectNone e false para nome desconhecido.
func RejectReasonNamed(name string) (RejectReason, bool) {
	for reason, known := range rejectReasonNames {
		if known == name {
			return RejectReason(reason), true
		}
	}

	return RejectNone, false
}

var _ [len(rejectReasonNames) - int(rejectReasonCount)]struct{}

// String devolve "invalid" fora do enum.
func (r RejectReason) String() string {
	if r >= rejectReasonCount {
		return invalidName
	}

	return rejectReasonNames[r]
}

// Action e um pedido agendado para o tick At; os demais campos so valem conforme
// Kind, e Amount vem em peixes, microgramas ou centavos.
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

// Outcome responde a uma Action por ID; se Applied for falso, Reason da o motivo e
// Needed traz em centavos o que faltou de caixa.
type Outcome struct {
	ID      ActionID
	At      Tick
	Applied bool
	Reason  RejectReason
	Needed  Coins
}
