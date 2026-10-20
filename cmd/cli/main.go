// Command cli is the command-line front end for the poker engine.
//
// Usage:
//
//	go run ./cmd/cli advise --hand "As Kd" --board "Qs Jh 2c" --pot 40 --to-call 10 --players 3
//	go run ./cmd/cli equity --hands "AhAs KdKc" [--board "..."] [--opponents N]
//	go run ./cmd/cli config [--init]
//
// The advise command prints a recommendation for your spot, the equity
// command prints each hand's chance of winning, and the config command shows
// or creates the advisor settings file. All of the poker logic lives in
// internal/advisor and internal/equity; this package only parses flags,
// reads settings and formats text.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/armin-moattari/poker-player/internal/advisor"
	"github.com/armin-moattari/poker-player/internal/card"
	"github.com/armin-moattari/poker-player/internal/equity"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

const usage = `usage:
  cli advise --hand "As Kd" [--board "Qs Jh 2c"] [--pot 40] [--to-call 10] [--players 3] [--position late] ...
  cli equity --hands "AhAs KdKc" [--board "Qs Jh 2c"] [--opponents 2]
  cli config [--init]

Run "cli <command> -h" for every option.
`

// run is main without the process exit, so tests can call it with their own
// arguments and output buffers. It returns the exit code: 0 on success, 2
// for an error such as a bad flag, bad cards or a bad settings file.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	var err error
	switch args[0] {
	case "advise":
		err = runAdvise(ctx, args[1:], stdout, stderr)
	case "equity":
		err = runEquity(ctx, args[1:], stdout, stderr)
	case "config":
		err = runConfig(args[1:], stdout, stderr)
	case "-h", "-help", "--help", "help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		err = fmt.Errorf("unknown command %q", args[0])
	}
	switch {
	case err == nil:
		return 0
	case errors.Is(err, flag.ErrHelp):
		return 0 // -h already printed the flag list
	default:
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}
}

// runAdvise handles "cli advise".
//
// Settings come from three layers, each overriding the one before: the
// built-in defaults, the settings file, and any setting flags given on this
// command line. Only flags you actually type override the file, so
// "--bet-size 0.5" changes bet sizing for one hand without touching the rest
// of your saved settings.
func runAdvise(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("advise", flag.ContinueOnError)
	fs.SetOutput(stderr)

	// The spot.
	hand := fs.String("hand", "", `your two cards, e.g. "As Kd" (required)`)
	board := fs.String("board", "", `community cards so far, e.g. "Qs Jh 2c"; leave out preflop`)
	pot := fs.Float64("pot", 0, "amount in the pot, including bets made this round")
	toCall := fs.Float64("to-call", 0, "amount you must put in to call (0 if you can check)")
	stack := fs.Float64("stack", 0, "amount you have behind (0 = ignore stack size)")
	players := fs.Int("players", 2, "players still in the hand, including you")
	position := fs.String("position", "", "where you act: early, middle or late (leave out to ignore)")
	seed := fs.Uint64("seed", 0, "random seed; the same seed gives the same advice")

	// Settings that override the settings file for this hand.
	configFile := fs.String("config", "", "settings file (default: the file shown by \"cli config\")")
	def := advisor.DefaultConfig()
	unit := fs.String("unit", def.Unit, `how to show amounts: "chips", "$" or any word`)
	chip := fs.Float64("chip", def.ChipStep, "smallest bet you can make, e.g. 1 or 0.25; bets are rounded to it")
	bigBlind := fs.Float64("big-blind", def.BigBlind, "big blind size; shows amounts in big blinds and sizes preflop raises (0 = off)")
	betSize := fs.Float64("bet-size", def.BetFraction, "bet size after the flop, as a fraction of the pot")
	preflopRaise := fs.Float64("preflop-raise", def.PreflopRaise, "preflop raise size, times the current bet (needs --big-blind)")
	implied := fs.Float64("implied", def.ImpliedOdds, "extra you expect to win later when you hit, as a fraction of the pot (0 = off)")
	positionEdge := fs.Float64("position-edge", def.PositionEdge, "how much equity position is worth, 0 to 0.5 (0 = off)")
	trials := fs.Int("trials", def.Trials, "simulated deals (more = more precise)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	path := *configFile
	if path == "" {
		var err error
		if path, err = configPath(); err != nil {
			return err
		}
	}
	cfg, err := loadConfig(path)
	if err != nil {
		return err
	}
	// fs.Visit only visits flags that were set on the command line, which
	// is exactly the set that should override the file.
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "unit":
			cfg.Unit = *unit
		case "chip":
			cfg.ChipStep = *chip
		case "big-blind":
			cfg.BigBlind = *bigBlind
		case "bet-size":
			cfg.BetFraction = *betSize
		case "preflop-raise":
			cfg.PreflopRaise = *preflopRaise
		case "implied":
			cfg.ImpliedOdds = *implied
		case "position-edge":
			cfg.PositionEdge = *positionEdge
		case "trials":
			cfg.Trials = *trials
		}
	})

	spot := advisor.Spot{Pot: *pot, ToCall: *toCall, Stack: *stack, Players: *players, Seed: *seed}
	if spot.Hand, err = card.ParseCards(*hand); err != nil {
		return err
	}
	if spot.Board, err = card.ParseCards(*board); err != nil {
		return err
	}
	if spot.Position, err = advisor.ParsePosition(*position); err != nil {
		return err
	}
	a, err := advisor.Advise(ctx, spot, cfg)
	if err != nil {
		return err
	}
	fmt.Fprint(stdout, formatAdvice(a, spot, cfg))
	return nil
}

