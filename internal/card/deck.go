package card

import (
	"fmt"
	"math/rand/v2"
)

// Deck is a pile of cards that are dealt from the top.
//
// The "top" of the deck is the end of the slice. Dealing from the end lets
// Deal shrink the slice in place without moving the remaining cards.
//
// A Deck is not safe for concurrent use. Give each goroutine its own deck.
type Deck struct {
	cards []Card
}

// NewDeck returns a full, unshuffled 52-card deck, ordered 2c, 2d, ..., As.
func NewDeck() *Deck {
	d := &Deck{cards: make([]Card, NumCards)}
	for i := range d.cards {
		d.cards[i] = Card(i)
	}
	return d
}

// Len returns the number of cards left in the deck.
func (d *Deck) Len() int { return len(d.cards) }

// Shuffle randomizes the order of the remaining cards using r.
//
// The random source is a parameter rather than a global so that callers
// control it: a fixed seed gives the same deal every run (useful in tests),
// and each simulation goroutine can own a generator instead of sharing one.
// rand.Shuffle uses the Fisher-Yates algorithm, so every order is equally
// likely.
func (d *Deck) Shuffle(r *rand.Rand) {
	r.Shuffle(len(d.cards), func(i, j int) {
		d.cards[i], d.cards[j] = d.cards[j], d.cards[i]
	})
}

// Deal removes n cards from the top of the deck and returns them.
// It returns an error, and leaves the deck unchanged, if n is negative or
// larger than the number of cards left.
func (d *Deck) Deal(n int) ([]Card, error) {
	if n < 0 || n > len(d.cards) {
		return nil, fmt.Errorf("card: cannot deal %d cards from a deck of %d", n, len(d.cards))
	}
	// Copy the dealt cards into a new slice. Returning a sub-slice of
	// d.cards instead would share memory with the deck, so a later Remove
	// (which reuses that memory) could change cards the caller holds.
	top := len(d.cards) - n
	dealt := make([]Card, n)
	copy(dealt, d.cards[top:])
	d.cards = d.cards[:top]
	return dealt, nil
}

// Remove takes the given cards out of the deck, for example the hole cards
// and board cards already known when computing equity.
//
// It returns an error, and leaves the deck unchanged, if any card is not in
// the deck, is listed twice, or is not a valid card. Checking everything
// before changing anything means a failed call has no side effects.
func (d *Deck) Remove(cards ...Card) error {
	// Build a bitmask of the cards currently in the deck, so each
	// membership check below is a single AND instead of a scan.
	var inDeck uint64
	for _, c := range d.cards {
		inDeck |= 1 << c
	}
	// Validate every card and collect the ones to remove in a second mask.
	// The c >= NumCards check matters: shifting 1 by 64 or more gives 0 in
	// Go, which would make an invalid card look like it is not in the mask.
	var remove uint64
	for _, c := range cards {
		if c >= NumCards || inDeck&(1<<c) == 0 || remove&(1<<c) != 0 {
			return fmt.Errorf("card: %s is not in the deck", c)
		}
		remove |= 1 << c
	}
	// Filter in place: kept shares d.cards' memory and is never longer than
	// the part already read, so no allocation is needed.
	kept := d.cards[:0]
	for _, c := range d.cards {
		if remove&(1<<c) == 0 {
			kept = append(kept, c)
		}
	}
	d.cards = kept
	return nil
}
