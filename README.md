# poker-player

A Texas Hold'em hand evaluator and equity calculator written in Go. It's headed toward a hand advisor you can use from the command line or from a web app, which can be added to an iPhone home screen.

Work in progress. See [CLAUDE.md](CLAUDE.md) for the full plan and milestones.

## What's here

| Package | What it does | Status |
| --- | --- | --- |
| `internal/card` | Cards, parsing (`"As"`, `"Td"`), and a seedable deck | Done |
| `internal/eval` | Ranks any 5 to 7 cards; fast bitmask evaluator plus a naive reference | Done |
| `internal/equity` | Monte Carlo equity against one or more random hands, across all cores | Planned |
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
go test -bench=. ./internal/eval    # evaluator benchmarks
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
