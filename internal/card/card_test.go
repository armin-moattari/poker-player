package card

import "testing"

func TestNewRoundTrip(t *testing.T) {
	seen := make(map[Card]bool)
	for r := Two; r <= Ace; r++ {
		for s := Clubs; s <= Spades; s++ {
			c := New(r, s)
			if c.Rank() != r || c.Suit() != s {
				t.Errorf("New(%v, %v) = %v; Rank=%v Suit=%v", r, s, c, c.Rank(), c.Suit())
			}
			if c >= NumCards {
				t.Errorf("New(%v, %v) = %d, out of range", r, s, c)
			}
			if seen[c] {
				t.Errorf("New(%v, %v) = %d, duplicate index", r, s, c)
			}
			seen[c] = true
		}
	}
	if len(seen) != NumCards {
		t.Errorf("got %d distinct cards, want %d", len(seen), NumCards)
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		in   string
		want Card
	}{
		{"As", New(Ace, Spades)},
		{"Td", New(Ten, Diamonds)},
		{"2c", New(Two, Clubs)},
		{"9h", New(Nine, Hearts)},
		{"kH", New(King, Hearts)},
		{"qs", New(Queen, Spades)},
		{"10c", New(Ten, Clubs)},
		{"JD", New(Jack, Diamonds)},
	}
	for _, tt := range tests {
		got, err := Parse(tt.in)
		if err != nil {
			t.Errorf("Parse(%q) error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Parse(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, in := range []string{"", "A", "Asx", "1s", "Ax", "Xs", "11s", "  "} {
		if c, err := Parse(in); err == nil {
			t.Errorf("Parse(%q) = %v, want error", in, c)
		}
	}
}

func TestStringRoundTrip(t *testing.T) {
	for c := Card(0); c < NumCards; c++ {
		got, err := Parse(c.String())
		if err != nil || got != c {
			t.Errorf("Parse(%q) = %v, %v; want %v", c.String(), got, err, c)
		}
	}
}

func TestStringInvalid(t *testing.T) {
	if got := Card(NumCards).String(); got != "??" {
		t.Errorf("Card(52).String() = %q, want \"??\"", got)
	}
}

func TestParseCards(t *testing.T) {
	want := []Card{MustParse("As"), MustParse("Kd"), MustParse("Tc")}
	for _, in := range []string{"AsKdTc", "As Kd Tc", "As,Kd,Tc", "as kd 10c"} {
		got, err := ParseCards(in)
		if err != nil {
			t.Errorf("ParseCards(%q) error: %v", in, err)
			continue
		}
		if len(got) != len(want) {
			t.Errorf("ParseCards(%q) = %v, want %v", in, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("ParseCards(%q) = %v, want %v", in, got, want)
				break
			}
		}
	}
}

func TestParseCardsEmpty(t *testing.T) {
	got, err := ParseCards("")
	if err != nil || len(got) != 0 {
		t.Errorf("ParseCards(\"\") = %v, %v; want empty, nil", got, err)
	}
}

func TestParseCardsErrors(t *testing.T) {
	for _, in := range []string{"AsAs", "As K", "AsXx", "As Kd Q"} {
		if got, err := ParseCards(in); err == nil {
			t.Errorf("ParseCards(%q) = %v, want error", in, got)
		}
	}
}

func TestMustParsePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("MustParse(\"Zz\") did not panic")
		}
	}()
	MustParse("Zz")
}
