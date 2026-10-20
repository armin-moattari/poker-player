// Package card represents playing cards and decks, and parses card strings
// such as "As" (ace of spades) and "Td" (ten of diamonds).
//
// A Card is a single byte holding an index from 0 to 51. Keeping cards this
// small makes them cheap to copy and compare, and lets any set of cards be
// stored as one uint64 bitmask (bit i set means card i is in the set).
// Deck uses such masks, and the optimized evaluator will too.
//
// Typical use:
//
//	hand, err := card.ParseCards("As Kd")
//	d := card.NewDeck()
//	d.Remove(hand...)                       // take known cards out
//	d.Shuffle(rand.New(rand.NewPCG(1, 2)))  // seeded, so repeatable
//	board, err := d.Deal(5)
package card

import (
	"fmt"
	"strings"
)

// Rank is a card rank from Two (2) to Ace (14).
//
// Ranks use their face value so that comparing two ranks with < and >
// gives the poker order directly. Ace is high (14); code that needs the
// ace-low "wheel" straight (A-2-3-4-5) handles that case on its own.
type Rank uint8

// Card ranks. Values match the face value, with Jack=11 up to Ace=14.
const (
	Two Rank = iota + 2 // iota starts at 0, so +2 makes Two == 2
	Three
	Four
	Five
	Six
	Seven
	Eight
	Nine
	Ten
	Jack
	Queen
	King
	Ace
)

// Suit is one of the four card suits. Suits have no ranking in hold'em;
// the order below only fixes how cards are numbered.
type Suit uint8

// Card suits, in alphabetical order to match the characters "cdhs".
const (
	Clubs Suit = iota
	Diamonds
	Hearts
	Spades
)

// rankChars and suitChars map ranks and suits to their text form.
// rankChars[r-Two] is the character for rank r, and suitChars[s] is the
// character for suit s. Parse uses the same strings in reverse.
const (
	rankChars = "23456789TJQKA"
	suitChars = "cdhs"
)

// String returns the rank's character, such as "A" or "T".
// It returns "?" for a value outside Two..Ace.
func (r Rank) String() string {
	if r < Two || r > Ace {
		return "?"
	}
	return string(rankChars[r-Two])
}

// String returns the suit's lowercase character: "c", "d", "h" or "s".
// It returns "?" for a value outside Clubs..Spades.
func (s Suit) String() string {
	if s > Spades {
		return "?"
	}
	return string(suitChars[s])
}

// Card is one of the 52 cards, stored as an index from 0 to 51.
//
// The index is (rank-2)*4 + suit, so 0 is 2c, 1 is 2d, ..., 51 is As.
// Sorting cards by index therefore sorts them by rank, then by suit.
// The zero value is a valid card (2c), so use a separate flag or slice
// length, not Card(0), to mean "no card".
type Card uint8

// NumCards is the number of cards in a standard deck.
const NumCards = 52

// New returns the card with the given rank and suit.
// The caller must pass a valid rank (Two..Ace) and suit (Clubs..Spades).
func New(r Rank, s Suit) Card {
	return Card(uint8(r-Two)*4 + uint8(s))
}

// Rank returns the card's rank. Dividing the index by 4 undoes the
// "*4" in New, leaving rank-2.
func (c Card) Rank() Rank { return Rank(c/4) + Two }

// Suit returns the card's suit, which is the remainder after dividing
// the index by 4.
func (c Card) Suit() Suit { return Suit(c % 4) }

// String returns the card in two-character form, such as "As" or "Td".
// It implements fmt.Stringer, so %v and %s print cards this way.
// It returns "??" for an index of 52 or more.
func (c Card) String() string {
	if c >= NumCards {
		return "??"
	}
	return c.Rank().String() + c.Suit().String()
}

// Parse parses a single card such as "As", "td" or "9H".
//
// The first character is the rank (2-9, T, J, Q, K, A) and the second is
// the suit (c, d, h, s). Both are case-insensitive, and "10" is accepted
// as another way to write Ten, so "10c" parses as Tc.
func Parse(s string) (Card, error) {
	// Rewrite "10x" as "Tx" so the rest of the function only has to
	// handle two-character input.
	if strings.HasPrefix(s, "10") {
		s = "T" + s[2:]
	}
	if len(s) != 2 {
		return 0, fmt.Errorf("card: invalid card %q: want 2 characters like \"As\"", s)
	}
	// The position of a character in rankChars/suitChars is its value,
	// offset by Two for ranks.
	ri := strings.IndexByte(rankChars, upper(s[0]))
	if ri < 0 {
		return 0, fmt.Errorf("card: invalid rank in %q", s)
	}
	si := strings.IndexByte(suitChars, lower(s[1]))
	if si < 0 {
		return 0, fmt.Errorf("card: invalid suit in %q", s)
	}
	return New(Rank(ri)+Two, Suit(si)), nil
}

// MustParse is like Parse but panics if s is not a valid card.
// It is meant for hard-coded cards in tests, where a typo should fail loudly.
func MustParse(s string) Card {
	c, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return c
}

// ParseCards parses a list of cards such as "AsKd", "As Kd", "As,Kd" or
// "10c 9c". Cards may be written together or separated by spaces, tabs or
// commas. An empty string gives an empty list.
//
// It returns an error if any card is invalid or if the same card appears
// twice, since a real deal can never contain duplicates.
func ParseCards(s string) ([]Card, error) {
	// Strip separators, then turn every "10" into "T" so that each card is
	// exactly two characters and the string can be read in pairs.
	s = strings.NewReplacer(" ", "", ",", "", "\t", "").Replace(s)
	s = strings.ReplaceAll(s, "10", "T")
	if len(s)%2 != 0 {
		return nil, fmt.Errorf("card: invalid card list %q", s)
	}
	cards := make([]Card, 0, len(s)/2)
	// seen is a bitmask of cards parsed so far: bit c is set once card c
	// has appeared. Checking a bit is cheaper than a map lookup.
	var seen uint64
	for i := 0; i < len(s); i += 2 {
		c, err := Parse(s[i : i+2])
		if err != nil {
			return nil, err
		}
		if seen&(1<<c) != 0 {
			return nil, fmt.Errorf("card: duplicate card %s", c)
		}
		seen |= 1 << c
		cards = append(cards, c)
	}
	return cards, nil
}

// upper converts an ASCII letter to uppercase and leaves other bytes alone.
// It avoids strings.ToUpper, which would allocate a new string.
func upper(b byte) byte {
	if 'a' <= b && b <= 'z' {
		return b - 'a' + 'A'
	}
	return b
}

// lower converts an ASCII letter to lowercase and leaves other bytes alone.
func lower(b byte) byte {
	if 'A' <= b && b <= 'Z' {
		return b - 'A' + 'a'
	}
	return b
}
