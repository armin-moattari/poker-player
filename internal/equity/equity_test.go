package equity

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/armin-moattari/poker-player/internal/card"
)

// hands parses "AsAh KcKd" into known hands, two cards each.
func hands(t *testing.T, s string) [][]card.Card {
	t.Helper()
	var hs [][]card.Card
	for _, f := range strings.Fields(s) {
		h, err := card.ParseCards(f)
		if err != nil || len(h) != 2 {
			t.Fatalf("bad test hand %q: %v", f, err)
		}
		hs = append(hs, h)
	}
	return hs
}

// board parses board cards, allowing "" for preflop.
func board(t *testing.T, s string) []card.Card {
	t.Helper()
	b, err := card.ParseCards(s)
	if err != nil {
		t.Fatalf("bad test board %q: %v", s, err)
	}
	return b
}

func calc(t *testing.T, req Request) Result {
	t.Helper()
	r, err := Calculate(context.Background(), req)
	if err != nil {
		t.Fatalf("Calculate: %v", err)
	}
	return r
}

func near(got, want, tol float64) bool { return math.Abs(got-want) <= tol }

func TestRiverIsDecided(t *testing.T) {
	r := calc(t, Request{Hands: hands(t, "AsAh KcKd"), Board: board(t, "2c 7d 9h Js 3s")})
	if !r.Exact || r.Deals != 1 {
		t.Errorf("river: Exact=%v Deals=%d, want true 1", r.Exact, r.Deals)
	}
	if r.Hands[0].Equity != 1 || r.Hands[1].Equity != 0 {
		t.Errorf("river: equities %v, want 1 and 0", r.Hands)
	}
}

func TestBoardPlaysSplitsPot(t *testing.T) {
	r := calc(t, Request{Hands: hands(t, "2c3d 4h5s"), Board: board(t, "Ac Kd Qh Js Tc")})
	for i, h := range r.Hands {
		if h.Equity != 0.5 || h.Tie != 1 || h.Win != 0 {
			t.Errorf("hand %d: %+v, want equity 0.5 and a tie every time", i, h)
		}
	}
}

// TestFlushDrawOuts checks a spot whose answer can be counted by hand. On the
// turn, 7h6h needs a heart against AcAd: 9 hearts are left among the 44
// unseen cards, any of them wins, and nothing else does.
func TestFlushDrawOuts(t *testing.T) {
	r := calc(t, Request{Hands: hands(t, "7h6h AcAd"), Board: board(t, "2h 9h Ks Qd")})
	if !r.Exact || r.Deals != 44 {
		t.Fatalf("Exact=%v Deals=%d, want exact over 44 rivers", r.Exact, r.Deals)
	}
	if want := 9.0 / 44; !near(r.Hands[0].Equity, want, 1e-12) {
		t.Errorf("flush draw equity = %v, want 9/44 = %v", r.Hands[0].Equity, want)
	}
}

func TestEquitiesSumToOne(t *testing.T) {
	for _, tt := range []struct{ hands, board string }{
		{"AsAh KcKd", ""},
		{"AsKs QhQd 7c6c", "Qs 8c 2d"},
		{"Ah2h KcKd", "3h 4h 9c Td"},
	} {
		r := calc(t, Request{Hands: hands(t, tt.hands), Board: board(t, tt.board)})
		sum := 0.0
		for _, h := range r.Hands {
			sum += h.Equity
		}
		if !near(sum, 1, 1e-9) {
			t.Errorf("%s on %q: equities sum to %v, want 1", tt.hands, tt.board, sum)
		}
	}
}

// TestKnownPreflopOdds checks exact preflop matchups against published
// exact enumerations (DeucesCracked, "Preflop poker equity matchups: exact
// data", which also enumerates all 1,712,304 boards). The published win and
// tie percentages are rounded to two decimals, hence the tolerance.
func TestKnownPreflopOdds(t *testing.T) {
	if testing.Short() {
		t.Skip("enumerates 1.7M boards per matchup")
	}
	tests := []struct {
		hands    string
		win, tie float64 // first hand, as published
		why      string
	}{
		{"AhAs KdKc", 0.8106, 0.0038, "aces vs kings"},
		{"AhAs KhKs", 0.8236, 0.0054, "kings that share the aces' suits do worse"},
		{"QhQs AdKc", 0.5699, 0.0034, "the classic coin flip"},
		{"AhAs 7d2c", 0.8724, 0.0036, "best vs worst starting hand"},
		{"2h2s AdKd", 0.4960, 0.0063, "small pair vs suited ace-king"},
	}
	for _, tt := range tests {
		r := calc(t, Request{Hands: hands(t, tt.hands)})
		if !r.Exact || r.Deals != 1712304 {
			t.Errorf("%s: Exact=%v Deals=%d, want exact over 1712304 boards", tt.hands, r.Exact, r.Deals)
		}
		h := r.Hands[0]
		if !near(h.Win, tt.win, 0.00006) || !near(h.Tie, tt.tie, 0.00006) {
			t.Errorf("%s (%s): win %.4f tie %.4f, want %.4f %.4f", tt.hands, tt.why, h.Win, h.Tie, tt.win, tt.tie)
		}
	}
}

