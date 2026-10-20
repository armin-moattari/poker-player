package eval

import (
	"math/rand/v2"
	"testing"

	"github.com/armin-moattari/poker-player/internal/card"
)

// benchHands deals n random hands of size cards from a fixed seed, so every
// benchmark run measures the same inputs. Generating them up front keeps the
// cost of shuffling out of the timed loop.
func benchHands(n, size int) [][]card.Card {
	r := rand.New(rand.NewPCG(1, 2))
	hands := make([][]card.Card, n)
	for i := range hands {
		d := card.NewDeck()
		d.Shuffle(r)
		hands[i], _ = d.Deal(size)
	}
	return hands
}

// sink stores benchmark results so the compiler cannot discard the calls
// being measured as dead code.
var sink HandRank

// BenchmarkEvaluate compares the naive and bitmask evaluators on random
// 5- and 7-card hands. Run it with:
//
//	go test -bench=. ./internal/eval
func BenchmarkEvaluate(b *testing.B) {
	impls := []struct {
		name string
		fn   func([]card.Card) HandRank
	}{
		{"naive", evaluateNaive},
		{"bitmask", Evaluate},
	}
	for _, size := range []int{5, 7} {
		hands := benchHands(1024, size)
		for _, impl := range impls {
			b.Run(impl.name+"/"+string(rune('0'+size))+"cards", func(b *testing.B) {
				for i := 0; b.Loop(); i++ {
					sink = impl.fn(hands[i%len(hands)])
				}
			})
		}
	}
}
