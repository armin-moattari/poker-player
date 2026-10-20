// Package eval ranks Texas Hold'em hands.
//
// Evaluate5 ranks exactly five cards, and Evaluate finds the best five-card
// hand among five to seven cards (two hole cards plus up to five board cards).
// Both return a HandRank: a single number where a higher value always means a
// stronger hand and equal values mean a split pot. Comparing hands is then just
// comparing numbers, which keeps showdown and equity code simple and fast.
//
// This is the straightforward ("naive") evaluator. It is written for clarity
// and serves as the reference that a faster lookup-table evaluator will be
// tested against.
package eval

import (
	"fmt"
	"strings"

	"github.com/armin-moattari/poker-player/internal/card"
)

// Category is the type of a poker hand, from HighCard (weakest) to
// StraightFlush (strongest). A royal flush is simply the ace-high
// straight flush, so it has no separate category.
type Category uint8

// Hand categories, weakest to strongest. The order matters: HandRank
// stores the category in its top bits, so a higher category always beats
// a lower one regardless of the cards.
const (
	HighCard Category = iota
	OnePair
	TwoPair
	ThreeOfAKind
	Straight
	Flush
	FullHouse
	FourOfAKind
	StraightFlush
)

var categoryNames = [...]string{
	HighCard:      "high card",
	OnePair:       "one pair",
	TwoPair:       "two pair",
	ThreeOfAKind:  "three of a kind",
	Straight:      "straight",
	Flush:         "flush",
	FullHouse:     "full house",
	FourOfAKind:   "four of a kind",
	StraightFlush: "straight flush",
}

// String returns the category's name in lowercase, such as "full house".
func (c Category) String() string {
	if int(c) >= len(categoryNames) {
		return "unknown"
	}
	return categoryNames[c]
}

// HandRank is the strength of a five-card hand. Higher is stronger, and two
// hands with the same HandRank tie.
//
// Layout, from the most significant bits down:
//
//	bits 20-23: Category
//	bits 16-19: most important rank
//	bits 12-15: second rank
//	bits  8-11: third rank
//	bits  4-7:  fourth rank
//	bits  0-3:  fifth rank
//
// Each rank fits in 4 bits because ranks run from 2 to 14. The ranks are
// the hand's tie-breakers in the order poker compares them. For example a
// full house KKK22 stores K then 2, and two pair QQ77A stores Q, 7, A.
// Unused rank slots are zero. Because the category sits above every rank,
// ordinary integer comparison gives the correct poker order.
type HandRank uint32

const (
	categoryShift = 20
	rankBits      = 4
	maxTiebreaks  = 5
)

// Category returns the hand's category.
func (h HandRank) Category() Category { return Category(h >> categoryShift) }

// ranks returns the tie-breaking ranks stored in h, most important first,
// stopping at the first empty slot.
func (h HandRank) ranks() []card.Rank {
	var rs []card.Rank
	for i := maxTiebreaks - 1; i >= 0; i-- {
		r := card.Rank(h >> (i * rankBits) & 0xF)
		if r == 0 {
			break
		}
		rs = append(rs, r)
	}
	return rs
}

// String describes the hand, such as "full house (K 2)" or
// "straight (5)" for the A-2-3-4-5 wheel.
func (h HandRank) String() string {
	parts := make([]string, 0, maxTiebreaks)
	for _, r := range h.ranks() {
		parts = append(parts, r.String())
	}
	return fmt.Sprintf("%s (%s)", h.Category(), strings.Join(parts, " "))
}

// makeRank packs a category and its tie-breaking ranks into a HandRank.
// ranks must be in comparison order, most important first, at most five.
func makeRank(c Category, ranks ...card.Rank) HandRank {
	h := HandRank(c) << categoryShift
	for i, r := range ranks {
		h |= HandRank(r) << ((maxTiebreaks - 1 - i) * rankBits)
	}
	return h
}

// Evaluate5 returns the rank of exactly five cards. The cards must be
// distinct; the order they are passed in does not matter.
func Evaluate5(cards [5]card.Card) HandRank {
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

// Evaluate returns the rank of the best five-card hand that can be made from
// cards, which must hold between five and seven distinct cards. It panics
// otherwise, since a wrong count is a programming error, not bad input.
//
// It tries every five-card combination (21 of them for seven cards) and keeps
// the strongest. That is easy to verify but slow; milestone 4 replaces it with
// a faster evaluator and uses this one to check the results.
func Evaluate(cards []card.Card) HandRank {
	n := len(cards)
	if n < 5 || n > 7 {
		panic(fmt.Sprintf("eval: Evaluate needs 5 to 7 cards, got %d", n))
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
						if r := Evaluate5(hand); r > best {
							best = r
						}
					}
				}
			}
		}
	}
	return best
}
