package eval

import (
	"testing"

	"github.com/armin-moattari/poker-player/internal/card"
)

// five parses a five-card hand like "As Ks Qs Js Ts" for Evaluate5.
func five(t *testing.T, s string) [5]card.Card {
	t.Helper()
	cs, err := card.ParseCards(s)
	if err != nil || len(cs) != 5 {
		t.Fatalf("bad test hand %q: %v", s, err)
	}
	return [5]card.Card(cs)
}

// cards parses any list of cards for Evaluate.
func cards(t *testing.T, s string) []card.Card {
	t.Helper()
	cs, err := card.ParseCards(s)
	if err != nil {
		t.Fatalf("bad test cards %q: %v", s, err)
	}
	return cs
}

func TestEvaluate5Categories(t *testing.T) {
	tests := []struct {
		hand string
		want Category
		str  string
	}{
		{"As Ks Qs Js Ts", StraightFlush, "straight flush (A)"},
		{"5d 4d 3d 2d Ad", StraightFlush, "straight flush (5)"}, // steel wheel
		{"9c 9d 9h 9s 2c", FourOfAKind, "four of a kind (9 2)"},
		{"Kc Kd Kh 2s 2c", FullHouse, "full house (K 2)"},
		{"Ah Jh 8h 4h 2h", Flush, "flush (A J 8 4 2)"},
		{"Tc 9d 8h 7s 6c", Straight, "straight (T)"},
		{"Ac 2d 3h 4s 5c", Straight, "straight (5)"}, // wheel, ace low
		{"7c 7d 7h Ks 2c", ThreeOfAKind, "three of a kind (7 K 2)"},
		{"Qc Qd 7h 7s Ac", TwoPair, "two pair (Q 7 A)"},
		{"Jc Jd 9h 5s 3c", OnePair, "one pair (J 9 5 3)"},
		{"Ac Qd 9h 5s 3c", HighCard, "high card (A Q 9 5 3)"},
	}
	for _, tt := range tests {
		got := Evaluate5(five(t, tt.hand))
		if got.Category() != tt.want {
			t.Errorf("%s: category %v, want %v", tt.hand, got.Category(), tt.want)
		}
		if got.String() != tt.str {
			t.Errorf("%s: String() = %q, want %q", tt.hand, got.String(), tt.str)
		}
	}
}

// TestEvaluate5Ordering lists pairs where the first hand must beat the
// second. Each pair targets one tie-break rule or edge case.
func TestEvaluate5Ordering(t *testing.T) {
	tests := []struct{ stronger, weaker, why string }{
		{"2c 3c 4c 5c 6c", "Ac Ad Ah As Kc", "straight flush beats quads"},
		{"2c 2d 2h 2s 3c", "Ac Ad Ah Kc Kd", "quads beat full house"},
		{"2c 2d 2h 3s 3c", "Ac Kc Qc Jc 9c", "full house beats flush"},
		{"7c 5c 4c 3c 2c", "Ac Kd Qh Js Tc", "flush beats straight"},
		{"Ac 2d 3h 4s 5c", "Ac Ad Ah Kc Qd", "straight (even the wheel) beats trips"},
		{"2c 2d 2h 4s 5c", "Ac Ad Kh Ks Qc", "trips beat two pair"},
		{"2c 2d 3h 3s 4c", "Ac Ad Kh Qs Jc", "two pair beats one pair"},
		{"2c 2d 3h 4s 5c", "Ac Kd Qh Js 9c", "one pair beats high card"},

		{"6c 5d 4h 3s 2c", "Ac 2d 3h 4s 5c", "six-high straight beats the wheel"},
		{"6h 5h 4h 3h 2h", "5d 4d 3d 2d Ad", "six-high straight flush beats steel wheel"},
		{"Ac Ad Ah As 3c", "Kc Kd Kh Ks Ac", "higher quads win"},
		{"Ac Ad Ah As 3c", "Ac Ad Ah As 2c", "quads: kicker decides"},
		{"3c 3d 3h 2s 2c", "2c 2d 2h As Ac", "full house: trips rank first"},
		{"Ac Ad Ah 3s 3c", "Ac Ad Ah 2s 2c", "full house: then the pair"},
		{"Ac Kc Qc Jc 9c", "Ac Kc Qc Jc 8c", "flush: fifth card decides"},
		{"Ac Ad Ah 3s 2c", "Kc Kd Kh As Qc", "higher trips win over kickers"},
		{"7c 7d 7h As 2c", "7c 7d 7h Ks Qc", "trips: first kicker"},
		{"7c 7d 7h As 3c", "7c 7d 7h As 2c", "trips: second kicker"},
		{"Ac Ad 2h 2s 3c", "Kc Kd Qh Qs Jc", "two pair: top pair first"},
		{"Ac Ad 3h 3s 2c", "Ac Ad 2h 2s Kc", "two pair: then bottom pair"},
		{"Ac Ad 3h 3s 5c", "Ac Ad 3h 3s 4c", "two pair: then kicker"},
		{"Jc Jd 9h 5s 4c", "Jc Jd 9h 5s 3c", "one pair: third kicker"},
		{"Ac Qd 9h 5s 3c", "Ac Qd 9h 5s 2c", "high card: last card"},
		{"Ac Qd 9h 5s 3c", "Kc Qd Jh Ts 8c", "high card: ace beats king-high"},
	}
	for _, tt := range tests {
		s, w := Evaluate5(five(t, tt.stronger)), Evaluate5(five(t, tt.weaker))
		if s <= w {
			t.Errorf("%s: %s (%v) should beat %s (%v)", tt.why, tt.stronger, s, tt.weaker, w)
		}
	}
}

