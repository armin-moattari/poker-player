# Poker Equity Engine

A Texas Hold'em hand evaluator and equity calculator written in Go, compiled to WebAssembly and wrapped in a mobile-friendly web app (PWA) that can be added to an iPhone home screen.

## Language rules (read first)
- **This project is Go only.** All evaluator, equity, CLI and WASM code is Go (Go 1.27, installed via Homebrew at /opt/homebrew/bin/go).
- **Never write Python** in this repo: no .py files, no requirements.txt, no venv, no pytest. If you notice yourself reaching for Python, stop and use Go.
- The only non-Go code allowed is the frontend in `web/` (HTML/CSS/TypeScript or plain JS) and the CI workflow YAML.
- Tests use Go's built-in `testing` package (`*_test.go` files), and benchmarks use `testing.B`.
- Start by running `go mod init github.com/<armin's-github-username>/poker-engine` (ask for the username).

## Goals
- Learn Go and solid engineering habits; this is a resume project for SWE roles.
- Measured performance story: naive evaluator first, then optimized (bit representation / lookup tables), with benchmarks recorded in the README.
- Correctness proven by tests (wheel straights, kickers, split pots, flush vs. straight edge cases).
- Concurrent Monte Carlo equity simulation across cores.

## How we work together
- **Claude writes the code, including core evaluator/equity logic and tests** (Armin asked for this on 2026-10-20). Explain the key design choices in each change so Armin can learn from and review it.
- **Comment and document code well.** Every package gets a package doc, every exported identifier a doc comment, and non-obvious logic (bit tricks, invariants, why a choice was made) an inline comment. Check the result with `go doc`.
- Keep packages small and modular. Suggest a refactor when a file or function grows unwieldy.
- Run `go test ./...` (and `go vet ./...`) before every commit. Never commit failing tests.
- Show the proposed commit message and wait for approval before committing. Use small, focused commits with conventional-style messages (e.g. `feat(eval): rank five-card hands`).
- Explain *why* when recommending a change, briefly.

## Planned structure
```
cmd/cli/          # CLI front end: advisor, equity queries, bot matches
internal/card/    # card and deck representation, parsing ("As", "Td")
internal/eval/    # hand ranking (5- and 7-card)
internal/equity/  # Monte Carlo + exhaustive equity, heads-up and multiway
internal/advisor/ # recommends fold/call/raise from hand, board, pot, to-call, players
internal/game/    # heads-up no-limit hold'em engine: blinds, streets, showdown, matches
internal/bot/     # bots that play the engine (RandomBot, EquityBot)
wasm/             # WebAssembly entry point (exposes advisor + equity to JS)
web/              # GUI: frontend + PWA manifest
```

The advisor is a library, not a CLI feature. `internal/advisor` takes plain inputs and returns a result struct (action, amount, equity, pot odds, reasons) with no printing or flag parsing, so the CLI and the web GUI are both thin front ends over the same code.

## Milestones
1. Project setup: go module, git, GitHub repo, CI (GitHub Actions running tests).
2. Cards and deck with parsing and tests.
3. Naive 5-card evaluator plus thorough tests; then 7-card (best 5 of 7).
4. Benchmarks; optimized evaluator; record before/after numbers.
5. Equity calculator (Monte Carlo, then parallelized with goroutines), including multiway equity against several random opponents.
6. Hand advisor library: recommendation from equity vs. pot odds, with the reasoning; first front end in `cmd/cli`.
7. Game engine (heads-up no-limit: blinds, four betting streets, side cases like all-ins, showdown, matches) plus RandomBot and EquityBot, and a CLI to run seeded bot-vs-bot matches.
8. WASM build plus touch-friendly web GUI for the advisor (and equity); PWA manifest; deploy (Vercel or GitHub Pages).
9. Stretch: play against the bots in the web GUI; multiplayer server.

## Commands
- Test: `go test ./...`
- Benchmark: `go test -bench=. ./internal/eval`
- WASM build: `GOOS=js GOARCH=wasm go build -o web/main.wasm ./wasm`
