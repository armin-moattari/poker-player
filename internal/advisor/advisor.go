// Package advisor recommends a poker action for your current spot.
//
// You describe the spot (your two cards, the board, the pot, what it costs
// to call, your stack, how many players are left, your position) and Advise
// returns an Advice: fold, check, call, bet or raise, how much to put in,
// and the numbers and reasoning behind it. A Config holds every setting a
// user can customize, from bet sizing to units and chip sizes.
//
// The package does no printing and no flag parsing. It takes plain values
// and returns a plain struct, so the command-line tool and the web GUI are
// both thin front ends over the same logic.
//
// The strategy is deliberately simple and explainable. It estimates your
// equity against random opponent hands, then:
//
//   - bets or raises when your equity is well above a fair share of the pot
//     (Config.StrongMultiple times 1/players, capped at Config.StrongCap);
//   - otherwise checks if it's free;
//   - otherwise calls if your equity beats the equity a call needs: the pot
//     odds, lowered by implied odds before the river;
//   - otherwise folds.
//
// Position shifts both thresholds: early position needs more, late
// position needs less (Config.PositionEdge).
//
// It does not model how opponents actually play (real opponents don't hold
// random hands) or their stack sizes, so treat it as a baseline.
package advisor

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/armin-moattari/poker-player/internal/card"
	"github.com/armin-moattari/poker-player/internal/equity"
	"github.com/armin-moattari/poker-player/internal/eval"
)

// Action is what the advisor recommends you do.
type Action int

// Possible actions. Bet means putting chips in when nobody has yet; Raise
// means putting in more than the current bet.
const (
	Fold Action = iota
	Check
	Call
	Bet
	Raise
)

// String returns the action in lowercase, such as "raise".
func (a Action) String() string {
	switch a {
	case Fold:
		return "fold"
	case Check:
		return "check"
	case Call:
		return "call"
	case Bet:
		return "bet"
	case Raise:
		return "raise"
	}
	return "unknown"
}

// Street is the betting round, set by how many board cards are out.
type Street int

// Betting rounds in order.
const (
	Preflop Street = iota // no board cards
	Flop                  // 3 board cards
	Turn                  // 4 board cards
	River                 // 5 board cards
)

// String returns the street in lowercase, such as "flop".
func (s Street) String() string {
	switch s {
	case Preflop:
		return "preflop"
	case Flop:
		return "flop"
	case Turn:
		return "turn"
	case River:
		return "river"
	}
	return "unknown"
}

// Position is roughly where you act in the betting round. Late position
// (acting last, or nearly) is an advantage: you see what others do first,
// and fewer players can raise after you.
type Position int

// Positions. The zero value, PositionUnknown, ignores position.
const (
	PositionUnknown Position = iota
	Early                    // many players still to act after you
	Middle                   // some players still to act after you
	Late                     // you act last or nearly last
)

// String returns the position in lowercase, such as "late", or "" for
// PositionUnknown.
func (p Position) String() string {
	switch p {
	case Early:
		return "early"
	case Middle:
		return "middle"
	case Late:
		return "late"
	}
	return ""
}

// ParsePosition parses "early", "middle" or "late" (any case), and "" as
// PositionUnknown.
func ParsePosition(s string) (Position, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return PositionUnknown, nil
	case "early":
		return Early, nil
	case "middle":
		return Middle, nil
	case "late":
		return Late, nil
	}
	return PositionUnknown, fmt.Errorf("advisor: unknown position %q (want early, middle or late)", s)
}

// Spot describes your situation at the table. Amounts can be in any unit
// (chips, dollars, big blinds) as long as they all use the same one.
type Spot struct {
	// Hand is your two hole cards.
	Hand []card.Card
	// Board is the community cards so far: 0, 3, 4 or 5. It also sets the
	// street.
	Board []card.Card
	// Pot is everything already in the middle, including bets made in this
	// betting round.
	Pot float64
	// ToCall is how much you must put in to call; 0 if you can check.
	ToCall float64
	// Stack is what you have behind. It caps calls and raises. Zero means
	// "unknown", and the advisor ignores stack size.
	Stack float64
	// Players is how many players are still in the hand, you included.
	// Zero means 2 (heads-up).
	Players int
	// Position is where you act this round. Leave it unset to ignore
	// position.
	Position Position
	// Seed makes the equity estimate, and so the advice, repeatable.
	Seed uint64
}