// TestMonteCarloMatchesExact runs the same spot both ways. With 400,000
// trials the Monte Carlo standard error is under 0.1 percentage point, so
// 0.4 points is a wide (four-sigma) margin.
func TestMonteCarloMatchesExact(t *testing.T) {
	req := Request{Hands: hands(t, "AsKs QhQd 7c6c"), Board: board(t, "Qs 8c 2d")}
	exact := calc(t, req)
	req.Method, req.Trials = MonteCarlo, 400_000
	mc := calc(t, req)
	if mc.Exact || mc.Deals != 400_000 {
		t.Fatalf("Monte Carlo: Exact=%v Deals=%d", mc.Exact, mc.Deals)
	}
	for i := range exact.Hands {
		if !near(mc.Hands[i].Equity, exact.Hands[i].Equity, 0.004) {
			t.Errorf("hand %d: Monte Carlo %.4f, exact %.4f", i, mc.Hands[i].Equity, exact.Hands[i].Equity)
		}
	}
}

// TestDeterministicAcrossWorkers checks that the result depends only on the
// request and seed, not on how many goroutines did the work.
func TestDeterministicAcrossWorkers(t *testing.T) {
	base := Request{Hands: hands(t, "AsKd"), Opponents: 3, Trials: 50_000, Seed: 42}
	var first Result
	for i, w := range []int{1, 3, 8} {
		req := base
		req.Workers = w
		r := calc(t, req)
		if i == 0 {
			first = r
			continue
		}
		if r.Hands[0] != first.Hands[0] {
			t.Errorf("Workers=%d gave %+v, Workers=1 gave %+v", w, r.Hands[0], first.Hands[0])
		}
	}
	req := base
	req.Seed = 43
	if r := calc(t, req); r.Hands[0] == first.Hands[0] {
		t.Error("a different seed gave the identical result")
	}
}

// TestMultiway checks equity against random opponents. Pocket aces win
// about 85.2% against one random hand (a widely published figure), and
// every extra opponent should lower that.
func TestMultiway(t *testing.T) {
	aces := hands(t, "AsAh")
	prev := 1.0
	for opp := 1; opp <= 4; opp++ {
		r := calc(t, Request{Hands: aces, Opponents: opp, Trials: 200_000, Seed: 1})
		eq := r.Hands[0].Equity
		if r.Exact {
			t.Errorf("%d opponents: used exact enumeration, want Monte Carlo", opp)
		}
		if opp == 1 && !near(eq, 0.852, 0.005) {
			t.Errorf("aces vs 1 random hand: %.4f, want about 0.852", eq)
		}
		if eq >= prev {
			t.Errorf("aces vs %d opponents: %.4f, not below %.4f with one fewer", opp, eq, prev)
		}
		prev = eq
	}
}

func TestCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, m := range []Method{Exact, MonteCarlo} {
		_, err := Calculate(ctx, Request{Hands: hands(t, "AsAh KcKd"), Method: m})
		if err != context.Canceled {
			t.Errorf("method %d: err = %v, want context.Canceled", m, err)
		}
	}
}

func TestRequestErrors(t *testing.T) {
	aces := hands(t, "AsAh")
	tests := map[string]Request{
		"no hands":           {Opponents: 1},
		"only one player":    {Hands: aces},
		"too many players":   {Hands: aces, Opponents: MaxPlayers},
		"negative opponents": {Hands: hands(t, "AsAh KcKd"), Opponents: -1},
		"one-card hand":      {Hands: [][]card.Card{{card.MustParse("As")}}, Opponents: 1},
		"two-card board":     {Hands: aces, Opponents: 1, Board: board(t, "2c 3d")},
		"duplicate in hands": {Hands: hands(t, "AsAh AsKd")},
		"hand card on board": {Hands: aces, Opponents: 1, Board: board(t, "As 2c 3d")},
		"invalid card":       {Hands: [][]card.Card{{card.MustParse("As"), 60}}, Opponents: 1},
		"exact with randoms": {Hands: aces, Opponents: 1, Method: Exact},
		"unknown method":     {Hands: aces, Opponents: 1, Method: Method(9)},
	}
	for name, req := range tests {
		if _, err := Calculate(context.Background(), req); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestBinomial(t *testing.T) {
	tests := []struct{ n, k, want int }{
		{48, 5, 1712304}, {44, 1, 44}, {52, 0, 1}, {5, 6, 0}, {5, -1, 0},
	}
	for _, tt := range tests {
		if got := binomial(tt.n, tt.k); got != tt.want {
			t.Errorf("binomial(%d, %d) = %d, want %d", tt.n, tt.k, got, tt.want)
		}
	}
}
