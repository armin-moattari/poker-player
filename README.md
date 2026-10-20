# poker-player

A Texas Hold'em hand evaluator and equity calculator written in Go. It's headed toward a hand advisor you can use from the command line or from a web app, which can be added to an iPhone home screen.

Work in progress. See [CLAUDE.md](CLAUDE.md) for the full plan and milestones.

## What's here

| Package | What it does | Status |
| --- | --- | --- |
| `internal/card` | Cards, parsing (`"As"`, `"Td"`), and a seedable deck | Done |
| `internal/eval` | Ranks any 5 to 7 cards; fast bitmask evaluator plus a naive reference | Done |
| `internal/equity` | Exact and Monte Carlo equity, heads-up or multiway, across all cores | Done |
| `internal/advisor` | Recommends fold, call or raise for your spot, with the reasoning | Planned |
| `internal/game` | Plays heads-up no-limit hands (blinds, four streets, showdown) and matches | Planned |
| `internal/bot` | `RandomBot` and `EquityBot` that play the game engine | Planned |
| `cmd/cli` | Command-line front end for the advisor, equity and bot matches | Planned |
| `wasm`, `web` | Web GUI for the advisor and equity, installable as an app | Planned |

## Quick start

Needs Go 1.27 or newer.

```bash
go test ./...                       # all tests, including exhaustive all-hands checks (a few seconds)
go test -short ./...                # quick run that skips the exhaustive checks
go test -bench=. ./internal/...      # evaluator and equity benchmarks
go doc ./internal/eval              # package documentation
```

## Cards and deck (`internal/card`)

A `Card` is one byte holding an index from 0 to 51, so cards are cheap to copy and any set of cards fits in a `uint64` bitmask.

```go
hand, err := card.ParseCards("As Kd")     // also "AsKd", "As,Kd", "as kd", "10c 9c"
fmt.Println(hand, hand[0].Rank(), hand[0].Suit())   // [As Kd] A s

d := card.NewDeck()
d.Remove(hand...)                         // take known cards out
d.Shuffle(rand.New(rand.NewPCG(1, 2)))    // seeded, so the same every run
board, err := d.Deal(5)                   // 5 cards off the top; 45 left
```

| Function | What it does |
| --- | --- |
| `New(rank, suit)` | Makes a card from a `Rank` (`Two` to `Ace`) and a `Suit` (`Clubs`, `Diamonds`, `Hearts`, `Spades`) |
| `Parse(s)` | Parses one card like `"As"`, `"td"` or `"10c"`; returns an error for anything else |
| `ParseCards(s)` | Parses a list of cards, with or without spaces or commas; rejects duplicates |
| `MustParse(s)` | Like `Parse` but panics on a bad card; for hard-coded cards in tests |
| `c.Rank()`, `c.Suit()`, `c.String()` | A card's rank, suit, and two-character form (`"As"`) |
| `NewDeck()` | A full, unshuffled 52-card deck |
| `d.Shuffle(r)` | Shuffles the remaining cards with the random generator you pass in |
| `d.Deal(n)` | Removes and returns the top `n` cards; errors if there aren't enough |
| `d.Remove(cards...)` | Takes specific cards out, such as players' hole cards; on error the deck is unchanged |
| `d.Len()` | Cards left in the deck |

## Hand evaluator (`internal/eval`)

Every hand becomes one `HandRank` number: a higher number always wins and equal numbers split the pot, so a showdown is a plain `>` comparison.

```go
board, _ := card.ParseCards("Ks 7h 7d 2c 9s")
alice, _ := card.ParseCards("Kd Kc")
bob, _ := card.ParseCards("As 7c")

a := eval.Evaluate(append(alice, board...))   // full house (K 7)
b := eval.Evaluate(append(bob, board...))     // three of a kind (7 A K)
fmt.Println(a > b, a.Category())              // true full house
```

| Function | What it does |
| --- | --- |
| `Evaluate(cards)` | Ranks the best 5-card hand from 5 to 7 cards (hole cards plus board) |
| `Evaluate5(cards)` | Same, for exactly five cards held in a `[5]card.Card` array |
| `h.Category()` | The hand type, from `HighCard` up to `StraightFlush` (a royal flush is the ace-high straight flush) |
| `h.String()` | A readable description, such as `"full house (K 7)"` or `"straight (5)"` for the A-2-3-4-5 wheel |