// formatAdvice turns an Advice into the text the CLI prints, for example:
//
//	Recommendation: CALL, putting in 10 chips
//	  flop, holding high card
//	  win share vs random hands: 44% (fair share 33%)
//	  pot odds: 20%; needed to call: 17%
//	  why: you need 17% (pot odds 20%, adjusted for implied odds) and win about 44%
func formatAdvice(a advisor.Advice, spot advisor.Spot, cfg advisor.Config) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Recommendation: %s", strings.ToUpper(a.Action.String()))
	if a.Amount > 0 {
		fmt.Fprintf(&b, ", putting in %s", cfg.Money(a.Amount))
		if a.AllIn {
			b.WriteString(", all-in")
		}
	}
	fmt.Fprintf(&b, "\n  %s, holding %s", a.Street, a.HandName)
	if spot.Position != advisor.PositionUnknown {
		fmt.Fprintf(&b, ", %s position", spot.Position)
	}
	fmt.Fprintf(&b, "\n  win share vs random hands: %.0f%% (fair share %.0f%%)\n", 100*a.Equity, 100*a.FairShare)
	if a.PotOdds > 0 {
		fmt.Fprintf(&b, "  pot odds: %.0f%%; needed to call: %.0f%%\n", 100*a.PotOdds, 100*a.Needed)
	}
	fmt.Fprintf(&b, "  why: %s\n", a.Reason)
	return b.String()
}

// runConfig handles "cli config": it prints where the settings file lives
// and the settings in effect, or with --init creates the file with the
// defaults so you can edit it.
func runConfig(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configFile := fs.String("config", "", "settings file (default: the standard location)")
	initFile := fs.Bool("init", false, "create the settings file with the default settings")
	if err := fs.Parse(args); err != nil {
		return err
	}
	path := *configFile
	if path == "" {
		var err error
		if path, err = configPath(); err != nil {
			return err
		}
	}
	if *initFile {
		if err := writeConfig(path, advisor.DefaultConfig()); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "created %s\n", path)
		return nil
	}
	cfg, err := loadConfig(path)
	if err != nil {
		return err
	}
	data, _ := json.MarshalIndent(cfg, "", "  ") // a Config always marshals
	fmt.Fprintf(stdout, "settings file: %s\n%s\n", path, data)
	return nil
}

// runEquity handles "cli equity".
func runEquity(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("equity", flag.ContinueOnError)
	fs.SetOutput(stderr)
	hands := fs.String("hands", "", `known hands separated by spaces, e.g. "AhAs KdKc" (required)`)
	board := fs.String("board", "", "community cards so far")
	opponents := fs.Int("opponents", 0, "extra players with unknown cards")
	trials := fs.Int("trials", equity.DefaultTrials, "Monte Carlo deals, used when there are unknown opponents")
	seed := fs.Uint64("seed", 0, "random seed for Monte Carlo")
	if err := fs.Parse(args); err != nil {
		return err
	}

	req := equity.Request{Opponents: *opponents, Trials: *trials, Seed: *seed}
	for _, h := range strings.Fields(*hands) {
		cs, err := card.ParseCards(h)
		if err != nil {
			return err
		}
		req.Hands = append(req.Hands, cs)
	}
	var err error
	if req.Board, err = card.ParseCards(*board); err != nil {
		return err
	}
	r, err := equity.Calculate(ctx, req)
	if err != nil {
		return err
	}

	method := fmt.Sprintf("Monte Carlo, %d deals", r.Deals)
	if r.Exact {
		method = fmt.Sprintf("exact, %d boards", r.Deals)
	}
	fmt.Fprintf(stdout, "%-6s %8s %8s %8s   (%s)\n", "hand", "equity", "win", "tie", method)
	for i, h := range r.Hands {
		fmt.Fprintf(stdout, "%-6s %7.2f%% %7.2f%% %7.2f%%\n",
			fmt.Sprint(req.Hands[i][0], req.Hands[i][1]), 100*h.Equity, 100*h.Win, 100*h.Tie)
	}
	return nil
}
