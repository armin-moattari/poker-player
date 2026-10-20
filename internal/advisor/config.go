package advisor

import (
	"errors"
	"fmt"
	"math"
	"strconv"
)

// Config holds every setting a user can customize. Start from
// DefaultConfig and change what you need; front ends can save it as JSON
// (the field tags give the names used in the settings file).
//
// All amounts, here and in Spot, are in the same unit as each other: chips,
// dollars, big blinds, whatever is convenient at your table. The advisor only
// compares amounts to each other, so the unit never changes the decision;
// Unit and ChipStep only affect how amounts are rounded and shown.
type Config struct {
	// Unit labels amounts in descriptions. "$" prints amounts as dollars
	// ("$12.50"); anything else is used as a word after the number
	// ("25 chips"). Default "chips".
	Unit string `json:"unit"`
	// ChipStep is the smallest amount you can bet, such as 1 chip or $0.25.
	// Suggested bets are rounded to a multiple of it. Default 1.
	ChipStep float64 `json:"chip_step"`
	// BigBlind is the size of the big blind, or 0 if you don't want to
	// enter it. When set, amounts are also shown in big blinds and preflop
	// raises use PreflopRaise instead of a fraction of the pot.
	BigBlind float64 `json:"big_blind"`

	// StrongMultiple is how many times a fair share of the pot (1/players)
	// your equity must reach to bet or raise for value. Default 1.4.
	StrongMultiple float64 `json:"strong_multiple"`
	// StrongCap caps that threshold so heads-up spots still bet clearly
	// winning hands (1.4 x 50% would be 70%). Default 0.9.
	StrongCap float64 `json:"strong_cap"`
	// BetFraction sizes bets and raises after the flop as a fraction of the
	// pot after calling. Default 0.75.
	BetFraction float64 `json:"bet_fraction"`
	// PreflopRaise sizes preflop raises as a multiple of the current bet
	// (at least the big blind) when BigBlind is set. Default 3.
	PreflopRaise float64 `json:"preflop_raise"`

	// PositionEdge is how much equity position is worth, from 0 to 0.5.
	// In early position (many players still to act and able to raise
	// behind you) both the bet and the call thresholds go up by this much;
	// in late position they go down by it. 0 turns position off. Default
	// 0.05 (5 percentage points).
	PositionEdge float64 `json:"position_edge"`
	// ImpliedOdds is how much more you expect to win on later streets when
	// you hit, as a fraction of the current pot. It is added to the pot
	// when working out the equity a call needs, but only before the river
	// (on the river nothing more can come in). 0 turns it off. Default
	// 0.25.
	ImpliedOdds float64 `json:"implied_odds"`

	// Trials is the number of Monte Carlo deals for the equity estimate.
	// More is more precise and slower. Default 100,000 (a few
	// milliseconds, accurate to about 0.3 percentage points).
	Trials int `json:"trials"`
}

// DefaultConfig returns the default settings.
func DefaultConfig() Config {
	return Config{
		Unit:           "chips",
		ChipStep:       1,
		StrongMultiple: 1.4,
		StrongCap:      0.9,
		BetFraction:    0.75,
		PreflopRaise:   3,
		PositionEdge:   0.05,
		ImpliedOdds:    0.25,
		Trials:         100_000,
	}
}

// Validate reports the first setting that is out of range, if any.
func (c Config) Validate() error {
	switch {
	case c.ChipStep <= 0:
		return errors.New("advisor: chip step must be above 0")
	case c.BigBlind < 0:
		return errors.New("advisor: big blind can't be negative")
	case c.StrongMultiple <= 0:
		return errors.New("advisor: strong multiple must be above 0")
	case c.StrongCap <= 0 || c.StrongCap > 1:
		return errors.New("advisor: strong cap must be above 0 and at most 1")
	case c.BetFraction <= 0:
		return errors.New("advisor: bet fraction must be above 0")
	case c.PreflopRaise < 1:
		return errors.New("advisor: preflop raise must be at least 1 (times the current bet)")
	case c.PositionEdge < 0 || c.PositionEdge > 0.5:
		return errors.New("advisor: position edge must be from 0 to 0.5")
	case c.ImpliedOdds < 0:
		return errors.New("advisor: implied odds can't be negative")
	case c.Trials <= 0:
		return errors.New("advisor: trials must be above 0")
	}
	return nil
}

// Money formats an amount in the configured unit, such as "$12.50" or
// "25 chips". With a big blind set it adds the size in big blinds, as in
// "30 chips (15 BB)".
func (c Config) Money(amount float64) string {
	var s string
	if c.Unit == "$" {
		s = fmt.Sprintf("$%.2f", amount)
	} else {
		s = trimFloat(amount) + " " + c.Unit
	}
	if c.BigBlind > 0 {
		s += fmt.Sprintf(" (%s BB)", trimFloat(math.Round(amount/c.BigBlind*10)/10))
	}
	return s
}

// roundToChip rounds an amount to the nearest multiple of ChipStep. Bet
// sizes come out of fractions of the pot (0.75 x 37 = 27.75), and a
// suggestion you can't actually make with your chips isn't useful.
func (c Config) roundToChip(amount float64) float64 {
	steps := math.Round(amount / c.ChipStep)
	// Round away tiny floating-point leftovers so 0.1 + 0.2 shows as 0.3.
	return math.Round(steps*c.ChipStep*1e9) / 1e9
}

// trimFloat prints a number with as few decimals as it needs: 25, 12.5,
// 0.25.
func trimFloat(f float64) string {
	return strconv.FormatFloat(math.Round(f*1e9)/1e9, 'f', -1, 64)
}
