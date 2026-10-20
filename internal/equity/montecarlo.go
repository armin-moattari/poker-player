package equity

import (
	"context"
	"math/rand/v2"
	"sync"
	"sync/atomic"

	"github.com/armin-moattari/poker-player/internal/card"
	"github.com/armin-moattari/poker-player/internal/eval"
)

// chunkTrials is how many Monte Carlo trials make up one unit of work.
// Chunks are small enough to spread evenly over the cores and to notice a
// cancelled context quickly, and large enough that handing them out costs
// nothing measurable.
const chunkTrials = 4096

// simulate runs Monte Carlo trials: each one deals the rest of the board and
// the random opponents' hands at random, then runs a showdown.
//
// Trials are grouped into fixed chunks, and chunk i always uses a random
// generator seeded with (seed, i). Workers take chunks from a shared counter
// and store each chunk's tally in its own slot, summed in order at the end.
// Because every chunk's random numbers depend only on the seed and the chunk
// index, not on which goroutine ran it, the result is the same on any
// number of cores.
func (s *spot) simulate(ctx context.Context, workers, trials int, seed uint64) (tally, error) {
	chunks := make([]tally, (trials+chunkTrials-1)/chunkTrials)
	var next atomic.Int64
	var wg sync.WaitGroup
	for range min(workers, len(chunks)) {
		wg.Go(func() {
			// Per-worker scratch space, reused for every trial so the hot
			// loop never allocates. deck is a private copy because the
			// shuffle below reorders it.
			deck := append([]card.Card(nil), s.deck...)
			ranks := make([]eval.HandRank, s.players())
			var board [5]card.Card
			copy(board[:], s.board)
			need := s.missing + 2*s.opponents

			for {
				i := int(next.Add(1) - 1)
				if i >= len(chunks) || ctx.Err() != nil {
					return
				}
				// Reset the deck order at the start of every chunk. Within
				// a chunk the order carries over from trial to trial,
				// which is fine for randomness, but it must not carry over
				// between chunks: that would make a chunk's deals depend
				// on which chunks this worker happened to run before it.
				copy(deck, s.deck)
				r := rand.New(rand.NewPCG(seed, uint64(i)))
				n := min(chunkTrials, trials-i*chunkTrials)
				t := newTally(len(s.hands))
				for range n {
					// Partial Fisher-Yates shuffle: move `need` random
					// cards to the front of deck. Only those cards are
					// used, so shuffling the rest would be wasted work.
					// The deck does not need resetting between trials:
					// any starting order gives a uniformly random pick.
					for j := range need {
						k := j + r.IntN(len(deck)-j)
						deck[j], deck[k] = deck[k], deck[j]
					}
					copy(board[len(s.board):], deck[:s.missing])
					s.showdown(&board, deck[s.missing:need], ranks, &t)
				}
				chunks[i] = t
			}
		})
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return tally{}, err
	}

	total := newTally(len(s.hands))
	for _, t := range chunks {
		total.add(t)
	}
	return total, nil
}
