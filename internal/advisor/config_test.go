package advisor

import "testing"

func TestDefaultConfigIsValid(t *testing.T) {
	if err := DefaultConfig().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := map[string]func(*Config){
		"zero chip step":        func(c *Config) { c.ChipStep = 0 },
		"negative big blind":    func(c *Config) { c.BigBlind = -1 },
		"zero strong multiple":  func(c *Config) { c.StrongMultiple = 0 },
		"strong cap above 1":    func(c *Config) { c.StrongCap = 1.5 },
		"zero bet fraction":     func(c *Config) { c.BetFraction = 0 },
		"preflop raise below 1": func(c *Config) { c.PreflopRaise = 0.5 },
		"position edge too big": func(c *Config) { c.PositionEdge = 0.6 },
		"negative implied odds": func(c *Config) { c.ImpliedOdds = -0.1 },
		"zero trials":           func(c *Config) { c.Trials = 0 },
	}
	for name, change := range tests {
		c := DefaultConfig()
		change(&c)
		if c.Validate() == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestMoney(t *testing.T) {
	chips := DefaultConfig()
	dollars := DefaultConfig()
	dollars.Unit = "$"
	blinds := DefaultConfig()
	blinds.BigBlind = 2
	tests := []struct {
		cfg    Config
		amount float64
		want   string
	}{
		{chips, 25, "25 chips"},
		{chips, 12.5, "12.5 chips"},
		{dollars, 12.5, "$12.50"},
		{dollars, 0.1 + 0.2, "$0.30"},
		{blinds, 30, "30 chips (15 BB)"},
		{blinds, 5, "5 chips (2.5 BB)"},
	}
	for _, tt := range tests {
		if got := tt.cfg.Money(tt.amount); got != tt.want {
			t.Errorf("Money(%v) = %q, want %q", tt.amount, got, tt.want)
		}
	}
}

func TestRoundToChip(t *testing.T) {
	c := DefaultConfig()
	c.ChipStep = 0.25
	for in, want := range map[float64]float64{2.325: 2.25, 2.4: 2.5, 0.1 + 0.2: 0.25, 10: 10} {
		if got := c.roundToChip(in); got != want {
			t.Errorf("roundToChip(%v) = %v, want %v", in, got, want)
		}
	}
}