// Advice is the recommendation and everything that went into it.
type Advice struct {
	// Action is what to do.
	Action Action
	// Amount is how much to put in now: the call amount for Call, the total
	// bet for Bet and Raise, and 0 for Fold and Check. Bets are rounded to
	// Config.ChipStep.
	Amount float64
	// AllIn reports whether Amount is your whole stack.
	AllIn bool
	// Street is the current betting round.
	Street Street
	// HandName describes your hand: shorthand like "AKs" or "77" preflop,
	// and the made hand like "two pair" after the flop.
	HandName string
	// Equity is your average share of the pot against random hands, 0 to 1.
	Equity float64
	// FairShare is 1/players: the equity an average hand would have.
	FairShare float64
	// PotOdds is the plain pot odds: the equity a call needs to break even
	// counting only the current pot. 0 when there is nothing to call.
	PotOdds float64
	// Needed is the equity the advisor actually required to call, after
	// implied odds and position. 0 when there is nothing to call.
	Needed float64
	// BetThreshold is the equity the advisor required to bet or raise,
	// after position.
	BetThreshold float64
	// Reason explains the recommendation in one sentence.
	Reason string
}

// Advise recommends an action for spot using the settings in cfg (start
// from DefaultConfig). It returns an error if the spot or settings are
// invalid, or if ctx is cancelled while estimating equity.
func Advise(ctx context.Context, spot Spot, cfg Config) (Advice, error) {
	players := spot.Players
	if players == 0 {
		players = 2
	}
	if err := validate(spot, players); err != nil {
		return Advice{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Advice{}, err
	}

	// Estimate equity against the other players, each holding a random
	// hand. Monte Carlo is always used here, because unknown opponents
	// make exact enumeration far too slow.
	res, err := equity.Calculate(ctx, equity.Request{
		Hands:     [][]card.Card{spot.Hand},
		Board:     spot.Board,
		Opponents: players - 1,
		Trials:    cfg.Trials,
		Seed:      spot.Seed,
	})
	if err != nil {
		return Advice{}, err
	}

	a := Advice{
		Street:    Street(max(len(spot.Board)-2, 0)), // 0, 3, 4, 5 cards -> 0, 1, 2, 3
		HandName:  handName(spot.Hand, spot.Board),
		Equity:    res.Hands[0].Equity,
		FairShare: 1 / float64(players),
	}

	// Position moves both thresholds by the same amount: up in early
	// position, where players behind can still raise, and down in late.
	edge := 0.0
	switch spot.Position {
	case Early:
		edge = cfg.PositionEdge
	case Late:
		edge = -cfg.PositionEdge
	}
	a.BetThreshold = clamp01(min(cfg.StrongMultiple*a.FairShare, cfg.StrongCap) + edge)

	// You can never put in more than you have, so a call for more than
	// your stack is really an all-in call for the stack.
	callCost := spot.ToCall
	if spot.Stack > 0 {
		callCost = min(callCost, spot.Stack)
	}
	implied := 0.0
	if callCost > 0 {
		// Pot odds: the share of the final pot your call makes up. Winning
		// at least that often makes the call break even in the long run.
		a.PotOdds = callCost / (spot.Pot + callCost)
		// Implied odds: before the river, count chips you expect to win
		// later as if they were already in the pot.
		if a.Street != River {
			implied = cfg.ImpliedOdds * spot.Pot
		}
		a.Needed = clamp01(callCost/(spot.Pot+callCost+implied) + edge)
	}

	canRaise := spot.Stack == 0 || spot.Stack > callCost
	switch {
	case a.Equity >= a.BetThreshold && canRaise:
		a.Amount, a.AllIn = betSize(spot, cfg, a.Street, callCost)
		a.Action = Bet
		if callCost > 0 {
			a.Action = Raise
		}
		opponents := players - 1
		a.Reason = fmt.Sprintf("you win %s against %d %s, above the %s needed to bet%s; build the pot",
			pct(a.Equity), opponents, plural(opponents, "opponent", "opponents"),
			pct(a.BetThreshold), positionNote(spot.Position, cfg))
	case callCost == 0:
		a.Action = Check
		a.Reason = fmt.Sprintf("you win %s, below the %s needed to bet%s; check and see the next card for free",
			pct(a.Equity), pct(a.BetThreshold), positionNote(spot.Position, cfg))
	case a.Equity >= a.Needed:
		a.Action, a.Amount = Call, callCost
		a.AllIn = spot.Stack > 0 && callCost == spot.Stack
		a.Reason = fmt.Sprintf("you need %s%s and win about %s",
			pct(a.Needed), neededNote(a, implied, spot.Position, cfg), pct(a.Equity))
	default:
		a.Action = Fold
		a.Reason = fmt.Sprintf("calling needs %s%s but you only win about %s",
			pct(a.Needed), neededNote(a, implied, spot.Position, cfg), pct(a.Equity))
	}
	return a, nil
}

// betSize works out how much to put in for a bet or raise, and whether that
// is all-in.
//
// Preflop with a known big blind it uses the usual "raise to 3x" style:
// PreflopRaise times the current bet (at least one big blind). Otherwise it
// calls first and then adds BetFraction of the pot as it would be after the
// call. Either way the size is rounded to the chip step, is always at least
// one chip step more than a call, and never exceeds the stack.
func betSize(spot Spot, cfg Config, street Street, callCost float64) (float64, bool) {
	var size float64
	if street == Preflop && cfg.BigBlind > 0 {
		size = cfg.PreflopRaise * max(callCost, cfg.BigBlind)
	} else {
		size = callCost + cfg.BetFraction*(spot.Pot+callCost)
	}
	size = max(cfg.roundToChip(size), cfg.roundToChip(callCost+cfg.ChipStep))
	if spot.Stack > 0 && size >= spot.Stack {
		return spot.Stack, true
	}
	return size, false
}

// neededNote explains how the equity needed to call differs from the plain
// pot odds, or returns "" when it doesn't.
func neededNote(a Advice, implied float64, pos Position, cfg Config) string {
	var why []string
	if implied > 0 {
		why = append(why, "implied odds")
	}
	if pos != PositionUnknown && pos != Middle && cfg.PositionEdge > 0 {
		why = append(why, pos.String()+" position")
	}
	if len(why) == 0 {
		return ""
	}
	return fmt.Sprintf(" (pot odds %s, adjusted for %s)", pct(a.PotOdds), strings.Join(why, " and "))
}

// positionNote mentions position when it moved the bet threshold.
func positionNote(pos Position, cfg Config) string {
	if pos == PositionUnknown || pos == Middle || cfg.PositionEdge == 0 {
		return ""
	}
	return " in " + pos.String() + " position"
}

// validate rejects spots the advisor can't reason about. Card-level checks
// (duplicates, invalid cards) are left to equity.Calculate, which already
// does them.
func validate(spot Spot, players int) error {
	if len(spot.Hand) != 2 {
		return fmt.Errorf("advisor: need exactly 2 hole cards, got %d", len(spot.Hand))
	}
	switch len(spot.Board) {
	case 0, 3, 4, 5:
	default:
		return fmt.Errorf("advisor: board must have 0, 3, 4 or 5 cards, got %d", len(spot.Board))
	}
	if players < 2 || players > equity.MaxPlayers {
		return fmt.Errorf("advisor: need 2 to %d players in the hand, got %d", equity.MaxPlayers, players)
	}
	if spot.Pot < 0 || spot.ToCall < 0 || spot.Stack < 0 {
		return errors.New("advisor: pot, to-call and stack can't be negative")
	}
	if spot.Position < PositionUnknown || spot.Position > Late {
		return fmt.Errorf("advisor: invalid position %d", spot.Position)
	}
	return nil
}

// handName describes the hand: hole-card shorthand preflop, and the best
// made hand (such as "two pair") once there is a board.
func handName(hand, board []card.Card) string {
	if len(board) == 0 {
		return HoleName(hand)
	}
	return eval.Evaluate(append(append([]card.Card(nil), hand...), board...)).Category().String()
}

// HoleName returns the usual shorthand for two hole cards: the higher rank
// first, then "s" if they share a suit or "o" (offsuit) if not, as in "AKs"
// or "QJo". Pairs have no suffix, as in "77". It returns "" unless given
// exactly two cards.
func HoleName(hand []card.Card) string {
	if len(hand) != 2 {
		return ""
	}
	hi, lo := hand[0], hand[1]
	if lo.Rank() > hi.Rank() {
		hi, lo = lo, hi
	}
	name := hi.Rank().String() + lo.Rank().String()
	switch {
	case hi.Rank() == lo.Rank():
		return name
	case hi.Suit() == lo.Suit():
		return name + "s"
	default:
		return name + "o"
	}
}

// pct formats a 0-1 fraction as a whole percentage, such as "39%".
func pct(f float64) string { return fmt.Sprintf("%.0f%%", 100*f) }

func clamp01(f float64) float64 { return math.Max(0, math.Min(1, f)) }

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
