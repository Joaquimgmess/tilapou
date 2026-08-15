# TILAPIA

```
                    ______
                 _-´      `--__
      /|      _-´   ·           `--__
     / |   _-´                        `--_
    /  | -´        ______                  `-_
   /   |         -´      `-_                  \
  <    |        ( ●  )       `-_               >
   \   |         `-____´         `-_          /
    \  | -_                     __-´       _-´
     \ |    `--__          __--´      __--´
      \|         `--______-´    `-____-´
```

Idle game de piscicultura no terminal. Um daemon simula a fazenda em segundo plano e uma TUI com
cara de Game Boy conecta nele. A física é de tilápia de verdade — crescimento por TGC, oxigênio
que despenca de madrugada, conversão alimentar emergente — e o jogo é sobre a economia: preço
oscilando, classe de peso, margem por ciclo, crédito e doença.

## Como rodar

```sh
make up     # infra: postgres + daemon
make play   # a TUI
```
