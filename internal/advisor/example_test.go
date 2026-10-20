package advisor_test

import (
	"context"
	"fmt"

	"github.com/armin-moattari/poker-player/internal/advisor"
	"github.com/armin-moattari/poker-player/internal/card"
)

// Examples are compiled and run by go test, and the // Output: comment is
// checked, so the usage shown here (and copied into the README) cannot go
// stale.

func ExampleAdvise() {
	hand, _ := card.ParseCards("As Kd")
	board, _ := card.ParseCards("Qs Jh 2c")

	cfg := advisor.DefaultConfig()
	cfg.Unit, cfg.ChipStep, cfg.BigBlind = "$", 0.25, 2 // a $1/$2 game

	a, err := advisor.Advise(context.Background(), advisor.Spot{
		Hand: hand, Board: board, Pot: 40, ToCall: 10, Players: 3,
		Position: advisor.Early, Seed: 1,
	}, cfg)
	if err != nil {
		panic(err)
	}
	fmt.Println(a.Action, cfg.Money(a.Amount), a.Street, a.HandName)
	fmt.Printf("equity %.0f%%, pot odds %.0f%%, needed %.0f%%\n", 100*a.Equity, 100*a.PotOdds, 100*a.Needed)
	fmt.Println(a.Reason)
	// Output:
	// call $10.00 (5 BB) flop high card
	// equity 44%, pot odds 20%, needed 22%
	// you need 22% (pot odds 20%, adjusted for implied odds and early position) and win about 44%
}