func TestEvaluate5Ties(t *testing.T) {
	tests := [][2]string{
		{"As Kd Qh Js 9c", "Ac Kh Qd Jc 9s"}, // same ranks, suits differ
		{"Ac Ad Kh Ks Qc", "Ah As Kc Kd Qs"}, // same two pair and kicker
		{"Tc 9d 8h 7s 6c", "Th 9s 8c 7d 6h"}, // same straight
		{"Ah Jh 8h 4h 2h", "As Js 8s 4s 2s"}, // same flush, different suit
	}
	for _, tt := range tests {
		a, b := Evaluate5(five(t, tt[0])), Evaluate5(five(t, tt[1]))
		if a != b {
			t.Errorf("%s (%v) and %s (%v) should tie", tt[0], a, tt[1], b)
		}
	}
}

func TestEvaluate5OrderDoesNotMatter(t *testing.T) {
	want := Evaluate5(five(t, "Kc Kd Kh 2s 2c"))
	for _, s := range []string{"2c Kc 2s Kd Kh", "2s 2c Kh Kd Kc", "Kd 2c Kh 2s Kc"} {
		if got := Evaluate5(five(t, s)); got != want {
			t.Errorf("Evaluate5(%s) = %v, want %v", s, got, want)
		}
	}
}

// TestEvaluate5AllHands ranks every one of the 2,598,960 five-card hands and
// checks the number in each category against the well-known totals. Any
// mistake in straight, flush or grouping logic shows up as a wrong count.
func TestEvaluate5AllHands(t *testing.T) {
	if testing.Short() {
		t.Skip("enumerates all five-card hands")
	}
	want := map[Category]int{
		StraightFlush: 40,
		FourOfAKind:   624,
		FullHouse:     3744,
		Flush:         5108,
		Straight:      10200,
		ThreeOfAKind:  54912,
		TwoPair:       123552,
		OnePair:       1098240,
		HighCard:      1302540,
	}
	got := make(map[Category]int)
	// distinct counts how many different HandRank values exist; there are
	// exactly 7462 distinct five-card hand strengths.
	distinct := make(map[HandRank]bool)
	for a := card.Card(0); a < card.NumCards; a++ {
		for b := a + 1; b < card.NumCards; b++ {
			for c := b + 1; c < card.NumCards; c++ {
				for d := c + 1; d < card.NumCards; d++ {
					for e := d + 1; e < card.NumCards; e++ {
						r := Evaluate5([5]card.Card{a, b, c, d, e})
						got[r.Category()]++
						distinct[r] = true
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
	if len(distinct) != 7462 {
		t.Errorf("got %d distinct hand ranks, want 7462", len(distinct))
	}
}

func TestEvaluateSevenCards(t *testing.T) {
	tests := []struct {
		cards string
		want  string
		why   string
	}{
		{"As Ks Qs Js Ts 2c 3d", "straight flush (A)", "royal flush on the board plays"},
		{"9h 8h 7h 6h 5h 4h 3h", "straight flush (9)", "longest straight flush picks the top five"},
		{"Ah Kh 9h 5h 2h 3h 4c", "flush (A K 9 5 3)", "six suited cards: best five of the flush"},
		{"9h 8d 7h 6h 5c 4h 2h", "flush (9 7 6 4 2)", "flush beats a straight in the same seven"},
		{"Kc Kd Kh 7s 7c 7d 2h", "full house (K 7)", "two sets make a full house"},
		{"Ac Ad 3h 3s 2c 2d Kh", "two pair (A 3 K)", "three pairs: best two plus best kicker"},
		{"Qc Qd Qh 4s 4c 4d 4h", "four of a kind (4 Q)", "quads with trips: kicker from the trips"},
		{"Ac 2d 3h 4s 5c 6d Kh", "straight (6)", "wheel cards plus a six make a six-high straight"},
		{"Ac 2d 3h 4s 5c Kd Qh", "straight (5)", "wheel found among seven cards"},
		{"Ac Kd 8h 7s 5c 3d 2h", "high card (A K 8 7 5)", "high card drops the two lowest"},
	}
	for _, tt := range tests {
		if got := Evaluate(cards(t, tt.cards)).String(); got != tt.want {
			t.Errorf("%s: Evaluate(%s) = %q, want %q", tt.why, tt.cards, got, tt.want)
		}
	}
}

func TestEvaluateFiveAndSixCards(t *testing.T) {
	five := "Kc Kd Kh 2s 2c"
	if got, want := Evaluate(cards(t, five)), Evaluate5([5]card.Card(cards(t, five))); got != want {
		t.Errorf("Evaluate(5 cards) = %v, want %v", got, want)
	}
	if got := Evaluate(cards(t, "Ac Kc Qc Jc 2d Tc")).String(); got != "straight flush (A)" {
		t.Errorf("Evaluate(6 cards) = %q, want straight flush (A)", got)
	}
}

// TestEvaluateSplitPot checks a hold'em showdown where the board plays for
// both players, so their seven-card ranks must be equal.
func TestEvaluateSplitPot(t *testing.T) {
	board := "Ac Kd Qh Js Tc"
	p1 := Evaluate(cards(t, "2c 3d "+board))
	p2 := Evaluate(cards(t, "4h 5s "+board))
	if p1 != p2 {
		t.Errorf("board straight should split: %v vs %v", p1, p2)
	}
	// A pocket pair that is lower than the board's two pair does not play,
	// so the hands still tie on the board's best five.
	board = "Ac Ad Kh Ks Qc"
	if a, b := Evaluate(cards(t, "2c 2d "+board)), Evaluate(cards(t, "3h 4s "+board)); a != b {
		t.Errorf("counterfeited pair should split: %v vs %v", a, b)
	}
}

func TestEvaluatePanicsOnBadCount(t *testing.T) {
	for _, s := range []string{"Ac Kd Qh Js", "Ac Kd Qh Js Tc 9c 8c 7c"} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("Evaluate(%s) did not panic", s)
				}
			}()
			Evaluate(cards(t, s))
		}()
	}
}

func TestCategoryString(t *testing.T) {
	if got := FullHouse.String(); got != "full house" {
		t.Errorf("FullHouse.String() = %q", got)
	}
	if got := Category(99).String(); got != "unknown" {
		t.Errorf("Category(99).String() = %q, want unknown", got)
	}
}
