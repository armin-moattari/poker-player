package eval

import (
	"math/rand/v2"
	"testing"

	"github.com/armin-moattari/poker-player/internal/card"
)

// These tests check the fast bitmask Evaluate against the simple
// evaluateNaive, which is easy to trust because it literally tries every
// five-card combination. If both agree on every hand, the fast one is right.

// TestEvaluateMatchesNaiveAllFiveCardHands compares the two on all
// 2,598,960 five-card hands.
func TestEvaluateMatchesNaiveAllFiveCardHands(t *testing.T) {
	if testing.Short() {
		t.Skip("enumerates all five-card hands")
	}
	var hand [5]card.Card
	for a := card.Card(0); a < card.NumCards; a++ {
		for b := a + 1; b < card.NumCards; b++ {
			for c := b + 1; c < card.NumCards; c++ {
				for d := c + 1; d < card.NumCards; d++ {
					for e := d + 1; e < card.NumCards; e++ {
						hand = [5]card.Card{a, b, c, d, e}
						if got, want := Evaluate(hand[:]), evaluate5Naive(hand); got != want {
							t.Fatalf("%v: Evaluate = %v, naive = %v", hand, got, want)
						}
					}
				}
			}
		}
	}
}

// TestEvaluateMatchesNaiveRandom compares the two on a million random
// six- and seven-card hands from a fixed seed. Checking every seven-card
// hand against the naive evaluator would take minutes, so the exhaustive
// seven-card check below uses known category totals instead.
func TestEvaluateMatchesNaiveRandom(t *testing.T) {
	n := 1_000_000
	if testing.Short() {
		n = 20_000
	}
	r := rand.New(rand.NewPCG(3, 4))
	for _, size := range []int{6, 7} {
		for range n / 2 {
			d := card.NewDeck()
			d.Shuffle(r)
			hand, _ := d.Deal(size)
			if got, want := Evaluate(hand), evaluateNaive(hand); got != want {
				t.Fatalf("%v: Evaluate = %v, naive = %v", hand, got, want)
			}
		}
	}
}

// TestEvaluateAllSevenCardHands ranks all 133,784,560 seven-card hands
// with the fast evaluator and checks the count in each category against
// the published totals for seven-card poker.
func TestEvaluateAllSevenCardHands(t *testing.T) {
	if testing.Short() {
		t.Skip("enumerates all seven-card hands")
	}
	want := map[Category]int{
		StraightFlush: 41584,
		FourOfAKind:   224848,
		FullHouse:     3473184,
		Flush:         4047644,
		Straight:      6180020,
		ThreeOfAKind:  6461620,
		TwoPair:       31433400,
		OnePair:       58627800,
		HighCard:      23294460,
	}
	var got [StraightFlush + 1]int
	var hand [7]card.Card
	const n = card.NumCards
	for a := card.Card(0); a < n; a++ {
		hand[0] = a
		for b := a + 1; b < n; b++ {
			hand[1] = b
			for c := b + 1; c < n; c++ {
				hand[2] = c
				for d := c + 1; d < n; d++ {
					hand[3] = d
					for e := d + 1; e < n; e++ {
						hand[4] = e
						for f := e + 1; f < n; f++ {
							hand[5] = f
							for g := f + 1; g < n; g++ {
								hand[6] = g
								got[Evaluate(hand[:]).Category()]++
							}
						}
					}
				}
			}
		}
	}
	for cat, n := range want {
		if got[cat] != n {
			t.Errorf("%v: got %d hands, want %d", cat, got[cat], n)
		}
	}
}
