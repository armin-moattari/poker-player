package equity

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/armin-moattari/poker-player/internal/card"
	"github.com/armin-moattari/poker-player/internal/eval"
)

// enumerate deals every possible completion of the board and tallies the
// result of each. It requires every hand to be known.
//
// The work is split into jobs by the first one or two new board cards: a
// job fixes those cards and covers every board that starts with them, taking
// the remaining cards only from later in the deck so each board is counted
// once. Splitting on two cards gives about a thousand jobs preflop, so the
// load stays even across cores (splitting on one card gave only ~44 jobs,
// and the first ones were far bigger than the last, which left cores idle).
//
// Workers take jobs from a shared counter until none are left. Each job
// writes its tally into its own slot, and the slots are summed in order at
// the end, so the result is identical whatever the number of workers.
func (s *spot) enumerate(ctx context.Context, workers int) (tally, error) {
	if s.missing == 0 {
		// River: the board is complete, so there is exactly one deal.
		t := newTally(len(s.hands))
		var board [5]card.Card
		copy(board[:], s.board)
		s.showdown(&board, nil, make([]eval.HandRank, s.players()), &t)
		return t, nil
	}

	// Each job is a prefix of deck indexes for the first new board cards.
	// A one-card prefix is stored with b = -1.
	type prefix struct{ a, b int }
	var prefixes []prefix
	n := len(s.deck)
	for a := 0; a <= n-s.missing; a++ {
		if s.missing == 1 {
			prefixes = append(prefixes, prefix{a, -1})
			continue
		}
		for b := a + 1; b <= n-s.missing+1; b++ {
			prefixes = append(prefixes, prefix{a, b})
		}
	}

	jobs := make([]tally, len(prefixes))
	var next atomic.Int64 // index of the next job to hand out
	var wg sync.WaitGroup
	for range min(workers, len(jobs)) {
		wg.Go(func() {
			ranks := make([]eval.HandRank, s.players())
			var board [5]card.Card
			copy(board[:], s.board)
			known := len(s.board)
			for {
				j := int(next.Add(1) - 1)
				if j >= len(jobs) || ctx.Err() != nil {
					return
				}
				p := prefixes[j]
				t := newTally(len(s.hands))
				board[known] = s.deck[p.a]
				if p.b < 0 {
					s.fill(&board, known+1, p.a+1, ranks, &t)
				} else {
					board[known+1] = s.deck[p.b]
					s.fill(&board, known+2, p.b+1, ranks, &t)
				}
				jobs[j] = t
			}
		})
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return tally{}, err
	}

	total := newTally(len(s.hands))
	for _, t := range jobs {
		total.add(t)
	}
	return total, nil
}

// fill places deck cards into board slots pos..4, choosing only cards at
// deck index from or later so that each combination appears exactly once,
// and runs a showdown for every complete board.
func (s *spot) fill(board *[5]card.Card, pos, from int, ranks []eval.HandRank, t *tally) {
	if pos == 5 {
		s.showdown(board, nil, ranks, t)
		return
	}
	// Leave enough cards after i for the slots still to fill.
	for i := from; i <= len(s.deck)-(5-pos); i++ {
		board[pos] = s.deck[i]
		s.fill(board, pos+1, i+1, ranks, t)
	}
}
