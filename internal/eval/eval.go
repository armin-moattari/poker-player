// Package eval ranks Texas Hold'em hands.
//
// Evaluate5 ranks exactly five cards, and Evaluate finds the best five-card
// hand among five to seven cards (two hole cards plus up to five board cards).
// Both return a HandRank: a single number where a higher value always means a
// stronger hand and equal values mean a split pot. Comparing hands is then just
// comparing numbers, which keeps showdown and equity code simple and fast.
//
// Evaluate uses a bitmask method (see its doc comment) and ranks a 7-card
// hand in about 30 ns with no allocations. naive.go keeps the original
// try-every-combination evaluator, about 60 times slower, as the reference
// the fast one is tested against. Benchmark numbers are in the README.
package eval

import (
	"fmt"
	"math/bits"
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

// Evaluate5 returns the rank of exactly five cards. It is a convenience
// wrapper around Evaluate for callers that hold a fixed-size array.
func Evaluate5(cards [5]card.Card) HandRank {
	return Evaluate(cards[:])
}

// Evaluate returns the rank of the best five-card hand that can be made from
// cards, which must hold between five and seven distinct cards. It panics
// otherwise, since a wrong count is a programming error, not bad input.
//
// Instead of trying every five-card combination (21 for seven cards), it
// looks at all the cards at once using bitmasks: one 16-bit mask per suit,
// where bit r is set if that suit holds a card of rank r. Flushes, straights,
// pairs, trips and quads can all be read off those four masks with a handful
// of bitwise operations, and there are no loops over combinations, no
// allocations and no lookup tables to build.
func Evaluate(cards []card.Card) HandRank {
	if n := len(cards); n < 5 || n > 7 {
		panic(fmt.Sprintf("eval: Evaluate needs 5 to 7 cards, got %d", n))
	}

	// suits[s] has bit r set when the hand holds rank r in suit s.
	var suits [4]uint16
	for _, c := range cards {
		suits[c.Suit()] |= 1 << c.Rank()
	}

	// Flush check. With at most seven cards, a hand that holds a flush
	// cannot also hold four of a kind or a full house (both would need more
	// than seven cards), so the flush suit alone decides the hand.
	for _, m := range suits {
		if bits.OnesCount16(m) >= 5 {
			if high, ok := straightTop(m); ok {
				return makeRank(StraightFlush, high)
			}
			return addTop(makeRank(Flush), 0, m, 5)
		}
	}

	// Sort every rank into exactly one of four disjoint masks by how many
	// times it appears: ones, twos, threes, fours. Each suit mask adds at
	// most one card to each rank, so feeding the masks in one at a time
	// moves a rank up by one level whenever its bit is set. All four new
	// values are computed from the old ones, which is why they are
	// assigned in a single statement.
	var ones, twos, threes, fours uint16
	for _, m := range suits {
		ones, twos, threes, fours =
			(ones&^m)|(m&^(ones|twos|threes|fours)),
			(twos&^m)|(ones&m),
			(threes&^m)|(twos&m),
			fours|(threes&m)
	}
	all := ones | twos | threes | fours // every rank present at least once

	switch {
	case fours != 0:
		quad := topBit(fours)
		return makeRank(FourOfAKind, quad, topBit(all&^(1<<quad)))

	case threes != 0 && bits.OnesCount16(threes)+bits.OnesCount16(twos) >= 2:
		// Full house: the best trips, plus the best pair from what is
		// left. A second set of trips counts as a pair here (KKK 777 x
		// plays as kings full of sevens).
		trip := topBit(threes)
		return makeRank(FullHouse, trip, topBit((threes&^(1<<trip))|twos))
	}

	if high, ok := straightTop(all); ok {
		return makeRank(Straight, high)
	}

	switch {
	case threes != 0:
		trip := topBit(threes)
		return addTop(makeRank(ThreeOfAKind, trip), 1, all&^(1<<trip), 2)

	case bits.OnesCount16(twos) >= 2:
		// With three pairs, the lowest pair can still be the kicker,
		// so the kicker comes from every rank except the two used.
		hi := topBit(twos)
		lo := topBit(twos &^ (1 << hi))
		return makeRank(TwoPair, hi, lo, topBit(all&^(1<<hi|1<<lo)))

	case twos != 0:
		pair := topBit(twos)
		return addTop(makeRank(OnePair, pair), 1, all&^(1<<pair), 3)

	default:
		return addTop(makeRank(HighCard), 0, all, 5)
	}
}

// topBit returns the highest rank set in mask, which must not be zero.
// bits.Len16 gives the position of the highest set bit plus one.
func topBit(mask uint16) card.Rank {
	return card.Rank(bits.Len16(mask) - 1)
}

// addTop writes the n highest ranks in mask into h, starting at tie-break
// slot (0 is the most important slot). Writing straight into the HandRank,
// rather than building a slice of ranks first, keeps Evaluate free of heap
// allocations.
func addTop(h HandRank, slot int, mask uint16, n int) HandRank {
	for i := 0; i < n && mask != 0; i++ {
		r := topBit(mask)
		h |= HandRank(r) << ((maxTiebreaks - 1 - slot - i) * rankBits)
		mask &^= 1 << r
	}
	return h
}

// straightTop reports whether mask holds five consecutive ranks and returns
// the top rank of the highest such run. mask may hold more than five ranks.
func straightTop(mask uint16) (card.Rank, bool) {
	// An ace can also play low, below the two. Bit 1 is unused by real
	// ranks (they start at 2), so copy the ace there to let the wheel
	// A-2-3-4-5 look like any other run of five.
	if mask&(1<<card.Ace) != 0 {
		mask |= 1 << 1
	}
	// ANDing the mask with itself shifted by 1, 2, 3 and 4 leaves a bit set
	// only where that rank and the four ranks above it are all present.
	// Each surviving bit marks the bottom card of a straight.
	run := mask & (mask >> 1) & (mask >> 2) & (mask >> 3) & (mask >> 4)
	if run == 0 {
		return 0, false
	}
	return topBit(run) + 4, true
}
