package advisor

import (
	"context"
	"strings"
	"testing"

	"github.com/armin-moattari/poker-player/internal/card"
)

func cards(t *testing.T, s string) []card.Card {
	t.Helper()
	cs, err := card.ParseCards(s)
	if err != nil {
		t.Fatalf("bad test cards %q: %v", s, err)
	}
	return cs
}

// advise runs Advise with a fixed seed and fails the test on error.
func advise(t *testing.T, spot Spot, cfg Config) Advice {
	t.Helper()
	spot.Seed = 1
	a, err := Advise(context.Background(), spot, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// TestAdvise covers each branch of the strategy with default settings. The
// spots are chosen far from the decision thresholds, so Monte Carlo noise
// can't flip the answer.
func TestAdvise(t *testing.T) {
	tests := []struct {
		name   string
		spot   Spot
		action Action
		amount float64
		allIn  bool
	}{
		{
			name:   "aces bet when checked to",
			spot:   Spot{Hand: cards(t, "AsAh"), Pot: 3},
			action: Bet, amount: 2, // 0 to call + 0.75 * 3 = 2.25, rounded to 2
		},
		{
			name:   "aces raise facing a bet",
			spot:   Spot{Hand: cards(t, "AsAh"), Pot: 15, ToCall: 10},
			action: Raise, amount: 29, // 10 to call + 0.75 * 25 = 28.75, rounded to 29
		},
		{
			name:   "raise is capped at the stack",
			spot:   Spot{Hand: cards(t, "AsAh"), Pot: 15, ToCall: 10, Stack: 20},
			action: Raise, amount: 20, allIn: true,
		},
		{
			name:   "strong but can only call all-in",
			spot:   Spot{Hand: cards(t, "AsAh"), Pot: 15, ToCall: 50, Stack: 30},
			action: Call, amount: 30, allIn: true,
		},
		{
			name:   "weak hand checks when it's free",
			spot:   Spot{Hand: cards(t, "7h2c"), Pot: 3},
			action: Check,
		},
		{
			name:   "draw calls with good pot odds",
			spot:   Spot{Hand: cards(t, "AsKd"), Board: cards(t, "Qs Jh 2c"), Pot: 40, ToCall: 10, Players: 3},
			action: Call, amount: 10,
		},
		{
			name:   "junk folds to a huge bet",
			spot:   Spot{Hand: cards(t, "7h2c"), Pot: 10, ToCall: 100, Players: 3},
			action: Fold,
		},
		{
			name:   "the nuts on the river bets",
			spot:   Spot{Hand: cards(t, "Ts3c"), Board: cards(t, "As Ks Qs Js 2d"), Pot: 100},
			action: Bet, amount: 75,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := advise(t, tt.spot, DefaultConfig())
			if a.Action != tt.action || a.Amount != tt.amount || a.AllIn != tt.allIn {
				t.Errorf("got %v %v (all-in %v), want %v %v (all-in %v); equity %.3f, needed %.3f, reason %q",
					a.Action, a.Amount, a.AllIn, tt.action, tt.amount, tt.allIn, a.Equity, a.Needed, a.Reason)
			}
			if a.Reason == "" {
				t.Error("empty reason")
			}
		})
	}
}

func TestAdviceDetails(t *testing.T) {
	a := advise(t, Spot{Hand: cards(t, "AsKd"), Board: cards(t, "Qs Jh 2c"), Pot: 40, ToCall: 10, Players: 3}, DefaultConfig())
	if a.Street != Flop || a.HandName != "high card" {
		t.Errorf("street %v, hand %q; want flop, high card", a.Street, a.HandName)
	}
	if a.FairShare != 1.0/3 {
		t.Errorf("fair share %v, want 1/3", a.FairShare)
	}
	if a.PotOdds != 0.2 { // 10 / (40 + 10)
		t.Errorf("pot odds %v, want 0.2", a.PotOdds)
	}
	if want := 10.0 / (40 + 10 + 0.25*40); a.Needed != want { // implied odds add 25% of the pot
		t.Errorf("needed %v, want %v", a.Needed, want)
	}
	if a.Equity < 0.40 || a.Equity > 0.48 {
		t.Errorf("equity %.3f, want about 0.44", a.Equity)
	}
	// Players left at zero means heads-up.
	cfg := DefaultConfig()
	cfg.Trials = 1000
	if a := advise(t, Spot{Hand: cards(t, "AsKd")}, cfg); a.FairShare != 0.5 {
		t.Errorf("default players: fair share %v, want 0.5", a.FairShare)
	}
}

// TestImpliedOdds checks that implied odds lower the equity a call needs
// before the river, and are ignored on the river.
func TestImpliedOdds(t *testing.T) {
	off := DefaultConfig()
	off.ImpliedOdds = 0
	flop := Spot{Hand: cards(t, "9h8h"), Board: cards(t, "Kh 4c 2h"), Pot: 20, ToCall: 10}
	if a := advise(t, flop, off); a.Needed != a.PotOdds {
		t.Errorf("implied off: needed %v, want pot odds %v", a.Needed, a.PotOdds)
	}
	if a := advise(t, flop, DefaultConfig()); a.Needed >= a.PotOdds || !strings.Contains(a.Reason, "implied odds") {
		t.Errorf("implied on: needed %v should be below pot odds %v; reason %q", a.Needed, a.PotOdds, a.Reason)
	}
	river := Spot{Hand: cards(t, "9h8h"), Board: cards(t, "Kh 4c 2h 5s 7d"), Pot: 20, ToCall: 10}
	if a := advise(t, river, DefaultConfig()); a.Needed != a.PotOdds {
		t.Errorf("river: needed %v, want pot odds %v (no implied odds)", a.Needed, a.PotOdds)
	}
}

// TestPosition checks that early position raises both thresholds and late
// position lowers them, by Config.PositionEdge.
func TestPosition(t *testing.T) {
	spot := Spot{Hand: cards(t, "AsKd"), Board: cards(t, "Qs Jh 2c"), Pot: 40, ToCall: 10, Players: 3}
	base := advise(t, spot, DefaultConfig())
	for pos, sign := range map[Position]float64{Early: 1, Middle: 0, Late: -1} {
		spot.Position = pos
		a := advise(t, spot, DefaultConfig())
		if d := a.Needed - base.Needed; !near(d, sign*0.05) {
			t.Errorf("%v: needed moved by %v, want %v", pos, d, sign*0.05)
		}
		if d := a.BetThreshold - base.BetThreshold; !near(d, sign*0.05) {
			t.Errorf("%v: bet threshold moved by %v, want %v", pos, d, sign*0.05)
		}
	}
	// Late position can turn a check into a bet. QQ against three opponents
	// wins about 54%. With StrongMultiple 2.25 the bet threshold is 56%
	// (2.25 x 25%), and late position lowers it to 51%, so the hand sits
	// right between the two.
	qq := Spot{Hand: cards(t, "QhQd"), Pot: 3, Players: 4}
	cfg := DefaultConfig()
	cfg.StrongMultiple = 2.25
	if a := advise(t, qq, cfg); a.Action != Check {
		t.Fatalf("no position: %v with equity %.3f, threshold %.3f; want check", a.Action, a.Equity, a.BetThreshold)
	}
	qq.Position = Late
	if a := advise(t, qq, cfg); a.Action != Bet || !strings.Contains(a.Reason, "late position") {
		t.Errorf("late position: %v (%q), want a bet that mentions late position", a.Action, a.Reason)
	}
}

func near(a, b float64) bool { return a-b < 1e-9 && b-a < 1e-9 }

// TestBetSizing checks the big-blind preflop sizing and rounding to chips.
func TestBetSizing(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BigBlind = 2
	// Facing just the big blind: raise to 3 x 2 = 6.
	if a := advise(t, Spot{Hand: cards(t, "AsAh"), Pot: 3, ToCall: 2}, cfg); a.Action != Raise || a.Amount != 6 {
		t.Errorf("preflop open: %v %v, want raise 6", a.Action, a.Amount)
	}
	// Facing a raise to 8 (to call 8): 3 x 8 = 24.
	if a := advise(t, Spot{Hand: cards(t, "AsAh"), Pot: 11, ToCall: 8}, cfg); a.Amount != 24 {
		t.Errorf("preflop 3-bet: %v, want 24", a.Amount)
	}
	// Dollar game with $0.25 chips: 0.75 x $3.10 = $2.325, rounded to $2.25.
	cash := DefaultConfig()
	cash.Unit, cash.ChipStep = "$", 0.25
	if a := advise(t, Spot{Hand: cards(t, "AsAh"), Pot: 3.10}, cash); a.Amount != 2.25 {
		t.Errorf("rounded bet %v, want 2.25", a.Amount)
	}
	// A tiny pot still bets at least one chip step.
	big := DefaultConfig()
	big.ChipStep = 5
	if a := advise(t, Spot{Hand: cards(t, "AsAh"), Pot: 1}, big); a.Amount != 5 {
		t.Errorf("minimum bet %v, want one chip step (5)", a.Amount)
	}
}

func TestAdviseIsRepeatable(t *testing.T) {
	spot := Spot{Hand: cards(t, "9h8h"), Board: cards(t, "7h 6c 2h"), Pot: 30, ToCall: 15, Players: 4, Seed: 7}
	a, _ := Advise(context.Background(), spot, DefaultConfig())
	b, _ := Advise(context.Background(), spot, DefaultConfig())
	if a != b {
		t.Errorf("same seed gave different advice:\n%+v\n%+v", a, b)
	}
}

func TestStreetAndHandName(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Trials = 1000
	tests := []struct {
		hand, board string
		street      Street
		name        string
	}{
		{"AsKd", "", Preflop, "AKo"},
		{"KdAd", "", Preflop, "AKs"},
		{"7c7h", "", Preflop, "77"},
		{"AsKd", "Ah 7c 2d", Flop, "one pair"},
		{"AsKd", "Ah 7c 2d Kc", Turn, "two pair"},
		{"Ts3c", "As Ks Qs Js 2d", River, "straight flush"},
	}
	for _, tt := range tests {
		a := advise(t, Spot{Hand: cards(t, tt.hand), Board: cards(t, tt.board)}, cfg)
		if a.Street != tt.street || a.HandName != tt.name {
			t.Errorf("%s on %q: %v %q, want %v %q", tt.hand, tt.board, a.Street, a.HandName, tt.street, tt.name)
		}
	}
	if got := HoleName(cards(t, "As")); got != "" {
		t.Errorf("HoleName(one card) = %q, want empty", got)
	}
}

func TestAdviseErrors(t *testing.T) {
	aces := cards(t, "AsAh")
	tests := map[string]Spot{
		"one hole card":    {Hand: cards(t, "As")},
		"two-card board":   {Hand: aces, Board: cards(t, "2c 3d")},
		"one player":       {Hand: aces, Players: 1},
		"too many players": {Hand: aces, Players: 11},
		"negative pot":     {Hand: aces, Pot: -1},
		"negative to-call": {Hand: aces, ToCall: -1},
		"negative stack":   {Hand: aces, Stack: -1},
		"bad position":     {Hand: aces, Position: Position(9)},
		"card used twice":  {Hand: aces, Board: cards(t, "As 2c 3d")},
	}
	for name, spot := range tests {
		if _, err := Advise(context.Background(), spot, DefaultConfig()); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
	if _, err := Advise(context.Background(), Spot{Hand: aces}, Config{}); err == nil {
		t.Error("zero Config: no error")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Advise(ctx, Spot{Hand: aces}, DefaultConfig()); err == nil {
		t.Error("cancelled context: no error")
	}
}

func TestParsePosition(t *testing.T) {
	for in, want := range map[string]Position{"": PositionUnknown, "early": Early, " Middle ": Middle, "LATE": Late} {
		if got, err := ParsePosition(in); err != nil || got != want {
			t.Errorf("ParsePosition(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := ParsePosition("button"); err == nil {
		t.Error(`ParsePosition("button"): no error`)
	}
}

func TestStrings(t *testing.T) {
	for want, s := range map[string]interface{ String() string }{
		"fold": Fold, "check": Check, "call": Call, "bet": Bet, "raise": Raise, "unknown": Action(9),
		"preflop": Preflop, "flop": Flop, "turn": Turn, "river": River,
		"early": Early, "middle": Middle, "late": Late, "": PositionUnknown,
	} {
		if s.String() != want {
			t.Errorf("String() = %q, want %q", s.String(), want)
		}
	}
	if Street(9).String() != "unknown" {
		t.Errorf("Street(9).String() = %q", Street(9).String())
	}
}
