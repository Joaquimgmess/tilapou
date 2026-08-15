# tilapou

Idle game de piscicultura no terminal. Um daemon simula a fazenda em segundo plano; uma TUI com
cara de Game Boy se conecta nele. A física é de tilápia de verdade — crescimento por TGC, oxigênio
que despenca de madrugada, conversão alimentar emergente — calibrada contra dados de campo.

```sh
make up                      # postgres + daemon em :8080
go run ./cmd/tilapou play
```

## Os quatro modos

```
tilapou serve     daemon: API, simulação, persistência
tilapou play      TUI: mapa navegável, painéis e ações
tilapou status    uma linha para tmux ou prompt
tilapou sim       simulador headless de balanceamento
```

O daemon é a fonte da verdade. A TUI não calcula nada — nem para prévia. É o que mantém dois
clientes coerentes e o que torna trapaça impossível por construção.

## Layout

```
cmd/tilapou            subcomandos
internal/sim           NÚCLEO PURO: física, ações, eventos, determinismo
  scenario             cenários golden — o SSOT de teste
  save                 codec do estado
internal/balance       balance.toml calibrado, quantizado para inteiro
internal/farm          slice: porta Store, catch-up, ações idempotentes, HTTP
internal/client        cliente tipado do daemon
internal/tui           TUI (bubbletea v2)
  gb                   canvas de pixels, paleta de 4 tons, sprites
internal/platform      shared kernel: config, httpx, logging, postgres
internal/migrations    goose embutido
internal/arch          testes de fronteira arquitetural
```

## As regras que o repo executa, não promete

- **`internal/sim` é puro.** Não importa nada do projeto, nem `time`, `context`, `os`, `net/http`
  ou `math/rand`. Cobrado pelo `depguard` e por um teste que varre `go list -deps`.
- **Determinismo é testado, não assumido.** `Advance(0→N)` tem que dar exatamente o mesmo estado e
  os mesmos eventos que qualquer particionamento do intervalo. Fuzz nativo, centenas de milhares
  de execuções.
- **Zero float no estado persistido.** Massa em micrograma, dinheiro em centavo, temperatura em
  milésimo de grau. O teste de determinismo compara por igualdade exata.
- **A TUI não alcança a simulação nem o banco.** Também via `depguard`: quebra o build.
- **Ganho offline não é caso especial.** A simulação é preguiçosa; voltar depois de dois dias usa
  o mesmo código de quando o jogo está aberto.
- **Zero comentários em código.** Nome e tipo explicam; o lint cobra o resto.

## A física, e de onde vieram os números

Calibrada contra Embrapa, Peixe BR, CEPEA e literatura zootécnica, e travada por teste:

| Medida | Campo | Modelo |
| --- | --- | --- |
| 140 dias a 22 °C | 49 g | 44 g |
| 140 dias a 30 °C | 398 g | 389 g |
| GPD a 28 °C com 400 g | 4–5 g/dia | 4,9 g/dia |
| Ciclo 30 g → 800 g | 119–199 dias | 199 dias a 28 °C |

O oxigênio segue o ciclo diurno real: as algas produzem de dia e consomem à noite, então a queda é
de madrugada. Densidade alta derruba a curva — 12.000 peixes num viveiro de 1.000 m³ perdem 65% do
lote numa noite, e o aerador salva 4.000 deles. Nada disso é regra escrita: cai da física.

`internal/balance/balance.toml` é o arquivo de balanceamento. Mexer nele não exige recompilar
lógica, e o `tilapou sim` mostra o efeito em segundos.

## Progressão

Comedouro, aerador automático, peão, técnico e contrato — cada automação remove um clique e libera
atenção para a camada seguinte, que é a mesma progressão tecnológica da aquicultura real. Custo em
escada `base × 1.15ⁿ`. Quando o vitalício justificar, dá para **tilapar**: vender tudo, recomeçar e
ficar com matrizes genéticas que multiplicam o crescimento para sempre.

## Cenários golden

```sh
go run ./cmd/tilapou sim -list
go run ./cmd/tilapou sim -run densidade-contra-oxigenio
make golden                  # regenera os arquivos após mudar o balanceamento
```

Cada cenário é código Go que produz uma tabela diffável. Mudou a física ou os números? O diff do
golden mostra exatamente o que mudou, para você aprovar de propósito.

## Migrations

SQL em `internal/migrations/sql`, embutido no binário e aplicado pelo goose como biblioteca — o
deploy leva um binário só. Mudança de schema sai como expand → backfill → contract, nunca um
`ALTER` destrutivo num deploy só.

## Checks

```sh
make check                   # fmt, lint, test -race, build, govulncheck
```

O `golangci-lint` roda com `default: all` — cerca de 100 linters, incluindo as fronteiras de
arquitetura.

## Stack

chi · huma (OpenAPI e RFC 7807) · pgx · goose · bubbletea v2 · lipgloss v2 · slog

## Licença

MIT — veja [LICENSE](LICENSE).
