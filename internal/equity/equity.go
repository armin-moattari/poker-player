// Package equity estimates how often each hand wins a Texas Hold'em pot.
//
// Equity is a hand's share of the pot on average: 1.0 means it always wins,
// and a two-way split counts as half a win. Calculate takes the known hole
// cards, any board cards already dealt, and how many extra opponents hold
// unknown (random) cards, and returns each known hand's equity.
//
// Two methods are available:
//
//   - Exact enumeration deals every possible remaining board and counts the
//     results. It gives the true answer and is used automatically when all
//     hands are known and the number of boards is small enough (even preflop
//     heads-up is only 1.7 million boards).
//   - Monte Carlo simulation deals random boards (and random opponent hands)
//     many times and averages the results. It is used when there are random
//     opponents, since enumerating their hands too would take far too long.
//     Its error shrinks with the square root of the trial count: 100,000
//     trials gives answers within about 0.3 percentage points.
//
// Both methods split the work across all CPU cores with goroutines, and both
// are deterministic: the same Request (including Seed) always gives the same
// Result, however many cores the machine has.
package equity

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/armin-moattari/poker-player/internal/card"
)

// Method selects how Calculate computes equity.
type Method int

const (
	// Auto uses Exact when there are no random opponents and at most
	// MaxExactBoards boards to check, and MonteCarlo otherwise.
	Auto Method = iota
	// Exact enumerates every possible remaining board.
	Exact
	// MonteCarlo samples random boards and opponent hands.
	MonteCarlo
)

const (
	// DefaultTrials is the Monte Carlo trial count used when Request.Trials
	// is zero.
	DefaultTrials = 100_000
	// MaxExactBoards is the most boards Auto will enumerate exactly.
	// Preflop with two known hands is C(48,5) = 1,712,304 boards, which
	// takes well under a second across all cores.
	MaxExactBoards = 2_000_000
	// MaxPlayers is the largest table supported, known hands plus random
	// opponents.
	MaxPlayers = 10
)

// Request describes the spot to evaluate.
type Request struct {
	// Hands holds the known hole cards, two cards per hand. The results
	// come back in the same order. At least one hand is required.
	Hands [][]card.Card
	// Board holds the community cards dealt so far: 0 (preflop), 3 (flop),
	// 4 (turn) or 5 (river).
	Board []card.Card
	// Opponents is the number of extra players whose hole cards are
	// unknown. Each trial deals them random cards.
	Opponents int
	// Method picks exact enumeration or Monte Carlo. The zero value, Auto,
	// chooses for you.
	Method Method
	// Trials is the number of Monte Carlo deals. Zero means DefaultTrials.
	// It is ignored by exact enumeration.
	Trials int
	// Seed makes Monte Carlo results repeatable: the same seed gives the
	// same result. Use a different seed for an independent estimate.
	Seed uint64
	// Workers is the number of goroutines to use. Zero means one per CPU
	// core (runtime.GOMAXPROCS). The result does not depend on it.
	Workers int
}

// HandResult is the outcome for one known hand.
type HandResult struct {
	// Equity is the hand's average share of the pot, from 0 to 1. A win
	// counts 1 and a split between k players counts 1/k.
	Equity float64
	// Win is the fraction of deals the hand wins outright.
	Win float64
	// Tie is the fraction of deals where the hand splits the pot.
	Tie float64
}

// Result is the outcome of Calculate.
type Result struct {
	// Hands has one entry per Request.Hands, in the same order.
	Hands []HandResult
	// Deals is the number of boards evaluated: every possible board for
	// exact enumeration, or the trial count for Monte Carlo.
	Deals int
	// Exact reports whether the result came from exact enumeration (true)
	// or Monte Carlo sampling (false).
	Exact bool
}

// Calculate computes the equity of each known hand in req.
//
// It returns an error if the request is invalid (wrong card counts,
// duplicate cards, too many players) or if ctx is cancelled before the work
// finishes, which lets a GUI abandon a calculation the user no longer needs.
func Calculate(ctx context.Context, req Request) (Result, error) {
	if err := validate(req); err != nil {
		return Result{}, err
	}
	workers := req.Workers
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	s := newSpot(req)

	method := req.Method
	if method == Auto {
		method = MonteCarlo
		if req.Opponents == 0 && binomial(len(s.deck), s.missing) <= MaxExactBoards {
			method = Exact
		}
	}

	var t tally
	var err error
	switch method {
	case Exact:
		if req.Opponents > 0 {
			return Result{}, errors.New("equity: exact enumeration needs every hand known (Opponents must be 0)")
		}
		t, err = s.enumerate(ctx, workers)
	case MonteCarlo:
		trials := req.Trials
		if trials <= 0 {
			trials = DefaultTrials
		}
		t, err = s.simulate(ctx, workers, trials, req.Seed)
	default:
		return Result{}, fmt.Errorf("equity: unknown method %d", method)
	}
	if err != nil {
		return Result{}, err
	}
	return t.result(method == Exact), nil
}

