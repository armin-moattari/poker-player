package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cli runs the command with args and returns its exit code and output.
// It points the user config directory at a fresh temporary folder, so tests
// never read or write the real settings file.
func cli(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)            // macOS: ~/Library/Application Support
	t.Setenv("XDG_CONFIG_HOME", home) // Linux
	var out, errOut bytes.Buffer
	code = run(context.Background(), args, &out, &errOut)
	return code, out.String(), errOut.String()
}

// writeFile creates a settings file with the given JSON and returns its path.
func writeFile(t *testing.T, json string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "advisor.json")
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAdvise(t *testing.T) {
	code, out, _ := cli(t, "advise", "--hand", "As Kd", "--board", "Qs Jh 2c",
		"--pot", "40", "--to-call", "10", "--players", "3", "--seed", "1")
	want := `Recommendation: CALL, putting in 10 chips
  flop, holding high card
  win share vs random hands: 44% (fair share 33%)
  pot odds: 20%; needed to call: 17%
  why: you need 17% (pot odds 20%, adjusted for implied odds) and win about 44%
`
	if code != 0 || out != want {
		t.Errorf("exit %d, output:\n%s\nwant:\n%s", code, out, want)
	}
}

func TestAdviseWithSettingFlags(t *testing.T) {
	code, out, _ := cli(t, "advise", "--hand", "As Kd", "--board", "Qs Jh 2c",
		"--pot", "40", "--to-call", "10", "--players", "3", "--seed", "1",
		"--position", "early", "--unit", "$", "--chip", "0.25", "--big-blind", "2")
	want := `Recommendation: CALL, putting in $10.00 (5 BB)
  flop, holding high card, early position
  win share vs random hands: 44% (fair share 33%)
  pot odds: 20%; needed to call: 22%
  why: you need 22% (pot odds 20%, adjusted for implied odds and early position) and win about 44%
`
	if code != 0 || out != want {
		t.Errorf("exit %d, output:\n%s\nwant:\n%s", code, out, want)
	}
}

func TestAdviseAllIn(t *testing.T) {
	code, out, _ := cli(t, "advise", "--hand", "AsAh", "--pot", "15", "--to-call", "10", "--stack", "20", "--seed", "1")
	if code != 0 || !strings.HasPrefix(out, "Recommendation: RAISE, putting in 20 chips, all-in\n") {
		t.Errorf("exit %d, output:\n%s", code, out)
	}
}

func TestAdviseCheckHasNoPotOdds(t *testing.T) {
	_, out, _ := cli(t, "advise", "--hand", "7h2c", "--pot", "3", "--seed", "1")
	if !strings.HasPrefix(out, "Recommendation: CHECK\n") || strings.Contains(out, "pot odds") {
		t.Errorf("output:\n%s", out)
	}
}

// TestSettingsLayers checks the order: defaults, then the settings file,
// then flags typed on the command line.
func TestSettingsLayers(t *testing.T) {
	path := writeFile(t, `{"unit": "$", "big_blind": 2, "bet_fraction": 0.5}`)
	args := []string{"advise", "--config", path, "--hand", "AsAh", "--board", "Ks 7d 2c", "--pot", "20", "--seed", "1"}

	// From the file: dollars, big blinds, and a half-pot bet of $10.
	_, out, _ := cli(t, args...)
	if !strings.HasPrefix(out, "Recommendation: BET, putting in $10.00 (5 BB)\n") {
		t.Errorf("file settings not applied:\n%s", out)
	}
	// A flag overrides just that one setting: a full-pot bet, still in dollars.
	_, out, _ = cli(t, append(args, "--bet-size", "1")...)
	if !strings.HasPrefix(out, "Recommendation: BET, putting in $20.00 (10 BB)\n") {
		t.Errorf("flag did not override the file:\n%s", out)
	}
}

func TestConfigCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "advisor.json")

	code, out, _ := cli(t, "config", "--config", path)
	if code != 0 || !strings.Contains(out, "settings file: "+path) || !strings.Contains(out, `"bet_fraction": 0.75`) {
		t.Errorf("config with no file: exit %d, output:\n%s", code, out)
	}

	code, out, _ = cli(t, "config", "--config", path, "--init")
	if code != 0 || out != "created "+path+"\n" {
		t.Errorf("config --init: exit %d, output %q", code, out)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("settings file not created: %v", err)
	}

	code, _, stderr := cli(t, "config", "--config", path, "--init")
	if code != 2 || !strings.Contains(stderr, "already exists") {
		t.Errorf("second --init: exit %d, stderr %q; want a refusal", code, stderr)
	}

	// The default location is used when --config is left out.
	code, out, _ = cli(t, "config")
	if code != 0 || !strings.Contains(out, filepath.Join("poker-player", "advisor.json")) {
		t.Errorf("default path: exit %d, output:\n%s", code, out)
	}
}

func TestEquity(t *testing.T) {
	code, out, _ := cli(t, "equity", "--hands", "AhAs KdKc")
	want := `hand     equity      win      tie   (exact, 1712304 boards)
Ah As    81.26%   81.06%    0.38%
Kd Kc    18.74%   18.55%    0.38%
`
	if code != 0 || out != want {
		t.Errorf("exit %d, output:\n%s\nwant:\n%s", code, out, want)
	}
	code, out, _ = cli(t, "equity", "--hands", "AsKd", "--board", "Qs Jh 2c", "--opponents", "2", "--seed", "1")
	if code != 0 || !strings.Contains(out, "(Monte Carlo, 100000 deals)") || !strings.Contains(out, "As Kd    43.81%") {
		t.Errorf("exit %d, output:\n%s", code, out)
	}
}

func TestUsageAndErrors(t *testing.T) {
	badKey := writeFile(t, `{"bet_fracton": 0.5}`)
	badValue := writeFile(t, `{"chip_step": 0}`)
	notJSON := writeFile(t, `bet_fraction = 0.5`)
	tests := []struct {
		args     []string
		code     int
		inStderr string
	}{
		{nil, 2, "usage:"},
		{[]string{"help"}, 0, ""},
		{[]string{"advise", "-h"}, 0, "-to-call"},
		{[]string{"bogus"}, 2, `unknown command "bogus"`},
		{[]string{"advise", "--hand", "Zz"}, 2, "invalid"},
		{[]string{"advise", "--hand", "AsKd", "--board", "Xx"}, 2, "invalid"},
		{[]string{"advise", "--hand", "AsKd", "--position", "button"}, 2, "unknown position"},
		{[]string{"advise", "--hand", "As"}, 2, "2 hole cards"},
		{[]string{"advise", "--hand", "AsKd", "--chip", "0"}, 2, "chip step"},
		{[]string{"advise", "--nope"}, 2, "flag provided but not defined"},
		{[]string{"advise", "--hand", "AsKd", "--config", badKey}, 2, `unknown field "bet_fracton"`},
		{[]string{"advise", "--hand", "AsKd", "--config", badValue}, 2, "chip step"},
		{[]string{"advise", "--hand", "AsKd", "--config", notJSON}, 2, "reading"},
		{[]string{"advise", "--hand", "AsKd", "--config", t.TempDir()}, 2, "is a directory"},
		{[]string{"config", "--config", badKey}, 2, "unknown field"},
		{[]string{"config", "--nope"}, 2, "flag provided but not defined"},
		{[]string{"equity", "--hands", "AsAh Zz"}, 2, "invalid"},
		{[]string{"equity", "--hands", "AsAh", "--board", "Xx"}, 2, "invalid"},
		{[]string{"equity", "--hands", "AsAh"}, 2, "need 2 to 10 players"},
		{[]string{"equity", "--nope"}, 2, "flag provided but not defined"},
	}
	for _, tt := range tests {
		code, _, stderr := cli(t, tt.args...)
		if code != tt.code || !strings.Contains(stderr, tt.inStderr) {
			t.Errorf("cli %q: exit %d, stderr %q; want exit %d with %q", tt.args, code, stderr, tt.code, tt.inStderr)
		}
	}
}
