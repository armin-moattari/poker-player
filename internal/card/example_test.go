package card_test

import (
	"fmt"
	"math/rand/v2"

	"github.com/armin-moattari/poker-player/internal/card"
)

// Examples are compiled and run by go test, and the // Output: comment is
// checked, so the usage shown here (and copied into the README) cannot go
// stale. They also appear in go doc output.

func ExampleParseCards() {
	hand, err := card.ParseCards("As Kd")
	if err != nil {
		panic(err)
	}
	fmt.Println(hand, hand[0].Rank(), hand[0].Suit())
	// Output: [As Kd] A s
}

func ExampleDeck() {
	hand, _ := card.ParseCards("As Kd")
	d := card.NewDeck()
	_ = d.Remove(hand...)                  // take known cards out
	d.Shuffle(rand.New(rand.NewPCG(1, 2))) // seeded, so the same every run
	board, _ := d.Deal(5)
	fmt.Println(len(board), d.Len())
	// Output: 5 45
}
