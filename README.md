# poker-player

A Texas Hold'em hand evaluator and equity calculator written in Go, compiled to WebAssembly and wrapped in a mobile-friendly web app.

Work in progress. See [CLAUDE.md](CLAUDE.md) for the plan and milestones.

## Commands

```bash
go test ./...                                   # run tests
go test -bench=. ./internal/eval                # benchmarks
GOOS=js GOARCH=wasm go build -o web/main.wasm ./wasm   # WASM build
```
