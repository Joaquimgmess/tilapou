package sim

// GrossValue is what a mass of fish is worth at a price per kilo, in cents.
//
// A conta existia em cinco lugares e so a da tela divergia: ela truncava a biomassa em quilos
// inteiros, entao a coluna congelava num degrau de 1 kg e — depois que a despesca passou a
// decidir por valor — chegava a mostrar o lado errado do piso que diz se a fazenda ainda tem
// jogada. O oraculo e o caixa: e esta a conta que sell credita.
func GrossValue(pricePerKg Coins, mass Micrograms) Coins {
	return Coins(mulDivFloor(int64(pricePerKg), int64(mass), int64(MicrogramsPerKilogram)))
}

// TankPayout is what this tank would credit if everything in it were sold now: market value
// of the batches plus the contract bonus, when the tank has it.
//
// Quem tem o tanque na mao usa esta; quem so tem o lote usa GrossValue. A pergunta "a fazenda
// ainda tem saida" e desta: quem repovoa e o caixa que entra, e nao o preco de mercado.
func TankPayout(b *Balance, t *Tank, at Tick) Coins {
	var total int64
	for i := range t.BatchCount {
		batch := &t.Batches[i]
		total = addSat(total, int64(GrossValue(b.PriceFor(batch.MeanMass, at), batch.Biomass())))
	}

	return applyContract(b, t, Coins(total))
}

// applyContract e a formula do bonus num lugar so: sell e TankPayout tem de creditar e
// prometer o mesmo numero, senao a tela e a decisao voltam a divergir por outro caminho.
func applyContract(b *Balance, t *Tank, revenue Coins) Coins {
	if !t.Owns(AutoContract) {
		return revenue
	}

	return Coins(mulDivFloor(int64(revenue), int64(UnitPPM)+int64(b.Progression.ContractBonusPPM), int64(UnitPPM)))
}
