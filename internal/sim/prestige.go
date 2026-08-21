package sim

const prestigeScale = 10

// RaisingCost estimates in cents the fingerlings plus the feed implied by the target FCR, at the tick's prices.
func RaisingCost(b *Balance, fish FishCount, mass Micrograms, at Tick) Coins {
	fingerlings := int64(b.Economy.FingerlingPrice) * int64(fish)
	gained := subSat(int64(mass), int64(b.Growth.FingerlingMass)) * int64(fish)
	feed := mulDivFloor(gained, int64(b.Ration.TargetFCRPPM), int64(UnitPPM))

	return Coins(addSat(fingerlings, mulDivFloor(feed, int64(MarketAt(b, at).FeedKg), int64(MicrogramsPerKilogram))))
}

// PrestigePointsFor returns 0 if the divisor is not positive.
func PrestigePointsFor(lifetime Coins, divisor int64) uint32 {
	if divisor <= 0 || lifetime <= 0 {
		return 0
	}

	return uint32(prestigeScale * isqrt(int64(lifetime)/divisor))
}

func (s *State) prestigeBonus(b *Balance) PPM {
	return PPM(addSat(int64(UnitPPM), int64(s.Prestige)*int64(b.Progression.PrestigeBonusPPM)))
}

// Broke reports whether the farm can no longer start a cycle, which is what frees the
// restart key. Prestige left to claim does not count: it comes from LifetimeEarned, which
// never goes down.
//
// A conta e a mesma de stuck, e nao a de um alevino: perguntar se cabe um peixe no credito
// dizia sim com o juro da divida engolindo o ciclo inteiro, e a tela indicava uma tecla que
// ela mesma negava. O cronometro do resgate e a tecla respondem a mesma pergunta.
func (s *State) Broke(b *Balance, plans Plans) bool {
	return s.stuck(b, plans)
}

// stuck reports whether nothing can improve the farm any more: no cash for the smallest
// cycle, no fish worth harvesting, and no feed left to grow the ones that are small. Feed
// in a tank is a way out on its own — the batch keeps growing without the player.
//
// O criterio e o ciclo, e nao o saco de racao: caixa alto de emprestimo com tanque vazio
// nunca contava como preso, e o jogador ficava semanas de tela olhando o juro comer o caixa
// sem que nenhuma tecla fizesse nada.
func (s *State) stuck(b *Balance, plans Plans) bool {
	for i := range s.TankCount {
		t := &s.Tanks[i]
		// A pergunta e "existe jogada possivel", e nao "existe ciclo viavel": [b] e
		// irreversivel, entao falso positivo custa a fazenda e falso negativo custa uma tela
		// feia. O piso e a acao mais barata que muda o estado, e cada acao responde por si no
		// registro de playable.
		if s.anyPlay(b, t, plans[t.Kind]) {
			return false
		}
		// Racao so e saida enquanto houver lote comendo: com o tanque vazio ela e caixa que
		// virou estoque parado, e nao um ciclo que anda sozinho. O lote no ponto de abate ja
		// e coberto pela entrada de ActionHarvest no registro.
		if t.FeedStock > 0 && t.Fish() > 0 {
			return false
		}
	}

	return true
}

func restart(s *State, b *Balance, at Tick, sink *eventSink, plans Plans) RejectReason {
	if !s.Broke(b, plans) {
		return RejectNotBroke
	}
	// Tilapar reconstroi a mesma fazenda e ainda devolve os pontos que a partida rendeu:
	// recomecar aqui e a porta pior, e ela nao tem volta. O jogo recusa em vez de deixar o
	// jogador trocar a jogada certa pela que ele apertou por habito.
	if PrestigePointsFor(s.LifetimeEarned, b.Progression.PrestigeDivisor) > s.Prestige {
		return RejectPrestigeFirst
	}

	rebuild(s, b, at, s.Prestige)

	sink.emit(Event{Kind: EventRestarted, From: at, To: at})

	return RejectNone
}

func prestige(s *State, b *Balance, at Tick, sink *eventSink) RejectReason {
	earned := PrestigePointsFor(s.LifetimeEarned, b.Progression.PrestigeDivisor)
	if earned <= s.Prestige {
		return RejectNotEnoughLifetime
	}

	rebuild(s, b, at, earned)

	sink.emit(Event{Kind: EventPrestiged, From: at, To: at})

	return RejectNone
}

// rebuild also writes off the debt: the restart cash only survives a day of play while the
// debt is under RestartCash divided by the daily rate, which any compounded debt passes.
// bankrupt winds the farm up when the debt passes the point of no return, and reports
// whether it fired. It hands back the same restart package as [b], so nobody is better off
// starving the farm on purpose than going broke.
func bankrupt(s *State, b *Balance, at Tick, sink *eventSink, plans Plans) bool {
	if s.stuck(b, plans) {
		s.StuckTicks++
	} else {
		s.StuckTicks = 0
	}

	overDebt := b.Credit.BankruptcyPrincipal > 0 && s.Debt >= b.Credit.BankruptcyPrincipal
	stuckTooLong := b.Progression.StuckDaysToBankruptcy > 0 &&
		s.StuckTicks >= Tick(b.Progression.StuckDaysToBankruptcy)*TicksPerDay

	if !overDebt && !stuckTooLong {
		return false
	}

	forgiven := s.Debt
	rebuild(s, b, at, s.Prestige)
	s.StuckTicks = 0

	sink.emit(Event{Kind: EventBankrupt, From: at, To: at, Cash: forgiven})

	return true
}

func rebuild(s *State, b *Balance, at Tick, points uint32) {
	kept := State{
		Version:        s.Version,
		BalanceVersion: s.BalanceVersion,
		RngVersion:     s.RngVersion,
		Seed:           s.Seed,
		Zone:           s.Zone,
		Tick:           s.Tick,
		Cash:           b.Progression.RestartCash,
		LifetimeEarned: s.LifetimeEarned,
		Prestige:       points,
		NextTankID:     1,
		NextBatchID:    1,
		EventSeq:       s.EventSeq,
		Debt:           0,
		DebtCarry:      0,
	}

	tank, ok := kept.AddTank(b, TankEarthPond, b.Tanks[TankEarthPond].Litres)
	if ok {
		target := kept.tank(tank)
		target.addBatch(kept.NextBatchID, b.Progression.RestartFish, b.Growth.FingerlingMass, at)
		target.Batches[0].Cost = RaisingCost(b, b.Progression.RestartFish, b.Growth.FingerlingMass, at)
		kept.NextBatchID++
		loadFeed(target, b.Progression.RestartFeed,
			Coins(mulDivFloor(int64(b.Progression.RestartFeed), int64(MarketAt(b, at).FeedKg), int64(MicrogramsPerKilogram))))
	}

	*s = kept
}
