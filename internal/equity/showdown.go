package equity

import (
	"github.com/armin-moattari/poker-player/internal/card"
	"github.com/armin-moattari/poker-player/internal/eval"
)

// showdown ranks every player on a complete five-card board and records the
// outcome for the known hands in t.
//
// oppCards holds the random opponents' hole cards, two per opponent (empty
// when there are none). ranks is scratch space with room for every player,
// passed in so the hot loop never allocates.
func (s *spot) showdown(board *[5]card.Card, oppCards []card.Card, ranks []eval.HandRank, t *tally) {
	// Seven cards: two hole cards in slots 0-1, the board in slots 2-6.
	// The board part is the same for everyone, so copy it once.
	var seven [7]card.Card
	copy(seven[2:], board[:])

	best := eval.HandRank(0)
	i := 0
	for _, h := range s.hands {
		seven[0], seven[1] = h[0], h[1]
		ranks[i] = eval.Evaluate(seven[:])
		best = max(best, ranks[i])
		i++
	}
	for j := 0; j < len(oppCards); j += 2 {
		seven[0], seven[1] = oppCards[j], oppCards[j+1]
		ranks[i] = eval.Evaluate(seven[:])
		best = max(best, ranks[i])
		i++
	}

	// Count how many players share the best hand; they split the pot.
	winners := 0
	for _, r := range ranks[:i] {
		if r == best {
			winners++
		}
	}

	t.deals++
	for k := range s.hands {
		if ranks[k] != best {
			continue
		}
		if winners == 1 {
			t.win[k]++
		} else {
			t.tie[k]++
		}
		t.share[k] += 1 / float64(winners)
	}
}

// players is the total number of players at the table.
func (s *spot) players() int { return len(s.hands) + s.opponents }
