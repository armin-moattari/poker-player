package equity

import (
	"context"
	"fmt"
	"runtime"
	"testing"

	"github.com/armin-moattari/poker-player/internal/card"
)

// BenchmarkEquity measures the two methods on one core and on every core,
// which shows how well the goroutine split scales. Run it with:
//
//	go test -bench=. ./internal/equity
func BenchmarkEquity(b *testing.B) {
	aces, _ := card.ParseCards("AhAs")
	kings, _ := card.ParseCards("KdKc")
	cases := []struct {
		name string
		req  Request
	}{
		{"exact/preflop-heads-up", Request{Hands: [][]card.Card{aces, kings}, Method: Exact}},
		{"montecarlo/100k-vs-3-random", Request{Hands: [][]card.Card{aces}, Opponents: 3, Method: MonteCarlo, Trials: 100_000}},
	}
	for _, c := range cases {
		for _, w := range []int{1, runtime.GOMAXPROCS(0)} {
			b.Run(fmt.Sprintf("%s/%dcores", c.name, w), func(b *testing.B) {
				req := c.req
				req.Workers = w
				for b.Loop() {
					if _, err := Calculate(context.Background(), req); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
