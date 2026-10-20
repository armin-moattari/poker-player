# poker-player

A Texas Hold'em hand evaluator and equity calculator written in Go, compiled to WebAssembly and wrapped in a mobile-friendly web app.

Work in progress. See [CLAUDE.md](CLAUDE.md) for the plan and milestones.

## Commands

```bash
go test ./...                                   # run tests
go test -bench=. ./internal/eval                # benchmarks
GOOS=js GOARCH=wasm go build -o web/main.wasm ./wasm   # WASM build
```

## Hand evaluator performance

`internal/eval` has two evaluators that return identical results:

- **Naive** (`naive.go`): ranks a 5-card hand by counting ranks and checking for flushes and straights. For 7 cards it tries all 21 five-card combinations and keeps the best.
- **Bitmask** (`Evaluate`): builds one 16-bit rank mask per suit and reads flushes, straights, pairs, trips and quads straight off those masks with bitwise operations. It doesn't loop over combinations, allocate memory or use lookup tables.

The bitmask evaluator is checked against the naive one on all 2,598,960 five-card hands and a million random 6- and 7-card hands. It also reproduces the published category counts for all 133,784,560 seven-card hands.

Median of 5 runs, `go test -bench=. -count=5 ./internal/eval`, Apple M2 Pro, Go 1.27, one core:

| Hand size | Naive | Bitmask | Speedup |
| --- | ---: | ---: | ---: |
| 5 cards | 103 ns | 32 ns | 3.2x |
| 7 cards | 1,813 ns | 31 ns | 59x |

At 31 ns per 7-card hand, one core ranks about 32 million hands per second. Neither evaluator allocates memory.
