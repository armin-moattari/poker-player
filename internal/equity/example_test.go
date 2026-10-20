package equity_test

import (
	"context"
	"fmt"

	"github.com/armin-moattari/poker-player/internal/card"
	"github.com/armin-moattari/poker-player/internal/equity"
)

// Examples are compiled and run by go test, and the // Output: comment is
// checked, so the usage shown here (and copied into the README) cannot go
// stale.

func ExampleCalculate() {
	aces, _ := card.ParseCards("AhAs")
	kings, _ := card.ParseCards("KdKc")

	// Both hands known, preflop: every one of the 1,712,304 boards is checked.
	r, err := equity.Calculate(context.Background(), equity.Request{
		Hands: [][]card.Card{aces, kings},
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("exact=%v boards=%d\n", r.Exact, r.Deals)
	fmt.Printf("aces win %.2f%%, tie %.2f%%, equity %.2f%%\n",
		100*r.Hands[0].Win, 100*r.Hands[0].Tie, 100*r.Hands[0].Equity)
	// Output:
	// exact=true boards=1712304
	// aces win 81.06%, tie 0.38%, equity 81.26%
}

func ExampleCalculate_randomOpponents() {
	hand, _ := card.ParseCards("AsKd")
	board, _ := card.ParseCards("Qs Jh 2c")

	// Two opponents with unknown cards: Monte Carlo, seeded so it repeats.
	r, _ := equity.Calculate(context.Background(), equity.Request{
		Hands:     [][]card.Card{hand},
		Board:     board,
		Opponents: 2,
		Trials:    100_000,
		Seed:      1,
	})
	fmt.Printf("exact=%v deals=%d equity=%.1f%%\n", r.Exact, r.Deals, 100*r.Hands[0].Equity)
	// Output: exact=false deals=100000 equity=43.8%
}