How a `HandRank` is built: the category sits in the top bits, followed by up to five tie-break ranks in the order poker compares them. For example, two pair `QQ77A` stores Q, then 7, then A.

### Performance

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

## Equity calculator (`internal/equity`)

Equity is a hand's average share of the pot: 100% always wins, and a two-way split counts as half a win. `Calculate` takes the known hole cards, the board so far, and how many extra opponents hold unknown cards.

```go
aces, _ := card.ParseCards("AhAs")
kings, _ := card.ParseCards("KdKc")
r, err := equity.Calculate(ctx, equity.Request{Hands: [][]card.Card{aces, kings}})
// r.Exact == true, r.Deals == 1712304 (every possible board)
// r.Hands[0]: win 81.06%, tie 0.38%, equity 81.26%

hand, _ := card.ParseCards("AsKd")
board, _ := card.ParseCards("Qs Jh 2c")
r, err = equity.Calculate(ctx, equity.Request{
	Hands: [][]card.Card{hand}, Board: board, Opponents: 2, Seed: 1,
})
// Monte Carlo against two random hands: equity about 43.8%
```

| Name | What it does |
| --- | --- |
| `Calculate(ctx, req)` | Returns each known hand's equity, win rate and tie rate; stops early with an error if `ctx` is cancelled |
| `Request.Hands` | Known hole cards, two per hand; results come back in the same order |
| `Request.Board` | Board cards so far: 0 (preflop), 3, 4 or 5 |
| `Request.Opponents` | Extra players with unknown cards; up to 10 players in total |
| `Request.Method` | `Auto` (default), `Exact` or `MonteCarlo` |
| `Request.Trials`, `Request.Seed` | Monte Carlo trial count (default 100,000) and seed; the same seed always gives the same result |
| `Request.Workers` | Goroutines to use; default is one per CPU core, and the result never depends on it |
| `Result.Hands[i]` | `Equity`, `Win` and `Tie` for known hand `i`, each from 0 to 1 |
| `Result.Exact`, `Result.Deals` | Which method ran, and how many boards it evaluated |

**How it chooses a method.** When every hand is known, `Auto` enumerates every remaining board exactly. Even preflop heads-up, that's only 1.7 million boards. With random opponents it uses Monte Carlo: it deals random boards and opponent hands many times and averages the results. 100,000 trials land within about 0.3 percentage points of the true value.

**How it uses every core.** The work is cut into many small jobs. For Monte Carlo, a job is a chunk of 4,096 trials. For exact enumeration, it's every board that starts with a given pair of cards. Goroutines take jobs from a shared atomic counter until none are left. Each job keeps its own tally, so the hot loop never takes a lock. Each Monte Carlo chunk seeds its own random generator from `(Seed, chunk number)`, and tallies are added up in job order. That's why the result is identical on 1 core or 10.

**How it's tested.** Exact preflop results match [published exact enumerations](https://www.deucescracked.com/blog/preflop-poker-equity-matchups-exact-data) for five matchups, to the two decimals published (e.g. AhAs vs KdKc: 81.06% win, 0.38% tie). Other tests cover:
- a flush draw on the turn that must equal exactly 9/44
- split pots where the board plays
- Monte Carlo agreeing with exact enumeration
- identical results with 1, 3 and 8 workers
- aces against one random hand coming out at about 85.2%
- cancellation and every kind of invalid request

### Performance

Median of 5 runs, `go test -bench=. -count=5 ./internal/equity`, Apple M2 Pro (6 performance + 4 efficiency cores), Go 1.27:

| Job | 1 core | 10 cores | Speedup |
| --- | ---: | ---: | ---: |
| Exact, preflop heads-up (1,712,304 boards) | 96 ms | 15 ms | 6.2x |
| Monte Carlo, 100,000 trials vs 3 random hands | 18.4 ms | 3.1 ms | 5.9x |

The speedup tops out around 6x because the efficiency cores are much slower than the performance cores. An earlier version split exact enumeration on only the first new board card. That gave 44 uneven jobs and a 4.4x speedup. Splitting on the first two cards gives about a thousand jobs and brought it to 6.2x.
