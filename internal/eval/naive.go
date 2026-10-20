package eval

import (
	"fmt"

	"github.com/armin-moattari/poker-player/internal/card"
)

// This file holds the original, straightforward evaluator. It is no longer
// used by the rest of the program; it stays as the reference implementation
// that the fast evaluator in eval.go is tested against, and as the "before"
// side of the benchmarks.

// evaluate5Naive returns the rank of exactly five cards. The cards must be
// distinct; the order they are passed in does not matter.
func evaluate5Naive(cards [5]card.Card) HandRank {
	// counts[r] is how many cards of rank r the hand holds. Index by rank
	// directly (2..14) and waste a couple of slots, which is simpler than
	// subtracting 2 everywhere.
	var counts [card.Ace + 1]int
	// rankMask has bit r set for each rank present, used to spot straights.
	var rankMask uint16
	flush := true
	for i, c := range cards {
		counts[c.Rank()]++
		rankMask |= 1 << c.Rank()
		if i > 0 && c.Suit() != cards[0].Suit() {
			flush = false
		}
	}

	// A straight needs five different ranks, so check that first.
	if high, ok := straightHigh(rankMask); ok {
		if flush {
			return makeRank(StraightFlush, high)
		}
		return makeRank(Straight, high)
	}

	// Group ranks by how many times they appear. Walking from Ace down to
	// Two and filling one list per count gives each list in descending
	// rank order, which is exactly the tie-break order poker uses within a
	// group (e.g. the higher of two pairs comes first).
	var quads, trips, pairs, singles []card.Rank
	for r := card.Ace; r >= card.Two; r-- {
		switch counts[r] {
		case 4:
			quads = append(quads, r)
		case 3:
			trips = append(trips, r)
		case 2:
			pairs = append(pairs, r)
		case 1:
			singles = append(singles, r)
		}
	}

	switch {
	case len(quads) == 1:
		return makeRank(FourOfAKind, quads[0], singles[0])
	case len(trips) == 1 && len(pairs) == 1:
		return makeRank(FullHouse, trips[0], pairs[0])
	case flush:
		// No pairs are possible in a flush (same suit, distinct cards),
		// so all five cards are singles, already sorted high to low.
		return makeRank(Flush, singles...)
	case len(trips) == 1:
		return makeRank(ThreeOfAKind, trips[0], singles[0], singles[1])
	case len(pairs) == 2:
		return makeRank(TwoPair, pairs[0], pairs[1], singles[0])
	case len(pairs) == 1:
		return makeRank(OnePair, pairs[0], singles[0], singles[1], singles[2])
	default:
		return makeRank(HighCard, singles...)
	}
}

// wheelMask is the rank mask of A-2-3-4-5, the lowest straight, where the
// ace plays low.
const wheelMask = 1<<card.Ace | 1<<card.Two | 1<<card.Three | 1<<card.Four | 1<<card.Five

// straightHigh reports whether the ranks in mask form a five-card straight
// and, if so, returns its top card. mask must hold exactly five ranks.
func straightHigh(mask uint16) (card.Rank, bool) {
	// Five consecutive set bits, shifted so the lowest is bit 0, equal
	// 0b11111. Checking each possible top card from Ace down is simple and
	// fast enough for this reference evaluator.
	for high := card.Ace; high >= card.Six; high-- {
		if mask == 0b11111<<(high-4) {
			return high, true
		}
	}
	// The wheel counts as a five-high straight, so it loses to 6-high.
	if mask == wheelMask {
		return card.Five, true
	}
	return 0, false
}

// evaluateNaive returns the rank of the best five-card hand that can be made from
// cards, which must hold between five and seven distinct cards. It panics
// otherwise, since a wrong count is a programming error, not bad input.
//
// It tries every five-card combination (21 of them for seven cards) and keeps
// the strongest. That is easy to verify but slow (about 21 times the work of
// one five-card evaluation); Evaluate in eval.go does the same job directly.
func evaluateNaive(cards []card.Card) HandRank {
	n := len(cards)
	if n < 5 || n > 7 {
		panic(fmt.Sprintf("eval: evaluateNaive needs 5 to 7 cards, got %d", n))
	}
	var best HandRank
	// Five nested loops pick indexes a < b < c < d < e, which visits every
	// five-card combination exactly once. Plain loops are easier to check
	// than a general combinations helper, and n is at most 7.
	for a := 0; a < n; a++ {
		for b := a + 1; b < n; b++ {
			for c := b + 1; c < n; c++ {
				for d := c + 1; d < n; d++ {
					for e := d + 1; e < n; e++ {
						hand := [5]card.Card{cards[a], cards[b], cards[c], cards[d], cards[e]}
						if r := evaluate5Naive(hand); r > best {
							best = r
						}
					}
				}
			}
		}
	}
	return best
}