// validate checks the request before any work starts, so the hot loops can
// assume the input is sane.
func validate(req Request) error {
	if len(req.Hands) == 0 {
		return errors.New("equity: need at least one known hand")
	}
	if req.Opponents < 0 {
		return errors.New("equity: Opponents cannot be negative")
	}
	if players := len(req.Hands) + req.Opponents; players < 2 || players > MaxPlayers {
		return fmt.Errorf("equity: need 2 to %d players, got %d", MaxPlayers, players)
	}
	switch len(req.Board) {
	case 0, 3, 4, 5:
	default:
		return fmt.Errorf("equity: board must have 0, 3, 4 or 5 cards, got %d", len(req.Board))
	}
	// seen is a bitmask of every card used so far, to catch duplicates
	// across hands and board (the same card can't be dealt twice).
	var seen uint64
	use := func(c card.Card) error {
		if c >= card.NumCards {
			return fmt.Errorf("equity: invalid card %d", c)
		}
		if seen&(1<<c) != 0 {
			return fmt.Errorf("equity: card %s is used twice", c)
		}
		seen |= 1 << c
		return nil
	}
	for i, h := range req.Hands {
		if len(h) != 2 {
			return fmt.Errorf("equity: hand %d has %d cards, want 2", i+1, len(h))
		}
		for _, c := range h {
			if err := use(c); err != nil {
				return err
			}
		}
	}
	for _, c := range req.Board {
		if err := use(c); err != nil {
			return err
		}
	}
	return nil
}

// spot is a validated request prepared for the hot loops.
type spot struct {
	hands     [][2]card.Card // known hole cards
	board     []card.Card    // known board cards
	opponents int            // random opponents
	missing   int            // board cards still to come
	deck      []card.Card    // every card not in a hand or on the board
}

func newSpot(req Request) *spot {
	s := &spot{
		board:     req.Board,
		opponents: req.Opponents,
		missing:   5 - len(req.Board),
	}
	var used uint64
	for _, h := range req.Hands {
		s.hands = append(s.hands, [2]card.Card{h[0], h[1]})
		used |= 1<<h[0] | 1<<h[1]
	}
	for _, c := range req.Board {
		used |= 1 << c
	}
	for c := card.Card(0); c < card.NumCards; c++ {
		if used&(1<<c) == 0 {
			s.deck = append(s.deck, c)
		}
	}
	return s
}

// tally accumulates results for the known hands. Each worker fills its own
// tally, so there is no locking in the hot loop; tallies are merged at the
// end in a fixed order, which keeps floating-point sums deterministic.
type tally struct {
	deals int
	win   []int     // outright wins per known hand
	tie   []int     // splits per known hand
	share []float64 // total pot share per known hand
}

func newTally(hands int) tally {
	return tally{win: make([]int, hands), tie: make([]int, hands), share: make([]float64, hands)}
}

// add merges o into t.
func (t *tally) add(o tally) {
	t.deals += o.deals
	for i := range t.win {
		t.win[i] += o.win[i]
		t.tie[i] += o.tie[i]
		t.share[i] += o.share[i]
	}
}

func (t tally) result(exact bool) Result {
	r := Result{Deals: t.deals, Exact: exact, Hands: make([]HandResult, len(t.win))}
	n := float64(t.deals)
	for i := range r.Hands {
		r.Hands[i] = HandResult{
			Equity: t.share[i] / n,
			Win:    float64(t.win[i]) / n,
			Tie:    float64(t.tie[i]) / n,
		}
	}
	return r
}

// binomial returns n choose k, the number of ways to pick k items from n.
func binomial(n, k int) int {
	if k < 0 || k > n {
		return 0
	}
	r := 1
	for i := 1; i <= k; i++ {
		r = r * (n - k + i) / i // stays an integer at every step
	}
	return r
}
