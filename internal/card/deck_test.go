package card

import (
	"math/rand/v2"
	"slices"
	"testing"
)

func TestNewDeckHasAllCards(t *testing.T) {
	d := NewDeck()
	if d.Len() != NumCards {
		t.Fatalf("Len() = %d, want %d", d.Len(), NumCards)
	}
	cards, err := d.Deal(NumCards)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(cards)
	for i, c := range cards {
		if c != Card(i) {
			t.Fatalf("card %d = %v, want %v", i, c, Card(i))
		}
	}
	if d.Len() != 0 {
		t.Errorf("Len() after dealing all = %d, want 0", d.Len())
	}
}

func TestShuffleIsSeededAndKeepsCards(t *testing.T) {
	deal := func(seed uint64) []Card {
		d := NewDeck()
		d.Shuffle(rand.New(rand.NewPCG(seed, 0)))
		cards, _ := d.Deal(NumCards)
		return cards
	}
	a, b, c := deal(1), deal(1), deal(2)
	if !slices.Equal(a, b) {
		t.Error("same seed gave different orders")
	}
	if slices.Equal(a, c) {
		t.Error("different seeds gave the same order")
	}
	slices.Sort(a)
	if !slices.Equal(a, NewDeck().cards) {
		t.Error("shuffled deck does not hold the same 52 cards")
	}
}

func TestDeal(t *testing.T) {
	d := NewDeck()
	d.Shuffle(rand.New(rand.NewPCG(7, 7)))
	hand, err := d.Deal(2)
	if err != nil {
		t.Fatal(err)
	}
	board, err := d.Deal(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hand) != 2 || len(board) != 5 || d.Len() != 45 {
		t.Fatalf("got hand %d, board %d, left %d; want 2, 5, 45", len(hand), len(board), d.Len())
	}
	for _, c := range append(hand, board...) {
		if slices.Contains(d.cards, c) {
			t.Errorf("dealt card %v is still in the deck", c)
		}
	}
	if _, err := d.Deal(0); err != nil {
		t.Errorf("Deal(0) error: %v", err)
	}
}

func TestDealErrors(t *testing.T) {
	d := NewDeck()
	if _, err := d.Deal(NumCards + 1); err == nil {
		t.Error("Deal(53) did not return an error")
	}
	if _, err := d.Deal(-1); err == nil {
		t.Error("Deal(-1) did not return an error")
	}
	if d.Len() != NumCards {
		t.Errorf("failed deals changed the deck: Len() = %d", d.Len())
	}
}

func TestRemove(t *testing.T) {
	d := NewDeck()
	held, _ := ParseCards("As Kd")
	if err := d.Remove(held...); err != nil {
		t.Fatal(err)
	}
	if d.Len() != 50 {
		t.Errorf("Len() = %d, want 50", d.Len())
	}
	for _, c := range held {
		if slices.Contains(d.cards, c) {
			t.Errorf("%v still in deck after Remove", c)
		}
	}
}

func TestRemoveErrorsLeaveDeckUnchanged(t *testing.T) {
	tests := map[string][]Card{
		"already removed": {MustParse("As")},
		"duplicate":       {MustParse("Kd"), MustParse("Kd")},
		"invalid card":    {Card(60)},
		"one bad of two":  {MustParse("2c"), MustParse("As")},
	}
	for name, cards := range tests {
		t.Run(name, func(t *testing.T) {
			d := NewDeck()
			if err := d.Remove(MustParse("As")); err != nil {
				t.Fatal(err)
			}
			if err := d.Remove(cards...); err == nil {
				t.Error("Remove did not return an error")
			}
			if d.Len() != 51 {
				t.Errorf("Len() = %d, want 51 (deck changed on error)", d.Len())
			}
		})
	}
}
