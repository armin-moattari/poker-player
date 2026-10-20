package eval_test

import (
	"fmt"

	"github.com/armin-moattari/poker-player/internal/card"
	"github.com/armin-moattari/poker-player/internal/eval"
)

// Examples are compiled and run by go test, and the // Output: comment is
// checked, so the usage shown here (and copied into the README) cannot go
// stale.

func ExampleEvaluate() {
	board, _ := card.ParseCards("Ks 7h 7d 2c 9s")
	alice, _ := card.ParseCards("Kd Kc")
	bob, _ := card.ParseCards("As 7c")

	a := eval.Evaluate(append(alice, board...))
	b := eval.Evaluate(append(bob, board...))
	fmt.Println(a)
	fmt.Println(b)
	fmt.Println(a > b, a.Category())
	// Output:
	// full house (K 7)
	// three of a kind (7 A K)
	// true full house
}
