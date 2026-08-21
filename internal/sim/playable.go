package sim

// playableFor responde se esta acao, sozinha, ainda muda o rumo da fazenda neste estado.
//
// O contrato tem DUAS partes, e o comentario antigo so dizia a primeira: (1) a entrada nunca
// e mais permissiva que o applyX correspondente — prometer jogada que o jogo recusa deixa o
// jogador preso sem resgate; e (2) ela pode ser mais rigorosa quando a acao e aceita mas nao
// muda o rumo, e ai a excecao e declarada na tabela do teste, nao descoberta depois. Comprar
// racao sem lote e despescar lote verde sao as duas de hoje.
type playableFor func(s *State, b *Balance, t *Tank, plan CyclePlan) bool

// playable e o registro exaustivo por ActionKind. Ele existe para que uma acao nova quebre a
// compilacao em vez de reabrir, pela quinta vez, a divergencia entre "o que a tela diz" e "o
// que o jogo aceita": stuck media se cabia um ciclo inteiro enquanto a acao media se cabiam
// 100 alevinos, e a fazenda com caixa no bolso aparecia quebrada.
var playable = [...]playableFor{
	// Acao desconhecida nao e jogada.
	ActionUnknown: never,

	ActionBuyTank: func(s *State, b *Balance, _ *Tank, _ CyclePlan) bool {
		return s.Cash >= s.NextTankCost(b, TankEarthPond)
	},
	ActionStock: func(s *State, b *Balance, t *Tank, _ CyclePlan) bool {
		if t.BatchCount >= MaxBatchesPerTank || t.Capacity(b)-int64(t.Fish()) < MinStockFish {
			return false
		}

		return int64(s.Cash) >= mulDivCeil(int64(b.Economy.FingerlingPrice), MinStockFish, 1)
	},
	// O piso e o que applyBuyFeed aceita — um quilo —, e nao o saco de 100 kg do conselho de
	// credito: com o saco aqui, a fazenda com caixa para 1 kg contava como quebrada e comprar
	// 3,27 TC de racao a trazia de volta, com a tela mandando recomecar no meio.
	ActionBuyFeed: func(s *State, b *Balance, t *Tank, plan CyclePlan) bool {
		// Racao so muda o rumo com lote para comer: no tanque vazio ela e caixa que vira
		// estoque parado, a mesma regra que o ramo do FeedStock ja aplicava.
		return t.Fish() > 0 && s.Cash >= MarketAt(b, plan.At).FeedKg
	},
	ActionFeed: func(_ *State, _ *Balance, t *Tank, _ CyclePlan) bool {
		return t.FeedStock > 0 && t.Fish() > 0
	},
	// Ligar o aerador gasta energia e nao levanta caixa: muda a agua, nao o rumo.
	ActionAerate: never,
	// O criterio e VALOR, e nao massa: applyHarvest esvazia o lote inteiro, entao a jogada
	// que a despesca destrava e sempre povoar, e o piso dela e o mesmo da jogada mais barata.
	// Medindo por massa, a fazenda dizia "nao resta jogada possivel" com 7553,00 TC de peixe
	// dentro; medindo por valor, tres alevinos de 30 g continuam nao sendo saida.
	ActionHarvest: func(s *State, b *Balance, t *Tank, _ CyclePlan) bool {
		worth := int64(s.Cash) + int64(tankWorth(b, t, s.Tick))

		return worth >= mulDivCeil(int64(b.Economy.FingerlingPrice), MinStockFish, 1)
	},
	ActionBuyUpgrade: func(s *State, b *Balance, t *Tank, _ CyclePlan) bool {
		for kind := range autoKindCount {
			if !t.Owns(kind) && s.Cash >= b.Automation[kind].Cost {
				return true
			}
		}

		return false
	},
	// Tilapar reinicia a fazenda: e resgate, como o [b], e nao jogada dentro da partida.
	// Prestigio pendente vem de LifetimeEarned, que nunca desce — contar isso como saida
	// diria que a fazenda tem liquidez que ela nao tem.
	ActionPrestige: never,
	// Recomecar e o proprio resgate: conta-lo como jogada faria a fazenda nunca quebrar.
	ActionRestart: never,
	ActionBorrow: func(s *State, b *Balance, t *Tank, plan CyclePlan) bool {
		return lendable(b, t, plan, s.Debt, s.Cash) > 0
	},
	// Pagar divida nao levanta caixa nem lote: com caixa e divida ela seria sempre possivel, e
	// a fazenda nunca quebraria.
	ActionRepay: never,
	// Tratar cobra o custo do tratamento: pedir so o lote doente marcava como jogada uma acao
	// que o jogo recusa por caixa, e ai a fazenda com lote doente e caixa zero nao contava
	// como quebrada, nenhuma tecla funcionava e o cronometro do resgate nem corria.
	ActionTreat: func(s *State, b *Balance, t *Tank, _ CyclePlan) bool {
		if s.Cash < b.Shock.TreatmentCost {
			return false
		}

		for i := range t.BatchCount {
			if t.Batches[i].Sick > 0 {
				return true
			}
		}

		return false
	},
}

// A sentinela quebra a compilacao quando uma acao nova nasce depois da ultima do registro,
// no mesmo molde das tabelas de nomes. Ela nao pega buraco no meio — para isso existe o teste
// que percorre o registro e recusa entrada vazia.
var _ [len(playable) - int(actionKindCount)]struct{}

// tankWorth e o que este tanque creditaria se fosse despescado agora — com o bonus de
// contrato, porque quem repovoa e o caixa que entra. Ignorar o bonus so errava para baixo, e
// para baixo e a direcao cara: marca fazenda viva como quebrada.
func tankWorth(b *Balance, t *Tank, at Tick) Coins {
	return TankPayout(b, t, at)
}

// harvestWorth e o que a fazenda inteira levantaria despescando agora. Existe para o teste
// nomear o numero que a decisao usa.
func (s *State) harvestWorth(b *Balance) Coins {
	var total int64
	for i := range s.TankCount {
		total = addSat(total, int64(tankWorth(b, &s.Tanks[i], s.Tick)))
	}

	return Coins(total)
}

func never(_ *State, _ *Balance, _ *Tank, _ CyclePlan) bool {
	return false
}

// anyPlay percorre o registro para o tanque: basta uma jogada possivel para a fazenda nao
// estar quebrada.
func (s *State) anyPlay(b *Balance, t *Tank, plan CyclePlan) bool {
	for _, can := range playable {
		if can(s, b, t, plan) {
			return true
		}
	}

	return false
}
